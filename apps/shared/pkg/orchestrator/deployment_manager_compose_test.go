package orchestrator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

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

func TestParseCurrentSwarmTaskGenerationIgnoresHistoryAndOrder(t *testing.T) {
	first := parseCurrentSwarmTaskGeneration("deploy-app.2\ttask-b\n\\_ deploy-app.1\told-task\ndeploy-app.1\ttask-a\n")
	second := parseCurrentSwarmTaskGeneration("deploy-app.1\ttask-a\ndeploy-app.2\ttask-b\n")
	if first != "task-a,task-b" || second != first {
		t.Fatalf("task generations = %q and %q, want stable current generation", first, second)
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
