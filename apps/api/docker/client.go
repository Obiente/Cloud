package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/moby/moby/client"
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

// (List/Inspect helpers removed; not needed by orchestrator)

// StartContainer starts the specified container if it is not already running.
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	if c == nil || c.api == nil {
		return ErrUninitialized
	}

	if err := c.api.ContainerStart(ctx, containerID, client.ContainerStartOptions{}); err != nil {
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

	if err := c.api.ContainerStop(ctx, containerID, client.ContainerStopOptions{Timeout: timeoutSeconds}); err != nil {
		return fmt.Errorf("docker: stop container %s: %w", containerID, err)
	}

	return nil
}

// IsNotFound reports whether the provided error indicates a missing resource.
// func IsNotFound(err error) bool { return client.IsErrNotFound(err) }

// RemoveContainer removes the specified container, optionally forcing removal.
func (c *Client) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	if c == nil || c.api == nil {
		return ErrUninitialized
	}
	if err := c.api.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{Force: force}); err != nil {
		return fmt.Errorf("docker: remove container %s: %w", containerID, err)
	}
	return nil
}

// RestartContainer restarts the container with an optional timeout.
func (c *Client) RestartContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	if c == nil || c.api == nil {
		return ErrUninitialized
	}
	var timeoutSeconds *int
	if timeout > 0 {
		secs := int(timeout.Round(time.Second) / time.Second)
		timeoutSeconds = &secs
	}
	if err := c.api.ContainerRestart(ctx, containerID, client.ContainerStopOptions{Timeout: timeoutSeconds}); err != nil {
		return fmt.Errorf("docker: restart container %s: %w", containerID, err)
	}
	return nil
}

// ContainerLogs fetches the container logs as an io.ReadCloser.
func (c *Client) ContainerLogs(ctx context.Context, containerID string, tail string) (io.ReadCloser, error) {
	if c == nil || c.api == nil {
		return nil, ErrUninitialized
	}
	logs, err := c.api.ContainerLogs(ctx, containerID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
	})
	if err != nil {
		return nil, fmt.Errorf("docker: logs for %s: %w", containerID, err)
	}
	return logs, nil
}
