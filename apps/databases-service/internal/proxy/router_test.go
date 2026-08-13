package proxy

import (
	"context"
	"testing"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/docker"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLoadFromDatabasePreservesAndBackfillsRedisProxyPorts(t *testing.T) {
	previousDB := database.DB
	db, err := gorm.Open(sqlite.Open("file:redis-proxy-port-routing?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })
	if err := db.AutoMigrate(&database.DatabaseInstance{}, &database.DatabaseConnection{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	instances := []database.DatabaseInstance{
		{ID: "db-redis-existing", Type: 4, Status: 3},
		{ID: "db-redis-legacy", Type: 4, Status: 3},
	}
	if err := db.Create(&instances).Error; err != nil {
		t.Fatalf("create Redis instances: %v", err)
	}
	connections := []database.DatabaseConnection{
		{ID: "conn-redis-existing", DatabaseID: instances[0].ID, ProxyPort: 16380},
		{ID: "conn-redis-legacy", DatabaseID: instances[1].ID},
	}
	if err := db.Create(&connections).Error; err != nil {
		t.Fatalf("create Redis connections: %v", err)
	}

	registry := NewRouteRegistry(nil)
	if err := registry.LoadFromDatabase(context.Background()); err != nil {
		t.Fatalf("load Redis routes: %v", err)
	}
	existingRoute, ok := registry.LookupByID(instances[0].ID)
	if !ok || existingRoute.RedisPort != 16380 {
		t.Fatalf("persisted Redis proxy port was not preserved: %#v", existingRoute)
	}
	legacyRoute, ok := registry.LookupByID(instances[1].ID)
	if !ok || legacyRoute.RedisPort == 0 || legacyRoute.RedisPort == existingRoute.RedisPort {
		t.Fatalf("legacy Redis proxy port was not uniquely backfilled: %#v", legacyRoute)
	}
	var storedLegacy database.DatabaseConnection
	if err := db.First(&storedLegacy, "database_id = ?", instances[1].ID).Error; err != nil {
		t.Fatalf("load backfilled Redis connection: %v", err)
	}
	if int(storedLegacy.ProxyPort) != legacyRoute.RedisPort {
		t.Fatalf("stored proxy port %d does not match route %d", storedLegacy.ProxyPort, legacyRoute.RedisPort)
	}
}

func TestControlPlaneDatabaseSyncRefreshesRoutes(t *testing.T) {
	previousDB := database.DB
	db, err := gorm.Open(sqlite.Open("file:control-plane-route-sync?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })
	if err := db.AutoMigrate(&database.DatabaseInstance{}, &database.DatabaseConnection{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	registry := NewRouteRegistry(nil)
	syncCtx, cancel := context.WithCancel(t.Context())
	done := registry.StartDatabaseSync(syncCtx, 5*time.Millisecond)
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("control-plane route sync did not stop")
		}
	})

	instance := &database.DatabaseInstance{ID: "db-control-plane-sync-test", Type: 1, Status: 12}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("create database instance: %v", err)
	}
	if err := db.Create(&database.DatabaseConnection{
		ID:         "conn-control-plane-sync-test",
		DatabaseID: instance.ID,
		Username:   "test-user",
		Password:   "test-secret",
		Port:       5432,
	}).Error; err != nil {
		t.Fatalf("create database connection: %v", err)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if route, ok := registry.LookupByID(instance.ID); ok && route.Stopped && route.DBStatus == 12 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("control-plane route registry did not refresh")
}

func TestLocalRouteRefreshRetainsSnapshotWithoutNodeDiscovery(t *testing.T) {
	previousDB := database.DB
	db, err := gorm.Open(sqlite.Open("file:local-route-refresh-retention?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	database.DB = db
	t.Cleanup(func() { database.DB = previousDB })
	if err := db.AutoMigrate(&database.DatabaseInstance{}, &database.DatabaseConnection{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	instance := &database.DatabaseInstance{ID: "db-route-retention-test", Type: 1, Status: 3}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("create database instance: %v", err)
	}
	if err := db.Create(&database.DatabaseConnection{ID: "conn-route-retention-test", DatabaseID: instance.ID, Port: 5432}).Error; err != nil {
		t.Fatalf("create database connection: %v", err)
	}

	registry := NewRouteRegistry(nil)
	registry.SetLocalRoutesOnly(true)
	registry.Register(&Route{
		DatabaseID:   instance.ID,
		ContainerID:  "container-route-retention-test",
		ContainerIP:  "obiente-db-route-retention-test",
		InternalPort: 5432,
	})
	if err := registry.LoadFromDatabase(t.Context()); err == nil {
		t.Fatal("expected local refresh without Docker discovery to fail")
	}
	route, ok := registry.LookupByID(instance.ID)
	if !ok || route.ContainerIP != "obiente-db-route-retention-test" || route.ContainerID != "container-route-retention-test" {
		t.Fatalf("working route snapshot was replaced: %#v", route)
	}
}

func TestDatabaseLocationStatus(t *testing.T) {
	tests := map[int32]string{
		0:  "created",
		1:  "creating",
		2:  "starting",
		3:  "running",
		4:  "stopping",
		5:  "stopped",
		6:  "backing_up",
		7:  "restoring",
		8:  "failed",
		9:  "deleting",
		10: "deleted",
		11: "suspended",
		12: "sleeping",
	}

	for status, want := range tests {
		if got := databaseLocationStatus(status); got != want {
			t.Errorf("databaseLocationStatus(%d) = %q, want %q", status, got, want)
		}
	}
}

func TestDatabaseUptimeNeedsRepairOnlyForMissingLocalInterval(t *testing.T) {
	if databaseUptimeNeedsRepair(3, true, false, true) {
		t.Fatal("healthy open interval unexpectedly needs repair")
	}
	if databaseUptimeNeedsRepair(3, false, false, false) {
		t.Fatal("remote database unexpectedly needs local interval repair")
	}
	if databaseUptimeNeedsRepair(3, true, true, false) {
		t.Fatal("location upsert already repaired the missing interval")
	}
	if !databaseUptimeNeedsRepair(3, true, false, false) {
		t.Fatal("missing local running interval did not request repair")
	}
}

func TestDatabaseContainerOwnedLocallyUsesPersistedOwnerForMissingContainer(t *testing.T) {
	localNodeID := "node-local-owner-test"
	instance := &database.DatabaseInstance{NodeID: &localNodeID}
	location := databaseLocationSnapshot{NodeID: localNodeID}
	if !databaseContainerOwnedLocally(instance, location, localNodeID, false) {
		t.Fatal("missing container with a persisted local owner was treated as remote")
	}

	remoteNodeID := "node-remote-owner-test"
	remoteInstance := &database.DatabaseInstance{NodeID: &remoteNodeID}
	remoteLocation := databaseLocationSnapshot{NodeID: remoteNodeID}
	if databaseContainerOwnedLocally(remoteInstance, remoteLocation, localNodeID, false) {
		t.Fatal("remote container was treated as locally owned")
	}
	if !databaseContainerOwnedLocally(remoteInstance, remoteLocation, localNodeID, true) {
		t.Fatal("locally listed container did not override stale ownership metadata")
	}

	staleInstance := &database.DatabaseInstance{NodeID: &localNodeID}
	if databaseContainerOwnedLocally(staleInstance, remoteLocation, localNodeID, false) {
		t.Fatal("stale instance owner overrode the authoritative location owner")
	}
}

func TestDatabaseContainerTerminallyUnavailableIgnoresTransientStates(t *testing.T) {
	for _, state := range []string{"running", "restarting", "created", "paused", "removing"} {
		if databaseContainerTerminallyUnavailable(state, true) {
			t.Fatalf("transient container state %q was treated as terminal", state)
		}
	}
	for _, state := range []string{"exited", "dead"} {
		if !databaseContainerTerminallyUnavailable(state, true) {
			t.Fatalf("terminal container state %q was not reconciled", state)
		}
	}
	if !databaseContainerTerminallyUnavailable("", false) {
		t.Fatal("missing locally owned container was not treated as terminal")
	}
}

func TestReconcileStoppedDatabaseClosesTracking(t *testing.T) {
	previousDB := database.DB
	db, err := gorm.Open(sqlite.Open("file:proxy-stopped-reconciliation?mode=memory&cache=shared"), &gorm.Config{})
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

	databaseID := "db-external-stop-test"
	containerID := "container-external-stop-test"
	instance := &database.DatabaseInstance{ID: databaseID, InstanceID: &containerID, Status: 3}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("create database instance: %v", err)
	}
	if err := db.Create(&database.DatabaseLocation{
		ID:          database.DatabaseLocationID(databaseID, containerID),
		DatabaseID:  databaseID,
		NodeID:      "node-external-stop-test",
		ContainerID: containerID,
		Status:      "running",
	}).Error; err != nil {
		t.Fatalf("create database location: %v", err)
	}
	if err := db.Create(&database.DatabaseUptimeInterval{
		ID:          "interval-external-stop-test",
		DatabaseID:  databaseID,
		ContainerID: containerID,
		NodeID:      "node-external-stop-test",
		StartedAt:   time.Now().Add(-time.Hour),
	}).Error; err != nil {
		t.Fatalf("create uptime interval: %v", err)
	}

	if err := reconcileStoppedDatabase(t.Context(), instance); err != nil {
		t.Fatalf("reconcile stopped database: %v", err)
	}
	var storedInstance database.DatabaseInstance
	if err := db.First(&storedInstance, "id = ?", databaseID).Error; err != nil {
		t.Fatalf("load database instance: %v", err)
	}
	if storedInstance.Status != 5 {
		t.Fatalf("database status = %d, want stopped", storedInstance.Status)
	}
	var storedLocation database.DatabaseLocation
	if err := db.First(&storedLocation, "container_id = ?", containerID).Error; err != nil {
		t.Fatalf("load database location: %v", err)
	}
	if storedLocation.Status != "stopped" {
		t.Fatalf("location status = %q, want stopped", storedLocation.Status)
	}
	var openIntervals int64
	if err := db.Model(&database.DatabaseUptimeInterval{}).
		Where("database_id = ? AND ended_at IS NULL", databaseID).
		Count(&openIntervals).Error; err != nil {
		t.Fatalf("count open uptime intervals: %v", err)
	}
	if openIntervals != 0 {
		t.Fatalf("open uptime intervals = %d, want 0", openIntervals)
	}
}

func TestReconcileRunningDatabaseReopensTracking(t *testing.T) {
	previousDB := database.DB
	db, err := gorm.Open(sqlite.Open("file:proxy-running-reconciliation?mode=memory&cache=shared"), &gorm.Config{})
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

	databaseID := "db-external-start-test"
	containerID := "container-external-start-test"
	instance := &database.DatabaseInstance{ID: databaseID, InstanceID: &containerID, Status: 12}
	if err := db.Create(instance).Error; err != nil {
		t.Fatalf("create database instance: %v", err)
	}
	if err := db.Create(&database.DatabaseLocation{
		ID:          database.DatabaseLocationID(databaseID, containerID),
		DatabaseID:  databaseID,
		NodeID:      "node-external-start-test",
		ContainerID: containerID,
		Status:      "sleeping",
	}).Error; err != nil {
		t.Fatalf("create sleeping database location: %v", err)
	}

	if !databaseContainerIsRunning(" running ") {
		t.Fatal("running container state was not recognized")
	}
	if err := reconcileRunningDatabase(t.Context(), instance); err != nil {
		t.Fatalf("reconcile running database: %v", err)
	}
	var storedInstance database.DatabaseInstance
	if err := db.First(&storedInstance, "id = ?", databaseID).Error; err != nil {
		t.Fatalf("load database instance: %v", err)
	}
	if storedInstance.Status != 3 {
		t.Fatalf("database status = %d, want running", storedInstance.Status)
	}
	var storedLocation database.DatabaseLocation
	if err := db.First(&storedLocation, "container_id = ?", containerID).Error; err != nil {
		t.Fatalf("load database location: %v", err)
	}
	if storedLocation.Status != "running" {
		t.Fatalf("location status = %q, want running", storedLocation.Status)
	}
	var openIntervals int64
	if err := db.Model(&database.DatabaseUptimeInterval{}).
		Where("database_id = ? AND ended_at IS NULL", databaseID).
		Count(&openIntervals).Error; err != nil {
		t.Fatalf("count open uptime intervals: %v", err)
	}
	if openIntervals != 1 {
		t.Fatalf("open uptime intervals = %d, want 1", openIntervals)
	}
}

func TestDatabaseLocationNeedsReconciliation(t *testing.T) {
	nodeID := "node-routing-test"
	instance := &database.DatabaseInstance{NodeID: &nodeID, Status: 3}
	localNode := docker.NodeIdentity{
		ID:       nodeID,
		IP:       "192.0.2.10",
		Hostname: "routing-host",
	}
	current := databaseLocationSnapshot{
		NodeID:       nodeID,
		NodeIP:       localNode.IP,
		NodeHostname: localNode.Hostname,
		Status:       "running",
	}

	if databaseLocationNeedsReconciliation(instance, current, localNode) {
		t.Fatal("current location unexpectedly needs reconciliation")
	}

	tests := map[string]databaseLocationSnapshot{
		"changed address": {
			NodeID:       nodeID,
			NodeIP:       "198.51.100.20",
			NodeHostname: localNode.Hostname,
			Status:       "running",
		},
		"changed hostname": {
			NodeID:       nodeID,
			NodeIP:       localNode.IP,
			NodeHostname: "old-routing-host",
			Status:       "running",
		},
		"stale status": {
			NodeID:       nodeID,
			NodeIP:       localNode.IP,
			NodeHostname: localNode.Hostname,
			Status:       "starting",
		},
	}
	for name, location := range tests {
		t.Run(name, func(t *testing.T) {
			if !databaseLocationNeedsReconciliation(instance, location, localNode) {
				t.Fatal("stale location did not request reconciliation")
			}
		})
	}
}
