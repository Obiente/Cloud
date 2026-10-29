package docker

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestShelllessContainerFileBrowser(t *testing.T) {
	if os.Getenv("OBIENTE_DOCKER_TESTS") != "1" {
		t.Skip("requires isolated Docker")
	}
	if os.Getenv("DOCKER_HOST") == "" {
		t.Fatal("explicit DOCKER_HOST required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	run := func(args ...string) string {
		t.Helper()
		out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
		if err != nil {
			t.Fatalf("docker %s: %v: %s", args[0], err, out)
		}
		return strings.TrimSpace(string(out))
	}
	name := "obiente-files-test-" + uuid.NewString()
	volume := name + "-data"
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		_ = exec.CommandContext(cleanup, "docker", "rm", "-fv", name).Run()
		_ = exec.CommandContext(cleanup, "docker", "volume", "rm", volume).Run()
	})
	run("volume", "create", volume)
	id := run("run", "-d", "--name", name, "--network", "none", "--mount", "type=volume,source="+volume+",target=/data", "busybox:1.37.0", "sleep", "600")
	// Remove file utilities from the disposable container while retaining its
	// runtime libraries. The running process is unaffected by unlinking sleep.
	run("exec", name, "/bin/busybox", "sh", "-c", `mkdir /app; echo 'container content' > '/app/hello world.txt'; echo persistent-content > /data/record; ln -s /data/record /data/link; for f in /bin/*; do [ "$f" = /bin/busybox ] || /bin/busybox rm "$f"; done`)
	c, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	for _, stopped := range []bool{false, true} {
		if stopped {
			run("stop", "-t", "1", name)
		}
		for _, directory := range []string{"/", "/app"} {
			files, err := c.ContainerListFiles(ctx, id, directory)
			if err != nil {
				t.Fatalf("list %s (stopped=%v): %v", directory, stopped, err)
			}
			want := "app"
			if directory == "/app" {
				want = "hello world.txt"
			}
			found := false
			for _, f := range files {
				if f.Name == want {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing %s: %+v", want, files)
			}
		}
		data, err := c.ContainerReadFile(ctx, id, "/app/hello world.txt")
		if err != nil || string(data) != "container content\n" {
			t.Fatalf("read: %q %v", data, err)
		}
		volumes, err := c.GetContainerVolumes(ctx, id)
		if err != nil || len(volumes) != 1 {
			t.Fatalf("mount discovery: %+v %v", volumes, err)
		}
		files, err := c.ListMountedVolumeFiles(ctx, id, volumes[0], "/")
		if err != nil || len(files) != 2 {
			t.Fatalf("volume listing: %+v %v", files, err)
		}
		for _, p := range []string{"/record", "/link"} {
			data, _, err := c.ReadMountedVolumeFile(ctx, id, volumes[0], p)
			if err != nil || string(data) != "persistent-content\n" {
				t.Fatalf("volume read %s: %q %v", p, data, err)
			}
		}
		if _, err := c.ContainerReadFile(ctx, id, "/missing"); err == nil {
			t.Fatal("missing path succeeded")
		}
		state, err := c.ContainerInspect(ctx, id)
		if err != nil || state.State.Running == stopped {
			t.Fatalf("browse changed workload state: %+v %v", state.State, err)
		}
	}
}
