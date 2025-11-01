package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeSanitizer sanitizes Docker Compose YAML to prevent security issues
type ComposeSanitizer struct {
	deploymentID string
	safeBaseDir  string // Base directory where user volumes should be stored
}

// NewComposeSanitizer creates a new compose sanitizer for a deployment
func NewComposeSanitizer(deploymentID string) *ComposeSanitizer {
	// Determine safe base directory for user volumes
	// Try multiple locations, prefer persistent storage
	var safeBaseDir string
	possibleDirs := []string{
		"/var/lib/obiente/volumes",
		"/var/lib/obiente/deployments",
		"/tmp/obiente-volumes",
	}

	for _, baseDir := range possibleDirs {
		testDir := filepath.Join(baseDir, deploymentID)
		if err := os.MkdirAll(testDir, 0755); err == nil {
			// Verify we can write to it
			testFile := filepath.Join(testDir, ".test")
			if err := os.WriteFile(testFile, []byte("test"), 0644); err == nil {
				os.Remove(testFile)
				safeBaseDir = testDir
				break
			}
		}
	}

	if safeBaseDir == "" {
		// Fallback to temp directory if all else fails
		safeBaseDir = filepath.Join(os.TempDir(), "obiente-volumes", deploymentID)
		os.MkdirAll(safeBaseDir, 0755)
	}

	return &ComposeSanitizer{
		deploymentID: deploymentID,
		safeBaseDir:  safeBaseDir,
	}
}

// SanitizeComposeYAML sanitizes a Docker Compose YAML string
// It transforms volumes and removes host port bindings
func (cs *ComposeSanitizer) SanitizeComposeYAML(composeYaml string) (string, error) {
	// Parse YAML
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(composeYaml), &compose); err != nil {
		return "", fmt.Errorf("failed to parse compose YAML: %w", err)
	}

	// Sanitize services
	if services, ok := compose["services"].(map[string]interface{}); ok {
		for serviceName, serviceData := range services {
			if service, ok := serviceData.(map[string]interface{}); ok {
				cs.sanitizeService(service, serviceName)
			}
		}
	}

	// Sanitize volumes (top-level volumes definitions)
	if volumes, ok := compose["volumes"].(map[string]interface{}); ok {
		for volName, volData := range volumes {
			cs.sanitizeVolumeDefinition(volName, volData)
		}
	}

	// Marshal back to YAML
	sanitizedYaml, err := yaml.Marshal(compose)
	if err != nil {
		return "", fmt.Errorf("failed to marshal sanitized compose YAML: %w", err)
	}

	return string(sanitizedYaml), nil
}

// sanitizeService sanitizes a single service in the compose file
func (cs *ComposeSanitizer) sanitizeService(service map[string]interface{}, serviceName string) {
	// Sanitize volumes
	if volumes, ok := service["volumes"].([]interface{}); ok {
		sanitizedVolumes := []interface{}{}
		for _, vol := range volumes {
			if sanitized := cs.sanitizeVolumeBinding(vol, serviceName); sanitized != nil {
				sanitizedVolumes = append(sanitizedVolumes, sanitized)
			}
		}
		service["volumes"] = sanitizedVolumes
	}

	// Sanitize ports - remove host port bindings, keep only container ports
	if ports, ok := service["ports"].([]interface{}); ok {
		sanitizedPorts := []interface{}{}
		for _, port := range ports {
			if sanitized := cs.sanitizePortBinding(port); sanitized != nil {
				sanitizedPorts = append(sanitizedPorts, sanitized)
			}
		}
		// If we have ports, keep them but without host bindings
		// Otherwise, remove the ports field entirely to avoid conflicts
		if len(sanitizedPorts) > 0 {
			service["ports"] = sanitizedPorts
		} else {
			delete(service, "ports")
		}
	}

	// Sanitize network_mode to prevent host network
	if networkMode, ok := service["network_mode"].(string); ok {
		if networkMode == "host" {
			// Remove host network mode for security
			delete(service, "network_mode")
		}
	}

	// Sanitize privileged mode
	if privileged, ok := service["privileged"].(bool); ok && privileged {
		// Remove privileged mode for security
		delete(service, "privileged")
	}

	// Sanitize cap_add and cap_drop for extra security
	// We'll be conservative and remove dangerous capabilities
	if capAdd, ok := service["cap_add"].([]interface{}); ok {
		dangerousCaps := map[string]bool{
			"SYS_ADMIN":    true,
			"NET_ADMIN":    true,
			"SYS_MODULE":   true,
			"SYS_RAWIO":    true,
			"SYS_TIME":     true,
			"MKNOD":        true,
			"DAC_OVERRIDE": true,
		}
		filteredCaps := []interface{}{}
		for _, cap := range capAdd {
			if capStr, ok := cap.(string); ok {
				if !dangerousCaps[strings.ToUpper(capStr)] {
					filteredCaps = append(filteredCaps, cap)
				}
			}
		}
		if len(filteredCaps) > 0 {
			service["cap_add"] = filteredCaps
		} else {
			delete(service, "cap_add")
		}
	}
}

// sanitizeVolumeBinding sanitizes a volume binding
// Transforms host paths to safe user directories
func (cs *ComposeSanitizer) sanitizeVolumeBinding(vol interface{}, serviceName string) interface{} {
	var volStr string

	switch v := vol.(type) {
	case string:
		volStr = v
	case map[string]interface{}:
		// Handle named volume or bind mount object format
		if source, ok := v["source"].(string); ok {
			// Bind mount - transform source path
			target := v["target"]
			if target == nil {
				target = v["bind"]
			}
			if target == nil {
				return nil // Invalid volume spec
			}

			// Transform source path to safe directory
			sanitizedSource := cs.sanitizeHostPath(source, serviceName)

			return map[string]interface{}{
				"type":   "bind",
				"source": sanitizedSource,
				"target": target,
			}
		}
		// Named volume - keep as is (it's managed by Docker)
		return vol
	default:
		return vol
	}

	// Parse string format: "host_path:container_path" or "/host:/container" or "named_volume:container_path"
	if strings.Contains(volStr, ":") {
		parts := strings.SplitN(volStr, ":", 2)
		if len(parts) != 2 {
			return vol
		}

		hostPath := strings.TrimSpace(parts[0])
		containerPath := strings.TrimSpace(parts[1])

		// Check if it's a named volume (no leading slash, not an absolute path)
		if !strings.HasPrefix(hostPath, "/") && !strings.HasPrefix(hostPath, "~") && !filepath.IsAbs(hostPath) {
			// Named volume - keep as is
			return vol
		}

		// It's a bind mount - sanitize host path
		sanitizedHostPath := cs.sanitizeHostPath(hostPath, serviceName)

		// Return as string format
		return fmt.Sprintf("%s:%s", sanitizedHostPath, containerPath)
	}

	// Not a bind mount, likely a named volume - keep as is
	return vol
}

// sanitizeHostPath transforms a host path to a safe user directory
func (cs *ComposeSanitizer) sanitizeHostPath(hostPath string, serviceName string) string {
	// Clean the path to prevent directory traversal
	hostPath = filepath.Clean(hostPath)

	// Remove leading slash or ~ to get relative path component
	relativePath := strings.TrimPrefix(hostPath, "/")
	relativePath = strings.TrimPrefix(relativePath, "~/")
	relativePath = strings.TrimPrefix(relativePath, "~")

	// If it's an absolute path, extract the basename and relative components
	if filepath.IsAbs(hostPath) {
		// Extract meaningful parts while preventing traversal
		parts := strings.Split(relativePath, string(filepath.Separator))
		safeParts := []string{}
		for _, part := range parts {
			if part != "" && part != "." && part != ".." {
				safeParts = append(safeParts, part)
			}
		}
		relativePath = strings.Join(safeParts, string(filepath.Separator))
	}

	// If relative path is empty or just dots, use a default name
	if relativePath == "" || strings.Trim(relativePath, ".") == "" {
		relativePath = "data"
	}

	// Create safe path under user's directory
	// Structure: {safeBaseDir}/{serviceName}/{sanitized_path}
	safePath := filepath.Join(cs.safeBaseDir, serviceName, relativePath)

	// Ensure directory exists
	os.MkdirAll(safePath, 0755)

	return safePath
}

// sanitizeVolumeDefinition sanitizes top-level volume definitions
func (cs *ComposeSanitizer) sanitizeVolumeDefinition(volName string, volData interface{}) {
	// Most volume definitions are named volumes managed by Docker, which is safe
	// We mainly need to handle driver_opts with device/bind mounts
	if volMap, ok := volData.(map[string]interface{}); ok {
		if driverOpts, ok := volMap["driver_opts"].(map[string]interface{}); ok {
			// Check for device or type=bind options
			if device, ok := driverOpts["device"].(string); ok {
				// Transform device path to safe directory
				sanitizedDevice := cs.sanitizeHostPath(device, "volume-"+volName)
				driverOpts["device"] = sanitizedDevice
			}
			if volType, ok := driverOpts["type"].(string); ok && volType == "bind" {
				// Ensure bind mounts use sanitized paths
				if o, ok := driverOpts["o"].(string); ok {
					// Parse and sanitize bind options
					driverOpts["o"] = cs.sanitizeBindOptions(o, volName)
				}
			}
		}
	}
}

// sanitizeBindOptions sanitizes bind mount options
func (cs *ComposeSanitizer) sanitizeBindOptions(options string, volName string) string {
	// Parse bind options like "bind" or "bind,ro"
	parts := strings.Split(options, ",")
	sanitizedParts := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "bind" || part == "ro" || part == "rw" {
			sanitizedParts = append(sanitizedParts, part)
		}
	}
	return strings.Join(sanitizedParts, ",")
}

// sanitizePortBinding sanitizes port bindings to remove host ports
// Returns container port only (no host binding) in Docker Compose format
func (cs *ComposeSanitizer) sanitizePortBinding(port interface{}) interface{} {
	var portStr string

	switch v := port.(type) {
	case string:
		portStr = v
	case map[string]interface{}:
		// Port mapping object format - keep only target (container) port
		if target, ok := v["target"].(int); ok {
			// Return in short format (just container port)
			return fmt.Sprintf("%d", target)
		}
		if published, ok := v["published"].(int); ok {
			// Use published as container port if target not specified
			return fmt.Sprintf("%d", published)
		}
		return nil
	default:
		return nil
	}

	// Parse string format: "host_port:container_port" or "host_port:container_port/protocol" or "container_port" or "container_port/protocol"
	if strings.Contains(portStr, ":") {
		parts := strings.SplitN(portStr, ":", 2)
		if len(parts) == 2 {
			// Extract container port (may include protocol like "8080/tcp")
			containerPart := strings.TrimSpace(parts[1])
			if strings.Contains(containerPart, "/") {
				// Has protocol specified
				portParts := strings.Split(containerPart, "/")
				return strings.TrimSpace(portParts[0]) // Return just port number
			}
			return containerPart // Return container port
		}
	}

	// Already just container port - preserve protocol if specified
	if strings.Contains(portStr, "/") {
		// Format like "8080/tcp" - keep as is (no host binding)
		return portStr
	}

	// Simple port number - return as is
	return portStr
}

// GetSafeBaseDir returns the safe base directory for this deployment's volumes
func (cs *ComposeSanitizer) GetSafeBaseDir() string {
	return cs.safeBaseDir
}
