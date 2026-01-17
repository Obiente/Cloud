package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/sftp"
)

// APIKeyValidator validates API keys against the database
type APIKeyValidator struct {
}

// NewAPIKeyValidator creates a new API key validator
func NewAPIKeyValidator() *APIKeyValidator {
	return &APIKeyValidator{}
}

// ValidateAPIKey validates an API key and returns user info and permissions
func (v *APIKeyValidator) ValidateAPIKey(ctx context.Context, apiKey string) (string, string, []sftp.Permission, error) {
	if database.DB == nil {
		return "", "", nil, fmt.Errorf("database not initialized")
	}

	// Query API key from database
	var key database.APIKey
	if err := database.DB.WithContext(ctx).
		Where("key_hash = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?)", 
			hashAPIKey(apiKey), time.Now()).
		Preload("Organization").
		First(&key).Error; err != nil {
		logger.Debug("[SFTP Auth] API key validation failed: %v", err)
		return "", "", nil, fmt.Errorf("invalid API key")
	}

	// Check if key has SFTP scopes
	scopes := parseScopes(key.Scopes)
	permissions := scopesToPermissions(scopes)
	
	if len(permissions) == 0 {
		logger.Debug("[SFTP Auth] API key %s has no SFTP permissions", key.ID)
		return "", "", nil, fmt.Errorf("API key does not have SFTP permissions")
	}

	// Update last used timestamp
	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		database.DB.WithContext(updateCtx).Model(&key).Update("last_used_at", time.Now())
	}()

	logger.Info("[SFTP Auth] API key validated: user=%s, org=%s, permissions=%v", 
		key.UserID, key.OrganizationID, permissions)

	return key.UserID, key.OrganizationID, permissions, nil
}

// SFTPAuditLogger logs SFTP operations to the audit log
type SFTPAuditLogger struct{}

// NewSFTPAuditLogger creates a new SFTP audit logger
func NewSFTPAuditLogger() *SFTPAuditLogger {
	return &SFTPAuditLogger{}
}

// LogOperation logs an SFTP operation
func (l *SFTPAuditLogger) LogOperation(ctx context.Context, entry sftp.AuditEntry) error {
	if database.MetricsDB == nil {
		logger.Debug("[SFTP Audit] Skipping audit log: metrics database not initialized")
		return nil
	}

	auditLog := database.AuditLog{
		ID:             generateID(),
		UserID:         entry.UserID,
		OrganizationID: &entry.OrgID,
		Action:         entry.Operation,
		Service:        "SFTPService",
		ResourceType:   stringPtr("sftp_file"),
		ResourceID:     stringPtr(entry.Path),
		IPAddress:      "sftp", // SFTP doesn't have HTTP-style IP tracking
		UserAgent:      "sftp-client",
		RequestData:    fmt.Sprintf(`{"path":"%s","bytes_written":%d,"bytes_read":%d}`, 
			entry.Path, entry.BytesWritten, entry.BytesRead),
		ResponseStatus: responseStatus(entry.Success),
		ErrorMessage:   errorMessage(entry.ErrorMessage),
		DurationMs:     0, // We don't track duration for individual file operations
		CreatedAt:      time.Now(),
	}

	if err := database.MetricsDB.WithContext(ctx).Create(&auditLog).Error; err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	logger.Debug("[SFTP Audit] Logged operation: user=%s, action=%s, path=%s, success=%v", 
		entry.UserID, entry.Operation, entry.Path, entry.Success)

	return nil
}

// Helper functions

func hashAPIKey(apiKey string) string {
	// In production, use proper hashing (SHA-256)
	// For now, we'll use the key as-is for simplicity
	// TODO: Implement proper API key hashing
	return apiKey
}

func parseScopes(scopesStr string) []string {
	if scopesStr == "" {
		return nil
	}
	
	scopes := strings.Split(scopesStr, ",")
	result := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func scopesToPermissions(scopes []string) []sftp.Permission {
	permissions := make([]sftp.Permission, 0)
	hasRead := false
	hasWrite := false
	
	for _, scope := range scopes {
		switch scope {
		case "sftp:read", "sftp", "sftp:*":
			if !hasRead {
				permissions = append(permissions, sftp.PermissionRead)
				hasRead = true
			}
		case "sftp:write":
			if !hasWrite {
				permissions = append(permissions, sftp.PermissionWrite)
				hasWrite = true
			}
		}
	}
	
	return permissions
}

func generateID() string {
	return uuid.New().String()
}

func stringPtr(s string) *string {
	return &s
}

func responseStatus(success bool) int32 {
	if success {
		return 200
	}
	return 500
}

func errorMessage(msg string) *string {
	if msg == "" {
		return nil
	}
	return &msg
}
