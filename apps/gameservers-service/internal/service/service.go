package gameservers

import (
	"context"
	"fmt"
	"sync"
	"time"

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
	repo                  *database.GameServerRepository
	permissionChecker     *auth.PermissionChecker
	manager               *orchestrator.GameServerManager // Manager created directly in gameservers-service
	modClient             *modrinth.Client
	resourcePressureMu    sync.Mutex
	resourcePressureState map[string]*resourcePressureState
}

type resourcePressureState struct {
	memoryFirstExceededAt   time.Time
	restartInProgress       bool
	cooldownUntil           time.Time
	lastObservedMemoryUsage int64
}

func NewService(repo *database.GameServerRepository, manager *orchestrator.GameServerManager) *Service {
	return &Service{
		repo:                  repo,
		permissionChecker:     auth.NewPermissionChecker(),
		manager:               manager,
		modClient:             modrinth.NewClient(nil),
		resourcePressureState: make(map[string]*resourcePressureState),
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

// createSystemContext creates a context with a system user that has admin permissions.
// This is used for internal operations that need to bypass permission checks.
func (s *Service) createSystemContext() context.Context {
	return auth.WithSystemUser(context.Background())
}

// checkGameServerPermission verifies user permissions for a game server
// Uses the reusable CheckResourcePermissionWithError helper
func (s *Service) checkGameServerPermission(ctx context.Context, gameServerID string, permission string) error {
	return auth.CheckResourcePermissionWithError(ctx, s.permissionChecker, "gameserver", gameServerID, permission)
}
