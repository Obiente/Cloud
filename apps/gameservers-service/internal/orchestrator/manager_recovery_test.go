package orchestrator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/lifecycle"
)

func TestManagerRecoversWhenDockerStartsLate(t *testing.T) {
	var available atomic.Bool
	var attempts atomic.Int32
	daemon := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !available.Load() {
			attempts.Add(1)
			http.Error(w, "daemon starting", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Name":"test-node","Swarm":{}}`))
	}))
	defer daemon.Close()
	t.Setenv("DOCKER_HOST", "tcp://"+strings.TrimPrefix(daemon.URL, "http://"))
	t.Setenv("DOCKER_API_VERSION", "1.52")
	t.Setenv("ENABLE_SWARM", "false")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go func() {
		for attempts.Load() < 2 && ctx.Err() == nil {
			time.Sleep(time.Millisecond)
		}
		available.Store(true)
	}()
	manager, err := lifecycle.UntilReady(ctx, 10*time.Millisecond, 200*time.Millisecond, func(ctx context.Context) (*GameServerManager, error) {
		return NewGameServerManagerContext(ctx, "least-loaded", 50)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	if attempts.Load() < 2 {
		t.Fatal("did not exercise late dependency")
	}
	if err := manager.CheckReady(ctx); err != nil {
		t.Fatal(err)
	}
	available.Store(false)
	if err := manager.CheckReady(ctx); err == nil {
		t.Fatal("outage reported ready")
	}
	available.Store(true)
	if err := manager.CheckReady(ctx); err != nil {
		t.Fatalf("existing manager failed to recover: %v", err)
	}
}
