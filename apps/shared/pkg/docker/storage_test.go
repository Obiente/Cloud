package docker

import (
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"reflect"
	"strings"
	"testing"
)

func TestStorageMountsDeduplicateSharedAndNestedData(t *testing.T) {
	infos := []container.InspectResponse{
		{Mounts: []container.MountPoint{{Type: mount.TypeVolume, Name: "shared-data"}, {Type: mount.TypeBind, Source: "/var/lib/obiente/volumes/test"}, {Type: mount.TypeBind, Source: "/etc/resolv.conf"}}},
		{Mounts: []container.MountPoint{{Type: mount.TypeVolume, Name: "shared-data"}, {Type: mount.TypeBind, Source: "/var/lib/obiente/volumes/test/subdir"}}},
	}
	got, err := storageMounts(infos)
	if err != nil {
		t.Fatal(err)
	}
	want := []storageMount{{"volume", "shared-data"}, {"bind", "/var/lib/obiente/volumes/test"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mount inventory %v", got)
	}
}
func TestStorageProbeReadsDaemonMountWithoutWriteOrNetwork(t *testing.T) {
	args, err := storageProbeArgs("test-probe", storageMount{"bind", "/var/lib/obiente/volumes/test"})
	if err != nil {
		t.Fatal(err)
	}
	cmd := strings.Join(args, " ")
	for _, required := range []string{"--network none", "--read-only", "--cap-drop ALL", "no-new-privileges", "source=/var/lib/obiente/volumes/test,target=/data,readonly", "du -sk /data"} {
		if !strings.Contains(cmd, required) {
			t.Fatalf("missing probe constraint %s", required)
		}
	}
	if _, err := storageProbeArgs("test-probe", storageMount{"bind", "/data,readonly=false"}); err == nil {
		t.Fatal("accepted mount option injection")
	}
	if _, err := CalculateStorage(t.Context(), nil); err == nil {
		t.Fatal("missing inventory treated as zero storage")
	}
}
