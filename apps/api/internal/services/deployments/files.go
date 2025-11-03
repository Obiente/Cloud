package deployments

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"api/docker"
	deploymentsv1 "api/gen/proto/obiente/cloud/deployments/v1"
	"api/internal/auth"
	"api/internal/database"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ListContainerFiles lists files in a deployment container or volume
func (s *Service) ListContainerFiles(ctx context.Context, req *connect.Request[deploymentsv1.ListContainerFilesRequest]) (*connect.Response[deploymentsv1.ListContainerFilesResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// If we're only listing volumes, we can be more lenient with container lookup
	if req.Msg.GetListVolumes() {
		// For volume listing, try to find any container from the deployment
		// Volumes are accessible even when containers are stopped, so use GetAllDeploymentLocations
		// instead of ValidateAndRefreshLocations which only returns running containers
		allLocations, locationsErr := database.GetAllDeploymentLocations(deploymentID)
		if locationsErr != nil {
			return connect.NewResponse(&deploymentsv1.ListContainerFilesResponse{
				Volumes:          []*deploymentsv1.VolumeInfo{},
				ContainerRunning: false,
			}), nil
		}

		// Try each container until we find one that works for listing volumes
		// We can list volumes even from stopped containers
		var volumes []docker.VolumeMount
		var isRunning bool
		foundVolumes := false
		for _, testLoc := range allLocations {
			containerInfo, inspectErr := dcli.ContainerInspect(ctx, testLoc.ContainerID)
			if inspectErr != nil {
				continue
			}
			if !foundVolumes {
				isRunning = containerInfo.State.Running
			}
			testVolumes, volErr := dcli.GetContainerVolumes(ctx, testLoc.ContainerID)
			if volErr == nil {
				volumes = testVolumes
				foundVolumes = true
				isRunning = containerInfo.State.Running
				break
			}
		}
		if !foundVolumes {
			volumes = []docker.VolumeMount{}
		}

		volumeInfos := make([]*deploymentsv1.VolumeInfo, len(volumes))
		for i, vol := range volumes {
			volumeInfos[i] = &deploymentsv1.VolumeInfo{
				Name:         vol.Name,
				MountPoint:   vol.MountPoint,
				Source:       vol.Source,
				IsPersistent: vol.IsNamed,
			}
		}

		return connect.NewResponse(&deploymentsv1.ListContainerFilesResponse{
			Volumes:          volumeInfos,
			ContainerRunning: isRunning,
		}), nil
	}

	// Normal file listing (not just volumes)
	// Find container by container_id or service_name, or use first if neither specified
	containerID := req.Msg.GetContainerId()
	serviceName := req.Msg.GetServiceName()
	loc, err := s.findContainerForDeployment(ctx, deploymentID, containerID, serviceName, dcli)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
	if err != nil {
		refreshedLocations, refreshErr := database.ValidateAndRefreshLocations(deploymentID)
		if refreshErr != nil || len(refreshedLocations) == 0 {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("container not found and could not be refreshed: %w", err))
		}
		// Try to find the same container again, or use first available
		for _, refLoc := range refreshedLocations {
			if refLoc.ContainerID == loc.ContainerID {
				*loc = refLoc
				break
			}
		}
		if loc.ContainerID != refreshedLocations[0].ContainerID {
			*loc = refreshedLocations[0]
		}
		containerInfo, err = dcli.ContainerInspect(ctx, loc.ContainerID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to inspect container after refresh: %w", err))
		}
	}
	isRunning := containerInfo.State.Running

	cursor := req.Msg.GetCursor()
	pageSize := req.Msg.GetPageSize()
	if pageSize < 0 {
		pageSize = 0
	}

	path := req.Msg.GetPath()
	if path == "" {
		path = "/"
	}
	
	// Sanitize path to prevent directory traversal attacks
	// Ensure path is absolute and normalized
	// Use path/filepath carefully - on Windows filepath.Clean might change slashes
	// For container paths, always use forward slashes
	path = strings.TrimSpace(path)
	// Remove any invalid characters that might have been parsed incorrectly
	path = strings.Trim(path, "\x00\r\n")
	
	// Ensure path is absolute and normalized (use Unix-style paths)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	
	// Clean up the path - remove any double slashes, resolve . and ..
	path = filepath.ToSlash(filepath.Clean(path))
	// Ensure it starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	
	// Final validation: path should only contain valid characters for Unix paths
	if strings.Contains(path, "\x00") || strings.Contains(path, "..") {
		path = "/"
	}

	volumeName := req.Msg.GetVolumeName()
	if volumeName != "" {
		volumes, err := dcli.GetContainerVolumes(ctx, loc.ContainerID)
		if err != nil {
			refreshedLoc, refreshErr := s.findContainerForDeployment(ctx, deploymentID, containerID, serviceName, dcli)
			if refreshErr != nil {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("container not found and could not be refreshed: %w", err))
			}
			loc = refreshedLoc
			volumes, err = dcli.GetContainerVolumes(ctx, loc.ContainerID)
			if err != nil {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to get volumes after refresh: %w", err))
			}
		}

		var targetVolume *docker.VolumeMount
		for _, vol := range volumes {
			// Match by name (works for both named and anonymous volumes)
			// Anonymous volumes have names like "anonymous-<mountpoint>"
			if vol.Name == volumeName {
				targetVolume = &vol
				break
			}
		}
		if targetVolume == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("volume not found: %s", volumeName))
		}

		fileInfos, err := dcli.ListVolumeFiles(targetVolume.Source, path)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list volume files: %w", err))
		}

		pagedInfos, hasMore, nextCursor := paginateFileInfos(fileInfos, cursor, pageSize)
		files := make([]*deploymentsv1.ContainerFile, 0, len(pagedInfos))
		for _, fi := range pagedInfos {
			files = append(files, fileInfoToProto(fi, volumeName))
		}

		resp := &deploymentsv1.ListContainerFilesResponse{
			Files:            files,
			CurrentPath:      path,
			IsVolume:         true,
			ContainerRunning: isRunning,
			HasMore:          hasMore,
		}
		if nextCursor != "" {
			resp.NextCursor = proto.String(nextCursor)
		}
		return connect.NewResponse(resp), nil
	}

	// Try to list files - ContainerListFiles will handle stopped containers by temporarily starting them
	// If the container can't be started automatically, it will return an error
	fileInfos, err := dcli.ContainerListFiles(ctx, loc.ContainerID, path)
	if err != nil {
		// Check if error mentions container being stopped or can't be started
		errStr := err.Error()
		if strings.Contains(errStr, "container is stopped") || 
		   strings.Contains(errStr, "cannot be started automatically") ||
		   strings.Contains(errStr, "failed to start stopped container") {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running and cannot be started automatically for file listing. Use volume_name parameter to access persistent volumes (volumes are accessible even when containers are stopped), or manually start the container"))
		}
		
		// Check if the error is from ContainerExecRun (command failed with exit code)
		if strings.Contains(errStr, "failed with exit code") || strings.Contains(errStr, "command") {
			if !isRunning {
				return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running and the file listing command failed. Use volume_name parameter to access persistent volumes (volumes are accessible even when containers are stopped), or start the container"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list files: %w", err))
		}
		
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list files: %w", err))
	}

	pagedInfos, hasMore, nextCursor := paginateFileInfos(fileInfos, cursor, pageSize)
	files := make([]*deploymentsv1.ContainerFile, 0, len(pagedInfos))
	for _, fi := range pagedInfos {
		files = append(files, fileInfoToProto(fi, ""))
	}

	resp := &deploymentsv1.ListContainerFilesResponse{
		Files:            files,
		CurrentPath:      path,
		IsVolume:         false,
		ContainerRunning: isRunning,
		HasMore:          hasMore,
	}
	if nextCursor != "" {
		resp.NextCursor = proto.String(nextCursor)
	}

	return connect.NewResponse(resp), nil
}

// GetContainerFile reads a file from a deployment container or volume
func (s *Service) GetContainerFile(ctx context.Context, req *connect.Request[deploymentsv1.GetContainerFileRequest]) (*connect.Response[deploymentsv1.GetContainerFileResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.view", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path is required"))
	}
	
	// Sanitize path to prevent directory traversal attacks
	// Ensure path is absolute and normalized
	// Use path/filepath carefully - on Windows filepath.Clean might change slashes
	// For container paths, always use forward slashes
	path = strings.TrimSpace(path)
	// Remove any invalid characters that might have been parsed incorrectly
	path = strings.Trim(path, "\x00\r\n")
	
	// Ensure path is absolute and normalized (use Unix-style paths)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	
	// Clean up the path - remove any double slashes, resolve . and ..
	path = filepath.ToSlash(filepath.Clean(path))
	// Ensure it starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	
	// Final validation: path should only contain valid characters for Unix paths
	if strings.Contains(path, "\x00") || strings.Contains(path, "..") {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid path: %q", path))
	}

	// Find container by container_id or service_name, or use first if neither specified
	containerID := req.Msg.GetContainerId()
	serviceName := req.Msg.GetServiceName()
	loc, err := s.findContainerForDeployment(ctx, deploymentID, containerID, serviceName, dcli)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// If volume_name is specified, read file from volume
	volumeName := req.Msg.GetVolumeName()
	if volumeName != "" {
		volumes, err := dcli.GetContainerVolumes(ctx, loc.ContainerID)
		if err != nil {
			// Container might have been deleted - try to refresh and find again
			refreshedLoc, refreshErr := s.findContainerForDeployment(ctx, deploymentID, containerID, serviceName, dcli)
			if refreshErr != nil {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("container not found and could not be refreshed: %w", err))
			}
			loc = refreshedLoc
			volumes, err = dcli.GetContainerVolumes(ctx, loc.ContainerID)
			if err != nil {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to get volumes after refresh: %w", err))
			}
		}

		var targetVolume *docker.VolumeMount
		for _, vol := range volumes {
			if vol.Name == volumeName {
				targetVolume = &vol
				break
			}
		}

		if targetVolume == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("volume not found: %s", volumeName))
		}

		// Resolve host path and ensure it is within the volume boundary
		if _, err := resolveVolumePath(targetVolume, path); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		content, err := dcli.ReadVolumeFile(targetVolume.Source, path)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read volume file: %w", err))
		}

		fileInfo, err := dcli.StatVolumeFile(targetVolume.Source, path)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stat volume file: %w", err))
		}

		metadata := fileInfoToProto(fileInfo, volumeName)

		resp := &deploymentsv1.GetContainerFileResponse{
			Content:   string(content),
			Encoding:  "text",
			Size:      int64(len(content)),
			Metadata:  metadata,
			Truncated: proto.Bool(false),
		}

		return connect.NewResponse(resp), nil
	}

	// Otherwise, read file from container filesystem (only works if running)
	containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
	if err != nil {
			// Container might have been deleted - try to refresh and find again
			refreshedLoc, refreshErr := s.findContainerForDeployment(ctx, deploymentID, containerID, serviceName, dcli)
			if refreshErr != nil {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("container not found and could not be refreshed: %w", err))
			}
			loc = refreshedLoc
		containerInfo, err = dcli.ContainerInspect(ctx, loc.ContainerID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to inspect container after refresh: %w", err))
		}
	}

	if !containerInfo.State.Running {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running. Use volume_name parameter to read files from persistent volumes"))
	}

	// Read file using Docker exec (container must be running)
	content, err := dcli.ContainerReadFile(ctx, loc.ContainerID, path)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read file: %w", err))
	}

	statInfo, err := dcli.ContainerStat(ctx, loc.ContainerID, path)
	if err != nil {
		log.Printf("[GetContainerFile] Failed to stat file (non-fatal, continuing): %v", err)
		// Don't fail if stat fails - we can still return the content
		statInfo = nil
	}

	// Check if content is valid UTF-8
	encoding := "text"
	if !utf8.Valid(content) {
		// Invalid UTF-8 - encode as base64
		encoding = "base64"
		content = []byte(base64.StdEncoding.EncodeToString(content))
		log.Printf("[GetContainerFile] Content contains invalid UTF-8, encoding as base64")
	}
	
	resp := &deploymentsv1.GetContainerFileResponse{
		Content:   string(content),
		Encoding:  encoding,
		Size:      int64(len(content)),
		Truncated: proto.Bool(false),
	}
	if statInfo != nil {
		resp.Metadata = fileInfoToProto(*statInfo, "")
	}
	
	return connect.NewResponse(resp), nil
}

// UploadContainerFiles uploads files to a deployment container
func (s *Service) UploadContainerFiles(ctx context.Context, req *connect.Request[deploymentsv1.UploadContainerFilesRequest]) (*connect.Response[deploymentsv1.UploadContainerFilesResponse], error) {
	metadata := req.Msg.GetMetadata()
	if metadata == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("metadata is required"))
	}

	fileData := req.Msg.GetTarData()
	if len(fileData) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("tar_data is required"))
	}

	deploymentID := metadata.GetDeploymentId()
	orgID := metadata.GetOrganizationId()

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.update", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Get deployment locations
	// Validate and refresh locations to ensure we have valid container IDs
	locations, err := database.ValidateAndRefreshLocations(deploymentID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to validate locations: %w", err))
	}
	if len(locations) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no containers for deployment"))
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	destPath := metadata.GetDestinationPath()
	if destPath == "" {
		destPath = "/"
	}

	// Extract files from tar archive
	files := make(map[string][]byte)
	tarReader := tar.NewReader(bytes.NewReader(fileData))
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read tar: %w", err))
		}

		// Only process regular files
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		// Read file content
		content, err := io.ReadAll(tarReader)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read file from tar: %w", err))
		}

		files[hdr.Name] = content
	}

	volumeName := metadata.GetVolumeName()
	if volumeName != "" {
		// Upload to volume
		volumes, err := dcli.GetContainerVolumes(ctx, deploymentID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get volumes: %w", err))
		}

		var targetVolume *docker.VolumeMount
		for _, vol := range volumes {
			if vol.Name == volumeName {
				targetVolume = &vol
				break
			}
		}

		if targetVolume == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("volume not found: %s", volumeName))
		}

		// Upload files directly to volume path
		// Combine destination path with file paths from tar archive
		uploadFiles := make(map[string][]byte)
		for filePath, content := range files {
			// Join destination path with the file path from archive
			fullPath := filepath.Join(destPath, filePath)
			uploadFiles[fullPath] = content
		}
		err = dcli.UploadVolumeFiles(targetVolume.Source, uploadFiles)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to upload files to volume: %w", err))
		}
	} else {
		// Upload to container filesystem
		// Find container by container_id or service_name from metadata, or use first if neither specified
		containerID := metadata.GetContainerId()
		serviceName := metadata.GetServiceName()
		loc, err := s.findContainerForDeployment(ctx, deploymentID, containerID, serviceName, dcli)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		err = dcli.ContainerUploadFiles(ctx, loc.ContainerID, destPath, files)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to upload files: %w", err))
		}
	}

	return connect.NewResponse(&deploymentsv1.UploadContainerFilesResponse{
		Success:       true,
		FilesUploaded: int32(len(files)),
	}), nil
}

// DeleteContainerEntries removes files or directories from a deployment
func (s *Service) DeleteContainerEntries(ctx context.Context, req *connect.Request[deploymentsv1.DeleteContainerEntriesRequest]) (*connect.Response[deploymentsv1.DeleteContainerEntriesResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	paths := req.Msg.GetPaths()
	if len(paths) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("paths are required"))
	}

	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.update", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// Find container by container_id or service_name, or use first if neither specified
	// Note: DeleteContainerEntriesRequest doesn't have container_id/service_name yet, so we use "" for now
	containerID := "" // TODO: Add container_id and service_name to DeleteContainerEntriesRequest
	serviceName := ""
	loc, err := s.findContainerForDeployment(ctx, deploymentID, containerID, serviceName, dcli)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	volumeName := req.Msg.GetVolumeName()
	recursive := req.Msg.GetRecursive()
	force := req.Msg.GetForce()

	deleted := make([]string, 0, len(paths))
	errs := make([]*deploymentsv1.DeleteContainerEntriesError, 0)

	if volumeName != "" {
		volumes, err := dcli.GetContainerVolumes(ctx, loc.ContainerID)
		if err != nil {
			refreshedLoc, refreshErr := s.findContainerForDeployment(ctx, deploymentID, containerID, serviceName, dcli)
			if refreshErr != nil {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("container not found and could not be refreshed: %w", err))
			}
			loc = refreshedLoc
			volumes, err = dcli.GetContainerVolumes(ctx, loc.ContainerID)
			if err != nil {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to get volumes after refresh: %w", err))
			}
		}

		var targetVolume *docker.VolumeMount
		for _, vol := range volumes {
			// Match by name (works for both named and anonymous volumes)
			// Anonymous volumes have names like "anonymous-<mountpoint>"
			if vol.Name == volumeName {
				targetVolume = &vol
				break
			}
		}
		if targetVolume == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("volume not found: %s", volumeName))
		}

		for _, p := range paths {
			if strings.TrimSpace(p) == "" {
				continue
			}
			hostPath, err := resolveVolumePath(targetVolume, p)
			if err != nil {
				errs = append(errs, &deploymentsv1.DeleteContainerEntriesError{Path: p, Message: err.Error()})
				continue
			}
			if err := removeVolumeEntry(hostPath, recursive, force); err != nil {
				errs = append(errs, &deploymentsv1.DeleteContainerEntriesError{Path: p, Message: err.Error()})
				continue
			}
			deleted = append(deleted, p)
		}

		return connect.NewResponse(&deploymentsv1.DeleteContainerEntriesResponse{
			Success:      len(errs) == 0,
			DeletedPaths: deleted,
			Errors:       errs,
		}), nil
	}

	containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
	if err != nil {
		refreshedLocations, refreshErr := database.ValidateAndRefreshLocations(deploymentID)
		if refreshErr != nil || len(refreshedLocations) == 0 {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("container not found and could not be refreshed: %w", err))
		}
		// Try to find the same container again, or use first available
		for _, refLoc := range refreshedLocations {
			if refLoc.ContainerID == loc.ContainerID {
				*loc = refLoc
				break
			}
		}
		if loc.ContainerID != refreshedLocations[0].ContainerID {
			*loc = refreshedLocations[0]
		}
		containerInfo, err = dcli.ContainerInspect(ctx, loc.ContainerID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to inspect container after refresh: %w", err))
		}
	}

	if !containerInfo.State.Running {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running"))
	}

	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if err := dcli.ContainerRemoveEntries(ctx, loc.ContainerID, []string{p}, recursive, force); err != nil {
			errs = append(errs, &deploymentsv1.DeleteContainerEntriesError{Path: p, Message: err.Error()})
			continue
		}
		deleted = append(deleted, p)
	}

	return connect.NewResponse(&deploymentsv1.DeleteContainerEntriesResponse{
		Success:      len(errs) == 0,
		DeletedPaths: deleted,
		Errors:       errs,
	}), nil
}

// CreateContainerEntry creates an empty file, directory, or symlink within a deployment
func (s *Service) CreateContainerEntry(ctx context.Context, req *connect.Request[deploymentsv1.CreateContainerEntryRequest]) (*connect.Response[deploymentsv1.CreateContainerEntryResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	parentPath := req.Msg.GetParentPath()
	name := req.Msg.GetName()
	entryType := req.Msg.GetType()
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}
	if strings.Contains(name, "/") {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name cannot contain '/'"))
	}
	if entryType == deploymentsv1.ContainerEntryType_CONTAINER_ENTRY_TYPE_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("type is required"))
	}

	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.update", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// Find container by container_id or service_name, or use first if neither specified
	// Note: CreateContainerEntryRequest doesn't have container_id/service_name yet, so we use "" for now
	containerID := "" // TODO: Add container_id and service_name to CreateContainerEntryRequest
	serviceName := ""
	loc, err := s.findContainerForDeployment(ctx, deploymentID, containerID, serviceName, dcli)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	volumeName := req.Msg.GetVolumeName()

	targetPath := joinContainerPath(parentPath, name)
	mode := req.Msg.GetModeOctal()
	if mode == 0 {
		if entryType == deploymentsv1.ContainerEntryType_CONTAINER_ENTRY_TYPE_DIRECTORY {
			mode = 0o755
		} else {
			mode = 0o644
		}
	}

	if volumeName != "" {
		volumes, err := dcli.GetContainerVolumes(ctx, loc.ContainerID)
		if err != nil {
			refreshedLoc, refreshErr := s.findContainerForDeployment(ctx, deploymentID, containerID, serviceName, dcli)
			if refreshErr != nil {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("container not found and could not be refreshed: %w", err))
			}
			loc = refreshedLoc
			volumes, err = dcli.GetContainerVolumes(ctx, loc.ContainerID)
			if err != nil {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to get volumes after refresh: %w", err))
			}
		}

		var targetVolume *docker.VolumeMount
		for _, vol := range volumes {
			// Match by name (works for both named and anonymous volumes)
			// Anonymous volumes have names like "anonymous-<mountpoint>"
			if vol.Name == volumeName {
				targetVolume = &vol
				break
			}
		}
		if targetVolume == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("volume not found: %s", volumeName))
		}

		hostPath, err := resolveVolumePath(targetVolume, targetPath)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		switch entryType {
		case deploymentsv1.ContainerEntryType_CONTAINER_ENTRY_TYPE_DIRECTORY:
			if err := createVolumeDirectory(hostPath, os.FileMode(mode)); err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		case deploymentsv1.ContainerEntryType_CONTAINER_ENTRY_TYPE_FILE:
			if err := createVolumeFile(hostPath, os.FileMode(mode)); err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		case deploymentsv1.ContainerEntryType_CONTAINER_ENTRY_TYPE_SYMLINK:
			target := req.Msg.GetTemplate()
			if target == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("template is required for symlink creation"))
			}
			if err := createVolumeSymlink(target, hostPath, true); err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unsupported entry type"))
		}

		info, err := dcli.StatVolumeFile(targetVolume.Source, targetPath)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stat new entry: %w", err))
		}

		return connect.NewResponse(&deploymentsv1.CreateContainerEntryResponse{
			Entry: fileInfoToProto(info, volumeName),
		}), nil
	}

	containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
	if err != nil {
		refreshedLocations, refreshErr := database.ValidateAndRefreshLocations(deploymentID)
		if refreshErr != nil || len(refreshedLocations) == 0 {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("container not found and could not be refreshed: %w", err))
		}
		// Try to find the same container again, or use first available
		for _, refLoc := range refreshedLocations {
			if refLoc.ContainerID == loc.ContainerID {
				*loc = refLoc
				break
			}
		}
		if loc.ContainerID != refreshedLocations[0].ContainerID {
			*loc = refreshedLocations[0]
		}
		containerInfo, err = dcli.ContainerInspect(ctx, loc.ContainerID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to inspect container after refresh: %w", err))
		}
	}

	if !containerInfo.State.Running {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running"))
	}

	switch entryType {
	case deploymentsv1.ContainerEntryType_CONTAINER_ENTRY_TYPE_DIRECTORY:
		if err := dcli.ContainerCreateDirectory(ctx, loc.ContainerID, targetPath, mode); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	case deploymentsv1.ContainerEntryType_CONTAINER_ENTRY_TYPE_FILE:
		if err := dcli.ContainerWriteFile(ctx, loc.ContainerID, targetPath, []byte{}, mode); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	case deploymentsv1.ContainerEntryType_CONTAINER_ENTRY_TYPE_SYMLINK:
		target := req.Msg.GetTemplate()
		if target == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("template is required for symlink creation"))
		}
		if err := dcli.ContainerCreateSymlink(ctx, loc.ContainerID, target, targetPath, true); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unsupported entry type"))
	}

	info, err := dcli.ContainerStat(ctx, loc.ContainerID, targetPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stat new entry: %w", err))
	}

	return connect.NewResponse(&deploymentsv1.CreateContainerEntryResponse{
		Entry: fileInfoToProto(*info, ""),
	}), nil
}

// WriteContainerFile writes or creates file contents within a deployment
func (s *Service) WriteContainerFile(ctx context.Context, req *connect.Request[deploymentsv1.WriteContainerFileRequest]) (*connect.Response[deploymentsv1.WriteContainerFileResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	pathValue := req.Msg.GetPath()
	if pathValue == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path is required"))
	}

	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.update", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	content, err := decodeFileContent(req.Msg.GetContent(), req.Msg.GetEncoding())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// Find container by container_id or service_name, or use first if neither specified
	// Note: WriteContainerFileRequest doesn't have container_id/service_name yet, so we use "" for now
	containerID := "" // TODO: Add container_id and service_name to WriteContainerFileRequest
	serviceName := ""
	loc, err := s.findContainerForDeployment(ctx, deploymentID, containerID, serviceName, dcli)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	volumeName := req.Msg.GetVolumeName()
	createIfMissing := req.Msg.GetCreateIfMissing()
	mode := req.Msg.GetModeOctal()
	if volumeName != "" {
		volumes, err := dcli.GetContainerVolumes(ctx, loc.ContainerID)
		if err != nil {
			refreshedLoc, refreshErr := s.findContainerForDeployment(ctx, deploymentID, containerID, serviceName, dcli)
			if refreshErr != nil {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("container not found and could not be refreshed: %w", err))
			}
			loc = refreshedLoc
			volumes, err = dcli.GetContainerVolumes(ctx, loc.ContainerID)
			if err != nil {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to get volumes after refresh: %w", err))
			}
		}

		var targetVolume *docker.VolumeMount
		for _, vol := range volumes {
			// Match by name (works for both named and anonymous volumes)
			// Anonymous volumes have names like "anonymous-<mountpoint>"
			if vol.Name == volumeName {
				targetVolume = &vol
				break
			}
		}
		if targetVolume == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("volume not found: %s", volumeName))
		}

		hostPath, err := resolveVolumePath(targetVolume, pathValue)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		var fileMode os.FileMode
		if mode != 0 {
			fileMode = os.FileMode(mode)
		} else {
			info, statErr := os.Lstat(hostPath)
			if statErr == nil {
				fileMode = info.Mode().Perm()
			} else if createIfMissing {
				fileMode = 0o644
			} else if os.IsNotExist(statErr) {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("file does not exist"))
			} else {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("stat file: %w", statErr))
			}
		}

		if err := writeVolumeFile(hostPath, content, fileMode, createIfMissing); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		info, err := dcli.StatVolumeFile(targetVolume.Source, pathValue)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stat file: %w", err))
		}

		return connect.NewResponse(&deploymentsv1.WriteContainerFileResponse{
			Success: true,
			Entry:   fileInfoToProto(info, volumeName),
		}), nil
	}

	containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
	if err != nil {
		refreshedLocations, refreshErr := database.ValidateAndRefreshLocations(deploymentID)
		if refreshErr != nil || len(refreshedLocations) == 0 {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("container not found and could not be refreshed: %w", err))
		}
		// Try to find the same container again, or use first available
		for _, refLoc := range refreshedLocations {
			if refLoc.ContainerID == loc.ContainerID {
				*loc = refLoc
				break
			}
		}
		if loc.ContainerID != refreshedLocations[0].ContainerID {
			*loc = refreshedLocations[0]
		}
		containerInfo, err = dcli.ContainerInspect(ctx, loc.ContainerID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to inspect container after refresh: %w", err))
		}
	}

	if !containerInfo.State.Running {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running"))
	}

	var priorInfo *docker.FileInfo
	if !createIfMissing {
		var statErr error
		priorInfo, statErr = dcli.ContainerStat(ctx, loc.ContainerID, pathValue)
		if statErr != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("file does not exist: %w", statErr))
		}
	}

	modeToUse := mode
	if modeToUse == 0 {
		if priorInfo != nil {
			modeToUse = priorInfo.Mode
		} else {
			modeToUse = 0o644
		}
	}

	if err := dcli.ContainerWriteFile(ctx, loc.ContainerID, pathValue, content, modeToUse); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	info, err := dcli.ContainerStat(ctx, loc.ContainerID, pathValue)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stat file: %w", err))
	}

	return connect.NewResponse(&deploymentsv1.WriteContainerFileResponse{
		Success: true,
		Entry:   fileInfoToProto(*info, ""),
	}), nil
}

// verifyContainersRunning verifies that at least one container for the deployment is actually running
func (s *Service) verifyContainersRunning(ctx context.Context, deploymentID string) error {
	// Get deployment locations
	// Validate and refresh locations to ensure we have valid container IDs
	locations, err := database.ValidateAndRefreshLocations(deploymentID)
	if err != nil {
		return fmt.Errorf("failed to validate deployment locations: %w", err)
	}

	if len(locations) == 0 {
		return fmt.Errorf("no containers found for deployment %s", deploymentID)
	}

	// Check if we have a manager to inspect containers
	if s.manager == nil {
		// If no manager, we can't verify, so assume OK if locations exist
		log.Printf("[verifyContainersRunning] Warning: No manager available to verify containers, assuming OK")
		return nil
	}

	// Get Docker client to inspect containers
	dcli, err := docker.New()
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	defer dcli.Close()

	var runningCount int
	for _, location := range locations {
		containerInfo, err := dcli.ContainerInspect(ctx, location.ContainerID)
		if err != nil {
			log.Printf("[verifyContainersRunning] Warning: Failed to inspect container %s: %v", location.ContainerID[:12], err)
			continue
		}

		if containerInfo.State.Running {
			runningCount++
		}
	}

	if runningCount == 0 {
		return fmt.Errorf("no running containers found for deployment %s (%d containers found but all are stopped)", deploymentID, len(locations))
	}

	log.Printf("[verifyContainersRunning] Verified %d running container(s) for deployment %s", runningCount, deploymentID)
	return nil
}

func fileInfoToProto(fi docker.FileInfo, volumeName string) *deploymentsv1.ContainerFile {
	entry := &deploymentsv1.ContainerFile{
		Name:        fi.Name,
		Path:        fi.Path,
		IsDirectory: fi.IsDirectory,
		Size:        fi.Size,
		Permissions: fi.Permissions,
	}
	if volumeName != "" {
		entry.VolumeName = proto.String(volumeName)
	}
	if fi.Owner != "" {
		entry.Owner = proto.String(fi.Owner)
	}
	if fi.Group != "" {
		entry.Group = proto.String(fi.Group)
	}
	if fi.Mode != 0 {
		entry.ModeOctal = proto.Uint32(fi.Mode)
	}
	if fi.IsSymlink {
		entry.IsSymlink = proto.Bool(true)
		if fi.SymlinkTarget != "" {
			entry.SymlinkTarget = proto.String(fi.SymlinkTarget)
		}
	}
	if !fi.ModifiedAt.IsZero() {
		if ts := timestamppb.New(fi.ModifiedAt); ts.IsValid() {
			entry.ModifiedTime = ts
		}
	}
	return entry
}

func paginateFileInfos(infos []docker.FileInfo, cursor string, pageSize int32) ([]docker.FileInfo, bool, string) {
	if len(infos) == 0 {
		return infos, false, ""
	}

	start := 0
	if cursor != "" {
		if idx, err := strconv.Atoi(cursor); err == nil && idx >= 0 {
			if idx >= len(infos) {
				return []docker.FileInfo{}, false, ""
			}
			start = idx
		}
	}

	end := len(infos)
	hasMore := false
	if pageSize > 0 && start+int(pageSize) < len(infos) {
		end = start + int(pageSize)
		hasMore = true
	}

	paged := infos[start:end]
	var nextCursor string
	if hasMore {
		nextCursor = strconv.Itoa(end)
	}

	return paged, hasMore, nextCursor
}

func resolveVolumePath(volume *docker.VolumeMount, requested string) (string, error) {
	if volume == nil {
		return "", fmt.Errorf("volume not provided")
	}
	root, err := filepath.Abs(volume.Source)
	if err != nil {
		return "", fmt.Errorf("resolve volume root: %w", err)
	}

	relative := strings.TrimPrefix(requested, "/")
	full := filepath.Join(root, relative)
	full, err = filepath.Abs(full)
	if err != nil {
		return "", fmt.Errorf("resolve volume path: %w", err)
	}

	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes volume")
	}

	return full, nil
}

func joinContainerPath(parent, name string) string {
	if strings.TrimSpace(parent) == "" {
		parent = "/"
	}
	joined := path.Join(parent, name)
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	return joined
}

func decodeFileContent(payload, encoding string) ([]byte, error) {
	enc := strings.ToLower(strings.TrimSpace(encoding))
	switch enc {
	case "", "text":
		return []byte(payload), nil
	case "base64":
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, fmt.Errorf("invalid base64 content: %w", err)
		}
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported encoding: %s", encoding)
	}
}

func removeVolumeEntry(path string, recursive, force bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) && force {
			return nil
		}
		return fmt.Errorf("remove entry: %w", err)
	}

	if info.IsDir() {
		if !recursive {
			return fmt.Errorf("cannot delete directory without recursive=true")
		}
		return os.RemoveAll(path)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove entry: %w", err)
	}
	return nil
}

func renameVolumeEntry(src, dst string, overwrite bool) error {
	if !overwrite {
		if _, err := os.Lstat(dst); err == nil {
			return fmt.Errorf("target already exists: %s", dst)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat target: %w", err)
		}
	} else {
		if err := os.RemoveAll(dst); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove existing target: %w", err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create target parent: %w", err)
	}

	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

func createVolumeDirectory(path string, mode os.FileMode) error {
	if err := os.MkdirAll(path, mode); err != nil {
		return fmt.Errorf("make directory: %w", err)
	}
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("chmod directory: %w", err)
	}
	return nil
}

func createVolumeFile(path string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directories: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("file already exists")
		}
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	if err := file.Chmod(mode); err != nil {
		return fmt.Errorf("chmod file: %w", err)
	}

	return nil
}

func createVolumeSymlink(target, link string, overwrite bool) error {
	if overwrite {
		if err := os.Remove(link); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove existing symlink: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return fmt.Errorf("create parent directories: %w", err)
	}
	if err := os.Symlink(target, link); err != nil {
		return fmt.Errorf("create symlink: %w", err)
	}
	return nil
}

func writeVolumeFile(path string, content []byte, mode os.FileMode, create bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directories: %w", err)
	}

	flags := os.O_WRONLY | os.O_TRUNC
	if create {
		flags |= os.O_CREATE | os.O_EXCL
	} else {
		if _, err := os.Lstat(path); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("file does not exist")
			}
			return fmt.Errorf("stat file: %w", err)
		}
	}

	file, err := os.OpenFile(path, flags, mode)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("file already exists")
		}
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(content); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	if mode != 0 {
		if err := file.Chmod(mode); err != nil {
			return fmt.Errorf("chmod file: %w", err)
		}
	}

	return nil
}

