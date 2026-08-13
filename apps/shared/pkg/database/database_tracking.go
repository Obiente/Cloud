package database

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	var existing DatabaseLocation
	if err := DB.Where("container_id = ?", location.ContainerID).First(&existing).Error; err == nil {
		location.ID = existing.ID
		location.CreatedAt = existing.CreatedAt
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return DB.Save(location).Error
}

// DeleteDatabaseLocation removes a database location
func DeleteDatabaseLocation(containerID string) error {
	return DB.Where("container_id = ?", containerID).Delete(&DatabaseLocation{}).Error
}

// UpdateDatabaseLocationStatus keeps location-based metrics aligned with the
// managed database lifecycle.
func UpdateDatabaseLocationStatus(ctx context.Context, databaseID, status string) error {
	return DB.WithContext(ctx).Model(&DatabaseLocation{}).
		Where("database_id = ?", databaseID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
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
