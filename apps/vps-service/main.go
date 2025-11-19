package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/health"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/middleware"
	vpsorch "vps-service/orchestrator"
	"github.com/obiente/cloud/apps/shared/pkg/quota"
	vpssvc "vps-service/internal/service"

	vpsv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1/vpsv1connect"

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

	logger.Info("=== VPS Service Starting ===")
	logger.Debug("LOG_LEVEL: %s", os.Getenv("LOG_LEVEL"))

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		logger.Fatalf("failed to initialize database: %v", err)
	}
	logger.Info("✓ Database initialized")

	port := os.Getenv("PORT")
	if port == "" {
		port = "3008"
	}

	// Create HTTP mux
	mux := http.NewServeMux()

	// Create auth configuration and interceptor
	authConfig := auth.NewAuthConfig()
	authInterceptor := auth.MiddlewareInterceptor(authConfig)

	// Create audit interceptor
	auditInterceptor := middleware.AuditLogInterceptor()

	// Initialize VPS manager
	// Create VPS manager directly (orchestrator service doesn't manage VPS manager)
	var vpsManager *vpsorch.VPSManager
	var err error
	vpsManager, err = vpsorch.NewVPSManager()
	if err != nil {
		logger.Warn("⚠️  Failed to create VPS manager: %v", err)
		logger.Warn("⚠️  VPS operations will not work until Proxmox is configured")
		vpsManager = nil
	} else {
		logger.Info("✓ Created VPS manager")
	}

	// Create services
	qc := quota.NewChecker()
	vpsService := vpssvc.NewService(vpsManager, qc)

	// Register VPS service
	vpsPath, vpsHandler := vpsv1connect.NewVPSServiceHandler(
		vpsService,
		connect.WithInterceptors(auditInterceptor, authInterceptor),
	)
	mux.Handle(vpsPath, vpsHandler)

	// VPS Config service (for cloud-init and user management)
	vpsConfigService := vpssvc.NewConfigService(vpsManager)
	vpsConfigPath, vpsConfigHandler := vpsv1connect.NewVPSConfigServiceHandler(
		vpsConfigService,
		connect.WithInterceptors(auditInterceptor, authInterceptor),
	)
	mux.Handle(vpsConfigPath, vpsConfigHandler)

	// VPS terminal WebSocket endpoint
	// Route pattern: /vps/{vps_id}/terminal/ws
	mux.HandleFunc("/vps/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/terminal/ws") {
			vpsService.HandleVPSTerminalWebSocket(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	// Start SSH proxy server for VPS access (users connect via SSH to API server)
	var sshProxyServer *vpssvc.SSHProxyServer
	sshProxyPort := 2222
	if portStr := os.Getenv("SSH_PROXY_PORT"); portStr != "" {
		if port, parseErr := strconv.Atoi(portStr); parseErr == nil {
			sshProxyPort = port
		}
	}
	sshProxyServer, err = vpssvc.NewSSHProxyServer(sshProxyPort, vpsService)
	if err != nil {
		logger.Warn("⚠️  Failed to create SSH proxy server: %v", err)
		logger.Warn("⚠️  SSH proxy will not be available")
		sshProxyServer = nil
	} else {
		go func() {
			ctx := context.Background()
			if err := sshProxyServer.Start(ctx); err != nil {
				logger.Error("SSH proxy server error: %v", err)
			}
		}()
		logger.Info("✓ SSH proxy server started on port %d", sshProxyPort)
	}

	// Health check endpoint with replica ID
	mux.HandleFunc("/health", health.HandleHealth("vps-service", func() (bool, string, map[string]interface{}) {
		// Check database connection
		sqlDB, err := database.DB.DB()
		if err != nil || sqlDB.Ping() != nil {
			return false, "database unavailable", nil
		}

		// Check if VPS manager is available
		extra := make(map[string]interface{})
		if vpsManager == nil {
			extra["vps_manager"] = "unavailable"
			return false, "VPS manager unavailable", extra
		}
		extra["vps_manager"] = "available"
		return true, "healthy", extra
	}))

	// Root endpoint
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("vps-service"))
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
		logger.Info("=== VPS Service Ready - Listening on %s ===", httpServer.Addr)
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

		// Stop SSH proxy server if running
		if sshProxyServer != nil {
			logger.Info("Stopping SSH proxy server...")
			sshProxyServer.Stop(10 * time.Second)
		}

		if err := httpServer.Shutdown(ctx); err != nil {
			logger.Warn("Error during server shutdown: %v", err)
		} else {
			logger.Info(gracefulShutdownMessage)
		}
	}
}

