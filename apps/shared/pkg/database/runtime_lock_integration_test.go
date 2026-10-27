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

func TestResourceRuntimeLockPostgres(t *testing.T) {
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
	previous := DB
	DB = db
	defer func() { DB = previous }()
	var active, completed atomic.Int32
	errors := make(chan error, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errors <- WithResourceRuntimeLock(ctx, "test-resource-runtime", func(_ context.Context) error {
				if active.Add(1) != 1 {
					t.Error("concurrent lifecycle operations entered the same resource")
				}
				defer active.Add(-1)
				// Runtime work can use the database pool while the lock pins its own connection.
				var one int
				if err := db.Raw("SELECT 1").Scan(&one).Error; err != nil {
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
		t.Fatalf("completed %d operations", completed.Load())
	}
	// A waiter must respect its deadline and leave the lock reusable.
	err = WithResourceRuntimeLock(ctx, "test-resource-runtime", func(_ context.Context) error {
		short, stop := context.WithTimeout(ctx, 40*time.Millisecond)
		defer stop()
		if err := WithResourceRuntimeLock(short, "test-resource-runtime", func(context.Context) error { t.Error("waiter acquired held lock"); return nil }); err == nil {
			t.Error("cancelled waiter returned success")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := WithResourceRuntimeLock(ctx, "test-resource-runtime", func(context.Context) error { return nil }); err != nil {
		t.Fatalf("lock leaked after cancellation: %v", err)
	}
}
