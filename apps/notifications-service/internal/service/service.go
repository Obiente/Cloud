package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/email"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/services/organizations"

	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	notificationsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1"
	notificationsv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/notifications/v1/notificationsv1connect"

	notificationsauth "notifications-service/internal/auth"

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
	// This is an internal/admin endpoint - require either:
	// 1. Valid internal service secret (service-to-service call)
	// 2. Superadmin user (admin UI call)
	
	// Check if this is an authenticated internal service call
	internalAuth := notificationsauth.IsInternalServiceCall(ctx)
	
	// Check if user is authenticated
	user, err := auth.GetUserFromContext(ctx)
	userAuthenticated := err == nil && user != nil
	
	// Require either internal service auth OR superadmin user
	if !internalAuth && !userAuthenticated {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: internal service secret or user token"))
	}
	
	if userAuthenticated && !internalAuth {
		// User is present but not internal service call - require superadmin
		if !auth.HasRole(user, auth.RoleSuperAdmin) {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
		}
	}

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

	logger.Info("[Notifications] Created notification via RPC for user %s, type %s, severity %s: %s", req.Msg.GetUserId(), notificationTypeToString(req.Msg.GetType()), notificationSeverityToString(req.Msg.GetSeverity()), req.Msg.GetTitle())

	// Check user preferences and send email if enabled
	// Use background context to avoid cancellation when the original request completes
	go func() {
		emailCtx := context.Background()
		logger.Info("[Notifications] Starting email check for user %s, type %s", req.Msg.GetUserId(), notificationTypeToString(req.Msg.GetType()))
		if err := sendNotificationEmailIfEnabled(emailCtx, req.Msg.GetUserId(), req.Msg.GetType(), req.Msg.GetSeverity(), req.Msg.GetTitle(), req.Msg.GetMessage(), req.Msg.ActionUrl, req.Msg.ActionLabel); err != nil {
			logger.Warn("[Notifications] Failed to send email notification for user %s: %v", req.Msg.GetUserId(), err)
		}
	}()

	return connect.NewResponse(&notificationsv1.CreateNotificationResponse{
		Notification: notificationToProto(notification),
	}), nil
}

func (s *Service) CreateOrganizationNotification(ctx context.Context, req *connect.Request[notificationsv1.CreateOrganizationNotificationRequest]) (*connect.Response[notificationsv1.CreateOrganizationNotificationResponse], error) {
	// This is an internal/admin endpoint - require either:
	// 1. Valid internal service secret (service-to-service call)
	// 2. Superadmin user (admin UI call)
	
	// Check if this is an authenticated internal service call
	internalAuth := notificationsauth.IsInternalServiceCall(ctx)
	
	// Check if user is authenticated
	user, err := auth.GetUserFromContext(ctx)
	userAuthenticated := err == nil && user != nil
	
	// Require either internal service auth OR superadmin user
	if !internalAuth && !userAuthenticated {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: internal service secret or user token"))
	}
	
	if userAuthenticated && !internalAuth {
		// User is present but not internal service call - require superadmin
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

		logger.Info("[Notifications] Created notification for org member %s, type %s, severity %s: %s", member.UserID, notificationTypeToString(req.Msg.GetType()), notificationSeverityToString(req.Msg.GetSeverity()), req.Msg.GetTitle())

		// Check user preferences and send email if enabled
		// Use background context to avoid cancellation when the original request completes
		go func(userID string, notifType notificationsv1.NotificationType, notifSeverity notificationsv1.NotificationSeverity, notifTitle, notifMessage string, notifActionURL, notifActionLabel *string) {
			emailCtx := context.Background()
			logger.Info("[Notifications] Starting email check for user %s, type %s", userID, notificationTypeToString(notifType))
			if err := sendNotificationEmailIfEnabled(emailCtx, userID, notifType, notifSeverity, notifTitle, notifMessage, notifActionURL, notifActionLabel); err != nil {
				logger.Warn("[Notifications] Failed to send email notification for user %s: %v", userID, err)
			}
		}(member.UserID, req.Msg.GetType(), req.Msg.GetSeverity(), req.Msg.GetTitle(), req.Msg.GetMessage(), req.Msg.ActionUrl, req.Msg.ActionLabel)

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

	logger.Info("[Notifications] Created notification for user %s, type %s, severity %s: %s", userID, notificationTypeToString(notificationType), notificationSeverityToString(severity), title)

	// Check user preferences and send email if enabled
	// Use background context to avoid cancellation when the original request completes
	go func() {
		emailCtx := context.Background()
		logger.Info("[Notifications] Starting email check for user %s, type %s", userID, notificationTypeToString(notificationType))
		if err := sendNotificationEmailIfEnabled(emailCtx, userID, notificationType, severity, title, message, actionURL, actionLabel); err != nil {
			logger.Warn("[Notifications] Failed to send email notification for user %s: %v", userID, err)
		}
	}()

	return nil
}

// sendNotificationEmailIfEnabled checks user preferences and sends email if enabled
func sendNotificationEmailIfEnabled(ctx context.Context, userID string, notificationType notificationsv1.NotificationType, severity notificationsv1.NotificationSeverity, title, message string, actionURL, actionLabel *string) error {
	notificationTypeStr := notificationTypeToString(notificationType)
	
	logger.Info("[Notifications] Checking email preferences for user %s, type %s, severity %s", userID, notificationTypeStr, notificationSeverityToString(severity))
	
	// Get user preference for this notification type
	var preference database.NotificationPreference
	err := database.DB.Where("user_id = ? AND notification_type = ?", userID, notificationTypeStr).First(&preference).Error
	
	// If no preference found, use defaults based on notification type
	emailEnabled := false
	minSeverity := "LOW"
	frequency := "immediate"
	
	if err == gorm.ErrRecordNotFound {
		// Use defaults based on notification type (matching GetNotificationTypes defaults)
		switch notificationType {
		case notificationsv1.NotificationType_NOTIFICATION_TYPE_WARNING,
			notificationsv1.NotificationType_NOTIFICATION_TYPE_ERROR,
			notificationsv1.NotificationType_NOTIFICATION_TYPE_DEPLOYMENT,
			notificationsv1.NotificationType_NOTIFICATION_TYPE_BILLING,
			notificationsv1.NotificationType_NOTIFICATION_TYPE_QUOTA,
			notificationsv1.NotificationType_NOTIFICATION_TYPE_INVITE,
			notificationsv1.NotificationType_NOTIFICATION_TYPE_SYSTEM:
			emailEnabled = true
		case notificationsv1.NotificationType_NOTIFICATION_TYPE_INFO,
			notificationsv1.NotificationType_NOTIFICATION_TYPE_SUCCESS:
			emailEnabled = false
		}
		logger.Info("[Notifications] No preference found for user %s, type %s, using default emailEnabled=%v, minSeverity=%s, frequency=%s", userID, notificationTypeStr, emailEnabled, minSeverity, frequency)
	} else if err != nil {
		logger.Warn("[Notifications] Error checking preferences for user %s, type %s: %v", userID, notificationTypeStr, err)
		return fmt.Errorf("get preference: %w", err)
	} else {
		emailEnabled = preference.EmailEnabled
		minSeverity = preference.MinSeverity
		frequency = preference.Frequency
		logger.Info("[Notifications] Found preference for user %s, type %s: emailEnabled=%v, frequency=%s, minSeverity=%s", userID, notificationTypeStr, emailEnabled, frequency, minSeverity)
	}

	// Check if email is enabled and frequency is not "never"
	if !emailEnabled {
		logger.Info("[Notifications] Email disabled for user %s, type %s (emailEnabled=false)", userID, notificationTypeStr)
		return nil // Email not enabled for this user/type
	}
	
	if frequency == "never" {
		logger.Info("[Notifications] Email frequency is 'never' for user %s, type %s", userID, notificationTypeStr)
		return nil // Email frequency set to never
	}

	// Check severity threshold
	severityMap := map[string]int{
		"LOW":      1,
		"MEDIUM":   2,
		"HIGH":     3,
		"CRITICAL": 4,
	}
	notificationSeverityValue := severityMap[notificationSeverityToString(severity)]
	minSeverityValue := severityMap[minSeverity]
	if notificationSeverityValue < minSeverityValue {
		logger.Info("[Notifications] Severity below threshold for user %s, type %s (notification=%d, min=%d)", userID, notificationTypeStr, notificationSeverityValue, minSeverityValue)
		return nil // Severity below threshold
	}

	// Get user email from profile resolver
	resolver := organizations.GetUserProfileResolver()
	if resolver == nil || !resolver.IsConfigured() {
		return fmt.Errorf("user profile resolver not configured")
	}

	userProfile, err := resolver.Resolve(ctx, userID)
	if err != nil {
		return fmt.Errorf("resolve user profile: %w", err)
	}

	if userProfile.Email == "" {
		return fmt.Errorf("user has no email address")
	}

	// Initialize email sender
	mailer := email.NewSenderFromEnv()
	if !mailer.Enabled() {
		logger.Debug("[Notifications] Email sender not enabled (SMTP not configured)")
		return nil // Email not configured, not an error
	}

	// Get console URL for action links
	consoleURL := os.Getenv("DASHBOARD_URL")
	if consoleURL == "" {
		consoleURL = "https://cloud.obiente.com"
	}

	// Build action URL
	actionLink := consoleURL
	if actionURL != nil && *actionURL != "" {
		if (*actionURL)[0] == '/' {
			actionLink = consoleURL + *actionURL
		} else {
			actionLink = *actionURL
		}
	}

	// Determine email category based on notification type
	emailCategory := email.CategoryNotification
	switch notificationType {
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_BILLING:
		emailCategory = email.CategoryBilling
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_SYSTEM:
		emailCategory = email.CategorySystem
	case notificationsv1.NotificationType_NOTIFICATION_TYPE_INVITE:
		emailCategory = email.CategoryInvite
	}

	// Build email template
	greetingName := userProfile.Name
	if greetingName == "" {
		greetingName = userProfile.Email
	}

	template := email.TemplateData{
		Subject:     title,
		PreviewText: message,
		Greeting:    fmt.Sprintf("Hi %s,", greetingName),
		Heading:     title,
		IntroLines:  []string{message},
		Category:    emailCategory,
		BaseURL:     consoleURL,
		BrandURL:    consoleURL,
		SupportEmail: os.Getenv("SUPPORT_EMAIL"),
	}

	if actionURL != nil && actionLabel != nil {
		template.CTA = &email.CTA{
			Label: *actionLabel,
			URL:   actionLink,
		}
	} else if actionURL != nil {
		template.CTA = &email.CTA{
			Label: "View Details",
			URL:   actionLink,
		}
	}

	// Send email
	emailMsg := &email.Message{
		To:       []string{userProfile.Email},
		Template: &template,
		Category: emailCategory,
	}

	if err := mailer.Send(ctx, emailMsg); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	logger.Info("[Notifications] Sent email notification to %s for notification type %s", userProfile.Email, notificationTypeToString(notificationType))
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

func (s *Service) GetNotificationTypes(ctx context.Context, req *connect.Request[notificationsv1.GetNotificationTypesRequest]) (*connect.Response[notificationsv1.GetNotificationTypesResponse], error) {
	// Define notification types with their metadata
	types := []*notificationsv1.NotificationTypeInfo{
		{
			Type:                notificationsv1.NotificationType_NOTIFICATION_TYPE_INFO,
			Name:                "Info",
			Description:         "General informational notifications",
			DefaultEmailEnabled: false,
			DefaultInAppEnabled: true,
			DefaultMinSeverity:   notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_LOW,
		},
		{
			Type:                notificationsv1.NotificationType_NOTIFICATION_TYPE_SUCCESS,
			Name:                "Success",
			Description:         "Success and completion notifications",
			DefaultEmailEnabled: false,
			DefaultInAppEnabled: true,
			DefaultMinSeverity:   notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_LOW,
		},
		{
			Type:                notificationsv1.NotificationType_NOTIFICATION_TYPE_WARNING,
			Name:                "Warning",
			Description:         "Warning notifications that may require attention",
			DefaultEmailEnabled: true,
			DefaultInAppEnabled: true,
			DefaultMinSeverity:   notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_MEDIUM,
		},
		{
			Type:                notificationsv1.NotificationType_NOTIFICATION_TYPE_ERROR,
			Name:                "Error",
			Description:         "Error notifications that require immediate attention",
			DefaultEmailEnabled: true,
			DefaultInAppEnabled: true,
			DefaultMinSeverity:   notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_HIGH,
		},
		{
			Type:                notificationsv1.NotificationType_NOTIFICATION_TYPE_DEPLOYMENT,
			Name:                "Deployment",
			Description:         "Deployment status and updates",
			DefaultEmailEnabled: true,
			DefaultInAppEnabled: true,
			DefaultMinSeverity:   notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_MEDIUM,
		},
		{
			Type:                notificationsv1.NotificationType_NOTIFICATION_TYPE_BILLING,
			Name:                "Billing",
			Description:         "Billing and payment notifications",
			DefaultEmailEnabled: true,
			DefaultInAppEnabled: true,
			DefaultMinSeverity:   notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_MEDIUM,
		},
		{
			Type:                notificationsv1.NotificationType_NOTIFICATION_TYPE_QUOTA,
			Name:                "Quota",
			Description:         "Resource quota and limit notifications",
			DefaultEmailEnabled: true,
			DefaultInAppEnabled: true,
			DefaultMinSeverity:   notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_MEDIUM,
		},
		{
			Type:                notificationsv1.NotificationType_NOTIFICATION_TYPE_INVITE,
			Name:                "Invite",
			Description:         "Organization invitation notifications",
			DefaultEmailEnabled: true,
			DefaultInAppEnabled: true,
			DefaultMinSeverity:   notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_LOW,
		},
		{
			Type:                notificationsv1.NotificationType_NOTIFICATION_TYPE_SYSTEM,
			Name:                "System",
			Description:         "System and maintenance notifications",
			DefaultEmailEnabled: true,
			DefaultInAppEnabled: true,
			DefaultMinSeverity:   notificationsv1.NotificationSeverity_NOTIFICATION_SEVERITY_HIGH,
		},
	}

	return connect.NewResponse(&notificationsv1.GetNotificationTypesResponse{
		Types: types,
	}), nil
}

func (s *Service) GetNotificationPreferences(ctx context.Context, req *connect.Request[notificationsv1.GetNotificationPreferencesRequest]) (*connect.Response[notificationsv1.GetNotificationPreferencesResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	var preferences []database.NotificationPreference
	if err := database.DB.Where("user_id = ?", user.Id).Find(&preferences).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get preferences: %w", err))
	}

	protoPreferences := make([]*notificationsv1.NotificationPreference, 0, len(preferences))
	for _, p := range preferences {
		protoPreferences = append(protoPreferences, &notificationsv1.NotificationPreference{
			NotificationType: stringToNotificationType(p.NotificationType),
			EmailEnabled:     p.EmailEnabled,
			InAppEnabled:     p.InAppEnabled,
			Frequency:        stringToNotificationFrequency(p.Frequency),
			MinSeverity:      stringToNotificationSeverity(p.MinSeverity),
		})
	}

	return connect.NewResponse(&notificationsv1.GetNotificationPreferencesResponse{
		Preferences: protoPreferences,
	}), nil
}

func (s *Service) UpdateNotificationPreferences(ctx context.Context, req *connect.Request[notificationsv1.UpdateNotificationPreferencesRequest]) (*connect.Response[notificationsv1.UpdateNotificationPreferencesResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	// Process each preference update
	updatedPreferences := make([]*notificationsv1.NotificationPreference, 0, len(req.Msg.Preferences))
	for _, pref := range req.Msg.Preferences {
		notificationTypeStr := notificationTypeToString(pref.NotificationType)
		
		// Use a transaction to make check-and-create/update atomic
		var finalPreference database.NotificationPreference
		err := database.DB.Transaction(func(tx *gorm.DB) error {
			// Check if preference already exists
			var existing database.NotificationPreference
			err := tx.Where("user_id = ? AND notification_type = ?", user.Id, notificationTypeStr).First(&existing).Error
			
			if err == gorm.ErrRecordNotFound {
				// Create new preference with generated ID
				prefID := generateID("notif-pref")
				if prefID == "" {
					prefID = fmt.Sprintf("notif-pref-%d", time.Now().UnixNano())
				}
				preference := &database.NotificationPreference{
					ID:             prefID,
					UserID:         user.Id,
					NotificationType: notificationTypeStr,
					EmailEnabled:     pref.EmailEnabled,
					InAppEnabled:     pref.InAppEnabled,
					Frequency:        notificationFrequencyToString(pref.Frequency),
					MinSeverity:      notificationSeverityToString(pref.MinSeverity),
				}
				// Double-check ID is set (BeforeCreate hook should also set it, but ensure it here)
				if preference.ID == "" {
					preference.ID = fmt.Sprintf("notif-pref-%d", time.Now().UnixNano())
				}
				logger.Debug("[Notifications] Creating new preference with ID=%s for user=%s, type=%s", preference.ID, user.Id, notificationTypeStr)
				if err := tx.Create(preference).Error; err != nil {
					errStr := err.Error()
					// Check if it's a unique constraint violation (race condition)
					// Check for various error message formats
					isDuplicateKey := strings.Contains(errStr, "duplicate key") || 
						strings.Contains(errStr, "unique constraint") || 
						strings.Contains(errStr, "idx_user_type") ||
						strings.Contains(errStr, "23505") || // PostgreSQL error code for unique violation
						strings.Contains(errStr, "SQLSTATE 23505")
					
					if isDuplicateKey {
						// Race condition: record was created between our check and create
						// Try to update it instead
						logger.Debug("[Notifications] Race condition detected (error: %s), updating existing preference for user=%s, type=%s", errStr, user.Id, notificationTypeStr)
						var racePreference database.NotificationPreference
						if err := tx.Where("user_id = ? AND notification_type = ?", user.Id, notificationTypeStr).First(&racePreference).Error; err != nil {
							return fmt.Errorf("race condition: failed to find existing preference: %w", err)
						}
						updateData := map[string]interface{}{
							"email_enabled": pref.EmailEnabled,
							"in_app_enabled": pref.InAppEnabled,
							"frequency":      notificationFrequencyToString(pref.Frequency),
							"min_severity":    notificationSeverityToString(pref.MinSeverity),
						}
						if err := tx.Model(&database.NotificationPreference{}).
							Where("user_id = ? AND notification_type = ?", user.Id, notificationTypeStr).
							Updates(updateData).Error; err != nil {
							return fmt.Errorf("update preference after race condition: %w", err)
						}
						// Reload to get updated timestamps
						if err := tx.Where("user_id = ? AND notification_type = ?", user.Id, notificationTypeStr).First(&finalPreference).Error; err != nil {
							return fmt.Errorf("get updated preference after race condition: %w", err)
						}
					} else {
						logger.Error("[Notifications] Failed to create preference: ID=%s, user=%s, type=%s, error=%v", preference.ID, user.Id, notificationTypeStr, err)
						return fmt.Errorf("create preference: %w", err)
					}
				} else {
					// Reload to get all fields (timestamps, etc.)
					if err := tx.Where("user_id = ? AND notification_type = ?", user.Id, notificationTypeStr).First(&finalPreference).Error; err != nil {
						return fmt.Errorf("get created preference: %w", err)
					}
				}
			} else if err != nil {
				return fmt.Errorf("get preference: %w", err)
			} else {
				// Update existing preference
				// Use Updates() with explicit WHERE to ensure it's an UPDATE, not INSERT
				updateData := map[string]interface{}{
					"email_enabled": pref.EmailEnabled,
					"in_app_enabled": pref.InAppEnabled,
					"frequency":      notificationFrequencyToString(pref.Frequency),
					"min_severity":    notificationSeverityToString(pref.MinSeverity),
				}
				if err := tx.Model(&database.NotificationPreference{}).
					Where("user_id = ? AND notification_type = ?", user.Id, notificationTypeStr).
					Updates(updateData).Error; err != nil {
					return fmt.Errorf("update preference: %w", err)
				}
				// Reload to get updated timestamps
				if err := tx.Where("user_id = ? AND notification_type = ?", user.Id, notificationTypeStr).First(&finalPreference).Error; err != nil {
					return fmt.Errorf("get updated preference: %w", err)
				}
			}
			return nil
		})
		
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		
		updatedPreferences = append(updatedPreferences, &notificationsv1.NotificationPreference{
			NotificationType: pref.NotificationType,
			EmailEnabled:     finalPreference.EmailEnabled,
			InAppEnabled:     finalPreference.InAppEnabled,
			Frequency:        stringToNotificationFrequency(finalPreference.Frequency),
			MinSeverity:      stringToNotificationSeverity(finalPreference.MinSeverity),
		})
	}

	return connect.NewResponse(&notificationsv1.UpdateNotificationPreferencesResponse{
		Preferences: updatedPreferences,
	}), nil
}

func notificationFrequencyToString(f notificationsv1.NotificationFrequency) string {
	switch f {
	case notificationsv1.NotificationFrequency_NOTIFICATION_FREQUENCY_IMMEDIATE:
		return "immediate"
	case notificationsv1.NotificationFrequency_NOTIFICATION_FREQUENCY_DAILY:
		return "daily"
	case notificationsv1.NotificationFrequency_NOTIFICATION_FREQUENCY_WEEKLY:
		return "weekly"
	case notificationsv1.NotificationFrequency_NOTIFICATION_FREQUENCY_NEVER:
		return "never"
	default:
		return "immediate"
	}
}

func stringToNotificationFrequency(s string) notificationsv1.NotificationFrequency {
	switch s {
	case "immediate":
		return notificationsv1.NotificationFrequency_NOTIFICATION_FREQUENCY_IMMEDIATE
	case "daily":
		return notificationsv1.NotificationFrequency_NOTIFICATION_FREQUENCY_DAILY
	case "weekly":
		return notificationsv1.NotificationFrequency_NOTIFICATION_FREQUENCY_WEEKLY
	case "never":
		return notificationsv1.NotificationFrequency_NOTIFICATION_FREQUENCY_NEVER
	default:
		return notificationsv1.NotificationFrequency_NOTIFICATION_FREQUENCY_IMMEDIATE
	}
}

