package orchestrator

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/registry"

	"github.com/moby/moby/client"
)

// Core deployment manager types and initialization

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

type dockerHelper interface {
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string, timeout time.Duration) error
	RemoveContainer(ctx context.Context, containerID string, force bool) error
	RestartContainer(ctx context.Context, containerID string, timeout time.Duration) error
	ContainerLogs(ctx context.Context, containerID string, tail string, follow bool, since *time.Time, until *time.Time) (io.ReadCloser, error)
	ContainerExecRun(ctx context.Context, containerID string, cmd []string) (string, error)
}

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

func (dm *DeploymentManager) GetNodeID() string {
	return dm.nodeID
}

// GetDockerClient returns the Docker client (for internal use by orchestrator service)
func (dm *DeploymentManager) GetDockerClient() interface{} {
	return dm.dockerClient
}

// Close closes the deployment manager and cleans up resources
func (dm *DeploymentManager) Close() error {
	if dm.dockerClient != nil {
		return dm.dockerClient.Close()
	}
	return nil
}

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

