package docker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/moby/moby/api/types"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/filters"
	"github.com/moby/moby/client"
	"github.com/moby/moby/errdefs"
)

// ErrUninitialized is returned when a client method is invoked before the
// Docker API client has been constructed. This makes failure modes explicit.
var ErrUninitialized = errors.New("docker: client not initialized")

// Client wraps the Docker API client to provide a constrained set of helper
// methods used by the dashboard API.
type Client struct {
	api client.APIClient
}

// New constructs a Docker client using environment variables and API version
// negotiation so it works across Docker Desktop and remote engines.
func New() (*Client, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("docker: create client: %w", err)
	}

	return &Client{api: cli}, nil
}

// Close releases any underlying resources held by the Docker client.
func (c *Client) Close() error {
	if c == nil || c.api == nil {
		return nil
	}
	return c.api.Close()
}

// ListContainersOptions controls the behaviour of the ListContainers helper.
type ListContainersOptions struct {
	All     bool
	Filters filters.Args
}

// ListContainers fetches container metadata using the provided filters.
func (c *Client) ListContainers(ctx context.Context, opts ListContainersOptions) ([]types.Container, error) {
	if c == nil || c.api == nil {
		return nil, ErrUninitialized
	}

	listOptions := types.ContainerListOptions{
		All:     opts.All,
		Filters: opts.Filters,
	}

	containers, err := c.api.ContainerList(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("docker: list containers: %w", err)
	}

	return containers, nil
}

// InspectContainer returns detailed information for a single container.
func (c *Client) InspectContainer(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	if c == nil || c.api == nil {
		return types.ContainerJSON{}, ErrUninitialized
	}

	info, err := c.api.ContainerInspect(ctx, containerID)
	if err != nil {
		return types.ContainerJSON{}, fmt.Errorf("docker: inspect container %s: %w", containerID, err)
	}

	return info, nil
}

// StartContainer starts the specified container if it is not already running.
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	if c == nil || c.api == nil {
		return ErrUninitialized
	}

	if err := c.api.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("docker: start container %s: %w", containerID, err)
	}

	return nil
}

// StopContainer attempts to gracefully stop the container within an optional
// timeout window.
func (c *Client) StopContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	if c == nil || c.api == nil {
		return ErrUninitialized
	}

	var timeoutSeconds *int
	if timeout > 0 {
		secs := int(timeout.Round(time.Second) / time.Second)
		timeoutSeconds = &secs
	}

	if err := c.api.ContainerStop(ctx, containerID, container.StopOptions{Timeout: timeoutSeconds}); err != nil {
		return fmt.Errorf("docker: stop container %s: %w", containerID, err)
	}

	return nil
}

// IsNotFound reports whether the provided error indicates a missing resource.
func IsNotFound(err error) bool {
	return errdefs.IsNotFound(err)
}
