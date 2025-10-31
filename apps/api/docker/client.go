package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
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
// If follow is true, the logs will be streamed continuously.
func (c *Client) ContainerLogs(ctx context.Context, containerID string, tail string, follow bool) (io.ReadCloser, error) {
	if c == nil || c.api == nil {
		return nil, ErrUninitialized
	}
	logs, err := c.api.ContainerLogs(ctx, containerID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Follow:     follow,
	})
	if err != nil {
		return nil, fmt.Errorf("docker: logs for %s: %w", containerID, err)
	}
	return logs, nil
}

// ContainerExec creates an interactive exec session in the container
// Uses Docker exec API to create a TTY session for interactive terminal
// Returns a ReadWriteCloser that can be used for bidirectional communication
func (c *Client) ContainerExec(ctx context.Context, containerID string, cols, rows int) (io.ReadWriteCloser, error) {
	if c == nil || c.api == nil {
		return nil, ErrUninitialized
	}

	// Create exec configuration for interactive TTY session
	execConfig := container.ExecOptions{
		Cmd:          []string{"/bin/sh"}, // Default shell
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Env:          []string{"TERM=xterm-256color"},
	}

	// Create exec instance
	execIDResp, err := c.api.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("create exec: %w", err)
	}

	// Attach to exec with TTY support
	attachResp, err := c.api.ContainerExecAttach(ctx, execIDResp.ID, container.ExecAttachOptions{
		Detach: false,
		Tty:    true,
	})
	if err != nil {
		return nil, fmt.Errorf("attach exec: %w", err)
	}

	// Start the exec instance
	if err := c.api.ContainerExecStart(ctx, execIDResp.ID, container.ExecStartOptions{
		Detach: false,
		Tty:    true,
	}); err != nil {
		attachResp.Close()
		return nil, fmt.Errorf("start exec: %w", err)
	}

	// Return the attach connection which implements ReadWriteCloser
	// Note: Docker multiplexes stdout/stderr with 8-byte headers [stream_type(1) + 3 padding + size(4)]
	// For TTY mode, output is raw without headers, but we need to handle both
	return attachResp.Conn, nil
}

// ContainerExecRun runs a command in the container and returns the output
func (c *Client) ContainerExecRun(ctx context.Context, containerID string, cmd []string) (string, error) {
	if c == nil || c.api == nil {
		return "", ErrUninitialized
	}

	// Create exec configuration
	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	// Create exec instance
	execIDResp, err := c.api.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", fmt.Errorf("create exec: %w", err)
	}

	// Attach to exec to get output
	attachResp, err := c.api.ContainerExecAttach(ctx, execIDResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", fmt.Errorf("attach exec: %w", err)
	}
	defer attachResp.Close()

	// Read output - Docker multiplexes stdout/stderr with 8-byte headers
	var stdout bytes.Buffer
	outputDone := make(chan error, 1)

	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := attachResp.Reader.Read(buf)
			if n > 0 {
				// Skip 8-byte header and extract stdout (type 1)
				if n > 8 {
					streamType := buf[0]
					if streamType == 1 { // stdout
						stdout.Write(buf[8:n])
					}
					// Ignore stderr (type 2) for now
				}
			}
			if err == io.EOF {
				outputDone <- nil
				return
			}
			if err != nil {
				outputDone <- err
				return
			}
		}
	}()

	// Start the exec
	err = c.api.ContainerExecStart(ctx, execIDResp.ID, container.ExecStartOptions{})
	if err != nil {
		attachResp.Close()
		return "", fmt.Errorf("start exec: %w", err)
	}

	// Wait for output to complete
	if err := <-outputDone; err != nil && err != io.EOF {
		return "", fmt.Errorf("read output: %w", err)
	}

	return stdout.String(), nil
}

// ContainerListFiles lists files in a directory using ls command
// If container is stopped, it temporarily starts it, performs the operation, then stops it again
func (c *Client) ContainerListFiles(ctx context.Context, containerID, path string) ([]FileInfo, error) {
	// Check if container is running
	containerInfo, err := c.api.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}

	wasRunning := containerInfo.State.Running
	wasStarted := false

	// If stopped, temporarily start it
	if !wasRunning {
		if err := c.StartContainer(ctx, containerID); err != nil {
			return nil, fmt.Errorf("failed to start stopped container for file listing: %w", err)
		}
		wasStarted = true
		
		// Wait a moment for container to be ready
		time.Sleep(500 * time.Millisecond)
	}

	// Ensure we stop the container if we started it
	defer func() {
		if wasStarted && !wasRunning {
			// Give it a moment before stopping
			time.Sleep(100 * time.Millisecond)
			_ = c.StopContainer(ctx, containerID, 5*time.Second)
		}
	}()

	// Use ls -la to get detailed file info
	cmd := []string{"ls", "-la", "--time-style=long-iso", path}
	output, err := c.ContainerExecRun(ctx, containerID, cmd)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	return parseLsOutput(output, path), nil
}

// ContainerReadFile reads a file using cat command
// If container is stopped, it temporarily starts it, performs the operation, then stops it again
func (c *Client) ContainerReadFile(ctx context.Context, containerID, filePath string) ([]byte, error) {
	// Check if container is running
	containerInfo, err := c.api.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}

	wasRunning := containerInfo.State.Running
	wasStarted := false

	// If stopped, temporarily start it
	if !wasRunning {
		if err := c.StartContainer(ctx, containerID); err != nil {
			return nil, fmt.Errorf("failed to start stopped container for file read: %w", err)
		}
		wasStarted = true
		
		// Wait a moment for container to be ready
		time.Sleep(500 * time.Millisecond)
	}

	// Ensure we stop the container if we started it
	defer func() {
		if wasStarted && !wasRunning {
			// Give it a moment before stopping
			time.Sleep(100 * time.Millisecond)
			_ = c.StopContainer(ctx, containerID, 5*time.Second)
		}
	}()

	cmd := []string{"cat", filePath}
	output, err := c.ContainerExecRun(ctx, containerID, cmd)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return []byte(output), nil
}

// ContainerInspect checks if a container is running
// Returns the container information including state and mounts
func (c *Client) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	if c == nil || c.api == nil {
		return container.InspectResponse{}, ErrUninitialized
	}
	
	return c.api.ContainerInspect(ctx, containerID)
}

// ContainerUploadFiles uploads files to a container directory using Docker Copy API
// If container is stopped, it temporarily starts it, performs the upload, then stops it again
func (c *Client) ContainerUploadFiles(ctx context.Context, containerID, destPath string, files map[string][]byte) error {
	if c == nil || c.api == nil {
		return ErrUninitialized
	}

	// Check if container is running
	containerInfo, err := c.api.ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("inspect container: %w", err)
	}

	wasRunning := containerInfo.State.Running
	wasStarted := false

	// If stopped, temporarily start it
	if !wasRunning {
		if err := c.StartContainer(ctx, containerID); err != nil {
			return fmt.Errorf("failed to start stopped container for upload: %w", err)
		}
		wasStarted = true
		
		// Wait a moment for container to be ready
		time.Sleep(500 * time.Millisecond)
	}

	// Ensure we stop the container if we started it
	defer func() {
		if wasStarted && !wasRunning {
			// Give it a moment before stopping
			time.Sleep(100 * time.Millisecond)
			_ = c.StopContainer(ctx, containerID, 5*time.Second)
		}
	}()

	// Create a tar archive in memory
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Add files to tar archive
	for filePath, content := range files {
		// Normalize path
		filePath = strings.TrimPrefix(filePath, "/")
		
		hdr := &tar.Header{
			Name: filePath,
			Mode: 0644,
			Size: int64(len(content)),
		}
		
		if err := tw.WriteHeader(hdr); err != nil {
			tw.Close()
			return fmt.Errorf("write tar header: %w", err)
		}
		
		if _, err := tw.Write(content); err != nil {
			tw.Close()
			return fmt.Errorf("write tar content: %w", err)
		}
	}

	if err := tw.Close(); err != nil {
		return fmt.Errorf("close tar writer: %w", err)
	}

	// Upload tar archive to container
	err = c.api.CopyToContainer(ctx, containerID, destPath, &buf, client.CopyToContainerOptions{})
	if err != nil {
		return fmt.Errorf("copy to container: %w", err)
	}

	return nil
}

// FileInfo represents a file or directory
type FileInfo struct {
	Name        string
	Path        string
	IsDirectory bool
	Size        int64
	Permissions string
	ModifiedAt  string
}

// parseLsOutput parses ls -la output into FileInfo structs
func parseLsOutput(output, basePath string) []FileInfo {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var files []FileInfo

	// Skip header line (total X) and empty lines
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 9 {
			continue
		}

		perms := parts[0]
		isDir := perms[0] == 'd'

		var size int64
		fmt.Sscanf(parts[4], "%d", &size)

		name := strings.Join(parts[8:], " ")

		// Skip . and ..
		if name == "." || name == ".." {
			continue
		}

		modifiedAt := ""
		if len(parts) >= 7 {
			modifiedAt = strings.Join(parts[5:7], " ")
		}

		// Build full path
		fullPath := basePath
		if !strings.HasSuffix(fullPath, "/") && fullPath != "/" {
			fullPath += "/"
		}
		if fullPath == "/" {
			fullPath = ""
		}
		fullPath += name

		files = append(files, FileInfo{
			Name:        name,
			Path:        fullPath,
			IsDirectory: isDir,
			Size:        size,
			Permissions: perms,
			ModifiedAt:  modifiedAt,
		})
	}

	return files
}

// GetContainerVolumes returns information about persistent volumes mounted in the container
func (c *Client) GetContainerVolumes(ctx context.Context, containerID string) ([]VolumeMount, error) {
	containerInfo, err := c.api.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}

	var volumes []VolumeMount
	for _, mount := range containerInfo.Mounts {
		// Only include named volumes (persistent volumes)
		// Named volumes have Type="volume" and Name is set
		if mount.Type == "volume" && mount.Name != "" {
			volumes = append(volumes, VolumeMount{
				Name:       mount.Name,
				MountPoint: mount.Destination,
				Source:     mount.Source, // Host path where volume is stored
				IsNamed:    true,
			})
		}
	}

	return volumes, nil
}

// VolumeMount represents a volume mounted in a container
type VolumeMount struct {
	Name       string // Volume name
	MountPoint string // Where it's mounted in container (e.g., "/data")
	Source     string // Host filesystem path
	IsNamed    bool   // Whether this is a named volume (persistent)
}

// ListVolumeFiles lists files in a volume directly from the host filesystem
// This works even when the container is stopped
func (c *Client) ListVolumeFiles(volumePath, path string) ([]FileInfo, error) {
	fullPath := filepath.Join(volumePath, path)
	if path == "/" || path == "" {
		fullPath = volumePath
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		filePath := filepath.Join(path, entry.Name())
		if path == "/" || path == "" {
			filePath = "/" + entry.Name()
		}
		if !strings.HasPrefix(filePath, "/") {
			filePath = "/" + filePath
		}

		files = append(files, FileInfo{
			Name:        entry.Name(),
			Path:        filePath,
			IsDirectory: entry.IsDir(),
			Size:        info.Size(),
			Permissions: info.Mode().String(),
			ModifiedAt:  info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	return files, nil
}

// ReadVolumeFile reads a file from a volume directly from the host filesystem
// This works even when the container is stopped
func (c *Client) ReadVolumeFile(volumePath, filePath string) ([]byte, error) {
	// Remove leading slash if present
	filePath = strings.TrimPrefix(filePath, "/")
	fullPath := filepath.Join(volumePath, filePath)

	// Security check: ensure the path is within the volume
	absVolumePath, err := filepath.Abs(volumePath)
	if err != nil {
		return nil, fmt.Errorf("invalid volume path: %w", err)
	}
	absFilePath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}
	if !strings.HasPrefix(absFilePath, absVolumePath) {
		return nil, fmt.Errorf("file path outside volume boundary")
	}

	return os.ReadFile(fullPath)
}

// UploadVolumeFiles uploads files directly to a volume on the host filesystem
// This works even when the container is stopped
func (c *Client) UploadVolumeFiles(volumePath string, files map[string][]byte) error {
	for filePath, content := range files {
		// Remove leading slash if present
		filePath = strings.TrimPrefix(filePath, "/")
		fullPath := filepath.Join(volumePath, filePath)

		// Security check: ensure the path is within the volume
		absVolumePath, err := filepath.Abs(volumePath)
		if err != nil {
			return fmt.Errorf("invalid volume path: %w", err)
		}
		absFilePath, err := filepath.Abs(fullPath)
		if err != nil {
			return fmt.Errorf("invalid file path: %w", err)
		}
		if !strings.HasPrefix(absFilePath, absVolumePath) {
			return fmt.Errorf("file path outside volume boundary")
		}

		// Create directory if needed
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}

		// Write file
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
	}

	return nil
}
