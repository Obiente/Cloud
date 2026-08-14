package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSwarmIngressProxyServiceNameUsesCurrentStack(t *testing.T) {
	t.Setenv("TRAEFIK_SERVICE_NAME", "")
	t.Setenv("STACK_NAME", "obiente-dev")

	serviceName, err := swarmIngressProxyServiceName(context.Background())
	if err != nil {
		t.Fatalf("resolve ingress proxy service: %v", err)
	}
	if serviceName != "obiente-dev_traefik" {
		t.Fatalf("ingress proxy service = %q, want obiente-dev_traefik", serviceName)
	}
}

func TestSwarmIngressProxyServiceNameHonorsExplicitOverride(t *testing.T) {
	t.Setenv("TRAEFIK_SERVICE_NAME", "custom-edge")
	t.Setenv("STACK_NAME", "obiente-dev")

	serviceName, err := swarmIngressProxyServiceName(context.Background())
	if err != nil {
		t.Fatalf("resolve ingress proxy service: %v", err)
	}
	if serviceName != "custom-edge" {
		t.Fatalf("ingress proxy service = %q, want custom-edge", serviceName)
	}
}

func TestDockerNetworkMissingRecognizesDockerInspectMessage(t *testing.T) {
	if !dockerNetworkMissing("Error response from daemon: network preview-123 not found") {
		t.Fatal("expected Docker's inspect error to be treated as an already removed network")
	}
}

func TestSwarmServiceNetworkUpdateRemovesInspectedLegacyAttachments(t *testing.T) {
	t.Parallel()
	want := []string{
		"--network-rm", "older_obiente-network",
		"--network-rm", "current_obiente-network",
	}
	got := swarmServiceNetworkUpdateArgs([]string{
		"older_obiente-network",
		legacyPreviewIngressNetworkName,
		"current_obiente-network",
		"older_obiente-network",
	}, legacyPreviewIngressNetworkName)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("network update args = %#v, want %#v", got, want)
	}

	want = []string{
		"--network-add", legacyPreviewIngressNetworkName,
		"--network-rm", "custom_obiente-network",
	}
	got = swarmServiceNetworkUpdateArgs([]string{"custom_obiente-network"}, legacyPreviewIngressNetworkName)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("network migration args = %#v, want %#v", got, want)
	}
}

func TestSwarmNodePlacementUpdateArgsReconcilesLocalBindConstraint(t *testing.T) {
	t.Parallel()
	existing := []string{"node.labels.pool==customer", "node.id==old-node"}
	want := []string{
		"--constraint-rm", "node.id==old-node",
		"--constraint-add", "node.id==selected-node",
	}
	if got := swarmNodePlacementUpdateArgs(existing, "selected-node", true); !reflect.DeepEqual(got, want) {
		t.Fatalf("placement update args = %#v, want %#v", got, want)
	}

	want = []string{"--constraint-rm", "node.id==old-node"}
	if got := swarmNodePlacementUpdateArgs(existing, "", false); !reflect.DeepEqual(got, want) {
		t.Fatalf("placement removal args = %#v, want %#v", got, want)
	}
}

func TestParseSwarmTaskSummaryRecognizesStoppedServiceOutcomes(t *testing.T) {
	t.Parallel()

	completed := parseSwarmTaskSummary("deploy-app.1\tComplete 3 seconds ago\tShutdown\t\n")
	if completed.active || !completed.completed || completed.failed {
		t.Fatalf("completed task summary = %#v", completed)
	}

	active := parseSwarmTaskSummary("deploy-app.1\tRunning 2 seconds ago\tRunning\t\n")
	if !active.active || active.completed || active.failed {
		t.Fatalf("active task summary = %#v", active)
	}

	failed := parseSwarmTaskSummary("deploy-app.1\tRejected 1 second ago\tShutdown\tpermission denied\n")
	if failed.active || failed.completed || !failed.failed {
		t.Fatalf("failed task summary = %#v", failed)
	}

	currentFailureWithSuccessfulHistory := parseSwarmTaskSummary("deploy-app.1\tRejected 1 second ago\tShutdown\tpermission denied\n\\_ deploy-app.1\tComplete 1 minute ago\tShutdown\t\n")
	if currentFailureWithSuccessfulHistory.completed || !currentFailureWithSuccessfulHistory.failed {
		t.Fatalf("historical completion masked current failure: %#v", currentFailureWithSuccessfulHistory)
	}

	mixedCurrentOutcomes := parseSwarmTaskSummary("deploy-app.1\tComplete 3 seconds ago\tShutdown\t\ndeploy-app.2\tFailed 2 seconds ago\tShutdown\texit code 1\n")
	if !mixedCurrentOutcomes.completed || !mixedCurrentOutcomes.failed {
		t.Fatalf("mixed current task outcomes = %#v, want both completion and failure recorded", mixedCurrentOutcomes)
	}
}

func TestPreviewIngressNetworkIsUniquePerDeployment(t *testing.T) {
	t.Parallel()
	first := PreviewIngressNetworkNameForDeployment("preview-one")
	second := PreviewIngressNetworkNameForDeployment("preview-two")
	if first == second || first == legacyPreviewIngressNetworkName || second == legacyPreviewIngressNetworkName {
		t.Fatalf("preview ingress networks are not isolated: first=%q second=%q", first, second)
	}
}

func TestSwarmMemoryReservation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		limitBytes int64
		want       int64
	}{
		{
			name:       "zero limit",
			limitBytes: 0,
			want:       0,
		},
		{
			name:       "small limit uses floor",
			limitBytes: 256 * 1024 * 1024,
			want:       32 * 1024 * 1024,
		},
		{
			name:       "default two gigabyte limit reserves five percent",
			limitBytes: 2 * 1024 * 1024 * 1024,
			want:       107374182,
		},
		{
			name:       "large limit uses cap",
			limitBytes: 8 * 1024 * 1024 * 1024,
			want:       128 * 1024 * 1024,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := swarmMemoryReservation(tc.limitBytes); got != tc.want {
				t.Fatalf("swarmMemoryReservation(%d) = %d, want %d", tc.limitBytes, got, tc.want)
			}
		})
	}
}

func TestSwarmCPUReservation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		limitCores float64
		want       float64
	}{
		{
			name:       "zero limit",
			limitCores: 0,
			want:       0,
		},
		{
			name:       "small limit uses floor",
			limitCores: 0.25,
			want:       0.01,
		},
		{
			name:       "default two core limit reserves five hundredths",
			limitCores: 2,
			want:       0.05,
		},
		{
			name:       "large limit uses cap",
			limitCores: 8,
			want:       0.10,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := swarmCPUReservation(tc.limitCores); got != tc.want {
				t.Fatalf("swarmCPUReservation(%f) = %f, want %f", tc.limitCores, got, tc.want)
			}
		})
	}
}

func TestSwarmDisableHealthcheckArgs(t *testing.T) {
	t.Parallel()

	got := swarmDisableHealthcheckArgs()
	if len(got) != 1 || got[0] != "--no-healthcheck" {
		t.Fatalf("swarmDisableHealthcheckArgs() = %#v, want [--no-healthcheck]", got)
	}
}

func TestSwarmRestoreImageHealthcheckArgs(t *testing.T) {
	t.Parallel()

	got := swarmRestoreImageHealthcheckArgs()
	want := []string{
		"--health-cmd", "",
		"--health-interval", "0s",
		"--health-timeout", "0s",
		"--health-retries", "0",
		"--health-start-period", "0s",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("swarmRestoreImageHealthcheckArgs() = %#v, want %#v", got, want)
	}
}

func TestSwarmEnvUpdateArgsRemovesStaleEnv(t *testing.T) {
	t.Parallel()

	got := swarmEnvUpdateArgs(
		[]string{"APP_ENV", "OLD_SECRET", "PORT"},
		[]string{"APP_ENV=production", "PORT=3000", "NEW_FLAG=true"},
	)
	want := []string{
		"--env-rm", "OLD_SECRET",
		"--env-add", "APP_ENV=production",
		"--env-add", "PORT=3000",
		"--env-add", "NEW_FLAG=true",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("swarmEnvUpdateArgs() = %#v, want %#v", got, want)
	}
}

func TestSwarmServiceLabelUpdateArgsReconcilesManagedLabels(t *testing.T) {
	t.Parallel()

	existing := map[string]string{
		"cloud.obiente.managed":            "true",
		"cloud.obiente.traefik":            "true",
		"cloud.obiente.healthcheck_source": "platform",
		"traefik.docker.network":           "preview-network",
		"traefik.http.routers.old.rule":    "Host(`old.example.com`)",
		"com.example.external-label":       "keep",
	}
	desired := map[string]string{
		"cloud.obiente.managed":             "true",
		"cloud.obiente.traefik":             "true",
		"traefik.swarm.network":             "preview-network",
		"traefik.http.routers.current.rule": "Host(`preview.example.com`)",
	}
	want := []string{
		"--label-rm", "traefik.docker.network",
		"--label-rm", "traefik.http.routers.old.rule",
		"--label-add", "cloud.obiente.managed=true",
		"--label-add", "cloud.obiente.traefik=true",
		"--label-add", "traefik.http.routers.current.rule=Host(`preview.example.com`)",
		"--label-add", "traefik.swarm.network=preview-network",
	}
	if got := swarmServiceLabelUpdateArgs(existing, desired); !reflect.DeepEqual(got, want) {
		t.Fatalf("label update args = %#v, want %#v", got, want)
	}
}

func TestSwarmServiceLabelUpdateArgsRemovesDisabledRouting(t *testing.T) {
	t.Parallel()

	existing := map[string]string{
		"cloud.obiente.managed":         "true",
		"cloud.obiente.traefik":         "true",
		"traefik.enable":                "true",
		"traefik.http.routers.app.rule": "Host(`preview.example.com`)",
		"com.example.external-label":    "keep",
	}
	desired := map[string]string{"cloud.obiente.managed": "true"}
	want := []string{
		"--label-rm", "cloud.obiente.traefik",
		"--label-rm", "traefik.enable",
		"--label-rm", "traefik.http.routers.app.rule",
		"--label-add", "cloud.obiente.managed=true",
	}
	if got := swarmServiceLabelUpdateArgs(existing, desired); !reflect.DeepEqual(got, want) {
		t.Fatalf("disabled-routing label args = %#v, want %#v", got, want)
	}
}

func TestParseSwarmServiceEnvNames(t *testing.T) {
	t.Parallel()

	got := parseSwarmServiceEnvNames([]byte(`["PORT=3000","APP_ENV=production","PORT=4000","MALFORMED"]`))
	want := []string{"APP_ENV", "PORT"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("parseSwarmServiceEnvNames() = %#v, want %#v", got, want)
	}
}

func TestSwarmStartCommandUpdateArgsClearsStaleCommand(t *testing.T) {
	t.Parallel()

	got := swarmStartCommandUpdateArgs(nil)
	want := []string{"--entrypoint", "", "--args", ""}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("swarmStartCommandUpdateArgs(nil) = %#v, want %#v", got, want)
	}

	empty := "  "
	got = swarmStartCommandUpdateArgs(&empty)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("swarmStartCommandUpdateArgs(empty) = %#v, want %#v", got, want)
	}
}

func TestSwarmStartCommandUpdateArgsSetsShellCommandAtomically(t *testing.T) {
	t.Parallel()

	start := "npm run start && echo ready"
	got := swarmStartCommandUpdateArgs(&start)
	want := []string{"--entrypoint", "sh", "--args", "-c npm run start && echo ready"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("swarmStartCommandUpdateArgs() = %#v, want %#v", got, want)
	}
}

func TestDockerfileVolumeSanitizers(t *testing.T) {
	t.Parallel()

	validNames := []string{"data", "uploads.v1", "cache-dir", "_private"}
	for _, name := range validNames {
		name := name
		t.Run("valid name "+name, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeVolumeName(name); got != name {
				t.Fatalf("sanitizeVolumeName(%q) = %q, want %q", name, got, name)
			}
		})
	}

	invalidNames := []string{"", ".", "..", "../host", "data/slash", "bad name", "bad:mode", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	for _, name := range invalidNames {
		name := name
		t.Run("invalid name "+name, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeVolumeName(name); got != "" {
				t.Fatalf("sanitizeVolumeName(%q) = %q, want empty", name, got)
			}
		})
	}

	validMounts := map[string]string{
		"/data":          "/data",
		" /app/uploads/": "/app/uploads",
	}
	for input, want := range validMounts {
		input, want := input, want
		t.Run("valid mount "+input, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeContainerMountPath(input); got != want {
				t.Fatalf("sanitizeContainerMountPath(%q) = %q, want %q", input, got, want)
			}
		})
	}

	invalidMounts := []string{
		"",
		"data",
		"/",
		"/proc",
		"/proc/self",
		"/sys/kernel",
		"/dev/shm",
		"/app/../proc/self",
		"/var/run/docker.sock",
		"/run/docker.sock",
		"/data:rw",
		"/data\x00evil",
	}
	for _, mount := range invalidMounts {
		mount := mount
		t.Run("invalid mount "+mount, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeContainerMountPath(mount); got != "" {
				t.Fatalf("sanitizeContainerMountPath(%q) = %q, want empty", mount, got)
			}
		})
	}
}

func TestSanitizedVolumeMountsPrepareWritableAndReadOnlyRoots(t *testing.T) {
	volumeRoot := t.TempDir()
	deploymentID := "deploy-volume-permission-test"

	binds, mountFlags, err := sanitizedVolumeMountsAt(volumeRoot, deploymentID, []DeploymentVolume{
		{Name: "data", MountPath: "/data"},
		{Name: "assets", MountPath: "/assets", ReadOnly: true},
	})
	if err != nil {
		t.Fatalf("prepare sanitized volume mounts: %v", err)
	}

	writablePath := filepath.Join(volumeRoot, deploymentID, "data")
	readOnlyPath := filepath.Join(volumeRoot, deploymentID, "assets")
	wantBinds := []string{
		writablePath + ":/data",
		readOnlyPath + ":/assets:ro",
	}
	wantMountFlags := []string{
		"type=bind,src=" + writablePath + ",dst=/data",
		"type=bind,src=" + readOnlyPath + ",dst=/assets,readonly",
	}
	if !reflect.DeepEqual(binds, wantBinds) {
		t.Fatalf("binds = %#v, want %#v", binds, wantBinds)
	}
	if !reflect.DeepEqual(mountFlags, wantMountFlags) {
		t.Fatalf("mount flags = %#v, want %#v", mountFlags, wantMountFlags)
	}

	writableInfo, err := os.Stat(writablePath)
	if err != nil {
		t.Fatalf("stat writable volume root: %v", err)
	}
	if got := writableInfo.Mode().Perm(); got != 0o777 {
		t.Fatalf("writable volume mode = %#o, want 0777", got)
	}
	if writableInfo.Mode()&os.ModeSticky != 0 {
		t.Fatal("writable volume root unexpectedly has the sticky bit")
	}

	readOnlyInfo, err := os.Stat(readOnlyPath)
	if err != nil {
		t.Fatalf("stat read-only volume root: %v", err)
	}
	if got := readOnlyInfo.Mode().Perm(); got != 0o755 {
		t.Fatalf("read-only volume mode = %#o, want 0755", got)
	}
	if readOnlyInfo.Mode()&os.ModeSticky != 0 {
		t.Fatal("read-only volume root unexpectedly has the sticky bit")
	}
}

func TestSanitizedVolumeMountsRestoreReadOnlyRootPermissions(t *testing.T) {
	volumeRoot := t.TempDir()
	deploymentID := "deploy-read-only-transition-test"
	volume := DeploymentVolume{Name: "data", MountPath: "/data"}

	if _, _, err := sanitizedVolumeMountsAt(volumeRoot, deploymentID, []DeploymentVolume{volume}); err != nil {
		t.Fatalf("prepare writable volume root: %v", err)
	}
	volume.ReadOnly = true
	if _, _, err := sanitizedVolumeMountsAt(volumeRoot, deploymentID, []DeploymentVolume{volume}); err != nil {
		t.Fatalf("prepare read-only volume root: %v", err)
	}

	info, err := os.Stat(filepath.Join(volumeRoot, deploymentID, "data"))
	if err != nil {
		t.Fatalf("stat transitioned volume root: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Fatalf("transitioned read-only volume mode = %#o, want 0755", got)
	}
}

func TestSanitizedVolumeMountsKeepSharedRootWritable(t *testing.T) {
	volumeRoot := t.TempDir()
	deploymentID := "deploy-shared-volume-test"
	volumes := []DeploymentVolume{
		{Name: "shared", MountPath: "/data"},
		{Name: "shared", MountPath: "/snapshot", ReadOnly: true},
	}

	binds, _, err := sanitizedVolumeMountsAt(volumeRoot, deploymentID, volumes)
	if err != nil {
		t.Fatalf("prepare shared volume root: %v", err)
	}
	if len(binds) != 2 || !strings.HasSuffix(binds[0], ":/data") || !strings.HasSuffix(binds[1], ":/snapshot:ro") {
		t.Fatalf("shared volume binds = %#v", binds)
	}

	info, err := os.Stat(filepath.Join(volumeRoot, deploymentID, "shared"))
	if err != nil {
		t.Fatalf("stat shared volume root: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o777 {
		t.Fatalf("shared volume mode = %#o, want 0777", got)
	}
}

func TestSanitizedVolumeMountsFailClosedWhenPreparationFails(t *testing.T) {
	volumeRoot := t.TempDir()
	deploymentID := "deploy-volume-error-test"
	deploymentRoot := filepath.Join(volumeRoot, deploymentID)
	if err := os.MkdirAll(deploymentRoot, 0o755); err != nil {
		t.Fatalf("create deployment volume root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(deploymentRoot, "data"), []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("create conflicting volume path: %v", err)
	}

	binds, mountFlags, err := sanitizedVolumeMountsAt(volumeRoot, deploymentID, []DeploymentVolume{
		{Name: "data", MountPath: "/data"},
	})
	if err == nil {
		t.Fatal("expected volume preparation failure")
	}
	if len(binds) != 0 || len(mountFlags) != 0 {
		t.Fatalf("failed preparation returned mounts: binds=%#v flags=%#v", binds, mountFlags)
	}
}

func TestSanitizedVolumeMountsRestoreEarlierModesOnPreparationFailure(t *testing.T) {
	volumeRoot := t.TempDir()
	deploymentID := "deploy-volume-rollback-test"
	deploymentRoot := filepath.Join(volumeRoot, deploymentID)
	readOnlyPath := filepath.Join(deploymentRoot, "existing")
	conflictingPath := filepath.Join(deploymentRoot, "conflict")
	if err := os.MkdirAll(readOnlyPath, 0o777); err != nil {
		t.Fatalf("create existing writable root: %v", err)
	}
	if err := os.Chmod(readOnlyPath, 0o777); err != nil {
		t.Fatalf("set existing writable mode: %v", err)
	}
	if err := os.WriteFile(conflictingPath, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("create conflicting volume path: %v", err)
	}

	binds, mountFlags, err := sanitizedVolumeMountsAt(volumeRoot, deploymentID, []DeploymentVolume{
		{Name: "existing", MountPath: "/existing", ReadOnly: true},
		{Name: "conflict", MountPath: "/conflict"},
	})
	if err == nil {
		t.Fatal("expected volume preparation failure")
	}
	if len(binds) != 0 || len(mountFlags) != 0 {
		t.Fatalf("failed preparation returned mounts: binds=%#v flags=%#v", binds, mountFlags)
	}

	info, statErr := os.Stat(readOnlyPath)
	if statErr != nil {
		t.Fatalf("stat restored volume root: %v", statErr)
	}
	if got := info.Mode().Perm(); got != 0o777 {
		t.Fatalf("restored volume mode = %#o, want 0777", got)
	}
}

func TestHTTPHealthcheckCommandUsesSafePath(t *testing.T) {
	t.Parallel()

	got := httpHealthcheckCommand(3000, "/health", 204)
	if !strings.Contains(got, "http://127.0.0.1:3000/health") {
		t.Fatalf("httpHealthcheckCommand() = %q, want health URL", got)
	}
	if !strings.Contains(got, "expected=204") {
		t.Fatalf("httpHealthcheckCommand() = %q, want expected status", got)
	}

	injected := httpHealthcheckCommand(3000, "/health'; touch /tmp/owned; '", 200)
	if strings.Contains(injected, "touch /tmp/owned") || !strings.Contains(injected, "http://127.0.0.1:3000/") {
		t.Fatalf("httpHealthcheckCommand() did not neutralize unsafe path: %q", injected)
	}
	for _, packageManager := range []string{"apk add", "apt-get", "yum install"} {
		if strings.Contains(got, packageManager) {
			t.Fatalf("httpHealthcheckCommand() mutates the running image with %q: %q", packageManager, got)
		}
	}
	if !strings.Contains(got, "command -v node") {
		t.Fatalf("httpHealthcheckCommand() = %q, want node fallback", got)
	}
	if !strings.Contains(got, `awk "/HTTP\\//`) {
		t.Fatalf("httpHealthcheckCommand() = %q, want a closed HTTP status regex", got)
	}
	if !strings.Contains(got, `{print \$2; exit}`) {
		t.Fatalf("httpHealthcheckCommand() = %q, want wget to retain the initial response status", got)
	}
	if !strings.Contains(got, "http.client.HTTPConnection") {
		t.Fatalf("httpHealthcheckCommand() = %q, want Python status handling that supports expected non-2xx responses", got)
	}
}

func TestIsPlatformManagedHealthcheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
		test   []string
		want   bool
	}{
		{
			name:   "current platform label",
			source: "platform",
			test:   []string{"CMD-SHELL", "any command"},
			want:   true,
		},
		{
			name: "legacy generated tcp check",
			test: []string{"CMD-SHELL", "HEALTHCHECK_PORT=4321 node -e 'probe'"},
			want: true,
		},
		{
			name: "legacy runtime package check",
			test: []string{"CMD-SHELL", "apt-get install -y -qq netcat && nc -z localhost 4321"},
			want: true,
		},
		{
			name:   "disabled platform check",
			source: "disabled",
			want:   false,
		},
		{
			name: "image defined check",
			test: []string{"CMD-SHELL", "curl --fail http://localhost:8080/ready || exit 1"},
			want: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := isPlatformManagedHealthcheck(test.source, test.test); got != test.want {
				t.Fatalf("isPlatformManagedHealthcheck(%q, %q) = %t, want %t", test.source, test.test, got, test.want)
			}
		})
	}
}

func TestTCPHealthcheckCommandDoesNotInstallRuntimePackages(t *testing.T) {
	t.Parallel()

	explicit := tcpHealthcheckCommand(4321, false)
	automatic := tcpHealthcheckCommand(4321, true)
	for _, command := range []string{explicit, automatic} {
		for _, packageManager := range []string{"apk add", "apt-get", "yum install"} {
			if strings.Contains(command, packageManager) {
				t.Fatalf("tcpHealthcheckCommand() mutates the running image with %q: %q", packageManager, command)
			}
		}
		if !strings.Contains(command, "HEALTHCHECK_PORT=4321 node") {
			t.Fatalf("tcpHealthcheckCommand() = %q, want node fallback", command)
		}
		if strings.Contains(command, "busybox nc -z") {
			t.Fatalf("tcpHealthcheckCommand() uses unsupported BusyBox nc -z: %q", command)
		}
	}
	if !strings.Contains(explicit, "exit 1; fi") {
		t.Fatalf("explicit tcp health check must fail without a probe: %q", explicit)
	}
	if !strings.Contains(automatic, "exit 0; fi") {
		t.Fatalf("automatic tcp health check must not fail solely because the image has no probe: %q", automatic)
	}
}

func TestRollbackPreserved(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("update deployment: %w", &SwarmRolloutError{
		ServiceName:             "deploy-example-default",
		State:                   "rollback_completed",
		PreviousRevisionRunning: true,
	})
	if !RollbackPreserved(err) {
		t.Fatal("RollbackPreserved() = false, want true for wrapped completed rollback")
	}
	if RollbackPreserved(errors.New("ordinary failure")) {
		t.Fatal("RollbackPreserved() = true for ordinary failure")
	}
}
