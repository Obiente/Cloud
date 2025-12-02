package vps

import (
	"context"
	"fmt"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/notifications"
	notificationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1"
	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"
)

// notifyVPSEvent sends a notification for a VPS event
func (s *Service) notifyVPSEvent(ctx context.Context, vps *database.VPSInstance, eventType string, severity notificationsv1.NotificationSeverity, title, message string, metadata map[string]string) {
	// Get user from context if available
	userInfo, err := auth.GetUserFromContext(ctx)
	userID := ""
	if err == nil && userInfo != nil {
		userID = userInfo.Id
	}

	// If no user in context, try to get from VPS creator
	if userID == "" && vps.CreatedBy != "" {
		userID = vps.CreatedBy
	}

	// If still no user ID, try to notify organization members
	if userID == "" {
		// Notify organization members instead
		if vps.OrganizationID != "" {
			orgID := vps.OrganizationID
			if err := notifications.CreateNotificationForOrganization(
				ctx,
				orgID,
				notificationsv1.NotificationType_NOTIFICATION_TYPE_SYSTEM,
				severity,
				title,
				message,
				nil, // actionURL
				nil, // actionLabel
				metadata,
				nil, // roles - notify all members
			); err != nil {
				logger.Warn("[VPS Notifications] Failed to create organization notification for VPS %s event %s: %v", vps.ID, eventType, err)
			} else {
				logger.Info("[VPS Notifications] Created organization notification for VPS %s event %s", vps.ID, eventType)
			}
		}
		return
	}

	// Create notification for the user
	orgID := &vps.OrganizationID
	if vps.OrganizationID == "" {
		orgID = nil
	}

	// Build action URL to VPS detail page
	actionURL := fmt.Sprintf("/vps/%s", vps.ID)
	actionLabel := "View VPS"

	// Add VPS metadata
	if metadata == nil {
		metadata = make(map[string]string)
	}
	metadata["vps_id"] = vps.ID
	metadata["vps_name"] = vps.Name
	metadata["event_type"] = eventType
	if vps.InstanceID != nil {
		metadata["vm_id"] = *vps.InstanceID
	}

	if err := notifications.CreateNotificationForUser(
		ctx,
		userID,
		orgID,
		notificationsv1.NotificationType_NOTIFICATION_TYPE_SYSTEM,
		severity,
		title,
		message,
		&actionURL,
		&actionLabel,
		metadata,
	); err != nil {
		logger.Warn("[VPS Notifications] Failed to create notification for VPS %s event %s: %v", vps.ID, eventType, err)
	} else {
		logger.Info("[VPS Notifications] Created notification for VPS %s event %s (user: %s)", vps.ID, eventType, userID)
	}
}

// notifyVPSCreated sends a notification when a VPS is created
func (s *Service) notifyVPSCreated(ctx context.Context, vps *database.VPSInstance) {
	title := fmt.Sprintf("VPS Created: %s", vps.Name)
	message := fmt.Sprintf("Your VPS instance '%s' is being created. You'll be notified when it's ready.", vps.Name)
	
	metadata := map[string]string{
		"vps_status": fmt.Sprintf("%d", vps.Status),
		"vps_region": vps.Region,
		"vps_size":   vps.Size,
	}

	s.notifyVPSEvent(ctx, vps, "vps_created", notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_MEDIUM, title, message, metadata)
}

// notifyVPSReady sends a notification when a VPS becomes ready (running)
func (s *Service) notifyVPSReady(ctx context.Context, vps *database.VPSInstance) {
	title := fmt.Sprintf("VPS Ready: %s", vps.Name)
	message := fmt.Sprintf("Your VPS instance '%s' is now running and ready to use.", vps.Name)
	
	metadata := map[string]string{
		"vps_status": fmt.Sprintf("%d", vps.Status),
	}

	s.notifyVPSEvent(ctx, vps, "vps_ready", notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_LOW, title, message, metadata)
}

// notifyVPSDeleted sends a notification when a VPS is deleted
func (s *Service) notifyVPSDeleted(ctx context.Context, vps *database.VPSInstance) {
	title := fmt.Sprintf("VPS Deleted: %s", vps.Name)
	message := fmt.Sprintf("Your VPS instance '%s' has been deleted.", vps.Name)
	
	metadata := map[string]string{
		"vps_status": fmt.Sprintf("%d", vps.Status),
	}

	s.notifyVPSEvent(ctx, vps, "vps_deleted", notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_MEDIUM, title, message, metadata)
}

// notifyVPSStarted sends a notification when a VPS is started
func (s *Service) notifyVPSStarted(ctx context.Context, vps *database.VPSInstance) {
	title := fmt.Sprintf("VPS Started: %s", vps.Name)
	message := fmt.Sprintf("Your VPS instance '%s' has been started.", vps.Name)
	
	metadata := map[string]string{
		"vps_status": fmt.Sprintf("%d", vps.Status),
	}

	s.notifyVPSEvent(ctx, vps, "vps_started", notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_LOW, title, message, metadata)
}

// notifyVPSStopped sends a notification when a VPS is stopped
func (s *Service) notifyVPSStopped(ctx context.Context, vps *database.VPSInstance) {
	title := fmt.Sprintf("VPS Stopped: %s", vps.Name)
	message := fmt.Sprintf("Your VPS instance '%s' has been stopped.", vps.Name)
	
	metadata := map[string]string{
		"vps_status": fmt.Sprintf("%d", vps.Status),
	}

	s.notifyVPSEvent(ctx, vps, "vps_stopped", notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_MEDIUM, title, message, metadata)
}

// notifyVPSRebooted sends a notification when a VPS is rebooted
func (s *Service) notifyVPSRebooted(ctx context.Context, vps *database.VPSInstance) {
	title := fmt.Sprintf("VPS Rebooted: %s", vps.Name)
	message := fmt.Sprintf("Your VPS instance '%s' has been rebooted.", vps.Name)
	
	metadata := map[string]string{
		"vps_status": fmt.Sprintf("%d", vps.Status),
	}

	s.notifyVPSEvent(ctx, vps, "vps_rebooted", notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_LOW, title, message, metadata)
}

// notifyVPSFailed sends a notification when a VPS fails
func (s *Service) notifyVPSFailed(ctx context.Context, vps *database.VPSInstance, reason string) {
	title := fmt.Sprintf("VPS Failed: %s", vps.Name)
	message := fmt.Sprintf("Your VPS instance '%s' has failed. %s", vps.Name, reason)
	
	metadata := map[string]string{
		"vps_status": fmt.Sprintf("%d", vps.Status),
		"failure_reason": reason,
	}

	s.notifyVPSEvent(ctx, vps, "vps_failed", notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_HIGH, title, message, metadata)
}

// notifyVPSDeletedFromProxmox sends a notification when a VPS is detected as deleted from Proxmox
func (s *Service) notifyVPSDeletedFromProxmox(ctx context.Context, vps *database.VPSInstance) {
	title := fmt.Sprintf("VPS Removed: %s", vps.Name)
	message := fmt.Sprintf("Your VPS instance '%s' was detected as deleted from Proxmox. It has been marked as deleted in the system.", vps.Name)
	
	metadata := map[string]string{
		"vps_status": fmt.Sprintf("%d", vps.Status),
		"deletion_source": "proxmox",
	}

	s.notifyVPSEvent(ctx, vps, "vps_deleted_from_proxmox", notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_HIGH, title, message, metadata)
}

// handleVPSStatusChange sends notifications when VPS status changes
func (s *Service) handleVPSStatusChange(ctx context.Context, vps *database.VPSInstance, oldStatus, newStatus int32) {
	// Only send notifications if status actually changed
	if oldStatus == newStatus {
		return
	}

	// Use VPSStatus enum values from proto
	creatingStatus := int32(vpsv1.VPSStatus_CREATING)
	startingStatus := int32(vpsv1.VPSStatus_STARTING)
	runningStatus := int32(vpsv1.VPSStatus_RUNNING)
	failedStatus := int32(vpsv1.VPSStatus_FAILED)
	deletedStatus := int32(vpsv1.VPSStatus_DELETED)

	// Handle status transitions
	switch newStatus {
	case runningStatus:
		// Only notify if transitioning from CREATING or STARTING to RUNNING (VPS is ready)
		if oldStatus == creatingStatus || oldStatus == startingStatus {
			s.notifyVPSReady(ctx, vps)
		}
	case failedStatus:
		// Notify when VPS fails
		reason := "The VPS provisioning or operation failed."
		s.notifyVPSFailed(ctx, vps, reason)
	case deletedStatus:
		// Notify when VPS is marked as deleted (from Proxmox)
		if oldStatus != deletedStatus {
			s.notifyVPSDeletedFromProxmox(ctx, vps)
		}
	}
}

