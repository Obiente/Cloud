package database

import (
	"context"
	"github.com/google/uuid"
	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"testing"
	"time"
)

func TestVPSSuspensionReconciliationFencesConcurrentUnsuspend(t *testing.T) {
	dsn := os.Getenv("OBIENTE_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("requires isolated PostgreSQL")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err := db.AutoMigrate(&VPSInstance{}, &VPSSuspension{}); err != nil {
		t.Fatal(err)
	}
	id := uuid.NewString()
	if err := db.Create(&VPSInstance{ID: id, IPv4Addresses: "[]", IPv6Addresses: "[]", Metadata: "{}", Status: int32(vpsv1.VPSStatus_SUSPENDED)}).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Where("vps_id = ?", id).Delete(&VPSSuspension{})
	defer db.Where("id = ?", id).Delete(&VPSInstance{})
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		_, err := ReconcileVPSSuspension(ctx, db, id, func() error {
			close(entered)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		done <- err
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	// Unsuspend must wait for the in-flight stop, rather than deleting the
	// restriction before reconciliation has finished using it.
	short, stop := context.WithTimeout(ctx, 100*time.Millisecond)
	_, err = UnsuspendVPS(short, db, id)
	stop()
	close(release)
	if err == nil {
		t.Error("unsuspend raced through the active stop")
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if _, err := UnsuspendVPS(ctx, db, id); err != nil {
		t.Fatal(err)
	}
	if suspended, err := ReconcileVPSSuspension(ctx, db, id, func() error { t.Error("stopped after unsuspend"); return nil }); err != nil || suspended {
		t.Fatalf("suspension resurrected: %v %v", suspended, err)
	}
}
