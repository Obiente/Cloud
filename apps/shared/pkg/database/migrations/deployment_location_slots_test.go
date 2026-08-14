package migrations

import (
	"testing"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnforceUniqueDeploymentLocationSlots(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open migration test database: %v", err)
	}
	if err := db.AutoMigrate(&database.DeploymentLocation{}, &database.NodeMetadata{}); err != nil {
		t.Fatalf("migrate deployment tracking tables: %v", err)
	}
	if err := db.Exec("DROP INDEX IF EXISTS idx_deployment_service_slot_unique").Error; err != nil {
		t.Fatalf("drop unique slot index before migration: %v", err)
	}
	for _, nodeID := range []string{"node-old", "node-current"} {
		if err := db.Create(&database.NodeMetadata{
			ID:              nodeID,
			Hostname:        nodeID,
			DeploymentCount: 9,
		}).Error; err != nil {
			t.Fatalf("create node %s: %v", nodeID, err)
		}
	}
	now := time.Now().UTC()
	locations := []database.DeploymentLocation{
		{
			ID:           "location-old",
			DeploymentID: "deployment-example",
			NodeID:       "node-old",
			ContainerID:  "container-old",
			ServiceID:    "service-example",
			TaskSlot:     "1",
			Status:       "running",
			CreatedAt:    now.Add(-time.Minute),
			UpdatedAt:    now.Add(-time.Minute),
		},
		{
			ID:           "location-current",
			DeploymentID: "deployment-example",
			NodeID:       "node-current",
			ContainerID:  "container-current",
			ServiceID:    "service-example",
			TaskSlot:     "1",
			Status:       "running",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}
	if err := db.Create(&locations).Error; err != nil {
		t.Fatalf("seed duplicate deployment slots: %v", err)
	}

	if err := enforceUniqueDeploymentLocationSlots(db); err != nil {
		t.Fatalf("enforce unique deployment slots: %v", err)
	}

	var retained []database.DeploymentLocation
	if err := db.Find(&retained).Error; err != nil {
		t.Fatalf("list retained deployment locations: %v", err)
	}
	if len(retained) != 1 || retained[0].ID != "location-current" {
		t.Fatalf("retained deployment locations = %#v, want newest slot only", retained)
	}
	var nodes []database.NodeMetadata
	if err := db.Order("id").Find(&nodes).Error; err != nil {
		t.Fatalf("list recalculated node counts: %v", err)
	}
	if len(nodes) != 2 || nodes[0].DeploymentCount != 1 || nodes[1].DeploymentCount != 0 {
		t.Fatalf("recalculated node counts = %#v, want current=1 and old=0", nodes)
	}
	duplicate := database.DeploymentLocation{
		ID:           "location-duplicate",
		DeploymentID: "deployment-example",
		NodeID:       "node-current",
		ContainerID:  "container-duplicate",
		ServiceID:    "service-example",
		TaskSlot:     "1",
		Status:       "running",
	}
	if err := db.Create(&duplicate).Error; err == nil {
		t.Fatal("migration did not enforce unique deployment slots")
	}
}
