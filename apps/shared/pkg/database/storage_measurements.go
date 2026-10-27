package database

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const StorageMeasurementMaxAge = 15 * time.Minute

// DeploymentStorageMeasurement is a node-local snapshot. Its inventory hash
// prevents a measurement of an old replica set from becoming a current total.
type DeploymentStorageMeasurement struct {
	DeploymentID string `gorm:"primaryKey"`
	NodeID       string `gorm:"primaryKey"`
	Mounts       string
	Inventory    string
	StorageBytes int64
	MeasuredAt   time.Time
	Error        string
}

func StorageMeasurementAvailable(at *time.Time, failure string, now time.Time) bool {
	return at != nil && failure == "" && !at.After(now.Add(time.Minute)) && now.Sub(*at) <= StorageMeasurementMaxAge
}

func storageInventory(ids []string) string {
	unique := map[string]bool{}
	for _, id := range ids {
		if id != "" {
			unique[id] = true
		}
	}
	values := make([]string, 0, len(unique))
	for id := range unique {
		values = append(values, id)
	}
	sort.Strings(values)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(values, "\n"))))
}

// RecordDeploymentStorage serializes updates on the deployment row. Each node
// contributes only its own containers; no incomplete total is marked measured.
func RecordDeploymentStorage(ctx context.Context, db *gorm.DB, id, node string, ids []string, bytes int64, measurementErr error, mounts ...string) error {
	captured := len(mounts) > 0 && mounts[0] != ""
	if node == "" || (len(ids) == 0 && !captured && measurementErr == nil) || bytes < 0 {
		return fmt.Errorf("invalid storage inventory")
	}
	now := time.Now()
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var dep Deployment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", id).First(&dep).Error; err != nil {
			return err
		}
		m := DeploymentStorageMeasurement{DeploymentID: id, NodeID: node, Inventory: storageInventory(ids), StorageBytes: bytes, MeasuredAt: now}
		if captured {
			m.Mounts = mounts[0]
		} else {
			var prior DeploymentStorageMeasurement
			if err := tx.Where("deployment_id = ? AND node_id = ?", id, node).First(&prior).Error; err != nil && err != gorm.ErrRecordNotFound {
				return err
			}
			m.Mounts = prior.Mounts
			if measurementErr != nil {
				m.Inventory = prior.Inventory
			}
		}
		if measurementErr != nil {
			m.Error = "Storage measurement unavailable"
		}
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&m).Error; err != nil {
			return err
		}
		var locations []DeploymentLocation
		if err := tx.Where("deployment_id = ? AND status IN ?", id, []string{"running", "stopped"}).Find(&locations).Error; err != nil {
			return err
		}
		inventory := map[string][]string{}
		for _, l := range locations {
			inventory[l.NodeID] = append(inventory[l.NodeID], l.ContainerID)
		}
		var snapshots []DeploymentStorageMeasurement
		if err := tx.Where("deployment_id = ?", id).Find(&snapshots).Error; err != nil {
			return err
		}
		byNode := map[string]DeploymentStorageMeasurement{}
		for _, s := range snapshots {
			byNode[s.NodeID] = s
			if s.Mounts != "" {
				if _, ok := inventory[s.NodeID]; !ok {
					inventory[s.NodeID] = nil
				}
			}
		}
		total, oldest := int64(0), now
		complete := len(inventory) > 0
		for owner, containers := range inventory {
			s, ok := byNode[owner]
			if !ok || owner == "" || s.Inventory != storageInventory(containers) || !StorageMeasurementAvailable(&s.MeasuredAt, s.Error, now) || s.StorageBytes < 0 || total > math.MaxInt64-s.StorageBytes {
				complete = false
				break
			}
			for _, container := range containers {
				if container == "" {
					complete = false
				}
			}
			total += s.StorageBytes
			if s.MeasuredAt.Before(oldest) {
				oldest = s.MeasuredAt
			}
		}
		updates := map[string]interface{}{"storage_measurement_error": "Storage is awaiting measurements from all owning nodes"}
		if complete {
			updates = map[string]interface{}{"storage_state": "measured", "storage_bytes": total, "storage_measured_at": oldest, "storage_measurement_error": ""}
		}
		return tx.Model(&Deployment{}).Where("id = ?", id).Updates(updates).Error
	})
	if err == nil && RedisClient != nil {
		RedisClient.Delete(ctx, fmt.Sprintf("deployment:%s", id))
	}
	return err
}

// BillableStorageBytes refuses to invoice uncertain values. Provisioned VPS and
// database capacity is billed separately from measured deployment/game storage.
func BillableStorageBytes(ctx context.Context, db *gorm.DB, org string, now time.Time) (int64, error) {
	total := int64(0)
	for _, table := range []string{"deployments", "game_servers"} {
		var rows []struct {
			StorageState            string
			StorageBytes            int64
			StorageMeasuredAt       *time.Time
			StorageMeasurementError string
		}
		if err := db.WithContext(ctx).Table(table).Select("storage_state, storage_bytes, storage_measured_at, storage_measurement_error").Where("organization_id = ? AND deleted_at IS NULL", org).Scan(&rows).Error; err != nil {
			return 0, err
		}
		for _, r := range rows {
			if r.StorageState == "empty" && r.StorageBytes == 0 {
				continue
			}
			if !StorageMeasurementAvailable(r.StorageMeasuredAt, r.StorageMeasurementError, now) || r.StorageBytes < 0 || total > math.MaxInt64-r.StorageBytes {
				return 0, fmt.Errorf("storage measurements are incomplete; billing will retry after measurement recovers")
			}
			total += r.StorageBytes
		}
	}
	for _, table := range []string{"vps_instances", "database_instances"} {
		var capacities []int64
		if err := db.WithContext(ctx).Table(table).Where("organization_id = ? AND deleted_at IS NULL", org).Pluck("disk_bytes", &capacities).Error; err != nil {
			return 0, err
		}
		for _, size := range capacities {
			if size < 0 || total > math.MaxInt64-size {
				return 0, fmt.Errorf("invalid provisioned storage capacity")
			}
			total += size
		}
	}
	return total, nil
}

// Newly created records are provably empty until a runtime operation is accepted.
// Legacy records keep an unknown state and must be measured before invoicing.
func ResourceStorageAvailable(state string, bytes int64, at *time.Time, failure string, now time.Time) bool {
	return (state == "empty" && bytes == 0) || StorageMeasurementAvailable(at, failure, now)
}

func MarkStorageMaterializing(ctx context.Context, db *gorm.DB, model interface{}, id string) error {
	result := db.WithContext(ctx).Model(model).Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{"storage_state": "unknown", "storage_measurement_error": "Storage measurement pending"})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	if RedisClient != nil {
		RedisClient.Delete(ctx, "deployment:"+id)
		RedisClient.Delete(ctx, "gameserver:"+id)
	}
	return nil
}

func DeploymentStorageInventoryMatches(snapshot DeploymentStorageMeasurement, ids []string) bool {
	return snapshot.Mounts != "" && (len(ids) == 0 || snapshot.Inventory == storageInventory(ids))
}
