package docker

import (
	"context"
	"github.com/google/uuid"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestPostgresRecordsSurviveNetworkReplacement(t *testing.T) {
	if os.Getenv("OBIENTE_DOCKER_TESTS") != "1" {
		t.Skip("requires an isolated Docker daemon")
	}
	if os.Getenv("DOCKER_HOST") == "" {
		t.Fatal("explicit DOCKER_HOST required")
	}
	for _, driver := range []string{"bridge", "overlay"} {
		t.Run(driver, func(t *testing.T) {
			if driver == "overlay" && os.Getenv("OBIENTE_SWARM_TESTS") != "1" {
				t.Skip("overlay recovery requires an isolated Swarm manager")
			}
			networkArgs := []string{"network", "create", "--driver", driver}
			if driver == "overlay" {
				networkArgs = append(networkArgs, "--attachable")
			} else {
				networkArgs = append(networkArgs, "--opt", "com.docker.network.bridge.enable_ip_masquerade=false")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			run := func(args ...string) string {
				t.Helper()
				out, err := exec.CommandContext(ctx, "docker", args...).Output()
				if err != nil {
					t.Fatalf("docker %s: %v: %s", strings.Join(args, " "), err, err.(*exec.ExitError).Stderr)
				}
				return strings.TrimSpace(string(out))
			}
			name := "obiente-test-" + uuid.NewString()
			net := name + "-net"
			t.Cleanup(func() {
				cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
				defer stop()
				_ = exec.CommandContext(cleanup, "docker", "rm", "-fv", name).Run()
				_ = exec.CommandContext(cleanup, "docker", "network", "rm", net).Run()
			})
			run(append(networkArgs, net)...)
			id := run("run", "-d", "--name", name, "--network", net, "-e", "POSTGRES_HOST_AUTH_METHOD=trust", "postgres:17-alpine")
			ready := func() {
				t.Helper()
				for i := 0; i < 60; i++ {
					if exec.CommandContext(ctx, "docker", "exec", name, "pg_isready", "-h", "127.0.0.1", "-U", "postgres").Run() == nil {
						return
					}
					select {
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					case <-time.After(250 * time.Millisecond):
					}
				}
				t.Fatal("Postgres did not become ready")
			}
			ready()
			run("exec", name, "psql", "-U", "postgres", "-v", "ON_ERROR_STOP=1", "-c", "CREATE TABLE recovery_fixture(id integer PRIMARY KEY, value text); INSERT INTO recovery_fixture VALUES (1, 'preserved'), (2, 'also-preserved');")
			run("stop", name)
			run("network", "disconnect", "-f", net, name)
			run("network", "rm", net)
			// Overlay deletion is asynchronous in Swarm. Wait for the old
			// network name to disappear before creating its replacement.
			for {
				if exec.CommandContext(ctx, "docker", "network", "inspect", net).Run() != nil {
					break
				}
				select {
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				case <-time.After(100 * time.Millisecond):
				}
			}
			networkID := run(append(networkArgs, net)...)
			c, err := New()
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			if err = c.EnsureContainerNetwork(ctx, id, net); err != nil {
				t.Fatalf("%v: %s", err, run("inspect", "--format", "{{json .NetworkSettings.Networks}}", name))
			}
			run("start", name)
			ready()
			if err = c.EnsureContainerNetwork(ctx, id, net); err != nil {
				t.Fatal(err)
			}
			inspect, err := c.ContainerInspect(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if !networkEndpointUsable(inspect, networkID, net) {
				t.Fatal("replacement network endpoint unavailable")
			}
			records := run("exec", name, "psql", "-U", "postgres", "-At", "-c", "SELECT id || ':' || value FROM recovery_fixture ORDER BY id")
			if records != "1:preserved\n2:also-preserved" {
				t.Fatalf("records changed: %q", records)
			}
		})
	}
}

func TestSwarmRecoversBothCleanAndFailedExits(t *testing.T) {
	if os.Getenv("OBIENTE_SWARM_TESTS") != "1" {
		t.Skip("requires an isolated Swarm manager")
	}
	if os.Getenv("DOCKER_HOST") == "" {
		t.Fatal("explicit DOCKER_HOST required")
	}
	for _, code := range []string{"0", "1"} {
		t.Run("exit-"+code, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			defer cancel()
			name := "obiente-test-" + uuid.NewString()
			t.Cleanup(func() {
				cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
				defer stop()
				_ = exec.CommandContext(cleanup, "docker", "service", "rm", name).Run()
			})
			out, err := exec.CommandContext(ctx, "docker", "service", "create", "--detach", "--name", name, "--no-resolve-image", "--network", "host", "--restart-condition", "any", "--restart-delay", "1s", "--restart-max-attempts", "0", "busybox:1.37.0", "sh", "-c", "exit "+code).CombinedOutput()
			if err != nil {
				t.Fatalf("create service: %v: %s", err, out)
			}
			for {
				out, err = exec.CommandContext(ctx, "docker", "service", "ps", name, "--format", "{{.ID}}").Output()
				if err == nil && len(strings.Fields(string(out))) >= 3 {
					args := append([]string{"inspect", "--type", "task", "--format", "{{.Status.State}} {{.Status.ContainerStatus.ExitCode}}"}, strings.Fields(string(out))...)
					states, inspectErr := exec.CommandContext(ctx, "docker", args...).Output()
					if inspectErr == nil {
						exited := 0
						for _, line := range strings.Split(string(states), "\n") {
							fields := strings.Fields(line)
							if len(fields) == 2 && fields[1] == code && (fields[0] == "complete" || fields[0] == "failed") {
								exited++
							}
						}
						if exited >= 2 {
							return
						}
					}
				}
				select {
				case <-ctx.Done():
					t.Fatalf("service did not repeatedly recover after exit %s: %s", code, out)
				case <-time.After(time.Second):
				}
			}
		})
	}
}
