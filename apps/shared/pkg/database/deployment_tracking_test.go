package database

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRecordDeploymentLocationPreservesSwarmReplicaSlots(t *testing.T) {
	originalDB := DB
	t.Cleanup(func() { DB = originalDB })

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open deployment tracking database: %v", err)
	}
	DB = db
	if err := DB.AutoMigrate(&DeploymentLocation{}, &NodeMetadata{}); err != nil {
		t.Fatalf("migrate deployment locations: %v", err)
	}

	record := func(id, containerID, slot string) {
		t.Helper()
		location := &DeploymentLocation{
			ID:           id,
			DeploymentID: "deployment-replicas",
			NodeID:       "node-example",
			ContainerID:  containerID,
			ServiceID:    "service-example",
			TaskID:       "task-" + containerID,
			TaskSlot:     slot,
			Status:       "running",
		}
		if err := RecordDeploymentLocation(location); err != nil {
			t.Fatalf("record deployment slot %s: %v", slot, err)
		}
	}

	record("location-slot-1", "container-slot-1-old", "1")
	record("location-slot-2", "container-slot-2", "2")
	record("location-slot-1-new", "container-slot-1-new", "1")

	var locations []DeploymentLocation
	if err := DB.Order("task_slot").Find(&locations).Error; err != nil {
		t.Fatalf("list deployment locations: %v", err)
	}
	if len(locations) != 2 {
		t.Fatalf("deployment location count = %d, want 2", len(locations))
	}
	if locations[0].TaskSlot != "1" || locations[0].ContainerID != "container-slot-1-new" {
		t.Fatalf("slot 1 location = %#v, want replacement container", locations[0])
	}
	if locations[1].TaskSlot != "2" || locations[1].ContainerID != "container-slot-2" {
		t.Fatalf("slot 2 location = %#v, want preserved replica", locations[1])
	}
}
