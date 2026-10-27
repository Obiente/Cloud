package proxy

import (
	"fmt"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSleepingMySQLRefreshPreservesCredentialsForWake(t *testing.T) {
	t.Setenv("ENABLE_SWARM", "false")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/_ping"):
			w.Header().Set("API-Version", "1.53")
			fmt.Fprint(w, "OK")
		case strings.HasSuffix(r.URL.Path, "/info"):
			fmt.Fprint(w, `{"Name":"test-node"}`)
		case strings.HasSuffix(r.URL.Path, "/containers/json"):
			fmt.Fprint(w, `[{"Id":"mysql-container","State":"exited"},{"Id":"mariadb-container","State":"exited"}]`)
		default:
			t.Errorf("unexpected Docker request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	t.Setenv("DOCKER_HOST", "tcp://"+strings.TrimPrefix(server.URL, "http://"))
	t.Setenv("DOCKER_API_VERSION", "1.53")
	client, err := docker.New()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	defer raw.Close()
	previous := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previous })
	if err := db.AutoMigrate(&database.DatabaseInstance{}, &database.DatabaseConnection{}, &database.DatabaseLocation{}, &database.DatabaseUptimeInterval{}); err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"mysql", "mariadb"} {
		id := name + "-container"
		for _, row := range []interface{}{&database.DatabaseInstance{ID: name, Type: []int32{2, 5}[i], Status: 12, DesiredState: "sleeping", InstanceID: &id}, &database.DatabaseConnection{ID: name, DatabaseID: name, Username: "admin", Password: "encrypted-test-credential"}} {
			if err := db.Create(row).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	registry := NewRouteRegistry(client)
	registry.SetLocalRoutesOnly(true)
	for i := 0; i < 2; i++ {
		if err := registry.LoadFromDatabase(t.Context()); err != nil {
			t.Fatal(err)
		}
		for _, id := range []string{"mysql", "mariadb"} {
			route, ok := registry.LookupByID(id)
			if !ok || !route.Stopped || route.DBStatus != 12 || route.Username != "admin" || route.Password != "encrypted-test-credential" {
				t.Fatalf("sleeping route lost authentication: %+v", route)
			}
			registry.MarkRunning(id, "127.0.0.1")
			route, _ = registry.LookupByID(id)
			if route.Password != "encrypted-test-credential" {
				t.Fatal("wake lost credentials")
			}
		}
	}
}
