package deployments

import (
	"context"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/orchestrator"
	"github.com/obiente/cloud/apps/shared/pkg/quota"
	"github.com/obiente/cloud/apps/shared/pkg/services/common"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
	deploymentsv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1/deploymentsv1connect"

	"connectrpc.com/connect"
)

type Service struct {
	deploymentsv1connect.UnimplementedDeploymentServiceHandler
	repo              *database.DeploymentRepository
	buildHistoryRepo  *database.BuildHistoryRepository
	permissionChecker *auth.PermissionChecker
	manager           *orchestrator.DeploymentManager
	quotaChecker      *quota.Checker
	buildRegistry     *BuildStrategyRegistry
	forwarder         *orchestrator.NodeForwarder
}

func NewService(repo *database.DeploymentRepository, manager *orchestrator.DeploymentManager, qc *quota.Checker) *Service {
	forwarder := orchestrator.NewNodeForwarder()
	return &Service{
		repo:              repo,
		buildHistoryRepo:  database.NewBuildHistoryRepository(database.DB),
		permissionChecker: auth.NewPermissionChecker(),
		manager:           manager,
		quotaChecker:      qc,
		buildRegistry:     NewBuildStrategyRegistry(),
		forwarder:         forwarder,
	}
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
