package deployments

import (
	"context"
	"fmt"
	"log"
	"strings"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	deploymentsv1connect "api/gen/proto/obiente/cloud/deployments/v1/deploymentsv1connect"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/orchestrator"
	"api/internal/quota"

	"connectrpc.com/connect"
)

type Service struct {
	deploymentsv1connect.UnimplementedDeploymentServiceHandler
	repo              *database.DeploymentRepository
	permissionChecker *auth.PermissionChecker
	manager           *orchestrator.DeploymentManager
	quotaChecker      *quota.Checker
	buildRegistry     *BuildStrategyRegistry
}

func NewService(repo *database.DeploymentRepository, manager *orchestrator.DeploymentManager, qc *quota.Checker) *Service {
	return &Service{
		repo:              repo,
		permissionChecker: auth.NewPermissionChecker(),
		manager:           manager,
		quotaChecker:      qc,
		buildRegistry:     NewBuildStrategyRegistry(),
	}
}

// ensureAuthenticated ensures the user is authenticated for streaming RPCs
// This is needed because unary interceptors may not run for streaming RPCs
func (s *Service) ensureAuthenticated(ctx context.Context, req connect.AnyRequest) (context.Context, error) {
	// Check if user is already in context (interceptor ran)
	if userInfo, err := auth.GetUserFromContext(ctx); err == nil && userInfo != nil {
		return ctx, nil
	}

	// Extract token from Authorization header
	authHeader := req.Header().Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	// Use AuthenticateAndSetContext helper which handles token validation and context setup
	ctx, userInfo, err := auth.AuthenticateAndSetContext(ctx, authHeader)
	if err != nil {
		log.Printf("[StreamAuth] Token validation failed for procedure %s: %v", req.Spec().Procedure, err)
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	log.Printf("[StreamAuth] Authenticated user: %s for procedure: %s", userInfo.Id, req.Spec().Procedure)

	return ctx, nil
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

// Config operations (GetDeploymentEnvVars, UpdateDeploymentEnvVars, parseEnvVars, parseEnvFileToMap)
// are now in config.go
// Compose operations (GetDeploymentCompose, ValidateDeploymentCompose, UpdateDeploymentCompose)
// are now in compose.go
// Routing operations (GetDeploymentRoutings, UpdateDeploymentRoutings, GetDeploymentServiceNames)
// are now in routing.go

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
