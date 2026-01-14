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
	// All volumes should go to /var/lib/obiente/volumes/{deploymentID}
	// This keeps Obiente Cloud volumes separate from Docker's default volumes
	var safeBaseDir string
	possibleDirs := []string{
		"/var/lib/obiente/volumes",
		"/var/obiente/tmp/obiente-volumes",
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
				// Add network connection for service discovery
				cs.addServiceToNetwork(service)
				// Add service name labels for container identification
				cs.addServiceNameLabel(service, serviceName)
			}
		}
	}

	// Sanitize volumes (top-level volumes definitions)
	if volumes, ok := compose["volumes"].(map[string]interface{}); ok {
		for volName, volData := range volumes {
			cs.sanitizeVolumeDefinition(volName, volData)
		}
	}

	// Inject network definition for service-to-service communication
	cs.injectNetworkDefinition(compose)

	// Marshal back to YAML
	sanitizedYaml, err := yaml.Marshal(compose)
	if err != nil {
		return "", fmt.Errorf("failed to marshal sanitized compose YAML: %w", err)
	}

	return string(sanitizedYaml), nil
}



// sanitizeService sanitizes a single service in the compose file
func (cs *ComposeSanitizer) sanitizeService(service map[string]interface{}, serviceName string) {
	// Sanitize environment variables to ensure proper formatting
	cs.sanitizeEnvironment(service)

	// Convert depends_on from map format to list format for Docker Swarm compatibility
	// Swarm's `docker stack deploy` only accepts list format, not map with conditions
	cs.convertDependsOnToList(service)
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

// convertDependsOnToList converts depends_on from map format to list format
// Docker Swarm's `docker stack deploy` only accepts list format, not map with conditions
// This ensures compatibility when deploying to Swarm clusters
func (cs *ComposeSanitizer) convertDependsOnToList(service map[string]interface{}) {
	if dependsOn, ok := service["depends_on"]; ok {
		// If it's a map (service conditions format), convert to list
		if depMap, ok := dependsOn.(map[string]interface{}); ok {
			// Extract service names from map keys
			serviceList := make([]interface{}, 0, len(depMap))
			for serviceName := range depMap {
				serviceList = append(serviceList, serviceName)
			}
			// Replace with list format
			service["depends_on"] = serviceList
		}
		// If it's already a list, leave it as-is
	}
}

// sanitizeEnvironment ensures environment variables are properly formatted for Docker Compose
// This fixes issues with:
// 1. Boolean values that need to be strings
// 2. Special characters like $ that need escaping
// 3. Type coercion issues when YAML is unmarshaled/marshaled
func (cs *ComposeSanitizer) sanitizeEnvironment(service map[string]interface{}) {
	env, ok := service["environment"]
	if !ok {
		return
	}

	// Handle both map and array formats for environment variables
	switch e := env.(type) {
	case map[string]interface{}:
		// Map format: { KEY: value }
		sanitizedEnv := make(map[string]interface{})
		for key, value := range e {
			sanitizedEnv[key] = cs.sanitizeEnvValue(value)
		}
		service["environment"] = sanitizedEnv

	case []interface{}:
		// Array format: [ "KEY=value", "KEY2=value2" ]
		sanitizedEnv := make([]interface{}, len(e))
		for i, item := range e {
			if str, ok := item.(string); ok {
				// Parse KEY=value format
				if idx := strings.Index(str, "="); idx > 0 {
					key := str[:idx]
					value := str[idx+1:]
					// Reconstruct with sanitized value
					sanitizedEnv[i] = fmt.Sprintf("%s=%s", key, cs.sanitizeEnvStringValue(value))
				} else {
					sanitizedEnv[i] = item
				}
			} else {
				sanitizedEnv[i] = item
			}
		}
		service["environment"] = sanitizedEnv
	}
}

// sanitizeEnvValue converts environment variable values to proper string format
// and escapes special characters for Docker Compose
func (cs *ComposeSanitizer) sanitizeEnvValue(value interface{}) string {
	var strValue string

	switch v := value.(type) {
	case bool:
		// Convert boolean to string
		if v {
			strValue = "true"
		} else {
			strValue = "false"
		}
	case int, int64, float64:
		// Convert numbers to strings
		strValue = fmt.Sprintf("%v", v)
	case string:
		strValue = v
	case nil:
		// Null values should remain as empty strings
		return ""
	default:
		// For any other type, convert to string
		strValue = fmt.Sprintf("%v", v)
	}

	return cs.sanitizeEnvStringValue(strValue)
}

// sanitizeEnvStringValue escapes special characters in environment variable values
func (cs *ComposeSanitizer) sanitizeEnvStringValue(value string) string {
	// Escape $ characters for Docker Compose variable interpolation
	// In Docker Compose, $$ is used to represent a literal $
	value = strings.ReplaceAll(value, "$", "$$")

	return value
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
		target := v["target"]
		if target == nil {
			target = v["bind"]
		}
		if target == nil {
			return nil // Invalid volume spec - no target
		}

		// Check volume type
		volType, _ := v["type"].(string)

		// If it has a source, check if it's a bind mount (absolute path) or named volume
		if source, ok := v["source"].(string); ok {
			if strings.HasPrefix(source, "/") {
				// Bind mount with absolute path - sanitize it
				sanitizedSource := cs.sanitizeHostPath(source, serviceName)
				return map[string]interface{}{
					"type":   "bind",
					"source": sanitizedSource,
					"target": target,
				}
			} else {
				// Named volume - convert to bind mount in /var/lib/obiente
				obienteVolumePath := filepath.Join("/var/lib/obiente/volumes", cs.deploymentID, source)
				os.MkdirAll(obienteVolumePath, 0755)
				return map[string]interface{}{
					"type":   "bind",
					"source": obienteVolumePath,
					"target": target,
				}
			}
		}

		// Check if it's explicitly a named volume type (without source, just name reference)
		if volType == "volume" {
			// This is a named volume reference - we need to check if there's a name
			// In compose, this might be referenced by service name or explicit name
			// For now, we'll handle it based on the volume definition context
			// This case is handled in sanitizeVolumeDefinition
			return vol
		}

		// No source, no explicit type - could be a simple named volume reference
		// This case should be handled in string parsing below
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
			// Named volume - convert to bind mount in /var/lib/obiente
			// Structure: /var/lib/obiente/volumes/{deploymentID}/{volumeName}
			volumeName := hostPath
			obienteVolumePath := filepath.Join("/var/lib/obiente/volumes", cs.deploymentID, volumeName)
			// Ensure directory exists
			os.MkdirAll(obienteVolumePath, 0755)
			// Return as bind mount
			return fmt.Sprintf("%s:%s", obienteVolumePath, containerPath)
		}

		// It's a bind mount - sanitize host path
		sanitizedHostPath := cs.sanitizeHostPath(hostPath, serviceName)

		// Return as string format
		return fmt.Sprintf("%s:%s", sanitizedHostPath, containerPath)
	}

	// Not a bind mount string, likely a named volume reference
	// If it looks like a named volume (no path separators, simple name), convert to bind mount
	if volStr != "" && !strings.Contains(volStr, "/") && !strings.Contains(volStr, ":") {
		// This is a named volume - convert to bind mount
		obienteVolumePath := filepath.Join("/var/lib/obiente/volumes", cs.deploymentID, volStr)
		os.MkdirAll(obienteVolumePath, 0755)
		// Return as bind mount with default container path
		return fmt.Sprintf("%s:/data", obienteVolumePath)
	}

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
	// Convert named volume definitions to bind mounts pointing to /var/lib/obiente
	// This ensures all volumes are stored in Obiente's directory structure
	if volMap, ok := volData.(map[string]interface{}); ok {
		// If it's an empty map or only has driver_opts, convert to bind mount
		if len(volMap) == 0 || (len(volMap) == 1 && volMap["driver_opts"] != nil) {
			// This is a named volume - convert to bind mount specification
			obienteVolumePath := filepath.Join("/var/lib/obiente/volumes", cs.deploymentID, volName)
			os.MkdirAll(obienteVolumePath, 0755)

			// Replace with bind mount configuration
			// Note: We can't fully represent bind mounts in top-level volumes,
			// but we'll ensure the directory exists and remove the volume definition
			// The actual bind mount will be created in sanitizeVolumeBinding
			delete(volMap, "driver")
			delete(volMap, "driver_opts")
		} else {
			// Handle driver_opts with device/bind mounts
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

// addServiceToNetwork adds a service to the deployment-specific network for isolation
// Services are ONLY added to their private deployment network here
// The obiente-network is added LATER by addTraefikNetworkToRoutedServices() only for services with routing
func (cs *ComposeSanitizer) addServiceToNetwork(service map[string]interface{}) {
	// Create networks section if it doesn't exist
	if _, ok := service["networks"]; !ok {
		// Connect ONLY to deployment-{id} - Private network for inter-service communication
		// This ensures complete isolation from other deployments by default
		// Traefik network access is granted separately based on routing configuration
		deploymentNetworkName := fmt.Sprintf("deployment-%s", cs.deploymentID)
		service["networks"] = []string{deploymentNetworkName}
	}
}

// injectNetworkDefinition adds a top-level networks section for the deployment
// Creates the deployment-specific network (for isolation) while referencing existing obiente-network
func (cs *ComposeSanitizer) injectNetworkDefinition(compose map[string]interface{}) {
	// Deployment network: isolated network for this deployment's services
	deploymentNetworkName := fmt.Sprintf("deployment-%s", cs.deploymentID)

	// Create networks section if it doesn't exist
	if _, ok := compose["networks"]; !ok {
		compose["networks"] = make(map[string]interface{})
	}

	networks, ok := compose["networks"].(map[string]interface{})
	if !ok {
		// If networks exists but is not a map, recreate it
		networks = make(map[string]interface{})
		compose["networks"] = networks
	}

	// Add the deployment network with external=true (created by ensureDeploymentNetwork)
	// This is the PRIVATE network for inter-service communication within this deployment
	// Note: Network has internet access (not internal) so containers can pull images, install packages, etc.
	networks[deploymentNetworkName] = map[string]interface{}{
		"external": true,
	}

	// NOTE: obiente-network is NOT added here by default for security
	// It is added selectively by addTraefikNetworkToRoutedServices() in deployment_manager_config.go
	// only for services that have routing rules configured
	// This ensures complete isolation - user containers cannot reach other deployments
}

// addServiceNameLabel adds a label to the service identifying its name
// This allows containers to be matched with their service names for display and querying
func (cs *ComposeSanitizer) addServiceNameLabel(service map[string]interface{}, serviceName string) {
	// Create labels section if it doesn't exist
	if _, ok := service["labels"]; !ok {
		service["labels"] = make(map[string]interface{})
	}

	labels, ok := service["labels"].(map[string]interface{})
	if !ok {
		// If labels exists but is not a map, recreate it
		labels = make(map[string]interface{})
		service["labels"] = labels
	}

	// Add service name label for container identification
	// This label is used by the frontend to group containers by service
	labels["cloud.obiente.service_name"] = serviceName
}

// GetSafeBaseDir returns the safe base directory for this deployment's volumes
func (cs *ComposeSanitizer) GetSafeBaseDir() string {
	return cs.safeBaseDir
}
