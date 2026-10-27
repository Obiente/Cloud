package database

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Run against an isolated Timescale server. Every run owns a fresh database.
func TestConcurrentTimescaleStartupPreservesChunks(t *testing.T) {
	dsn := os.Getenv("OBIENTE_TEST_TIMESCALE_DSN")
	if dsn == "" {
		t.Skip("requires isolated TimescaleDB")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	adminPool, err := admin.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer adminPool.Close()
	name := "obiente_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := admin.Exec("CREATE DATABASE " + name).Error; err != nil {
		t.Fatal(err)
	}
	defer admin.Exec("DROP DATABASE " + name + " WITH (FORCE)")
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.Database = name
	pool := stdlib.OpenDB(*config)
	defer pool.Close()
	pool.SetMaxOpenConns(8)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if err := db.WithContext(ctx).Exec("CREATE EXTENSION IF NOT EXISTS timescaledb").Error; err != nil {
		t.Fatal(err)
	}
	if err := withMetricsSchemaLock(ctx, db, initMetricsTables); err != nil {
		t.Fatal(err)
	}
	if err := db.WithContext(ctx).Exec(`INSERT INTO deployment_metrics (id, deployment_id, container_id, timestamp)
 SELECT n, 'example-deployment', 'example-container', now() - n * interval '1 hour' FROM generate_series(1, 240) n`).Error; err != nil {
		t.Fatal(err)
	}
	var chunks int64
	if err := db.Raw("SELECT count(*) FROM timescaledb_information.chunks WHERE hypertable_name = 'deployment_metrics'").Scan(&chunks).Error; err != nil {
		t.Fatal(err)
	}
	if chunks < 200 {
		t.Fatalf("only %d chunks created", chunks)
	}
	// Simulate an interrupted earlier rollout with a required index missing.
	if err := db.Exec("DROP INDEX idx_deployment_metrics_deployment_timestamp").Error; err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	failures := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); failures <- withMetricsSchemaLock(ctx, db, initMetricsTables) }()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Error(err)
		}
	}
	var count int64
	if err := db.Raw("SELECT count(*) FROM deployment_metrics WHERE deployment_id = 'example-deployment'").Scan(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 240 {
		t.Fatalf("metrics lost: %d", count)
	}
	if !db.Migrator().HasIndex("deployment_metrics", "idx_deployment_metrics_deployment_timestamp") {
		t.Fatal("missing index was not restored")
	}
	t.Log(fmt.Sprintf("4 simultaneous startups preserved %d records across %d Timescale chunks", count, chunks))
}
