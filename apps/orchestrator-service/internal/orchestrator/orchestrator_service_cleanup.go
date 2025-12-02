package orchestrator

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/logger"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// Cleanup operations for orchestrator service

func (os *OrchestratorService) cleanupTasks() {
	// Run every 1 hour to aggregate and clean old metrics frequently
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Debug("[Orchestrator] Running cleanup tasks...")

			// Keep last 24 hours of raw metrics for real-time viewing
			// Aggregate older metrics into hourly summaries
			aggregateCutoff := time.Now().Add(-24 * time.Hour).Truncate(time.Hour)

			// Get all deployments and game servers that have metrics older than cutoff
			var deploymentIDs []string
			metricsDB := database.GetMetricsDB()
			metricsDB.Table("deployment_metrics").
				Select("DISTINCT deployment_id").
				Where("timestamp < ?", aggregateCutoff).
				Pluck("deployment_id", &deploymentIDs)

			var gameServerIDs []string
			metricsDB.Table("game_server_metrics").
				Select("DISTINCT game_server_id").
				Where("timestamp < ?", aggregateCutoff).
				Pluck("game_server_id", &gameServerIDs)

			var vpsIDs []string
			metricsDB.Table("vps_metrics").
				Select("DISTINCT vps_instance_id").
				Where("timestamp < ?", aggregateCutoff).
				Pluck("vps_instance_id", &vpsIDs)

			if len(deploymentIDs) == 0 && len(gameServerIDs) == 0 && len(vpsIDs) == 0 {
				logger.Debug("[Orchestrator] No old metrics to aggregate")
				logger.Debug("[Orchestrator] Cleanup tasks completed")
				continue
			}

			logger.Debug("[Orchestrator] Aggregating metrics for %d deployments, %d game servers, %d VPS instances", len(deploymentIDs), len(gameServerIDs), len(vpsIDs))

			// Process deployments and game servers in parallel batches
			const batchSize = 10 // Process 10 resources concurrently
			totalAggregated := 0
			totalDeleted := int64(0)
			var aggMutex sync.Mutex

			// Process deployments in batches
			for i := 0; i < len(deploymentIDs); i += batchSize {
				end := i + batchSize
				if end > len(deploymentIDs) {
					end = len(deploymentIDs)
				}
				batch := deploymentIDs[i:end]

				var wg sync.WaitGroup
				for _, deploymentID := range batch {
					wg.Add(1)
					go func(depID string) {
						defer wg.Done()
						aggregated, deleted := os.aggregateDeploymentMetrics(depID, aggregateCutoff)
						if aggregated > 0 || deleted > 0 {
							aggMutex.Lock()
							totalAggregated += aggregated
							totalDeleted += deleted
							aggMutex.Unlock()
						}
					}(deploymentID)
				}
				wg.Wait()
			}

			// Process game servers in batches
			for i := 0; i < len(gameServerIDs); i += batchSize {
				end := i + batchSize
				if end > len(gameServerIDs) {
					end = len(gameServerIDs)
				}
				batch := gameServerIDs[i:end]

				var wg sync.WaitGroup
				for _, gameServerID := range batch {
					wg.Add(1)
					go func(gsID string) {
						defer wg.Done()
						aggregated, deleted := os.aggregateGameServerMetrics(gsID, aggregateCutoff)
						if aggregated > 0 || deleted > 0 {
							aggMutex.Lock()
							totalAggregated += aggregated
							totalDeleted += deleted
							aggMutex.Unlock()
						}
					}(gameServerID)
				}
				wg.Wait()
			}

			// Process VPS instances in batches
			for i := 0; i < len(vpsIDs); i += batchSize {
				end := i + batchSize
				if end > len(vpsIDs) {
					end = len(vpsIDs)
				}
				batch := vpsIDs[i:end]

				var wg sync.WaitGroup
				for _, vpsID := range batch {
					wg.Add(1)
					go func(vID string) {
						defer wg.Done()
						aggregated, deleted := os.aggregateVPSMetrics(vID, aggregateCutoff)
						if aggregated > 0 || deleted > 0 {
							aggMutex.Lock()
							totalAggregated += aggregated
							totalDeleted += deleted
							aggMutex.Unlock()
						}
					}(vpsID)
				}
				wg.Wait()
			}

			logger.Debug("[Orchestrator] Aggregated %d hours, deleted %d raw metrics, cleanup tasks completed", totalAggregated, totalDeleted)
		case <-os.ctx.Done():
			return
		}
	}
}

func (os *OrchestratorService) cleanupBuildHistory() {
	// Run daily at midnight
	// Calculate time until next midnight
	now := time.Now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	initialDelay := nextMidnight.Sub(now)

	// Wait for first run
	select {
	case <-time.After(initialDelay):
		// Run immediately after initial delay
	case <-os.ctx.Done():
		return
	}

	// Run daily after initial delay
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Info("[Orchestrator] Running build history cleanup...")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			// Delete builds older than 30 days
			buildHistoryRepo := database.NewBuildHistoryRepository(database.DB)
			buildIDs, deletedCount, err := buildHistoryRepo.DeleteBuildsOlderThan(ctx, 30*24*time.Hour)
			if err != nil {
				logger.Warn("[Orchestrator] Failed to cleanup build history: %v", err)
			} else {
				// Delete logs from TimescaleDB for the deleted builds
				if len(buildIDs) > 0 {
					buildLogsRepo := database.NewBuildLogsRepository(database.MetricsDB)
					for _, buildID := range buildIDs {
						if err := buildLogsRepo.DeleteBuildLogs(ctx, buildID); err != nil {
							logger.Warn("[Orchestrator] Failed to delete logs for build %s: %v", buildID, err)
							// Continue with other builds
						}
					}
					logger.Info("[Orchestrator] Deleted logs for %d build(s) from TimescaleDB", len(buildIDs))
				}
				logger.Info("[Orchestrator] Deleted %d build(s) older than 30 days", deletedCount)
			}
		case <-os.ctx.Done():
			return
		}
	}
}

func (os *OrchestratorService) cleanupStrayContainers() {
	// Run every 6 hours
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	// Run immediately on startup
	os.runStrayContainerCleanup()

	for {
		select {
		case <-ticker.C:
			os.runStrayContainerCleanup()
		case <-os.ctx.Done():
			return
		}
	}
}

func (os *OrchestratorService) runStrayContainerCleanup() {
	logger.Info("[Orchestrator] Running stray container cleanup...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Get Docker client
	dcli, err := docker.New()
	if err != nil {
		logger.Warn("[Orchestrator] Failed to create Docker client for stray cleanup: %v", err)
		return
	}
	defer dcli.Close()

	// Get node ID
	nodeID := os.deploymentManager.GetNodeID()

	// Get all containers managed by Obiente
	filterArgs := make(client.Filters)
	filterArgs.Add("label", "cloud.obiente.managed=true")

	dockerClient, ok := os.deploymentManager.GetDockerClient().(client.APIClient)
	if !ok {
		logger.Warn("[Orchestrator] Failed to get Docker client for cleanup")
		return
	}

	containers, err := dockerClient.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		logger.Warn("[Orchestrator] Failed to list containers for stray cleanup: %v", err)
		return
	}

	// Get all container IDs from deployment_locations table
	var dbContainerIDs []string
	if err := database.DB.Table("deployment_locations").
		Select("container_id").
		Where("node_id = ?", nodeID).
		Pluck("container_id", &dbContainerIDs).Error; err != nil {
		logger.Warn("[Orchestrator] Failed to query deployment locations: %v", err)
		return
	}

	// Build a map for fast lookup
	dbContainerMap := make(map[string]bool)
	for _, id := range dbContainerIDs {
		dbContainerMap[id] = true
	}

	// Find stray containers (running but not in DB)
	// ContainerList returns client.ContainerListResult
	var strayContainers []container.Summary
	for _, container := range containers.Items {
		// Verify container has cloud.obiente.managed=true (should already be filtered, but double-check)
		if container.Labels["cloud.obiente.managed"] != "true" {
			continue
		}

		// Check if container is running
		containerInfoResult, err := dockerClient.ContainerInspect(ctx, container.ID, client.ContainerInspectOptions{})
		if err != nil {
			logger.Debug("[Orchestrator] Failed to inspect container %s: %v", container.ID[:12], err)
			continue
		}
		containerInfo := containerInfoResult.Container

		// Only process running containers
		if !containerInfo.State.Running {
			continue
		}

		// Check if container exists in database
		if !dbContainerMap[container.ID] {
			strayContainers = append(strayContainers, container)
		}
	}

	if len(strayContainers) == 0 {
		logger.Debug("[Orchestrator] No stray containers found")
	} else {
		logger.Info("[Orchestrator] Found %d stray container(s)", len(strayContainers))
	}

	// Stop stray containers and record them
	now := time.Now()
	for _, container := range strayContainers {
		// Check if we've already recorded this container
		var existingStray database.StrayContainer
		if err := database.DB.Where("container_id = ?", container.ID).First(&existingStray).Error; err == nil {
			// Already recorded, skip
			continue
		}

		// Stop the container
		logger.Info("[Orchestrator] Stopping stray container %s", container.ID[:12])
		timeout := 30 * time.Second
		if err := dcli.StopContainer(ctx, container.ID, timeout); err != nil {
			logger.Warn("[Orchestrator] Failed to stop stray container %s: %v", container.ID[:12], err)
			continue
		}

		// Record in database
		stray := database.StrayContainer{
			ContainerID: container.ID,
			NodeID:      nodeID,
			StoppedAt:   now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := database.DB.Create(&stray).Error; err != nil {
			logger.Warn("[Orchestrator] Failed to record stray container %s: %v", container.ID[:12], err)
		} else {
			logger.Info("[Orchestrator] Recorded stray container %s (stopped at %s)", container.ID[:12], now.Format(time.RFC3339))
		}
	}

	// Delete volumes for containers stopped more than 7 days ago
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)
	var oldStrayContainers []database.StrayContainer
	if err := database.DB.Where("stopped_at < ? AND volumes_deleted_at IS NULL", sevenDaysAgo).Find(&oldStrayContainers).Error; err != nil {
		// Check if error is due to missing table (table might not exist yet)
		if database.DB.Migrator().HasTable(&database.StrayContainer{}) {
			logger.Warn("[Orchestrator] Failed to query old stray containers: %v", err)
		} else {
			logger.Debug("[Orchestrator] stray_containers table does not exist yet, skipping volume cleanup")
		}
		return
	}

	if len(oldStrayContainers) == 0 {
		logger.Debug("[Orchestrator] No old stray containers to clean up")
		return
	}

	logger.Info("[Orchestrator] Cleaning up volumes for %d old stray container(s)", len(oldStrayContainers))

	for _, stray := range oldStrayContainers {
		// Verify container has cloud.obiente tags before deleting volumes
		// Check container labels to ensure it's an Obiente-managed container
		// Even though container might be gone, we can check if volumes are Obiente volumes
		containerInfoResult, err := dockerClient.ContainerInspect(ctx, stray.ContainerID, client.ContainerInspectOptions{})
		hasObienteTags := false
		containerExists := err == nil
		if containerExists {
			containerInfo := containerInfoResult.Container
			if containerInfo.Config != nil && containerInfo.Config.Labels != nil {
				// Container still exists - verify it has cloud.obiente tags
				if containerInfo.Config.Labels["cloud.obiente.managed"] == "true" {
					hasObienteTags = true
				}
			}
		} else {
			// Container is gone - we can only verify by checking volume paths
			// We'll only delete volumes if they're Obiente volumes (path-based check)
			// For Docker volumes, we'll skip if container is gone (too risky)
			hasObienteTags = true // Assume it had tags since it's in stray_containers table
		}

		// Only proceed if container has Obiente tags (or container is gone and we can verify via paths)
		if !hasObienteTags {
			logger.Warn("[Orchestrator] Skipping volume deletion for container %s - missing cloud.obiente tags", stray.ContainerID[:12])
			continue
		}

		// Get volumes for this container
		volumes, err := dcli.GetContainerVolumes(ctx, stray.ContainerID)
		if err != nil {
			logger.Warn("[Orchestrator] Failed to get volumes for container %s: %v", stray.ContainerID[:12], err)
			// Mark as deleted even if we couldn't get volumes (container might be gone)
			database.DB.Model(&stray).Update("volumes_deleted_at", now)
			continue
		}

		// Delete each volume - only delete Obiente-managed volumes
		for _, volume := range volumes {
			// For Obiente Cloud volumes (bind mounts), delete the directory
			// These are stored in /var/lib/obiente/volumes/{deploymentID}/{volumeName}
			if strings.HasPrefix(volume.Source, "/var/lib/obiente/volumes") {
				logger.Info("[Orchestrator] Deleting Obiente volume at %s", volume.Source)
				if err := exec.Command("rm", "-rf", volume.Source).Run(); err != nil {
					logger.Warn("[Orchestrator] Failed to delete Obiente volume %s: %v", volume.Source, err)
				} else {
					logger.Info("[Orchestrator] Deleted Obiente volume %s", volume.Source)
				}
			} else if volume.IsNamed {
				// For Docker named volumes, only delete if container still exists and has cloud.obiente tags
				// This is safer - we don't want to delete volumes from containers that might not be Obiente-managed
				if containerExists {
					containerInfo := containerInfoResult.Container
					if containerInfo.Config != nil && containerInfo.Config.Labels != nil {
						if containerInfo.Config.Labels["cloud.obiente.managed"] == "true" {
							logger.Info("[Orchestrator] Removing Docker volume %s (container has cloud.obiente tags)", volume.Name)
						// Extract volume name from mount name if needed
						volumeName := volume.Name
						if volumeName == "" {
							// Try to extract from path
							if strings.Contains(volume.Source, "/volumes/") {
								parts := strings.Split(volume.Source, "/volumes/")
								if len(parts) > 1 {
									volumeName = strings.Split(parts[1], "/")[0]
								}
							}
						}
						if volumeName != "" {
							if _, err := dockerClient.VolumeRemove(ctx, volumeName, client.VolumeRemoveOptions{}); err != nil {
								logger.Warn("[Orchestrator] Failed to remove Docker volume %s: %v", volumeName, err)
							} else {
								logger.Info("[Orchestrator] Removed Docker volume %s", volumeName)
							}
						}
						} else {
							logger.Debug("[Orchestrator] Skipping Docker volume %s - container missing cloud.obiente tags", volume.Name)
						}
					}
				} else {
					logger.Debug("[Orchestrator] Skipping Docker volume %s - container no longer exists (cannot verify tags)", volume.Name)
				}
			}
			// Anonymous volumes are typically cleaned up when the container is removed
		}

		// Mark volumes as deleted
		database.DB.Model(&stray).Update("volumes_deleted_at", now)
		logger.Info("[Orchestrator] Marked volumes as deleted for container %s", stray.ContainerID[:12])
	}
}
