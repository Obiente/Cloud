package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/utils"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// NodeSelector selects the best node for a deployment
type NodeSelector struct {
	strategy              string
	maxDeploymentsPerNode int
	dockerClient          client.APIClient
}

// NewNodeSelector creates a new node selector
func NewNodeSelector(strategy string, maxDeploymentsPerNode int) (*NodeSelector, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &NodeSelector{
		strategy:              strategy,
		maxDeploymentsPerNode: maxDeploymentsPerNode,
		dockerClient:          cli,
	}, nil
}

// SelectNode selects the best node for deployment based on the configured strategy
func (ns *NodeSelector) SelectNode(ctx context.Context) (*database.NodeMetadata, error) {
	// Sync node information with Docker Swarm first (this populates the database)
	if err := ns.syncNodeMetadata(ctx); err != nil {
		log.Printf("[NodeSelector] ERROR: Failed to sync node metadata: %v", err)
		// Don't continue - sync is critical, especially in development mode
		return nil, fmt.Errorf("failed to sync node metadata: %w", err)
	}

	// Get available nodes from database
	nodes, err := database.GetAvailableNodes()
	if err != nil {
		log.Printf("[NodeSelector] ERROR: Failed to query available nodes: %v", err)
		return nil, fmt.Errorf("failed to get available nodes: %w", err)
	}

	if len(nodes) == 0 {
		log.Printf("[NodeSelector] ERROR: No available nodes found in database. Node sync may have failed or no nodes meet criteria.")
		// Try to debug - check if any nodes exist at all
		var allNodes []database.NodeMetadata
		if err := database.DB.Find(&allNodes).Error; err == nil {
			log.Printf("[NodeSelector] DEBUG: Found %d total nodes in database", len(allNodes))
			for _, node := range allNodes {
				log.Printf("[NodeSelector] DEBUG: Node %s - availability=%s, status=%s, deployment_count=%d, max=%d",
					node.ID, node.Availability, node.Status, node.DeploymentCount, node.MaxDeployments)
			}
		}
		return nil, fmt.Errorf("no available nodes found (need nodes with availability='active' AND status='ready')")
	}

	log.Printf("[NodeSelector] Found %d available node(s)", len(nodes))

	// Select node based on strategy
	switch ns.strategy {
	case "least-loaded":
		return ns.selectLeastLoaded(nodes), nil
	case "round-robin":
		return ns.selectRoundRobin(nodes), nil
	case "resource-based":
		return ns.selectByResources(nodes), nil
	default:
		return ns.selectLeastLoaded(nodes), nil
	}
}

// selectLeastLoaded selects the node with the lowest deployment count
func (ns *NodeSelector) selectLeastLoaded(nodes []database.NodeMetadata) *database.NodeMetadata {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].DeploymentCount != nodes[j].DeploymentCount {
			return nodes[i].DeploymentCount < nodes[j].DeploymentCount
		}
		// If deployment counts are equal, prefer node with lower CPU usage
		return nodes[i].UsedCPU < nodes[j].UsedCPU
	})

	return &nodes[0]
}

// selectRoundRobin selects nodes in round-robin fashion
func (ns *NodeSelector) selectRoundRobin(nodes []database.NodeMetadata) *database.NodeMetadata {
	// Sort by deployment count to balance load
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].DeploymentCount < nodes[j].DeploymentCount
	})

	return &nodes[0]
}

// selectByResources selects the node with the most available resources
func (ns *NodeSelector) selectByResources(nodes []database.NodeMetadata) *database.NodeMetadata {
	type nodeScore struct {
		node  *database.NodeMetadata
		score float64
	}

	scores := make([]nodeScore, len(nodes))
	for i := range nodes {
		// Calculate available resources
		availableCPU := float64(nodes[i].TotalCPU) - nodes[i].UsedCPU
		availableMemory := float64(nodes[i].TotalMemory - nodes[i].UsedMemory)

		// Normalize scores (0-1 range)
		cpuScore := availableCPU / float64(nodes[i].TotalCPU)
		memoryScore := availableMemory / float64(nodes[i].TotalMemory)

		// Weighted score (CPU: 40%, Memory: 40%, Deployment count: 20%)
		deploymentScore := 1.0 - (float64(nodes[i].DeploymentCount) / float64(ns.maxDeploymentsPerNode))
		totalScore := (cpuScore * 0.4) + (memoryScore * 0.4) + (deploymentScore * 0.2)

		scores[i] = nodeScore{node: &nodes[i], score: totalScore}
	}

	// Sort by score (highest first)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	return scores[0].node
}

// syncNodeMetadata synchronizes node metadata from Docker Swarm to the database
// Falls back to registering local Docker daemon if not in Swarm mode (development)
func (ns *NodeSelector) syncNodeMetadata(ctx context.Context) error {
	// Check if swarm is explicitly disabled via environment variable
	// This allows non-swarm compose files to disable swarm features even if node is in swarm
	if !utils.IsSwarmModeEnabled() {
		// Swarm explicitly disabled - treat as non-swarm mode
		log.Printf("[NodeSelector] Swarm features disabled via ENABLE_SWARM=false, registering local Docker daemon as node")
		infoResult, err := ns.dockerClient.Info(ctx, client.InfoOptions{})
		if err != nil {
			return fmt.Errorf("failed to get Docker info: %w", err)
		}
		info := infoResult.Info
		return ns.registerLocalNode(ctx, info)
	}

	// Get Docker info to check if Swarm is enabled
	infoResult, err := ns.dockerClient.Info(ctx, client.InfoOptions{})
	if err != nil {
		return fmt.Errorf("failed to get Docker info: %w", err)
	}
	info := infoResult.Info

	// If Swarm is enabled, sync from Swarm nodes (but only if we're a manager)
	if info.Swarm.NodeID != "" {
		// Check if this node is a manager (can list nodes)
		// If not, fall back to registering local node
		if !info.Swarm.ControlAvailable {
			log.Printf("[NodeSelector] Docker Swarm enabled but node is a worker (not manager), registering local node only")
			// Fall through to register local node
		} else {
			// We're a manager - can list all nodes
			nodesResult, err := ns.dockerClient.NodeList(ctx, client.NodeListOptions{})
			if err != nil {
				// If NodeList fails (e.g., permission denied), fall back to local node
				if strings.Contains(err.Error(), "swarm manager") || strings.Contains(err.Error(), "not a swarm manager") {
					log.Printf("[NodeSelector] Cannot list Docker Swarm nodes (worker node), registering local node only: %v", err)
					// Fall through to register local node
				} else {
					return fmt.Errorf("failed to list Docker nodes: %w", err)
				}
			} else {
				// Successfully got nodes - sync them
				// Track all Swarm node IDs to identify nodes that should be removed
				swarmNodeIDs := make(map[string]bool)
				
				for _, node := range nodesResult.Items {
					// Get node info
					nodeInfoResult, err := ns.dockerClient.NodeInspect(ctx, node.ID, client.NodeInspectOptions{})
					if err != nil {
						continue
					}
					nodeInfo := nodeInfoResult.Node

					hostname := nodeInfo.Description.Hostname
					nodeID := node.ID
					
					// Track this node ID as existing in Swarm
					swarmNodeIDs[nodeID] = true

					// Check if a node with this hostname already exists (might have different ID after Swarm reset)
					var existingNode database.NodeMetadata
					err = database.DB.Where("hostname = ?", hostname).First(&existingNode).Error
					hostnameExists := err == nil

					// Calculate resources
					totalCPU := int(nodeInfo.Description.Resources.NanoCPUs / 1e9)
					totalMemory := nodeInfo.Description.Resources.MemoryBytes

					// Get current resource usage by aggregating container stats
					deploymentCount := ns.getNodeDeploymentCount(ctx, nodeID)
					usedCPU, usedMemory := ns.calculateNodeResourceUsage(ctx, nodeID, totalCPU)

					// Update or create node metadata
					labelsJSON := "{}"
					if len(node.Spec.Annotations.Labels) > 0 {
						labelsBytes, err := json.Marshal(node.Spec.Annotations.Labels)
						if err == nil {
							labelsJSON = string(labelsBytes)
						}
					}

					var metadata *database.NodeMetadata
					if hostnameExists && existingNode.ID != nodeID {
						// Node with same hostname but different ID exists - update it to use new ID
						// This handles Swarm resets where nodes get new IDs but same hostnames
						log.Printf("[NodeSelector] Node with hostname %s exists with ID %s, updating to use ID %s (Swarm reset detected)", hostname, existingNode.ID, nodeID)
						oldID := existingNode.ID

						// Update the existing node's ID and other fields
						existingNode.ID = nodeID
						existingNode.Hostname = hostname
						existingNode.Role = string(node.Spec.Role)
						existingNode.Availability = string(node.Spec.Availability)
						existingNode.Status = string(node.Status.State)
						existingNode.TotalCPU = totalCPU
						existingNode.TotalMemory = totalMemory
						existingNode.UsedCPU = usedCPU
						existingNode.UsedMemory = usedMemory
						existingNode.DeploymentCount = deploymentCount
						existingNode.MaxDeployments = ns.maxDeploymentsPerNode
						existingNode.Labels = labelsJSON
						existingNode.UpdatedAt = time.Now()

						// Delete old record and create new one with correct ID
						// We need to delete first to avoid hostname constraint violation
						if err := database.DB.Delete(&database.NodeMetadata{}, "id = ?", oldID).Error; err != nil {
							log.Printf("[NodeSelector] WARNING: Failed to delete old node %s: %v", oldID, err)
						}
						metadata = &existingNode
					} else {
						// Create new metadata (either new node or existing node with same ID)
						metadata = &database.NodeMetadata{
							ID:              nodeID,
							Hostname:        hostname,
							Role:            string(node.Spec.Role),
							Availability:    string(node.Spec.Availability),
							Status:          string(node.Status.State),
							TotalCPU:        totalCPU,
							TotalMemory:     totalMemory,
							UsedCPU:         usedCPU,
							UsedMemory:      usedMemory,
							DeploymentCount: deploymentCount,
							MaxDeployments:  ns.maxDeploymentsPerNode,
							Labels:          labelsJSON,
						}
					}

					// Save to database
					database.DB.Save(metadata)

					// Also update metrics separately to ensure last_heartbeat is updated
					database.UpdateNodeMetrics(nodeID, usedCPU, usedMemory)
				}
				
				// Clean up nodes that are no longer in the Swarm
				// Only remove Swarm nodes (those that don't start with "local-")
				var allDBNodes []database.NodeMetadata
				if err := database.DB.Find(&allDBNodes).Error; err == nil {
					for _, dbNode := range allDBNodes {
						// Only remove Swarm nodes (not local compose nodes)
						if !strings.HasPrefix(dbNode.ID, "local-") {
							if !swarmNodeIDs[dbNode.ID] {
								log.Printf("[NodeSelector] Removing node %s (%s) from database - no longer in Swarm", dbNode.ID, dbNode.Hostname)
								if err := database.DB.Delete(&database.NodeMetadata{}, "id = ?", dbNode.ID).Error; err != nil {
									log.Printf("[NodeSelector] WARNING: Failed to remove node %s from database: %v", dbNode.ID, err)
								}
							}
						}
					}
				}
				
				// Successfully synced all nodes, return
				return nil
			}
		}
	}

	// Not in Swarm mode OR we're a worker node - register local Docker daemon as a node
	return ns.registerLocalNode(ctx, info)
}

// registerLocalNode registers the local Docker daemon as a node in the database
func (ns *NodeSelector) registerLocalNode(ctx context.Context, info interface{}) error {
	// Use reflection to access fields since we don't know the exact type name
	v := reflect.ValueOf(info)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Get Swarm field
	swarmField := v.FieldByName("Swarm")
	var swarmNodeID string
	var controlAvailable bool
	if swarmField.IsValid() {
		if nodeIDField := swarmField.FieldByName("NodeID"); nodeIDField.IsValid() {
			swarmNodeID = nodeIDField.String()
		}
		if controlField := swarmField.FieldByName("ControlAvailable"); controlField.IsValid() {
			controlAvailable = controlField.Bool()
		}
	}

	// Get Name field
	nameField := v.FieldByName("Name")
	name := ""
	if nameField.IsValid() {
		name = nameField.String()
	}

	// Get NCPU field
	ncpuField := v.FieldByName("NCPU")
	ncpu := 0
	if ncpuField.IsValid() {
		ncpu = int(ncpuField.Int())
	}

	// Get MemTotal field
	memTotalField := v.FieldByName("MemTotal")
	memTotal := int64(0)
	if memTotalField.IsValid() {
		memTotal = memTotalField.Int()
	}

	// Determine role and availability from Swarm info if available
	// For worker nodes registering themselves, try to get info from Swarm
	var role string = "worker"         // Default
	var availability string = "active" // Default
	var status string = "ready"        // Default

	if swarmNodeID != "" && utils.IsSwarmModeEnabled() {
		// Try to get node info from Swarm to get actual role/availability
		// This works even on worker nodes if they can query their own info
		nodeInfoResult, err := ns.dockerClient.NodeInspect(ctx, swarmNodeID, client.NodeInspectOptions{})
		if err == nil {
			// Successfully got node info - use actual values
			nodeInfo := nodeInfoResult.Node
			role = string(nodeInfo.Spec.Role)
			availability = string(nodeInfo.Spec.Availability)
			status = string(nodeInfo.Status.State)
			log.Printf("[NodeSelector] Got role=%s, availability=%s, status=%s from Swarm for node %s", role, availability, status, swarmNodeID)
		} else {
			// Can't get node info (might be worker node) - infer from ControlAvailable
			if controlAvailable {
				role = "manager"
			} else {
				role = "worker"
			}
			log.Printf("[NodeSelector] Could not get node info from Swarm, using inferred role=%s (ControlAvailable=%v)", role, controlAvailable)
		}
	}

	if swarmNodeID == "" {
		log.Printf("[NodeSelector] Docker Swarm not enabled, registering local Docker daemon as node")
	} else {
		log.Printf("[NodeSelector] Docker Swarm enabled but node is a worker (or swarm disabled), registering local Docker daemon as node")
	}

	// Use system info to get resources
	totalCPU := ncpu
	totalMemory := memTotal

	// Determine node ID - respect ENABLE_SWARM environment variable
	// If ENABLE_SWARM=false, always use local- prefix even if Swarm is enabled in Docker
	var nodeID string
	if utils.IsSwarmModeEnabled() {
		// Swarm mode enabled - use Swarm node ID if available
		nodeID = swarmNodeID
		if nodeID == "" {
			// Swarm enabled but not in Swarm - use synthetic ID
			nodeID = "local-" + name
		}
	} else {
		// Swarm mode disabled - always use local- prefix
		nodeID = "local-" + name
	}
	deploymentCount := ns.getNodeDeploymentCount(ctx, nodeID)

	// Check if a node with this hostname already exists (might have different ID)
	var existingNode database.NodeMetadata
	err := database.DB.Where("hostname = ?", name).First(&existingNode).Error
	hostnameExists := err == nil

	var metadata *database.NodeMetadata
	if hostnameExists && existingNode.ID != nodeID {
		// Node with same hostname but different ID exists - update it to use our ID
		log.Printf("[NodeSelector] Node with hostname %s exists with ID %s, updating to use ID %s", name, existingNode.ID, nodeID)
		// Store old ID for deletion
		oldID := existingNode.ID
		// Calculate resource usage for local node
		usedCPU, usedMemory := ns.calculateNodeResourceUsage(ctx, nodeID, totalCPU)

		// Update the existing node's ID and other fields
		existingNode.ID = nodeID
		existingNode.Role = role
		existingNode.Availability = availability
		existingNode.Status = status
		existingNode.TotalCPU = totalCPU
		existingNode.TotalMemory = totalMemory
		existingNode.UsedCPU = usedCPU
		existingNode.UsedMemory = usedMemory
		existingNode.DeploymentCount = deploymentCount
		existingNode.MaxDeployments = ns.maxDeploymentsPerNode
		existingNode.Labels = "{}"
		existingNode.UpdatedAt = time.Now()

		// Delete old record and create new one with correct ID
		// We need to delete first to avoid hostname constraint violation
		if err := database.DB.Delete(&database.NodeMetadata{}, "id = ?", oldID).Error; err != nil {
			log.Printf("[NodeSelector] WARNING: Failed to delete old node %s: %v", oldID, err)
		}
		// Now create the new one
		metadata = &existingNode
	} else if !hostnameExists {
		// Calculate resource usage for local node
		usedCPU, usedMemory := ns.calculateNodeResourceUsage(ctx, nodeID, totalCPU)

		// No existing node with this hostname, create new one
		metadata = &database.NodeMetadata{
			ID:              nodeID,
			Hostname:        name,
			Role:            role,
			Availability:    availability,
			Status:          status,
			TotalCPU:        totalCPU,
			TotalMemory:     totalMemory,
			UsedCPU:         usedCPU,
			UsedMemory:      usedMemory,
			DeploymentCount: deploymentCount,
			MaxDeployments:  ns.maxDeploymentsPerNode,
			Labels:          "{}", // Empty JSON object for jsonb field
		}

		// Also update metrics separately to ensure last_heartbeat is updated
		database.UpdateNodeMetrics(nodeID, usedCPU, usedMemory)
	} else {
		// Node exists with same hostname and ID - just update it
		metadata = &existingNode
		metadata.Role = role
		metadata.Availability = availability
		metadata.Status = status
		metadata.TotalCPU = totalCPU
		metadata.TotalMemory = totalMemory
		metadata.DeploymentCount = deploymentCount
		metadata.MaxDeployments = ns.maxDeploymentsPerNode
		// Calculate resource usage for local node
		usedCPU, usedMemory := ns.calculateNodeResourceUsage(ctx, nodeID, totalCPU)
		metadata.UsedCPU = usedCPU
		metadata.UsedMemory = usedMemory
		metadata.Labels = "{}"
		metadata.UpdatedAt = time.Now()
	}

	// Use Save which will create or update based on primary key (ID)
	result := database.DB.Save(metadata)

	// Also update metrics separately to ensure last_heartbeat is updated
	database.UpdateNodeMetrics(nodeID, metadata.UsedCPU, metadata.UsedMemory)
	if result.Error != nil {
		log.Printf("[NodeSelector] ERROR: Failed to save local node: %v", result.Error)
		return fmt.Errorf("failed to save local node: %w", result.Error)
	}

	log.Printf("[NodeSelector] Registered/Updated local node: %s (hostname: %s, CPU: %d, Memory: %d bytes, availability=%s, status=%s, rows=%d)",
		nodeID, name, totalCPU, totalMemory, metadata.Availability, metadata.Status, result.RowsAffected)

	// Verify the node was saved correctly
	var verifyNode database.NodeMetadata
	if err := database.DB.First(&verifyNode, "id = ?", nodeID).Error; err != nil {
		log.Printf("[NodeSelector] ERROR: Failed to verify saved node: %v", err)
		return fmt.Errorf("failed to verify saved node: %w", err)
	}
	log.Printf("[NodeSelector] Verified node in DB: %s - availability=%s, status=%s",
		verifyNode.ID, verifyNode.Availability, verifyNode.Status)

	return nil
}

// calculateNodeResourceUsage calculates CPU and memory usage for a node by aggregating container stats
func (ns *NodeSelector) calculateNodeResourceUsage(ctx context.Context, nodeID string, totalCPU int) (float64, int64) {
	// Get all containers on this node
	var containers []container.Summary
	// err will be used below

	// For Swarm nodes, try filtering by node ID label first
	if !strings.HasPrefix(nodeID, "local-") {
		filterArgs := make(client.Filters)
		filterArgs.Add("label", fmt.Sprintf("com.docker.swarm.node.id=%s", nodeID))
		filterArgs.Add("status", "running")

		containersResult, err := ns.dockerClient.ContainerList(ctx, client.ContainerListOptions{
			Filters: filterArgs,
		})
		if err != nil {
			log.Printf("[NodeSelector] Failed to list containers for node %s: %v", nodeID, err)
			return 0.0, 0
		}
		containers = containersResult.Items

		// If no containers found with Swarm label, fall back to all running containers
		// This handles cases where containers are from compose deployments on a Swarm node
		if len(containers) == 0 {
			log.Printf("[NodeSelector] No containers found with Swarm node ID label for %s, trying all running containers", nodeID)
			containersResult, err = ns.dockerClient.ContainerList(ctx, client.ContainerListOptions{
				All:     false,
				Filters: func() client.Filters { f := make(client.Filters); f.Add("status", "running"); return f }(),
			})
			if err != nil {
				log.Printf("[NodeSelector] Failed to list all running containers: %v", err)
				return 0.0, 0
			}
			containers = containersResult.Items
		}
	} else {
		// For local nodes (explicitly local- prefix), get all running containers (we're on the local node)
		containersResult, err := ns.dockerClient.ContainerList(ctx, client.ContainerListOptions{
			All:     false,
			Filters: func() client.Filters { f := make(client.Filters); f.Add("status", "running"); return f }(),
		})
		if err != nil {
			log.Printf("[NodeSelector] Failed to list containers for local node: %v", err)
			return 0.0, 0
		}
		containers = containersResult.Items
	}

	if len(containers) == 0 {
		log.Printf("[NodeSelector] No running containers found on node %s", nodeID)
		return 0.0, 0
	}

	log.Printf("[NodeSelector] Calculating resource usage for node %s from %d containers", nodeID, len(containers))

	// Aggregate CPU and memory usage from all containers
	var totalCPUUsage float64
	var totalMemoryUsage int64

	for _, container := range containers {
		// Get container stats with a timeout to prevent hanging
		// Use a shorter timeout for stats (they should be quick)
		statsCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		stats, err := ns.dockerClient.ContainerStats(statsCtx, container.ID, client.ContainerStatsOptions{Stream: false})
		if err != nil {
			cancel()
			// Ignore context canceled errors - they're not critical for resource calculation
			if err == context.Canceled || strings.Contains(err.Error(), "context canceled") {
				log.Printf("[NodeSelector] Stats request canceled for container %s (skipping)", container.ID[:12])
			} else {
				log.Printf("[NodeSelector] Failed to get stats for container %s: %v", container.ID[:12], err)
			}
			continue
		}

		// Decode stats JSON
		var statsJSON struct {
			CPUStats struct {
				CPUUsage struct {
					TotalUsage  uint64   `json:"total_usage"`
					PercpuUsage []uint64 `json:"percpu_usage"`
				} `json:"cpu_usage"`
				SystemUsage uint64 `json:"system_cpu_usage"`
				OnlineCPUs  uint   `json:"online_cpus"`
			} `json:"cpu_stats"`
			PreCPUStats struct {
				CPUUsage struct {
					TotalUsage uint64 `json:"total_usage"`
				} `json:"cpu_usage"`
				SystemUsage uint64 `json:"system_cpu_usage"`
			} `json:"precpu_stats"`
			MemoryStats struct {
				Usage uint64 `json:"usage"`
				Limit uint64 `json:"limit"`
			} `json:"memory_stats"`
		}

		if err := json.NewDecoder(stats.Body).Decode(&statsJSON); err != nil {
			stats.Body.Close()
			cancel()
			log.Printf("[NodeSelector] Failed to decode stats for container %s: %v", container.ID[:12], err)
			continue
		}
		stats.Body.Close()
		cancel() // Cancel after we're done with the stats

		// Calculate CPU usage percentage
		// Docker CPU stats are in nanoseconds. We need to validate the deltas to prevent division by tiny numbers
		cpuUsage := 0.0
		if statsJSON.PreCPUStats.SystemUsage > 0 && statsJSON.CPUStats.SystemUsage > statsJSON.PreCPUStats.SystemUsage {
			cpuDelta := int64(statsJSON.CPUStats.CPUUsage.TotalUsage - statsJSON.PreCPUStats.CPUUsage.TotalUsage)
			systemDelta := int64(statsJSON.CPUStats.SystemUsage - statsJSON.PreCPUStats.SystemUsage)
			
			// Validate deltas to prevent invalid calculations
			// Minimum systemDelta: 1 millisecond (1,000,000 nanoseconds) to prevent division by tiny numbers
			const minSystemDelta = 1_000_000 // 1ms in nanoseconds
			
			if systemDelta >= minSystemDelta && statsJSON.CPUStats.OnlineCPUs > 0 {
				// Handle counter wraparound (uint64 overflow)
				if cpuDelta < 0 {
					// Counter wraparound detected - skip this calculation
					log.Printf("[NodeSelector] CPU counter wraparound detected for container %s, skipping", container.ID[:12])
					cpuUsage = 0.0
				} else {
					cpuUsage = (float64(cpuDelta) / float64(systemDelta)) * float64(statsJSON.CPUStats.OnlineCPUs) * 100.0
					
					// Validate the result is physically reasonable
					maxReasonableCPU := float64(statsJSON.CPUStats.OnlineCPUs) * 100.0
					if cpuUsage < 0 || cpuUsage > maxReasonableCPU {
						cpuUsage = 0.0 // Skip invalid measurements
					}
				}
			} else if systemDelta > 0 && systemDelta < minSystemDelta {
				// systemDelta too small - skip calculation
				cpuUsage = 0.0
			}
		}

		// Get memory usage
		memoryUsage := int64(statsJSON.MemoryStats.Usage)

		totalCPUUsage += cpuUsage
		totalMemoryUsage += memoryUsage
	}

	// CPU usage is already a percentage, return as-is
	// Memory usage is in bytes
	log.Printf("[NodeSelector] Node %s resource usage: CPU=%.2f%%, Memory=%d bytes", nodeID, totalCPUUsage, totalMemoryUsage)
	return totalCPUUsage, totalMemoryUsage
}

// getNodeDeploymentCount counts deployments on a specific node
// Also checks by hostname as fallback for cases where node ID changed
// This handles the case where a Swarm node has compose deployments with old node IDs
func (ns *NodeSelector) getNodeDeploymentCount(ctx context.Context, nodeID string) int {
	// Get node info to access hostname
	var node database.NodeMetadata
	if err := database.DB.Where("id = ?", nodeID).First(&node).Error; err != nil {
		log.Printf("[NodeSelector] Failed to find node %s: %v", nodeID, err)
		return 0
	}

	count := int64(0)

	// First try by node ID
	database.DB.Model(&database.DeploymentLocation{}).
		Where("node_id = ? AND status = ?", nodeID, "running").
		Count(&count)

	// Always check by hostname and update if needed (handles node ID changes)
	// This is important for Swarm nodes that have compose deployments with old node IDs
	if node.Hostname != "" {
		var hostnameCount int64
		database.DB.Model(&database.DeploymentLocation{}).
			Where("node_hostname = ? AND status = ?", node.Hostname, "running").
			Count(&hostnameCount)

		// If we found deployments by hostname that don't match current node ID, update them
		if hostnameCount > 0 {
			updateResult := database.DB.Model(&database.DeploymentLocation{}).
				Where("node_hostname = ? AND node_id != ?", node.Hostname, nodeID).
				Update("node_id", nodeID)

			if updateResult.Error == nil && updateResult.RowsAffected > 0 {
				log.Printf("[NodeSelector] Updated %d deployment locations from old node ID to %s (hostname: %s)",
					updateResult.RowsAffected, nodeID, node.Hostname)
			}

			// Use the hostname count if it's higher (includes both old and new node IDs)
			if hostnameCount > count {
				count = hostnameCount
			}
		}
	}

	return int(count)
}

// Close closes the Docker client
func (ns *NodeSelector) Close() error {
	return ns.dockerClient.Close()
}
