package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vps-gateway/internal/dhcp"
	"vps-gateway/internal/logger"
	"vps-gateway/internal/metrics"
	"vps-gateway/internal/server"
	"vps-gateway/internal/sshproxy"

	"strconv"
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
	grpcPort := flag.Int("grpc-port", getEnvInt("GATEWAY_GRPC_PORT", 1537), "gRPC server port (default: 1537 = OCG)")
	_ = flag.Int("metrics-port", getEnvInt("GATEWAY_METRICS_PORT", 9091), "Prometheus metrics port")
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

	// Create and start gateway server (forward connection pattern)
	gatewayServer, err := server.NewGatewayServer(dhcpManager, sshProxy, *grpcPort)
	if err != nil {
		log.Fatalf("Failed to create gateway server: %v", err)
	}

	// Start server in background
	serverErrChan := make(chan error, 1)
	go func() {
		if err := gatewayServer.Start(); err != nil {
			serverErrChan <- err
		}
	}()

	// Wait for interrupt signal or server error
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.Info("Received signal %v, shutting down...", sig)
	case err := <-serverErrChan:
		logger.Error("Server error: %v", err)
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := gatewayServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error shutting down gateway server: %v", err)
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
