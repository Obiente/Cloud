package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	adminv1connect "api/gen/proto/obiente/cloud/admin/v1/adminv1connect"
	auditv1connect "api/gen/proto/obiente/cloud/audit/v1/auditv1connect"
	authv1connect "api/gen/proto/obiente/cloud/auth/v1/authv1connect"
	billingv1connect "api/gen/proto/obiente/cloud/billing/v1/billingv1connect"
	deploymentsv1connect "api/gen/proto/obiente/cloud/deployments/v1/deploymentsv1connect"
	gameserversv1connect "api/gen/proto/obiente/cloud/gameservers/v1/gameserversv1connect"
	organizationsv1connect "api/gen/proto/obiente/cloud/organizations/v1/organizationsv1connect"
	superadminv1connect "api/gen/proto/obiente/cloud/superadmin/v1/superadminv1connect"
	supportv1connect "api/gen/proto/obiente/cloud/support/v1/supportv1connect"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/dnsdelegation"
	"api/internal/email"
	"api/internal/metrics"
	"api/internal/middleware"
	"api/internal/orchestrator"
	"api/internal/quota"
	adminsvc "api/internal/services/admin"
	auditsvc "api/internal/services/audit"
	authsvc "api/internal/services/auth"
	billingsvc "api/internal/services/billing"
	deploymentsvc "api/internal/services/deployments"
	gameserversvc "api/internal/services/gameservers"
	orgsvc "api/internal/services/organizations"
	superadminsvc "api/internal/services/superadmin"
	supportsvc "api/internal/services/support"
	"api/internal/stripe"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// ServerInfo contains the HTTP handler and deployment service
type ServerInfo struct {
	Handler          http.Handler
	DeploymentService *deploymentsvc.Service
}

// New constructs the primary Connect handler with all service registrations and reflection.
// It returns both the handler and the deployment service so it can be used for background tasks.
func New() *ServerInfo {
	log.Println("[Server] Registering routes...")
	mux := http.NewServeMux()
	registerRoot(mux)
	deploymentService := registerServices(mux)
	registerReflection(mux)

	log.Println("[Server] Wrapping with h2c for HTTP/2...")
	// Wrap with h2c for HTTP/2
	h2cHandler := h2c.NewHandler(mux, &http2.Server{})

	log.Println("[Server] Applying middleware stack...")
	// Wrap with middleware (order matters: metrics -> logging -> CORS -> handler)
	handler := h2cHandler
	handler = metrics.HTTPMetricsMiddleware(handler) // Prometheus metrics first
	handler = middleware.CORSHandler(handler)
	handler = middleware.CORSDebugLogger(handler)
	handler = middleware.RequestLogger(handler)

	log.Println("[Server] Handler chain complete")
	return &ServerInfo{
		Handler:          handler,
		DeploymentService: deploymentService,
	}
}

func registerRoot(mux *http.ServeMux) {
	// Root endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("obiente-cloud-api"))
	})

	// Health check endpoint (no auth required)
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

		// Check metrics streamer health if available
		streamer := orchestrator.GetGlobalMetricsStreamer()
		metricsHealthy := true
		if streamer != nil {
			healthy, failures := streamer.GetHealth()
			metricsHealthy = healthy
			if !healthy {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":"unhealthy","message":"metrics collection unhealthy","consecutive_failures":` + fmt.Sprintf("%d", failures) + `}`))
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","version":"1.0.0","metrics_healthy":` + fmt.Sprintf("%t", metricsHealthy) + `}`))
	})

	// Metrics observability endpoint (no auth required, useful for monitoring)
	mux.HandleFunc("/metrics/observability", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		streamer := orchestrator.GetGlobalMetricsStreamer()
		if streamer == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"error":"metrics streamer not available"}`))
			return
		}

		stats := streamer.GetStats()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		// Return JSON stats (simplified - in production use proper JSON marshaling)
		json := fmt.Sprintf(`{
			"collection_count": %d,
			"collection_errors": %d,
			"collections_per_second": %.2f,
			"containers_processed": %d,
			"containers_failed": %d,
			"storage_batches_written": %d,
			"storage_batches_failed": %d,
			"storage_metrics_written": %d,
			"storage_metrics_failed": %d,
			"retry_queue_size": %d,
			"retry_batches_processed": %d,
			"retry_batches_success": %d,
			"active_subscribers": %d,
			"slow_subscribers": %d,
			"subscriber_overflows": %d,
			"live_metrics_cache_size": %d,
			"previous_stats_cache_size": %d,
			"circuit_breaker_state": %d,
			"circuit_breaker_failures": %d,
			"healthy": %t,
			"consecutive_failures": %d,
			"last_collection_time": "%s",
			"last_storage_time": "%s",
			"last_health_check_time": "%s"
		}`,
			stats.CollectionCount,
			stats.CollectionErrors,
			stats.CollectionsPerSecond,
			stats.ContainersProcessed,
			stats.ContainersFailed,
			stats.StorageBatchesWritten,
			stats.StorageBatchesFailed,
			stats.StorageMetricsWritten,
			stats.StorageMetricsFailed,
			stats.RetryQueueSize,
			stats.RetryBatchesProcessed,
			stats.RetryBatchesSuccess,
			stats.ActiveSubscribers,
			stats.SlowSubscribers,
			stats.SubscriberOverflows,
			stats.LiveMetricsCacheSize,
			stats.PreviousStatsCacheSize,
			stats.CircuitBreakerState,
			stats.CircuitBreakerFailures,
			stats.IsHealthy,
			stats.ConsecutiveFailures,
			stats.LastCollectionTime.Format(time.RFC3339),
			stats.LastStorageTime.Format(time.RFC3339),
			stats.LastHealthCheckTime.Format(time.RFC3339),
		)
		_, _ = w.Write([]byte(json))
	})

	// Prometheus metrics endpoint (no auth required)
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		// Update Prometheus metrics from metrics streamer if available
		streamer := orchestrator.GetGlobalMetricsStreamer()
		if streamer != nil {
			stats := streamer.GetStats()
			metrics.UpdateMetricsFromStats(
				stats.CollectionCount,
				stats.CollectionErrors,
				stats.ContainersProcessed,
				stats.ContainersFailed,
				stats.StorageBatchesWritten,
				stats.StorageBatchesFailed,
				stats.RetryQueueSize,
				stats.ActiveSubscribers,
				stats.CircuitBreakerState,
				stats.IsHealthy,
			)
		}

		// Serve Prometheus metrics
		metrics.Handler().ServeHTTP(w, r)
	})
}

func registerServices(mux *http.ServeMux) *deploymentsvc.Service {
	// Create auth configuration with JWKS from Zitadel
	authConfig := auth.NewAuthConfig()

	// Configure email sender and shared links
	mailer := email.NewSenderFromEnv()
	consoleURL := firstNonEmpty(
		os.Getenv("CONSOLE_URL"),
		os.Getenv("DASHBOARD_URL"),
		os.Getenv("APP_CONSOLE_URL"),
	)
	supportEmail := os.Getenv("SUPPORT_EMAIL")

	// AutoMigrate new schemas (best-effort)
	if database.DB != nil {
		if err := database.DB.AutoMigrate(
			&database.OrganizationPlan{},
			&database.OrgQuota{},
			&database.OrgRole{},
			&database.OrgRoleBinding{},
			&database.Organization{},
			&database.OrganizationMember{},
			&database.GitHubIntegration{},
			&database.BuildHistory{},
			&database.BuildLog{},
			&database.BillingAccount{},
			&database.CreditTransaction{},
			&database.GameServer{},
			&database.GameServerUsageHourly{},
			&database.SupportTicket{},
			&database.TicketComment{},
			// Note: AuditLog is migrated in MetricsDB (TimescaleDB), not here
		); err != nil {
			log.Printf("[Server] AutoMigrate warning: %v", err)
		}
		if err := database.InitDeploymentTracking(); err != nil {
			log.Printf("[Server] InitDeploymentTracking warning: %v", err)
		}
	} else {
		log.Printf("[Server] Skipping AutoMigrate: database not initialized")
	}

	// Create auth interceptor for token validation
	// Note: Unary interceptors work for both unary and streaming RPCs in Connect
	authInterceptor := auth.MiddlewareInterceptor(authConfig)

	// Create audit log interceptor
	auditInterceptor := middleware.AuditLogInterceptor()

	// Configure services
	// Note: Login RPC does not require authentication (public endpoint)
	// Chain interceptors: In Connect, interceptors wrap from inside to outside.
	// With (auditInterceptor, authInterceptor), auth runs first (innermost), audit runs second (outermost).
	// This allows audit to see the user context set by auth.
	authPath, authHandler := authv1connect.NewAuthServiceHandler(
		authsvc.NewService(),
		// Login doesn't need auth interceptor, but other methods do
		// We'll handle this in the service itself
		connect.WithInterceptors(auditInterceptor, authInterceptor),
	)
	mux.Handle(authPath, authHandler)

	// Create deployment repository and service
	deploymentRepo := database.NewDeploymentRepository(database.DB, database.RedisClient)
	// Get orchestrator service and its deployment manager
	// The orchestrator service should be initialized in main.go before server is created
	orchService := orchestrator.GetGlobalOrchestratorService()
	var manager *orchestrator.DeploymentManager
	if orchService != nil {
		manager = orchService.GetDeploymentManager()
		if manager != nil {
			log.Println("[Server] Using deployment manager from orchestrator service")
		} else {
			log.Printf("[Server] Warning: Orchestrator service exists but deployment manager is nil")
		}
	}
	
	// Fallback: create a deployment manager directly if orchestrator service is not available or manager is nil
	if manager == nil {
		log.Println("[Server] Attempting to create deployment manager as fallback...")
		var err error
		manager, err = orchestrator.NewDeploymentManager("least-loaded", 50)
		if err != nil {
			log.Printf("[Server] ❌ CRITICAL: Failed to create deployment manager: %v", err)
			log.Printf("[Server] ❌ Deployments will not work until Docker is accessible and orchestrator is initialized")
			log.Printf("[Server] ❌ Please check Docker connection and ensure Docker daemon is running")
			manager = nil
		} else {
			log.Println("[Server] ✓ Created deployment manager directly (orchestrator service not available or manager was nil)")
		}
	}
	qc := quota.NewChecker()
	deploymentService := deploymentsvc.NewService(deploymentRepo, manager, qc)
	deploymentsPath, deploymentsHandler := deploymentsv1connect.NewDeploymentServiceHandler(
		deploymentService,
		connect.WithInterceptors(auditInterceptor, authInterceptor),
	)
	mux.Handle(deploymentsPath, deploymentsHandler)

	// WebSocket terminal endpoint (bypasses Connect RPC for direct access)
	mux.HandleFunc("/terminal/ws", deploymentService.HandleTerminalWebSocket)

	// Organization service with auth
	organizationsPath, organizationsHandler := organizationsv1connect.NewOrganizationServiceHandler(
		orgsvc.NewService(orgsvc.Config{
			EmailSender:  mailer,
			ConsoleURL:   consoleURL,
			SupportEmail: supportEmail,
		}),
		connect.WithInterceptors(auditInterceptor, authInterceptor),
	)
	mux.Handle(organizationsPath, organizationsHandler)

	// Superadmin service
	superadminPath, superadminHandler := superadminv1connect.NewSuperadminServiceHandler(
		superadminsvc.NewService(),
		connect.WithInterceptors(auditInterceptor, authInterceptor),
	)
	mux.Handle(superadminPath, superadminHandler)

	// Admin Connect service
	adminPath, adminHandler := adminv1connect.NewAdminServiceHandler(
		adminsvc.NewService(),
		connect.WithInterceptors(auditInterceptor, authInterceptor),
	)
	mux.Handle(adminPath, adminHandler)

	// Billing service (always register, but Stripe features require STRIPE_SECRET_KEY)
	stripeClient, err := stripe.NewClient()
	if err != nil {
		log.Printf("[Server] ⚠️  Warning: Stripe client initialization failed: %v", err)
		log.Printf("[Server] ⚠️  Billing features will return errors. Set STRIPE_SECRET_KEY to enable.")
		stripeClient = nil
	}

	// Always register billing service (it will return appropriate errors if Stripe is not configured or billing is disabled)
	billingEnabled := os.Getenv("BILLING_ENABLED") != "false" && os.Getenv("BILLING_ENABLED") != "0"
	billingService := billingsvc.NewService(stripeClient, consoleURL, billingEnabled)
	billingPath, billingHandler := billingv1connect.NewBillingServiceHandler(
		billingService,
		connect.WithInterceptors(auditInterceptor, authInterceptor),
	)
	mux.Handle(billingPath, billingHandler)

	// Game Server service with auth
	gameServerRepo := database.NewGameServerRepository(database.DB, database.RedisClient)
	gameServerService := gameserversvc.NewService(gameServerRepo)
	gameServersPath, gameServersHandler := gameserversv1connect.NewGameServerServiceHandler(
		gameServerService,
		connect.WithInterceptors(auditInterceptor, authInterceptor),
	)
	mux.Handle(gameServersPath, gameServersHandler)

	// WebSocket terminal endpoint for game servers (bypasses Connect RPC for direct access)
	mux.HandleFunc("/gameservers/terminal/ws", gameServerService.HandleTerminalWebSocket)

	// Support service with auth
	supportService := supportsvc.NewService(database.DB)
	supportPath, supportHandler := supportv1connect.NewSupportServiceHandler(
		supportService,
		connect.WithInterceptors(auditInterceptor, authInterceptor),
	)
	mux.Handle(supportPath, supportHandler)

	// Audit service with auth
	auditService := auditsvc.NewService(database.DB)
	auditPath, auditHandler := auditv1connect.NewAuditServiceHandler(
		auditService,
		connect.WithInterceptors(auditInterceptor, authInterceptor),
	)
	mux.Handle(auditPath, auditHandler)

	// Stripe webhook endpoint (no auth required, uses webhook signature verification)
	// Only register webhook handler if Stripe is configured
	if stripeClient != nil {
		mux.HandleFunc("/webhooks/stripe", billingsvc.HandleStripeWebhook)
		log.Println("[Server] ✓ Billing service registered with Stripe webhook support")
	} else {
		log.Println("[Server] ⚠️  Billing service registered but Stripe not configured (webhook disabled)")
	}

	// DNS delegation push endpoints (public, API key authenticated)
	// Allows dev/self-hosted APIs to push DNS records to production DNS
	mux.HandleFunc("/dns/push", dnsdelegation.HandlePushDNSRecord)
	mux.HandleFunc("/dns/push/batch", dnsdelegation.HandlePushDNSRecords)
	log.Println("[Server] ✓ DNS delegation push endpoints registered at /dns/push and /dns/push/batch")

	return deploymentService
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func registerReflection(mux *http.ServeMux) {
	reflector := grpcreflect.NewStaticReflector(
		authv1connect.AuthServiceName,
		deploymentsv1connect.DeploymentServiceName,
		gameserversv1connect.GameServerServiceName,
		organizationsv1connect.OrganizationServiceName,
		billingv1connect.BillingServiceName,
		supportv1connect.SupportServiceName,
	)

	grpcPath, grpcHandler := grpcreflect.NewHandlerV1(reflector)
	mux.Handle(grpcPath, grpcHandler)

	grpcAlphaPath, grpcAlphaHandler := grpcreflect.NewHandlerV1Alpha(reflector)
	mux.Handle(grpcAlphaPath, grpcAlphaHandler)
}
