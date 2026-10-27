package databases

import (
	"context"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueuedDatabaseCommandUsesLatestIntent(t *testing.T) {
	db := newDatabaseServiceTestDB(t)
	seedDatabaseServiceIsolationData(t, db)
	id := "db-org-a-owner"
	if err := database.SetDatabaseDesiredState(context.Background(), id, "running"); err != nil {
		t.Fatal(err)
	}
	svc := &Service{}
	var executed atomic.Bool
	done := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := database.WithResourceRuntimeLock(ctx, "database:"+id, func(context.Context) error {
		go func() {
			defer close(done)
			svc.runDatabaseOperation(ctx, id, "running", func() { executed.Store(true) })
		}()
		return database.SetDatabaseDesiredState(ctx, id, "stopped")
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if executed.Load() {
		t.Fatal("old queued start executed after operator stop")
	}
	svc.runDatabaseOperation(ctx, id, "stopped", func() { executed.Store(true) })
	if !executed.Load() {
		t.Fatal("latest stop request did not execute")
	}
}
