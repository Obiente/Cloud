package databases

import (
	"testing"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRestoreDatabaseAfterFailedStopKeepsTrackingActive(t *testing.T) {
	previousDB := database.DB
	db, err := gorm.Open(sqlite.Open("file:failed-stop-tracking-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })
	if err := db.AutoMigrate(
		&database.DatabaseInstance{},
		&database.DatabaseLocation{},
		&database.DatabaseUptimeInterval{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	databaseID := "db-failed-stop-tracking-test"
	containerID := "container-failed-stop-tracking-test"
	instance := &database.DatabaseInstance{ID: databaseID, InstanceID: &containerID, Status: 4}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("create database instance: %v", err)
	}
	if err := db.Create(&database.DatabaseLocation{
		ID:          database.DatabaseLocationID(databaseID, containerID),
		DatabaseID:  databaseID,
		NodeID:      "node-failed-stop-tracking-test",
		ContainerID: containerID,
		Status:      "stopping",
	}).Error; err != nil {
		t.Fatalf("create database location: %v", err)
	}
	endedAt := time.Now().Add(-time.Minute)
	if err := db.Create(&database.DatabaseUptimeInterval{
		ID:          "interval-before-failed-stop",
		DatabaseID:  databaseID,
		ContainerID: containerID,
		NodeID:      "node-failed-stop-tracking-test",
		StartedAt:   endedAt.Add(-time.Hour),
		EndedAt:     &endedAt,
	}).Error; err != nil {
		t.Fatalf("create closed uptime interval: %v", err)
	}

	service := &Service{repo: database.NewDatabaseRepository(db, nil)}
	service.restoreDatabaseAfterFailedStop(t.Context(), instance)

	var storedInstance database.DatabaseInstance
	if err := db.First(&storedInstance, "id = ?", databaseID).Error; err != nil {
		t.Fatalf("load database instance: %v", err)
	}
	if storedInstance.Status != 3 {
		t.Fatalf("expected running database status, got %d", storedInstance.Status)
	}
	var storedLocation database.DatabaseLocation
	if err := db.First(&storedLocation, "container_id = ?", containerID).Error; err != nil {
		t.Fatalf("load database location: %v", err)
	}
	if storedLocation.Status != "running" {
		t.Fatalf("expected running location status, got %q", storedLocation.Status)
	}
	var openIntervals int64
	if err := db.Model(&database.DatabaseUptimeInterval{}).
		Where("database_id = ? AND container_id = ? AND ended_at IS NULL", databaseID, containerID).
		Count(&openIntervals).Error; err != nil {
		t.Fatalf("count open uptime intervals: %v", err)
	}
	if openIntervals != 1 {
		t.Fatalf("expected one open uptime interval, got %d", openIntervals)
	}
}
