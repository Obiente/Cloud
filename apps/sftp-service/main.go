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

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/health"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/middleware"
	"github.com/obiente/cloud/apps/shared/pkg/sftp"

	"sftp-service/internal/service"

	_ "github.com/joho/godotenv/autoload"
)

const (
	readHeaderTimeout       = 10 * time.Second
	writeTimeout            = 30 * time.Second
	idleTimeout             = 2 * time.Minute
	gracefulShutdownMessage = "shutting down server"
)

func main() {
	// Set log output and flags
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	// Initialize logger
	logger.Init()

	logger.Info("=== SFTP Service Starting ===")
	logger.Debug("LOG_LEVEL: %s", os.Getenv("LOG_LEVEL"))

	database.RegisterModels(&database.APIKey{})

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		logger.Fatalf("failed to initialize database: %v", err)
	}
	logger.Info("✓ Database initialized")

	// Initialize metrics database (TimescaleDB for audit logs)
	if err := database.InitMetricsDatabase(); err != nil {
		logger.Fatalf("failed to initialize metrics database: %v", err)
	}
	logger.Info("✓ Metrics database initialized")

	// Get configuration from environment
	sftpPort := os.Getenv("SFTP_PORT")
	if sftpPort == "" {
		sftpPort = "2222"
	}
	sftpAddress := "0.0.0.0:" + sftpPort

	basePath := os.Getenv("SFTP_BASE_PATH")
	if basePath == "" {
		basePath = "/var/lib/sftp"
	}

	hostKeyPath := os.Getenv("SFTP_HOST_KEY_PATH")
	if hostKeyPath == "" {
		hostKeyPath = "/var/lib/sftp/host_key"
	}

	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = "3020"
	}

	// Create auth validator and audit logger
	authValidator := service.NewAPIKeyValidator()
	auditLogger := service.NewSFTPAuditLogger()

	// Create SFTP server
	sftpServer, err := sftp.NewServer(&sftp.Config{
		Address:       sftpAddress,
		BasePath:      basePath,
		HostKeyPath:   hostKeyPath,
		AuthValidator: authValidator,
		AuditLogger:   auditLogger,
	})
	if err != nil {
		logger.Fatalf("failed to create SFTP server: %v", err)
	}
	logger.Info("✓ SFTP server initialized on %s", sftpAddress)

	// Start SFTP server in background
	sftpErrChan := make(chan error, 1)
	go func() {
		logger.Info("=== SFTP Server Starting on %s ===", sftpAddress)
		if err := sftpServer.Start(); err != nil {
			sftpErrChan <- err
		}
	}()

	// Create HTTP server for health checks
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", health.HandleHealth("sftp-service", func() (bool, string, map[string]interface{}) {
		// Check database connection
		sqlDB, err := database.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			return false, "database unavailable", nil
		}
		
		// Check metrics database connection
		metricsDB, err := database.MetricsDB.DB()
		if err != nil || metricsDB.Ping() != nil {
			return false, "metrics database unavailable", nil
		}
		
		return true, "healthy", map[string]interface{}{
			"sftp_address": sftpAddress,
			"base_path":    basePath,
		}
	}))

	// Root endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("sftp-service"))
	})

	// Apply middleware
	var handler http.Handler = mux
	handler = middleware.CORSHandler(handler)
	handler = middleware.RequestLogger(handler)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:              ":" + httpPort,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	// Set up graceful shutdown
	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start HTTP server in a goroutine
	httpErrChan := make(chan error, 1)
	go func() {
		logger.Info("=== HTTP Health Server Ready - Listening on %s ===", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			httpErrChan <- err
		}
	}()

	// Wait for interrupt signal or server error
	select {
	case err := <-sftpErrChan:
		logger.Fatalf("SFTP server failed: %v", err)
	case err := <-httpErrChan:
		logger.Fatalf("HTTP server failed: %v", err)
	case <-shutdownCtx.Done():
		logger.Info("\n=== Shutting down gracefully ===")
		
		// Shutdown HTTP server
		shutdownTimeout := 30 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			logger.Warn("Error during HTTP server shutdown: %v", err)
		} else {
			logger.Info("HTTP server shutdown complete")
		}

		// Shutdown SFTP server
		if err := sftpServer.Shutdown(); err != nil {
			logger.Warn("Error during SFTP server shutdown: %v", err)
		} else {
			logger.Info("SFTP server shutdown complete")
		}

		logger.Info(gracefulShutdownMessage)
	}
}
