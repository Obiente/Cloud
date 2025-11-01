package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"api/docker"
	"api/internal/database"
	"api/internal/registry"

	"github.com/docker/go-connections/nat"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/filters"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"gopkg.in/yaml.v3"
)

// DeploymentManager manages the lifecycle of user deployments
type DeploymentManager struct {
    dockerClient client.APIClient
    dockerHelper dockerHelper
    nodeSelector *NodeSelector
    registry     *registry.ServiceRegistry
    networkName  string
    nodeID       string
    nodeHostname string
}

// dockerHelper defines the subset of docker helper methods used here.
type dockerHelper interface {
    StartContainer(ctx context.Context, containerID string) error
    StopContainer(ctx context.Context, containerID string, timeout time.Duration) error
    RemoveContainer(ctx context.Context, containerID string, force bool) error
    RestartContainer(ctx context.Context, containerID string, timeout time.Duration) error
    ContainerLogs(ctx context.Context, containerID string, tail string, follow bool) (io.ReadCloser, error)
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

	return &DeploymentManager{
        dockerClient: cli,
        dockerHelper: helper,
		nodeSelector: nodeSelector,
		registry:     serviceRegistry,
		networkName:  "obiente-network",
		nodeID:       nodeID,
		nodeHostname: info.Name,
	}, nil
}

// CreateDeployment creates a new deployment on the cluster
func (dm *DeploymentManager) CreateDeployment(ctx context.Context, config *DeploymentConfig) error {
	log.Printf("[DeploymentManager] Creating deployment %s", config.DeploymentID)

	// Select best node for deployment
	targetNode, err := dm.nodeSelector.SelectNode(ctx)
	if err != nil {
		log.Printf("[DeploymentManager] ERROR: Failed to select node for deployment %s: %v", config.DeploymentID, err)
		return fmt.Errorf("failed to select node: %w", err)
	}

	log.Printf("[DeploymentManager] Selected node %s (%s) for deployment %s",
		targetNode.ID, targetNode.Hostname, config.DeploymentID)

	// Check if we're on the target node
	if targetNode.ID != dm.nodeID {
		// TODO: Forward request to the correct node's API
		return fmt.Errorf("deployment should be created on node %s, but we're on %s",
			targetNode.ID, dm.nodeID)
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

	// Create containers for each service and replica
	for _, serviceName := range serviceNames {
		for i := 0; i < config.Replicas; i++ {
			containerName := fmt.Sprintf("%s-%s-replica-%d", config.DeploymentID, serviceName, i)

			containerID, err := dm.createContainer(ctx, config, containerName, i, serviceName)
			if err != nil {
				return fmt.Errorf("failed to create container: %w", err)
			}

			// Start container
			if err := dm.dockerHelper.StartContainer(ctx, containerID); err != nil {
				return fmt.Errorf("failed to start container: %w", err)
			}

			// Get container details
			containerInfo, err := dm.dockerClient.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("failed to inspect container: %w", err)
			}

			// Determine the public port (find port for this service from routing)
			publicPort := config.Port
			for _, routing := range routings {
				if routing.ServiceName == serviceName || (serviceName == "default" && routing.ServiceName == "") {
					publicPort = routing.TargetPort
					break
				}
			}
			if len(containerInfo.NetworkSettings.Ports) > 0 {
				for _, bindings := range containerInfo.NetworkSettings.Ports {
					if len(bindings) > 0 {
						if port, err := strconv.Atoi(bindings[0].HostPort); err == nil {
							publicPort = port
						}
					}
				}
			}

			// Register deployment location
			location := &database.DeploymentLocation{
				ID:           fmt.Sprintf("loc-%s-%s", config.DeploymentID, containerID[:12]),
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
				log.Printf("[DeploymentManager] Warning: Failed to register deployment: %v", err)
			}

			log.Printf("[DeploymentManager] Successfully created container %s for deployment %s (service: %s)",
				containerID[:12], config.DeploymentID, serviceName)
		}
	}

	// Create default deployment routing (for backward compatibility)
	routing := &database.DeploymentRouting{
		ID:                fmt.Sprintf("route-%s", config.DeploymentID),
		DeploymentID:      config.DeploymentID,
		Domain:            config.Domain,
		ServiceName:       "default",
		TargetPort:        config.Port,
		Protocol:          "http",
		SSLEnabled:        true,
		SSLCertResolver:   "letsencrypt",
		Middleware:        "{}", // Empty JSON object for jsonb field
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := database.UpsertDeploymentRouting(routing); err != nil {
		log.Printf("[DeploymentManager] Warning: Failed to create routing: %v", err)
	}

	log.Printf("[DeploymentManager] Deployment %s created successfully", config.DeploymentID)
	return nil
}

// generateTraefikLabels generates Traefik labels from routing rules
func generateTraefikLabels(deploymentID string, serviceName string, routings []database.DeploymentRouting) map[string]string {
	labels := make(map[string]string)
	labels["traefik.enable"] = "true"
	labels["com.obiente.traefik"] = "true" // Required for Traefik to discover this container
	
	// Filter routings for this service name
	serviceRoutings := []database.DeploymentRouting{}
	for _, routing := range routings {
		if routing.ServiceName == serviceName || (serviceName == "default" && routing.ServiceName == "") {
			serviceRoutings = append(serviceRoutings, routing)
		}
	}
	
	// If no specific routing found, don't create any routing rules
	// User must configure routing before the service will be accessible
	if len(serviceRoutings) == 0 {
		return labels
	}
	
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
		
		// Entrypoints
		if routing.SSLEnabled {
			labels["traefik.http.routers."+routerName+".entrypoints"] = "websecure"
			if routing.SSLCertResolver != "" && routing.SSLCertResolver != "internal" {
				labels["traefik.http.routers."+routerName+".tls.certresolver"] = routing.SSLCertResolver
			} else if routing.SSLCertResolver == "internal" {
				// For internal SSL, don't set certresolver (let app handle it)
				labels["traefik.http.routers."+routerName+".entrypoints"] = "web"
			}
		} else {
			labels["traefik.http.routers."+routerName+".entrypoints"] = "web"
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

	// Get deployment domain from routings if available
	var deploymentDomain string
	if len(routings) > 0 {
		deploymentDomain = routings[0].Domain
	}

	// Inject labels into each service
	if services, ok := compose["services"].(map[string]interface{}); ok {
		for serviceName, serviceData := range services {
			if service, ok := serviceData.(map[string]interface{}); ok {
				// Generate Traefik labels for this service
				traefikLabels := generateTraefikLabels(deploymentID, serviceName, routings)
				
				// Get or create labels map for this service
				var labels map[string]interface{}
				if existingLabels, ok := service["labels"].(map[string]interface{}); ok {
					labels = existingLabels
				} else if existingLabelsList, ok := service["labels"].([]interface{}); ok {
					// Convert list format to map format
					labels = make(map[string]interface{})
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
				} else {
					labels = make(map[string]interface{})
				}
				
				// Merge Traefik labels (Traefik labels take precedence)
				for k, v := range traefikLabels {
					labels[k] = v
				}
				
				// Add management labels
				labels["com.obiente.managed"] = "true"
				labels["com.obiente.deployment_id"] = deploymentID
				labels["com.obiente.service_name"] = serviceName
				labels["com.obiente.traefik"] = "true" // Required for Traefik discovery
				if deploymentDomain != "" {
					labels["com.obiente.domain"] = deploymentDomain
				}
				
				// Update service with labels
				service["labels"] = labels
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

// createContainer creates a single container
func (dm *DeploymentManager) createContainer(ctx context.Context, config *DeploymentConfig, name string, replicaIndex int, serviceName string) (string, error) {
	// Get routing rules for this deployment
	routings, _ := database.GetDeploymentRoutings(config.DeploymentID)
	
	// Prepare labels
	labels := map[string]string{
		"com.obiente.managed":       "true",
		"com.obiente.deployment_id": config.DeploymentID,
		"com.obiente.domain":        config.Domain,
		"com.obiente.service_name":  serviceName,
		"com.obiente.replica":       strconv.Itoa(replicaIndex),
		"com.obiente.traefik":       "true", // Required for Traefik discovery
	}
	
	// Generate Traefik labels from routing rules
	traefikLabels := generateTraefikLabels(config.DeploymentID, serviceName, routings)
	for k, v := range traefikLabels {
		labels[k] = v
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

	// Prepare port bindings
	// SECURITY: Always use random host ports (HostPort: "0") to prevent users from binding to specific host ports
	// This prevents port conflicts and unauthorized access to host services
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}

	containerPort := nat.Port(fmt.Sprintf("%d/tcp", config.Port))
	exposedPorts[containerPort] = struct{}{}
	portBindings[containerPort] = []nat.PortBinding{
		{
			HostIP:   "0.0.0.0",
			HostPort: "0", // SECURITY: Docker assigns random port - users cannot bind to specific host ports
		},
	}

	// Container configuration
	containerConfig := &container.Config{
		Image:        config.Image,
		Env:          env,
		Labels:       labels,
		ExposedPorts: exposedPorts,
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD-SHELL", "wget --no-verbose --tries=1 --spider http://localhost:" + strconv.Itoa(config.Port) + "/health || exit 1"},
			Interval: 30 * time.Second,
			Timeout:  10 * time.Second,
			Retries:  3,
		},
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
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

// StartDeployment starts all containers for a deployment
func (dm *DeploymentManager) StartDeployment(ctx context.Context, deploymentID string) error {
	log.Printf("[DeploymentManager] Starting deployment %s", deploymentID)

	locations, err := dm.registry.GetDeploymentLocations(deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get deployment locations: %w", err)
	}

	// Check if we have any containers for this deployment
	if len(locations) == 0 {
		log.Printf("[DeploymentManager] No containers found for deployment %s, need to create them", deploymentID)
		return fmt.Errorf("no containers found for deployment %s - deployment may need to be created first", deploymentID)
	}

	for _, location := range locations {
		// Only start containers on this node
		if location.NodeID != dm.nodeID {
			continue
		}

		// Check if container exists and is stopped
		containerInfo, err := dm.dockerClient.ContainerInspect(ctx, location.ContainerID)
		if err != nil {
			log.Printf("[DeploymentManager] Container %s not found or error inspecting: %v", location.ContainerID[:12], err)
			continue
		}

		// Only start if not already running
		if !containerInfo.State.Running {
			// Start container
			if err := dm.dockerHelper.StartContainer(ctx, location.ContainerID); err != nil {
				log.Printf("[DeploymentManager] Failed to start container %s: %v", location.ContainerID[:12], err)
				continue
			}

			// Update status
			database.DB.Model(&database.DeploymentLocation{}).
				Where("container_id = ?", location.ContainerID).
				Update("status", "running")

			log.Printf("[DeploymentManager] Started container %s", location.ContainerID[:12])
		} else {
			log.Printf("[DeploymentManager] Container %s is already running", location.ContainerID[:12])
		}
	}

	return nil
}

// StopDeployment stops all containers for a deployment
func (dm *DeploymentManager) StopDeployment(ctx context.Context, deploymentID string) error {
	log.Printf("[DeploymentManager] Stopping deployment %s", deploymentID)

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
			log.Printf("[DeploymentManager] Failed to stop container %s: %v", location.ContainerID, err)
			continue
		}

		// Update status
		database.DB.Model(&database.DeploymentLocation{}).
			Where("container_id = ?", location.ContainerID).
			Update("status", "stopped")

		log.Printf("[DeploymentManager] Stopped container %s", location.ContainerID[:12])
	}

	return nil
}

// DeleteDeployment removes all containers and data for a deployment
func (dm *DeploymentManager) DeleteDeployment(ctx context.Context, deploymentID string) error {
	log.Printf("[DeploymentManager] Deleting deployment %s", deploymentID)

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
			log.Printf("[DeploymentManager] Failed to remove container %s: %v", location.ContainerID, err)
			continue
		}

		// Unregister from registry
		if err := dm.registry.UnregisterDeployment(ctx, location.ContainerID); err != nil {
			log.Printf("[DeploymentManager] Failed to unregister deployment: %v", err)
		}

		log.Printf("[DeploymentManager] Deleted container %s", location.ContainerID[:12])
	}

	return nil
}

// RestartDeployment restarts all containers for a deployment
func (dm *DeploymentManager) RestartDeployment(ctx context.Context, deploymentID string) error {
	log.Printf("[DeploymentManager] Restarting deployment %s", deploymentID)

	locations, err := dm.registry.GetDeploymentLocations(deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get deployment locations: %w", err)
	}

	for _, location := range locations {
		// Only restart containers on this node
		if location.NodeID != dm.nodeID {
			continue
		}

        timeout := int(30)
        if err := dm.dockerHelper.RestartContainer(ctx, location.ContainerID, time.Duration(timeout)*time.Second); err != nil {
			log.Printf("[DeploymentManager] Failed to restart container %s: %v", location.ContainerID, err)
			continue
		}

		// Update status
		database.DB.Model(&database.DeploymentLocation{}).
			Where("container_id = ?", location.ContainerID).
			Updates(map[string]interface{}{
				"status":     "running",
				"updated_at": time.Now(),
			})

		log.Printf("[DeploymentManager] Restarted container %s", location.ContainerID[:12])
	}

	return nil
}

// ScaleDeployment changes the number of replicas for a deployment
func (dm *DeploymentManager) ScaleDeployment(ctx context.Context, deploymentID string, replicas int) error {
	log.Printf("[DeploymentManager] Scaling deployment %s to %d replicas", deploymentID, replicas)

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

			log.Printf("[DeploymentManager] Removed replica %s", location.ContainerID[:12])
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
	log.Printf("[DeploymentManager] Deploying compose file for deployment %s", deploymentID)

	// Sanitize compose file for security (transform volumes, remove host ports, etc.)
	sanitizer := NewComposeSanitizer(deploymentID)
	sanitizedYaml, err := sanitizer.SanitizeComposeYAML(composeYaml)
	if err != nil {
		log.Printf("[DeploymentManager] Warning: Failed to sanitize compose YAML for deployment %s: %v. Using original YAML.", deploymentID, err)
		// Continue with original YAML if sanitization fails (but log the warning)
		sanitizedYaml = composeYaml
	} else {
		log.Printf("[DeploymentManager] Sanitized compose YAML for deployment %s (volumes mapped to: %s)", deploymentID, sanitizer.GetSafeBaseDir())
	}

	// Get routing rules (create default if none exist)
	routings, _ := database.GetDeploymentRoutings(deploymentID)
	if len(routings) == 0 {
		// Try to parse compose file to detect port, otherwise use default
		var targetPort int = 8080
		
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
		
		// Create default routing for compose deployment
		defaultRouting := &database.DeploymentRouting{
			ID:                fmt.Sprintf("route-%s-default", deploymentID),
			DeploymentID:      deploymentID,
			Domain:            "", // Domain can be set later through routing UI
			ServiceName:       "default",
			TargetPort:        targetPort,
			Protocol:          "http",
			SSLEnabled:        true,
			SSLCertResolver:   "letsencrypt",
			Middleware:        "{}",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
		
		if err := database.UpsertDeploymentRouting(defaultRouting); err != nil {
			log.Printf("[DeploymentManager] Warning: Failed to create default routing: %v", err)
		} else {
			routings = []database.DeploymentRouting{*defaultRouting}
			log.Printf("[DeploymentManager] Created default routing for compose deployment %s (port: %d)", deploymentID, targetPort)
		}
	}

	// Inject Traefik labels into compose file based on routing rules
	labeledYaml, err := dm.injectTraefikLabelsIntoCompose(sanitizedYaml, deploymentID, routings)
	if err != nil {
		log.Printf("[DeploymentManager] Warning: Failed to inject Traefik labels into compose YAML for deployment %s: %v. Using sanitized YAML without labels.", deploymentID, err)
		labeledYaml = sanitizedYaml
	} else {
		log.Printf("[DeploymentManager] Injected Traefik labels into compose YAML for deployment %s", deploymentID)
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

	// Run docker compose up -d
	cmd := exec.CommandContext(ctx, "docker", "compose", "-p", projectName, "-f", composeFile, "up", "-d")
	cmd.Dir = deployDir
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		errorOutput := stderr.String()
		stdOutput := stdout.String()
		log.Printf("[DeploymentManager] ERROR: Failed to deploy compose file for deployment %s: %v\nStderr: %s\nStdout: %s", deploymentID, err, errorOutput, stdOutput)
		return fmt.Errorf("failed to deploy compose file: %w\nStderr: %s\nStdout: %s", err, errorOutput, stdOutput)
	}

	stdOutput := stdout.String()
	log.Printf("[DeploymentManager] Docker compose up output for deployment %s:\n%s", deploymentID, stdOutput)
	log.Printf("[DeploymentManager] Successfully deployed compose file for deployment %s (project: %s)", deploymentID, projectName)

	// Wait a moment for containers to be fully created and started
	time.Sleep(1 * time.Second)

	// List containers created by this compose project and register them
	return dm.registerComposeContainers(ctx, deploymentID, projectName)
}

// registerComposeContainers finds containers created by a compose project and registers them
func (dm *DeploymentManager) registerComposeContainers(ctx context.Context, deploymentID string, projectName string) error {
	// List containers with the project label
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
		log.Printf("[DeploymentManager] No containers found with project name %s, trying lowercase version", projectName)
		filterArgsLower := filters.NewArgs()
		filterArgsLower.Add("label", fmt.Sprintf("com.docker.compose.project=%s", strings.ToLower(projectName)))
		
		containers, err = dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{
			All:     true,
			Filters: filterArgsLower,
		})
		if err != nil {
			log.Printf("[DeploymentManager] Failed to list containers with lowercase project name: %v", err)
		}
	}

	// Also try listing all containers with compose labels and filter manually (fallback)
	if len(containers) == 0 {
		log.Printf("[DeploymentManager] Still no containers found, listing all containers with compose labels")
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
					log.Printf("[DeploymentManager] Found container %s with project label: %s", cnt.ID[:12], projectLabel)
				}
			}
		}
	}

	if len(containers) == 0 {
		log.Printf("[DeploymentManager] WARNING: No containers found for compose project %s (deployment %s). "+
			"This might indicate the compose file failed to create containers. Checking all containers...", projectName, deploymentID)
		
		// Last resort: list all containers to see what exists
		allContainers, err := dm.dockerClient.ContainerList(ctx, client.ContainerListOptions{All: true})
		if err == nil {
			log.Printf("[DeploymentManager] Total containers on system: %d", len(allContainers))
			for i, cnt := range allContainers {
				if i < 5 { // Log first 5 containers for debugging
					projectLabel := cnt.Labels["com.docker.compose.project"]
					log.Printf("[DeploymentManager] Container %s: compose project label = '%s'", cnt.ID[:12], projectLabel)
				}
			}
		}
		
		return fmt.Errorf("no containers found for compose project %s", projectName)
	}

	log.Printf("[DeploymentManager] Found %d container(s) for compose project %s", len(containers), projectName)

	// Get routing rules
	routings, _ := database.GetDeploymentRoutings(deploymentID)

	var runningCount int
	for _, cnt := range containers {
		// Verify container is actually running by inspecting it
		containerInfo, err := dm.dockerClient.ContainerInspect(ctx, cnt.ID)
		if err != nil {
			log.Printf("[DeploymentManager] Warning: Failed to inspect container %s: %v", cnt.ID[:12], err)
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

		// Determine public port
		publicPort := 8080 // Default
		for _, routing := range routings {
			if routing.ServiceName == serviceName || (serviceName == "default" && routing.ServiceName == "") {
				publicPort = routing.TargetPort
				break
			}
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
			log.Printf("[DeploymentManager] Warning: Failed to register compose container %s: %v", cnt.ID[:12], err)
		} else {
			log.Printf("[DeploymentManager] Registered compose container %s (service: %s, status: %s) for deployment %s",
				cnt.ID[:12], serviceName, containerStatus, deploymentID)
		}
	}

	if runningCount == 0 {
		return fmt.Errorf("no running containers found for compose project %s (%d containers found but all are stopped)", projectName, len(containers))
	}

	log.Printf("[DeploymentManager] Successfully registered %d running container(s) for deployment %s", runningCount, deploymentID)
	return nil
}

// StopComposeDeployment stops containers created by a compose file
func (dm *DeploymentManager) StopComposeDeployment(ctx context.Context, deploymentID string) error {
	log.Printf("[DeploymentManager] Stopping compose deployment %s", deploymentID)

	projectName := fmt.Sprintf("deploy-%s", deploymentID)

	// Find compose file directory - we stored it, but since we clean up temp dirs,
	// we need to find containers by label and stop them individually
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
		timeout := 30 * time.Second
		if err := dm.dockerHelper.StopContainer(ctx, cnt.ID, timeout); err != nil {
			log.Printf("[DeploymentManager] Failed to stop compose container %s: %v", cnt.ID[:12], err)
		} else {
			log.Printf("[DeploymentManager] Stopped compose container %s", cnt.ID[:12])
		}
	}

	return nil
}

// RemoveComposeDeployment removes containers created by a compose file
func (dm *DeploymentManager) RemoveComposeDeployment(ctx context.Context, deploymentID string) error {
	log.Printf("[DeploymentManager] Removing compose deployment %s", deploymentID)

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
			log.Printf("[DeploymentManager] Failed to remove compose container %s: %v", cnt.ID[:12], err)
		} else {
			log.Printf("[DeploymentManager] Removed compose container %s", cnt.ID[:12])
			// Unregister
			_ = dm.registry.UnregisterDeployment(ctx, cnt.ID)
		}
	}

	return nil
}

// Close closes all connections
func (dm *DeploymentManager) Close() error {
	if err := dm.nodeSelector.Close(); err != nil {
		log.Printf("[DeploymentManager] Error closing node selector: %v", err)
	}
	if err := dm.registry.Close(); err != nil {
		log.Printf("[DeploymentManager] Error closing registry: %v", err)
	}
	return dm.dockerClient.Close()
}

