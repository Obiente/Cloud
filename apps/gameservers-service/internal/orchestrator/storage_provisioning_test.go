package orchestrator

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/client"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGameServerProvisioningStorageAndBilling(t *testing.T) {
	for _, failure := range []string{"network", "node", "image", "routes", "port", "replace", "create"} {
		for _, state := range []string{"empty", "measured", "unknown"} {
			t.Run(failure+"/"+state, func(t *testing.T) {
				// The create boundary uses the real bind-volume preparation path.
				// Opt in only with an isolated writable volume root; all Docker API
				// traffic still goes to the fake daemon below.
				if failure == "create" && os.Getenv("OBIENTE_VOLUME_TESTS") != "1" {
					t.Skip("requires an isolated /var/lib/obiente/volumes mount and OBIENTE_VOLUME_TESTS=1")
				}
				db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
				if err != nil {
					t.Fatal(err)
				}
				pool, err := db.DB()
				if err != nil {
					t.Fatal(err)
				}
				pool.SetMaxOpenConns(1)
				t.Cleanup(func() { _ = pool.Close() })
				if err := db.AutoMigrate(&database.GameServer{}, &database.GameServerSuspension{}, &database.Organization{}, &database.NodeMetadata{}, &database.GameServerHTTPRoute{}, &database.Deployment{}, &database.VPSInstance{}, &database.DatabaseInstance{}); err != nil {
					t.Fatal(err)
				}
				previousDB := database.DB
				database.DB = db
				t.Cleanup(func() { database.DB = previousDB })
				now := time.Now().UTC()
				server := database.GameServer{ID: "provisioning-test", OrganizationID: "test-org", StorageState: state}
				var retainedFile string
				if failure == "create" {
					server.ID = "test-" + uuid.NewString()
					volumePath := filepath.Join("/var/lib/obiente/volumes", "gameserver-"+server.ID+"-data")
					if err := os.Mkdir(volumePath, 0755); err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { _ = os.RemoveAll(volumePath) })
					if state != "empty" {
						retainedFile = filepath.Join(volumePath, "saved-game")
						if err := os.WriteFile(retainedFile, []byte("retained game data"), 0600); err != nil {
							t.Fatal(err)
						}
					}
				}
				if state == "measured" {
					server.StorageBytes = 512
					server.StorageMeasuredAt = &now
				} else if state == "unknown" {
					server.StorageMeasurementError = "Prior storage unavailable"
				}
				for _, record := range []any{
					&database.Organization{ID: server.OrganizationID, Slug: "test-org", Status: "active"},
					&database.NodeMetadata{ID: "test-node"},
					&database.Deployment{ID: "healthy-peer", OrganizationID: server.OrganizationID, StorageState: "measured", StorageBytes: 4096, StorageMeasuredAt: &now},
					&server,
				} {
					if err := db.Create(record).Error; err != nil {
						t.Fatal(err)
					}
				}
				if failure == "routes" {
					if err := db.Migrator().DropTable(&database.GameServerHTTPRoute{}); err != nil {
						t.Fatal(err)
					}
				}
				var removed atomic.Bool
				daemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					switch {
					case strings.HasSuffix(r.URL.Path, "/containers/create"):
						if failure != "create" {
							t.Error("preflight failure reached container creation")
						}
						var current database.GameServer
						if err := db.First(&current, "id = ?", server.ID).Error; err != nil || current.StorageState != "unknown" || current.StorageMeasurementError == "" {
							t.Errorf("storage was not invalidated before Docker creation: state=%q err=%v", current.StorageState, err)
						}
						// A daemon error does not prove the container was never created.
						http.Error(w, `{"message":"create response unavailable"}`, http.StatusInternalServerError)
					case strings.HasSuffix(r.URL.Path, "/networks"):
						if failure == "network" {
							http.Error(w, `{"message":"network unavailable"}`, http.StatusServiceUnavailable)
							return
						}
						fmt.Fprint(w, `[{"Name":"test-network"}]`)
					case strings.Contains(r.URL.Path, "/containers/"):
						if failure == "replace" {
							if strings.HasSuffix(r.URL.Path, "/json") {
								fmt.Fprint(w, `{"Id":"existing-container","Config":{"Labels":{"cloud.obiente.managed":"true"}}}`)
							} else {
								var current database.GameServer
								if err := db.First(&current, "id = ?", server.ID).Error; err != nil || current.StorageState != "unknown" {
									t.Errorf("container replacement did not invalidate storage: state=%q err=%v", current.StorageState, err)
								}
								if r.Method == http.MethodDelete {
									removed.Store(true)
								}
								w.WriteHeader(http.StatusNoContent)
							}
							return
						}
						if !strings.HasSuffix(r.URL.Path, "/json") {
							t.Errorf("preflight failure reached container mutation: %s", r.URL.Path)
						}
						http.Error(w, `{"message":"no such container"}`, http.StatusNotFound)
					case strings.HasSuffix(r.URL.Path, "/images/json"):
						if failure == "image" || failure == "replace" {
							fmt.Fprint(w, `[]`)
						} else {
							fmt.Fprint(w, `[{"Id":"test-image"}]`)
						}
					case strings.HasSuffix(r.URL.Path, "/images/create"):
						http.Error(w, `{"message":"image unavailable"}`, http.StatusNotFound)
					default:
						t.Errorf("unexpected Docker request: %s", r.URL.Path)
						http.Error(w, `{"message":"unexpected request"}`, http.StatusInternalServerError)
					}
				}))
				defer daemon.Close()
				cli, err := client.New(client.WithHost(daemon.URL), client.WithAPIVersion("1.52"))
				if err != nil {
					t.Fatal(err)
				}
				defer cli.Close()
				manager := &GameServerManager{dockerClient: cli, nodeID: "test-node", networkName: "test-network"}
				if failure == "replace" {
					t.Setenv("DOCKER_HOST", daemon.URL)
					t.Setenv("DOCKER_API_VERSION", "1.52")
					helper, err := docker.New()
					if err != nil {
						t.Fatal(err)
					}
					defer helper.Close()
					manager.dockerHelper = helper
				}
				config := &GameServerConfig{GameServerID: server.ID, TargetNodeID: "test-node", Image: "test-image", Port: 25565}
				if failure == "node" {
					config.TargetNodeID = "missing-node"
				} else if failure == "port" {
					config.Port = -1
				}
				wantError := failure
				if failure == "replace" {
					wantError = "image"
				}
				if err := manager.CreateGameServer(context.Background(), config); err == nil || !strings.Contains(strings.ToLower(err.Error()), wantError) {
					t.Fatalf("expected %s failure, got %v", wantError, err)
				}
				var current database.GameServer
				if err := db.First(&current, "id = ?", server.ID).Error; err != nil {
					t.Fatal(err)
				}
				if failure == "create" || failure == "replace" {
					if failure == "replace" && !removed.Load() {
						t.Fatal("did not exercise existing container removal")
					}
					if current.StorageState != "unknown" || current.StorageMeasurementError == "" {
						t.Fatal("ambiguous Docker creation was reset to trusted storage")
					}
					if _, err := database.BillableStorageBytes(context.Background(), db, server.OrganizationID, time.Now()); err == nil {
						t.Fatal("billing accepted ambiguous container storage")
					}
					if retainedFile != "" {
						data, err := os.ReadFile(retainedFile)
						if err != nil || string(data) != "retained game data" {
							t.Fatalf("provisioning damaged retained data: %v", err)
						}
					}
					return
				}
				if current.StorageState != state || current.StorageBytes != server.StorageBytes || current.StorageMeasurementError != server.StorageMeasurementError {
					t.Fatalf("preflight changed storage: state=%q bytes=%d error=%q", current.StorageState, current.StorageBytes, current.StorageMeasurementError)
				}
				if (current.StorageMeasuredAt == nil) != (server.StorageMeasuredAt == nil) || (current.StorageMeasuredAt != nil && !current.StorageMeasuredAt.Equal(*server.StorageMeasuredAt)) {
					t.Fatal("preflight changed storage measurement timestamp")
				}
				bytes, err := database.BillableStorageBytes(context.Background(), db, server.OrganizationID, time.Now())
				if state == "unknown" {
					if err == nil {
						t.Fatal("uncertain existing storage became billable")
					}
				} else if err != nil || bytes != 4096+server.StorageBytes {
					t.Fatalf("unmaterialized failure blocked organization billing: bytes=%d err=%v", bytes, err)
				}
			})
		}
	}
}
