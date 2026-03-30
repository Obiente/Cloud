package gameservers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"gameservers-service/internal/catalog/modrinth"
	"gameservers-service/internal/orchestrator"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	sharedorchestrator "github.com/obiente/cloud/apps/shared/pkg/orchestrator"
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
	forwarder             *sharedorchestrator.NodeForwarder
	resourcePressureMu    sync.Mutex
	resourcePressureState map[string]*resourcePressureState
	backgroundCtx         context.Context
}

type resourcePressureState struct {
	memoryFirstExceededAt   time.Time
	restartInProgress       bool
	cooldownUntil           time.Time
	lastObservedMemoryUsage int64
}

func NewService(backgroundCtx context.Context, repo *database.GameServerRepository, manager *orchestrator.GameServerManager) *Service {
	return &Service{
		repo:                  repo,
		permissionChecker:     auth.NewPermissionChecker(),
		manager:               manager,
		modClient:             modrinth.NewClient(nil),
		forwarder:             sharedorchestrator.NewNodeForwarder(),
		resourcePressureState: make(map[string]*resourcePressureState),
		backgroundCtx:         backgroundCtx,
	}
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
	baseCtx := s.backgroundCtx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	return auth.WithSystemUser(baseCtx)
}

// checkGameServerPermission verifies user permissions for a game server
// Uses the reusable CheckResourcePermissionWithError helper
func (s *Service) checkGameServerPermission(ctx context.Context, gameServerID string, permission string) error {
	return auth.CheckResourcePermissionWithError(ctx, s.permissionChecker, "gameserver", gameServerID, permission)
}

func (s *Service) shouldForwardToNode(location *database.GameServerLocation) (bool, string) {
	if s.manager == nil || s.forwarder == nil || location == nil {
		return false, ""
	}

	currentNodeID := s.manager.GetNodeID()
	if location.NodeID == "" || location.NodeID == currentNodeID {
		return false, ""
	}
	if s.forwarder.CanForward(location.NodeID) {
		return true, location.NodeID
	}
	return false, ""
}

func (s *Service) getGameServerForwardTarget(ctx context.Context, gameServerID string) (bool, string) {
	var location database.GameServerLocation
	if err := database.DB.WithContext(ctx).
		Where("game_server_id = ?", gameServerID).
		Order("updated_at DESC").
		First(&location).Error; err != nil {
		return false, ""
	}

	return s.shouldForwardToNode(&location)
}

func (s *Service) forwardUnaryRequest(ctx context.Context, reqBody []byte, targetNodeID string, path string, headers map[string]string) ([]byte, error) {
	if s.forwarder == nil {
		return nil, fmt.Errorf("node forwarder not available")
	}

	resp, err := s.forwarder.ForwardConnectRPCRequest(ctx, targetNodeID, "POST", path, bytes.NewReader(reqBody), headers)
	if err != nil {
		return nil, fmt.Errorf("failed to forward request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return bodyBytes, nil
}
