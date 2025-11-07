package deployments

import (
	"context"
	"fmt"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	deploymentsv1connect "api/gen/proto/obiente/cloud/deployments/v1/deploymentsv1connect"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/orchestrator"
	"api/internal/quota"
	"api/internal/services/common"

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
func (s *Service) checkDeploymentPermission(ctx context.Context, deploymentID string, permission string) error {
	// Get deployment by ID to check ownership
	deployment, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
	}
	
	// Get user from context
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}
	
	// First check if user is admin (always has access)
	if auth.HasRole(userInfo, auth.RoleAdmin) {
		return nil
	}
	
	// Check if user is the resource owner
    if deployment.CreatedBy == userInfo.Id {
		return nil // Resource owners have full access to their resources
	}
	
	// For more complex permissions (organization-based, team-based, etc.)
	err = s.permissionChecker.CheckPermission(ctx, auth.ResourceTypeDeployment, deploymentID, permission)
	if err != nil {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied: %w", err))
	}
	
	return nil
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
