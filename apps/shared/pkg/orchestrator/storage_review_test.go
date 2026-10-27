package orchestrator

import (
	"github.com/moby/moby/api/types/container"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestRepeatedStopPreservesMeasuredStorage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	defer raw.Close()
	previous := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = previous })
	if err := db.AutoMigrate(&database.Deployment{}, &database.DeploymentLocation{}, &database.DeploymentStorageMeasurement{}, &database.GameServer{}, &database.VPSInstance{}, &database.DatabaseInstance{}); err != nil {
		t.Fatal(err)
	}
	dep := database.Deployment{ID: "storage-test", OrganizationID: "org-test"}
	location := database.DeploymentLocation{DeploymentID: dep.ID, NodeID: "node-test", ContainerID: "container-test", Status: "stopped"}
	if err := db.Create(&dep).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&location).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.RecordDeploymentStorage(t.Context(), db, dep.ID, location.NodeID, []string{location.ContainerID}, 12345, nil, `[]`); err != nil {
		t.Fatal(err)
	}
	var before database.DeploymentStorageMeasurement
	if err := db.First(&before).Error; err != nil {
		t.Fatal(err)
	}
	client := &recoveryInspectClient{result: container.InspectResponse{State: &container.State{Status: container.StateExited}}}
	dm := &DeploymentManager{nodeID: location.NodeID, dockerClient: client}
	for i := 0; i < 10; i++ {
		dm.captureStorageBeforeStop(t.Context(), dep.ID)
	}
	var after database.DeploymentStorageMeasurement
	if err := db.First(&after).Error; err != nil {
		t.Fatal(err)
	}
	if after.Error != "" || after.StorageBytes != 12345 || !after.MeasuredAt.Equal(before.MeasuredAt) {
		t.Fatalf("measurement invalidated or made artificially fresh: %+v", after)
	}
	if bytes, err := database.BillableStorageBytes(t.Context(), db, dep.OrganizationID, time.Now()); err != nil || bytes != 12345 {
		t.Fatalf("billing lost known storage: %d %v", bytes, err)
	}
	changed := location
	changed.ContainerID = "replacement"
	if dm.stoppedStorageInventoryUnchanged(t.Context(), []database.DeploymentLocation{changed}, before) {
		t.Fatal("replacement container reused old inventory")
	}
	client.result.State.Running = true
	if dm.stoppedStorageInventoryUnchanged(t.Context(), []database.DeploymentLocation{location}, before) {
		t.Fatal("externally restarted container skipped capture")
	}
}
