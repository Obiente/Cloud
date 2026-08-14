package orchestrator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/platform"
	"github.com/obiente/cloud/apps/shared/pkg/utils"

	"github.com/moby/moby/client"
	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v3"
)

// Compose operations for deployments

const legacyPreviewIngressNetworkName = "obiente-preview-ingress"

const legacyProjectRootMetadataFile = ".obiente-legacy-relative-project-root-v1"

func PreviewIngressNetworkNameForDeployment(deploymentID string) string {
	return fmt.Sprintf("deployment-%s", deploymentID)
}

func composeUpArgs(projectName, composeFile string) []string {
	return []string{"compose", "-p", projectName, "-f", composeFile, "up", "-d", "--build", "--pull", "always", "--force-recreate", "--remove-orphans"}
}

func stackDeployArgs(projectName, composeFile string) []string {
	return []string{"stack", "deploy", "-c", composeFile, "--with-registry-auth=true", "--resolve-image", "always", "--prune", projectName}
}

func stackRollbackRestoresVolumePreparation(serviceNames []string, err error) bool {
	return len(serviceNames) == 1 && RollbackPreserved(err)
}

func (dm *DeploymentManager) waitForSwarmStackConverged(ctx context.Context, deploymentID, projectName string) (bool, error) {
	listCmd := exec.CommandContext(ctx, "docker", "stack", "services", projectName, "--format", "{{.Name}}")
	output, err := listCmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("list services while waiting for stack %s: %w (%s)", projectName, err, strings.TrimSpace(string(output)))
	}
	serviceNames := strings.Fields(string(output))
	if len(serviceNames) == 0 {
		return false, fmt.Errorf("stack %s has no services after deployment", projectName)
	}
	for _, serviceName := range serviceNames {
		if err := dm.waitForSwarmStackServiceConverged(ctx, deploymentID, serviceName); err != nil {
			return stackRollbackRestoresVolumePreparation(serviceNames, err), fmt.Errorf("wait for stack service %s: %w", serviceName, err)
		}
	}
	return false, nil
}

func (dm *DeploymentManager) DeployComposeFile(ctx context.Context, deploymentID string, composeYaml string) error {
	return dm.deployComposeFile(ctx, deploymentID, composeYaml, "")
}

func (dm *DeploymentManager) RestartComposeFile(ctx context.Context, deploymentID string, composeYaml string) error {
	if !utils.IsSwarmModeEnabled() {
		// composeUpArgs already includes --force-recreate.
		return dm.DeployComposeFile(ctx, deploymentID, composeYaml)
	}

	projectName := fmt.Sprintf("deploy-%s", deploymentID)
	beforeSpecs, err := swarmStackServiceSpecs(ctx, projectName, true)
	if err != nil {
		return fmt.Errorf("inspect services before Compose restart: %w", err)
	}
	if err := dm.DeployComposeFile(ctx, deploymentID, composeYaml); err != nil {
		return err
	}
	afterSpecs, err := swarmStackServiceSpecs(ctx, projectName, false)
	if err != nil {
		return fmt.Errorf("inspect services after Compose restart deployment: %w", err)
	}
	forcedRestart := false
	for _, serviceName := range unchangedSwarmStackServices(beforeSpecs, afterSpecs) {
		forceCmd := exec.CommandContext(ctx, "docker", "service", "update", "--detach=true", "--force", serviceName)
		forceOutput, forceErr := forceCmd.CombinedOutput()
		if forceErr != nil {
			return fmt.Errorf("force restart service %s: %w (%s)", serviceName, forceErr, strings.TrimSpace(string(forceOutput)))
		}
		if waitErr := dm.waitForSwarmStackServiceConverged(ctx, deploymentID, serviceName); waitErr != nil {
			return fmt.Errorf("wait for forced restart of service %s: %w", serviceName, waitErr)
		}
		forcedRestart = true
	}
	if forcedRestart {
		// DeployComposeFile registered the pre-force tasks. Refresh the stable
		// service rows after every requested replacement has converged.
		return dm.registerComposeContainers(ctx, deploymentID, projectName)
	}
	return nil
}

func unchangedSwarmStackServices(before, after map[string]string) []string {
	serviceNames := make([]string, 0, len(before))
	for serviceName, beforeSpec := range before {
		if afterSpec, stillExists := after[serviceName]; stillExists && afterSpec == beforeSpec {
			serviceNames = append(serviceNames, serviceName)
		}
	}
	sort.Strings(serviceNames)
	return serviceNames
}

func swarmStackServiceSpecs(ctx context.Context, projectName string, allowMissing bool) (map[string]string, error) {
	listCmd := exec.CommandContext(ctx, "docker", "stack", "services", projectName, "--format", "{{.Name}}")
	output, err := listCmd.CombinedOutput()
	if err != nil {
		if allowMissing && isMissingSwarmStackOutput(string(output)) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("list stack services: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	serviceNames := strings.Fields(string(output))
	if len(serviceNames) == 0 {
		if allowMissing {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("stack %s has no services", projectName)
	}
	specs := make(map[string]string, len(serviceNames))
	for _, serviceName := range serviceNames {
		inspectCmd := exec.CommandContext(ctx, "docker", "service", "inspect", serviceName, "--format", "{{json .Spec}}")
		inspectOutput, inspectErr := inspectCmd.CombinedOutput()
		if inspectErr != nil {
			return nil, fmt.Errorf("inspect specification for service %s: %w (%s)", serviceName, inspectErr, strings.TrimSpace(string(inspectOutput)))
		}
		spec := strings.TrimSpace(string(inspectOutput))
		if spec == "" {
			return nil, fmt.Errorf("inspect specification for service %s returned no data", serviceName)
		}
		specs[serviceName] = spec
	}
	return specs, nil
}

func isMissingSwarmStackOutput(output string) bool {
	return strings.Contains(strings.ToLower(output), "nothing found in stack")
}

// DeployIsolatedComposeFile routes an untrusted preview through a dedicated
// ingress network that contains Traefik but no Obiente control-plane service.
func (dm *DeploymentManager) DeployIsolatedComposeFile(ctx context.Context, deploymentID string, composeYaml string) error {
	ingressNetworkName := PreviewIngressNetworkNameForDeployment(deploymentID)
	if err := dm.ensurePreviewIngressNetwork(ctx, ingressNetworkName); err != nil {
		return fmt.Errorf("preview ingress network is required: %w", err)
	}
	return dm.deployComposeFile(ctx, deploymentID, composeYaml, ingressNetworkName)
}

func (dm *DeploymentManager) deployComposeFile(ctx context.Context, deploymentID string, composeYaml string, ingressNetworkName string) (retErr error) {
	logger.Info("[DeploymentManager] Deploying compose file for deployment %s", deploymentID)
	isSwarmMode := utils.IsSwarmModeEnabled()
	var sanitizer *ComposeSanitizer
	var releaseVolumeLock func()
	volumePreparationCommitted := false
	defer func() {
		if retErr != nil && !volumePreparationCommitted && sanitizer != nil {
			if rollbackErr := sanitizer.rollbackVolumePreparation(); rollbackErr != nil {
				retErr = fmt.Errorf("%w; restore previous volume permissions: %v", retErr, rollbackErr)
			}
		}
		if releaseVolumeLock != nil {
			releaseVolumeLock()
		}
	}()

	// Ensure per-deployment network exists before deploying
	// This provides isolation from other deployments (while services stay on obiente-network for Traefik discovery)
	deploymentNetworkName := fmt.Sprintf("deployment-%s", deploymentID)
	if err := dm.ensureDeploymentNetwork(ctx, deploymentNetworkName); err != nil {
		return fmt.Errorf("deployment network is required but could not be created: %w", err)
	}

	// Sanitize compose file for security (transform volumes, remove host ports, etc.)
	if deploymentID != "" && sanitizeVolumeName(deploymentID) == deploymentID {
		var lockErr error
		releaseVolumeLock, lockErr = acquireDeploymentVolumeLock(ctx, deploymentID)
		if lockErr != nil {
			return fmt.Errorf("wait for deployment volume preparation: %w", lockErr)
		}
	}
	sanitizer = NewComposeSanitizer(deploymentID)
	if sanitizer.GetSafeBaseDir() != "" {
		legacyProjectRoot, err := storedComposeProvesLegacyProjectRoot(deploymentID, sanitizer.GetSafeBaseDir())
		if err != nil {
			return fmt.Errorf("inspect persisted Compose volume metadata: %w", err)
		}
		sanitizer.legacyProjectRoot = legacyProjectRoot
	}
	if isSwarmMode {
		sanitizer.swarmVolumeNodeID = dm.nodeID
	}
	sanitizedYaml, err := sanitizer.SanitizeComposeYAML(composeYaml)
	if err != nil {
		return fmt.Errorf("refusing to deploy compose YAML that could not be sanitized: %w", err)
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
	labeledYaml, err := dm.injectTraefikLabelsIntoCompose(sanitizedYaml, deploymentID, routings, ingressNetworkName)
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

	// If there are routing rules, connect routed services to the selected Traefik
	// ingress network. Isolated previews reuse their deployment network here.
	if len(routings) > 0 {
		routedYaml, err := dm.addTraefikNetworkToRoutedServices(sanitizedYaml, routings, ingressNetworkName)
		if err != nil {
			logger.Warn("[DeploymentManager] Failed to add Traefik network to routed services: %v. Continuing with current YAML.", err)
		} else {
			sanitizedYaml = routedYaml
			logger.Info("[DeploymentManager] Connected %d routed services to the Traefik ingress network", len(routings))
		}
	}

	// Keep the existing Swarm services during updates. Removing the stack first
	// causes an avoidable outage while the replacement tasks are starting.
	if utils.IsSwarmModeEnabled() {
		rollingYaml, err := injectSwarmRollingUpdatePolicy(sanitizedYaml)
		if err != nil {
			return fmt.Errorf("failed to configure Swarm rolling update policy: %w", err)
		}
		sanitizedYaml = rollingYaml
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
	if sanitizer.legacyProjectRoot {
		if err := persistLegacyProjectRootMetadata(deployDir, sanitizer.GetSafeBaseDir()); err != nil {
			return fmt.Errorf("persist legacy project-root volume metadata: %w", err)
		}
	}

	composeFile := filepath.Join(deployDir, "docker-compose.yml")
	if err := os.WriteFile(composeFile, []byte(sanitizedYaml), 0644); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	// Set project name to deployment ID to avoid conflicts
	// Note: Docker Compose normalizes project names (lowercase, etc.), but we'll use the label to find containers
	projectName := fmt.Sprintf("deploy-%s", deploymentID)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if isSwarmMode {
		// In Swarm mode, we MUST use docker stack deploy to create Swarm services
		// docker compose creates plain containers, not Swarm services, which Traefik can't discover
		logger.Info("[DeploymentManager] Deploying in Swarm mode (ENABLE_SWARM=true) - using docker stack deploy to create Swarm services")

		// Ensure registry authentication is set up for multi-node deployments
		// This ensures credentials are available for --with-registry-auth=true
		registryURL := platform.RegistryURL()

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

		// Deploy in place so Swarm can start replacement tasks before stopping old
		// ones according to each service's update_config.
		// Use --with-registry-auth=true to pass registry credentials to Swarm
		args := stackDeployArgs(projectName, composeFile)
		logger.Info("[DeploymentManager] Deploying stack %s with docker stack deploy (creates Swarm services)", projectName)

		cmd := exec.CommandContext(ctx, "docker", args...)
		cmd.Dir = deployDir
		cmd.Stderr = &stderr
		cmd.Stdout = &stdout

		beforeRuntime, err := swarmStackRuntimeFingerprint(ctx, projectName)
		if err != nil {
			return fmt.Errorf("inspect stack before deployment: %w", err)
		}
		if err := cmd.Run(); err != nil {
			// A failed stack deploy can still update a subset of services. Retain
			// broadened writable roots only when the daemon accepted observable
			// work, or when a post-failure inspection cannot prove that it did not.
			volumePreparationCommitted = deploymentRuntimeChangedAfterFailure(beforeRuntime, func(inspectCtx context.Context) (string, error) {
				return swarmStackRuntimeFingerprint(inspectCtx, projectName)
			})
			errorOutput := stderr.String()
			stdOutput := stdout.String()
			logger.Error("[DeploymentManager] Failed to deploy stack for deployment %s: %v\nStderr: %s\nStdout: %s", deploymentID, err, errorOutput, stdOutput)
			return fmt.Errorf("failed to deploy stack: %w\nStderr: %s\nStdout: %s", err, errorOutput, stdOutput)
		}
		volumePreparationCommitted = true
		rollbackRestoredPreviousRevision, err := dm.waitForSwarmStackConverged(ctx, deploymentID, projectName)
		if err != nil {
			if rollbackRestoredPreviousRevision {
				// A single-service stack that completed its rollback cannot have
				// another accepted revision depending on the prepared modes.
				volumePreparationCommitted = false
			}
			return fmt.Errorf("stack deployment did not converge: %w", err)
		}
	} else {
		// In non-Swarm mode, use docker compose (creates containers)
		args := composeUpArgs(projectName, composeFile)
		logger.Info("[DeploymentManager] Deploying in non-Swarm mode (ENABLE_SWARM=false or not set) - will rebuild buildable services, pull tagged images, and force recreate containers with updated labels")

		cmd := exec.CommandContext(ctx, "docker", args...)
		cmd.Dir = deployDir
		cmd.Stderr = &stderr
		cmd.Stdout = &stdout

		beforeRuntime, err := composeProjectRuntimeFingerprint(ctx, projectName)
		if err != nil {
			return fmt.Errorf("inspect Compose project before deployment: %w", err)
		}
		if err := cmd.Run(); err != nil {
			// Compose can recreate an early service before a later service fails.
			// Retain writable roots only after an observable container-generation
			// change, or when post-failure inspection cannot rule one out.
			volumePreparationCommitted = deploymentRuntimeChangedAfterFailure(beforeRuntime, func(inspectCtx context.Context) (string, error) {
				return composeProjectRuntimeFingerprint(inspectCtx, projectName)
			})
			errorOutput := stderr.String()
			stdOutput := stdout.String()
			logger.Error("[DeploymentManager] Failed to deploy compose file for deployment %s: %v\nStderr: %s\nStdout: %s", deploymentID, err, errorOutput, stdOutput)
			return fmt.Errorf("failed to deploy compose file: %w\nStderr: %s\nStdout: %s", err, errorOutput, stdOutput)
		}
		volumePreparationCommitted = true
	}
	if err := sanitizer.applyDeferredReadOnly(); err != nil {
		return fmt.Errorf("apply read-only Compose volume permissions: %w", err)
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

func deploymentRuntimeChangedAfterFailure(before string, inspect func(context.Context) (string, error)) bool {
	inspectCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	after, err := inspect(inspectCtx)
	if err != nil {
		// Without a reliable post-state, restoring permissions could break a
		// replacement that the daemon accepted before connectivity was lost.
		return true
	}
	return after != before
}

func composeProjectRuntimeFingerprint(ctx context.Context, projectName string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "--all", "--no-trunc", "--filter", "label=com.docker.compose.project="+projectName, "--format", "{{.ID}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("list project containers: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	containerIDs := strings.Fields(string(output))
	sort.Strings(containerIDs)
	return strings.Join(containerIDs, ","), nil
}

func swarmStackRuntimeFingerprint(ctx context.Context, projectName string) (string, error) {
	listCmd := exec.CommandContext(ctx, "docker", "stack", "services", projectName, "--format", "{{.Name}}")
	output, err := listCmd.CombinedOutput()
	if err != nil {
		if isMissingSwarmStackOutput(string(output)) {
			return "", nil
		}
		return "", fmt.Errorf("list stack services: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	serviceNames := strings.Fields(string(output))
	versions := make([]string, 0, len(serviceNames))
	for _, serviceName := range serviceNames {
		inspectCmd := exec.CommandContext(ctx, "docker", "service", "inspect", serviceName, "--format", "{{.ID}}\t{{.Version.Index}}")
		inspectOutput, inspectErr := inspectCmd.CombinedOutput()
		if inspectErr != nil {
			return "", fmt.Errorf("inspect stack service %s version: %w (%s)", serviceName, inspectErr, strings.TrimSpace(string(inspectOutput)))
		}
		versions = append(versions, serviceName+"\t"+strings.TrimSpace(string(inspectOutput)))
	}
	sort.Strings(versions)
	return strings.Join(versions, "\n"), nil
}

func storedComposeProvesLegacyProjectRoot(deploymentID, safeBaseDir string) (bool, error) {
	if deploymentID == "" || sanitizeVolumeName(deploymentID) != deploymentID {
		return false, fmt.Errorf("invalid deployment identifier")
	}
	if !filepath.IsAbs(safeBaseDir) {
		return false, fmt.Errorf("volume root %q is not absolute", safeBaseDir)
	}
	possibleDirs := []string{
		"/var/lib/obiente/deployments",
		"/var/obiente/tmp/obiente-deployments",
		"/tmp/obiente-deployments",
		os.TempDir(),
	}
	for _, baseDir := range possibleDirs {
		deployDir := filepath.Join(baseDir, deploymentID)
		metadata, found, err := readDeploymentFileNoFollow(deployDir, legacyProjectRootMetadataFile)
		if err != nil {
			return false, fmt.Errorf("read legacy project-root metadata in %s: %w", deployDir, err)
		}
		if found {
			expected := legacyProjectRootMetadataContents(safeBaseDir)
			if string(metadata) != expected {
				return false, fmt.Errorf("legacy project-root metadata in %s does not match the selected volume root", deployDir)
			}
			return true, nil
		}

		contents, found, err := readDeploymentFileNoFollow(deployDir, "docker-compose.yml")
		if err != nil {
			return false, fmt.Errorf("read persisted Compose file in %s: %w", deployDir, err)
		}
		if found {
			proven, proveErr := persistedComposeProvesLegacyProjectRoot(string(contents), safeBaseDir)
			if proveErr != nil {
				return false, fmt.Errorf("inspect persisted Compose file in %s: %w", deployDir, proveErr)
			}
			if proven {
				return true, nil
			}
		}
	}
	return false, nil
}

func legacyProjectRootMetadataContents(safeBaseDir string) string {
	return "legacy-relative-project-root-v1\n" + filepath.Clean(safeBaseDir) + "\n"
}

func readDeploymentFileNoFollow(deployDir, name string) ([]byte, bool, error) {
	dirFD, err := secureOpenDirectory(deployDir, false)
	if errors.Is(err, unix.ENOENT) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	defer unix.Close(dirFD)

	fileFD, err := unix.Openat(dirFD, name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(err, unix.ENOENT) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	file := os.NewFile(uintptr(fileFD), name)
	if file == nil {
		unix.Close(fileFD)
		return nil, false, fmt.Errorf("open %s", name)
	}
	defer file.Close()
	var stat unix.Stat_t
	if err := unix.Fstat(fileFD, &stat); err != nil {
		return nil, false, err
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG {
		return nil, false, fmt.Errorf("%s is not a regular file", name)
	}
	contents, err := io.ReadAll(file)
	if err != nil {
		return nil, false, err
	}
	return contents, true, nil
}

func persistLegacyProjectRootMetadata(deployDir, safeBaseDir string) error {
	expected := legacyProjectRootMetadataContents(safeBaseDir)
	dirFD, err := secureOpenDirectory(deployDir, false)
	if err != nil {
		return err
	}
	defer unix.Close(dirFD)

	fileFD, err := unix.Openat(dirFD, legacyProjectRootMetadataFile, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if errors.Is(err, unix.EEXIST) {
		contents, found, readErr := readDeploymentFileNoFollow(deployDir, legacyProjectRootMetadataFile)
		if readErr != nil {
			return readErr
		}
		if !found || string(contents) != expected {
			return fmt.Errorf("existing metadata does not match the selected volume root")
		}
		return nil
	}
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fileFD), legacyProjectRootMetadataFile)
	if file == nil {
		unix.Close(fileFD)
		return errors.Join(fmt.Errorf("open newly created metadata"), removeIncompleteLegacyProjectRootMetadata(dirFD))
	}
	if _, err := file.WriteString(expected); err != nil {
		closeErr := file.Close()
		return errors.Join(err, closeErr, removeIncompleteLegacyProjectRootMetadata(dirFD))
	}
	if err := file.Sync(); err != nil {
		closeErr := file.Close()
		return errors.Join(err, closeErr, removeIncompleteLegacyProjectRootMetadata(dirFD))
	}
	if err := file.Close(); err != nil {
		return errors.Join(err, removeIncompleteLegacyProjectRootMetadata(dirFD))
	}
	return unix.Fsync(dirFD)
}

func removeIncompleteLegacyProjectRootMetadata(dirFD int) error {
	err := unix.Unlinkat(dirFD, legacyProjectRootMetadataFile, 0)
	if errors.Is(err, unix.ENOENT) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("remove incomplete legacy project-root metadata: %w", err)
	}
	return nil
}

// registerComposeContainers finds containers created by a compose project and registers them
func (dm *DeploymentManager) registerComposeContainers(ctx context.Context, deploymentID string, projectName string) error {
	// Check if we're in Swarm mode
	isSwarmMode := utils.IsSwarmModeEnabled()

	// containers will be initialized from ContainerList - type inferred from return value
	// We initialize with an empty list to establish the type, then reassign in branches
	containersResult, _ := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{All: true, Filters: make(client.Filters)})
	containers := containersResult.Items
	containers = containers[:0] // Clear the list but keep the type

	if isSwarmMode {
		// In Swarm mode, containers are created by services in the stack
		// List containers with the deployment ID label (set by our Traefik label injection)
		filterArgs := make(client.Filters)
		filterArgs.Add("label", fmt.Sprintf("cloud.obiente.deployment_id=%s", deploymentID))

		// Assign to containers - type already established
		containersResult, _ := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
			All:     true,
			Filters: filterArgs,
		})
		containers = containersResult.Items

		// Fallback: try listing containers by stack name
		if len(containers) == 0 {
			logger.Info("[DeploymentManager] No containers found with deployment ID label, trying stack name %s", projectName)
			// In Swarm, containers have com.docker.swarm.service.name label
			// Service names are in format: {stack}_{service}
			allContainersResult, err := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{All: true})
			if err == nil {
				for _, cnt := range allContainersResult.Items {
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
		filterArgs := make(client.Filters)
		filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

		containersResult, err := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
			All:     true,
			Filters: filterArgs,
		})
		if err != nil {
			return fmt.Errorf("failed to list compose containers: %w", err)
		}
		containers = containersResult.Items

		// If no containers found with exact project name, try lowercase version (Docker Compose normalization)
		if len(containers) == 0 {
			logger.Info("[DeploymentManager] No containers found with project name %s, trying lowercase version", projectName)
			filterArgsLower := make(client.Filters)
			filterArgsLower.Add("label", fmt.Sprintf("com.docker.compose.project=%s", strings.ToLower(projectName)))

			containersResult, err = dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
				All:     true,
				Filters: filterArgsLower,
			})
			if err == nil {
				containers = containersResult.Items
			}
			if err != nil {
				logger.Info("[DeploymentManager] Failed to list containers with lowercase project name: %v", err)
			}
		}
	}

	// Also try listing all containers with compose labels and filter manually (fallback)
	if len(containers) == 0 {
		logger.Info("[DeploymentManager] Still no containers found, listing all containers with compose labels")
		allFilterArgs := make(client.Filters)
		allFilterArgs.Add("label", "com.docker.compose.project")

		allContainersResult, err := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
			All:     true,
			Filters: allFilterArgs,
		})
		if err == nil {
			// Filter manually by checking labels
			for _, cnt := range allContainersResult.Items {
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
		allContainersResult, err := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{All: true})
		if err == nil {
			logger.Info("[DeploymentManager] Total containers on system: %d", len(allContainersResult.Items))
			for i, cnt := range allContainersResult.Items {
				if i < 5 { // Log first 5 containers for debugging
					projectLabel := cnt.Labels["com.docker.compose.project"]
					logger.Info("[DeploymentManager] Container %s: compose project label = '%s'", cnt.ID[:12], projectLabel)
				}
			}
		}

		return fmt.Errorf("no containers found for compose project %s", projectName)
	}
	if isSwarmMode {
		// ContainerList is normally newest-first. Register oldest-first so the
		// stable service row ends on the latest replacement task even when
		// stopped task containers are still retained by the daemon.
		sort.SliceStable(containers, func(i, j int) bool {
			return containers[i].Created < containers[j].Created
		})
	}

	logger.Info("[DeploymentManager] Found %d container(s) for compose project %s", len(containers), projectName)

	// Get routing rules
	routings, _ := database.GetDeploymentRoutings(deploymentID)

	var runningCount int
	for _, cnt := range containers {
		// Verify container is actually running by inspecting it
		containerInfoResult, err := dm.dockerClient.ContainerInspect(ctx, cnt.ID, client.ContainerInspectOptions{})
		if err != nil {
			logger.Warn("[DeploymentManager] Failed to inspect container %s: %v", cnt.ID[:12], err)
			continue
		}
		containerInfo := containerInfoResult.Container

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
			ServiceID:    cnt.Labels["com.docker.swarm.service.id"],
			TaskID:       cnt.Labels["com.docker.swarm.task.id"],
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

	if runningCount == 0 && !isSwarmMode {
		return fmt.Errorf("no running containers found for compose project %s (%d containers found but all are stopped)", projectName, len(containers))
	}
	if runningCount == 0 {
		logger.Info("[DeploymentManager] Registered %d completed Swarm task container(s) for deployment %s", len(containers), deploymentID)
		return nil
	}

	logger.Info("[DeploymentManager] Successfully registered %d running container(s) for deployment %s", runningCount, deploymentID)
	return nil
}

// StopComposeDeployment stops containers created by a compose file using docker compose down
func (dm *DeploymentManager) StopComposeDeployment(ctx context.Context, deploymentID string) error {
	logger.Info("[DeploymentManager] Stopping compose deployment %s", deploymentID)

	projectName := fmt.Sprintf("deploy-%s", deploymentID)

	// Check if we're in Swarm mode using the shared helper function
	isSwarmMode := utils.IsSwarmModeEnabled()

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
	filterArgs := make(client.Filters)
	filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

	containersResult, err := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	containers := containersResult.Items
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
	// Stop the Compose project through the mode-aware path first. In Swarm this
	// runs `docker stack rm`; container-label cleanup alone cannot remove Swarm
	// services, which would otherwise recreate their tasks after the database
	// record has been deleted.
	if err := dm.StopComposeDeployment(ctx, deploymentID); err != nil {
		return fmt.Errorf("stop compose runtime before removal: %w", err)
	}

	projectName := fmt.Sprintf("deploy-%s", deploymentID)

	// Find containers by project label
	filterArgs := make(client.Filters)
	filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

	containersResult, err := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	containers := containersResult.Items
	if err != nil {
		return fmt.Errorf("failed to list compose containers: %w", err)
	}

	var removalErrors []error
	for _, cnt := range containers {
		// SECURITY: Verify container was created by our API
		if cnt.Labels["cloud.obiente.managed"] != "true" {
			logger.Error("[DeploymentManager] SECURITY: Refusing to delete compose container %s: not managed by Obiente Cloud (missing cloud.obiente.managed=true label)", cnt.ID[:12])
			removalErrors = append(removalErrors, fmt.Errorf("refused to remove unmanaged compose container %s", cnt.ID))
			continue
		}

		// Stop first
		timeout := 10 * time.Second
		_ = dm.dockerHelper.StopContainer(ctx, cnt.ID, timeout)

		// Remove
		if err := dm.dockerHelper.RemoveContainer(ctx, cnt.ID, true); err != nil {
			logger.Info("[DeploymentManager] Failed to remove compose container %s: %v", cnt.ID[:12], err)
			removalErrors = append(removalErrors, fmt.Errorf("remove compose container %s: %w", cnt.ID, err))
		} else {
			logger.Info("[DeploymentManager] Removed compose container %s", cnt.ID[:12])
			// Unregister
			if err := dm.registry.UnregisterDeployment(ctx, cnt.ID); err != nil {
				logger.Warn("[DeploymentManager] Failed to unregister compose container %s: %v", cnt.ID, err)
			}
		}
	}
	if err := errors.Join(removalErrors...); err != nil {
		return err
	}

	// Clean up volumes and deployment data
	dm.cleanupDeploymentData(deploymentID)

	return nil
}

// cleanupDeploymentData removes all volumes and data directories for a deployment
