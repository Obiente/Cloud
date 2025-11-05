package gameservers

import (
	"context"
	"fmt"
	"strings"

	gameserversv1connect "api/gen/proto/obiente/cloud/gameservers/v1/gameserversv1connect"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/logger"
	"api/internal/orchestrator"

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

// ensureAuthenticated ensures the user is authenticated for streaming RPCs
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
		logger.Warn("[StreamAuth] Token validation failed for procedure %s: %v", req.Spec().Procedure, err)
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	logger.Debug("[StreamAuth] Authenticated user: %s for procedure: %s", userInfo.Id, req.Spec().Procedure)

	return ctx, nil
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

