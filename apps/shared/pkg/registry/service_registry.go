package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/utils"

	"github.com/moby/moby/client"
)

// ServiceRegistry tracks all deployments across the cluster
type ServiceRegistry struct {
	dockerClient   client.APIClient
	cache          sync.Map // In-memory cache for fast lookups
	mu             sync.RWMutex
	nodeID         string
	nodeHostname   string
	isSwarmManager bool
}

// DeploymentInfo holds information about a deployment's location and status
type DeploymentInfo struct {
	DeploymentID string                        `json:"deployment_id"`
	Locations    []database.DeploymentLocation `json:"locations"`
	UpdatedAt    time.Time                     `json:"updated_at"`
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry() (*ServiceRegistry, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Get current node information
	infoResult, err := cli.Info(context.Background(), client.InfoOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker info: %w", err)
	}
	info := infoResult.Info

	// Determine node ID - respect ENABLE_SWARM environment variable
	// If ENABLE_SWARM=false, always use local- prefix even if Swarm is enabled in Docker
	var nodeID string
	if utils.IsSwarmModeEnabled() {
		// Swarm mode enabled - use Swarm node ID if available
		if info.Swarm.NodeID != "" {
			nodeID = info.Swarm.NodeID
		} else {
			// Swarm enabled but not in Swarm - use synthetic ID
			nodeID = info.Name
			if nodeID == "" {
				nodeID = "local-node"
			} else {
				nodeID = "local-" + nodeID
			}
		}
	} else {
		// Swarm mode disabled - always use local- prefix
		nodeID = info.Name
		if nodeID == "" {
			nodeID = "local-node"
		} else {
			nodeID = "local-" + nodeID
		}
	}

	registry := &ServiceRegistry{
		dockerClient:   cli,
		nodeID:         nodeID,
		nodeHostname:   info.Name,
		isSwarmManager: info.Swarm.ControlAvailable,
	}

	return registry, nil
}

// RegisterDeployment registers a new deployment location
func (sr *ServiceRegistry) RegisterDeployment(ctx context.Context, location *database.DeploymentLocation) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	// Save to database
	if err := database.RecordDeploymentLocation(location); err != nil {
		return fmt.Errorf("failed to record deployment location: %w", err)
	}

	// Update cache
	sr.updateCache(location.DeploymentID)

	logger.Info("[Registry] Registered deployment %s on node %s (container: %s)",
		location.DeploymentID, location.NodeID, location.ContainerID)

	return nil
}

// UnregisterDeployment removes a deployment location
func (sr *ServiceRegistry) UnregisterDeployment(ctx context.Context, containerID string) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	// Get deployment info before deletion
	var location database.DeploymentLocation
	if err := database.DB.Where("container_id = ?", containerID).First(&location).Error; err != nil {
		return fmt.Errorf("deployment location not found: %w", err)
	}

	deploymentID := location.DeploymentID

	// Remove from database
	if err := database.RemoveDeploymentLocation(containerID); err != nil {
		return fmt.Errorf("failed to remove deployment location: %w", err)
	}

	// Update cache
	sr.updateCache(deploymentID)

	logger.Info("[Registry] Unregistered deployment %s from node %s (container: %s)",
		deploymentID, location.NodeID, containerID)

	return nil
}

// GetDeploymentLocations returns all locations where a deployment is running
func (sr *ServiceRegistry) GetDeploymentLocations(deploymentID string) ([]database.DeploymentLocation, error) {
	// Try cache first
	if cached, ok := sr.cache.Load(deploymentID); ok {
		if info, ok := cached.(*DeploymentInfo); ok {
			// Check if cache is fresh (less than 30 seconds old)
			if time.Since(info.UpdatedAt) < 30*time.Second {
				return info.Locations, nil
			}
		}
	}

	// Cache miss or stale, fetch from database
	locations, err := database.GetDeploymentLocations(deploymentID)
	if err != nil {
		return nil, err
	}

	// Update cache
	sr.cache.Store(deploymentID, &DeploymentInfo{
		DeploymentID: deploymentID,
		Locations:    locations,
		UpdatedAt:    time.Now(),
	})

	return locations, nil
}

// GetDeploymentByDomain finds a deployment by its domain name
func (sr *ServiceRegistry) GetDeploymentByDomain(domain string) (*database.DeploymentRouting, error) {
	routing, err := database.GetDeploymentRouting(domain)
	if err != nil {
		return nil, fmt.Errorf("deployment not found for domain %s: %w", domain, err)
	}
	return routing, nil
}

// GetNodeDeployments returns all deployments on a specific node
func (sr *ServiceRegistry) GetNodeDeployments(nodeID string) ([]database.DeploymentLocation, error) {
	var locations []database.DeploymentLocation
	err := database.DB.Where("node_id = ? AND status = ?", nodeID, "running").Find(&locations).Error
	return locations, err
}

// GetNodeGameServers returns all game servers on a specific node
func (sr *ServiceRegistry) GetNodeGameServers(nodeID string) ([]database.GameServerLocation, error) {
	var locations []database.GameServerLocation
	err := database.DB.Where("node_id = ? AND status = ?", nodeID, "running").Find(&locations).Error
	return locations, err
}

// GetAllDeployments returns all active deployments in the cluster
func (sr *ServiceRegistry) GetAllDeployments() ([]database.DeploymentLocation, error) {
	var locations []database.DeploymentLocation
	err := database.DB.Where("status = ?", "running").Find(&locations).Error
	return locations, err
}

// UpdateDeploymentHealth updates the health status of a deployment
func (sr *ServiceRegistry) UpdateDeploymentHealth(containerID string, healthStatus string) error {
	return database.DB.Model(&database.DeploymentLocation{}).
		Where("container_id = ?", containerID).
		Updates(map[string]interface{}{
			"health_status":     healthStatus,
			"last_health_check": time.Now(),
		}).Error
}

// UpdateDeploymentMetrics updates resource usage metrics for a deployment
func (sr *ServiceRegistry) UpdateDeploymentMetrics(containerID string, cpuUsage float64, memoryUsage int64) error {
	return database.DB.Model(&database.DeploymentLocation{}).
		Where("container_id = ?", containerID).
		Updates(map[string]interface{}{
			"cpu_usage":    cpuUsage,
			"memory_usage": memoryUsage,
		}).Error
}

// SyncWithDocker synchronizes the registry with actual Docker containers
func (sr *ServiceRegistry) SyncWithDocker(ctx context.Context) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	logger.Debug("[Registry] Starting sync with Docker...")

	// Get all containers managed by Obiente
	filterArgs := make(client.Filters)
	filterArgs.Add("label", "cloud.obiente.managed=true")

	containersResult, err := sr.dockerClient.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}
	containers := containersResult.Items

	// Build fast-lookup index of actual containers by ID
	actualIndex := make(map[string]int, len(containers))
	for i, c := range containers {
		actualIndex[c.ID] = i
	}

	// Get all locations from database for this node
	// Only clean up locations that belong to this node to avoid removing records
	// for containers running on other nodes in Swarm mode
	var dbLocations []database.DeploymentLocation
	if err := database.DB.Where("node_id = ?", sr.nodeID).Find(&dbLocations).Error; err != nil {
		return fmt.Errorf("failed to query database: %w", err)
	}

	// Check for containers that exist in DB but not in Docker (cleanup needed)
	// Only remove if the location's node_id matches our node_id (defensive check)
	for _, location := range dbLocations {
		// Double-check that this location belongs to our node before removing
		if location.NodeID != sr.nodeID {
			logger.Debug("[Registry] Skipping location %s - belongs to different node (%s != %s)", location.ContainerID[:12], location.NodeID, sr.nodeID)
			continue
		}

		if _, exists := actualIndex[location.ContainerID]; !exists {
			// Verify container doesn't exist by attempting to inspect it
			// This prevents race conditions where container is temporarily unavailable
			_, inspectErr := sr.dockerClient.ContainerInspect(ctx, location.ContainerID, client.ContainerInspectOptions{})
			if inspectErr != nil {
				// Container truly doesn't exist - safe to remove
				logger.Info("[Registry] Container %s no longer exists on node %s, removing from registry", location.ContainerID[:12], sr.nodeID)
				if err := database.RemoveDeploymentLocation(location.ContainerID); err != nil {
					logger.Warn("[Registry] Error removing stale location: %v", err)
				}
			} else {
				// Container exists but wasn't in our list - might be a timing issue
				// Update status instead of removing
				logger.Debug("[Registry] Container %s exists but wasn't in list, updating status", location.ContainerID[:12])
				database.DB.Model(&database.DeploymentLocation{}).
					Where("container_id = ?", location.ContainerID).
					Update("status", "unknown")
			}
		}
	}

	// Check for containers in Docker but not in DB (need to register)
	dbContainerIDs := make(map[string]bool)
	for _, location := range dbLocations {
		dbContainerIDs[location.ContainerID] = true
	}

	for _, c := range containers {
		containerID := c.ID
		if !dbContainerIDs[containerID] {
			// Extract deployment info from labels
			deploymentID := c.Labels["cloud.obiente.deployment_id"]
			if deploymentID == "" {
				continue
			}

			logger.Info("[Registry] Found unregistered container %s, registering...", containerID)

			// Create location record
			location := &database.DeploymentLocation{
				ID:           fmt.Sprintf("loc-%s-%s", deploymentID, containerID[:12]),
				DeploymentID: deploymentID,
				NodeID:       sr.nodeID,
				NodeHostname: sr.nodeHostname,
				ContainerID:  containerID,
				Status:       string(c.State),
				Domain:       c.Labels["cloud.obiente.domain"],
			}

			// Extract port if available
			if len(c.Ports) > 0 {
				location.Port = int(c.Ports[0].PublicPort)
			}

			if err := database.RecordDeploymentLocation(location); err != nil {
				logger.Warn("[Registry] Error registering container: %v", err)
			}
		} else {
			// Update status for existing containers
			status := "running"
			if string(c.State) != "running" {
				status = string(c.State)
			}

			database.DB.Model(&database.DeploymentLocation{}).
				Where("container_id = ?", containerID).
				Update("status", status)
		}
	}

	if err := sr.syncSwarmServices(ctx); err != nil {
		logger.Warn("[Registry] Failed to sync swarm services: %v", err)
	}

	logger.Debug("[Registry] Sync completed. Found %d containers", len(containers))
	return nil
}

// syncSwarmServices ensures deployments running as Swarm services get registered even
// when the service is scheduled on remote nodes. This allows DNS/API lookups to work
// even if the registry is only running on a manager node.
func (sr *ServiceRegistry) syncSwarmServices(ctx context.Context) error {
	if !utils.IsSwarmModeEnabled() || !sr.isSwarmManager {
		// Workers (or non-swarm nodes) can only see their local containers, which are
		// already handled in the main sync logic above.
		return nil
	}

	servicesResult, err := sr.dockerClient.ServiceList(ctx, client.ServiceListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list swarm services: %w", err)
	}

	// Get all tasks once (more efficient than calling per service)
	tasksResult, err := sr.dockerClient.TaskList(ctx, client.TaskListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list swarm tasks: %w", err)
	}

	for _, svc := range servicesResult.Items {
		if svc.Spec.Labels["cloud.obiente.managed"] != "true" {
			continue
		}

		deploymentID := svc.Spec.Labels["cloud.obiente.deployment_id"]
		if deploymentID == "" {
			continue
		}

		domain := svc.Spec.Labels["cloud.obiente.domain"]

		activeContainers := make(map[string]bool)

		for _, task := range tasksResult.Items {
			if task.ServiceID != svc.ID {
				continue
			}
			if !isTaskActive(string(task.Status.State)) {
				continue
			}
			if task.Status.ContainerStatus == nil || task.Status.ContainerStatus.ContainerID == "" {
				continue
			}

			containerID := task.Status.ContainerStatus.ContainerID
			activeContainers[containerID] = true

			location := &database.DeploymentLocation{
				ID:              fmt.Sprintf("loc-%s-%s", deploymentID, containerID[:12]),
				DeploymentID:    deploymentID,
				NodeID:          task.NodeID,
				ContainerID:     containerID,
				ServiceID:       svc.ID,
				TaskID:          task.ID,
				Status:          string(task.Status.State),
				Domain:          domain,
				NodeHostname:    task.NodeID,
				NodeIP:          "",
				UpdatedAt:       time.Now(),
				CreatedAt:       time.Now(),
				HealthStatus:    "unknown",
				LastHealthCheck: time.Now(),
			}

			// Populate node metadata if available
			if task.NodeID != "" {
				if nodeResult, err := sr.dockerClient.NodeInspect(ctx, task.NodeID, client.NodeInspectOptions{}); err == nil {
					location.NodeHostname = nodeResult.Node.Description.Hostname
					location.NodeIP = nodeResult.Node.Status.Addr
				}
			}

			if err := database.RecordDeploymentLocation(location); err != nil {
				logger.Warn("[Registry] Failed to upsert swarm location for %s: %v", containerID[:12], err)
			} else {
				sr.updateCache(deploymentID)
			}
		}

		// Remove stale locations for this service (tasks that no longer exist)
		var existing []database.DeploymentLocation
		if err := database.DB.Where("deployment_id = ? AND service_id = ?", deploymentID, svc.ID).
			Find(&existing).Error; err == nil {
			for _, loc := range existing {
				if !activeContainers[loc.ContainerID] {
					if err := database.RemoveDeploymentLocation(loc.ContainerID); err != nil {
						logger.Warn("[Registry] Failed to remove stale swarm location %s: %v", loc.ContainerID[:12], err)
					} else {
						sr.updateCache(deploymentID)
					}
				}
			}
		}
	}

	return nil
}

func isTaskActive(state string) bool {
	switch state {
	case "new",
		"allocated",
		"pending",
		"assigned",
		"accepted",
		"preparing",
		"ready",
		"starting",
		"running":
		return true
	default:
		return false
	}
}

// GetClusterStats returns overall cluster statistics
func (sr *ServiceRegistry) GetClusterStats() (*ClusterStats, error) {
	stats := &ClusterStats{
		Timestamp: time.Now(),
	}

	// Count total deployments
	var totalDeployments int64
	database.DB.Model(&database.DeploymentLocation{}).
		Where("status = ?", "running").
		Count(&totalDeployments)
	stats.TotalDeployments = int(totalDeployments)

	// Count active nodes
	var activeNodes int64
	database.DB.Model(&database.NodeMetadata{}).
		Where("availability = ? AND status = ?", "active", "ready").
		Count(&activeNodes)
	stats.ActiveNodes = int(activeNodes)

	// Get node distribution
	var nodeStats []struct {
		NodeID string
		Count  int
	}
	database.DB.Model(&database.DeploymentLocation{}).
		Select("node_id, count(*) as count").
		Where("status = ?", "running").
		Group("node_id").
		Scan(&nodeStats)

	stats.DeploymentsPerNode = make(map[string]int)
	for _, ns := range nodeStats {
		stats.DeploymentsPerNode[ns.NodeID] = ns.Count
	}

	// Calculate total resources
	var nodes []database.NodeMetadata
	database.DB.Where("availability = ? AND status = ?", "active", "ready").Find(&nodes)

	for _, node := range nodes {
		stats.TotalCPU += node.TotalCPU
		stats.TotalMemory += node.TotalMemory
		stats.UsedCPU += node.UsedCPU
		stats.UsedMemory += node.UsedMemory
	}

	return stats, nil
}

// GetNodeStats returns statistics for a specific node
func (sr *ServiceRegistry) GetNodeStats(nodeID string) (*NodeStats, error) {
	node, err := database.GetNodeByID(nodeID)
	if err != nil {
		return nil, err
	}

	var deploymentCount int64
	database.DB.Model(&database.DeploymentLocation{}).
		Where("node_id = ? AND status = ?", nodeID, "running").
		Count(&deploymentCount)

	stats := &NodeStats{
		NodeID:          node.ID,
		Hostname:        node.Hostname,
		DeploymentCount: int(deploymentCount),
		TotalCPU:        node.TotalCPU,
		TotalMemory:     node.TotalMemory,
		UsedCPU:         node.UsedCPU,
		UsedMemory:      node.UsedMemory,
		Availability:    node.Availability,
		Status:          node.Status,
		LastHeartbeat:   node.LastHeartbeat,
	}

	return stats, nil
}

// ExportRegistry exports the entire registry as JSON (for backup/debugging)
func (sr *ServiceRegistry) ExportRegistry() ([]byte, error) {
	var locations []database.DeploymentLocation
	if err := database.DB.Find(&locations).Error; err != nil {
		return nil, err
	}

	return json.MarshalIndent(locations, "", "  ")
}

// updateCache updates the cache for a specific deployment
func (sr *ServiceRegistry) updateCache(deploymentID string) {
	locations, err := database.GetDeploymentLocations(deploymentID)
	if err != nil {
		logger.Warn("[Registry] Failed to update cache for deployment %s: %v", deploymentID, err)
		return
	}

	sr.cache.Store(deploymentID, &DeploymentInfo{
		DeploymentID: deploymentID,
		Locations:    locations,
		UpdatedAt:    time.Now(),
	})
}

// StartPeriodicSync starts a background goroutine that periodically syncs with Docker
func (sr *ServiceRegistry) StartPeriodicSync(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := sr.SyncWithDocker(ctx); err != nil {
					logger.Warn("[Registry] Periodic sync failed: %v", err)
				}
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

// Close closes the Docker client
func (sr *ServiceRegistry) Close() error {
	return sr.dockerClient.Close()
}

// DockerClient returns the Docker API client
func (sr *ServiceRegistry) DockerClient() client.APIClient {
	return sr.dockerClient
}

// NodeID returns the current node ID
func (sr *ServiceRegistry) NodeID() string {
	return sr.nodeID
}

// ClusterStats holds cluster-wide statistics
type ClusterStats struct {
	TotalDeployments   int            `json:"total_deployments"`
	ActiveNodes        int            `json:"active_nodes"`
	DeploymentsPerNode map[string]int `json:"deployments_per_node"`
	TotalCPU           int            `json:"total_cpu"`
	TotalMemory        int64          `json:"total_memory"`
	UsedCPU            float64        `json:"used_cpu"`
	UsedMemory         int64          `json:"used_memory"`
	Timestamp          time.Time      `json:"timestamp"`
}

// NodeStats holds statistics for a single node
type NodeStats struct {
	NodeID          string    `json:"node_id"`
	Hostname        string    `json:"hostname"`
	DeploymentCount int       `json:"deployment_count"`
	TotalCPU        int       `json:"total_cpu"`
	TotalMemory     int64     `json:"total_memory"`
	UsedCPU         float64   `json:"used_cpu"`
	UsedMemory      int64     `json:"used_memory"`
	Availability    string    `json:"availability"`
	Status          string    `json:"status"`
	LastHeartbeat   time.Time `json:"last_heartbeat"`
}
