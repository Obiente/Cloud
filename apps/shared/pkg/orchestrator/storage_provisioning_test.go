package orchestrator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type storageProvisioningDocker struct {
	client.APIClient
	t       *testing.T
	db      *gorm.DB
	id      string
	replace bool
	mutated bool
}

func (c *storageProvisioningDocker) ContainerList(context.Context, client.ContainerListOptions) (client.ContainerListResult, error) {
	result := client.ContainerListResult{}
	if c.replace {
		result.Items = []container.Summary{{ID: "existing-test-container", Names: []string{"/" + c.id + "-default-replica-0"}, State: "exited"}}
	}
	return result, nil
}

func (c *storageProvisioningDocker) assertInvalidated() {
	c.t.Helper()
	var dep database.Deployment
	if err := c.db.First(&dep, "id = ?", c.id).Error; err != nil || dep.StorageState != "unknown" || dep.StorageMeasurementError == "" {
		c.t.Errorf("Docker mutation preceded storage invalidation: state=%q err=%v", dep.StorageState, err)
	}
	c.mutated = true
}

func (c *storageProvisioningDocker) ContainerCreate(context.Context, client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
	c.assertInvalidated()
	return client.ContainerCreateResult{}, errors.New("injected container create response failure")
}

func (c *storageProvisioningDocker) ContainerInspect(context.Context, string, client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	return client.ContainerInspectResult{Container: container.InspectResponse{
		ID: "existing-test-container", Config: &container.Config{Labels: map[string]string{"cloud.obiente.managed": "true", "cloud.obiente.deployment_id": c.id}},
		State: &container.State{Status: container.StateExited},
	}}, nil
}

type storageProvisioningHelper struct {
	dockerHelper
	client *storageProvisioningDocker
}

func (h *storageProvisioningHelper) RemoveContainer(context.Context, string, bool) error {
	h.client.assertInvalidated()
	return errors.New("injected container removal response failure")
}

func (h *storageProvisioningHelper) StartContainer(context.Context, string) error {
	h.client.assertInvalidated()
	return errors.New("injected container start response failure")
}

func newStorageProvisioningFixture(t *testing.T, state, failure string, swarm bool) (*gorm.DB, database.Deployment, *DeploymentManager, *storageProvisioningDocker, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Docker command stub requires POSIX shell")
	}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.AutoMigrate(&database.Deployment{}, &database.Organization{}, &database.NodeMetadata{}, &database.DeploymentRouting{}, &database.GameServer{}, &database.VPSInstance{}, &database.DatabaseInstance{}); err != nil {
		t.Fatal(err)
	}
	previous := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previous })
	now := time.Now().UTC()
	dep := database.Deployment{ID: "storage-test-" + uuid.NewString(), OrganizationID: "test-org", DesiredState: "running", StorageState: state}
	if state == "measured" {
		dep.StorageBytes = 1024
		dep.StorageMeasuredAt = &now
	}
	if state == "unknown" {
		dep.StorageMeasurementError = "Prior measurement unavailable"
	}
	for _, row := range []any{
		&database.Organization{ID: dep.OrganizationID, Slug: "test-org", Status: "active"},
		&database.NodeMetadata{ID: "test-node"}, &dep,
		&database.GameServer{ID: "healthy-peer", OrganizationID: dep.OrganizationID, StorageState: "measured", StorageBytes: 2048, StorageMeasuredAt: &now},
	} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "mutation")
	t.Setenv("PATH", dir)
	t.Setenv("ENABLE_SWARM", strconv.FormatBool(swarm))
	t.Setenv("REGISTRY_PASSWORD", "")
	t.Setenv("STORAGE_TEST_FAILURE", failure)
	t.Setenv("STORAGE_TEST_MARKER", marker)
	script := `#!/bin/sh
case "$1 $2" in
  "network ls")
    if [ "$STORAGE_TEST_FAILURE" = network ]; then echo 'Cannot connect to the Docker daemon' >&2; exit 61; fi
    for arg in "$@"; do case "$arg" in name=*) printf '%s\n' "${arg#name=}"; exit 0;; esac; done
    echo test-network; exit 0;;
  "network inspect") echo test-network; exit 0;;
  "ps --all"|"stack services")
    if [ "$STORAGE_TEST_FAILURE" = inventory ]; then echo 'inventory unavailable' >&2; exit 62; fi
    exit 0;;
  "service inspect")
    if [ "$STORAGE_TEST_FAILURE" = inspect ]; then echo 'inspection unavailable' >&2; exit 63; fi
    if [ "$STORAGE_TEST_FAILURE" = update ] || [ "$STORAGE_TEST_FAILURE" = update-db ]; then
      case "$*" in
        *'json .Spec.Labels'*) echo '{}';;
        *'json '*) echo '[]';;
        *) echo existing-service;;
      esac
      exit 0
    fi
    echo 'no such service' >&2; exit 1;;
  "service create"|"service update"|"stack deploy"|compose\ *)
    printf 'attempted' > "$STORAGE_TEST_MARKER"
    echo 'injected mutation response failure' >&2; exit 64;;
esac
echo "unexpected Docker command: $*" >&2
exit 90
`
	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	cli := &storageProvisioningDocker{t: t, db: db, id: dep.ID, replace: strings.HasPrefix(failure, "replace")}
	dm := &DeploymentManager{dockerClient: cli, dockerHelper: &storageProvisioningHelper{client: cli}, nodeID: "test-node", networkName: "test-network"}
	return db, dep, dm, cli, marker
}

func assertProvisioningStorage(t *testing.T, db *gorm.DB, before database.Deployment, invalidated bool) {
	t.Helper()
	// Failed and explicitly stopped resources remain in the billing inventory.
	repo := database.NewDeploymentRepository(db, nil)
	if err := repo.UpdateStatus(t.Context(), before.ID, int32(deploymentsv1.DeploymentStatus_FAILED)); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetDesiredState(t.Context(), before.ID, "stopped"); err != nil {
		t.Fatal(err)
	}
	var current database.Deployment
	if err := db.First(&current, "id = ?", before.ID).Error; err != nil {
		t.Fatal(err)
	}
	if invalidated {
		if current.StorageState != "unknown" || current.StorageMeasurementError == "" {
			t.Fatalf("uncertain runtime became trusted: %+v", current)
		}
	} else {
		if current.StorageState != before.StorageState || current.StorageBytes != before.StorageBytes || current.StorageMeasurementError != before.StorageMeasurementError {
			t.Fatalf("preflight changed storage: state=%s bytes=%d error=%q", current.StorageState, current.StorageBytes, current.StorageMeasurementError)
		}
		if (current.StorageMeasuredAt == nil) != (before.StorageMeasuredAt == nil) || (current.StorageMeasuredAt != nil && !current.StorageMeasuredAt.Equal(*before.StorageMeasuredAt)) {
			t.Fatal("preflight changed measurement timestamp")
		}
	}
	bytes, err := database.BillableStorageBytes(t.Context(), db, before.OrganizationID, time.Now())
	if invalidated || before.StorageState == "unknown" {
		if err == nil {
			t.Fatal("billing accepted uncertain storage")
		}
	} else if err != nil || bytes != 2048+before.StorageBytes {
		t.Fatalf("preflight failure blocked organization billing: bytes=%d err=%v", bytes, err)
	}
}

func TestDeploymentProvisioningStorage(t *testing.T) {
	for _, swarm := range []bool{false, true} {
		failures := []string{"network", "node", "volume", "create", "create-db"}
		if swarm {
			failures = append(failures, "inspect", "update", "update-db")
		} else {
			failures = append(failures, "replace", "replace-db", "recover", "recover-db")
		}
		for _, failure := range failures {
			for _, state := range []string{"empty", "measured", "unknown"} {
				t.Run(strconv.FormatBool(swarm)+"/"+failure+"/"+state, func(t *testing.T) {
					if failure == "volume" && os.Getenv("OBIENTE_VOLUME_TESTS") != "1" {
						t.Skip("requires isolated volume root and OBIENTE_VOLUME_TESTS=1")
					}
					db, dep, dm, cli, marker := newStorageProvisioningFixture(t, state, failure, swarm)
					config := &DeploymentConfig{DeploymentID: dep.ID, TargetNodeID: "test-node", NetworkName: "test-network", Image: "test-image", Replicas: 1, Port: 8080, EnvVars: map[string]string{}}
					config.RecoveryOnly = strings.HasPrefix(failure, "recover")
					if failure == "node" {
						config.TargetNodeID = "missing-node"
					}
					if failure == "volume" {
						root := filepath.Join(deploymentVolumeRoot, dep.ID)
						if err := os.Mkdir(root, 0755); err != nil {
							t.Fatal(err)
						}
						t.Cleanup(func() { _ = os.RemoveAll(root) })
						if err := os.WriteFile(filepath.Join(root, "data"), []byte("retained data"), 0600); err != nil {
							t.Fatal(err)
						}
						config.Volumes = []DeploymentVolume{{Name: "data", MountPath: "/data"}}
					}
					if strings.HasSuffix(failure, "-db") {
						if err := db.Exec("CREATE TRIGGER reject_storage BEFORE UPDATE OF storage_state ON deployments BEGIN SELECT RAISE(ABORT, 'storage write unavailable'); END").Error; err != nil {
							t.Fatal(err)
						}
					}
					err := dm.CreateDeploymentLocked(t.Context(), config)
					if err == nil {
						t.Fatal("injected failure did not fail provisioning")
					}
					invalidated := failure == "create" || failure == "replace" || failure == "update" || failure == "recover"
					_, markerErr := os.Stat(marker)
					mutated := cli.mutated || markerErr == nil
					if mutated != invalidated {
						t.Fatalf("unexpected mutation=%t: %v", mutated, err)
					}
					assertProvisioningStorage(t, db, dep, invalidated)
				})
			}
		}
	}
}

func TestComposeProvisioningStorage(t *testing.T) {
	for _, swarm := range []bool{false, true} {
		for _, failure := range []string{"network", "yaml", "volume", "inventory", "create", "create-db"} {
			for _, state := range []string{"empty", "measured", "unknown"} {
				t.Run(strconv.FormatBool(swarm)+"/"+failure+"/"+state, func(t *testing.T) {
					if failure != "network" && os.Getenv("OBIENTE_VOLUME_TESTS") != "1" {
						t.Skip("requires isolated deployments/volumes roots and OBIENTE_VOLUME_TESTS=1")
					}
					db, dep, dm, cli, marker := newStorageProvisioningFixture(t, state, failure, swarm)
					t.Cleanup(func() {
						for _, base := range []string{"/var/lib/obiente/volumes", "/var/lib/obiente/deployments"} {
							_ = os.RemoveAll(filepath.Join(base, dep.ID))
						}
					})
					spec := "services:\n  worker:\n    image: test-image\n"
					if failure == "yaml" {
						spec = "services: ["
					}
					if failure == "volume" {
						root := filepath.Join(deploymentVolumeRoot, dep.ID)
						if err := os.Mkdir(root, 0755); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(filepath.Join(root, "data"), []byte("retained data"), 0600); err != nil {
							t.Fatal(err)
						}
						spec += "    volumes:\n      - data:/data\nvolumes:\n  data: {}\n"
					}
					if strings.HasSuffix(failure, "-db") {
						if err := db.Exec("CREATE TRIGGER reject_storage BEFORE UPDATE OF storage_state ON deployments BEGIN SELECT RAISE(ABORT, 'storage write unavailable'); END").Error; err != nil {
							t.Fatal(err)
						}
					}
					err := dm.deployComposeFileLocked(t.Context(), dep.ID, spec, "")
					if err == nil {
						t.Fatal("injected failure did not fail Compose provisioning")
					}
					invalidated := failure == "create"
					_, markerErr := os.Stat(marker)
					if (cli.mutated || markerErr == nil) != invalidated {
						t.Fatalf("unexpected Compose mutation: %v", err)
					}
					assertProvisioningStorage(t, db, dep, invalidated)
				})
			}
		}
	}
}
