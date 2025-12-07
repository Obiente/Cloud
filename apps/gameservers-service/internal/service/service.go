package gameservers

import (
	"context"
	"fmt"

	"gameservers-service/internal/catalog/modrinth"
	"gameservers-service/internal/orchestrator"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/services/common"

	gameserversv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1/gameserversv1connect"

	"connectrpc.com/connect"
)

type Service struct {
	gameserversv1connect.UnimplementedGameServerServiceHandler
	repo              *database.GameServerRepository
	permissionChecker *auth.PermissionChecker
	manager           *orchestrator.GameServerManager // Manager created directly in gameservers-service
	modClient         *modrinth.Client
}

func NewService(repo *database.GameServerRepository, manager *orchestrator.GameServerManager) *Service {
	return &Service{
		repo:              repo,
		permissionChecker: auth.NewPermissionChecker(),
		manager:           manager,
		modClient:         modrinth.NewClient(nil),
	}
}

// getGameServerManager returns the game server manager
func (s *Service) getGameServerManager() (*orchestrator.GameServerManager, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("game server manager not initialized")
	}
	return s.manager, nil
}

// ensureAuthenticated ensures the user is authenticated for streaming RPCs.
// This is needed because unary interceptors may not run for streaming RPCs.
func (s *Service) ensureAuthenticated(ctx context.Context, req connect.AnyRequest) (context.Context, error) {
	return common.EnsureAuthenticated(ctx, req)
}

// checkGameServerPermission verifies user permissions for a game server
func (s *Service) checkGameServerPermission(ctx context.Context, gameServerID string, permission string) error {
	// Get game server by ID to check ownership and organization context
	gameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
	}

	// Get user from context
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}

	// First check if user is global admin (always has access)
	if auth.HasRole(userInfo, auth.RoleAdmin) {
		return nil
	}

	// Check if user is the resource owner
	if gameServer.CreatedBy == userInfo.Id {
		return nil // Resource owners have full access to their resources
	}

	// Map permission strings to the correct scoped permission format
	// "view" -> "gameservers.read", "update" -> "gameservers.update", etc.
	scopedPermission := mapGameServerPermission(permission)

	// Check organization-scoped permissions (includes system roles like system:admin)
	// This properly evaluates organization-level roles with gameservers.* permissions
	err = s.permissionChecker.CheckScopedPermission(ctx, gameServer.OrganizationID, auth.ScopedPermission{
		Permission:   scopedPermission,
		ResourceType: "gameserver",
		ResourceID:   gameServerID,
	})
	if err != nil {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied: insufficient permissions"))
	}

	return nil
}

// mapGameServerPermission maps action strings to scoped permission format
func mapGameServerPermission(permission string) string {
	switch permission {
	case "view":
		return "gameservers.read"
	case "create":
		return "gameservers.create"
	case "update":
		return "gameservers.update"
	case "delete":
		return "gameservers.delete"
	case "start":
		return "gameservers.start"
	case "stop":
		return "gameservers.stop"
	case "restart":
		return "gameservers.restart"
	default:
		// If already in correct format, return as-is
		return permission
	}
}
