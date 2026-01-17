package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/sftp"
)

const (
	resourceDeployment = "deployment"
	resourceGameServer = "gameserver"
)

// APIKeyValidator validates API keys against the database
type APIKeyValidator struct{}

// NewAPIKeyValidator creates a new API key validator
func NewAPIKeyValidator() *APIKeyValidator { return &APIKeyValidator{} }

// ValidateAPIKey validates an API key and returns user info, resource scope, and permissions
func (v *APIKeyValidator) ValidateAPIKey(ctx context.Context, apiKey string) (string, string, string, string, []sftp.Permission, error) {
	if database.DB == nil {
		return "", "", "", "", nil, fmt.Errorf("database not initialized")
	}

	var key database.APIKey
	if err := database.DB.WithContext(ctx).
		Where("key_hash = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?)", hashAPIKey(apiKey), time.Now()).
		Preload("Organization").
		First(&key).Error; err != nil {
		logger.Debug("[SFTP Auth] API key validation failed: %v", err)
		return "", "", "", "", nil, fmt.Errorf("invalid API key")
	}

	resourceType := normalizeResourceType(key.ResourceType)
	if resourceType == "" || key.ResourceID == "" {
		return "", "", "", "", nil, fmt.Errorf("API key missing resource binding")
	}

	if err := v.ensureResourceExists(ctx, resourceType, key.ResourceID, key.OrganizationID); err != nil {
		logger.Debug("[SFTP Auth] API key %s has invalid resource binding: %v", key.ID, err)
		return "", "", "", "", nil, fmt.Errorf("API key is not bound to a valid resource")
	}

	scopes := parseScopes(key.Scopes)
	permissions := scopesToPermissions(resourceType, scopes)
	if len(permissions) == 0 {
		return "", "", "", "", nil, fmt.Errorf("API key does not have SFTP permissions")
	}

	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		database.DB.WithContext(updateCtx).Model(&key).Update("last_used_at", time.Now())
	}()

	logger.Info("[SFTP Auth] API key validated: user=%s, org=%s, resource=%s:%s, permissions=%v", key.UserID, key.OrganizationID, resourceType, key.ResourceID, permissions)

	return key.UserID, key.OrganizationID, resourceType, key.ResourceID, permissions, nil
}

// SFTPAuditLogger logs SFTP operations to the audit log
type SFTPAuditLogger struct{}

// NewSFTPAuditLogger creates a new SFTP audit logger
func NewSFTPAuditLogger() *SFTPAuditLogger { return &SFTPAuditLogger{} }

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
		IPAddress:      "sftp",
		UserAgent:      "sftp-client",
		RequestData:    fmt.Sprintf(`{"path":"%s","bytes_written":%d,"bytes_read":%d}`, entry.Path, entry.BytesWritten, entry.BytesRead),
		ResponseStatus: responseStatus(entry.Success),
		ErrorMessage:   errorMessage(entry.ErrorMessage),
		DurationMs:     0,
		CreatedAt:      time.Now(),
	}

	if err := database.MetricsDB.WithContext(ctx).Create(&auditLog).Error; err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	logger.Debug("[SFTP Audit] Logged operation: user=%s, action=%s, path=%s, success=%v", entry.UserID, entry.Operation, entry.Path, entry.Success)
	return nil
}

// Helper functions

func hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
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

func normalizeResourceType(rt string) string {
	switch strings.ToLower(strings.TrimSpace(rt)) {
	case "deployment", "deployments":
		return resourceDeployment
	case "gameserver", "gameservers", "game_server", "game-servers":
		return resourceGameServer
	default:
		return ""
	}
}

func scopesToPermissions(resourceType string, scopes []string) []sftp.Permission {
	permissions := make([]sftp.Permission, 0)
	hasRead := false
	hasWrite := false

	// Normalize once
	normalized := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		s := strings.TrimSpace(strings.ToLower(scope))
		if s != "" {
			normalized = append(normalized, s)
		}
	}

	// Legacy SFTP scopes
	if containsScope(normalized, "sftp:read") || containsScope(normalized, "sftp:*") || containsScope(normalized, "sftp") {
		hasRead = true
		permissions = append(permissions, sftp.PermissionRead)
	}
	if containsScope(normalized, "sftp:write") || containsScope(normalized, "sftp:*") || containsScope(normalized, "sftp") {
		hasWrite = true
		permissions = append(permissions, sftp.PermissionWrite)
	}

	// Auth permission-based scopes
	if !hasRead && hasReadPermission(resourceType, normalized) {
		hasRead = true
		permissions = append(permissions, sftp.PermissionRead)
	}
	if !hasWrite && hasWritePermission(resourceType, normalized) {
		hasWrite = true
		permissions = append(permissions, sftp.PermissionWrite)
	}

	return permissions
}

func hasReadPermission(resourceType string, scopes []string) bool {
	switch resourceType {
	case resourceDeployment:
		return matchesAnyScope(scopes,
			auth.PermissionDeploymentRead,
			auth.PermissionDeploymentLogs,
			auth.PermissionDeploymentAll,
		)
	case resourceGameServer:
		return matchesAnyScope(scopes,
			auth.PermissionGameServersRead,
			auth.PermissionGameServersAll,
		)
	default:
		return false
	}
}

func hasWritePermission(resourceType string, scopes []string) bool {
	switch resourceType {
	case resourceDeployment:
		return matchesAnyScope(scopes,
			auth.PermissionDeploymentUpdate,
			auth.PermissionDeploymentManage,
			auth.PermissionDeploymentDeploy,
			auth.PermissionDeploymentAll,
		)
	case resourceGameServer:
		return matchesAnyScope(scopes,
			auth.PermissionGameServersUpdate,
			auth.PermissionGameServersManage,
			auth.PermissionGameServersAll,
		)
	default:
		return false
	}
}

func matchesAnyScope(scopes []string, candidates ...string) bool {
	for _, cand := range candidates {
		candLower := strings.ToLower(strings.TrimSpace(cand))
		for _, s := range scopes {
			if s == candLower {
				return true
			}
			// wildcard candidate e.g., deployment.*
			if strings.HasSuffix(candLower, ".*") {
				prefix := strings.TrimSuffix(candLower, "*")
				if strings.HasPrefix(s, prefix) {
					return true
				}
			}
			// wildcard in scope value
			if strings.HasSuffix(s, ".*") {
				prefix := strings.TrimSuffix(s, "*")
				if strings.HasPrefix(candLower, prefix) {
					return true
				}
			}
		}
	}
	return false
}

func containsScope(scopes []string, target string) bool {
	target = strings.ToLower(target)
	for _, s := range scopes {
		if s == target {
			return true
		}
	}
	return false
}

func (v *APIKeyValidator) ensureResourceExists(ctx context.Context, resourceType, resourceID, orgID string) error {
	switch resourceType {
	case resourceDeployment:
		var deployment database.Deployment
		if err := database.DB.WithContext(ctx).Where("id = ? AND organization_id = ?", resourceID, orgID).First(&deployment).Error; err != nil {
			return fmt.Errorf("deployment not found or not in organization")
		}
	case resourceGameServer:
		var gs database.GameServer
		if err := database.DB.WithContext(ctx).Where("id = ? AND organization_id = ?", resourceID, orgID).First(&gs).Error; err != nil {
			return fmt.Errorf("game server not found or not in organization")
		}
	default:
		return fmt.Errorf("unsupported resource type: %s", resourceType)
	}
	return nil
}

func generateID() string { return uuid.New().String() }

func stringPtr(s string) *string { return &s }

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
