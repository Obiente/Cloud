package server

import (
	"log"
	"net/http"
	"os"
	"strings"

	adminv1connect "api/gen/proto/obiente/cloud/admin/v1/adminv1connect"
	authv1connect "api/gen/proto/obiente/cloud/auth/v1/authv1connect"
	deploymentsv1connect "api/gen/proto/obiente/cloud/deployments/v1/deploymentsv1connect"
	organizationsv1connect "api/gen/proto/obiente/cloud/organizations/v1/organizationsv1connect"
	superadminv1connect "api/gen/proto/obiente/cloud/superadmin/v1/superadminv1connect"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/email"
	"api/internal/middleware"
	"api/internal/orchestrator"
	"api/internal/quota"
	adminsvc "api/internal/services/admin"
	authsvc "api/internal/services/auth"
	deploymentsvc "api/internal/services/deployments"
	orgsvc "api/internal/services/organizations"
	superadminsvc "api/internal/services/superadmin"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// New constructs the primary Connect handler with all service registrations and reflection.
func New() http.Handler {
	log.Println("[Server] Registering routes...")
	mux := http.NewServeMux()
	registerRoot(mux)
	registerServices(mux)
	registerReflection(mux)

	log.Println("[Server] Wrapping with h2c for HTTP/2...")
	// Wrap with h2c for HTTP/2
	h2cHandler := h2c.NewHandler(mux, &http2.Server{})

	log.Println("[Server] Applying middleware stack...")
	// Wrap with middleware (order matters: logging -> CORS -> handler)
	handler := h2cHandler
	handler = middleware.CORSHandler(handler)
	handler = middleware.CORSDebugLogger(handler)
	handler = middleware.RequestLogger(handler)

	log.Println("[Server] Handler chain complete")
	return handler
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

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","version":"1.0.0"}`))
	})
}

func registerServices(mux *http.ServeMux) {
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

	// Configure services
	authPath, authHandler := authv1connect.NewAuthServiceHandler(
		authsvc.NewService(),
		connect.WithInterceptors(authInterceptor),
	)
	mux.Handle(authPath, authHandler)

	// Create deployment repository and service
	deploymentRepo := database.NewDeploymentRepository(database.DB, database.RedisClient)
	// Orchestrator dependencies
	manager, err := orchestrator.NewDeploymentManager("least-loaded", 50)
	if err != nil {
		log.Printf("[Server] Failed to init deployment manager: %v", err)
		manager = nil
	}
	qc := quota.NewChecker()
	deploymentService := deploymentsvc.NewService(deploymentRepo, manager, qc)
	deploymentsPath, deploymentsHandler := deploymentsv1connect.NewDeploymentServiceHandler(
		deploymentService,
		connect.WithInterceptors(authInterceptor),
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
		connect.WithInterceptors(authInterceptor),
	)
	mux.Handle(organizationsPath, organizationsHandler)

	// Superadmin service
	superadminPath, superadminHandler := superadminv1connect.NewSuperadminServiceHandler(
		superadminsvc.NewService(),
		connect.WithInterceptors(authInterceptor),
	)
	mux.Handle(superadminPath, superadminHandler)

	// Admin Connect service
	adminPath, adminHandler := adminv1connect.NewAdminServiceHandler(
		adminsvc.NewService(),
		connect.WithInterceptors(authInterceptor),
	)
	mux.Handle(adminPath, adminHandler)

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
		organizationsv1connect.OrganizationServiceName,
	)

	grpcPath, grpcHandler := grpcreflect.NewHandlerV1(reflector)
	mux.Handle(grpcPath, grpcHandler)

	grpcAlphaPath, grpcAlphaHandler := grpcreflect.NewHandlerV1Alpha(reflector)
	mux.Handle(grpcAlphaPath, grpcAlphaHandler)
}
