package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"vps-gateway/internal/client"
	"vps-gateway/internal/dhcp"
	"vps-gateway/internal/logger"
	"vps-gateway/internal/metrics"
	"vps-gateway/internal/network"
	"vps-gateway/internal/server"
	"vps-gateway/internal/sshproxy"
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

	// Initialize DHCP manager first (needed for subnet info)
	dhcpManager, err := dhcp.NewManager()
	if err != nil {
		log.Fatalf("Failed to initialize DHCP manager: %v", err)
	}

	// Initialize SSH proxy (used by gateway server for SSH tunneling)
	sshProxy, err := sshproxy.NewProxy(dhcpManager)
	if err != nil {
		log.Fatalf("Failed to initialize SSH proxy: %v", err)
	}

	// Get subnet configuration from DHCP manager
	_, _, subnetMask, gatewayIP, _ := dhcpManager.GetConfig()

	// Create API client for bidirectional communication with VPS service
	apiClient, err := client.NewAPIClient(dhcpManager)
	if err != nil {
		log.Fatalf("Failed to create API client: %v", err)
	}

	// Provide API client to DHCP manager for FindVPSByLease requests
	dhcpManager.SetAPIClient(apiClient)

	// Start API client in background to maintain persistent connections
	go func() {
		if err := apiClient.Connect(context.Background()); err != nil {
			logger.Error("API client error: %v", err)
		}
	}()

	// Get outbound IP configuration (optional)
	outboundIP := os.Getenv("GATEWAY_OUTBOUND_IP")
	outboundIface := os.Getenv("GATEWAY_OUTBOUND_INTERFACE") // Optional: manual interface selection

	// Initialize SNAT manager if outbound IP is configured
	var snatManager *network.SNATManager
	if outboundIP != "" {
		logger.Info("GATEWAY_OUTBOUND_IP configured: %s", outboundIP)
		snatManager, err = network.NewSNATManager(outboundIP, gatewayIP, subnetMask, outboundIface)
		if err != nil {
			log.Fatalf("Failed to initialize SNAT manager: %v", err)
		}
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

	// Remove SNAT rules
	if snatManager != nil {
		if err := snatManager.RemoveSNAT(); err != nil {
			logger.Error("Error removing SNAT rules: %v", err)
		}
	}

	logger.Info("Shutdown complete")
}
