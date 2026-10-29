package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
)

func fileArchive(t *testing.T, headers ...tar.Header) []byte {
	t.Helper()
	var b bytes.Buffer
	w := tar.NewWriter(&b)
	for _, h := range headers {
		if err := w.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if h.Size > 0 {
			if _, err := w.Write(bytes.Repeat([]byte("x"), int(h.Size))); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestDirectoryArchive(t *testing.T) {
	for _, root := range []string{".", "app"} {
		t.Run(root, func(t *testing.T) {
			data := fileArchive(t, tar.Header{Name: root + "/", Typeflag: tar.TypeDir, Mode: 0755},
				tar.Header{Name: root + "/nested/", Typeflag: tar.TypeDir, Mode: 0755},
				tar.Header{Name: root + "/nested/hidden.txt", Size: 3, Mode: 0644},
				tar.Header{Name: root + "/hello world.txt", Size: 4, Mode: 0640, Uid: 123, Gid: 456},
				tar.Header{Name: root + "/link", Typeflag: tar.TypeSymlink, Linkname: "hello world.txt", Mode: 0777})
			files, err := readDirectoryArchive(bytes.NewReader(data), "/app")
			if err != nil {
				t.Fatal(err)
			}
			if len(files) != 3 || files[1].Name != "hello world.txt" || files[1].Owner != "123" || files[1].Path != "/app/hello world.txt" || !files[2].IsSymlink {
				t.Fatalf("bad listing: %+v", files)
			}
		})
	}
	for _, name := range []string{"../outside", "app/../../outside", "outside"} {
		data := fileArchive(t, tar.Header{Name: "app/", Typeflag: tar.TypeDir}, tar.Header{Name: name})
		if _, err := readDirectoryArchive(bytes.NewReader(data), "/app"); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
	if _, err := readDirectoryArchive(bytes.NewReader([]byte("OCI runtime exec failed")), "/"); err == nil {
		t.Fatal("accepted diagnostic")
	}
}

func TestLsDiagnosticsAreNotFiles(t *testing.T) {
	output := `OCI runtime exec failed: exec failed: unable to start container process: exec: "ls": executable file not found in $PATH`
	if files := parseLsOutput(output, "/"); len(files) != 0 {
		t.Fatalf("diagnostic became files: %+v", files)
	}
	for _, perms := range []string{"-rw-r--r--", "-rw-r--r--+", "-rw-r--r--."} {
		files := parseLsOutput(perms+" 1 root root 4 2026-01-01 12:00 fixture.txt", "/")
		if len(files) != 1 || files[0].Name != "fixture.txt" {
			t.Fatalf("valid listing rejected: %s %+v", perms, files)
		}
	}
}

type zeroFileReader struct{}

func (zeroFileReader) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func TestDirectoryArchiveTransferLimit(t *testing.T) {
	var prefix bytes.Buffer
	w := tar.NewWriter(&prefix)
	if err := w.WriteHeader(&tar.Header{Name: "data/", Typeflag: tar.TypeDir}); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteHeader(&tar.Header{Name: "data/large", Size: maxDirectoryArchiveBytes + 1}); err != nil {
		t.Fatal(err)
	}
	// Stream the declared body without allocating a large fixture. Hitting the
	// boundary must be an error, never a successful but incomplete directory.
	files, err := readDirectoryArchive(io.MultiReader(&prefix, zeroFileReader{}), "/data")
	if err == nil || files != nil {
		t.Fatalf("large listing silently truncated: %v %v", files, err)
	}
}

func TestDockerRootArchiveUsesAbsoluteNames(t *testing.T) {
	data := fileArchive(t, tar.Header{Name: "/", Typeflag: tar.TypeDir}, tar.Header{Name: "/app/", Typeflag: tar.TypeDir}, tar.Header{Name: "/app/record", Size: 4})
	files, err := readDirectoryArchive(bytes.NewReader(data), "/")
	if err != nil || len(files) != 1 || files[0].Name != "app" {
		t.Fatalf("Docker root listing: %+v %v", files, err)
	}
}

type fileBrowserAPI struct {
	client.APIClient
	inspect container.InspectResponse
	data    []byte
	stat    container.PathStat
	paths   []string
}

func (f *fileBrowserAPI) ContainerInspect(context.Context, string, client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	return client.ContainerInspectResult{Container: f.inspect}, nil
}
func (f *fileBrowserAPI) CopyFromContainer(_ context.Context, _ string, o client.CopyFromContainerOptions) (client.CopyFromContainerResult, error) {
	f.paths = append(f.paths, o.SourcePath)
	return client.CopyFromContainerResult{Content: io.NopCloser(bytes.NewReader(f.data)), Stat: f.stat}, nil
}
func (f *fileBrowserAPI) ContainerStatPath(_ context.Context, _ string, o client.ContainerStatPathOptions) (client.ContainerStatPathResult, error) {
	return client.ContainerStatPathResult{Stat: f.stat}, nil
}

func TestStoppedContainerAndRemoteVolumeReads(t *testing.T) {
	f := &fileBrowserAPI{inspect: container.InspectResponse{State: &container.State{}, Mounts: []container.MountPoint{
		{Type: mount.TypeVolume, Name: "data", Source: "/inaccessible/docker/data", Destination: "/data"},
		{Type: mount.TypeBind, Source: "/var/lib/obiente/volumes/managed", Destination: "/managed"},
		{Type: mount.TypeBind, Source: "/unmanaged", Destination: "/other"},
	}}, stat: container.PathStat{Mode: os.ModeDir | 0755}, data: fileArchive(t, tar.Header{Name: "data/", Typeflag: tar.TypeDir}, tar.Header{Name: "data/record", Size: 4, Mode: 0644})}
	c := &Client{api: f}
	volumes, err := c.GetContainerVolumes(context.Background(), "test")
	if err != nil || len(volumes) != 2 {
		t.Fatalf("remote mounts missing: %+v %v", volumes, err)
	}
	files, err := c.ListMountedVolumeFiles(context.Background(), "test", volumes[0], "/")
	if err != nil || len(files) != 1 || files[0].Path != "/record" {
		t.Fatalf("wrong listing: %+v %v", files, err)
	}
	f.stat = container.PathStat{Mode: 0644, Size: 4}
	f.data = fileArchive(t, tar.Header{Name: "record", Size: 4, Mode: 0644})
	data, info, err := c.ReadMountedVolumeFile(context.Background(), "test", volumes[0], "/record")
	if err != nil || string(data) != "xxxx" || info.Path != "/record" {
		t.Fatalf("wrong read: %q %+v %v", data, info, err)
	}
	if !reflect.DeepEqual(f.paths, []string{"/data", "/data/record"}) {
		t.Fatal(f.paths)
	}
	// The embedded nil API deliberately panics on any lifecycle/exec call.
	for _, p := range []string{"../outside", "/a/../../outside", "/bad\x00path"} {
		if _, err := mountedVolumePath(volumes[0], p); err == nil {
			t.Fatalf("accepted %q", p)
		}
	}
	f.stat.Size = maxFilePreviewBytes + 1
	if _, err := c.ContainerReadFile(context.Background(), "test", "/huge"); err == nil {
		t.Fatal("oversized file accepted")
	}
}
