package database

import (
	"time"

	"gorm.io/gorm"
)

// Deployment represents a deployment in the database
type Deployment struct {
	ID             string    `gorm:"primaryKey;column:id" json:"id"`
	Name           string    `gorm:"column:name" json:"name"`
	Domain         string    `gorm:"column:domain" json:"domain"`
	CustomDomains  string    `gorm:"column:custom_domains;type:jsonb" json:"custom_domains"` // Stored as JSON array
	Type           int32     `gorm:"column:type" json:"type"`                                // DeploymentType enum
	RepositoryURL  *string   `gorm:"column:repository_url" json:"repository_url"`
	Branch         string    `gorm:"column:branch" json:"branch"`
	BuildCommand   *string   `gorm:"column:build_command" json:"build_command"`
	InstallCommand *string   `gorm:"column:install_command" json:"install_command"`
	Status         int32     `gorm:"column:status;default:0" json:"status"` // DeploymentStatus enum
	HealthStatus   string    `gorm:"column:health_status" json:"health_status"`
	Environment    int32     `gorm:"column:environment" json:"environment"` // Environment enum
	BandwidthUsage int64     `gorm:"column:bandwidth_usage;default:0" json:"bandwidth_usage"`
	StorageUsage   int64     `gorm:"column:storage_usage;default:0" json:"storage_usage"`
	BuildTime      int32     `gorm:"column:build_time;default:0" json:"build_time"`
	Size           string    `gorm:"column:size" json:"size"`
	LastDeployedAt time.Time `gorm:"column:last_deployed_at" json:"last_deployed_at"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	OrganizationID string    `gorm:"column:organization_id;index" json:"organization_id"`
	CreatedBy      string    `gorm:"column:created_by;index" json:"created_by"`
}

func (Deployment) TableName() string {
	return "deployments"
}

// BeforeCreate hook to set timestamps
func (d *Deployment) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	if d.LastDeployedAt.IsZero() {
		d.LastDeployedAt = now
	}
	return nil
}

// BeforeUpdate hook to set updated timestamp
func (d *Deployment) BeforeUpdate(tx *gorm.DB) error {
	d.LastDeployedAt = time.Now()
	return nil
}
