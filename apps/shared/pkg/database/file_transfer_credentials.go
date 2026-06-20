package database

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	FileTransferResourceGameServer = "gameserver"

	FileTransferScopeRead  = "read"
	FileTransferScopeWrite = "write"
)

// FileTransferCredential stores credentials for out-of-band file transfer protocols.
type FileTransferCredential struct {
	ID             string         `gorm:"type:text;primaryKey" json:"id"`
	Name           string         `gorm:"type:text;not null" json:"name"`
	KeyHash        string         `gorm:"type:text;uniqueIndex;not null" json:"-"`
	UserID         string         `gorm:"type:text;not null;index" json:"user_id"`
	OrganizationID string         `gorm:"type:text;not null;index" json:"organization_id"`
	ResourceType   string         `gorm:"type:text;not null;index:idx_file_transfer_resource" json:"resource_type"`
	ResourceID     string         `gorm:"type:text;not null;index:idx_file_transfer_resource" json:"resource_id"`
	Scopes         string         `gorm:"type:text;not null" json:"scopes"`
	LastUsedAt     *time.Time     `gorm:"type:timestamptz" json:"last_used_at,omitempty"`
	ExpiresAt      *time.Time     `gorm:"type:timestamptz" json:"expires_at,omitempty"`
	RevokedAt      *time.Time     `gorm:"type:timestamptz" json:"revoked_at,omitempty"`
	CreatedAt      time.Time      `gorm:"type:timestamptz;not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"type:timestamptz;not null;default:now()" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

func (FileTransferCredential) TableName() string {
	return "file_transfer_credentials"
}

func (c *FileTransferCredential) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if c.ID == "" {
		c.ID = fmt.Sprintf("ftc-%s", uuid.NewString())
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	c.ResourceType = NormalizeFileTransferResourceType(c.ResourceType)
	c.Scopes = NormalizeFileTransferScopes(c.Scopes)
	return nil
}

func (c *FileTransferCredential) BeforeUpdate(tx *gorm.DB) error {
	c.UpdatedAt = time.Now()
	c.ResourceType = NormalizeFileTransferResourceType(c.ResourceType)
	c.Scopes = NormalizeFileTransferScopes(c.Scopes)
	return nil
}

type FileTransferCredentialRepository struct {
	db *gorm.DB
}

func NewFileTransferCredentialRepository(db *gorm.DB) *FileTransferCredentialRepository {
	return &FileTransferCredentialRepository{db: db}
}

func (r *FileTransferCredentialRepository) Create(ctx context.Context, credential *FileTransferCredential) error {
	if credential == nil {
		return fmt.Errorf("credential is required")
	}
	return r.db.WithContext(ctx).Create(credential).Error
}

func (r *FileTransferCredentialRepository) ListActiveByResource(ctx context.Context, resourceType string, resourceID string, now time.Time) ([]*FileTransferCredential, error) {
	resourceType = NormalizeFileTransferResourceType(resourceType)
	var credentials []*FileTransferCredential
	err := r.db.WithContext(ctx).
		Where("resource_type = ? AND resource_id = ? AND revoked_at IS NULL AND deleted_at IS NULL AND (expires_at IS NULL OR expires_at > ?)", resourceType, resourceID, now).
		Order("created_at DESC").
		Find(&credentials).Error
	return credentials, err
}

func (r *FileTransferCredentialRepository) GetActiveBySecret(ctx context.Context, secret string, now time.Time) (*FileTransferCredential, error) {
	var credential FileTransferCredential
	err := r.db.WithContext(ctx).
		Where("key_hash = ? AND revoked_at IS NULL AND deleted_at IS NULL AND (expires_at IS NULL OR expires_at > ?)", HashFileTransferSecret(secret), now).
		First(&credential).Error
	if err != nil {
		return nil, err
	}
	return &credential, nil
}

func (r *FileTransferCredentialRepository) TouchLastUsed(ctx context.Context, id string, usedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&FileTransferCredential{}).
		Where("id = ?", id).
		Update("last_used_at", usedAt).Error
}

func (r *FileTransferCredentialRepository) RevokeByResource(ctx context.Context, id string, resourceType string, resourceID string, revokedAt time.Time) error {
	resourceType = NormalizeFileTransferResourceType(resourceType)
	return r.db.WithContext(ctx).
		Model(&FileTransferCredential{}).
		Where("id = ? AND resource_type = ? AND resource_id = ? AND revoked_at IS NULL AND deleted_at IS NULL", id, resourceType, resourceID).
		Updates(map[string]interface{}{
			"revoked_at": revokedAt,
			"updated_at": revokedAt,
		}).Error
}

func GenerateFileTransferSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "oft_" + base64.RawURLEncoding.EncodeToString(buf), nil
}

func HashFileTransferSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func NormalizeFileTransferResourceType(resourceType string) string {
	switch strings.ToLower(strings.TrimSpace(resourceType)) {
	case "gameserver", "game_server", "game-server", "game_servers", "game-servers", "gameservers":
		return FileTransferResourceGameServer
	default:
		return strings.ToLower(strings.TrimSpace(resourceType))
	}
}

func NormalizeFileTransferScopes(scopes string) string {
	parts := strings.Split(scopes, ",")
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		scope := strings.ToLower(strings.TrimSpace(part))
		switch scope {
		case "all", "*", "read_write", "read-write", "rw", "sftp", "sftp:*":
			scope = FileTransferScopeRead + "," + FileTransferScopeWrite
		case "sftp:read":
			scope = FileTransferScopeRead
		case "sftp:write":
			scope = FileTransferScopeWrite
		}
		for _, normalized := range strings.Split(scope, ",") {
			normalized = strings.TrimSpace(normalized)
			if normalized == "" {
				continue
			}
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			out = append(out, normalized)
		}
	}
	if len(out) == 0 {
		return FileTransferScopeRead
	}
	return strings.Join(out, ",")
}

func FileTransferCredentialHasScope(scopes string, required string) bool {
	required = strings.ToLower(strings.TrimSpace(required))
	for _, scope := range strings.Split(NormalizeFileTransferScopes(scopes), ",") {
		if scope == required {
			return true
		}
	}
	return false
}
