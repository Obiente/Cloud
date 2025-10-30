package docker

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/moby/moby/client"
)

// ExecHelper provides helper methods for executing commands in containers
type ExecHelper struct {
	api client.APIClient
}

// NewExecHelper creates a new exec helper
func NewExecHelper(api client.APIClient) *ExecHelper {
	return &ExecHelper{api: api}
}

// RunCommand executes a command in a container and returns stdout
// Note: This uses the client interface - actual types depend on moby version
func (e *ExecHelper) RunCommand(ctx context.Context, containerID string, cmd []string) (string, error) {
	// Use Docker API directly - the exact type structure depends on moby version
	// For now, return error indicating this needs proper type definitions
	return "", fmt.Errorf("RunCommand: requires proper moby API type definitions - use docker CLI or update moby types")
}

// ListFiles lists files in a directory using ls command
func (e *ExecHelper) ListFiles(ctx context.Context, containerID, path string) ([]FileInfo, error) {
	// Use ls -la to get detailed file info
	cmd := []string{"ls", "-la", "--time-style=long-iso", path}
	output, err := e.RunCommand(ctx, containerID, cmd)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}

	return parseLsOutput(output), nil
}

// ReadFile reads a file using cat command
func (e *ExecHelper) ReadFile(ctx context.Context, containerID, filePath string) (string, error) {
	cmd := []string{"cat", filePath}
	return e.RunCommand(ctx, containerID, cmd)
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
func parseLsOutput(output string) []FileInfo {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var files []FileInfo

	// Skip header line (total X)
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

		files = append(files, FileInfo{
			Name:        name,
			IsDirectory: isDir,
			Size:        size,
			Permissions: perms,
			ModifiedAt:  modifiedAt,
		})
	}

	return files
}

