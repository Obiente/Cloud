package gameservers

import (
	"context"
	"fmt"
	"strings"

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
// Uses the unified CheckResourcePermission which handles all permission logic
func (s *Service) checkGameServerPermission(ctx context.Context, gameServerID string, permission string) error {
	if err := s.permissionChecker.CheckResourcePermission(ctx, "gameserver", gameServerID, permission); err != nil {
		// Convert to Connect error with appropriate code
		if strings.Contains(err.Error(), "not found") {
			return connect.NewError(connect.CodeNotFound, err)
		}
		if strings.Contains(err.Error(), "unauthenticated") {
			return connect.NewError(connect.CodeUnauthenticated, err)
		}
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return nil
}
