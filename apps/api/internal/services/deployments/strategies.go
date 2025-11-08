package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"api/internal/database"
	"api/internal/logger"
)

// RailpackStrategy handles Railpack deployments
// Railpack is a variant of Nixpacks that works out of the box with minimal configuration.
// It supports: Clojure, Cobol, Crystal, C#/.NET, Dart, Deno, Elixir, F#, Gleam, Go,
// Haskell, Java, Lunatic, Node, PHP, Python, Ruby, Rust, Scheme, Staticfile,
// Swift, Scala, Zig
// The detection is Rails-optimized, but the build can handle any supported language.
type RailpackStrategy struct{}

func NewRailpackStrategy() *RailpackStrategy {
	return &RailpackStrategy{}
}

func (s *RailpackStrategy) Name() string {
	return "Railpack"
}

func (s *RailpackStrategy) Detect(ctx context.Context, repoPath string) (bool, error) {
	// Railpack can build any language that Nixpacks can build:
	// Clojure, Cobol, Crystal, C#/.NET, Dart, Deno, Elixir, F#, Gleam, Go,
	// Haskell, Java, Lunatic, Node, PHP, Python, Ruby, Rust, Scheme,
	// Staticfile, Swift, Scala, Zig
	// Check for common indicators that Railpack can handle (same as Nixpacks)

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
		// Ruby/Rails
		"Gemfile",
		// Elixir/Phoenix
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
		// Swift
		"Package.swift",
		// Staticfile
		"Staticfile",
		"static.json",
		// Clojure
		"project.clj",
		"build.boot",
		"deps.edn",
		// Lunatic
		"lunatic.toml",
	}

	for _, indicator := range exactIndicators {
		if fileExists(filepath.Join(repoPath, indicator)) {
			return true, nil
		}
	}

	// Check for pattern-based indicators
	patterns := []string{
		"*.csproj",    // C#/.NET
		"*.fsproj",    // F#
		"*.sln",       // .NET Solution
		"*.vbproj",    // VB.NET
		"*.cabal",     // Haskell
		"stack.yaml",  // Haskell Stack
		"package.yaml", // Haskell
	}

	for _, pattern := range patterns {
		if hasAnyFile(repoPath, pattern) {
			return true, nil
		}
	}

	// Check for HTML/static sites (index.html without package.json or Dockerfile)
	if fileExists(filepath.Join(repoPath, "index.html")) &&
		!fileExists(filepath.Join(repoPath, "package.json")) &&
		!fileExists(filepath.Join(repoPath, "Dockerfile")) {
		return true, nil
	}

	// Don't use Railpack if Dockerfile exists (should use Dockerfile strategy instead)
	if fileExists(filepath.Join(repoPath, "Dockerfile")) {
		return false, nil
	}

	// Don't use Railpack if docker-compose.yml exists (should use Plain Compose strategy instead)
	if hasFile(repoPath, "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml") {
		return false, nil
	}

	// Default fallback: Railpack can try to build anything
	// This makes Railpack the default fallback instead of Nixpacks
	return true, nil
}

func (s *RailpackStrategy) Build(ctx context.Context, deployment *database.Deployment, config *BuildConfig) (*BuildResult, error) {
	// Helper function to write to build logs if writer is available
	writeBuildLog := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		if config.LogWriter != nil {
			config.LogWriter.Write([]byte(msg + "\n"))
		}
		logger.Debug("[Railpack] %s", msg)
	}

	writeBuildLog("🚀 Obiente Cloud: Starting deployment build")
	writeBuildLog("   📦 Build strategy: Railpack (Railway optimized)")
	writeBuildLog("   🔗 Repository: %s (branch: %s)", config.RepositoryURL, config.Branch)

	buildDir, err := ensureBuildDir(deployment.ID)
	if err != nil {
		return &BuildResult{Success: false, Error: err}, err
	}

	// Clone repository
	writeBuildLog("   📥 Cloning repository...")
	if err := cloneRepository(ctx, config.RepositoryURL, config.Branch, buildDir, config.GitHubToken); err != nil {
		return &BuildResult{Success: false, Error: err}, err
	}
	writeBuildLog("   ✅ Repository cloned successfully")

	imageName := fmt.Sprintf("obiente/%s:%s", deployment.ID, deployment.Branch)

	// Determine build working directory (default to repo root)
	buildWorkDir := buildDir
	if config.BuildPath != "" {
		buildWorkDir = filepath.Join(buildDir, config.BuildPath)
		// Ensure build directory exists
		if err := os.MkdirAll(buildWorkDir, 0755); err != nil {
			return &BuildResult{Success: false, Error: fmt.Errorf("failed to create build path: %w", err)}, nil
		}
		writeBuildLog("   📁 Build path: %s", config.BuildPath)
	}

	// Railpack doesn't need config files - it auto-detects everything
	// However, we need to ensure static sites that need building get built
	// For Astro and similar frameworks, we may need to modify the start command
	// to include a build step if the build output doesn't exist
	if config.LogWriter != nil {
		config.LogWriter.Write([]byte("   🔧 Analyzing project...\n"))
	}

	// Check if this is a static site that needs building (like Astro)
	// If so, ensure the start command includes building
	startCommand := config.StartCommand
	if startCommand == "" {
		startCommand = detectDefaultStartCommand(buildDir)
	}

	// COMMENTED OUT: Astro detection logic - temporarily disabled for testing
	// For Astro projects using preview, ensure build happens first
	// isAstro := fileExists(filepath.Join(buildDir, "astro.config.js")) ||
	// 	fileExists(filepath.Join(buildDir, "astro.config.mjs")) ||
	// 	fileExists(filepath.Join(buildDir, "astro.config.ts"))

	// if isAstro && strings.Contains(startCommand, "preview") {
	// 	// Railpack will run the start command, but we need to ensure build happens first
	// 	// Check if dist folder exists - if not, we need to build first
	// 	distExists := fileExists(filepath.Join(buildDir, "dist"))

	// 	// Detect package manager for build command
	// 	var buildCmd string
	// 	if fileExists(filepath.Join(buildDir, "pnpm-lock.yaml")) {
	// 		buildCmd = "pnpm build"
	// 	} else if fileExists(filepath.Join(buildDir, "yarn.lock")) {
	// 		buildCmd = "yarn build"
	// 	} else {
	// 		buildCmd = "npm run build"
	// 	}

	// 	// If dist doesn't exist, we need to build before preview
	// 	// Railpack doesn't have a separate build phase, so we chain the commands
	// 	if !distExists {
	// 		// Combine build and preview into one command
	// 		// Use && to ensure build completes before preview starts
	// 		if strings.Contains(startCommand, "pnpm") {
	// 			startCommand = fmt.Sprintf("%s && pnpm preview --host", buildCmd)
	// 		} else if strings.Contains(startCommand, "yarn") {
	// 			startCommand = fmt.Sprintf("%s && yarn preview --host", buildCmd)
	// 		} else {
	// 			startCommand = fmt.Sprintf("%s && npm run preview -- --host", buildCmd)
	// 		}
	// 		if config.LogWriter != nil {
	// 			config.LogWriter.Write([]byte(fmt.Sprintf("   🔧 Detected Astro project - configured start command to build first: %s\n", startCommand)))
	// 		}
	// 	} else {
	// 		// Dist exists, just ensure preview has --host flag
	// 		if !strings.Contains(startCommand, "--host") {
	// 			if strings.Contains(startCommand, "pnpm") {
	// 				startCommand = strings.Replace(startCommand, "pnpm preview", "pnpm preview --host", 1)
	// 			} else if strings.Contains(startCommand, "yarn") {
	// 				startCommand = strings.Replace(startCommand, "yarn preview", "yarn preview --host", 1)
	// 			} else {
	// 				startCommand = strings.Replace(startCommand, "npm run preview", "npm run preview -- --host", 1)
	// 			}
	// 		}
	// 	}
	// }

	// Store the modified start command back in config so it's used when the container starts
	// The start command will be set at container runtime by the orchestrator
	config.StartCommand = startCommand

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
	// Use buildWorkDir (which includes BuildPath if set) instead of buildDir
	cmd := exec.CommandContext(ctx, railpackPath, "build", buildWorkDir, "--name", imageName)
	
	// Prepare environment variables - add RAILPACK_* vars to config.EnvVars before converting
	// This ensures they override any existing values
	railpackEnvVars := make(map[string]string)
	// Copy existing env vars
	for k, v := range config.EnvVars {
		railpackEnvVars[k] = v
	}
	// Override with RAILPACK_* commands if provided
	// See https://railpack.com/config/environment-variables
	if config.InstallCommand != "" {
		railpackEnvVars["RAILPACK_INSTALL_CMD"] = config.InstallCommand
		writeBuildLog("   📦 Install command: %s", config.InstallCommand)
	}
	if config.BuildCommand != "" {
		railpackEnvVars["RAILPACK_BUILD_CMD"] = config.BuildCommand
		writeBuildLog("   🔨 Build command: %s", config.BuildCommand)
	}
	if config.StartCommand != "" {
		railpackEnvVars["RAILPACK_START_CMD"] = config.StartCommand
		writeBuildLog("   🚀 Start command: %s", config.StartCommand)
	}
	
	envVars := getEnvAsStringSlice(railpackEnvVars)

	// Railpack requires BUILDKIT_HOST to be set for BuildKit builds
	// Check if buildx is available; if not, disable BuildKit as fallback
	if !isBuildxAvailable(ctx) {
		logger.Warn("[Railpack] Buildx not available, disabling BuildKit")
		envVars = append(envVars, "DOCKER_BUILDKIT=0")
	} else {
		// Enable BuildKit and ensure BuildKit daemon container is running
		// Railpack uses BuildKit for building, so we need to provide a BuildKit daemon
		// Start buildkit container if it doesn't exist or isn't running
		startBuildkitContainer(ctx)
		envVars = append(envVars, "DOCKER_BUILDKIT=1")
		envVars = append(envVars, "BUILDKIT_HOST=docker-container://obiente-cloud-buildkit")
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

	// Extract start command from railpack image if not already set
	// Railpack embeds the start command in the image's CMD, so we should extract it
	if config.StartCommand == "" {
		extractedCmd, _ := extractImageCmd(ctx, imageName)
		if extractedCmd != "" {
			config.StartCommand = extractedCmd
			writeBuildLog("   📌 Extracted start command from image: %s", extractedCmd)
		} else {
			// If extraction failed or returned empty, try to detect from repository
			detectedCmd := detectDefaultStartCommand(buildWorkDir)
			if detectedCmd != "" {
				config.StartCommand = detectedCmd
				writeBuildLog("   📌 Detected start command from repository: %s", detectedCmd)
			}
		}
	}

	// Get image size
	imageSize := int64(0)
	if size, err := getImageSize(ctx, imageName); err != nil {
		logger.Warn("[Railpack] Warning: Failed to get image size: %v", err)
	} else {
		imageSize = size
	}

	// Auto-detect port based on detected language
	port := config.Port
	if port == 0 {
		// Try to detect from repository, default to 3000 (common for Rails/Node)
		// Use buildWorkDir (which includes BuildPath if set) for port detection
		port = detectPortFromRepo(buildWorkDir, 3000)
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
// Checks multiple sources in order of priority:
// 1. .env files (PORT=)
// 2. package.json (scripts, config)
// 3. Framework-specific config files (next.config.js, vite.config.js, etc.)
// 4. Dockerfile (EXPOSE)
// 5. Language-specific defaults
func detectPortFromRepo(repoPath string, defaultPort int) int {
	// Priority 1: Check .env files for PORT
	envPort := detectPortFromEnv(repoPath)
	if envPort > 0 {
		return envPort
	}

	// Priority 2: Check package.json for Node.js projects
	if fileExists(filepath.Join(repoPath, "package.json")) {
		if port := detectPortFromPackageJson(repoPath); port > 0 {
			return port
		}
		// Check framework-specific configs
		if port := detectPortFromNodeFramework(repoPath); port > 0 {
			return port
		}
		// Default Node.js port
		return 8080
	}

	// Priority 3: Check Dockerfile for EXPOSE directive
	if fileExists(filepath.Join(repoPath, "Dockerfile")) {
		if port := detectPortFromDockerfile(filepath.Join(repoPath, "Dockerfile")); port > 0 {
			return port
		}
	}

	// Priority 4: Check Deno config
	if fileExists(filepath.Join(repoPath, "deno.json")) || fileExists(filepath.Join(repoPath, "deno.jsonc")) {
		if port := detectPortFromDenoConfig(repoPath); port > 0 {
			return port
		}
		return 8080 // Default Deno port
	}

	// Priority 5: Check Python config
	if fileExists(filepath.Join(repoPath, "requirements.txt")) ||
		fileExists(filepath.Join(repoPath, "Pipfile")) ||
		fileExists(filepath.Join(repoPath, "pyproject.toml")) {
		if port := detectPortFromPythonConfig(repoPath); port > 0 {
			return port
		}
		return 8000 // Default Python port
	}

	// Priority 6: Check Go
	if fileExists(filepath.Join(repoPath, "go.mod")) {
		if port := detectPortFromGoCode(repoPath); port > 0 {
			return port
		}
		return 8080 // Default Go port
	}

	// Priority 7: Check Ruby/Rails
	if fileExists(filepath.Join(repoPath, "Gemfile")) {
		if port := detectPortFromRailsConfig(repoPath); port > 0 {
			return port
		}
		return 3000 // Default Rails port
	}

	// Priority 8: Other languages
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

// detectPortFromEnv checks .env files for PORT variable
func detectPortFromEnv(repoPath string) int {
	envFiles := []string{".env", ".env.local", ".env.production", ".env.development"}
	for _, envFile := range envFiles {
		envPath := filepath.Join(repoPath, envFile)
		if !fileExists(envPath) {
			continue
		}
		content, err := readFile(envPath)
		if err != nil {
			continue
		}
		// Look for PORT=1234 pattern (with optional whitespace)
		portRegex := regexp.MustCompile(`(?i)^\s*PORT\s*=\s*(\d+)\s*$`)
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			matches := portRegex.FindStringSubmatch(strings.TrimSpace(line))
			if len(matches) > 1 {
				if port, err := strconv.Atoi(matches[1]); err == nil && port > 0 && port < 65536 {
					return port
				}
			}
		}
	}
	return 0
}

// detectPortFromPackageJson parses package.json for PORT in scripts or config
func detectPortFromPackageJson(repoPath string) int {
	packageJsonPath := filepath.Join(repoPath, "package.json")
	content, err := readFile(packageJsonPath)
	if err != nil {
		return 0
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return 0
	}

	// Check for PORT in scripts
	if scripts, ok := pkg["scripts"].(map[string]interface{}); ok {
		for _, script := range scripts {
			if scriptStr, ok := script.(string); ok {
				// Look for PORT=1234 or -p 1234 or --port 1234 in scripts
				portRegex := regexp.MustCompile(`(?i)(?:PORT\s*=\s*|--port\s+|--port\s*=\s*|-p\s+)(\d+)`)
				matches := portRegex.FindStringSubmatch(scriptStr)
				if len(matches) > 1 {
					if port, err := strconv.Atoi(matches[1]); err == nil && port > 0 && port < 65536 {
						return port
					}
				}
			}
		}
	}

	// Check for config.port or config.PORT
	if config, ok := pkg["config"].(map[string]interface{}); ok {
		if portVal, ok := config["port"]; ok {
			if port, ok := portVal.(float64); ok && port > 0 && port < 65536 {
				return int(port)
			}
		}
		if portVal, ok := config["PORT"]; ok {
			if port, ok := portVal.(float64); ok && port > 0 && port < 65536 {
				return int(port)
			}
		}
	}

	return 0
}

// detectPortFromNodeFramework checks framework-specific config files
func detectPortFromNodeFramework(repoPath string) int {
	// Astro
	if fileExists(filepath.Join(repoPath, "astro.config.js")) ||
		fileExists(filepath.Join(repoPath, "astro.config.mjs")) ||
		fileExists(filepath.Join(repoPath, "astro.config.ts")) {
		// Check astro config for port
		configFiles := []string{"astro.config.js", "astro.config.mjs", "astro.config.ts"}
		for _, configFile := range configFiles {
			configPath := filepath.Join(repoPath, configFile)
			if !fileExists(configPath) {
				continue
			}
			content, err := readFile(configPath)
			if err != nil {
				continue
			}
			// Look for server.port in Astro config
			portRegex := regexp.MustCompile(`(?i)(?:server\s*:\s*\{[^}]*)?port\s*[:=]\s*(\d+)`)
			matches := portRegex.FindStringSubmatch(content)
			if len(matches) > 1 {
				if port, err := strconv.Atoi(matches[1]); err == nil && port > 0 && port < 65536 {
					return port
				}
			}
		}
		return 4321 // Astro default port
	}

	// Next.js
	if fileExists(filepath.Join(repoPath, "next.config.js")) || fileExists(filepath.Join(repoPath, "next.config.mjs")) {
		if port := detectPortFromNextConfig(repoPath); port > 0 {
			return port
		}
	}

	// Vite
	if fileExists(filepath.Join(repoPath, "vite.config.js")) || fileExists(filepath.Join(repoPath, "vite.config.ts")) {
		if port := detectPortFromViteConfig(repoPath); port > 0 {
			return port
		}
	}

	// Nuxt.js
	if fileExists(filepath.Join(repoPath, "nuxt.config.js")) || fileExists(filepath.Join(repoPath, "nuxt.config.ts")) {
		if port := detectPortFromNuxtConfig(repoPath); port > 0 {
			return port
		}
	}

	// Express.js (check server.js, app.js, index.js for listen())
	if port := detectPortFromExpressCode(repoPath); port > 0 {
		return port
	}

	return 0
}

// detectPortFromNextConfig checks Next.js config for port
func detectPortFromNextConfig(repoPath string) int {
	configFiles := []string{"next.config.js", "next.config.mjs", "next.config.ts"}
	for _, configFile := range configFiles {
		configPath := filepath.Join(repoPath, configFile)
		if !fileExists(configPath) {
			continue
		}
		content, err := readFile(configPath)
		if err != nil {
			continue
		}
		// Look for port in config (common patterns)
		portRegex := regexp.MustCompile(`(?i)port\s*[:=]\s*(\d+)`)
		matches := portRegex.FindStringSubmatch(content)
		if len(matches) > 1 {
			if port, err := strconv.Atoi(matches[1]); err == nil && port > 0 && port < 65536 {
				return port
			}
		}
	}
	return 3000 // Next.js default
}

// detectPortFromViteConfig checks Vite config for port
func detectPortFromViteConfig(repoPath string) int {
	configFiles := []string{"vite.config.js", "vite.config.ts", "vite.config.mjs"}
	for _, configFile := range configFiles {
		configPath := filepath.Join(repoPath, configFile)
		if !fileExists(configPath) {
			continue
		}
		content, err := readFile(configPath)
		if err != nil {
			continue
		}
		// Look for port in server config
		portRegex := regexp.MustCompile(`(?i)(?:server\s*:\s*\{[^}]*)?port\s*[:=]\s*(\d+)`)
		matches := portRegex.FindStringSubmatch(content)
		if len(matches) > 1 {
			if port, err := strconv.Atoi(matches[1]); err == nil && port > 0 && port < 65536 {
				return port
			}
		}
	}
	return 5173 // Vite default
}

// detectPortFromNuxtConfig checks Nuxt config for port
func detectPortFromNuxtConfig(repoPath string) int {
	configFiles := []string{"nuxt.config.js", "nuxt.config.ts"}
	for _, configFile := range configFiles {
		configPath := filepath.Join(repoPath, configFile)
		if !fileExists(configPath) {
			continue
		}
		content, err := readFile(configPath)
		if err != nil {
			continue
		}
		// Look for port in server config
		portRegex := regexp.MustCompile(`(?i)(?:server\s*:\s*\{[^}]*)?port\s*[:=]\s*(\d+)`)
		matches := portRegex.FindStringSubmatch(content)
		if len(matches) > 1 {
			if port, err := strconv.Atoi(matches[1]); err == nil && port > 0 && port < 65536 {
				return port
			}
		}
	}
	return 3000 // Nuxt default
}

// detectPortFromExpressCode checks Express.js code for app.listen() or server.listen()
func detectPortFromExpressCode(repoPath string) int {
	jsFiles := []string{"server.js", "app.js", "index.js", "main.js", "src/server.js", "src/app.js", "src/index.js"}
	for _, jsFile := range jsFiles {
		filePath := filepath.Join(repoPath, jsFile)
		if !fileExists(filePath) {
			continue
		}
		content, err := readFile(filePath)
		if err != nil {
			continue
		}
		// Look for .listen(port) or .listen(PORT) or process.env.PORT
		portRegex := regexp.MustCompile(`(?i)\.listen\s*\(\s*(?:process\.env\.PORT\s*\|\s*)?(\d+)`)
		matches := portRegex.FindStringSubmatch(content)
		if len(matches) > 1 {
			if port, err := strconv.Atoi(matches[1]); err == nil && port > 0 && port < 65536 {
				return port
			}
		}
	}
	return 0
}

// detectPortFromDockerfile checks Dockerfile for EXPOSE directive
func detectPortFromDockerfile(dockerfilePath string) int {
	content, err := readFile(dockerfilePath)
	if err != nil {
		return 0
	}
	// Look for EXPOSE directive
	exposeRegex := regexp.MustCompile(`(?i)^\s*EXPOSE\s+(\d+)`)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		matches := exposeRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			if port, err := strconv.Atoi(matches[1]); err == nil && port > 0 && port < 65536 {
				return port
			}
		}
	}
	return 0
}

// detectPortFromDenoConfig checks Deno config files
func detectPortFromDenoConfig(repoPath string) int {
	configFiles := []string{"deno.json", "deno.jsonc"}
	for _, configFile := range configFiles {
		configPath := filepath.Join(repoPath, configFile)
		if !fileExists(configPath) {
			continue
		}
		content, err := readFile(configPath)
		if err != nil {
			continue
		}
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(content), &config); err != nil {
			continue
		}
		// Check for port in config
		if portVal, ok := config["port"]; ok {
			if port, ok := portVal.(float64); ok && port > 0 && port < 65536 {
				return int(port)
			}
		}
	}
	return 0
}

// detectPortFromPythonConfig checks Python config files
func detectPortFromPythonConfig(repoPath string) int {
	// Check for common patterns in Python files
	pythonFiles := []string{"main.py", "app.py", "server.py", "wsgi.py", "asgi.py"}
	for _, pyFile := range pythonFiles {
		filePath := filepath.Join(repoPath, pyFile)
		if !fileExists(filePath) {
			continue
		}
		content, err := readFile(filePath)
		if err != nil {
			continue
		}
		// Look for port in app.run(port=) or uvicorn.run(port=)
		portRegex := regexp.MustCompile(`(?i)(?:app\.run|uvicorn\.run|server\.run).*?port\s*=\s*(\d+)`)
		matches := portRegex.FindStringSubmatch(content)
		if len(matches) > 1 {
			if port, err := strconv.Atoi(matches[1]); err == nil && port > 0 && port < 65536 {
				return port
			}
		}
	}
	return 0
}

// detectPortFromGoCode checks Go code for port patterns
func detectPortFromGoCode(repoPath string) int {
	// Check main.go for common patterns
	mainGoPath := filepath.Join(repoPath, "main.go")
	if fileExists(mainGoPath) {
		content, err := readFile(mainGoPath)
		if err == nil {
			// Look for :8080 or ":8080" patterns
			portRegex := regexp.MustCompile(`(?i):(\d+)(?:\s|"|'|,|\)|$)`)
			matches := portRegex.FindStringSubmatch(content)
			if len(matches) > 1 {
				if port, err := strconv.Atoi(matches[1]); err == nil && port > 0 && port < 65536 {
					return port
				}
			}
		}
	}
	return 0
}

// detectPortFromRailsConfig checks Rails config for port
func detectPortFromRailsConfig(repoPath string) int {
	// Check config/puma.rb or config/application.rb
	configFiles := []string{"config/puma.rb", "config/application.rb"}
	for _, configFile := range configFiles {
		configPath := filepath.Join(repoPath, configFile)
		if !fileExists(configPath) {
			continue
		}
		content, err := readFile(configPath)
		if err != nil {
			continue
		}
		// Look for port in config
		portRegex := regexp.MustCompile(`(?i)port\s+(\d+)`)
		matches := portRegex.FindStringSubmatch(content)
		if len(matches) > 1 {
			if port, err := strconv.Atoi(matches[1]); err == nil && port > 0 && port < 65536 {
				return port
			}
		}
	}
	return 0
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
	// Helper function to write to build logs if writer is available
	writeBuildLog := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		if config.LogWriter != nil {
			config.LogWriter.Write([]byte(msg + "\n"))
		}
		logger.Debug("[Nixpacks] %s", msg)
	}

	writeBuildLog("🚀 Obiente Cloud: Starting deployment build")
	writeBuildLog("   📦 Build strategy: Nixpacks")
	writeBuildLog("   🔗 Repository: %s (branch: %s)", config.RepositoryURL, config.Branch)

	buildDir, err := ensureBuildDir(deployment.ID)
	if err != nil {
		return &BuildResult{Success: false, Error: err}, err
	}

	// Clone repository
	writeBuildLog("   📥 Cloning repository...")
	if err := cloneRepository(ctx, config.RepositoryURL, config.Branch, buildDir, config.GitHubToken); err != nil {
		return &BuildResult{Success: false, Error: err}, err
	}
	writeBuildLog("   ✅ Repository cloned successfully")

	imageName := fmt.Sprintf("obiente/%s:%s", deployment.ID, deployment.Branch)

	// Determine build working directory (default to repo root)
	buildWorkDir := buildDir
	if config.BuildPath != "" {
		buildWorkDir = filepath.Join(buildDir, config.BuildPath)
		// Ensure build directory exists
		if err := os.MkdirAll(buildWorkDir, 0755); err != nil {
			return &BuildResult{Success: false, Error: fmt.Errorf("failed to create build path: %w", err)}, nil
		}
		writeBuildLog("   📁 Build path: %s", config.BuildPath)
	}

	// Create nixpacks.toml with install, build, and start commands if provided
	// Use standard nixpacks provider (not Railway's)
	writeBuildLog("   🔧 Analyzing project and configuring build...")
	// Create config in buildWorkDir (which includes BuildPath if set)
	if err := createNixpacksConfig(buildWorkDir, config.InstallCommand, config.BuildCommand, config.StartCommand, false, config.LogWriter); err != nil {
		logger.Warn("[Nixpacks] Warning: Failed to create nixpacks.toml: %v", err)
	}

	// Use Nixpacks to build application
	writeBuildLog("   🔨 Building application with Nixpacks...")
	// Use buildWorkDir (which includes BuildPath if set) instead of buildDir
	cmd := exec.CommandContext(ctx, "nixpacks", "build", buildWorkDir, "--name", imageName)
	envVars := getEnvAsStringSlice(config.EnvVars)
	// Check if buildx is available; if not, disable BuildKit as fallback
	if !isBuildxAvailable(ctx) {
		logger.Warn("[Nixpacks] Buildx not available, disabling BuildKit")
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
		logger.Warn("[Nixpacks] Warning: Failed to get image size: %v", err)
	} else {
		imageSize = size
	}

	// Auto-detect port based on framework
	// Use buildWorkDir (which includes BuildPath if set) for port detection
	port := s.detectPort(buildWorkDir, config.Port)

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

// checkBuildkitContainer checks if a BuildKit container named "obiente-cloud-buildkit" is running
func checkBuildkitContainer(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "ps", "--filter", "name=obiente-cloud-buildkit", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "obiente-cloud-buildkit"
}

// startBuildkitContainer starts a BuildKit daemon container if it doesn't exist
// We spawn BuildKit manually because Railpack requires BUILDKIT_HOST=docker-container://buildkit
// Docker does NOT automatically spawn BuildKit containers - we need to provide a BuildKit daemon
// when using the docker-container:// protocol. This allows BuildKit to run in a separate container
// with better isolation and caching capabilities.
func startBuildkitContainer(ctx context.Context) {
	containerName := "obiente-cloud-buildkit"

	// Check if buildkit container already exists (running or stopped)
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		logger.Warn("[Railpack] Failed to check for buildkit container: %v", err)
		return
	}

	if strings.TrimSpace(string(output)) == containerName {
		// Container exists, try to start it if it's stopped
		startCmd := exec.CommandContext(ctx, "docker", "start", containerName)
		if err := startCmd.Run(); err != nil {
			logger.Warn("[Railpack] Failed to start existing buildkit container: %v", err)
		} else {
			logger.Info("[Railpack] Started existing BuildKit container: %s", containerName)
		}
		return
	}

	// Container doesn't exist, create and start it
	// Create BuildKit cache volume in obiente volume path
	buildkitCacheDir := "/var/lib/obiente/volumes/buildkit-cache"
	// Fallback to temp directory if obiente volume path doesn't exist
	if _, err := os.Stat("/var/lib/obiente/volumes"); os.IsNotExist(err) {
		buildkitCacheDir = "/var/obiente/tmp/obiente-volumes/buildkit-cache"
		// Final fallback to /tmp if /var/obiente/tmp doesn't exist
		if _, err := os.Stat("/var/obiente/tmp"); os.IsNotExist(err) {
			buildkitCacheDir = "/tmp/obiente-volumes/buildkit-cache"
		}
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(buildkitCacheDir, 0755); err != nil {
		logger.Warn("[Railpack] Warning: Failed to create BuildKit cache directory %s: %v", buildkitCacheDir, err)
		// Continue without cache directory
		buildkitCacheDir = ""
	}

	logger.Debug("[Railpack] Starting BuildKit daemon container: %s", containerName)

	// Build docker run command with volume mount for BuildKit cache
	// Note: We don't use --rm because we want the container to persist for caching
	args := []string{
		"run",
		"--privileged",
		"-d",
		"--name", containerName,
		"--restart", "unless-stopped",
	}

	// Add volume mount for BuildKit cache if directory was created
	if buildkitCacheDir != "" {
		args = append(args, "-v", fmt.Sprintf("%s:/var/lib/buildkit", buildkitCacheDir))
		logger.Debug("[Railpack] Using BuildKit cache directory: %s", buildkitCacheDir)
	}

	args = append(args, "moby/buildkit:latest")

	createCmd := exec.CommandContext(ctx, "docker", args...)
	if err := createCmd.Run(); err != nil {
		logger.Warn("[Railpack] Failed to start BuildKit container: %v. Railpack may fail.", err)
	} else {
		logger.Info("[Railpack] Successfully started BuildKit container: %s", containerName)
	}
}

// createNixpacksConfig creates a nixpacks.toml file with install, build, and start commands
// This is used by both NixpacksStrategy and RailpackStrategy
// If startCommand is empty, it attempts to detect a default from the repository
// If Node.js version is not specified, it attempts to detect from package.json engines field
// useRailwayProvider: if true, configures for Railway's Railpack provider (uses Railway base images)
// logWriter: optional writer for build logs (if nil, only logs to API server)
func createNixpacksConfig(buildDir, installCommand, buildCommand, startCommand string, useRailwayProvider bool, logWriter io.Writer) error {
	nixpacksConfigPath := filepath.Join(buildDir, "nixpacks.toml")

	// Helper function to write to build logs if writer is available
	writeBuildLog := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		if logWriter != nil {
			logWriter.Write([]byte(msg + "\n"))
		}
		logger.Debug("[Nixpacks] %s", msg)
	}

	// Detect start command if not provided
	detectedStartCommand := ""
	if startCommand == "" {
		detectedStartCommand = detectDefaultStartCommand(buildDir)
		if detectedStartCommand != "" {
			startCommand = detectedStartCommand
			writeBuildLog("✨ Obiente Cloud: Auto-detected start command: %s", startCommand)
		}
	} else {
		writeBuildLog("✨ Obiente Cloud: Using provided start command: %s", startCommand)
	}

	// Detect Node.js version requirement from package.json or .nvmrc
	// detectNodeVersion checks .nvmrc first, then package.json engines.node
	nvmrcPath := filepath.Join(buildDir, ".nvmrc")
	packageJsonPath := filepath.Join(buildDir, "package.json")
	nodeVersion := detectNodeVersion(buildDir)
	versionSource := ""

	if nodeVersion != "" {
		// Determine source for better logging
		versionSource = "package.json engines.node"
		if fileExists(nvmrcPath) {
			// Check if .nvmrc has content (it was checked first in detectNodeVersion)
			content, err := os.ReadFile(nvmrcPath)
			if err == nil {
				version := strings.TrimSpace(string(content))
				if version != "" && normalizeNodeVersion(version) == nodeVersion {
					versionSource = ".nvmrc file"
				}
			}
		}
		writeBuildLog("✨ Obiente Cloud: Auto-detected Node.js version requirement: %s (from %s)", nodeVersion, versionSource)
		writeBuildLog("   📦 Configuring Nixpacks to use Node.js %s", nodeVersion)
		writeBuildLog("   ℹ️  This ensures your app uses the exact Node.js version you specified")
	} else {
		// Use latest LTS Node.js version as default (Node.js 20 as of 2024)
		// This is better than Nixpacks' default which uses an older version
		nodeVersion = getDefaultNodeVersion()
		versionSource = "Obiente Cloud default (latest LTS)"

		// Check if package.json exists to provide helpful message
		if fileExists(packageJsonPath) {
			writeBuildLog("✨ Obiente Cloud: No Node.js version specified in package.json or .nvmrc")
			writeBuildLog("   📦 Using latest LTS Node.js version: %s (Obiente Cloud default)", nodeVersion)
			writeBuildLog("   💡 Tip: Add \"engines\": { \"node\": \">=20.0.0\" } to package.json")
			writeBuildLog("      Or create a .nvmrc file to pin your Node.js version")
			writeBuildLog("      This ensures consistent Node.js versions across environments")
		} else {
			writeBuildLog("✨ Obiente Cloud: Using latest LTS Node.js version: %s (default)", nodeVersion)
		}
	}

	// Check if this is an Astro project and needs special build configuration
	isAstro := fileExists(filepath.Join(buildDir, "astro.config.js")) ||
		fileExists(filepath.Join(buildDir, "astro.config.mjs")) ||
		fileExists(filepath.Join(buildDir, "astro.config.ts"))

	// Build nixpacks.toml content
	var configParts []string

	// Add provider configuration for Railway Railpack if requested
	if useRailwayProvider {
		// Railway's Railpack uses railway provider which selects Railway's optimized base images
		// This ensures we use ghcr.io/railwayapp/nixpacks base images instead of standard ones
		configParts = append(configParts, "[provider]\nname = \"railway\"\n")
	}

	// Set Node.js version via environment variable
	// Nixpacks will read this along with .nvmrc file to determine the Node.js version
	configParts = append(configParts, fmt.Sprintf("[variables]\nNODE_VERSION = %q\n", nodeVersion))

	// Add install phase if provided (overrides default npm ci)
	if installCommand != "" {
		configParts = append(configParts, fmt.Sprintf("[phases.install]\ncmds = [%q]\n", installCommand))
		writeBuildLog("   📦 Configured install command: %s", installCommand)
	}

	// Add build phase if provided
	if buildCommand != "" {
		configParts = append(configParts, fmt.Sprintf("[phases.build]\ncmds = [%q]\n", buildCommand))
		writeBuildLog("   🔨 Configured build command: %s", buildCommand)
	}

	// For Astro projects, ensure build command runs before start
	// Only add if buildCommand wasn't already provided
	if isAstro && startCommand != "" && buildCommand == "" {
		// Check if start command is a preview command (needs build first)
		if strings.Contains(startCommand, "preview") {
			// Detect package manager for build command
			var buildCmd string
			if fileExists(filepath.Join(buildDir, "pnpm-lock.yaml")) {
				buildCmd = "pnpm build"
			} else if fileExists(filepath.Join(buildDir, "yarn.lock")) {
				buildCmd = "yarn build"
			} else {
				buildCmd = "npm run build"
			}

			// Add build phase to ensure Astro builds before preview
			configParts = append(configParts, fmt.Sprintf("[phases.build]\ncmds = [%q]\n", buildCmd))
			writeBuildLog("   🔧 Detected Astro project - configuring build phase: %s", buildCmd)
		}
	}

	// Add start command if provided
	if startCommand != "" {
		configParts = append(configParts, fmt.Sprintf("[start]\ncmd = %q\n", startCommand))
	}

	// If no configuration needed, don't create file (let nixpacks auto-detect)
	if len(configParts) == 0 {
		return nil
	}

	// Check if files exist before creating them (to preserve user's custom configs)
	nixpacksConfigExists := fileExists(nixpacksConfigPath)
	nvmrcExists := fileExists(nvmrcPath)

	// Only create nixpacks.toml if it doesn't already exist (don't overwrite user's custom config)
	if !nixpacksConfigExists {
		configContent := strings.Join(configParts, "\n")
		if err := os.WriteFile(nixpacksConfigPath, []byte(configContent), 0644); err != nil {
			return fmt.Errorf("failed to write nixpacks.toml: %w", err)
		}
		logger.Debug("[Nixpacks] Created nixpacks.toml with content:\n%s", configContent)
	} else {
		logger.Debug("[Nixpacks] nixpacks.toml already exists, skipping creation to preserve user's configuration")
		// Still log what we would have configured for debugging
		configContent := strings.Join(configParts, "\n")
		logger.Debug("[Nixpacks] Would have created nixpacks.toml with content:\n%s", configContent)
	}

	// Create .nvmrc file for Node.js version specification
	// Nixpacks automatically reads .nvmrc to determine the Node.js version
	// This is the primary method for Node.js version specification in Nixpacks
	// Only create if .nvmrc doesn't already exist (don't overwrite user's file)
	if !nvmrcExists {
		if err := os.WriteFile(nvmrcPath, []byte(nodeVersion+"\n"), 0644); err != nil {
			// Non-fatal - NODE_VERSION variable is also set
			logger.Debug("[Nixpacks] Failed to create .nvmrc file: %v", err)
		} else {
			logger.Debug("[Nixpacks] Created .nvmrc file with version: %s", nodeVersion)
		}
	}

	// Provide user-friendly summary of what was configured
	writeBuildLog("✅ Obiente Cloud: Build configuration complete")
	writeBuildLog("   📌 Node.js version: %s (%s)", nodeVersion, versionSource)
	if installCommand != "" {
		writeBuildLog("   📌 Install command: %s", installCommand)
	}
	if buildCommand != "" {
		writeBuildLog("   📌 Build command: %s", buildCommand)
	}
	if startCommand != "" {
		writeBuildLog("   📌 Start command: %s", startCommand)
	}

	// List which files were created vs preserved
	var configFiles []string
	if !nixpacksConfigExists {
		configFiles = append(configFiles, "nixpacks.toml (created)")
	} else {
		configFiles = append(configFiles, "nixpacks.toml (preserved)")
	}
	if !nvmrcExists {
		configFiles = append(configFiles, ".nvmrc (created)")
	} else {
		configFiles = append(configFiles, ".nvmrc (preserved)")
	}
	writeBuildLog("   📌 Configuration files: %s", strings.Join(configFiles, ", "))

	return nil
}

// createRailpackConfig creates a nixpacks.toml file with install, build, and start commands
// Railpack reads nixpacks.toml files (same format as nixpacks)
// This is more reliable than environment variables for BuildKit builds
func createRailpackConfig(buildDir, installCommand, buildCommand, startCommand string, logWriter io.Writer) error {
	nixpacksConfigPath := filepath.Join(buildDir, "nixpacks.toml")

	// Helper function to write to build logs if writer is available
	writeBuildLog := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		if logWriter != nil {
			logWriter.Write([]byte(msg + "\n"))
		}
		logger.Debug("[Railpack] %s", msg)
	}

	// Check if config file already exists - don't overwrite user's custom config
	if fileExists(nixpacksConfigPath) {
		writeBuildLog("   📄 nixpacks.toml already exists, will append commands (user config preserved)")
		// Read existing config to check if we need to add anything
		existingContent, err := os.ReadFile(nixpacksConfigPath)
		if err == nil {
			existingStr := string(existingContent)
			// If it already has install/build phases, don't override
			if strings.Contains(existingStr, "[phases.install]") && installCommand != "" {
				writeBuildLog("   ⚠️  Install command already specified in config, skipping")
				installCommand = ""
			}
			if strings.Contains(existingStr, "[phases.build]") && buildCommand != "" {
				writeBuildLog("   ⚠️  Build command already specified in config, skipping")
				buildCommand = ""
			}
			if strings.Contains(existingStr, "[start]") && startCommand != "" {
				writeBuildLog("   ⚠️  Start command already specified in config, skipping")
				startCommand = ""
			}
		}
	}

	// Build config parts
	var configParts []string

	// Add install phase if provided
	if installCommand != "" {
		configParts = append(configParts, fmt.Sprintf("[phases.install]\ncmds = [%q]\n", installCommand))
		writeBuildLog("   📦 Configured install command: %s", installCommand)
	}

	// Add build phase if provided
	if buildCommand != "" {
		configParts = append(configParts, fmt.Sprintf("[phases.build]\ncmds = [%q]\n", buildCommand))
		writeBuildLog("   🔨 Configured build command: %s", buildCommand)
	}

	// Add start command if provided
	if startCommand != "" {
		configParts = append(configParts, fmt.Sprintf("[start]\ncmd = %q\n", startCommand))
		writeBuildLog("   🚀 Configured start command: %s", startCommand)
	}

	// If no config to add, return
	if len(configParts) == 0 {
		return nil
	}

	// Append to existing file or create new one
	configContent := strings.Join(configParts, "\n")
	if fileExists(nixpacksConfigPath) {
		// Append to existing file
		existingContent, err := os.ReadFile(nixpacksConfigPath)
		if err == nil {
			existingStr := string(existingContent)
			// Append new config to existing content
			if !strings.HasSuffix(existingStr, "\n") {
				configContent = existingStr + "\n" + configContent
			} else {
				configContent = existingStr + configContent
			}
		}
	}

	if err := os.WriteFile(nixpacksConfigPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write nixpacks.toml: %w", err)
	}

	writeBuildLog("   ✅ Created/updated nixpacks.toml with custom commands")
	logger.Debug("[Railpack] Created nixpacks.toml with content:\n%s", configContent)

	return nil
}

// detectNodeVersion attempts to detect the required Node.js version from package.json
// Returns the version string (e.g., "18.20.8" or "20") or empty string if not found
// Also checks for .nvmrc file as a fallback
func detectNodeVersion(buildDir string) string {
	// First, try to read from .nvmrc file (if it exists, it's the explicit preference)
	nvmrcPath := filepath.Join(buildDir, ".nvmrc")
	if fileExists(nvmrcPath) {
		content, err := os.ReadFile(nvmrcPath)
		if err == nil {
			version := strings.TrimSpace(string(content))
			if version != "" {
				logger.Debug("[Nixpacks] Detected Node.js version from .nvmrc: %s", version)
				return normalizeNodeVersion(version)
			}
		}
	}

	// Then check package.json engines field
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
				detected := normalizeNodeVersion(matches[1])
				logger.Debug("[Nixpacks] Detected Node.js version from package.json (regex fallback): %s", detected)
				return detected
			}
		}
		return ""
	}

	if pkg.Engines.Node != "" {
		detected := normalizeNodeVersion(pkg.Engines.Node)
		logger.Debug("[Nixpacks] Detected Node.js version from package.json engines.node: %s", detected)
		return detected
	}

	return ""
}

// getDefaultNodeVersion returns the latest LTS Node.js version to use as default
// Currently Node.js 20 is the latest stable LTS (as of 2024)
// This can be updated when newer LTS versions are released
func getDefaultNodeVersion() string {
	// Node.js 20 is the current LTS version (as of October 2024)
	// When Node.js 22 becomes LTS or we want to use a newer version, update this
	return "20"
}

// normalizeNodeVersion normalizes Node.js version strings to a format nixpacks accepts
// Handles patterns like ">=18.20.8", "18.x", "20", "^18.20.8", "~18.20.8"
// Returns the minimum version that satisfies the constraint
// For ">=" constraints, ensures we use the exact minimum version specified
func normalizeNodeVersion(version string) string {
	// Remove common version prefixes
	version = strings.TrimSpace(version)

	// Handle ">=" constraints - take the minimum version and ensure it's used as-is
	if strings.HasPrefix(version, ">=") {
		version = strings.TrimPrefix(version, ">=")
		version = strings.TrimSpace(version)
		// For ">=18.20.8", return "18.20.8" to ensure exact version
		// This ensures Nixpacks uses at least this version, not an older one
		return version
	}

	// Handle ">" constraints - bump patch version if needed
	if strings.HasPrefix(version, ">") {
		version = strings.TrimPrefix(version, ">")
		version = strings.TrimSpace(version)
		// For ">18.20.8", we'd want at least 18.20.9, but Nixpacks should handle this
		// For now, return as-is and let Nixpacks resolve
		return version
	}

	// Remove other prefixes that don't affect minimum version
	version = strings.TrimPrefix(version, "^")
	version = strings.TrimPrefix(version, "~")
	version = strings.TrimPrefix(version, "=")

	// Handle "x" versions like "18.x" or "20.x"
	if strings.Contains(version, ".x") {
		parts := strings.Split(version, ".")
		if len(parts) > 0 {
			major := parts[0]
			// For "18.x", return "18" to let Nixpacks pick latest 18.x
			// But if we detected a specific requirement like ">=18.20.8",
			// we should have caught it above
			return major
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

	// Check for Astro first (needs special handling)
	if fileExists(filepath.Join(buildDir, "astro.config.js")) || fileExists(filepath.Join(buildDir, "astro.config.mjs")) || fileExists(filepath.Join(buildDir, "astro.config.ts")) {
		// Check if Astro is configured for server mode
		astroConfig := detectAstroOutputMode(buildDir)
		if astroConfig == "server" {
			// Astro SSR mode - needs Node.js adapter
			// Check for built server files
			if fileExists(filepath.Join(buildDir, "dist", "server", "entry.mjs")) {
				return "node ./dist/server/entry.mjs"
			}
			// For SSR mode, check package.json for start script first
			packageJsonPath := filepath.Join(buildDir, "package.json")
			if fileExists(packageJsonPath) {
				content, err := os.ReadFile(packageJsonPath)
				if err == nil {
					contentStr := string(content)
					if strings.Contains(contentStr, `"start"`) {
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
			// Fallback: use preview (will work after build)
			if fileExists(filepath.Join(buildDir, "pnpm-lock.yaml")) {
				return "pnpm preview --host"
			}
			if fileExists(filepath.Join(buildDir, "yarn.lock")) {
				return "yarn preview --host"
			}
			return "npm run preview -- --host"
		} else {
			// Astro static mode - check if dist exists, if so serve it, otherwise use preview
			// In production, built static files should be served
			if fileExists(filepath.Join(buildDir, "dist")) {
				// Use a simple static server if available, or fallback to preview
				// Check for serve package
				packageJsonPath := filepath.Join(buildDir, "package.json")
				if fileExists(packageJsonPath) {
					content, err := os.ReadFile(packageJsonPath)
					if err == nil && strings.Contains(string(content), `"serve"`) {
						if fileExists(filepath.Join(buildDir, "pnpm-lock.yaml")) {
							return "pnpm serve dist"
						}
						if fileExists(filepath.Join(buildDir, "yarn.lock")) {
							return "yarn serve dist"
						}
						return "npx serve dist"
					}
				}
				// Fallback to preview (works for static sites too)
				if fileExists(filepath.Join(buildDir, "pnpm-lock.yaml")) {
					return "pnpm preview --host"
				}
				if fileExists(filepath.Join(buildDir, "yarn.lock")) {
					return "yarn preview --host"
				}
				return "npm run preview -- --host"
			}
			// No dist folder yet - will be built by build phase
			if fileExists(filepath.Join(buildDir, "pnpm-lock.yaml")) {
				return "pnpm preview --host"
			}
			if fileExists(filepath.Join(buildDir, "yarn.lock")) {
				return "yarn preview --host"
			}
			return "npm run preview -- --host"
		}
	}

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
			// Check for preview script (common for static sites)
			if strings.Contains(contentStr, `"preview"`) {
				if fileExists(filepath.Join(buildDir, "pnpm-lock.yaml")) {
					return "pnpm preview"
				}
				if fileExists(filepath.Join(buildDir, "yarn.lock")) {
					return "yarn preview"
				}
				return "npm run preview"
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

// detectAstroOutputMode checks Astro config to determine if it's in server or static mode
func detectAstroOutputMode(buildDir string) string {
	configFiles := []string{"astro.config.js", "astro.config.mjs", "astro.config.ts"}
	for _, configFile := range configFiles {
		configPath := filepath.Join(buildDir, configFile)
		if !fileExists(configPath) {
			continue
		}
		content, err := readFile(configPath)
		if err != nil {
			continue
		}
		// Look for output: "server" or adapter configuration
		if strings.Contains(content, `output: "server"`) || strings.Contains(content, `output: 'server'`) {
			return "server"
		}
		// Check for Node adapter
		if strings.Contains(content, `@astrojs/node`) || strings.Contains(content, `"@astrojs/node"`) {
			return "server"
		}
		// Check for hybrid mode
		if strings.Contains(content, `output: "hybrid"`) || strings.Contains(content, `output: 'hybrid'`) {
			return "server"
		}
	}
	// Default to static if not specified
	return "static"
}

func (s *NixpacksStrategy) detectPort(repoPath string, defaultPort int) int {
	if defaultPort != 0 {
		return defaultPort
	}
	// Use the improved port detection
	return detectPortFromRepo(repoPath, 8080)
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
	logger.Info("[Dockerfile] Building deployment %s", deployment.ID)

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
		return &BuildResult{Success: false, Error: fmt.Errorf("dockerfile not found at path: %s", dockerfile)}, nil
	}

	if err := buildDockerImage(ctx, buildDir, imageName, dockerfile, config.LogWriter, config.LogWriterErr); err != nil {
		return &BuildResult{Success: false, Error: fmt.Errorf("docker build failed: %w", err)}, nil
	}

	// Get image size
	var imageSize int64
	if size, err := getImageSize(ctx, imageName); err != nil {
		logger.Warn("[Dockerfile] Warning: Failed to get image size: %v", err)
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

	// Try Dockerfile first
	if port := detectPortFromDockerfile(dockerfilePath); port > 0 {
		return port
	}

	// Fallback to checking the repo directory for other configs
	dockerfileDir := filepath.Dir(dockerfilePath)
	return detectPortFromRepo(dockerfileDir, 8080)
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
	logger.Info("[PlainCompose] Building deployment %s", deployment.ID)

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
	logger.Info("[ComposeRepo] Building deployment %s from repository", deployment.ID)

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
	logger.Info("[Static] Building deployment %s", deployment.ID)

	buildDir, err := ensureBuildDir(deployment.ID)
	if err != nil {
		return &BuildResult{Success: false, Error: err}, err
	}

	// Clone repository
	if err := cloneRepository(ctx, config.RepositoryURL, config.Branch, buildDir, config.GitHubToken); err != nil {
		return &BuildResult{Success: false, Error: err}, nil
	}

	// Determine build working directory (default to repo root)
	buildWorkDir := buildDir
	if config.BuildPath != "" {
		buildWorkDir = filepath.Join(buildDir, config.BuildPath)
		// Ensure build directory exists
		if err := os.MkdirAll(buildWorkDir, 0755); err != nil {
			return &BuildResult{Success: false, Error: fmt.Errorf("failed to create build path: %w", err)}, nil
		}
	}

	// Step 1: Use Railpack to build the application
	// This will create an image with all dependencies and built files
	railpackImageName := fmt.Sprintf("obiente/%s-railpack:%s", deployment.ID, deployment.Branch)

	writeBuildLog := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		if config.LogWriter != nil {
			config.LogWriter.Write([]byte(msg + "\n"))
		}
		logger.Debug("[Static] %s", msg)
	}

	writeBuildLog("🔨 Building with Railpack...")
	writeBuildLog("   Expected image name: %s", railpackImageName)
	
	// Use Railpack to build - Railpack is a CLI tool that builds images
	// Check if railpack is available, if not use the RailpackStrategy approach
	railpackPath := "/usr/local/bin/railpack"
	var usingRailpackCLI bool
	if path, err := exec.LookPath("railpack"); err == nil {
		railpackPath = path
		usingRailpackCLI = true
		writeBuildLog("   Found railpack CLI at: %s", path)
	} else if _, err := os.Stat(railpackPath); err == nil {
		usingRailpackCLI = true
		writeBuildLog("   Found railpack CLI at: %s", railpackPath)
	} else {
		// Railpack CLI not found, use RailpackStrategy's build method instead
		usingRailpackCLI = false
		writeBuildLog("⚠️  Railpack CLI not found, using Railpack build method...")
		writeBuildLog("   Searched in PATH and %s", railpackPath)
		railpackStrategy := NewRailpackStrategy()

		// Create a temporary build config for Railpack
		railpackConfig := *config
		railpackConfig.LogWriter = config.LogWriter
		railpackConfig.LogWriterErr = config.LogWriterErr

		// Build with Railpack (this will write all logs to LogWriter/LogWriterErr)
		writeBuildLog("📝 Railpack build logs will be shown below...")
		railpackResult, err := railpackStrategy.Build(ctx, deployment, &railpackConfig)
		if err != nil || !railpackResult.Success {
			writeBuildLog("❌ Railpack build failed")
			if err != nil {
				writeBuildLog("   Error: %v", err)
			}
			if railpackResult != nil && railpackResult.Error != nil {
				writeBuildLog("   Build error: %v", railpackResult.Error)
			}
			return &BuildResult{Success: false, Error: fmt.Errorf("railpack build failed: %w", err)}, nil
		}

		// Use the Railpack-built image as our source
		// Note: RailpackStrategy returns image name as obiente/{id}:{branch} (no -railpack suffix)
		oldImageName := railpackImageName
		railpackImageName = railpackResult.ImageName
		writeBuildLog("✅ Railpack build completed (via RailpackStrategy)")
		writeBuildLog("   Original expected name: %s", oldImageName)
		writeBuildLog("   Actual image name: %s", railpackImageName)
		
		// Verify the image exists
		verifyCmd := exec.CommandContext(ctx, "docker", "image", "inspect", railpackImageName)
		verifyOutput, verifyErr := verifyCmd.CombinedOutput()
		if verifyErr != nil {
			writeBuildLog("❌ Railpack image %s not found after build", railpackImageName)
			writeBuildLog("   Error: %v", verifyErr)
			if len(verifyOutput) > 0 {
				writeBuildLog("   Docker output: %s", strings.TrimSpace(string(verifyOutput)))
			}
			writeBuildLog("💡 This may indicate the build failed silently")
			return &BuildResult{Success: false, Error: fmt.Errorf("railpack build completed but image %s was not created", railpackImageName)}, nil
		}
		writeBuildLog("   ✅ Image verified: %s", railpackImageName)
	}
	
	if usingRailpackCLI {
		// Use railpack CLI directly
		writeBuildLog("📦 Using railpack CLI at: %s", railpackPath)
		writeBuildLog("📁 Build directory: %s", buildWorkDir)
		// Use buildWorkDir (which includes BuildPath if set) instead of buildDir
		cmd := exec.CommandContext(ctx, railpackPath, "build", buildWorkDir, "--name", railpackImageName)
		
		// Prepare environment variables - add RAILPACK_* vars to config.EnvVars before converting
		// This ensures they override any existing values
		railpackEnvVars := make(map[string]string)
		// Copy existing env vars
		for k, v := range config.EnvVars {
			railpackEnvVars[k] = v
		}
		// Override with RAILPACK_* commands if provided
		// See https://railpack.com/config/environment-variables
		if config.InstallCommand != "" {
			railpackEnvVars["RAILPACK_INSTALL_CMD"] = config.InstallCommand
			writeBuildLog("   📦 Install command: %s", config.InstallCommand)
		}
		if config.BuildCommand != "" {
			railpackEnvVars["RAILPACK_BUILD_CMD"] = config.BuildCommand
			writeBuildLog("   🔨 Build command: %s", config.BuildCommand)
		}
		if config.StartCommand != "" {
			railpackEnvVars["RAILPACK_START_CMD"] = config.StartCommand
			writeBuildLog("   🚀 Start command: %s", config.StartCommand)
		}
		
		envVars := getEnvAsStringSlice(railpackEnvVars)

		// Railpack requires BUILDKIT_HOST to be set for BuildKit builds
		if !isBuildxAvailable(ctx) {
			logger.Warn("[Static] Buildx not available, disabling BuildKit")
			envVars = append(envVars, "DOCKER_BUILDKIT=0")
		} else {
			startBuildkitContainer(ctx)
			envVars = append(envVars, "DOCKER_BUILDKIT=1")
			envVars = append(envVars, "BUILDKIT_HOST=docker-container://obiente-cloud-buildkit")
		}
		cmd.Env = envVars

		// Ensure railpack logs are captured in build logs
		writeBuildLog("📝 Railpack build logs will be shown below...")
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

		// Run railpack and wait for completion (cmd.Run() blocks until done)
		if err := cmd.Run(); err != nil {
			writeBuildLog("❌ Railpack build failed: %v", err)
			return &BuildResult{Success: false, Error: fmt.Errorf("railpack build failed: %w", err)}, nil
		}

		// Verify the image was actually created (suppress output to avoid polluting logs)
		verifyCmd := exec.CommandContext(ctx, "docker", "image", "inspect", railpackImageName)
		verifyCmd.Stdout = nil
		verifyCmd.Stderr = nil
		if err := verifyCmd.Run(); err != nil {
			writeBuildLog("❌ Railpack image %s not found after build completion", railpackImageName)
			writeBuildLog("💡 This may indicate the build failed silently. Check railpack logs above for details.")
			return &BuildResult{Success: false, Error: fmt.Errorf("railpack build completed but image %s was not created", railpackImageName)}, nil
		}

		writeBuildLog("✅ Railpack build completed and image verified")
	}

	writeBuildLog("🐳 Creating multi-stage Dockerfile for static deployment...")

	// Step 2: Determine output directory path in the built Railpack image
	// First, verify the image exists
	verifyCmd := exec.CommandContext(ctx, "docker", "image", "inspect", railpackImageName)
	verifyOutput, verifyErr := verifyCmd.CombinedOutput()
	if verifyErr != nil {
		writeBuildLog("❌ Railpack image %s not found", railpackImageName)
		writeBuildLog("   Error: %v", verifyErr)
		if len(verifyOutput) > 0 {
			writeBuildLog("   Docker output: %s", strings.TrimSpace(string(verifyOutput)))
		}
		return &BuildResult{Success: false, Error: fmt.Errorf("railpack build completed but image %s was not created", railpackImageName)}, nil
	}

	// Determine output directory path in the built image
	outputDir := config.BuildOutputPath
	if outputDir == "" {
		writeBuildLog("   Auto-detecting output directory from repository...")
		// Auto-detect from repository
		outputDir = s.findOutputDir(buildDir)
		writeBuildLog("   Detected output directory: %s", outputDir)
		// If auto-detection fails, default to "dist"
		if outputDir == "" || outputDir == "public" {
			outputDir = "dist"
			writeBuildLog("   Using default output directory: dist")
		}
	} else {
		writeBuildLog("   Using configured output directory: %s", outputDir)
	}

	// Try to find the actual output directory in the built image
	// Check common locations in the Railpack image
	possibleOutputPaths := []string{
		"/app/" + outputDir,
		"/app/dist",
		"/app/build",
		"/app/public",
		"/app/out",
		"/app/.next/out", // Next.js static export
		"/app",
	}

	writeBuildLog("   Detecting output path in built image...")
	var detectedOutputPath string
	for _, path := range possibleOutputPaths {
		// Check if directory exists and has files
		checkCmd := exec.CommandContext(ctx, "docker", "run", "--rm", "--entrypoint", "sh", railpackImageName,
			"-c", fmt.Sprintf("test -d %s && (test -f %s/index.html || ls -A %s | head -1 > /dev/null) && echo %s", path, path, path, path))
		if output, err := checkCmd.Output(); err == nil {
			detectedPath := strings.TrimSpace(string(output))
			if detectedPath != "" && detectedPath == path {
				detectedOutputPath = path
				writeBuildLog("   ✅ Found output at: %s", path)
				break
			}
		}
	}

	// Fallback: search for index.html
	if detectedOutputPath == "" {
		writeBuildLog("   Searching for index.html in image...")
		findCmd := exec.CommandContext(ctx, "docker", "run", "--rm", "--entrypoint", "sh", railpackImageName,
			"-c", "find /app -type f -name 'index.html' 2>/dev/null | head -1")
		if output, err := findCmd.Output(); err == nil {
			indexPath := strings.TrimSpace(string(output))
			if indexPath != "" {
				detectedOutputPath = filepath.Dir(indexPath)
				writeBuildLog("   ✅ Found index.html at: %s (using parent directory)", indexPath)
			}
		}
	}

	// Final fallback: use /app
	if detectedOutputPath == "" {
		detectedOutputPath = "/app"
		writeBuildLog("   ⚠️  Using /app as fallback")
	}

	writeBuildLog("   📦 Source path in Railpack image: %s", detectedOutputPath)

	// Step 3: Create multi-stage Dockerfile that copies from the built Railpack image
	writeBuildLog("🐳 Creating multi-stage Nginx Dockerfile...")
	dockerfileContent := s.generateStaticDockerfile(railpackImageName, detectedOutputPath, config.NginxConfig)
	dockerfilePath := filepath.Join(buildDir, ".obiente.Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
		return &BuildResult{Success: false, Error: fmt.Errorf("failed to write Dockerfile: %w", err)}, nil
	}

	finalImageName := fmt.Sprintf("obiente/%s:%s", deployment.ID, deployment.Branch)

	// Build final minimal nginx image
	if err := buildDockerImage(ctx, buildDir, finalImageName, ".obiente.Dockerfile", config.LogWriter, config.LogWriterErr); err != nil {
		return &BuildResult{Success: false, Error: fmt.Errorf("docker build failed: %w", err)}, nil
	}
	
	writeBuildLog("✅ Created minimal nginx image")
	
	// Clean up Railpack image (optional - we could keep it for caching)
	// exec.CommandContext(ctx, "docker", "rmi", railpackImageName).Run()

	// Nginx always uses port 80
	port := 80

	return &BuildResult{
		ImageName: finalImageName,
		Port:      port,
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

	return "dist" // Default fallback
}

func (s *StaticStrategy) generateStaticDockerfile(sourceImage string, sourcePath string, nginxConfig string) string {
	// Multi-stage Dockerfile that copies files directly from the built Railpack image
	// This is much more reliable than extracting files manually
	
	// Always use nginx for static deployments
	// Use nginx with custom or default config
	nginxConfContent := nginxConfig
	if nginxConfContent == "" {
		// Default nginx config for static sites
		nginxConfContent = `server {
    listen 80;
    server_name _;
    root /usr/share/nginx/html;
    index index.html index.htm;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/x-javascript application/xml+rss application/json;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # SPA routing support - try files, then fallback to index.html
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Cache static assets
    location ~* \.(jpg|jpeg|png|gif|ico|css|js|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # Deny access to hidden files
    location ~ /\. {
        deny all;
    }
}`
	}

	// Escape the nginx config for embedding in shell command
	// Need to escape single quotes, backslashes, and newlines
	escapedConfig := strings.ReplaceAll(nginxConfContent, "\\", "\\\\")
	escapedConfig = strings.ReplaceAll(escapedConfig, "'", "'\\''")
	escapedConfig = strings.ReplaceAll(escapedConfig, "\n", "\\n")

	// Multi-stage Dockerfile:
	// Stage 1: Use the built Railpack image as source
	// Stage 2: Copy static files to minimal Nginx image
	return fmt.Sprintf(`# Stage 1: Source image (built with Railpack)
FROM %s AS builder

# Stage 2: Minimal Nginx image with only static files
FROM nginx:alpine
COPY --from=builder %s /usr/share/nginx/html
RUN echo '%s' > /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]`, sourceImage, sourcePath, escapedConfig)
}
