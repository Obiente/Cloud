package deployments

import (
	"archive/tar"
	"archive/zip"
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

	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ListContainerFiles lists files in a deployment container or volume
func (s *Service) ListContainerFiles(ctx context.Context, req *connect.Request[deploymentsv1.ListContainerFilesRequest]) (*connect.Response[deploymentsv1.ListContainerFilesResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.read", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
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
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.read", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
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

		// Check if file is a zip archive and extract it
		if isZipFile(hdr.Name) {
			extractedFiles, err := extractZipArchive(content, hdr.Name, destPath)
			if err != nil {
				// Log error but continue - upload the zip file as-is if extraction fails
				log.Printf("Failed to extract zip file %s: %v", hdr.Name, err)
				files[hdr.Name] = content
			} else {
				// Add extracted files to the files map
				for extractedPath, extractedContent := range extractedFiles {
					files[extractedPath] = extractedContent
				}
				// Don't include the zip file itself
			}
		} else {
			files[hdr.Name] = content
		}
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

// isZipFile checks if a file is a zip archive based on its name
func isZipFile(filename string) bool {
	name := strings.ToLower(filename)
	return strings.HasSuffix(name, ".zip")
}

// extractZipArchive extracts files from a zip archive and returns them as a map
// The map keys are relative paths from the destination path
func extractZipArchive(zipData []byte, zipFileName, destPath string) (map[string][]byte, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}

	extractedFiles := make(map[string][]byte)
	zipBaseName := strings.TrimSuffix(filepath.Base(zipFileName), filepath.Ext(zipFileName))

	for _, file := range zipReader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Open the file from the zip
		rc, err := file.Open()
		if err != nil {
			log.Printf("Failed to open file %s from zip: %v", file.Name, err)
			continue
		}

		// Read file content
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			log.Printf("Failed to read file %s from zip: %v", file.Name, err)
			continue
		}

		// Construct the relative path
		// If zip contains a single root directory, extract files relative to that
		// Otherwise, extract files relative to the zip file's base name
		var relativePath string
		if strings.Contains(file.Name, "/") {
			// File has a path, use it as-is but relative to destPath
			relativePath = file.Name
		} else {
			// File is at root of zip, place it relative to zip name
			relativePath = filepath.Join(zipBaseName, file.Name)
		}

		// Clean the path to avoid issues
		relativePath = filepath.Clean(relativePath)
		// Remove leading slash if present
		relativePath = strings.TrimPrefix(relativePath, "/")

		extractedFiles[relativePath] = content
	}

	return extractedFiles, nil
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

	var modeToUse uint32
	if mode != 0 {
		modeToUse = mode
	} else if priorInfo != nil {
		modeToUse = priorInfo.Mode
	} else {
		modeToUse = 0o644
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

// ExtractDeploymentFile extracts a zip file to a destination directory
func (s *Service) ExtractDeploymentFile(ctx context.Context, req *connect.Request[deploymentsv1.ExtractDeploymentFileRequest]) (*connect.Response[deploymentsv1.ExtractDeploymentFileResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	zipPath := req.Msg.GetZipPath()
	destPath := req.Msg.GetDestinationPath()

	if zipPath == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("zip_path is required"))
	}
	if destPath == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("destination_path is required"))
	}

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.update", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// Find container by container_id or service_name, or use first if neither specified
	containerID := req.Msg.GetContainerId()
	serviceName := req.Msg.GetServiceName()
	loc, err := s.findContainerForDeployment(ctx, deploymentID, containerID, serviceName, dcli)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Read the zip file
	var zipData []byte
	volumeName := req.Msg.GetVolumeName()
	if volumeName != "" {
		// Read from volume
		volumes, err := dcli.GetContainerVolumes(ctx, loc.ContainerID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to get volumes: %w", err))
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

		zipData, err = dcli.ReadVolumeFile(targetVolume.Source, zipPath)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read zip file: %w", err))
		}
	} else {
		// Read from container filesystem
		containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to inspect container: %w", err))
		}

		if !containerInfo.State.Running {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running. Use volume_name parameter to read files from persistent volumes"))
		}

		zipData, err = dcli.ContainerReadFile(ctx, loc.ContainerID, zipPath)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read zip file: %w", err))
		}
	}

	// Extract zip archive
	extractedFiles, err := extractZipArchiveToPath(zipData, destPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to extract zip: %w", err))
	}

	// Upload extracted files
	if volumeName != "" {
		volumes, err := dcli.GetContainerVolumes(ctx, loc.ContainerID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to get volumes: %w", err))
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

		err = dcli.UploadVolumeFiles(targetVolume.Source, extractedFiles)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to upload extracted files: %w", err))
		}
	} else {
		// Upload to container filesystem
		containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to inspect container: %w", err))
		}

		if !containerInfo.State.Running {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running"))
		}

		// Upload files to container
		err = dcli.ContainerUploadFiles(ctx, loc.ContainerID, destPath, extractedFiles)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to upload extracted files: %w", err))
		}
	}

	return connect.NewResponse(&deploymentsv1.ExtractDeploymentFileResponse{
		Success:       true,
		FilesExtracted: int32(len(extractedFiles)),
	}), nil
}

// extractZipArchiveToPath extracts files from a zip archive and returns them as a map
// The map keys are full paths relative to the destination path
func extractZipArchiveToPath(zipData []byte, destPath string) (map[string][]byte, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip: %w", err)
	}

	extractedFiles := make(map[string][]byte)

	for _, file := range zipReader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Open the file from the zip
		rc, err := file.Open()
		if err != nil {
			log.Printf("Failed to open file %s from zip: %v", file.Name, err)
			continue
		}

		// Read file content
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			log.Printf("Failed to read file %s from zip: %v", file.Name, err)
			continue
		}

		// Construct the full path relative to destination
		// Clean the file path to avoid directory traversal
		filePath := filepath.Clean(file.Name)
		// Remove leading slash if present
		filePath = strings.TrimPrefix(filePath, "/")
		// Join with destination path
		fullPath := filepath.Join(destPath, filePath)
		// Normalize to use forward slashes for consistency
		fullPath = filepath.ToSlash(fullPath)
		// Ensure it starts with /
		if !strings.HasPrefix(fullPath, "/") {
			fullPath = "/" + fullPath
		}

		extractedFiles[fullPath] = content
	}

	return extractedFiles, nil
}

// CreateDeploymentFileArchive creates a zip archive from files or folders
func (s *Service) CreateDeploymentFileArchive(ctx context.Context, req *connect.Request[deploymentsv1.CreateDeploymentFileArchiveRequest]) (*connect.Response[deploymentsv1.CreateDeploymentFileArchiveResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	orgID := req.Msg.GetOrganizationId()
	archiveReq := req.Msg.GetArchiveRequest()
	if archiveReq == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("archive_request is required"))
	}
	
	sourcePaths := archiveReq.GetSourcePaths()
	destPath := archiveReq.GetDestinationPath()
	includeParentFolder := archiveReq.GetIncludeParentFolder()

	if len(sourcePaths) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("source_paths are required"))
	}
	if destPath == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("destination_path is required"))
	}

	// Check permissions
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: "deployments.update", ResourceType: "deployment", ResourceID: deploymentID}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// Find container by container_id or service_name, or use first if neither specified
	containerID := req.Msg.GetContainerId()
	serviceName := req.Msg.GetServiceName()
	loc, err := s.findContainerForDeployment(ctx, deploymentID, containerID, serviceName, dcli)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	volumeName := req.Msg.GetVolumeName()

	// Collect all files to archive
	filesToArchive := make(map[string][]byte)
	var filesArchived int32

	for _, sourcePath := range sourcePaths {
		if sourcePath == "" {
			continue
		}

		var collectedFiles map[string][]byte
		var count int32
		var err error

		if volumeName != "" {
			collectedFiles, count, err = collectFilesFromVolume(dcli, ctx, loc.ContainerID, volumeName, sourcePath, includeParentFolder)
		} else {
			collectedFiles, count, err = collectFilesFromContainer(dcli, ctx, loc.ContainerID, sourcePath, includeParentFolder)
		}

		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to collect files from %s: %w", sourcePath, err))
		}

		// Merge collected files into the archive map
		for path, content := range collectedFiles {
			filesToArchive[path] = content
		}
		filesArchived += count
	}

	if len(filesToArchive) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("no files found to archive"))
	}

	// Create zip archive
	zipData, err := createZipArchive(filesToArchive)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create zip archive: %w", err))
	}

	// Determine zip file name from destination path
	zipFileName := filepath.Base(destPath)
	if !strings.HasSuffix(strings.ToLower(zipFileName), ".zip") {
		// If destination doesn't end with .zip, append it
		if destPath == "/" || strings.HasSuffix(destPath, "/") {
			// If destination is a directory, use first source path name
			if len(sourcePaths) > 0 && sourcePaths[0] != "" {
				baseName := filepath.Base(sourcePaths[0])
				if baseName == "" || baseName == "/" {
					baseName = "archive"
				}
				zipFileName = baseName + ".zip"
			} else {
				zipFileName = "archive.zip"
			}
			destPath = filepath.Join(destPath, zipFileName)
		} else {
			zipFileName = destPath + ".zip"
			destPath = zipFileName
		}
	} else {
		zipFileName = destPath
	}

	// Ensure destination path is absolute
	if !strings.HasPrefix(destPath, "/") {
		destPath = "/" + destPath
	}

	// Write zip file
	if volumeName != "" {
		volumes, err := dcli.GetContainerVolumes(ctx, loc.ContainerID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to get volumes: %w", err))
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

		// Write zip file to volume
		err = writeVolumeFile(filepath.Join(targetVolume.Source, strings.TrimPrefix(destPath, "/")), zipData, 0o644, true)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to write zip file: %w", err))
		}
	} else {
		// Write to container filesystem
		containerInfo, err := dcli.ContainerInspect(ctx, loc.ContainerID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to inspect container: %w", err))
		}

		if !containerInfo.State.Running {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running"))
		}

		err = dcli.ContainerWriteFile(ctx, loc.ContainerID, destPath, zipData, 0o644)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to write zip file: %w", err))
		}
	}

	return connect.NewResponse(&deploymentsv1.CreateDeploymentFileArchiveResponse{
		ArchiveResponse: &commonv1.CreateServerFileArchiveResponse{
			Success:       true,
			ArchivePath:   destPath,
			FilesArchived: filesArchived,
		},
	}), nil
}

// collectFilesFromVolume recursively collects files from a volume path
func collectFilesFromVolume(dcli *docker.Client, ctx context.Context, containerID, volumeName, sourcePath string, includeParentFolder bool) (map[string][]byte, int32, error) {
	volumes, err := dcli.GetContainerVolumes(ctx, containerID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get volumes: %w", err)
	}

	var targetVolume *docker.VolumeMount
	for _, vol := range volumes {
		if vol.Name == volumeName {
			targetVolume = &vol
			break
		}
	}

	if targetVolume == nil {
		return nil, 0, fmt.Errorf("volume not found: %s", volumeName)
	}

	files := make(map[string][]byte)
	var count int32

	// Get the base name for the archive entry
	baseName := filepath.Base(sourcePath)
	if baseName == "" || baseName == "/" {
		baseName = "archive"
	}

	// Resolve the actual path within the volume
	resolvedPath, err := resolvePathWithinVolume(targetVolume.Source, sourcePath)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid path: %w", err)
	}

	// Check if it's a file or directory
	info, err := os.Lstat(resolvedPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to stat path: %w", err)
	}

	if info.IsDir() {
		// Recursively collect files from directory
		err = collectFilesFromVolumeDir(targetVolume.Source, resolvedPath, sourcePath, files, &count, includeParentFolder, baseName)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to collect files from directory: %w", err)
		}
	} else {
		// Single file
		content, err := dcli.ReadVolumeFile(targetVolume.Source, sourcePath)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read file: %w", err)
		}

		archivePath := filepath.Base(sourcePath)
		if includeParentFolder {
			archivePath = filepath.Join(baseName, archivePath)
		}
		files[archivePath] = content
		count = 1
	}

	return files, count, nil
}

// collectFilesFromVolumeDir recursively collects files from a directory in a volume
func collectFilesFromVolumeDir(volumePath, resolvedPath, sourcePath string, files map[string][]byte, count *int32, includeParentFolder bool, baseName string) error {
	entries, err := os.ReadDir(resolvedPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(resolvedPath, entry.Name())
		relativePath := strings.TrimPrefix(entryPath, volumePath)
		if !strings.HasPrefix(relativePath, "/") {
			relativePath = "/" + relativePath
		}

		// Validate path is within volume
		if _, err := resolvePathWithinVolume(volumePath, relativePath); err != nil {
			continue
		}

		info, err := os.Lstat(entryPath)
		if err != nil {
			continue
		}

		if info.IsDir() {
			// Recursively process subdirectory
			err = collectFilesFromVolumeDir(volumePath, entryPath, relativePath, files, count, includeParentFolder, baseName)
			if err != nil {
				continue
			}
		} else {
			// Read file content
			content, err := os.ReadFile(entryPath)
			if err != nil {
				continue
			}

			// Determine archive path
			// Remove the source path prefix to get relative path within the source
			archivePath := strings.TrimPrefix(relativePath, sourcePath)
			archivePath = strings.TrimPrefix(archivePath, "/")
			
			// If includeParentFolder is true, prepend the base name
			// If false and archivePath is empty, it means this is the root file
			if includeParentFolder {
				if archivePath == "" {
					// Root file, use base name
					archivePath = baseName
				} else {
					archivePath = filepath.Join(baseName, archivePath)
				}
			}

			// Normalize path separators
			archivePath = filepath.ToSlash(archivePath)
			files[archivePath] = content
			*count++
		}
	}

	return nil
}

// collectFilesFromContainer recursively collects files from a container path
func collectFilesFromContainer(dcli *docker.Client, ctx context.Context, containerID, sourcePath string, includeParentFolder bool) (map[string][]byte, int32, error) {
	containerInfo, err := dcli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to inspect container: %w", err)
	}

	if !containerInfo.State.Running {
		return nil, 0, fmt.Errorf("container is not running")
	}

	files := make(map[string][]byte)
	var count int32

	// Get file info to determine if it's a directory
	// First, try to list the parent directory to see if sourcePath is a file or directory
	parentPath := filepath.Dir(sourcePath)
	if parentPath == "." || parentPath == "/" {
		parentPath = "/"
	}

	fileList, err := dcli.ContainerListFiles(ctx, containerID, parentPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list files: %w", err)
	}

	// Check if source is a directory or a single file
	isDirectory := false
	for _, file := range fileList {
		if file.Path == sourcePath {
			isDirectory = file.IsDirectory
			break
		}
	}

	// If not found in parent listing, try listing the path itself (might be a directory)
	if !isDirectory {
		dirList, err := dcli.ContainerListFiles(ctx, containerID, sourcePath)
		if err == nil && len(dirList) > 0 {
			// If we can list it and get results, it's a directory
			isDirectory = true
		}
	}

	baseName := filepath.Base(sourcePath)
	if baseName == "" || baseName == "/" {
		baseName = "archive"
	}

	if isDirectory {
		// Recursively collect files from directory
		err = collectFilesFromContainerDir(dcli, ctx, containerID, sourcePath, sourcePath, files, &count, includeParentFolder, baseName)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to collect files from directory: %w", err)
		}
	} else {
		// Single file
		content, err := dcli.ContainerReadFile(ctx, containerID, sourcePath)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read file: %w", err)
		}

		archivePath := filepath.Base(sourcePath)
		if includeParentFolder {
			archivePath = filepath.Join(baseName, archivePath)
		}
		files[archivePath] = content
		count = 1
	}

	return files, count, nil
}

// collectFilesFromContainerDir recursively collects files from a directory in a container
func collectFilesFromContainerDir(dcli *docker.Client, ctx context.Context, containerID, rootPath, currentPath string, files map[string][]byte, count *int32, includeParentFolder bool, baseName string) error {
	fileList, err := dcli.ContainerListFiles(ctx, containerID, currentPath)
	if err != nil {
		return err
	}

	for _, file := range fileList {
		if file.Path == currentPath {
			// Skip the directory itself
			continue
		}

		if file.IsDirectory {
			// Recursively process subdirectory
			err = collectFilesFromContainerDir(dcli, ctx, containerID, rootPath, file.Path, files, count, includeParentFolder, baseName)
			if err != nil {
				continue
			}
		} else {
			// Read file content
			content, err := dcli.ContainerReadFile(ctx, containerID, file.Path)
			if err != nil {
				continue
			}

			// Determine archive path
			// Remove the root path prefix to get relative path within the source
			archivePath := strings.TrimPrefix(file.Path, rootPath)
			archivePath = strings.TrimPrefix(archivePath, "/")
			
			// If includeParentFolder is true, prepend the base name
			// If false and archivePath is empty, it means this is the root file
			if includeParentFolder {
				if archivePath == "" {
					// Root file, use base name
					archivePath = baseName
				} else {
					archivePath = filepath.Join(baseName, archivePath)
				}
			}

			// Normalize path separators
			archivePath = filepath.ToSlash(archivePath)
			files[archivePath] = content
			*count++
		}
	}

	return nil
}

// createZipArchive creates a zip archive from a map of file paths to content
func createZipArchive(files map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	for archivePath, content := range files {
		// Ensure path uses forward slashes
		archivePath = filepath.ToSlash(archivePath)
		// Remove leading slash
		archivePath = strings.TrimPrefix(archivePath, "/")

		writer, err := zipWriter.Create(archivePath)
		if err != nil {
			zipWriter.Close()
			return nil, fmt.Errorf("failed to create zip entry %s: %w", archivePath, err)
		}

		_, err = writer.Write(content)
		if err != nil {
			zipWriter.Close()
			return nil, fmt.Errorf("failed to write zip entry %s: %w", archivePath, err)
		}
	}

	err := zipWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close zip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// resolvePathWithinVolume is a helper to resolve paths within volume boundaries
func resolvePathWithinVolume(volumePath, requested string) (string, error) {
	absVolumePath, err := filepath.Abs(volumePath)
	if err != nil {
		return "", fmt.Errorf("invalid volume path: %w", err)
	}

	trimmed := strings.TrimPrefix(requested, "/")
	if trimmed == "" {
		return absVolumePath, nil
	}

	joined := filepath.Join(absVolumePath, trimmed)
	absRequested, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	if absRequested != absVolumePath && !strings.HasPrefix(absRequested+string(os.PathSeparator), absVolumePath+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes volume boundary: %s (volume: %s)", absRequested, absVolumePath)
	}

	return absRequested, nil
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

