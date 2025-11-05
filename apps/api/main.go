package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api/internal/database"
	"api/internal/dnsdelegation"
	"api/internal/logger"
	"api/internal/orchestrator"
	apisrv "api/internal/server"

	_ "github.com/joho/godotenv/autoload"
)

const (
	defaultPort             = "3001"
	readHeaderTimeout       = 10 * time.Second
	writeTimeout            = 30 * time.Second
	idleTimeout             = 2 * time.Minute
	gracefulShutdownMessage = "shutting down server"
)

func main() {
	// Set log output and flags
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	// Initialize logger with LOG_LEVEL
	logger.Init()

	logger.Info("=== Obiente Cloud API Starting ===")
	logger.Debug("LOG_LEVEL: %s", os.Getenv("LOG_LEVEL"))
	logger.Debug("CORS_ORIGIN: %s", os.Getenv("CORS_ORIGIN"))

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		logger.Fatalf("failed to initialize database: %v", err)
	}
	logger.Info("✓ Database initialized")

	// Initialize Redis
	if err := database.InitRedis(); err != nil {
		logger.Warn("Redis initialization failed: %v", err)
	} else {
		logger.Info("✓ Redis initialized")
	}

	// Initialize orchestrator service for metrics collection, health checks, etc.
	var orchService *orchestrator.OrchestratorService
	syncInterval := 30 * time.Second
	if syncIntervalStr := os.Getenv("ORCHESTRATOR_SYNC_INTERVAL"); syncIntervalStr != "" {
		if parsed, err := time.ParseDuration(syncIntervalStr); err == nil {
			syncInterval = parsed
		}
	}

	orchService, err := orchestrator.NewOrchestratorService("least-loaded", 50, syncInterval)
	if err != nil {
		logger.Warn("⚠️  Failed to initialize orchestrator service: %v", err)
		logger.Debug("⚠️  Error details: %+v", err)
		logger.Warn("⚠️  Metrics collection will not be available")
		logger.Warn("⚠️  The server will attempt to create a deployment manager as fallback")
		logger.Warn("⚠️  However, deployments may fail if Docker is not accessible")
		logger.Warn("⚠️  Please check Docker connection and ensure Docker daemon is running")
	} else {
		logger.Info("✓ Orchestrator service initialized")
		orchService.Start()
		logger.Info("✓ Orchestrator service started (metrics collection, health checks, usage aggregation)")
		defer func() {
			if orchService != nil {
				orchService.Stop()
			}
		}()
	}

	// Start DNS pusher service (for dev/self-hosted APIs to push DNS records to production)
	pusherConfig := dnsdelegation.ParsePusherConfig()
	if pusherConfig.ProductionAPIURL != "" && pusherConfig.APIKey != "" {
		dnsdelegation.StartDNSPusher(pusherConfig)
		logger.Info("✓ DNS pusher service started (pushing DNS records to production)")
	} else {
		logger.Debug("DNS pusher not configured (set DNS_DELEGATION_PRODUCTION_API_URL and DNS_DELEGATION_API_KEY to enable)")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	logger.Info("✓ Creating HTTP server with middleware...")
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           apisrv.New(),
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	// Set up graceful shutdown
	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("=== Server Ready - Listening on %s ===", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Wait for interrupt or server error
	select {
	case err := <-serverErr:
		logger.Fatalf("server failed: %v", err)
	case <-shutdownCtx.Done():
		logger.Info("\n=== Shutting down gracefully ===")
		shutdownTimeout := 30 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Warn("Error during server shutdown: %v", err)
		} else {
			logger.Info(gracefulShutdownMessage)
		}
	}
}
