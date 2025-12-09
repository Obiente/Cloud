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
	"sync"
	"syscall"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
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

	if _, err := c.api.ContainerStart(ctx, containerID, client.ContainerStartOptions{}); err != nil {
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

	if _, err := c.api.ContainerStop(ctx, containerID, client.ContainerStopOptions{Timeout: timeoutSeconds}); err != nil {
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
	if _, err := c.api.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{Force: force}); err != nil {
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
	if _, err := c.api.ContainerRestart(ctx, containerID, client.ContainerRestartOptions{Timeout: timeoutSeconds}); err != nil {
		return fmt.Errorf("docker: restart container %s: %w", containerID, err)
	}
	return nil
}

// ContainerLogs fetches the container logs as an io.ReadCloser.
// If follow is true, the logs will be streamed continuously.
// since and until are optional time.Time values for filtering logs by timestamp.
func (c *Client) ContainerLogs(ctx context.Context, containerID string, tail string, follow bool, since *time.Time, until *time.Time) (io.ReadCloser, error) {
	if c == nil || c.api == nil {
		return nil, ErrUninitialized
	}
	opts := client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Follow:     follow,
		Timestamps: true, // Enable timestamps for parsing
	}
	if since != nil {
		opts.Since = since.Format(time.RFC3339Nano)
	}
	if until != nil {
		opts.Until = until.Format(time.RFC3339Nano)
		// When using until for historical loading, we want to fetch a reasonable chunk at a time
		// Using "all" would try to read all logs before the timestamp, which is very slow
		// So we limit to a reasonable number (default 500 if not specified)
		if opts.Tail == "all" || opts.Tail == "" {
			// Use 500 as default for historical loading - this is a good balance
			// between getting enough logs and not timing out
			opts.Tail = "500"
		}
	}
	
	// Log the exact parameters being sent to Docker API for debugging
	containerIDShort := containerID
	if len(containerID) > 12 {
		containerIDShort = containerID[:12]
	}
	log.Printf("[ContainerLogs] Calling Docker API: containerID=%s, tail=%q, follow=%v, since=%v, until=%v", 
		containerIDShort, tail, follow, opts.Since, opts.Until)
	
	logs, err := c.api.ContainerLogs(ctx, containerID, opts)
	if err != nil {
		log.Printf("[ContainerLogs] Docker API error: %v", err)
		return nil, fmt.Errorf("docker: logs for %s: %w", containerID, err)
	}
	log.Printf("[ContainerLogs] Successfully obtained logs reader from Docker API")
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
	execConfig := client.ExecCreateOptions{
		Cmd:          []string{"/bin/sh", "-i"}, // Interactive shell to show prompt
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          true,
		Env:          []string{"TERM=xterm-256color"},
	}

	// Create exec instance
	execIDResp, err := c.api.ExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("create exec: %w", err)
	}

	// Attach to exec with TTY support
	attachResp, err := c.api.ExecAttach(ctx, execIDResp.ID, client.ExecAttachOptions{
		TTY: true,
	})
	if err != nil {
		return nil, fmt.Errorf("attach exec: %w", err)
	}

	// For TTY mode, the connection is bidirectional:
	// - Write to Conn to send input to the container
	// - Read from Conn to receive output from the container (raw, no headers in TTY mode)
	// Start the exec instance - this makes the connection active
	if _, err := c.api.ExecStart(ctx, execIDResp.ID, client.ExecStartOptions{
		Detach: false,
		TTY:    true,
	}); err != nil {
		attachResp.Close()
		return nil, fmt.Errorf("start exec: %w", err)
	}

	// Return the attach connection which implements ReadWriteCloser
	// Note: In TTY mode, output is raw without headers (unlike non-TTY mode which has 8-byte headers)
	// Conn is a bidirectional stream that can be used for both reading and writing
	return attachResp.Conn, nil
}

// ContainerAttach attaches to the container's main process stdin/stdout/stderr
// This attaches directly to PID 1 (the main process), not a new exec session
// Returns a ReadCloser (for stdout/stderr), WriteCloser (for stdin), and Close function
// If Tty is true, the reader will return raw output (no 8-byte headers)
// If Tty is false, the reader will demultiplex output with 8-byte headers
func (c *Client) ContainerAttach(ctx context.Context, containerID string, opts ContainerAttachOptions) (io.ReadCloser, io.WriteCloser, func() error, error) {
	if c == nil || c.api == nil {
		return nil, nil, nil, ErrUninitialized
	}

	attachOpts := client.ContainerAttachOptions{
		Stream: true,
		Stdin:  opts.Stdin,
		Stdout: opts.Stdout,
		Stderr: opts.Stderr,
	}

	attachResp, err := c.api.ContainerAttach(ctx, containerID, attachOpts)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("attach container: %w", err)
	}

	// Wrap the reader based on TTY mode
	var reader io.ReadCloser
	if opts.Tty {
		// TTY mode: raw output, no headers, use Conn directly for bidirectional communication
		// In TTY mode, stdout/stderr are combined and raw (no 8-byte headers)
		reader = &attachReadCloser{
			reader: attachResp.Conn, // In TTY mode, read from Conn directly
			closeFn: func() error {
				attachResp.Close()
				return nil
			},
		}
	} else {
		// Non-TTY mode: multiplexed output with 8-byte headers
		reader = &attachReadCloser{
			reader: attachResp.Reader, // In non-TTY mode, use Reader which handles headers
			closeFn: func() error {
				attachResp.Close()
				return nil
			},
		}
	}

	// Return reader, writer, and close function
	// Writer handles stdin (always use Conn for writing)
	// Close function wraps attachResp.Close() to return error
	closeFn := func() error {
		attachResp.Close()
		return nil
	}
	return reader, attachResp.Conn, closeFn, nil
}

// attachReadCloser wraps a bufio.Reader to implement io.ReadCloser
type attachReadCloser struct {
	reader  io.Reader
	closeFn func() error
	closed  bool
	mu      sync.Mutex
}

func (a *attachReadCloser) Read(p []byte) (n int, err error) {
	return a.reader.Read(p)
}

func (a *attachReadCloser) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed {
		return nil
	}
	a.closed = true
	if a.closeFn != nil {
		return a.closeFn()
	}
	return nil
}

// ContainerAttachOptions specifies what streams to attach to
type ContainerAttachOptions struct {
	Stdin  bool
	Stdout bool
	Stderr bool
	Tty    bool // Whether to attach with TTY mode (must match container TTY setting)
}

// ContainerExecRun runs a command in the container and returns the output
func (c *Client) ContainerExecRun(ctx context.Context, containerID string, cmd []string) (string, error) {
	if c == nil || c.api == nil {
		return "", ErrUninitialized
	}

	// Create exec configuration
	execConfig := client.ExecCreateOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	// Create exec instance
	execIDResp, err := c.api.ExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", fmt.Errorf("create exec: %w", err)
	}

	// Attach to exec to get output
	attachResp, err := c.api.ExecAttach(ctx, execIDResp.ID, client.ExecAttachOptions{})
	if err != nil {
		return "", fmt.Errorf("attach exec: %w", err)
	}
	defer attachResp.Close()

	// Read output - Docker multiplexes stdout/stderr with 8-byte headers
	// Format: [stream_type(1)][reserved(3)][payload_length(4 bytes, big-endian)]
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	outputDone := make(chan error, 1)

	go func() {
		header := make([]byte, 8)
		frameBuf := make([]byte, 32*1024)
		for {
			// Read 8-byte header
			if _, err := io.ReadFull(attachResp.Reader, header); err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					outputDone <- nil
					return
				}
				outputDone <- err
				return
			}

			streamType := header[0]
			// Read payload length (bytes 4-7, big-endian)
			payloadLength := int(uint32(header[4])<<24 | uint32(header[5])<<16 | uint32(header[6])<<8 | uint32(header[7]))

			if payloadLength == 0 {
				continue
			}

			// Ensure we have enough buffer space
			if payloadLength > len(frameBuf) {
				frameBuf = make([]byte, payloadLength)
			}

			// Read the payload
			payload := frameBuf[:payloadLength]
			if _, err := io.ReadFull(attachResp.Reader, payload); err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					// Partial frame, write what we have
					if streamType == 1 {
						stdout.Write(payload)
					} else if streamType == 2 {
						stderr.Write(payload)
					}
					outputDone <- nil
					return
				}
				outputDone <- err
				return
			}

			// Write to appropriate buffer
			if streamType == 1 { // stdout
				stdout.Write(payload)
			} else if streamType == 2 { // stderr
				stderr.Write(payload)
			}
		}
	}()

	// Start the exec
	if _, err := c.api.ExecStart(ctx, execIDResp.ID, client.ExecStartOptions{Detach: false}); err != nil {
		attachResp.Close()
		return "", fmt.Errorf("start exec: %w", err)
	}

	// Wait for output to complete
	if err := <-outputDone; err != nil && err != io.EOF {
		return "", fmt.Errorf("read output: %w", err)
	}

	inspect, err := c.api.ExecInspect(ctx, execIDResp.ID, client.ExecInspectOptions{})
	if err != nil {
		return "", fmt.Errorf("inspect exec: %w", err)
	}
	if inspect.ExitCode != 0 {
		errMsg := fmt.Sprintf("command %q failed with exit code %d", strings.Join(cmd, " "), inspect.ExitCode)
		if stderr.Len() > 0 {
			errMsg += ": " + stderr.String()
		}
		return stdout.String(), fmt.Errorf("%s", errMsg)
	}

	return stdout.String(), nil
}

// ContainerListFiles lists files in a directory using ls command
// If container is stopped, it temporarily starts it, performs the operation, then stops it again
func (c *Client) ContainerListFiles(ctx context.Context, containerID, path string) ([]FileInfo, error) {
	// Check if container is running
	containerInfo, err := c.api.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}

	wasRunning := containerInfo.Container.State.Running
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
	// Try GNU ls format first (--time-style=long-iso), fall back to BusyBox format (--full-time)
	cmd := []string{"ls", "-la", "--time-style=long-iso", path}
	log.Printf("[ContainerListFiles] Running command in container %s: %v", containerID, cmd)
	output, err := c.ContainerExecRun(ctx, containerID, cmd)
	if err != nil {
		log.Printf("[ContainerListFiles] Command failed: %v, output (if any): %q", err, output)
		
		// Check if error is due to unsupported option (BusyBox)
		errStr := err.Error()
		if strings.Contains(errStr, "unrecognized option") || strings.Contains(errStr, "time-style") {
			log.Printf("[ContainerListFiles] GNU ls format failed, trying BusyBox format (--full-time)")
			// Fall back to BusyBox-compatible format
			cmd = []string{"ls", "-la", "--full-time", path}
			output, err = c.ContainerExecRun(ctx, containerID, cmd)
			if err != nil {
				log.Printf("[ContainerListFiles] BusyBox format also failed: %v", err)
				// If we got some output despite the error, try to parse it (might be partial success)
				if output != "" {
					log.Printf("[ContainerListFiles] Attempting to parse partial output despite error")
					return parseLsOutput(output, path), nil
				}
				return nil, fmt.Errorf("failed to list files in %q: %w", path, err)
			}
		} else {
			// If we got some output despite the error, try to parse it (might be partial success)
			if output != "" {
				log.Printf("[ContainerListFiles] Attempting to parse partial output despite error")
				return parseLsOutput(output, path), nil
			}
			// Provide more context about the failure
			return nil, fmt.Errorf("failed to list files in %q: %w", path, err)
		}
	}

	log.Printf("[ContainerListFiles] Command succeeded, output length: %d chars", len(output))
	files := parseLsOutput(output, path)
	
	// Memory safety: Very high limit to allow normal operations while catching memory leaks
	// With 2G memory limit, we can handle large directories, but still need a hard cap
	// This limit is high enough that normal operations won't hit it, but a memory leak will
	const maxFilesLimit = 100000 // Very high limit - only catches genuine memory leaks
	if len(files) > maxFilesLimit {
		log.Printf("[ContainerListFiles] WARNING: Directory contains %d files (exceeds safety limit of %d). This may indicate a memory leak or misconfiguration. Limiting to %d files.", len(files), maxFilesLimit, maxFilesLimit)
		files = files[:maxFilesLimit]
	}
	
	return files, nil
}

// ContainerReadFile reads a file using cat command
// If container is stopped, it temporarily starts it, performs the operation, then stops it again
func (c *Client) ContainerReadFile(ctx context.Context, containerID, filePath string) ([]byte, error) {
	// Sanitize filePath to ensure it's valid
	filePath = strings.TrimSpace(filePath)
	filePath = filepath.ToSlash(filepath.Clean(filePath))
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}
	// Final validation
	if strings.Contains(filePath, "\x00") || strings.Contains(filePath, "..") {
		log.Printf("[ContainerReadFile] Invalid filePath detected: %q", filePath)
		return nil, fmt.Errorf("invalid file path: %q", filePath)
	}
	
	// Check if container is running
	containerInfo, err := c.api.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}

	wasRunning := containerInfo.Container.State.Running
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

	// Use cat directly - Docker exec passes arguments safely, so special characters in paths are handled
	// This works on both GNU coreutils and BusyBox
	cmd := []string{"cat", filePath}
	output, err := c.ContainerExecRun(ctx, containerID, cmd)
	if err != nil {
		log.Printf("[ContainerReadFile] cat command failed: %v", err)
		errStr := err.Error()
		
		// Check if file doesn't exist
		if strings.Contains(errStr, "No such file") || strings.Contains(errStr, "not found") {
			return nil, fmt.Errorf("file not found: %q", filePath)
		}
		
		// Check if it's a permission issue
		if strings.Contains(errStr, "Permission denied") || strings.Contains(errStr, "EACCES") {
			return nil, fmt.Errorf("permission denied reading file %q", filePath)
		}
		
		// Check if it's a directory
		if strings.Contains(errStr, "Is a directory") || strings.Contains(errStr, "EISDIR") {
			return nil, fmt.Errorf("path is a directory, not a file: %q", filePath)
		}
		
		// Return error with full context - the stderr should already be included in the error message
		return nil, fmt.Errorf("failed to read file %q: %w", filePath, err)
	}
	
	// Empty files are valid - return empty byte slice
	return []byte(output), nil
}

// ContainerInspect checks if a container is running
// Returns the container information including state and mounts
func (c *Client) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	if c == nil || c.api == nil {
		return container.InspectResponse{}, ErrUninitialized
	}
	
	result, err := c.api.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return container.InspectResponse{}, err
	}
	return result.Container, nil
}

// ContainerResize resizes the TTY for a container
// This is only effective if the container was created with TTY enabled
func (c *Client) ContainerResize(ctx context.Context, containerID string, height, width int) error {
	if c == nil || c.api == nil {
		return ErrUninitialized
	}
	
	_, err := c.api.ContainerResize(ctx, containerID, client.ContainerResizeOptions{
		Height: uint(height),
		Width:  uint(width),
	})
	return err
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

	if _, err := c.api.CopyToContainer(ctx, containerID, client.CopyToContainerOptions{
		DestinationPath: destPath,
		Content:         &buf,
	}); err != nil {
		return fmt.Errorf("copy to container: %w", err)
	}

	return nil
}

// ContainerUploadFromTar uploads a tar stream directly to a container path using Docker Copy API.
// The provided tarReader will be streamed directly to Docker without buffering the entire tar in memory.
func (c *Client) ContainerUploadFromTar(ctx context.Context, containerID, destPath string, tarReader io.Reader) error {
	if c == nil || c.api == nil {
		return ErrUninitialized
	}

	if !strings.HasPrefix(destPath, "/") {
		destPath = "/" + destPath
	}

	if _, err := c.api.CopyToContainer(ctx, containerID, client.CopyToContainerOptions{
		DestinationPath: destPath,
		Content:         tarReader,
	}); err != nil {
		return fmt.Errorf("copy to container: %w", err)
	}
	return nil
}

// UploadVolumeFromTar extracts a tar stream directly into a host volume path without buffering whole files in memory.
func (c *Client) UploadVolumeFromTar(volumePath string, tarReader io.Reader) error {
	// Ensure the volume path exists
	if _, err := os.Stat(volumePath); os.IsNotExist(err) {
		return fmt.Errorf("volume path does not exist: %s", volumePath)
	}

	tr := tar.NewReader(tarReader)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// Skip directories
		if hdr.Typeflag == tar.TypeDir {
			continue
		}

		// Construct destination path inside volume
		dest := filepath.Join(volumePath, filepath.Clean(hdr.Name))

		// Ensure path stays within volume
		resolved, err := resolvePathWithinVolume(volumePath, dest)
		if err != nil {
			return fmt.Errorf("invalid tar entry path: %w", err)
		}

		dir := filepath.Dir(resolved)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}

		// Create file and stream copy
		f, err := os.OpenFile(resolved, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
		if err != nil {
			return fmt.Errorf("create file: %w", err)
		}
		if _, err := io.CopyN(f, tr, hdr.Size); err != nil && err != io.EOF {
			f.Close()
			return fmt.Errorf("write file: %w", err)
		}
		f.Close()
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
	if _, err := c.api.CopyToContainer(ctx, containerID, client.CopyToContainerOptions{
		DestinationPath:           destDir,
		Content:                   &buf,
		AllowOverwriteDirWithFile: true,
	}); err != nil {
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
	
	// Try GNU ls format first (--time-style=long-iso), fall back to BusyBox format (--full-time)
	cmd := []string{"ls", "-ld", "--time-style=long-iso", path}
	output, err := c.ContainerExecRun(ctx, containerID, cmd)
	if err != nil {
		// Check if error is due to unsupported option (BusyBox)
		errStr := err.Error()
		if strings.Contains(errStr, "unrecognized option") || strings.Contains(errStr, "time-style") {
			// Fall back to BusyBox-compatible format
			cmd = []string{"ls", "-ld", "--full-time", path}
			output, err = c.ContainerExecRun(ctx, containerID, cmd)
			if err != nil {
				return nil, fmt.Errorf("failed to stat file %q: %w", path, err)
			}
		} else {
			return nil, fmt.Errorf("failed to stat file %q: %w", path, err)
		}
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
	// Sanitize basePath to ensure it's valid
	basePath = strings.TrimSpace(basePath)
	basePath = filepath.ToSlash(filepath.Clean(basePath))
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	// Final validation
	if strings.Contains(basePath, "\x00") || strings.Contains(basePath, "..") {
		log.Printf("[parseLsOutput] Invalid basePath detected: %q, defaulting to /", basePath)
		basePath = "/"
	}
	
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
			log.Printf("[parseLsOutput] Skipping line with < 8 fields (%d): %q", len(parts), line)
			continue
		}
		
		// Log first few lines to debug parsing issues
		if i < start+3 {
			log.Printf("[parseLsOutput] Parsing line %d: %q, fields: %d", i, line, len(parts))
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

		// Determine where the filename starts based on timestamp format
		// Standard format: perms links owner group size date time filename (8 fields, filename at index 7)
		// BusyBox --full-time: perms links owner group size date time timezone filename (9+ fields, filename at index 8)
		// GNU --full-time: perms links owner group size date time timezone filename (9+ fields, filename at index 8)
		// Check if we have a timezone field (parts[7] might be timezone like +0000, -0500, etc.)
		nameStartIdx := 7
		if len(parts) > 7 {
			// Check if parts[7] looks like a timezone (starts with + or - and is 4-6 characters)
			timezoneField := parts[7]
			if (strings.HasPrefix(timezoneField, "+") || strings.HasPrefix(timezoneField, "-")) && 
			   len(timezoneField) >= 5 && len(timezoneField) <= 6 {
				// This is a timezone field, filename starts at index 8
				nameStartIdx = 8
			} else if len(parts) > 8 {
				// Check parts[8] as well, in case format is different
				nextField := parts[8]
				// If parts[7] looks like a partial timestamp and parts[8] looks like timezone
				if (strings.HasPrefix(nextField, "+") || strings.HasPrefix(nextField, "-")) && 
				   len(nextField) >= 5 && len(nextField) <= 6 {
					nameStartIdx = 9
				}
			}
		}
		
		if len(parts) <= nameStartIdx {
			log.Printf("[parseLsOutput] Not enough fields for filename: %d fields, need > %d", len(parts), nameStartIdx)
			continue
		}

		rawName := strings.Join(parts[nameStartIdx:], " ")
		
		// Sanitize the raw name - remove any control characters or invalid path characters
		rawName = strings.TrimSpace(rawName)
		if rawName == "" {
			log.Printf("[parseLsOutput] Empty filename after join, skipping line")
			continue
		}

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
			// Try to parse timestamp - handle multiple formats:
			// 1. GNU ls --time-style=long-iso: "2006-01-02 15:04"
			// 2. BusyBox --full-time: "2006-01-02 15:04:05.123456 +0000" or "2006-01-02 15:04:05.123456"
			// 3. Standard ls: "Jan 02 15:04" or "Jan 02 2006"
			
			// For long-iso or full-time, join date and time parts
			timestamp := strings.Join(parts[5:7], " ")
			
			// Try GNU/BusyBox ISO format first (YYYY-MM-DD HH:MM:SS...)
			if ts, err := time.Parse("2006-01-02 15:04:05.000000 -0700", timestamp); err == nil {
				modifiedAt = ts
			} else if ts, err := time.Parse("2006-01-02 15:04:05 -0700", timestamp); err == nil {
				modifiedAt = ts
			} else if ts, err := time.Parse("2006-01-02 15:04:05.000000", timestamp); err == nil {
				modifiedAt = ts
			} else if ts, err := time.Parse("2006-01-02 15:04:05", timestamp); err == nil {
				modifiedAt = ts
			} else if ts, err := time.Parse("2006-01-02 15:04", timestamp); err == nil {
				// GNU ls --time-style=long-iso format
				modifiedAt = ts
			} else if len(parts) >= 8 {
				// Try with timezone if present
				timestampWithTZ := strings.Join(parts[5:8], " ")
				if ts, err := time.Parse("2006-01-02 15:04:05.000000 -0700", timestampWithTZ); err == nil {
					modifiedAt = ts
				} else if ts, err := time.Parse("2006-01-02 15:04:05 -0700", timestampWithTZ); err == nil {
					modifiedAt = ts
				}
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
	containerInfo, err := c.api.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}

	var volumes []VolumeMount
	log.Printf("[GetContainerVolumes] Inspecting container %s, found %d mounts", containerID, len(containerInfo.Container.Mounts))
	
	for i, mount := range containerInfo.Container.Mounts {
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
				// Extract volume name from path
				// Structure can be:
				// - /var/lib/obiente/volumes/{deploymentID}/{volumeName} (for deployments)
				// - /var/lib/obiente/volumes/{volumeName} (for game servers)
				parts := strings.Split(strings.TrimPrefix(volumePath, "/var/lib/obiente/volumes/"), "/")
				volumeName := ""
				if len(parts) >= 2 {
					// For deployments: second part after deploymentID
					volumeName = parts[1]
				} else if len(parts) == 1 {
					// For game servers: use the volume name directly
					volumeName = parts[0]
				}
				if volumeName == "" {
					// Fallback: use the last component of the path
					volumeName = filepath.Base(volumePath)
				}
				
				// Use mount destination as the display name if volume name is still empty
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
				// Generate a display name for volumes
				displayName := mount.Name
				if displayName == "" {
					// For anonymous volumes, create a name based on mount point
					// e.g., "/config" -> "config" or "/data" -> "data"
					if mount.Destination != "" {
						// Extract the last part of the mount point path
						mountPointName := strings.Trim(mount.Destination, "/")
						if mountPointName == "" {
							mountPointName = "root"
						}
						// Use the mount point name as the display name for anonymous volumes
						displayName = fmt.Sprintf("anonymous-%s", mountPointName)
					} else {
						displayName = "anonymous-volume"
					}
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

// SearchVolumeFiles recursively searches for files matching the query in a volume
func (c *Client) SearchVolumeFiles(volumePath, rootPath, query string, maxResults int, filesOnly, directoriesOnly bool) ([]FileInfo, error) {
	// Verify the volume path exists first
	if _, err := os.Stat(volumePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("volume path does not exist: %s", volumePath)
	}

	// Resolve and validate the root path
	resolvedRoot, err := resolvePathWithinVolume(volumePath, rootPath)
	if err != nil {
		return nil, fmt.Errorf("invalid root path: %w", err)
	}

	queryLower := strings.ToLower(query)
	var results []FileInfo

	// Use filepath.WalkDir for recursive search
	err = filepath.WalkDir(resolvedRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Skip directories we can't access
			return nil
		}

		// Check if we've reached max results
		if len(results) >= maxResults {
			return filepath.SkipAll
		}

		// Validate path stays within volume boundary
		if _, err := resolvePathWithinVolume(volumePath, path); err != nil {
			return nil // Skip this entry
		}

		// Check if name matches query (case-insensitive)
		name := d.Name()
		if !strings.Contains(strings.ToLower(name), queryLower) {
			return nil // Continue searching
		}

		// Apply filters
		if filesOnly && d.IsDir() {
			return nil // Skip directories
		}
		if directoriesOnly && !d.IsDir() {
			return nil // Skip files
		}

		// Get file info
		info, err := os.Lstat(path)
		if err != nil {
			return nil // Skip if we can't stat
		}

		// Calculate relative path from volume root
		relativePath := strings.TrimPrefix(path, volumePath)
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
			if target, err := os.Readlink(path); err == nil {
				symlinkTarget = target
			}
		}

		results = append(results, FileInfo{
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
		})

		return nil
	})

	return results, err
}

// SearchContainerFiles recursively searches for files matching the query in a container
func (c *Client) SearchContainerFiles(ctx context.Context, containerID, rootPath, query string, maxResults int, filesOnly, directoriesOnly bool) ([]FileInfo, error) {
	// Check if container is running
	containerInfo, err := c.api.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}

	wasRunning := containerInfo.Container.State.Running
	wasStarted := false

	// If stopped, temporarily start it
	if !wasRunning {
		if err := c.StartContainer(ctx, containerID); err != nil {
			return nil, fmt.Errorf("container is stopped and cannot be started automatically for file search: %w", err)
		}
		wasStarted = true
		time.Sleep(500 * time.Millisecond)
	}

	// Ensure we stop the container if we started it
	defer func() {
		if wasStarted && !wasRunning {
			time.Sleep(100 * time.Millisecond)
			_ = c.StopContainer(ctx, containerID, 5*time.Second)
		}
	}()

	// Use find command to search recursively
	// Escape special characters in query for shell safety
	queryEscaped := strings.ReplaceAll(query, `\`, `\\`)
	queryEscaped = strings.ReplaceAll(queryEscaped, `"`, `\"`)
	queryEscaped = strings.ReplaceAll(queryEscaped, `'`, `\'`)
	queryEscaped = strings.ReplaceAll(queryEscaped, `$`, `\$`)
	queryEscaped = strings.ReplaceAll(queryEscaped, "`", "\\`")
	
	// Build find command
	var findCmd []string
	if filesOnly {
		findCmd = []string{"find", rootPath, "-name", fmt.Sprintf("*%s*", queryEscaped), "-type", "f"}
	} else if directoriesOnly {
		findCmd = []string{"find", rootPath, "-name", fmt.Sprintf("*%s*", queryEscaped), "-type", "d"}
	} else {
		findCmd = []string{"find", rootPath, "-name", fmt.Sprintf("*%s*", queryEscaped)}
	}

	// Run find command
	output, err := c.ContainerExecRun(ctx, containerID, findCmd)
	if err != nil {
		// If find command fails, return empty results rather than error
		// (might be permission issues or path doesn't exist)
		return []FileInfo{}, nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var results []FileInfo
	queryLower := strings.ToLower(query)

	for _, line := range lines {
		if len(results) >= maxResults {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Verify the path matches (case-insensitive) - double check
		fileName := filepath.Base(line)
		if !strings.Contains(strings.ToLower(fileName), queryLower) {
			continue
		}

		// Get file info using stat
		stat, err := c.ContainerStat(ctx, containerID, line)
		if err != nil {
			continue // Skip if we can't stat
		}

		// Apply filters
		if filesOnly && stat.IsDirectory {
			continue
		}
		if directoriesOnly && !stat.IsDirectory {
			continue
		}

		results = append(results, *stat)
	}

	return results, nil
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
	// Convert map[string][]string to client.Filters
	filterArgs := make(client.Filters)
	for key, values := range filterMap {
		for _, value := range values {
			filterArgs.Add(key, value)
		}
	}
	
	eventsResult := c.api.Events(ctx, client.EventsListOptions{
		Since:   "",
		Until:   "",
		Filters: filterArgs,
	})
	eventChan := eventsResult.Messages
	errChan := eventsResult.Err

	// Return cleanup function
	cleanup := func() {
		// The event stream will close when context is cancelled
		// No explicit cleanup needed for the Docker API event stream
	}

	return eventChan, errChan, cleanup, nil
}
