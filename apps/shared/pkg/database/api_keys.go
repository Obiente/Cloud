package database

import (
	"time"

	"gorm.io/gorm"
)

// APIKey represents an API key for service authentication
type APIKey struct {
	ID             string         `gorm:"type:text;primaryKey" json:"id"`
	Name           string         `gorm:"type:text;not null" json:"name"`
	KeyHash        string         `gorm:"type:text;unique;not null;index" json:"-"` // Hashed API key
	UserID         string         `gorm:"type:text;not null;index" json:"user_id"`
	OrganizationID string         `gorm:"type:text;not null;index" json:"organization_id"`
	Organization   *Organization  `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Scopes         string         `gorm:"type:text;not null" json:"scopes"` // Comma-separated scopes
	LastUsedAt     *time.Time     `gorm:"type:timestamptz" json:"last_used_at,omitempty"`
	ExpiresAt      *time.Time     `gorm:"type:timestamptz" json:"expires_at,omitempty"`
	RevokedAt      *time.Time     `gorm:"type:timestamptz" json:"revoked_at,omitempty"`
	CreatedAt      time.Time      `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"type:timestamptz;not null;default:now()" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// TableName specifies the table name for APIKey
func (APIKey) TableName() string {
	return "api_keys"
}
