package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"

	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	notificationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1"
	notificationsv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1/notificationsv1connect"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type Service struct {
	notificationsv1connect.UnimplementedNotificationServiceHandler
}

func NewService() notificationsv1connect.NotificationServiceHandler {
	return &Service{}
}

func (s *Service) ListNotifications(ctx context.Context, req *connect.Request[notificationsv1.ListNotificationsRequest]) (*connect.Response[notificationsv1.ListNotificationsResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	// Build query
	query := database.DB.Where("user_id = ? AND client_only = ?", user.Id, false)

	// Apply filters
	if req.Msg.GetUnreadOnly() {
		query = query.Where("read = ?", false)
	}
	if req.Msg.Type != nil {
		typeStr := notificationTypeToString(*req.Msg.Type)
		if typeStr != "" {
			query = query.Where("type = ?", typeStr)
		}
	}
	if req.Msg.Severity != nil {
		severityStr := notificationSeverityToString(*req.Msg.Severity)
		if severityStr != "" {
			query = query.Where("severity = ?", severityStr)
		}
	}

	// Pagination
	page := int(req.Msg.GetPage())
	if page < 1 {
		page = 1
	}
	perPage := int(req.Msg.GetPerPage())
	if perPage < 1 {
		perPage = 50
	}
	if perPage > 100 {
		perPage = 100
	}
	offset := (page - 1) * perPage

	// Count total
	var total int64
	if err := query.Model(&database.Notification{}).Count(&total).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("count notifications: %w", err))
	}

	// Fetch notifications
	var notifications []database.Notification
	if err := query.Order("created_at DESC").Limit(perPage).Offset(offset).Find(&notifications).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list notifications: %w", err))
	}

	// Convert to proto
	protoNotifications := make([]*notificationsv1.Notification, 0, len(notifications))
	for _, n := range notifications {
		protoNotifications = append(protoNotifications, notificationToProto(&n))
	}

	totalPages := (int(total) + perPage - 1) / perPage

	return connect.NewResponse(&notificationsv1.ListNotificationsResponse{
		Notifications: protoNotifications,
		Pagination: &commonv1.Pagination{
			Page:       int32(page),
			PerPage:    int32(perPage),
			Total:      int32(total),
			TotalPages: int32(totalPages),
		},
	}), nil
}

func (s *Service) GetNotification(ctx context.Context, req *connect.Request[notificationsv1.GetNotificationRequest]) (*connect.Response[notificationsv1.GetNotificationResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	var notification database.Notification
	if err := database.DB.Where("id = ? AND user_id = ?", req.Msg.GetNotificationId(), user.Id).First(&notification).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("notification not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get notification: %w", err))
	}

	return connect.NewResponse(&notificationsv1.GetNotificationResponse{
		Notification: notificationToProto(&notification),
	}), nil
}

func (s *Service) MarkAsRead(ctx context.Context, req *connect.Request[notificationsv1.MarkAsReadRequest]) (*connect.Response[notificationsv1.MarkAsReadResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	var notification database.Notification
	if err := database.DB.Where("id = ? AND user_id = ?", req.Msg.GetNotificationId(), user.Id).First(&notification).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("notification not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("mark as read: %w", err))
	}

	now := time.Now()
	notification.Read = true
	notification.ReadAt = &now
	if err := database.DB.Save(&notification).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("mark as read: %w", err))
	}

	return connect.NewResponse(&notificationsv1.MarkAsReadResponse{
		Notification: notificationToProto(&notification),
	}), nil
}

func (s *Service) MarkAllAsRead(ctx context.Context, req *connect.Request[notificationsv1.MarkAllAsReadRequest]) (*connect.Response[notificationsv1.MarkAllAsReadResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	query := database.DB.Model(&database.Notification{}).Where("user_id = ? AND read = ?", user.Id, false)

	// Apply filters
	if req.Msg.Type != nil {
		typeStr := notificationTypeToString(*req.Msg.Type)
		if typeStr != "" {
			query = query.Where("type = ?", typeStr)
		}
	}
	if req.Msg.Severity != nil {
		severityStr := notificationSeverityToString(*req.Msg.Severity)
		if severityStr != "" {
			query = query.Where("severity = ?", severityStr)
		}
	}

	now := time.Now()
	result := query.Update("read", true).Update("read_at", now)
	if result.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("mark all as read: %w", result.Error))
	}

	return connect.NewResponse(&notificationsv1.MarkAllAsReadResponse{
		MarkedCount: int32(result.RowsAffected),
	}), nil
}

func (s *Service) DeleteNotification(ctx context.Context, req *connect.Request[notificationsv1.DeleteNotificationRequest]) (*connect.Response[notificationsv1.DeleteNotificationResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	result := database.DB.Where("id = ? AND user_id = ?", req.Msg.GetNotificationId(), user.Id).Delete(&database.Notification{})
	if result.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete notification: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("notification not found"))
	}

	return connect.NewResponse(&notificationsv1.DeleteNotificationResponse{
		Success: true,
	}), nil
}

func (s *Service) DeleteAllNotifications(ctx context.Context, req *connect.Request[notificationsv1.DeleteAllNotificationsRequest]) (*connect.Response[notificationsv1.DeleteAllNotificationsResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	query := database.DB.Where("user_id = ?", user.Id)

	// Apply filters
	if req.Msg.GetReadOnly() {
		query = query.Where("read = ?", true)
	}
	if req.Msg.Type != nil {
		typeStr := notificationTypeToString(*req.Msg.Type)
		if typeStr != "" {
			query = query.Where("type = ?", typeStr)
		}
	}

	result := query.Delete(&database.Notification{})
	if result.Error != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete all notifications: %w", result.Error))
	}

	return connect.NewResponse(&notificationsv1.DeleteAllNotificationsResponse{
		DeletedCount: int32(result.RowsAffected),
	}), nil
}

func (s *Service) GetUnreadCount(ctx context.Context, req *connect.Request[notificationsv1.GetUnreadCountRequest]) (*connect.Response[notificationsv1.GetUnreadCountResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	query := database.DB.Model(&database.Notification{}).Where("user_id = ? AND read = ? AND client_only = ?", user.Id, false, false)

	// Apply filters
	if req.Msg.Type != nil {
		typeStr := notificationTypeToString(*req.Msg.Type)
		if typeStr != "" {
			query = query.Where("type = ?", typeStr)
		}
	}
	if req.Msg.MinSeverity != nil {
		severityStr := notificationSeverityToString(*req.Msg.MinSeverity)
		if severityStr != "" {
			// Map severity to numeric values for comparison
			severityMap := map[string]int{
				"LOW":      1,
				"MEDIUM":   2,
				"HIGH":     3,
				"CRITICAL": 4,
			}
			minSeverityValue := severityMap[severityStr]
			// Use CASE statement for severity comparison
			query = query.Where(`CASE severity 
				WHEN 'LOW' THEN 1
				WHEN 'MEDIUM' THEN 2
				WHEN 'HIGH' THEN 3
				WHEN 'CRITICAL' THEN 4
				ELSE 0
			END >= ?`, minSeverityValue)
		}
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get unread count: %w", err))
	}

	return connect.NewResponse(&notificationsv1.GetUnreadCountResponse{
		Count: int32(count),
	}), nil
}

func (s *Service) CreateNotification(ctx context.Context, req *connect.Request[notificationsv1.CreateNotificationRequest]) (*connect.Response[notificationsv1.CreateNotificationResponse], error) {
	// This is an internal/admin endpoint - check for superadmin or internal service auth
	user, err := auth.GetUserFromContext(ctx)
	if err == nil {
		// If user is present, require superadmin
		if !auth.HasRole(user, auth.RoleSuperAdmin) {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
		}
	}
	// If no user (internal service call), allow it

	notification := &database.Notification{
		ID:             generateID("notif"),
		UserID:         req.Msg.GetUserId(),
		Type:           notificationTypeToString(req.Msg.GetType()),
		Severity:       notificationSeverityToString(req.Msg.GetSeverity()),
		Title:          req.Msg.GetTitle(),
		Message:        req.Msg.GetMessage(),
		Read:           false,
		ClientOnly:     false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if req.Msg.OrganizationId != nil {
		orgID := req.Msg.GetOrganizationId()
		notification.OrganizationID = &orgID
	}
	if req.Msg.ActionUrl != nil {
		actionURL := req.Msg.GetActionUrl()
		notification.ActionURL = &actionURL
	}
	if req.Msg.ActionLabel != nil {
		actionLabel := req.Msg.GetActionLabel()
		notification.ActionLabel = &actionLabel
	}

	// Convert metadata map to JSON
	if len(req.Msg.Metadata) > 0 {
		metadataJSON, err := json.Marshal(req.Msg.Metadata)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal metadata: %w", err))
		}
		notification.Metadata = string(metadataJSON)
	}

	if err := database.DB.Create(notification).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create notification: %w", err))
	}

	return connect.NewResponse(&notificationsv1.CreateNotificationResponse{
		Notification: notificationToProto(notification),
	}), nil
}

func (s *Service) CreateOrganizationNotification(ctx context.Context, req *connect.Request[notificationsv1.CreateOrganizationNotificationRequest]) (*connect.Response[notificationsv1.CreateOrganizationNotificationResponse], error) {
	// This is an internal/admin endpoint
	user, err := auth.GetUserFromContext(ctx)
	if err == nil {
		if !auth.HasRole(user, auth.RoleSuperAdmin) {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
		}
	}

	orgID := req.Msg.GetOrganizationId()

	// Get organization members
	var members []database.OrganizationMember
	query := database.DB.Where("organization_id = ? AND status = ?", orgID, "active")
	
	// Filter by roles if specified
	if len(req.Msg.Roles) > 0 {
		query = query.Where("role IN ?", req.Msg.Roles)
	}
	
	if err := query.Find(&members).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get organization members: %w", err))
	}

	if len(members) == 0 {
		return connect.NewResponse(&notificationsv1.CreateOrganizationNotificationResponse{
			CreatedCount:  0,
			Notifications: []*notificationsv1.Notification{},
		}), nil
	}

	// Create notifications for each member
	notifications := make([]*notificationsv1.Notification, 0, len(members))
	for _, member := range members {
		// Skip pending invites
		if member.UserID == "" || member.UserID[0:7] == "pending" {
			continue
		}

		notification := &database.Notification{
			ID:             generateID("notif"),
			UserID:         member.UserID,
			OrganizationID: &orgID,
			Type:           notificationTypeToString(req.Msg.GetType()),
			Severity:       notificationSeverityToString(req.Msg.GetSeverity()),
			Title:          req.Msg.GetTitle(),
			Message:        req.Msg.GetMessage(),
			Read:           false,
			ClientOnly:     false,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if req.Msg.ActionUrl != nil {
			actionURL := req.Msg.GetActionUrl()
			notification.ActionURL = &actionURL
		}
		if req.Msg.ActionLabel != nil {
			actionLabel := req.Msg.GetActionLabel()
			notification.ActionLabel = &actionLabel
		}

		// Convert metadata map to JSON
		if len(req.Msg.Metadata) > 0 {
			metadataJSON, err := json.Marshal(req.Msg.Metadata)
			if err != nil {
				logger.Warn("[Notifications] Failed to marshal metadata for notification: %v", err)
			} else {
				notification.Metadata = string(metadataJSON)
			}
		}

		if err := database.DB.Create(notification).Error; err != nil {
			logger.Warn("[Notifications] Failed to create notification for user %s: %v", member.UserID, err)
			continue
		}

		notifications = append(notifications, notificationToProto(notification))
	}

	return connect.NewResponse(&notificationsv1.CreateOrganizationNotificationResponse{
		CreatedCount:  int32(len(notifications)),
		Notifications: notifications,
	}), nil
}

// Helper functions

func generateID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func notificationToProto(n *database.Notification) *notificationsv1.Notification {
	proto := &notificationsv1.Notification{
		Id:         n.ID,
		UserId:     n.UserID,
		Type:       stringToNotificationType(n.Type),
		Severity:   stringToNotificationSeverity(n.Severity),
		Title:      n.Title,
		Message:    n.Message,
		Read:       n.Read,
		ClientOnly: n.ClientOnly,
		CreatedAt:  timestamppb.New(n.CreatedAt),
		UpdatedAt:  timestamppb.New(n.UpdatedAt),
	}

	if n.OrganizationID != nil {
		proto.OrganizationId = n.OrganizationID
	}
	if n.ReadAt != nil {
		proto.ReadAt = timestamppb.New(*n.ReadAt)
	}
	if n.ActionURL != nil {
		proto.ActionUrl = n.ActionURL
	}
	if n.ActionLabel != nil {
		proto.ActionLabel = n.ActionLabel
	}

	// Parse metadata JSON
	if n.Metadata != "" {
		var metadata map[string]string
		if err := json.Unmarshal([]byte(n.Metadata), &metadata); err == nil {
			proto.Metadata = metadata
		}
	}

	return proto
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

func stringToNotificationType(s string) notificationsv1.NotificationType {
	switch s {
	case "INFO":
		return notificationsv1.NotificationType_NOTIFICATION_TYPE_INFO
	case "SUCCESS":
		return notificationsv1.NotificationType_NOTIFICATION_TYPE_SUCCESS
	case "WARNING":
		return notificationsv1.NotificationType_NOTIFICATION_TYPE_WARNING
	case "ERROR":
		return notificationsv1.NotificationType_NOTIFICATION_TYPE_ERROR
	case "DEPLOYMENT":
		return notificationsv1.NotificationType_NOTIFICATION_TYPE_DEPLOYMENT
	case "BILLING":
		return notificationsv1.NotificationType_NOTIFICATION_TYPE_BILLING
	case "QUOTA":
		return notificationsv1.NotificationType_NOTIFICATION_TYPE_QUOTA
	case "INVITE":
		return notificationsv1.NotificationType_NOTIFICATION_TYPE_INVITE
	case "SYSTEM":
		return notificationsv1.NotificationType_NOTIFICATION_TYPE_SYSTEM
	default:
		return notificationsv1.NotificationType_NOTIFICATION_TYPE_INFO
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

func stringToNotificationSeverity(s string) notificationsv1.NotificationSeverity {
	switch s {
	case "LOW":
		return notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_LOW
	case "MEDIUM":
		return notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_MEDIUM
	case "HIGH":
		return notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_HIGH
	case "CRITICAL":
		return notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_CRITICAL
	default:
		return notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_MEDIUM
	}
}

// CreateNotificationForUser is a helper function that can be called from other services
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
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
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

// CreateNotificationForOrganization is a helper function that can be called from other services
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

