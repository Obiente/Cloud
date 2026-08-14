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

	"github.com/obiente/cloud/apps/shared/pkg/database"

	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v3"
)

func TestComposeUpArgs(t *testing.T) {
	got := composeUpArgs("deploy-123", "/tmp/deploy/docker-compose.yml")
	want := []string{
		"compose",
		"-p",
		"deploy-123",
		"-f",
		"/tmp/deploy/docker-compose.yml",
		"up",
		"-d",
		"--build",
		"--pull",
		"always",
		"--force-recreate",
		"--remove-orphans",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("composeUpArgs mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestStackRollbackRestoresVolumePreparationOnlyForSingleService(t *testing.T) {
	rollbackErr := fmt.Errorf("wait for service: %w", &SwarmRolloutError{
		ServiceName:             "deploy-example_worker",
		State:                   "rollback_completed",
		PreviousRevisionRunning: true,
	})
	if !stackRollbackRestoresVolumePreparation([]string{"deploy-example_worker"}, rollbackErr) {
		t.Fatal("single-service completed rollback should restore prepared volume modes")
	}
	if stackRollbackRestoresVolumePreparation([]string{"deploy-example_api", "deploy-example_worker"}, rollbackErr) {
		t.Fatal("multi-service rollback must retain prepared volume modes")
	}
	if stackRollbackRestoresVolumePreparation([]string{"deploy-example_worker"}, errors.New("update failed")) {
		t.Fatal("unconfirmed rollback must retain prepared volume modes")
	}
}

func TestUnchangedSwarmStackServicesUsesTaskTemplates(t *testing.T) {
	before := map[string]string{
		"deploy-app_api":    `{"Name":"api","TaskTemplate":{"ContainerSpec":{"Image":"example/api:v1"}}}`,
		"deploy-app_worker": `{"Name":"worker","TaskTemplate":{"ContainerSpec":{"Image":"example/worker:v1"}}}`,
	}
	after := map[string]string{
		"deploy-app_api":    `{"Name":"api","TaskTemplate":{"ContainerSpec":{"Image":"example/api:v2"}}}`,
		"deploy-app_worker": before["deploy-app_worker"],
		"deploy-app_new":    `{"Name":"new"}`,
	}
	want := []string{"deploy-app_worker"}
	if got := unchangedSwarmStackServices(before, after); !reflect.DeepEqual(got, want) {
		t.Fatalf("unchangedSwarmStackServices() = %#v, want %#v", got, want)
	}
}

func TestParseCurrentSwarmTaskIDsIgnoresHistoricalTasks(t *testing.T) {
	got := parseCurrentSwarmTaskIDs("deploy-app.1\tcurrent-a\n\\_ deploy-app.1\told-a\ndeploy-app.2\tcurrent-b\n")
	if _, found := got["old-a"]; found {
		t.Fatal("historical task was treated as current")
	}
	for _, taskID := range []string{"current-a", "current-b"} {
		if _, found := got[taskID]; !found {
			t.Fatalf("current task %s was omitted", taskID)
		}
	}
}

func TestObsoleteComposeSwarmLocationsKeepsCurrentAndLegacyRows(t *testing.T) {
	locations := []database.DeploymentLocation{
		{ContainerID: "current-container", ServiceID: "service-example", TaskID: "current-task"},
		{ContainerID: "stale-container", ServiceID: "service-example", TaskID: "stale-task"},
		{ContainerID: "plain-container"},
	}
	current := map[string]struct{}{"current-task": {}}
	got := obsoleteComposeSwarmLocations(locations, current)
	if len(got) != 1 || got[0].ContainerID != "stale-container" {
		t.Fatalf("obsolete locations = %#v, want only stale-container", got)
	}
}

func TestSwarmTaskSlotUsesStableReplicaIdentity(t *testing.T) {
	labels := map[string]string{
		"com.docker.swarm.service.name": "deploy-example_api",
		"com.docker.swarm.task.name":    "deploy-example_api.2.current-task",
	}
	if got := swarmTaskSlot(labels); got != "2" {
		t.Fatalf("swarmTaskSlot() = %q, want %q", got, "2")
	}
	labels["com.docker.swarm.task.name"] = "unrelated.2.current-task"
	if got := swarmTaskSlot(labels); got != "" {
		t.Fatalf("swarmTaskSlot() accepted unrelated task name: %q", got)
	}
}

func TestParseLegacyProjectRootMetadataUsesRecordedManagedRoot(t *testing.T) {
	deploymentID := "compose-metadata-root-test"
	recordedRoot := filepath.Join("/tmp/obiente-volumes", deploymentID)
	if err := os.MkdirAll(recordedRoot, 0o755); err != nil {
		t.Fatalf("create recorded volume root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(recordedRoot) })

	got, err := parseLegacyProjectRootMetadata([]byte(legacyProjectRootMetadataContents(recordedRoot)), deploymentID)
	if err != nil {
		t.Fatalf("parse recorded volume root: %v", err)
	}
	if got != recordedRoot {
		t.Fatalf("recorded volume root = %q, want %q", got, recordedRoot)
	}
	if _, err := parseLegacyProjectRootMetadata([]byte("legacy-relative-project-root-v1\n/etc\n"), deploymentID); err == nil {
		t.Fatal("metadata outside managed volume roots should be rejected")
	}
}

func TestRecordedLegacyProjectRootPrecedesNewRootSelection(t *testing.T) {
	deploymentID := fmt.Sprintf("compose-recorded-root-test-%d", os.Getpid())
	recordedRoot := filepath.Join("/tmp/obiente-volumes", deploymentID)
	deployDir := filepath.Join("/tmp/obiente-deployments", deploymentID)
	if err := os.MkdirAll(recordedRoot, 0o755); err != nil {
		t.Fatalf("create recorded volume root: %v", err)
	}
	if err := os.MkdirAll(deployDir, 0o755); err != nil {
		t.Fatalf("create deployment metadata directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(recordedRoot)
		_ = os.RemoveAll(deployDir)
	})
	if err := persistLegacyProjectRootMetadata(deployDir, recordedRoot); err != nil {
		t.Fatalf("persist recorded volume root: %v", err)
	}

	got, found, err := recordedLegacyProjectRoot(deploymentID)
	if err != nil {
		t.Fatalf("read recorded volume root: %v", err)
	}
	if !found || got != recordedRoot {
		t.Fatalf("recorded volume root found=%t root=%q, want %q", found, got, recordedRoot)
	}
}

func TestIsMissingSwarmStackOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{name: "docker missing stack", output: "Nothing found in stack: deploy-example", want: true},
		{name: "case insensitive", output: "nothing found in stack: deploy-example", want: true},
		{name: "daemon failure", output: "error during connect: connection refused", want: false},
		{name: "missing docker binary", output: "executable file not found", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isMissingSwarmStackOutput(test.output); got != test.want {
				t.Fatalf("isMissingSwarmStackOutput(%q) = %t, want %t", test.output, got, test.want)
			}
		})
	}
}

func TestPersistLegacyProjectRootMetadata(t *testing.T) {
	deployDir := t.TempDir()
	volumeRoot := filepath.Join(t.TempDir(), "volumes", "deploy-example")
	if err := persistLegacyProjectRootMetadata(deployDir, volumeRoot); err != nil {
		t.Fatalf("persist legacy project-root metadata: %v", err)
	}
	contents, found, err := readDeploymentFileNoFollow(deployDir, legacyProjectRootMetadataFile)
	if err != nil {
		t.Fatalf("read legacy project-root metadata: %v", err)
	}
	if !found || string(contents) != legacyProjectRootMetadataContents(volumeRoot) {
		t.Fatalf("metadata found=%t contents=%q", found, contents)
	}
	if err := persistLegacyProjectRootMetadata(deployDir, volumeRoot); err != nil {
		t.Fatalf("repeat metadata persistence: %v", err)
	}
	if err := persistLegacyProjectRootMetadata(deployDir, filepath.Join(t.TempDir(), "different")); err == nil {
		t.Fatal("expected mismatched volume-root metadata to be rejected")
	}
}

func TestPersistLegacyProjectRootMetadataRejectsSymlink(t *testing.T) {
	deployDir := t.TempDir()
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, []byte("preserve me"), 0o600); err != nil {
		t.Fatalf("create symlink target: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(deployDir, legacyProjectRootMetadataFile)); err != nil {
		t.Fatalf("create metadata symlink: %v", err)
	}
	if err := persistLegacyProjectRootMetadata(deployDir, filepath.Join(t.TempDir(), "volumes")); err == nil {
		t.Fatal("expected metadata symlink to be rejected")
	}
	contents, err := os.ReadFile(target)
	if err != nil || string(contents) != "preserve me" {
		t.Fatalf("symlink target changed: contents=%q err=%v", contents, err)
	}
}

func TestRemoveIncompleteLegacyProjectRootMetadata(t *testing.T) {
	deployDir := t.TempDir()
	dirFD, err := secureOpenDirectory(deployDir, false)
	if err != nil {
		t.Fatalf("open deployment directory: %v", err)
	}
	defer unix.Close(dirFD)
	fileFD, err := unix.Openat(dirFD, legacyProjectRootMetadataFile, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		t.Fatalf("create incomplete metadata: %v", err)
	}
	if _, err := unix.Write(fileFD, []byte("partial")); err != nil {
		unix.Close(fileFD)
		t.Fatalf("write incomplete metadata: %v", err)
	}
	if err := unix.Close(fileFD); err != nil {
		t.Fatalf("close incomplete metadata: %v", err)
	}
	if err := removeIncompleteLegacyProjectRootMetadata(dirFD); err != nil {
		t.Fatalf("remove incomplete metadata: %v", err)
	}
	if _, found, err := readDeploymentFileNoFollow(deployDir, legacyProjectRootMetadataFile); err != nil || found {
		t.Fatalf("incomplete metadata still present: found=%t err=%v", found, err)
	}
}

func TestDeploymentRuntimeChangedAfterFailure(t *testing.T) {
	tests := []struct {
		name    string
		before  string
		after   string
		err     error
		changed bool
	}{
		{name: "daemon rejected before mutation", before: "container-a", after: "container-a", changed: false},
		{name: "partial replacement", before: "container-a", after: "container-b", changed: true},
		{name: "unknown post-state", before: "container-a", err: errors.New("daemon unavailable"), changed: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := deploymentRuntimeChangedAfterFailure(test.before, func(context.Context) (string, error) {
				return test.after, test.err
			})
			if got != test.changed {
				t.Fatalf("deploymentRuntimeChangedAfterFailure = %t, want %t", got, test.changed)
			}
		})
	}
}

func TestInjectSwarmRollingUpdatePolicy(t *testing.T) {
	input := `services:
  app:
    image: example/app:latest
    deploy:
      replicas: 2
      restart_policy:
        condition: on-failure
`

	got, err := injectSwarmRollingUpdatePolicy(input)
	if err != nil {
		t.Fatalf("injectSwarmRollingUpdatePolicy returned error: %v", err)
	}
	if !strings.Contains(got, "order: start-first") {
		t.Fatalf("expected start-first update order in output:\n%s", got)
	}

	var compose map[string]interface{}
	if err := yaml.Unmarshal([]byte(got), &compose); err != nil {
		t.Fatalf("output is not valid YAML: %v", err)
	}
	services := compose["services"].(map[string]interface{})
	app := services["app"].(map[string]interface{})
	deploy := app["deploy"].(map[string]interface{})
	if deploy["replicas"] != 2 {
		t.Fatalf("existing deploy replicas were not preserved: %#v", deploy["replicas"])
	}
	update := deploy["update_config"].(map[string]interface{})
	if update["order"] != "start-first" || update["failure_action"] != "rollback" {
		t.Fatalf("unexpected update config: %#v", update)
	}
	rollback := deploy["rollback_config"].(map[string]interface{})
	if rollback["order"] != "start-first" {
		t.Fatalf("unexpected rollback config: %#v", rollback)
	}
}

func TestStackDeployArgs(t *testing.T) {
	got := stackDeployArgs("deploy-123", "/tmp/deploy/docker-compose.yml")
	want := []string{
		"stack",
		"deploy",
		"-c",
		"/tmp/deploy/docker-compose.yml",
		"--with-registry-auth=true",
		"--resolve-image",
		"always",
		"--prune",
		"deploy-123",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stackDeployArgs mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}
