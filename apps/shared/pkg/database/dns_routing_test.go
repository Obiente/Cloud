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
	if err := DB.Create(&NodeMetadata{ID: nodeID, Hostname: "us-host", IP: "104.243.46.110"}).Error; err != nil {
		t.Fatalf("create node metadata: %v", err)
	}

	ips, err := GetDatabaseNodeIP(databaseID, multiRegionNodeIPs())
	if err != nil {
		t.Fatalf("resolve database node IP: %v", err)
	}
	assertNodeIPs(t, ips, "104.243.46.110")
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
		NodeIP:      "104.243.46.110",
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
	assertNodeIPs(t, ips, "104.243.46.110")
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

	ips, err := GetDatabaseNodeIP(databaseID, map[string][]string{"us-east": {"104.243.46.110"}})
	if err != nil {
		t.Fatalf("resolve legacy single-region database: %v", err)
	}
	assertNodeIPs(t, ips, "104.243.46.110")
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

func TestUpsertDatabaseLocationPreservesExistingPrimaryKey(t *testing.T) {
	setupDNSRoutingTestDB(t)

	existing := &DatabaseLocation{
		ID:          "legacy-location-id",
		DatabaseID:  "db-upsert-test",
		NodeID:      "node-nl",
		NodeIP:      "175.110.112.242",
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
		NodeIP:      "104.243.46.110",
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
	if locations[0].ID != existing.ID || locations[0].NodeIP != "104.243.46.110" {
		t.Fatalf("unexpected reconciled location: %#v", locations[0])
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
		"us-east": {"104.243.46.110"},
		"nl":      {"175.110.112.242"},
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
