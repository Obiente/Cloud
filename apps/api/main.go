package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	authv1connect "api/gen/proto/obiente/cloud/auth/v1/authv1connect"
	deploymentsv1connect "api/gen/proto/obiente/cloud/deployments/v1/deploymentsv1connect"
	organizationsv1connect "api/gen/proto/obiente/cloud/organizations/v1/organizationsv1connect"

	"github.com/moby/moby/client"
)

const (
	defaultPort             = "3001"
	readHeaderTimeout       = 10 * time.Second
	writeTimeout            = 30 * time.Second
	idleTimeout             = 2 * time.Minute
	gracefulShutdownMessage = "shutting down server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           newServeMux(),
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(context.Background(), client.ContainerListOptions{All: true})
	if err != nil {
		panic(err)
	}

	for _, ctr := range containers {
		fmt.Printf("%s %s\n", ctr.ID, ctr.Image)
	}

	log.Printf("Connect RPC API listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			log.Print(gracefulShutdownMessage)
			return
		}
		log.Fatalf("server failed: %v", err)
	}
}

func newServeMux() *http.ServeMux {
	mux := http.NewServeMux()

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

	authPath, authHandler := authv1connect.NewAuthServiceHandler(newAuthService())
	mux.Handle(authPath, authHandler)

	deploymentsPath, deploymentsHandler := deploymentsv1connect.NewDeploymentServiceHandler(newDeploymentService())
	mux.Handle(deploymentsPath, deploymentsHandler)

	organizationsPath, organizationsHandler := organizationsv1connect.NewOrganizationServiceHandler(newOrganizationService())
	mux.Handle(organizationsPath, organizationsHandler)

	return mux
}

type authService struct {
	authv1connect.UnimplementedAuthServiceHandler
}

func newAuthService() authv1connect.AuthServiceHandler {
	return &authService{}
}

type deploymentService struct {
	deploymentsv1connect.UnimplementedDeploymentServiceHandler
}

func newDeploymentService() deploymentsv1connect.DeploymentServiceHandler {
	return &deploymentService{}
}

type organizationService struct {
	organizationsv1connect.UnimplementedOrganizationServiceHandler
}

func newOrganizationService() organizationsv1connect.OrganizationServiceHandler {
	return &organizationService{}
}

// These compile-time assertions ensure that our service structs satisfy the generated handler
// interfaces. The blank identifier assignments make failures easy to spot during builds.
var (
	_ authv1connect.AuthServiceHandler                  = (*authService)(nil)
	_ deploymentsv1connect.DeploymentServiceHandler     = (*deploymentService)(nil)
	_ organizationsv1connect.OrganizationServiceHandler = (*organizationService)(nil)
)
