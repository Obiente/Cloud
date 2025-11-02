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

	log.Println("=== Obiente Cloud API Starting ===")
	log.Printf("LOG_LEVEL: %s", os.Getenv("LOG_LEVEL"))
	log.Printf("CORS_ORIGIN: %s", os.Getenv("CORS_ORIGIN"))

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	log.Println("✓ Database initialized")

	// Initialize Redis
	if err := database.InitRedis(); err != nil {
		log.Printf("Redis initialization failed: %v", err)
	} else {
		log.Println("✓ Redis initialized")
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
		log.Printf("⚠️  Failed to initialize orchestrator service: %v", err)
		log.Println("⚠️  Metrics collection will not be available")
	} else {
		log.Println("✓ Orchestrator service initialized")
		orchService.Start()
		log.Println("✓ Orchestrator service started (metrics collection, health checks, usage aggregation)")
		defer func() {
			if orchService != nil {
				orchService.Stop()
			}
		}()
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	log.Println("✓ Creating HTTP server with middleware...")
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
		log.Printf("=== Server Ready - Listening on %s ===", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Wait for interrupt or server error
	select {
	case err := <-serverErr:
		log.Fatalf("server failed: %v", err)
	case <-shutdownCtx.Done():
		log.Println("\n=== Shutting down gracefully ===")
		shutdownTimeout := 30 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
		} else {
			log.Print(gracefulShutdownMessage)
		}
	}
}
