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
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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

	service, err := NewGatewayService(dhcpManager, sshProxy)
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway service: %w", err)
	}

	// Create auth interceptor
	authInterceptor := newGatewayAuthInterceptor(apiSecret)

	// Create service handler with auth
	path, handler := vpsgatewayv1connect.NewVPSGatewayServiceHandler(
		service,
		connect.WithInterceptors(authInterceptor),
	)

	// Create HTTP mux with request logging
	mux := http.NewServeMux()
	
	// Wrap handler with logging middleware
	loggingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("[GatewayServer] Incoming request: %s %s from %s (HTTP/%s)", r.Method, r.URL.Path, r.RemoteAddr, r.Proto)
		handler.ServeHTTP(w, r)
	})
	mux.Handle(path, loggingHandler)

	// Wrap with h2c for HTTP/2 support (cleartext HTTP/2)
	// This allows Connect RPC clients to use HTTP/2, which is more efficient for streaming
	// h2c handler is backward compatible with HTTP/1.1
	h2cHandler := h2c.NewHandler(mux, &http2.Server{})

	// Create HTTP server with h2c handler
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: h2cHandler,
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

// GetService returns the gateway service for use by DHCP manager
func (s *GatewayServer) GetService() *GatewayService {
	return s.service
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
