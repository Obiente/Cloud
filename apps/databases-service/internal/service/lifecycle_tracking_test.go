package databases

import (
	"context"
	"errors"
	"testing"
	"time"

	"databases-service/internal/proxy"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRetryDatabaseWriteRetriesTransientFailure(t *testing.T) {
	attempts := 0
	err := retryDatabaseWrite(t.Context(), 3, 0, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("transient write failure")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("retry database write: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryDatabaseWriteReturnsAfterExhaustion(t *testing.T) {
	attempts := 0
	err := retryDatabaseWrite(t.Context(), 2, 0, func() error {
		attempts++
		return errors.New("persistent write failure")
	})
	if err == nil {
		t.Fatal("persistent write failure was ignored")
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestPersistDatabaseConnectionIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:connection-persistence-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&database.DatabaseConnection{}); err != nil {
		t.Fatalf("migrate database connection: %v", err)
	}
	service := &Service{connRepo: database.NewDatabaseConnectionRepository(db)}
	connection := &database.DatabaseConnection{
		ID:           "conn-persistence-test",
		DatabaseID:   "db-persistence-test",
		DatabaseName: "db-persistence-test",
		Username:     "test-user",
		Password:     "test-secret",
		Host:         "db-persistence-test.example.test",
		Port:         5432,
	}
	if err := service.persistDatabaseConnection(t.Context(), connection); err != nil {
		t.Fatalf("persist database connection: %v", err)
	}
	connection.Password = "updated-test-secret"
	if err := service.persistDatabaseConnection(t.Context(), connection); err != nil {
		t.Fatalf("repeat database connection persistence: %v", err)
	}

	var stored []database.DatabaseConnection
	if err := db.Find(&stored).Error; err != nil {
		t.Fatalf("load database connections: %v", err)
	}
	if len(stored) != 1 || stored[0].Password != connection.Password {
		t.Fatalf("unexpected persisted connections: %#v", stored)
	}
}

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

func TestRestoreContainerAfterAutoSleepFailureRestartsRunningRoute(t *testing.T) {
	registry := proxy.NewRouteRegistry(nil)
	route := &proxy.Route{
		DatabaseID:  "db-auto-sleep-rollback-test",
		ContainerID: "container-auto-sleep-rollback-test",
		ContainerIP: "obiente-db-auto-sleep-rollback-test",
		DBStatus:    3,
	}
	registry.Register(route)

	startedContainerID := ""
	service := &Service{
		backgroundCtx: t.Context(),
		routeRegistry: registry,
		startStoppedDB: func(_ context.Context, containerID string) error {
			startedContainerID = containerID
			return nil
		},
	}
	if err := service.restoreContainerAfterAutoSleepFailure(route); err != nil {
		t.Fatalf("restore stopped database container: %v", err)
	}
	if startedContainerID != route.ContainerID {
		t.Fatalf("started container %q, want %q", startedContainerID, route.ContainerID)
	}
	storedRoute, ok := registry.LookupByID(route.DatabaseID)
	if !ok {
		t.Fatal("restored route is missing")
	}
	if storedRoute.Stopped || storedRoute.DBStatus != 3 {
		t.Fatalf("expected running route after rollback, got %#v", storedRoute)
	}
}

func TestRestoreContainerAfterAutoSleepFailureRetainsSleepingRouteWhenRestartFails(t *testing.T) {
	registry := proxy.NewRouteRegistry(nil)
	route := &proxy.Route{
		DatabaseID:  "db-auto-sleep-retry-test",
		ContainerID: "container-auto-sleep-retry-test",
		ContainerIP: "obiente-db-auto-sleep-retry-test",
		DBStatus:    3,
	}
	registry.Register(route)

	service := &Service{
		backgroundCtx: t.Context(),
		routeRegistry: registry,
		startStoppedDB: func(context.Context, string) error {
			return errors.New("restart failed")
		},
	}
	if err := service.restoreContainerAfterAutoSleepFailure(route); err == nil {
		t.Fatal("expected the failed container restart to be reported")
	}
	storedRoute, ok := registry.LookupByID(route.DatabaseID)
	if !ok {
		t.Fatal("sleeping route is missing")
	}
	if !storedRoute.Stopped || storedRoute.DBStatus != 12 {
		t.Fatalf("expected sleeping route after failed rollback, got %#v", storedRoute)
	}
}
