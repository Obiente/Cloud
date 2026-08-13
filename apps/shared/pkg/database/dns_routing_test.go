package database

import (
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetDatabaseNodeIPPrefersRecordedHost(t *testing.T) {
	setupDNSRoutingTestDB(t)

	nodeID := "node-us"
	databaseID := "db-routing-test"
	if err := DB.Create(&DatabaseInstance{ID: databaseID, NodeID: &nodeID}).Error; err != nil {
		t.Fatalf("create database instance: %v", err)
	}
	if err := DB.Create(&NodeMetadata{ID: nodeID, Hostname: "us-host", IP: "192.0.2.10"}).Error; err != nil {
		t.Fatalf("create node metadata: %v", err)
	}

	ips, err := GetDatabaseNodeIP(databaseID, multiRegionNodeIPs())
	if err != nil {
		t.Fatalf("resolve database node IP: %v", err)
	}
	assertNodeIPs(t, ips, "192.0.2.10")
}

func TestGetDatabaseNodeIPPrefersCurrentLocation(t *testing.T) {
	setupDNSRoutingTestDB(t)

	databaseID := "db-location-test"
	oldNodeID := "node-nl"
	if err := DB.Create(&DatabaseInstance{ID: databaseID, NodeID: &oldNodeID}).Error; err != nil {
		t.Fatalf("create database instance: %v", err)
	}
	if err := DB.Create(&DatabaseLocation{
		ID:          DatabaseLocationID(databaseID, "container-1"),
		DatabaseID:  databaseID,
		NodeID:      "node-us",
		NodeIP:      "192.0.2.10",
		ContainerID: "container-1",
		Status:      "running",
		UpdatedAt:   time.Now(),
	}).Error; err != nil {
		t.Fatalf("create database location: %v", err)
	}

	ips, err := GetDatabaseNodeIP(databaseID, multiRegionNodeIPs())
	if err != nil {
		t.Fatalf("resolve database location IP: %v", err)
	}
	assertNodeIPs(t, ips, "192.0.2.10")
}

func TestGetDatabaseNodeIPUsesSleepingLocationForWakeableDatabase(t *testing.T) {
	setupDNSRoutingTestDB(t)

	databaseID := "db-sleeping-location-test"
	staleNodeID := "node-nl"
	if err := DB.Create(&DatabaseInstance{ID: databaseID, NodeID: &staleNodeID, Status: 12}).Error; err != nil {
		t.Fatalf("create sleeping database instance: %v", err)
	}
	if err := DB.Create(&DatabaseLocation{
		ID:          DatabaseLocationID(databaseID, "container-sleeping"),
		DatabaseID:  databaseID,
		NodeID:      "node-us",
		NodeIP:      "192.0.2.10",
		ContainerID: "container-sleeping",
		Status:      "sleeping",
		UpdatedAt:   time.Now(),
	}).Error; err != nil {
		t.Fatalf("create sleeping database location: %v", err)
	}

	ips, err := GetDatabaseNodeIP(databaseID, multiRegionNodeIPs())
	if err != nil {
		t.Fatalf("resolve sleeping database location IP: %v", err)
	}
	assertNodeIPs(t, ips, "192.0.2.10")
}

func TestGetDatabaseNodeIPDoesNotFallBackFromUnresolvedActiveLocation(t *testing.T) {
	setupDNSRoutingTestDB(t)

	databaseID := "db-authoritative-location-test"
	staleNodeID := "node-nl"
	if err := DB.Create(&DatabaseInstance{ID: databaseID, NodeID: &staleNodeID}).Error; err != nil {
		t.Fatalf("create database instance: %v", err)
	}
	if err := DB.Create(&NodeMetadata{ID: staleNodeID, Hostname: "nl-host", Region: "nl"}).Error; err != nil {
		t.Fatalf("create stale node metadata: %v", err)
	}
	if err := DB.Create(&DatabaseLocation{
		ID:          DatabaseLocationID(databaseID, "container-authoritative"),
		DatabaseID:  databaseID,
		NodeID:      "node-current",
		NodeIP:      "203.0.113.55",
		ContainerID: "container-authoritative",
		Status:      "running",
		UpdatedAt:   time.Now(),
	}).Error; err != nil {
		t.Fatalf("create active database location: %v", err)
	}

	_, err := GetDatabaseNodeIP(databaseID, multiRegionNodeIPs())
	if err == nil {
		t.Fatal("expected unresolved active location to remain authoritative")
	}
	if !strings.Contains(err.Error(), "failed to resolve active database location") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetDatabaseNodeIPRejectsDefaultFallbackForActiveLocation(t *testing.T) {
	setupDNSRoutingTestDB(t)

	databaseID := "db-authoritative-default-test"
	containerID := "container-authoritative-default"
	if err := DB.Create(&DatabaseInstance{ID: databaseID, InstanceID: &containerID}).Error; err != nil {
		t.Fatalf("create database instance: %v", err)
	}
	if err := DB.Create(&DatabaseLocation{
		ID:          DatabaseLocationID(databaseID, containerID),
		DatabaseID:  databaseID,
		NodeID:      "node-without-metadata",
		NodeIP:      "203.0.113.55",
		ContainerID: containerID,
		Status:      "running",
		UpdatedAt:   time.Now(),
	}).Error; err != nil {
		t.Fatalf("create active database location: %v", err)
	}

	_, err := GetDatabaseNodeIP(databaseID, map[string][]string{
		"default": {"192.0.2.10"},
		"eu-west": {"198.51.100.20"},
	})
	if err == nil {
		t.Fatal("expected authoritative location to reject the default fallback")
	}
	if !strings.Contains(err.Error(), "authoritative DNS fallback is disabled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetDatabaseNodeIPRejectsAmbiguousRegionFallback(t *testing.T) {
	setupDNSRoutingTestDB(t)

	databaseID := "db-unassigned-test"
	if err := DB.Create(&DatabaseInstance{ID: databaseID}).Error; err != nil {
		t.Fatalf("create database instance: %v", err)
	}

	_, err := GetDatabaseNodeIP(databaseID, multiRegionNodeIPs())
	if err == nil {
		t.Fatal("expected unassigned multi-region database resolution to fail")
	}
	if !strings.Contains(err.Error(), "refusing ambiguous cross-node DNS fallback") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetDatabaseNodeIPAllowsSingleRegionCompatibilityFallback(t *testing.T) {
	setupDNSRoutingTestDB(t)

	databaseID := "db-legacy-test"
	if err := DB.Create(&DatabaseInstance{ID: databaseID}).Error; err != nil {
		t.Fatalf("create database instance: %v", err)
	}

	ips, err := GetDatabaseNodeIP(databaseID, map[string][]string{"us-east": {"192.0.2.10"}})
	if err != nil {
		t.Fatalf("resolve legacy single-region database: %v", err)
	}
	assertNodeIPs(t, ips, "192.0.2.10")
}

func TestResolvePreferredNodeIPsRejectsAmbiguousUnknownNode(t *testing.T) {
	setupDNSRoutingTestDB(t)

	_, err := resolvePreferredNodeIPs("missing-node", "", multiRegionNodeIPs())
	if err == nil {
		t.Fatal("expected unknown node with multi-region fallback to fail")
	}
	if !strings.Contains(err.Error(), "refusing ambiguous cross-node DNS fallback") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolvePreferredNodeIPsIgnoresUnconfiguredAdvertiseAddress(t *testing.T) {
	setupDNSRoutingTestDB(t)

	if err := DB.Create(&NodeMetadata{
		ID:       "node-private",
		Hostname: "private-host",
		IP:       "203.0.113.55",
		Region:   "us-east",
	}).Error; err != nil {
		t.Fatalf("create private node metadata: %v", err)
	}

	ips, err := resolvePreferredNodeIPs("node-private", "203.0.113.55", multiRegionNodeIPs())
	if err != nil {
		t.Fatalf("resolve configured public address: %v", err)
	}
	assertNodeIPs(t, ips, "192.0.2.10")
}

func TestUpsertDatabaseLocationPreservesExistingPrimaryKey(t *testing.T) {
	setupDNSRoutingTestDB(t)

	existing := &DatabaseLocation{
		ID:          "legacy-location-id",
		DatabaseID:  "db-upsert-test",
		NodeID:      "node-nl",
		NodeIP:      "198.51.100.20",
		ContainerID: "container-upsert",
		Status:      "running",
	}
	if err := DB.Create(existing).Error; err != nil {
		t.Fatalf("create existing database location: %v", err)
	}

	updated := &DatabaseLocation{
		ID:          DatabaseLocationID("db-upsert-test", "container-upsert"),
		DatabaseID:  "db-upsert-test",
		NodeID:      "node-us",
		NodeIP:      "192.0.2.10",
		ContainerID: "container-upsert",
		Status:      "running",
	}
	if err := UpsertDatabaseLocation(updated); err != nil {
		t.Fatalf("upsert database location: %v", err)
	}

	var locations []DatabaseLocation
	if err := DB.Find(&locations).Error; err != nil {
		t.Fatalf("query database locations: %v", err)
	}
	if len(locations) != 1 {
		t.Fatalf("expected one location, got %d", len(locations))
	}
	if locations[0].ID != existing.ID || locations[0].NodeIP != "192.0.2.10" {
		t.Fatalf("unexpected reconciled location: %#v", locations[0])
	}
}

func TestUpdateDatabaseLocationStatus(t *testing.T) {
	setupDNSRoutingTestDB(t)

	containerID := "container-status-test"
	if err := DB.Create(&DatabaseInstance{ID: "db-status-test", InstanceID: &containerID}).Error; err != nil {
		t.Fatalf("create database instance: %v", err)
	}
	historical := &DatabaseLocation{
		ID:          "historical-location-status-test",
		DatabaseID:  "db-status-test",
		NodeID:      "node-old",
		ContainerID: "container-old-status-test",
		Status:      "running",
	}
	if err := DB.Create(historical).Error; err != nil {
		t.Fatalf("create historical database location: %v", err)
	}
	if err := DB.Create(&DatabaseUptimeInterval{
		ID:          "historical-interval-status-test",
		DatabaseID:  historical.DatabaseID,
		ContainerID: historical.ContainerID,
		NodeID:      historical.NodeID,
		StartedAt:   time.Now().Add(-time.Hour),
	}).Error; err != nil {
		t.Fatalf("create historical uptime interval: %v", err)
	}

	location := &DatabaseLocation{
		ID:          "location-status-test",
		DatabaseID:  "db-status-test",
		NodeID:      "node-us",
		ContainerID: containerID,
		Status:      "running",
	}
	if err := UpsertDatabaseLocation(location); err != nil {
		t.Fatalf("create database location: %v", err)
	}

	if err := UpdateDatabaseLocationStatus(t.Context(), location.DatabaseID, "sleeping"); err != nil {
		t.Fatalf("update database location status: %v", err)
	}

	var updated DatabaseLocation
	if err := DB.First(&updated, "id = ?", location.ID).Error; err != nil {
		t.Fatalf("load database location: %v", err)
	}
	if updated.Status != "sleeping" {
		t.Fatalf("expected sleeping location status, got %q", updated.Status)
	}

	var firstInterval DatabaseUptimeInterval
	if err := DB.First(&firstInterval, "database_id = ?", location.DatabaseID).Error; err != nil {
		t.Fatalf("load closed uptime interval: %v", err)
	}
	if firstInterval.EndedAt == nil {
		t.Fatal("expected sleeping transition to close the running interval")
	}

	if err := UpdateDatabaseLocationStatus(t.Context(), location.DatabaseID, "running"); err != nil {
		t.Fatalf("restart database location: %v", err)
	}
	var intervals []DatabaseUptimeInterval
	if err := DB.Order("started_at").Find(&intervals, "database_id = ?", location.DatabaseID).Error; err != nil {
		t.Fatalf("load uptime intervals: %v", err)
	}
	if len(intervals) != 3 {
		t.Fatalf("expected historical and current uptime intervals, got %#v", intervals)
	}
	var historicalAfter DatabaseLocation
	if err := DB.First(&historicalAfter, "id = ?", historical.ID).Error; err != nil {
		t.Fatalf("load historical location: %v", err)
	}
	if historicalAfter.Status != "stopped" {
		t.Fatalf("expected historical location to remain stopped, got %q", historicalAfter.Status)
	}
	var openIntervals int64
	if err := DB.Model(&DatabaseUptimeInterval{}).
		Where("database_id = ? AND ended_at IS NULL", location.DatabaseID).
		Count(&openIntervals).Error; err != nil {
		t.Fatalf("count open uptime intervals: %v", err)
	}
	if openIntervals != 1 {
		t.Fatalf("expected exactly one open interval for the current container, got %d", openIntervals)
	}
}

func TestBackfillDatabaseUptimeIntervalsPreservesLegacyTimestamps(t *testing.T) {
	setupDNSRoutingTestDB(t)

	runningStart := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	stoppedStart := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)
	stoppedEnd := time.Date(2026, time.May, 3, 0, 0, 0, 0, time.UTC)
	locations := []DatabaseLocation{
		{
			ID:          "legacy-running-location",
			DatabaseID:  "db-legacy-running",
			NodeID:      "node-us",
			ContainerID: "container-legacy-running",
			Status:      "running",
			CreatedAt:   runningStart,
			UpdatedAt:   runningStart.Add(time.Hour),
		},
		{
			ID:          "legacy-stopped-location",
			DatabaseID:  "db-legacy-stopped",
			NodeID:      "node-nl",
			ContainerID: "container-legacy-stopped",
			Status:      "stopped",
			CreatedAt:   stoppedStart,
			UpdatedAt:   stoppedEnd,
		},
	}
	if err := DB.Create(&locations).Error; err != nil {
		t.Fatalf("create legacy database locations: %v", err)
	}
	if err := BackfillDatabaseUptimeIntervals(); err != nil {
		t.Fatalf("backfill database uptime intervals: %v", err)
	}
	if err := BackfillDatabaseUptimeIntervals(); err != nil {
		t.Fatalf("rerun database uptime backfill: %v", err)
	}

	var intervals []DatabaseUptimeInterval
	if err := DB.Order("started_at").Find(&intervals).Error; err != nil {
		t.Fatalf("load backfilled uptime intervals: %v", err)
	}
	if len(intervals) != 2 {
		t.Fatalf("expected two backfilled intervals, got %d", len(intervals))
	}
	if !intervals[0].StartedAt.Equal(stoppedStart) || intervals[0].EndedAt == nil || !intervals[0].EndedAt.Equal(stoppedEnd) {
		t.Fatalf("unexpected stopped legacy interval: %#v", intervals[0])
	}
	if !intervals[1].StartedAt.Equal(runningStart) || intervals[1].EndedAt != nil {
		t.Fatalf("unexpected running legacy interval: %#v", intervals[1])
	}
}

func TestDatabaseUptimeIntervalAllowsOnlyOneOpenRowPerContainer(t *testing.T) {
	setupDNSRoutingTestDB(t)

	location := &DatabaseLocation{
		DatabaseID:  "db-atomic-uptime-test",
		ContainerID: "container-atomic-uptime-test",
		NodeID:      "node-us",
	}
	now := time.Date(2026, time.June, 10, 12, 0, 0, 0, time.UTC)
	if err := ensureDatabaseUptimeInterval(DB, location, now); err != nil {
		t.Fatalf("create first uptime interval: %v", err)
	}
	if err := ensureDatabaseUptimeInterval(DB, location, now.Add(time.Second)); err != nil {
		t.Fatalf("ignore duplicate uptime interval: %v", err)
	}

	var openIntervals int64
	if err := DB.Model(&DatabaseUptimeInterval{}).
		Where("database_id = ? AND container_id = ? AND ended_at IS NULL", location.DatabaseID, location.ContainerID).
		Count(&openIntervals).Error; err != nil {
		t.Fatalf("count open uptime intervals: %v", err)
	}
	if openIntervals != 1 {
		t.Fatalf("expected one open uptime interval, got %d", openIntervals)
	}

	duplicate := DatabaseUptimeInterval{
		ID:          "duplicate-open-interval",
		DatabaseID:  location.DatabaseID,
		ContainerID: location.ContainerID,
		NodeID:      location.NodeID,
		StartedAt:   now.Add(2 * time.Second),
	}
	if err := DB.Create(&duplicate).Error; err == nil {
		t.Fatal("expected the database to reject a second open uptime interval")
	}
}

func setupDNSRoutingTestDB(t *testing.T) {
	t.Helper()
	previousDB := DB
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&DatabaseInstance{}, &DatabaseLocation{}, &DatabaseUptimeInterval{}, &NodeMetadata{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	DB = db
	t.Cleanup(func() { DB = previousDB })
}

func multiRegionNodeIPs() map[string][]string {
	return map[string][]string{
		"us-east": {"192.0.2.10"},
		"nl":      {"198.51.100.20"},
	}
}

func assertNodeIPs(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}
