package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/api/types/filters"
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
	// Use interactive shell (-i) to ensure prompt is shown
	execConfig := container.ExecOptions{
		Cmd:          []string{"/bin/sh", "-i"}, // Interactive shell to show prompt
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

	// For TTY mode, the connection is bidirectional:
	// - Write to Conn to send input to the container
	// - Read from Conn to receive output from the container (raw, no headers in TTY mode)
	// Start the exec instance - this makes the connection active
	if err := c.api.ContainerExecStart(ctx, execIDResp.ID, container.ExecStartOptions{
		Detach: false,
		Tty:    true,
	}); err != nil {
		attachResp.Close()
		return nil, fmt.Errorf("start exec: %w", err)
	}

	// Return the attach connection which implements ReadWriteCloser
	// Note: In TTY mode, output is raw without headers (unlike non-TTY mode which has 8-byte headers)
	// Conn is a bidirectional stream that can be used for both reading and writing
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

	inspect, err := c.api.ContainerExecInspect(ctx, execIDResp.ID)
	if err != nil {
		return "", fmt.Errorf("inspect exec: %w", err)
	}
	if inspect.ExitCode != 0 {
		return stdout.String(), fmt.Errorf("command %q failed with exit code %d", strings.Join(cmd, " "), inspect.ExitCode)
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
			// If we can't start the container, return a clear error
			return nil, fmt.Errorf("container is stopped and cannot be started automatically for file listing. The container may be in an error state or require manual intervention. Error: %w", err)
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
		// Provide more context about the failure
		return nil, fmt.Errorf("failed to list files in %q: %w", path, err)
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

	if !strings.HasPrefix(destPath, "/") {
		destPath = "/" + destPath
	}

	var buf bytes.Buffer
	tarWriter := tar.NewWriter(&buf)

	for name, content := range files {
		if err := tarWriter.WriteHeader(&tar.Header{
			Name:    name,
			Mode:    0o644,
			Size:    int64(len(content)),
			ModTime: time.Now(),
		}); err != nil {
			return fmt.Errorf("write tar header: %w", err)
		}
		
		if _, err := tarWriter.Write(content); err != nil {
			return fmt.Errorf("write tar content: %w", err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("close tar writer: %w", err)
	}

	if err := c.api.CopyToContainer(ctx, containerID, destPath, &buf, client.CopyToContainerOptions{}); err != nil {
		return fmt.Errorf("copy to container: %w", err)
	}

	return nil
}

func (c *Client) ContainerRemoveEntries(ctx context.Context, containerID string, paths []string, recursive, force bool) error {
	if c == nil || c.api == nil {
		return ErrUninitialized
	}
	if len(paths) == 0 {
		return nil
	}
	cmd := []string{"rm"}
	if recursive {
		cmd = append(cmd, "-r")
	}
	if force {
		cmd = append(cmd, "-f")
	}
	cmd = append(cmd, "--")
	cmd = append(cmd, paths...)
	if _, err := c.ContainerExecRun(ctx, containerID, cmd); err != nil {
		return err
	}
	return nil
}

func (c *Client) ContainerRenameEntry(ctx context.Context, containerID, sourcePath, targetPath string, overwrite bool) error {
	if c == nil || c.api == nil {
		return ErrUninitialized
	}
	cmd := []string{"mv"}
	if overwrite {
		cmd = append(cmd, "-f")
	}
	cmd = append(cmd, sourcePath, targetPath)
	if _, err := c.ContainerExecRun(ctx, containerID, cmd); err != nil {
		return err
	}
	return nil
}

func (c *Client) ContainerCreateDirectory(ctx context.Context, containerID, path string, mode uint32) error {
	if c == nil || c.api == nil {
		return ErrUninitialized
	}
	if _, err := c.ContainerExecRun(ctx, containerID, []string{"mkdir", "-p", path}); err != nil {
		return err
	}
	if mode != 0 {
		chmod := fmt.Sprintf("%#o", mode)
		if _, err := c.ContainerExecRun(ctx, containerID, []string{"chmod", chmod, path}); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) ContainerWriteFile(ctx context.Context, containerID, filePath string, content []byte, mode uint32) error {
	if c == nil || c.api == nil {
		return ErrUninitialized
	}
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}
	destDir := filepath.Dir(filePath)
	if destDir == "." {
		destDir = "/"
	}
	var buf bytes.Buffer
	tarWriter := tar.NewWriter(&buf)
	fileMode := int64(0o644)
	if mode != 0 {
		fileMode = int64(mode)
	}
	if err := tarWriter.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     filepath.Base(filePath),
		Mode:     fileMode,
		Size:     int64(len(content)),
		ModTime:  time.Now(),
	}); err != nil {
		return fmt.Errorf("write tar header: %w", err)
	}
	if _, err := tarWriter.Write(content); err != nil {
		return fmt.Errorf("write tar content: %w", err)
	}
	if err := tarWriter.Close(); err != nil {
		return fmt.Errorf("close tar writer: %w", err)
	}
	if err := c.api.CopyToContainer(ctx, containerID, destDir, &buf, client.CopyToContainerOptions{AllowOverwriteDirWithFile: true}); err != nil {
		return fmt.Errorf("copy to container: %w", err)
	}
	return nil
}

func (c *Client) ContainerStat(ctx context.Context, containerID, path string) (*FileInfo, error) {
	if c == nil || c.api == nil {
		return nil, ErrUninitialized
	}
	base := filepath.Dir(path)
	if base == "" {
		base = "/"
	}
	output, err := c.ContainerExecRun(ctx, containerID, []string{"ls", "-ld", "--time-style=long-iso", path})
	if err != nil {
		return nil, err
	}
	infos := parseLsOutput(output, base)
	if len(infos) == 0 {
		return nil, fmt.Errorf("no metadata for path %s", path)
	}
	return &infos[0], nil
}

// FileInfo represents a file or directory
type FileInfo struct {
	Name          string
	Path          string
	IsDirectory   bool
	Size          int64
	Permissions   string
	Owner         string
	Group         string
	Mode          uint32
	ModifiedAt    time.Time
	IsSymlink     bool
	SymlinkTarget string
}

// parseLsOutput parses ls -la output into FileInfo structs
func parseLsOutput(output, basePath string) []FileInfo {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var files []FileInfo

	if len(lines) == 0 {
		return files
	}

	start := 0
	if strings.HasPrefix(strings.TrimSpace(lines[0]), "total ") {
		start = 1
	}

	for i := start; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 8 {
			continue
		}

		perms := parts[0]
		if len(perms) == 0 {
			continue
		}

		// Check file type from permissions string
		// 'd' = directory, 'l' = symlink, '-' = regular file, and others like 'c', 'b', etc.
		isDir := perms[0] == 'd'
		isSymlink := perms[0] == 'l'

		var size int64
		if len(parts) > 4 {
		fmt.Sscanf(parts[4], "%d", &size)
		}

		// File name starts at index 8, but handle cases where it might be different
		// Join all parts from index 8 onwards to handle spaces in filenames
		nameStartIdx := 7
		if len(parts) <= nameStartIdx {
			continue
		}

		rawName := strings.Join(parts[nameStartIdx:], " ")

		displayName := rawName
		symlinkTarget := ""
		if isSymlink {
			if arrow := strings.Index(rawName, " -> "); arrow >= 0 {
				displayName = strings.TrimSpace(rawName[:arrow])
				symlinkTarget = strings.TrimSpace(rawName[arrow+4:])
			}
		}

		// Skip . and ..
		if displayName == "." || displayName == ".." {
			continue
		}

		var modifiedAt time.Time
		if len(parts) >= 7 {
			timestamp := strings.Join(parts[5:7], " ")
			if ts, err := time.Parse("2006-01-02 15:04", timestamp); err == nil {
				modifiedAt = ts
			}
		}

		// Build full path with a leading slash so the UI has consistent absolute paths
		cleanBase := strings.Trim(strings.TrimSpace(basePath), "/")
		var fullPath string
		if cleanBase == "" {
			fullPath = path.Join("/", displayName)
		} else {
			fullPath = path.Join("/", cleanBase, displayName)
		}
		fullPath = path.Clean(fullPath)

		owner := ""
		if len(parts) > 2 {
			owner = parts[2]
		}

		group := ""
		if len(parts) > 3 {
			group = parts[3]
		}

		mode := permissionsToMode(perms)

		files = append(files, FileInfo{
			Name:          displayName,
			Path:          fullPath,
			IsDirectory:   isDir,
			Size:          size,
			Permissions:   perms,
			Owner:         owner,
			Group:         group,
			Mode:          mode,
			ModifiedAt:    modifiedAt,
			IsSymlink:     isSymlink,
			SymlinkTarget: symlinkTarget,
		})
	}

	return files
}

func permissionsToMode(perms string) uint32 {
	if len(perms) < 10 {
		return 0
	}
	bits := []uint32{0400, 0200, 0100, 0040, 0020, 0010, 0004, 0002, 0001}
	var mode uint32
	for idx, bit := range bits {
		// +1 to skip file type character
		if perms[idx+1] != '-' {
			mode |= bit
		}
	}
	return mode
}

// GetContainerVolumes returns information about persistent volumes mounted in the container
func (c *Client) GetContainerVolumes(ctx context.Context, containerID string) ([]VolumeMount, error) {
	containerInfo, err := c.api.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}

	var volumes []VolumeMount
	log.Printf("[GetContainerVolumes] Inspecting container %s, found %d mounts", containerID, len(containerInfo.Mounts))
	
	for i, mount := range containerInfo.Mounts {
		log.Printf("[GetContainerVolumes] Mount %d: Type=%s, Name=%s, Source=%s, Destination=%s", i, mount.Type, mount.Name, mount.Source, mount.Destination)
		
		// Include Docker volumes (Type="volume") and Obiente Cloud bind mounts
		// Obiente Cloud volumes are stored as bind mounts in /var/lib/obiente/volumes
		isObienteVolume := mount.Type == "bind" && strings.HasPrefix(mount.Source, "/var/lib/obiente/volumes")
		isDockerVolume := mount.Type == "volume"
		
		if isDockerVolume || isObienteVolume {
			volumePath := mount.Source
			log.Printf("[GetContainerVolumes] Processing volume mount: Name=%s, Source=%s", mount.Name, volumePath)

			// For Obiente Cloud volumes (bind mounts), use the source path directly
			if isObienteVolume {
				// Extract volume name from path: /var/lib/obiente/volumes/{deploymentID}/{volumeName}
				parts := strings.Split(strings.TrimPrefix(volumePath, "/var/lib/obiente/volumes/"), "/")
				volumeName := ""
				if len(parts) >= 2 {
					volumeName = parts[1] // Second part after deploymentID
				} else if len(parts) == 1 {
					// Fallback: use the last component
					volumeName = filepath.Base(volumePath)
				}
				if volumeName == "" {
					volumeName = filepath.Base(volumePath)
				}
				
				// Use mount destination as the display name if volume name is empty
				if volumeName == "" {
					volumeName = filepath.Base(mount.Destination)
				}
				
				if _, err := os.Stat(volumePath); err == nil {
					volumeMount := VolumeMount{
						Name:       volumeName,
						MountPoint: mount.Destination,
						Source:     volumePath,
						IsNamed:    true, // Obiente volumes are always "named" (persistent)
					}
					log.Printf("[GetContainerVolumes] Adding Obiente volume: Name=%s, MountPoint=%s, Source=%s", volumeMount.Name, volumeMount.MountPoint, volumeMount.Source)
					volumes = append(volumes, volumeMount)
				} else {
					log.Printf("[GetContainerVolumes] Skipping Obiente volume - path does not exist: %s (error: %v)", volumePath, err)
				}
				continue // Skip Docker volume processing for Obiente volumes
			}

			// For Docker volumes, try to resolve the correct volume path
			// Docker volumes are typically at /var/lib/docker/volumes/<name>/_data
			// but mount.Source might point to the volume directory or the _data subdirectory
			// For anonymous volumes, mount.Source is the direct path to the volume data
			
			isNamedVolume := mount.Name != ""
			if isNamedVolume {
				// Check if the path exists as-is
				if _, err := os.Stat(volumePath); os.IsNotExist(err) {
					// Path doesn't exist, try common variations
					// Try removing _data if it's in the path
					if strings.HasSuffix(volumePath, "_data") {
						volumePath = strings.TrimSuffix(volumePath, "_data")
						volumePath = strings.TrimSuffix(volumePath, "/")
					}

					// Try appending _data
					withData := filepath.Join(volumePath, "_data")
					if _, err := os.Stat(withData); err == nil {
						volumePath = withData
					} else if _, err := os.Stat(volumePath); os.IsNotExist(err) {
						// If still doesn't exist, construct from volume name
						// This is a fallback if mount.Source is completely wrong
						potentialPath := filepath.Join("/var/lib/docker/volumes", mount.Name, "_data")
						if _, err := os.Stat(potentialPath); err == nil {
							volumePath = potentialPath
						}
					}
				} else {
					// Path exists, but check if it's a directory (might be the volume root)
					// If it's not the _data directory, try appending _data
					info, err := os.Stat(volumePath)
					if err == nil && info.IsDir() {
						// Check if _data subdirectory exists
						withData := filepath.Join(volumePath, "_data")
						if _, err := os.Stat(withData); err == nil {
							// Prefer _data subdirectory as that's where actual files are
							volumePath = withData
						}
					}
				}
			}
			// For anonymous volumes, use mount.Source directly as it's already the correct path

			// Only include volumes with valid, accessible paths
			// Skip if the path still doesn't exist after all attempts
			if _, err := os.Stat(volumePath); err == nil {
				// Generate a display name for anonymous volumes
				displayName := mount.Name
				if displayName == "" {
					// Use a hash of the mount point or source path as identifier
					displayName = fmt.Sprintf("anonymous-%s", mount.Destination)
				}
				
				volumeMount := VolumeMount{
					Name:       displayName,
					MountPoint: mount.Destination,
					Source:     volumePath, // Host path where volume is stored
					IsNamed:    isNamedVolume,
				}
				log.Printf("[GetContainerVolumes] Adding volume: Name=%s, MountPoint=%s, Source=%s, IsNamed=%v", volumeMount.Name, volumeMount.MountPoint, volumeMount.Source, volumeMount.IsNamed)
				volumes = append(volumes, volumeMount)
			} else {
				log.Printf("[GetContainerVolumes] Skipping volume - path does not exist: %s (error: %v)", volumePath, err)
			}
			// If volume path doesn't exist, skip it (might be deleted or inaccessible)
		} else {
			log.Printf("[GetContainerVolumes] Skipping mount - not a volume type: Type=%s", mount.Type)
		}
	}

	log.Printf("[GetContainerVolumes] Returning %d volumes", len(volumes))
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
	// Verify the volume path exists first
	if _, err := os.Stat(volumePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("volume path does not exist: %s (volume may have been deleted)", volumePath)
	}

	// Resolve and validate the path to ensure it stays within the volume boundary
	resolvedPath, err := resolvePathWithinVolume(volumePath, path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	entries, err := os.ReadDir(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory does not exist: %s (volume path: %s)", resolvedPath, volumePath)
		}
		return nil, fmt.Errorf("read directory: %w", err)
	}

	var files []FileInfo
	for _, entry := range entries {
		entryPath := filepath.Join(resolvedPath, entry.Name())
		
		// Validate each entry path to prevent symlink attacks
		if _, err := resolvePathWithinVolume(volumePath, entryPath); err != nil {
			// Skip entries that escape the volume boundary (e.g., malicious symlinks)
			continue
		}
		
		info, err := os.Lstat(entryPath)
		if err != nil {
			continue
		}

		// Calculate relative path from volume root for display
		relativePath := strings.TrimPrefix(entryPath, volumePath)
		if relativePath == "" {
			relativePath = "/"
		}
		filePath := "/" + strings.TrimPrefix(relativePath, "/")

		owner := ""
		group := ""
		mode := permissionsToMode(info.Mode().String())
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			owner = strconv.FormatUint(uint64(stat.Uid), 10)
			group = strconv.FormatUint(uint64(stat.Gid), 10)
			mode = uint32(stat.Mode & 0o777)
		}

		isSymlink := info.Mode()&os.ModeSymlink != 0
		symlinkTarget := ""
		if isSymlink {
			if target, err := os.Readlink(entryPath); err == nil {
				symlinkTarget = target
			}
		}

		files = append(files, FileInfo{
			Name:          entry.Name(),
			Path:          filePath,
			IsDirectory:   info.IsDir(),
			Size:          info.Size(),
			Permissions:   info.Mode().String(),
			Owner:         owner,
			Group:         group,
			Mode:          mode,
			ModifiedAt:    info.ModTime(),
			IsSymlink:     isSymlink,
			SymlinkTarget: symlinkTarget,
		})
	}

	return files, nil
}

// resolvePathWithinVolume ensures a path stays within the volume boundary
// This prevents directory traversal attacks
func resolvePathWithinVolume(volumePath, requested string) (string, error) {
	// Get absolute paths to prevent symlink and traversal attacks
	absVolumePath, err := filepath.Abs(volumePath)
	if err != nil {
		return "", fmt.Errorf("invalid volume path: %w", err)
	}

	// Normalize the requested path
	trimmed := strings.TrimPrefix(requested, "/")
	if trimmed == "" {
		return absVolumePath, nil
	}

	// Join and resolve to absolute path
	joined := filepath.Join(absVolumePath, trimmed)
	absRequested, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Ensure the resolved path is within the volume boundary
	// Use string comparison with path separator to prevent escaping
	if absRequested != absVolumePath && !strings.HasPrefix(absRequested+string(os.PathSeparator), absVolumePath+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes volume boundary: %s (volume: %s)", absRequested, absVolumePath)
	}

	return absRequested, nil
}

// ReadVolumeFile reads a file from a volume directly from the host filesystem
// This works even when the container is stopped
func (c *Client) ReadVolumeFile(volumePath, filePath string) ([]byte, error) {
	// Resolve and validate the path to ensure it stays within the volume boundary
	resolvedPath, err := resolvePathWithinVolume(volumePath, filePath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	return os.ReadFile(resolvedPath)
}

// UploadVolumeFiles uploads files directly to a volume on the host filesystem
// This works even when the container is stopped
func (c *Client) UploadVolumeFiles(volumePath string, files map[string][]byte) error {
	for filePath, content := range files {
		// Resolve and validate the path to ensure it stays within the volume boundary
		resolvedPath, err := resolvePathWithinVolume(volumePath, filePath)
		if err != nil {
			return fmt.Errorf("invalid path for file %s: %w", filePath, err)
		}

		// Create directory if needed
		dir := filepath.Dir(resolvedPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}

		// Write file
		if err := os.WriteFile(resolvedPath, content, 0644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
	}

	return nil
}

func (c *Client) StatVolumeFile(volumePath, path string) (FileInfo, error) {
	if _, err := os.Stat(volumePath); os.IsNotExist(err) {
		return FileInfo{}, fmt.Errorf("volume path does not exist: %s", volumePath)
	}

	// Resolve and validate the path to ensure it stays within the volume boundary
	resolvedPath, err := resolvePathWithinVolume(volumePath, path)
	if err != nil {
		return FileInfo{}, fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Lstat(resolvedPath)
	if err != nil {
		return FileInfo{}, fmt.Errorf("stat: %w", err)
	}

	owner := ""
	group := ""
	mode := permissionsToMode(info.Mode().String())
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		owner = strconv.FormatUint(uint64(stat.Uid), 10)
		group = strconv.FormatUint(uint64(stat.Gid), 10)
		mode = uint32(stat.Mode & 0o777)
	}

	isSymlink := info.Mode()&os.ModeSymlink != 0
	symlinkTarget := ""
	if isSymlink {
		if target, err := os.Readlink(resolvedPath); err == nil {
			// Validate symlink target stays within volume
			if resolvedTarget, err := resolvePathWithinVolume(volumePath, target); err == nil {
				symlinkTarget = resolvedTarget
			}
		}
	}

	// Calculate relative path from volume root for display
	relativePath := strings.TrimPrefix(resolvedPath, volumePath)
	if relativePath == "" {
		relativePath = "/"
	}
	filePath := "/" + strings.TrimPrefix(relativePath, "/")
	
	name := filepath.Base(filePath)
	if name == "" || name == "." {
		name = "/"
	}

	return FileInfo{
		Name:          name,
		Path:          filePath,
		IsDirectory:   info.IsDir(),
		Size:          info.Size(),
		Permissions:   info.Mode().String(),
		Owner:         owner,
		Group:         group,
		Mode:          mode,
		ModifiedAt:    info.ModTime(),
		IsSymlink:     isSymlink,
		SymlinkTarget: symlinkTarget,
	}, nil
}

func (c *Client) ContainerCreateSymlink(ctx context.Context, containerID, target, linkPath string, overwrite bool) error {
	if c == nil || c.api == nil {
		return ErrUninitialized
	}
	cmd := []string{"ln", "-s"}
	if overwrite {
		cmd = append(cmd, "-f")
	}
	cmd = append(cmd, target, linkPath)
	if _, err := c.ContainerExecRun(ctx, containerID, cmd); err != nil {
		return err
	}
	return nil
}

// Events streams Docker events, filtered by the provided filters
// Returns a channel that emits events and a function to stop listening
func (c *Client) Events(ctx context.Context, filterMap map[string][]string) (<-chan events.Message, <-chan error, func(), error) {
	if c == nil || c.api == nil {
		return nil, nil, nil, ErrUninitialized
	}

	// Build event options
	// Convert map[string][]string to filters.Args
	filterArgs := filters.NewArgs()
	for key, values := range filterMap {
		for _, value := range values {
			filterArgs.Add(key, value)
		}
	}
	
	eventChan, errChan := c.api.Events(ctx, client.EventsListOptions{
		Since:   "",
		Until:   "",
		Filters: filterArgs,
	})

	// Return cleanup function
	cleanup := func() {
		// The event stream will close when context is cancelled
		// No explicit cleanup needed for the Docker API event stream
	}

	return eventChan, errChan, cleanup, nil
}
