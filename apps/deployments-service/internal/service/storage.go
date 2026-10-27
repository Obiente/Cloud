package deployments

import (
	"context"
	"fmt"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"os/exec"
	"strings"
)

type StorageInfo = docker.StorageInfo

func CalculateStorage(ctx context.Context, _ string, ids []string) (*StorageInfo, error) {
	return docker.CalculateStorage(ctx, ids)
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
		return nil, fmt.Errorf("storage inventory is unavailable")
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
