package orchestrator

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"api/docker"
	"api/internal/database"
	"api/internal/logger"

	"github.com/docker/go-connections/nat"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/filters"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// GameServerManager manages the lifecycle of game server containers
type GameServerManager struct {
	dockerClient client.APIClient
	dockerHelper dockerHelper
	nodeSelector *NodeSelector
	networkName  string
	nodeID       string
	nodeHostname string
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

	nodeSelector, err := NewNodeSelector(strategy, maxGameServersPerNode)
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

	// Determine node ID - use Swarm node ID if available, otherwise use synthetic local ID
	nodeID := info.Swarm.NodeID
	if nodeID == "" {
		nodeID = "local-" + info.Name
	}

	gsm := &GameServerManager{
		dockerClient: cli,
		dockerHelper: helper,
		nodeSelector: nodeSelector,
		networkName:  "obiente-network",
		nodeID:       nodeID,
		nodeHostname: info.Name,
	}

	return gsm, nil
}

// GetNodeID returns the node ID for this game server manager
func (gsm *GameServerManager) GetNodeID() string {
	return gsm.nodeID
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
		// TODO: Forward request to the correct node's API
		return fmt.Errorf("game server should be created on node %s, but we're on %s",
			targetNode.ID, gsm.nodeID)
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
	// Get node IP for the location - try to get from node metadata
	// Note: NodeMetadata.Address is JSONB and might contain IP info
	// For now, we'll leave NodeIP empty and let DNS resolve it via hostname
	nodeIP := ""

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

	// Update database with container info
	if err := gsm.updateGameServerContainerInfo(ctx, config.GameServerID, containerID, containerName); err != nil {
		logger.Warn("[GameServerManager] Failed to update game server container info: %v", err)
	}

	logger.Info("[GameServerManager] Successfully created container %s for game server %s",
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

	if gameServer.ContainerID == nil {
		return fmt.Errorf("game server %s has no container ID - may need to be created first", gameServerID)
	}

	// Check if container exists and is stopped
	containerInfo, err := gsm.dockerClient.ContainerInspect(ctx, *gameServer.ContainerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	// Only start if not already running
	if !containerInfo.State.Running {
		if err := gsm.dockerHelper.StartContainer(ctx, *gameServer.ContainerID); err != nil {
			return fmt.Errorf("failed to start container: %w", err)
		}
		logger.Info("[GameServerManager] Started container %s", (*gameServer.ContainerID)[:12])
	} else {
		logger.Info("[GameServerManager] Container %s is already running", (*gameServer.ContainerID)[:12])
	}

	// Update location status
	if err := database.DB.Model(&database.GameServerLocation{}).
		Where("container_id = ?", *gameServer.ContainerID).
		Update("status", "running").Error; err != nil {
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
		return fmt.Errorf("game server %s has no container ID", gameServerID)
	}

	timeout := 30 * time.Second
	if err := gsm.dockerHelper.RestartContainer(ctx, *gameServer.ContainerID, timeout); err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	logger.Info("[GameServerManager] Restarted container %s", (*gameServer.ContainerID)[:12])

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

	// Stop container first
	containerInfo, err := gsm.dockerClient.ContainerInspect(ctx, *gameServer.ContainerID)
	if err == nil && containerInfo.State.Running {
		timeout := 30 * time.Second
		if err := gsm.dockerHelper.StopContainer(ctx, *gameServer.ContainerID, timeout); err != nil {
			logger.Warn("[GameServerManager] Failed to stop container before deletion: %v", err)
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
func (gsm *GameServerManager) GetGameServerLogs(ctx context.Context, gameServerID string, tail string, follow bool) (io.ReadCloser, error) {
	// Get game server from database
	gameServer, err := database.NewGameServerRepository(database.DB, database.RedisClient).GetByID(ctx, gameServerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get game server: %w", err)
	}

	if gameServer.ContainerID == nil {
		return nil, fmt.Errorf("game server %s has no container ID", gameServerID)
	}

	return gsm.dockerHelper.ContainerLogs(ctx, *gameServer.ContainerID, tail, follow)
}

// SendGameServerCommand sends a command to a running game server container
func (gsm *GameServerManager) SendGameServerCommand(ctx context.Context, gameServerID string, command string) error {
	// Get game server from database
	gameServer, err := database.NewGameServerRepository(database.DB, database.RedisClient).GetByID(ctx, gameServerID)
	if err != nil {
		return fmt.Errorf("failed to get game server: %w", err)
	}

	if gameServer.ContainerID == nil {
		return fmt.Errorf("game server %s has no container ID", gameServerID)
	}

	// Check if container is running
	containerInfo, err := gsm.dockerClient.ContainerInspect(ctx, *gameServer.ContainerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	if !containerInfo.State.Running {
		return fmt.Errorf("game server container is not running")
	}

	// Method 1: Try Pterodactyl-style console input (named pipe or file)
	// Pterodactyl servers use /tmp/console-input or similar
	consoleInputPaths := []string{
		"/tmp/console-input",
		"/tmp/server-console",
		"/var/run/server-console",
		"/server/console-input",
	}
	for _, consolePath := range consoleInputPaths {
		// Try writing to console input file/FIFO
		cmd := []string{"sh", "-c", fmt.Sprintf("test -p %s && echo '%s' > %s 2>/dev/null || echo '%s' > %s 2>/dev/null || true", consolePath, strings.ReplaceAll(command, "'", "'\"'\"'"), consolePath, strings.ReplaceAll(command, "'", "'\"'\"'"), consolePath)}
		output, err := gsm.dockerHelper.ContainerExecRun(ctx, *gameServer.ContainerID, cmd)
		if err == nil {
			logger.Debug("[GameServerManager] Sent command via console input %s to game server %s: %s (output: %s)", consolePath, gameServerID, command, output)
			// Check if command was actually written (some methods return success even if it fails)
			// If we got here without error, assume success
			return nil
		}
	}

	// Method 2: Try using rcon-cli if available (Minecraft servers often have this)
	// RCON is the most reliable method for Minecraft servers
	rconCmd := []string{"rcon-cli", command}
	output, rconErr := gsm.dockerHelper.ContainerExecRun(ctx, *gameServer.ContainerID, rconCmd)
	if rconErr == nil {
		logger.Debug("[GameServerManager] Sent command via rcon-cli to game server %s: %s", gameServerID, command)
		if output != "" {
			logger.Debug("[GameServerManager] RCON response: %s", output)
		}
		return nil
	}
	logger.Debug("[GameServerManager] RCON method failed for game server %s: %v", gameServerID, rconErr)

	// Method 3: Try using Docker attach API to write to main process stdin
	// This works for servers that have stdin open and accept commands directly
	attachErr := gsm.sendCommandViaAttach(ctx, *gameServer.ContainerID, command)
	if attachErr == nil {
		logger.Debug("[GameServerManager] Sent command via Docker attach to game server %s: %s", gameServerID, command)
		return nil
	}
	logger.Debug("[GameServerManager] Docker attach method failed for game server %s: %v", gameServerID, attachErr)

	// Method 4: Try common Minecraft server wrapper scripts
	// itzg/minecraft-server and similar images often have helper scripts
	wrapperScripts := []string{
		"/usr/local/bin/mc-send-to-console",
		"/usr/local/bin/mc-send-console",
		"/usr/bin/mc-send-to-console",
		"/bin/mc-send-to-console",
		"/usr/local/bin/docker-entrypoint.sh", // Some images use this
	}
	for _, script := range wrapperScripts {
		cmd := []string{"sh", "-c", fmt.Sprintf("test -f %s && %s '%s' || exit 1", script, script, strings.ReplaceAll(command, "'", "'\"'\"'"))}
		_, err := gsm.dockerHelper.ContainerExecRun(ctx, *gameServer.ContainerID, cmd)
		if err == nil {
			logger.Debug("[GameServerManager] Sent command via wrapper script %s to game server %s: %s", script, gameServerID, command)
			return nil
		}
	}

	// Method 5: Try writing to stdin file descriptor (works if stdin is a regular file)
	// Escape the command properly for shell
	commandEscaped := strings.ReplaceAll(command, "'", "'\"'\"'")
	commandEscaped = strings.ReplaceAll(commandEscaped, "\n", "\\n")
	commandEscaped = strings.ReplaceAll(commandEscaped, "\r", "\\r")

	// Try different methods to write to stdin
	stdinMethods := []string{
		fmt.Sprintf("printf '%%s\\n' '%s' > /proc/1/fd/0 2>/dev/null || true", commandEscaped),
		fmt.Sprintf("echo '%s' > /proc/1/fd/0 2>/dev/null || true", commandEscaped),
		fmt.Sprintf("echo '%s' | dd of=/proc/1/fd/0 bs=1 2>/dev/null || true", commandEscaped),
		fmt.Sprintf("echo '%s' >> /proc/1/fd/0 2>/dev/null || true", commandEscaped),
	}

	for _, method := range stdinMethods {
		cmd := []string{"sh", "-c", method}
		_, err := gsm.dockerHelper.ContainerExecRun(ctx, *gameServer.ContainerID, cmd)
		if err == nil {
			logger.Debug("[GameServerManager] Sent command via stdin file descriptor to game server %s: %s", gameServerID, command)
			return nil
		}
	}

	// All methods failed - provide helpful error message with all attempted methods
	return fmt.Errorf("failed to send command to game server (tried Pterodactyl console input, RCON, Docker attach, wrapper scripts, and stdin methods). RCON error: %v, Attach error: %v. Ensure the server accepts commands via stdin or has RCON enabled", rconErr, attachErr)
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
func (gsm *GameServerManager) removeContainerByName(ctx context.Context, containerName string) error {
	// Try to inspect container directly by name
	containerNameWithSlash := "/" + containerName
	containerNameWithoutSlash := strings.TrimPrefix(containerName, "/")

	for _, nameToTry := range []string{containerNameWithSlash, containerNameWithoutSlash} {
		containerInfo, err := gsm.dockerClient.ContainerInspect(ctx, nameToTry)
		if err == nil {
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
		OpenStdin:    true,  // Enable stdin for command sending
		Tty:          false, // Don't allocate TTY (stdin should still work)
		// Don't clear ENTRYPOINT - let the image use its default entrypoint
		// Most game server images have proper entrypoints configured
	}

	// Override container CMD if start command is provided
	// If no start command is provided, the image will use its default CMD/ENTRYPOINT
	if config.StartCommand != nil && *config.StartCommand != "" {
		// When overriding CMD, we need to clear ENTRYPOINT to avoid conflicts
		containerConfig.Entrypoint = []string{}
		containerConfig.Cmd = []string{"sh", "-c", *config.StartCommand}
	}

	// Convert CPU cores to CPU shares (Docker uses shares, not cores)
	// 1024 shares = 1 CPU core, so multiply by 1024
	cpuShares := int64(config.CPUCores) * 1024

	// Host configuration
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
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

// ensureImage pulls the Docker image if it doesn't exist locally
func (gsm *GameServerManager) ensureImage(ctx context.Context, imageName string) error {
	// Check if image exists locally
	images, err := gsm.dockerClient.ImageList(ctx, client.ImageListOptions{
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

	pullReader, err := gsm.dockerClient.ImagePull(ctx, imageName, pullOptions)
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
func (gsm *GameServerManager) updateGameServerContainerInfo(ctx context.Context, gameServerID, containerID, containerName string) error {
	updates := map[string]interface{}{
		"container_id":   containerID,
		"container_name": containerName,
		"status":         2, // STARTING = 2
		"updated_at":     time.Now(),
	}

	return database.DB.Model(&database.GameServer{}).
		Where("id = ?", gameServerID).
		Updates(updates).Error
}
