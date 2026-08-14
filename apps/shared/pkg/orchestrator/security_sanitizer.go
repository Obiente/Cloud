package orchestrator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v3"
)

// ComposeSanitizer sanitizes Docker Compose YAML to prevent security issues
type ComposeSanitizer struct {
	deploymentID         string
	safeBaseDir          string // Base directory where user volumes should be stored
	swarmVolumeNodeID    string // Node that owns deployment-local bind roots in Swarm mode
	volumeRootStates     map[string]volumeRootState
	preparedVolumeRoots  []string
	preparedVolumeSet    map[string]struct{}
	preparedWritable     map[string]bool
	deferredReadOnly     []string
	deferredReadOnlySet  map[string]struct{}
	legacyRelativePaths  map[string]string
	legacyProjectRoot    bool
	usesLocalBind        bool
	persistedComposeYaml string
}

const DefaultMaxUntrustedComposeServices = 8

// Keep deployment-relative bind sources in a namespace that cannot be mistaken
// for an ordinary Compose service directory. This preserves sharing when two
// services mount the same relative source while absolute sources remain
// isolated by service as before.
const (
	relativeComposeBindScope   = "@obiente-relative-binds"
	relativeComposeProjectRoot = "@project-root"
)

type UntrustedComposeLimits struct {
	MaxServices      int
	TotalMemoryBytes int64
	TotalCPUShares   int64
}

// NewComposeSanitizer creates a new compose sanitizer for a deployment
func NewComposeSanitizer(deploymentID string) *ComposeSanitizer {
	if deploymentID == "" || sanitizeVolumeName(deploymentID) != deploymentID {
		// SanitizeComposeYAML will reject the invalid identifier before preparing
		// any volume. Avoid joining it into host paths here.
		return &ComposeSanitizer{deploymentID: deploymentID}
	}
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
func (cs *ComposeSanitizer) SanitizeComposeYAML(composeYaml string) (sanitizedResult string, err error) {
	if cs.deploymentID == "" || sanitizeVolumeName(cs.deploymentID) != cs.deploymentID {
		return "", fmt.Errorf("invalid deployment identifier for Compose volume paths")
	}
	cs.volumeRootStates = make(map[string]volumeRootState)
	cs.preparedVolumeRoots = nil
	cs.preparedVolumeSet = make(map[string]struct{})
	cs.preparedWritable = make(map[string]bool)
	cs.deferredReadOnly = nil
	cs.deferredReadOnlySet = make(map[string]struct{})
	defer func() {
		if err == nil || len(cs.preparedVolumeRoots) == 0 {
			return
		}
		if rollbackErr := cs.rollbackVolumePreparation(); rollbackErr != nil {
			err = fmt.Errorf("%w; restore previous volume permissions: %v", err, rollbackErr)
		}
	}()

	// Parse YAML
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(composeYaml), &compose); err != nil {
		return "", fmt.Errorf("failed to parse compose YAML: %w", err)
	}

	// docker stack deploy does not accept Compose's top-level project name.
	// Obiente controls the stack/project name with the deployment ID instead.
	delete(compose, "name")
	if err := cs.snapshotLegacyRelativeBindPaths(compose); err != nil {
		return "", err
	}

	// Sanitize services
	if services, ok := compose["services"].(map[string]interface{}); ok {
		for serviceName, serviceData := range services {
			if service, ok := serviceData.(map[string]interface{}); ok {
				if err := cs.sanitizeService(service, serviceName); err != nil {
					return "", fmt.Errorf("sanitize service %q: %w", serviceName, err)
				}
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
			if err := cs.sanitizeVolumeDefinition(volName, volData); err != nil {
				return "", fmt.Errorf("sanitize volume %q: %w", volName, err)
			}
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

func composeVolumeSource(volume interface{}) (source string, relativeBind bool, namedVolume bool) {
	switch typed := volume.(type) {
	case string:
		if !strings.Contains(typed, ":") {
			return "", false, false
		}
		source = strings.TrimSpace(strings.SplitN(typed, ":", 2)[0])
		if source != "" && sanitizeVolumeName(source) == source {
			return source, false, true
		}
	case map[string]interface{}:
		var ok bool
		source, ok = typed["source"].(string)
		if !ok {
			return "", false, false
		}
		source = strings.TrimSpace(source)
		volumeType, _ := typed["type"].(string)
		if strings.EqualFold(strings.TrimSpace(volumeType), "volume") {
			return source, false, source != ""
		}
		if !strings.EqualFold(strings.TrimSpace(volumeType), "bind") && !filepath.IsAbs(source) && !strings.HasPrefix(source, "~") && !strings.ContainsRune(source, filepath.Separator) {
			return "", false, false
		}
	default:
		return "", false, false
	}
	if source == "" || filepath.IsAbs(source) || strings.HasPrefix(source, "~") {
		return source, false, false
	}
	return filepath.Clean(source), true, false
}

func composeVolumeTarget(volume interface{}) string {
	switch typed := volume.(type) {
	case string:
		parts := strings.SplitN(typed, ":", 3)
		if len(parts) >= 2 && strings.TrimSpace(parts[1]) != "" {
			return filepath.Clean(strings.TrimSpace(parts[1]))
		}
	case map[string]interface{}:
		if target, ok := typed["target"].(string); ok && strings.TrimSpace(target) != "" {
			return filepath.Clean(strings.TrimSpace(target))
		}
		if target, ok := typed["bind"].(string); ok && strings.TrimSpace(target) != "" {
			return filepath.Clean(strings.TrimSpace(target))
		}
	}
	return ""
}

func legacyRelativeBindKey(serviceName, source string) string {
	return serviceName + "\x00" + filepath.Clean(source)
}

func expectedManagedRelativeBindSources(safeBaseDir, serviceName, source string) (map[string]struct{}, error) {
	if sanitizeVolumeName(serviceName) != serviceName {
		return nil, fmt.Errorf("invalid Compose service name %q for managed volume paths", serviceName)
	}
	cleaned := filepath.Clean(source)
	if filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "~") {
		return nil, fmt.Errorf("relative volume source %q is not relative", source)
	}
	parts := strings.Split(cleaned, string(filepath.Separator))
	safeParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == ".." {
			return nil, fmt.Errorf("relative volume source %q contains path traversal", source)
		}
		if part != "" && part != "." {
			safeParts = append(safeParts, part)
		}
	}
	relativePath := filepath.Join(safeParts...)
	legacyLeaf := relativePath
	if legacyLeaf == "" {
		legacyLeaf = "data"
	}
	namespacedPath := filepath.Join(safeBaseDir, relativeComposeBindScope, relativeComposeProjectRoot, relativePath)
	legacyServicePath := filepath.Join(safeBaseDir, serviceName, legacyLeaf)
	legacyProjectPath := filepath.Join(safeBaseDir, relativePath)
	return map[string]struct{}{
		filepath.Clean(namespacedPath):    {},
		filepath.Clean(legacyServicePath): {},
		filepath.Clean(legacyProjectPath): {},
	}, nil
}

func (cs *ComposeSanitizer) snapshotPersistedRelativeBindPaths(compose map[string]interface{}) error {
	if strings.TrimSpace(cs.persistedComposeYaml) == "" {
		return nil
	}
	var persisted map[string]interface{}
	if err := yaml.Unmarshal([]byte(cs.persistedComposeYaml), &persisted); err != nil {
		return fmt.Errorf("parse persisted sanitized Compose file: %w", err)
	}
	currentServices, _ := compose["services"].(map[string]interface{})
	persistedServices, _ := persisted["services"].(map[string]interface{})
	for serviceName, currentData := range currentServices {
		currentService, _ := currentData.(map[string]interface{})
		persistedService, _ := persistedServices[serviceName].(map[string]interface{})
		persistedVolumes, _ := persistedService["volumes"].([]interface{})
		currentVolumes, _ := currentService["volumes"].([]interface{})
		for _, currentVolume := range currentVolumes {
			currentSource, relativeBind, _ := composeVolumeSource(currentVolume)
			currentTarget := composeVolumeTarget(currentVolume)
			if !relativeBind || currentTarget == "" {
				continue
			}
			expectedSources, err := expectedManagedRelativeBindSources(cs.safeBaseDir, serviceName, currentSource)
			if err != nil {
				return err
			}
			for _, persistedVolume := range persistedVolumes {
				if composeVolumeTarget(persistedVolume) != currentTarget {
					continue
				}
				persistedSource, _, namedVolume := composeVolumeSource(persistedVolume)
				if namedVolume || !filepath.IsAbs(persistedSource) {
					continue
				}
				persistedSource = filepath.Clean(persistedSource)
				if _, expected := expectedSources[persistedSource]; !expected {
					continue
				}
				rel, err := filepath.Rel(cs.safeBaseDir, persistedSource)
				if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
					return fmt.Errorf("persisted relative bind source %s escapes the deployment root", persistedSource)
				}
				if _, err := os.Lstat(persistedSource); err == nil {
					cs.legacyRelativePaths[legacyRelativeBindKey(serviceName, currentSource)] = persistedSource
				} else if !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("inspect persisted relative bind directory %s: %w", persistedSource, err)
				}
				break
			}
		}
	}
	return nil
}

func (cs *ComposeSanitizer) snapshotLegacyRelativeBindPaths(compose map[string]interface{}) error {
	cs.legacyRelativePaths = make(map[string]string)
	if err := cs.snapshotPersistedRelativeBindPaths(compose); err != nil {
		return err
	}
	namedSources := make(map[string]struct{})
	if volumes, ok := compose["volumes"].(map[string]interface{}); ok {
		for name := range volumes {
			namedSources[name] = struct{}{}
		}
	}
	var relativeSources []string
	if services, ok := compose["services"].(map[string]interface{}); ok {
		for _, serviceData := range services {
			service, _ := serviceData.(map[string]interface{})
			volumes, _ := service["volumes"].([]interface{})
			for _, volume := range volumes {
				source, relativeBind, namedVolume := composeVolumeSource(volume)
				if namedVolume {
					namedSources[source] = struct{}{}
				}
				if relativeBind {
					relativeSources = append(relativeSources, source)
				}
			}
		}
	}
	for _, source := range relativeSources {
		if cs.legacyProjectRoot {
			if source == "." {
				cs.legacyRelativePaths[source] = cs.safeBaseDir
			} else {
				cs.legacyRelativePaths[source] = filepath.Join(cs.safeBaseDir, source)
			}
			continue
		}
		if source == "." {
			continue
		}
		firstComponent := strings.Split(source, string(filepath.Separator))[0]
		if _, named := namedSources[firstComponent]; named {
			continue
		}
		candidate := filepath.Join(cs.safeBaseDir, source)
		_, err := os.Lstat(candidate)
		switch {
		case err == nil:
			cs.legacyRelativePaths[source] = candidate
		case errors.Is(err, os.ErrNotExist):
			continue
		default:
			return fmt.Errorf("inspect legacy relative volume directory %s: %w", candidate, err)
		}
	}
	return nil
}

func persistedComposeProvesLegacyProjectRoot(composeYaml, safeBaseDir string) (bool, error) {
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(composeYaml), &compose); err != nil {
		return false, fmt.Errorf("parse persisted Compose metadata: %w", err)
	}
	services, _ := compose["services"].(map[string]interface{})
	wantSource := filepath.Clean(safeBaseDir)
	for _, serviceData := range services {
		service, _ := serviceData.(map[string]interface{})
		volumes, _ := service["volumes"].([]interface{})
		for _, volume := range volumes {
			source, _, namedVolume := composeVolumeSource(volume)
			if !namedVolume && filepath.IsAbs(source) && filepath.Clean(source) == wantSource {
				return true, nil
			}
		}
	}
	return false, nil
}

func persistedComposeLegacyProjectRoot(composeYaml, deploymentID string) (string, bool, error) {
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(composeYaml), &compose); err != nil {
		return "", false, fmt.Errorf("parse persisted Compose metadata: %w", err)
	}
	allowedRoots := make(map[string]struct{})
	for _, root := range managedDeploymentVolumeRoots(deploymentID) {
		allowedRoots[root] = struct{}{}
	}
	services, _ := compose["services"].(map[string]interface{})
	var recordedRoot string
	for _, serviceData := range services {
		service, _ := serviceData.(map[string]interface{})
		volumes, _ := service["volumes"].([]interface{})
		for _, volume := range volumes {
			source, _, namedVolume := composeVolumeSource(volume)
			candidate := filepath.Clean(source)
			if namedVolume || !filepath.IsAbs(candidate) {
				continue
			}
			if _, allowed := allowedRoots[candidate]; !allowed {
				continue
			}
			if recordedRoot != "" && recordedRoot != candidate {
				return "", false, fmt.Errorf("persisted Compose file references conflicting managed project roots %q and %q", recordedRoot, candidate)
			}
			recordedRoot = candidate
		}
	}
	return recordedRoot, recordedRoot != "", nil
}

// persistedComposeManagedVolumeRoot recovers the selected deployment root
// from any previously sanitized bind. In particular, named volumes are stored
// one level beneath this root, so checking only for a bind of the root itself
// would silently remap their data when a different fallback becomes writable.
func persistedComposeManagedVolumeRoot(composeYaml, deploymentID string) (string, bool, error) {
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(composeYaml), &compose); err != nil {
		return "", false, fmt.Errorf("parse persisted Compose metadata: %w", err)
	}
	services, _ := compose["services"].(map[string]interface{})
	var recordedRoot string
	for _, serviceData := range services {
		service, _ := serviceData.(map[string]interface{})
		volumes, _ := service["volumes"].([]interface{})
		for _, volume := range volumes {
			source, _, namedVolume := composeVolumeSource(volume)
			candidate := filepath.Clean(source)
			if namedVolume || !filepath.IsAbs(candidate) {
				continue
			}
			for _, managedRoot := range managedDeploymentVolumeRoots(deploymentID) {
				if candidate != managedRoot && !strings.HasPrefix(candidate, managedRoot+string(filepath.Separator)) {
					continue
				}
				if recordedRoot != "" && recordedRoot != managedRoot {
					return "", false, fmt.Errorf("persisted Compose file references conflicting managed volume roots %q and %q", recordedRoot, managedRoot)
				}
				recordedRoot = managedRoot
				break
			}
		}
	}
	return recordedRoot, recordedRoot != "", nil
}

// SanitizeUntrustedComposeYAML removes host- and cluster-control options from
// repository Compose files before they are allowed into the normal deployment
// sanitizer. It is used for pull request previews, where every byte of the
// Compose file must be treated as attacker controlled.
func (cs *ComposeSanitizer) SanitizeUntrustedComposeYAML(composeYaml string) (string, error) {
	return cs.SanitizeUntrustedComposeYAMLWithLimits(composeYaml, UntrustedComposeLimits{
		MaxServices:      DefaultMaxUntrustedComposeServices,
		TotalMemoryBytes: 512 * 1024 * 1024,
		TotalCPUShares:   256,
	})
}

// SanitizeUntrustedComposeYAMLWithLimits also divides a preview's reserved
// runtime budget across all attacker-controlled services. This keeps the sum
// of service limits within the quota reservation made for the preview row.
func (cs *ComposeSanitizer) SanitizeUntrustedComposeYAMLWithLimits(composeYaml string, budget UntrustedComposeLimits) (string, error) {
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(composeYaml), &compose); err != nil {
		return "", fmt.Errorf("failed to parse untrusted compose YAML: %w", err)
	}
	// Compose interpolates both $NAME and ${NAME} after YAML decoding. Reject
	// every dollar marker in untrusted input, including YAML-escaped markers and
	// $$ forms, so the deployments-service environment can never be consulted.
	if containsComposeInterpolationMarker(compose) {
		return "", fmt.Errorf("environment interpolation is not allowed in pull request Compose files")
	}
	services, ok := compose["services"].(map[string]interface{})
	if !ok || len(services) == 0 {
		return "", fmt.Errorf("pull request Compose file must define at least one service")
	}
	if budget.MaxServices <= 0 {
		budget.MaxServices = DefaultMaxUntrustedComposeServices
	}
	if len(services) > budget.MaxServices {
		return "", fmt.Errorf("pull request Compose file defines %d services; at most %d are allowed", len(services), budget.MaxServices)
	}
	if budget.TotalMemoryBytes <= 0 || budget.TotalCPUShares <= 0 {
		return "", fmt.Errorf("pull request Compose resource budget must include positive memory and CPU limits")
	}
	perServiceMemory := budget.TotalMemoryBytes / int64(len(services))
	perServiceCPUShares := budget.TotalCPUShares / int64(len(services))
	if perServiceMemory <= 0 || perServiceCPUShares <= 0 {
		return "", fmt.Errorf("pull request Compose resource budget is too small for %d services", len(services))
	}

	dangerousServiceKeys := []string{
		"build", "cap_add", "cap_drop", "cgroup", "cgroup_parent", "cpu_count",
		"cpu_percent", "cpu_shares", "cpus",
		"configs", "container_name", "credential_spec", "deploy", "develop",
		"device_cgroup_rules", "devices", "env_file", "external_links",
		"extra_hosts", "ipc", "isolation", "labels", "links", "network_mode",
		"networks", "oom_kill_disable", "oom_score_adj", "pid", "privileged", "scale",
		"mem_limit", "mem_reservation", "memswap_limit", "runtime", "secrets", "security_opt", "shm_size", "sysctls", "ulimits",
		"userns_mode", "uts", "volumes", "volumes_from",
	}
	for serviceName, serviceData := range services {
		service, ok := serviceData.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("service %q has an invalid definition", serviceName)
		}
		if _, buildsFromRepository := service["build"]; buildsFromRepository {
			return "", fmt.Errorf("service %q uses build, which is not allowed in pull request Compose previews; publish an image or use the Dockerfile strategy", serviceName)
		}
		for _, key := range dangerousServiceKeys {
			delete(service, key)
		}
		memoryLimit := fmt.Sprintf("%dB", perServiceMemory)
		cpuLimit := fmt.Sprintf("%.6f", float64(perServiceCPUShares)/1024.0)
		service["deploy"] = map[string]interface{}{
			"resources": map[string]interface{}{
				"limits": map[string]interface{}{"memory": memoryLimit, "cpus": cpuLimit},
			},
		}
		service["mem_limit"] = memoryLimit
		service["cpus"] = cpuLimit
	}

	// Only services and the optional version marker are carried forward. The
	// normal sanitizer creates deployment-owned networks. Service volumes are
	// intentionally discarded so one untrusted revision cannot leave durable
	// files that influence a later revision of the same preview.
	filtered := map[string]interface{}{"services": services}
	if version, exists := compose["version"]; exists {
		filtered["version"] = version
	}
	result, err := yaml.Marshal(filtered)
	if err != nil {
		return "", fmt.Errorf("failed to marshal untrusted compose YAML: %w", err)
	}
	return string(result), nil
}

func containsComposeInterpolationMarker(value interface{}) bool {
	switch typed := value.(type) {
	case string:
		return strings.Contains(typed, "$")
	case map[string]interface{}:
		for key, child := range typed {
			if strings.Contains(key, "$") || containsComposeInterpolationMarker(child) {
				return true
			}
		}
	case map[interface{}]interface{}:
		for key, child := range typed {
			if containsComposeInterpolationMarker(key) || containsComposeInterpolationMarker(child) {
				return true
			}
		}
	case []interface{}:
		for _, child := range typed {
			if containsComposeInterpolationMarker(child) {
				return true
			}
		}
	}
	return false
}

// sanitizeService sanitizes a single service in the compose file
func (cs *ComposeSanitizer) sanitizeService(service map[string]interface{}, serviceName string) error {
	// Sanitize environment variables to ensure proper formatting
	cs.sanitizeEnvironment(service)
	cs.sanitizeDNS(service)

	// Convert depends_on from map format to list format for Docker Swarm compatibility
	// Swarm's `docker stack deploy` only accepts list format, not map with conditions
	cs.convertDependsOnToList(service)
	// Sanitize volumes
	if volumes, ok := service["volumes"].([]interface{}); ok {
		sanitizedVolumes := []interface{}{}
		hasLocalBind := false
		for _, vol := range volumes {
			sanitized, err := cs.sanitizeVolumeBinding(vol, serviceName)
			if err != nil {
				return err
			}
			if sanitized != nil {
				sanitizedVolumes = append(sanitizedVolumes, sanitized)
				hasLocalBind = hasLocalBind || composeVolumeUsesHostBind(sanitized)
			}
		}
		service["volumes"] = sanitizedVolumes
		cs.usesLocalBind = cs.usesLocalBind || hasLocalBind
		if hasLocalBind && cs.swarmVolumeNodeID != "" {
			if err := pinComposeServiceToNode(service, cs.swarmVolumeNodeID); err != nil {
				return fmt.Errorf("service %s placement: %w", serviceName, err)
			}
		}
	}

	// Sanitize ports - remove host port publishing and keep container ports as
	// expose hints so Obiente routing can still auto-detect the target port.
	if ports, ok := service["ports"].([]interface{}); ok {
		exposedPorts := existingExposePorts(service["expose"])
		for _, port := range ports {
			if sanitized := cs.sanitizePortBinding(port); sanitized != "" {
				exposedPorts = appendUniqueExposePort(exposedPorts, sanitized)
			}
		}
		delete(service, "ports")
		if len(exposedPorts) > 0 {
			service["expose"] = exposedPorts
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

	return nil
}

func composeVolumeUsesHostBind(volume interface{}) bool {
	switch typed := volume.(type) {
	case string:
		return strings.Contains(typed, ":")
	case map[string]interface{}:
		volumeType, _ := typed["type"].(string)
		return strings.EqualFold(strings.TrimSpace(volumeType), "bind")
	default:
		return false
	}
}

func pinComposeServiceToNode(service map[string]interface{}, nodeID string) error {
	deploy, _ := service["deploy"].(map[string]interface{})
	if deploy == nil {
		deploy = make(map[string]interface{})
		service["deploy"] = deploy
	}
	placement, _ := deploy["placement"].(map[string]interface{})
	if placement == nil {
		placement = make(map[string]interface{})
		deploy["placement"] = placement
	}

	existing, _ := placement["constraints"].([]interface{})
	constraints := make([]interface{}, 0, len(existing)+1)
	for _, raw := range existing {
		constraint, ok := raw.(string)
		if !ok {
			return fmt.Errorf("cannot validate non-string placement constraint against required local-volume node %s", strings.TrimSpace(nodeID))
		}
		if err := validateSwarmConstraintForPinnedNode(constraint, nodeID); err != nil {
			return err
		}
		if isSwarmNodeIDConstraint(constraint) {
			continue
		}
		constraints = append(constraints, constraint)
	}
	constraints = append(constraints, fmt.Sprintf("node.id == %s", strings.TrimSpace(nodeID)))
	placement["constraints"] = constraints
	return nil
}

func (cs *ComposeSanitizer) UsesLocalBindVolumes() bool {
	return cs.usesLocalBind
}

func (cs *ComposeSanitizer) HasExistingVolumeRoots() bool {
	for _, state := range cs.volumeRootStates {
		if state.existed {
			return true
		}
	}
	return false
}

// sanitizeDNS applies safe DNS defaults for Compose deployments.
// We preserve explicit user DNS settings and avoid forcing platform nameservers,
// because public *.my.obiente.cloud records should resolve through normal DNS.
func (cs *ComposeSanitizer) sanitizeDNS(service map[string]interface{}) {
	if _, exists := service["dns_search"]; !exists {
		service["dns_search"] = []interface{}{}
	}
}

// convertDependsOnToList converts depends_on from map format to list format
// Docker Swarm's `docker stack deploy` only accepts list format, not map with conditions
// This ensures compatibility when deploying to Swarm clusters
func (cs *ComposeSanitizer) convertDependsOnToList(service map[string]interface{}) {
	if dependsOn, ok := service["depends_on"]; ok {
		// If it's a map (service conditions format), convert to list
		if depMap, ok := dependsOn.(map[string]interface{}); ok {
			// Extract service names from map keys and sort for deterministic output
			serviceList := make([]string, 0, len(depMap))
			for serviceName := range depMap {
				serviceList = append(serviceList, serviceName)
			}
			// Sort to ensure deterministic order (maps have random iteration order in Go)
			// This prevents flapping in generated YAML
			for i := 0; i < len(serviceList)-1; i++ {
				for j := i + 1; j < len(serviceList); j++ {
					if serviceList[i] > serviceList[j] {
						serviceList[i], serviceList[j] = serviceList[j], serviceList[i]
					}
				}
			}
			// Convert to []interface{} for YAML marshaling
			interfaceList := make([]interface{}, len(serviceList))
			for i, s := range serviceList {
				interfaceList[i] = s
			}
			// Replace with list format
			service["depends_on"] = interfaceList
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
func (cs *ComposeSanitizer) sanitizeVolumeBinding(vol interface{}, serviceName string) (interface{}, error) {
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
			return nil, nil // Invalid volume spec - no target
		}

		// Check volume type
		volType, _ := v["type"].(string)
		readOnly, _ := v["read_only"].(bool)

		// If it has a source, check if it's a bind mount (absolute path) or named volume
		if source, ok := v["source"].(string); ok {
			source = strings.TrimSpace(source)
			if volType == "volume" {
				if source == "" || sanitizeVolumeName(source) != source {
					return nil, fmt.Errorf("invalid named volume source %q", source)
				}
			} else if volType == "bind" || filepath.IsAbs(source) || strings.HasPrefix(source, "~") || strings.ContainsRune(source, filepath.Separator) {
				// Bind mount - sanitize absolute and relative paths.
				sanitizedSource, err := cs.sanitizeHostPath(source, serviceName, readOnly)
				if err != nil {
					return nil, err
				}
				result := map[string]interface{}{
					"type":   "bind",
					"source": sanitizedSource,
					"target": target,
				}
				if readOnly {
					result["read_only"] = true
				}
				return result, nil
			} else if source == "" || sanitizeVolumeName(source) != source {
				return nil, fmt.Errorf("invalid volume source %q", source)
			}
			if source != "" && sanitizeVolumeName(source) == source {
				// Named volume - convert to a bind mount beneath the selected root.
				obienteVolumePath := filepath.Join(cs.safeBaseDir, source)
				if err := cs.prepareBindDir(obienteVolumePath, !readOnly); err != nil {
					return nil, fmt.Errorf("prepare volume directory %s: %w", obienteVolumePath, err)
				}
				result := map[string]interface{}{
					"type":   "bind",
					"source": obienteVolumePath,
					"target": target,
				}
				if readOnly {
					result["read_only"] = true
				}
				return result, nil
			}
			return nil, fmt.Errorf("invalid volume source %q", source)
		}

		// Check if it's explicitly a named volume type (without source, just name reference)
		if volType == "volume" {
			// This is a named volume reference - we need to check if there's a name
			// In compose, this might be referenced by service name or explicit name
			// For now, we'll handle it based on the volume definition context
			// This case is handled in sanitizeVolumeDefinition
			return vol, nil
		}

		// No source, no explicit type - could be a simple named volume reference
		// This case should be handled in string parsing below
		return vol, nil
	default:
		return vol, nil
	}

	// Parse string format: "host_path:container_path" or "/host:/container" or "named_volume:container_path"
	if strings.Contains(volStr, ":") {
		parts := strings.SplitN(volStr, ":", 2)
		if len(parts) != 2 {
			return vol, nil
		}

		hostPath := strings.TrimSpace(parts[0])
		containerPath := strings.TrimSpace(parts[1])
		readOnly := composeShortVolumeReadOnly(containerPath)

		// A source matching the platform volume-name grammar is a named volume.
		// Every other source is treated as a bind path and must pass containment.
		if hostPath != "" && sanitizeVolumeName(hostPath) == hostPath {
			// Named volume - convert to a bind mount beneath the selected root.
			volumeName := hostPath
			obienteVolumePath := filepath.Join(cs.safeBaseDir, volumeName)
			// Ensure directory exists
			if err := cs.prepareBindDir(obienteVolumePath, !readOnly); err != nil {
				return nil, fmt.Errorf("prepare volume directory %s: %w", obienteVolumePath, err)
			}
			// Return as bind mount
			return fmt.Sprintf("%s:%s", obienteVolumePath, containerPath), nil
		}

		// It's a bind mount - sanitize host path
		sanitizedHostPath, err := cs.sanitizeHostPath(hostPath, serviceName, readOnly)
		if err != nil {
			return nil, err
		}

		// Return as string format
		return fmt.Sprintf("%s:%s", sanitizedHostPath, containerPath), nil
	}

	// Not a bind mount string, likely a named volume reference
	// If it looks like a named volume (no path separators, simple name), convert to bind mount
	if volStr != "" && !strings.Contains(volStr, ":") && sanitizeVolumeName(volStr) == volStr {
		// This is a named volume - convert to bind mount
		obienteVolumePath := filepath.Join(cs.safeBaseDir, volStr)
		if err := cs.prepareWritableBindDir(obienteVolumePath); err != nil {
			return nil, fmt.Errorf("prepare volume directory %s: %w", obienteVolumePath, err)
		}
		// Return as bind mount with default container path
		return fmt.Sprintf("%s:/data", obienteVolumePath), nil
	}

	return vol, nil
}

func composeShortVolumeReadOnly(containerAndMode string) bool {
	parts := strings.Split(containerAndMode, ":")
	if len(parts) < 2 {
		return false
	}
	for _, option := range strings.Split(parts[len(parts)-1], ",") {
		if strings.TrimSpace(option) == "ro" {
			return true
		}
	}
	return false
}

// sanitizeHostPath transforms a host path to a safe user directory
func (cs *ComposeSanitizer) sanitizeHostPath(hostPath string, serviceName string, readOnly bool) (string, error) {
	if strings.ContainsRune(hostPath, '\x00') {
		return "", fmt.Errorf("volume source contains a null byte")
	}
	isDeploymentRelative := !filepath.IsAbs(hostPath) && !strings.HasPrefix(hostPath, "~")
	// Clean the path to prevent directory traversal
	hostPath = filepath.Clean(hostPath)

	// Remove leading slash or ~ to get relative path component
	relativePath := strings.TrimPrefix(hostPath, "/")
	relativePath = strings.TrimPrefix(relativePath, "~/")
	relativePath = strings.TrimPrefix(relativePath, "~")

	// Extract meaningful parts and reject traversal for both absolute and
	// relative bind sources.
	parts := strings.Split(relativePath, string(filepath.Separator))
	safeParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == ".." {
			return "", fmt.Errorf("volume source %q contains path traversal", hostPath)
		}
		if part != "" && part != "." {
			safeParts = append(safeParts, part)
		}
	}
	relativePath = strings.Join(safeParts, string(filepath.Separator))

	// Deployment-relative Compose sources are resolved once for the deployment,
	// just as Compose resolves them against one project directory. Keep that
	// project root as an explicit directory so "." and "./data" stay distinct
	// while retaining their parent/child relationship. Absolute and home-relative
	// sources retain the existing per-service isolation.
	pathScope := serviceName
	safePath := ""
	if isDeploymentRelative {
		pathScope = relativeComposeBindScope
		namespacedRelativePath := filepath.Join(relativeComposeProjectRoot, relativePath)
		namespacedPath := filepath.Join(cs.safeBaseDir, pathScope, namespacedRelativePath)
		if legacyPath, found := cs.legacyRelativePaths[legacyRelativeBindKey(serviceName, hostPath)]; found {
			safePath = legacyPath
		} else if legacyPath, found := cs.legacyRelativePaths[hostPath]; cs.legacyProjectRoot && found {
			safePath = legacyPath
		} else if _, err := os.Lstat(namespacedPath); err == nil {
			safePath = namespacedPath
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect namespaced relative volume directory %s: %w", namespacedPath, err)
		} else if legacyPath, found := cs.legacyRelativePaths[hostPath]; found {
			safePath = legacyPath
		} else {
			relativePath = namespacedRelativePath
		}
	} else if serviceName == relativeComposeBindScope {
		return "", fmt.Errorf("service name %q conflicts with the relative bind namespace", serviceName)
	}
	if relativePath == "" || strings.Trim(relativePath, ".") == "" {
		relativePath = "data"
	}
	if safePath == "" {
		safePath = filepath.Join(cs.safeBaseDir, pathScope, relativePath)
	}
	rel, err := filepath.Rel(cs.safeBaseDir, safePath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("sanitized volume source escapes the deployment root")
	}

	// Ensure directory exists
	if err := cs.prepareBindDir(safePath, !readOnly); err != nil {
		return "", fmt.Errorf("prepare volume directory %s: %w", safePath, err)
	}

	return safePath, nil
}

// sanitizeVolumeDefinition sanitizes top-level volume definitions
func (cs *ComposeSanitizer) sanitizeVolumeDefinition(volName string, volData interface{}) error {
	if volName == "" || sanitizeVolumeName(volName) != volName {
		return fmt.Errorf("invalid top-level volume name %q", volName)
	}
	// Convert named volume definitions to bind mounts beneath the selected root.
	if volMap, ok := volData.(map[string]interface{}); ok {
		// Empty named-volume definitions are resolved from service references.
		if len(volMap) == 0 {
			// Replace with bind mount configuration
			// The actual bind and its access mode are prepared from each service
			// reference, so an unused definition does not create host state.
			delete(volMap, "driver")
			delete(volMap, "driver_opts")
		} else {
			// Handle driver_opts with device/bind mounts
			if driverOpts, ok := volMap["driver_opts"].(map[string]interface{}); ok {
				readOnly := false
				if options, ok := driverOpts["o"].(string); ok {
					readOnly = composeBindOptionsReadOnly(options)
				}
				// Check for device or type=bind options
				if device, ok := driverOpts["device"].(string); ok {
					// Transform device path to safe directory
					sanitizedDevice, err := cs.sanitizeHostPath(device, "volume-"+volName, readOnly)
					if err != nil {
						return err
					}
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
	return nil
}

func composeBindOptionsReadOnly(options string) bool {
	for _, option := range strings.Split(options, ",") {
		if strings.TrimSpace(option) == "ro" {
			return true
		}
	}
	return false
}

func strictDescendantPath(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func (cs *ComposeSanitizer) rejectWritableBindHierarchy(path string, writable bool) error {
	for preparedPath, preparedWritable := range cs.preparedWritable {
		if preparedPath == path {
			continue
		}
		if preparedWritable && strictDescendantPath(preparedPath, path) {
			return fmt.Errorf("volume source %s is nested beneath writable source %s", path, preparedPath)
		}
		if writable && strictDescendantPath(path, preparedPath) {
			return fmt.Errorf("writable volume source %s contains another mounted source %s", path, preparedPath)
		}
	}
	return nil
}

func (cs *ComposeSanitizer) prepareBindDir(path string, writable bool) error {
	if err := cs.rejectWritableBindHierarchy(path, writable); err != nil {
		return err
	}
	if _, prepared := cs.preparedVolumeSet[path]; !prepared {
		state, err := snapshotVolumeRootState(path)
		if err != nil {
			return err
		}
		cs.volumeRootStates[path] = state
		cs.preparedVolumeRoots = append(cs.preparedVolumeRoots, path)
		cs.preparedVolumeSet[path] = struct{}{}
	} else if cs.preparedWritable[path] || !writable {
		return nil
	}

	cs.preparedWritable[path] = writable
	if writable {
		return ensureWritableBindDir(path)
	}
	if err := secureEnsureDirectory(path); err != nil {
		return err
	}
	if _, deferred := cs.deferredReadOnlySet[path]; !deferred {
		cs.deferredReadOnly = append(cs.deferredReadOnly, path)
		cs.deferredReadOnlySet[path] = struct{}{}
	}
	return nil
}

func (cs *ComposeSanitizer) prepareWritableBindDir(path string) error {
	return cs.prepareBindDir(path, true)
}

func (cs *ComposeSanitizer) rollbackVolumePreparation() error {
	err := rollbackVolumeRootStates(cs.volumeRootStates, cs.preparedVolumeRoots)
	cs.volumeRootStates = nil
	cs.preparedVolumeRoots = nil
	cs.preparedVolumeSet = nil
	cs.preparedWritable = nil
	cs.deferredReadOnly = nil
	cs.deferredReadOnlySet = nil
	return err
}

func (cs *ComposeSanitizer) applyDeferredReadOnly() error {
	for _, path := range cs.deferredReadOnly {
		if cs.preparedWritable[path] {
			continue
		}
		if err := ensureReadOnlyBindDir(path); err != nil {
			return fmt.Errorf("restrict read-only volume directory %s: %w", path, err)
		}
	}
	cs.deferredReadOnly = nil
	cs.deferredReadOnlySet = nil
	return nil
}

func ensureWritableBindDir(path string) error {
	existingMode := os.FileMode(0)
	if mode, err := secureDirectoryMode(path); err == nil {
		existingMode = mode
	} else if !errors.Is(err, unix.ENOENT) {
		return err
	}
	// Workload images may run as any non-root UID/GID. Make only the isolated
	// bind root universally writable so a fresh volume can be initialized. This
	// is intentionally non-recursive: changing application-owned contents can
	// break mode-sensitive data, and UID migrations remain the application's
	// responsibility. Preserve special bits deliberately applied to an existing
	// root, but do not impose sticky or setgid semantics on a new one.
	// Descriptor-relative traversal rejects symlinks in every component and
	// prevents chmod from escaping the deployment root.
	specialBits := existingMode & (os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
	return secureChmodDirectory(path, 0o777|specialBits, true)
}

func ensureReadOnlyBindDir(path string) error {
	existingMode := os.FileMode(0o755)
	if mode, err := secureDirectoryMode(path); err == nil {
		existingMode = mode
	} else if !errors.Is(err, unix.ENOENT) {
		return err
	}
	// Preserve the mode of an existing directory. Remove group and
	// other write access when a writable volume becomes read-only, without
	// broadening an intentionally more restrictive existing mode or discarding
	// setgid/setuid/sticky semantics used by host-side maintenance.
	return secureChmodDirectory(path, existingMode&^0o022, true)
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

// sanitizePortBinding extracts the container port from a Compose port binding.
// Host publishing is intentionally discarded because Obiente exposes services
// through routing rules rather than host-level port bindings.
func (cs *ComposeSanitizer) sanitizePortBinding(port interface{}) string {
	var portStr string

	switch v := port.(type) {
	case string:
		portStr = v
	case map[string]interface{}:
		// Port mapping object format - keep only target (container) port
		if target, ok := v["target"].(int); ok {
			return fmt.Sprintf("%d", target)
		}
		if published, ok := v["published"].(int); ok {
			return fmt.Sprintf("%d", published)
		}
		return ""
	default:
		return ""
	}

	portStr = strings.TrimSpace(portStr)
	if portStr == "" {
		return ""
	}

	// Parse string formats:
	// - "host_port:container_port"
	// - "host_ip:host_port:container_port"
	// - "${PORT:-8080}:container_port"
	// - "container_port"
	// Use the last colon so env default expressions like ${PORT:-8080}:8080
	// are parsed correctly.
	if colon := strings.LastIndex(portStr, ":"); colon >= 0 {
		portStr = strings.TrimSpace(portStr[colon+1:])
	}

	if slash := strings.Index(portStr, "/"); slash >= 0 {
		portStr = strings.TrimSpace(portStr[:slash])
	}

	return portStr
}

func existingExposePorts(expose interface{}) []interface{} {
	exposedPorts := []interface{}{}
	if existing, ok := expose.([]interface{}); ok {
		exposedPorts = append(exposedPorts, existing...)
	}
	return exposedPorts
}

func appendUniqueExposePort(exposedPorts []interface{}, port string) []interface{} {
	for _, existing := range exposedPorts {
		if fmt.Sprint(existing) == port {
			return exposedPorts
		}
	}
	return append(exposedPorts, port)
}

// addServiceToNetwork adds a service to the deployment-specific network for isolation
// Services are ONLY added to their private deployment network here
// The obiente-network is added LATER by addTraefikNetworkToRoutedServices() only for services with routing
func (cs *ComposeSanitizer) addServiceToNetwork(service map[string]interface{}) {
	// Always ensure the service is connected to its deployment-specific network
	deploymentNetworkName := fmt.Sprintf("deployment-%s", cs.deploymentID)

	networksVal, ok := service["networks"]
	if !ok || networksVal == nil {
		// No networks defined: connect ONLY to deployment network for isolation
		service["networks"] = []string{deploymentNetworkName}
		return
	}

	// Service already has networks - merge deployment network in
	switch networks := networksVal.(type) {
	case []interface{}:
		// List-style networks: ensure deployment network is present
		for _, n := range networks {
			if s, ok := n.(string); ok && s == deploymentNetworkName {
				// Already connected to deployment network
				return
			}
		}
		service["networks"] = append(networks, deploymentNetworkName)

	case map[string]interface{}:
		// Map-style networks: add deployment network key if missing
		if _, exists := networks[deploymentNetworkName]; !exists {
			// Empty config uses defaults for this network
			networks[deploymentNetworkName] = map[string]interface{}{}
		}

	default:
		// Unexpected type: enforce isolation by resetting to deployment network only
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
