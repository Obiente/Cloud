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
	"gorm.io/gorm/clause"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
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
	now := time.Now()
	var record DelegatedDNSRecord
	// Check the current key and organization on every answer. Existing records
	// with excessive TTLs also stop resolving one hour after their last refresh.
	query := DB.Model(&DelegatedDNSRecord{}).
		Joins("LEFT JOIN dns_delegation_api_keys k ON k.id = delegated_dns_records.api_key_id AND k.organization_id = delegated_dns_records.organization_id").
		Joins("LEFT JOIN organizations o ON o.id = k.organization_id").
		Where("delegated_dns_records.domain = ? AND delegated_dns_records.record_type = ? AND expires_at > ? AND last_updated > ?", normalizeDelegatedDomain(domain), strings.ToUpper(recordType), now, now.Add(-time.Duration(MaxDelegatedDNSTTL)*time.Second)).
		Where("(delegated_dns_records.api_key_id = '' AND delegated_dns_records.organization_id = '') OR (k.is_active = ? AND k.revoked_at IS NULL AND (LOWER(o.status) = 'active' OR (LOWER(o.status) = 'suspended' AND o.suspension_expires <= ?)))", true, now)
	if err := query.Select("delegated_dns_records.*").First(&record).Error; err != nil {
		return nil, err
	}
	// Never let a resolver cache an answer beyond the remaining authorization lease.
	remaining := int64(time.Until(record.ExpiresAt).Seconds())
	maxRemaining := int64(time.Until(record.LastUpdated.Add(time.Duration(MaxDelegatedDNSTTL) * time.Second)).Seconds())
	if remaining > maxRemaining {
		remaining = maxRemaining
	}
	if record.TTL > remaining {
		record.TTL = remaining
	}
	if record.TTL < 1 {
		return nil, gorm.ErrRecordNotFound
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
	domain = normalizeDelegatedDomain(domain)
	recordType = strings.ToUpper(strings.TrimSpace(recordType))
	if ttl == 0 {
		ttl = 300
	}
	if err := validateDelegatedRecord(domain, recordType, recordsJSON, ttl); err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		base := delegatedResourceDomain(domain)
		// Serialize all A/SRV claims for a resource across replicas, including the
		// first insert, which a SELECT FOR UPDATE cannot lock yet.
		if tx.Dialector.Name() == "postgres" {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", "dns-delegation:"+base).Error; err != nil {
				return err
			}
		}
		if err := authorizeDelegationWrite(tx, domain, apiKeyID, organizationID); err != nil {
			return err
		}
		// Compare normalized legacy names as well. Expiry stops DNS answers but
		// does not release a remote installation's hostname to another tenant.
		var existing []DelegatedDNSRecord
		if err := tx.Where("LOWER(RTRIM(domain, '.')) = ? OR LOWER(RTRIM(domain, '.')) LIKE ?", base, "%._tcp."+base).
			Or("LOWER(RTRIM(domain, '.')) LIKE ?", "%._udp."+base).Find(&existing).Error; err != nil {
			return err
		}
		for _, record := range existing {
			if delegatedResourceDomain(normalizeDelegatedDomain(record.Domain)) == base && (record.OrganizationID != organizationID || (record.APIKeyID == "") != (apiKeyID == "")) {
				return ErrDNSDelegationForbidden
			}
		}
		now := time.Now()
		record := DelegatedDNSRecord{ID: uuid.NewString(), Domain: domain, RecordType: recordType, Records: recordsJSON, SourceAPI: sourceAPI, APIKeyID: apiKeyID, OrganizationID: organizationID, TTL: ttl, ExpiresAt: now.Add(time.Duration(ttl) * time.Second), LastUpdated: now}
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "domain"}, {Name: "record_type"}},
			DoUpdates: clause.AssignmentColumns([]string{"records", "source_api", "api_key_id", "ttl", "expires_at", "last_updated", "updated_at"}),
			Where:     clause.Where{Exprs: []clause.Expression{clause.Eq{Column: clause.Column{Table: "delegated_dns_records", Name: "organization_id"}, Value: organizationID}}},
		}).Create(&record)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrDNSDelegationForbidden
		}
		return nil
	})
}

// CleanupExpiredDelegatedRecords removes expired local mirrors. Authenticated
// records retain their ownership after expiry/revocation, so an outage cannot
// let another organization claim a hostname before its owner reconnects.
func CleanupExpiredDelegatedRecords() error {
	return DB.Where("expires_at < ? AND api_key_id = '' AND organization_id = ''", time.Now()).Delete(&DelegatedDNSRecord{}).Error
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
	if err := activeDelegationOrganization(DB, key.OrganizationID, false); err != nil {
		return nil, err
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
	return revokeDelegationKeys("stripe_subscription_id = ?", stripeSubscriptionID)
}

// RevokeDNSDelegationAPIKey revokes an API key by hash.
func RevokeDNSDelegationAPIKey(keyHash string) error {
	return revokeDelegationKeys("key_hash = ?", keyHash)
}

func revokeDelegationKeys(condition string, value string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := tx.Model(&DNSDelegationAPIKey{}).Where(condition, value).
			Updates(map[string]any{"is_active": false, "revoked_at": now}).Error; err != nil {
			return err
		}
		keys := tx.Model(&DNSDelegationAPIKey{}).Select("id").Where(condition, value)
		// UpdateColumn avoids the record hook that normally renews its expiry.
		return tx.Model(&DelegatedDNSRecord{}).Where("api_key_id IN (?)", keys).UpdateColumn("expires_at", now).Error
	})
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
	now := time.Now()

	query := DB.Model(&DelegatedDNSRecord{}).Where("expires_at > ?", now).Order("created_at DESC")

	if organizationID != "" {
		query = query.Where("organization_id = ?", organizationID)
	}

	if apiKeyID != "" {
		query = query.Where("api_key_id = ?", apiKeyID)
	}

	if recordType != "" {
		query = query.Where("record_type = ?", recordType)
	}

	// Note: We can't easily get SQL from GORM v2, but the query structure is clear from the logs above

	if err := query.Find(&records).Error; err != nil {
		logger.Error("[Database] Query error: %v", err)
		return nil, err
	}

	return records, nil
}
