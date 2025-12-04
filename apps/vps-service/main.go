package main

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/obiente/cloud/apps/shared/pkg/notifications"
	"github.com/obiente/cloud/apps/shared/pkg/quota"
	vpssvc "github.com/obiente/cloud/apps/vps-service/internal/service"
	orchestrator "github.com/obiente/cloud/apps/vps-service/orchestrator"

	notificationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1"
	vpsv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1/vpsv1connect"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	_ "github.com/joho/godotenv/autoload"
)

const (
	readHeaderTimeout       = 10 * time.Second
	writeTimeout            = 6 * time.Minute // Increased for VPS creation which can take 1-2 minutes
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

	database.RegisterModels(
		&database.Organization{},
		&database.OrganizationMember{},
	)

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
	var vpsManager *orchestrator.VPSManager
	var err error
	vpsManager, err = orchestrator.NewVPSManager()
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

	// Start background sync jobs if VPS manager is available
	if vpsManager != nil {
		// Start periodic VPS status sync (every 5 minutes)
		// This detects VPSs that were deleted from Proxmox and marks them as DELETED
		go startVPSStatusSync(shutdownCtx, vpsManager)
		logger.Info("✓ VPS status sync service started")

		// Start periodic VPS import (every 15 minutes)
		// This imports VPSs that exist in Proxmox but are missing from the database
		go startVPSImportSync(shutdownCtx, vpsManager)
		logger.Info("✓ VPS import sync service started")
	}

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

// startVPSStatusSync starts the periodic VPS status sync background service
// This syncs all VPS statuses from Proxmox to detect deleted VPSs
func startVPSStatusSync(ctx context.Context, vpsManager *orchestrator.VPSManager) {
	// Run every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Run once on startup (after a short delay to ensure DB is ready)
	time.Sleep(10 * time.Second)
	syncCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	deletedVPSs, err := vpsManager.SyncAllVPSStatuses(syncCtx)
	cancel()
	if err != nil {
		logger.Warn("[VPS Status Sync] Error on startup sync: %v", err)
	} else {
		// Send notifications for deleted VPSs
		sendDeletedVPSNotifications(context.Background(), deletedVPSs)
	}

	for {
		select {
		case <-ctx.Done():
			logger.Info("[VPS Status Sync] Status sync service stopped")
			return
		case <-ticker.C:
			syncCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			deletedVPSs, err := vpsManager.SyncAllVPSStatuses(syncCtx)
			cancel()
			if err != nil {
				logger.Warn("[VPS Status Sync] Error during periodic sync: %v", err)
			} else {
				// Send notifications for deleted VPSs
				sendDeletedVPSNotifications(context.Background(), deletedVPSs)
			}
		}
	}
}

// sendDeletedVPSNotifications sends notifications for VPSs that were marked as deleted
func sendDeletedVPSNotifications(ctx context.Context, deletedVPSs map[string]int32) {
	if len(deletedVPSs) == 0 {
		return
	}

	for vpsID, oldStatus := range deletedVPSs {
		var vps database.VPSInstance
		if err := database.DB.Where("id = ?", vpsID).First(&vps).Error; err != nil {
			logger.Warn("[VPS Status Sync] Failed to get VPS %s for notification: %v", vpsID, err)
			continue
		}

		// Only send notification if status changed to DELETED (9)
		if vps.Status == 9 && oldStatus != 9 {
			title := fmt.Sprintf("VPS Removed: %s", vps.Name)
			message := fmt.Sprintf("Your VPS instance '%s' was detected as deleted from the infrastructure. It has been marked as deleted in the system.", vps.Name)

			metadata := map[string]string{
				"vps_id":         vps.ID,
				"vps_name":       vps.Name,
				"vps_status":     fmt.Sprintf("%d", vps.Status),
				"deletion_source": "infrastructure",
				"event_type":     "vps_deleted_from_proxmox",
			}
			if vps.InstanceID != nil {
				metadata["vm_id"] = *vps.InstanceID
			}

			// Send notification to VPS creator
			if vps.CreatedBy != "" {
				actionURL := fmt.Sprintf("/vps/%s", vps.ID)
				actionLabel := "View VPS"
				orgID := vps.OrganizationID
				if err := notifications.CreateNotificationForUser(
					ctx,
					vps.CreatedBy,
					&orgID,
					notificationsv1.NotificationType_NOTIFICATION_TYPE_SYSTEM,
					notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_HIGH,
					title,
					message,
					&actionURL,
					&actionLabel,
					metadata,
				); err != nil {
					logger.Warn("[VPS Status Sync] Failed to send notification for deleted VPS %s: %v", vpsID, err)
				} else {
					logger.Info("[VPS Status Sync] Sent notification for deleted VPS %s", vpsID)
				}
			}
		}
	}
}

// startVPSImportSync starts the periodic VPS import background service
// This imports VPSs that exist in Proxmox but are missing from the database
func startVPSImportSync(ctx context.Context, vpsManager *orchestrator.VPSManager) {
	// Run every 15 minutes
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	// Run once on startup (after a short delay to ensure DB is ready)
	time.Sleep(15 * time.Second)
	importCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if err := vpsManager.ImportMissingVPSForAllOrgs(importCtx); err != nil {
		logger.Warn("[VPS Import Sync] Error on startup import: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			logger.Info("[VPS Import Sync] Import sync service stopped")
			return
		case <-ticker.C:
			importCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			if err := vpsManager.ImportMissingVPSForAllOrgs(importCtx); err != nil {
				logger.Warn("[VPS Import Sync] Error during periodic import: %v", err)
			}
			cancel()
		}
	}
}
