package docker

import (
	"context"
	"fmt"
	"github.com/containerd/errdefs"
	"github.com/google/uuid"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"math"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

// StorageInfo separates immutable images from tenant volume and writable-layer bytes.
type StorageInfo struct {
	ImageSize, VolumeSize, ContainerDisk, TotalStorage int64
	Mounts                                             []StorageMount
	InventoryCaptured                                  bool
}
type StorageMount struct {
	Kind   string `json:"kind"`
	Source string `json:"source"`
}

type storageMount struct{ kind, source string }

func storageMounts(containers []container.InspectResponse) ([]storageMount, error) {
	unique := map[string]storageMount{}
	for _, info := range containers {
		for _, mount := range info.Mounts {
			switch string(mount.Type) {
			case "volume":
				if mount.Name == "" {
					return nil, fmt.Errorf("volume has no daemon identity")
				}
				unique["volume:"+mount.Name] = storageMount{"volume", mount.Name}
			case "bind":
				source := path.Clean(mount.Source)
				// System/config mounts are not tenant storage. Managed roots are measured on the daemon.
				if !strings.HasPrefix(source, "/var/lib/obiente/") && !strings.HasPrefix(source, "/var/obiente/") {
					continue
				}
				unique["bind:"+source] = storageMount{"bind", source}
			}
		}
	}
	mounts := make([]storageMount, 0, len(unique))
	for _, m := range unique {
		mounts = append(mounts, m)
	}
	sort.Slice(mounts, func(i, j int) bool { return len(mounts[i].source) < len(mounts[j].source) })
	selected := make([]storageMount, 0, len(mounts))
	for _, m := range mounts {
		nested := false
		for _, parent := range selected {
			if m.kind == "bind" && parent.kind == "bind" && strings.HasPrefix(m.source, parent.source+"/") {
				nested = true
				break
			}
		}
		if !nested {
			selected = append(selected, m)
		}
	}
	return selected, nil
}

func storageProbeArgs(name string, m storageMount) ([]string, error) {
	if strings.ContainsAny(m.source, ",\x00\r\n") || m.source == "" {
		return nil, fmt.Errorf("unsupported storage mount source")
	}
	return []string{"run", "--rm", "--name", name, "--network", "none", "--read-only", "--cap-drop", "ALL", "--cap-add", "DAC_READ_SEARCH", "--security-opt", "no-new-privileges", "--pids-limit", "16", "--memory", "64m", "--cpus", "0.25", "--mount", "type=" + m.kind + ",source=" + m.source + ",target=/data,readonly", "busybox:1.37.0", "du", "-sk", "/data"}, nil
}
func measureStorageMount(ctx context.Context, m storageMount) (int64, error) {
	name := "obiente-storage-" + uuid.NewString()
	args, err := storageProbeArgs(name, m)
	if err != nil {
		return 0, err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = exec.CommandContext(cleanup, "docker", "rm", "-f", name).Run()
	}()
	output, err := exec.CommandContext(ctx, "docker", args...).Output()
	if err != nil {
		return 0, fmt.Errorf("measure daemon storage: %w", err)
	}
	fields := strings.Fields(string(output))
	if len(fields) != 2 {
		return 0, fmt.Errorf("unexpected storage measurement")
	}
	blocks, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil || blocks < 0 || blocks > (1<<63-1)/1024 {
		return 0, fmt.Errorf("invalid storage measurement")
	}
	return blocks * 1024, nil
}

// CalculateStorage never reads Docker-host paths in the service filesystem and
// never substitutes df (host filesystem capacity) for tenant storage. Any missing
// container or failed measurement invalidates the entire snapshot.
func CalculateStorage(ctx context.Context, containerIDs []string) (*StorageInfo, error) {
	return CalculateStorageWithRetainedMounts(ctx, containerIDs, nil, false)
}

// Retained mount inventory survives Swarm removing a stopped task. A confirmed
// missing container has no writable layer, but its persistent volumes remain.
func CalculateStorageWithRetainedMounts(ctx context.Context, containerIDs []string, retained []StorageMount, inventoryKnown bool) (*StorageInfo, error) {
	if len(containerIDs) == 0 && !inventoryKnown {
		return nil, fmt.Errorf("storage inventory is unavailable")
	}
	c, err := New()
	if err != nil {
		return nil, err
	}
	defer c.Close()
	info := &StorageInfo{}
	containers := []container.InspectResponse{}
	seen := map[string]bool{}
	for _, id := range containerIDs {
		if id == "" {
			return nil, fmt.Errorf("container identity unavailable")
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		result, err := c.api.ContainerInspect(ctx, id, client.ContainerInspectOptions{Size: true})
		if err != nil {
			if inventoryKnown && errdefs.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		inspected := result.Container
		if inspected.SizeRw == nil || *inspected.SizeRw < 0 {
			return nil, fmt.Errorf("writable-layer size unavailable for %s", id)
		}
		if info.ContainerDisk > math.MaxInt64-*inspected.SizeRw {
			return nil, fmt.Errorf("storage size overflow")
		}
		info.ContainerDisk += *inspected.SizeRw
		containers = append(containers, inspected)
	}
	mounts, err := storageMounts(containers)
	if err != nil {
		return nil, err
	}
	current := map[string]bool{}
	all := map[string]storageMount{}
	for _, m := range mounts {
		current[m.kind+":"+m.source] = true
		all[m.kind+":"+m.source] = m
	}
	for _, m := range retained {
		if m.Kind != "volume" && m.Kind != "bind" {
			return nil, fmt.Errorf("invalid retained storage mount")
		}
		if m.Kind == "bind" && !strings.HasPrefix(path.Clean(m.Source), "/var/lib/obiente/") && !strings.HasPrefix(path.Clean(m.Source), "/var/obiente/") {
			return nil, fmt.Errorf("invalid retained storage root")
		}
		all[m.Kind+":"+m.Source] = storageMount{m.Kind, m.Source}
	}
	mounts = nil
	for key, m := range all {
		if m.kind == "volume" {
			if _, err := c.api.VolumeInspect(ctx, m.source, client.VolumeInspectOptions{}); err != nil {
				if !current[key] && errdefs.IsNotFound(err) {
					continue
				}
				return nil, fmt.Errorf("inspect storage volume: %w", err)
			}
		}
		mounts = append(mounts, m)
	}
	sort.Slice(mounts, func(i, j int) bool { return len(mounts[i].source) < len(mounts[j].source) })
	selected := []storageMount{}
	for _, m := range mounts {
		nested := false
		for _, parent := range selected {
			if m.kind == "bind" && parent.kind == "bind" && strings.HasPrefix(m.source, parent.source+"/") {
				nested = true
				break
			}
		}
		if !nested {
			selected = append(selected, m)
			info.Mounts = append(info.Mounts, StorageMount{m.kind, m.source})
		}
	}
	info.InventoryCaptured = true
	for _, m := range selected {
		size, err := measureStorageMount(ctx, m)
		if err != nil {
			return info, err
		}
		if info.VolumeSize > math.MaxInt64-size {
			return info, fmt.Errorf("storage size overflow")
		}
		info.VolumeSize += size
	}
	if info.VolumeSize > math.MaxInt64-info.ContainerDisk {
		return nil, fmt.Errorf("storage size overflow")
	}
	info.TotalStorage = info.VolumeSize + info.ContainerDisk
	return info, nil
}

// CaptureStorageMounts records storage identities before an explicit stop removes
// Swarm task containers. It reads metadata only and never mounts or changes data.
func CaptureStorageMounts(ctx context.Context, ids []string) ([]StorageMount, error) {
	c, err := New()
	if err != nil {
		return nil, err
	}
	defer c.Close()
	containers := []container.InspectResponse{}
	for _, id := range ids {
		result, err := c.api.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
		if err != nil {
			return nil, err
		}
		containers = append(containers, result.Container)
	}
	mounts, err := storageMounts(containers)
	if err != nil {
		return nil, err
	}
	result := make([]StorageMount, 0, len(mounts))
	for _, m := range mounts {
		result = append(result, StorageMount{m.kind, m.source})
	}
	return result, nil
}
