package gameservers

import (
	"context"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/notifications"

	"github.com/moby/moby/client"
	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"
	notificationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1"
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
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
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

		// Inspect container to get actual status
		containerInfo, err := dockerClient.ContainerInspect(ctx, *gameServer.ContainerID, client.ContainerInspectOptions{})
		if err != nil {
			// Container doesn't exist - update status to STOPPED
			containerIDShort := *gameServer.ContainerID
			if len(containerIDShort) > 12 {
				containerIDShort = containerIDShort[:12]
			}
			logger.Info("[HealthMonitor] Game server %s container %s doesn't exist, updating status to STOPPED", gameServer.ID, containerIDShort)

			// Update game server status
			if err := s.repo.UpdateStatus(ctx, gameServer.ID, int32(gameserversv1.GameServerStatus_STOPPED)); err != nil {
				logger.Warn("[HealthMonitor] Failed to update game server %s status to STOPPED: %v", gameServer.ID, err)
				errorCount++
			} else {
				// Update location status
				if err := database.DB.Model(&database.GameServerLocation{}).
					Where("container_id = ?", *gameServer.ContainerID).
					Update("status", "stopped").Error; err != nil {
					logger.Warn("[HealthMonitor] Failed to update location status for game server %s: %v", gameServer.ID, err)
				}
				syncedCount++
			}
			continue
		}

		// Check if container is actually running
		isRunning := containerInfo.Container.State.Running
		currentStatus := int32(gameServer.Status)

		// Sync status based on actual container state
		if isRunning {
			// Container is running - update to RUNNING if not already
			if currentStatus != statusRunning {
				logger.Info("[HealthMonitor] Game server %s container is running but DB status is %d, updating to RUNNING", gameServer.ID, currentStatus)
				if err := s.repo.UpdateStatus(ctx, gameServer.ID, statusRunning); err != nil {
					logger.Warn("[HealthMonitor] Failed to update game server %s status to RUNNING: %v", gameServer.ID, err)
					errorCount++
				} else {
					// Update location status
					if err := database.DB.Model(&database.GameServerLocation{}).
						Where("container_id = ?", *gameServer.ContainerID).
						Update("status", "running").Error; err != nil {
						logger.Warn("[HealthMonitor] Failed to update location status for game server %s: %v", gameServer.ID, err)
					}
					syncedCount++
				}
			} else {
				skippedCount++
			}
		} else {
			// Container is not running - check exit code to determine status
			exitCode := containerInfo.Container.State.ExitCode
			if exitCode == 0 {
				// Container stopped normally - update to STOPPED
				if currentStatus != int32(gameserversv1.GameServerStatus_STOPPED) {
					logger.Info("[HealthMonitor] Game server %s container stopped (exit code %d) but DB status is %d, updating to STOPPED", gameServer.ID, exitCode, currentStatus)
					if err := s.repo.UpdateStatus(ctx, gameServer.ID, int32(gameserversv1.GameServerStatus_STOPPED)); err != nil {
						logger.Warn("[HealthMonitor] Failed to update game server %s status to STOPPED: %v", gameServer.ID, err)
						errorCount++
					} else {
						// Update location status
						if err := database.DB.Model(&database.GameServerLocation{}).
							Where("container_id = ?", *gameServer.ContainerID).
							Update("status", "stopped").Error; err != nil {
							logger.Warn("[HealthMonitor] Failed to update location status for game server %s: %v", gameServer.ID, err)
						}
						syncedCount++
					}
				} else {
					skippedCount++
				}
			} else {
				// Container crashed or failed - check if it's an OOM kill
				isOOMKill := exitCode == 137 // Exit code 137 = OOM kill (128 + 9, where 9 is SIGKILL)
				
				if currentStatus != int32(gameserversv1.GameServerStatus_FAILED) {
					logger.Info("[HealthMonitor] Game server %s container exited with code %d but DB status is %d, updating to FAILED", gameServer.ID, exitCode, currentStatus)
					if err := s.repo.UpdateStatus(ctx, gameServer.ID, int32(gameserversv1.GameServerStatus_FAILED)); err != nil {
						logger.Warn("[HealthMonitor] Failed to update game server %s status to FAILED: %v", gameServer.ID, err)
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
