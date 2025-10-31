package orchestrator

import (
	"context"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	"api/docker"
	"api/internal/database"
	"api/internal/registry"

	"github.com/docker/go-connections/nat"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
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
		LoadBalancerAlgo:  "round-robin",
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
	
	// Filter routings for this service name
	serviceRoutings := []database.DeploymentRouting{}
	for _, routing := range routings {
		if routing.ServiceName == serviceName || (serviceName == "default" && routing.ServiceName == "") {
			serviceRoutings = append(serviceRoutings, routing)
		}
	}
	
	// If no specific routing found, create default
	if len(serviceRoutings) == 0 {
		labels["traefik.http.routers."+deploymentID+"-default.rule"] = "Host(`localhost`)"
		labels["traefik.http.routers."+deploymentID+"-default.entrypoints"] = "web"
		labels["traefik.http.services."+deploymentID+"-default.loadbalancer.server.port"] = "80"
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
		
		// Load balancer algorithm
		if routing.LoadBalancerAlgo != "" && routing.LoadBalancerAlgo != "round-robin" {
			labels["traefik.http.services."+serviceNameLabel+".loadbalancer.sticky.cookie"] = "true"
		}
	}
	
	return labels
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
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}

	containerPort := nat.Port(fmt.Sprintf("%d/tcp", config.Port))
	exposedPorts[containerPort] = struct{}{}
	portBindings[containerPort] = []nat.PortBinding{
		{
			HostIP:   "0.0.0.0",
			HostPort: "0", // Let Docker assign a random port
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
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		RestartPolicy: container.RestartPolicy{
			Name: "unless-stopped",
		},
		Resources: container.Resources{
			Memory:    config.Memory,
			CPUShares: config.CPUShares,
		},
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
