package gameservers

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"api/docker"

	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// findContainerForGameServer finds the container for a game server
func (s *Service) findContainerForGameServer(ctx context.Context, gameServerID string, dcli *docker.Client) (string, error) {
	// Get game server from database
	gameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return "", fmt.Errorf("game server not found: %w", err)
	}

	if gameServer.ContainerID == nil || *gameServer.ContainerID == "" {
		return "", fmt.Errorf("game server %s has no container ID", gameServerID)
	}

	// Verify container exists
	_, err = dcli.ContainerInspect(ctx, *gameServer.ContainerID)
	if err != nil {
		return "", fmt.Errorf("container not found: %w", err)
	}

	return *gameServer.ContainerID, nil
}

// ListGameServerFiles lists files in a game server container or volume
func (s *Service) ListGameServerFiles(ctx context.Context, req *connect.Request[gameserversv1.ListGameServerFilesRequest]) (*connect.Response[gameserversv1.ListGameServerFilesResponse], error) {
	gameServerID := req.Msg.GetGameServerId()

	// Check permissions
	if err := s.checkGameServerPermission(ctx, gameServerID, "gameservers.view"); err != nil {
		return nil, err
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	// If we're only listing volumes, we can be more lenient with container lookup
	if req.Msg.GetListVolumes() {
		containerID, err := s.findContainerForGameServer(ctx, gameServerID, dcli)
		if err != nil {
			return connect.NewResponse(&gameserversv1.ListGameServerFilesResponse{
				Volumes:          []*gameserversv1.GameServerVolumeInfo{},
				ContainerRunning: false,
			}), nil
		}

		containerInfo, err := dcli.ContainerInspect(ctx, containerID)
		isRunning := err == nil && containerInfo.State.Running

		volumes, err := dcli.GetContainerVolumes(ctx, containerID)
		if err != nil {
			volumes = []docker.VolumeMount{}
		}

		volumeInfos := make([]*gameserversv1.GameServerVolumeInfo, len(volumes))
		for i, vol := range volumes {
			volumeInfos[i] = &gameserversv1.GameServerVolumeInfo{
				Name:         vol.Name,
				MountPoint:   vol.MountPoint,
				Source:       vol.Source,
				IsPersistent: vol.IsNamed,
			}
		}

		return connect.NewResponse(&gameserversv1.ListGameServerFilesResponse{
			Volumes:          volumeInfos,
			ContainerRunning: isRunning,
		}), nil
	}

	// Normal file listing (not just volumes)
	containerID, err := s.findContainerForGameServer(ctx, gameServerID, dcli)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	containerInfo, err := dcli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to inspect container: %w", err))
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
	path = strings.TrimSpace(path)
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
		volumes, err := dcli.GetContainerVolumes(ctx, containerID)
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

		fileInfos, err := dcli.ListVolumeFiles(targetVolume.Source, path)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list volume files: %w", err))
		}

		pagedInfos, hasMore, nextCursor := paginateFileInfos(fileInfos, cursor, pageSize)
		files := make([]*gameserversv1.GameServerFile, 0, len(pagedInfos))
		for _, fi := range pagedInfos {
			files = append(files, fileInfoToProto(fi, volumeName))
		}

		resp := &gameserversv1.ListGameServerFilesResponse{
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
	fileInfos, err := dcli.ContainerListFiles(ctx, containerID, path)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "container is stopped") ||
			strings.Contains(errStr, "cannot be started automatically") ||
			strings.Contains(errStr, "failed to start stopped container") {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running and cannot be started automatically for file listing. Use volume_name parameter to access persistent volumes (volumes are accessible even when containers are stopped), or manually start the container"))
		}

		if strings.Contains(errStr, "failed with exit code") || strings.Contains(errStr, "command") {
			if !isRunning {
				return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running and the file listing command failed. Use volume_name parameter to access persistent volumes (volumes are accessible even when containers are stopped), or start the container"))
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list files: %w", err))
		}

		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list files: %w", err))
	}

	pagedInfos, hasMore, nextCursor := paginateFileInfos(fileInfos, cursor, pageSize)
	files := make([]*gameserversv1.GameServerFile, 0, len(pagedInfos))
	for _, fi := range pagedInfos {
		files = append(files, fileInfoToProto(fi, ""))
	}

	resp := &gameserversv1.ListGameServerFilesResponse{
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

// GetGameServerFile reads a file from a game server container or volume
func (s *Service) GetGameServerFile(ctx context.Context, req *connect.Request[gameserversv1.GetGameServerFileRequest]) (*connect.Response[gameserversv1.GetGameServerFileResponse], error) {
	gameServerID := req.Msg.GetGameServerId()

	// Check permissions
	if err := s.checkGameServerPermission(ctx, gameServerID, "gameservers.view"); err != nil {
		return nil, err
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
	path = strings.TrimSpace(path)
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

	containerID, err := s.findContainerForGameServer(ctx, gameServerID, dcli)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// If volume_name is specified, read file from volume
	volumeName := req.Msg.GetVolumeName()
	if volumeName != "" {
		volumes, err := dcli.GetContainerVolumes(ctx, containerID)
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

		resp := &gameserversv1.GetGameServerFileResponse{
			Content:   string(content),
			Encoding:  "text",
			Size:      int64(len(content)),
			Metadata:  metadata,
			Truncated: proto.Bool(false),
		}

		return connect.NewResponse(resp), nil
	}

	// Otherwise, read file from container filesystem (only works if running)
	containerInfo, err := dcli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to inspect container: %w", err))
	}

	if !containerInfo.State.Running {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running. Use volume_name parameter to read files from persistent volumes"))
	}

	// Read file using Docker exec (container must be running)
	content, err := dcli.ContainerReadFile(ctx, containerID, path)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read file: %w", err))
	}

	statInfo, err := dcli.ContainerStat(ctx, containerID, path)
	if err != nil {
		// Don't fail if stat fails - we can still return the content
		statInfo = nil
	}

	// Check if content is valid UTF-8
	encoding := "text"
	if !utf8.Valid(content) {
		// Invalid UTF-8 - encode as base64
		encoding = "base64"
		content = []byte(base64.StdEncoding.EncodeToString(content))
	}

	resp := &gameserversv1.GetGameServerFileResponse{
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

// UploadGameServerFiles uploads files to a game server container
func (s *Service) UploadGameServerFiles(ctx context.Context, req *connect.Request[gameserversv1.UploadGameServerFilesRequest]) (*connect.Response[gameserversv1.UploadGameServerFilesResponse], error) {
	metadata := req.Msg.GetMetadata()
	if metadata == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("metadata is required"))
	}

	fileData := req.Msg.GetTarData()
	if len(fileData) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("tar_data is required"))
	}

	gameServerID := metadata.GetGameServerId()

	// Check permissions
	if err := s.checkGameServerPermission(ctx, gameServerID, "gameservers.update"); err != nil {
		return nil, err
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

	containerID, err := s.findContainerForGameServer(ctx, gameServerID, dcli)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	volumeName := metadata.GetVolumeName()
	if volumeName != "" {
		// Upload to volume
		volumes, err := dcli.GetContainerVolumes(ctx, containerID)
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
		err = dcli.ContainerUploadFiles(ctx, containerID, destPath, files)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to upload files: %w", err))
		}
	}

	return connect.NewResponse(&gameserversv1.UploadGameServerFilesResponse{
		Success:       true,
		FilesUploaded: int32(len(files)),
	}), nil
}

// DeleteGameServerEntries removes files or directories from a game server
func (s *Service) DeleteGameServerEntries(ctx context.Context, req *connect.Request[gameserversv1.DeleteGameServerEntriesRequest]) (*connect.Response[gameserversv1.DeleteGameServerEntriesResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	paths := req.Msg.GetPaths()
	if len(paths) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("paths are required"))
	}

	// Check permissions
	if err := s.checkGameServerPermission(ctx, gameServerID, "gameservers.update"); err != nil {
		return nil, err
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	containerID, err := s.findContainerForGameServer(ctx, gameServerID, dcli)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	volumeName := req.Msg.GetVolumeName()
	recursive := req.Msg.GetRecursive()
	force := req.Msg.GetForce()

	deleted := make([]string, 0, len(paths))
	errs := make([]*gameserversv1.DeleteGameServerEntriesError, 0)

	if volumeName != "" {
		volumes, err := dcli.GetContainerVolumes(ctx, containerID)
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

		for _, p := range paths {
			if strings.TrimSpace(p) == "" {
				continue
			}
			hostPath, err := resolveVolumePath(targetVolume, p)
			if err != nil {
				errs = append(errs, &gameserversv1.DeleteGameServerEntriesError{Path: p, Message: err.Error()})
				continue
			}
			if err := removeVolumeEntry(hostPath, recursive, force); err != nil {
				errs = append(errs, &gameserversv1.DeleteGameServerEntriesError{Path: p, Message: err.Error()})
				continue
			}
			deleted = append(deleted, p)
		}

		return connect.NewResponse(&gameserversv1.DeleteGameServerEntriesResponse{
			Success:      len(errs) == 0,
			DeletedPaths: deleted,
			Errors:       errs,
		}), nil
	}

	containerInfo, err := dcli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to inspect container: %w", err))
	}

	if !containerInfo.State.Running {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running"))
	}

	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if err := dcli.ContainerRemoveEntries(ctx, containerID, []string{p}, recursive, force); err != nil {
			errs = append(errs, &gameserversv1.DeleteGameServerEntriesError{Path: p, Message: err.Error()})
			continue
		}
		deleted = append(deleted, p)
	}

	return connect.NewResponse(&gameserversv1.DeleteGameServerEntriesResponse{
		Success:      len(errs) == 0,
		DeletedPaths: deleted,
		Errors:       errs,
	}), nil
}

// CreateGameServerEntry creates an empty file, directory, or symlink within a game server
func (s *Service) CreateGameServerEntry(ctx context.Context, req *connect.Request[gameserversv1.CreateGameServerEntryRequest]) (*connect.Response[gameserversv1.CreateGameServerEntryResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	parentPath := req.Msg.GetParentPath()
	name := req.Msg.GetName()
	entryType := req.Msg.GetType()
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}
	if strings.Contains(name, "/") {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name cannot contain '/'"))
	}
	if entryType == gameserversv1.GameServerEntryType_GAME_SERVER_ENTRY_TYPE_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("type is required"))
	}

	// Check permissions
	if err := s.checkGameServerPermission(ctx, gameServerID, "gameservers.update"); err != nil {
		return nil, err
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	containerID, err := s.findContainerForGameServer(ctx, gameServerID, dcli)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	volumeName := req.Msg.GetVolumeName()

	targetPath := joinContainerPath(parentPath, name)
	mode := req.Msg.GetModeOctal()
	if mode == 0 {
		if entryType == gameserversv1.GameServerEntryType_GAME_SERVER_ENTRY_TYPE_DIRECTORY {
			mode = 0o755
		} else {
			mode = 0o644
		}
	}

	if volumeName != "" {
		volumes, err := dcli.GetContainerVolumes(ctx, containerID)
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

		hostPath, err := resolveVolumePath(targetVolume, targetPath)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		switch entryType {
		case gameserversv1.GameServerEntryType_GAME_SERVER_ENTRY_TYPE_DIRECTORY:
			if err := createVolumeDirectory(hostPath, os.FileMode(mode)); err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		case gameserversv1.GameServerEntryType_GAME_SERVER_ENTRY_TYPE_FILE:
			if err := createVolumeFile(hostPath, os.FileMode(mode)); err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		case gameserversv1.GameServerEntryType_GAME_SERVER_ENTRY_TYPE_SYMLINK:
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

		return connect.NewResponse(&gameserversv1.CreateGameServerEntryResponse{
			Entry: fileInfoToProto(info, volumeName),
		}), nil
	}

	containerInfo, err := dcli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to inspect container: %w", err))
	}

	if !containerInfo.State.Running {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running"))
	}

	switch entryType {
	case gameserversv1.GameServerEntryType_GAME_SERVER_ENTRY_TYPE_DIRECTORY:
		if err := dcli.ContainerCreateDirectory(ctx, containerID, targetPath, mode); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	case gameserversv1.GameServerEntryType_GAME_SERVER_ENTRY_TYPE_FILE:
		if err := dcli.ContainerWriteFile(ctx, containerID, targetPath, []byte{}, mode); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	case gameserversv1.GameServerEntryType_GAME_SERVER_ENTRY_TYPE_SYMLINK:
		target := req.Msg.GetTemplate()
		if target == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("template is required for symlink creation"))
		}
		if err := dcli.ContainerCreateSymlink(ctx, containerID, target, targetPath, true); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unsupported entry type"))
	}

	info, err := dcli.ContainerStat(ctx, containerID, targetPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stat new entry: %w", err))
	}

	return connect.NewResponse(&gameserversv1.CreateGameServerEntryResponse{
		Entry: fileInfoToProto(*info, ""),
	}), nil
}

// WriteGameServerFile writes or creates file contents within a game server
func (s *Service) WriteGameServerFile(ctx context.Context, req *connect.Request[gameserversv1.WriteGameServerFileRequest]) (*connect.Response[gameserversv1.WriteGameServerFileResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	pathValue := req.Msg.GetPath()
	if pathValue == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path is required"))
	}

	// Check permissions
	if err := s.checkGameServerPermission(ctx, gameServerID, "gameservers.update"); err != nil {
		return nil, err
	}

	content, err := decodeFileContent(req.Msg.GetContent(), req.Msg.GetEncoding())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// For server.properties files, filter out restricted properties that are managed by the platform
	if strings.HasSuffix(pathValue, "server.properties") {
		content = sanitizeServerProperties(content)
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	containerID, err := s.findContainerForGameServer(ctx, gameServerID, dcli)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	volumeName := req.Msg.GetVolumeName()
	createIfMissing := req.Msg.GetCreateIfMissing()
	mode := req.Msg.GetModeOctal()
	if volumeName != "" {
		volumes, err := dcli.GetContainerVolumes(ctx, containerID)
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

		return connect.NewResponse(&gameserversv1.WriteGameServerFileResponse{
			Success: true,
			Entry:   fileInfoToProto(info, volumeName),
		}), nil
	}

	containerInfo, err := dcli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to inspect container: %w", err))
	}

	if !containerInfo.State.Running {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running"))
	}

	var priorInfo *docker.FileInfo
	if !createIfMissing {
		var statErr error
		priorInfo, statErr = dcli.ContainerStat(ctx, containerID, pathValue)
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

	if err := dcli.ContainerWriteFile(ctx, containerID, pathValue, content, modeToUse); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	info, err := dcli.ContainerStat(ctx, containerID, pathValue)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stat file: %w", err))
	}

	return connect.NewResponse(&gameserversv1.WriteGameServerFileResponse{
		Success: true,
		Entry:   fileInfoToProto(*info, ""),
	}), nil
}

// RenameGameServerEntry renames a file or directory in a game server
func (s *Service) RenameGameServerEntry(ctx context.Context, req *connect.Request[gameserversv1.RenameGameServerEntryRequest]) (*connect.Response[gameserversv1.RenameGameServerEntryResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	sourcePath := req.Msg.GetSourcePath()
	targetPath := req.Msg.GetTargetPath()

	// Check permissions
	if err := s.checkGameServerPermission(ctx, gameServerID, "gameservers.update"); err != nil {
		return nil, err
	}

	dcli, err := docker.New()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("docker client: %w", err))
	}
	defer dcli.Close()

	containerID, err := s.findContainerForGameServer(ctx, gameServerID, dcli)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	volumeName := req.Msg.GetVolumeName()
	overwrite := req.Msg.GetOverwrite()

	if volumeName != "" {
		volumes, err := dcli.GetContainerVolumes(ctx, containerID)
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

		sourceHostPath, err := resolveVolumePath(targetVolume, sourcePath)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		targetHostPath, err := resolveVolumePath(targetVolume, targetPath)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		if err := renameVolumeEntry(sourceHostPath, targetHostPath, overwrite); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		info, err := dcli.StatVolumeFile(targetVolume.Source, targetPath)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stat renamed entry: %w", err))
		}

		return connect.NewResponse(&gameserversv1.RenameGameServerEntryResponse{
			Success: true,
			Entry:   fileInfoToProto(info, volumeName),
		}), nil
	}

	containerInfo, err := dcli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("failed to inspect container: %w", err))
	}

	if !containerInfo.State.Running {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("container is not running"))
	}

	if err := dcli.ContainerRenameEntry(ctx, containerID, sourcePath, targetPath, overwrite); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	info, err := dcli.ContainerStat(ctx, containerID, targetPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stat renamed entry: %w", err))
	}

	return connect.NewResponse(&gameserversv1.RenameGameServerEntryResponse{
		Success: true,
		Entry:   fileInfoToProto(*info, ""),
	}), nil
}

// Helper functions (similar to deployments/files.go)

func fileInfoToProto(fi docker.FileInfo, volumeName string) *gameserversv1.GameServerFile {
	entry := &gameserversv1.GameServerFile{
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

// sanitizeServerProperties removes restricted properties from server.properties files
// These properties (server-port, server-ip) are managed by the platform and should not be user-editable
func sanitizeServerProperties(content []byte) []byte {
	// Properties that are managed by the platform and should be removed
	restrictedProperties := []string{"server-port", "server-ip"}

	lines := strings.Split(string(content), "\n")
	var filteredLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			filteredLines = append(filteredLines, line)
			continue
		}

		// Check if this line contains a restricted property
		shouldFilter := false
		for _, restricted := range restrictedProperties {
			// Match property at start of line (with optional whitespace)
			if strings.HasPrefix(trimmed, restricted+"=") {
				shouldFilter = true
				break
			}
		}

		if !shouldFilter {
			filteredLines = append(filteredLines, line)
		}
	}

	return []byte(strings.Join(filteredLines, "\n"))
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

