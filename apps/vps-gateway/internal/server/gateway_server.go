package server

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"vps-gateway/internal/dhcp"
	"vps-gateway/internal/logger"
	"vps-gateway/internal/sshproxy"

	vpsgatewayv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1/vpsgatewayv1connect"

	"connectrpc.com/connect"
)

// GatewayServer wraps the GatewayService and manages the HTTP server
type GatewayServer struct {
	service    *GatewayService
	httpServer *http.Server
	port       int
	apiSecret  string
}

func NewGatewayServer(dhcpManager *dhcp.Manager, sshProxy *sshproxy.Proxy, port int) (*GatewayServer, error) {
	apiSecret := os.Getenv("GATEWAY_API_SECRET")
	if apiSecret == "" {
		return nil, fmt.Errorf("GATEWAY_API_SECRET environment variable is required")
	}

	service := NewGatewayService(dhcpManager, sshProxy)

	// Create auth interceptor
	authInterceptor := newGatewayAuthInterceptor(apiSecret)

	// Create service handler with auth
	path, handler := vpsgatewayv1connect.NewVPSGatewayServiceHandler(
		service,
		connect.WithInterceptors(authInterceptor),
	)

	// Create HTTP mux
	mux := http.NewServeMux()
	mux.Handle(path, handler)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return &GatewayServer{
		service:    service,
		httpServer: httpServer,
		port:       port,
		apiSecret:  apiSecret,
	}, nil
}

// Start starts the gateway server
func (s *GatewayServer) Start() error {
	logger.Info("Starting gateway gRPC server on port %d (OCG - Obiente Cloud Gateway)", s.port)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *GatewayServer) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// gatewayAuthInterceptor validates the API secret header
type gatewayAuthInterceptor struct {
	secret string
}

func newGatewayAuthInterceptor(secret string) connect.Interceptor {
	return &gatewayAuthInterceptor{secret: secret}
}

func (i *gatewayAuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if err := i.validateSecret(req.Header()); err != nil {
			return nil, err
		}
		return next(ctx, req)
	}
}

func (i *gatewayAuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *gatewayAuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if err := i.validateSecret(conn.RequestHeader()); err != nil {
			return err
		}
		return next(ctx, conn)
	}
}

func (i *gatewayAuthInterceptor) validateSecret(header http.Header) error {
	secret := header.Get("x-api-secret")
	if secret != i.secret {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid API secret"))
	}
	return nil
}
