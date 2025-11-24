package gameservers

import (
	"context"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"

	"github.com/moby/moby/client"
	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"
)

// checkAndSyncGameServerStatus checks all game servers that should be running
// and syncs their status with actual Docker container status
func (s *Service) checkAndSyncGameServerStatus(ctx context.Context) {
	logger.Debug("[HealthMonitor] Checking game servers that should be running...")

	// Query all game servers with status RUNNING, STARTING, or RESTARTING that are not deleted
	var gameServers []database.GameServer
	statusRunning := int32(gameserversv1.GameServerStatus_RUNNING)
	statusStarting := int32(gameserversv1.GameServerStatus_STARTING)
	statusRestarting := int32(gameserversv1.GameServerStatus_RESTARTING)

	err := database.DB.WithContext(ctx).
		Where("(status = ? OR status = ? OR status = ?) AND deleted_at IS NULL", statusRunning, statusStarting, statusRestarting).
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
		containerInfo, err := dockerClient.ContainerInspect(ctx, *gameServer.ContainerID)
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
		isRunning := containerInfo.State.Running
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
			exitCode := containerInfo.State.ExitCode
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
				// Container crashed or failed - update to FAILED
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
						syncedCount++
					}
				} else {
					skippedCount++
				}
			}
		}
	}

	logger.Debug("[HealthMonitor] Check complete: %d synced, %d skipped (already correct), %d errors", syncedCount, skippedCount, errorCount)
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
