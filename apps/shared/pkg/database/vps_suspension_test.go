package database

import (
	"context"
	"errors"
	"testing"

	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestVPSSuspensionSurvivesStaleObservation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := db.DB()
	raw.SetMaxOpenConns(1)
	defer raw.Close()
	if err := db.AutoMigrate(&VPSInstance{}, &VPSSuspension{}, &Organization{}); err != nil {
		t.Fatal(err)
	}
	for _, row := range []interface{}{&Organization{ID: "org-test", Slug: "org-test", Status: "active"}, &VPSInstance{ID: "vps-test", OrganizationID: "org-test", Status: int32(vpsv1.VPSStatus_RUNNING)}} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	if err := EnsureVPSCanStart(ctx, db, "vps-test"); err != nil {
		t.Fatal(err)
	}
	if err := SuspendVPS(ctx, db, "vps-test", "admin-test", ""); err != nil {
		t.Fatal(err)
	}
	for _, state := range []vpsv1.VPSStatus{vpsv1.VPSStatus_RUNNING, vpsv1.VPSStatus_STOPPED} {
		if err := UpdateVPSObservedStatus(ctx, db, "vps-test", int32(state)); err != nil {
			t.Fatal(err)
		}
		var vps VPSInstance
		if err := db.First(&vps, "id = ?", "vps-test").Error; err != nil {
			t.Fatal(err)
		}
		if vps.Status != int32(vpsv1.VPSStatus_SUSPENDED) {
			t.Fatalf("observation cleared suspension: %d", vps.Status)
		}
		if err := EnsureVPSCanStart(ctx, db, "vps-test"); !errors.Is(err, ErrVPSSuspended) {
			t.Fatalf("suspended VPS allowed: %v", err)
		}
	}
	// Even a legacy writer replacing status cannot erase the separate restriction.
	if err := db.Model(&VPSInstance{}).Where("id = ?", "vps-test").Update("status", int32(vpsv1.VPSStatus_RUNNING)).Error; err != nil {
		t.Fatal(err)
	}
	if err := EnsureVPSCanStart(ctx, db, "vps-test"); !errors.Is(err, ErrVPSSuspended) {
		t.Fatalf("durable restriction ignored: %v", err)
	}
	if err := PreserveLegacyVPSSuspension(ctx, db, "vps-test"); err != nil {
		t.Fatal(err)
	}
	var record VPSSuspension
	if err := db.First(&record).Error; err != nil {
		t.Fatal(err)
	}
	if record.SuspendedBy != "admin-test" {
		t.Fatal("reconciliation replaced administrator attribution")
	}
	raw.Close()
	if err := EnsureVPSCanStart(ctx, db, "vps-test"); err == nil {
		t.Fatal("unavailable database allowed start")
	}
}

func TestLegacySuspensionReconcileDoesNotUndoUnsuspend(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err := db.AutoMigrate(&VPSInstance{}, &VPSSuspension{}, &Organization{}); err != nil {
		t.Fatal(err)
	}
	for _, row := range []interface{}{&Organization{ID: "review-org", Slug: "review-org", Status: "active"}, &VPSInstance{ID: "review-vps", OrganizationID: "review-org", Status: int32(vpsv1.VPSStatus_SUSPENDED), Metadata: `{"suspended_at":"previous","custom":7}`}} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	// Sync has already loaded the suspended snapshot and is awaiting Proxmox.
	var stale VPSInstance
	if err := db.First(&stale, "id = ?", "review-vps").Error; err != nil {
		t.Fatal(err)
	}
	updated, err := UnsuspendVPS(t.Context(), db, stale.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Metadata != `{"custom":7}` {
		t.Fatalf("metadata changed: %s", updated.Metadata)
	}
	stopped := false
	suspended, err := ReconcileVPSSuspension(t.Context(), db, stale.ID, func() error { stopped = true; return nil })
	if err != nil || suspended || stopped {
		t.Fatalf("stale reconciliation restored suspension: %v %v %v", suspended, stopped, err)
	}
	if err := EnsureVPSCanStart(t.Context(), db, stale.ID); err != nil {
		t.Fatal(err)
	}
}
