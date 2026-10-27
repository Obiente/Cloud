package database

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStorageSnapshotsRequireAllNodesAndCurrentInventory(t *testing.T) {
	db := newDeploymentRepositoryTestDB(t)
	if err := db.AutoMigrate(&DeploymentLocation{}, &DeploymentStorageMeasurement{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	seedDeployments(t, db, &Deployment{ID: "dep", OrganizationID: "org"})
	for _, l := range []DeploymentLocation{{ID: "a", DeploymentID: "dep", NodeID: "node-a", ContainerID: "container-a", Status: "running"}, {ID: "b", DeploymentID: "dep", NodeID: "node-b", ContainerID: "container-b", Status: "stopped"}} {
		if err := db.Create(&l).Error; err != nil {
			t.Fatal(err)
		}
	}
	assertSnapshot := func(available bool, bytes int64) {
		t.Helper()
		var dep Deployment
		if err := db.First(&dep, "id = ?", "dep").Error; err != nil {
			t.Fatal(err)
		}
		if got := StorageMeasurementAvailable(dep.StorageMeasuredAt, dep.StorageMeasurementError, time.Now()); got != available {
			t.Fatalf("availability=%v, want %v (%s)", got, available, dep.StorageMeasurementError)
		}
		if available && dep.StorageBytes != bytes {
			t.Fatalf("storage=%d, want %d", dep.StorageBytes, bytes)
		}
	}
	record := func(node, container string, bytes int64, failure error) {
		t.Helper()
		if err := RecordDeploymentStorage(ctx, db, "dep", node, []string{container, container}, bytes, failure); err != nil {
			t.Fatal(err)
		}
	}
	record("node-a", "container-a", 1024, nil)
	assertSnapshot(false, 0)
	record("node-b", "container-b", 2048, nil)
	assertSnapshot(true, 3072)
	record("node-b", "container-b", 0, errors.New("Docker unavailable"))
	assertSnapshot(false, 0)
	record("node-b", "container-b", 2048, nil)
	assertSnapshot(true, 3072)
	if err := db.Model(&DeploymentLocation{}).Where("id = ?", "b").Update("container_id", "replacement-b").Error; err != nil {
		t.Fatal(err)
	}
	record("node-a", "container-a", 1024, nil)
	assertSnapshot(false, 0)
	record("node-b", "replacement-b", 4096, nil)
	assertSnapshot(true, 5120)
	if err := db.Model(&DeploymentStorageMeasurement{}).Where("node_id = ?", "node-b").Update("measured_at", time.Now().Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	record("node-a", "container-a", 1024, nil)
	assertSnapshot(false, 0)
}

func TestBillingRejectsMissingStaleAndFailedStorage(t *testing.T) {
	db := newDeploymentRepositoryTestDB(t)
	if err := db.AutoMigrate(&GameServer{}, &VPSInstance{}, &DatabaseInstance{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now()
	seedDeployments(t, db, &Deployment{ID: "dep", OrganizationID: "org", StorageBytes: 123}, &Deployment{ID: "deleted", OrganizationID: "org", StorageBytes: 999, DeletedAt: &now}, &Deployment{ID: "other", OrganizationID: "other", StorageBytes: 999})
	if _, err := BillableStorageBytes(ctx, db, "org", now); err == nil {
		t.Fatal("unmeasured storage became billable")
	}
	for _, tc := range []struct {
		at      time.Time
		failure string
		valid   bool
	}{{now.Add(-time.Hour), "", false}, {now, "probe failed", false}, {now.Add(time.Hour), "", false}, {now, "", true}} {
		if err := db.Model(&Deployment{}).Where("id = ?", "dep").Updates(map[string]interface{}{"storage_measured_at": tc.at, "storage_measurement_error": tc.failure}).Error; err != nil {
			t.Fatal(err)
		}
		total, err := BillableStorageBytes(ctx, db, "org", now)
		if (err == nil) != tc.valid {
			t.Fatalf("valid=%v, error=%v", tc.valid, err)
		}
		if tc.valid && total != 123 {
			t.Fatalf("total=%d, want 123", total)
		}
	}
	if err := db.Migrator().DropTable(&GameServer{}); err != nil {
		t.Fatal(err)
	}
	if _, err := BillableStorageBytes(ctx, db, "org", now); err == nil {
		t.Fatal("failed inventory query became a zero bill")
	}
}

func TestGameStorageCannotPublishForAReplacedContainer(t *testing.T) {
	db := newDeploymentRepositoryTestDB(t)
	if err := db.AutoMigrate(&GameServer{}); err != nil {
		t.Fatal(err)
	}
	old, replacement := "old-container", "replacement-container"
	gs := GameServer{ID: "server", OrganizationID: "org", ContainerID: &old}
	if err := db.Create(&gs).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewGameServerRepository(db, nil)
	ctx := context.Background()
	if err := repo.RecordStorage(ctx, gs.ID, old, 2048, nil); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&GameServer{}).Where("id = ?", gs.ID).Update("container_id", replacement).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordStorage(ctx, gs.ID, old, 9999, nil); err == nil {
		t.Fatal("stale inventory published")
	}
	if err := repo.RecordStorage(ctx, gs.ID, replacement, 0, errors.New("probe failed")); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&gs, "id = ?", gs.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gs.StorageBytes != 2048 || StorageMeasurementAvailable(gs.StorageMeasuredAt, gs.StorageMeasurementError, time.Now()) {
		t.Fatal("failed probe destroyed last measurement or appeared current")
	}
	if err := repo.RecordStorage(ctx, gs.ID, replacement, 4096, nil); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&gs, "id = ?", gs.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gs.StorageBytes != 4096 || !StorageMeasurementAvailable(gs.StorageMeasuredAt, gs.StorageMeasurementError, time.Now()) {
		t.Fatal("replacement storage did not recover")
	}
}

func TestNeverMaterializedStorageIsZeroUntilRuntimeCreation(t *testing.T) {
	db := newDeploymentRepositoryTestDB(t)
	if err := db.AutoMigrate(&GameServer{}, &VPSInstance{}, &DatabaseInstance{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	dep := &Deployment{ID: "new-empty", OrganizationID: "org"}
	repo := NewDeploymentRepository(db, nil)
	if err := repo.Create(ctx, dep); err != nil {
		t.Fatal(err)
	}
	if total, err := BillableStorageBytes(ctx, db, "org", time.Now().Add(time.Hour)); err != nil || total != 0 {
		t.Fatalf("unused deployment prevented billing: %d %v", total, err)
	}
	if err := MarkStorageMaterializing(ctx, db, &Deployment{}, dep.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(ctx, dep); err != nil {
		t.Fatal(err)
	}
	if _, err := BillableStorageBytes(ctx, db, "org", time.Now()); err == nil {
		t.Fatal("possibly materialized data was billed as empty")
	}
}

func TestRetainedNodeStorageRemainsRequiredAfterLocationRemoval(t *testing.T) {
	db := newDeploymentRepositoryTestDB(t)
	if err := db.AutoMigrate(&DeploymentLocation{}, &DeploymentStorageMeasurement{}); err != nil {
		t.Fatal(err)
	}
	dep := Deployment{ID: "retained", OrganizationID: "org"}
	seedDeployments(t, db, &dep)
	loc := DeploymentLocation{ID: "location", DeploymentID: dep.ID, NodeID: "node", ContainerID: "container", Status: "running"}
	if err := db.Create(&loc).Error; err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := RecordDeploymentStorage(ctx, db, dep.ID, "node", []string{"container"}, 8192, nil, `[{"kind":"volume","source":"example-volume"}]`); err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&loc).Error; err != nil {
		t.Fatal(err)
	}
	if err := RecordDeploymentStorage(ctx, db, dep.ID, "node", nil, 8192, nil, `[{"kind":"volume","source":"example-volume"}]`); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&dep, "id = ?", dep.ID).Error; err != nil {
		t.Fatal(err)
	}
	if dep.StorageBytes != 8192 || !StorageMeasurementAvailable(dep.StorageMeasuredAt, dep.StorageMeasurementError, time.Now()) {
		t.Fatal("retained node inventory lost when task removed")
	}
}
