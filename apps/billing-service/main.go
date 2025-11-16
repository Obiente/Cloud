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
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/middleware"
	"billing-service/internal/service"
	"github.com/obiente/cloud/apps/shared/pkg/stripe"

	billingv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/billing/v1/billingv1connect"

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

	logger.Info("=== Billing Service Starting ===")
	logger.Debug("LOG_LEVEL: %s", os.Getenv("LOG_LEVEL"))

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		logger.Fatalf("failed to initialize database: %v", err)
	}
	logger.Info("✓ Database initialized")

	// Initialize metrics database (TimescaleDB for usage stats)
	if err := database.InitMetricsDatabase(); err != nil {
		logger.Warn("Metrics database initialization failed: %v. Usage stats may not work correctly.", err)
	} else {
		logger.Info("✓ Metrics database initialized")
	}

	// Seed default plans if none exist
	if err := billing.SeedDefaultPlans(); err != nil {
		logger.Warn("Failed to seed default plans: %v", err)
	} else {
		logger.Info("✓ Default plans seeded (if needed)")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3004"
	}

	// Create HTTP mux
	mux := http.NewServeMux()

	// Create auth configuration and interceptor
	authConfig := auth.NewAuthConfig()
	authInterceptor := auth.MiddlewareInterceptor(authConfig)

	// Create audit interceptor
	auditInterceptor := middleware.AuditLogInterceptor()

	// Configure Stripe client
	stripeClient, err := stripe.NewClient()
	if err != nil {
		logger.Warn("⚠️  Warning: Stripe client initialization failed: %v", err)
		logger.Warn("⚠️  Billing features will return errors. Set STRIPE_SECRET_KEY to enable.")
		stripeClient = nil
	}

	// Get console URL
	consoleURL := os.Getenv("DASHBOARD_URL")
	if consoleURL == "" {
		consoleURL = "https://obiente.cloud"
	}

	// Check if billing is enabled
	billingEnabled := os.Getenv("BILLING_ENABLED") != "false" && os.Getenv("BILLING_ENABLED") != "0"

	// Register billing service
	billingService := billing.NewService(stripeClient, consoleURL, billingEnabled)
	billingPath, billingHandler := billingv1connect.NewBillingServiceHandler(
		billingService,
		connect.WithInterceptors(auditInterceptor, authInterceptor),
	)
	mux.Handle(billingPath, billingHandler)

	// Register Stripe webhook endpoint (no auth required, uses signature verification)
	mux.HandleFunc("/webhooks/stripe", billing.HandleStripeWebhook)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		// Check database connection
		sqlDB, err := database.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"unhealthy","message":"database unavailable"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"billing-service"}`))
	})

	// Root endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("billing-service"))
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

	// Start monthly billing background service
	go startMonthlyBillingService(shutdownCtx)
	logger.Info("✓ Monthly billing service started")

	// Start monthly credits background service
	go startMonthlyCreditsService(shutdownCtx)
	logger.Info("✓ Monthly free credits service started")

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("=== Billing Service Ready - Listening on %s ===", httpServer.Addr)
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

// startMonthlyBillingService starts the monthly billing background service
func startMonthlyBillingService(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run once on startup (after a short delay to ensure DB is ready)
	time.Sleep(5 * time.Second)
	if err := billing.ProcessMonthlyBilling(); err != nil {
		logger.Warn("Monthly billing process error: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			logger.Info("Monthly billing service stopped")
			return
		case <-ticker.C:
			if err := billing.ProcessMonthlyBilling(); err != nil {
				logger.Warn("Monthly billing process error: %v", err)
			}
		}
	}
}

// startMonthlyCreditsService starts the monthly free credits background service
func startMonthlyCreditsService(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run once on startup (after a short delay to ensure DB is ready)
	time.Sleep(5 * time.Second)
	if err := billing.GrantMonthlyFreeCredits(); err != nil {
		logger.Warn("Monthly credits process error: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			logger.Info("Monthly credits service stopped")
			return
		case <-ticker.C:
			if err := billing.GrantMonthlyFreeCredits(); err != nil {
				logger.Warn("Monthly credits process error: %v", err)
			}
		}
	}
}

