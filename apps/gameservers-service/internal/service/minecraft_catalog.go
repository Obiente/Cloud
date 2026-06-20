package gameservers

import (
	"context"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gameservers-service/internal/catalog/modrinth"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"github.com/obiente/cloud/apps/shared/pkg/logger"

	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultDataVolumePrefix = "/var/lib/obiente/volumes"
	minecraftModsDir        = "/mods"
	minecraftPluginsDir     = "/plugins"
	minecraftMetadataFile   = ".obiente-minecraft-projects.json"
	maxSearchLimit          = 50
	maxVersionLimit         = 200 // Increased to allow fetching more versions
)

var downloadHTTPClient = &http.Client{
	Timeout: 4 * time.Minute,
}

type minecraftInstallMetadata struct {
	Files map[string]minecraftInstallMetadataEntry `json:"files"`
}

type minecraftInstallMetadataEntry struct {
	ProjectID     string                             `json:"project_id"`
	ProjectSlug   string                             `json:"project_slug,omitempty"`
	Title         string                             `json:"title,omitempty"`
	IconURL       string                             `json:"icon_url,omitempty"`
	ProjectType   gameserversv1.MinecraftProjectType `json:"project_type"`
	VersionID     string                             `json:"version_id"`
	VersionNumber string                             `json:"version_number"`
	GameVersions  []string                           `json:"game_versions,omitempty"`
	Loaders       []string                           `json:"loaders,omitempty"`
	Filename      string                             `json:"filename"`
	InstalledPath string                             `json:"installed_path"`
	SizeBytes     int64                              `json:"size_bytes,omitempty"`
	Hashes        map[string]string                  `json:"hashes,omitempty"`
	InstalledAt   time.Time                          `json:"installed_at"`
}

// ListMinecraftProjects integrates with Modrinth to surface mods/plugins.
func (s *Service) ListMinecraftProjects(ctx context.Context, req *connect.Request[gameserversv1.ListMinecraftProjectsRequest]) (*connect.Response[gameserversv1.ListMinecraftProjectsResponse], error) {
	gameServerID := strings.TrimSpace(req.Msg.GetGameServerId())
	if gameServerID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id is required"))
	}

	if err := s.checkGameServerPermission(ctx, gameServerID, auth.PermissionGameServersRead); err != nil {
		return nil, err
	}

	dbGameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
	}

	env := parseEnvVars(dbGameServer.EnvVars)
	serverType := strings.ToUpper(env["TYPE"])
	projectType := req.Msg.GetProjectType()
	if projectType == gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_UNSPECIFIED {
		projectType = defaultProjectType(serverType)
	}
	if projectType == gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_UNSPECIFIED {
		// Fallback to mods
		projectType = gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_MOD
	}

	loaders := dedupeStrings(req.Msg.GetLoaders())
	if len(loaders) == 0 {
		if inferred := loaderFromServerType(serverType); inferred != "" {
			loaders = []string{inferred}
		}
	}

	gameVersions := dedupeStrings(req.Msg.GetGameVersions())
	if len(gameVersions) == 0 {
		if dbGameServer.ServerVersion != nil && *dbGameServer.ServerVersion != "" {
			gameVersions = []string{normalizeVersionString(*dbGameServer.ServerVersion)}
		} else if v := env["VERSION"]; v != "" {
			gameVersions = []string{normalizeVersionString(v)}
		}
	}

	categories := dedupeStrings(req.Msg.GetCategories())

	limit := int(req.Msg.GetLimit())
	if limit <= 0 || limit > maxSearchLimit {
		limit = 20
	}

	offset, err := decodeCursor(req.Msg.GetCursor())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid cursor"))
	}

	params := modrinth.SearchParams{
		Query:        req.Msg.GetQuery(),
		Limit:        limit,
		Offset:       offset,
		ProjectType:  projectTypeToModrinth(projectType),
		Loaders:      loaders,
		GameVersions: gameVersions,
		Categories:   categories,
	}

	result, err := s.modClient.SearchProjects(ctx, params)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("mod catalog unavailable: %w", err))
	}

	items := make([]*gameserversv1.MinecraftProject, 0, len(result.Projects))
	for _, project := range result.Projects {
		items = append(items, mapProjectToProto(project))
	}

	hasMore := result.Offset+len(result.Projects) < result.TotalHits
	resp := &gameserversv1.ListMinecraftProjectsResponse{
		Projects: items,
		HasMore:  hasMore,
	}
	if hasMore {
		resp.NextCursor = proto.String(encodeCursor(result.Offset + len(result.Projects)))
	}

	return connect.NewResponse(resp), nil
}

// ListInstalledMinecraftProjects scans the server data volume for managed and unmanaged mod/plugin jars.
func (s *Service) ListInstalledMinecraftProjects(ctx context.Context, req *connect.Request[gameserversv1.ListInstalledMinecraftProjectsRequest]) (*connect.Response[gameserversv1.ListInstalledMinecraftProjectsResponse], error) {
	gameServerID := strings.TrimSpace(req.Msg.GetGameServerId())
	if gameServerID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id is required"))
	}
	if err := s.checkGameServerPermission(ctx, gameServerID, auth.PermissionGameServersRead); err != nil {
		return nil, err
	}

	dbGameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
	}

	env := parseEnvVars(dbGameServer.EnvVars)
	serverType := strings.ToUpper(env["TYPE"])
	projectType := req.Msg.GetProjectType()
	if projectType == gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_UNSPECIFIED {
		projectType = defaultProjectType(serverType)
		if projectType == gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_UNSPECIFIED {
			projectType = gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_MOD
		}
	}

	_, installDir, profile, err := s.resolveMinecraftInstallDirectory(ctx, dbGameServer.ID, serverType, projectType)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	metadata, err := loadMinecraftInstallMetadata(installDir)
	if err != nil {
		logger.Warn("[MinecraftCatalog] Failed to load install metadata for %s: %v", dbGameServer.ID, err)
		metadata = newMinecraftInstallMetadata()
	}

	entries, err := os.ReadDir(installDir)
	if err != nil {
		if os.IsNotExist(err) {
			return connect.NewResponse(&gameserversv1.ListInstalledMinecraftProjectsResponse{}), nil
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to scan %s: %w", profile.Description, err))
	}

	serverVersion := minecraftServerVersion(dbGameServer.ServerVersion, env)
	files := make([]*gameserversv1.InstalledMinecraftProjectFile, 0, len(entries))
	metadataDirty := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".jar") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			logger.Warn("[MinecraftCatalog] Failed to stat installed file %s: %v", entry.Name(), err)
			continue
		}

		meta, managed := metadata.Files[entry.Name()]
		if req.Msg.GetCheckUpdates() && !managed {
			discoveredMeta, ok := s.discoverMinecraftInstallMetadata(ctx, installDir, profile.InstallDir, entry.Name(), projectType)
			if ok {
				meta = discoveredMeta
				managed = true
				metadata.Files[entry.Name()] = discoveredMeta
				metadataDirty = true
			}
		}
		file := installedFileToProto(profile.InstallDir, entry.Name(), info, projectType, meta, managed)
		if req.Msg.GetCheckUpdates() && managed && meta.ProjectID != "" && meta.VersionID != "" {
			if latest := s.latestCompatibleVersion(ctx, meta.ProjectID, serverType, projectType, serverVersion); latest != nil && latest.ID != meta.VersionID {
				file.UpdateAvailable = true
				file.LatestVersionId = proto.String(latest.ID)
				file.LatestVersionNumber = proto.String(latest.VersionNumber)
				file.LatestGameVersions = latest.GameVersions
			}
		}
		files = append(files, file)
	}
	if metadataDirty {
		if err := saveMinecraftInstallMetadata(installDir, metadata); err != nil {
			logger.Warn("[MinecraftCatalog] Failed to persist discovered install metadata for %s: %v", dbGameServer.ID, err)
		}
	}

	return connect.NewResponse(&gameserversv1.ListInstalledMinecraftProjectsResponse{
		Files: files,
	}), nil
}

// GetMinecraftProjectVersions exposes version metadata for a project.
func (s *Service) GetMinecraftProjectVersions(ctx context.Context, req *connect.Request[gameserversv1.GetMinecraftProjectVersionsRequest]) (*connect.Response[gameserversv1.GetMinecraftProjectVersionsResponse], error) {
	gameServerID := strings.TrimSpace(req.Msg.GetGameServerId())
	if gameServerID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id is required"))
	}
	if err := s.checkGameServerPermission(ctx, gameServerID, auth.PermissionGameServersRead); err != nil {
		return nil, err
	}

	dbGameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
	}

	env := parseEnvVars(dbGameServer.EnvVars)
	serverType := strings.ToUpper(env["TYPE"])

	projectType := req.Msg.GetProjectType()
	if projectType == gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_UNSPECIFIED {
		projectType = defaultProjectType(serverType)
		if projectType == gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_UNSPECIFIED {
			projectType = gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_MOD
		}
	}

	limit := int(req.Msg.GetLimit())
	if limit <= 0 || limit > maxVersionLimit {
		limit = 100 // Increased default limit to show more versions
	}

	// High-limit requests are used by broad pickers. Plugins should still support
	// explicit game-version filters, but never get loader auto-filled.
	skipAutoFill := limit >= 50
	isPlugin := projectType == gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_PLUGIN

	loaders := dedupeStrings(req.Msg.GetLoaders())
	if isPlugin {
		loaders = nil
	} else if len(loaders) == 0 && !skipAutoFill {
		if inferred := loaderFromServerType(serverType); inferred != "" {
			loaders = []string{inferred}
		}
	}

	gameVersions := dedupeStrings(req.Msg.GetGameVersions())
	if len(gameVersions) == 0 && !skipAutoFill {
		if version := minecraftServerVersion(dbGameServer.ServerVersion, env); version != "" {
			gameVersions = []string{version}
		}
	}

	// Log filter details for debugging
	logger.Debug("[MinecraftCatalog] Fetching project versions: project_id=%s, project_type=%s, limit=%d, skip_auto_fill=%v, loaders=%v, game_versions=%v",
		req.Msg.GetProjectId(),
		projectType.String(),
		limit,
		skipAutoFill,
		loaders,
		gameVersions,
	)

	versions, err := s.modClient.GetProjectVersions(ctx, req.Msg.GetProjectId(), modrinth.VersionFilter{
		Loaders:      loaders,
		GameVersions: gameVersions,
		Limit:        limit,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load versions: %w", err))
	}

	logger.Debug("[MinecraftCatalog] Received versions from Modrinth: project_id=%s, version_count=%d",
		req.Msg.GetProjectId(),
		len(versions),
	)

	items := make([]*gameserversv1.MinecraftProjectVersion, 0, len(versions))
	for _, version := range versions {
		if strings.EqualFold(version.ServerSide, "unsupported") {
			continue
		}
		items = append(items, mapVersionToProto(version))
	}

	logger.Debug("[MinecraftCatalog] Filtered versions after server_side check: project_id=%s, final_version_count=%d",
		req.Msg.GetProjectId(),
		len(items),
	)

	return connect.NewResponse(&gameserversv1.GetMinecraftProjectVersionsResponse{
		Versions: items,
	}), nil
}

// GetMinecraftProject fetches full project details including body and gallery.
func (s *Service) GetMinecraftProject(ctx context.Context, req *connect.Request[gameserversv1.GetMinecraftProjectRequest]) (*connect.Response[gameserversv1.GetMinecraftProjectResponse], error) {
	gameServerID := strings.TrimSpace(req.Msg.GetGameServerId())
	if gameServerID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id is required"))
	}
	projectID := strings.TrimSpace(req.Msg.GetProjectId())
	if projectID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("project_id is required"))
	}

	if err := s.checkGameServerPermission(ctx, gameServerID, auth.PermissionGameServersRead); err != nil {
		return nil, err
	}

	project, err := s.modClient.GetProject(ctx, projectID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch project: %w", err))
	}

	return connect.NewResponse(&gameserversv1.GetMinecraftProjectResponse{
		Project: mapProjectToProto(*project),
	}), nil
}

// InstallMinecraftProjectFile downloads a selected version file into the server data volume.
func (s *Service) InstallMinecraftProjectFile(ctx context.Context, req *connect.Request[gameserversv1.InstallMinecraftProjectFileRequest]) (*connect.Response[gameserversv1.InstallMinecraftProjectFileResponse], error) {
	gameServerID := strings.TrimSpace(req.Msg.GetGameServerId())
	if gameServerID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id is required"))
	}
	if req.Msg.GetProjectId() == "" || req.Msg.GetVersionId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("project_id and version_id are required"))
	}
	if err := s.checkGameServerPermission(ctx, gameServerID, auth.PermissionGameServersUpdate); err != nil {
		return nil, err
	}

	dbGameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
	}

	env := parseEnvVars(dbGameServer.EnvVars)
	serverType := strings.ToUpper(env["TYPE"])
	projectType := req.Msg.GetProjectType()
	if projectType == gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_UNSPECIFIED {
		projectType = defaultProjectType(serverType)
		if projectType == gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_UNSPECIFIED {
			projectType = gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_MOD
		}
	}

	profile, err := buildInstallProfile(serverType, projectType)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	version, err := s.modClient.GetVersion(ctx, req.Msg.GetVersionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch version: %w", err))
	}
	if !strings.EqualFold(version.ProjectID, req.Msg.GetProjectId()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("version does not belong to project"))
	}
	if strings.EqualFold(version.ServerSide, "unsupported") {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("selected version is not server-compatible"))
	}

	file := selectDownloadFile(version.Files)
	if file == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("version has no downloadable files"))
	}

	dataPath, installDir, _, err := s.resolveMinecraftInstallDirectory(ctx, dbGameServer.ID, serverType, projectType)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	targetRelPath := filepath.Join(profile.InstallDir, file.Filename)
	targetAbsPath, err := resolveWithinVolume(dataPath, targetRelPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid target path: %w", err))
	}

	if err := downloadAndVerify(ctx, file.URL, targetAbsPath, file.Hashes); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to download file: %w", err))
	}
	if err := s.upsertMinecraftInstallMetadata(installDir, file.Filename, targetRelPath, projectType, req.Msg.GetProjectId(), req.Msg.GetProjectTitle(), req.Msg.GetProjectSlug(), req.Msg.GetProjectIconUrl(), version, file); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to record installation: %w", err))
	}

	logger.Info("[MinecraftCatalog] Installed %s (%s) to %s", file.Filename, profile.Description, targetAbsPath)

	resp := &gameserversv1.InstallMinecraftProjectFileResponse{
		Success:         true,
		Filename:        file.Filename,
		InstalledPath:   targetRelPath,
		RestartRequired: true,
		Message:         proto.String("File installed. Restart the server to apply changes."),
	}
	return connect.NewResponse(resp), nil
}

// UpdateMinecraftProjectFile installs the requested version and removes the replaced managed file.
func (s *Service) UpdateMinecraftProjectFile(ctx context.Context, req *connect.Request[gameserversv1.UpdateMinecraftProjectFileRequest]) (*connect.Response[gameserversv1.UpdateMinecraftProjectFileResponse], error) {
	gameServerID := strings.TrimSpace(req.Msg.GetGameServerId())
	if gameServerID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id is required"))
	}
	if req.Msg.GetProjectId() == "" || req.Msg.GetVersionId() == "" || strings.TrimSpace(req.Msg.GetCurrentFilename()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("project_id, version_id, and current_filename are required"))
	}
	if err := s.checkGameServerPermission(ctx, gameServerID, auth.PermissionGameServersUpdate); err != nil {
		return nil, err
	}

	dbGameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
	}

	env := parseEnvVars(dbGameServer.EnvVars)
	serverType := strings.ToUpper(env["TYPE"])
	projectType := req.Msg.GetProjectType()
	if projectType == gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_UNSPECIFIED {
		projectType = defaultProjectType(serverType)
		if projectType == gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_UNSPECIFIED {
			projectType = gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_MOD
		}
	}

	profile, err := buildInstallProfile(serverType, projectType)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	currentFilename := filepath.Base(strings.TrimSpace(req.Msg.GetCurrentFilename()))
	if currentFilename != strings.TrimSpace(req.Msg.GetCurrentFilename()) || !strings.HasSuffix(strings.ToLower(currentFilename), ".jar") {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid current_filename"))
	}

	version, err := s.modClient.GetVersion(ctx, req.Msg.GetVersionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch version: %w", err))
	}
	if !strings.EqualFold(version.ProjectID, req.Msg.GetProjectId()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("version does not belong to project"))
	}
	if strings.EqualFold(version.ServerSide, "unsupported") {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("selected version is not server-compatible"))
	}

	file := selectDownloadFile(version.Files)
	if file == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("version has no downloadable files"))
	}

	dataPath, installDir, _, err := s.resolveMinecraftInstallDirectory(ctx, dbGameServer.ID, serverType, projectType)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	metadata, err := loadMinecraftInstallMetadata(installDir)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("failed to read install metadata: %w", err))
	}
	currentMeta, ok := metadata.Files[currentFilename]
	if !ok || currentMeta.ProjectID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("only managed files can be updated"))
	}
	if !strings.EqualFold(currentMeta.ProjectID, req.Msg.GetProjectId()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("current file belongs to a different project"))
	}

	targetRelPath := filepath.Join(profile.InstallDir, file.Filename)
	targetAbsPath, err := resolveWithinVolume(dataPath, targetRelPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid target path: %w", err))
	}

	if err := downloadAndVerify(ctx, file.URL, targetAbsPath, file.Hashes); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to download file: %w", err))
	}
	if err := s.upsertMinecraftInstallMetadata(installDir, file.Filename, targetRelPath, projectType, req.Msg.GetProjectId(), req.Msg.GetProjectTitle(), req.Msg.GetProjectSlug(), req.Msg.GetProjectIconUrl(), version, file); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to record installation: %w", err))
	}
	if currentFilename != file.Filename {
		metadata, err = loadMinecraftInstallMetadata(installDir)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read updated install metadata: %w", err))
		}
		delete(metadata.Files, currentFilename)
		if err := saveMinecraftInstallMetadata(installDir, metadata); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update install metadata: %w", err))
		}
		currentRelPath := filepath.Join(profile.InstallDir, currentFilename)
		currentAbsPath, err := resolveWithinVolume(dataPath, currentRelPath)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid current path: %w", err))
		}
		if err := os.Remove(currentAbsPath); err != nil && !os.IsNotExist(err) {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to remove replaced file: %w", err))
		}
	}

	logger.Info("[MinecraftCatalog] Updated %s to %s (%s)", currentFilename, file.Filename, profile.Description)

	return connect.NewResponse(&gameserversv1.UpdateMinecraftProjectFileResponse{
		Success:          true,
		Filename:         file.Filename,
		InstalledPath:    targetRelPath,
		ReplacedFilename: proto.String(currentFilename),
		RestartRequired:  true,
		Message:          proto.String("File updated. Restart the server to apply changes."),
	}), nil
}

// --- helpers ---

func (s *Service) resolveMinecraftInstallDirectory(ctx context.Context, gameServerID, serverType string, projectType gameserversv1.MinecraftProjectType) (string, string, *installProfile, error) {
	profile, err := buildInstallProfile(serverType, projectType)
	if err != nil {
		return "", "", nil, err
	}

	dataPath, err := s.resolveGameServerVolume(ctx, gameServerID)
	if err != nil {
		logger.Warn("[MinecraftCatalog] Falling back to default data path for %s: %v", gameServerID, err)
	}
	if dataPath == "" {
		dataPath = filepath.Join(defaultDataVolumePrefix, fmt.Sprintf("gameserver-%s-data", gameServerID))
	}

	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		return "", "", nil, fmt.Errorf("failed to prepare data path: %w", err)
	}

	installDir, err := resolveWithinVolume(dataPath, profile.InstallDir)
	if err != nil {
		return "", "", nil, fmt.Errorf("invalid install path: %w", err)
	}
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return "", "", nil, fmt.Errorf("failed to prepare %s: %w", profile.Description, err)
	}
	return dataPath, installDir, profile, nil
}

func newMinecraftInstallMetadata() *minecraftInstallMetadata {
	return &minecraftInstallMetadata{
		Files: make(map[string]minecraftInstallMetadataEntry),
	}
}

func loadMinecraftInstallMetadata(installDir string) (*minecraftInstallMetadata, error) {
	path := filepath.Join(installDir, minecraftMetadataFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newMinecraftInstallMetadata(), nil
		}
		return nil, err
	}

	var metadata minecraftInstallMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}
	if metadata.Files == nil {
		metadata.Files = make(map[string]minecraftInstallMetadataEntry)
	}
	return &metadata, nil
}

func saveMinecraftInstallMetadata(installDir string, metadata *minecraftInstallMetadata) error {
	if metadata == nil {
		metadata = newMinecraftInstallMetadata()
	}
	if metadata.Files == nil {
		metadata.Files = make(map[string]minecraftInstallMetadataEntry)
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(installDir, minecraftMetadataFile)
	tmp, err := os.CreateTemp(installDir, ".metadata-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString("\n"); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o644)
}

func (s *Service) upsertMinecraftInstallMetadata(installDir, filename, installedPath string, projectType gameserversv1.MinecraftProjectType, projectID, title, slug, iconURL string, version *modrinth.Version, file *modrinth.VersionFile) error {
	metadata, err := loadMinecraftInstallMetadata(installDir)
	if err != nil {
		return err
	}
	if metadata.Files == nil {
		metadata.Files = make(map[string]minecraftInstallMetadataEntry)
	}

	size := int64(0)
	if file != nil {
		size = file.Size
	}
	entry := minecraftInstallMetadataEntry{
		ProjectID:     projectID,
		ProjectSlug:   slug,
		Title:         title,
		IconURL:       iconURL,
		ProjectType:   projectType,
		VersionID:     version.ID,
		VersionNumber: version.VersionNumber,
		GameVersions:  version.GameVersions,
		Loaders:       version.Loaders,
		Filename:      filename,
		InstalledPath: installedPath,
		SizeBytes:     size,
		InstalledAt:   time.Now().UTC(),
	}
	if file != nil && len(file.Hashes) > 0 {
		entry.Hashes = file.Hashes
	}

	if existing, ok := metadata.Files[filename]; ok && !existing.InstalledAt.IsZero() {
		entry.InstalledAt = existing.InstalledAt
	}
	metadata.Files[filename] = entry
	return saveMinecraftInstallMetadata(installDir, metadata)
}

func (s *Service) discoverMinecraftInstallMetadata(ctx context.Context, installDir, installRelDir, filename string, projectType gameserversv1.MinecraftProjectType) (minecraftInstallMetadataEntry, bool) {
	absPath := filepath.Join(installDir, filename)
	sha1sum, err := fileHash(absPath, sha1.New())
	if err != nil {
		logger.Warn("[MinecraftCatalog] Failed to hash installed file %s: %v", filename, err)
		return minecraftInstallMetadataEntry{}, false
	}

	version, err := s.modClient.GetVersionByFileHash(ctx, sha1sum, "sha1")
	if err != nil {
		if !errors.Is(err, modrinth.ErrNotFound) {
			logger.Warn("[MinecraftCatalog] Failed to identify installed file %s by hash: %v", filename, err)
		}
		return minecraftInstallMetadataEntry{}, false
	}
	if version == nil || version.ProjectID == "" || version.ID == "" {
		return minecraftInstallMetadataEntry{}, false
	}

	var project *modrinth.Project
	if fetched, err := s.modClient.GetProject(ctx, version.ProjectID); err == nil {
		project = fetched
		if modrinthTypeToProto(project.ProjectType) != projectType {
			return minecraftInstallMetadataEntry{}, false
		}
	} else {
		logger.Warn("[MinecraftCatalog] Failed to fetch project metadata for discovered file %s: %v", filename, err)
	}

	file := matchingVersionFile(version.Files, sha1sum, "sha1", filename)
	hashes := map[string]string{"sha1": sha1sum}
	size := int64(0)
	if file != nil {
		size = file.Size
		if len(file.Hashes) > 0 {
			hashes = file.Hashes
		}
	}
	info, err := os.Stat(absPath)
	if err == nil && size == 0 {
		size = info.Size()
	}

	title := filename
	slug := ""
	iconURL := ""
	if project != nil {
		title = firstNonEmpty(project.Title, filename)
		slug = project.Slug
		iconURL = project.IconURL
	}

	entry := minecraftInstallMetadataEntry{
		ProjectID:     version.ProjectID,
		ProjectSlug:   slug,
		Title:         title,
		IconURL:       iconURL,
		ProjectType:   projectType,
		VersionID:     version.ID,
		VersionNumber: version.VersionNumber,
		GameVersions:  version.GameVersions,
		Loaders:       version.Loaders,
		Filename:      filename,
		InstalledPath: filepath.Join(installRelDir, filename),
		SizeBytes:     size,
		Hashes:        hashes,
		InstalledAt:   time.Now().UTC(),
	}
	logger.Info("[MinecraftCatalog] Matched existing installed file %s to Modrinth project %s version %s", filename, entry.ProjectID, entry.VersionID)
	return entry, true
}

func installedFileToProto(installRelDir, filename string, info os.FileInfo, projectType gameserversv1.MinecraftProjectType, meta minecraftInstallMetadataEntry, managed bool) *gameserversv1.InstalledMinecraftProjectFile {
	relPath := filepath.Join(installRelDir, filename)
	modifiedAt := timestamppb.New(info.ModTime())
	file := &gameserversv1.InstalledMinecraftProjectFile{
		Id:            filename,
		Filename:      filename,
		InstalledPath: relPath,
		ProjectType:   projectType,
		SizeBytes:     info.Size(),
		ModifiedAt:    modifiedAt,
		Managed:       managed,
		Title:         proto.String(filename),
	}

	if managed {
		file.Id = meta.ProjectID
		file.ProjectId = proto.String(meta.ProjectID)
		file.ProjectSlug = proto.String(meta.ProjectSlug)
		file.Title = proto.String(firstNonEmpty(meta.Title, filename))
		file.IconUrl = proto.String(meta.IconURL)
		file.VersionId = proto.String(meta.VersionID)
		file.VersionNumber = proto.String(meta.VersionNumber)
		file.GameVersions = meta.GameVersions
		file.Loaders = meta.Loaders
		if !meta.InstalledAt.IsZero() {
			file.InstalledAt = timestamppb.New(meta.InstalledAt)
		}
	}
	return file
}

func matchingVersionFile(files []modrinth.VersionFile, expectedHash, algorithm, filename string) *modrinth.VersionFile {
	for i := range files {
		if strings.EqualFold(files[i].Hashes[algorithm], expectedHash) {
			return &files[i]
		}
	}
	for i := range files {
		if strings.EqualFold(files[i].Filename, filename) {
			return &files[i]
		}
	}
	return selectDownloadFile(files)
}

func (s *Service) latestCompatibleVersion(ctx context.Context, projectID, serverType string, projectType gameserversv1.MinecraftProjectType, serverVersion string) *modrinth.Version {
	filter := modrinth.VersionFilter{
		Limit: 25,
	}
	if serverVersion != "" {
		filter.GameVersions = []string{serverVersion}
	}
	if projectType != gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_PLUGIN {
		if loader := loaderFromServerType(serverType); loader != "" {
			filter.Loaders = []string{loader}
		}
	}

	versions, err := s.modClient.GetProjectVersions(ctx, projectID, filter)
	if err != nil {
		logger.Warn("[MinecraftCatalog] Failed to check updates for %s: %v", projectID, err)
		return nil
	}
	for _, version := range versions {
		if strings.EqualFold(version.ServerSide, "unsupported") {
			continue
		}
		if selectDownloadFile(version.Files) == nil {
			continue
		}
		return &version
	}
	return nil
}

func minecraftServerVersion(serverVersion *string, env map[string]string) string {
	if serverVersion != nil && *serverVersion != "" {
		return normalizeVersionString(*serverVersion)
	}
	if v := env["VERSION"]; v != "" {
		return normalizeVersionString(v)
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func mapProjectToProto(project modrinth.Project) *gameserversv1.MinecraftProject {
	proj := &gameserversv1.MinecraftProject{
		Id:              project.ID,
		Slug:            project.Slug,
		Title:           project.Title,
		Description:     project.Description,
		ProjectType:     modrinthTypeToProto(project.ProjectType),
		IconUrl:         project.IconURL,
		Categories:      project.Categories,
		Loaders:         project.Loaders,
		GameVersions:    project.GameVersions,
		Authors:         project.Authors,
		Downloads:       project.Downloads,
		Rating:          project.Rating,
		LatestVersionId: proto.String(project.LatestVersion),
		ProjectUrl:      proto.String(project.ProjectURL),
		SourceUrl:       proto.String(project.SourceURL),
		IssuesUrl:       proto.String(project.IssuesURL),
	}
	if project.Body != "" {
		proj.Body = proto.String(project.Body)
	}
	if len(project.Gallery) > 0 {
		proj.Gallery = project.Gallery
	}
	return proj
}

func mapVersionToProto(version modrinth.Version) *gameserversv1.MinecraftProjectVersion {
	files := make([]*gameserversv1.MinecraftProjectFile, 0, len(version.Files))
	for _, file := range version.Files {
		files = append(files, &gameserversv1.MinecraftProjectFile{
			Filename:  file.Filename,
			Url:       file.URL,
			SizeBytes: file.Size,
			Hashes:    file.Hashes,
			Primary:   file.Primary,
		})
	}

	var published *timestamppb.Timestamp
	if !version.DatePublished.IsZero() {
		published = timestamppb.New(version.DatePublished)
	}

	return &gameserversv1.MinecraftProjectVersion{
		Id:                  version.ID,
		Name:                version.Name,
		VersionNumber:       version.VersionNumber,
		GameVersions:        version.GameVersions,
		Loaders:             version.Loaders,
		ServerSideSupported: !strings.EqualFold(version.ServerSide, "unsupported"),
		ClientSideSupported: !strings.EqualFold(version.ClientSide, "unsupported"),
		PublishedAt:         published,
		Changelog:           proto.String(version.Changelog),
		Files:               files,
	}
}

func modrinthTypeToProto(value string) gameserversv1.MinecraftProjectType {
	switch strings.ToLower(value) {
	case "plugin":
		return gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_PLUGIN
	default:
		return gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_MOD
	}
}

func projectTypeToModrinth(value gameserversv1.MinecraftProjectType) string {
	switch value {
	case gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_PLUGIN:
		return "plugin"
	case gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_MOD:
		return "mod"
	default:
		return ""
	}
}

func parseEnvVars(raw string) map[string]string {
	result := make(map[string]string)
	if strings.TrimSpace(raw) == "" {
		return result
	}
	_ = json.Unmarshal([]byte(raw), &result)
	return result
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, val := range values {
		trimmed := strings.TrimSpace(val)
		if trimmed == "" {
			continue
		}
		set[strings.ToLower(trimmed)] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	return out
}

func normalizeVersionString(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	return version
}

func defaultProjectType(serverType string) gameserversv1.MinecraftProjectType {
	switch strings.ToUpper(serverType) {
	case "FORGE", "FABRIC", "QUILT", "NEOFORGE", "MAGMA", "CATSERVER":
		return gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_MOD
	case "PAPER", "PURPUR", "SPIGOT", "BUKKIT", "FOLIA", "VELOCITY", "WATERFALL":
		return gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_PLUGIN
	default:
		return gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_UNSPECIFIED
	}
}

func loaderFromServerType(serverType string) string {
	switch strings.ToUpper(serverType) {
	case "FORGE":
		return "forge"
	case "NEOFORGE":
		return "neoforge"
	case "FABRIC":
		return "fabric"
	case "QUILT":
		return "quilt"
	case "MAGMA":
		return "magma"
	case "CATSERVER":
		return "catserver"
	case "PAPER":
		return "paper"
	case "PURPUR":
		return "purpur"
	case "SPIGOT":
		return "spigot"
	case "BUKKIT":
		return "bukkit"
	case "FOLIA":
		return "folia"
	case "VELOCITY":
		return "velocity"
	case "WATERFALL":
		return "waterfall"
	default:
		return ""
	}
}

type installProfile struct {
	InstallDir  string
	Description string
}

func buildInstallProfile(serverType string, requested gameserversv1.MinecraftProjectType) (*installProfile, error) {
	serverType = strings.ToUpper(serverType)
	switch requested {
	case gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_MOD:
		if !isModCapableServer(serverType) {
			return nil, fmt.Errorf("server type %s does not support Forge/Fabric mods", serverType)
		}
		return &installProfile{
			InstallDir:  minecraftModsDir,
			Description: "mods directory",
		}, nil
	case gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_PLUGIN:
		if !isPluginCapableServer(serverType) {
			return nil, fmt.Errorf("server type %s does not support plugins", serverType)
		}
		return &installProfile{
			InstallDir:  minecraftPluginsDir,
			Description: "plugins directory",
		}, nil
	default:
		return nil, fmt.Errorf("unsupported project type")
	}
}

func isModCapableServer(serverType string) bool {
	switch strings.ToUpper(serverType) {
	case "FORGE", "NEOFORGE", "FABRIC", "QUILT", "MAGMA", "CATSERVER":
		return true
	default:
		return false
	}
}

func isPluginCapableServer(serverType string) bool {
	switch strings.ToUpper(serverType) {
	case "PAPER", "PURPUR", "SPIGOT", "BUKKIT", "FOLIA", "VELOCITY", "WATERFALL":
		return true
	default:
		return false
	}
}

func selectDownloadFile(files []modrinth.VersionFile) *modrinth.VersionFile {
	if len(files) == 0 {
		return nil
	}
	for _, file := range files {
		if file.Primary {
			return &file
		}
	}
	// fallback: prefer JAR files
	for _, file := range files {
		if strings.HasSuffix(strings.ToLower(file.Filename), ".jar") {
			return &file
		}
	}
	return &files[0]
}

func encodeCursor(offset int) string {
	if offset <= 0 {
		return ""
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", offset)))
	return encoded
}

func decodeCursor(cursor string) (int, error) {
	if strings.TrimSpace(cursor) == "" {
		return 0, nil
	}
	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, err
	}
	var offset int
	if _, err := fmt.Sscanf(string(data), "%d", &offset); err != nil {
		return 0, err
	}
	if offset < 0 {
		offset = 0
	}
	return offset, nil
}

func (s *Service) resolveGameServerVolume(ctx context.Context, gameServerID string) (string, error) {
	dcli, err := docker.New()
	if err != nil {
		return "", err
	}
	defer dcli.Close()

	containerID, err := s.findContainerForGameServer(ctx, gameServerID, dcli)
	if err != nil {
		return "", err
	}

	volumes, err := dcli.GetContainerVolumes(ctx, containerID)
	if err != nil {
		return "", err
	}

	for _, vol := range volumes {
		if vol.MountPoint == "/data" && vol.Source != "" {
			return vol.Source, nil
		}
	}
	return "", fmt.Errorf("data volume not found for game server %s", gameServerID)
}

func resolveWithinVolume(volumeRoot, relativePath string) (string, error) {
	trimmed := strings.TrimPrefix(relativePath, "/")
	target := filepath.Join(volumeRoot, trimmed)
	target, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}

	root, err := filepath.Abs(volumeRoot)
	if err != nil {
		return "", err
	}

	if target != root && !strings.HasPrefix(target, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes volume")
	}
	return target, nil
}

func downloadAndVerify(ctx context.Context, url string, dest string, hashes map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "obiente-cloud-gameservers-service")

	resp, err := downloadHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return fmt.Errorf("download failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(dest), ".download-*")
	if err != nil {
		return err
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	var writers []io.Writer
	writers = append(writers, tmpFile)

	var sha1Hash hash.Hash
	if expected, ok := hashes["sha1"]; ok && expected != "" {
		sha1Hash = sha1.New()
		writers = append(writers, sha1Hash)
	}

	var sha512Hash hash.Hash
	if expected, ok := hashes["sha512"]; ok && expected != "" {
		sha512Hash = sha512.New()
		writers = append(writers, sha512Hash)
	}

	if _, err := io.Copy(io.MultiWriter(writers...), resp.Body); err != nil {
		return err
	}

	if sha1Hash != nil {
		if err := verifyHash("sha1", sha1Hash, hashes["sha1"]); err != nil {
			return err
		}
	}
	if sha512Hash != nil {
		if err := verifyHash("sha512", sha512Hash, hashes["sha512"]); err != nil {
			return err
		}
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpFile.Name(), dest); err != nil {
		return err
	}

	return os.Chmod(dest, 0o644)
}

func fileHash(path string, hasher hash.Hash) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func verifyHash(name string, hasher hash.Hash, expected string) error {
	actual := fmt.Sprintf("%x", hasher.Sum(nil))
	expected = strings.ToLower(strings.TrimSpace(expected))
	if actual != expected {
		return fmt.Errorf("%s hash mismatch (expected %s got %s)", name, expected, actual)
	}
	return nil
}
