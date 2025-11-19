package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"

	"github.com/moby/moby/api/types/filters"
	"github.com/moby/moby/client"
	"gopkg.in/yaml.v3"
)

// isSwarmModeEnabled checks if Swarm mode is enabled via ENABLE_SWARM environment variable

// Compose operations for deployments

func (dm *DeploymentManager) DeployComposeFile(ctx context.Context, deploymentID string, composeYaml string) error {
	logger.Info("[DeploymentManager] Deploying compose file for deployment %s", deploymentID)

	// Ensure network exists before deploying (retry if it failed during initialization)
	if err := dm.ensureNetwork(ctx); err != nil {
		return fmt.Errorf("network is required but could not be created: %w", err)
	}

	// Sanitize compose file for security (transform volumes, remove host ports, etc.)
	sanitizer := NewComposeSanitizer(deploymentID)
	sanitizedYaml, err := sanitizer.SanitizeComposeYAML(composeYaml)
	if err != nil {
		logger.Warn("[DeploymentManager] Failed to sanitize compose YAML for deployment %s: %v. Using original YAML.", deploymentID, err)
		// Continue with original YAML if sanitization fails (but log the warning)
		sanitizedYaml = composeYaml
	} else {
		logger.Info("[DeploymentManager] Sanitized compose YAML for deployment %s (volumes mapped to: %s)", deploymentID, sanitizer.GetSafeBaseDir())
	}

	// Get routing rules (create default if none exist)
	routings, _ := database.GetDeploymentRoutings(deploymentID)
	if len(routings) == 0 {
		// Check if a default routing already exists (might have been created previously)
		// This handles the case where GetDeploymentRoutings returns empty but a routing exists in DB
		// (e.g., due to race conditions or if user manually set a port)
		defaultRoutingID := fmt.Sprintf("route-%s-default", deploymentID)
		var existingDefaultRouting database.DeploymentRouting
		dbErr := database.DB.Where("id = ?", defaultRoutingID).First(&existingDefaultRouting).Error

		// If a default routing exists, preserve all user settings (especially port)
		if dbErr == nil {
			// User has set routing rules - preserve them completely
			routings = []database.DeploymentRouting{existingDefaultRouting}
			logger.Info("[DeploymentManager] Found existing default routing for deployment %s, preserving all user settings (port: %d)", deploymentID, existingDefaultRouting.TargetPort)
		} else {
			// No existing routing found, try to parse compose file to detect port
			var targetPort int = 0
			// Parse compose to detect exposed ports from first service
			var compose map[string]interface{}
			if err := yaml.Unmarshal([]byte(composeYaml), &compose); err == nil {
				if services, ok := compose["services"].(map[string]interface{}); ok {
					// Get first service to detect port
					for _, serviceData := range services {
						if service, ok := serviceData.(map[string]interface{}); ok {
							// Check for exposed port or port mapping
							if ports, ok := service["ports"].([]interface{}); ok && len(ports) > 0 {
								// Try to extract port from first port mapping
								if portStr, ok := ports[0].(string); ok {
									// Format: "host:container" or just "container"
									parts := strings.Split(portStr, ":")
									if len(parts) >= 2 {
										if p, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
											targetPort = p
										}
									} else if len(parts) == 1 {
										if p, err := strconv.Atoi(parts[0]); err == nil {
											targetPort = p
										}
									}
								}
							} else if expose, ok := service["expose"].([]interface{}); ok && len(expose) > 0 {
								// Check exposed ports
								if portStr, ok := expose[0].(string); ok {
									if p, err := strconv.Atoi(portStr); err == nil {
										targetPort = p
									}
								}
							}
							break // Only check first service for default
						}
					}
				}
			}

			// Only create default routing if we found a port from compose file
			if targetPort > 0 {
				defaultRouting := &database.DeploymentRouting{
					ID:              defaultRoutingID,
					DeploymentID:    deploymentID,
					Domain:          "", // Domain can be set later through routing UI
					ServiceName:     "default",
					TargetPort:      targetPort,
					Protocol:        "http",
					SSLEnabled:      false, // Default to no SSL for HTTP protocol
					SSLCertResolver: "letsencrypt",
					Middleware:      "{}",
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}

				if upsertErr := database.UpsertDeploymentRouting(defaultRouting); upsertErr != nil {
					logger.Warn("[DeploymentManager] Failed to create default routing: %v", upsertErr)
				} else {
					routings = []database.DeploymentRouting{*defaultRouting}
					logger.Info("[DeploymentManager] Created default routing for compose deployment %s (port: %d)", deploymentID, targetPort)
				}
			} else {
				logger.Warn("[DeploymentManager] Could not detect port from compose file for deployment %s - routing must be configured manually", deploymentID)
			}
		}
	}

	// Get plan limits for this deployment
	maxMemoryBytes, maxCPUCores, err := dm.getPlanLimitsForDeployment(deploymentID)
	if err != nil {
		logger.Warn("[DeploymentManager] Failed to get plan limits for deployment %s: %v (continuing without limits)", deploymentID, err)
	}

	// Inject plan limits into compose file (if limits are set)
	if maxMemoryBytes > 0 || maxCPUCores > 0 {
		limitedYaml, err := dm.injectPlanLimitsIntoCompose(sanitizedYaml, deploymentID, maxMemoryBytes, maxCPUCores)
		if err != nil {
			logger.Warn("[DeploymentManager] Failed to inject plan limits into compose file: %v (using original YAML)", err)
		} else {
			sanitizedYaml = limitedYaml
			logger.Info("[DeploymentManager] Applied plan limits to compose file: memory=%d bytes, cpu=%d cores", maxMemoryBytes, maxCPUCores)
		}
	}

	// Inject Traefik labels into compose file based on routing rules
	labeledYaml, err := dm.injectTraefikLabelsIntoCompose(sanitizedYaml, deploymentID, routings)
	if err != nil {
		logger.Warn("[DeploymentManager] Failed to inject Traefik labels into compose YAML for deployment %s: %v. Using sanitized YAML without labels.", deploymentID, err)
		labeledYaml = sanitizedYaml
	} else {
		logger.Info("[DeploymentManager] Injected Traefik labels into compose YAML for deployment %s (found %d routing rules)", deploymentID, len(routings))
		// Log a sample of the labels for debugging
		if len(routings) > 0 {
			logger.Debug("[DeploymentManager] Sample Traefik labels for deployment %s: traefik.enable=true, cloud.obiente.traefik=true", deploymentID)
		}
		sanitizedYaml = labeledYaml
	}

	// Create persistent directory for compose file
	// Try multiple possible locations, fallback to temp if needed
	var deployDir string
	possibleDirs := []string{
		"/var/lib/obiente/deployments",
		"/var/obiente/tmp/obiente-deployments",
		"/tmp/obiente-deployments",
		os.TempDir(),
	}

	for _, baseDir := range possibleDirs {
		testDir := filepath.Join(baseDir, deploymentID)
		if err := os.MkdirAll(testDir, 0755); err == nil {
			// Verify we can write to it
			testFile := filepath.Join(testDir, ".test")
			if err := os.WriteFile(testFile, []byte("test"), 0644); err == nil {
				os.Remove(testFile)
				deployDir = testDir
				break
			}
		}
	}

	if deployDir == "" {
		return fmt.Errorf("failed to create deployment directory in any of the attempted locations")
	}

	composeFile := filepath.Join(deployDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, []byte(sanitizedYaml), 0644); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	// Set project name to deployment ID to avoid conflicts
	// Note: Docker Compose normalizes project names (lowercase, etc.), but we'll use the label to find containers
	projectName := fmt.Sprintf("deploy-%s", deploymentID)

	// Check if we're in Swarm mode using ENABLE_SWARM environment variable
	isSwarmMode := isSwarmModeEnabled()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if isSwarmMode {
		// In Swarm mode, we MUST use docker stack deploy to create Swarm services
		// docker compose creates plain containers, not Swarm services, which Traefik can't discover
		logger.Info("[DeploymentManager] Deploying in Swarm mode (ENABLE_SWARM=true) - using docker stack deploy to create Swarm services")

		// Ensure registry authentication is set up for multi-node deployments
		// This ensures credentials are available for --with-registry-auth=true
		registryURL := os.Getenv("REGISTRY_URL")
		if registryURL == "" {
			domain := os.Getenv("DOMAIN")
			if domain == "" {
				domain = "obiente.cloud"
			}
			registryURL = fmt.Sprintf("https://registry.%s", domain)
		} else {
			// Handle unexpanded docker-compose variables
			if strings.Contains(registryURL, "${DOMAIN") {
				domain := os.Getenv("DOMAIN")
				if domain == "" {
					domain = "obiente.cloud"
				}
				registryURL = strings.ReplaceAll(registryURL, "${DOMAIN:-obiente.cloud}", domain)
				registryURL = strings.ReplaceAll(registryURL, "${DOMAIN}", domain)
			}
		}

		registryUsername := os.Getenv("REGISTRY_USERNAME")
		registryPassword := os.Getenv("REGISTRY_PASSWORD")
		if registryUsername == "" {
			registryUsername = "obiente"
		}

		// docker login needs just the hostname, not the full URL with protocol
		registryHost := strings.TrimPrefix(registryURL, "https://")
		registryHost = strings.TrimPrefix(registryHost, "http://")

		if registryPassword != "" {
			logger.Info("[DeploymentManager] Authenticating with registry %s to enable multi-node image pulls...", registryHost)
			loginCmd := exec.CommandContext(ctx, "docker", "login", registryHost, "-u", registryUsername, "-p", registryPassword)
			var loginStderr bytes.Buffer
			loginCmd.Stderr = &loginStderr
			if loginErr := loginCmd.Run(); loginErr != nil {
				logger.Warn("[DeploymentManager] Failed to authenticate with registry %s: %v (stderr: %s). Worker nodes may fail to pull images.", registryHost, loginErr, loginStderr.String())
				// Don't fail here - the stack deploy might still work if credentials are cached
			} else {
				logger.Info("[DeploymentManager] Successfully authenticated with registry %s. Credentials will be passed to worker nodes.", registryHost)
			}
		} else {
			logger.Warn("[DeploymentManager] REGISTRY_PASSWORD not set - worker nodes may fail to pull private images")
		}

		// First, try to remove existing stack (ignore errors if it doesn't exist)
		rmArgs := []string{"stack", "rm", projectName}
		rmCmd := exec.CommandContext(ctx, "docker", rmArgs...)
		rmCmd.Run() // Ignore errors - stack might not exist

		// Wait a moment for stack removal to complete
		time.Sleep(2 * time.Second)

		// Deploy as a Swarm stack - this creates Swarm services that Traefik can discover
		// Use --with-registry-auth=true to pass registry credentials to Swarm
		args := []string{"stack", "deploy", "-c", composeFile, "--with-registry-auth=true", projectName}
		logger.Info("[DeploymentManager] Deploying stack %s with docker stack deploy (creates Swarm services)", projectName)

		cmd := exec.CommandContext(ctx, "docker", args...)
		cmd.Dir = deployDir
		cmd.Stderr = &stderr
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			errorOutput := stderr.String()
			stdOutput := stdout.String()
			logger.Error("[DeploymentManager] Failed to deploy stack for deployment %s: %v\nStderr: %s\nStdout: %s", deploymentID, err, errorOutput, stdOutput)
			return fmt.Errorf("failed to deploy stack: %w\nStderr: %s\nStdout: %s", err, errorOutput, stdOutput)
		}
	} else {
		// In non-Swarm mode, use docker compose (creates containers)
		args := []string{"compose", "-p", projectName, "-f", composeFile, "up", "-d", "--force-recreate", "--remove-orphans"}
		logger.Info("[DeploymentManager] Deploying in non-Swarm mode (ENABLE_SWARM=false or not set) - will force recreate containers with updated labels")

		cmd := exec.CommandContext(ctx, "docker", args...)
		cmd.Dir = deployDir
		cmd.Stderr = &stderr
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			errorOutput := stderr.String()
			stdOutput := stdout.String()
			logger.Error("[DeploymentManager] Failed to deploy compose file for deployment %s: %v\nStderr: %s\nStdout: %s", deploymentID, err, errorOutput, stdOutput)
			return fmt.Errorf("failed to deploy compose file: %w\nStderr: %s\nStdout: %s", err, errorOutput, stdOutput)
		}
	}

	stdOutput := stdout.String()
	if isSwarmMode {
		logger.Info("[DeploymentManager] Docker stack deploy output for deployment %s:\n%s", deploymentID, stdOutput)
		logger.Info("[DeploymentManager] Successfully deployed stack %s (creates Swarm services)", projectName)

		// In Swarm mode, verify that services have the correct labels
		// Traefik reads from service labels, not container labels
		logger.Info("[DeploymentManager] Verifying service labels in Swarm mode...")
		// List services in the stack
		listArgs := []string{"stack", "services", projectName, "--format", "{{.Name}}"}
		listCmd := exec.CommandContext(ctx, "docker", listArgs...)
		var listStdout bytes.Buffer
		listCmd.Stdout = &listStdout
		if err := listCmd.Run(); err == nil {
			services := strings.TrimSpace(listStdout.String())
			if services != "" {
				serviceList := strings.Split(services, "\n")
				for _, fullServiceName := range serviceList {
					fullServiceName = strings.TrimSpace(fullServiceName)
					if fullServiceName == "" {
						continue
					}
					// Check service deploy labels (where Traefik reads from in Swarm mode)
					inspectArgs := []string{"service", "inspect", fullServiceName, "--format", "{{json .Spec.TaskTemplate.ContainerSpec.Labels}}"}
					inspectCmd := exec.CommandContext(ctx, "docker", inspectArgs...)
					var inspectStdout bytes.Buffer
					inspectCmd.Stdout = &inspectStdout
					if err := inspectCmd.Run(); err == nil {
						labelsJSON := strings.TrimSpace(inspectStdout.String())
						logger.Debug("[DeploymentManager] Service %s container labels: %s", fullServiceName, labelsJSON)

						// Also check deploy labels (where we set them)
						deployInspectArgs := []string{"service", "inspect", fullServiceName, "--format", "{{json .Spec.Labels}}"}
						deployInspectCmd := exec.CommandContext(ctx, "docker", deployInspectArgs...)
						var deployInspectStdout bytes.Buffer
						deployInspectCmd.Stdout = &deployInspectStdout
						if err := deployInspectCmd.Run(); err == nil {
							deployLabelsJSON := strings.TrimSpace(deployInspectStdout.String())
							logger.Debug("[DeploymentManager] Service %s deploy labels: %s", fullServiceName, deployLabelsJSON)

							// In Swarm mode, Traefik reads from deploy.labels (service labels)
							// Check if cloud.obiente.traefik label exists in deploy labels
							if strings.Contains(deployLabelsJSON, "cloud.obiente.traefik") || strings.Contains(labelsJSON, "cloud.obiente.traefik") {
								logger.Info("[DeploymentManager] Service %s has Traefik labels - Traefik should discover it", fullServiceName)
							} else {
								logger.Warn("[DeploymentManager] Service %s is missing cloud.obiente.traefik label in deploy labels - Traefik may not discover it", fullServiceName)
							}
						}
					}
				}
			}
		}
	} else {
		logger.Info("[DeploymentManager] Docker compose up output for deployment %s:\n%s", deploymentID, stdOutput)
		logger.Info("[DeploymentManager] Successfully deployed compose file for deployment %s (project: %s)", deploymentID, projectName)
	}

	// Wait a moment for containers to be fully created and started
	time.Sleep(1 * time.Second)

	// List containers created by this compose project and register them
	return dm.registerComposeContainers(ctx, deploymentID, projectName)
}

// registerComposeContainers finds containers created by a compose project and registers them
func (dm *DeploymentManager) registerComposeContainers(ctx context.Context, deploymentID string, projectName string) error {
	// Check if we're in Swarm mode
	isSwarmMode := isSwarmModeEnabled()

	// containers will be initialized from ContainerList - type inferred from return value
	// We initialize with an empty list to establish the type, then reassign in branches
	containers, _ := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{All: true, Filters: filters.NewArgs()})
	containers = containers[:0] // Clear the list but keep the type

	if isSwarmMode {
		// In Swarm mode, containers are created by services in the stack
		// List containers with the deployment ID label (set by our Traefik label injection)
		filterArgs := filters.NewArgs()
		filterArgs.Add("label", fmt.Sprintf("cloud.obiente.deployment_id=%s", deploymentID))

		// Assign to containers - type already established
		containers, _ = dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
			All:     true,
			Filters: filterArgs,
		})

		// Fallback: try listing containers by stack name
		if len(containers) == 0 {
			logger.Info("[DeploymentManager] No containers found with deployment ID label, trying stack name %s", projectName)
			// In Swarm, containers have com.docker.swarm.service.name label
			// Service names are in format: {stack}_{service}
			allContainers, err := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{All: true})
			if err == nil {
				for _, cnt := range allContainers {
					serviceName := cnt.Labels["com.docker.swarm.service.name"]
					if strings.HasPrefix(serviceName, projectName+"_") || strings.HasPrefix(serviceName, strings.ToLower(projectName)+"_") {
						containers = append(containers, cnt)
					}
				}
			}
		}
	} else {
		// In non-Swarm mode, list containers with the compose project label
		// Note: Docker Compose may normalize the project name (e.g., lowercase), so we try both
		filterArgs := filters.NewArgs()
		filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

		containers, err := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
			All:     true,
			Filters: filterArgs,
		})
		if err != nil {
			return fmt.Errorf("failed to list compose containers: %w", err)
		}

		// If no containers found with exact project name, try lowercase version (Docker Compose normalization)
		if len(containers) == 0 {
			logger.Info("[DeploymentManager] No containers found with project name %s, trying lowercase version", projectName)
			filterArgsLower := filters.NewArgs()
			filterArgsLower.Add("label", fmt.Sprintf("com.docker.compose.project=%s", strings.ToLower(projectName)))

			containers, err = dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
				All:     true,
				Filters: filterArgsLower,
			})
			if err != nil {
				logger.Info("[DeploymentManager] Failed to list containers with lowercase project name: %v", err)
			}
		}
	}

	// Also try listing all containers with compose labels and filter manually (fallback)
	if len(containers) == 0 {
		logger.Info("[DeploymentManager] Still no containers found, listing all containers with compose labels")
		allFilterArgs := filters.NewArgs()
		allFilterArgs.Add("label", "com.docker.compose.project")

		allContainers, err := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
			All:     true,
			Filters: allFilterArgs,
		})
		if err == nil {
			// Filter manually by checking labels
			for _, cnt := range allContainers {
				if projectLabel := cnt.Labels["com.docker.compose.project"]; projectLabel == projectName || projectLabel == strings.ToLower(projectName) {
					containers = append(containers, cnt)
					logger.Info("[DeploymentManager] Found container %s with project label: %s", cnt.ID[:12], projectLabel)
				}
			}
		}
	}

	if len(containers) == 0 {
		logger.Info("[DeploymentManager] WARNING: No containers found for compose project %s (deployment %s). "+
			"This might indicate the compose file failed to create containers. Checking all containers...", projectName, deploymentID)

		// Last resort: list all containers to see what exists
		allContainers, err := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{All: true})
		if err == nil {
			logger.Info("[DeploymentManager] Total containers on system: %d", len(allContainers))
			for i, cnt := range allContainers {
				if i < 5 { // Log first 5 containers for debugging
					projectLabel := cnt.Labels["com.docker.compose.project"]
					logger.Info("[DeploymentManager] Container %s: compose project label = '%s'", cnt.ID[:12], projectLabel)
				}
			}
		}

		return fmt.Errorf("no containers found for compose project %s", projectName)
	}

	logger.Info("[DeploymentManager] Found %d container(s) for compose project %s", len(containers), projectName)

	// Get routing rules
	routings, _ := database.GetDeploymentRoutings(deploymentID)

	var runningCount int
	for _, cnt := range containers {
		// Verify container is actually running by inspecting it
		containerInfo, err := dm.dockerClient.ContainerInspect(ctx, cnt.ID)
		if err != nil {
			logger.Warn("[DeploymentManager] Failed to inspect container %s: %v", cnt.ID[:12], err)
			continue
		}

		// Determine actual container status
		containerStatus := "stopped"
		if containerInfo.State.Running {
			containerStatus = "running"
			runningCount++
		}

		// Extract service name from container labels
		serviceName := cnt.Labels["com.docker.compose.service"]
		if serviceName == "" {
			serviceName = "default"
		}

		// Determine public port from routing (required - no default)
		publicPort := 0
		for _, routing := range routings {
			if routing.ServiceName == serviceName || (serviceName == "default" && routing.ServiceName == "") {
				if routing.TargetPort > 0 {
					publicPort = routing.TargetPort
					break
				}
			}
		}
		// If no exact match, use first routing's target port
		if publicPort == 0 && len(routings) > 0 && routings[0].TargetPort > 0 {
			publicPort = routings[0].TargetPort
		}

		// Extract port from container info if available
		if len(cnt.Ports) > 0 {
			publicPort = int(cnt.Ports[0].PublicPort)
		}

		// Register deployment location with actual status
		location := &database.DeploymentLocation{
			ID:           fmt.Sprintf("loc-%s-%s", deploymentID, cnt.ID[:12]),
			DeploymentID: deploymentID,
			NodeID:       dm.nodeID,
			NodeHostname: dm.nodeHostname,
			ContainerID:  cnt.ID,
			Status:       containerStatus,
			Port:         publicPort,
			Domain:       "", // Will be set from deployment config
			HealthStatus: "unknown",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := dm.registry.RegisterDeployment(ctx, location); err != nil {
			logger.Warn("[DeploymentManager] Failed to register compose container %s: %v", cnt.ID[:12], err)
		} else {
			logger.Info("[DeploymentManager] Registered compose container %s (service: %s, status: %s) for deployment %s",
				cnt.ID[:12], serviceName, containerStatus, deploymentID)
		}
	}

	if runningCount == 0 {
		return fmt.Errorf("no running containers found for compose project %s (%d containers found but all are stopped)", projectName, len(containers))
	}

	logger.Info("[DeploymentManager] Successfully registered %d running container(s) for deployment %s", runningCount, deploymentID)
	return nil
}

// StopComposeDeployment stops containers created by a compose file using docker compose down
func (dm *DeploymentManager) StopComposeDeployment(ctx context.Context, deploymentID string) error {
	logger.Info("[DeploymentManager] Stopping compose deployment %s", deploymentID)

	projectName := fmt.Sprintf("deploy-%s", deploymentID)

	// Check if we're in Swarm mode
	enableSwarm := os.Getenv("ENABLE_SWARM")
	isSwarmMode := enableSwarm == "true" || enableSwarm == "1"

	if isSwarmMode {
		// In Swarm mode, use docker stack rm to remove the stack
		logger.Info("[DeploymentManager] Stopping Swarm stack %s", projectName)
		cmd := exec.CommandContext(ctx, "docker", "stack", "rm", projectName)
		var stderr bytes.Buffer
		var stdout bytes.Buffer
		cmd.Stderr = &stderr
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			errorOutput := stderr.String()
			stdOutput := stdout.String()
			// Ignore error if stack doesn't exist
			if strings.Contains(errorOutput, "not found") || strings.Contains(errorOutput, "does not exist") {
				logger.Info("[DeploymentManager] Stack %s does not exist, nothing to stop", projectName)
				return nil
			}
			logger.Error("[DeploymentManager] Failed to stop Swarm stack %s: %v\nStderr: %s\nStdout: %s", projectName, err, errorOutput, stdOutput)
			return fmt.Errorf("failed to stop Swarm stack: %w\nStderr: %s\nStdout: %s", err, errorOutput, stdOutput)
		}

		logger.Info("[DeploymentManager] Successfully stopped Swarm stack %s", projectName)
		return nil
	}

	// In non-Swarm mode, use docker compose down
	// Find compose file directory using the same logic as DeployComposeFile
	var deployDir string
	possibleDirs := []string{
		"/var/lib/obiente/deployments",
		"/var/obiente/tmp/obiente-deployments",
		"/tmp/obiente-deployments",
		os.TempDir(),
	}

	for _, baseDir := range possibleDirs {
		testDir := filepath.Join(baseDir, deploymentID)
		composeFile := filepath.Join(testDir, "docker-compose.yml")
		// Check if compose file exists in this directory
		if _, err := os.Stat(composeFile); err == nil {
			deployDir = testDir
			break
		}
	}

	if deployDir == "" {
		// Fallback: if we can't find the compose file, try to stop by project name
		// This handles cases where the directory was cleaned up but containers still exist
		logger.Info("[DeploymentManager] Compose file not found for deployment %s, falling back to container-based stop", deploymentID)
		return dm.stopComposeContainersByLabel(ctx, projectName)
	}

	composeFile := filepath.Join(deployDir, "docker-compose.yml")

	// Use docker compose down to stop all containers in the project
	cmd := exec.CommandContext(ctx, "docker", "compose", "-p", projectName, "-f", composeFile, "down")
	cmd.Dir = deployDir
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		errorOutput := stderr.String()
		stdOutput := stdout.String()
		logger.Error("[DeploymentManager] Failed to stop compose deployment %s: %v\nStderr: %s\nStdout: %s", deploymentID, err, errorOutput, stdOutput)
		// Fallback to individual container stop if compose down fails
		logger.Info("[DeploymentManager] Falling back to container-based stop for deployment %s", deploymentID)
		return dm.stopComposeContainersByLabel(ctx, projectName)
	}

	stdOutput := stdout.String()
	logger.Info("[DeploymentManager] Docker compose down output for deployment %s:\n%s", deploymentID, stdOutput)
	logger.Info("[DeploymentManager] Successfully stopped compose deployment %s (project: %s)", deploymentID, projectName)

	return nil
}

// stopComposeContainersByLabel stops containers by compose project label (fallback method)
func (dm *DeploymentManager) stopComposeContainersByLabel(ctx context.Context, projectName string) error {
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

	containers, err := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return fmt.Errorf("failed to list compose containers: %w", err)
	}

	if len(containers) == 0 {
		logger.Info("[DeploymentManager] No containers found for project %s", projectName)
		return nil
	}

	for _, cnt := range containers {
		timeout := 30 * time.Second
		if err := dm.dockerHelper.StopContainer(ctx, cnt.ID, timeout); err != nil {
			logger.Info("[DeploymentManager] Failed to stop compose container %s: %v", cnt.ID[:12], err)
		} else {
			logger.Info("[DeploymentManager] Stopped compose container %s", cnt.ID[:12])
		}
	}

	return nil
}

// RemoveComposeDeployment removes containers created by a compose file
func (dm *DeploymentManager) RemoveComposeDeployment(ctx context.Context, deploymentID string) error {
	logger.Info("[DeploymentManager] Removing compose deployment %s", deploymentID)

	projectName := fmt.Sprintf("deploy-%s", deploymentID)

	// Find containers by project label
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

	containers, err := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return fmt.Errorf("failed to list compose containers: %w", err)
	}

	for _, cnt := range containers {
		// SECURITY: Verify container was created by our API
		if cnt.Labels["cloud.obiente.managed"] != "true" {
			logger.Error("[DeploymentManager] SECURITY: Refusing to delete compose container %s: not managed by Obiente Cloud (missing cloud.obiente.managed=true label)", cnt.ID[:12])
			continue
		}

		// Stop first
		timeout := 10 * time.Second
		_ = dm.dockerHelper.StopContainer(ctx, cnt.ID, timeout)

		// Remove
		if err := dm.dockerHelper.RemoveContainer(ctx, cnt.ID, true); err != nil {
			logger.Info("[DeploymentManager] Failed to remove compose container %s: %v", cnt.ID[:12], err)
		} else {
			logger.Info("[DeploymentManager] Removed compose container %s", cnt.ID[:12])
			// Unregister
			_ = dm.registry.UnregisterDeployment(ctx, cnt.ID)
		}
	}

	// Clean up volumes and deployment data
	dm.cleanupDeploymentData(deploymentID)

	return nil
}

// cleanupDeploymentData removes all volumes and data directories for a deployment
