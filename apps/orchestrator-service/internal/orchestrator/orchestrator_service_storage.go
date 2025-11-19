package orchestrator

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

// Storage operations for orchestrator service

type storageInfo struct {
	ImageSize     int64 // Docker image size in bytes
	VolumeSize    int64 // Total volume size in bytes
	ContainerDisk int64 // Container root filesystem usage in bytes
	TotalStorage  int64 // Total storage (image + volumes + container disk)
}

func (os *OrchestratorService) calculateStorage(ctx context.Context, imageName string, containerIDs []string) (*storageInfo, error) {
	info := &storageInfo{}

	// Get Docker client
	dcli, err := docker.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dcli.Close()

	// 1. Get image size
	if imageName != "" {
		imageSize, err := getImageSize(ctx, imageName)
		if err != nil {
			logger.Warn("[calculateStorage] Failed to get image size for %s: %v", imageName, err)
		} else {
			info.ImageSize = imageSize
		}
	}

	// 2. Get volume sizes and container disk usage for all containers
	totalVolumeSize := int64(0)
	totalContainerDisk := int64(0)

	for _, containerID := range containerIDs {
		// Get volume sizes
		volumeSize, err := os.getContainerVolumeSize(ctx, dcli, containerID)
		if err != nil {
			logger.Warn("[calculateStorage] Failed to get volume size for container %s: %v", containerID, err)
		} else {
			totalVolumeSize += volumeSize
		}

		// Get container root filesystem disk usage
		containerDisk, err := os.getContainerDiskUsage(ctx, dcli, containerID)
		if err != nil {
			logger.Warn("[calculateStorage] Failed to get container disk usage for %s: %v", containerID, err)
		} else {
			totalContainerDisk += containerDisk
		}
	}

	info.VolumeSize = totalVolumeSize
	info.ContainerDisk = totalContainerDisk
	info.TotalStorage = info.ImageSize + info.VolumeSize + info.ContainerDisk

	return info, nil
}

func getImageSize(ctx context.Context, imageName string) (int64, error) {
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", imageName, "--format", "{{.Size}}")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get image size: %w", err)
	}

	sizeStr := strings.TrimSpace(string(output))
	var size int64
	if _, err := fmt.Sscanf(sizeStr, "%d", &size); err != nil {
		return 0, fmt.Errorf("failed to parse image size: %w", err)
	}

	return size, nil
}

func (os *OrchestratorService) getContainerVolumeSize(ctx context.Context, dcli *docker.Client, containerID string) (int64, error) {
	containerInfo, err := dcli.ContainerInspect(ctx, containerID)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect container: %w", err)
	}

	totalSize := int64(0)

	volumes, err := dcli.GetContainerVolumes(ctx, containerID)
	if err != nil {
		logger.Warn("[getContainerVolumeSize] Failed to get volumes for container %s: %v", containerID, err)
		for _, mount := range containerInfo.Mounts {
			if mount.Type == "volume" || (mount.Type == "bind" && strings.HasPrefix(mount.Source, "/var/lib/obiente/volumes")) {
				size, err := getDirectorySize(ctx, mount.Source)
				if err != nil {
					logger.Warn("[getContainerVolumeSize] Failed to get size for volume %s: %v", mount.Source, err)
					continue
				}
				totalSize += size
			}
		}
	} else {
		for _, volume := range volumes {
			size, err := getDirectorySize(ctx, volume.Source)
			if err != nil {
				logger.Warn("[getContainerVolumeSize] Failed to get size for volume %s: %v", volume.Source, err)
				continue
			}
			totalSize += size
		}
	}

	return totalSize, nil
}

func (os *OrchestratorService) getContainerDiskUsage(ctx context.Context, dcli *docker.Client, containerID string) (int64, error) {
	cmd := []string{"sh", "-c", "du -sb / 2>/dev/null | cut -f1"}
	output, err := dcli.ContainerExecRun(ctx, containerID, cmd)
	if err != nil {
		// Fallback: try df command
		cmd = []string{"df", "-B1", "/"}
		output, err = dcli.ContainerExecRun(ctx, containerID, cmd)
		if err != nil {
			return 0, fmt.Errorf("failed to get disk usage: %w", err)
		}

		lines := strings.Split(strings.TrimSpace(output), "\n")
		if len(lines) < 2 {
			return 0, fmt.Errorf("unexpected df output format")
		}

		fields := strings.Fields(lines[1])
		if len(fields) < 3 {
			return 0, fmt.Errorf("unexpected df output format")
		}

		var used int64
		if _, err := fmt.Sscanf(fields[2], "%d", &used); err != nil {
			return 0, fmt.Errorf("failed to parse used size: %w", err)
		}

		return used, nil
	}

	sizeStr := strings.TrimSpace(output)
	var size int64
	if _, err := fmt.Sscanf(sizeStr, "%d", &size); err != nil {
		return 0, fmt.Errorf("failed to parse disk usage: %w", err)
	}

	return size, nil
}

func getDirectorySize(ctx context.Context, path string) (int64, error) {
	cmd := exec.CommandContext(ctx, "du", "-sb", path)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get directory size: %w", err)
	}

	parts := strings.Fields(string(output))
	if len(parts) < 1 {
		return 0, fmt.Errorf("unexpected du output format")
	}

	var size int64
	if _, err := fmt.Sscanf(parts[0], "%d", &size); err != nil {
		return 0, fmt.Errorf("failed to parse size: %w", err)
	}

	return size, nil
}

func (os *OrchestratorService) updateStoragePeriodically() {
	// Run every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Debug("[Orchestrator] Updating storage for all running deployments...")

			// Get all running deployments
			var locations []database.DeploymentLocation
			if err := database.DB.Where("status = ?", "running").Find(&locations).Error; err != nil {
				logger.Warn("[Orchestrator] Failed to get running deployments: %v", err)
				continue
			}

			// Group by deployment ID
			deploymentMap := make(map[string][]database.DeploymentLocation)
			for _, loc := range locations {
				deploymentMap[loc.DeploymentID] = append(deploymentMap[loc.DeploymentID], loc)
			}

			logger.Debug("[Orchestrator] Updating storage for %d deployments", len(deploymentMap))

			// Process deployments in parallel batches
			const batchSize = 5 // Process 5 deployments concurrently
			var wg sync.WaitGroup
			var mu sync.Mutex
			updatedCount := 0
			errorCount := 0

			deploymentIDs := make([]string, 0, len(deploymentMap))
			for depID := range deploymentMap {
				deploymentIDs = append(deploymentIDs, depID)
			}

			for i := 0; i < len(deploymentIDs); i += batchSize {
				end := i + batchSize
				if end > len(deploymentIDs) {
					end = len(deploymentIDs)
				}
				batch := deploymentIDs[i:end]

				for _, deploymentID := range batch {
					wg.Add(1)
					go func(depID string) {
						defer wg.Done()

						ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
						defer cancel()

						// Get container IDs for this deployment
						containerIDs := make([]string, 0)
						if locs, ok := deploymentMap[depID]; ok {
							for _, loc := range locs {
								if loc.ContainerID != "" {
									containerIDs = append(containerIDs, loc.ContainerID)
								}
							}
						}

						if len(containerIDs) == 0 {
							return
						}

						// Get deployment to find image name
						var deployment database.Deployment
						if err := database.DB.Where("id = ?", depID).First(&deployment).Error; err != nil {
							logger.Warn("[Orchestrator] Failed to get deployment %s: %v", depID, err)
							return
						}

						imageName := ""
						if deployment.Image != nil {
							imageName = *deployment.Image
						}

						// Calculate storage
						storageInfo, err := os.calculateStorage(ctx, imageName, containerIDs)
						if err != nil {
							logger.Warn("[Orchestrator] Failed to calculate storage for deployment %s: %v", depID, err)
							mu.Lock()
							errorCount++
							mu.Unlock()
							return
						}

						// Update storage in database
						if err := database.DB.Model(&database.Deployment{}).
							Where("id = ?", depID).
							Update("storage_bytes", storageInfo.TotalStorage).Error; err != nil {
							logger.Warn("[Orchestrator] Failed to update storage for deployment %s: %v", depID, err)
							mu.Lock()
							errorCount++
							mu.Unlock()
							return
						}

						mu.Lock()
						updatedCount++
						mu.Unlock()
					}(deploymentID)
				}

				wg.Wait()
			}

			logger.Debug("[Orchestrator] Storage update completed: %d updated, %d errors", updatedCount, errorCount)
		case <-os.ctx.Done():
			return
		}
	}
}
