package deployments

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"api/docker"
)

// StorageInfo contains storage information for a deployment
type StorageInfo struct {
	ImageSize     int64 // Docker image size in bytes
	VolumeSize    int64 // Total volume size in bytes
	ContainerDisk int64 // Container root filesystem usage in bytes
	TotalStorage  int64 // Total storage (image + volumes + container disk)
}

// CalculateStorage calculates total storage for a deployment
func CalculateStorage(ctx context.Context, imageName string, containerIDs []string) (*StorageInfo, error) {
	info := &StorageInfo{}

	// Get Docker client
	dcli, err := docker.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dcli.Close()

	// 1. Get image size
	if imageName != "" {
		// Use the getImageSize function from build_strategy.go
		imageSize, err := getImageSize(ctx, imageName)
		if err != nil {
			log.Printf("[CalculateStorage] Failed to get image size for %s: %v", imageName, err)
		} else {
			info.ImageSize = imageSize
		}
	}

	// 2. Get volume sizes and container disk usage for all containers
	totalVolumeSize := int64(0)
	totalContainerDisk := int64(0)

	for _, containerID := range containerIDs {
		// Get volume sizes
		volumeSize, err := getContainerVolumeSize(ctx, dcli, containerID)
		if err != nil {
			log.Printf("[CalculateStorage] Failed to get volume size for container %s: %v", containerID, err)
		} else {
			totalVolumeSize += volumeSize
		}

		// Get container root filesystem disk usage
		containerDisk, err := getContainerDiskUsage(ctx, dcli, containerID)
		if err != nil {
			log.Printf("[CalculateStorage] Failed to get container disk usage for %s: %v", containerID, err)
		} else {
			totalContainerDisk += containerDisk
		}
	}

	info.VolumeSize = totalVolumeSize
	info.ContainerDisk = totalContainerDisk
	info.TotalStorage = info.ImageSize + info.VolumeSize + info.ContainerDisk

	return info, nil
}

// getContainerVolumeSize calculates total size of all volumes attached to a container
func getContainerVolumeSize(ctx context.Context, dcli *docker.Client, containerID string) (int64, error) {
	// Inspect container to get volume mounts
	containerInfo, err := dcli.ContainerInspect(ctx, containerID)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect container: %w", err)
	}

	totalSize := int64(0)

	// Get all volume mounts
	volumes, err := dcli.GetContainerVolumes(ctx, containerID)
	if err != nil {
		log.Printf("[getContainerVolumeSize] Failed to get volumes for container %s: %v", containerID, err)
		// Try to get size from mounts directly
		for _, mount := range containerInfo.Mounts {
			if mount.Type == "volume" || (mount.Type == "bind" && strings.HasPrefix(mount.Source, "/var/lib/obiente/volumes")) {
				size, err := getDirectorySize(ctx, mount.Source)
				if err != nil {
					log.Printf("[getContainerVolumeSize] Failed to get size for volume %s: %v", mount.Source, err)
					continue
				}
				totalSize += size
			}
		}
	} else {
		// Use GetContainerVolumes helper
		for _, volume := range volumes {
			size, err := getDirectorySize(ctx, volume.Source)
			if err != nil {
				log.Printf("[getContainerVolumeSize] Failed to get size for volume %s: %v", volume.Source, err)
				continue
			}
			totalSize += size
		}
	}

	return totalSize, nil
}

// getContainerDiskUsage gets the root filesystem disk usage of a container
func getContainerDiskUsage(ctx context.Context, dcli *docker.Client, containerID string) (int64, error) {
	// Use docker exec to run du -sb / inside the container
	// This gives us the total size of the container's root filesystem
	cmd := []string{"sh", "-c", "du -sb / 2>/dev/null | cut -f1"}

	output, err := dcli.ContainerExecRun(ctx, containerID, cmd)
	if err != nil {
		// Fallback: try to get from container stats or use df
		return getContainerDiskUsageFallback(ctx, dcli, containerID)
	}

	// Parse output
	sizeStr := strings.TrimSpace(output)
	var size int64
	if _, err := fmt.Sscanf(sizeStr, "%d", &size); err != nil {
		return 0, fmt.Errorf("failed to parse disk usage: %w", err)
	}

	return size, nil
}

// getContainerDiskUsageFallback tries alternative methods to get container disk usage
func getContainerDiskUsageFallback(ctx context.Context, dcli *docker.Client, containerID string) (int64, error) {
	// Try using df command inside container
	cmd := []string{"df", "-B1", "/"}
	output, err := dcli.ContainerExecRun(ctx, containerID, cmd)
	if err != nil {
		return 0, fmt.Errorf("failed to get disk usage: %w", err)
	}

	// Parse df output (skip header line, get used size from second line)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return 0, fmt.Errorf("unexpected df output format")
	}

	// df output format: Filesystem 1B-blocks Used Available Use% Mounted on
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

// getDirectorySize calculates the total size of a directory
func getDirectorySize(ctx context.Context, path string) (int64, error) {
	// Use du command to get directory size
	cmd := exec.CommandContext(ctx, "du", "-sb", path)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get directory size: %w", err)
	}

	// Parse output (format: "size\tpath")
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

// CalculateStorageForDeployment calculates storage for all containers in a deployment
func CalculateStorageForDeployment(ctx context.Context, deploymentID string) (*StorageInfo, error) {
	// Get Docker client
	dcli, err := docker.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dcli.Close()

	// Find all containers for this deployment
	// Use label filter to find containers
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", fmt.Sprintf("label=cloud.obiente.deployment_id=%s", deploymentID), "--format", "{{.ID}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	containerIDs := []string{}
	if outputStr := strings.TrimSpace(string(output)); outputStr != "" {
		containerIDs = strings.Split(outputStr, "\n")
	}

	if len(containerIDs) == 0 {
		// No containers found, return empty storage
		return &StorageInfo{}, nil
	}

	// Get image name from first container
	imageName := ""
	if len(containerIDs) > 0 {
		containerInfo, err := dcli.ContainerInspect(ctx, containerIDs[0])
		if err == nil && containerInfo.Config != nil {
			imageName = containerInfo.Config.Image
		}
	}

	return CalculateStorage(ctx, imageName, containerIDs)

}
