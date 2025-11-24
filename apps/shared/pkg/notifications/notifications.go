package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	notificationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1"
)

// CreateNotificationForUser creates a notification for a specific user
func CreateNotificationForUser(ctx context.Context, userID string, orgID *string, notificationType notificationsv1.NotificationType, severity notificationsv1.NotificationSeverity, title, message string, actionURL, actionLabel *string, metadata map[string]string) error {
	notification := &database.Notification{
		ID:             generateID("notif"),
		UserID:         userID,
		OrganizationID: orgID,
		Type:           notificationTypeToString(notificationType),
		Severity:       notificationSeverityToString(severity),
		Title:          title,
		Message:        message,
		Read:           false,
		ClientOnly:     false,
	}

	if actionURL != nil {
		notification.ActionURL = actionURL
	}
	if actionLabel != nil {
		notification.ActionLabel = actionLabel
	}

	// Convert metadata map to JSON
	if len(metadata) > 0 {
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
		notification.Metadata = string(metadataJSON)
	}

	if err := database.DB.Create(notification).Error; err != nil {
		return fmt.Errorf("create notification: %w", err)
	}

	return nil
}

// CreateNotificationForOrganization creates notifications for all active members of an organization
func CreateNotificationForOrganization(ctx context.Context, orgID string, notificationType notificationsv1.NotificationType, severity notificationsv1.NotificationSeverity, title, message string, actionURL, actionLabel *string, metadata map[string]string, roles []string) error {
	// Get organization members
	var members []database.OrganizationMember
	query := database.DB.Where("organization_id = ? AND status = ?", orgID, "active")

	// Filter by roles if specified
	if len(roles) > 0 {
		query = query.Where("role IN ?", roles)
	}

	if err := query.Find(&members).Error; err != nil {
		return fmt.Errorf("get organization members: %w", err)
	}

	// Create notifications for each member
	for _, member := range members {
		// Skip pending invites
		if member.UserID == "" || len(member.UserID) < 7 || member.UserID[0:7] == "pending" {
			continue
		}

		if err := CreateNotificationForUser(ctx, member.UserID, &orgID, notificationType, severity, title, message, actionURL, actionLabel, metadata); err != nil {
			logger.Warn("[Notifications] Failed to create notification for user %s: %v", member.UserID, err)
			// Continue with other members
		}
	}

	return nil
}

// CreateNotificationForUserByEmail creates a notification for a user by their email
// This is useful for invites where we only have the email
// Note: For pending invites, we'll create the notification when the user accepts the invite
func CreateNotificationForUserByEmail(ctx context.Context, email string, orgID *string, notificationType notificationsv1.NotificationType, severity notificationsv1.NotificationSeverity, title, message string, actionURL, actionLabel *string, metadata map[string]string) error {
	// For invites, we need to find the user via Zitadel or wait until they accept
	// For now, we'll skip creating notifications for users that don't exist yet
	// The notification will be created when they accept the invite (see AcceptInvite)
	return nil
}

// Helper functions

func generateID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func notificationTypeToString(t notificationsv1.NotificationType) string {
	switch t {
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_INFO:
		return "INFO"
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_SUCCESS:
		return "SUCCESS"
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_WARNING:
		return "WARNING"
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_ERROR:
		return "ERROR"
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_DEPLOYMENT:
		return "DEPLOYMENT"
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_BILLING:
		return "BILLING"
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_QUOTA:
		return "QUOTA"
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_INVITE:
		return "INVITE"
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_SYSTEM:
		return "SYSTEM"
	default:
		return "INFO"
	}
}

func notificationSeverityToString(s notificationsv1.NotificationSeverity) string {
	switch s {
	case notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_LOW:
		return "LOW"
	case notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_MEDIUM:
		return "MEDIUM"
	case notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_HIGH:
		return "HIGH"
	case notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_CRITICAL:
		return "CRITICAL"
	default:
		return "MEDIUM"
	}
}
