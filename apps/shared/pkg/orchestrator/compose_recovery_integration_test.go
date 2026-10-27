package orchestrator

import (
	"context"
	"github.com/google/uuid"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func recoveryDocker(t *testing.T, ctx context.Context, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			t.Fatalf("docker %s: %v: %s", args[0], err, e.Stderr)
		}
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}
func TestComposeRecoveryPreservesHealthyReplicasAndCompletedJobs(t *testing.T) {
	if os.Getenv("OBIENTE_COMPOSE_TESTS") != "1" {
		t.Skip("requires isolated Docker with Compose")
	}
	if os.Getenv("DOCKER_HOST") == "" {
		t.Fatal("explicit DOCKER_HOST required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	project := "obiente-test-" + uuid.NewString()
	file := filepath.Join(t.TempDir(), "compose.yml")
	content := `services:
  init:
    image: busybox:1.37.0
    network_mode: none
    command: [sh, -c, "echo preserved >> /data/record"]
    volumes: [data:/data]
  worker:
    image: busybox:1.37.0
    network_mode: none
    command: [sleep, '600']
    depends_on:
      init:
        condition: service_completed_successfully
    deploy:
      replicas: 2
    volumes: [data:/data]
volumes:
  data:
`
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		_ = exec.CommandContext(cleanup, "docker", "compose", "-p", project, "-f", file, "down", "-v").Run()
	})
	recoveryDocker(t, ctx, "compose", "-p", project, "-f", file, "up", "-d", "--pull", "missing")
	ids := strings.Fields(recoveryDocker(t, ctx, "compose", "-p", project, "-f", file, "ps", "-q", "worker"))
	if len(ids) != 2 {
		t.Fatalf("expected 2 workers, got %v", ids)
	}
	job := recoveryDocker(t, ctx, "compose", "-p", project, "-f", file, "ps", "-a", "-q", "init")
	recoveryDocker(t, ctx, "rm", "-f", ids[1])
	args := append(composeRecoveryArgs(project, file), "--no-deps", "worker")
	for i := 0; i < 2; i++ {
		recoveryDocker(t, ctx, args...)
	}
	after := strings.Fields(recoveryDocker(t, ctx, "compose", "-p", project, "-f", file, "ps", "-q", "worker"))
	if len(after) != 2 || (after[0] != ids[0] && after[1] != ids[0]) {
		t.Fatalf("healthy replica changed or missing replica not restored: %v", after)
	}
	if recoveryDocker(t, ctx, "compose", "-p", project, "-f", file, "ps", "-a", "-q", "init") != job {
		t.Fatal("completed job replaced")
	}
	if record := recoveryDocker(t, ctx, "exec", ids[0], "cat", "/data/record"); record != "preserved" {
		t.Fatalf("completed job replayed or data changed: %q", record)
	}
}

func TestPartialSwarmRecoveryDoesNotRollHealthyService(t *testing.T) {
	if os.Getenv("OBIENTE_SWARM_TESTS") != "1" {
		t.Skip("requires isolated Swarm")
	}
	if os.Getenv("DOCKER_HOST") == "" {
		t.Fatal("explicit DOCKER_HOST required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	project := "obiente-test-" + uuid.NewString()
	file := filepath.Join(t.TempDir(), "compose.yml")
	content := []byte(`version: '3.8'
services:
  healthy:
    image: busybox:1.37.0
    command: [sleep, '600']
    networks: [hostnet]
  missing:
    image: busybox:1.37.0
    command: [sleep, '600']
    networks: [hostnet]
networks:
  hostnet:
    external: true
    name: host
`)
	if err := os.WriteFile(file, content, 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 20*time.Second)
		defer stop()
		_ = exec.CommandContext(cleanup, "docker", "stack", "rm", project).Run()
	})
	recoveryDocker(t, ctx, stackRecoveryArgs(project, file)...)
	wait := func() {
		t.Helper()
		for {
			rows := recoveryDocker(t, ctx, "stack", "services", project, "--format", "{{.Replicas}}")
			if rows == "1/1\n1/1" {
				return
			}
			select {
			case <-ctx.Done():
				t.Fatalf("stack did not converge: %s", rows)
			case <-time.After(time.Second):
			}
		}
	}
	wait()
	healthy := project + "_healthy"
	before := recoveryDocker(t, ctx, "service", "inspect", healthy, "--format", "{{.ID}} {{.Version.Index}}")
	task := recoveryDocker(t, ctx, "service", "ps", healthy, "--filter", "desired-state=running", "--format", "{{.ID}}")
	recoveryDocker(t, ctx, "service", "rm", project+"_missing")
	subset, err := missingComposeServices(content, []string{"missing"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, subset, 0600); err != nil {
		t.Fatal(err)
	}
	recoveryDocker(t, ctx, stackRecoveryArgs(project, file)...)
	wait()
	if recoveryDocker(t, ctx, "service", "inspect", healthy, "--format", "{{.ID}} {{.Version.Index}}") != before {
		t.Fatal("healthy service spec was rewritten")
	}
	if recoveryDocker(t, ctx, "service", "ps", healthy, "--filter", "desired-state=running", "--format", "{{.ID}}") != task {
		t.Fatal("healthy task was replaced")
	}
}
