package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

var ErrUserRestricted = errors.New("account is suspended or banned")
var ErrOrganizationRestricted = errors.New("organization is suspended or banned")

// EnsureUserActive reads current moderation state even when identity is cached.
// Failure to read that state must not grant access.
func EnsureUserActive(ctx context.Context, db *gorm.DB, userID string) error {
	if db == nil || userID == "" {
		return fmt.Errorf("cannot verify account access")
	}
	var count int64
	err := db.WithContext(ctx).Model(&UserBan{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Where("type = ? OR (type = ? AND (expires_at IS NULL OR expires_at > ?))", "banned", "suspended", time.Now()).
		Count(&count).Error
	if err != nil {
		return fmt.Errorf("cannot verify account access: %w", err)
	}
	if count > 0 {
		return ErrUserRestricted
	}
	return nil
}

// EnsureOrganizationActive is also used by background resource start paths.
func EnsureOrganizationActive(ctx context.Context, db *gorm.DB, orgID string) error {
	if db == nil || orgID == "" {
		return fmt.Errorf("cannot verify organization access")
	}
	var org Organization
	if err := db.WithContext(ctx).Select("id", "status", "suspension_expires").First(&org, "id = ?", orgID).Error; err != nil {
		return fmt.Errorf("cannot verify organization access: %w", err)
	}
	if org.Status == "banned" || (org.Status == "suspended" && (org.SuspensionExpires == nil || org.SuspensionExpires.After(time.Now()))) {
		return ErrOrganizationRestricted
	}
	return nil
}

// EnsureResourceOrganizationActive protects unattended recovery as well as API starts.
func EnsureResourceOrganizationActive(ctx context.Context, db *gorm.DB, model interface{}, id string) error {
	if db == nil {
		return fmt.Errorf("cannot verify organization access")
	}
	var resource struct{ OrganizationID string }
	if err := db.WithContext(ctx).Model(model).Select("organization_id").Where("id = ? AND deleted_at IS NULL", id).Take(&resource).Error; err != nil {
		return fmt.Errorf("cannot verify resource organization: %w", err)
	}
	return EnsureOrganizationActive(ctx, db, resource.OrganizationID)
}
