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
	superadminsvc "superadmin-service/internal/service"

	superadminv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/superadmin/v1/superadminv1connect"

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

	logger.Info("=== Superadmin Service Starting ===")
	logger.Debug("LOG_LEVEL: %s", os.Getenv("LOG_LEVEL"))

	database.RegisterModels(
		&database.Organization{},
		&database.OrganizationMember{},
		&database.OrgRole{},
		&database.OrgRoleBinding{},
		&database.BillingAccount{},
		&database.CreditTransaction{},
		&database.StripeWebhookEvent{},
		&database.SupportTicket{},
		&database.TicketComment{},
		&database.GameServer{},
		&database.GitHubIntegration{},
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "3011"
	}

	// Create HTTP mux
	mux := http.NewServeMux()

	// Create auth configuration and interceptor
	authConfig := auth.NewAuthConfig()
	authInterceptor := auth.MiddlewareInterceptor(authConfig)

	// Create audit interceptor
	auditInterceptor := middleware.AuditLogInterceptor()

	// Register superadmin service
	superadminService := superadminsvc.NewService()
	superadminPath, superadminHandler := superadminv1connect.NewSuperadminServiceHandler(
		superadminService,
		connect.WithInterceptors(auditInterceptor, authInterceptor),
	)
	mux.Handle(superadminPath, superadminHandler)

	// Health check endpoint with replica ID
	mux.HandleFunc("/health", health.HandleHealth("superadmin-service", func() (bool, string, map[string]interface{}) {
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
		_, _ = w.Write([]byte("superadmin-service"))
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

	// Start periodic abuse detection (runs every hour and on startup after 1 minute)
	go startAbuseDetectionService(shutdownCtx)
	logger.Info("✓ Abuse detection service started")

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("=== Superadmin Service Ready - Listening on %s ===", httpServer.Addr)
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

// startAbuseDetectionService runs abuse detection periodically and on startup
// This ensures abuse is detected even if no one is actively monitoring the dashboard
func startAbuseDetectionService(ctx context.Context) {
	// Run once on startup after a short delay to ensure DB is ready
	time.Sleep(1 * time.Minute)
	logger.Info("[AbuseDetection] Running initial abuse detection on startup...")
	runAbuseDetection(ctx)

	// Then run every hour
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("[AbuseDetection] Abuse detection service stopped")
			return
		case <-ticker.C:
			logger.Info("[AbuseDetection] Running periodic abuse detection...")
			runAbuseDetection(ctx)
		}
	}
}

// runAbuseDetection executes the abuse detection and handles errors
// DetectAbuse will automatically send notifications if abuse is found
func runAbuseDetection(ctx context.Context) {
	// Use background context with timeout to avoid blocking
	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	logger.Info("[AbuseDetection] Starting abuse detection scan...")
	
	// Call DetectAbuse directly from the superadmin package
	// It will automatically send notifications to superadmins if abuse is detected
	result, err := superadminsvc.DetectAbuse(bgCtx)
	if err != nil {
		logger.Warn("[AbuseDetection] Failed to run abuse detection: %v", err)
	} else {
		totalOrgs := len(result.SuspiciousOrganizations)
		totalActivities := len(result.SuspiciousActivities)
		logger.Info("[AbuseDetection] Abuse detection completed: %d suspicious orgs, %d suspicious activities", totalOrgs, totalActivities)
		
		if totalOrgs > 0 || totalActivities > 0 {
			logger.Info("[AbuseDetection] Abuse detected! Triggering notification to superadmins (in background goroutine).")
		}
	}
}
