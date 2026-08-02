package deployments

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/orchestrator"
	"github.com/obiente/cloud/apps/shared/pkg/quota"
	"github.com/obiente/cloud/apps/shared/pkg/services/common"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
	deploymentsv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1/deploymentsv1connect"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type activeDeploymentBuild struct {
	token  string
	cancel context.CancelFunc
}

type Service struct {
	deploymentsv1connect.UnimplementedDeploymentServiceHandler
	repo              *database.DeploymentRepository
	buildHistoryRepo  *database.BuildHistoryRepository
	runtimeLogsRepo   *database.DeploymentRuntimeLogsRepository
	permissionChecker *auth.PermissionChecker
	manager           *orchestrator.DeploymentManager
	quotaChecker      *quota.Checker
	buildRegistry     *BuildStrategyRegistry
	forwarder         *orchestrator.NodeForwarder
	backgroundCtx     context.Context
	activeBuildsMu    sync.Mutex
	activeBuilds      map[string]activeDeploymentBuild
}

func NewService(backgroundCtx context.Context, repo *database.DeploymentRepository, manager *orchestrator.DeploymentManager, qc *quota.Checker) *Service {
	forwarder := orchestrator.NewNodeForwarder()
	return &Service{
		repo:              repo,
		buildHistoryRepo:  database.NewBuildHistoryRepository(database.DB),
		runtimeLogsRepo:   database.NewDeploymentRuntimeLogsRepository(database.MetricsDB),
		permissionChecker: auth.NewPermissionChecker(),
		manager:           manager,
		quotaChecker:      qc,
		buildRegistry:     NewBuildStrategyRegistry(),
		forwarder:         forwarder,
		backgroundCtx:     backgroundCtx,
		activeBuilds:      make(map[string]activeDeploymentBuild),
	}
}

func (s *Service) registerDeploymentBuild(ctx context.Context, deploymentID string, cancel context.CancelFunc) (string, error) {
	token := uuid.NewString()
	ownerNodeID := ""
	if s.manager != nil {
		ownerNodeID = s.manager.GetNodeID()
	}
	registered := false
	err := withDistributedLock(ctx, "deployment-build:"+deploymentID, func() error {
		var deployment database.Deployment
		if err := database.DB.WithContext(ctx).Select("id").Where("id = ? AND deleted_at IS NULL", deploymentID).First(&deployment).Error; err != nil {
			return err
		}
		var existing database.DeploymentBuildControl
		err := database.DB.WithContext(ctx).Where("deployment_id = ?", deploymentID).First(&existing).Error
		if err == nil && existing.CancelRequestedAt != nil {
			return nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		now := time.Now()
		control := database.DeploymentBuildControl{DeploymentID: deploymentID, BuildToken: token, OwnerNodeID: ownerNodeID, CreatedAt: now, UpdatedAt: now}
		if err := database.DB.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "deployment_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"build_token": token, "owner_node_id": ownerNodeID, "cancel_requested_at": nil, "updated_at": now,
			}),
		}).Create(&control).Error; err != nil {
			return err
		}
		registered = true
		return nil
	})
	if err != nil || !registered {
		cancel()
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("deployment has a pending cancellation request")
	}
	s.activeBuildsMu.Lock()
	if s.activeBuilds == nil {
		s.activeBuilds = make(map[string]activeDeploymentBuild)
	}
	s.activeBuilds[deploymentID] = activeDeploymentBuild{token: token, cancel: cancel}
	s.activeBuildsMu.Unlock()
	go s.monitorDeploymentBuild(ctx, deploymentID, token, cancel)
	return token, nil
}

func (s *Service) unregisterDeploymentBuild(deploymentID, token string) {
	s.activeBuildsMu.Lock()
	if current, ok := s.activeBuilds[deploymentID]; ok && current.token == token {
		delete(s.activeBuilds, deploymentID)
	}
	s.activeBuildsMu.Unlock()
	if token == "" {
		return
	}

	ctx, cancel := s.detachedContext(10 * time.Second)
	defer cancel()
	_ = withDistributedLock(ctx, "deployment-build:"+deploymentID, func() error {
		var control database.DeploymentBuildControl
		if err := database.DB.WithContext(ctx).Where("deployment_id = ? AND build_token = ?", deploymentID, token).First(&control).Error; err != nil {
			return nil
		}
		if control.CancelRequestedAt != nil {
			return database.DB.WithContext(ctx).Model(&control).Updates(map[string]interface{}{
				"build_token": "", "owner_node_id": "", "updated_at": time.Now(),
			}).Error
		}
		return database.DB.WithContext(ctx).Delete(&control).Error
	})
}

func (s *Service) cancelDeploymentBuild(deploymentID string) error {
	ctx, cancel := s.detachedContext(10 * time.Second)
	defer cancel()
	buildToken := ""
	persistErr := withDistributedLock(ctx, "deployment-build:"+deploymentID, func() error {
		now := time.Now()
		var existing database.DeploymentBuildControl
		if err := database.DB.WithContext(ctx).Where("deployment_id = ?", deploymentID).First(&existing).Error; err == nil {
			buildToken = existing.BuildToken
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		control := database.DeploymentBuildControl{DeploymentID: deploymentID, BuildToken: "", CancelRequestedAt: &now, CreatedAt: now, UpdatedAt: now}
		return database.DB.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "deployment_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"cancel_requested_at": now, "updated_at": now,
			}),
		}).Create(&control).Error
	})
	s.activeBuildsMu.Lock()
	current := s.activeBuilds[deploymentID]
	if current.cancel != nil {
		delete(s.activeBuilds, deploymentID)
	}
	s.activeBuildsMu.Unlock()
	if current.cancel != nil {
		current.cancel()
	}
	if persistErr != nil || buildToken == "" {
		return persistErr
	}

	// Do not delete the deployment until the replica which owns the build has
	// observed the durable cancellation request and released the build token.
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		var control database.DeploymentBuildControl
		err := database.DB.WithContext(ctx).Where("deployment_id = ?", deploymentID).First(&control).Error
		if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && control.BuildToken == "") {
			return nil
		}
		if err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for build owner to acknowledge cancellation: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func (s *Service) monitorDeploymentBuild(ctx context.Context, deploymentID, token string, cancel context.CancelFunc) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var control database.DeploymentBuildControl
			err := database.DB.WithContext(ctx).Where("deployment_id = ?", deploymentID).First(&control).Error
			if err != nil || control.BuildToken != token || control.CancelRequestedAt != nil {
				cancel()
				return
			}
			var count int64
			if err := database.DB.WithContext(ctx).Model(&database.Deployment{}).Where("id = ? AND deleted_at IS NULL", deploymentID).Count(&count).Error; err != nil || count != 1 {
				cancel()
				return
			}
		}
	}
}

func deploymentBuildIsCurrent(ctx context.Context, deploymentID, token string) bool {
	var control database.DeploymentBuildControl
	if err := database.DB.WithContext(ctx).Where("deployment_id = ?", deploymentID).First(&control).Error; err != nil {
		return false
	}
	if control.BuildToken != token || control.CancelRequestedAt != nil {
		return false
	}
	var count int64
	return database.DB.WithContext(ctx).Model(&database.Deployment{}).Where("id = ? AND deleted_at IS NULL", deploymentID).Count(&count).Error == nil && count == 1
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

// ensureAuthenticated ensures the user is authenticated for streaming RPCs.
// This is needed because unary interceptors may not run for streaming RPCs.
func (s *Service) ensureAuthenticated(ctx context.Context, req connect.AnyRequest) (context.Context, error) {
	return common.EnsureAuthenticated(ctx, req)
}

// checkDeploymentPermission is a helper to verify user permissions
// checkDeploymentPermission verifies user permissions for a deployment
// Uses the reusable CheckResourcePermissionWithError helper
func (s *Service) checkDeploymentPermission(ctx context.Context, deploymentID string, permission string) error {
	return auth.CheckResourcePermissionWithError(ctx, s.permissionChecker, "deployment", deploymentID, permission)
}

// shouldForwardToNode checks if a container location is on a different node and forwarding is possible
func (s *Service) shouldForwardToNode(location *database.DeploymentLocation) (bool, string) {
	if s.manager == nil {
		return false, ""
	}
	currentNodeID := s.manager.GetNodeID()
	if location.NodeID == currentNodeID {
		return false, ""
	}
	// Check if forwarding is possible
	if s.forwarder != nil && s.forwarder.CanForward(location.NodeID) {
		return true, location.NodeID
	}
	return false, ""
}

// Config operations (GetDeploymentEnvVars, UpdateDeploymentEnvVars, parseEnvVars, parseEnvFileToMap)
// are now in config.go
// Compose operations (GetDeploymentCompose, ValidateDeploymentCompose, UpdateDeploymentCompose)
// are now in compose.go
// Routing operations (GetDeploymentRoutings, UpdateDeploymentRoutings, GetDeploymentServiceNames)
// are now in routing.go

// createSystemContext creates a context with a system user that has admin permissions
// This is used for internal operations that need to bypass permission checks
func (s *Service) createSystemContext() context.Context {
	return auth.WithSystemUser(context.Background())
}

func getStatusName(status int32) string {
	switch deploymentsv1.DeploymentStatus(status) {
	case deploymentsv1.DeploymentStatus_CREATED:
		return "CREATED"
	case deploymentsv1.DeploymentStatus_BUILDING:
		return "BUILDING"
	case deploymentsv1.DeploymentStatus_RUNNING:
		return "RUNNING"
	case deploymentsv1.DeploymentStatus_STOPPED:
		return "STOPPED"
	case deploymentsv1.DeploymentStatus_FAILED:
		return "FAILED"
	case deploymentsv1.DeploymentStatus_DEPLOYING:
		return "DEPLOYING"
	default:
		return "UNSPECIFIED"
	}
}
