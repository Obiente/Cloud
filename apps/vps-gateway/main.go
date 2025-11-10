package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vps-gateway/internal/auth"
	"vps-gateway/internal/dhcp"
	"vps-gateway/internal/logger"
	"vps-gateway/internal/metrics"
	"vps-gateway/internal/server"
	"vps-gateway/internal/sshproxy"

	vpsgatewayv1connect "vps-gateway/gen/proto/obiente/cloud/vpsgateway/v1/vpsgatewayv1connect"

	"strconv"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// getEnvInt gets an integer value from environment variable or returns default
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func main() {
	// Parse command line flags
	grpcPort := flag.Int("grpc-port", getEnvInt("GATEWAY_GRPC_PORT", 8080), "gRPC server port")
	metricsPort := flag.Int("metrics-port", getEnvInt("GATEWAY_METRICS_PORT", 9091), "Prometheus metrics port")
	flag.Parse()

	// Initialize logger
	logger.Init()

	// Get configuration from environment
	apiSecret := os.Getenv("GATEWAY_API_SECRET")
	if apiSecret == "" {
		log.Fatal("GATEWAY_API_SECRET environment variable is required")
	}

	// Initialize DHCP manager
	dhcpManager, err := dhcp.NewManager()
	if err != nil {
		log.Fatalf("Failed to initialize DHCP manager: %v", err)
	}

	// Initialize SSH proxy
	sshProxy, err := sshproxy.NewProxy(dhcpManager)
	if err != nil {
		log.Fatalf("Failed to initialize SSH proxy: %v", err)
	}

	// Initialize metrics
	metrics.Init()

	// Create gRPC server
	gatewayService := server.NewGatewayService(dhcpManager, sshProxy)
	gatewayPath, gatewayHandler := vpsgatewayv1connect.NewVPSGatewayServiceHandler(
		gatewayService,
		connect.WithInterceptors(auth.NewAuthInterceptor(apiSecret)),
	)

	// Create reflection service for gRPC tools
	reflector := grpcreflect.NewStaticReflector(
		vpsgatewayv1connect.VPSGatewayServiceName,
	)

	// Create HTTP mux
	mux := http.NewServeMux()
	mux.Handle(gatewayPath, gatewayHandler)

	// Add gRPC reflection handlers
	reflectionV1Path, reflectionV1Handler := grpcreflect.NewHandlerV1(reflector)
	reflectionV1AlphaPath, reflectionV1AlphaHandler := grpcreflect.NewHandlerV1Alpha(reflector)
	mux.Handle(reflectionV1Path, reflectionV1Handler)
	mux.Handle(reflectionV1AlphaPath, reflectionV1AlphaHandler)

	// Add Prometheus metrics endpoint
	mux.Handle("/metrics", metrics.Handler())

	// Create HTTP server with h2c (HTTP/2 Cleartext) for gRPC
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *grpcPort),
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	// Create metrics HTTP server
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *metricsPort),
		Handler: mux,
	}

	// Start servers
	go func() {
		logger.Info("Starting gRPC server on port %d", *grpcPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start gRPC server: %v", err)
		}
	}()

	go func() {
		logger.Info("Starting metrics server on port %d", *metricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start metrics server: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down servers...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("Error shutting down gRPC server: %v", err)
	}

	if err := metricsServer.Shutdown(ctx); err != nil {
		logger.Error("Error shutting down metrics server: %v", err)
	}

	// Cleanup
	if err := dhcpManager.Close(); err != nil {
		logger.Error("Error closing DHCP manager: %v", err)
	}

	if err := sshProxy.Close(); err != nil {
		logger.Error("Error closing SSH proxy: %v", err)
	}

	logger.Info("Shutdown complete")
}
