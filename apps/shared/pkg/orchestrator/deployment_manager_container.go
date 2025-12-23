package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// Container operations for deployments

func (dm *DeploymentManager) createContainer(ctx context.Context, config *DeploymentConfig, name string, replicaIndex int, serviceName string) (string, error) {
	// Get routing rules for this deployment
	routings, _ := database.GetDeploymentRoutings(config.DeploymentID)

	// Prepare labels
	labels := map[string]string{
		"cloud.obiente.managed":       "true",
		"cloud.obiente.deployment_id": config.DeploymentID,
		"cloud.obiente.domain":        config.Domain,
		"cloud.obiente.service_name":  serviceName,
		"cloud.obiente.replica":       strconv.Itoa(replicaIndex),
	}

	// Get the actual Swarm network name (may be prefixed with stack name)
	swarmNetworkName, err := dm.getSwarmNetworkName(ctx)
	if err != nil {
		logger.Warn("[DeploymentManager] Failed to get Swarm network name, using fallback: %v", err)
		swarmNetworkName = "obiente_obiente-network" // Fallback to common name
	}

	// Generate Traefik labels from routing rules
	// Use config.Port for service port (which should be from routing target port if available)
	servicePort := config.Port
	traefikLabels := generateTraefikLabels(config.DeploymentID, serviceName, routings, &servicePort, swarmNetworkName)
	for k, v := range traefikLabels {
		labels[k] = v
	}
	// Only set cloud.obiente.traefik if we actually generated Traefik labels (i.e., routing rules exist)
	if len(traefikLabels) > 0 {
		labels["cloud.obiente.traefik"] = "true" // Required for Traefik discovery
	}

	// Determine health check port from routing rules only.
	// Priority: 1) Matching service routing, 2) First routing (if routings exist)
	healthCheckPort := 0
	if len(routings) > 0 {
		// First, try to find a routing that matches the service name
		for _, routing := range routings {
			serviceMatches := routing.ServiceName == serviceName ||
				(serviceName == "default" && (routing.ServiceName == "" || routing.ServiceName == "default")) ||
				(routing.ServiceName == "default" && serviceName == "")
			
			if serviceMatches && routing.TargetPort > 0 {
				healthCheckPort = routing.TargetPort
				logger.Info("[DeploymentManager] Using routing target port %d for health check (service: %s, routing service: %s)", healthCheckPort, serviceName, routing.ServiceName)
				break
			}
		}
		
		// If no exact match found but routings exist, use the first routing's target port
		// This ensures we always use routing port over config.Port when routings are available
		if healthCheckPort == 0 {
			for _, routing := range routings {
				if routing.TargetPort > 0 {
					healthCheckPort = routing.TargetPort
					logger.Info("[DeploymentManager] Using first available routing target port %d for health check (service: %s, routing service: %s) - no exact service match found", healthCheckPort, serviceName, routing.ServiceName)
					break
				}
			}
		}
	}

	// If no routing-based port, we will not add the default nc-based healthcheck.
	// (Custom healthcheck command does not require a port.)

	// Add custom labels
	for k, v := range config.Labels {
		labels[k] = v
	}

	// Prepare environment variables
	env := []string{}
	for k, v := range config.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Ensure netcat is available by adding environment variables for nixpacks/railpacks.
	// Only required for the default nc-based healthcheck (routing-based).
	shouldAddNetcatEnv := (config.HealthcheckType != nil && *config.HealthcheckType == 2) && // TCP check
		healthCheckPort > 0 && len(routings) > 0
	if shouldAddNetcatEnv {
		// Check if env vars are already set (don't override user's custom values)
		addedVars := []string{}
		if _, exists := config.EnvVars["NIXPACKS_APT_PKGS"]; !exists {
			env = append(env, "NIXPACKS_APT_PKGS=netcat-openbsd")
			addedVars = append(addedVars, "NIXPACKS_APT_PKGS")
		}
		if _, exists := config.EnvVars["RAILPACK_DEPLOY_APT_PACKAGES"]; !exists {
			// RAILPACK_DEPLOY_APT_PACKAGES installs packages in the final image (what we need for health checks)
			env = append(env, "RAILPACK_DEPLOY_APT_PACKAGES=netcat-openbsd")
			addedVars = append(addedVars, "RAILPACK_DEPLOY_APT_PACKAGES")
		}
		if len(addedVars) > 0 {
			logger.Info("[DeploymentManager] Added netcat installation env vars for container %s: %v (these work during build if using Nixpacks/Railpacks; health check will install netcat at runtime if needed)", name, addedVars)
		} else {
			logger.Debug("[DeploymentManager] Netcat installation env vars already set by user for container %s", name)
		}
	}

	// Determine container port - use routing target port if available, otherwise config.Port
	containerPortNum := config.Port
	if len(routings) > 0 {
		// Find routing for this service
		for _, routing := range routings {
			if routing.ServiceName == serviceName || (serviceName == "default" && (routing.ServiceName == "" || routing.ServiceName == "default")) {
				if routing.TargetPort > 0 {
					containerPortNum = routing.TargetPort
					logger.Debug("[DeploymentManager] Using routing target port %d for container port (service: %s)", containerPortNum, serviceName)
					break
				}
			}
		}
		// If no exact match, use first routing's target port
		if containerPortNum == config.Port && len(routings) > 0 && routings[0].TargetPort > 0 {
			containerPortNum = routings[0].TargetPort
			logger.Debug("[DeploymentManager] Using first routing target port %d for container port (service: %s)", containerPortNum, serviceName)
		}
	}

	// Prepare port bindings
	// When Traefik handles routing, we should NOT expose ports to the host
	// Only expose ports internally (no host binding) when Traefik labels are present
	exposedPorts := network.PortSet{}
	portBindings := network.PortMap{}

	if containerPortNum > 0 {
		containerPort, err := network.ParsePort(fmt.Sprintf("%d/tcp", containerPortNum))
		if err != nil {
			return "", fmt.Errorf("invalid port %d: %w", containerPortNum, err)
		}
		exposedPorts[containerPort] = struct{}{}

		// Only bind to host if Traefik is NOT handling routing
		// If Traefik labels exist, don't expose to host (Traefik will route internally)
		if len(traefikLabels) == 0 {
			// No Traefik routing - expose to host with random port for security
			hostIP, _ := netip.ParseAddr("0.0.0.0")
			portBindings[containerPort] = []network.PortBinding{
				{
					HostIP:   hostIP,
					HostPort: "0", // SECURITY: Docker assigns random port - users cannot bind to specific host ports
				},
			}
			logger.Debug("[DeploymentManager] Exposing container port %d to host (random port) - no Traefik routing", containerPortNum)
		} else {
			// Traefik handles routing - don't expose to host, only expose internally
			logger.Info("[DeploymentManager] Not exposing container port %d to host - Traefik will handle routing", containerPortNum)
		}
	}

	// Generate health check based on type
	var healthcheck *container.HealthConfig
	
	// Determine healthcheck port: use override if set, otherwise use detected port
	effectiveHealthCheckPort := healthCheckPort
	if config.HealthcheckPort != nil && *config.HealthcheckPort > 0 {
		effectiveHealthCheckPort = int(*config.HealthcheckPort)
	}
	
	// Check healthcheck type (default to DISABLED/UNSPECIFIED if not set)
	healthcheckType := int32(0) // HEALTHCHECK_TYPE_UNSPECIFIED
	if config.HealthcheckType != nil {
		healthcheckType = *config.HealthcheckType
	}
	
	switch healthcheckType {
	case 1: // HEALTHCHECK_DISABLED
		// Explicitly disabled - no healthcheck
		logger.Info("[DeploymentManager] Health check explicitly disabled for container %s", name)
		healthcheck = nil
		
	case 2: // HEALTHCHECK_TCP
		if effectiveHealthCheckPort > 0 {
			// TCP port check using netcat
			healthCheckCmd := fmt.Sprintf(`sh -c 'if command -v nc >/dev/null 2>&1; then nc -z localhost %d || exit 1; else (apk add --no-cache netcat-openbsd >/dev/null 2>&1 || apt-get update -qq && apt-get install -y -qq netcat-openbsd >/dev/null 2>&1 || yum install -y -q nc >/dev/null 2>&1) && nc -z localhost %d || exit 1; fi'`, effectiveHealthCheckPort, effectiveHealthCheckPort)
			healthcheck = &container.HealthConfig{
				Test:        []string{"CMD-SHELL", healthCheckCmd},
				Interval:    30 * time.Second,
				Timeout:     10 * time.Second,
				Retries:     3,
				StartPeriod: 40 * time.Second,
			}
			logger.Info("[DeploymentManager] Added TCP health check for container %s on port %d", name, effectiveHealthCheckPort)
		} else {
			logger.Warn("[DeploymentManager] TCP health check requested but no port available for container %s", name)
		}
		
	case 3: // HEALTHCHECK_HTTP
		if effectiveHealthCheckPort > 0 {
			// HTTP endpoint check using curl
			path := "/"
			if config.HealthcheckPath != nil && *config.HealthcheckPath != "" {
				path = *config.HealthcheckPath
			}
			expectedStatus := 200
			if config.HealthcheckExpectedStatus != nil && *config.HealthcheckExpectedStatus > 0 {
				expectedStatus = int(*config.HealthcheckExpectedStatus)
			}
			// HTTP check: curl the endpoint and check status code
			healthCheckCmd := fmt.Sprintf(`sh -c 'if command -v curl >/dev/null 2>&1; then status=$(curl -s -o /dev/null -w "%%{http_code}" http://localhost:%d%s); [ "$status" -eq "%d" ] && exit 0 || exit 1; else (apk add --no-cache curl >/dev/null 2>&1 || apt-get update -qq && apt-get install -y -qq curl >/dev/null 2>&1 || yum install -y -q curl >/dev/null 2>&1) && status=$(curl -s -o /dev/null -w "%%{http_code}" http://localhost:%d%s); [ "$status" -eq "%d" ] && exit 0 || exit 1; fi'`, effectiveHealthCheckPort, path, expectedStatus, effectiveHealthCheckPort, path, expectedStatus)
			healthcheck = &container.HealthConfig{
				Test:        []string{"CMD-SHELL", healthCheckCmd},
				Interval:    30 * time.Second,
				Timeout:     10 * time.Second,
				Retries:     3,
				StartPeriod: 40 * time.Second,
			}
			logger.Info("[DeploymentManager] Added HTTP health check for container %s on port %d%s (expecting %d)", name, effectiveHealthCheckPort, path, expectedStatus)
		} else {
			logger.Warn("[DeploymentManager] HTTP health check requested but no port available for container %s", name)
		}
		
	case 4: // HEALTHCHECK_CUSTOM
		if config.HealthcheckCustomCommand != nil && *config.HealthcheckCustomCommand != "" {
			// Use custom command (already sanitized in CRUD layer)
			healthcheck = &container.HealthConfig{
				Test:        []string{"CMD-SHELL", *config.HealthcheckCustomCommand},
				Interval:    30 * time.Second,
				Timeout:     10 * time.Second,
				Retries:     3,
				StartPeriod: 40 * time.Second,
			}
			logger.Info("[DeploymentManager] Added custom health check for container %s: %s", name, *config.HealthcheckCustomCommand)
		} else {
			logger.Warn("[DeploymentManager] Custom health check requested but no command provided for container %s", name)
		}
		
	default: // HEALTHCHECK_TYPE_UNSPECIFIED (0) or unknown
		// Auto-detect: Use TCP check if routing exists
		if effectiveHealthCheckPort > 0 && len(routings) > 0 {
			healthCheckCmd := fmt.Sprintf(`sh -c 'if command -v nc >/dev/null 2>&1; then nc -z localhost %d || exit 1; else (apk add --no-cache netcat-openbsd >/dev/null 2>&1 || apt-get update -qq && apt-get install -y -qq netcat-openbsd >/dev/null 2>&1 || yum install -y -q nc >/dev/null 2>&1) && nc -z localhost %d || exit 1; fi'`, effectiveHealthCheckPort, effectiveHealthCheckPort)
			healthcheck = &container.HealthConfig{
				Test:        []string{"CMD-SHELL", healthCheckCmd},
				Interval:    30 * time.Second,
				Timeout:     10 * time.Second,
				Retries:     3,
				StartPeriod: 40 * time.Second,
			}
			logger.Info("[DeploymentManager] Added auto TCP health check for container %s on port %d (routing exists)", name, effectiveHealthCheckPort)
		} else {
			logger.Info("[DeploymentManager] No health check for container %s - type unspecified and no routing rules", name)
		}
	}

	// Container configuration
	containerConfig := &container.Config{
		Image:        config.Image,
		Env:          env, // Environment variables including NIXPACKS_PKGS/RAILPACK_* if health check is needed
		Labels:       labels,
		ExposedPorts: exposedPorts,
		// Clear ENTRYPOINT to avoid conflicts when overriding CMD
		Entrypoint:  []string{},
		Healthcheck: healthcheck, // Only set if we have a valid port from routing
	}

	// Log environment variables for debugging (only curl-related ones to avoid spam)
	if healthCheckPort > 0 {
		curlEnvVars := []string{}
		for _, e := range env {
			if strings.Contains(e, "NIXPACKS") || strings.Contains(e, "RAILPACK") {
				curlEnvVars = append(curlEnvVars, e)
			}
		}
		if len(curlEnvVars) > 0 {
			logger.Debug("[DeploymentManager] Container %s will have these curl-related env vars: %v", name, curlEnvVars)
		}
	}

	// Override container CMD if start command is provided
	if config.StartCommand != nil && *config.StartCommand != "" {
		// Use sh -c to preserve working directory and handle relative paths
		containerConfig.Cmd = []string{"sh", "-c", *config.StartCommand}
	}

	// Convert CPU shares to NanoCPUs for hard CPU limit
	// CPUShares is in units where 1024 = 1 CPU core
	// NanoCPUs: 1 CPU = 1,000,000,000 nanoseconds (1e9)
	// This sets an absolute CPU limit, not just relative priority
	cpuCores := float64(config.CPUShares) / 1024.0
	nanoCPUs := int64(cpuCores * 1e9)

	// Host configuration
	// SECURITY: No volumes or bind mounts are configured here by default
	// If volumes are needed in the future, they MUST be sanitized through ComposeSanitizer
	// to ensure they are contained within the user's safe directory structure
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		Resources: container.Resources{
			Memory:    config.Memory,
			CPUShares: config.CPUShares, // Relative priority (for scheduling)
			NanoCPUs:  nanoCPUs,         // Hard CPU limit (prevents exceeding allocated CPUs)
		},
		// SECURITY: Explicitly set network mode to bridge (default) to prevent host network access
		NetworkMode: container.NetworkMode(dm.networkName),
		// SECURITY: Disable privileged mode to prevent container from gaining host access
		Privileged: false,
	}

	// Network configuration
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			dm.networkName: {},
		},
	}

	// Create container
	createResp, err := dm.dockerClient.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:           containerConfig,
		HostConfig:      hostConfig,
		NetworkingConfig: networkConfig,
		Name:            name,
	})
	if err != nil {
		// Check for name conflict error - if container was created by another process
		if strings.Contains(err.Error(), "is already in use") || strings.Contains(err.Error(), "already exists") {
			logger.Info("[DeploymentManager] Container name conflict for %s: %v. Attempting to remove and retry...", name, err)

			// Try to remove the conflicting container
			if removeErr := dm.removeContainerByName(ctx, name); removeErr != nil {
				logger.Info("[DeploymentManager] Failed to remove conflicting container %s: %v", name, removeErr)
				return "", fmt.Errorf("container name %s is in use and could not be removed: %w (original error: %v)", name, removeErr, err)
			}

			// Retry container creation once
			logger.Info("[DeploymentManager] Retrying container creation for %s after removing conflicting container", name)
			createResp, err = dm.dockerClient.ContainerCreate(ctx, client.ContainerCreateOptions{
				Config:           containerConfig,
				HostConfig:      hostConfig,
				NetworkingConfig: networkConfig,
				Name:            name,
			})
			if err != nil {
				return "", fmt.Errorf("failed to create container after removing conflicting container: %w", err)
			}
		} else {
			return "", fmt.Errorf("failed to create container: %w", err)
		}
	}

	return createResp.ID, nil
}

func (dm *DeploymentManager) createSwarmService(ctx context.Context, config *DeploymentConfig, serviceName string, replicaIndex int) (string, string, error) {
	// Get routing rules for this deployment
	routings, _ := database.GetDeploymentRoutings(config.DeploymentID)

	// Prepare labels
	labels := map[string]string{
		"cloud.obiente.managed":       "true",
		"cloud.obiente.deployment_id": config.DeploymentID,
		"cloud.obiente.domain":        config.Domain,
		"cloud.obiente.service_name":  serviceName,
		"cloud.obiente.replica":       strconv.Itoa(replicaIndex),
	}

	// Get the actual Swarm network name (may be prefixed with stack name)
	swarmNetworkName, err := dm.getSwarmNetworkName(ctx)
	if err != nil {
		logger.Warn("[DeploymentManager] Failed to get Swarm network name, using fallback: %v", err)
		swarmNetworkName = "obiente_obiente-network" // Fallback to common name
	}

	// Generate Traefik labels from routing rules
	servicePort := config.Port
	traefikLabels := generateTraefikLabels(config.DeploymentID, serviceName, routings, &servicePort, swarmNetworkName)
	for k, v := range traefikLabels {
		labels[k] = v
	}
	// Only set cloud.obiente.traefik if we actually generated Traefik labels
	if len(traefikLabels) > 0 {
		labels["cloud.obiente.traefik"] = "true" // Required for Traefik discovery
	}

	// Determine health check port - ALWAYS use routing target port if available
	// Priority: 1) Matching service routing, 2) First routing (if routings exist), 3) config.Port (only if no routings)
	healthCheckPort := 0
	if len(routings) > 0 {
		// First, try to find a routing that matches the service name
		for _, routing := range routings {
			serviceMatches := routing.ServiceName == serviceName ||
				(serviceName == "default" && (routing.ServiceName == "" || routing.ServiceName == "default")) ||
				(routing.ServiceName == "default" && serviceName == "")
			
			if serviceMatches && routing.TargetPort > 0 {
				healthCheckPort = routing.TargetPort
				logger.Info("[DeploymentManager] Using routing target port %d for health check (service: %s, routing service: %s)", healthCheckPort, serviceName, routing.ServiceName)
				break
			}
		}
		
		// If no exact match found but routings exist, use the first routing's target port
		// This ensures we always use routing port over config.Port when routings are available
		if healthCheckPort == 0 {
			for _, routing := range routings {
				if routing.TargetPort > 0 {
					healthCheckPort = routing.TargetPort
					logger.Info("[DeploymentManager] Using first available routing target port %d for health check (service: %s, routing service: %s) - no exact service match found", healthCheckPort, serviceName, routing.ServiceName)
					break
				}
			}
		}
	}
	
	// Only fall back to config.Port if NO routings exist at all
	// This prevents using a default port (like 8080) when routing is configured
	if healthCheckPort == 0 {
		if config.Port > 0 {
			healthCheckPort = config.Port
			logger.Warn("[DeploymentManager] No routing found, using config port %d for health check (service: %s) - this may be incorrect if routing should exist", healthCheckPort, serviceName)
		} else {
			logger.Warn("[DeploymentManager] Cannot determine health check port for service %s - no routing target port or config port available", serviceName)
		}
	}

	// Note: For Swarm services, we don't need to determine container port for port bindings
	// as Swarm handles networking internally. The health check port is determined above.

	// Add custom labels
	for k, v := range config.Labels {
		labels[k] = v
	}

	// Prepare environment variables
	env := []string{}
	for k, v := range config.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Ensure netcat is available for health checks
	if healthCheckPort > 0 {
		if _, exists := config.EnvVars["NIXPACKS_APT_PKGS"]; !exists {
			env = append(env, "NIXPACKS_APT_PKGS=netcat-openbsd")
		}
		if _, exists := config.EnvVars["RAILPACK_DEPLOY_APT_PACKAGES"]; !exists {
			env = append(env, "RAILPACK_DEPLOY_APT_PACKAGES=netcat-openbsd")
		}
	}

	// Service name format: deploy-{deploymentID}-{serviceName}
	swarmServiceName := fmt.Sprintf("deploy-%s-%s", config.DeploymentID, serviceName)
	if replicaIndex > 0 {
		swarmServiceName = fmt.Sprintf("deploy-%s-%s-replica-%d", config.DeploymentID, serviceName, replicaIndex)
	}

	// Build docker service create command
	args := []string{"service", "create",
		"--name", swarmServiceName,
		"--network", swarmNetworkName, // Use the dynamically determined Swarm network name
		"--replicas", "1",
		"--with-registry-auth=true", // Enable registry auth for private images
	}

	// Add labels
	for k, v := range labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}

	// Add environment variables
	for _, e := range env {
		args = append(args, "--env", e)
	}

	// Add health check if we have a port
	if healthCheckPort > 0 {
		healthCheckCmd := fmt.Sprintf(`sh -c 'if command -v nc >/dev/null 2>&1; then nc -z localhost %d || exit 1; else (apk add --no-cache netcat-openbsd >/dev/null 2>&1 || apt-get update -qq && apt-get install -y -qq netcat-openbsd >/dev/null 2>&1 || yum install -y -q nc >/dev/null 2>&1) && nc -z localhost %d || exit 1; fi'`, healthCheckPort, healthCheckPort)
		args = append(args,
			"--health-cmd", healthCheckCmd,
			"--health-interval", "30s",
			"--health-timeout", "10s",
			"--health-retries", "3",
			"--health-start-period", "40s",
		)
	}

	// Add resource limits
	// Memory is in bytes (correct for Swarm)
	args = append(args, "--limit-memory", fmt.Sprintf("%d", config.Memory))

	// CPU: Docker Swarm expects CPU cores, not CPU shares
	// Convert CPU shares to cores (1024 shares = 1 core)
	// Use float64 to support fractional cores (e.g., 0.5, 0.25)
	cpuCores := float64(config.CPUShares) / 1024.0
	args = append(args, "--limit-cpu", fmt.Sprintf("%.2f", cpuCores))

	// Add resource reservations (minimal reservations for idle workloads)
	// Reserve 25% of limit for memory (idle sites don't need much)
	reserveMemory := config.Memory / 4 // Reserve 25% of limit
	if reserveMemory < 32*1024*1024 {  // Minimum 32MB for idle sites
		reserveMemory = 32 * 1024 * 1024
	}
	args = append(args, "--reserve-memory", fmt.Sprintf("%d", reserveMemory))

	// Reserve minimal CPU (idle sites use almost no CPU)
	reserveCPU := cpuCores / 4.0 // Reserve 25% of limit
	if reserveCPU < 0.01 {       // Minimum 0.01 cores (10m) for idle workloads
		reserveCPU = 0.01
	}
	args = append(args, "--reserve-cpu", fmt.Sprintf("%.2f", reserveCPU))

	// Add restart policy
	// For Swarm services, valid conditions are: none, on-failure, any
	// Use "any" to always restart (closest to "unless-stopped" behavior)
	args = append(args, "--restart-condition", "any")

	// Add update config with auto-rollback on failure
	// This ensures that if a deployment update fails, Swarm automatically rolls back
	args = append(args,
		"--update-failure-action", "rollback",
		"--update-monitor", "60s",
		"--update-parallelism", "1",
		"--update-delay", "10s",
		"--update-order", "start-first",
	)

	// Add rollback config
	// Controls how rollback is performed when triggered
	args = append(args,
		"--rollback-parallelism", "1",
		"--rollback-delay", "10s",
		"--rollback-order", "start-first",
	)

	// For multi-node Swarm deployments, we need to ensure registry authentication
	// is set up on the manager node so credentials can be passed to worker nodes
	// via --with-registry-auth=true. Worker nodes will pull the image automatically.
	registryURL := os.Getenv("REGISTRY_URL")
	if registryURL == "" {
		domain := os.Getenv("DOMAIN")
		if domain == "" {
			domain = "obiente.cloud"
		}
		registryURL = fmt.Sprintf("https://registry.%s", domain)
	} else {
		// Handle unexpanded docker-compose variables (e.g., "https://registry.${DOMAIN:-obiente.cloud}")
		if strings.Contains(registryURL, "${DOMAIN") {
			domain := os.Getenv("DOMAIN")
			if domain == "" {
				domain = "obiente.cloud"
			}
			registryURL = strings.ReplaceAll(registryURL, "${DOMAIN:-obiente.cloud}", domain)
			registryURL = strings.ReplaceAll(registryURL, "${DOMAIN}", domain)
		}
	}

	// Strip protocol from registry URL for comparison (Docker image names don't include protocols)
	registryHost := strings.TrimPrefix(registryURL, "https://")
	registryHost = strings.TrimPrefix(registryHost, "http://")

	// Check if this image is from our registry or a known registry
	isRegistryImage := strings.HasPrefix(config.Image, registryHost+"/") ||
		strings.HasPrefix(config.Image, "registry.obiente.cloud/") ||
		strings.Contains(config.Image, "/obiente/deploy-") ||
		strings.Contains(config.Image, "ghcr.io/") ||
		strings.Contains(config.Image, "docker.io/")

	// Always authenticate with registry if this is a registry image
	// This ensures credentials are available for --with-registry-auth=true
	if isRegistryImage {
		registryUsername := os.Getenv("REGISTRY_USERNAME")
		registryPassword := os.Getenv("REGISTRY_PASSWORD")
		if registryUsername == "" {
			registryUsername = "obiente"
		}

		// Determine which registry to authenticate with based on image name
		// docker login needs just the hostname, not the full URL with protocol
		imageRegistryHost := registryHost
		if strings.Contains(config.Image, "ghcr.io/") {
			imageRegistryHost = "ghcr.io"
		} else if strings.Contains(config.Image, "docker.io/") || (!strings.Contains(config.Image, "/") && !strings.Contains(config.Image, ":")) {
			// Docker Hub images (docker.io is implicit)
			imageRegistryHost = "docker.io"
		}

		if registryPassword != "" {
			logger.Info("[DeploymentManager] Authenticating with registry %s to enable multi-node image pulls...", imageRegistryHost)
			loginCmd := exec.CommandContext(ctx, "docker", "login", imageRegistryHost, "-u", registryUsername, "-p", registryPassword)
			var loginStderr bytes.Buffer
			loginCmd.Stderr = &loginStderr
			if loginErr := loginCmd.Run(); loginErr != nil {
				logger.Warn("[DeploymentManager] Failed to authenticate with registry %s: %v (stderr: %s). Worker nodes may fail to pull image.", imageRegistryHost, loginErr, loginStderr.String())
				// Don't fail here - the service creation might still work if credentials are cached
			} else {
				logger.Info("[DeploymentManager] Successfully authenticated with registry %s. Credentials will be passed to worker nodes.", imageRegistryHost)
			}
		} else {
			logger.Warn("[DeploymentManager] REGISTRY_PASSWORD not set - worker nodes may fail to pull private images")
		}
	}

	// Note: We don't pull the image locally here because:
	// 1. In multi-node Swarm, worker nodes need to pull the image themselves
	// 2. Docker Swarm will automatically pull the image on worker nodes when --with-registry-auth=true is set
	// 3. The manager node doesn't need the image locally unless it's also running tasks
	logger.Debug("[DeploymentManager] Service will use image %s. Swarm will pull on worker nodes automatically.", config.Image)

	// Add image
	args = append(args, config.Image)

	// Add start command if provided (must come after image)
	// docker service create format: [OPTIONS] IMAGE [COMMAND] [ARG...]
	if config.StartCommand != nil && *config.StartCommand != "" {
		// Split the command into parts for proper argument handling
		// Use sh -c to preserve working directory and handle relative paths
		args = append(args, "sh", "-c", *config.StartCommand)
	}

	// Execute docker service create
	// Use a longer timeout context for Docker operations to prevent cancellation
	// Docker service create can take time, especially when pulling images on worker nodes
	dockerCtx, dockerCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer dockerCancel()
	
	cmd := exec.CommandContext(dockerCtx, "docker", args...)
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		errorOutput := stderr.String()
		stdOutput := stdout.String()
		// Check if the error is due to context cancellation
		if dockerCtx.Err() == context.DeadlineExceeded {
			logger.Error("[DeploymentManager] Docker service create timed out after 5 minutes for %s", swarmServiceName)
			return "", "", fmt.Errorf("failed to create Swarm service: operation timed out after 5 minutes. This may indicate slow image pulls or Swarm cluster issues")
		} else if dockerCtx.Err() == context.Canceled {
			logger.Error("[DeploymentManager] Docker service create was canceled for %s", swarmServiceName)
			return "", "", fmt.Errorf("failed to create Swarm service: operation was canceled")
		}
		logger.Error("[DeploymentManager] Failed to create Swarm service %s: %v\nStderr: %s\nStdout: %s", swarmServiceName, err, errorOutput, stdOutput)
		return "", "", fmt.Errorf("failed to create Swarm service: %w\nStderr: %s\nStdout: %s", err, errorOutput, stdOutput)
	}

	serviceID := strings.TrimSpace(stdout.String())
	logger.Info("[DeploymentManager] Created Swarm service %s (ID: %s)", swarmServiceName, serviceID)
	logger.Info("[DeploymentManager] Service image: %s, start command: %v", config.Image, config.StartCommand)

	// Verify the service exists immediately (before rollback can remove it)
	// Use the same extended timeout context for verification
	verifyArgs := []string{"service", "inspect", swarmServiceName, "--format", "{{.ID}}"}
	verifyCmd := exec.CommandContext(dockerCtx, "docker", verifyArgs...)
	var verifyStderr bytes.Buffer
	verifyCmd.Stderr = &verifyStderr
	if err := verifyCmd.Run(); err != nil {
		errorMsg := verifyStderr.String()
		logger.Error("[DeploymentManager] Service %s was created (ID: %s) but cannot be found immediately - may have been rolled back", swarmServiceName, serviceID)
		logger.Error("[DeploymentManager] Error details: %s", errorMsg)
		logger.Error("[DeploymentManager] This usually means the task failed immediately and rollback removed the service")
		logger.Error("[DeploymentManager] Check: Start command '%v', Health check port: %d", config.StartCommand, healthCheckPort)
		return "", "", fmt.Errorf("service %s was created but immediately removed (likely rolled back due to immediate task failure). Start command: %v, Health check port: %d. Error: %s", swarmServiceName, config.StartCommand, healthCheckPort, errorMsg)
	}

	// Wait a moment for the service to create a task
	time.Sleep(2 * time.Second)

	// Immediately try to get logs to see what's happening
	logsArgs := []string{"service", "logs", "--tail", "50", "--raw", swarmServiceName}
	logsCmd := exec.CommandContext(ctx, "docker", logsArgs...)
	var initialLogsStdout bytes.Buffer
	var initialLogsStderr bytes.Buffer
	logsCmd.Stdout = &initialLogsStdout
	logsCmd.Stderr = &initialLogsStderr
	if err := logsCmd.Run(); err == nil {
		initialLogs := strings.TrimSpace(initialLogsStdout.String())
		if initialLogs != "" {
			logger.Info("[DeploymentManager] Initial service logs for %s:\n%s", swarmServiceName, initialLogs)
		}
	}

	// Check task status and log any errors - do this multiple times to catch failures
	maxChecks := 5
	checkInterval := 2 * time.Second
	var lastTaskStatus string
	for i := 0; i < maxChecks; i++ {
		taskStatusArgs := []string{"service", "ps", swarmServiceName, "--format", "{{.ID}}\t{{.Name}}\t{{.CurrentState}}\t{{.Error}}", "--no-trunc"}
		taskStatusCmd := exec.CommandContext(ctx, "docker", taskStatusArgs...)
		var taskStatusStdout bytes.Buffer
		var taskStatusStderr bytes.Buffer
		taskStatusCmd.Stdout = &taskStatusStdout
		taskStatusCmd.Stderr = &taskStatusStderr
		if err := taskStatusCmd.Run(); err == nil {
			taskStatus := strings.TrimSpace(taskStatusStdout.String())
			if taskStatus != "" {
				lastTaskStatus = taskStatus
				logger.Info("[DeploymentManager] Service %s task status (check %d/%d):\n%s", swarmServiceName, i+1, maxChecks, taskStatus)

				// Check if there are any errors
				if strings.Contains(taskStatus, "Error") || strings.Contains(taskStatus, "Failed") || strings.Contains(taskStatus, "Rejected") {
					logger.Error("[DeploymentManager] Service %s has task errors detected!", swarmServiceName)
					// Extract and log the error message
					lines := strings.Split(taskStatus, "\n")
					for _, line := range lines {
						if strings.Contains(line, "Error") || strings.Contains(line, "Failed") || strings.Contains(line, "Rejected") {
							logger.Error("[DeploymentManager] Task error: %s", line)
						}
					}
					// Try to get detailed task error - get the most recent task (first line)
					taskIDArgs := []string{"service", "ps", swarmServiceName, "--format", "{{.ID}}", "--no-trunc"}
					taskIDCmd := exec.CommandContext(ctx, "docker", taskIDArgs...)
					var taskIDStdout bytes.Buffer
					taskIDCmd.Stdout = &taskIDStdout
					if taskIDCmd.Run() == nil {
						taskIDLines := strings.Split(strings.TrimSpace(taskIDStdout.String()), "\n")
						taskID := ""
						if len(taskIDLines) > 0 {
							taskID = strings.TrimSpace(taskIDLines[0])
						}
						if taskID != "" {
							// Get detailed error from task
							taskErrArgs := []string{"inspect", taskID, "--format", "{{.Status.Err}}"}
							taskErrCmd := exec.CommandContext(ctx, "docker", taskErrArgs...)
							var taskErrStdout bytes.Buffer
							if taskErrCmd.Run() == nil {
								taskErr := strings.TrimSpace(taskErrStdout.String())
								if taskErr != "" && taskErr != "<no value>" {
									logger.Error("[DeploymentManager] Detailed task error: %s", taskErr)
								}
							}
							// Get container exit code if available
							exitCodeArgs := []string{"inspect", taskID, "--format", "{{.Status.ContainerStatus.ExitCode}}"}
							exitCodeCmd := exec.CommandContext(ctx, "docker", exitCodeArgs...)
							var exitCodeStdout bytes.Buffer
							exitCodeCmd.Stdout = &exitCodeStdout
							if exitCodeCmd.Run() == nil {
								exitCode := strings.TrimSpace(exitCodeStdout.String())
								if exitCode != "" && exitCode != "<no value>" {
									if exitCode == "0" {
										logger.Error("[DeploymentManager] Container exited with code 0 (success) - command completed instead of running as a server. Start command may be incorrect: %v", config.StartCommand)
									} else {
										logger.Error("[DeploymentManager] Container exit code: %s", exitCode)
									}
								}
							}
						}
					}
					// Try to get service logs - get more lines and also try to get from all tasks
					logsArgs := []string{"service", "logs", "--tail", "200", "--raw", swarmServiceName}
					logsCmd := exec.CommandContext(ctx, "docker", logsArgs...)
					var logsStdout bytes.Buffer
					var logsStderr bytes.Buffer
					logsCmd.Stdout = &logsStdout
					logsCmd.Stderr = &logsStderr
					if err := logsCmd.Run(); err == nil {
						logs := strings.TrimSpace(logsStdout.String())
						if logs != "" {
							logger.Error("[DeploymentManager] Service %s logs (last 200 lines):\n%s", swarmServiceName, logs)
						} else {
							logger.Warn("[DeploymentManager] Service %s has no logs yet - container may have exited before producing output", swarmServiceName)
						}
					} else {
						logger.Warn("[DeploymentManager] Failed to get service logs: %v (stderr: %s)", err, logsStderr.String())
					}

					// Also try to get logs from the specific task/container if we can find it
					// Get task ID again for container log retrieval
					taskIDForLogsArgs := []string{"service", "ps", swarmServiceName, "--format", "{{.ID}}", "--no-trunc"}
					taskIDForLogsCmd := exec.CommandContext(ctx, "docker", taskIDForLogsArgs...)
					var taskIDForLogsStdout bytes.Buffer
					if taskIDForLogsCmd.Run() == nil {
						taskIDForLogsLines := strings.Split(strings.TrimSpace(taskIDForLogsStdout.String()), "\n")
						if len(taskIDForLogsLines) > 0 {
							taskIDForLogs := strings.TrimSpace(taskIDForLogsLines[0])
							if taskIDForLogs != "" {
								// Try to get container ID from task
								containerIDArgs := []string{"inspect", taskIDForLogs, "--format", "{{.Status.ContainerStatus.ContainerID}}"}
								containerIDCmd := exec.CommandContext(ctx, "docker", containerIDArgs...)
								var containerIDStdout bytes.Buffer
								containerIDCmd.Stdout = &containerIDStdout
								if containerIDCmd.Run() == nil {
									containerID := strings.TrimSpace(containerIDStdout.String())
									if containerID != "" && containerID != "<no value>" {
										// Try to get container logs directly
										containerLogsArgs := []string{"logs", "--tail", "200", containerID}
										containerLogsCmd := exec.CommandContext(ctx, "docker", containerLogsArgs...)
										var containerLogsStdout bytes.Buffer
										var containerLogsStderr bytes.Buffer
										containerLogsCmd.Stdout = &containerLogsStdout
										containerLogsCmd.Stderr = &containerLogsStderr
										if err := containerLogsCmd.Run(); err == nil {
											containerLogs := strings.TrimSpace(containerLogsStdout.String())
											if containerLogs != "" {
												logger.Error("[DeploymentManager] Container %s logs (last 200 lines):\n%s", containerID[:12], containerLogs)
											}
											containerErrLogs := strings.TrimSpace(containerLogsStderr.String())
											if containerErrLogs != "" {
												logger.Error("[DeploymentManager] Container %s stderr (last 200 lines):\n%s", containerID[:12], containerErrLogs)
											}
										}
									}
								}
							}
						}
					}
				} else if strings.Contains(taskStatus, "Running") || strings.Contains(taskStatus, "Starting") {
					// Task is running or starting, that's good - we can break early
					status := "starting"
					if strings.Contains(taskStatus, "Running") {
						status = "running"
					}
					logger.Info("[DeploymentManager] Service %s task is %s - deployment appears successful", swarmServiceName, status)
					break
				}
			} else {
				if i == 0 {
					logger.Warn("[DeploymentManager] Service %s has no tasks yet - waiting...", swarmServiceName)
				}
			}
		} else {
			logger.Warn("[DeploymentManager] Failed to check task status for service %s: %v (stderr: %s)", swarmServiceName, err, taskStatusStderr.String())
		}

		// Get logs on each check to see what's happening in real-time
		if i > 0 { // Skip first check since we already got initial logs
			logsArgs := []string{"service", "logs", "--tail", "20", "--raw", swarmServiceName}
			logsCmd := exec.CommandContext(ctx, "docker", logsArgs...)
			var checkLogsStdout bytes.Buffer
			logsCmd.Stdout = &checkLogsStdout
			if logsCmd.Run() == nil {
				checkLogs := strings.TrimSpace(checkLogsStdout.String())
				if checkLogs != "" {
					logger.Info("[DeploymentManager] Service %s logs (check %d/%d, last 20 lines):\n%s", swarmServiceName, i+1, maxChecks, checkLogs)
				}
			}
		}

		// Wait before next check (except on last iteration)
		if i < maxChecks-1 {
			time.Sleep(checkInterval)
		}
	}

	// Final status summary
	if lastTaskStatus != "" {
		logger.Info("[DeploymentManager] Final task status for service %s:\n%s", swarmServiceName, lastTaskStatus)
	}

	// Get container ID from service tasks
	// In Swarm, we need to inspect the service and get the task's container ID
	inspectArgs := []string{"service", "inspect", swarmServiceName, "--format", "{{.ID}}"}
	inspectCmd := exec.CommandContext(ctx, "docker", inspectArgs...)
	var inspectStdout bytes.Buffer
	inspectCmd.Stdout = &inspectStdout
	if err := inspectCmd.Run(); err == nil {
		serviceID = strings.TrimSpace(inspectStdout.String())
	}

	// Get tasks for this service to find the container ID
	taskArgs := []string{"service", "ps", swarmServiceName, "--format", "{{.ID}}", "--no-trunc"}
	taskCmd := exec.CommandContext(ctx, "docker", taskArgs...)
	var taskStdout bytes.Buffer
	taskCmd.Stdout = &taskStdout
	var containerID string
	if err := taskCmd.Run(); err == nil {
		taskIDs := strings.TrimSpace(taskStdout.String())
		if taskIDs != "" {
			taskIDList := strings.Split(taskIDs, "\n")
			if len(taskIDList) > 0 {
				// Get container ID from the first task
				taskID := strings.TrimSpace(taskIDList[0])
				if taskID != "" {
					// Inspect the task to get container ID and check for errors
					taskInspectArgs := []string{"inspect", taskID, "--format", "{{.Status.ContainerStatus.ContainerID}}"}
					taskInspectCmd := exec.CommandContext(ctx, "docker", taskInspectArgs...)
					var taskInspectStdout bytes.Buffer
					taskInspectCmd.Stdout = &taskInspectStdout
					if err := taskInspectCmd.Run(); err == nil {
						containerID = strings.TrimSpace(taskInspectStdout.String())
					}

					// Also check task error message and state
					taskErrorArgs := []string{"inspect", taskID, "--format", "{{.Status.Err}}"}
					taskErrorCmd := exec.CommandContext(ctx, "docker", taskErrorArgs...)
					var taskErrorStdout bytes.Buffer
					taskErrorCmd.Stdout = &taskErrorStdout
					if err := taskErrorCmd.Run(); err == nil {
						taskError := strings.TrimSpace(taskErrorStdout.String())
						if taskError != "" && taskError != "<no value>" {
							logger.Error("[DeploymentManager] Task %s for service %s has error: %s", taskID[:12], swarmServiceName, taskError)
						}
					}

					// Check task state and exit code
					taskStateArgs := []string{"inspect", taskID, "--format", "{{.Status.State}}\t{{.Status.ContainerStatus.ExitCode}}"}
					taskStateCmd := exec.CommandContext(ctx, "docker", taskStateArgs...)
					var taskStateStdout bytes.Buffer
					taskStateCmd.Stdout = &taskStateStdout
					if err := taskStateCmd.Run(); err == nil {
						taskState := strings.TrimSpace(taskStateStdout.String())
						if taskState != "" {
							logger.Info("[DeploymentManager] Task %s state: %s", taskID[:12], taskState)
							// If task exited, try to get service logs (Swarm services log differently)
							if strings.Contains(taskState, "complete") || strings.Contains(taskState, "shutdown") {
								// Try to get service logs (Swarm aggregates logs from all tasks)
								logsArgs := []string{"service", "logs", "--tail", "50", "--raw", swarmServiceName}
								logsCmd := exec.CommandContext(ctx, "docker", logsArgs...)
								var logsStdout bytes.Buffer
								var logsStderr bytes.Buffer
								logsCmd.Stdout = &logsStdout
								logsCmd.Stderr = &logsStderr
								if err := logsCmd.Run(); err == nil {
									logs := strings.TrimSpace(logsStdout.String())
									if logs != "" {
										logger.Error("[DeploymentManager] Service %s logs (last 50 lines):\n%s", swarmServiceName, logs)
									}
								} else {
									// Fallback: try container logs if we have container ID
									if containerID != "" {
										containerLogsArgs := []string{"logs", "--tail", "50", containerID}
										containerLogsCmd := exec.CommandContext(ctx, "docker", containerLogsArgs...)
										var containerLogsStdout bytes.Buffer
										var containerLogsStderr bytes.Buffer
										containerLogsCmd.Stdout = &containerLogsStdout
										containerLogsCmd.Stderr = &containerLogsStderr
										if err := containerLogsCmd.Run(); err == nil {
											logs := strings.TrimSpace(containerLogsStdout.String())
											if logs != "" {
												logger.Error("[DeploymentManager] Container %s logs (last 50 lines):\n%s", containerID[:12], logs)
											}
											errLogs := strings.TrimSpace(containerLogsStderr.String())
											if errLogs != "" {
												logger.Error("[DeploymentManager] Container %s stderr (last 50 lines):\n%s", containerID[:12], errLogs)
											}
										}
									}
								}
							}
						}
					}
				}
			}
		} else {
			logger.Warn("[DeploymentManager] Service %s has no tasks - service may be failing to start. Check: docker service ps %s", swarmServiceName, swarmServiceName)
		}
	}

	// If we couldn't get container ID from task, we'll use the service ID as a placeholder
	// The container will be created by Swarm and we can look it up later
	if containerID == "" {
		logger.Warn("[DeploymentManager] Could not get container ID from Swarm service %s tasks - will look up later", swarmServiceName)
		// Try to find container by service label
		filterArgs := make(client.Filters)
		filterArgs.Add("label", fmt.Sprintf("com.docker.swarm.service.name=%s", swarmServiceName))
		containersResult, _ := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
			All:     true,
			Filters: filterArgs,
		})
		if len(containersResult.Items) > 0 {
			containerID = containersResult.Items[0].ID
		}
	}

	return serviceID, containerID, nil
}

// updateSwarmService updates an existing Swarm service with new configuration
// This enables zero-downtime deployments by using docker service update with start-first strategy
func (dm *DeploymentManager) updateSwarmService(ctx context.Context, config *DeploymentConfig, serviceName string, replicaIndex int, swarmServiceName string) (string, string, error) {
	// Get routing rules for this deployment
	routings, _ := database.GetDeploymentRoutings(config.DeploymentID)

	// Prepare labels
	labels := map[string]string{
		"cloud.obiente.managed":       "true",
		"cloud.obiente.deployment_id": config.DeploymentID,
		"cloud.obiente.domain":        config.Domain,
		"cloud.obiente.service_name":  serviceName,
		"cloud.obiente.replica":       strconv.Itoa(replicaIndex),
	}

	// Get the actual Swarm network name (may be prefixed with stack name)
	swarmNetworkName, err := dm.getSwarmNetworkName(ctx)
	if err != nil {
		logger.Warn("[DeploymentManager] Failed to get Swarm network name, using fallback: %v", err)
		swarmNetworkName = "obiente_obiente-network" // Fallback to common name
	}

	// Generate Traefik labels from routing rules
	servicePort := config.Port
	traefikLabels := generateTraefikLabels(config.DeploymentID, serviceName, routings, &servicePort, swarmNetworkName)
	for k, v := range traefikLabels {
		labels[k] = v
	}
	// Only set cloud.obiente.traefik if we actually generated Traefik labels
	if len(traefikLabels) > 0 {
		labels["cloud.obiente.traefik"] = "true" // Required for Traefik discovery
	}

	// Determine health check port - same logic as createSwarmService
	healthCheckPort := 0
	if len(routings) > 0 {
		for _, routing := range routings {
			serviceMatches := routing.ServiceName == serviceName ||
				(serviceName == "default" && (routing.ServiceName == "" || routing.ServiceName == "default")) ||
				(routing.ServiceName == "default" && serviceName == "")
			
			if serviceMatches && routing.TargetPort > 0 {
				healthCheckPort = routing.TargetPort
				break
			}
		}
		if healthCheckPort == 0 {
			for _, routing := range routings {
				if routing.TargetPort > 0 {
					healthCheckPort = routing.TargetPort
					break
				}
			}
		}
	}
	if healthCheckPort == 0 && config.Port > 0 {
		healthCheckPort = config.Port
	}

	// Add custom labels
	for k, v := range config.Labels {
		labels[k] = v
	}

	// Prepare environment variables
	env := []string{}
	for k, v := range config.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Ensure netcat is available for health checks
	if healthCheckPort > 0 {
		if _, exists := config.EnvVars["NIXPACKS_APT_PKGS"]; !exists {
			env = append(env, "NIXPACKS_APT_PKGS=netcat-openbsd")
		}
		if _, exists := config.EnvVars["RAILPACK_DEPLOY_APT_PACKAGES"]; !exists {
			env = append(env, "RAILPACK_DEPLOY_APT_PACKAGES=netcat-openbsd")
		}
	}

	// Build docker service update command
	args := []string{"service", "update",
		"--with-registry-auth=true", // Enable registry auth for private images
	}

	// Update labels - Docker service update will merge labels
	// We add/update labels, and they'll replace any existing ones with the same key
	for k, v := range labels {
		args = append(args, "--label-add", fmt.Sprintf("%s=%s", k, v))
	}

	// Update environment variables
	// Remove all existing env vars first (we'll add them back)
	// Note: docker service update doesn't have a direct way to remove all env vars,
	// so we'll add the new ones and they'll replace the old ones
	for _, e := range env {
		args = append(args, "--env-add", e)
	}

	// Update health check if we have a port
	if healthCheckPort > 0 {
		healthCheckCmd := fmt.Sprintf(`sh -c 'if command -v nc >/dev/null 2>&1; then nc -z localhost %d || exit 1; else (apk add --no-cache netcat-openbsd >/dev/null 2>&1 || apt-get update -qq && apt-get install -y -qq netcat-openbsd >/dev/null 2>&1 || yum install -y -q nc >/dev/null 2>&1) && nc -z localhost %d || exit 1; fi'`, healthCheckPort, healthCheckPort)
		args = append(args,
			"--health-cmd", healthCheckCmd,
			"--health-interval", "30s",
			"--health-timeout", "10s",
			"--health-retries", "3",
			"--health-start-period", "40s",
		)
	}

	// Update resource limits
	args = append(args, "--limit-memory", fmt.Sprintf("%d", config.Memory))

	// CPU: Docker Swarm expects CPU cores, not CPU shares
	cpuCores := float64(config.CPUShares) / 1024.0
	args = append(args, "--limit-cpu", fmt.Sprintf("%.2f", cpuCores))

	// Update resource reservations
	reserveMemory := config.Memory / 4
	if reserveMemory < 32*1024*1024 {
		reserveMemory = 32 * 1024 * 1024
	}
	args = append(args, "--reserve-memory", fmt.Sprintf("%d", reserveMemory))

	reserveCPU := cpuCores / 4.0
	if reserveCPU < 0.01 {
		reserveCPU = 0.01
	}
	args = append(args, "--reserve-cpu", fmt.Sprintf("%.2f", reserveCPU))

	// Update restart policy
	args = append(args, "--restart-condition", "any")

	// Update config with start-first strategy (ensures zero-downtime)
	args = append(args,
		"--update-failure-action", "rollback",
		"--update-monitor", "60s",
		"--update-parallelism", "1",
		"--update-delay", "10s",
		"--update-order", "start-first",
	)

	// Update rollback config
	args = append(args,
		"--rollback-parallelism", "1",
		"--rollback-delay", "10s",
		"--rollback-order", "start-first",
	)

	// Update image
	args = append(args, "--image", config.Image)

	// Note: Docker service update doesn't support changing the command directly.
	// If the start command needs to change, the service would need to be recreated.
	// For zero-downtime deployments, image and config updates are the primary concern.

	// Add service name
	args = append(args, swarmServiceName)

	// Execute docker service update
	// Use a longer timeout context for Docker operations
	dockerCtx, dockerCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer dockerCancel()
	
	cmd := exec.CommandContext(dockerCtx, "docker", args...)
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	logger.Info("[DeploymentManager] Updating Swarm service %s with zero-downtime strategy (start-first)", swarmServiceName)
	if err := cmd.Run(); err != nil {
		errorOutput := stderr.String()
		stdOutput := stdout.String()
		if dockerCtx.Err() == context.DeadlineExceeded {
			logger.Error("[DeploymentManager] Docker service update timed out after 5 minutes for %s", swarmServiceName)
			return "", "", fmt.Errorf("failed to update Swarm service: operation timed out after 5 minutes")
		} else if dockerCtx.Err() == context.Canceled {
			logger.Error("[DeploymentManager] Docker service update was canceled for %s", swarmServiceName)
			return "", "", fmt.Errorf("failed to update Swarm service: operation was canceled")
		}
		logger.Error("[DeploymentManager] Failed to update Swarm service %s: %v\nStderr: %s\nStdout: %s", swarmServiceName, err, errorOutput, stdOutput)
		return "", "", fmt.Errorf("failed to update Swarm service: %w\nStderr: %s\nStdout: %s", err, errorOutput, stdOutput)
	}

	serviceID := strings.TrimSpace(stdout.String())
	logger.Info("[DeploymentManager] Updated Swarm service %s (ID: %s) - new tasks will start before old ones stop", swarmServiceName, serviceID)

	// Wait a moment for the update to propagate
	time.Sleep(2 * time.Second)

	// Get container ID from service tasks (will be the new task once it starts)
	taskArgs := []string{"service", "ps", swarmServiceName, "--format", "{{.ID}}", "--no-trunc", "--filter", "desired-state=running"}
	taskCmd := exec.CommandContext(ctx, "docker", taskArgs...)
	var taskStdout bytes.Buffer
	taskCmd.Stdout = &taskStdout
	var containerID string
	if err := taskCmd.Run(); err == nil {
		taskIDs := strings.TrimSpace(taskStdout.String())
		if taskIDs != "" {
			taskIDList := strings.Split(taskIDs, "\n")
			// Get the first running task (should be the new one with start-first)
			if len(taskIDList) > 0 {
				taskID := strings.TrimSpace(taskIDList[0])
				if taskID != "" {
					taskInspectArgs := []string{"inspect", taskID, "--format", "{{.Status.ContainerStatus.ContainerID}}"}
					taskInspectCmd := exec.CommandContext(ctx, "docker", taskInspectArgs...)
					var taskInspectStdout bytes.Buffer
					if err := taskInspectCmd.Run(); err == nil {
						containerID = strings.TrimSpace(taskInspectStdout.String())
					}
				}
			}
		}
	}

	// If we couldn't get container ID from task, try to find by service label
	if containerID == "" {
		filterArgs := make(client.Filters)
		filterArgs.Add("label", fmt.Sprintf("com.docker.swarm.service.name=%s", swarmServiceName))
		containersResult, _ := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
			All:     true,
			Filters: filterArgs,
		})
		if len(containersResult.Items) > 0 {
			// Get the most recently created container (should be the new one)
			containerID = containersResult.Items[0].ID
		}
	}

	return serviceID, containerID, nil
}

// removeContainerByName removes a container by name (used for cleanup before creating new containers)
func (dm *DeploymentManager) removeContainerByName(ctx context.Context, containerName string) error {
	// Try to find container by name
	containersResult, err := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: func() client.Filters { f := make(client.Filters); f.Add("name", containerName); return f }(),
	})
	containers := containersResult.Items
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Remove all containers with this name (should only be one, but handle multiple)
	for _, container := range containers {
		// Check if any of the container names match
		for _, name := range container.Names {
			if strings.TrimPrefix(name, "/") == containerName {
				// Stop container first if running
				if container.State == "running" {
					if err := dm.dockerHelper.StopContainer(ctx, container.ID, 10*time.Second); err != nil {
						logger.Warn("[DeploymentManager] Failed to stop container %s: %v", container.ID[:12], err)
					}
				}

				// Remove container
				if err := dm.dockerHelper.RemoveContainer(ctx, container.ID, true); err != nil {
					return fmt.Errorf("failed to remove container %s: %w", container.ID[:12], err)
				}

				logger.Info("[DeploymentManager] Removed existing container %s (%s)", containerName, container.ID[:12])
				return nil
			}
		}
	}

	// Container not found - that's OK, just return
	return nil
}
