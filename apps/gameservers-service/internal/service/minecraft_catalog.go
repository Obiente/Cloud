package gameservers

import (
	"context"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gameservers-service/internal/catalog/modrinth"

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
	maxSearchLimit          = 50
	maxVersionLimit         = 200 // Increased to allow fetching more versions
)

var downloadHTTPClient = &http.Client{
	Timeout: 4 * time.Minute,
}

// ListMinecraftProjects integrates with Modrinth to surface mods/plugins.
func (s *Service) ListMinecraftProjects(ctx context.Context, req *connect.Request[gameserversv1.ListMinecraftProjectsRequest]) (*connect.Response[gameserversv1.ListMinecraftProjectsResponse], error) {
	gameServerID := strings.TrimSpace(req.Msg.GetGameServerId())
	if gameServerID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id is required"))
	}

	if err := s.checkGameServerPermission(ctx, gameServerID, "gameservers.view"); err != nil {
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

// GetMinecraftProjectVersions exposes version metadata for a project.
func (s *Service) GetMinecraftProjectVersions(ctx context.Context, req *connect.Request[gameserversv1.GetMinecraftProjectVersionsRequest]) (*connect.Response[gameserversv1.GetMinecraftProjectVersionsResponse], error) {
	gameServerID := strings.TrimSpace(req.Msg.GetGameServerId())
	if gameServerID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game_server_id is required"))
	}
	if err := s.checkGameServerPermission(ctx, gameServerID, "gameservers.view"); err != nil {
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
	
	// If limit is very high (>= 50) OR it's a plugin (plugins don't use loaders), don't auto-fill filters
	// This allows the frontend to request all versions without strict filtering
	skipAutoFill := limit >= 50 || projectType == gameserversv1.MinecraftProjectType_MINECRAFT_PROJECT_TYPE_PLUGIN

	loaders := dedupeStrings(req.Msg.GetLoaders())
	// Plugins don't use loaders, so don't filter by loader for plugins
	if len(loaders) == 0 && !skipAutoFill {
		if inferred := loaderFromServerType(serverType); inferred != "" {
			loaders = []string{inferred}
		}
	}

	gameVersions := dedupeStrings(req.Msg.GetGameVersions())
	// For plugins or when limit is high, don't auto-fill game version filter
	if len(gameVersions) == 0 && !skipAutoFill {
		if dbGameServer.ServerVersion != nil && *dbGameServer.ServerVersion != "" {
			gameVersions = []string{normalizeVersionString(*dbGameServer.ServerVersion)}
		} else if v := env["VERSION"]; v != "" {
			gameVersions = []string{normalizeVersionString(v)}
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

	if err := s.checkGameServerPermission(ctx, gameServerID, "gameservers.read"); err != nil {
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
	if err := s.checkGameServerPermission(ctx, gameServerID, "gameservers.update"); err != nil {
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

	dataPath, err := s.resolveGameServerVolume(ctx, dbGameServer.ID)
	if err != nil {
		logger.Warn("[MinecraftCatalog] Falling back to default data path for %s: %v", dbGameServer.ID, err)
	}
	if dataPath == "" {
		dataPath = filepath.Join(defaultDataVolumePrefix, fmt.Sprintf("gameserver-%s-data", dbGameServer.ID))
	}

	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to prepare data path: %w", err))
	}

	targetRelPath := filepath.Join(profile.InstallDir, file.Filename)
	targetAbsPath, err := resolveWithinVolume(dataPath, targetRelPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid target path: %w", err))
	}

	if err := os.MkdirAll(filepath.Dir(targetAbsPath), 0o755); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to prepare directory: %w", err))
	}

	if err := downloadAndVerify(ctx, file.URL, targetAbsPath, file.Hashes); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to download file: %w", err))
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

// --- helpers ---

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

func verifyHash(name string, hasher hash.Hash, expected string) error {
	actual := fmt.Sprintf("%x", hasher.Sum(nil))
	expected = strings.ToLower(strings.TrimSpace(expected))
	if actual != expected {
		return fmt.Errorf("%s hash mismatch (expected %s got %s)", name, expected, actual)
	}
	return nil
}
