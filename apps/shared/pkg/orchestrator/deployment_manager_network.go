package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

// Network operations for deployments

func (dm *DeploymentManager) getSwarmNetworkName(ctx context.Context) (string, error) {
	// Try multiple approaches to find the network
	// 1. Look for exact match: obiente_obiente-network (external network)
	checkCmd := exec.CommandContext(ctx, "docker", "network", "inspect", "obiente_obiente-network", "--format", "{{.Name}}")
	output, err := checkCmd.Output()
	if err == nil {
		networkName := strings.TrimSpace(string(output))
		if networkName != "" {
			logger.Debug("[DeploymentManager] Found Swarm network (exact match): %s", networkName)
			return networkName, nil
		}
	}

	// 2. List all networks and find one matching the pattern
	listCmd := exec.CommandContext(ctx, "docker", "network", "ls", "--format", "{{.Name}}")
	output, err = listCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list networks: %w", err)
	}

	networks := strings.Split(strings.TrimSpace(string(output)), "\n")
	// Priority order: exact match > stack-prefixed > simple name
	var exactMatch, stackPrefixed, simpleName string
	for _, network := range networks {
		network = strings.TrimSpace(network)
		if network == "" {
			continue
		}
		if network == "obiente_obiente-network" {
			exactMatch = network
		} else if strings.HasSuffix(network, "_obiente-network") {
			if stackPrefixed == "" {
				stackPrefixed = network
			}
		} else if network == "obiente-network" {
			simpleName = network
		}
	}

	// Return in priority order
	if exactMatch != "" {
		logger.Debug("[DeploymentManager] Found Swarm network (exact): %s", exactMatch)
		return exactMatch, nil
	}
	if stackPrefixed != "" {
		logger.Debug("[DeploymentManager] Found Swarm network (stack-prefixed): %s", stackPrefixed)
		return stackPrefixed, nil
	}
	if simpleName != "" {
		logger.Debug("[DeploymentManager] Found Swarm network (simple): %s", simpleName)
		return simpleName, nil
	}

	// Fallback: use the expected name (will fail if network doesn't exist, but that's better than silent failure)
	fallbackName := "obiente_obiente-network"
	logger.Warn("[DeploymentManager] Network not found in network list, using fallback name: %s", fallbackName)
	return fallbackName, nil
}

func (dm *DeploymentManager) ensureNetwork(ctx context.Context) error {
	// Use exec to check and create network since Docker API types may vary
	// Check if network exists
	checkCmd := exec.CommandContext(ctx, "docker", "network", "ls", "--filter", fmt.Sprintf("name=%s", dm.networkName), "--format", "{{.Name}}")
	output, err := checkCmd.Output()
	if err != nil {
		// Check if Docker is available
		if exitError, ok := err.(*exec.ExitError); ok {
			stderr := string(exitError.Stderr)
			logger.Info("[DeploymentManager] Failed to check for network (exit code %d): %s", exitError.ExitCode(), stderr)
			// If Docker is not available, return a more helpful error
			if strings.Contains(stderr, "Cannot connect to the Docker daemon") ||
				strings.Contains(stderr, "Is the docker daemon running") {
				return fmt.Errorf("docker daemon is not accessible: %s", stderr)
			}
		}
		logger.Warn("[DeploymentManager] Failed to check for network: %v", err)
	}

	if strings.TrimSpace(string(output)) == dm.networkName {
		logger.Info("[DeploymentManager] Network %s already exists", dm.networkName)
		return nil
	}

	// Network doesn't exist, create it
	logger.Info("[DeploymentManager] Creating network %s", dm.networkName)
	createCmd := exec.CommandContext(ctx, "docker", "network", "create", "--driver", "bridge", "--label", "cloud.obiente.managed=true", dm.networkName)
	var stderr bytes.Buffer
	createCmd.Stderr = &stderr
	if err := createCmd.Run(); err != nil {
		// Check if network was created by another process (race condition)
		output, checkErr := checkCmd.Output()
		if checkErr == nil && strings.TrimSpace(string(output)) == dm.networkName {
			logger.Info("[DeploymentManager] Network %s was created by another process", dm.networkName)
			return nil
		}

		// Capture stderr for better error messages
		errorOutput := stderr.String()
		if errorOutput == "" {
			if exitError, ok := err.(*exec.ExitError); ok {
				errorOutput = string(exitError.Stderr)
			}
		}

		// Provide more specific error messages
		if strings.Contains(errorOutput, "already exists") {
			logger.Info("[DeploymentManager] Network %s already exists (race condition)", dm.networkName)
			return nil
		}
		if strings.Contains(errorOutput, "Cannot connect to the Docker daemon") ||
			strings.Contains(errorOutput, "Is the docker daemon running") {
			return fmt.Errorf("docker daemon is not accessible: %s", errorOutput)
		}
		if strings.Contains(errorOutput, "permission denied") {
			return fmt.Errorf("permission denied: unable to create Docker network (check Docker permissions): %s", errorOutput)
		}

		logger.Info("[DeploymentManager] Failed to create network: %v, stderr: %s", err, errorOutput)
		return fmt.Errorf("failed to create network: %w (stderr: %s)", err, errorOutput)
	}

	logger.Info("[DeploymentManager] Successfully created network %s", dm.networkName)
	return nil
}

// ensureDeploymentNetwork creates a per-deployment network for service-to-service communication
// Each deployment gets its own isolated network so services can discover each other via DNS
func (dm *DeploymentManager) ensureDeploymentNetwork(ctx context.Context, deploymentNetworkName string) error {
	// Check if network exists
	checkCmd := exec.CommandContext(ctx, "docker", "network", "ls", "--filter", fmt.Sprintf("name=%s", deploymentNetworkName), "--format", "{{.Name}}")
	output, err := checkCmd.Output()
	if err != nil {
		// Check if Docker is available
		if exitError, ok := err.(*exec.ExitError); ok {
			stderr := string(exitError.Stderr)
			logger.Info("[DeploymentManager] Failed to check for deployment network (exit code %d): %s", exitError.ExitCode(), stderr)
			if strings.Contains(stderr, "Cannot connect to the Docker daemon") ||
				strings.Contains(stderr, "Is the docker daemon running") {
				return fmt.Errorf("docker daemon is not accessible: %s", stderr)
			}
		}
		logger.Warn("[DeploymentManager] Failed to check for deployment network: %v", err)
	}

	if strings.TrimSpace(string(output)) == deploymentNetworkName {
		logger.Info("[DeploymentManager] Deployment network %s already exists", deploymentNetworkName)
		return nil
	}

	// Network doesn't exist, create it
	logger.Info("[DeploymentManager] Creating deployment network %s", deploymentNetworkName)
	
	// Detect if Docker Swarm is active and use overlay driver if so
	args := []string{"network", "create"}
	swarmCheckCmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{.Swarm.LocalNodeState}}")
	if swarmOutput, swarmErr := swarmCheckCmd.Output(); swarmErr == nil {
		if strings.TrimSpace(string(swarmOutput)) == "active" {
			logger.Info("[DeploymentManager] Swarm detected, creating overlay network for %s", deploymentNetworkName)
			args = append(args, "--driver", "overlay", "--attachable")
		}
	}
	
	args = append(args,
		"--label", "cloud.obiente.managed=true",
		"--label", fmt.Sprintf("cloud.obiente.deployment=%s", strings.TrimPrefix(deploymentNetworkName, "deployment-")),
		deploymentNetworkName,
	)
	
	createCmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	createCmd.Stderr = &stderr
	if err := createCmd.Run(); err != nil {
		// Check if network was created by another process (race condition)
		output, checkErr := checkCmd.Output()
		if checkErr == nil && strings.TrimSpace(string(output)) == deploymentNetworkName {
			logger.Info("[DeploymentManager] Deployment network %s was created by another process", deploymentNetworkName)
			return nil
		}

		// Capture stderr for better error messages
		errorOutput := stderr.String()
		if errorOutput == "" {
			if exitError, ok := err.(*exec.ExitError); ok {
				errorOutput = string(exitError.Stderr)
			}
		}

		// Provide more specific error messages
		if strings.Contains(errorOutput, "already exists") {
			logger.Info("[DeploymentManager] Deployment network %s already exists (race condition)", deploymentNetworkName)
			return nil
		}
		if strings.Contains(errorOutput, "Cannot connect to the Docker daemon") ||
			strings.Contains(errorOutput, "Is the docker daemon running") {
			return fmt.Errorf("docker daemon is not accessible: %s", errorOutput)
		}
		if strings.Contains(errorOutput, "permission denied") {
			return fmt.Errorf("permission denied: unable to create Docker network (check Docker permissions): %s", errorOutput)
		}

		logger.Info("[DeploymentManager] Failed to create deployment network: %v, stderr: %s", err, errorOutput)
		return fmt.Errorf("failed to create deployment network: %w (stderr: %s)", err, errorOutput)
	}

	logger.Info("[DeploymentManager] Successfully created deployment network %s", deploymentNetworkName)
	return nil
}