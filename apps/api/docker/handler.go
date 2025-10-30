package docker

import (
	"context"
	"fmt"
	"sync"
	"time"

	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"

	"github.com/moby/moby/client"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	mu     sync.RWMutex
	client *client.Client
}

var globalHandler *Handler

func InitClient() error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}

	h := &Handler{
		client: cli,
	}
	globalHandler = h
	return nil
}

func GetHandler() *Handler {
	return globalHandler
}

func (h *Handler) ListContainersAsDeployments(ctx context.Context) ([]*deploymentsv1.Deployment, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.client == nil {
		return nil, fmt.Errorf("docker client not initialized")
	}

	containers, err := h.client.ContainerList(ctx, client.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	deployments := make([]*deploymentsv1.Deployment, 0, len(containers))
	for _, ctr := range containers {
		name := "unknown"
		if len(ctr.Names) > 0 {
			name = ctr.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}

		var depStatus deploymentsv1.DeploymentStatus
		switch ctr.State {
		case "running":
			depStatus = deploymentsv1.DeploymentStatus_RUNNING
		case "exited":
			depStatus = deploymentsv1.DeploymentStatus_STOPPED
		case "paused":
			depStatus = deploymentsv1.DeploymentStatus_STOPPED
		default:
			depStatus = deploymentsv1.DeploymentStatus_DEPLOYMENT_STATUS_UNSPECIFIED
		}

		createdTime := time.Unix(ctr.Created, 0)

		deployment := &deploymentsv1.Deployment{
			Id:           ctr.ID[:12],
			Name:         name,
			Domain:       fmt.Sprintf("%s.local", name),
			Type:         deploymentsv1.DeploymentType_DOCKER,
			Status:       depStatus,
			HealthStatus: ctr.Status,
			CreatedAt:    timestamppb.New(createdTime),
		}

		deployments = append(deployments, deployment)
	}

	return deployments, nil
}

func ListContainersAsDeployments(ctx context.Context) ([]*deploymentsv1.Deployment, error) {
	if globalHandler == nil {
		return nil, fmt.Errorf("docker handler not initialized")
	}
	return globalHandler.ListContainersAsDeployments(ctx)
}
