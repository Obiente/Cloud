package deployments

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	if err := cloneRepository(ctx, config.RepositoryURL, config.Branch, buildDir); err != nil {
		return &BuildResult{Success: false, Error: err}, err
	}

	imageName := fmt.Sprintf("obiente/%s:%s", deployment.ID, deployment.Branch)

	// Use Nixpacks to build application (Railpacks uses Nixpacks under the hood)
	// Nixpacks will auto-detect the language (supports: Clojure, Cobol, Crystal, C#/.NET,
	// Dart, Deno, Elixir, F#, Gleam, Go, Haskell, Java, Lunatic, Node, PHP, Python,
	// Ruby, Rust, Scheme, Staticfile, Swift, Scala, Zig)
	// and build accordingly with minimal configuration
	cmd := exec.CommandContext(ctx, "nixpacks", "build", buildDir, "--name", imageName)
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
		return &BuildResult{Success: false, Error: fmt.Errorf("nixpacks build failed: %w", err)}, nil
	}

	// Auto-detect port based on detected language
	port := config.Port
	if port == 0 {
		// Try to detect from repository, default to 3000 (common for Rails/Node)
		port = detectPortFromRepo(buildDir, 3000)
	}

	return &BuildResult{
		ImageName: imageName,
		Port:      port,
		Success:   true,
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
	if err := cloneRepository(ctx, config.RepositoryURL, config.Branch, buildDir); err != nil {
		return &BuildResult{Success: false, Error: err}, err
	}

	imageName := fmt.Sprintf("obiente/%s:%s", deployment.ID, deployment.Branch)

	// Use Nixpacks to build application
	cmd := exec.CommandContext(ctx, "nixpacks", "build", buildDir, "--name", imageName)
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
		return &BuildResult{Success: false, Error: fmt.Errorf("nixpacks build failed: %w", err)}, nil
	}

	// Auto-detect port based on framework
	port := s.detectPort(buildDir, config.Port)

	return &BuildResult{
		ImageName: imageName,
		Port:      port,
		Success:   true,
	}, nil
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
	if err := cloneRepository(ctx, config.RepositoryURL, config.Branch, buildDir); err != nil {
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

	// Try to detect port from Dockerfile EXPOSE directive
	port := s.detectPortFromDockerfile(dockerfilePath, config.Port)

	return &BuildResult{
		ImageName: imageName,
		Port:      port,
		Success:   true,
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
	if err := cloneRepository(ctx, config.RepositoryURL, branch, buildDir); err != nil {
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
	if err := cloneRepository(ctx, config.RepositoryURL, config.Branch, buildDir); err != nil {
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
