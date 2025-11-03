package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"api/internal/database"
)

// RailpacksStrategy handles Railpacks deployments
// Railpacks is a variant of Nixpacks that works out of the box with minimal configuration.
// It supports: Clojure, Cobol, Crystal, C#/.NET, Dart, Deno, Elixir, F#, Gleam, Go,
// Haskell, Java, Lunatic, Node, PHP, Python, Ruby, Rust, Scheme, Staticfile,
// Swift, Scala, Zig
// The detection is Rails-optimized, but the build can handle any supported language.
type RailpacksStrategy struct{}

func NewRailpacksStrategy() *RailpacksStrategy {
	return &RailpacksStrategy{}
}

func (s *RailpacksStrategy) Name() string {
	return "Railpacks"
}

func (s *RailpacksStrategy) Detect(ctx context.Context, repoPath string) (bool, error) {
	// Check for Rails-specific files (Railpacks is optimized for Rails detection)
	// Note: While Railpacks can build any language, this detection prioritizes Rails apps
	hasGemfile := fileExists(filepath.Join(repoPath, "Gemfile"))
	hasRailsApp := fileExists(filepath.Join(repoPath, "config", "application.rb")) ||
		fileExists(filepath.Join(repoPath, "config.ru"))

	return hasGemfile && hasRailsApp, nil
}

func (s *RailpacksStrategy) Build(ctx context.Context, deployment *database.Deployment, config *BuildConfig) (*BuildResult, error) {
	log.Printf("[Railpacks] Building deployment %s", deployment.ID)

	buildDir, err := ensureBuildDir(deployment.ID)
	if err != nil {
		return &BuildResult{Success: false, Error: err}, err
	}

	// Clone repository
	if err := cloneRepository(ctx, config.RepositoryURL, config.Branch, buildDir, config.GitHubToken); err != nil {
		return &BuildResult{Success: false, Error: err}, err
	}

	imageName := fmt.Sprintf("obiente/%s:%s", deployment.ID, deployment.Branch)

	// Create railpack.toml or nixpacks.toml configuration if needed
	// Railpack uses railpack.toml (or railpack.json), but can also read nixpacks.toml
	// For now, create nixpacks.toml for compatibility
	if err := createNixpacksConfig(buildDir, config.StartCommand, false); err != nil {
		log.Printf("[Railpacks] Warning: Failed to create nixpacks.toml: %v", err)
	}

	// Use Railpack (Railway's build tool, separate from nixpacks)
	// Railpack is a different CLI tool that uses Railway's optimized build process
	// It uses ghcr.io/railwayapp/nixpacks base images but with different build logic
	// Use full path to railpack since PATH might not include /usr/local/bin in all contexts
	railpackPath := "/usr/local/bin/railpack"
	// First try to find railpack in PATH (preferred)
	if path, err := exec.LookPath("railpack"); err == nil {
		railpackPath = path
	} else if _, err := os.Stat(railpackPath); err != nil {
		// If PATH lookup fails and full path doesn't exist, return error
		return &BuildResult{Success: false, Error: fmt.Errorf("railpack executable not found in PATH or %s. Please ensure railpack is installed in the Docker image", railpackPath)}, nil
	}
	cmd := exec.CommandContext(ctx, railpackPath, "build", buildDir, "--name", imageName)
	envVars := getEnvAsStringSlice(config.EnvVars)
	
	// Railpack requires BUILDKIT_HOST to be set for BuildKit builds
	// Check if buildx is available; if not, disable BuildKit as fallback
	if !isBuildxAvailable(ctx) {
		log.Printf("[Railpacks] Buildx not available, disabling BuildKit")
		envVars = append(envVars, "DOCKER_BUILDKIT=0")
	} else {
		// Enable BuildKit and ensure BuildKit daemon container is running
		// Railpack uses BuildKit for building, so we need to provide a BuildKit daemon
		// Start buildkit container if it doesn't exist or isn't running
		startBuildkitContainer(ctx)
		envVars = append(envVars, "DOCKER_BUILDKIT=1")
		envVars = append(envVars, "BUILDKIT_HOST=docker-container://buildkit")
	}
	cmd.Env = envVars

	// Use log writers if provided, otherwise fallback to os.Stdout/Stderr
	if config.LogWriter != nil {
		cmd.Stdout = NewMultiWriter(os.Stdout, config.LogWriter)
	} else {
		cmd.Stdout = os.Stdout
	}
	if config.LogWriterErr != nil {
		cmd.Stderr = NewMultiWriter(os.Stderr, config.LogWriterErr)
	} else {
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return &BuildResult{Success: false, Error: fmt.Errorf("railpack build failed: %w", err)}, nil
	}

	// Get image size
	var imageSize int64
	if size, err := getImageSize(ctx, imageName); err != nil {
		log.Printf("[Railpacks] Warning: Failed to get image size: %v", err)
	} else {
		imageSize = size
	}

	// Auto-detect port based on detected language
	port := config.Port
	if port == 0 {
		// Try to detect from repository, default to 3000 (common for Rails/Node)
		port = detectPortFromRepo(buildDir, 3000)
	}

	return &BuildResult{
		ImageName:      imageName,
		ImageSizeBytes: imageSize,
		Port:           port,
		Success:        true,
	}, nil
}

// detectPortFromRepo attempts to detect the port from repository files
// Supports ports for: Node, Python, Go, PHP, Java, Ruby, Deno, Rust, Elixir
func detectPortFromRepo(repoPath string, defaultPort int) int {
	// Check for common port indicators in various frameworks
	if fileExists(filepath.Join(repoPath, "deno.json")) || fileExists(filepath.Join(repoPath, "deno.jsonc")) {
		return 8080 // Default Deno port
	}
	if fileExists(filepath.Join(repoPath, "package.json")) {
		return 8080 // Default Node.js port
	}
	if fileExists(filepath.Join(repoPath, "requirements.txt")) ||
		fileExists(filepath.Join(repoPath, "Pipfile")) ||
		fileExists(filepath.Join(repoPath, "pyproject.toml")) {
		return 8000 // Default Python port
	}
	if fileExists(filepath.Join(repoPath, "go.mod")) {
		return 8080 // Default Go port
	}
	if fileExists(filepath.Join(repoPath, "Gemfile")) {
		return 3000 // Default Rails port
	}
	if fileExists(filepath.Join(repoPath, "composer.json")) {
		return 8080 // Default PHP port
	}
	if fileExists(filepath.Join(repoPath, "pom.xml")) || fileExists(filepath.Join(repoPath, "build.gradle")) {
		return 8080 // Default Java port
	}
	if fileExists(filepath.Join(repoPath, "Cargo.toml")) {
		return 8080 // Default Rust port
	}
	if fileExists(filepath.Join(repoPath, "mix.exs")) {
		return 4000 // Default Phoenix/Elixir port
	}
	return defaultPort
}

// NixpacksStrategy handles generic Nixpacks buildpack deployments
// Nixpacks supports: Clojure, Cobol, Crystal, C#/.NET, Dart, Deno, Elixir, F#,
// Gleam, Go, Haskell, Java, Lunatic, Node, PHP, Python, Ruby, Rust, Scheme,
// Staticfile, Swift, Scala, Zig
// Works out of the box with minimal configuration.
type NixpacksStrategy struct{}

func NewNixpacksStrategy() *NixpacksStrategy {
	return &NixpacksStrategy{}
}

func (s *NixpacksStrategy) Name() string {
	return "Nixpacks"
}

func (s *NixpacksStrategy) Detect(ctx context.Context, repoPath string) (bool, error) {
	// Nixpacks supports: Clojure, Cobol, Crystal, C#/.NET, Dart, Deno, Elixir, F#,
	// Gleam, Go, Haskell, Java, Lunatic, Node, PHP, Python, Ruby, Rust, Scheme,
	// Staticfile, Swift, Scala, Zig
	// Check for common indicators that Nixpacks can handle

	// Check exact filenames first
	exactIndicators := []string{
		// Node.js/Deno
		"package.json",
		"deno.json",
		"deno.jsonc",
		// Python
		"requirements.txt",
		"setup.py",
		"Pipfile",
		"pyproject.toml",
		// Rust
		"Cargo.toml",
		// Go
		"go.mod",
		// Java/Scala
		"pom.xml",
		"build.gradle",
		"build.gradle.kts",
		"build.sbt",
		// PHP
		"composer.json",
		// Ruby
		"Gemfile",
		// Elixir
		"mix.exs",
		// Dart
		"pubspec.yaml",
		"pubspec.yml",
		// Crystal
		"shard.yml",
		// Gleam
		"gleam.toml",
		// Zig
		"build.zig",
		"build.zig.zon",
		// Swift
		"Package.swift",
		// Haskell
		"stack.yaml",
		"package.yaml",
		// Clojure
		"project.clj",
		"build.boot",
		"deps.edn",
		// Lunatic
		"lunatic.toml",
		// Static
		"Staticfile",
		"static.json",
		"index.html",
		// .NET (exact names)
		"project.json",
	}

	for _, indicator := range exactIndicators {
		if fileExists(filepath.Join(repoPath, indicator)) {
			return true, nil
		}
	}

	// Check for glob patterns (C#/.NET, Haskell .cabal files)
	if hasAnyFile(repoPath, "*.csproj", "*.fsproj", "*.sln", "*.vbproj", "*.cabal") {
		return true, nil
	}

	// Also check for Dockerfile (should use Dockerfile strategy instead)
	if fileExists(filepath.Join(repoPath, "Dockerfile")) {
		return false, nil
	}

	// Check for docker-compose.yml (should use Plain Compose strategy instead)
	if hasFile(repoPath, "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml") {
		return false, nil
	}

	// Default fallback: can try Nixpacks if nothing else detected
	return true, nil
}

func (s *NixpacksStrategy) Build(ctx context.Context, deployment *database.Deployment, config *BuildConfig) (*BuildResult, error) {
	log.Printf("[Nixpacks] Building deployment %s", deployment.ID)

	buildDir, err := ensureBuildDir(deployment.ID)
	if err != nil {
		return &BuildResult{Success: false, Error: err}, err
	}

	// Clone repository
	if err := cloneRepository(ctx, config.RepositoryURL, config.Branch, buildDir, config.GitHubToken); err != nil {
		return &BuildResult{Success: false, Error: err}, err
	}

	imageName := fmt.Sprintf("obiente/%s:%s", deployment.ID, deployment.Branch)

	// Create nixpacks.toml with start command if provided
	// Use standard nixpacks provider (not Railway's)
	if err := createNixpacksConfig(buildDir, config.StartCommand, false); err != nil {
		log.Printf("[Nixpacks] Warning: Failed to create nixpacks.toml: %v", err)
	}

	// Use Nixpacks to build application
	cmd := exec.CommandContext(ctx, "nixpacks", "build", buildDir, "--name", imageName)
	envVars := getEnvAsStringSlice(config.EnvVars)
	// Check if buildx is available; if not, disable BuildKit as fallback
	if !isBuildxAvailable(ctx) {
		log.Printf("[Nixpacks] Buildx not available, disabling BuildKit")
		envVars = append(envVars, "DOCKER_BUILDKIT=0")
	} else {
		// Enable BuildKit for faster, more efficient builds
		envVars = append(envVars, "DOCKER_BUILDKIT=1")
	}
	cmd.Env = envVars

	// Use log writers if provided, otherwise fallback to os.Stdout/Stderr
	if config.LogWriter != nil {
		cmd.Stdout = NewMultiWriter(os.Stdout, config.LogWriter)
	} else {
		cmd.Stdout = os.Stdout
	}
	if config.LogWriterErr != nil {
		cmd.Stderr = NewMultiWriter(os.Stderr, config.LogWriterErr)
	} else {
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return &BuildResult{Success: false, Error: fmt.Errorf("nixpacks build failed: %w", err)}, nil
	}

	// Get image size
	var imageSize int64
	if size, err := getImageSize(ctx, imageName); err != nil {
		log.Printf("[Nixpacks] Warning: Failed to get image size: %v", err)
	} else {
		imageSize = size
	}

	// Auto-detect port based on framework
	port := s.detectPort(buildDir, config.Port)

	return &BuildResult{
		ImageName:      imageName,
		ImageSizeBytes: imageSize,
		Port:           port,
		Success:        true,
	}, nil
}

// isBuildxAvailable checks if Docker buildx is available on the host system
// Since we mount the Docker socket, we check the host's Docker installation
func isBuildxAvailable(ctx context.Context) bool {
	// Try to check if buildx is available via docker buildx version
	cmd := exec.CommandContext(ctx, "docker", "buildx", "version")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// checkBuildkitContainer checks if a BuildKit container named "buildkit" is running
func checkBuildkitContainer(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "ps", "--filter", "name=buildkit", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "buildkit"
}

// startBuildkitContainer starts a BuildKit daemon container if it doesn't exist
func startBuildkitContainer(ctx context.Context) {
	// Check if buildkit container already exists (running or stopped)
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name=buildkit", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("[Railpacks] Failed to check for buildkit container: %v", err)
		return
	}
	
	if strings.TrimSpace(string(output)) != "" {
		// Container exists, try to start it if it's stopped
		startCmd := exec.CommandContext(ctx, "docker", "start", "buildkit")
		if err := startCmd.Run(); err != nil {
			log.Printf("[Railpacks] Failed to start existing buildkit container: %v", err)
		}
		return
	}
	
	// Container doesn't exist, create and start it
	log.Printf("[Railpacks] Starting BuildKit daemon container...")
	createCmd := exec.CommandContext(ctx, "docker", "run", "--rm", "--privileged", "-d", "--name", "buildkit", "moby/buildkit:latest")
	if err := createCmd.Run(); err != nil {
		log.Printf("[Railpacks] Failed to start BuildKit container: %v. Railpack may fail.", err)
	}
}

// createNixpacksConfig creates a nixpacks.toml file with the start command and Node.js version if provided
// This is used by both NixpacksStrategy and RailpacksStrategy
// If startCommand is empty, it attempts to detect a default from the repository
// If Node.js version is not specified, it attempts to detect from package.json engines field
// useRailwayProvider: if true, configures for Railway's Railpack provider (uses Railway base images)
func createNixpacksConfig(buildDir, startCommand string, useRailwayProvider bool) error {
	nixpacksConfigPath := filepath.Join(buildDir, "nixpacks.toml")
	
	// Detect start command if not provided
	if startCommand == "" {
		startCommand = detectDefaultStartCommand(buildDir)
	}
	
	// Detect Node.js version requirement from package.json
	nodeVersion := detectNodeVersion(buildDir)
	
	// Build nixpacks.toml content
	var configParts []string
	
	// Add provider configuration for Railway Railpack if requested
	if useRailwayProvider {
		// Railway's Railpack uses railway provider which selects Railway's optimized base images
		// This ensures we use ghcr.io/railwayapp/nixpacks base images instead of standard ones
		configParts = append(configParts, "[provider]\nname = \"railway\"\n")
	}
	
	// Add Node.js version if detected
	if nodeVersion != "" {
		configParts = append(configParts, fmt.Sprintf("[variables]\nNODE_VERSION = %q\n", nodeVersion))
	}
	
	// Add start command if provided
	if startCommand != "" {
		configParts = append(configParts, fmt.Sprintf("[start]\ncmd = %q\n", startCommand))
	}
	
	// If no configuration needed, don't create file (let nixpacks auto-detect)
	if len(configParts) == 0 {
		return nil
	}
	
	configContent := strings.Join(configParts, "\n")
	
	if err := os.WriteFile(nixpacksConfigPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write nixpacks.toml: %w", err)
	}
	
	log.Printf("[Nixpacks] Created nixpacks.toml with Node.js version: %s, start command: %s", nodeVersion, startCommand)
	return nil
}

// detectNodeVersion attempts to detect the required Node.js version from package.json
// Returns the version string (e.g., "18.20.8" or "20") or empty string if not found
func detectNodeVersion(buildDir string) string {
	packageJsonPath := filepath.Join(buildDir, "package.json")
	if !fileExists(packageJsonPath) {
		return ""
	}
	
	content, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return ""
	}
	
	// Parse package.json to extract engines.node
	var pkg struct {
		Engines struct {
			Node string `json:"node"`
		} `json:"engines"`
	}
	
	if err := json.Unmarshal(content, &pkg); err != nil {
		// If JSON parsing fails, try simple string matching as fallback
		contentStr := string(content)
		if strings.Contains(contentStr, `"engines"`) && strings.Contains(contentStr, `"node"`) {
			// Try to extract version using regex as fallback
			// Look for patterns like "node": ">=18.20.8" or "node": "18.x"
			re := regexp.MustCompile(`"node"\s*:\s*"([^"]+)"`)
			matches := re.FindStringSubmatch(contentStr)
			if len(matches) > 1 {
				return normalizeNodeVersion(matches[1])
			}
		}
		return ""
	}
	
	if pkg.Engines.Node != "" {
		return normalizeNodeVersion(pkg.Engines.Node)
	}
	
	return ""
}

// normalizeNodeVersion normalizes Node.js version strings to a format nixpacks accepts
// Handles patterns like ">=18.20.8", "18.x", "20", "^18.20.8", "~18.20.8"
// Returns the minimum version that satisfies the constraint
func normalizeNodeVersion(version string) string {
	// Remove common version prefixes
	version = strings.TrimSpace(version)
	
	// Handle ">=" constraints - take the minimum version
	if strings.HasPrefix(version, ">=") {
		version = strings.TrimPrefix(version, ">=")
		return version // Return full version like "18.20.8"
	}
	
	// Remove other prefixes that don't affect minimum version
	version = strings.TrimPrefix(version, "^")
	version = strings.TrimPrefix(version, "~")
	version = strings.TrimPrefix(version, "=")
	
	// Handle "x" versions like "18.x" or "20.x" -> convert to major version
	if strings.Contains(version, ".x") {
		parts := strings.Split(version, ".")
		if len(parts) > 0 {
			return parts[0]
		}
	}
	
	// If it's a full version string like "18.20.8", return as-is
	// If it's just major.minor like "18.20", that should work too
	// For major only like "20", return as-is
	
	return version
}

// detectDefaultStartCommand attempts to detect a default start command from repository files
func detectDefaultStartCommand(buildDir string) string {
	packageJsonPath := filepath.Join(buildDir, "package.json")
	if fileExists(packageJsonPath) {
		// Try to read package.json and extract start script
		content, err := os.ReadFile(packageJsonPath)
		if err == nil {
			contentStr := string(content)
			// Simple detection: look for "start" script
			if strings.Contains(contentStr, `"start"`) {
				// Fallback: use npm/pnpm/yarn start
				// Check for lockfiles to determine package manager
				if fileExists(filepath.Join(buildDir, "pnpm-lock.yaml")) {
					return "pnpm start"
				}
				if fileExists(filepath.Join(buildDir, "yarn.lock")) {
					return "yarn start"
				}
				return "npm start"
			}
		}
	}
	
	// Check for other common files
	if fileExists(filepath.Join(buildDir, "main.py")) {
		return "python main.py"
	}
	if fileExists(filepath.Join(buildDir, "app.py")) {
		return "python app.py"
	}
	if fileExists(filepath.Join(buildDir, "server.js")) {
		return "node server.js"
	}
	if fileExists(filepath.Join(buildDir, "index.js")) {
		return "node index.js"
	}
	
	return ""
}

func (s *NixpacksStrategy) detectPort(repoPath string, defaultPort int) int {
	if defaultPort != 0 {
		return defaultPort
	}

	// Check for common port indicators
	if fileExists(filepath.Join(repoPath, "package.json")) {
		// Check package.json for start script port
		content, err := readFile(filepath.Join(repoPath, "package.json"))
		if err == nil && strings.Contains(content, "PORT") {
			// Could parse PORT from package.json scripts, but defaulting to 8080
			return 8080
		}
		return 8080 // Default Node.js port
	}

	if fileExists(filepath.Join(repoPath, "requirements.txt")) {
		return 8000 // Default Python port
	}

	if fileExists(filepath.Join(repoPath, "go.mod")) {
		return 8080 // Default Go port
	}

	return 8080 // Default fallback
}

// DockerfileStrategy handles Dockerfile-based builds
type DockerfileStrategy struct{}

func NewDockerfileStrategy() *DockerfileStrategy {
	return &DockerfileStrategy{}
}

func (s *DockerfileStrategy) Name() string {
	return "Dockerfile"
}

func (s *DockerfileStrategy) Detect(ctx context.Context, repoPath string) (bool, error) {
	return fileExists(filepath.Join(repoPath, "Dockerfile")), nil
}

func (s *DockerfileStrategy) Build(ctx context.Context, deployment *database.Deployment, config *BuildConfig) (*BuildResult, error) {
	log.Printf("[Dockerfile] Building deployment %s", deployment.ID)

	buildDir, err := ensureBuildDir(deployment.ID)
	if err != nil {
		return &BuildResult{Success: false, Error: err}, err
	}

	// Clone repository
	if err := cloneRepository(ctx, config.RepositoryURL, config.Branch, buildDir, config.GitHubToken); err != nil {
		return &BuildResult{Success: false, Error: err}, err
	}

	imageName := fmt.Sprintf("obiente/%s:%s", deployment.ID, deployment.Branch)

	// Use configured Dockerfile path or default to "Dockerfile"
	dockerfile := config.DockerfilePath
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}

	// Ensure dockerfile path is relative to buildDir
	dockerfilePath := filepath.Join(buildDir, dockerfile)
	if !fileExists(dockerfilePath) {
		return &BuildResult{Success: false, Error: fmt.Errorf("Dockerfile not found at path: %s", dockerfile)}, nil
	}

	if err := buildDockerImage(ctx, buildDir, imageName, dockerfile, config.LogWriter, config.LogWriterErr); err != nil {
		return &BuildResult{Success: false, Error: fmt.Errorf("docker build failed: %w", err)}, nil
	}

	// Get image size
	var imageSize int64
	if size, err := getImageSize(ctx, imageName); err != nil {
		log.Printf("[Dockerfile] Warning: Failed to get image size: %v", err)
	} else {
		imageSize = size
	}

	// Try to detect port from Dockerfile EXPOSE directive
	port := s.detectPortFromDockerfile(dockerfilePath, config.Port)

	return &BuildResult{
		ImageName:      imageName,
		ImageSizeBytes: imageSize,
		Port:           port,
		Success:        true,
	}, nil
}

func (s *DockerfileStrategy) detectPortFromDockerfile(dockerfilePath string, defaultPort int) int {
	if defaultPort != 0 {
		return defaultPort
	}

	content, err := readFile(dockerfilePath)
	if err != nil {
		return 8080 // Default fallback
	}

	// Look for EXPOSE directive
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(line), "EXPOSE ") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				var port int
				if _, err := fmt.Sscanf(parts[1], "%d", &port); err == nil {
					return port
				}
			}
		}
	}

	return 8080 // Default fallback
}

// PlainComposeStrategy handles plain Docker Compose deployments
type PlainComposeStrategy struct{}

func NewPlainComposeStrategy() *PlainComposeStrategy {
	return &PlainComposeStrategy{}
}

func (s *PlainComposeStrategy) Name() string {
	return "Plain Compose"
}

func (s *PlainComposeStrategy) Detect(ctx context.Context, repoPath string) (bool, error) {
	// Plain Compose doesn't use repositories, so it should never auto-detect from a repo
	// This should be manually selected by the user
	return false, nil
}

func (s *PlainComposeStrategy) Build(ctx context.Context, deployment *database.Deployment, config *BuildConfig) (*BuildResult, error) {
	log.Printf("[PlainCompose] Building deployment %s", deployment.ID)

	// Plain Compose uses compose YAML directly from database, not from a repository
	if deployment.ComposeYaml == "" {
		return &BuildResult{Success: false, Error: fmt.Errorf("no compose YAML configured. Please set compose configuration via UpdateDeploymentCompose")}, nil
	}

	// Validate compose file
	if err := ValidateCompose(ctx, deployment.ComposeYaml); len(err) > 0 {
		for _, ve := range err {
			if ve.Severity == "error" {
				return &BuildResult{Success: false, Error: fmt.Errorf("compose validation error: %s", ve.Message)}, nil
			}
		}
	}

	// Build any images defined in compose file if needed
	// docker compose build is handled by docker compose up

	return &BuildResult{
		ComposeYaml: deployment.ComposeYaml,
		Success:     true,
	}, nil
}

// ComposeRepoStrategy handles Docker Compose deployments from a repository
type ComposeRepoStrategy struct{}

func NewComposeRepoStrategy() *ComposeRepoStrategy {
	return &ComposeRepoStrategy{}
}

func (s *ComposeRepoStrategy) Name() string {
	return "Compose from Repository"
}

func (s *ComposeRepoStrategy) Detect(ctx context.Context, repoPath string) (bool, error) {
	return hasFile(repoPath, "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"), nil
}

func (s *ComposeRepoStrategy) Build(ctx context.Context, deployment *database.Deployment, config *BuildConfig) (*BuildResult, error) {
	log.Printf("[ComposeRepo] Building deployment %s from repository", deployment.ID)

	if config.RepositoryURL == "" {
		return &BuildResult{Success: false, Error: fmt.Errorf("repository URL is required for COMPOSE_REPO strategy")}, nil
	}

	buildDir, err := ensureBuildDir(deployment.ID)
	if err != nil {
		return &BuildResult{Success: false, Error: err}, err
	}

	// Clone repository
	branch := config.Branch
	if branch == "" {
		branch = "main"
	}
	if err := cloneRepository(ctx, config.RepositoryURL, branch, buildDir, config.GitHubToken); err != nil {
		return &BuildResult{Success: false, Error: err}, err
	}

	// Use configured compose file path or auto-detect
	composeFile := config.ComposeFilePath
	if composeFile == "" {
		// Auto-detect compose file
		composeFiles := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
		for _, filename := range composeFiles {
			if fileExists(filepath.Join(buildDir, filename)) {
				composeFile = filename
				break
			}
		}
	}

	if composeFile == "" {
		return &BuildResult{Success: false, Error: fmt.Errorf("no compose file found in repository")}, nil
	}

	// Ensure compose file path is relative to buildDir
	composeFilePath := filepath.Join(buildDir, composeFile)
	if !fileExists(composeFilePath) {
		return &BuildResult{Success: false, Error: fmt.Errorf("compose file not found at path: %s", composeFile)}, nil
	}

	// Read compose file content
	composeYaml, err := readFile(composeFilePath)
	if err != nil {
		return &BuildResult{Success: false, Error: fmt.Errorf("failed to read compose file: %w", err)}, nil
	}

	// Validate compose file
	if err := ValidateCompose(ctx, composeYaml); len(err) > 0 {
		for _, ve := range err {
			if ve.Severity == "error" {
				return &BuildResult{Success: false, Error: fmt.Errorf("compose validation error: %s", ve.Message)}, nil
			}
		}
	}

	// Build any images defined in compose file if needed
	// docker compose build is handled by docker compose up

	return &BuildResult{
		ComposeYaml: composeYaml,
		Success:     true,
	}, nil
}

// StaticStrategy handles static site deployments
type StaticStrategy struct{}

func NewStaticStrategy() *StaticStrategy {
	return &StaticStrategy{}
}

func (s *StaticStrategy) Name() string {
	return "Static"
}

func (s *StaticStrategy) Detect(ctx context.Context, repoPath string) (bool, error) {
	// Check for static site indicators
	hasStaticFiles := hasFile(repoPath, "index.html", "index.htm") ||
		hasFile(repoPath, "public", "index.html") ||
		hasFile(repoPath, "dist", "index.html") ||
		hasFile(repoPath, "build", "index.html")

	// Also check for common static site frameworks
	hasStaticFramework := fileExists(filepath.Join(repoPath, "_config.yml")) || // Jekyll
		fileExists(filepath.Join(repoPath, "hugo.toml")) || // Hugo
		fileExists(filepath.Join(repoPath, "gatsby-config.js")) || // Gatsby
		fileExists(filepath.Join(repoPath, "next.config.js")) // Next.js static export

	return hasStaticFiles || hasStaticFramework, nil
}

func (s *StaticStrategy) Build(ctx context.Context, deployment *database.Deployment, config *BuildConfig) (*BuildResult, error) {
	log.Printf("[Static] Building deployment %s", deployment.ID)

	buildDir, err := ensureBuildDir(deployment.ID)
	if err != nil {
		return &BuildResult{Success: false, Error: err}, err
	}

	// Clone repository
	if err := cloneRepository(ctx, config.RepositoryURL, config.Branch, buildDir, config.GitHubToken); err != nil {
		return &BuildResult{Success: false, Error: err}, nil
	}

	// Determine output directory and build command
	outputDir := s.findOutputDir(buildDir)

	// Run build command if provided
	if config.BuildCommand != "" {
		parts := strings.Fields(config.BuildCommand)
		if len(parts) > 0 {
			cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
			cmd.Dir = buildDir
			cmd.Env = getEnvAsStringSlice(config.EnvVars)

			// Use log writers if provided, otherwise fallback to os.Stdout/Stderr
			if config.LogWriter != nil {
				cmd.Stdout = NewMultiWriter(os.Stdout, config.LogWriter)
			} else {
				cmd.Stdout = os.Stdout
			}
			if config.LogWriterErr != nil {
				cmd.Stderr = NewMultiWriter(os.Stderr, config.LogWriterErr)
			} else {
				cmd.Stderr = os.Stderr
			}

			if err := cmd.Run(); err != nil {
				return &BuildResult{Success: false, Error: fmt.Errorf("build command failed: %w", err)}, nil
			}
		}
	}

	// Create Dockerfile for static site
	dockerfileContent := s.generateStaticDockerfile(outputDir)
	dockerfilePath := filepath.Join(buildDir, ".obiente.Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return &BuildResult{Success: false, Error: fmt.Errorf("failed to write Dockerfile: %w", err)}, nil
	}

	imageName := fmt.Sprintf("obiente/%s:%s", deployment.ID, deployment.Branch)

	// Build Docker image
	if err := buildDockerImage(ctx, buildDir, imageName, ".obiente.Dockerfile", config.LogWriter, config.LogWriterErr); err != nil {
		return &BuildResult{Success: false, Error: fmt.Errorf("docker build failed: %w", err)}, nil
	}

	return &BuildResult{
		ImageName: imageName,
		Port:      80, // Static sites typically use port 80
		Success:   true,
	}, nil
}

func (s *StaticStrategy) findOutputDir(repoPath string) string {
	// Common output directories
	outputDirs := []string{
		"dist",
		"build",
		"public",
		"out",
		"_site",  // Jekyll
		"public", // Hugo
		"out",    // Next.js static export
	}

	for _, dir := range outputDirs {
		if fileExists(filepath.Join(repoPath, dir, "index.html")) {
			return dir
		}
	}

	// Check if root has index.html
	if fileExists(filepath.Join(repoPath, "index.html")) {
		return "."
	}

	return "public" // Default fallback
}

func (s *StaticStrategy) generateStaticDockerfile(outputDir string) string {
	return fmt.Sprintf(`FROM nginx:alpine
COPY %s /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]`, outputDir)
}
