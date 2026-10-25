package database

import (
	"context"
	"errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestGameServerSuspension(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&GameServerSuspension{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, r := range []GameServerSuspension{{ID: "active", GameServerID: "blocked", OrganizationID: "test-org", SuspendedBy: "test-actor", SuspendedAt: now}, {ID: "lifted", GameServerID: "allowed", OrganizationID: "test-org", SuspendedBy: "test-actor", SuspendedAt: now, LiftedAt: &now}} {
		if err = db.Create(&r).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err = EnsureGameServerNotSuspended(context.Background(), db, "blocked"); !errors.Is(err, ErrGameServerSuspended) {
		t.Fatalf("active suspension allowed: %v", err)
	}
	for _, id := range []string{"allowed", "unrelated"} {
		if err = EnsureGameServerNotSuspended(context.Background(), db, id); err != nil {
			t.Fatalf("%s blocked: %v", id, err)
		}
	}
	if err = EnsureGameServerNotSuspended(context.Background(), nil, "blocked"); err == nil {
		t.Fatal("nil database did not fail closed")
	}
	sqlDB, _ := db.DB()
	sqlDB.Close()
	if err = EnsureGameServerNotSuspended(context.Background(), db, "unrelated"); err == nil {
		t.Fatal("unavailable database did not fail closed")
	}
}
