package databases

import (
	"context"
	"fmt"
	"time"

	"databases-service/internal/provisioner"
	"databases-service/internal/proxy"
	"databases-service/internal/secrets"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/notifications"
	"github.com/obiente/cloud/apps/shared/pkg/services/common"

	databasesv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1/databasesv1connect"
	notificationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1"

	"connectrpc.com/connect"
)

type Service struct {
	databasesv1connect.UnimplementedDatabaseServiceHandler
	permissionChecker *auth.PermissionChecker
	repo              *database.DatabaseRepository
	connRepo          *database.DatabaseConnectionRepository
	backupRepo        *database.DatabaseBackupRepository
	provisioner       *provisioner.DockerProvisioner
	secretManager     *secrets.SecretManager
	proxy             *proxy.Proxy
	routeRegistry     *proxy.RouteRegistry
	backgroundCtx     context.Context
}

func NewService(
	backgroundCtx context.Context,
	repo *database.DatabaseRepository,
	connRepo *database.DatabaseConnectionRepository,
	backupRepo *database.DatabaseBackupRepository,
) *Service {
	// Initialize Docker provisioner
	prov, err := provisioner.NewDockerProvisioner()
	if err != nil {
		logger.Warn("Failed to initialize Docker provisioner: %v. Database provisioning will not work.", err)
	}

	// Initialize secret manager
	secretMgr, err := secrets.NewSecretManager()
	if err != nil {
		logger.Warn("Failed to initialize secret manager: %v. Passwords will not be encrypted.", err)
	}

	// Initialize Docker client for proxy
	dockerClient, err := docker.New()
	if err != nil {
		logger.Warn("Failed to initialize Docker client for proxy: %v", err)
	}

	// Initialize route registry and proxy
	registry := proxy.NewRouteRegistry(dockerClient)
	proxyServer := proxy.NewProxy(registry, dockerClient, secretMgr)

	svc := &Service{
		permissionChecker: auth.NewPermissionChecker(),
		repo:              repo,
		connRepo:          connRepo,
		backupRepo:        backupRepo,
		provisioner:       prov,
		secretManager:     secretMgr,
		proxy:             proxyServer,
		routeRegistry:     registry,
		backgroundCtx:     backgroundCtx,
	}

	// Wire wake/sleep callbacks
	registry.OnWake = svc.wakeDatabase
	registry.OnSleep = svc.sleepDatabaseAuto

	return svc
}

func (s *Service) detachedContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	baseCtx := s.backgroundCtx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	if timeout <= 0 {
		return context.WithCancel(baseCtx)
	}
	return context.WithTimeout(baseCtx, timeout)
}

// wakeDatabase starts a sleeping database container and returns its IP
func (s *Service) wakeDatabase(ctx context.Context, route *proxy.Route) (string, error) {
	if s.provisioner == nil || route.ContainerID == "" {
		return "", fmt.Errorf("cannot wake database: provisioner or container ID not available")
	}

	logger.Info("Waking database %s", route.DatabaseID)

	// Start the container
	if err := s.provisioner.StartDatabase(ctx, route.ContainerID); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	// Use Docker DNS container name (stable, unlike IPs)
	ip := fmt.Sprintf("obiente-%s", route.DatabaseID)

	// Wait briefly for container to be ready
	time.Sleep(2 * time.Second)

	// Update DB status to RUNNING
	dbInstance, err := s.repo.GetByID(ctx, route.DatabaseID)
	if err == nil {
		dbInstance.Status = 3 // RUNNING
		dbInstance.LastStartedAt = timePtr(time.Now())
		s.repo.Update(ctx, dbInstance)
	}

	// Update route
	s.routeRegistry.MarkRunning(route.DatabaseID, ip)

	// Send notification
	go func() {
		notifCtx, cancel := s.detachedContext(10 * time.Second)
		defer cancel()
		actionURL := fmt.Sprintf("/databases/%s", route.DatabaseID)
		notifications.CreateNotificationForOrganization(
			notifCtx,
			route.OrganizationID,
			notificationsv1.NotificationType_NOTIFICATION_TYPE_INFO,
			notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_LOW,
			"Database woke up",
			"Database was started automatically by an incoming connection.",
			&actionURL, nil,
			map[string]string{"database_id": route.DatabaseID},
			nil,
		)
	}()

	logger.Info("Database %s woke up (ip: %s)", route.DatabaseID, ip)
	return ip, nil
}

// sleepDatabaseAuto puts a database to sleep due to inactivity
func (s *Service) sleepDatabaseAuto(ctx context.Context, route *proxy.Route) error {
	if s.provisioner == nil || route.ContainerID == "" {
		return fmt.Errorf("cannot sleep database: provisioner or container ID not available")
	}

	logger.Info("Auto-sleeping database %s due to inactivity", route.DatabaseID)

	// Stop the container
	if err := s.provisioner.StopDatabase(ctx, route.ContainerID); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Update DB status to SLEEPING
	dbInstance, err := s.repo.GetByID(ctx, route.DatabaseID)
	if err == nil {
		dbInstance.Status = 12 // SLEEPING
		s.repo.Update(ctx, dbInstance)
	}

	// Update route
	s.routeRegistry.MarkStopped(route.DatabaseID, 12)

	// Send notification
	go func() {
		notifCtx, cancel := s.detachedContext(10 * time.Second)
		defer cancel()
		actionURL := fmt.Sprintf("/databases/%s", route.DatabaseID)
		dbName := route.DatabaseID
		if dbInstance != nil {
			dbName = dbInstance.Name
		}
		notifications.CreateNotificationForOrganization(
			notifCtx,
			route.OrganizationID,
			notificationsv1.NotificationType_NOTIFICATION_TYPE_INFO,
			notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_LOW,
			"Database put to sleep",
			fmt.Sprintf("%s was put to sleep due to inactivity. It will start automatically on the next connection.", dbName),
			&actionURL, nil,
			map[string]string{"database_id": route.DatabaseID},
			nil,
		)
	}()

	logger.Info("Database %s put to sleep", route.DatabaseID)
	return nil
}

// GetProxy returns the proxy server instance
func (s *Service) GetProxy() *proxy.Proxy {
	return s.proxy
}

// GetRouteRegistry returns the route registry
func (s *Service) GetRouteRegistry() *proxy.RouteRegistry {
	return s.routeRegistry
}

// ensureAuthenticated ensures the user is authenticated for streaming RPCs
func (s *Service) ensureAuthenticated(ctx context.Context, req connect.AnyRequest) (context.Context, error) {
	return common.EnsureAuthenticated(ctx, req)
}

// checkDatabasePermission verifies user permissions for a database
func (s *Service) checkDatabasePermission(ctx context.Context, databaseID string, permission string) error {
	return auth.CheckResourcePermissionWithError(ctx, s.permissionChecker, "database", databaseID, permission)
}

// checkOrganizationPermission verifies user has access to an organization
func (s *Service) checkOrganizationPermission(ctx context.Context, organizationID string) error {
	return auth.CheckScopedPermissionWithError(ctx, s.permissionChecker, organizationID, auth.ScopedPermission{
		Permission: auth.PermissionOrganizationRead,
	})
}
