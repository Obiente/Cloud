package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	databasessvc "databases-service/internal/service"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/health"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/middleware"

	databasesv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1/databasesv1connect"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	_ "github.com/joho/godotenv/autoload"
)

const (
	readHeaderTimeout       = 10 * time.Second
	writeTimeout            = 60 * time.Second
	idleTimeout             = 2 * time.Minute
	gracefulShutdownMessage = "shutting down server"
)

func main() {
	// Set log output and flags
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	// Initialize logger
	logger.Init()

	logger.Info("=== Databases Service Starting ===")
	logger.Debug("LOG_LEVEL: %s", os.Getenv("LOG_LEVEL"))

	database.RegisterModels(
		&database.DatabaseInstance{},
		&database.DatabaseConnection{},
		&database.DatabaseBackup{},
	)

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		logger.Fatalf("failed to initialize database: %v", err)
	}
	logger.Info("✓ Database initialized")

	// Initialize Redis (for caching)
	if err := database.InitRedis(); err != nil {
		logger.Warn("Redis initialization failed: %v. Some features may not work correctly.", err)
	} else {
		logger.Info("✓ Redis initialized")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3014"
	}

	// Create HTTP mux
	mux := http.NewServeMux()

	// Create auth configuration and interceptor
	authConfig := auth.NewAuthConfig()
	authInterceptor := auth.MiddlewareInterceptor(authConfig)

	// Create audit interceptor
	auditInterceptor := middleware.AuditLogInterceptor()

	// Create repositories and services
	databaseRepo := database.NewDatabaseRepository(database.DB, database.RedisClient)
	connRepo := database.NewDatabaseConnectionRepository(database.DB)
	backupRepo := database.NewDatabaseBackupRepository(database.DB)
	databaseService := databasessvc.NewService(databaseRepo, connRepo, backupRepo)

	// Register databases service
	databasesPath, databasesHandler := databasesv1connect.NewDatabaseServiceHandler(
		databaseService,
		connect.WithInterceptors(auditInterceptor, authInterceptor),
	)
	mux.Handle(databasesPath, databasesHandler)

	// Load existing routes and start proxy
	proxyServer := databaseService.GetProxy()
	routeRegistry := databaseService.GetRouteRegistry()

	loadCtx, loadCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := routeRegistry.LoadFromDatabase(loadCtx); err != nil {
		logger.Warn("Failed to load routes from database: %v", err)
	} else {
		logger.Info("✓ Routes loaded from database (%d routes)", routeRegistry.RouteCount())
	}
	loadCancel()

	// Start proxy in background
	go func() {
		if err := proxyServer.Start(context.Background()); err != nil {
			logger.Error("Failed to start proxy: %v", err)
		}
	}()
	logger.Info("✓ Database proxy starting")

	// Health check endpoint
	mux.HandleFunc("/health", health.HandleHealth("databases-service", func() (bool, string, map[string]interface{}) {
		extra := map[string]interface{}{
			"proxy_running": proxyServer.Healthy(),
			"routes":        routeRegistry.RouteCount(),
		}

		databaseReachable := false
		if database.DB != nil {
			sqlDB, err := database.DB.DB()
			if err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				databaseReachable = sqlDB.PingContext(ctx) == nil
				cancel()
			}
		}
		extra["database_reachable"] = databaseReachable

		if !proxyServer.Healthy() {
			return false, "proxy unavailable", extra
		}

		// Keep liveness healthy while the proxy is running, even if metadata DB is transiently unavailable.
		if !databaseReachable {
			return true, "proxy healthy (database degraded)", extra
		}

		return true, "healthy", extra
	}))

	// Proxy health endpoint
	mux.HandleFunc("/health/proxy", func(w http.ResponseWriter, r *http.Request) {
		if proxyServer.Healthy() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy","routes":` + fmt.Sprintf("%d", routeRegistry.RouteCount()) + `}`))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy"}`))
		}
	})

	// Root endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("databases-service"))
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
		logger.Info("=== Databases Service Ready - Listening on %s ===", httpServer.Addr)
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

		// Stop proxy first
		proxyServer.Stop()
		logger.Info("✓ Proxy stopped")

		if err := httpServer.Shutdown(ctx); err != nil {
			logger.Warn("Error during server shutdown: %v", err)
		} else {
			logger.Info(gracefulShutdownMessage)
		}
	}
}
