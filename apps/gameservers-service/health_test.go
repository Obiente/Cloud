package main

import (
	"context"
	gameserverorchestrator "gameservers-service/internal/orchestrator"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestHTTPReadinessTracksInitializationAndDependencyRecovery(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	prior := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prior; raw.Close() })
	var dockerUp atomic.Bool
	daemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !dockerUp.Load() {
			http.Error(w, "starting", 503)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"Name":"test-node","Swarm":{}}`))
	}))
	defer daemon.Close()
	t.Setenv("DOCKER_HOST", "tcp://"+strings.TrimPrefix(daemon.URL, "http://"))
	t.Setenv("DOCKER_API_VERSION", "1.52")
	t.Setenv("ENABLE_SWARM", "false")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mux := http.NewServeMux()
	mux.HandleFunc("/operation", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	var ready atomic.Pointer[gameserverorchestrator.GameServerManager]
	server := httptest.NewServer(newRuntimeHandler(ctx, mux, &ready))
	defer server.Close()
	check := func(path string, want int) {
		t.Helper()
		res, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != want {
			t.Fatalf("%s: %d, want %d", path, res.StatusCode, want)
		}
		if path != "/operation" && res.Header.Get("Content-Type") != "application/json" {
			t.Fatal("health response missing JSON content type")
		}
	}
	check("/livez", 200)
	check("/readyz", 503)
	check("/operation", 503)
	dockerUp.Store(true)
	manager, err := gameserverorchestrator.NewGameServerManagerContext(ctx, "least-loaded", 50)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	ready.Store(manager)
	check("/readyz", 200)
	check("/operation", 204)
	dockerUp.Store(false)
	check("/readyz", 503)
	check("/livez", 200)
	dockerUp.Store(true)
	check("/readyz", 200)
	raw.Close()
	check("/readyz", 503)
	check("/livez", 200)
	cancel()
	check("/livez", 503)
}
