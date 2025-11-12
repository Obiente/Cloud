package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"api/internal/database"
	"api/internal/orchestrator"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
)

// BuildStrategy defines the interface for different deployment build strategies
type BuildStrategy interface {
	// Name returns the strategy name
	Name() string

	// Build builds the deployment and returns container configuration
	// Returns the final image name and any compose YAML if applicable
	Build(ctx context.Context, deployment *database.Deployment, config *BuildConfig) (*BuildResult, error)

	// Detect returns true if this strategy can be auto-detected for the given repository
	Detect(ctx context.Context, repoPath string) (bool, error)
}

// BuildConfig contains configuration for building a deployment
type BuildConfig struct {
	DeploymentID    string
	RepositoryURL   string
	Branch          string
	GitHubToken     string // GitHub token for authenticating with private repositories
	BuildCommand    string
	InstallCommand  string
	StartCommand    string // Start command for running the application
	DockerfilePath  string // Path to Dockerfile (relative to repo root, defaults to "Dockerfile")
	ComposeFilePath string // Path to compose file (relative to repo root, auto-detected if empty)
	BuildPath       string // Working directory for build (relative to repo root, defaults to ".")
	BuildOutputPath string // Path to built output files (relative to repo root, auto-detected if empty)
	UseNginx        bool   // Use nginx for static deployments
	NginxConfig     string // Custom nginx configuration (optional, uses default if empty)
	EnvVars         map[string]string
	Port            int
	MemoryBytes     int64
	CPUShares       int64
	LogWriter       io.Writer // Optional writer for build logs (stdout)
	LogWriterErr    io.Writer // Optional writer for build logs (stderr)
}

// BuildResult contains the result of a build operation
type BuildResult struct {
	ImageName      string // Final image name (for single container deployments)
	ImageSizeBytes int64  // Size of the built image in bytes
	ComposeYaml    string // Docker Compose YAML (for compose-based deployments)
	Port           int    // Exposed port
	Success        bool
	Error          error
}

// BuildStrategyRegistry manages available build strategies
type BuildStrategyRegistry struct {
	strategies map[deploymentsv1.BuildStrategy]BuildStrategy
}

// NewBuildStrategyRegistry creates a new registry with all strategies
func NewBuildStrategyRegistry() *BuildStrategyRegistry {
	registry := &BuildStrategyRegistry{
		strategies: make(map[deploymentsv1.BuildStrategy]BuildStrategy),
	}

	// Register all strategies
	registry.Register(deploymentsv1.BuildStrategy_RAILPACK, NewRailpackStrategy())
	registry.Register(deploymentsv1.BuildStrategy_NIXPACKS, NewNixpacksStrategy())
	registry.Register(deploymentsv1.BuildStrategy_DOCKERFILE, NewDockerfileStrategy())
	registry.Register(deploymentsv1.BuildStrategy_PLAIN_COMPOSE, NewPlainComposeStrategy())
	registry.Register(deploymentsv1.BuildStrategy_COMPOSE_REPO, NewComposeRepoStrategy())
	registry.Register(deploymentsv1.BuildStrategy_STATIC_SITE, NewStaticStrategy())

	return registry
}

// Register registers a build strategy
func (r *BuildStrategyRegistry) Register(strategyType deploymentsv1.BuildStrategy, strategy BuildStrategy) {
	r.strategies[strategyType] = strategy
}

// Get returns a build strategy by type
func (r *BuildStrategyRegistry) Get(strategyType deploymentsv1.BuildStrategy) (BuildStrategy, error) {
	strategy, ok := r.strategies[strategyType]
	if !ok {
		return nil, fmt.Errorf("unknown build strategy: %v", strategyType)
	}
	return strategy, nil
}

// AutoDetect detects the appropriate build strategy for a repository
func (r *BuildStrategyRegistry) AutoDetect(ctx context.Context, repoPath string) (deploymentsv1.BuildStrategy, error) {
	// Try strategies in order of preference
	detectionOrder := []deploymentsv1.BuildStrategy{
		deploymentsv1.BuildStrategy_COMPOSE_REPO, // Check for docker-compose.yml in repo first
		deploymentsv1.BuildStrategy_DOCKERFILE,   // Then Dockerfile
		deploymentsv1.BuildStrategy_RAILPACK,    // Then Railpack (default for most languages)
		deploymentsv1.BuildStrategy_NIXPACKS,     // Then generic Nixpacks (fallback)
		deploymentsv1.BuildStrategy_STATIC_SITE,  // Finally static
	}

	for _, strategyType := range detectionOrder {
		strategy, ok := r.strategies[strategyType]
		if !ok {
			continue
		}

		detected, err := strategy.Detect(ctx, repoPath)
		if err != nil {
			log.Printf("[AutoDetect] Error detecting strategy %v: %v", strategyType, err)
			continue
		}

		if detected {
			log.Printf("[AutoDetect] Detected build strategy: %v", strategyType)
			return strategyType, nil
		}
	}

	// Default to RAILPACK if nothing detected
	return deploymentsv1.BuildStrategy_RAILPACK, nil
}

// InferDeploymentType infers the deployment type from build strategy and repository contents
func (r *BuildStrategyRegistry) InferDeploymentType(ctx context.Context, buildStrategy deploymentsv1.BuildStrategy, repoPath string) deploymentsv1.DeploymentType {
	switch buildStrategy {
	case deploymentsv1.BuildStrategy_STATIC_SITE:
		return deploymentsv1.DeploymentType_STATIC

	case deploymentsv1.BuildStrategy_PLAIN_COMPOSE, deploymentsv1.BuildStrategy_DOCKERFILE:
		return deploymentsv1.DeploymentType_DOCKER

	case deploymentsv1.BuildStrategy_RAILPACK, deploymentsv1.BuildStrategy_NIXPACKS:
		// Both Railpack and Nixpacks can detect and build various languages:
		// Clojure, Cobol, Crystal, C#/.NET, Dart, Deno, Elixir, F#, Gleam, Go,
		// Haskell, Java, Lunatic, Node, PHP, Python, Ruby, Rust, Scheme,
		// Staticfile, Swift, Scala, Zig
		// Detect language from repository contents
		return detectLanguageFromRepo(repoPath)

	default:
		return deploymentsv1.DeploymentType_GENERIC
	}
}

// detectLanguageFromRepo detects the programming language from repository contents
// Supports all languages that Railpack/Nixpacks can build:
// Clojure, Cobol, Crystal, C#/.NET, Dart, Deno, Elixir, F#, Gleam, Go, Haskell,
// Java, Lunatic, Node, PHP, Python, Ruby, Rust, Scheme, Staticfile, Swift, Scala, Zig
func detectLanguageFromRepo(repoPath string) deploymentsv1.DeploymentType {
	// Check for language-specific files in order of specificity

	// Ruby/Rails (Gemfile)
	if fileExists(filepath.Join(repoPath, "Gemfile")) || fileExists(filepath.Join(repoPath, "Gemfile.lock")) {
		return deploymentsv1.DeploymentType_RUBY
	}

	// Deno (deno.json) - check before Node.js as it's more specific
	if fileExists(filepath.Join(repoPath, "deno.json")) || fileExists(filepath.Join(repoPath, "deno.jsonc")) {
		// Deno is close to Node.js but uses different runtime
		// For now, map to NODE since we don't have a Deno type
		return deploymentsv1.DeploymentType_NODE
	}

	// Node.js (package.json)
	if fileExists(filepath.Join(repoPath, "package.json")) {
		return deploymentsv1.DeploymentType_NODE
	}

	// Go (go.mod)
	if fileExists(filepath.Join(repoPath, "go.mod")) || fileExists(filepath.Join(repoPath, "go.sum")) {
		return deploymentsv1.DeploymentType_GO
	}

	// Python (requirements.txt, setup.py, Pipfile, pyproject.toml)
	if fileExists(filepath.Join(repoPath, "requirements.txt")) ||
		fileExists(filepath.Join(repoPath, "setup.py")) ||
		fileExists(filepath.Join(repoPath, "Pipfile")) ||
		fileExists(filepath.Join(repoPath, "pyproject.toml")) ||
		fileExists(filepath.Join(repoPath, "Pipfile.lock")) {
		return deploymentsv1.DeploymentType_PYTHON
	}

	// Rust (Cargo.toml)
	if fileExists(filepath.Join(repoPath, "Cargo.toml")) || fileExists(filepath.Join(repoPath, "Cargo.lock")) {
		return deploymentsv1.DeploymentType_RUST
	}

	// Java/Scala (pom.xml, build.gradle) - Scala runs on JVM
	if fileExists(filepath.Join(repoPath, "pom.xml")) ||
		fileExists(filepath.Join(repoPath, "build.gradle")) ||
		fileExists(filepath.Join(repoPath, "build.gradle.kts")) ||
		fileExists(filepath.Join(repoPath, "build.sbt")) || // Scala SBT
		fileExists(filepath.Join(repoPath, "project", "plugins.sbt")) { // Scala SBT
		return deploymentsv1.DeploymentType_JAVA
	}

	// PHP (composer.json)
	if fileExists(filepath.Join(repoPath, "composer.json")) || fileExists(filepath.Join(repoPath, "composer.lock")) {
		return deploymentsv1.DeploymentType_PHP
	}

	// C#/.NET (including F#)
	if hasAnyFile(repoPath, "*.csproj", "*.fsproj", "*.sln", "project.json", "*.vbproj") {
		// C#/.NET maps to GENERIC since we don't have a specific type
		return deploymentsv1.DeploymentType_GENERIC
	}

	// Elixir (mix.exs)
	if fileExists(filepath.Join(repoPath, "mix.exs")) || fileExists(filepath.Join(repoPath, "mix.lock")) {
		// Elixir maps to GENERIC since we don't have a specific type
		return deploymentsv1.DeploymentType_GENERIC
	}

	// Dart (pubspec.yaml)
	if fileExists(filepath.Join(repoPath, "pubspec.yaml")) || fileExists(filepath.Join(repoPath, "pubspec.yml")) {
		return deploymentsv1.DeploymentType_GENERIC
	}

	// Crystal (shard.yml)
	if fileExists(filepath.Join(repoPath, "shard.yml")) {
		return deploymentsv1.DeploymentType_GENERIC
	}

	// Gleam (gleam.toml)
	if fileExists(filepath.Join(repoPath, "gleam.toml")) {
		return deploymentsv1.DeploymentType_GENERIC
	}

	// Zig (build.zig)
	if fileExists(filepath.Join(repoPath, "build.zig")) || fileExists(filepath.Join(repoPath, "build.zig.zon")) {
		return deploymentsv1.DeploymentType_GENERIC
	}

	// Swift (Package.swift)
	if fileExists(filepath.Join(repoPath, "Package.swift")) {
		return deploymentsv1.DeploymentType_GENERIC
	}

	// Haskell (*.cabal, stack.yaml, package.yaml)
	if hasAnyFile(repoPath, "*.cabal", "stack.yaml", "package.yaml") ||
		fileExists(filepath.Join(repoPath, "stack.yaml")) ||
		fileExists(filepath.Join(repoPath, "package.yaml")) {
		return deploymentsv1.DeploymentType_GENERIC
	}

	// Clojure (project.clj, build.boot, deps.edn)
	if fileExists(filepath.Join(repoPath, "project.clj")) ||
		fileExists(filepath.Join(repoPath, "build.boot")) ||
		fileExists(filepath.Join(repoPath, "deps.edn")) {
		return deploymentsv1.DeploymentType_GENERIC
	}

	// Lunatic (lunatic.toml)
	if fileExists(filepath.Join(repoPath, "lunatic.toml")) {
		return deploymentsv1.DeploymentType_GENERIC
	}

	// Staticfile (Heroku static buildpack indicator)
	if fileExists(filepath.Join(repoPath, "Staticfile")) ||
		fileExists(filepath.Join(repoPath, "static.json")) {
		return deploymentsv1.DeploymentType_STATIC
	}

	// HTML/Static (index.html in root)
	if fileExists(filepath.Join(repoPath, "index.html")) &&
		!fileExists(filepath.Join(repoPath, "package.json")) && // Don't confuse with Node.js
		!fileExists(filepath.Join(repoPath, "Dockerfile")) {
		return deploymentsv1.DeploymentType_STATIC
	}

	// Default to generic if we can't detect
	// Could be: Cobol, Scheme, Shell scripts, or other languages
	return deploymentsv1.DeploymentType_GENERIC
}

// ensureBuildDir creates a build directory for a deployment
func ensureBuildDir(deploymentID string) (string, error) {
	// Try multiple possible locations
	possibleDirs := []string{
		"/var/lib/obiente/builds",
		"/tmp/obiente-builds",
		os.TempDir(),
	}

	for _, baseDir := range possibleDirs {
		buildDir := filepath.Join(baseDir, deploymentID)
		if err := os.MkdirAll(buildDir, 0755); err == nil {
			// Verify we can write to it
			testFile := filepath.Join(buildDir, ".test")
			if err := os.WriteFile(testFile, []byte("test"), 0644); err == nil {
				os.Remove(testFile)
				return buildDir, nil
			}
		}
	}

	return "", fmt.Errorf("failed to create build directory in any of the attempted locations")
}

// cloneRepository clones a git repository to the build directory
func cloneRepository(ctx context.Context, repoURL, branch, destDir string, githubToken string) error {
	// Remove destination if it exists
	os.RemoveAll(destDir)

	// If this is a GitHub repository and we have a token, inject it into the URL for authentication
	authenticatedURL := repoURL
	if githubToken != "" && isGitHubURL(repoURL) {
		authenticatedURL = injectGitHubToken(repoURL, githubToken)
	}

	// Clone repository
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", branch, authenticatedURL, destDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// isGitHubURL checks if the URL is a GitHub repository URL
func isGitHubURL(url string) bool {
	return strings.HasPrefix(url, "https://github.com/") || 
		   strings.HasPrefix(url, "http://github.com/") ||
		   strings.HasPrefix(url, "git@github.com:")
}

// injectGitHubToken injects a GitHub token into a repository URL for authentication
func injectGitHubToken(repoURL, token string) string {
	// Handle HTTPS GitHub URLs
	if strings.HasPrefix(repoURL, "https://github.com/") {
		// Replace https://github.com/ with https://token@github.com/
		repoPath := strings.TrimPrefix(repoURL, "https://github.com/")
		return fmt.Sprintf("https://%s@github.com/%s", token, repoPath)
	}
	if strings.HasPrefix(repoURL, "http://github.com/") {
		// Replace http://github.com/ with http://token@github.com/
		repoPath := strings.TrimPrefix(repoURL, "http://github.com/")
		return fmt.Sprintf("http://%s@github.com/%s", token, repoPath)
	}
	// For SSH URLs (git@github.com:), tokens aren't used - SSH keys are required instead
	// Return original URL if it's SSH or unknown format
	return repoURL
}

// execCommand runs a command in a directory
func execCommand(ctx context.Context, dir, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// readFile reads a file and returns its contents
func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// hasFile checks if any of the given files exist in the directory
func hasFile(dir string, filenames ...string) bool {
	for _, filename := range filenames {
		if fileExists(filepath.Join(dir, filename)) {
			return true
		}
	}
	return false
}

// hasAnyFile checks if any of the given file patterns exist
func hasAnyFile(dir string, patterns ...string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		name := entry.Name()
		for _, pattern := range patterns {
			matched, _ := filepath.Match(pattern, name)
			if matched {
				return true
			}
		}
	}

	return false
}

// getEnvAsStringSlice converts a map[string]string to []string for exec commands
// It preserves existing environment variables and allows overriding them
func getEnvAsStringSlice(envVars map[string]string) []string {
	// Start with current environment
	existingEnv := os.Environ()
	envMap := make(map[string]string)
	
	// Parse existing environment into a map to allow overrides
	for _, env := range existingEnv {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	
	// Override/add custom environment variables
	for k, v := range envVars {
		envMap[k] = v
	}
	
	// Convert back to []string format
	env := make([]string, 0, len(envMap))
	for k, v := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	
	return env
}

// buildDockerImage builds a Docker image from a directory
func buildDockerImage(ctx context.Context, dir, imageName, dockerfile string, logWriter, logWriterErr io.Writer) error {
	args := []string{"build", "-t", imageName}
	if dockerfile != "" {
		// If dockerfile is a relative path, make it absolute relative to dir
		// If it's already absolute, use it as-is
		if !filepath.IsAbs(dockerfile) {
			dockerfile = filepath.Join(dir, dockerfile)
		}
		args = append(args, "-f", dockerfile)
	}
	args = append(args, dir)

	cmd := exec.CommandContext(ctx, "docker", args...)

	// Use log writers if provided, otherwise fallback to os.Stdout/Stderr
	if logWriter != nil {
		cmd.Stdout = NewMultiWriter(os.Stdout, logWriter)
	} else {
		cmd.Stdout = os.Stdout
	}
	if logWriterErr != nil {
		cmd.Stderr = NewMultiWriter(os.Stderr, logWriterErr)
	} else {
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

// getImageSize gets the size of a Docker image in bytes
func getImageSize(ctx context.Context, imageName string) (int64, error) {
	// Use docker image inspect to get image size
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", imageName, "--format", "{{.Size}}")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get image size: %w", err)
	}

	// Parse size from output
	sizeStr := strings.TrimSpace(string(output))
	var size int64
	if _, err := fmt.Sscanf(sizeStr, "%d", &size); err != nil {
		return 0, fmt.Errorf("failed to parse image size: %w", err)
	}

	return size, nil
}

// extractImageCmd extracts the CMD from a Docker image
// Returns the CMD as a string that can be used as a start command, or empty string if not found
func extractImageCmd(ctx context.Context, imageName string) (string, string) {
	// Use docker image inspect to get the CMD and WorkingDir
	cmdCmd := exec.CommandContext(ctx, "docker", "image", "inspect", imageName, "--format", "{{json .Config.Cmd}}")
	cmdOutput, err := cmdCmd.Output()
	if err != nil {
		return "", ""
	}

	// Get working directory
	workdirCmd := exec.CommandContext(ctx, "docker", "image", "inspect", imageName, "--format", "{{.Config.WorkingDir}}")
	workdirOutput, err := workdirCmd.Output()
	workingDir := strings.TrimSpace(string(workdirOutput))
	if err != nil || workingDir == "" || workingDir == "<no value>" {
		workingDir = ""
	}

	// Parse the output - Docker inspect returns CMD as a JSON array
	cmdStr := strings.TrimSpace(string(cmdOutput))
	if cmdStr == "" || cmdStr == "null" || cmdStr == "[]" {
		return "", workingDir
	}

	// Parse as JSON array
	var cmdParts []string
	if err := json.Unmarshal([]byte(cmdStr), &cmdParts); err != nil {
		return "", workingDir
	}

	if len(cmdParts) == 0 {
		return "", workingDir
	}

	// If the command is just ["/bin/bash"] or similar shell, return empty (use image default)
	if len(cmdParts) == 1 && (cmdParts[0] == "/bin/bash" || cmdParts[0] == "/bin/sh" || cmdParts[0] == "sh" || cmdParts[0] == "bash") {
		return "", workingDir
	}

	// If it's ["/bin/bash", "-c", "command"], extract just the command
	if len(cmdParts) >= 3 && (cmdParts[0] == "/bin/bash" || cmdParts[0] == "/bin/sh" || cmdParts[0] == "sh" || cmdParts[0] == "bash") && cmdParts[1] == "-c" {
		return strings.Join(cmdParts[2:], " "), workingDir
	}

	// Otherwise, join all parts with spaces
	return strings.Join(cmdParts, " "), workingDir
}

// deployResultToOrchestrator converts a BuildResult to orchestrator configuration
func deployResultToOrchestrator(ctx context.Context, manager *orchestrator.DeploymentManager, deployment *database.Deployment, result *BuildResult) error {
	if manager == nil {
		return fmt.Errorf("deployment manager is not available (orchestrator not initialized)")
	}

	if result.ComposeYaml != "" {
		// Use compose deployment
		return manager.DeployComposeFile(ctx, deployment.ID, result.ComposeYaml)
	} else if result.ImageName != "" {
		// Use single container deployment
		port := result.Port
		if port == 0 {
			port = 8080
			if deployment.Port != nil {
				port = int(*deployment.Port)
			}
		}

		envVars := make(map[string]string)
		if deployment.EnvVars != "" {
			// Parse env vars from JSON (implement parseEnvVars if needed)
		}

		memory := int64(512 * 1024 * 1024) // Default 512MB
		if deployment.MemoryBytes != nil {
			memory = *deployment.MemoryBytes
		}
		cpuShares := int64(1024) // Default
		if deployment.CPUShares != nil {
			cpuShares = *deployment.CPUShares
		}

		var startCmd *string
		if deployment.StartCommand != nil && *deployment.StartCommand != "" {
			startCmd = deployment.StartCommand
		}
		
		// Ensure image name includes registry prefix for Swarm mode
		imageName := result.ImageName
		if imageName != "" {
			// Check if we're in Swarm mode and image doesn't have registry prefix
			// This ensures worker nodes can pull from the registry
			enableSwarm := os.Getenv("ENABLE_SWARM")
			isSwarmMode := false
			if enableSwarm != "" {
				// Parse as boolean (handles "true", "1", "yes", "on", etc.)
				enabled, err := strconv.ParseBool(strings.ToLower(enableSwarm))
				if err == nil {
					isSwarmMode = enabled
				} else {
					// Fallback: check for common true values
					lower := strings.ToLower(enableSwarm)
					isSwarmMode = lower == "true" || lower == "1" || lower == "yes" || lower == "on"
				}
			}
			
			if isSwarmMode {
				registryURL := os.Getenv("REGISTRY_URL")
				if registryURL == "" {
					domain := os.Getenv("DOMAIN")
					if domain == "" {
						domain = "obiente.cloud"
					}
					registryURL = fmt.Sprintf("https://registry.%s", domain)
				} else {
					// Handle unexpanded docker-compose variables
					if strings.Contains(registryURL, "${DOMAIN") {
						domain := os.Getenv("DOMAIN")
						if domain == "" {
							domain = "obiente.cloud"
						}
						registryURL = strings.ReplaceAll(registryURL, "${DOMAIN:-obiente.cloud}", domain)
						registryURL = strings.ReplaceAll(registryURL, "${DOMAIN}", domain)
					}
				}
				
				registryHost := strings.TrimPrefix(registryURL, "https://")
				registryHost = strings.TrimPrefix(registryHost, "http://")
				
				// If image doesn't have registry prefix and looks like our deployment image, add it
				if !strings.Contains(imageName, registryHost) && 
				   !strings.Contains(imageName, "ghcr.io/") && 
				   !strings.Contains(imageName, "docker.io/") &&
				   strings.Contains(imageName, "obiente/deploy-") {
					imageName = fmt.Sprintf("%s/%s", registryHost, imageName)
					log.Printf("[deployResultToOrchestrator] Added registry prefix to image name: %s", imageName)
				}
			}
		}
		
		cfg := &orchestrator.DeploymentConfig{
			DeploymentID: deployment.ID,
			Image:        imageName,
			Domain:       deployment.Domain,
			Port:         port,
			EnvVars:      envVars,
			Labels:       map[string]string{},
			Memory:       memory,
			CPUShares:    cpuShares,
			Replicas:     1,
			StartCommand: startCmd, // Pass start command to override container CMD
		}

		if deployment.Replicas != nil {
			cfg.Replicas = int(*deployment.Replicas)
		}

		return manager.CreateDeployment(ctx, cfg)
	}

	return fmt.Errorf("build result contains neither image nor compose yaml")
}
