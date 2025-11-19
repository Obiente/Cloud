package orchestrator

import (
	"os"
	"path/filepath"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

// Cleanup operations for deployments

func (dm *DeploymentManager) cleanupDeploymentData(deploymentID string) {
	logger.Info("[DeploymentManager] Cleaning up data for deployment %s", deploymentID)

	// List of directories to clean up
	cleanupDirs := []string{
		// Volumes directory
		filepath.Join("/var/lib/obiente/volumes", deploymentID),
		// Deployment directory (for compose files)
		filepath.Join("/var/lib/obiente/deployments", deploymentID),
		// Build directory
		filepath.Join("/var/lib/obiente/builds", deploymentID),
		// Fallback temp directories
		filepath.Join("/var/obiente/tmp/obiente-volumes", deploymentID),
		filepath.Join("/var/obiente/tmp/obiente-deployments", deploymentID),
		filepath.Join("/tmp/obiente-volumes", deploymentID),
		filepath.Join("/tmp/obiente-deployments", deploymentID),
	}

	for _, dir := range cleanupDirs {
		if err := os.RemoveAll(dir); err != nil {
			if !os.IsNotExist(err) {
				logger.Info("[DeploymentManager] Failed to remove directory %s: %v", dir, err)
			}
		} else {
			logger.Info("[DeploymentManager] Removed directory %s", dir)
		}
	}
}

// Close closes all connections
