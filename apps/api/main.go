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
	"api/internal/email"
	"api/internal/logger"
	"api/internal/orchestrator"
	apisrv "api/internal/server"
	"api/internal/services/billing"
	orgsvc "api/internal/services/organizations"

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

	// Seed default plans if none exist
	if err := billing.SeedDefaultPlans(); err != nil {
		logger.Warn("Failed to seed default plans: %v", err)
	} else {
		logger.Info("✓ Default plans seeded (if needed)")
	}

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
	serverInfo := apisrv.New()
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           serverInfo.Handler,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	// Set up graceful shutdown
	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start monthly free credits grant service
	go startMonthlyCreditsService(shutdownCtx)
	logger.Info("✓ Monthly free credits service started")

	// Start quota warning service (checks daily for organizations approaching limits)
	go startQuotaWarningService(shutdownCtx)
	logger.Info("✓ Quota warning service started")

	// Start deployment health monitor (checks and redeploys deployments that should be running)
	if serverInfo.DeploymentService != nil {
		// Check interval: default 5 minutes, configurable via env var
		checkInterval := 5 * time.Minute
		if intervalStr := os.Getenv("DEPLOYMENT_HEALTH_CHECK_INTERVAL"); intervalStr != "" {
			if parsed, err := time.ParseDuration(intervalStr); err == nil {
				checkInterval = parsed
			}
		}
		go serverInfo.DeploymentService.StartHealthMonitor(shutdownCtx, checkInterval)
		logger.Info("✓ Deployment health monitor started (interval: %v)", checkInterval)
	} else {
		logger.Warn("⚠️  Deployment service not available, health monitor not started")
	}

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

// startMonthlyCreditsService starts a background service that grants monthly free credits
// It runs once per day and checks if credits need to be granted for the current month
// Credits are granted based on the tracking table, allowing recovery and new plan assignments
func startMonthlyCreditsService(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run immediately on startup to grant credits for any missing months
	logger.Info("[Monthly Credits] Checking for pending monthly credit grants...")
	if err := billing.GrantMonthlyFreeCredits(); err != nil {
		logger.Error("[Monthly Credits] Failed to grant monthly free credits: %v", err)
	}

	// Then check daily
	for {
		select {
		case <-ctx.Done():
			logger.Info("[Monthly Credits] Service shutting down")
			return
		case <-ticker.C:
			logger.Debug("[Monthly Credits] Daily check for pending monthly credit grants...")
			if err := billing.GrantMonthlyFreeCredits(); err != nil {
				logger.Error("[Monthly Credits] Failed to grant monthly free credits: %v", err)
			}
		}
	}
}

// startQuotaWarningService starts a background service that checks resource usage and sends email warnings
// It runs once per day to check all organizations for quota warnings
func startQuotaWarningService(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run immediately on startup
	logger.Info("[Quota Warnings] Checking for organizations approaching resource limits...")
	checkAllOrganizationsQuotas(ctx)

	// Then check daily
	for {
		select {
		case <-ctx.Done():
			logger.Info("[Quota Warnings] Service shutting down")
			return
		case <-ticker.C:
			logger.Debug("[Quota Warnings] Daily check for quota warnings...")
			checkAllOrganizationsQuotas(ctx)
		}
	}
}

func checkAllOrganizationsQuotas(ctx context.Context) {
	// Get all organizations with assigned plans
	var quotas []database.OrgQuota
	if err := database.DB.Where("plan_id != '' AND plan_id IS NOT NULL").Find(&quotas).Error; err != nil {
		logger.Error("[Quota Warnings] Failed to get quotas: %v", err)
		return
	}

	// Create a service instance for quota checking with email sender
	emailSender := email.NewSenderFromEnv()
	orgService := orgsvc.NewService(orgsvc.Config{
		EmailSender:  emailSender,
		ConsoleURL:   os.Getenv("CONSOLE_URL"),
		SupportEmail: os.Getenv("SUPPORT_EMAIL"),
	})

	// Type assert to access CheckAndNotifyQuotaWarnings
	if svc, ok := orgService.(*orgsvc.Service); ok {
		for _, quota := range quotas {
			if err := svc.CheckAndNotifyQuotaWarnings(ctx, quota.OrganizationID); err != nil {
				logger.Error("[Quota Warnings] Failed to check quota for org %s: %v", quota.OrganizationID, err)
			}
		}
	} else {
		logger.Warn("[Quota Warnings] Could not access service methods, skipping quota checks")
	}
}
