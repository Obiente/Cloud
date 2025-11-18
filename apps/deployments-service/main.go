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

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/health"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/middleware"
	"github.com/obiente/cloud/apps/shared/pkg/orchestrator"
	"github.com/obiente/cloud/apps/shared/pkg/quota"
	deploymentsvc "deployments-service/internal/service"

	deploymentsv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1/deploymentsv1connect"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

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

	logger.Info("=== Deployments Service Starting ===")
	logger.Debug("LOG_LEVEL: %s", os.Getenv("LOG_LEVEL"))

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		logger.Fatalf("failed to initialize database: %v", err)
	}
	logger.Info("✓ Database initialized")

	// Initialize metrics database (TimescaleDB for metrics)
	if err := database.InitMetricsDatabase(); err != nil {
		logger.Warn("Metrics database initialization failed: %v. Metrics may not work correctly.", err)
	} else {
		logger.Info("✓ Metrics database initialized")
	}

	// Initialize Redis (for build logs, etc.)
	if err := database.InitRedis(); err != nil {
		logger.Warn("Redis initialization failed: %v. Some features may not work correctly.", err)
	} else {
		logger.Info("✓ Redis initialized")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3005"
	}

	// Create HTTP mux
	mux := http.NewServeMux()

	// Create auth configuration and interceptor
	authConfig := auth.NewAuthConfig()
	authInterceptor := auth.MiddlewareInterceptor(authConfig)

	// Create audit interceptor
	auditInterceptor := middleware.AuditLogInterceptor()

	// Initialize orchestrator service for deployment management
	// Try to get from global orchestrator service first
	var manager *orchestrator.DeploymentManager
	orchService := orchestrator.GetGlobalOrchestratorService()
	if orchService != nil {
		manager = orchService.GetDeploymentManager()
		logger.Info("✓ Got deployment manager from global orchestrator service")
	}

	// Fallback: create a deployment manager directly if orchestrator service is not available
	if manager == nil {
		logger.Warn("⚠️  Orchestrator service not available, attempting to create deployment manager directly...")
		var err error
		manager, err = orchestrator.NewDeploymentManager("least-loaded", 50)
		if err != nil {
			logger.Warn("⚠️  Failed to create deployment manager: %v", err)
			logger.Warn("⚠️  Deployments will not work until Docker is accessible")
			logger.Warn("⚠️  Please check Docker connection and ensure Docker daemon is running")
			manager = nil
		} else {
			logger.Info("✓ Created deployment manager directly")
		}
	}

	// Create repositories and services
	deploymentRepo := database.NewDeploymentRepository(database.DB, database.RedisClient)
	qc := quota.NewChecker()
	deploymentService := deploymentsvc.NewService(deploymentRepo, manager, qc)

	// Register deployments service
	deploymentsPath, deploymentsHandler := deploymentsv1connect.NewDeploymentServiceHandler(
		deploymentService,
		connect.WithInterceptors(auditInterceptor, authInterceptor),
	)
	mux.Handle(deploymentsPath, deploymentsHandler)

	// WebSocket terminal endpoint (bypasses Connect RPC for direct access)
	mux.HandleFunc("/terminal/ws", deploymentService.HandleTerminalWebSocket)

	// Health check endpoint with replica ID
	mux.HandleFunc("/health", health.HandleHealth("deployments-service", func() (bool, string, map[string]interface{}) {
		// Check database connection
		sqlDB, err := database.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			return false, "database unavailable", nil
		}

		// Check if orchestrator manager is available
		extra := make(map[string]interface{})
		if manager == nil {
			extra["orchestrator"] = "unavailable"
			return false, "orchestrator manager unavailable", extra
		}
		extra["orchestrator"] = "available"
		return true, "healthy", extra
	}))

	// Root endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("deployments-service"))
	})

	// Wrap with h2c for HTTP/2
	h2cHandler := h2c.NewHandler(mux, &http2.Server{})

	// Apply middleware
	var handler http.Handler = h2cHandler
	handler = middleware.CORSHandler(handler)
	handler = middleware.RequestLogger(handler)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
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
		logger.Info("=== Deployments Service Ready - Listening on %s ===", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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

		if err := httpServer.Shutdown(ctx); err != nil {
			logger.Warn("Error during server shutdown: %v", err)
		} else {
			logger.Info(gracefulShutdownMessage)
		}
	}
}
