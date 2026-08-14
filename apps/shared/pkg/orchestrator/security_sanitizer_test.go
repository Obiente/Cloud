package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSanitizeUntrustedComposeYAMLRemovesHostControlOptions(t *testing.T) {
	composeYaml := `services:
  app:
    image: nginx:alpine
    devices:
      - /dev/kvm:/dev/kvm
    pid: host
    ipc: host
    privileged: true
    scale: 500
    network_mode: host
    volumes:
      - /etc:/host-etc
`
	sanitizer := NewComposeSanitizer("preview-test")
	filtered, err := sanitizer.SanitizeUntrustedComposeYAML(composeYaml)
	if err != nil {
		t.Fatalf("sanitize untrusted compose: %v", err)
	}
	for _, forbidden := range []string{"devices:", "pid:", "ipc:", "privileged:", "network_mode:", "volumes:", "scale:"} {
		if strings.Contains(filtered, forbidden) {
			t.Fatalf("untrusted Compose retained %q:\n%s", forbidden, filtered)
		}
	}
}

func TestSanitizeUntrustedComposeYAMLRejectsRepositoryBuild(t *testing.T) {
	sanitizer := NewComposeSanitizer("preview-test")
	if _, err := sanitizer.SanitizeUntrustedComposeYAML("services:\n  app:\n    build: .\n"); err == nil {
		t.Fatal("repository Compose build should be rejected for pull request previews")
	}
}

func TestSanitizeUntrustedComposeYAMLRejectsInterpolation(t *testing.T) {
	sanitizer := NewComposeSanitizer("preview-test")
	for name, composeYaml := range map[string]string{
		"braced":        "services:\n  app:\n    image: ${IMAGE}\n",
		"unbraced":      "services:\n  app:\n    image: $IMAGE\n",
		"double-dollar": "services:\n  app:\n    image: nginx\n    command: '$${LITERAL}'\n",
		"yaml-escaped": `services:
  app:
    image: nginx
    command: "\u0024SECRET"
`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := sanitizer.SanitizeUntrustedComposeYAML(composeYaml); err == nil {
				t.Fatal("environment interpolation marker should be rejected for pull request previews")
			}
		})
	}
}

func TestSanitizeUntrustedComposeYAMLBoundsAggregateResources(t *testing.T) {
	sanitizer := NewComposeSanitizer("preview-test")
	filtered, err := sanitizer.SanitizeUntrustedComposeYAMLWithLimits(`services:
  web:
    image: nginx
  worker:
    image: alpine
`, UntrustedComposeLimits{MaxServices: 2, TotalMemoryBytes: 1024, TotalCPUShares: 512})
	if err != nil {
		t.Fatalf("sanitize budgeted compose: %v", err)
	}
	var result map[string]interface{}
	if err := yaml.Unmarshal([]byte(filtered), &result); err != nil {
		t.Fatalf("parse sanitized compose: %v", err)
	}
	services := result["services"].(map[string]interface{})
	for name, raw := range services {
		service := raw.(map[string]interface{})
		deploy := service["deploy"].(map[string]interface{})
		limits := deploy["resources"].(map[string]interface{})["limits"].(map[string]interface{})
		if limits["memory"] != "512B" || limits["cpus"] != "0.250000" {
			t.Fatalf("service %s limits = %#v", name, limits)
		}
	}
	if _, err := sanitizer.SanitizeUntrustedComposeYAMLWithLimits("services:\n  one:\n    image: nginx\n  two:\n    image: nginx\n  three:\n    image: nginx\n", UntrustedComposeLimits{MaxServices: 2, TotalMemoryBytes: 1024, TotalCPUShares: 512}); err == nil {
		t.Fatal("service count above the configured maximum should be rejected")
	}
}

func TestSanitizeEnvironment_BooleanValues(t *testing.T) {
	composeYaml := `version: '3.8'
services:
  web:
    image: nginx:alpine
    environment:
      ENABLE_SSL: true
      DEBUG_MODE: false
      PORT: 8080
      APP_NAME: "myapp"
`

	sanitizer := NewComposeSanitizer("test-deployment")
	sanitizedYaml, err := sanitizer.SanitizeComposeYAML(composeYaml)
	if err != nil {
		t.Fatalf("Failed to sanitize YAML: %v", err)
	}

	// Parse the result to verify environment variables are strings
	var result map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitizedYaml), &result); err != nil {
		t.Fatalf("Failed to parse sanitized YAML: %v", err)
	}

	services := result["services"].(map[string]interface{})
	web := services["web"].(map[string]interface{})
	env := web["environment"].(map[string]interface{})

	// Check that boolean values are converted to strings
	if enableSSL, ok := env["ENABLE_SSL"].(string); !ok {
		t.Errorf("ENABLE_SSL should be a string, got %T: %v", env["ENABLE_SSL"], env["ENABLE_SSL"])
	} else if enableSSL != "true" {
		t.Errorf("ENABLE_SSL should be 'true', got %q", enableSSL)
	}

	if debugMode, ok := env["DEBUG_MODE"].(string); !ok {
		t.Errorf("DEBUG_MODE should be a string, got %T: %v", env["DEBUG_MODE"], env["DEBUG_MODE"])
	} else if debugMode != "false" {
		t.Errorf("DEBUG_MODE should be 'false', got %q", debugMode)
	}

	// Check that numbers are converted to strings
	if port, ok := env["PORT"].(string); !ok {
		t.Errorf("PORT should be a string, got %T: %v", env["PORT"], env["PORT"])
	} else if port != "8080" {
		t.Errorf("PORT should be '8080', got %q", port)
	}

	// Check that strings remain strings
	if appName, ok := env["APP_NAME"].(string); !ok {
		t.Errorf("APP_NAME should be a string, got %T: %v", env["APP_NAME"], env["APP_NAME"])
	} else if appName != "myapp" {
		t.Errorf("APP_NAME should be 'myapp', got %q", appName)
	}
}

func TestSanitizeEnvironment_DollarSignEscaping(t *testing.T) {
	composeYaml := `version: '3.8'
services:
  api:
    image: myapp/api:latest
    environment:
      DATABASE_URL: "postgresql://user:p@ss$word@localhost/db"
      SECRET_KEY: "a1b2c3$d4e5$f6g7"
      API_TOKEN: "tok$en$here"
`

	sanitizer := NewComposeSanitizer("test-deployment")
	sanitizedYaml, err := sanitizer.SanitizeComposeYAML(composeYaml)
	if err != nil {
		t.Fatalf("Failed to sanitize YAML: %v", err)
	}

	// Parse the result to verify $ characters are escaped
	var result map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitizedYaml), &result); err != nil {
		t.Fatalf("Failed to parse sanitized YAML: %v", err)
	}

	services := result["services"].(map[string]interface{})
	api := services["api"].(map[string]interface{})
	env := api["environment"].(map[string]interface{})

	// Check that $ is escaped to $$
	if dbURL, ok := env["DATABASE_URL"].(string); !ok {
		t.Errorf("DATABASE_URL should be a string, got %T: %v", env["DATABASE_URL"], env["DATABASE_URL"])
	} else if !strings.Contains(dbURL, "$$") {
		t.Errorf("DATABASE_URL should contain escaped $$ characters, got %q", dbURL)
	} else if strings.Count(dbURL, "$$") != 1 {
		t.Errorf("DATABASE_URL should have 1 escaped $ character, got %q", dbURL)
	}

	if secretKey, ok := env["SECRET_KEY"].(string); !ok {
		t.Errorf("SECRET_KEY should be a string, got %T: %v", env["SECRET_KEY"], env["SECRET_KEY"])
	} else if expected := "a1b2c3$$d4e5$$f6g7"; secretKey != expected {
		t.Errorf("SECRET_KEY should be %q, got %q", expected, secretKey)
	}

	if apiToken, ok := env["API_TOKEN"].(string); !ok {
		t.Errorf("API_TOKEN should be a string, got %T: %v", env["API_TOKEN"], env["API_TOKEN"])
	} else if expected := "tok$$en$$here"; apiToken != expected {
		t.Errorf("API_TOKEN should be %q, got %q", expected, apiToken)
	}
}

func TestSanitizeEnvironment_ArrayFormat(t *testing.T) {
	composeYaml := `version: '3.8'
services:
  worker:
    image: worker:latest
    environment:
      - ENABLE_FEATURE=true
      - MAX_WORKERS=10
      - DB_PASS=my$ecret$pass
`

	sanitizer := NewComposeSanitizer("test-deployment")
	sanitizedYaml, err := sanitizer.SanitizeComposeYAML(composeYaml)
	if err != nil {
		t.Fatalf("Failed to sanitize YAML: %v", err)
	}

	// Check that the sanitized YAML contains properly escaped values
	if !strings.Contains(sanitizedYaml, "ENABLE_FEATURE=true") {
		t.Errorf("Sanitized YAML should contain 'ENABLE_FEATURE=true'")
	}

	if !strings.Contains(sanitizedYaml, "DB_PASS=my$$ecret$$pass") {
		t.Errorf("Sanitized YAML should contain 'DB_PASS=my$$ecret$$pass', got:\n%s", sanitizedYaml)
	}
}

func TestSanitizeEnvironment_NullValues(t *testing.T) {
	composeYaml := `version: '3.8'
services:
  cache:
    image: redis:alpine
    environment:
      OPTIONAL_CONFIG: null
      EMPTY_VALUE: ""
`

	sanitizer := NewComposeSanitizer("test-deployment")
	sanitizedYaml, err := sanitizer.SanitizeComposeYAML(composeYaml)
	if err != nil {
		t.Fatalf("Failed to sanitize YAML: %v", err)
	}

	// Parse the result
	var result map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitizedYaml), &result); err != nil {
		t.Fatalf("Failed to parse sanitized YAML: %v", err)
	}

	services := result["services"].(map[string]interface{})
	cache := services["cache"].(map[string]interface{})
	env := cache["environment"].(map[string]interface{})

	// Check that null values are converted to empty strings
	if optionalConfig, ok := env["OPTIONAL_CONFIG"].(string); !ok {
		t.Errorf("OPTIONAL_CONFIG should be a string, got %T: %v", env["OPTIONAL_CONFIG"], env["OPTIONAL_CONFIG"])
	} else if optionalConfig != "" {
		t.Errorf("OPTIONAL_CONFIG should be empty string, got %q", optionalConfig)
	}

	// Empty strings should remain empty
	if emptyValue, ok := env["EMPTY_VALUE"].(string); !ok {
		t.Errorf("EMPTY_VALUE should be a string, got %T: %v", env["EMPTY_VALUE"], env["EMPTY_VALUE"])
	} else if emptyValue != "" {
		t.Errorf("EMPTY_VALUE should be empty string, got %q", emptyValue)
	}
}

func TestSanitizeDNS_DefaultsSearchDomainOnly(t *testing.T) {
	composeYaml := `version: '3.8'
services:
  app:
    image: nginx:alpine
`

	sanitizer := NewComposeSanitizer("test-deployment")
	sanitizedYaml, err := sanitizer.SanitizeComposeYAML(composeYaml)
	if err != nil {
		t.Fatalf("Failed to sanitize YAML: %v", err)
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitizedYaml), &result); err != nil {
		t.Fatalf("Failed to parse sanitized YAML: %v", err)
	}

	services := result["services"].(map[string]interface{})
	app := services["app"].(map[string]interface{})

	dnsSearch, ok := app["dns_search"].([]interface{})
	if !ok {
		t.Fatalf("dns_search should be present as a list, got %T", app["dns_search"])
	}
	if len(dnsSearch) != 0 {
		t.Fatalf("dns_search should default to an empty list, got %#v", dnsSearch)
	}

	if _, ok := app["dns"]; ok {
		t.Fatalf("dns should not be injected automatically, got %#v", app["dns"])
	}
}

func TestSanitizeDNS_PreservesExplicitUserSettings(t *testing.T) {
	composeYaml := `version: '3.8'
services:
  app:
    image: nginx:alpine
    dns:
      - 1.1.1.1
    dns_search:
      - custom.internal
`

	sanitizer := NewComposeSanitizer("test-deployment")
	sanitizedYaml, err := sanitizer.SanitizeComposeYAML(composeYaml)
	if err != nil {
		t.Fatalf("Failed to sanitize YAML: %v", err)
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitizedYaml), &result); err != nil {
		t.Fatalf("Failed to parse sanitized YAML: %v", err)
	}

	services := result["services"].(map[string]interface{})
	app := services["app"].(map[string]interface{})

	dnsSearch := app["dns_search"].([]interface{})
	if len(dnsSearch) != 1 || dnsSearch[0] != "custom.internal" {
		t.Fatalf("dns_search should preserve user value, got %#v", dnsSearch)
	}

	dnsServers := app["dns"].([]interface{})
	if len(dnsServers) != 1 || dnsServers[0] != "1.1.1.1" {
		t.Fatalf("dns should preserve user value, got %#v", dnsServers)
	}
}

func TestSanitizeComposeYAML_RemovesTopLevelName(t *testing.T) {
	composeYaml := `name: upstream-project
services:
  app:
    image: nginx:alpine
`

	sanitizer := NewComposeSanitizer("test-deployment")
	sanitizedYaml, err := sanitizer.SanitizeComposeYAML(composeYaml)
	if err != nil {
		t.Fatalf("Failed to sanitize YAML: %v", err)
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitizedYaml), &result); err != nil {
		t.Fatalf("Failed to parse sanitized YAML: %v", err)
	}

	if _, ok := result["name"]; ok {
		t.Fatalf("top-level name should be removed for docker stack deploy, got %#v", result["name"])
	}
}

func TestSanitizeComposeYAML_MovesHostPortMappingsToExpose(t *testing.T) {
	composeYaml := `version: '3.8'
services:
  gowhisper:
    image: example/gowhisper:latest
    ports:
      - "${WHISPER_GO_PORT:-8080}:8080"
      - "127.0.0.1:9000:9000/tcp"
    expose:
      - "7000"
`

	sanitizer := NewComposeSanitizer("test-deployment")
	sanitizedYaml, err := sanitizer.SanitizeComposeYAML(composeYaml)
	if err != nil {
		t.Fatalf("Failed to sanitize YAML: %v", err)
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitizedYaml), &result); err != nil {
		t.Fatalf("Failed to parse sanitized YAML: %v", err)
	}

	services := result["services"].(map[string]interface{})
	gowhisper := services["gowhisper"].(map[string]interface{})

	if _, ok := gowhisper["ports"]; ok {
		t.Fatalf("ports should be removed after sanitization, got %#v", gowhisper["ports"])
	}

	expose := gowhisper["expose"].([]interface{})
	got := map[string]bool{}
	for _, port := range expose {
		got[port.(string)] = true
	}

	for _, want := range []string{"7000", "8080", "9000"} {
		if !got[want] {
			t.Fatalf("expected expose to contain %q, got %#v", want, expose)
		}
	}
}

func TestEnsureWritableBindDir_MakesDirectoryWritable(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "uploads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	if err := ensureWritableBindDir(dir); err != nil {
		t.Fatalf("ensureWritableBindDir failed: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("failed to stat dir: %v", err)
	}

	if got := info.Mode().Perm(); got != 0o777 {
		t.Fatalf("expected directory mode 0777, got %#o", got)
	}
	if info.Mode()&os.ModeSticky != 0 {
		t.Fatal("writable bind directory unexpectedly has the sticky bit")
	}
}

func TestEnsureWritableBindDirPreservesExistingContents(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	child := filepath.Join(dir, "existing.dat")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("create existing volume root: %v", err)
	}
	if err := os.WriteFile(child, []byte("preserve me"), 0o600); err != nil {
		t.Fatalf("create existing volume content: %v", err)
	}

	if err := ensureWritableBindDir(dir); err != nil {
		t.Fatalf("prepare existing writable bind directory: %v", err)
	}

	contents, err := os.ReadFile(child)
	if err != nil {
		t.Fatalf("read existing volume content: %v", err)
	}
	if string(contents) != "preserve me" {
		t.Fatalf("existing volume content changed to %q", contents)
	}
	info, err := os.Stat(child)
	if err != nil {
		t.Fatalf("stat existing volume content: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("existing child mode changed to %#o", got)
	}
}

func TestEnsureWritableBindDirKeepsParentHierarchyRestricted(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "deploy-volume-test")
	dir := filepath.Join(parent, "data")

	if err := ensureWritableBindDir(dir); err != nil {
		t.Fatalf("prepare writable bind directory: %v", err)
	}

	parentInfo, err := os.Stat(parent)
	if err != nil {
		t.Fatalf("stat deployment volume parent: %v", err)
	}
	if got := parentInfo.Mode().Perm(); got != 0o755 {
		t.Fatalf("deployment volume parent mode = %#o, want 0755", got)
	}
	if parentInfo.Mode()&os.ModeSticky != 0 {
		t.Fatal("deployment volume parent unexpectedly has the sticky bit")
	}
}

func TestEnsureReadOnlyBindDirPreservesRestrictiveMode(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private-data")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("create restrictive volume root: %v", err)
	}

	if err := ensureReadOnlyBindDir(dir); err != nil {
		t.Fatalf("prepare read-only bind directory: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat read-only bind directory: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("read-only bind mode = %#o, want 0700", got)
	}
}

func TestEnsureReadOnlyBindDirPreservesSetgid(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shared-data")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("create setgid volume root: %v", err)
	}
	if err := os.Chmod(dir, 0o750|os.ModeSetgid); err != nil {
		t.Fatalf("set setgid volume root mode: %v", err)
	}

	if err := ensureReadOnlyBindDir(dir); err != nil {
		t.Fatalf("prepare setgid read-only bind directory: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat setgid read-only bind directory: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o750 {
		t.Fatalf("read-only permissions = %#o, want 0750", got)
	}
	if info.Mode()&os.ModeSetgid == 0 {
		t.Fatal("read-only preparation removed setgid")
	}
}

func TestEnsureWritableBindDirRejectsSymlinkComponents(t *testing.T) {
	root := t.TempDir()
	safeBase := filepath.Join(root, "safe")
	hostTarget := filepath.Join(root, "host-target")
	if err := os.MkdirAll(safeBase, 0o755); err != nil {
		t.Fatalf("create safe base: %v", err)
	}
	if err := os.MkdirAll(hostTarget, 0o700); err != nil {
		t.Fatalf("create host target: %v", err)
	}
	if err := os.Symlink(hostTarget, filepath.Join(safeBase, "link")); err != nil {
		t.Fatalf("create malicious volume symlink: %v", err)
	}

	if err := ensureWritableBindDir(filepath.Join(safeBase, "link", "nested")); err == nil {
		t.Fatal("expected writable bind preparation to reject a symlink component")
	}
	info, err := os.Stat(hostTarget)
	if err != nil {
		t.Fatalf("stat host target: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("host target mode changed through symlink to %#o", got)
	}
}

func TestSanitizeComposeYAMLFailsWhenVolumePreparationFails(t *testing.T) {
	safeBaseDir := t.TempDir()
	conflictingPath := filepath.Join(safeBaseDir, "app", "data")
	if err := os.MkdirAll(filepath.Dir(conflictingPath), 0o755); err != nil {
		t.Fatalf("create service volume parent: %v", err)
	}
	if err := os.WriteFile(conflictingPath, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("create conflicting volume path: %v", err)
	}

	sanitizer := &ComposeSanitizer{
		deploymentID: "compose-volume-error-test",
		safeBaseDir:  safeBaseDir,
	}
	composeYAML := `services:
  app:
    image: example.invalid/app:latest
    volumes:
      - type: bind
        source: /data
        target: /data
`
	sanitized, err := sanitizer.SanitizeComposeYAML(composeYAML)
	if err == nil {
		t.Fatal("expected Compose volume preparation failure")
	}
	if sanitized != "" {
		t.Fatalf("failed sanitization returned output: %q", sanitized)
	}
	if !strings.Contains(err.Error(), "prepare volume directory") {
		t.Fatalf("unexpected Compose sanitization error: %v", err)
	}
}

func TestSanitizeComposeYAMLRestoresEarlierVolumeModesOnFailure(t *testing.T) {
	safeBaseDir := t.TempDir()
	existingPath := filepath.Join(safeBaseDir, "app", "existing")
	conflictingPath := filepath.Join(safeBaseDir, "app", "conflict")
	if err := os.MkdirAll(existingPath, 0o700); err != nil {
		t.Fatalf("create existing Compose volume root: %v", err)
	}
	if err := os.Chmod(existingPath, 0o700); err != nil {
		t.Fatalf("set existing Compose volume mode: %v", err)
	}
	if err := os.WriteFile(conflictingPath, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("create conflicting Compose volume path: %v", err)
	}

	sanitizer := &ComposeSanitizer{
		deploymentID: "compose-volume-rollback-test",
		safeBaseDir:  safeBaseDir,
	}
	composeYAML := `services:
  app:
    image: example.invalid/app:latest
    volumes:
      - /existing:/existing
      - /conflict:/conflict
`
	if _, err := sanitizer.SanitizeComposeYAML(composeYAML); err == nil {
		t.Fatal("expected Compose volume preparation failure")
	}

	info, err := os.Stat(existingPath)
	if err != nil {
		t.Fatalf("stat restored Compose volume root: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("restored Compose volume mode = %#o, want 0700", got)
	}
}

func TestSanitizeComposeYAMLRejectsRelativeVolumeTraversal(t *testing.T) {
	root := t.TempDir()
	safeBaseDir := filepath.Join(root, "safe")
	escapedPath := filepath.Join(root, "escape")
	if err := os.MkdirAll(safeBaseDir, 0o755); err != nil {
		t.Fatalf("create safe Compose root: %v", err)
	}
	if err := os.MkdirAll(escapedPath, 0o700); err != nil {
		t.Fatalf("create traversal target: %v", err)
	}
	if err := os.Chmod(escapedPath, 0o700); err != nil {
		t.Fatalf("set traversal target mode: %v", err)
	}

	sanitizer := &ComposeSanitizer{
		deploymentID: "compose-traversal-test",
		safeBaseDir:  safeBaseDir,
	}
	composeYAML := `services:
  app:
    image: example.invalid/app:latest
    volumes:
      - ../../escape:/data
`
	if _, err := sanitizer.SanitizeComposeYAML(composeYAML); err == nil {
		t.Fatal("expected relative volume traversal to be rejected")
	}

	info, err := os.Stat(escapedPath)
	if err != nil {
		t.Fatalf("stat traversal target: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("traversal target mode = %#o, want 0700", got)
	}
}

func TestSanitizeComposeYAMLPreservesSharedRelativeBindSources(t *testing.T) {
	safeBaseDir := t.TempDir()
	sanitizer := &ComposeSanitizer{
		deploymentID: "compose-shared-relative-bind-test",
		safeBaseDir:  safeBaseDir,
	}
	composeYAML := `services:
  app:
    image: example.invalid/app:latest
    volumes:
      - ./data:/var/lib/app
  worker:
    image: example.invalid/worker:latest
    volumes:
      - type: bind
        source: ./data
        target: /var/lib/worker
`

	sanitized, err := sanitizer.SanitizeComposeYAML(composeYAML)
	if err != nil {
		t.Fatalf("sanitize Compose with shared relative bind: %v", err)
	}

	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitized), &compose); err != nil {
		t.Fatalf("parse sanitized Compose: %v", err)
	}
	services := compose["services"].(map[string]interface{})
	appVolume := services["app"].(map[string]interface{})["volumes"].([]interface{})[0].(string)
	workerVolume := services["worker"].(map[string]interface{})["volumes"].([]interface{})[0].(map[string]interface{})
	appSource := strings.SplitN(appVolume, ":", 2)[0]
	workerSource := workerVolume["source"].(string)
	wantSource := filepath.Join(safeBaseDir, relativeComposeBindScope, relativeComposeProjectRoot, "data")
	if appSource != wantSource || workerSource != wantSource {
		t.Fatalf("shared relative sources = %q and %q, want %q", appSource, workerSource, wantSource)
	}
}

func TestSanitizeComposeYAMLPreservesRelativeProjectRootHierarchy(t *testing.T) {
	safeBaseDir := t.TempDir()
	sanitizer := &ComposeSanitizer{
		deploymentID: "compose-relative-root-test",
		safeBaseDir:  safeBaseDir,
	}
	composeYAML := `services:
  app:
    image: example/app:latest
    volumes:
      - .:/workspace
      - ./data:/workspace/data
`

	sanitized, err := sanitizer.SanitizeComposeYAML(composeYAML)
	if err != nil {
		t.Fatalf("sanitize Compose with project-root binds: %v", err)
	}

	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitized), &compose); err != nil {
		t.Fatalf("parse sanitized Compose: %v", err)
	}
	services := compose["services"].(map[string]interface{})
	volumes := services["app"].(map[string]interface{})["volumes"].([]interface{})
	projectSource := strings.SplitN(volumes[0].(string), ":", 2)[0]
	dataSource := strings.SplitN(volumes[1].(string), ":", 2)[0]
	wantProjectSource := filepath.Join(safeBaseDir, relativeComposeBindScope, relativeComposeProjectRoot)
	if projectSource != wantProjectSource {
		t.Fatalf("project-root source = %q, want %q", projectSource, wantProjectSource)
	}
	if want := filepath.Join(projectSource, "data"); dataSource != want {
		t.Fatalf("project data source = %q, want %q", dataSource, want)
	}
}

func TestSanitizeComposeYAMLReusesLegacyRelativeBindStorage(t *testing.T) {
	safeBaseDir := t.TempDir()
	legacyPath := filepath.Join(safeBaseDir, "data")
	if err := os.MkdirAll(legacyPath, 0o700); err != nil {
		t.Fatalf("create legacy relative bind: %v", err)
	}
	marker := filepath.Join(legacyPath, "existing.dat")
	if err := os.WriteFile(marker, []byte("preserve me"), 0o600); err != nil {
		t.Fatalf("create legacy volume content: %v", err)
	}
	sanitizer := &ComposeSanitizer{
		deploymentID: "compose-legacy-relative-bind-test",
		safeBaseDir:  safeBaseDir,
	}
	composeYAML := `services:
  app:
    image: example/app:latest
    volumes:
      - ./data:/data
  worker:
    image: example/worker:latest
    volumes:
      - ./data:/data
`

	sanitized, err := sanitizer.SanitizeComposeYAML(composeYAML)
	if err != nil {
		t.Fatalf("sanitize Compose with legacy relative bind: %v", err)
	}
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitized), &compose); err != nil {
		t.Fatalf("parse sanitized Compose: %v", err)
	}
	services := compose["services"].(map[string]interface{})
	for _, serviceName := range []string{"app", "worker"} {
		volume := services[serviceName].(map[string]interface{})["volumes"].([]interface{})[0].(string)
		if source := strings.SplitN(volume, ":", 2)[0]; source != legacyPath {
			t.Fatalf("%s legacy relative source = %q, want %q", serviceName, source, legacyPath)
		}
	}
	if contents, err := os.ReadFile(marker); err != nil || string(contents) != "preserve me" {
		t.Fatalf("legacy volume content changed: contents=%q err=%v", contents, err)
	}
}

func TestSanitizeComposeYAMLReusesLegacyRelativeProjectRoot(t *testing.T) {
	safeBaseDir := t.TempDir()
	marker := filepath.Join(safeBaseDir, "existing.dat")
	if err := os.WriteFile(marker, []byte("preserve me"), 0o600); err != nil {
		t.Fatalf("create legacy project-root content: %v", err)
	}
	sanitizer := &ComposeSanitizer{
		deploymentID: "compose-legacy-relative-root-test",
		safeBaseDir:  safeBaseDir,
	}
	composeYAML := `services:
  app:
    image: example/app:latest
    volumes:
      - .:/workspace
`

	sanitized, err := sanitizer.SanitizeComposeYAML(composeYAML)
	if err != nil {
		t.Fatalf("sanitize Compose with legacy project root: %v", err)
	}
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitized), &compose); err != nil {
		t.Fatalf("parse sanitized Compose: %v", err)
	}
	services := compose["services"].(map[string]interface{})
	volume := services["app"].(map[string]interface{})["volumes"].([]interface{})[0].(string)
	if source := strings.SplitN(volume, ":", 2)[0]; source != safeBaseDir {
		t.Fatalf("legacy project-root source = %q, want %q", source, safeBaseDir)
	}
}

func TestSanitizeComposeYAMLKeepsNamedAndRelativeSourcesDistinctAcrossRedeploys(t *testing.T) {
	safeBaseDir := t.TempDir()
	composeYAML := `services:
  database:
    image: example/database:latest
    volumes:
      - data:/var/lib/database
  importer:
    image: example/importer:latest
    volumes:
      - ./data:/input
volumes:
  data: {}
`
	wantNamed := filepath.Join(safeBaseDir, "data")
	wantRelative := filepath.Join(safeBaseDir, relativeComposeBindScope, relativeComposeProjectRoot, "data")

	for attempt := 1; attempt <= 2; attempt++ {
		sanitizer := &ComposeSanitizer{
			deploymentID: "compose-distinct-source-test",
			safeBaseDir:  safeBaseDir,
		}
		sanitized, err := sanitizer.SanitizeComposeYAML(composeYAML)
		if err != nil {
			t.Fatalf("sanitize named/relative sources on attempt %d: %v", attempt, err)
		}
		var compose map[string]interface{}
		if err := yaml.Unmarshal([]byte(sanitized), &compose); err != nil {
			t.Fatalf("parse sanitized Compose on attempt %d: %v", attempt, err)
		}
		services := compose["services"].(map[string]interface{})
		namedVolume := services["database"].(map[string]interface{})["volumes"].([]interface{})[0].(string)
		relativeVolume := services["importer"].(map[string]interface{})["volumes"].([]interface{})[0].(string)
		if source := strings.SplitN(namedVolume, ":", 2)[0]; source != wantNamed {
			t.Fatalf("attempt %d named source = %q, want %q", attempt, source, wantNamed)
		}
		if source := strings.SplitN(relativeVolume, ":", 2)[0]; source != wantRelative {
			t.Fatalf("attempt %d relative source = %q, want %q", attempt, source, wantRelative)
		}
	}
}

func TestSanitizeComposeYAMLUsesSelectedRootForNamedVolumes(t *testing.T) {
	safeBaseDir := t.TempDir()
	sanitizer := &ComposeSanitizer{
		deploymentID: "compose-named-volume-root-test",
		safeBaseDir:  safeBaseDir,
	}
	composeYAML := `services:
  app:
    image: example.invalid/app:latest
    volumes:
      - cache:/var/lib/app
  worker:
    image: example.invalid/worker:latest
    volumes:
      - type: volume
        source: cache
        target: /var/lib/worker
volumes:
  cache: {}
`

	sanitized, err := sanitizer.SanitizeComposeYAML(composeYAML)
	if err != nil {
		t.Fatalf("sanitize Compose named volumes: %v", err)
	}

	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitized), &compose); err != nil {
		t.Fatalf("parse sanitized Compose: %v", err)
	}
	services := compose["services"].(map[string]interface{})
	appVolume := services["app"].(map[string]interface{})["volumes"].([]interface{})[0].(string)
	workerVolume := services["worker"].(map[string]interface{})["volumes"].([]interface{})[0].(map[string]interface{})
	wantSource := filepath.Join(safeBaseDir, "cache")
	if strings.SplitN(appVolume, ":", 2)[0] != wantSource || workerVolume["source"] != wantSource {
		t.Fatalf("named volumes did not use selected root %q: app=%q worker=%#v", wantSource, appVolume, workerVolume)
	}
}

func TestSanitizeComposeYAMLRestrictsReadOnlyNamedVolumeRoot(t *testing.T) {
	safeBaseDir := t.TempDir()
	volumeRoot := filepath.Join(safeBaseDir, "cache")
	if err := os.MkdirAll(volumeRoot, 0o777); err != nil {
		t.Fatalf("create writable named volume root: %v", err)
	}
	if err := os.Chmod(volumeRoot, 0o777); err != nil {
		t.Fatalf("set writable named volume root mode: %v", err)
	}
	sanitizer := &ComposeSanitizer{
		deploymentID: "compose-read-only-volume-test",
		safeBaseDir:  safeBaseDir,
	}
	composeYAML := `services:
  app:
    image: example.invalid/app:latest
    volumes:
      - cache:/data:ro
volumes:
  cache: {}
`

	if _, err := sanitizer.SanitizeComposeYAML(composeYAML); err != nil {
		t.Fatalf("sanitize read-only named volume: %v", err)
	}
	beforeApply, err := os.Stat(volumeRoot)
	if err != nil {
		t.Fatalf("stat deferred read-only named volume root: %v", err)
	}
	if got := beforeApply.Mode().Perm(); got != 0o777 {
		t.Fatalf("read-only mode changed during preflight to %#o, want 0777", got)
	}
	if err := sanitizer.applyDeferredReadOnly(); err != nil {
		t.Fatalf("apply deferred read-only named volume mode: %v", err)
	}
	info, err := os.Stat(volumeRoot)
	if err != nil {
		t.Fatalf("stat read-only named volume root: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Fatalf("read-only named volume mode = %#o, want 0755", got)
	}
}

func TestSanitizeComposeYAMLKeepsSharedVolumeWritableWhenAnyBindingWrites(t *testing.T) {
	orders := []string{
		`  reader:
    image: example.invalid/reader:latest
    volumes:
      - cache:/data:ro
  writer:
    image: example.invalid/writer:latest
    volumes:
      - type: volume
        source: cache
        target: /data
`,
		`  writer:
    image: example.invalid/writer:latest
    volumes:
      - type: volume
        source: cache
        target: /data
  reader:
    image: example.invalid/reader:latest
    volumes:
      - cache:/data:ro
`,
	}
	for index, services := range orders {
		t.Run(fmt.Sprintf("order-%d", index), func(t *testing.T) {
			safeBaseDir := t.TempDir()
			sanitizer := &ComposeSanitizer{
				deploymentID: "compose-mixed-access-volume-test",
				safeBaseDir:  safeBaseDir,
			}
			composeYAML := "services:\n" + services + "volumes:\n  cache: {}\n"
			if _, err := sanitizer.SanitizeComposeYAML(composeYAML); err != nil {
				t.Fatalf("sanitize shared mixed-access volume: %v", err)
			}
			info, err := os.Stat(filepath.Join(safeBaseDir, "cache"))
			if err != nil {
				t.Fatalf("stat shared volume root: %v", err)
			}
			if got := info.Mode().Perm(); got != 0o777 {
				t.Fatalf("shared mixed-access volume mode = %#o, want 0777", got)
			}
		})
	}
}

func TestSanitizeComposeYAMLPinsLocalVolumesToSelectedSwarmNode(t *testing.T) {
	sanitizer := &ComposeSanitizer{
		deploymentID:      "compose-volume-placement-test",
		safeBaseDir:       t.TempDir(),
		swarmVolumeNodeID: "selected-node",
	}
	composeYAML := `services:
  app:
    image: example.invalid/app:latest
    volumes:
      - ./data:/data
    deploy:
      replicas: 2
      placement:
        constraints:
          - node.labels.pool == customer
          - node.id == old-node
`

	sanitized, err := sanitizer.SanitizeComposeYAML(composeYAML)
	if err != nil {
		t.Fatalf("sanitize Compose Swarm placement: %v", err)
	}
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitized), &compose); err != nil {
		t.Fatalf("parse sanitized Compose: %v", err)
	}
	services := compose["services"].(map[string]interface{})
	app := services["app"].(map[string]interface{})
	deploy := app["deploy"].(map[string]interface{})
	if deploy["replicas"] != 2 {
		t.Fatalf("deployment replicas changed: %#v", deploy["replicas"])
	}
	placement := deploy["placement"].(map[string]interface{})
	constraints := placement["constraints"].([]interface{})
	want := []interface{}{"node.labels.pool == customer", "node.id == selected-node"}
	if !reflect.DeepEqual(constraints, want) {
		t.Fatalf("placement constraints = %#v, want %#v", constraints, want)
	}
}

func TestSanitizeComposeYAMLDoesNotPinPortableSwarmVolumes(t *testing.T) {
	sanitizer := &ComposeSanitizer{
		deploymentID:      "compose-portable-volume-placement-test",
		safeBaseDir:       t.TempDir(),
		swarmVolumeNodeID: "selected-node",
	}
	composeYAML := `services:
  app:
    image: example.invalid/app:latest
    volumes:
      - /data
      - type: tmpfs
        target: /tmp
    deploy:
      placement:
        constraints:
          - node.role == worker
`

	sanitized, err := sanitizer.SanitizeComposeYAML(composeYAML)
	if err != nil {
		t.Fatalf("sanitize Compose with portable volumes: %v", err)
	}
	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(sanitized), &compose); err != nil {
		t.Fatalf("parse sanitized Compose: %v", err)
	}
	services := compose["services"].(map[string]interface{})
	app := services["app"].(map[string]interface{})
	deploy := app["deploy"].(map[string]interface{})
	placement := deploy["placement"].(map[string]interface{})
	want := []interface{}{"node.role == worker"}
	if constraints := placement["constraints"].([]interface{}); !reflect.DeepEqual(constraints, want) {
		t.Fatalf("portable-volume placement constraints = %#v, want %#v", constraints, want)
	}
}

func TestSanitizeComposeYAMLRejectsUnsafePathIdentifiers(t *testing.T) {
	composeYAML := `services:
  app:
    image: example.invalid/app:latest
`
	if _, err := NewComposeSanitizer("../escape").SanitizeComposeYAML(composeYAML); err == nil {
		t.Fatal("expected unsafe deployment identifier to be rejected")
	}

	sanitizer := &ComposeSanitizer{
		deploymentID: "compose-volume-name-test",
		safeBaseDir:  t.TempDir(),
	}
	unsafeVolumeYAML := composeYAML + `volumes:
  ../escape: {}
`
	if _, err := sanitizer.SanitizeComposeYAML(unsafeVolumeYAML); err == nil {
		t.Fatal("expected unsafe top-level volume name to be rejected")
	}
}
