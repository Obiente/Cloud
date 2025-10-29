package server

import (
	"net/http"

	authv1connect "api/gen/proto/obiente/cloud/auth/v1/authv1connect"
	deploymentsv1connect "api/gen/proto/obiente/cloud/deployments/v1/deploymentsv1connect"
	organizationsv1connect "api/gen/proto/obiente/cloud/organizations/v1/organizationsv1connect"
	"api/internal/auth"
	"api/internal/database"
	authsvc "api/internal/services/auth"
	deploymentsvc "api/internal/services/deployments"
	orgsvc "api/internal/services/organizations"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// New constructs the primary Connect handler with all service registrations and reflection.
func New() http.Handler {
	mux := http.NewServeMux()
	registerRoot(mux)
	registerServices(mux)
	registerReflection(mux)

	return h2c.NewHandler(mux, &http2.Server{})
}

func registerRoot(mux *http.ServeMux) {
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
}

func registerServices(mux *http.ServeMux) {
	// Create auth configuration with JWKS from Zitadel
	authConfig := auth.NewAuthConfig()
	
	// Create auth interceptor for JWT validation
	authInterceptor := auth.MiddlewareInterceptor(authConfig)
	
	// Configure services with authentication middleware
	authPath, authHandler := authv1connect.NewAuthServiceHandler(
		authsvc.NewService(),
		connect.WithInterceptors(authInterceptor),
	)
	mux.Handle(authPath, authHandler)

	// Create deployment repository and service
	deploymentRepo := database.NewDeploymentRepository(database.DB, database.RedisClient)
	deploymentService := deploymentsvc.NewService(deploymentRepo)
	deploymentsPath, deploymentsHandler := deploymentsv1connect.NewDeploymentServiceHandler(
		deploymentService,
		connect.WithInterceptors(authInterceptor),
	)
	mux.Handle(deploymentsPath, deploymentsHandler)

	// Organization service with auth
	organizationsPath, organizationsHandler := organizationsv1connect.NewOrganizationServiceHandler(
		orgsvc.NewService(),
		connect.WithInterceptors(authInterceptor),
	)
	mux.Handle(organizationsPath, organizationsHandler)
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
