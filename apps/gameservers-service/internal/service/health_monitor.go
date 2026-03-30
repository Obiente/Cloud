package gameservers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/notifications"

	"github.com/moby/moby/client"
	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"
	notificationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1"
)

const (
	resourcePressureGracePeriod     = 5 * time.Minute
	resourcePressureRestartCooldown = 10 * time.Minute
	resourcePressureStatsTimeout    = 5 * time.Second
)

// checkAndSyncGameServerStatus checks all game servers that should be running
// and syncs their status with actual Docker container status
func (s *Service) checkAndSyncGameServerStatus(ctx context.Context) {
	logger.Debug("[HealthMonitor] Checking game servers that should be running...")

	// Query all game servers with status RUNNING, STARTING, RESTARTING, or STOPPING that are not deleted
	var gameServers []database.GameServer
	statusRunning := int32(gameserversv1.GameServerStatus_RUNNING)
	statusStarting := int32(gameserversv1.GameServerStatus_STARTING)
	statusRestarting := int32(gameserversv1.GameServerStatus_RESTARTING)
	statusStopping := int32(gameserversv1.GameServerStatus_STOPPING)

	err := database.DB.WithContext(ctx).
		Where("(status = ? OR status = ? OR status = ? OR status = ?) AND deleted_at IS NULL", statusRunning, statusStarting, statusRestarting, statusStopping).
		Find(&gameServers).Error

	if err != nil {
		logger.Warn("[HealthMonitor] Failed to query game servers: %v", err)
		return
	}

	logger.Debug("[HealthMonitor] Found %d game servers that should be running", len(gameServers))

	syncedCount := 0
	skippedCount := 0
	errorCount := 0

	// Get Docker client for container inspection
	dockerClient, err := client.New(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Warn("[HealthMonitor] Failed to create Docker client: %v", err)
		return
	}
	defer dockerClient.Close()

	for _, gameServer := range gameServers {
		// Skip if no container ID
		if gameServer.ContainerID == nil || *gameServer.ContainerID == "" {
			logger.Debug("[HealthMonitor] Game server %s has no container ID, skipping", gameServer.ID)
			skippedCount++
			continue
		}

		currentStatus := int32(gameServer.Status)

		// Inspect container to get actual status
		containerInfo, err := dockerClient.ContainerInspect(ctx, *gameServer.ContainerID, client.ContainerInspectOptions{})
		if err != nil {
			// Container doesn't exist - update status to STOPPED
			s.clearResourcePressureState(gameServer.ID)
			containerIDShort := *gameServer.ContainerID
			if len(containerIDShort) > 12 {
				containerIDShort = containerIDShort[:12]
			}
			logger.Info("[HealthMonitor] Game server %s container %s doesn't exist, updating status to STOPPED", gameServer.ID, containerIDShort)

			// Update game server status
			if err := s.repo.UpdateStatus(ctx, gameServer.ID, int32(gameserversv1.GameServerStatus_STOPPED)); err != nil {
				logger.Warn("[HealthMonitor] Failed to update game server %s status to STOPPED: %v", gameServer.ID, err)
				s.createSystemGameServerAuditLog(&gameServer, gameServer.ID, "SyncGameServerStatus", "container_missing", gameServerAuditSourceMonitor, 500, map[string]interface{}{
					"containerId":    *gameServer.ContainerID,
					"previousStatus": gameserversv1.GameServerStatus(currentStatus).String(),
					"currentStatus":  gameserversv1.GameServerStatus_STOPPED.String(),
				}, err)
				errorCount++
			} else {
				// Update location status
				if err := database.DB.Model(&database.GameServerLocation{}).
					Where("container_id = ?", *gameServer.ContainerID).
					Update("status", "stopped").Error; err != nil {
					logger.Warn("[HealthMonitor] Failed to update location status for game server %s: %v", gameServer.ID, err)
				}
				s.createSystemGameServerAuditLog(&gameServer, gameServer.ID, "SyncGameServerStatus", "container_missing", gameServerAuditSourceMonitor, 200, map[string]interface{}{
					"containerId":    *gameServer.ContainerID,
					"previousStatus": gameserversv1.GameServerStatus(currentStatus).String(),
					"currentStatus":  gameserversv1.GameServerStatus_STOPPED.String(),
				}, nil)
				syncedCount++
			}
			continue
		}

		// Check if container is actually running
		isRunning := containerInfo.Container.State.Running

		// Sync status based on actual container state
		if isRunning {
			s.evaluateResourcePressure(ctx, dockerClient, &gameServer)

			// Container is running - update to RUNNING if not already
			if currentStatus != statusRunning {
				logger.Info("[HealthMonitor] Game server %s container is running but DB status is %d, updating to RUNNING", gameServer.ID, currentStatus)
				if err := s.repo.UpdateStatus(ctx, gameServer.ID, statusRunning); err != nil {
					logger.Warn("[HealthMonitor] Failed to update game server %s status to RUNNING: %v", gameServer.ID, err)
					s.createSystemGameServerAuditLog(&gameServer, gameServer.ID, "SyncGameServerStatus", "container_running", gameServerAuditSourceMonitor, 500, map[string]interface{}{
						"containerId":    *gameServer.ContainerID,
						"previousStatus": gameserversv1.GameServerStatus(currentStatus).String(),
						"currentStatus":  gameserversv1.GameServerStatus_RUNNING.String(),
					}, err)
					errorCount++
				} else {
					// Update location status
					if err := database.DB.Model(&database.GameServerLocation{}).
						Where("container_id = ?", *gameServer.ContainerID).
						Update("status", "running").Error; err != nil {
						logger.Warn("[HealthMonitor] Failed to update location status for game server %s: %v", gameServer.ID, err)
					}
					s.createSystemGameServerAuditLog(&gameServer, gameServer.ID, "SyncGameServerStatus", "container_running", gameServerAuditSourceMonitor, 200, map[string]interface{}{
						"containerId":    *gameServer.ContainerID,
						"previousStatus": gameserversv1.GameServerStatus(currentStatus).String(),
						"currentStatus":  gameserversv1.GameServerStatus_RUNNING.String(),
					}, nil)
					syncedCount++
				}
			} else {
				skippedCount++
			}
		} else {
			s.clearResourcePressureState(gameServer.ID)

			// Container is not running - check exit code to determine status
			exitCode := containerInfo.Container.State.ExitCode
			if exitCode == 0 {
				// Container stopped normally - update to STOPPED
				if currentStatus != int32(gameserversv1.GameServerStatus_STOPPED) {
					logger.Info("[HealthMonitor] Game server %s container stopped (exit code %d) but DB status is %d, updating to STOPPED", gameServer.ID, exitCode, currentStatus)
					if err := s.repo.UpdateStatus(ctx, gameServer.ID, int32(gameserversv1.GameServerStatus_STOPPED)); err != nil {
						logger.Warn("[HealthMonitor] Failed to update game server %s status to STOPPED: %v", gameServer.ID, err)
						s.createSystemGameServerAuditLog(&gameServer, gameServer.ID, "SyncGameServerStatus", "container_exit", gameServerAuditSourceMonitor, 500, map[string]interface{}{
							"containerId":    *gameServer.ContainerID,
							"exitCode":       exitCode,
							"previousStatus": gameserversv1.GameServerStatus(currentStatus).String(),
							"currentStatus":  gameserversv1.GameServerStatus_STOPPED.String(),
						}, err)
						errorCount++
					} else {
						// Update location status
						if err := database.DB.Model(&database.GameServerLocation{}).
							Where("container_id = ?", *gameServer.ContainerID).
							Update("status", "stopped").Error; err != nil {
							logger.Warn("[HealthMonitor] Failed to update location status for game server %s: %v", gameServer.ID, err)
						}
						s.createSystemGameServerAuditLog(&gameServer, gameServer.ID, "SyncGameServerStatus", "container_exit", gameServerAuditSourceMonitor, 200, map[string]interface{}{
							"containerId":    *gameServer.ContainerID,
							"exitCode":       exitCode,
							"previousStatus": gameserversv1.GameServerStatus(currentStatus).String(),
							"currentStatus":  gameserversv1.GameServerStatus_STOPPED.String(),
						}, nil)
						syncedCount++
					}
				} else {
					skippedCount++
				}
			} else {
				// Container crashed or failed - rely on Docker state for OOM attribution.
				// Exit code 137 can also come from non-OOM SIGKILL events (manual kill, daemon stop, node pressure).
				isOOMKill := containerInfo.Container.State.OOMKilled

				if currentStatus != int32(gameserversv1.GameServerStatus_FAILED) {
					logger.Info("[HealthMonitor] Game server %s container exited with code %d but DB status is %d, updating to FAILED", gameServer.ID, exitCode, currentStatus)
					if err := s.repo.UpdateStatus(ctx, gameServer.ID, int32(gameserversv1.GameServerStatus_FAILED)); err != nil {
						logger.Warn("[HealthMonitor] Failed to update game server %s status to FAILED: %v", gameServer.ID, err)
						s.createSystemGameServerAuditLog(&gameServer, gameServer.ID, "SyncGameServerStatus", "container_exit", gameServerAuditSourceMonitor, 500, map[string]interface{}{
							"containerId":    *gameServer.ContainerID,
							"exitCode":       exitCode,
							"isOOMKill":      isOOMKill,
							"previousStatus": gameserversv1.GameServerStatus(currentStatus).String(),
							"currentStatus":  gameserversv1.GameServerStatus_FAILED.String(),
						}, err)
						errorCount++
					} else {
						// Update location status
						if err := database.DB.Model(&database.GameServerLocation{}).
							Where("container_id = ?", *gameServer.ContainerID).
							Update("status", "stopped").Error; err != nil {
							logger.Warn("[HealthMonitor] Failed to update location status for game server %s: %v", gameServer.ID, err)
						}

						// Send notification if it's an OOM kill
						if isOOMKill {
							s.sendOOMKillNotification(ctx, &gameServer)
						}
						s.createSystemGameServerAuditLog(&gameServer, gameServer.ID, "SyncGameServerStatus", "container_exit", gameServerAuditSourceMonitor, 200, map[string]interface{}{
							"containerId":    *gameServer.ContainerID,
							"exitCode":       exitCode,
							"isOOMKill":      isOOMKill,
							"previousStatus": gameserversv1.GameServerStatus(currentStatus).String(),
							"currentStatus":  gameserversv1.GameServerStatus_FAILED.String(),
						}, nil)

						syncedCount++
					}
				} else {
					// Status already FAILED - don't send duplicate notifications
					// (notification was already sent when status changed to FAILED)
					skippedCount++
				}
			}
		}
	}

	logger.Debug("[HealthMonitor] Check complete: %d synced, %d skipped (already correct), %d errors", syncedCount, skippedCount, errorCount)
}

func (s *Service) clearResourcePressureState(gameServerID string) {
	s.resourcePressureMu.Lock()
	delete(s.resourcePressureState, gameServerID)
	s.resourcePressureMu.Unlock()
}

func (s *Service) evaluateResourcePressure(ctx context.Context, dockerClient *client.Client, gameServer *database.GameServer) {
	if gameServer == nil || gameServer.ContainerID == nil || *gameServer.ContainerID == "" {
		return
	}

	statsCtx, cancel := context.WithTimeout(ctx, resourcePressureStatsTimeout)
	defer cancel()

	memoryUsage, err := getContainerMemoryUsage(statsCtx, dockerClient, *gameServer.ContainerID)
	if err != nil {
		logger.Debug("[HealthMonitor] Failed to read memory usage for game server %s: %v", gameServer.ID, err)
		return
	}

	now := time.Now()
	memoryExceeded := gameServer.MemoryBytes > 0 && memoryUsage > gameServer.MemoryBytes

	s.resourcePressureMu.Lock()
	state, exists := s.resourcePressureState[gameServer.ID]
	if !exists {
		state = &resourcePressureState{}
		s.resourcePressureState[gameServer.ID] = state
	}
	state.lastObservedMemoryUsage = memoryUsage

	if !state.cooldownUntil.IsZero() && now.Before(state.cooldownUntil) {
		if !memoryExceeded {
			state.memoryFirstExceededAt = time.Time{}
		}
		s.resourcePressureMu.Unlock()
		return
	}

	if memoryExceeded {
		if state.memoryFirstExceededAt.IsZero() {
			state.memoryFirstExceededAt = now
		}
	} else {
		state.memoryFirstExceededAt = time.Time{}
	}

	if state.restartInProgress || !memoryExceeded || now.Sub(state.memoryFirstExceededAt) < resourcePressureGracePeriod {
		s.resourcePressureMu.Unlock()
		return
	}

	firstExceededAt := state.memoryFirstExceededAt
	state.restartInProgress = true
	state.cooldownUntil = now.Add(resourcePressureRestartCooldown)
	state.memoryFirstExceededAt = time.Time{}
	s.resourcePressureMu.Unlock()

	go s.restartForMemoryPressure(gameServer.ID, memoryUsage, gameServer.MemoryBytes, firstExceededAt)
}

func getContainerMemoryUsage(ctx context.Context, dockerClient *client.Client, containerID string) (int64, error) {
	statsResp, err := dockerClient.ContainerStats(ctx, containerID, client.ContainerStatsOptions{Stream: false})
	if err != nil {
		return 0, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer statsResp.Body.Close()

	var statsJSON struct {
		MemoryStats struct {
			Usage uint64 `json:"usage"`
		} `json:"memory_stats"`
	}

	if err := json.NewDecoder(statsResp.Body).Decode(&statsJSON); err != nil {
		return 0, fmt.Errorf("failed to decode container stats: %w", err)
	}

	return int64(statsJSON.MemoryStats.Usage), nil
}

func (s *Service) restartForMemoryPressure(gameServerID string, memoryUsage int64, memoryLimit int64, firstExceededAt time.Time) {
	defer func() {
		s.resourcePressureMu.Lock()
		if state, exists := s.resourcePressureState[gameServerID]; exists {
			state.restartInProgress = false
		}
		s.resourcePressureMu.Unlock()
	}()

	restartCtx, cancel := s.detachedContext(2 * time.Minute)
	defer cancel()

	manager, err := s.getGameServerManager()
	if err != nil {
		logger.Warn("[HealthMonitor] Cannot restart game server %s for resource pressure: %v", gameServerID, err)
		return
	}

	if err := manager.RestartGameServer(restartCtx, gameServerID); err != nil {
		logger.Warn("[HealthMonitor] Failed to restart game server %s after sustained memory pressure: %v", gameServerID, err)
		s.createSystemGameServerAuditLog(nil, gameServerID, "RestartGameServer", "memory_pressure", gameServerAuditSourceMonitor, 500, map[string]interface{}{
			"observedUsageBytes":   memoryUsage,
			"configuredLimitBytes": memoryLimit,
			"firstExceededAt":      firstExceededAt.UTC().Format(time.RFC3339),
			"gracePeriodSeconds":   int(resourcePressureGracePeriod.Seconds()),
			"cooldownSeconds":      int(resourcePressureRestartCooldown.Seconds()),
		}, err)
		notifyCtx, notifyCancel := s.detachedContext(15 * time.Second)
		defer notifyCancel()
		s.sendMemoryPressureNotification(notifyCtx, gameServerID, memoryUsage, memoryLimit, firstExceededAt, false, err.Error())
		return
	}

	go func() {
		updateCtx, updateCancel := s.detachedContext(30 * time.Second)
		defer updateCancel()
		if err := s.updateGameServerStorage(updateCtx, gameServerID); err != nil {
			logger.Warn("[HealthMonitor] Failed to update storage after memory-pressure restart for game server %s: %v", gameServerID, err)
		}
	}()

	logger.Warn("[HealthMonitor] Restarted game server %s after sustained memory pressure for over %v (usage=%d bytes, limit=%d bytes)", gameServerID, resourcePressureGracePeriod, memoryUsage, memoryLimit)
	s.createSystemGameServerAuditLog(nil, gameServerID, "RestartGameServer", "memory_pressure", gameServerAuditSourceMonitor, 200, map[string]interface{}{
		"observedUsageBytes":   memoryUsage,
		"configuredLimitBytes": memoryLimit,
		"firstExceededAt":      firstExceededAt.UTC().Format(time.RFC3339),
		"gracePeriodSeconds":   int(resourcePressureGracePeriod.Seconds()),
		"cooldownSeconds":      int(resourcePressureRestartCooldown.Seconds()),
	}, nil)
	notifyCtx, notifyCancel := s.detachedContext(15 * time.Second)
	defer notifyCancel()
	s.sendMemoryPressureNotification(notifyCtx, gameServerID, memoryUsage, memoryLimit, firstExceededAt, true, "")
}

func (s *Service) sendMemoryPressureNotification(ctx context.Context, gameServerID string, memoryUsage int64, memoryLimit int64, firstExceededAt time.Time, restarted bool, restartErr string) {
	gameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		logger.Warn("[HealthMonitor] Failed to load game server %s for resource-pressure notification: %v", gameServerID, err)
		return
	}

	if gameServer.CreatedBy == "" {
		logger.Warn("[HealthMonitor] Cannot send resource-pressure notification for game server %s: no CreatedBy user ID", gameServer.ID)
		return
	}

	observedDisplay := fmt.Sprintf("%.2f GB", float64(memoryUsage)/(1024*1024*1024))
	limitDisplay := fmt.Sprintf("%.2f GB", float64(memoryLimit)/(1024*1024*1024))
	metadata := map[string]string{
		"game_server_id":         gameServer.ID,
		"game_server_name":       gameServer.Name,
		"reason":                 "memory_pressure",
		"grace_period_seconds":   fmt.Sprintf("%d", int(resourcePressureGracePeriod.Seconds())),
		"first_exceeded_at":      firstExceededAt.UTC().Format(time.RFC3339),
		"auto_restarted":         fmt.Sprintf("%t", restarted),
		"observed_usage_bytes":   fmt.Sprintf("%d", memoryUsage),
		"configured_limit_bytes": fmt.Sprintf("%d", memoryLimit),
	}

	var title string
	var message string
	severity := notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_HIGH

	if restarted {
		title = fmt.Sprintf("Game Server \"%s\" Auto-Restarted After Sustained High Memory Usage", gameServer.Name)
		message = fmt.Sprintf(
			"Your game server \"%s\" exceeded its configured memory limit for more than 5 minutes and was gracefully restarted to prevent an abrupt crash. Observed usage: %s, configured limit: %s.",
			gameServer.Name,
			observedDisplay,
			limitDisplay,
		)
	} else {
		severity = notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_CRITICAL
		title = fmt.Sprintf("Game Server \"%s\" Hit Sustained High Memory Usage (Restart Failed)", gameServer.Name)
		message = fmt.Sprintf(
			"Your game server \"%s\" exceeded its configured memory limit for more than 5 minutes. An automatic graceful restart was attempted but failed: %s. Observed usage: %s, configured limit: %s.",
			gameServer.Name,
			restartErr,
			observedDisplay,
			limitDisplay,
		)
	}

	actionURL := fmt.Sprintf("/gameservers/%s", gameServer.ID)
	actionLabel := "View Game Server"

	var orgID *string
	if gameServer.OrganizationID != "" {
		orgID = &gameServer.OrganizationID
	}

	if restartErr != "" {
		metadata["restart_error"] = restartErr
	}

	if err := notifications.CreateNotificationForUser(
		ctx,
		gameServer.CreatedBy,
		orgID,
		notificationsv1.NotificationType_NOTIFICATION_TYPE_SYSTEM,
		severity,
		title,
		message,
		&actionURL,
		&actionLabel,
		metadata,
	); err != nil {
		logger.Warn("[HealthMonitor] Failed to send resource-pressure notification for game server %s to user %s: %v", gameServer.ID, gameServer.CreatedBy, err)
		return
	}

	logger.Info("[HealthMonitor] Sent resource-pressure notification for game server %s to user %s", gameServer.ID, gameServer.CreatedBy)
}

// sendOOMKillNotification sends a notification to the game server owner about an OOM kill
func (s *Service) sendOOMKillNotification(ctx context.Context, gameServer *database.GameServer) {
	if gameServer.CreatedBy == "" {
		logger.Warn("[HealthMonitor] Cannot send OOM kill notification for game server %s: no CreatedBy user ID", gameServer.ID)
		return
	}

	// Format memory limit for display
	memoryGB := float64(gameServer.MemoryBytes) / (1024 * 1024 * 1024)
	memoryLimitStr := fmt.Sprintf("%.2f GB", memoryGB)
	if memoryGB < 1 {
		memoryMB := float64(gameServer.MemoryBytes) / (1024 * 1024)
		memoryLimitStr = fmt.Sprintf("%.0f MB", memoryMB)
	}

	title := fmt.Sprintf("Game Server \"%s\" Stopped Due to Memory Limit", gameServer.Name)
	message := fmt.Sprintf(
		"Your game server \"%s\" was stopped because it exceeded its memory limit of %s. "+
			"Consider increasing the memory limit or optimizing your server configuration.",
		gameServer.Name,
		memoryLimitStr,
	)

	// Create action URL to view/edit the game server
	actionURL := fmt.Sprintf("/gameservers/%s", gameServer.ID)
	actionLabel := "View Game Server"

	// Get organization ID if available
	var orgID *string
	if gameServer.OrganizationID != "" {
		orgID = &gameServer.OrganizationID
	}

	// Send notification
	err := notifications.CreateNotificationForUser(
		ctx,
		gameServer.CreatedBy,
		orgID,
		notificationsv1.NotificationType_NOTIFICATION_TYPE_ERROR,
		notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_HIGH,
		title,
		message,
		&actionURL,
		&actionLabel,
		map[string]string{
			"game_server_id":   gameServer.ID,
			"game_server_name": gameServer.Name,
			"exit_code":        "137",
			"reason":           "oom_kill",
			"memory_limit":     fmt.Sprintf("%d", gameServer.MemoryBytes),
		},
	)

	if err != nil {
		logger.Warn("[HealthMonitor] Failed to send OOM kill notification for game server %s to user %s: %v", gameServer.ID, gameServer.CreatedBy, err)
	} else {
		logger.Info("[HealthMonitor] Sent OOM kill notification for game server %s to user %s", gameServer.ID, gameServer.CreatedBy)
	}
}

// StartHealthMonitor starts a background service that periodically checks
// and syncs game server status with actual Docker container status
func (s *Service) StartHealthMonitor(ctx context.Context, interval time.Duration) {
	logger.Info("[HealthMonitor] Starting health monitor service (interval: %v)", interval)

	// Run immediately on startup
	s.checkAndSyncGameServerStatus(ctx)

	// Then run periodically
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("[HealthMonitor] Health monitor service shutting down")
			return
		case <-ticker.C:
			s.checkAndSyncGameServerStatus(ctx)
		}
	}
}
