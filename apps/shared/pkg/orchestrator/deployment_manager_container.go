package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	containerpath "path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/platform"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

type swarmConvergedTask struct {
	ServiceID   string
	TaskID      string
	ContainerID string
}

// Container operations for deployments

const (
	swarmMemoryReservationDivisor = int64(20)                // 5% of the memory limit
	swarmMinMemoryReservation     = int64(32 * 1024 * 1024)  // 32MiB
	swarmMaxMemoryReservation     = int64(128 * 1024 * 1024) // 128MiB

	swarmCPUReservationFraction = 0.025 // 2.5% of the CPU limit
	swarmMinCPUReservation      = 0.01
	swarmMaxCPUReservation      = 0.10
)

func sanitizedVolumeMounts(deploymentID string, volumes []DeploymentVolume) ([]string, []string) {
	binds := make([]string, 0, len(volumes))
	mountFlags := make([]string, 0, len(volumes))
	for _, volume := range volumes {
		name := sanitizeVolumeName(volume.Name)
		mountPath := sanitizeContainerMountPath(volume.MountPath)
		if name == "" || mountPath == "" {
			continue
		}

		hostPath := filepath.Join("/var/lib/obiente/volumes", deploymentID, name)
		if err := os.MkdirAll(hostPath, 0o755); err != nil {
			logger.Warn("[DeploymentManager] Failed to create volume directory %s: %v", hostPath, err)
			continue
		}

		bind := fmt.Sprintf("%s:%s", hostPath, mountPath)
		mountFlag := fmt.Sprintf("type=bind,src=%s,dst=%s", hostPath, mountPath)
		if volume.ReadOnly {
			bind += ":ro"
			mountFlag += ",readonly"
		}
		binds = append(binds, bind)
		mountFlags = append(mountFlags, mountFlag)
	}
	return binds, mountFlags
}

func sanitizeVolumeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 64 || name == "." || name == ".." {
		return ""
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return ""
	}
	return name
}

func sanitizeContainerMountPath(mountPath string) string {
	mountPath = strings.TrimSpace(mountPath)
	if mountPath == "" || !strings.HasPrefix(mountPath, "/") || strings.Contains(mountPath, "\x00") || strings.Contains(mountPath, ":") {
		return ""
	}
	cleaned := containerpath.Clean(mountPath)
	if cleaned == "/" || cleaned == "." ||
		cleaned == "/proc" || strings.HasPrefix(cleaned, "/proc/") ||
		cleaned == "/sys" || strings.HasPrefix(cleaned, "/sys/") ||
		cleaned == "/dev" || strings.HasPrefix(cleaned, "/dev/") ||
		cleaned == "/var/run/docker.sock" || cleaned == "/run/docker.sock" {
		return ""
	}
	return cleaned
}

func existingSwarmServiceMountTargets(ctx context.Context, serviceName string) []string {
	cmd := exec.CommandContext(ctx, "docker", "service", "inspect", "--format", "{{json .Spec.TaskTemplate.ContainerSpec.Mounts}}", serviceName)
	output, err := cmd.Output()
	if err != nil {
		logger.Debug("[DeploymentManager] Failed to inspect mounts for service %s: %v", serviceName, err)
		return nil
	}
	var mounts []struct {
		Target string `json:"Target"`
		Source string `json:"Source"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(output), &mounts); err != nil {
		logger.Debug("[DeploymentManager] Failed to parse mounts for service %s: %v", serviceName, err)
		return nil
	}
	obienteVolumePrefix := filepath.Join("/var/lib/obiente/volumes")
	targets := make([]string, 0, len(mounts))
	for _, mount := range mounts {
		if mount.Target != "" && strings.HasPrefix(mount.Source, obienteVolumePrefix) {
			targets = append(targets, mount.Target)
		}
	}
	return targets
}

func swarmMemoryReservation(limitBytes int64) int64 {
	if limitBytes <= 0 {
		return 0
	}

	reservation := limitBytes / swarmMemoryReservationDivisor
	if reservation < swarmMinMemoryReservation {
		return swarmMinMemoryReservation
	}
	if reservation > swarmMaxMemoryReservation {
		return swarmMaxMemoryReservation
	}
	return reservation
}

func swarmCPUReservation(limitCores float64) float64 {
	if limitCores <= 0 {
		return 0
	}

	reservation := limitCores * swarmCPUReservationFraction
	if reservation < swarmMinCPUReservation {
		return swarmMinCPUReservation
	}
	if reservation > swarmMaxCPUReservation {
		return swarmMaxCPUReservation
	}
	return reservation
}

func swarmDisableHealthcheckArgs() []string {
	// Docker service health checks are disabled with a flag, not with
	// "--health-cmd NONE". The latter becomes CMD-SHELL "NONE" and fails at
	// runtime with "/bin/sh: NONE: not found".
	return []string{"--no-healthcheck"}
}

func normalizeStartCommand(raw string) string {
	cmd := strings.TrimSpace(raw)
	for range 4 {
		inner, ok := unwrapShellC(cmd)
		if !ok {
			break
		}
		cmd = inner
	}
	return cmd
}

func unwrapShellC(cmd string) (string, bool) {
	trimmed := strings.TrimSpace(cmd)
	prefixes := []string{"sh -c ", "bash -c ", "/bin/sh -c ", "/bin/bash -c "}
	for _, prefix := range prefixes {
		if strings.HasPrefix(trimmed, prefix) {
			inner := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
			if len(inner) >= 2 {
				if (inner[0] == '\'' && inner[len(inner)-1] == '\'') || (inner[0] == '"' && inner[len(inner)-1] == '"') {
					inner = strings.TrimSpace(inner[1 : len(inner)-1])
				}
			}
			return inner, true
		}
	}
	return "", false
}

func hasShellMetacharacters(cmd string) bool {
	return strings.ContainsAny(cmd, "|&;<>()$`\n")
}

func buildStartCommandParts(raw string) (entrypoint []string, args []string) {
	startCommand := normalizeStartCommand(raw)
	if startCommand == "" {
		return nil, nil
	}

	if !hasShellMetacharacters(startCommand) {
		fields := strings.Fields(startCommand)
		if len(fields) > 0 {
			return nil, fields
		}
	}

	return []string{"sh"}, []string{"-c", startCommand}
}

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
		logger.Info("[DeploymentManager] Healthcheck type explicitly set to %d for container %s", healthcheckType, name)
	} else {
		logger.Info("[DeploymentManager] Healthcheck type not set (nil), defaulting to %d (UNSPECIFIED) for container %s", healthcheckType, name)
	}

	switch healthcheckType {
	case 1: // HEALTHCHECK_DISABLED
		// Explicitly disabled - disable healthcheck even if image has one
		logger.Info("[DeploymentManager] Health check explicitly disabled for container %s", name)
		healthcheck = &container.HealthConfig{
			Test: []string{"NONE"},
		}

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
		_, startArgs := buildStartCommandParts(*config.StartCommand)
		containerConfig.Cmd = append([]string{}, startArgs...)
	}

	// Convert CPU shares to NanoCPUs for hard CPU limit
	// CPUShares is in units where 1024 = 1 CPU core
	// NanoCPUs: 1 CPU = 1,000,000,000 nanoseconds (1e9)
	// This sets an absolute CPU limit, not just relative priority
	cpuCores := float64(config.CPUShares) / 1024.0
	nanoCPUs := int64(cpuCores * 1e9)

	// Host configuration
	binds, _ := sanitizedVolumeMounts(config.DeploymentID, config.Volumes)
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Binds:        binds,
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
		HostConfig:       hostConfig,
		NetworkingConfig: networkConfig,
		Name:             name,
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
				HostConfig:       hostConfig,
				NetworkingConfig: networkConfig,
				Name:             name,
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

func persistDeploymentServiceLogSnapshot(ctx context.Context, deploymentID, serviceName, nodeID, output string) {
	if database.MetricsDB == nil || strings.TrimSpace(output) == "" {
		return
	}

	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	entries := make([]database.DeploymentRuntimeLog, 0, len(lines))
	now := time.Now()
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entries = append(entries, database.DeploymentRuntimeLog{
			DeploymentID: deploymentID,
			ServiceName:  serviceName,
			NodeID:       nodeID,
			Source:       "swarm_rollout",
			Line:         line,
			Timestamp:    now,
			Stderr:       false,
		})
	}
	if len(entries) == 0 {
		return
	}

	repo := database.NewDeploymentRuntimeLogsRepository(database.MetricsDB)
	if err := repo.AddLogsBatch(ctx, entries); err != nil {
		logger.Debug("[DeploymentManager] Failed to persist runtime log snapshot for %s: %v", deploymentID, err)
	}
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

	_, mountFlags := sanitizedVolumeMounts(config.DeploymentID, config.Volumes)
	for _, mountFlag := range mountFlags {
		args = append(args, "--mount", mountFlag)
	}

	// Add health check based on configuration
	// Check healthcheck type (default to UNSPECIFIED if not set)
	healthcheckType := int32(0) // HEALTHCHECK_TYPE_UNSPECIFIED
	if config.HealthcheckType != nil {
		healthcheckType = *config.HealthcheckType
		logger.Info("[createSwarmService] Healthcheck type set to %d for service %s", healthcheckType, swarmServiceName)
	} else {
		logger.Info("[createSwarmService] Healthcheck type is nil, defaulting to 0 for service %s", swarmServiceName)
	}

	// Only add healthcheck if not explicitly disabled
	if healthcheckType != 1 { // 1 = HEALTHCHECK_DISABLED
		// Determine effective health check port
		effectiveHealthCheckPort := healthCheckPort
		if config.HealthcheckPort != nil && *config.HealthcheckPort > 0 {
			effectiveHealthCheckPort = int(*config.HealthcheckPort)
		}

		switch healthcheckType {
		case 2: // HEALTHCHECK_TCP
			if effectiveHealthCheckPort > 0 {
				healthCheckCmd := fmt.Sprintf(`sh -c 'if command -v nc >/dev/null 2>&1; then nc -z localhost %d || exit 1; else (apk add --no-cache netcat-openbsd >/dev/null 2>&1 || apt-get update -qq && apt-get install -y -qq netcat-openbsd >/dev/null 2>&1 || yum install -y -q nc >/dev/null 2>&1) && nc -z localhost %d || exit 1; fi'`, effectiveHealthCheckPort, effectiveHealthCheckPort)
				args = append(args,
					"--health-cmd", healthCheckCmd,
					"--health-interval", "30s",
					"--health-timeout", "10s",
					"--health-retries", "3",
					"--health-start-period", "40s",
				)
				logger.Info("[DeploymentManager] Added TCP health check for Swarm service %s on port %d", swarmServiceName, effectiveHealthCheckPort)
			}

		case 3: // HEALTHCHECK_HTTP
			if effectiveHealthCheckPort > 0 {
				path := "/"
				if config.HealthcheckPath != nil && *config.HealthcheckPath != "" {
					path = *config.HealthcheckPath
				}
				expectedStatus := 200
				if config.HealthcheckExpectedStatus != nil && *config.HealthcheckExpectedStatus > 0 {
					expectedStatus = int(*config.HealthcheckExpectedStatus)
				}
				healthCheckCmd := fmt.Sprintf(`sh -c 'if command -v curl >/dev/null 2>&1; then status=$(curl -s -o /dev/null -w "%%{http_code}" http://localhost:%d%s); [ "$status" -eq "%d" ] && exit 0 || exit 1; else (apk add --no-cache curl >/dev/null 2>&1 || apt-get update -qq && apt-get install -y -qq curl >/dev/null 2>&1 || yum install -y -q curl >/dev/null 2>&1) && status=$(curl -s -o /dev/null -w "%%{http_code}" http://localhost:%d%s); [ "$status" -eq "%d" ] && exit 0 || exit 1; fi'`, effectiveHealthCheckPort, path, expectedStatus, effectiveHealthCheckPort, path, expectedStatus)
				args = append(args,
					"--health-cmd", healthCheckCmd,
					"--health-interval", "30s",
					"--health-timeout", "10s",
					"--health-retries", "3",
					"--health-start-period", "40s",
				)
				logger.Info("[DeploymentManager] Added HTTP health check for Swarm service %s on port %d%s (expecting %d)", swarmServiceName, effectiveHealthCheckPort, path, expectedStatus)
			}

		case 4: // HEALTHCHECK_CUSTOM
			if config.HealthcheckCustomCommand != nil && *config.HealthcheckCustomCommand != "" {
				args = append(args,
					"--health-cmd", *config.HealthcheckCustomCommand,
					"--health-interval", "30s",
					"--health-timeout", "10s",
					"--health-retries", "3",
					"--health-start-period", "40s",
				)
				logger.Info("[DeploymentManager] Added custom health check for Swarm service %s: %s", swarmServiceName, *config.HealthcheckCustomCommand)
			}

		default: // HEALTHCHECK_TYPE_UNSPECIFIED (0) - auto-detect
			if effectiveHealthCheckPort > 0 && len(routings) > 0 {
				healthCheckCmd := fmt.Sprintf(`sh -c 'if command -v nc >/dev/null 2>&1; then nc -z localhost %d || exit 1; else (apk add --no-cache netcat-openbsd >/dev/null 2>&1 || apt-get update -qq && apt-get install -y -qq netcat-openbsd >/dev/null 2>&1 || yum install -y -q nc >/dev/null 2>&1) && nc -z localhost %d || exit 1; fi'`, effectiveHealthCheckPort, effectiveHealthCheckPort)
				args = append(args,
					"--health-cmd", healthCheckCmd,
					"--health-interval", "30s",
					"--health-timeout", "10s",
					"--health-retries", "3",
					"--health-start-period", "40s",
				)
				logger.Info("[DeploymentManager] Added auto TCP health check for Swarm service %s on port %d (routing exists)", swarmServiceName, effectiveHealthCheckPort)
			} else {
				logger.Info("[DeploymentManager] No health check for Swarm service %s - type unspecified and no routing rules", swarmServiceName)
			}
		}
	} else {
		logger.Info("[DeploymentManager] Health check explicitly disabled for Swarm service %s", swarmServiceName)
		args = append(args, swarmDisableHealthcheckArgs()...)
	}

	// Add resource limits
	// Memory is in bytes (correct for Swarm)
	args = append(args, "--limit-memory", fmt.Sprintf("%d", config.Memory))

	// CPU: Docker Swarm expects CPU cores, not CPU shares
	// Convert CPU shares to cores (1024 shares = 1 core)
	// Use float64 to support fractional cores (e.g., 0.5, 0.25)
	cpuCores := float64(config.CPUShares) / 1024.0
	args = append(args, "--limit-cpu", fmt.Sprintf("%.2f", cpuCores))

	// Add small placement reservations for idle workloads. Limits still enforce
	// runtime ceilings, but reservations should not consume the whole node.
	reserveMemory := swarmMemoryReservation(config.Memory)
	args = append(args, "--reserve-memory", fmt.Sprintf("%d", reserveMemory))

	reserveCPU := swarmCPUReservation(cpuCores)
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
	registryURL := platform.RegistryURL()

	// Strip protocol from registry URL for comparison (Docker image names don't include protocols)
	registryHost := strings.TrimPrefix(registryURL, "https://")
	registryHost = strings.TrimPrefix(registryHost, "http://")

	// Check if this image is from our registry or a known registry
	isRegistryImage := strings.HasPrefix(config.Image, registryHost+"/") ||
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

	// Override entrypoint if start command is provided (must come BEFORE image)
	// docker service create format: [OPTIONS] IMAGE [COMMAND] [ARG...]
	if config.StartCommand != nil && *config.StartCommand != "" {
		entrypoint, _ := buildStartCommandParts(*config.StartCommand)
		if len(entrypoint) > 0 {
			// Override image entrypoint to avoid inheriting shell wrappers like
			// /bin/bash -c from build images, then run exactly one shell layer.
			args = append(args, "--entrypoint", entrypoint[0])
		}
	}

	// Add image
	args = append(args, config.Image)

	// Add start command if provided (must come after image as COMMAND args)
	if config.StartCommand != nil && *config.StartCommand != "" {
		_, startArgs := buildStartCommandParts(*config.StartCommand)
		args = append(args, startArgs...)
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
			persistDeploymentServiceLogSnapshot(ctx, config.DeploymentID, swarmServiceName, dm.nodeID, initialLogs)
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
							persistDeploymentServiceLogSnapshot(ctx, config.DeploymentID, swarmServiceName, dm.nodeID, logs)
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
					persistDeploymentServiceLogSnapshot(ctx, config.DeploymentID, swarmServiceName, dm.nodeID, checkLogs)
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
										persistDeploymentServiceLogSnapshot(ctx, config.DeploymentID, swarmServiceName, dm.nodeID, logs)
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

	for _, target := range existingSwarmServiceMountTargets(ctx, swarmServiceName) {
		args = append(args, "--mount-rm", target)
	}
	_, mountFlags := sanitizedVolumeMounts(config.DeploymentID, config.Volumes)
	for _, mountFlag := range mountFlags {
		args = append(args, "--mount-add", mountFlag)
	}

	// Update health check based on configuration
	// Check healthcheck type (default to UNSPECIFIED if not set)
	healthcheckType := int32(0) // HEALTHCHECK_TYPE_UNSPECIFIED
	if config.HealthcheckType != nil {
		healthcheckType = *config.HealthcheckType
	}

	// Only add healthcheck if not explicitly disabled
	if healthcheckType != 1 { // 1 = HEALTHCHECK_DISABLED
		// Determine effective health check port
		effectiveHealthCheckPort := healthCheckPort
		if config.HealthcheckPort != nil && *config.HealthcheckPort > 0 {
			effectiveHealthCheckPort = int(*config.HealthcheckPort)
		}

		switch healthcheckType {
		case 2: // HEALTHCHECK_TCP
			if effectiveHealthCheckPort > 0 {
				healthCheckCmd := fmt.Sprintf(`sh -c 'if command -v nc >/dev/null 2>&1; then nc -z localhost %d || exit 1; else (apk add --no-cache netcat-openbsd >/dev/null 2>&1 || apt-get update -qq && apt-get install -y -qq netcat-openbsd >/dev/null 2>&1 || yum install -y -q nc >/dev/null 2>&1) && nc -z localhost %d || exit 1; fi'`, effectiveHealthCheckPort, effectiveHealthCheckPort)
				args = append(args,
					"--health-cmd", healthCheckCmd,
					"--health-interval", "30s",
					"--health-timeout", "10s",
					"--health-retries", "3",
					"--health-start-period", "40s",
				)
				logger.Info("[DeploymentManager] Updated TCP health check for Swarm service %s on port %d", swarmServiceName, effectiveHealthCheckPort)
			}

		case 3: // HEALTHCHECK_HTTP
			if effectiveHealthCheckPort > 0 {
				path := "/"
				if config.HealthcheckPath != nil && *config.HealthcheckPath != "" {
					path = *config.HealthcheckPath
				}
				expectedStatus := 200
				if config.HealthcheckExpectedStatus != nil && *config.HealthcheckExpectedStatus > 0 {
					expectedStatus = int(*config.HealthcheckExpectedStatus)
				}
				healthCheckCmd := fmt.Sprintf(`sh -c 'if command -v curl >/dev/null 2>&1; then status=$(curl -s -o /dev/null -w "%%{http_code}" http://localhost:%d%s); [ "$status" -eq "%d" ] && exit 0 || exit 1; else (apk add --no-cache curl >/dev/null 2>&1 || apt-get update -qq && apt-get install -y -qq curl >/dev/null 2>&1 || yum install -y -q curl >/dev/null 2>&1) && status=$(curl -s -o /dev/null -w "%%{http_code}" http://localhost:%d%s); [ "$status" -eq "%d" ] && exit 0 || exit 1; fi'`, effectiveHealthCheckPort, path, expectedStatus, effectiveHealthCheckPort, path, expectedStatus)
				args = append(args,
					"--health-cmd", healthCheckCmd,
					"--health-interval", "30s",
					"--health-timeout", "10s",
					"--health-retries", "3",
					"--health-start-period", "40s",
				)
				logger.Info("[DeploymentManager] Updated HTTP health check for Swarm service %s on port %d%s (expecting %d)", swarmServiceName, effectiveHealthCheckPort, path, expectedStatus)
			}

		case 4: // HEALTHCHECK_CUSTOM
			if config.HealthcheckCustomCommand != nil && *config.HealthcheckCustomCommand != "" {
				args = append(args,
					"--health-cmd", *config.HealthcheckCustomCommand,
					"--health-interval", "30s",
					"--health-timeout", "10s",
					"--health-retries", "3",
					"--health-start-period", "40s",
				)
				logger.Info("[DeploymentManager] Updated custom health check for Swarm service %s: %s", swarmServiceName, *config.HealthcheckCustomCommand)
			}

		default: // HEALTHCHECK_TYPE_UNSPECIFIED (0) - auto-detect
			if effectiveHealthCheckPort > 0 && len(routings) > 0 {
				healthCheckCmd := fmt.Sprintf(`sh -c 'if command -v nc >/dev/null 2>&1; then nc -z localhost %d || exit 1; else (apk add --no-cache netcat-openbsd >/dev/null 2>&1 || apt-get update -qq && apt-get install -y -qq netcat-openbsd >/dev/null 2>&1 || yum install -y -q nc >/dev/null 2>&1) && nc -z localhost %d || exit 1; fi'`, effectiveHealthCheckPort, effectiveHealthCheckPort)
				args = append(args,
					"--health-cmd", healthCheckCmd,
					"--health-interval", "30s",
					"--health-timeout", "10s",
					"--health-retries", "3",
					"--health-start-period", "40s",
				)
				logger.Info("[DeploymentManager] Updated auto TCP health check for Swarm service %s on port %d (routing exists)", swarmServiceName, effectiveHealthCheckPort)
			} else {
				logger.Info("[DeploymentManager] No health check for Swarm service %s - type unspecified and no routing rules", swarmServiceName)
			}
		}
	} else {
		logger.Info("[DeploymentManager] Health check explicitly disabled for Swarm service %s", swarmServiceName)
		args = append(args, swarmDisableHealthcheckArgs()...)
	}

	// Update resource limits
	args = append(args, "--limit-memory", fmt.Sprintf("%d", config.Memory))

	// CPU: Docker Swarm expects CPU cores, not CPU shares
	cpuCores := float64(config.CPUShares) / 1024.0
	args = append(args, "--limit-cpu", fmt.Sprintf("%.2f", cpuCores))

	// Update small placement reservations for idle workloads. Limits still
	// enforce runtime ceilings, but reservations should not consume the whole node.
	reserveMemory := swarmMemoryReservation(config.Memory)
	args = append(args, "--reserve-memory", fmt.Sprintf("%d", reserveMemory))

	reserveCPU := swarmCPUReservation(cpuCores)
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

	// Update start command when provided.
	// Use an explicit sh entrypoint so image-level entrypoint wrappers do not nest.
	if config.StartCommand != nil && *config.StartCommand != "" {
		entrypoint, startArgs := buildStartCommandParts(*config.StartCommand)
		if len(entrypoint) > 0 {
			args = append(args, "--entrypoint", entrypoint[0])
		} else {
			// Clear any previously forced shell entrypoint so direct executable
			// commands like "./out" run using the image's normal process model.
			args = append(args, "--entrypoint", "")
		}
		for _, arg := range startArgs {
			args = append(args, "--args", arg)
		}
	}

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

	task, err := dm.waitForSwarmServiceConverged(ctx, swarmServiceName)
	if err != nil {
		return serviceID, "", err
	}
	if task.ServiceID != "" {
		serviceID = task.ServiceID
	}
	return serviceID, task.ContainerID, nil
}

func (dm *DeploymentManager) waitForSwarmServiceConverged(ctx context.Context, swarmServiceName string) (*swarmConvergedTask, error) {
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		serviceID, updateState, updateMessage, err := dm.inspectSwarmServiceUpdate(ctx, swarmServiceName)
		if err != nil {
			return nil, err
		}

		switch updateState {
		case "", "completed":
			task, taskErr := dm.currentRunningSwarmTask(ctx, swarmServiceName)
			if taskErr == nil && task != nil && task.ContainerID != "" {
				if task.ServiceID == "" {
					task.ServiceID = serviceID
				}
				return task, nil
			}
		case "rollback_started", "rollback_paused", "rollback_completed", "paused":
			if updateMessage == "" {
				updateMessage = updateState
			}
			return nil, fmt.Errorf("swarm rollout for %s did not converge: %s", swarmServiceName, updateMessage)
		}

		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("timed out waiting for swarm service %s to converge", swarmServiceName)
}

func (dm *DeploymentManager) inspectSwarmServiceUpdate(ctx context.Context, swarmServiceName string) (serviceID string, state string, message string, err error) {
	inspectArgs := []string{"service", "inspect", swarmServiceName, "--format", "{{.ID}}\t{{if .UpdateStatus}}{{.UpdateStatus.State}}\t{{.UpdateStatus.Message}}{{end}}"}
	cmd := exec.CommandContext(ctx, "docker", inspectArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", "", "", fmt.Errorf("failed to inspect swarm service %s: %w (stderr: %s)", swarmServiceName, err, stderr.String())
	}

	parts := strings.SplitN(strings.TrimSpace(stdout.String()), "\t", 3)
	if len(parts) > 0 {
		serviceID = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		state = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 {
		message = strings.TrimSpace(parts[2])
	}
	return serviceID, state, message, nil
}

func (dm *DeploymentManager) currentRunningSwarmTask(ctx context.Context, swarmServiceName string) (*swarmConvergedTask, error) {
	taskArgs := []string{"service", "ps", swarmServiceName, "--format", "{{.ID}}\t{{.CurrentState}}\t{{.DesiredState}}\t{{.Error}}", "--no-trunc"}
	cmd := exec.CommandContext(ctx, "docker", taskArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to inspect swarm tasks for %s: %w (stderr: %s)", swarmServiceName, err, stderr.String())
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 3 {
			continue
		}
		taskID := strings.TrimSpace(parts[0])
		currentState := strings.ToLower(strings.TrimSpace(parts[1]))
		desiredState := strings.ToLower(strings.TrimSpace(parts[2]))
		taskError := ""
		if len(parts) >= 4 {
			taskError = strings.TrimSpace(parts[3])
		}
		if desiredState != "running" || !strings.Contains(currentState, "running") || taskError != "" {
			continue
		}

		inspectArgs := []string{"inspect", taskID, "--format", "{{.ServiceID}}\t{{.Status.ContainerStatus.ContainerID}}"}
		inspectCmd := exec.CommandContext(ctx, "docker", inspectArgs...)
		var inspectStdout bytes.Buffer
		var inspectStderr bytes.Buffer
		inspectCmd.Stdout = &inspectStdout
		inspectCmd.Stderr = &inspectStderr
		if err := inspectCmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to inspect swarm task %s: %w (stderr: %s)", taskID, err, inspectStderr.String())
		}

		taskParts := strings.SplitN(strings.TrimSpace(inspectStdout.String()), "\t", 2)
		task := &swarmConvergedTask{TaskID: taskID}
		if len(taskParts) > 0 {
			task.ServiceID = strings.TrimSpace(taskParts[0])
		}
		if len(taskParts) > 1 {
			task.ContainerID = strings.TrimSpace(taskParts[1])
		}
		if task.ContainerID != "" {
			return task, nil
		}
	}

	return nil, fmt.Errorf("no converged running task found for swarm service %s", swarmServiceName)
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
