package database

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestMetricsSchemaLockPostgres(t *testing.T) {
	dsn := os.Getenv("OBIENTE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requires an isolated PostgreSQL database")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	pool.SetMaxOpenConns(10)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var active, completed atomic.Int32
	errors := make(chan error, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errors <- withMetricsSchemaLock(ctx, db, func(conn *gorm.DB) error {
				if active.Add(1) != 1 {
					t.Error("concurrent startup entered schema migration")
				}
				defer active.Add(-1)
				// Query proves the callback uses the pinned connection, not an open transaction.
				var one int
				if err := conn.Raw("SELECT 1").Scan(&one).Error; err != nil {
					return err
				}
				time.Sleep(15 * time.Millisecond)
				completed.Add(1)
				return nil
			})
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	if completed.Load() != 8 {
		t.Fatalf("completed %d migrations", completed.Load())
	}
	// A waiter must respect its deadline and leave the lock reusable.
	err = withMetricsSchemaLock(ctx, db, func(_ *gorm.DB) error {
		short, stop := context.WithTimeout(ctx, 40*time.Millisecond)
		defer stop()
		if err := withMetricsSchemaLock(short, db, func(*gorm.DB) error { t.Error("waiter acquired held lock"); return nil }); err == nil {
			t.Error("cancelled waiter returned success")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := withMetricsSchemaLock(ctx, db, func(*gorm.DB) error { return nil }); err != nil {
		t.Fatalf("lock leaked after cancellation: %v", err)
	}
}
