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
