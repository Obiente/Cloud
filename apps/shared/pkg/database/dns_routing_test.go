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

	location := &DatabaseLocation{
		ID:          "location-status-test",
		DatabaseID:  "db-status-test",
		NodeID:      "node-us",
		ContainerID: "container-status-test",
		Status:      "running",
	}
	if err := DB.Create(location).Error; err != nil {
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
}

func setupDNSRoutingTestDB(t *testing.T) {
	t.Helper()
	previousDB := DB
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&DatabaseInstance{}, &DatabaseLocation{}, &NodeMetadata{}); err != nil {
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
