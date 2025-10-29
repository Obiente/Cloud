package orchestrator

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"api/internal/database"
	"api/internal/registry"

	"github.com/docker/go-connections/nat"
	"github.com/moby/moby/api/types"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// DeploymentManager manages the lifecycle of user deployments
type DeploymentManager struct {
	dockerClient *client.Client
	nodeSelector *NodeSelector
	registry     *registry.ServiceRegistry
	networkName  string
	nodeID       string
	nodeHostname string
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

	return &DeploymentManager{
		dockerClient: cli,
		nodeSelector: nodeSelector,
		registry:     serviceRegistry,
		networkName:  "obiente-network",
		nodeID:       info.Swarm.NodeID,
		nodeHostname: info.Name,
	}, nil
}

// CreateDeployment creates a new deployment on the cluster
func (dm *DeploymentManager) CreateDeployment(ctx context.Context, config *DeploymentConfig) error {
	log.Printf("[DeploymentManager] Creating deployment %s", config.DeploymentID)

	// Select best node for deployment
	targetNode, err := dm.nodeSelector.SelectNode(ctx)
	if err != nil {
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

	// Create containers for each replica
	for i := 0; i < config.Replicas; i++ {
		containerName := fmt.Sprintf("%s-replica-%d", config.DeploymentID, i)

		containerID, err := dm.createContainer(ctx, config, containerName, i)
		if err != nil {
			return fmt.Errorf("failed to create container: %w", err)
		}

		// Start container
		if err := dm.dockerClient.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
			return fmt.Errorf("failed to start container: %w", err)
		}

		// Get container details
		containerInfo, err := dm.dockerClient.ContainerInspect(ctx, containerID)
		if err != nil {
			return fmt.Errorf("failed to inspect container: %w", err)
		}

		// Determine the public port
		publicPort := config.Port
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

		log.Printf("[DeploymentManager] Successfully created container %s for deployment %s",
			containerID[:12], config.DeploymentID)
	}

	// Create deployment routing
	routing := &database.DeploymentRouting{
		DeploymentID:     config.DeploymentID,
		Domain:           config.Domain,
		TargetPort:       config.Port,
		Protocol:         "http",
		LoadBalancerAlgo: "round-robin",
		SSLEnabled:       true,
		SSLCertResolver:  "letsencrypt",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := database.UpsertDeploymentRouting(routing); err != nil {
		log.Printf("[DeploymentManager] Warning: Failed to create routing: %v", err)
	}

	log.Printf("[DeploymentManager] Deployment %s created successfully", config.DeploymentID)
	return nil
}

// createContainer creates a single container
func (dm *DeploymentManager) createContainer(ctx context.Context, config *DeploymentConfig, name string, replicaIndex int) (string, error) {
	// Prepare labels
	labels := map[string]string{
		"com.obiente.managed":       "true",
		"com.obiente.deployment_id": config.DeploymentID,
		"com.obiente.domain":        config.Domain,
		"com.obiente.replica":       strconv.Itoa(replicaIndex),
		// Traefik labels for automatic routing
		"traefik.enable": "true",
		"traefik.http.routers." + config.DeploymentID + ".rule":                      "Host(`" + config.Domain + "`)",
		"traefik.http.routers." + config.DeploymentID + ".entrypoints":               "websecure",
		"traefik.http.routers." + config.DeploymentID + ".tls.certresolver":          "letsencrypt",
		"traefik.http.services." + config.DeploymentID + ".loadbalancer.server.port": strconv.Itoa(config.Port),
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
		if err := dm.dockerClient.ContainerStop(ctx, location.ContainerID, container.StopOptions{Timeout: &timeout}); err != nil {
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
		dm.dockerClient.ContainerStop(ctx, location.ContainerID, container.StopOptions{Timeout: &timeout})

		// Remove container
		if err := dm.dockerClient.ContainerRemove(ctx, location.ContainerID, container.RemoveOptions{
			Force: true,
		}); err != nil {
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
		if err := dm.dockerClient.ContainerRestart(ctx, location.ContainerID, container.StopOptions{Timeout: &timeout}); err != nil {
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
			dm.dockerClient.ContainerStop(ctx, location.ContainerID, container.StopOptions{Timeout: &timeout})
			dm.dockerClient.ContainerRemove(ctx, location.ContainerID, container.RemoveOptions{Force: true})
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
			options := container.LogsOptions{
				ShowStdout: true,
				ShowStderr: true,
				Tail:       tail,
			}

			logs, err := dm.dockerClient.ContainerLogs(ctx, location.ContainerID, options)
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
func (dm *DeploymentManager) GetDeploymentStats(ctx context.Context, deploymentID string) ([]types.StatsJSON, error) {
	locations, err := dm.registry.GetDeploymentLocations(deploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment locations: %w", err)
	}

	var stats []types.StatsJSON
	for _, location := range locations {
		if location.NodeID != dm.nodeID {
			continue
		}

		containerStats, err := dm.dockerClient.ContainerStats(ctx, location.ContainerID, false)
		if err != nil {
			log.Printf("[DeploymentManager] Failed to get stats for %s: %v", location.ContainerID, err)
			continue
		}
		defer containerStats.Body.Close()

		var stat types.StatsJSON
		// In production, properly decode the stats JSON
		stats = append(stats, stat)
	}

	return stats, nil
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
