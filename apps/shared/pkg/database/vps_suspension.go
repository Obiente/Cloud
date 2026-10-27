package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// VPSSuspension is desired administrative state, separate from observed VM status.
type VPSSuspension struct {
	VPSID       string `gorm:"primaryKey;column:vps_id"`
	Reason      string
	SuspendedBy string
	SuspendedAt time.Time
}

var ErrVPSSuspended = errors.New("VPS is suspended")

func VPSSuspended(ctx context.Context, db *gorm.DB, vps *VPSInstance) (bool, error) {
	if db == nil || vps == nil {
		return false, fmt.Errorf("cannot verify VPS suspension")
	}
	var count int64
	if err := db.WithContext(ctx).Model(&VPSSuspension{}).Where("vps_id = ?", vps.ID).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 || vps.Status == int32(vpsv1.VPSStatus_SUSPENDED) {
		return true, nil
	}
	var metadata map[string]interface{}
	_ = json.Unmarshal([]byte(vps.Metadata), &metadata)
	return metadata["suspended_at"] != nil, nil
}

func EnsureVPSCanStart(ctx context.Context, db *gorm.DB, id string) error {
	if db == nil {
		return fmt.Errorf("cannot verify VPS access")
	}
	var vps VPSInstance
	if err := db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&vps).Error; err != nil {
		return err
	}
	suspended, err := VPSSuspended(ctx, db, &vps)
	if err != nil {
		return err
	}
	if suspended {
		return ErrVPSSuspended
	}
	return EnsureOrganizationActive(ctx, db, vps.OrganizationID)
}

func SuspendVPS(ctx context.Context, db *gorm.DB, id, actor, reason string) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var vps VPSInstance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", id).First(&vps).Error; err != nil {
			return err
		}
		record := VPSSuspension{VPSID: id, Reason: reason, SuspendedBy: actor, SuspendedAt: time.Now()}
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&record).Error; err != nil {
			return err
		}
		return tx.Model(&VPSInstance{}).Where("id = ?", id).Updates(map[string]interface{}{"status": int32(vpsv1.VPSStatus_SUSPENDED), "updated_at": time.Now()}).Error
	})
}

func PreserveLegacyVPSSuspension(ctx context.Context, db *gorm.DB, id string) error {
	_, err := ReconcileVPSSuspension(ctx, db, id, nil)
	return err
}

// UpdateVPSObservedStatus never lets a delayed hypervisor response remove a suspension.
func UpdateVPSObservedStatus(ctx context.Context, db *gorm.DB, id string, status int32) error {
	return db.WithContext(ctx).Model(&VPSInstance{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     gorm.Expr("CASE WHEN status = ? OR EXISTS (SELECT 1 FROM vps_suspensions WHERE vps_id = ?) THEN ? ELSE ? END", int32(vpsv1.VPSStatus_SUSPENDED), id, int32(vpsv1.VPSStatus_SUSPENDED), status),
		"updated_at": time.Now(),
	}).Error
}

// ReconcileVPSSuspension rereads administrative state while holding the same row
// lock as suspend/unsuspend. The optional stop operation is fenced by that lock.
func ReconcileVPSSuspension(ctx context.Context, db *gorm.DB, id string, stop func() error) (bool, error) {
	suspended := false
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current VPSInstance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", id).First(&current).Error; err != nil {
			return err
		}
		var err error
		suspended, err = VPSSuspended(ctx, tx, &current)
		if err != nil || !suspended {
			return err
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&VPSSuspension{VPSID: id, SuspendedBy: "system", SuspendedAt: time.Now()}).Error; err != nil {
			return err
		}
		if stop != nil {
			return stop()
		}
		return nil
	})
	return suspended, err
}

func UnsuspendVPS(ctx context.Context, db *gorm.DB, id string) (*VPSInstance, error) {
	var current VPSInstance
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", id).First(&current).Error; err != nil {
			return err
		}
		if current.Metadata != "" {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(current.Metadata), &metadata); err != nil {
				return fmt.Errorf("read VPS metadata: %w", err)
			}
			delete(metadata, "suspended_at")
			delete(metadata, "suspended_reason")
			encoded, err := json.Marshal(metadata)
			if err != nil {
				return err
			}
			current.Metadata = string(encoded)
		}
		if err := tx.Where("vps_id = ?", id).Delete(&VPSSuspension{}).Error; err != nil {
			return err
		}
		current.Status = int32(vpsv1.VPSStatus_STOPPED)
		current.UpdatedAt = time.Now()
		return tx.Model(&current).Updates(map[string]interface{}{"status": current.Status, "metadata": current.Metadata, "updated_at": current.UpdatedAt}).Error
	})
	return &current, err
}
