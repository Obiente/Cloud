package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/utils"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// Lifecycle operations for deployments

func (dm *DeploymentManager) CreateDeployment(ctx context.Context, config *DeploymentConfig) error {
	logger.Info("[DeploymentManager] Creating deployment %s", config.DeploymentID)

	// Apply plan limits to cap memory and CPU
	if err := dm.applyPlanLimits(config); err != nil {
		logger.Warn("[DeploymentManager] Failed to apply plan limits: %v (continuing anyway)", err)
	}

	// Ensure network exists before creating containers (retry if it failed during initialization)
	if err := dm.ensureNetwork(ctx); err != nil {
		return fmt.Errorf("network is required but could not be created: %w", err)
	}

	// Select best node for deployment
	targetNode, err := dm.nodeSelector.SelectNode(ctx)
	if err != nil {
		logger.Error("[DeploymentManager] Failed to select node for deployment %s: %v", config.DeploymentID, err)
		return fmt.Errorf("failed to select node: %w", err)
	}

	logger.Info("[DeploymentManager] Selected node %s (%s) for deployment %s",
		targetNode.ID, targetNode.Hostname, config.DeploymentID)

	// Check if we're on the target node
	if targetNode.ID != dm.nodeID {
		// Try to forward the request to the target node
		if dm.forwarder.CanForward(targetNode.ID) {
			logger.Info("[DeploymentManager] Forwarding deployment creation to node %s (%s)",
				targetNode.ID, targetNode.Hostname)
			// For now, we'll proceed on current node since forwarding CreateDeployment
			// requires serializing the config and calling the internal API
			// TODO: Implement full forwarding for CreateDeployment via internal API endpoint
			logger.Warn("[DeploymentManager] Node forwarding available but CreateDeployment forwarding not fully implemented. "+
				"Proceeding with deployment on current node %s", dm.nodeID)
		} else {
			logger.Warn("[DeploymentManager] Cannot forward to node %s (%s) - proceeding with deployment on current node %s (%s)",
				targetNode.ID, targetNode.Hostname, dm.nodeID, dm.nodeHostname)
		}
		// Continue with deployment on current node
	}

	// Always reload environment variables from database for Dockerfile deployments
	// This ensures user-specified env vars are not missed
	var deployment database.Deployment
	if err := database.DB.Where("id = ?", config.DeploymentID).First(&deployment).Error; err == nil {
		envVars := make(map[string]string)
		if deployment.EnvVars != "" {
			if err := json.Unmarshal([]byte(deployment.EnvVars), &envVars); err == nil {
				// Merge/override config.EnvVars with database envVars
				for k, v := range envVars {
					config.EnvVars[k] = v
				}
			}
		}
	}
	// Get routing rules to determine service names
	routings, _ := database.GetDeploymentRoutings(config.DeploymentID)
	serviceNames := []string{"default"}
	if len(routings) > 0 {
		// Extract unique service names from routing rules
		serviceNameMap := make(map[string]bool)
		for _, routing := range routings {
			sn := routing.ServiceName
			if sn == "" {
				sn = "default"
			}
			serviceNameMap[sn] = true
		}
		serviceNames = make([]string, 0, len(serviceNameMap))
		for sn := range serviceNameMap {
			serviceNames = append(serviceNames, sn)
		}
	}

	// Check if we're in Swarm mode
	isSwarmMode := utils.IsSwarmModeEnabled()

	// Create containers/services for each service and replica
	for _, serviceName := range serviceNames {
		for i := 0; i < config.Replicas; i++ {
			containerName := fmt.Sprintf("%s-%s-replica-%d", config.DeploymentID, serviceName, i)

			var containerID string
			var err error

			if isSwarmMode {
				// In Swarm mode, create Swarm services instead of plain containers
				swarmServiceName := fmt.Sprintf("deploy-%s-%s", config.DeploymentID, serviceName)
				if i > 0 {
					swarmServiceName = fmt.Sprintf("deploy-%s-%s-replica-%d", config.DeploymentID, serviceName, i)
				}

				// Check if service already exists
				checkArgs := []string{"service", "inspect", swarmServiceName, "--format", "{{.ID}}"}
				checkCmd := exec.CommandContext(ctx, "docker", checkArgs...)
				var checkStderr bytes.Buffer
				checkCmd.Stderr = &checkStderr
				serviceExists := checkCmd.Run() == nil

				if serviceExists {
					// Service exists - update it for zero-downtime deployment
					logger.Info("[DeploymentManager] Swarm service %s already exists - updating with zero-downtime strategy (start-first)", swarmServiceName)
					_, containerID, err = dm.updateSwarmService(ctx, config, serviceName, i, swarmServiceName)
					if err != nil {
						return fmt.Errorf("failed to update Swarm service: %w", err)
					}
				} else {
					// Service doesn't exist - create it
					logger.Info("[DeploymentManager] Creating new Swarm service for deployment %s (service: %s, replica: %d)", config.DeploymentID, serviceName, i)
					_, containerID, err = dm.createSwarmService(ctx, config, serviceName, i)
					if err != nil {
						return fmt.Errorf("failed to create Swarm service: %w", err)
					}
				}

				// For Swarm services, containerID might be empty initially - try to get it from service tasks
				if containerID == "" {
					// Wait a bit more for task to be created
					time.Sleep(2 * time.Second)
					// Try to find container by service name label
					filterArgs := make(client.Filters)
					filterArgs.Add("label", fmt.Sprintf("com.docker.swarm.service.name=%s", swarmServiceName))
					containersResult, listErr := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
						All:     true,
						Filters: filterArgs,
					})
					if listErr == nil && len(containersResult.Items) > 0 {
						containerID = containersResult.Items[0].ID
					}
				}

				if containerID == "" {
					logger.Warn("[DeploymentManager] Could not get container ID from Swarm service %s - service created but container lookup failed", swarmServiceName)
					// Continue anyway - the service exists and Traefik can discover it
					// We'll use a placeholder container ID
					containerID = "swarm-service-" + swarmServiceName
				}
			} else {
				// In non-Swarm mode, create plain containers
				// Remove existing container with this name if it exists (for redeployments)
				if err := dm.removeContainerByName(ctx, containerName); err != nil {
					logger.Warn("[DeploymentManager] Failed to remove existing container %s: %v (will attempt to create anyway)", containerName, err)
				}

				containerID, err = dm.createContainer(ctx, config, containerName, i, serviceName)
				if err != nil {
					return fmt.Errorf("failed to create container: %w", err)
				}

				// Start container
				if err := dm.dockerHelper.StartContainer(ctx, containerID); err != nil {
					return fmt.Errorf("failed to start container: %w", err)
				}
			}

			// Get container details (if containerID is valid, not a placeholder)
			var publicPort int
			if !strings.HasPrefix(containerID, "swarm-service-") {
				infoResult, err := dm.dockerClient.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
				if err != nil {
					continue
				}
				info := infoResult.Container
				// Determine the public port (find port for this service from routing)
				publicPort = config.Port
				for _, routing := range routings {
					if routing.ServiceName == serviceName || (serviceName == "default" && routing.ServiceName == "") {
						publicPort = routing.TargetPort
						break
					}
				}
				if len(info.NetworkSettings.Ports) > 0 {
					for _, bindings := range info.NetworkSettings.Ports {
						if len(bindings) > 0 {
							if port, err := strconv.Atoi(bindings[0].HostPort); err == nil {
								publicPort = port
							}
						}
					}
				}
			} else {
				// Placeholder container ID - use routing port
				publicPort = config.Port
				for _, routing := range routings {
					if routing.ServiceName == serviceName || (serviceName == "default" && routing.ServiceName == "") {
						publicPort = routing.TargetPort
						break
					}
				}
			}

			// Register deployment location
			location := &database.DeploymentLocation{
				ID: func() string {
					shortID := containerID
					if len(shortID) > 12 {
						shortID = shortID[:12]
					}
					return fmt.Sprintf("loc-%s-%s", config.DeploymentID, shortID)
				}(),
				DeploymentID: config.DeploymentID,
				NodeID:       dm.nodeID,
				NodeHostname: dm.nodeHostname,
				ContainerID:  containerID,
				Status:       "running",
				Port:         publicPort,
				Domain:       config.Domain,
				HealthStatus: "unknown",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}

			if err := dm.registry.RegisterDeployment(ctx, location); err != nil {
				logger.Warn("[DeploymentManager] Failed to register deployment: %v", err)
			}

			if isSwarmMode {
				shortID := containerID
				if len(shortID) > 12 {
					shortID = shortID[:12]
				}
				logger.Info("[DeploymentManager] Successfully created Swarm service for deployment %s (service: %s, replica: %d, container: %s)",
					config.DeploymentID, serviceName, i, shortID)
			} else {
				logger.Info("[DeploymentManager] Successfully created container %s for deployment %s (service: %s)",
					containerID[:12], config.DeploymentID, serviceName)
			}
		}
	}

	// No longer create default routing rules automatically.
	// Users must explicitly configure routing rules if they need them.
	// This prevents unnecessary healthcheck injection for worker/process deployments.
	existingRoutings, _ := database.GetDeploymentRoutings(config.DeploymentID)
	if len(existingRoutings) > 0 {
		logger.Info("[DeploymentManager] Deployment %s has %d routing rule(s)", config.DeploymentID, len(existingRoutings))
	} else {
		logger.Info("[DeploymentManager] Deployment %s has no routing rules (process/worker deployment)", config.DeploymentID)
	}

	logger.Info("[DeploymentManager] Deployment %s created successfully", config.DeploymentID)
	return nil
}

func (dm *DeploymentManager) StartDeployment(ctx context.Context, deploymentID string) error {
	logger.Info("[DeploymentManager] Starting deployment %s", deploymentID)

	locations, err := dm.registry.GetDeploymentLocations(deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get deployment locations: %w", err)
	}

	// Check if we have any containers for this deployment
	if len(locations) == 0 {
		logger.Info("[DeploymentManager] No containers found for deployment %s, attempting to create them", deploymentID)

		// Try to get deployment from database to create containers
		var deployment database.Deployment
		if err := database.DB.Where("id = ?", deploymentID).First(&deployment).Error; err != nil {
			return fmt.Errorf("failed to get deployment from database: %w", err)
		}

		// Log healthcheck values from database
		logger.Info("[StartDeployment] Deployment %s loaded from DB - HealthcheckType: %v, HealthcheckPort: %v, HealthcheckPath: %v, HealthcheckExpectedStatus: %v, HealthcheckCustomCommand: %v",
			deploymentID, deployment.HealthcheckType, deployment.HealthcheckPort, deployment.HealthcheckPath, deployment.HealthcheckExpectedStatus, deployment.HealthcheckCustomCommand)

		// Only create containers if this is not a compose-based deployment
		// Compose deployments should be handled by DeployComposeFile
		if deployment.ComposeYaml == "" {
			// Parse environment variables from JSON
			envVars := make(map[string]string)
			if deployment.EnvVars != "" {
				if err := json.Unmarshal([]byte(deployment.EnvVars), &envVars); err != nil {
					logger.Warn("[DeploymentManager] Failed to parse env vars for deployment %s: %v", deploymentID, err)
					// Continue with empty env vars
				}
			}

			// Build config from database deployment
			image := ""
			if deployment.Image != nil {
				image = *deployment.Image
			}

			// Get port from routing configuration (required - no default)
			port := 0
			routings, routingErr := database.GetDeploymentRoutings(deploymentID)
			if routingErr == nil && len(routings) > 0 {
				// Find routing rule for "default" service (or first one if no service name specified)
				foundRouting := false
				for _, routing := range routings {
					if routing.ServiceName == "" || routing.ServiceName == "default" {
						if routing.TargetPort > 0 {
							port = routing.TargetPort
							logger.Info("[StartDeployment] Using target port %d from routing configuration (default service) for deployment %s", port, deploymentID)
							foundRouting = true
							break
						}
					}
				}
				// If no default service routing found, use first routing's target port
				if !foundRouting && len(routings) > 0 && routings[0].TargetPort > 0 {
					port = routings[0].TargetPort
					logger.Info("[StartDeployment] Using target port %d from first routing rule for deployment %s", port, deploymentID)
				}
			}

			// Fallback to deployment port only if no routing found
			if port == 0 && deployment.Port != nil && *deployment.Port > 0 {
				port = int(*deployment.Port)
				logger.Info("[StartDeployment] Using deployment port %d (no routing found) for deployment %s", port, deploymentID)
			}

			// Set backend default per-deployment limits: 2GB RAM, 0.5 CPU cores (512 shares)
			memory := int64(2 * 1024 * 1024 * 1024) // Default 2GB
			if deployment.MemoryBytes != nil && *deployment.MemoryBytes > 0 {
				memory = *deployment.MemoryBytes
			}
			cpuShares := int64(512) // Default 0.5 CPU cores (512 shares)
			if deployment.CPUShares != nil && *deployment.CPUShares > 0 {
				cpuShares = *deployment.CPUShares
			}
			// Enforce org plan-based max memory and CPU
			maxMemoryBytes, maxCPUCores, err := dm.getPlanLimitsForDeployment(deploymentID)
			if err == nil {
				if maxMemoryBytes > 0 && memory > maxMemoryBytes {
					logger.Info("[DeploymentManager] Capping memory for deployment %s from %d bytes to plan limit %d bytes (restart)", deploymentID, memory, maxMemoryBytes)
					memory = maxMemoryBytes
				}
				if maxCPUCores > 0 {
					maxCPUShares := int64(maxCPUCores) * 1024
					if cpuShares > maxCPUShares {
						logger.Info("[DeploymentManager] Capping CPU for deployment %s from %d shares (%d cores) to plan limit %d shares (%d cores) (restart)",
							deploymentID, cpuShares, cpuShares/1024, maxCPUShares, maxCPUCores)
						cpuShares = maxCPUShares
					}
				}
			}
			replicas := 1 // Default
			if deployment.Replicas != nil {
				replicas = int(*deployment.Replicas)
			}

			if port == 0 {
				logger.Info("[StartDeployment] Deployment %s has no exposed port configured; continuing without Traefik/health checks", deploymentID)
			}

			config := &DeploymentConfig{
				DeploymentID:              deploymentID,
				Image:                     image,
				Domain:                    deployment.Domain,
				Port:                      port,
				EnvVars:                   envVars,
				Labels:                    map[string]string{},
				Memory:                    memory,
				CPUShares:                 cpuShares,
				Replicas:                  replicas,
				StartCommand:              deployment.StartCommand,
				HealthcheckType:           deployment.HealthcheckType,
				HealthcheckPort:           deployment.HealthcheckPort,
				HealthcheckPath:           deployment.HealthcheckPath,
				HealthcheckExpectedStatus: deployment.HealthcheckExpectedStatus,
				HealthcheckCustomCommand:  deployment.HealthcheckCustomCommand,
			}

			// Log the config healthcheck values
			logger.Info("[StartDeployment] DeploymentConfig created - HealthcheckType: %v, HealthcheckPort: %v, HealthcheckPath: %v, HealthcheckExpectedStatus: %v, HealthcheckCustomCommand: %v",
				config.HealthcheckType, config.HealthcheckPort, config.HealthcheckPath, config.HealthcheckExpectedStatus, config.HealthcheckCustomCommand)

			// Create the containers
			if err := dm.CreateDeployment(ctx, config); err != nil {
				return fmt.Errorf("failed to create deployment containers: %w", err)
			}

			logger.Info("[DeploymentManager] Successfully created containers for deployment %s", deploymentID)

			// Refresh locations after creation
			locations, err = dm.registry.GetDeploymentLocations(deploymentID)
			if err != nil {
				return fmt.Errorf("failed to get deployment locations after creation: %w", err)
			}
		} else {
			// Compose-based deployment - return error suggesting to use DeployComposeFile
			return fmt.Errorf("no containers found for deployment %s - compose deployments should use DeployComposeFile", deploymentID)
		}
	}

	for _, location := range locations {
		// Only start containers on this node
		if location.NodeID != dm.nodeID {
			continue
		}

		// Check if container exists and is stopped
		var containerInfo container.InspectResponse
		containerInfoResult, err := dm.dockerClient.ContainerInspect(ctx, location.ContainerID, client.ContainerInspectOptions{})
		if err != nil {
			// Container doesn't exist - try to recreate it
			logger.Warn("[DeploymentManager] Container %s doesn't exist, attempting to recreate deployment", location.ContainerID[:12])

			// Get deployment from database to recreate containers
			var deployment database.Deployment
			if err := database.DB.Where("id = ?", deploymentID).First(&deployment).Error; err != nil {
				logger.Warn("[DeploymentManager] Failed to get deployment from database for recreation: %v", err)
				continue
			}

			// Log healthcheck values from database
			logger.Info("[StartDeployment-Recreate] Deployment %s loaded from DB - HealthcheckType: %v, HealthcheckPort: %v, HealthcheckPath: %v, HealthcheckExpectedStatus: %v, HealthcheckCustomCommand: %v",
				deploymentID, deployment.HealthcheckType, deployment.HealthcheckPort, deployment.HealthcheckPath, deployment.HealthcheckExpectedStatus, deployment.HealthcheckCustomCommand)

			// Only recreate if this is not a compose-based deployment
			if deployment.ComposeYaml == "" {
				// Parse environment variables from JSON
				envVars := make(map[string]string)
				if deployment.EnvVars != "" {
					if err := json.Unmarshal([]byte(deployment.EnvVars), &envVars); err != nil {
						logger.Warn("[DeploymentManager] Failed to parse env vars for deployment %s: %v", deploymentID, err)
						// Continue with empty env vars
					}
				}

				// Build config from database deployment
				image := ""
				if deployment.Image != nil {
					image = *deployment.Image
				}

				// Get port from routing configuration (required - no default)
				port := 0
				routings, routingErr := database.GetDeploymentRoutings(deploymentID)
				if routingErr == nil && len(routings) > 0 {
					// Find routing rule for "default" service (or first one if no service name specified)
					foundRouting := false
					for _, routing := range routings {
						if routing.ServiceName == "" || routing.ServiceName == "default" {
							if routing.TargetPort > 0 {
								port = routing.TargetPort
								logger.Info("[StartDeployment] Using target port %d from routing configuration (default service) for deployment %s (recreate)", port, deploymentID)
								foundRouting = true
								break
							}
						}
					}
					// If no default service routing found, use first routing's target port
					if !foundRouting && len(routings) > 0 && routings[0].TargetPort > 0 {
						port = routings[0].TargetPort
						logger.Info("[StartDeployment] Using target port %d from first routing rule for deployment %s (recreate)", port, deploymentID)
					}
				}

				// Fallback to deployment port only if no routing found
				if port == 0 && deployment.Port != nil && *deployment.Port > 0 {
					port = int(*deployment.Port)
					logger.Info("[StartDeployment] Using deployment port %d (no routing found) for deployment %s (recreate)", port, deploymentID)
				}
				if port == 0 {
					logger.Info("[StartDeployment] Deployment %s still has no exposed port after recreation lookup; continuing without health checks", deploymentID)
				}

				// Set a reasonable default hard memory limit (2GB)
				memory := int64(2 * 1024 * 1024 * 1024) // Default 2GB
				if deployment.MemoryBytes != nil {
					memory = *deployment.MemoryBytes
				}
				cpuShares := int64(102) // Default 0.1 CPU (102 shares = 0.1 cores)
				if deployment.CPUShares != nil {
					cpuShares = *deployment.CPUShares
				}
				replicas := 1 // Default
				if deployment.Replicas != nil {
					replicas = int(*deployment.Replicas)
				}

				config := &DeploymentConfig{
					DeploymentID:              deploymentID,
					Image:                     image,
					Domain:                    deployment.Domain,
					Port:                      port,
					EnvVars:                   envVars,
					Labels:                    map[string]string{},
					Memory:                    memory,
					CPUShares:                 cpuShares,
					Replicas:                  replicas,
					StartCommand:              deployment.StartCommand,
					HealthcheckType:           deployment.HealthcheckType,
					HealthcheckPort:           deployment.HealthcheckPort,
					HealthcheckPath:           deployment.HealthcheckPath,
					HealthcheckExpectedStatus: deployment.HealthcheckExpectedStatus,
					HealthcheckCustomCommand:  deployment.HealthcheckCustomCommand,
				}

				// Log the config healthcheck values
				logger.Info("[StartDeployment-Recreate] DeploymentConfig created - HealthcheckType: %v, HealthcheckPort: %v, HealthcheckPath: %v, HealthcheckExpectedStatus: %v, HealthcheckCustomCommand: %v",
					config.HealthcheckType, config.HealthcheckPort, config.HealthcheckPath, config.HealthcheckExpectedStatus, config.HealthcheckCustomCommand)

				// Recreate the containers
				if err := dm.CreateDeployment(ctx, config); err != nil {
					logger.Warn("[DeploymentManager] Failed to recreate deployment containers: %v", err)
					continue
				}

				logger.Info("[DeploymentManager] Successfully recreated containers for deployment %s", deploymentID)

				// Refresh locations after recreation
				locations, err = dm.registry.GetDeploymentLocations(deploymentID)
				if err != nil {
					logger.Warn("[DeploymentManager] Failed to get deployment locations after recreation: %v", err)
					continue
				}

				// Find the location for this container again
				found := false
				for _, loc := range locations {
					if loc.NodeID == dm.nodeID {
						location = loc
						found = true
						break
					}
				}
				if !found {
					logger.Warn("[DeploymentManager] Could not find recreated container location for deployment %s", deploymentID)
					continue
				}

				// Re-inspect the new container
				containerInfoResult, err := dm.dockerClient.ContainerInspect(ctx, location.ContainerID, client.ContainerInspectOptions{})
				if err != nil {
					logger.Warn("[DeploymentManager] Failed to inspect recreated container: %v", err)
					continue
				}
				containerInfo = containerInfoResult.Container
			} else {
				// Compose-based deployment - skip this container
				logger.Warn("[DeploymentManager] Container %s doesn't exist for compose deployment %s, skipping", location.ContainerID[:12], deploymentID)
				continue
			}
		} else {
			containerInfo = containerInfoResult.Container
		}

		// Only start if not already running
		if !containerInfo.State.Running {
			// Start container
			if err := dm.dockerHelper.StartContainer(ctx, location.ContainerID); err != nil {
				logger.Info("[DeploymentManager] Failed to start container %s: %v", location.ContainerID[:12], err)
				continue
			}

			// Update status
			database.DB.Model(&database.DeploymentLocation{}).
				Where("container_id = ?", location.ContainerID).
				Update("status", "running")

			logger.Info("[DeploymentManager] Started container %s", location.ContainerID[:12])
		} else {
			logger.Info("[DeploymentManager] Container %s is already running", location.ContainerID[:12])
		}
	}

	return nil
}

// StopDeployment stops all containers for a deployment
func (dm *DeploymentManager) StopDeployment(ctx context.Context, deploymentID string) error {
	logger.Info("[DeploymentManager] Stopping deployment %s", deploymentID)

	// Check if we're in Swarm mode
	isSwarmMode := utils.IsSwarmModeEnabled()

	if isSwarmMode {
		// In Swarm mode, we need to remove Swarm services
		// Find all services for this deployment
		// Service names follow pattern: deploy-{deploymentID}-{serviceName} or deploy-{deploymentID}-{serviceName}-replica-{i}
		cmd := exec.CommandContext(ctx, "docker", "service", "ls", "--format", "{{.Name}}")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			serviceNames := strings.Split(strings.TrimSpace(stdout.String()), "\n")
			prefix := fmt.Sprintf("deploy-%s-", deploymentID)
			for _, serviceName := range serviceNames {
				serviceName = strings.TrimSpace(serviceName)
				if strings.HasPrefix(serviceName, prefix) {
					// Remove the Swarm service
					rmArgs := []string{"service", "rm", serviceName}
					rmCmd := exec.CommandContext(ctx, "docker", rmArgs...)
					var rmStderr bytes.Buffer
					rmCmd.Stderr = &rmStderr
					if err := rmCmd.Run(); err != nil {
						logger.Warn("[DeploymentManager] Failed to remove Swarm service %s: %v (stderr: %s)", serviceName, err, rmStderr.String())
					} else {
						logger.Info("[DeploymentManager] Removed Swarm service %s", serviceName)
					}
				}
			}
		}

		// Update all locations to stopped status
		database.DB.Model(&database.DeploymentLocation{}).
			Where("deployment_id = ?", deploymentID).
			Update("status", "stopped")

		return nil
	}

	// Non-Swarm mode: stop containers
	locations, err := dm.registry.GetDeploymentLocations(deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get deployment locations: %w", err)
	}

	for _, location := range locations {
		// Only stop containers on this node
		if location.NodeID != dm.nodeID {
			continue
		}

		// Stop container
		timeout := int(30) // 30 seconds
		if err := dm.dockerHelper.StopContainer(ctx, location.ContainerID, time.Duration(timeout)*time.Second); err != nil {
			logger.Info("[DeploymentManager] Failed to stop container %s: %v", location.ContainerID, err)
			continue
		}

		// Update status
		database.DB.Model(&database.DeploymentLocation{}).
			Where("container_id = ?", location.ContainerID).
			Update("status", "stopped")

		logger.Info("[DeploymentManager] Stopped container %s", location.ContainerID[:12])
	}

	return nil
}

// DeleteDeployment removes all containers and data for a deployment
func (dm *DeploymentManager) DeleteDeployment(ctx context.Context, deploymentID string) error {
	logger.Info("[DeploymentManager] Deleting deployment %s", deploymentID)

	locations, err := dm.registry.GetDeploymentLocations(deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get deployment locations: %w", err)
	}

	for _, location := range locations {
		// Only delete containers on this node
		if location.NodeID != dm.nodeID {
			continue
		}

		// SECURITY: Verify container was created by our API before deletion
		containerInfoResult, err := dm.dockerClient.ContainerInspect(ctx, location.ContainerID, client.ContainerInspectOptions{})
		if err != nil {
			logger.Warn("[DeploymentManager] Container %s not found (may already be deleted): %v", location.ContainerID[:12], err)
			continue
		}
		containerInfo := containerInfoResult.Container

		// Verify container has our management label
		if containerInfo.Config.Labels["cloud.obiente.managed"] != "true" {
			logger.Error("[DeploymentManager] SECURITY: Refusing to delete container %s: not managed by Obiente Cloud (missing cloud.obiente.managed=true label)", location.ContainerID[:12])
			continue
		}

		// Stop container first
		timeout := int(10)
		_ = dm.dockerHelper.StopContainer(ctx, location.ContainerID, time.Duration(timeout)*time.Second)

		// Remove container
		if err := dm.dockerHelper.RemoveContainer(ctx, location.ContainerID, true); err != nil {
			logger.Info("[DeploymentManager] Failed to remove container %s: %v", location.ContainerID, err)
			continue
		}

		// Unregister from registry
		if err := dm.registry.UnregisterDeployment(ctx, location.ContainerID); err != nil {
			logger.Info("[DeploymentManager] Failed to unregister deployment: %v", err)
		}

		logger.Info("[DeploymentManager] Deleted container %s", location.ContainerID[:12])
	}

	// Clean up volumes and deployment data
	dm.cleanupDeploymentData(deploymentID)

	return nil
}

// RestartDeployment restarts all containers for a deployment by recreating them
// This ensures that configs (labels, health checks, etc.) are updated
func (dm *DeploymentManager) RestartDeployment(ctx context.Context, deploymentID string) error {
	logger.Info("[DeploymentManager] Restarting deployment %s (will recreate containers to update configs)", deploymentID)

	// Get deployment from database to check if it's compose-based
	var deployment database.Deployment
	if err := database.DB.Where("id = ?", deploymentID).First(&deployment).Error; err != nil {
		return fmt.Errorf("failed to get deployment from database: %w", err)
	}

	// Log healthcheck values from database
	logger.Info("[RestartDeployment] Deployment %s loaded from DB - HealthcheckType: %v, HealthcheckPort: %v, HealthcheckPath: %v, HealthcheckExpectedStatus: %v, HealthcheckCustomCommand: %v",
		deploymentID, deployment.HealthcheckType, deployment.HealthcheckPort, deployment.HealthcheckPath, deployment.HealthcheckExpectedStatus, deployment.HealthcheckCustomCommand)

	// For compose deployments, stop and redeploy (which already updates configs)
	if deployment.ComposeYaml != "" {
		logger.Info("[DeploymentManager] Compose-based deployment - stopping and redeploying to update configs")
		_ = dm.StopComposeDeployment(ctx, deploymentID)
		return dm.DeployComposeFile(ctx, deploymentID, deployment.ComposeYaml)
	}

	// For image-based deployments, recreate containers with updated configs
	logger.Info("[DeploymentManager] Image-based deployment - recreating containers to update configs")

	// Get all locations to stop and remove existing containers
	locations, err := database.GetAllDeploymentLocations(deploymentID)
	if err != nil {
		logger.Warn("[DeploymentManager] Failed to get deployment locations: %v, will proceed with recreation", err)
		locations = []database.DeploymentLocation{}
	}

	// Stop and remove existing containers on this node
	for _, location := range locations {
		if location.NodeID != dm.nodeID {
			continue
		}

		// Check if container exists
		_, err := dm.dockerClient.ContainerInspect(ctx, location.ContainerID, client.ContainerInspectOptions{})
		if err != nil {
			logger.Debug("[DeploymentManager] Container %s doesn't exist, skipping removal", location.ContainerID[:12])
			continue
		}

		// Stop container
		timeout := 10 * time.Second
		_ = dm.dockerHelper.StopContainer(ctx, location.ContainerID, timeout)

		// Remove container
		if err := dm.dockerHelper.RemoveContainer(ctx, location.ContainerID, true); err != nil {
			logger.Warn("[DeploymentManager] Failed to remove container %s: %v", location.ContainerID[:12], err)
			continue
		}

		// Unregister from registry
		_ = dm.registry.UnregisterDeployment(ctx, location.ContainerID)
		logger.Info("[DeploymentManager] Removed container %s for recreation", location.ContainerID[:12])
	}

	// Recreate containers with updated config from database
	// Parse environment variables from JSON
	envVars := make(map[string]string)
	if deployment.EnvVars != "" {
		if err := json.Unmarshal([]byte(deployment.EnvVars), &envVars); err != nil {
			logger.Warn("[DeploymentManager] Failed to parse env vars for deployment %s: %v", deploymentID, err)
		}
	}

	// Get image
	image := ""
	if deployment.Image != nil {
		image = *deployment.Image
	}
	if image == "" {
		return fmt.Errorf("deployment %s has no image configured", deploymentID)
	}

	// Get port from routing configuration (required - no default)
	port := 0
	routings, routingErr := database.GetDeploymentRoutings(deploymentID)
	if routingErr == nil && len(routings) > 0 {
		// Find routing rule for "default" service (or first one if no service name specified)
		foundRouting := false
		for _, routing := range routings {
			if routing.ServiceName == "" || routing.ServiceName == "default" {
				if routing.TargetPort > 0 {
					port = routing.TargetPort
					logger.Info("[DeploymentManager] Using target port %d from routing configuration (default service) for restart", port)
					foundRouting = true
					break
				}
			}
		}
		// If no default service routing found, use first routing's target port
		if !foundRouting && len(routings) > 0 && routings[0].TargetPort > 0 {
			port = routings[0].TargetPort
			logger.Info("[DeploymentManager] Using target port %d from first routing rule for restart", port)
		}
	}

	// Fallback to deployment port only if no routing found
	if port == 0 && deployment.Port != nil && *deployment.Port > 0 {
		port = int(*deployment.Port)
		logger.Info("[DeploymentManager] Using deployment port %d (no routing found) for restart", port)
	}
	if port == 0 {
		logger.Info("[DeploymentManager] Deployment %s restarting without an exposed port", deploymentID)
	}

	// Defaults match StartDeployment: 2GB RAM, 0.5 CPU cores (512 shares).
	// Per-deployment overrides (deployments.memory_bytes / deployments.cpu_shares) take precedence when set.
	// Org plan caps are applied inside CreateDeployment() via applyPlanLimits().
	memory := int64(2 * 1024 * 1024 * 1024) // Default 2GB
	if deployment.MemoryBytes != nil && *deployment.MemoryBytes > 0 {
		memory = *deployment.MemoryBytes
	}
	cpuShares := int64(512) // Default 0.5 CPU cores (512 shares)
	if deployment.CPUShares != nil && *deployment.CPUShares > 0 {
		cpuShares = *deployment.CPUShares
	}
	replicas := 1 // Default
	if deployment.Replicas != nil {
		replicas = int(*deployment.Replicas)
	}

	// Create deployment config
	config := &DeploymentConfig{
		DeploymentID:              deploymentID,
		Image:                     image,
		Domain:                    deployment.Domain,
		Port:                      port,
		EnvVars:                   envVars,
		Labels:                    map[string]string{},
		Memory:                    memory,
		CPUShares:                 cpuShares,
		Replicas:                  replicas,
		StartCommand:              deployment.StartCommand,
		HealthcheckType:           deployment.HealthcheckType,
		HealthcheckPort:           deployment.HealthcheckPort,
		HealthcheckPath:           deployment.HealthcheckPath,
		HealthcheckExpectedStatus: deployment.HealthcheckExpectedStatus,
		HealthcheckCustomCommand:  deployment.HealthcheckCustomCommand,
	}

	// Log the config healthcheck values
	logger.Info("[RestartDeployment] DeploymentConfig created - HealthcheckType: %v, HealthcheckPort: %v, HealthcheckPath: %v, HealthcheckExpectedStatus: %v, HealthcheckCustomCommand: %v",
		config.HealthcheckType, config.HealthcheckPort, config.HealthcheckPath, config.HealthcheckExpectedStatus, config.HealthcheckCustomCommand)

	// Recreate containers with updated config
	if err := dm.CreateDeployment(ctx, config); err != nil {
		return fmt.Errorf("failed to recreate deployment containers: %w", err)
	}

	logger.Info("[DeploymentManager] Successfully recreated containers for deployment %s with updated configs", deploymentID)
	return nil
}

// ScaleDeployment changes the number of replicas for a deployment
func (dm *DeploymentManager) ScaleDeployment(ctx context.Context, deploymentID string, replicas int) error {
	logger.Info("[DeploymentManager] Scaling deployment %s to %d replicas", deploymentID, replicas)

	locations, err := dm.registry.GetDeploymentLocations(deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get deployment locations: %w", err)
	}

	currentReplicas := len(locations)

	if replicas > currentReplicas {
		// Scale up: Need to get deployment config and create more containers
		// This would require storing deployment config in database
		return fmt.Errorf("scale up not yet implemented")
	} else if replicas < currentReplicas {
		// Scale down: Remove excess containers
		containersToRemove := currentReplicas - replicas
		for i := 0; i < containersToRemove && i < len(locations); i++ {
			location := locations[i]
			if location.NodeID != dm.nodeID {
				continue
			}

			// Stop and remove container
			timeout := int(10)
			_ = dm.dockerHelper.StopContainer(ctx, location.ContainerID, time.Duration(timeout)*time.Second)
			_ = dm.dockerHelper.RemoveContainer(ctx, location.ContainerID, true)
			dm.registry.UnregisterDeployment(ctx, location.ContainerID)

			logger.Info("[DeploymentManager] Removed replica %s", location.ContainerID[:12])
		}
	}

	return nil
}

// GetDeploymentLogs retrieves logs from a deployment
func (dm *DeploymentManager) GetDeploymentLogs(ctx context.Context, deploymentID string, tail string) (string, error) {
	locations, err := dm.registry.GetDeploymentLocations(deploymentID)
	if err != nil {
		return "", fmt.Errorf("failed to get deployment locations: %w", err)
	}

	if len(locations) == 0 {
		return "", fmt.Errorf("no containers found for deployment")
	}

	// Get logs from first container on this node
	for _, location := range locations {
		if location.NodeID == dm.nodeID {
			logs, err := dm.dockerHelper.ContainerLogs(ctx, location.ContainerID, tail, false, nil, nil) // follow=false for non-streaming logs
			if err != nil {
				return "", fmt.Errorf("failed to get logs: %w", err)
			}
			defer logs.Close()

			// Read logs (simplified - in production, handle streams properly)
			buf := make([]byte, 4096)
			n, _ := logs.Read(buf)
			return string(buf[:n]), nil
		}
	}

	return "", fmt.Errorf("no containers found on this node")
}

// GetDeploymentStats retrieves resource usage statistics
// GetDeploymentStats is currently not implemented; stats streaming will be added later.
// func (dm *DeploymentManager) GetDeploymentStats(ctx context.Context, deploymentID string) ([]types.StatsJSON, error) {
// 	return nil, fmt.Errorf("not implemented")
// }

// DeployComposeFile deploys a Docker Compose file for a deployment
