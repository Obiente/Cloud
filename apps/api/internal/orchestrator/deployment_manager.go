package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"api/docker"
	"api/internal/database"
	"api/internal/logger"
	"api/internal/registry"

	"github.com/docker/go-connections/nat"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/filters"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"gopkg.in/yaml.v3"
)

// isSwarmModeEnabled checks if Swarm mode is enabled via ENABLE_SWARM environment variable
// Returns true if ENABLE_SWARM is set to "true", "1", "yes", "on", or any case-insensitive variant
func isSwarmModeEnabled() bool {
	enableSwarm := os.Getenv("ENABLE_SWARM")
	if enableSwarm == "" {
		return false
	}
	// Parse as boolean (handles "true", "1", "yes", "on", etc.)
	enabled, err := strconv.ParseBool(strings.ToLower(enableSwarm))
	if err == nil {
		return enabled
	}
	// Fallback: check for common truthy strings
	lower := strings.ToLower(strings.TrimSpace(enableSwarm))
	return lower == "true" || lower == "1" || lower == "yes" || lower == "on"
}

// DeploymentManager manages the lifecycle of user deployments
type DeploymentManager struct {
	dockerClient client.APIClient
	dockerHelper dockerHelper
	nodeSelector *NodeSelector
	registry     *registry.ServiceRegistry
	networkName  string
	nodeID       string
	nodeHostname string
	forwarder    *NodeForwarder
}

// dockerHelper defines the subset of docker helper methods used here.
type dockerHelper interface {
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string, timeout time.Duration) error
	RemoveContainer(ctx context.Context, containerID string, force bool) error
	RestartContainer(ctx context.Context, containerID string, timeout time.Duration) error
	ContainerLogs(ctx context.Context, containerID string, tail string, follow bool) (io.ReadCloser, error)
	ContainerExecRun(ctx context.Context, containerID string, cmd []string) (string, error)
}

// DeploymentConfig holds configuration for a new deployment
type DeploymentConfig struct {
	DeploymentID string
	Image        string
	Domain       string
	Port         int
	EnvVars      map[string]string
	Labels       map[string]string
	Memory       int64 // in bytes
	CPUShares    int64
	Replicas     int
	StartCommand *string // Optional start command to override container CMD
}

// NewDeploymentManager creates a new deployment manager
func NewDeploymentManager(strategy string, maxDeploymentsPerNode int) (*DeploymentManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	nodeSelector, err := NewNodeSelector(strategy, maxDeploymentsPerNode)
	if err != nil {
		return nil, fmt.Errorf("failed to create node selector: %w", err)
	}

	serviceRegistry, err := registry.NewServiceRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to create service registry: %w", err)
	}

	// Get node info
	info, err := cli.Info(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker info: %w", err)
	}

	helper, err := docker.New()
	if err != nil {
		return nil, fmt.Errorf("failed to init docker helper: %w", err)
	}

	// Determine node ID - use Swarm node ID if available, otherwise use synthetic local ID
	nodeID := info.Swarm.NodeID
	if nodeID == "" {
		// Not in Swarm mode - use synthetic ID matching what node selector uses
		nodeID = "local-" + info.Name
	}

	dm := &DeploymentManager{
		dockerClient: cli,
		dockerHelper: helper,
		nodeSelector: nodeSelector,
		registry:     serviceRegistry,
		networkName:  "obiente-network",
		nodeID:       nodeID,
		nodeHostname: info.Name,
		forwarder:    NewNodeForwarder(),
	}

	// Ensure the network exists (non-blocking - we'll try again when needed)
	// If this fails, we'll attempt to create it later when actually deploying
	if err := dm.ensureNetwork(context.Background()); err != nil {
		logger.Warn("[DeploymentManager] Failed to ensure network exists during initialization: %v", err)
		logger.Info("[DeploymentManager] Network will be created on-demand when deploying containers")
		// Don't fail initialization - network creation will be retried during deployment
		// This allows the system to start even if Docker has temporary issues
	}

	return dm, nil
}

// GetNodeID returns the node ID for this deployment manager
func (dm *DeploymentManager) GetNodeID() string {
	return dm.nodeID
}

// ensureNetwork ensures the obiente-network exists, creating it if necessary
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

// CreateDeployment creates a new deployment on the cluster
func (dm *DeploymentManager) CreateDeployment(ctx context.Context, config *DeploymentConfig) error {
	logger.Info("[DeploymentManager] Creating deployment %s", config.DeploymentID)

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
	isSwarmMode := isSwarmModeEnabled()

	// Create containers/services for each service and replica
	for _, serviceName := range serviceNames {
		for i := 0; i < config.Replicas; i++ {
			containerName := fmt.Sprintf("%s-%s-replica-%d", config.DeploymentID, serviceName, i)

			var containerID string
			var err error

			if isSwarmMode {
				// In Swarm mode, create Swarm services instead of plain containers
				logger.Info("[DeploymentManager] Creating Swarm service for deployment %s (service: %s, replica: %d)", config.DeploymentID, serviceName, i)

				// Remove existing service if it exists
				swarmServiceName := fmt.Sprintf("deploy-%s-%s", config.DeploymentID, serviceName)
				if i > 0 {
					swarmServiceName = fmt.Sprintf("deploy-%s-%s-replica-%d", config.DeploymentID, serviceName, i)
				}
				rmArgs := []string{"service", "rm", swarmServiceName}
				rmCmd := exec.CommandContext(ctx, "docker", rmArgs...)
				rmCmd.Run() // Ignore errors - service might not exist

				// Wait a moment for service removal
				time.Sleep(1 * time.Second)

				// Create Swarm service
				_, containerID, err = dm.createSwarmService(ctx, config, serviceName, i)
				if err != nil {
					return fmt.Errorf("failed to create Swarm service: %w", err)
				}

				// For Swarm services, containerID might be empty initially - try to get it from service tasks
				if containerID == "" {
					// Wait a bit more for task to be created
					time.Sleep(2 * time.Second)
					// Try to find container by service name label
					filterArgs := filters.NewArgs()
					filterArgs.Add("label", fmt.Sprintf("com.docker.swarm.service.name=%s", swarmServiceName))
					containers, listErr := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
						All:     true,
						Filters: filterArgs,
					})
					if listErr == nil && len(containers) > 0 {
						containerID = containers[0].ID
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
				info, err := dm.dockerClient.ContainerInspect(ctx, containerID)
				if err == nil {
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
					logger.Warn("[DeploymentManager] Failed to inspect container %s: %v", containerID, err)
					publicPort = config.Port
					for _, routing := range routings {
						if routing.ServiceName == serviceName || (serviceName == "default" && routing.ServiceName == "") {
							publicPort = routing.TargetPort
							break
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

	// Create default deployment routing (for backward compatibility)
	// Only create if no routing rules exist - preserve user-configured routing
	existingRoutings, _ := database.GetDeploymentRoutings(config.DeploymentID)
	if len(existingRoutings) == 0 {
		// Check if a default routing already exists (might have been created previously)
		// This handles the case where GetDeploymentRoutings returns empty but a routing exists in DB
		defaultRoutingID := fmt.Sprintf("route-%s", config.DeploymentID)
		var existingDefaultRouting database.DeploymentRouting
		dbErr := database.DB.Where("id = ?", defaultRoutingID).First(&existingDefaultRouting).Error

		// If a default routing exists, preserve all user settings (especially port)
		if dbErr == nil {
			// User has set routing rules - preserve them completely, don't overwrite
			logger.Info("[DeploymentManager] Found existing default routing for deployment %s, preserving all user settings (port: %d)", config.DeploymentID, existingDefaultRouting.TargetPort)
		} else {
			// No existing routing found, create default routing
			routing := &database.DeploymentRouting{
				ID:              defaultRoutingID,
				DeploymentID:    config.DeploymentID,
				Domain:          config.Domain,
				ServiceName:     "default",
				TargetPort:      config.Port,
				Protocol:        "http",
				SSLEnabled:      false, // Default to no SSL for HTTP protocol
				SSLCertResolver: "letsencrypt",
				Middleware:      "{}", // Empty JSON object for jsonb field
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}

			if err := database.UpsertDeploymentRouting(routing); err != nil {
				logger.Warn("[DeploymentManager] Failed to create routing: %v", err)
			} else {
				logger.Info("[DeploymentManager] Created default routing for deployment %s (port: %d)", config.DeploymentID, config.Port)
			}
		}
	} else {
		// Routing rules already exist - preserve them, don't overwrite
		logger.Info("[DeploymentManager] Deployment %s already has %d routing rule(s), preserving existing configuration", config.DeploymentID, len(existingRoutings))
	}

	logger.Info("[DeploymentManager] Deployment %s created successfully", config.DeploymentID)
	return nil
}

// generateTraefikLabels generates Traefik labels from routing rules
// If no routing rules exist and servicePort is provided, it will create basic labels with port
func generateTraefikLabels(deploymentID string, serviceName string, routings []database.DeploymentRouting, servicePort *int) map[string]string {
	labels := make(map[string]string)

	// Filter routings for this service name
	serviceRoutings := []database.DeploymentRouting{}
	for _, routing := range routings {
		if routing.ServiceName == serviceName || (serviceName == "default" && routing.ServiceName == "") {
			serviceRoutings = append(serviceRoutings, routing)
		}
	}

	// If no specific routing found, don't enable Traefik unless we have a port
	// User must configure routing before the service will be accessible via Traefik
	if len(serviceRoutings) == 0 {
		// Only enable Traefik if we have port information (but no routing rules means no router will be created)
		// We still need to NOT enable it to avoid Traefik errors
		return labels // Return empty - don't enable Traefik for services without routing rules
	}

	// Enable Traefik only when we have routing rules
	labels["traefik.enable"] = "true"
	labels["cloud.obiente.traefik"] = "true" // Required for Traefik to discover this container

	// Generate labels for each routing rule
	for idx, routing := range serviceRoutings {
		routerName := deploymentID
		if serviceName != "default" {
			routerName = deploymentID + "-" + serviceName
		}
		if idx > 0 {
			routerName = fmt.Sprintf("%s-%d", routerName, idx)
		}

		// Build rule: Host or Host + PathPrefix
		rule := "Host(`" + routing.Domain + "`)"
		if routing.PathPrefix != "" {
			rule = rule + " && PathPrefix(`" + routing.PathPrefix + "`)"
		}
		labels["traefik.http.routers."+routerName+".rule"] = rule

		// Entrypoints - respect protocol field
		// HTTP protocol should use web (no SSL), HTTPS protocol or SSLEnabled=true should use websecure
		shouldUseSSL := false
		if routing.Protocol == "https" {
			// HTTPS protocol always uses SSL
			shouldUseSSL = true
		} else if routing.Protocol == "http" {
			// HTTP protocol never uses SSL, regardless of SSLEnabled flag
			shouldUseSSL = false
		} else {
			// For other protocols (grpc, etc.) or if protocol is not set, use SSLEnabled flag
			shouldUseSSL = routing.SSLEnabled
		}

		if shouldUseSSL {
			labels["traefik.http.routers."+routerName+".entrypoints"] = "websecure"
			if routing.SSLCertResolver != "" && routing.SSLCertResolver != "internal" {
				labels["traefik.http.routers."+routerName+".tls.certresolver"] = routing.SSLCertResolver
			} else if routing.SSLCertResolver == "internal" {
				// For internal SSL, don't set certresolver (let app handle it)
				labels["traefik.http.routers."+routerName+".entrypoints"] = "web"
			}
		} else {
			// HTTP-only: explicitly set web entrypoint and ensure no TLS labels
			labels["traefik.http.routers."+routerName+".entrypoints"] = "web"
			// Explicitly remove any TLS configuration for HTTP-only routers
			// Note: Docker label deletion in compose requires the label to not exist at all
			// We rely on only setting web entrypoint which won't trigger TLS
		}

		// Service port
		serviceNameLabel := routerName
		labels["traefik.http.services."+serviceNameLabel+".loadbalancer.server.port"] = strconv.Itoa(routing.TargetPort)
	}

	return labels
}

// injectTraefikLabelsIntoCompose injects Traefik labels into a Docker Compose YAML string
func (dm *DeploymentManager) injectTraefikLabelsIntoCompose(composeYaml string, deploymentID string, routings []database.DeploymentRouting) (string, error) {
	// Parse YAML
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(composeYaml), &compose); err != nil {
		return "", fmt.Errorf("failed to parse compose YAML: %w", err)
	}

	// Detect if we're in Swarm mode using ENABLE_SWARM environment variable
	// In Swarm mode, labels must be in deploy.labels
	// In non-Swarm mode, labels must be at the top-level labels
	isSwarmMode := isSwarmModeEnabled()
	if isSwarmMode {
		logger.Debug("[DeploymentManager] ENABLE_SWARM=true - labels will be placed in deploy.labels")
	} else {
		logger.Debug("[DeploymentManager] ENABLE_SWARM=false or not set - labels will be placed at top-level")
	}

	// Get deployment domain from routings if available
	var deploymentDomain string
	if len(routings) > 0 {
		deploymentDomain = routings[0].Domain
	}

	// Inject labels into each service
	if services, ok := compose["services"].(map[string]interface{}); ok {
		for serviceName, serviceData := range services {
			if service, ok := serviceData.(map[string]interface{}); ok {
				// Extract port from compose service if available
				var servicePort *int
				if ports, ok := service["ports"].([]interface{}); ok && len(ports) > 0 {
					// Try to extract port from first port mapping
					if portStr, ok := ports[0].(string); ok {
						// Format: "host:container" or just "container"
						parts := strings.Split(portStr, ":")
						if len(parts) >= 2 {
							if p, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
								servicePort = &p
							}
						} else if len(parts) == 1 {
							if p, err := strconv.Atoi(parts[0]); err == nil {
								servicePort = &p
							}
						}
					}
				} else if expose, ok := service["expose"].([]interface{}); ok && len(expose) > 0 {
					// Check exposed ports
					if portStr, ok := expose[0].(string); ok {
						if p, err := strconv.Atoi(portStr); err == nil {
							servicePort = &p
						}
					}
				}

				// Generate Traefik labels for this service
				traefikLabels := generateTraefikLabels(deploymentID, serviceName, routings, servicePort)

				// When Traefik handles routing, we should not expose ports to the host
				// Convert ports to expose (internal network only) if Traefik labels are present
				if len(traefikLabels) > 0 {
					// Check if service has ports that should be converted to expose
					if ports, ok := service["ports"].([]interface{}); ok && len(ports) > 0 {
						// Extract container ports from ports mapping
						exposedPorts := []interface{}{}
						for _, port := range ports {
							var containerPort string
							switch v := port.(type) {
							case string:
								// Format: "host:container" or "container" or "container/protocol"
								if strings.Contains(v, ":") {
									parts := strings.SplitN(v, ":", 2)
									if len(parts) == 2 {
										containerPort = strings.TrimSpace(parts[1])
									}
								} else {
									containerPort = v
								}
							case map[string]interface{}:
								// Port mapping object format
								if target, ok := v["target"].(int); ok {
									containerPort = strconv.Itoa(target)
								} else if published, ok := v["published"].(int); ok {
									containerPort = strconv.Itoa(published)
								}
							}
							if containerPort != "" {
								exposedPorts = append(exposedPorts, containerPort)
							}
						}

						// Remove ports and add expose instead (internal network only)
						delete(service, "ports")
						if len(exposedPorts) > 0 {
							// Merge with existing expose if present
							var existingExpose []interface{}
							if existing, ok := service["expose"].([]interface{}); ok {
								existingExpose = existing
							}
							// Combine and deduplicate
							exposeMap := make(map[string]bool)
							for _, p := range existingExpose {
								if portStr, ok := p.(string); ok {
									exposeMap[portStr] = true
								}
							}
							for _, p := range exposedPorts {
								if portStr, ok := p.(string); ok {
									exposeMap[portStr] = true
								}
							}
							// Convert back to list
							finalExpose := []interface{}{}
							for port := range exposeMap {
								finalExpose = append(finalExpose, port)
							}
							if len(finalExpose) > 0 {
								service["expose"] = finalExpose
								logger.Info("[DeploymentManager] Converted ports to expose for service %s (Traefik routing - no host port exposure)", serviceName)
							}
						}
					}
				}

				// Determine the health check port - ALWAYS use routing target port if available
				// Never use compose file port as it may be a default value
				var healthCheckPort *int
				routingPortUsed := false
				
				if len(routings) > 0 {
					// Find routing for this service
					// Service name matching: exact match, or "default" service matches routing with ServiceName="default" or ""
					for _, routing := range routings {
						serviceMatches := routing.ServiceName == serviceName ||
							(serviceName == "default" && (routing.ServiceName == "" || routing.ServiceName == "default")) ||
							(routing.ServiceName == "default" && serviceName == "")
						
						if serviceMatches {
							if routing.TargetPort > 0 {
								healthCheckPort = &routing.TargetPort
								routingPortUsed = true
								logger.Info("[DeploymentManager] Using routing target port %d for health check (service: %s, routing service: %s, domain: %s)", routing.TargetPort, serviceName, routing.ServiceName, routing.Domain)
								break
							}
						}
					}
					
					// If no matching service found but we have routings, use first routing's target port as fallback
					if !routingPortUsed && len(routings) > 0 && routings[0].TargetPort > 0 {
						healthCheckPort = &routings[0].TargetPort
						routingPortUsed = true
						logger.Info("[DeploymentManager] No exact service match found, using first routing's target port %d for health check (service: %s, routing service: %s)", routings[0].TargetPort, serviceName, routings[0].ServiceName)
					}
				}
				
				// Only fall back to compose file port if no routing found at all
				if healthCheckPort == nil && servicePort != nil && *servicePort > 0 {
					healthCheckPort = servicePort
					logger.Debug("[DeploymentManager] Health check will use compose file port: %d (no routing found)", *servicePort)
				}

				// If we still don't have a port, try to extract from expose (after ports conversion)
				if healthCheckPort == nil {
					if expose, ok := service["expose"].([]interface{}); ok && len(expose) > 0 {
						// Check exposed ports
						if portStr, ok := expose[0].(string); ok {
							// Remove protocol if present (e.g., "4321/tcp" -> "4321")
							portStr = strings.Split(portStr, "/")[0]
							if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
								healthCheckPort = &p
								logger.Debug("[DeploymentManager] Health check will use exposed port: %d (no routing found)", p)
							}
						}
					}
				}

				// Inject health check if we have a port and no existing health check
				// Health checks are important for Traefik to know when services are ready
				// Without health checks, containers may stay in "starting" status even when running
				if healthCheckPort != nil {
					// Check if health check already exists
					_, hasHealthcheck := service["healthcheck"]
					if !hasHealthcheck {
						// Add health check configuration using netcat (nc) - smallest tool for TCP port checks
						// Netcat is much lighter than curl/wget since we only need to check if port is listening
						// Try netcat first, if not found install it (supports Alpine, Debian/Ubuntu, and CentOS/RHEL)
						// Uses TCP connection test: nc -z localhost PORT (returns 0 if port is open)
						healthcheckCmd := fmt.Sprintf(`sh -c 'if command -v nc >/dev/null 2>&1; then nc -z localhost %d || exit 1; else (apk add --no-cache netcat-openbsd >/dev/null 2>&1 || apt-get update -qq && apt-get install -y -qq netcat-openbsd >/dev/null 2>&1 || yum install -y -q nc >/dev/null 2>&1) && nc -z localhost %d || exit 1; fi'`, *healthCheckPort, *healthCheckPort)
						healthcheck := map[string]interface{}{
							"test":        []interface{}{"CMD-SHELL", healthcheckCmd},
							"interval":    "30s",
							"timeout":     "10s",
							"retries":     3,
							"start_period": "40s", // Give container time to start before health checks begin
						}
						service["healthcheck"] = healthcheck
						logger.Info("[DeploymentManager] Added health check for service %s on port %d - using netcat for TCP port check", serviceName, *healthCheckPort)
					} else {
						logger.Debug("[DeploymentManager] Service %s already has a health check, skipping", serviceName)
					}
				} else {
					logger.Warn("[DeploymentManager] Cannot add health check for service %s - no port found (servicePort=%v, routings=%d)", serviceName, servicePort, len(routings))
				}

				// Ensure netcat is available by adding environment variables for nixpacks/railpacks
				// These env vars tell the build system to install netcat (only if we added a health check)
				if healthCheckPort != nil {
					var env map[string]interface{}
					if existingEnv, ok := service["environment"].(map[string]interface{}); ok {
						env = existingEnv
					} else if existingEnvList, ok := service["environment"].([]interface{}); ok {
						// Convert list format to map format
						env = make(map[string]interface{})
						for _, envItem := range existingEnvList {
							if envStr, ok := envItem.(string); ok {
								if strings.Contains(envStr, "=") {
									parts := strings.SplitN(envStr, "=", 2)
									if len(parts) == 2 {
										env[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
									}
								}
							}
						}
					} else {
						env = make(map[string]interface{})
					}

					// Add nixpacks/railpacks environment variables to ensure netcat is installed
					// Only add if not already set (don't override user's custom values)
					// NIXPACKS_APT_PKGS is for Apt packages (netcat-openbsd is an apt package)
					if _, exists := env["NIXPACKS_APT_PKGS"]; !exists {
						env["NIXPACKS_APT_PKGS"] = "netcat-openbsd"
					}
					// RAILPACK_DEPLOY_APT_PACKAGES installs packages in the final image (what we need for health checks)
					if _, exists := env["RAILPACK_DEPLOY_APT_PACKAGES"]; !exists {
						env["RAILPACK_DEPLOY_APT_PACKAGES"] = "netcat-openbsd"
					}

					// Update environment in service
					service["environment"] = env
					logger.Debug("[DeploymentManager] Added netcat installation env vars for service %s (NIXPACKS_APT_PKGS, RAILPACK_DEPLOY_APT_PACKAGES)", serviceName)
				}

				// Helper function to get or create labels map from various formats
				getLabelsMap := func(labelsValue interface{}) map[string]interface{} {
					labels := make(map[string]interface{})
					if labelsValue == nil {
						return labels
					}
					if existingLabels, ok := labelsValue.(map[string]interface{}); ok {
						labels = existingLabels
					} else if existingLabelsList, ok := labelsValue.([]interface{}); ok {
						// Convert list format to map format
						for _, labelItem := range existingLabelsList {
							if labelStr, ok := labelItem.(string); ok {
								// Parse "key=value" or "key: value" format
								if strings.Contains(labelStr, "=") {
									parts := strings.SplitN(labelStr, "=", 2)
									if len(parts) == 2 {
										labels[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
									}
								}
							}
						}
					}
					return labels
				}

				// Remove old Traefik labels that might conflict
				removeTraefikLabels := func(labels map[string]interface{}) {
					traefikKeysToRemove := []string{}
					for key := range labels {
						if strings.HasPrefix(key, "traefik.http.routers.") ||
							strings.HasPrefix(key, "traefik.http.services.") ||
							key == "traefik.enable" ||
							key == "cloud.obiente.traefik" {
							traefikKeysToRemove = append(traefikKeysToRemove, key)
						}
					}
					for _, key := range traefikKeysToRemove {
						delete(labels, key)
					}
				}

				// Add Traefik and management labels
				addLabels := func(labels map[string]interface{}) {
					// Merge new Traefik labels (Traefik labels take precedence)
					for k, v := range traefikLabels {
						labels[k] = v
					}

					// Add management labels
					labels["cloud.obiente.managed"] = "true"
					labels["cloud.obiente.deployment_id"] = deploymentID
					labels["cloud.obiente.service_name"] = serviceName
					// Only set cloud.obiente.traefik if Traefik labels were generated (i.e., routing rules exist)
					if len(traefikLabels) > 0 {
						labels["cloud.obiente.traefik"] = "true" // Required for Traefik discovery
					}
					if deploymentDomain != "" {
						labels["cloud.obiente.domain"] = deploymentDomain
					}
				}

				if isSwarmMode {
					// For Docker Swarm mode, Traefik requires labels to be in deploy.labels
					// Get or create deploy section
					var deploy map[string]interface{}
					if existingDeploy, ok := service["deploy"].(map[string]interface{}); ok {
						deploy = existingDeploy
					} else {
						deploy = make(map[string]interface{})
						service["deploy"] = deploy
					}

					// Get or create labels map under deploy.labels
					labels := getLabelsMap(deploy["labels"])
					removeTraefikLabels(labels)
					addLabels(labels)

					// Update deploy.labels (required for Swarm mode Traefik discovery)
					deploy["labels"] = labels
					if len(traefikLabels) > 0 {
						logger.Debug("[DeploymentManager] Added %d Traefik labels to deploy.labels for service %s (Swarm mode)", len(traefikLabels), serviceName)
					}
				} else {
					// For non-Swarm mode (Docker Compose), Traefik reads labels from top-level labels
					// Get or create top-level labels
					labels := getLabelsMap(service["labels"])
					removeTraefikLabels(labels)
					addLabels(labels)

					// Update top-level labels (required for non-Swarm mode Traefik discovery)
					service["labels"] = labels
					if len(traefikLabels) > 0 {
						logger.Debug("[DeploymentManager] Added %d Traefik labels to top-level labels for service %s (non-Swarm mode)", len(traefikLabels), serviceName)
					}
				}
			}
		}
	}

	// Ensure network configuration is set correctly for Swarm mode
	// In Swarm mode, services must be on the network that Traefik monitors
	if isSwarmMode {
		// Ensure networks section exists
		var networks map[string]interface{}
		if existingNetworks, ok := compose["networks"].(map[string]interface{}); ok {
			networks = existingNetworks
		} else {
			networks = make(map[string]interface{})
			compose["networks"] = networks
		}

		// Add or update obiente-network to be external (references the Swarm network)
		// In Swarm mode, the network name is prefixed with stack name: obiente_obiente-network
		// But we'll use the simple name and let Docker Compose handle the prefix
		networkConfig := map[string]interface{}{
			"external": true,
			"name":     "obiente_obiente-network", // Use the actual Swarm network name
		}
		networks["obiente-network"] = networkConfig

		// Ensure all services are connected to the network
		if services, ok := compose["services"].(map[string]interface{}); ok {
			for serviceName, serviceData := range services {
				if service, ok := serviceData.(map[string]interface{}); ok {
					// Get or create networks section for this service
					var serviceNetworks map[string]interface{}
					if existingServiceNetworks, ok := service["networks"].(map[string]interface{}); ok {
						serviceNetworks = existingServiceNetworks
					} else if existingServiceNetworksList, ok := service["networks"].([]interface{}); ok {
						// Convert list format to map format
						serviceNetworks = make(map[string]interface{})
						for _, netItem := range existingServiceNetworksList {
							if netStr, ok := netItem.(string); ok {
								serviceNetworks[netStr] = nil
							}
						}
					} else {
						serviceNetworks = make(map[string]interface{})
					}

					// Ensure obiente-network is in the service's networks
					serviceNetworks["obiente-network"] = nil
					service["networks"] = serviceNetworks
					logger.Debug("[DeploymentManager] Ensured service %s is connected to obiente-network (Swarm mode)", serviceName)
				}
			}
		}
	}

	// Marshal back to YAML
	labeledYaml, err := yaml.Marshal(compose)
	if err != nil {
		return "", fmt.Errorf("failed to marshal labeled compose YAML: %w", err)
	}

	return string(labeledYaml), nil
}

// removeContainerByName removes a container by name if it exists
func (dm *DeploymentManager) removeContainerByName(ctx context.Context, containerName string) error {
	// Try to inspect container directly by name (most efficient)
	// Docker API accepts both with and without leading "/"
	containerNameWithSlash := "/" + containerName
	containerNameWithoutSlash := strings.TrimPrefix(containerName, "/")

	// Try both variations
	for _, nameToTry := range []string{containerNameWithSlash, containerNameWithoutSlash} {
		containerInfo, err := dm.dockerClient.ContainerInspect(ctx, nameToTry)
		if err == nil {
			// Container exists, remove it
			logger.Info("[DeploymentManager] Removing existing container %s (ID: %s) for redeployment", containerName, containerInfo.ID[:12])

			// Stop container first
			timeout := 10 * time.Second
			_ = dm.dockerHelper.StopContainer(ctx, containerInfo.ID, timeout)

			// Remove container
			if err := dm.dockerHelper.RemoveContainer(ctx, containerInfo.ID, true); err != nil {
				return fmt.Errorf("failed to remove existing container %s: %w", containerName, err)
			}

			// Unregister from registry if it was registered
			_ = dm.registry.UnregisterDeployment(ctx, containerInfo.ID)

			logger.Info("[DeploymentManager] Successfully removed existing container %s", containerName)
			return nil
		}
		// If error is "not found", continue to next name variation
		// If it's another error, we'll try the list approach as fallback
	}

	// Fallback: List all containers and find by exact name match
	// This handles edge cases where inspect might not work
	allContainers, err := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	// Find container with exact name match
	for _, cnt := range allContainers {
		for _, n := range cnt.Names {
			// Docker names start with "/", so check both
			nTrimmed := strings.TrimPrefix(n, "/")
			if nTrimmed == containerNameWithoutSlash || n == containerNameWithSlash {
				logger.Info("[DeploymentManager] Removing existing container %s (ID: %s) for redeployment", containerName, cnt.ID[:12])

				// Stop container first
				timeout := 10 * time.Second
				_ = dm.dockerHelper.StopContainer(ctx, cnt.ID, timeout)

				// Remove container
				if err := dm.dockerHelper.RemoveContainer(ctx, cnt.ID, true); err != nil {
					return fmt.Errorf("failed to remove existing container %s: %w", containerName, err)
				}

				// Unregister from registry if it was registered
				_ = dm.registry.UnregisterDeployment(ctx, cnt.ID)

				logger.Info("[DeploymentManager] Successfully removed existing container %s", containerName)
				return nil
			}
		}
	}

	return nil // Container doesn't exist, which is fine
}

// createContainer creates a single container
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

	// Generate Traefik labels from routing rules
	// Use config.Port for service port (which should be from routing target port if available)
	servicePort := config.Port
	traefikLabels := generateTraefikLabels(config.DeploymentID, serviceName, routings, &servicePort)
	for k, v := range traefikLabels {
		labels[k] = v
	}
	// Only set cloud.obiente.traefik if we actually generated Traefik labels (i.e., routing rules exist)
	if len(traefikLabels) > 0 {
		labels["cloud.obiente.traefik"] = "true" // Required for Traefik discovery
	}

	// Determine health check port - ALWAYS use routing target port if available
	// Never use config.Port as it may be a default value
	healthCheckPort := 0
	if len(routings) > 0 {
		// Find routing for this service
		for _, routing := range routings {
			if routing.ServiceName == serviceName || (serviceName == "default" && (routing.ServiceName == "" || routing.ServiceName == "default")) {
				if routing.TargetPort > 0 {
					healthCheckPort = routing.TargetPort
					logger.Debug("[DeploymentManager] Using routing target port %d for health check (service: %s)", healthCheckPort, serviceName)
					break
				}
			}
		}
		// If no exact match, use first routing's target port
		if healthCheckPort == 0 && len(routings) > 0 && routings[0].TargetPort > 0 {
			healthCheckPort = routings[0].TargetPort
			logger.Debug("[DeploymentManager] Using first routing target port %d for health check (service: %s)", healthCheckPort, serviceName)
		}
	}
	
	// Fallback to config.Port only if no routing found and config.Port is set
	if healthCheckPort == 0 && config.Port > 0 {
		healthCheckPort = config.Port
		logger.Debug("[DeploymentManager] Using config port %d for health check (no routing found)", healthCheckPort)
	}
	
	// If still no port, we can't add health check
	if healthCheckPort == 0 {
		logger.Warn("[DeploymentManager] Cannot determine health check port for service %s - no routing target port or config port available", serviceName)
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

	// Ensure netcat is available by adding environment variables for nixpacks/railpacks
	// These env vars tell the build system to install netcat (only if we need a health check)
	// Note: These only work if the image is built with Nixpacks/Railpacks
	// For pre-built images, the health check will install netcat at runtime
	if healthCheckPort > 0 {
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
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}

	if containerPortNum > 0 {
		containerPort := nat.Port(fmt.Sprintf("%d/tcp", containerPortNum))
		exposedPorts[containerPort] = struct{}{}
		
		// Only bind to host if Traefik is NOT handling routing
		// If Traefik labels exist, don't expose to host (Traefik will route internally)
		if len(traefikLabels) == 0 {
			// No Traefik routing - expose to host with random port for security
			portBindings[containerPort] = []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "0", // SECURITY: Docker assigns random port - users cannot bind to specific host ports
				},
			}
			logger.Debug("[DeploymentManager] Exposing container port %d to host (random port) - no Traefik routing", containerPortNum)
		} else {
			// Traefik handles routing - don't expose to host, only expose internally
			logger.Info("[DeploymentManager] Not exposing container port %d to host - Traefik will handle routing", containerPortNum)
		}
	}

	// Only add health check if we have a valid port from routing
	// Health check port must be from routing target port, not a default
	var healthcheck *container.HealthConfig
	if healthCheckPort > 0 {
		// Health check command using netcat (nc) - smallest tool for TCP port checks
		// Netcat is much lighter than curl/wget since we only need to check if port is listening
		// Try netcat first, if not found install it (supports Alpine, Debian/Ubuntu, and CentOS/RHEL)
		// Uses TCP connection test: nc -z localhost PORT (returns 0 if port is open)
		healthCheckCmd := fmt.Sprintf(`sh -c 'if command -v nc >/dev/null 2>&1; then nc -z localhost %d || exit 1; else (apk add --no-cache netcat-openbsd >/dev/null 2>&1 || apt-get update -qq && apt-get install -y -qq netcat-openbsd >/dev/null 2>&1 || yum install -y -q nc >/dev/null 2>&1) && nc -z localhost %d || exit 1; fi'`, healthCheckPort, healthCheckPort)
		
		healthcheck = &container.HealthConfig{
			Test:        []string{"CMD-SHELL", healthCheckCmd},
			Interval:    30 * time.Second,
			Timeout:    10 * time.Second,
			Retries:    3,
			StartPeriod: 40 * time.Second, // Give container time to start before health checks begin
		}
		logger.Info("[DeploymentManager] Added health check for container %s on port %d (from routing) - using netcat for TCP port check", name, healthCheckPort)
	} else {
		logger.Warn("[DeploymentManager] Skipping health check for container %s - no valid port found from routing", name)
	}

	// Container configuration
	containerConfig := &container.Config{
		Image:        config.Image,
		Env:          env, // Environment variables including NIXPACKS_PKGS/RAILPACK_* if health check is needed
		Labels:       labels,
		ExposedPorts: exposedPorts,
		// Clear ENTRYPOINT to avoid conflicts when overriding CMD
		Entrypoint: []string{},
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
			CPUShares: config.CPUShares,
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
	resp, err := dm.dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, name)
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
			resp, err = dm.dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, name)
			if err != nil {
				return "", fmt.Errorf("failed to create container after removing conflicting container: %w", err)
			}
		} else {
			return "", fmt.Errorf("failed to create container: %w", err)
		}
	}

	return resp.ID, nil
}

// createSwarmService creates a Swarm service for a deployment
// Returns the service ID and container ID (from the first task)
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

	// Generate Traefik labels from routing rules
	servicePort := config.Port
	traefikLabels := generateTraefikLabels(config.DeploymentID, serviceName, routings, &servicePort)
	for k, v := range traefikLabels {
		labels[k] = v
	}
	// Only set cloud.obiente.traefik if we actually generated Traefik labels
	if len(traefikLabels) > 0 {
		labels["cloud.obiente.traefik"] = "true" // Required for Traefik discovery
	}

	// Determine health check port - ALWAYS use routing target port if available
	healthCheckPort := 0
	if len(routings) > 0 {
		for _, routing := range routings {
			if routing.ServiceName == serviceName || (serviceName == "default" && (routing.ServiceName == "" || routing.ServiceName == "default")) {
				if routing.TargetPort > 0 {
					healthCheckPort = routing.TargetPort
					break
				}
			}
		}
		if healthCheckPort == 0 && len(routings) > 0 && routings[0].TargetPort > 0 {
			healthCheckPort = routings[0].TargetPort
		}
	}
	if healthCheckPort == 0 && config.Port > 0 {
		healthCheckPort = config.Port
	}

	// Determine container port
	containerPortNum := config.Port
	if len(routings) > 0 {
		for _, routing := range routings {
			if routing.ServiceName == serviceName || (serviceName == "default" && (routing.ServiceName == "" || routing.ServiceName == "default")) {
				if routing.TargetPort > 0 {
					containerPortNum = routing.TargetPort
					break
				}
			}
		}
		if containerPortNum == config.Port && len(routings) > 0 && routings[0].TargetPort > 0 {
			containerPortNum = routings[0].TargetPort
		}
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

	// Service name format: deploy-{deploymentID}-{serviceName}
	swarmServiceName := fmt.Sprintf("deploy-%s-%s", config.DeploymentID, serviceName)
	if replicaIndex > 0 {
		swarmServiceName = fmt.Sprintf("deploy-%s-%s-replica-%d", config.DeploymentID, serviceName, replicaIndex)
	}

	// Build docker service create command
	args := []string{"service", "create",
		"--name", swarmServiceName,
		"--network", "obiente_obiente-network", // Use the Swarm network name
		"--replicas", "1",
	}

	// Add labels
	for k, v := range labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}

	// Add environment variables
	for _, e := range env {
		args = append(args, "--env", e)
	}

	// Add start command if provided
	if config.StartCommand != nil && *config.StartCommand != "" {
		args = append(args, "--command", *config.StartCommand)
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
	args = append(args,
		"--limit-memory", fmt.Sprintf("%d", config.Memory),
		"--limit-cpu", fmt.Sprintf("%d", config.CPUShares),
	)

	// Add restart policy
	args = append(args, "--restart-condition", "unless-stopped")

	// Add image
	args = append(args, config.Image)

	// Execute docker service create
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		errorOutput := stderr.String()
		stdOutput := stdout.String()
		logger.Error("[DeploymentManager] Failed to create Swarm service %s: %v\nStderr: %s\nStdout: %s", swarmServiceName, err, errorOutput, stdOutput)
		return "", "", fmt.Errorf("failed to create Swarm service: %w\nStderr: %s\nStdout: %s", err, errorOutput, stdOutput)
	}

	serviceID := strings.TrimSpace(stdout.String())
	logger.Info("[DeploymentManager] Created Swarm service %s (ID: %s)", swarmServiceName, serviceID)

	// Wait a moment for the service to create a task
	time.Sleep(2 * time.Second)

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
					// Inspect the task to get container ID
					taskInspectArgs := []string{"inspect", taskID, "--format", "{{.Status.ContainerStatus.ContainerID}}"}
					taskInspectCmd := exec.CommandContext(ctx, "docker", taskInspectArgs...)
					var taskInspectStdout bytes.Buffer
					taskInspectCmd.Stdout = &taskInspectStdout
					if err := taskInspectCmd.Run(); err == nil {
						containerID = strings.TrimSpace(taskInspectStdout.String())
					}
				}
			}
		}
	}

	// If we couldn't get container ID from task, we'll use the service ID as a placeholder
	// The container will be created by Swarm and we can look it up later
	if containerID == "" {
		logger.Warn("[DeploymentManager] Could not get container ID from Swarm service %s tasks - will look up later", swarmServiceName)
		// Try to find container by service label
		filterArgs := filters.NewArgs()
		filterArgs.Add("label", fmt.Sprintf("com.docker.swarm.service.name=%s", swarmServiceName))
		containers, _ := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
			All:     true,
			Filters: filterArgs,
		})
		if len(containers) > 0 {
			containerID = containers[0].ID
		}
	}

	return serviceID, containerID, nil
}

// StartDeployment starts all containers for a deployment
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
			if port == 0 {
				if deployment.Port != nil && *deployment.Port > 0 {
					port = int(*deployment.Port)
					logger.Info("[StartDeployment] Using deployment port %d (no routing found) for deployment %s", port, deploymentID)
				} else {
					return fmt.Errorf("deployment %s has no port configured in routing rules or deployment settings", deploymentID)
				}
			}
			
			memory := int64(512 * 1024 * 1024) // Default 512MB
			if deployment.MemoryBytes != nil {
				memory = *deployment.MemoryBytes
			}
			cpuShares := int64(1024) // Default
			if deployment.CPUShares != nil {
				cpuShares = *deployment.CPUShares
			}
			replicas := 1 // Default
			if deployment.Replicas != nil {
				replicas = int(*deployment.Replicas)
			}

			config := &DeploymentConfig{
				DeploymentID: deploymentID,
				Image:        image,
				Domain:       deployment.Domain,
				Port:         port,
				EnvVars:      envVars,
				Labels:       map[string]string{},
				Memory:       memory,
				CPUShares:    cpuShares,
				Replicas:     replicas,
				StartCommand: deployment.StartCommand,
			}

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
		containerInfo, err := dm.dockerClient.ContainerInspect(ctx, location.ContainerID)
		if err != nil {
			// Container doesn't exist - try to recreate it
			logger.Warn("[DeploymentManager] Container %s doesn't exist, attempting to recreate deployment", location.ContainerID[:12])

			// Get deployment from database to recreate containers
			var deployment database.Deployment
			if err := database.DB.Where("id = ?", deploymentID).First(&deployment).Error; err != nil {
				logger.Warn("[DeploymentManager] Failed to get deployment from database for recreation: %v", err)
				continue
			}

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
				if port == 0 {
					if deployment.Port != nil && *deployment.Port > 0 {
						port = int(*deployment.Port)
						logger.Info("[StartDeployment] Using deployment port %d (no routing found) for deployment %s (recreate)", port, deploymentID)
					} else {
						logger.Warn("[StartDeployment] Deployment %s has no port configured, skipping recreation", deploymentID)
						continue
					}
				}
				
				memory := int64(512 * 1024 * 1024) // Default 512MB
				if deployment.MemoryBytes != nil {
					memory = *deployment.MemoryBytes
				}
				cpuShares := int64(1024) // Default
				if deployment.CPUShares != nil {
					cpuShares = *deployment.CPUShares
				}
				replicas := 1 // Default
				if deployment.Replicas != nil {
					replicas = int(*deployment.Replicas)
				}

				config := &DeploymentConfig{
					DeploymentID: deploymentID,
					Image:        image,
					Domain:       deployment.Domain,
					Port:         port,
					EnvVars:      envVars,
					Labels:       map[string]string{},
					Memory:       memory,
					CPUShares:    cpuShares,
					Replicas:     replicas,
					StartCommand: deployment.StartCommand,
				}

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
				containerInfo, err = dm.dockerClient.ContainerInspect(ctx, location.ContainerID)
				if err != nil {
					logger.Warn("[DeploymentManager] Failed to inspect recreated container: %v", err)
					continue
				}
			} else {
				// Compose-based deployment - skip this container
				logger.Warn("[DeploymentManager] Container %s doesn't exist for compose deployment %s, skipping", location.ContainerID[:12], deploymentID)
				continue
			}
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
		_, err := dm.dockerClient.ContainerInspect(ctx, location.ContainerID)
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
	if port == 0 {
		if deployment.Port != nil && *deployment.Port > 0 {
			port = int(*deployment.Port)
			logger.Info("[DeploymentManager] Using deployment port %d (no routing found) for restart", port)
		} else {
			return fmt.Errorf("deployment %s has no port configured in routing rules or deployment settings", deploymentID)
		}
	}

	// Get resource limits
	memory := int64(512 * 1024 * 1024) // Default 512MB
	if deployment.MemoryBytes != nil {
		memory = *deployment.MemoryBytes
	}
	cpuShares := int64(1024) // Default
	if deployment.CPUShares != nil {
		cpuShares = *deployment.CPUShares
	}
	replicas := 1 // Default
	if deployment.Replicas != nil {
		replicas = int(*deployment.Replicas)
	}

	// Create deployment config
	config := &DeploymentConfig{
		DeploymentID: deploymentID,
		Image:        image,
		Domain:       deployment.Domain,
		Port:         port,
		EnvVars:      envVars,
		Labels:       map[string]string{},
		Memory:       memory,
		CPUShares:    cpuShares,
		Replicas:     replicas,
		StartCommand: deployment.StartCommand,
	}

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
			logs, err := dm.dockerHelper.ContainerLogs(ctx, location.ContainerID, tail, false) // follow=false for non-streaming logs
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
		
		// First, try to remove existing stack (ignore errors if it doesn't exist)
		rmArgs := []string{"stack", "rm", projectName}
		rmCmd := exec.CommandContext(ctx, "docker", rmArgs...)
		rmCmd.Run() // Ignore errors - stack might not exist
		
		// Wait a moment for stack removal to complete
		time.Sleep(2 * time.Second)
		
		// Deploy as a Swarm stack - this creates Swarm services that Traefik can discover
		args := []string{"stack", "deploy", "-c", composeFile, "--with-registry-auth", projectName}
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
func (dm *DeploymentManager) Close() error {
	if err := dm.nodeSelector.Close(); err != nil {
		logger.Info("[DeploymentManager] Error closing node selector: %v", err)
	}
	if err := dm.registry.Close(); err != nil {
		logger.Info("[DeploymentManager] Error closing registry: %v", err)
	}
	return dm.dockerClient.Close()
}
