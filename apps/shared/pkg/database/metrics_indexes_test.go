package database

import (
	"context"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"strings"
	"testing"
	"time"
)

type indexDDLLog struct {
	logger.Interface
	statements int
}

func (l *indexDDLLog) Trace(_ context.Context, _ time.Time, query func() (string, int64), _ error) {
	sql, _ := query()
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(sql)), "CREATE INDEX") {
		l.statements++
	}
}
func TestMetricsIndexesSkipExistingDDL(t *testing.T) {
	log := &indexDDLLog{Interface: logger.Default}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: log})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	raw.SetMaxOpenConns(1)
	defer raw.Close()
	for _, table := range []string{"deployment_metrics", "deployment_usage_hourly", "deployment_runtime_logs", "game_server_metrics", "game_server_usage_hourly"} {
		if err := db.Exec("CREATE TABLE " + table + " (deployment_id text, game_server_id text, container_id text, service_name text, timestamp datetime, hour datetime)").Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := createMetricsIndexes(db); err != nil {
		t.Fatal(err)
	}
	if log.statements != 10 {
		t.Fatalf("created %d indexes", log.statements)
	}
	log.statements = 0
	if err := createMetricsIndexes(db); err != nil {
		t.Fatal(err)
	}
	if log.statements != 0 {
		t.Fatalf("repeated startup issued %d index DDL statements", log.statements)
	}
	if err := createMetricsIndexes(nil); err == nil {
		t.Fatal("nil database accepted")
	}
}
