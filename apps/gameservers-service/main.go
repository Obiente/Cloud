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

	gameserversvc "gameservers-service/internal/service"

	gameserverorchestrator "gameservers-service/internal/orchestrator"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/health"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/middleware"
	"github.com/obiente/cloud/apps/shared/pkg/redis"

	gameserversv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1/gameserversv1connect"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	_ "github.com/joho/godotenv/autoload"
)

const (
	readHeaderTimeout       = 10 * time.Second
	writeTimeout            = 60 * time.Second // Increased to allow log processing to complete
	idleTimeout             = 2 * time.Minute
	gracefulShutdownMessage = "shutting down server"
)

func main() {
	// Set log output and flags
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	// Initialize logger
	logger.Init()

	logger.Info("=== Game Servers Service Starting ===")
	logger.Debug("LOG_LEVEL: %s", os.Getenv("LOG_LEVEL"))

	database.RegisterModels(
		&database.GameServer{},
	)

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

	// Initialize Redis (for log streaming)
	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}
	if err := redis.InitRedis(redis.Config{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	}); err != nil {
		logger.Fatalf("failed to initialize Redis: %v", err)
	}
	logger.Info("✓ Redis initialized for log streaming")

	// Also initialize the old Redis cache (database.InitRedis) if needed for other features
	if err := database.InitRedis(); err != nil {
		logger.Warn("Redis cache initialization failed: %v. Some features may not work correctly.", err)
	} else {
		logger.Info("✓ Redis cache initialized")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3006"
	}

	// Create HTTP mux
	mux := http.NewServeMux()

	// Create auth configuration and interceptor
	authConfig := auth.NewAuthConfig()
	authInterceptor := auth.MiddlewareInterceptor(authConfig)

	// Create audit interceptor
	auditInterceptor := middleware.AuditLogInterceptor()

	// Initialize game server manager
	manager, err := gameserverorchestrator.NewGameServerManager("least-loaded", 50)
	if err != nil {
		logger.Warn("⚠️  Failed to create game server manager: %v", err)
		logger.Warn("⚠️  Game servers will not work until Docker is accessible")
		logger.Warn("⚠️  Please check Docker connection and ensure Docker daemon is running")
		manager = nil
	} else {
		logger.Info("✓ Created game server manager")
	}

	// Create repositories and services
	gameServerRepo := database.NewGameServerRepository(database.DB, database.RedisClient)
	gameServerService := gameserversvc.NewService(gameServerRepo, manager)

	// Register game servers service
	gameServersPath, gameServersHandler := gameserversv1connect.NewGameServerServiceHandler(
		gameServerService,
		connect.WithInterceptors(auditInterceptor, authInterceptor),
	)
	mux.Handle(gameServersPath, gameServersHandler)

	// HTTP endpoint for chunked file uploads (multipart streaming)
	mux.HandleFunc("/internal/gameservers/upload-file", gameServerService.HandleUploadFile)

	// WebSocket terminal endpoint (bypasses Connect RPC for direct access)
	mux.HandleFunc("/terminal/ws", gameServerService.HandleTerminalWebSocket)

	// Health check endpoint with replica ID
	mux.HandleFunc("/health", health.HandleHealth("gameservers-service", func() (bool, string, map[string]interface{}) {
		// Check database connection
		sqlDB, err := database.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			return false, "database unavailable", nil
		}
		return true, "healthy", nil
	}))

	// Root endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("gameservers-service"))
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

	// Start health monitor in a background goroutine
	// Check every 30 seconds to sync game server status with Docker container status
	healthMonitorCtx, healthMonitorCancel := context.WithCancel(shutdownCtx)
	defer healthMonitorCancel()
	go func() {
		gameServerService.StartHealthMonitor(healthMonitorCtx, 30*time.Second)
	}()

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("=== Game Servers Service Ready - Listening on %s ===", httpServer.Addr)
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
