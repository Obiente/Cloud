package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	sharedorchestrator "github.com/obiente/cloud/apps/shared/pkg/orchestrator"
	"github.com/obiente/cloud/apps/shared/pkg/utils"

	"github.com/docker/go-connections/nat"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/filters"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// GameServerManager manages the lifecycle of game server containers
type GameServerManager struct {
	dockerClient client.APIClient
	dockerHelper *docker.Client
	nodeSelector *sharedorchestrator.NodeSelector
	networkName  string
	nodeID       string
	nodeHostname string
	forwarder    *sharedorchestrator.NodeForwarder
}

// GameServerConfig holds configuration for a game server container
type GameServerConfig struct {
	GameServerID string
	Image        string
	Port         int32
	EnvVars      map[string]string
	MemoryBytes  int64 // in bytes
	CPUCores     int32
	StartCommand *string // Optional start command to override container CMD
}

// NewGameServerManager creates a new game server manager
func NewGameServerManager(strategy string, maxGameServersPerNode int) (*GameServerManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	nodeSelector, err := sharedorchestrator.NewNodeSelector(strategy, maxGameServersPerNode)
	if err != nil {
		return nil, fmt.Errorf("failed to create node selector: %w", err)
	}

	// Get node info
	info, err := cli.Info(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker info: %w", err)
	}

	helper, err := docker.New()
	if err != nil {
		return nil, fmt.Errorf("failed to init docker helper: %w", err)
	}

	// Determine node ID - respect ENABLE_SWARM environment variable
	// If ENABLE_SWARM=false, always use local- prefix even if Swarm is enabled in Docker
	var nodeID string
	if utils.IsSwarmModeEnabled() {
		// Swarm mode enabled - use Swarm node ID if available
		nodeID = info.Swarm.NodeID
		if nodeID == "" {
			// Swarm enabled but not in Swarm - use synthetic ID
			nodeID = "local-" + info.Name
		}
	} else {
		// Swarm mode disabled - always use local- prefix
		nodeID = "local-" + info.Name
	}

	gsm := &GameServerManager{
		dockerClient: cli,
		dockerHelper: helper,
		nodeSelector: nodeSelector,
		networkName:  "obiente-network",
		nodeID:       nodeID,
		nodeHostname: info.Name,
		forwarder:    sharedorchestrator.NewNodeForwarder(),
	}

	return gsm, nil
}

// GetNodeID returns the node ID for this game server manager
func (gsm *GameServerManager) GetNodeID() string {
	return gsm.nodeID
}

// getNodeIP retrieves the IP address for the current node from NodeMetadata
func (gsm *GameServerManager) getNodeIP(ctx context.Context) string {
	var node database.NodeMetadata
	if err := database.DB.First(&node, "id = ?", gsm.nodeID).Error; err != nil {
		logger.Warn("[GameServerManager] Failed to get node metadata for node %s: %v", gsm.nodeID, err)
		return ""
	}

	if node.IP != "" {
		return node.IP
	}

	// If IP is not set in NodeMetadata, log a warning
	logger.Warn("[GameServerManager] Node %s (%s) has no IP address configured in NodeMetadata", gsm.nodeID, gsm.nodeHostname)
	return ""
}

// CreateGameServer creates a new game server container
func (gsm *GameServerManager) CreateGameServer(ctx context.Context, config *GameServerConfig) error {
	logger.Info("[GameServerManager] Creating game server container %s", config.GameServerID)

	// Ensure network exists
	if err := gsm.ensureNetwork(ctx); err != nil {
		return fmt.Errorf("network is required but could not be created: %w", err)
	}

	// Select best node for game server
	targetNode, err := gsm.nodeSelector.SelectNode(ctx)
	if err != nil {
		logger.Error("[GameServerManager] Failed to select node for game server %s: %v", config.GameServerID, err)
		return fmt.Errorf("failed to select node: %w", err)
	}

	logger.Info("[GameServerManager] Selected node %s (%s) for game server %s",
		targetNode.ID, targetNode.Hostname, config.GameServerID)

	// Check if we're on the target node
	if targetNode.ID != gsm.nodeID {
		// Try to forward the request to the target node
		if gsm.forwarder.CanForward(targetNode.ID) {
			logger.Info("[GameServerManager] Forwarding game server creation to node %s (%s)",
				targetNode.ID, targetNode.Hostname)
			// For now, we'll proceed on current node since forwarding CreateGameServer
			// requires serializing the config and calling the internal API
			// TODO: Implement full forwarding for CreateGameServer via internal API endpoint
			logger.Warn("[GameServerManager] Node forwarding available but CreateGameServer forwarding not fully implemented. "+
				"Proceeding with game server creation on current node %s", gsm.nodeID)
		} else {
			logger.Warn("[GameServerManager] Cannot forward to node %s (%s) - proceeding with game server creation on current node %s (%s)",
				targetNode.ID, targetNode.Hostname, gsm.nodeID, gsm.nodeHostname)
		}
		// Continue with game server creation on current node
	}

	containerName := fmt.Sprintf("gameserver-%s", config.GameServerID)

	// Remove existing container with this name if it exists (for redeployments)
	if err := gsm.removeContainerByName(ctx, containerName); err != nil {
		logger.Warn("[GameServerManager] Failed to remove existing container %s: %v (will attempt to create anyway)", containerName, err)
	}

	// Create container
	containerID, err := gsm.createContainer(ctx, config, containerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Register game server location
	// Get node IP for the location from node metadata
	nodeIP := gsm.getNodeIP(ctx)

	location := &database.GameServerLocation{
		ID:           fmt.Sprintf("loc-gs-%s-%s", config.GameServerID, containerID[:12]),
		GameServerID: config.GameServerID,
		NodeID:       gsm.nodeID,
		NodeHostname: gsm.nodeHostname,
		NodeIP:       nodeIP,
		ContainerID:  containerID,
		Status:       "created",
		Port:         config.Port,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := database.UpsertGameServerLocation(location); err != nil {
		logger.Warn("[GameServerManager] Failed to register game server location: %v", err)
	}

	// Update database with container info (but keep status as CREATED - user must start manually)
	if err := gsm.updateGameServerContainerInfo(ctx, config.GameServerID, containerID, containerName); err != nil {
		logger.Warn("[GameServerManager] Failed to update game server container info: %v", err)
	}

	// Ensure status is CREATED (not STARTING) - containers are created but not started
	// User must explicitly start the container via StartGameServer
	if err := database.NewGameServerRepository(database.DB, database.RedisClient).UpdateStatus(ctx, config.GameServerID, 0); err != nil { // CREATED = 0
		logger.Warn("[GameServerManager] Failed to ensure status is CREATED: %v", err)
	}

	logger.Info("[GameServerManager] Successfully created container %s for game server %s (status: CREATED)",
		containerID[:12], config.GameServerID)

	return nil
}

// StartGameServer starts a game server container
func (gsm *GameServerManager) StartGameServer(ctx context.Context, gameServerID string) error {
	logger.Info("[GameServerManager] Starting game server %s", gameServerID)

	// Get game server from database
	gameServer, err := database.NewGameServerRepository(database.DB, database.RedisClient).GetByID(ctx, gameServerID)
	if err != nil {
		return fmt.Errorf("failed to get game server: %w", err)
	}

	// If container doesn't exist, create it first
	if gameServer.ContainerID == nil {
		logger.Info("[GameServerManager] Game server %s has no container ID, creating container first", gameServerID)

		// Parse environment variables from JSON
		envVars := make(map[string]string)
		if gameServer.EnvVars != "" {
			if err := json.Unmarshal([]byte(gameServer.EnvVars), &envVars); err != nil {
				logger.Warn("[GameServerManager] Failed to parse env vars for game server %s: %v", gameServerID, err)
				// Continue with empty env vars
			}
		}

		// Build config from database game server
		config := &GameServerConfig{
			GameServerID: gameServerID,
			Image:        gameServer.DockerImage,
			Port:         gameServer.Port,
			EnvVars:      envVars,
			MemoryBytes:  gameServer.MemoryBytes,
			CPUCores:     gameServer.CPUCores,
			StartCommand: gameServer.StartCommand,
		}

		// Create the container
		if err := gsm.CreateGameServer(ctx, config); err != nil {
			return fmt.Errorf("failed to create game server container: %w", err)
		}

		// Refresh game server to get the new container ID
		gameServer, err = database.NewGameServerRepository(database.DB, database.RedisClient).GetByID(ctx, gameServerID)
		if err != nil {
			return fmt.Errorf("failed to refresh game server after creation: %w", err)
		}

		if gameServer.ContainerID == nil {
			return fmt.Errorf("container was created but container ID was not set in database")
		}

		// Restore STARTING status since we're in the middle of starting the server
		// CreateGameServer sets it to CREATED, but we want to keep it as STARTING
		if err := database.NewGameServerRepository(database.DB, database.RedisClient).UpdateStatus(ctx, gameServerID, 2); err != nil { // STARTING = 2
			logger.Warn("[GameServerManager] Failed to restore STARTING status after container creation: %v", err)
		}

		logger.Info("[GameServerManager] Successfully created container %s for game server %s", (*gameServer.ContainerID)[:12], gameServerID)
	}

	// Check if container exists and is stopped
	containerInfo, err := gsm.dockerClient.ContainerInspect(ctx, *gameServer.ContainerID)
	if err != nil {
		// Container doesn't exist - try to recreate it with existing volumes
		logger.Warn("[GameServerManager] Container %s doesn't exist, attempting to recreate it with existing volumes", *gameServer.ContainerID)

		// Parse environment variables from JSON
		envVars := make(map[string]string)
		if gameServer.EnvVars != "" {
			if err := json.Unmarshal([]byte(gameServer.EnvVars), &envVars); err != nil {
				logger.Warn("[GameServerManager] Failed to parse env vars for game server %s: %v", gameServerID, err)
				// Continue with empty env vars
			}
		}

		// Build config from database game server
		config := &GameServerConfig{
			GameServerID: gameServerID,
			Image:        gameServer.DockerImage,
			Port:         gameServer.Port,
			EnvVars:      envVars,
			MemoryBytes:  gameServer.MemoryBytes,
			CPUCores:     gameServer.CPUCores,
			StartCommand: gameServer.StartCommand,
		}

		// Create the container (this will reuse existing volumes)
		if err := gsm.CreateGameServer(ctx, config); err != nil {
			return fmt.Errorf("failed to recreate game server container: %w", err)
		}

		// Refresh game server to get the new container ID
		gameServer, err = database.NewGameServerRepository(database.DB, database.RedisClient).GetByID(ctx, gameServerID)
		if err != nil {
			return fmt.Errorf("failed to refresh game server after recreation: %w", err)
		}

		if gameServer.ContainerID == nil {
			return fmt.Errorf("container was recreated but container ID was not set in database")
		}

		// Restore STARTING status since we're in the middle of starting the server
		// CreateGameServer sets it to CREATED, but we want to keep it as STARTING
		if err := database.NewGameServerRepository(database.DB, database.RedisClient).UpdateStatus(ctx, gameServerID, 2); err != nil { // STARTING = 2
			logger.Warn("[GameServerManager] Failed to restore STARTING status after container recreation: %v", err)
		}

		logger.Info("[GameServerManager] Successfully recreated container %s for game server %s", (*gameServer.ContainerID)[:12], gameServerID)

		// Re-inspect the new container (use the NEW container ID)
		containerInfo, err = gsm.dockerClient.ContainerInspect(ctx, *gameServer.ContainerID)
		if err != nil {
			return fmt.Errorf("failed to inspect recreated container %s: %w", (*gameServer.ContainerID)[:12], err)
		}
	}

	// Only start if not already running
	if !containerInfo.State.Running {
		logger.Info("[GameServerManager] Container %s is not running, starting it...", (*gameServer.ContainerID)[:12])
		if err := gsm.dockerHelper.StartContainer(ctx, *gameServer.ContainerID); err != nil {
			// Check if the error is due to a missing network
			errStr := err.Error()
			if strings.Contains(errStr, "network") && strings.Contains(errStr, "not found") {
				logger.Warn("[GameServerManager] Container %s references a network that no longer exists, recreating container...", (*gameServer.ContainerID)[:12])

				// Remove the old container
				containerName := fmt.Sprintf("gameserver-%s", gameServerID)
				if err := gsm.removeContainerByName(ctx, containerName); err != nil {
					logger.Warn("[GameServerManager] Failed to remove old container %s: %v", containerName, err)
				}

				// Clear container ID from database so it gets recreated
				repo := database.NewGameServerRepository(database.DB, database.RedisClient)
				if err := repo.UpdateContainerInfo(ctx, gameServerID, nil, nil); err != nil {
					logger.Warn("[GameServerManager] Failed to clear container ID: %v", err)
				}

				// Parse environment variables from JSON
				envVars := make(map[string]string)
				if gameServer.EnvVars != "" {
					if err := json.Unmarshal([]byte(gameServer.EnvVars), &envVars); err != nil {
						logger.Warn("[GameServerManager] Failed to parse env vars for game server %s: %v", gameServerID, err)
					}
				}

				// Build config from database game server
				config := &GameServerConfig{
					GameServerID: gameServerID,
					Image:        gameServer.DockerImage,
					Port:         gameServer.Port,
					EnvVars:      envVars,
					MemoryBytes:  gameServer.MemoryBytes,
					CPUCores:     gameServer.CPUCores,
					StartCommand: gameServer.StartCommand,
				}

				// Recreate the container with current network
				if err := gsm.CreateGameServer(ctx, config); err != nil {
					_ = repo.UpdateStatus(ctx, gameServerID, 7) // FAILED = 7
					return fmt.Errorf("failed to recreate container after network error: %w", err)
				}

				// Refresh game server to get the new container ID
				gameServer, err = repo.GetByID(ctx, gameServerID)
				if err != nil {
					return fmt.Errorf("failed to refresh game server after recreation: %w", err)
				}

				if gameServer.ContainerID == nil {
					return fmt.Errorf("container was recreated but container ID was not set in database")
				}

				// Restore STARTING status since we're in the middle of starting the server
				// CreateGameServer sets it to CREATED, but we want to keep it as STARTING
				if err := repo.UpdateStatus(ctx, gameServerID, 2); err != nil { // STARTING = 2
					logger.Warn("[GameServerManager] Failed to restore STARTING status after network error recreation: %v", err)
				}

				logger.Info("[GameServerManager] Successfully recreated container %s for game server %s", (*gameServer.ContainerID)[:12], gameServerID)

				// Now try to start the recreated container
				if err := gsm.dockerHelper.StartContainer(ctx, *gameServer.ContainerID); err != nil {
					logger.Error("[GameServerManager] Failed to start recreated container %s: %v", (*gameServer.ContainerID)[:12], err)
					_ = repo.UpdateStatus(ctx, gameServerID, 7) // FAILED = 7
					return fmt.Errorf("failed to start recreated container: %w", err)
				}

				logger.Info("[GameServerManager] Successfully started recreated container %s", (*gameServer.ContainerID)[:12])
			} else {
				logger.Error("[GameServerManager] Failed to start container %s: %v", (*gameServer.ContainerID)[:12], err)
				// Update status to FAILED if start fails
				_ = database.NewGameServerRepository(database.DB, database.RedisClient).UpdateStatus(ctx, gameServerID, 7) // FAILED = 7
				return fmt.Errorf("failed to start container: %w", err)
			}
		}

		// Container started successfully (either original or recreated)
		logger.Info("[GameServerManager] Successfully started container %s", (*gameServer.ContainerID)[:12])

		// Wait a moment for container to initialize, then verify it's actually running
		// Some containers may exit immediately if misconfigured
		time.Sleep(2 * time.Second)

		// Re-inspect to check if container is still running
		containerInfo, err = gsm.dockerClient.ContainerInspect(ctx, *gameServer.ContainerID)
		if err != nil {
			return fmt.Errorf("failed to inspect container after start: %w", err)
		}

		// If container exited, check why and update status accordingly
		if !containerInfo.State.Running {
			exitCode := containerInfo.State.ExitCode
			logger.Warn("[GameServerManager] Container %s exited immediately with code %d", (*gameServer.ContainerID)[:12], exitCode)

			// Try to get container logs for debugging
			logs, logErr := gsm.dockerClient.ContainerLogs(ctx, *gameServer.ContainerID, client.ContainerLogsOptions{
				ShowStdout: true,
				ShowStderr: true,
				Tail:       "50",
			})
			if logErr == nil {
				defer logs.Close()
				logContent, _ := io.ReadAll(logs)
				if len(logContent) > 0 {
					logger.Warn("[GameServerManager] Container %s logs (last 50 lines):\n%s", (*gameServer.ContainerID)[:12], string(logContent))
				}
			}

			// Update status to STOPPED since container exited
			if err := database.NewGameServerRepository(database.DB, database.RedisClient).UpdateStatus(ctx, gameServerID, 5); err != nil { // STOPPED = 5
				logger.Warn("[GameServerManager] Failed to update game server status to STOPPED: %v", err)
			}

			// Update location status
			if err := database.DB.Model(&database.GameServerLocation{}).
				Where("container_id = ?", *gameServer.ContainerID).
				Update("status", "stopped").Error; err != nil {
				logger.Warn("[GameServerManager] Failed to update location status: %v", err)
			}

			return fmt.Errorf("container exited immediately with code %d (check container logs and configuration)", exitCode)
		}
	} else {
		logger.Info("[GameServerManager] Container %s is already running", (*gameServer.ContainerID)[:12])
	}

	// Update location status and ensure nodeIP is set
	nodeIP := gsm.getNodeIP(ctx)
	updateData := map[string]interface{}{
		"status": "running",
	}
	if nodeIP != "" {
		updateData["node_ip"] = nodeIP
	}
	if err := database.DB.Model(&database.GameServerLocation{}).
		Where("container_id = ?", *gameServer.ContainerID).
		Updates(updateData).Error; err != nil {
		logger.Warn("[GameServerManager] Failed to update location status: %v", err)
	}

	// Update status
	if err := database.NewGameServerRepository(database.DB, database.RedisClient).UpdateStatus(ctx, gameServerID, 3); err != nil { // RUNNING = 3
		logger.Warn("[GameServerManager] Failed to update game server status: %v", err)
	}

	// Update last started timestamp
	now := time.Now()
	if err := database.DB.Model(&database.GameServer{}).
		Where("id = ?", gameServerID).
		Update("last_started_at", now).Error; err != nil {
		logger.Warn("[GameServerManager] Failed to update last_started_at: %v", err)
	}

	return nil
}

// StopGameServer stops a game server container
func (gsm *GameServerManager) StopGameServer(ctx context.Context, gameServerID string) error {
	logger.Info("[GameServerManager] Stopping game server %s", gameServerID)

	// Get game server from database
	gameServer, err := database.NewGameServerRepository(database.DB, database.RedisClient).GetByID(ctx, gameServerID)
	if err != nil {
		return fmt.Errorf("failed to get game server: %w", err)
	}

	if gameServer.ContainerID == nil {
		return fmt.Errorf("game server %s has no container ID", gameServerID)
	}

	// Check if container exists and is running
	containerInfo, err := gsm.dockerClient.ContainerInspect(ctx, *gameServer.ContainerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	// Only stop if running
	if containerInfo.State.Running {
		timeout := 30 * time.Second // Game servers may need more time to shut down gracefully
		if err := gsm.dockerHelper.StopContainer(ctx, *gameServer.ContainerID, timeout); err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}
		logger.Info("[GameServerManager] Stopped container %s", (*gameServer.ContainerID)[:12])
	} else {
		logger.Info("[GameServerManager] Container %s is already stopped", (*gameServer.ContainerID)[:12])
	}

	// Update location status
	if err := database.DB.Model(&database.GameServerLocation{}).
		Where("container_id = ?", *gameServer.ContainerID).
		Update("status", "stopped").Error; err != nil {
		logger.Warn("[GameServerManager] Failed to update location status: %v", err)
	}

	// Update status
	if err := database.NewGameServerRepository(database.DB, database.RedisClient).UpdateStatus(ctx, gameServerID, 5); err != nil { // STOPPED = 5
		logger.Warn("[GameServerManager] Failed to update game server status: %v", err)
	}

	return nil
}

// RestartGameServer restarts a game server container
func (gsm *GameServerManager) RestartGameServer(ctx context.Context, gameServerID string) error {
	logger.Info("[GameServerManager] Restarting game server %s", gameServerID)

	// Get game server from database
	gameServer, err := database.NewGameServerRepository(database.DB, database.RedisClient).GetByID(ctx, gameServerID)
	if err != nil {
		return fmt.Errorf("failed to get game server: %w", err)
	}

	if gameServer.ContainerID == nil {
		// No container ID - try to start the game server (which will create the container)
		logger.Info("[GameServerManager] Game server %s has no container ID, starting instead of restarting", gameServerID)
		return gsm.StartGameServer(ctx, gameServerID)
	}

	// Check if container exists
	containerInfo, err := gsm.dockerClient.ContainerInspect(ctx, *gameServer.ContainerID)
	if err != nil {
		// Container doesn't exist - try to start the game server (which will recreate the container)
		logger.Warn("[GameServerManager] Container %s doesn't exist, starting game server instead (will recreate container)", (*gameServer.ContainerID)[:12])
		return gsm.StartGameServer(ctx, gameServerID)
	}

	// If container is not running, just start it instead of restarting
	if !containerInfo.State.Running {
		logger.Info("[GameServerManager] Container %s is not running, starting instead of restarting", (*gameServer.ContainerID)[:12])
		return gsm.StartGameServer(ctx, gameServerID)
	}

	// Container exists and is running - restart it
	timeout := 30 * time.Second
	if err := gsm.dockerHelper.RestartContainer(ctx, *gameServer.ContainerID, timeout); err != nil {
		// If restart fails, try to stop and start instead
		logger.Warn("[GameServerManager] Failed to restart container %s: %v, trying stop+start instead", (*gameServer.ContainerID)[:12], err)

		// Stop the container first
		if stopErr := gsm.dockerHelper.StopContainer(ctx, *gameServer.ContainerID, timeout); stopErr != nil {
			logger.Warn("[GameServerManager] Failed to stop container %s: %v", (*gameServer.ContainerID)[:12], stopErr)
		}

		// Then start it
		if startErr := gsm.dockerHelper.StartContainer(ctx, *gameServer.ContainerID); startErr != nil {
			// If start also fails, try the full StartGameServer flow which handles network errors
			logger.Warn("[GameServerManager] Failed to start container %s after stop: %v, using StartGameServer flow", (*gameServer.ContainerID)[:12], startErr)
			return gsm.StartGameServer(ctx, gameServerID)
		}

		logger.Info("[GameServerManager] Successfully stopped and started container %s", (*gameServer.ContainerID)[:12])
	} else {
		logger.Info("[GameServerManager] Restarted container %s", (*gameServer.ContainerID)[:12])
	}

	// Update status
	if err := database.NewGameServerRepository(database.DB, database.RedisClient).UpdateStatus(ctx, gameServerID, 3); err != nil { // RUNNING = 3
		logger.Warn("[GameServerManager] Failed to update game server status: %v", err)
	}

	// Update last started timestamp
	now := time.Now()
	if err := database.DB.Model(&database.GameServer{}).
		Where("id = ?", gameServerID).
		Update("last_started_at", now).Error; err != nil {
		logger.Warn("[GameServerManager] Failed to update last_started_at: %v", err)
	}

	return nil
}

// DeleteGameServer removes a game server container
func (gsm *GameServerManager) DeleteGameServer(ctx context.Context, gameServerID string) error {
	logger.Info("[GameServerManager] Deleting game server %s", gameServerID)

	// Get game server from database
	gameServer, err := database.NewGameServerRepository(database.DB, database.RedisClient).GetByID(ctx, gameServerID)
	if err != nil {
		return fmt.Errorf("failed to get game server: %w", err)
	}

	if gameServer.ContainerID == nil {
		// No container to delete, but that's okay
		logger.Info("[GameServerManager] Game server %s has no container ID, skipping container deletion", gameServerID)
		return nil
	}

	// SECURITY: Verify container was created by our API before deletion
	containerInfo, err := gsm.dockerClient.ContainerInspect(ctx, *gameServer.ContainerID)
	if err != nil {
		logger.Warn("[GameServerManager] Container %s not found (may already be deleted): %v", (*gameServer.ContainerID)[:12], err)
	} else {
		// Verify container has our management label
		if containerInfo.Config.Labels["cloud.obiente.managed"] != "true" {
			logger.Error("[GameServerManager] SECURITY: Refusing to delete container %s: not managed by Obiente Cloud (missing cloud.obiente.managed=true label)", (*gameServer.ContainerID)[:12])
			return fmt.Errorf("refusing to delete container: not managed by Obiente Cloud")
		}

		// Stop container first if it's running
		if containerInfo.State.Running {
			timeout := 30 * time.Second
			if err := gsm.dockerHelper.StopContainer(ctx, *gameServer.ContainerID, timeout); err != nil {
				logger.Warn("[GameServerManager] Failed to stop container before deletion: %v", err)
			}
		}
	}

	// Remove container
	if err := gsm.dockerHelper.RemoveContainer(ctx, *gameServer.ContainerID, true); err != nil {
		logger.Warn("[GameServerManager] Failed to remove container: %v", err)
		// Don't fail the operation if container removal fails - it might already be removed
	}

	logger.Info("[GameServerManager] Deleted container %s", (*gameServer.ContainerID)[:12])
	return nil
}

// GetGameServerLogs retrieves logs for a game server container
func (gsm *GameServerManager) GetGameServerLogs(ctx context.Context, gameServerID string, tail string, follow bool, since *time.Time, until *time.Time) (io.ReadCloser, error) {
	// Get game server from database
	gameServer, err := database.NewGameServerRepository(database.DB, database.RedisClient).GetByID(ctx, gameServerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get game server: %w", err)
	}

	if gameServer.ContainerID == nil {
		return nil, fmt.Errorf("game server %s has no container ID (container may not have been created yet)", gameServerID)
	}

	// Check if container exists before trying to get logs
	_, err = gsm.dockerClient.ContainerInspect(ctx, *gameServer.ContainerID)
	if err != nil {
		return nil, fmt.Errorf("container %s not found (may have been deleted): %w", (*gameServer.ContainerID)[:12], err)
	}

	return gsm.dockerHelper.ContainerLogs(ctx, *gameServer.ContainerID, tail, follow, since, until)
}

// sendCommandViaAttach uses Docker attach API to send command to container's main process stdin
func (gsm *GameServerManager) sendCommandViaAttach(ctx context.Context, containerID string, command string) error {
	// Create a context with timeout for the attach operation
	attachCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Attach to container with stdin, stdout, stderr
	attachResp, err := gsm.dockerClient.ContainerAttach(attachCtx, containerID, client.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: false, // We don't need to read output
		Stderr: false,
	})
	if err != nil {
		return fmt.Errorf("failed to attach to container: %w", err)
	}
	defer attachResp.Close()

	// Write command with newline to stdin
	commandWithNewline := command + "\n"
	n, err := attachResp.Conn.Write([]byte(commandWithNewline))
	if err != nil {
		return fmt.Errorf("failed to write command to stdin (wrote %d bytes): %w", n, err)
	}

	// Flush the connection to ensure data is sent
	if flusher, ok := attachResp.Conn.(interface{ Flush() error }); ok {
		if err := flusher.Flush(); err != nil {
			logger.Debug("[GameServerManager] Failed to flush stdin connection: %v", err)
		}
	}

	logger.Debug("[GameServerManager] Successfully wrote %d bytes to container stdin", n)
	return nil
}

// ensureNetwork ensures the obiente-network exists (reuses DeploymentManager logic)
func (gsm *GameServerManager) ensureNetwork(ctx context.Context) error {
	// Reuse the same network as deployments
	// Check if network exists
	networks, err := gsm.dockerClient.NetworkList(ctx, client.NetworkListOptions{
		Filters: filters.NewArgs(filters.Arg("name", gsm.networkName)),
	})
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	if len(networks) > 0 {
		logger.Debug("[GameServerManager] Network %s already exists", gsm.networkName)
		return nil
	}

	// Network doesn't exist - create it
	logger.Info("[GameServerManager] Creating network %s", gsm.networkName)
	_, err = gsm.dockerClient.NetworkCreate(ctx, gsm.networkName, client.NetworkCreateOptions{
		Driver: "bridge",
		Labels: map[string]string{
			"cloud.obiente.managed": "true",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	logger.Info("[GameServerManager] Successfully created network %s", gsm.networkName)
	return nil
}

// removeContainerByName removes a container by name if it exists
// SECURITY: Only removes containers that were created by our API (have cloud.obiente.managed=true label)
func (gsm *GameServerManager) removeContainerByName(ctx context.Context, containerName string) error {
	// Try to inspect container directly by name
	containerNameWithSlash := "/" + containerName
	containerNameWithoutSlash := strings.TrimPrefix(containerName, "/")

	for _, nameToTry := range []string{containerNameWithSlash, containerNameWithoutSlash} {
		containerInfo, err := gsm.dockerClient.ContainerInspect(ctx, nameToTry)
		if err == nil {
			// SECURITY: Verify container was created by our API
			if containerInfo.Config.Labels["cloud.obiente.managed"] != "true" {
				return fmt.Errorf("refusing to delete container %s: not managed by Obiente Cloud (missing cloud.obiente.managed=true label)", containerName)
			}

			logger.Info("[GameServerManager] Removing existing container %s (ID: %s)", containerName, containerInfo.ID[:12])

			// Stop container first
			timeout := 30 * time.Second
			_ = gsm.dockerHelper.StopContainer(ctx, containerInfo.ID, timeout)

			// Remove container
			if err := gsm.dockerHelper.RemoveContainer(ctx, containerInfo.ID, true); err != nil {
				return fmt.Errorf("failed to remove existing container %s: %w", containerName, err)
			}

			logger.Info("[GameServerManager] Successfully removed existing container %s", containerName)
			return nil
		}
	}

	return nil // Container doesn't exist, which is fine
}

// createContainer creates a single game server container
func (gsm *GameServerManager) createContainer(ctx context.Context, config *GameServerConfig, name string) (string, error) {
	// Pull image if it doesn't exist locally
	if err := gsm.ensureImage(ctx, config.Image); err != nil {
		return "", fmt.Errorf("failed to pull image %s: %w", config.Image, err)
	}

	// Prepare environment variables
	env := []string{}
	for key, value := range config.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	// Default environment variables for game servers (Pterodactyl style)
	env = append(env, "SERVER_PORT="+strconv.Itoa(int(config.Port)))
	env = append(env, "SERVER_MAX_PLAYERS=20") // Default, can be overridden via env vars

	// Ensure game server binds to all interfaces (0.0.0.0) inside the container
	// This allows connections from outside the container via Docker port mapping
	// Only set if not already provided by user (user can override if needed)
	if _, exists := config.EnvVars["SERVER_IP"]; !exists {
		env = append(env, "SERVER_IP=0.0.0.0")
	}
	if _, exists := config.EnvVars["HOST"]; !exists {
		env = append(env, "HOST=0.0.0.0")
	}
	if _, exists := config.EnvVars["BIND_IP"]; !exists {
		env = append(env, "BIND_IP=0.0.0.0")
	}

	// Prepare labels
	labels := map[string]string{
		"cloud.obiente.managed":       "true",
		"cloud.obiente.resource_type": "gameserver",
		"cloud.obiente.gameserver_id": config.GameServerID,
		"cloud.obiente.node_id":       gsm.nodeID,
		"cloud.obiente.node_hostname": gsm.nodeHostname,
		"cloud.obiente.metrics":       "true", // Enable metrics collection
	}

	// Port configuration
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}
	containerPort := nat.Port(fmt.Sprintf("%d/tcp", config.Port))
	exposedPorts[containerPort] = struct{}{}
	portBindings[containerPort] = []nat.PortBinding{
		{
			HostIP:   "0.0.0.0",
			HostPort: strconv.Itoa(int(config.Port)),
		},
	}

	// Container configuration
	containerConfig := &container.Config{
		Image:        config.Image,
		Env:          env,
		Labels:       labels,
		ExposedPorts: exposedPorts,
		OpenStdin:    true, // Enable stdin for command sending
		Tty:          true, // Enable TTY for proper terminal support and tab completion
		// Don't clear ENTRYPOINT - let the image use its default entrypoint
		// Most game server images have proper entrypoints configured
	}

	// Override container CMD if start command is provided
	if config.StartCommand != nil && *config.StartCommand != "" {
		containerConfig.Entrypoint = []string{}
		containerConfig.Cmd = []string{"sh", "-c", "exec " + *config.StartCommand}
	}

	// Convert CPU cores to CPU shares (Docker uses shares, not cores)
	// 1024 shares = 1 CPU core, so multiply by 1024
	cpuShares := int64(config.CPUCores) * 1024

	// Create or get volume for game server data persistence
	// Most game server images (especially itzg/minecraft-server) require /data mount
	// We use bind mounts to /var/lib/obiente/volumes so the API can access files directly
	volumeName := fmt.Sprintf("gameserver-%s-data", config.GameServerID)
	volumeMountPoint := "/data" // Standard mount point for most game server images
	volumeHostPath := fmt.Sprintf("/var/lib/obiente/volumes/%s", volumeName)

	// Ensure volume directory exists
	if err := gsm.ensureVolume(ctx, volumeName); err != nil {
		return "", fmt.Errorf("failed to ensure volume: %w", err)
	}

	// Configure bind mount (host path -> container path)
	binds := []string{
		fmt.Sprintf("%s:%s", volumeHostPath, volumeMountPoint),
	}

	// Host configuration
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Binds:        binds, // Mount volume for persistent data
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		Resources: container.Resources{
			Memory:    config.MemoryBytes,
			CPUShares: cpuShares,
		},
		NetworkMode: container.NetworkMode(gsm.networkName),
		Privileged:  false, // Never run game servers in privileged mode
	}

	// Network configuration
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			gsm.networkName: {},
		},
	}

	// Create container
	resp, err := gsm.dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, name)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

// ensureVolume ensures the volume directory exists in /var/lib/obiente/volumes
// We use bind mounts instead of Docker named volumes so the API can access files directly
func (gsm *GameServerManager) ensureVolume(ctx context.Context, volumeName string) error {
	// Volume path in /var/lib/obiente/volumes
	volumePath := fmt.Sprintf("/var/lib/obiente/volumes/%s", volumeName)

	// Check if directory already exists
	if _, err := os.Stat(volumePath); err == nil {
		// Directory exists
		logger.Debug("[GameServerManager] Volume directory %s already exists", volumePath)
		return nil
	}

	// Create the volume directory with proper permissions
	if err := os.MkdirAll(volumePath, 0755); err != nil {
		return fmt.Errorf("failed to create volume directory %s: %w", volumePath, err)
	}

	logger.Info("[GameServerManager] Created volume directory %s for game server data", volumePath)
	return nil
}

// ensureImage pulls the Docker image if it doesn't exist locally
func (gsm *GameServerManager) ensureImage(ctx context.Context, imageName string) error {
	// Use a longer timeout for image operations (image pulls can take a while)
	// Create a new context with timeout that's independent of the request context
	// This prevents context cancellation from interrupting long-running image pulls
	imageCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Check if image exists locally
	images, err := gsm.dockerClient.ImageList(imageCtx, client.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", imageName)),
	})
	if err != nil {
		logger.Warn("[GameServerManager] Failed to check for local image %s: %v", imageName, err)
		// Continue to try pulling anyway
	}

	if len(images) > 0 {
		logger.Debug("[GameServerManager] Image %s already exists locally", imageName)
		return nil
	}

	// Image doesn't exist, pull it
	logger.Info("[GameServerManager] Pulling image %s...", imageName)
	pullOptions := client.ImagePullOptions{
		All: false,
	}

	pullReader, err := gsm.dockerClient.ImagePull(imageCtx, imageName, pullOptions)
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w (image may not exist or may require authentication)", imageName, err)
	}
	defer pullReader.Close()

	// Read the pull output to completion (Docker streams progress)
	// Also check for errors in the stream
	buf := make([]byte, 1024)
	var pullError error
	for {
		n, err := pullReader.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Warn("[GameServerManager] Error reading pull output: %v", err)
			pullError = err
			break
		}
		// Log pull progress (Docker outputs JSON lines)
		if n > 0 {
			logger.Debug("[GameServerManager] Pull progress: %s", string(buf[:n]))
		}
	}

	// If we got an error reading the stream, return it
	if pullError != nil {
		return fmt.Errorf("failed to read pull output: %w", pullError)
	}

	logger.Info("[GameServerManager] Successfully pulled image %s", imageName)
	return nil
}

// updateGameServerContainerInfo updates the game server record with container information
// Note: This function does NOT update status to avoid overwriting STARTING/STOPPING states
// Status should be managed by the StartGameServer/StopGameServer functions
func (gsm *GameServerManager) updateGameServerContainerInfo(ctx context.Context, gameServerID, containerID, containerName string) error {
	updates := map[string]interface{}{
		"container_id":   containerID,
		"container_name": containerName,
		"updated_at":     time.Now(),
		// Do NOT update status here - let StartGameServer/StopGameServer manage status
		// This prevents overwriting STARTING/STOPPING states during container creation
	}

	return database.DB.Model(&database.GameServer{}).
		Where("id = ?", gameServerID).
		Updates(updates).Error
}
