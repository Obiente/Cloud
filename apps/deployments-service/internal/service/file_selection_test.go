package deployments

import (
	"archive/tar"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
	"google.golang.org/protobuf/proto"
)

// Each container exposes a different backing directory at the same volume name
// and mount path. Exercise the actual RPC handlers against a fake Docker daemon.
func fileSelectionFixture(t *testing.T, states ...string) (*Service, context.Context, map[string]string) {
	t.Helper()
	db := newDeploymentServiceTestDB(t)
	seedDeploymentServiceIsolationData(t, db)
	backing := map[string]string{}
	status := map[string]string{}
	for i, state := range states {
		id := string(rune('a'+i)) + "-container"
		backing[id] = t.TempDir()
		status[id] = state
		if err := os.WriteFile(filepath.Join(backing[id], "record.txt"), []byte(id), 0640); err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&database.DeploymentLocation{ID: id, DeploymentID: "dep-org-a-owner", ContainerID: id, Status: state}).Error; err != nil {
			t.Fatal(err)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 || parts[len(parts)-3] != "containers" {
			http.Error(w, "unexpected Docker request", http.StatusNotFound)
			return
		}
		id, op := parts[len(parts)-2], parts[len(parts)-1]
		if status[id] == "missing" || backing[id] == "" {
			http.Error(w, "container missing", http.StatusNotFound)
			return
		}
		switch op {
		case "json":
			_ = json.NewEncoder(w).Encode(container.InspectResponse{
				ID: id, State: &container.State{Running: status[id] == "running"},
				Config: &container.Config{Labels: map[string]string{"cloud.obiente.service_name": id}},
				Mounts: []container.MountPoint{{Type: mount.TypeVolume, Name: "data", Source: backing[id], Destination: "/data"}},
			})
		case "archive":
			if r.URL.Query().Get("path") != "/data/record.txt" || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
				http.Error(w, "unexpected file request", http.StatusBadRequest)
				return
			}
			data, err := os.ReadFile(filepath.Join(backing[id], "record.txt"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			stat, _ := json.Marshal(container.PathStat{Name: "record.txt", Mode: 0640, Size: int64(len(data))})
			w.Header().Set("X-Docker-Container-Path-Stat", base64.StdEncoding.EncodeToString(stat))
			if r.Method == http.MethodGet {
				archive := tar.NewWriter(w)
				_ = archive.WriteHeader(&tar.Header{Name: "record.txt", Mode: 0640, Size: int64(len(data))})
				_, _ = archive.Write(data)
				_ = archive.Close()
			}
		default:
			http.Error(w, "unexpected operation", http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("DOCKER_HOST", server.URL)
	t.Setenv("DOCKER_API_VERSION", "1.47")
	t.Setenv("DOCKER_TLS_VERIFY", "")
	t.Setenv("DOCKER_CERT_PATH", "")
	ctx := auth.WithUser(context.Background(), &authv1.User{Id: "user-org-a", Email: "user-org-a@example.com"})
	return NewService(ctx, database.NewDeploymentRepository(db, nil), nil, nil), ctx, backing
}

func TestVolumeSaveUsesReadContainer(t *testing.T) {
	for _, selector := range []string{"container", "service"} {
		t.Run(selector, func(t *testing.T) {
			s, ctx, backing := fileSelectionFixture(t, "stopped", "stopped")
			read := &deploymentsv1.GetContainerFileRequest{OrganizationId: "org-a", DeploymentId: "dep-org-a-owner", VolumeName: proto.String("data"), Path: "/record.txt"}
			write := &deploymentsv1.WriteContainerFileRequest{OrganizationId: "org-a", DeploymentId: "dep-org-a-owner", VolumeName: proto.String("data"), Path: "/record.txt", Content: "updated second file"}
			if selector == "container" {
				read.ContainerId, write.ContainerId = proto.String("b-container"), proto.String("b-container")
			} else {
				read.ServiceName, write.ServiceName = proto.String("b-container"), proto.String("b-container")
			}
			got, err := s.GetContainerFile(ctx, connect.NewRequest(read))
			if err != nil || got.Msg.Content != "b-container" {
				t.Fatalf("selected read: %v %v", got, err)
			}
			if _, err := s.WriteContainerFile(ctx, connect.NewRequest(write)); err != nil {
				t.Fatal(err)
			}
			for id, want := range map[string]string{"a-container": "a-container", "b-container": write.Content} {
				data, err := os.ReadFile(filepath.Join(backing[id], "record.txt"))
				if err != nil || string(data) != want {
					t.Fatalf("%s content = %q, %v; want %q", id, data, err, want)
				}
			}
		})
	}
}

func TestVolumeDiscoveryMatchesDefaultFileContainer(t *testing.T) {
	for _, test := range []struct {
		name   string
		states []string
		want   string
	}{
		{"all stopped", []string{"stopped", "stopped"}, "a-container"},
		{"running preferred", []string{"stopped", "running", "running"}, "b-container"},
		{"missing skipped", []string{"missing", "stopped", "stopped"}, "b-container"},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, ctx, backing := fileSelectionFixture(t, test.states...)
			got, err := s.ListContainerFiles(ctx, connect.NewRequest(&deploymentsv1.ListContainerFilesRequest{OrganizationId: "org-a", DeploymentId: "dep-org-a-owner", ListVolumes: true}))
			if err != nil || len(got.Msg.Volumes) != 1 || got.Msg.Volumes[0].Source != backing[test.want] {
				t.Fatalf("wrong volume discovery: %v %v", got, err)
			}
			dcli, err := docker.New()
			if err != nil {
				t.Fatal(err)
			}
			defer dcli.Close()
			loc, err := s.findContainerForDeployment(ctx, "dep-org-a-owner", "", "", dcli)
			if err != nil || loc.ContainerID != test.want {
				t.Fatalf("default file target differs: %v %v", loc, err)
			}
		})
	}
}

func TestVolumeDeleteAndCreateUseSelectedContainer(t *testing.T) {
	for _, selector := range []string{"container", "service"} {
		t.Run(selector, func(t *testing.T) {
			s, ctx, backing := fileSelectionFixture(t, "stopped", "stopped")
			remove := &deploymentsv1.DeleteContainerEntriesRequest{OrganizationId: "org-a", DeploymentId: "dep-org-a-owner", VolumeName: proto.String("data"), Paths: []string{"/record.txt"}}
			create := &deploymentsv1.CreateContainerEntryRequest{OrganizationId: "org-a", DeploymentId: "dep-org-a-owner", VolumeName: proto.String("data"), ParentPath: "/", Name: "created.txt", Type: deploymentsv1.ContainerEntryType_CONTAINER_ENTRY_TYPE_FILE}
			if selector == "container" {
				remove.ContainerId, create.ContainerId = proto.String("b-container"), proto.String("b-container")
			} else {
				remove.ServiceName, create.ServiceName = proto.String("b-container"), proto.String("b-container")
			}
			if got, err := s.DeleteContainerEntries(ctx, connect.NewRequest(remove)); err != nil || !got.Msg.Success {
				t.Fatalf("selected delete: %v %v", got, err)
			}
			if _, err := os.Stat(filepath.Join(backing["b-container"], "record.txt")); !os.IsNotExist(err) {
				t.Fatalf("selected file was not deleted: %v", err)
			}
			if data, err := os.ReadFile(filepath.Join(backing["a-container"], "record.txt")); err != nil || string(data) != "a-container" {
				t.Fatalf("default file was changed: %q %v", data, err)
			}
			if _, err := s.CreateContainerEntry(ctx, connect.NewRequest(create)); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(backing["b-container"], "created.txt")); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(backing["a-container"], "created.txt")); !os.IsNotExist(err) {
				t.Fatalf("file created in default container: %v", err)
			}
		})
	}
}

func TestLargeFileDownloadRetainsPreviewLimit(t *testing.T) {
	s, ctx, backing := fileSelectionFixture(t, "stopped", "stopped")
	const size = 64<<20 + 1
	if err := os.Truncate(filepath.Join(backing["b-container"], "record.txt"), size); err != nil {
		t.Fatal(err)
	}
	for _, volume := range []string{"", "data"} {
		req := &deploymentsv1.GetContainerFileRequest{OrganizationId: "org-a", DeploymentId: "dep-org-a-owner", ContainerId: proto.String("b-container"), VolumeName: proto.String(volume), Path: "/data/record.txt"}
		if volume != "" {
			req.Path = "/record.txt"
		}
		if got, err := s.GetContainerFile(ctx, connect.NewRequest(req)); err == nil || got != nil {
			t.Fatal("oversized preview succeeded")
		}
		req.Download = true
		got, err := s.GetContainerFile(ctx, connect.NewRequest(req))
		if err != nil {
			t.Fatal(err)
		}
		if got.Msg.Size != size || len(got.Msg.Content) != size || !strings.HasPrefix(got.Msg.Content, "b-container") {
			t.Fatalf("download did not preserve the entire selected file: size=%d", got.Msg.Size)
		}
	}
}
