package models

import (
	"time"

	"gorm.io/gorm"
)

// DatabaseInstance represents a managed database instance
type DatabaseInstance struct {
	ID             string     `gorm:"primaryKey;column:id" json:"id"`
	Name           string     `gorm:"column:name;not null" json:"name"`
	Description    *string    `gorm:"column:description" json:"description"`
	Status         int32      `gorm:"column:status;default:0;not null" json:"status"` // DatabaseStatus enum
	Type           int32      `gorm:"column:type;not null" json:"type"`               // DatabaseType enum
	Version        *string    `gorm:"column:version" json:"version"`                  // Database version (e.g., "15", "8.0")
	Size           string     `gorm:"column:size;not null" json:"size"`               // Database size/spec
	CPUCores       int32      `gorm:"column:cpu_cores;not null" json:"cpu_cores"`
	MemoryBytes    int64      `gorm:"column:memory_bytes;not null" json:"memory_bytes"`
	DiskBytes      int64      `gorm:"column:disk_bytes;not null" json:"disk_bytes"`
	DiskUsedBytes  int64      `gorm:"column:disk_used_bytes;default:0" json:"disk_used_bytes"`
	MaxConnections int64      `gorm:"column:max_connections;not null" json:"max_connections"`
	Host           *string    `gorm:"column:host" json:"host"`
	Port           *int32     `gorm:"column:port" json:"port"`
	InstanceID     *string    `gorm:"column:instance_id" json:"instance_id"` // Docker container ID
	NodeID         *string    `gorm:"column:node_id" json:"node_id"`         // Docker Swarm node ID
	Metadata       string     `gorm:"column:metadata;type:jsonb" json:"metadata"`
	CreatedAt      time.Time  `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;not null" json:"updated_at"`
	LastStartedAt  *time.Time `gorm:"column:last_started_at" json:"last_started_at"`
	DeletedAt      *time.Time `gorm:"column:deleted_at;index" json:"deleted_at"` // Soft delete
	OrganizationID string     `gorm:"column:organization_id;index;not null" json:"organization_id"`
	CreatedBy      string     `gorm:"column:created_by;index;not null" json:"created_by"`
}

func (DatabaseInstance) TableName() string {
	return "database_instances"
}

// BeforeCreate hook to set timestamps
func (d *DatabaseInstance) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate hook to set updated timestamp
func (d *DatabaseInstance) BeforeUpdate(tx *gorm.DB) error {
	d.UpdatedAt = time.Now()
	return nil
}

// DatabaseBackup represents a backup of a database instance
type DatabaseBackup struct {
	ID             string     `gorm:"primaryKey;column:id" json:"id"`
	DatabaseID     string     `gorm:"column:database_id;index;not null" json:"database_id"`
	Name           string     `gorm:"column:name;not null" json:"name"`
	Description    *string    `gorm:"column:description" json:"description"`
	SizeBytes      int64      `gorm:"column:size_bytes;default:0" json:"size_bytes"`
	Status         int32      `gorm:"column:status;default:0;not null" json:"status"` // DatabaseBackupStatus enum
	CreatedAt      time.Time  `gorm:"column:created_at;not null" json:"created_at"`
	CompletedAt    *time.Time `gorm:"column:completed_at" json:"completed_at"`
	ErrorMessage   *string    `gorm:"column:error_message;type:text" json:"error_message"`
	BackupPath     *string    `gorm:"column:backup_path" json:"backup_path"`
	OrganizationID string     `gorm:"column:organization_id;index;not null" json:"organization_id"`
	CreatedBy      string     `gorm:"column:created_by;index;not null" json:"created_by"`
}

func (DatabaseBackup) TableName() string {
	return "database_backups"
}

// BeforeCreate hook to set timestamps
func (db *DatabaseBackup) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if db.CreatedAt.IsZero() {
		db.CreatedAt = now
	}
	return nil
}

// DatabaseConnection represents connection credentials for a database
// Passwords are encrypted at rest using AES-256-GCM
type DatabaseConnection struct {
	ID             string    `gorm:"primaryKey;column:id" json:"id"`
	DatabaseID     string    `gorm:"column:database_id;index;not null;uniqueIndex" json:"database_id"`
	DatabaseName   string    `gorm:"column:database_name;not null" json:"database_name"`
	Username       string    `gorm:"column:username;not null" json:"username"`
	Password       string    `gorm:"column:password;not null" json:"-"` // Encrypted password, never in JSON
	Host           string    `gorm:"column:host;not null" json:"host"`
	Port           int32     `gorm:"column:port;not null" json:"port"`
	SSLRequired    bool      `gorm:"column:ssl_required;default:true" json:"ssl_required"`
	SSLCertificate *string   `gorm:"column:ssl_certificate;type:text" json:"ssl_certificate"`
	CreatedAt      time.Time `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null" json:"updated_at"`
}

func (DatabaseConnection) TableName() string {
	return "database_connections"
}

// BeforeCreate hook to set timestamps
func (dc *DatabaseConnection) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if dc.CreatedAt.IsZero() {
		dc.CreatedAt = now
	}
	if dc.UpdatedAt.IsZero() {
		dc.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate hook to set updated timestamp
func (dc *DatabaseConnection) BeforeUpdate(tx *gorm.DB) error {
	dc.UpdatedAt = time.Now()
	return nil
}
