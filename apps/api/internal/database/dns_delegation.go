package database

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"api/internal/logger"
)

// DelegatedDNSRecord stores DNS records pushed by remote APIs (dev/self-hosted)
// These records expire if not refreshed within TTL
type DelegatedDNSRecord struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	Domain         string    `gorm:"uniqueIndex:idx_domain_type;not null" json:"domain"`      // e.g., "deploy-123.my.obiente.cloud"
	RecordType     string    `gorm:"uniqueIndex:idx_domain_type;not null" json:"record_type"` // "A" or "SRV"
	Records        string    `gorm:"type:jsonb;not null" json:"records"`                      // JSON array of record values
	SourceAPI      string    `gorm:"index;not null" json:"source_api"`                        // URL of the API that pushed this record
	APIKeyID       string    `gorm:"index" json:"api_key_id"`                                 // ID of the API key that delegated this record
	OrganizationID string    `gorm:"index" json:"organization_id"`                            // Organization that owns the API key
	TTL            int64     `gorm:"default:300" json:"ttl"`                                  // TTL in seconds (default: 5 minutes)
	ExpiresAt      time.Time `gorm:"index;not null" json:"expires_at"`                        // When this record expires
	LastUpdated    time.Time `gorm:"not null" json:"last_updated"`                            // Last time this record was updated
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (DelegatedDNSRecord) TableName() string {
	return "delegated_dns_records"
}

// BeforeCreate hook to set timestamps
func (d *DelegatedDNSRecord) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	if d.LastUpdated.IsZero() {
		d.LastUpdated = now
	}
	if d.ExpiresAt.IsZero() {
		// Set expiration based on TTL
		d.ExpiresAt = now.Add(time.Duration(d.TTL) * time.Second)
	}
	return nil
}

// BeforeUpdate hook to update timestamps and expiration
func (d *DelegatedDNSRecord) BeforeUpdate(tx *gorm.DB) error {
	d.LastUpdated = time.Now()
	// Refresh expiration based on TTL
	d.ExpiresAt = d.LastUpdated.Add(time.Duration(d.TTL) * time.Second)
	return nil
}

// IsExpired checks if the record has expired
func (d *DelegatedDNSRecord) IsExpired() bool {
	return time.Now().After(d.ExpiresAt)
}

// GetDelegatedDNSRecord retrieves a delegated DNS record if it exists and is not expired
func GetDelegatedDNSRecord(domain, recordType string) (*DelegatedDNSRecord, error) {
	var record DelegatedDNSRecord
	result := DB.Where("domain = ? AND record_type = ? AND expires_at > ?", domain, recordType, time.Now()).
		First(&record)
	if result.Error != nil {
		return nil, result.Error
	}
	return &record, nil
}

// UpsertDelegatedDNSRecord creates or updates a delegated DNS record
func UpsertDelegatedDNSRecord(domain, recordType, recordsJSON, sourceAPI string, ttl int64) error {
	return UpsertDelegatedDNSRecordWithAPIKey(domain, recordType, recordsJSON, sourceAPI, "", "", ttl)
}

// UpsertDelegatedDNSRecordWithAPIKey creates or updates a delegated DNS record with API key tracking
// Uses a transaction to prevent race conditions and handle expired records correctly
func UpsertDelegatedDNSRecordWithAPIKey(domain, recordType, recordsJSON, sourceAPI, apiKeyID, organizationID string, ttl int64) error {
	now := time.Now()
	expiresAt := now.Add(time.Duration(ttl) * time.Second)

	return DB.Transaction(func(tx *gorm.DB) error {
		// First, try to find existing record (even if expired)
		var existing DelegatedDNSRecord
		result := tx.Where("domain = ? AND record_type = ?", domain, recordType).First(&existing)
		
		if result.Error == nil {
			// Record exists - update it (this refreshes expiration even if it was expired)
			updateData := map[string]interface{}{
				"records":         recordsJSON,
				"source_api":      sourceAPI,
				"api_key_id":      apiKeyID,
				"organization_id": organizationID,
				"ttl":             ttl,
				"expires_at":      expiresAt,
				"last_updated":    now,
				"updated_at":      now,
			}
			return tx.Model(&existing).Updates(updateData).Error
		} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Record doesn't exist - create new one
			record := DelegatedDNSRecord{
				ID:             uuid.New().String(),
				Domain:         domain,
				RecordType:     recordType,
				Records:        recordsJSON,
				SourceAPI:      sourceAPI,
				APIKeyID:       apiKeyID,
				OrganizationID: organizationID,
				TTL:            ttl,
				ExpiresAt:      expiresAt,
				LastUpdated:    now,
			}
			// If create fails due to unique constraint (race condition), try update instead
			if err := tx.Create(&record).Error; err != nil {
				// Check if it's a unique constraint violation
				if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
					// Race condition: record was created between our check and create
					// Try to update it instead
					var raceRecord DelegatedDNSRecord
					if tx.Where("domain = ? AND record_type = ?", domain, recordType).First(&raceRecord).Error == nil {
						updateData := map[string]interface{}{
							"records":         recordsJSON,
							"source_api":      sourceAPI,
							"api_key_id":      apiKeyID,
							"organization_id": organizationID,
							"ttl":             ttl,
							"expires_at":      expiresAt,
							"last_updated":    now,
							"updated_at":      now,
						}
						return tx.Model(&raceRecord).Updates(updateData).Error
					}
				}
				return err
			}
			return nil
		} else {
			// Database error
			return result.Error
		}
	})
}

// CleanupExpiredDelegatedRecords removes expired delegated DNS records
func CleanupExpiredDelegatedRecords() error {
	return DB.Where("expires_at < ?", time.Now()).Delete(&DelegatedDNSRecord{}).Error
}

// DNSDelegationAPIKey stores API keys for DNS delegation
// Self-hosters get an API key from production to push DNS records
// API keys are linked to organizations via subscriptions
type DNSDelegationAPIKey struct {
	ID                   string     `gorm:"primaryKey" json:"id"`
	KeyHash              string     `gorm:"uniqueIndex;not null" json:"key_hash"` // SHA256 hash of the API key
	Description          string     `json:"description"`                          // Description of who/what this key is for
	SourceAPI            string     `gorm:"index" json:"source_api"`              // URL of the API that uses this key (optional)
	OrganizationID       string     `gorm:"index" json:"organization_id"`         // Organization that owns this key
	StripeSubscriptionID *string    `gorm:"index" json:"stripe_subscription_id"`  // Stripe subscription ID (optional, for subscription-based keys)
	IsActive             bool       `gorm:"default:true;index" json:"is_active"`  // Can be revoked
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	RevokedAt            *time.Time `json:"revoked_at"` // When this key was revoked
}

func (DNSDelegationAPIKey) TableName() string {
	return "dns_delegation_api_keys"
}

// ValidateDNSDelegationAPIKey checks if an API key is valid
func ValidateDNSDelegationAPIKey(apiKey string) (bool, error) {
	_, err := GetDNSDelegationAPIKeyByHash(apiKey)
	return err == nil, err
}

// GetDNSDelegationAPIKeyByHash retrieves an API key by its hash
func GetDNSDelegationAPIKeyByHash(apiKey string) (*DNSDelegationAPIKey, error) {
	// Hash the provided key
	keyHash := hashAPIKey(apiKey)

	var key DNSDelegationAPIKey
	result := DB.Where("key_hash = ? AND is_active = ? AND revoked_at IS NULL", keyHash, true).
		First(&key)

	if result.Error != nil {
		return nil, result.Error
	}
	return &key, nil
}

// hashAPIKey hashes an API key using SHA256
func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// HashAPIKeyForDelegation exports the hash function for use in handlers
func HashAPIKeyForDelegation(key string) string {
	return hashAPIKey(key)
}

// CreateDNSDelegationAPIKey creates a new API key (returns the key, not the hash)
// If organizationID is provided, the key is linked to that organization
// If stripeSubscriptionID is provided, the key is linked to that subscription
func CreateDNSDelegationAPIKey(description, sourceAPI, organizationID string, stripeSubscriptionID *string) (string, error) {
	// Generate a random API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", err
	}
	apiKey := base64.URLEncoding.EncodeToString(keyBytes)

	// Hash the key for storage
	keyHash := hashAPIKey(apiKey)

	key := DNSDelegationAPIKey{
		ID:                   uuid.New().String(),
		KeyHash:              keyHash,
		Description:          description,
		SourceAPI:            sourceAPI,
		OrganizationID:       organizationID,
		StripeSubscriptionID: stripeSubscriptionID,
		IsActive:             true,
	}

	logger.Debug("[Database] Creating DNS delegation API key: org=%s, subscription=%v", organizationID, stripeSubscriptionID != nil && *stripeSubscriptionID != "")

	if err := DB.Create(&key).Error; err != nil {
		logger.Error("[Database] Failed to create DNS delegation API key: %v", err)
		return "", err
	}

	logger.Debug("[Database] Created DNS delegation API key: id=%s, org=%s, subscription=%v", key.ID, key.OrganizationID, key.StripeSubscriptionID != nil && *key.StripeSubscriptionID != "")

	return apiKey, nil
}

// GetActiveDNSDelegationAPIKeyForOrganization gets an active API key for an organization
func GetActiveDNSDelegationAPIKeyForOrganization(organizationID string) (*DNSDelegationAPIKey, error) {
	var key DNSDelegationAPIKey
	result := DB.Where("organization_id = ? AND is_active = ? AND revoked_at IS NULL", organizationID, true).
		First(&key)
	if result.Error != nil {
		return nil, result.Error
	}
	return &key, nil
}

// HasActiveDNSDelegationSubscription checks if an organization has an active DNS delegation subscription
// This queries the DNSDelegationAPIKey table for active keys linked to the organization
func HasActiveDNSDelegationSubscription(organizationID string) (bool, string, error) {
	var key DNSDelegationAPIKey
	result := DB.Where("organization_id = ? AND is_active = ? AND revoked_at IS NULL AND stripe_subscription_id IS NOT NULL", organizationID, true).
		First(&key)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, "", nil
	}
	if result.Error != nil {
		return false, "", result.Error
	}
	if key.StripeSubscriptionID != nil {
		return true, *key.StripeSubscriptionID, nil
	}
	return false, "", nil
}

// RevokeDNSDelegationAPIKeysForSubscription revokes all API keys for a subscription
func RevokeDNSDelegationAPIKeysForSubscription(stripeSubscriptionID string) error {
	now := time.Now()
	return DB.Model(&DNSDelegationAPIKey{}).
		Where("stripe_subscription_id = ?", stripeSubscriptionID).
		Updates(map[string]interface{}{
			"is_active":  false,
			"revoked_at": now,
		}).Error
}

// RevokeDNSDelegationAPIKey revokes an API key by hash
func RevokeDNSDelegationAPIKey(keyHash string) error {
	now := time.Now()
	return DB.Model(&DNSDelegationAPIKey{}).
		Where("key_hash = ?", keyHash).
		Updates(map[string]interface{}{
			"is_active":  false,
			"revoked_at": now,
		}).Error
}

// ListDNSDelegationAPIKeys lists DNS delegation API keys, optionally filtered by organization ID
func ListDNSDelegationAPIKeys(organizationID string) ([]DNSDelegationAPIKey, error) {
	var keys []DNSDelegationAPIKey
	query := DB.Model(&DNSDelegationAPIKey{}).Order("created_at DESC")

	if organizationID != "" {
		query = query.Where("organization_id = ?", organizationID)
	}

	if err := query.Find(&keys).Error; err != nil {
		return nil, err
	}

	return keys, nil
}

// ListDelegatedDNSRecords lists delegated DNS records with optional filters
func ListDelegatedDNSRecords(organizationID, apiKeyID, recordType string) ([]DelegatedDNSRecord, error) {
	var records []DelegatedDNSRecord
	query := DB.Model(&DelegatedDNSRecord{}).Where("expires_at > ?", time.Now()).Order("created_at DESC")

	if organizationID != "" {
		query = query.Where("organization_id = ?", organizationID)
	}

	if apiKeyID != "" {
		query = query.Where("api_key_id = ?", apiKeyID)
	}

	if recordType != "" {
		query = query.Where("record_type = ?", recordType)
	}

	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}

	return records, nil
}
