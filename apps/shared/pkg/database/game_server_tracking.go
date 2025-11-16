package database

import (
	"context"
	"time"
)

// GameServerLocation tracks where game servers are running across the cluster
type GameServerLocation struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	GameServerID string    `gorm:"index;not null" json:"game_server_id"`
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

func (GameServerLocation) TableName() string { return "game_server_locations" }

// GetGameServerLocations returns all locations where a game server is running
func GetGameServerLocations(gameServerID string) ([]GameServerLocation, error) {
	var locations []GameServerLocation
	result := DB.Where("game_server_id = ? AND status = ?", gameServerID, "running").Find(&locations)
	return locations, result.Error
}

// GetAllGameServerLocations returns all locations for a game server regardless of status
func GetAllGameServerLocations(gameServerID string) ([]GameServerLocation, error) {
	var locations []GameServerLocation
	result := DB.Where("game_server_id = ?", gameServerID).Find(&locations)
	return locations, result.Error
}

// UpsertGameServerLocation creates or updates a game server location
func UpsertGameServerLocation(location *GameServerLocation) error {
	return DB.Save(location).Error
}

// DeleteGameServerLocation removes a game server location
func DeleteGameServerLocation(containerID string) error {
	return DB.Where("container_id = ?", containerID).Delete(&GameServerLocation{}).Error
}

// RecordGameServerMetrics records game server metrics
func RecordGameServerMetrics(ctx context.Context, metrics *GameServerMetrics) error {
	targetDB := MetricsDB
	if targetDB == nil {
		targetDB = DB
	}
	return targetDB.WithContext(ctx).Create(metrics).Error
}

// GetRecentGameServerMetrics gets recent metrics for a game server
func GetRecentGameServerMetrics(ctx context.Context, gameServerID string, since time.Time) ([]GameServerMetrics, error) {
	var metrics []GameServerMetrics
	targetDB := MetricsDB
	if targetDB == nil {
		targetDB = DB
	}
	result := targetDB.WithContext(ctx).Where("game_server_id = ? AND timestamp >= ?", gameServerID, since).
		Order("timestamp DESC").
		Limit(1000).
		Find(&metrics)
	return metrics, result.Error
}

// CleanOldGameServerMetrics removes metrics older than retention period
func CleanOldGameServerMetrics(ctx context.Context, retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	targetDB := MetricsDB
	if targetDB == nil {
		targetDB = DB
	}
	return targetDB.WithContext(ctx).Where("timestamp < ?", cutoff).Delete(&GameServerMetrics{}).Error
}
