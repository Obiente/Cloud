package database

import (
	"context"
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

func TestRecordDeploymentLocationDoesNotServiceUpsertWithoutTaskSlot(t *testing.T) {
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

	for _, containerID := range []string{"container-without-slot-one", "container-without-slot-two"} {
		location := &DeploymentLocation{
			ID:           "location-" + containerID,
			DeploymentID: "deployment-without-slots",
			NodeID:       "node-example",
			ContainerID:  containerID,
			ServiceID:    "service-example",
			TaskID:       "task-" + containerID,
			Status:       "running",
		}
		if err := RecordDeploymentLocation(location); err != nil {
			t.Fatalf("record slotless location %s: %v", containerID, err)
		}
	}

	var locations []DeploymentLocation
	if err := DB.Order("container_id").Find(&locations).Error; err != nil {
		t.Fatalf("list slotless deployment locations: %v", err)
	}
	if len(locations) != 2 {
		t.Fatalf("slotless deployment location count = %d, want 2", len(locations))
	}
}

func TestPinDeploymentVolumeNodeRejectsAffinityMove(t *testing.T) {
	originalDB := DB
	t.Cleanup(func() { DB = originalDB })

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open deployment tracking database: %v", err)
	}
	DB = db
	if err := DB.AutoMigrate(&Deployment{}); err != nil {
		t.Fatalf("migrate deployments: %v", err)
	}
	if err := DB.Create(&Deployment{ID: "deployment-affinity", Name: "Affinity"}).Error; err != nil {
		t.Fatalf("create deployment: %v", err)
	}

	ctx := context.Background()
	if acquired, err := PinDeploymentVolumeNode(ctx, "deployment-affinity", "worker-one"); err != nil || !acquired {
		t.Fatalf("pin initial volume node: %v", err)
	}
	if acquired, err := PinDeploymentVolumeNode(ctx, "deployment-affinity", "worker-one"); err != nil || acquired {
		t.Fatalf("repeat volume node pin: %v", err)
	}
	if _, err := PinDeploymentVolumeNode(ctx, "deployment-affinity", "worker-two"); err == nil {
		t.Fatal("moving an existing volume node affinity succeeded")
	}
	if got, err := GetDeploymentVolumeNode(ctx, "deployment-affinity"); err != nil || got != "worker-one" {
		t.Fatalf("deployment volume node = %q, %v; want worker-one", got, err)
	}
	if err := ReleaseDeploymentVolumeNode(ctx, "deployment-affinity", "worker-two"); err != nil {
		t.Fatalf("release volume node with wrong owner: %v", err)
	}
	if got, err := GetDeploymentVolumeNode(ctx, "deployment-affinity"); err != nil || got != "worker-one" {
		t.Fatalf("volume node after mismatched release = %q, %v; want worker-one", got, err)
	}
	if err := ReleaseDeploymentVolumeNode(ctx, "deployment-affinity", "worker-one"); err != nil {
		t.Fatalf("release volume node: %v", err)
	}
	if got, err := GetDeploymentVolumeNode(ctx, "deployment-affinity"); err != nil || got != "" {
		t.Fatalf("released deployment volume node = %q, %v; want empty", got, err)
	}
}
