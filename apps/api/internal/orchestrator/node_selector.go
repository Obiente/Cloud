package orchestrator

import (
	"context"
	"fmt"
	"sort"

	"api/internal/database"

	"github.com/moby/moby/api/types"
	"github.com/moby/moby/client"
)

// NodeSelector selects the best node for a deployment
type NodeSelector struct {
	strategy              string
	maxDeploymentsPerNode int
	dockerClient          *client.Client
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
	// Get available nodes from database
	nodes, err := database.GetAvailableNodes()
	if err != nil {
		return nil, fmt.Errorf("failed to get available nodes: %w", err)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no available nodes found")
	}

	// Sync node information with Docker Swarm
	if err := ns.syncNodeMetadata(ctx); err != nil {
		return nil, fmt.Errorf("failed to sync node metadata: %w", err)
	}

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
func (ns *NodeSelector) syncNodeMetadata(ctx context.Context) error {
	// Get nodes from Docker Swarm
	nodes, err := ns.dockerClient.NodeList(ctx, types.NodeListOptions{})
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
		}

		// Save to database
		database.DB.Save(metadata)
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
