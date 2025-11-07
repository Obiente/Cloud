package gameservers

import (
	"context"
	"fmt"

	gameserversv1connect "api/gen/proto/obiente/cloud/gameservers/v1/gameserversv1connect"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/orchestrator"
	"api/internal/services/common"

	"connectrpc.com/connect"
)

type Service struct {
	gameserversv1connect.UnimplementedGameServerServiceHandler
	repo              *database.GameServerRepository
	permissionChecker *auth.PermissionChecker
}

func NewService(repo *database.GameServerRepository) *Service {
	return &Service{
		repo:              repo,
		permissionChecker: auth.NewPermissionChecker(),
	}
}

// getGameServerManager returns the game server manager from the orchestrator service
func (s *Service) getGameServerManager() (*orchestrator.GameServerManager, error) {
	orchestratorService := orchestrator.GetGlobalOrchestratorService()
	if orchestratorService == nil {
		return nil, fmt.Errorf("orchestrator service not initialized")
	}
	manager := orchestratorService.GetGameServerManager()
	if manager == nil {
		return nil, fmt.Errorf("game server manager not initialized")
	}
	return manager, nil
}

// ensureAuthenticated ensures the user is authenticated for streaming RPCs.
// This is needed because unary interceptors may not run for streaming RPCs.
func (s *Service) ensureAuthenticated(ctx context.Context, req connect.AnyRequest) (context.Context, error) {
	return common.EnsureAuthenticated(ctx, req)
}

// checkGameServerPermission verifies user permissions for a game server
func (s *Service) checkGameServerPermission(ctx context.Context, gameServerID string, permission string) error {
	// Get game server by ID to check ownership
	gameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
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
	if gameServer.CreatedBy == userInfo.Id {
		return nil // Resource owners have full access to their resources
	}

	// For more complex permissions (organization-based, team-based, etc.)
	err = s.permissionChecker.CheckPermission(ctx, auth.ResourceTypeGameServer, gameServerID, permission)
	if err != nil {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied: %w", err))
	}

	return nil
}

