package orchestrator

import (
	"context"
	"errors"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestQuarantinedGameServerCannotStartOrRecreate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&database.GameServerSuspension{}); err != nil {
		t.Fatal(err)
	}
	row := database.GameServerSuspension{ID: "test-suspension", GameServerID: "blocked", OrganizationID: "test-org", SuspendedBy: "test-actor", SuspendedAt: time.Now()}
	if err = db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	original := database.DB
	database.DB = db
	defer func() { database.DB = original }()
	gsm := &GameServerManager{}
	for name, fn := range map[string]func() error{
		"create": func() error {
			return gsm.CreateGameServer(context.Background(), &GameServerConfig{GameServerID: "blocked"})
		},
		"start":   func() error { return gsm.StartGameServer(context.Background(), "blocked") },
		"restart": func() error { return gsm.RestartGameServer(context.Background(), "blocked") },
	} {
		t.Run(name, func(t *testing.T) {
			if err := fn(); !errors.Is(err, database.ErrGameServerSuspended) {
				t.Fatalf("quarantine bypassed: %v", err)
			}
		})
	}
}
