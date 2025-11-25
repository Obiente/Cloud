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

	notificationsservice "notifications-service/internal/service"

	notificationsv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1/notificationsv1connect"

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

	logger.Info("=== Notifications Service Starting ===")
	logger.Debug("LOG_LEVEL: %s", os.Getenv("LOG_LEVEL"))

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		logger.Fatalf("failed to initialize database: %v", err)
	}
	logger.Info("✓ Database initialized")

	port := os.Getenv("PORT")
	if port == "" {
		port = "3012"
	}

	// Create HTTP mux
	mux := http.NewServeMux()

	// Create auth configuration and interceptor
	authConfig := auth.NewAuthConfig()
	authInterceptor := auth.MiddlewareInterceptor(authConfig)

	// Create audit interceptor
	auditInterceptor := middleware.AuditLogInterceptor()

	// Register notifications service
	notificationsService := notificationsservice.NewService()
	notificationsPath, notificationsHandler := notificationsv1connect.NewNotificationServiceHandler(
		notificationsService,
		connect.WithInterceptors(auditInterceptor, authInterceptor),
	)
	mux.Handle(notificationsPath, notificationsHandler)

	// Health check endpoint
	mux.HandleFunc("/health", health.HandleHealth("notifications-service", func() (bool, string, map[string]interface{}) {
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
		_, _ = w.Write([]byte("notifications-service"))
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
		logger.Info("=== Notifications Service Ready - Listening on %s ===", httpServer.Addr)
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
