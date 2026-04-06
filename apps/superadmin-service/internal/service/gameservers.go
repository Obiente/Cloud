package superadmin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/client"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"

	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"
	superadminv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/superadmin/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const defaultPageSize = 20
const maxPageSize = 100

// ListAllGameServers returns a paginated list of all game servers across all organisations.
func (s *Service) ListAllGameServers(ctx context.Context, req *connect.Request[superadminv1.ListAllGameServersRequest]) (*connect.Response[superadminv1.ListAllGameServersResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.gameservers.read") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	page := int(req.Msg.GetPage())
	if page < 1 {
		page = 1
	}
	perPage := int(req.Msg.GetPerPage())
	if perPage < 1 {
		perPage = defaultPageSize
	}
	if perPage > maxPageSize {
		perPage = maxPageSize
	}
	offset := (page - 1) * perPage

	query := database.DB.Model(&database.GameServer{}).Where("deleted_at IS NULL")

	if orgID := req.Msg.GetOrganizationId(); orgID != "" {
		query = query.Where("organization_id = ?", orgID)
	}
	if status := req.Msg.GetStatus(); status != gameserversv1.GameServerStatus_GAME_SERVER_STATUS_UNSPECIFIED {
		query = query.Where("status = ?", int32(status))
	}
	if search := req.Msg.GetSearch(); search != "" {
		like := "%" + search + "%"
		query = query.Where("name ILIKE ? OR id ILIKE ? OR organization_id ILIKE ?", like, like, like)
	}

	if req.Msg.GetFlaggedOnly() {
		// Only game servers that have an active suspension record
		query = query.Where("id IN (SELECT game_server_id FROM game_server_suspensions WHERE lifted_at IS NULL)")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to count game servers: %w", err))
	}

	var gameServers []*database.GameServer
	if err := query.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&gameServers).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list game servers: %w", err))
	}

	// Collect org IDs for batch lookup
	orgIDs := make(map[string]struct{})
	for _, gs := range gameServers {
		orgIDs[gs.OrganizationID] = struct{}{}
	}
	orgIDSlice := make([]string, 0, len(orgIDs))
	for id := range orgIDs {
		orgIDSlice = append(orgIDSlice, id)
	}

	// Batch load org names
	orgNames := make(map[string]string, len(orgIDSlice))
	if len(orgIDSlice) > 0 {
		var orgs []struct {
			ID   string
			Name string
		}
		database.DB.Model(&database.Organization{}).Select("id, name").Where("id IN ?", orgIDSlice).Scan(&orgs)
		for _, o := range orgs {
			orgNames[o.ID] = o.Name
		}
	}

	// Batch load active suspensions
	activeSuspensions := make(map[string]*database.GameServerSuspension)
	if len(gameServers) > 0 {
		gsIDs := make([]string, 0, len(gameServers))
		for _, gs := range gameServers {
			gsIDs = append(gsIDs, gs.ID)
		}
		var suspensions []*database.GameServerSuspension
		database.DB.Where("game_server_id IN ? AND lifted_at IS NULL", gsIDs).Find(&suspensions)
		for _, susp := range suspensions {
			activeSuspensions[susp.GameServerID] = susp
		}
	}

	overviews := make([]*superadminv1.GameServerOverview, 0, len(gameServers))
	for _, gs := range gameServers {
		ov := gameServerToOverview(gs, orgNames[gs.OrganizationID], activeSuspensions[gs.ID])
		overviews = append(overviews, ov)
	}

	totalPages := int32((total + int64(perPage) - 1) / int64(perPage))
	return connect.NewResponse(&superadminv1.ListAllGameServersResponse{
		GameServers: overviews,
		Pagination: &commonv1.Pagination{
			Page:       int32(page),
			PerPage:    int32(perPage),
			Total:      int32(total),
			TotalPages: totalPages,
		},
	}), nil
}

// SuperadminGetGameServer returns a single game server with full details.
func (s *Service) SuperadminGetGameServer(ctx context.Context, req *connect.Request[superadminv1.SuperadminGetGameServerRequest]) (*connect.Response[superadminv1.SuperadminGetGameServerResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.gameservers.read") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	gsID := req.Msg.GetGameServerId()
	if gsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id is required"))
	}

	var gs database.GameServer
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", gsID).First(&gs).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get game server: %w", err))
	}

	var org database.Organization
	var orgName string
	if err := database.DB.Select("name").Where("id = ?", gs.OrganizationID).First(&org).Error; err == nil {
		orgName = org.Name
	}

	var susp *database.GameServerSuspension
	var tmp database.GameServerSuspension
	if err := database.DB.Where("game_server_id = ? AND lifted_at IS NULL", gsID).First(&tmp).Error; err == nil {
		susp = &tmp
	}

	return connect.NewResponse(&superadminv1.SuperadminGetGameServerResponse{
		GameServer: gameServerToOverview(&gs, orgName, susp),
	}), nil
}

// SuperadminSuspendGameServer suspends a game server (records suspension + stops the container).
func (s *Service) SuperadminSuspendGameServer(ctx context.Context, req *connect.Request[superadminv1.SuperadminSuspendGameServerRequest]) (*connect.Response[superadminv1.SuperadminSuspendGameServerResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.gameservers.update") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	gsID := req.Msg.GetGameServerId()
	if gsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id is required"))
	}

	var gs database.GameServer
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", gsID).First(&gs).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get game server: %w", err))
	}

	// Idempotent: if already suspended, return OK
	var existing database.GameServerSuspension
	if err := database.DB.Where("game_server_id = ? AND lifted_at IS NULL", gsID).First(&existing).Error; err == nil {
		gs2 := gameServerProto(&gs)
		return connect.NewResponse(&superadminv1.SuperadminSuspendGameServerResponse{
			GameServer: gs2,
			Message:    "Game server is already suspended",
		}), nil
	}

	reason := req.Msg.GetReason()
	susp := &database.GameServerSuspension{
		ID:             uuid.New().String(),
		GameServerID:   gsID,
		OrganizationID: gs.OrganizationID,
		SuspendedBy:    user.Id,
		SuspendedAt:    time.Now(),
	}
	if reason != "" {
		susp.Reason = &reason
	}

	if err := database.DB.Create(susp).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to record suspension: %w", err))
	}

	// Stop the container in the background; if it fails, we still record the suspension.
	if gs.ContainerID != nil {
		if err := stopGameServerContainer(ctx, *gs.ContainerID); err != nil {
			logger.Warn("[Moderation] Failed to stop game server container %s: %v", *gs.ContainerID, err)
		}
	}

	// Mark status as STOPPED in DB
	stopStatus := int32(gameserversv1.GameServerStatus_STOPPED)
	database.DB.Model(&database.GameServer{}).Where("id = ?", gsID).Update("status", stopStatus)

	// Reload
	database.DB.Where("id = ?", gsID).First(&gs)
	logger.Info("[Moderation] Game server %s suspended by %s", gsID, user.Id)
	return connect.NewResponse(&superadminv1.SuperadminSuspendGameServerResponse{
		GameServer: gameServerProto(&gs),
		Message:    "Game server suspended",
	}), nil
}

// SuperadminUnsuspendGameServer lifts a suspension on a game server.
func (s *Service) SuperadminUnsuspendGameServer(ctx context.Context, req *connect.Request[superadminv1.SuperadminUnsuspendGameServerRequest]) (*connect.Response[superadminv1.SuperadminUnsuspendGameServerResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.gameservers.update") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	gsID := req.Msg.GetGameServerId()
	if gsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id is required"))
	}

	now := time.Now()
	liftedBy := user.Id
	result := database.DB.Model(&database.GameServerSuspension{}).
		Where("game_server_id = ? AND lifted_at IS NULL", gsID).
		Updates(map[string]interface{}{
			"lifted_at": now,
			"lifted_by": liftedBy,
		})
	if result.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to lift suspension: %w", result.Error))
	}

	var gs database.GameServer
	database.DB.Where("id = ?", gsID).First(&gs)

	logger.Info("[Moderation] Game server %s unsuspended by %s", gsID, user.Id)
	return connect.NewResponse(&superadminv1.SuperadminUnsuspendGameServerResponse{
		GameServer: gameServerProto(&gs),
		Message:    "Game server suspension lifted",
	}), nil
}

// SuperadminForceStopGameServer force-stops a game server container regardless of state.
func (s *Service) SuperadminForceStopGameServer(ctx context.Context, req *connect.Request[superadminv1.SuperadminForceStopGameServerRequest]) (*connect.Response[superadminv1.SuperadminForceStopGameServerResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.gameservers.update") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	gsID := req.Msg.GetGameServerId()
	if gsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id is required"))
	}

	var gs database.GameServer
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", gsID).First(&gs).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get game server: %w", err))
	}

	if gs.ContainerID != nil {
		if err := stopGameServerContainer(ctx, *gs.ContainerID); err != nil {
			logger.Warn("[Superadmin] Force stop container %s failed: %v", *gs.ContainerID, err)
		}
	}

	stopStatus := int32(gameserversv1.GameServerStatus_STOPPED)
	database.DB.Model(&database.GameServer{}).Where("id = ?", gsID).Update("status", stopStatus)
	database.DB.Where("id = ?", gsID).First(&gs)

	logger.Info("[Superadmin] Game server %s force-stopped by %s", gsID, user.Id)
	return connect.NewResponse(&superadminv1.SuperadminForceStopGameServerResponse{
		GameServer: gameServerProto(&gs),
		Message:    "Game server force stopped",
	}), nil
}

// SuperadminForceDeleteGameServer force-deletes a game server (stops container and removes DB record).
func (s *Service) SuperadminForceDeleteGameServer(ctx context.Context, req *connect.Request[superadminv1.SuperadminForceDeleteGameServerRequest]) (*connect.Response[superadminv1.SuperadminForceDeleteGameServerResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.gameservers.delete") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	gsID := req.Msg.GetGameServerId()
	if gsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id is required"))
	}

	var gs database.GameServer
	if err := database.DB.Where("id = ?", gsID).First(&gs).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get game server: %w", err))
	}

	// Stop + remove container
	if gs.ContainerID != nil {
		if err := removeGameServerContainer(ctx, *gs.ContainerID); err != nil {
			logger.Warn("[Superadmin] Failed to remove container %s: %v", *gs.ContainerID, err)
		}
	}

	// Hard-delete or soft-delete based on request flag
	if req.Msg.GetHardDelete() {
		if err := database.DB.Unscoped().Delete(&gs).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to hard-delete game server: %w", err))
		}
	} else {
		now := time.Now()
		if err := database.DB.Model(&gs).Update("deleted_at", now).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete game server: %w", err))
		}
	}

	logger.Info("[Superadmin] Game server %s force-deleted by %s (hard=%v)", gsID, user.Id, req.Msg.GetHardDelete())
	return connect.NewResponse(&superadminv1.SuperadminForceDeleteGameServerResponse{
		Success: true,
		Message: fmt.Sprintf("Game server %s deleted", gsID),
	}), nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func gameServerToOverview(gs *database.GameServer, orgName string, susp *database.GameServerSuspension) *superadminv1.GameServerOverview {
	ov := &superadminv1.GameServerOverview{
		GameServer:       gameServerProto(gs),
		OrganizationName: orgName,
	}
	if susp != nil {
		ov.IsSuspended = true
		if susp.Reason != nil {
			reason := *susp.Reason
			ov.SuspensionReason = &reason
		}
	}
	return ov
}

func gameServerProto(gs *database.GameServer) *gameserversv1.GameServer {
	if gs == nil {
		return nil
	}
	proto := &gameserversv1.GameServer{
		Id:             gs.ID,
		OrganizationId: gs.OrganizationID,
		Name:           gs.Name,
		Description:    gs.Description,
		GameType:       gameserversv1.GameType(gs.GameType),
		Status:         gameserversv1.GameServerStatus(gs.Status),
		MemoryBytes:    gs.MemoryBytes,
		CpuCores:       gs.CPUCores,
		Port:           gs.Port,
		DockerImage:    gs.DockerImage,
		StartCommand:   gs.StartCommand,
		PlayerCount:    gs.PlayerCount,
		MaxPlayers:     gs.MaxPlayers,
		StorageBytes:   gs.StorageBytes,
		CreatedAt:      timestamppb.New(gs.CreatedAt),
		UpdatedAt:      timestamppb.New(gs.UpdatedAt),
		CreatedBy:      gs.CreatedBy,
	}
	proto.ContainerId = gs.ContainerID
	proto.ContainerName = gs.ContainerName
	if gs.LastStartedAt != nil {
		proto.LastStartedAt = timestamppb.New(*gs.LastStartedAt)
	}
	return proto
}

// stopGameServerContainer gracefully stops a container.
func stopGameServerContainer(ctx context.Context, containerID string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("create docker client: %w", err)
	}
	defer cli.Close()

	timeout := 30
	_, err = cli.ContainerStop(ctx, containerID, client.ContainerStopOptions{Timeout: &timeout})
	return err
}

// removeGameServerContainer stops and removes a container.
func removeGameServerContainer(ctx context.Context, containerID string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("create docker client: %w", err)
	}
	defer cli.Close()

	// Verify the container is managed by Obiente before removal (security check)
	info, err := cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		// Container probably already gone; treat as success
		return nil
	}
	if info.Container.Config.Labels["cloud.obiente.managed"] != "true" {
		return fmt.Errorf("refusing to remove container %s: not managed by Obiente Cloud", containerID)
	}

	timeout := 30
	_, _ = cli.ContainerStop(ctx, containerID, client.ContainerStopOptions{Timeout: &timeout})
	_, err = cli.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{Force: true})
	return err
}
