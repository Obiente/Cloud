package gameservers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	gameserversv1 "api/gen/proto/obiente/cloud/gameservers/v1"
	"api/internal/auth"
	"api/internal/database"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Helper function to resolve user's default organization ID
func resolveUserDefaultOrgID(ctx context.Context) (string, bool) {
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil || userInfo == nil {
		return "", false
	}
	// Pick any organization the user belongs to (first by created_at desc)
	type row struct{ OrganizationID string }
	var r row
	if err := database.DB.Raw(`
        SELECT m.organization_id
        FROM organization_members m
        JOIN organizations o ON o.id = m.organization_id
        WHERE m.user_id = ?
        ORDER BY o.created_at DESC
        LIMIT 1
    `, userInfo.Id).Scan(&r).Error; err != nil {
		return "", false
	}
	if r.OrganizationID == "" {
		return "", false
	}
	return r.OrganizationID, true
}

// ListGameServers lists all game servers for an organization
func (s *Service) ListGameServers(ctx context.Context, req *connect.Request[gameserversv1.ListGameServersRequest]) (*connect.Response[gameserversv1.ListGameServersResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	// Get authenticated user from context
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user authentication required: %w", err))
	}

	// Create filters with user ID
	filters := &database.GameServerFilters{
		UserID:     userInfo.Id,
		IncludeAll: auth.HasRole(userInfo, auth.RoleAdmin),
	}

	// Add status filter if provided
	if req.Msg.Status != nil {
		statusVal := int32(*req.Msg.Status)
		filters.Status = &statusVal
	}

	// Add game type filter if provided
	if req.Msg.GameType != nil && *req.Msg.GameType != "" {
		// Convert string to int32 enum value
		// TODO: Implement proper enum conversion
	}

	// Get game servers filtered by organization and user ID
	dbGameServers, err := s.repo.GetAll(ctx, orgID, filters)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list game servers: %w", err))
	}

	// Convert DB models to proto models
	items := make([]*gameserversv1.GameServer, 0, len(dbGameServers))
	for _, dbGS := range dbGameServers {
		gameServer := dbGameServerToProto(dbGS)
		items = append(items, gameServer)
	}

	res := connect.NewResponse(&gameserversv1.ListGameServersResponse{
		GameServers: items,
	})
	return res, nil
}

// CreateGameServer creates a new game server
func (s *Service) CreateGameServer(ctx context.Context, req *connect.Request[gameserversv1.CreateGameServerRequest]) (*connect.Response[gameserversv1.CreateGameServerResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user authentication required: %w", err))
	}

	// Permission: org-level
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "gameservers.create"}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	id := fmt.Sprintf("gs-%d", time.Now().Unix())

	// Get Docker image for game type (defaults if not specified)
	dockerImage := req.Msg.GetDockerImage()
	if dockerImage == "" {
		dockerImage = getDefaultDockerImage(req.Msg.GetGameType())
	}

	// Get memory bytes (default to 2GB if not specified)
	memoryBytes := req.Msg.GetMemoryBytes()
	if memoryBytes == 0 {
		memoryBytes = 2147483648 // 2GB default
	}

	// Get CPU cores (default to 1 if not specified)
	cpuCores := req.Msg.GetCpuCores()
	if cpuCores == 0 {
		cpuCores = 1
	}

	// Get available port
	port := req.Msg.GetPort()
	if port == 0 {
		availablePort, err := s.repo.GetAvailablePort(ctx, 25565) // Start from Minecraft default port
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get available port: %w", err))
		}
		port = availablePort
	}

	// Serialize environment variables
	envVarsJSON := "{}"
	if len(req.Msg.GetEnvVars()) > 0 {
		envVarsBytes, err := json.Marshal(req.Msg.GetEnvVars())
		if err == nil {
			envVarsJSON = string(envVarsBytes)
		}
	}

	// Create game server in database
	dbGameServer := &database.GameServer{
		ID:             id,
		Name:           req.Msg.GetName(),
		Description:    req.Msg.Description,
		GameType:       int32(req.Msg.GetGameType()),
		Status:         int32(gameserversv1.GameServerStatus_CREATED),
		MemoryBytes:    memoryBytes,
		CPUCores:       cpuCores,
		Port:           port,
		DockerImage:    dockerImage,
		StartCommand:   req.Msg.StartCommand,
		EnvVars:        envVarsJSON,
		ServerVersion:  req.Msg.ServerVersion,
		StorageBytes:   0,
		BandwidthUsage: 0,
		OrganizationID: orgID,
		CreatedBy:      userInfo.Id,
	}

	if err := s.repo.Create(ctx, dbGameServer); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create game server: %w", err))
	}

	// Fetch the created game server
	createdGameServer, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch created game server: %w", err))
	}

	gameServer := dbGameServerToProto(createdGameServer)

	res := connect.NewResponse(&gameserversv1.CreateGameServerResponse{
		GameServer: gameServer,
	})
	return res, nil
}

// GetGameServer retrieves a game server by ID
func (s *Service) GetGameServer(ctx context.Context, req *connect.Request[gameserversv1.GetGameServerRequest]) (*connect.Response[gameserversv1.GetGameServerResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	if err := s.checkGameServerPermission(ctx, gameServerID, "view"); err != nil {
		return nil, err
	}

	dbGameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
	}

	gameServer := dbGameServerToProto(dbGameServer)

	res := connect.NewResponse(&gameserversv1.GetGameServerResponse{
		GameServer: gameServer,
	})
	return res, nil
}

// UpdateGameServer updates a game server configuration
func (s *Service) UpdateGameServer(ctx context.Context, req *connect.Request[gameserversv1.UpdateGameServerRequest]) (*connect.Response[gameserversv1.UpdateGameServerResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	if err := s.checkGameServerPermission(ctx, gameServerID, "update"); err != nil {
		return nil, err
	}

	dbGameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
	}

	// Update fields if provided
	if req.Msg.Name != nil {
		dbGameServer.Name = *req.Msg.Name
	}
	if req.Msg.MemoryBytes != nil {
		dbGameServer.MemoryBytes = *req.Msg.MemoryBytes
	}
	if req.Msg.CpuCores != nil {
		dbGameServer.CPUCores = *req.Msg.CpuCores
	}
	if req.Msg.StartCommand != nil {
		dbGameServer.StartCommand = req.Msg.StartCommand
	}
	if req.Msg.Description != nil {
		dbGameServer.Description = req.Msg.Description
	}
	if req.Msg.EnvVars != nil && len(req.Msg.EnvVars) > 0 {
		envVarsBytes, err := json.Marshal(req.Msg.EnvVars)
		if err == nil {
			dbGameServer.EnvVars = string(envVarsBytes)
		}
	}

	if err := s.repo.Update(ctx, dbGameServer); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update game server: %w", err))
	}

	// Fetch updated game server
	updatedGameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch updated game server: %w", err))
	}

	gameServer := dbGameServerToProto(updatedGameServer)

	res := connect.NewResponse(&gameserversv1.UpdateGameServerResponse{
		GameServer: gameServer,
	})
	return res, nil
}

// DeleteGameServer deletes a game server (soft delete)
func (s *Service) DeleteGameServer(ctx context.Context, req *connect.Request[gameserversv1.DeleteGameServerRequest]) (*connect.Response[gameserversv1.DeleteGameServerResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	if err := s.checkGameServerPermission(ctx, gameServerID, "delete"); err != nil {
		return nil, err
	}

	// TODO: Stop and remove container if running
	// if err := s.manager.StopGameServer(ctx, gameServerID); err != nil {
	// 	return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stop game server: %w", err))
	// }

	if err := s.repo.Delete(ctx, gameServerID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete game server: %w", err))
	}

	res := connect.NewResponse(&gameserversv1.DeleteGameServerResponse{
		Success: true,
	})
	return res, nil
}

// StartGameServer starts a stopped game server
func (s *Service) StartGameServer(ctx context.Context, req *connect.Request[gameserversv1.StartGameServerRequest]) (*connect.Response[gameserversv1.StartGameServerResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	if err := s.checkGameServerPermission(ctx, gameServerID, "start"); err != nil {
		return nil, err
	}

	dbGameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
	}

	// Update status to STARTING
	if err := s.repo.UpdateStatus(ctx, gameServerID, int32(gameserversv1.GameServerStatus_STARTING)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update status: %w", err))
	}

	// TODO: Integrate with orchestrator to start Docker container
	// if err := s.manager.StartGameServer(ctx, dbGameServer); err != nil {
	// 	s.repo.UpdateStatus(ctx, gameServerID, int32(gameserversv1.GameServerStatus_FAILED))
	// 	return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start game server: %w", err))
	// }

	// For now, just update status to RUNNING (will be replaced with actual container management)
	now := time.Now()
	dbGameServer.LastStartedAt = &now
	if err := s.repo.UpdateStatus(ctx, gameServerID, int32(gameserversv1.GameServerStatus_RUNNING)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update status: %w", err))
	}

	updatedGameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch updated game server: %w", err))
	}

	gameServer := dbGameServerToProto(updatedGameServer)

	res := connect.NewResponse(&gameserversv1.StartGameServerResponse{
		GameServer: gameServer,
	})
	return res, nil
}

// StopGameServer stops a running game server
func (s *Service) StopGameServer(ctx context.Context, req *connect.Request[gameserversv1.StopGameServerRequest]) (*connect.Response[gameserversv1.StopGameServerResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	if err := s.checkGameServerPermission(ctx, gameServerID, "stop"); err != nil {
		return nil, err
	}

	// Update status to STOPPING
	if err := s.repo.UpdateStatus(ctx, gameServerID, int32(gameserversv1.GameServerStatus_STOPPING)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update status: %w", err))
	}

	// TODO: Integrate with orchestrator to stop Docker container
	// if err := s.manager.StopGameServer(ctx, gameServerID); err != nil {
	// 	return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stop game server: %w", err))
	// }

	// Update status to STOPPED
	if err := s.repo.UpdateStatus(ctx, gameServerID, int32(gameserversv1.GameServerStatus_STOPPED)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update status: %w", err))
	}

	updatedGameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch updated game server: %w", err))
	}

	gameServer := dbGameServerToProto(updatedGameServer)

	res := connect.NewResponse(&gameserversv1.StopGameServerResponse{
		GameServer: gameServer,
	})
	return res, nil
}

// RestartGameServer restarts a game server
func (s *Service) RestartGameServer(ctx context.Context, req *connect.Request[gameserversv1.RestartGameServerRequest]) (*connect.Response[gameserversv1.RestartGameServerResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	if err := s.checkGameServerPermission(ctx, gameServerID, "restart"); err != nil {
		return nil, err
	}

	// Stop first
	if _, err := s.StopGameServer(ctx, connect.NewRequest(&gameserversv1.StopGameServerRequest{
		GameServerId: gameServerID,
	})); err != nil {
		return nil, err
	}

	// Wait a bit before starting (optional)
	time.Sleep(2 * time.Second)

	// Start again
	startResp, err := s.StartGameServer(ctx, connect.NewRequest(&gameserversv1.StartGameServerRequest{
		GameServerId: gameServerID,
	}))
	if err != nil {
		return nil, err
	}

	res := connect.NewResponse(&gameserversv1.RestartGameServerResponse{
		GameServer: startResp.Msg.GameServer,
	})
	return res, nil
}

// Helper functions for conversion

func dbGameServerToProto(dbGS *database.GameServer) *gameserversv1.GameServer {
	// Parse environment variables
	envVars := make(map[string]string)
	if dbGS.EnvVars != "" {
		json.Unmarshal([]byte(dbGS.EnvVars), &envVars)
	}

	gameServer := &gameserversv1.GameServer{
		Id:             dbGS.ID,
		OrganizationId: dbGS.OrganizationID,
		Name:           dbGS.Name,
		Description:    dbGS.Description,
		GameType:       gameserversv1.GameType(dbGS.GameType),
		Status:         gameserversv1.GameServerStatus(dbGS.Status),
		MemoryBytes:    dbGS.MemoryBytes,
		CpuCores:       dbGS.CPUCores,
		Port:           dbGS.Port,
		DockerImage:    dbGS.DockerImage,
		StartCommand:   dbGS.StartCommand,
		EnvVars:        envVars,
		ServerVersion:  dbGS.ServerVersion,
		PlayerCount:    dbGS.PlayerCount,
		MaxPlayers:     dbGS.MaxPlayers,
		ContainerId:    dbGS.ContainerID,
		ContainerName:  dbGS.ContainerName,
		StorageBytes:   dbGS.StorageBytes,
		CreatedAt:      timestamppb.New(dbGS.CreatedAt),
		UpdatedAt:      timestamppb.New(dbGS.UpdatedAt),
		CreatedBy:      dbGS.CreatedBy,
	}

	if dbGS.LastStartedAt != nil {
		gameServer.LastStartedAt = timestamppb.New(*dbGS.LastStartedAt)
	}

	return gameServer
}

// getDefaultDockerImage returns the default Docker image for a game type
// These are based on Pterodactyl's game server images
func getDefaultDockerImage(gameType gameserversv1.GameType) string {
	switch gameType {
	case gameserversv1.GameType_MINECRAFT, gameserversv1.GameType_MINECRAFT_JAVA:
		return "ghcr.io/pterodactyl/minecraft:latest"
	case gameserversv1.GameType_MINECRAFT_BEDROCK:
		return "ghcr.io/pterodactyl/minecraft-bedrock:latest"
	case gameserversv1.GameType_VALHEIM:
		return "ghcr.io/pterodactyl/valheim:latest"
	case gameserversv1.GameType_TERRARIA:
		return "ghcr.io/pterodactyl/terraria:latest"
	case gameserversv1.GameType_RUST:
		return "ghcr.io/pterodactyl/rust:latest"
	case gameserversv1.GameType_CS2:
		return "ghcr.io/pterodactyl/csgo:latest" // CS2 may use CSGO image
	case gameserversv1.GameType_TF2:
		return "ghcr.io/pterodactyl/tf2:latest"
	case gameserversv1.GameType_ARK:
		return "ghcr.io/pterodactyl/ark:latest"
	case gameserversv1.GameType_CONAN:
		return "ghcr.io/pterodactyl/conan:latest"
	case gameserversv1.GameType_SEVEN_DAYS:
		return "ghcr.io/pterodactyl/7days:latest"
	case gameserversv1.GameType_FACTORIO:
		return "ghcr.io/pterodactyl/factorio:latest"
	case gameserversv1.GameType_SPACED_ENGINEERS:
		return "ghcr.io/pterodactyl/space-engineers:latest"
	default:
		return "ghcr.io/pterodactyl/generic:latest"
	}
}
