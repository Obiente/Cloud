package gameservers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/orchestrator"

	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"

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

	// Parse environment variables for container creation
	envVars := make(map[string]string)
	if len(req.Msg.GetEnvVars()) > 0 {
		envVars = req.Msg.GetEnvVars()
	}

	// Add game-specific default environment variables
	var serverVersion *string
	if sv := req.Msg.GetServerVersion(); sv != "" {
		serverVersion = &sv
	}
	addGameSpecificEnvVars(envVars, req.Msg.GetGameType(), serverVersion)

	// Create Docker container using orchestrator
	manager, err := s.getGameServerManager()
	if err != nil {
		// Log error but don't fail - container can be created later when starting
		logger.Warn("[GameServerService] Failed to get game server manager during creation: %v", err)
	} else {
		config := &orchestrator.GameServerConfig{
			GameServerID: id,
			Image:        dockerImage,
			Port:         port,
			EnvVars:      envVars,
			MemoryBytes:  memoryBytes,
			CPUCores:     cpuCores,
			StartCommand: req.Msg.StartCommand,
		}

		if err := manager.CreateGameServer(ctx, config); err != nil {
			// Update status to FAILED if container creation fails
			_ = s.repo.UpdateStatus(ctx, id, int32(gameserversv1.GameServerStatus_FAILED))
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create game server container: %w", err))
		}

		// Update storage after container is created
		go func() {
			// Use background context for async storage update
			bgCtx := context.Background()
			if err := s.updateGameServerStorage(bgCtx, id); err != nil {
				logger.Warn("[CreateGameServer] Failed to update storage for game server %s: %v", id, err)
			}
		}()
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

	// Stop and remove container if running
	manager, err := s.getGameServerManager()
	if err == nil {
		// Try to delete container, but don't fail if it doesn't exist or is already removed
		if err := manager.DeleteGameServer(ctx, gameServerID); err != nil {
			logger.Warn("[GameServerService] Failed to delete game server container: %v", err)
		}
	}

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

	// Update status to STARTING
	if err := s.repo.UpdateStatus(ctx, gameServerID, int32(gameserversv1.GameServerStatus_STARTING)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update status: %w", err))
	}

	// Start Docker container using orchestrator
	manager, err := s.getGameServerManager()
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, gameServerID, int32(gameserversv1.GameServerStatus_FAILED))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get game server manager: %w", err))
	}

	if err := manager.StartGameServer(ctx, gameServerID); err != nil {
		_ = s.repo.UpdateStatus(ctx, gameServerID, int32(gameserversv1.GameServerStatus_FAILED))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start game server container: %w", err))
	}

	// Update storage after container is started
	go func() {
		// Use background context for async storage update
		bgCtx := context.Background()
		if err := s.updateGameServerStorage(bgCtx, gameServerID); err != nil {
			logger.Warn("[StartGameServer] Failed to update storage for game server %s: %v", gameServerID, err)
		}
	}()

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

	// Stop Docker container using orchestrator
	manager, err := s.getGameServerManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get game server manager: %w", err))
	}

	if err := manager.StopGameServer(ctx, gameServerID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stop game server container: %w", err))
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

	// Update status to RESTARTING
	if err := s.repo.UpdateStatus(ctx, gameServerID, int32(gameserversv1.GameServerStatus_RESTARTING)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update status: %w", err))
	}

	// Restart Docker container using orchestrator
	manager, err := s.getGameServerManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get game server manager: %w", err))
	}

	if err := manager.RestartGameServer(ctx, gameServerID); err != nil {
		_ = s.repo.UpdateStatus(ctx, gameServerID, int32(gameserversv1.GameServerStatus_FAILED))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to restart game server container: %w", err))
	}

	// Update storage after container is restarted
	go func() {
		// Use background context for async storage update
		bgCtx := context.Background()
		if err := s.updateGameServerStorage(bgCtx, gameServerID); err != nil {
			logger.Warn("[RestartGameServer] Failed to update storage for game server %s: %v", gameServerID, err)
		}
	}()

	// Fetch updated game server
	startResp, err := s.GetGameServer(ctx, connect.NewRequest(&gameserversv1.GetGameServerRequest{
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
// Uses commonly available game server images from Docker Hub
func getDefaultDockerImage(gameType gameserversv1.GameType) string {
	switch gameType {
	case gameserversv1.GameType_MINECRAFT, gameserversv1.GameType_MINECRAFT_JAVA:
		// itzg/minecraft-server is the most popular Minecraft server image
		// Using a specific version tag for better stability (Java 21, supports all modern versions)
		return "itzg/minecraft-server:java21"
	case gameserversv1.GameType_MINECRAFT_BEDROCK:
		// itzg/minecraft-bedrock-server for Bedrock Edition
		return "itzg/minecraft-bedrock-server:latest"
	case gameserversv1.GameType_VALHEIM:
		// lloesche/valheim-server is a popular Valheim server image
		return "lloesche/valheim-server:latest"
	case gameserversv1.GameType_TERRARIA:
		// beardedio/terraria is a well-maintained Terraria server image
		// Alternative: ryshe/terraria (if beardedio doesn't work)
		return "beardedio/terraria:latest"
	case gameserversv1.GameType_RUST:
		// didstopia/rust-server is a popular Rust server image
		return "didstopia/rust-server:latest"
	case gameserversv1.GameType_CS2:
		// CS2 server - joedwards32/cs2 is the most popular and well-maintained CS2 server image
		// Requires SRCDS_TOKEN environment variable for Steam authentication
		// See: https://github.com/joedwards32/CS2
		return "joedwards32/cs2:latest"
	case gameserversv1.GameType_TF2:
		// TF2 server - using cm2network image (well-maintained community image)
		// Alternative: joedwards32/tf2 (if available)
		return "cm2network/tf2:latest"
	case gameserversv1.GameType_ARK:
		// didstopia/ark-server is a popular ARK server image
		return "didstopia/ark-server:latest"
	case gameserversv1.GameType_CONAN:
		// didstopia/conan-exiles-server is a popular Conan Exiles server image
		return "didstopia/conan-exiles-server:latest"
	case gameserversv1.GameType_SEVEN_DAYS:
		// didstopia/7dtd-server is a popular 7 Days to Die server image
		return "didstopia/7dtd-server:latest"
	case gameserversv1.GameType_FACTORIO:
		// factoriotools/factorio is the official Factorio server image
		return "factoriotools/factorio:latest"
	case gameserversv1.GameType_SPACED_ENGINEERS:
		// spaceengineers/space-engineers is a Space Engineers server image
		return "spaceengineers/space-engineers:latest"
	default:
		// Use a generic Linux image as fallback
		return "alpine:latest"
	}
}

// addGameSpecificEnvVars adds default environment variables for specific game types
func addGameSpecificEnvVars(envVars map[string]string, gameType gameserversv1.GameType, serverVersion *string) {
	switch gameType {
	case gameserversv1.GameType_MINECRAFT, gameserversv1.GameType_MINECRAFT_JAVA:
		// itzg/minecraft-server requires EULA=TRUE
		if _, exists := envVars["EULA"]; !exists {
			envVars["EULA"] = "TRUE"
		}
		// Set version if provided
		if serverVersion != nil && *serverVersion != "" {
			if _, exists := envVars["VERSION"]; !exists {
				envVars["VERSION"] = *serverVersion
			}
		}
		// Default to VANILLA server type if not specified
		if _, exists := envVars["TYPE"]; !exists {
			envVars["TYPE"] = "VANILLA"
		}
	case gameserversv1.GameType_MINECRAFT_BEDROCK:
		// itzg/minecraft-bedrock-server requires EULA=TRUE
		if _, exists := envVars["EULA"]; !exists {
			envVars["EULA"] = "TRUE"
		}
		if serverVersion != nil && *serverVersion != "" {
			if _, exists := envVars["VERSION"]; !exists {
				envVars["VERSION"] = *serverVersion
			}
		}
	case gameserversv1.GameType_CS2:
		// joedwards32/cs2 requires SRCDS_TOKEN for Steam authentication
		// Note: Users must provide their own Steam Game Server Login Token
		// The token can be obtained from: https://steamcommunity.com/dev/managegameservers
		// We don't set a default value here - users must configure it
		// But we document the requirement in comments
	case gameserversv1.GameType_TF2:
		// cm2network/tf2 may require SRCDS_TOKEN for Steam authentication
		// Similar to CS2, users should configure this if needed
	}
}
