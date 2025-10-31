package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"

	"api/internal/database"

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
	// Get Docker info to check if Swarm is enabled
	info, err := ns.dockerClient.Info(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Docker info: %w", err)
	}

	// If Swarm is enabled, sync from Swarm nodes
	if info.Swarm.NodeID != "" {
		// Get nodes from Docker Swarm
		nodes, err := ns.dockerClient.NodeList(ctx, client.NodeListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list Docker nodes: %w", err)
		}

		for _, node := range nodes {
			// Get node info
			nodeInfo, _, err := ns.dockerClient.NodeInspectWithRaw(ctx, node.ID)
			if err != nil {
				continue
			}

			// Calculate resources
			totalCPU := int(nodeInfo.Description.Resources.NanoCPUs / 1e9)
			totalMemory := nodeInfo.Description.Resources.MemoryBytes

			// Get current resource usage (would require additional metrics)
			// For now, we'll estimate based on deployment count
			deploymentCount := ns.getNodeDeploymentCount(ctx, node.ID)

			// Update or create node metadata
			labelsJSON := "{}"
			if len(node.Spec.Annotations.Labels) > 0 {
				labelsBytes, err := json.Marshal(node.Spec.Annotations.Labels)
				if err == nil {
					labelsJSON = string(labelsBytes)
				}
			}
			
			metadata := &database.NodeMetadata{
				ID:              node.ID,
				Hostname:        nodeInfo.Description.Hostname,
				Role:            string(node.Spec.Role),
				Availability:    string(node.Spec.Availability),
				Status:          string(node.Status.State),
				TotalCPU:        totalCPU,
				TotalMemory:     totalMemory,
				DeploymentCount: deploymentCount,
				MaxDeployments:  ns.maxDeploymentsPerNode,
				Labels:          labelsJSON,
			}

			// Save to database
			database.DB.Save(metadata)
		}
	} else {
		// Not in Swarm mode - register local Docker daemon as a node (development mode)
		log.Printf("[NodeSelector] Docker Swarm not enabled, registering local Docker daemon as node")
		
		// Use system info to get resources
		totalCPU := int(info.NCPU)
		totalMemory := info.MemTotal
		
		// Create a synthetic node ID based on hostname (since we don't have a Swarm node ID)
		nodeID := "local-" + info.Name
		deploymentCount := ns.getNodeDeploymentCount(ctx, nodeID)

		// Register local node with availability='active' and status='ready'
		metadata := &database.NodeMetadata{
			ID:              nodeID,
			Hostname:        info.Name,
			Role:            "worker",
			Availability:    "active",
			Status:          "ready",
			TotalCPU:        totalCPU,
			TotalMemory:     totalMemory,
			DeploymentCount: deploymentCount,
			MaxDeployments:  ns.maxDeploymentsPerNode,
			UsedCPU:         0.0,
			UsedMemory:      0,
			Labels:          "{}", // Empty JSON object for jsonb field
		}

		// Use Save which will create or update based on primary key (ID)
		result := database.DB.Save(metadata)
		if result.Error != nil {
			log.Printf("[NodeSelector] ERROR: Failed to save local node: %v", result.Error)
			return fmt.Errorf("failed to save local node: %w", result.Error)
		}
		
		log.Printf("[NodeSelector] Registered/Updated local node: %s (hostname: %s, CPU: %d, Memory: %d bytes, availability=%s, status=%s, rows=%d)", 
			nodeID, info.Name, totalCPU, totalMemory, metadata.Availability, metadata.Status, result.RowsAffected)
		
		// Verify the node was saved correctly
		var verifyNode database.NodeMetadata
		if err := database.DB.First(&verifyNode, "id = ?", nodeID).Error; err != nil {
			log.Printf("[NodeSelector] ERROR: Failed to verify saved node: %v", err)
			return fmt.Errorf("failed to verify saved node: %w", err)
		}
		log.Printf("[NodeSelector] Verified node in DB: %s - availability=%s, status=%s", 
			verifyNode.ID, verifyNode.Availability, verifyNode.Status)
	}

	return nil
}

// getNodeDeploymentCount counts deployments on a specific node
func (ns *NodeSelector) getNodeDeploymentCount(ctx context.Context, nodeID string) int {
	count := int64(0)
	database.DB.Model(&database.DeploymentLocation{}).
		Where("node_id = ? AND status = ?", nodeID, "running").
		Count(&count)
	return int(count)
}

// Close closes the Docker client
func (ns *NodeSelector) Close() error {
	return ns.dockerClient.Close()
}
