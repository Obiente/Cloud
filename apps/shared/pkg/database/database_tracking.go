package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DatabaseLocation tracks where managed database containers are running across the cluster
type DatabaseLocation struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	DatabaseID   string    `gorm:"index;not null" json:"database_id"`
	NodeID       string    `gorm:"index;not null" json:"node_id"`
	NodeHostname string    `json:"node_hostname"`
	NodeIP       string    `json:"node_ip"`
	ContainerID  string    `gorm:"uniqueIndex" json:"container_id"`
	Status       string    `gorm:"index;not null" json:"status"` // running, stopped, failed, etc.
	Port         int32     `json:"port"`
	CPUUsage     float64   `json:"cpu_usage"`
	MemoryUsage  int64     `json:"memory_usage"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (DatabaseLocation) TableName() string { return "database_locations" }

// DatabaseUptimeInterval records one continuous period in which a managed
// database was running. Separate intervals prevent stopped time from being
// counted after a database starts again.
type DatabaseUptimeInterval struct {
	ID          string     `gorm:"primaryKey" json:"id"`
	DatabaseID  string     `gorm:"index;not null" json:"database_id"`
	ContainerID string     `gorm:"index;not null" json:"container_id"`
	NodeID      string     `gorm:"index;not null" json:"node_id"`
	StartedAt   time.Time  `gorm:"index;not null" json:"started_at"`
	EndedAt     *time.Time `gorm:"index" json:"ended_at"`
}

func (DatabaseUptimeInterval) TableName() string { return "database_uptime_intervals" }

// DatabaseLocationID returns a stable primary key for a database container's
// location record, allowing startup reconciliation to safely upsert it.
func DatabaseLocationID(databaseID, containerID string) string {
	return fmt.Sprintf("%s:%s", databaseID, containerID)
}

// GetDatabaseLocations returns all locations where a database is running
func GetDatabaseLocations(databaseID string) ([]DatabaseLocation, error) {
	var locations []DatabaseLocation
	result := DB.Where("database_id = ? AND status = ?", databaseID, "running").Find(&locations)
	return locations, result.Error
}

// GetAllDatabaseLocations returns all locations for a database regardless of status
func GetAllDatabaseLocations(databaseID string) ([]DatabaseLocation, error) {
	var locations []DatabaseLocation
	result := DB.Where("database_id = ?", databaseID).Find(&locations)
	return locations, result.Error
}

// UpsertDatabaseLocation creates or updates a database location
func UpsertDatabaseLocation(location *DatabaseLocation) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		var existing DatabaseLocation
		if err := tx.Where("container_id = ?", location.ContainerID).First(&existing).Error; err == nil {
			location.ID = existing.ID
			location.CreatedAt = existing.CreatedAt
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Save(location).Error; err != nil {
			return err
		}
		if err := deactivateOtherDatabaseLocations(tx, location.DatabaseID, location.ContainerID, now); err != nil {
			return err
		}
		if location.Status == "running" {
			return ensureDatabaseUptimeInterval(tx, location, now)
		}
		return closeDatabaseUptimeIntervals(tx, location.DatabaseID, location.ContainerID, now)
	})
}

// DeleteDatabaseLocation removes a database location
func DeleteDatabaseLocation(containerID string) error {
	return DB.Where("container_id = ?", containerID).Delete(&DatabaseLocation{}).Error
}

// UpdateDatabaseLocationStatus keeps location-based metrics aligned with the
// managed database lifecycle.
func UpdateDatabaseLocationStatus(ctx context.Context, databaseID, status string) error {
	now := time.Now().UTC()
	return DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		location, err := currentDatabaseLocation(tx, databaseID)
		if err != nil {
			return err
		}
		if location == nil {
			return nil
		}
		if err := deactivateOtherDatabaseLocations(tx, databaseID, location.ContainerID, now); err != nil {
			return err
		}
		if err := tx.Model(&DatabaseLocation{}).
			Where("id = ?", location.ID).
			Updates(map[string]interface{}{"status": status, "updated_at": now}).Error; err != nil {
			return err
		}
		if status != "running" {
			return closeDatabaseUptimeIntervals(tx, databaseID, location.ContainerID, now)
		}
		location.Status = status
		return ensureDatabaseUptimeInterval(tx, location, now)
	})
}

// EnsureDatabaseUptimeInterval backfills an open interval for a running
// location discovered during startup reconciliation.
func EnsureDatabaseUptimeInterval(ctx context.Context, databaseID string) error {
	now := time.Now().UTC()
	return DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		location, err := currentDatabaseLocation(tx, databaseID)
		if err != nil {
			return err
		}
		if location == nil {
			return nil
		}
		if err := deactivateOtherDatabaseLocations(tx, databaseID, location.ContainerID, now); err != nil {
			return err
		}
		if location.Status != "running" {
			return closeDatabaseUptimeIntervals(tx, databaseID, location.ContainerID, now)
		}
		return ensureDatabaseUptimeInterval(tx, location, now)
	})
}

// BackfillDatabaseUptimeIntervals preserves the best interval available from
// legacy location timestamps before usage reads switch to the interval table.
// A legacy row cannot describe earlier stop/start cycles, so the migration
// retains its previous continuous-lifetime interpretation without inventing
// more precise history.
func BackfillDatabaseUptimeIntervals() error {
	now := time.Now().UTC()
	return DB.Transaction(func(tx *gorm.DB) error {
		var locations []DatabaseLocation
		if err := tx.Find(&locations).Error; err != nil {
			return err
		}
		for i := range locations {
			var count int64
			if err := tx.Model(&DatabaseUptimeInterval{}).
				Where("database_id = ? AND container_id = ?", locations[i].DatabaseID, locations[i].ContainerID).
				Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				continue
			}

			startedAt := locations[i].CreatedAt.UTC()
			if startedAt.IsZero() {
				startedAt = locations[i].UpdatedAt.UTC()
			}
			if startedAt.IsZero() {
				startedAt = now
			}
			interval := &DatabaseUptimeInterval{
				ID:          uuid.NewString(),
				DatabaseID:  locations[i].DatabaseID,
				ContainerID: locations[i].ContainerID,
				NodeID:      locations[i].NodeID,
				StartedAt:   startedAt,
			}
			if locations[i].Status != "running" {
				endedAt := locations[i].UpdatedAt.UTC()
				if endedAt.IsZero() || endedAt.Before(startedAt) {
					endedAt = startedAt
				}
				interval.EndedAt = &endedAt
			}
			if err := tx.Create(interval).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func currentDatabaseLocation(tx *gorm.DB, databaseID string) (*DatabaseLocation, error) {
	var instance DatabaseInstance
	if err := tx.Select("instance_id").First(&instance, "id = ?", databaseID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	query := tx.Where("database_id = ?", databaseID)
	if instance.InstanceID != nil && *instance.InstanceID != "" {
		query = query.Where("container_id = ?", *instance.InstanceID)
	} else {
		query = query.Order("updated_at DESC")
	}
	var location DatabaseLocation
	if err := query.First(&location).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &location, nil
}

func deactivateOtherDatabaseLocations(tx *gorm.DB, databaseID, currentContainerID string, now time.Time) error {
	activeStatuses := []string{"running", "restarting", "starting", "created"}
	if err := tx.Model(&DatabaseLocation{}).
		Where("database_id = ? AND container_id <> ? AND status IN ?", databaseID, currentContainerID, activeStatuses).
		Updates(map[string]interface{}{"status": "stopped", "updated_at": now}).Error; err != nil {
		return err
	}
	return tx.Model(&DatabaseUptimeInterval{}).
		Where("database_id = ? AND container_id <> ? AND ended_at IS NULL", databaseID, currentContainerID).
		Update("ended_at", now).Error
}

func ensureDatabaseUptimeInterval(tx *gorm.DB, location *DatabaseLocation, now time.Time) error {
	var count int64
	if err := tx.Model(&DatabaseUptimeInterval{}).
		Where("database_id = ? AND container_id = ? AND ended_at IS NULL", location.DatabaseID, location.ContainerID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return tx.Create(&DatabaseUptimeInterval{
		ID:          uuid.NewString(),
		DatabaseID:  location.DatabaseID,
		ContainerID: location.ContainerID,
		NodeID:      location.NodeID,
		StartedAt:   now,
	}).Error
}

func closeDatabaseUptimeIntervals(tx *gorm.DB, databaseID, containerID string, now time.Time) error {
	return tx.Model(&DatabaseUptimeInterval{}).
		Where("database_id = ? AND container_id = ? AND ended_at IS NULL", databaseID, containerID).
		Update("ended_at", now).Error
}

// RecordDatabaseMetrics records database metrics
func RecordDatabaseMetrics(ctx context.Context, metrics *DatabaseMetrics) error {
	targetDB := MetricsDB
	if targetDB == nil {
		targetDB = DB
	}
	return targetDB.WithContext(ctx).Create(metrics).Error
}

// GetRecentDatabaseMetrics gets recent metrics for a database
func GetRecentDatabaseMetrics(ctx context.Context, databaseID string, since time.Time) ([]DatabaseMetrics, error) {
	var metrics []DatabaseMetrics
	targetDB := MetricsDB
	if targetDB == nil {
		targetDB = DB
	}
	result := targetDB.WithContext(ctx).Where("database_id = ? AND timestamp >= ?", databaseID, since).
		Order("timestamp DESC").
		Limit(1000).
		Find(&metrics)
	return metrics, result.Error
}

// CleanOldDatabaseMetrics removes metrics older than retention period
func CleanOldDatabaseMetrics(ctx context.Context, retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	targetDB := MetricsDB
	if targetDB == nil {
		targetDB = DB
	}
	return targetDB.WithContext(ctx).Where("timestamp < ?", cutoff).Delete(&DatabaseMetrics{}).Error
}
