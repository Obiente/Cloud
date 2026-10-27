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

// Opt-in: uses only uniquely named disposable resources on the selected daemon.
func TestDockerStorageAndNetworkRecovery(t *testing.T) {
	if os.Getenv("OBIENTE_DOCKER_TESTS") != "1" {
		t.Skip("set OBIENTE_DOCKER_TESTS=1 with an isolated DOCKER_HOST")
	}
	if os.Getenv("DOCKER_HOST") == "" {
		t.Fatal("an explicit isolated DOCKER_HOST is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	run := func(args ...string) string {
		t.Helper()
		out, err := exec.CommandContext(ctx, "docker", args...).Output()
		if err != nil {
			t.Fatalf("docker %s: %v: %s", args[0], err, out)
		}
		return strings.TrimSpace(string(out))
	}
	prefix := "obiente-test-" + uuid.NewString()
	volume, networkName := prefix+"-data", prefix+"-net"
	first, second := prefix+"-a", prefix+"-b"
	run("volume", "create", volume)
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		_ = exec.CommandContext(cleanup, "docker", "rm", "-fv", first, second).Run()
		_ = exec.CommandContext(cleanup, "docker", "network", "rm", networkName).Run()
		_ = exec.CommandContext(cleanup, "docker", "volume", "rm", volume).Run()
	})
	run("run", "--rm", "--network", "none", "--mount", "type=volume,source="+volume+",target=/data", "busybox:1.37.0", "sh", "-c", "dd if=/dev/zero of=/data/fixture bs=1024 count=8192 && echo preserved > /data/record")
	oldNetwork := run("network", "create", "--driver", "bridge", "--opt", "com.docker.network.bridge.enable_ip_masquerade=false", networkName)
	ids := []string{}
	for _, name := range []string{first, second} {
		net := "none"
		if name == first {
			net = networkName
		}
		ids = append(ids, run("run", "-d", "--name", name, "--network", net, "--mount", "type=volume,source="+volume+",target=/data", "busybox:1.37.0", "sleep", "600"))
	}
	info, err := CalculateStorage(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}
	if info.VolumeSize < 8*1024*1024 || info.VolumeSize > 9*1024*1024 {
		t.Fatalf("shared volume counted incorrectly: %+v", info)
	}
	if info.ImageSize != 0 || info.TotalStorage != info.VolumeSize+info.ContainerDisk {
		t.Fatalf("immutable images included: %+v", info)
	}
	c, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.EnsureContainerNetwork(ctx, ids[0], networkName); err != nil {
		t.Fatal(err)
	}
	if err := c.EnsureContainerNetwork(ctx, ids[0], networkName); err != nil {
		t.Fatal(err)
	}
	run("stop", first)
	run("network", "disconnect", "-f", networkName, first)
	run("network", "rm", networkName)
	newNetwork := run("network", "create", "--driver", "bridge", "--opt", "com.docker.network.bridge.enable_ip_masquerade=false", networkName)
	if oldNetwork == newNetwork {
		t.Fatal("test network was not replaced")
	}
	if err := c.EnsureContainerNetwork(ctx, ids[0], networkName); err != nil {
		t.Fatal(err)
	}
	run("start", first)
	if err := c.EnsureContainerNetwork(ctx, ids[0], networkName); err != nil {
		t.Fatal(err)
	}
	if record := run("exec", first, "cat", "/data/record"); record != "preserved" {
		t.Fatalf("data changed: %q", record)
	}
	if got := run("exec", first, "wc", "-c", "/data/fixture"); !strings.HasPrefix(got, "8388608") {
		t.Fatalf("fixture changed: %s", got)
	}
	inspected, err := c.ContainerInspect(ctx, ids[0])
	if err != nil {
		t.Fatal(err)
	}
	if !networkEndpointUsable(inspected, newNetwork, networkName) {
		t.Fatal("replacement endpoint not usable")
	}
}

func TestRetainedStorageIsMeasuredAfterContainerRemoval(t *testing.T) {
	if os.Getenv("OBIENTE_DOCKER_TESTS") != "1" {
		t.Skip("requires isolated Docker")
	}
	if os.Getenv("DOCKER_HOST") == "" {
		t.Fatal("explicit DOCKER_HOST required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	name := "obiente-test-" + uuid.NewString()
	volume := name + "-data"
	run := func(args ...string) string {
		t.Helper()
		out, err := exec.CommandContext(ctx, "docker", args...).Output()
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(out))
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		_ = exec.CommandContext(cleanup, "docker", "rm", "-f", name).Run()
		_ = exec.CommandContext(cleanup, "docker", "volume", "rm", volume).Run()
	})
	run("volume", "create", volume)
	id := run("run", "-d", "--name", name, "--network", "none", "--mount", "type=volume,source="+volume+",target=/data", "busybox:1.37.0", "sleep", "600")
	run("exec", name, "sh", "-c", "dd if=/dev/zero of=/data/fixture bs=1024 count=8192")
	saved, err := CaptureStorageMounts(ctx, []string{id})
	if err != nil {
		t.Fatal(err)
	}
	run("rm", "-f", name)
	info, err := CalculateStorageWithRetainedMounts(ctx, []string{id}, saved, true)
	if err != nil {
		t.Fatal(err)
	}
	if info.ContainerDisk != 0 || info.VolumeSize < 8*1024*1024 || info.VolumeSize > 9*1024*1024 {
		t.Fatalf("wrong retained size: %+v", info)
	}
	run("volume", "rm", volume)
	info, err = CalculateStorageWithRetainedMounts(ctx, nil, saved, true)
	if err != nil {
		t.Fatal(err)
	}
	if info.TotalStorage != 0 || len(info.Mounts) != 0 {
		t.Fatal("deleted volume remained billable")
	}
	if err := exec.CommandContext(ctx, "docker", "volume", "inspect", volume).Run(); err == nil {
		t.Fatal("storage probe recreated a deleted volume")
	}
}
