package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/utils"
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

func (dm *DeploymentManager) ensurePreviewIngressNetwork(ctx context.Context, networkName string) error {
	if strings.TrimSpace(networkName) == "" {
		return fmt.Errorf("preview ingress network name is required")
	}
	if err := dm.ensureDeploymentNetwork(ctx, networkName); err != nil {
		return err
	}
	if utils.IsSwarmModeEnabled() {
		return ensureSwarmIngressProxyNetwork(ctx, networkName)
	}
	return ensureContainerIngressProxyNetwork(ctx, networkName)
}

func ensureSwarmIngressProxyNetwork(ctx context.Context, networkName string) error {
	serviceName, err := swarmIngressProxyServiceName(ctx)
	if err != nil {
		return err
	}
	existingNetworks, err := existingSwarmServiceNetworkNames(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("inspect ingress proxy service %s: %w", serviceName, err)
	}
	if containsString(existingNetworks, networkName) {
		return nil
	}
	command := exec.CommandContext(ctx, "docker", "service", "update", "--network-add", networkName, serviceName)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("attach ingress proxy service %s to %s: %w (%s)", serviceName, networkName, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func swarmIngressProxyServiceName(ctx context.Context) (string, error) {
	if serviceName := strings.TrimSpace(os.Getenv("TRAEFIK_SERVICE_NAME")); serviceName != "" {
		return serviceName, nil
	}
	command := exec.CommandContext(ctx, "docker", "service", "ls", "--filter", "label=cloud.obiente.ingress-proxy=true", "--format", "{{.Name}}")
	if output, err := command.Output(); err == nil {
		serviceNames := strings.Fields(string(output))
		switch len(serviceNames) {
		case 1:
			return serviceNames[0], nil
		case 0:
			// The label may not exist yet during a rolling upgrade. Fall back to
			// the established stack-derived service name below.
		default:
			return "", fmt.Errorf("multiple ingress proxy services matched cloud.obiente.ingress-proxy=true")
		}
	}
	stackName := strings.TrimSpace(os.Getenv("STACK_NAME"))
	if stackName == "" {
		stackName = "obiente"
	}
	return stackName + "_traefik", nil
}

func ensureContainerIngressProxyNetwork(ctx context.Context, networkName string) error {
	command := exec.CommandContext(ctx, "docker", "ps", "--filter", "label=cloud.obiente.ingress-proxy=true", "--format", "{{.ID}}")
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("discover ingress proxy container: %w", err)
	}
	containerIDs := strings.Fields(string(output))
	if len(containerIDs) == 0 {
		return fmt.Errorf("no ingress proxy container with cloud.obiente.ingress-proxy=true is running")
	}
	for _, containerID := range containerIDs {
		connect := exec.CommandContext(ctx, "docker", "network", "connect", networkName, containerID)
		output, err := connect.CombinedOutput()
		if err != nil && !strings.Contains(strings.ToLower(string(output)), "already exists") {
			return fmt.Errorf("attach ingress proxy container %s to %s: %w (%s)", containerID, networkName, err, strings.TrimSpace(string(output)))
		}
	}
	return nil
}

// RemovePreviewIngressNetwork detaches the ingress proxy and removes the
// managed network created exclusively for one pull request preview.
func (dm *DeploymentManager) RemovePreviewIngressNetwork(ctx context.Context, deploymentID string) error {
	networkName := PreviewIngressNetworkNameForDeployment(deploymentID)
	exists, err := managedPreviewNetworkExists(ctx, networkName, deploymentID)
	if err != nil || !exists {
		return err
	}
	if utils.IsSwarmModeEnabled() {
		if err := removeSwarmIngressProxyNetwork(ctx, networkName); err != nil {
			return err
		}
	} else if err := removeContainerIngressProxyNetwork(ctx, networkName); err != nil {
		return err
	}

	var lastErr error
	var lastOutput string
	for attempt := 0; attempt < 10; attempt++ {
		command := exec.CommandContext(ctx, "docker", "network", "rm", networkName)
		output, removeErr := command.CombinedOutput()
		lastErr = removeErr
		lastOutput = strings.TrimSpace(string(output))
		if removeErr == nil || dockerNetworkMissing(lastOutput) {
			return nil
		}
		if err := waitForPreviewNetworkRetry(ctx, 500*time.Millisecond); err != nil {
			return err
		}
	}
	return fmt.Errorf("remove preview ingress network %s: %w (%s)", networkName, lastErr, lastOutput)
}

func managedPreviewNetworkExists(ctx context.Context, networkName, deploymentID string) (bool, error) {
	command := exec.CommandContext(ctx, "docker", "network", "inspect", "--format", "{{index .Labels \"cloud.obiente.managed\"}}\t{{index .Labels \"cloud.obiente.deployment\"}}", networkName)
	output, err := command.CombinedOutput()
	formatted := strings.TrimSpace(string(output))
	if err != nil {
		if dockerNetworkMissing(formatted) {
			return false, nil
		}
		return false, fmt.Errorf("inspect preview ingress network %s: %w (%s)", networkName, err, formatted)
	}
	parts := strings.SplitN(formatted, "\t", 2)
	if len(parts) != 2 || parts[0] != "true" || parts[1] != deploymentID {
		return false, fmt.Errorf("refusing to remove network %s without matching managed preview ownership", networkName)
	}
	return true, nil
}

func removeSwarmIngressProxyNetwork(ctx context.Context, networkName string) error {
	serviceName, err := swarmIngressProxyServiceName(ctx)
	if err != nil {
		return err
	}
	existingNetworks, err := existingSwarmServiceNetworkNames(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("inspect ingress proxy service %s: %w", serviceName, err)
	}
	if !containsString(existingNetworks, networkName) {
		return nil
	}
	command := exec.CommandContext(ctx, "docker", "service", "update", "--network-rm", networkName, serviceName)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("detach ingress proxy service %s from %s: %w (%s)", serviceName, networkName, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func removeContainerIngressProxyNetwork(ctx context.Context, networkName string) error {
	command := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "label=cloud.obiente.ingress-proxy=true", "--format", "{{.ID}}")
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("discover ingress proxy container: %w", err)
	}
	for _, containerID := range strings.Fields(string(output)) {
		disconnect := exec.CommandContext(ctx, "docker", "network", "disconnect", "--force", networkName, containerID)
		disconnectOutput, disconnectErr := disconnect.CombinedOutput()
		message := strings.ToLower(strings.TrimSpace(string(disconnectOutput)))
		if disconnectErr != nil && !dockerNetworkMissing(message) && !strings.Contains(message, "is not connected") {
			return fmt.Errorf("detach ingress proxy container %s from %s: %w (%s)", containerID, networkName, disconnectErr, strings.TrimSpace(string(disconnectOutput)))
		}
	}
	return nil
}

func dockerNetworkMissing(output string) bool {
	output = strings.ToLower(output)
	return strings.Contains(output, "no such network") || strings.Contains(output, "network not found")
}

func waitForPreviewNetworkRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
