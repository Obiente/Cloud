package audit

import (
	"context"
	"fmt"

	auditv1 "api/gen/proto/obiente/cloud/audit/v1"
	auditv1connect "api/gen/proto/obiente/cloud/audit/v1/auditv1connect"
	"api/internal/auth"
	"api/internal/database"
	"api/internal/logger"
	"api/internal/services/organizations"

	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"gorm.io/gorm"
)

const (
	defaultPageSize = 50
	maxPageSize     = 1000
)

type Service struct {
	auditv1connect.UnimplementedAuditServiceHandler
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
	}
}

// ListAuditLogs lists audit logs with filtering options
func (s *Service) ListAuditLogs(ctx context.Context, req *connect.Request[auditv1.ListAuditLogsRequest]) (*connect.Response[auditv1.ListAuditLogsResponse], error) {
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}

	// Check if user has permission to view audit logs
	// Only superadmins and admins can view audit logs
	hasPermission := false
	for _, role := range userInfo.Roles {
		if role == auth.RoleSuperAdmin || role == auth.RoleAdmin {
			hasPermission = true
			break
		}
	}

	if !hasPermission {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied: only admins and superadmins can view audit logs"))
	}

	// Use MetricsDB (TimescaleDB) for querying audit logs - no fallback to main DB
	if database.MetricsDB == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("metrics database (TimescaleDB) not initialized - audit logs require TimescaleDB"))
	}

	// Determine page size first
	pageSize := defaultPageSize
	if req.Msg.PageSize != nil {
		pageSize = int(*req.Msg.PageSize)
		if pageSize > maxPageSize {
			pageSize = maxPageSize
		}
		if pageSize < 1 {
			pageSize = defaultPageSize
		}
	}

	// Build query
	query := database.MetricsDB.Model(&database.AuditLog{})

	// Apply filters
	// Note: If OrganizationId is not provided, all audit logs from all organizations are returned
	// This allows superadmins/admins to view global audit logs
	if req.Msg.OrganizationId != nil && *req.Msg.OrganizationId != "" {
		query = query.Where("organization_id = ?", *req.Msg.OrganizationId)
	}

	if req.Msg.ResourceType != nil && *req.Msg.ResourceType != "" {
		query = query.Where("resource_type = ?", *req.Msg.ResourceType)
	}

	if req.Msg.ResourceId != nil && *req.Msg.ResourceId != "" {
		query = query.Where("resource_id = ?", *req.Msg.ResourceId)
	}

	if req.Msg.UserId != nil && *req.Msg.UserId != "" {
		query = query.Where("user_id = ?", *req.Msg.UserId)
	}

	if req.Msg.Service != nil && *req.Msg.Service != "" {
		query = query.Where("service = ?", *req.Msg.Service)
	}

	if req.Msg.Action != nil && *req.Msg.Action != "" {
		query = query.Where("action = ?", *req.Msg.Action)
	}

	if req.Msg.StartTime != nil {
		query = query.Where("created_at >= ?", req.Msg.StartTime.AsTime())
	}

	if req.Msg.EndTime != nil {
		query = query.Where("created_at <= ?", req.Msg.EndTime.AsTime())
	}

	// Handle pagination
	if req.Msg.PageToken != nil && *req.Msg.PageToken != "" {
		// Parse page token (simple offset-based for now)
		var offset int
		if _, err := fmt.Sscanf(*req.Msg.PageToken, "offset:%d", &offset); err == nil {
			query = query.Offset(offset)
		}
	}

	// Order by created_at descending (newest first)
	query = query.Order("created_at DESC")

	// Limit results
	query = query.Limit(pageSize + 1) // Fetch one extra to determine if there's a next page

	// Execute query
	var auditLogs []database.AuditLog
	if err := query.Find(&auditLogs).Error; err != nil {
		logger.Error("[AuditService] Failed to query audit logs: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query audit logs: %w", err))
	}

	// Determine if there's a next page
	hasNextPage := len(auditLogs) > pageSize
	if hasNextPage {
		auditLogs = auditLogs[:pageSize]
	}

	// Convert to proto (resolve user info for each log)
	protoLogs := make([]*auditv1.AuditLogEntry, 0, len(auditLogs))
	for _, log := range auditLogs {
		protoLogs = append(protoLogs, dbAuditLogToProto(ctx, &log))
	}

	// Generate next page token
	nextPageToken := ""
	if hasNextPage {
		offset := pageSize
		if req.Msg.PageToken != nil && *req.Msg.PageToken != "" {
			var currentOffset int
			if _, err := fmt.Sscanf(*req.Msg.PageToken, "offset:%d", &currentOffset); err == nil {
				offset = currentOffset + pageSize
			}
		}
		nextPageToken = fmt.Sprintf("offset:%d", offset)
	}

	return connect.NewResponse(&auditv1.ListAuditLogsResponse{
		AuditLogs:     protoLogs,
		NextPageToken: nextPageToken,
	}), nil
}

// GetAuditLog gets a specific audit log entry by ID
func (s *Service) GetAuditLog(ctx context.Context, req *connect.Request[auditv1.GetAuditLogRequest]) (*connect.Response[auditv1.GetAuditLogResponse], error) {
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}

	// Check if user has permission to view audit logs
	hasPermission := false
	for _, role := range userInfo.Roles {
		if role == auth.RoleSuperAdmin || role == auth.RoleAdmin {
			hasPermission = true
			break
		}
	}

	if !hasPermission {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied: only admins and superadmins can view audit logs"))
	}

	// Use MetricsDB (TimescaleDB) for querying audit logs - no fallback to main DB
	if database.MetricsDB == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("metrics database (TimescaleDB) not initialized - audit logs require TimescaleDB"))
	}

	// Query audit log
	var auditLog database.AuditLog
	if err := database.MetricsDB.Where("id = ?", req.Msg.AuditLogId).First(&auditLog).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("audit log not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to query audit log: %w", err))
	}

	return connect.NewResponse(&auditv1.GetAuditLogResponse{
		AuditLog: dbAuditLogToProto(ctx, &auditLog),
	}), nil
}

// dbAuditLogToProto converts a database audit log to proto
// It resolves user information (name and email) using the user profile resolver
func dbAuditLogToProto(ctx context.Context, log *database.AuditLog) *auditv1.AuditLogEntry {
	entry := &auditv1.AuditLogEntry{
		Id:             log.ID,
		UserId:         log.UserID,
		Action:         log.Action,
		Service:        log.Service,
		IpAddress:      log.IPAddress,
		UserAgent:      log.UserAgent,
		RequestData:    log.RequestData,
		ResponseStatus: log.ResponseStatus,
		DurationMs:     log.DurationMs,
		CreatedAt:      timestamppb.New(log.CreatedAt),
	}

	if log.OrganizationID != nil {
		entry.OrganizationId = log.OrganizationID
	}

	if log.ResourceType != nil {
		entry.ResourceType = log.ResourceType
	}

	if log.ResourceID != nil {
		entry.ResourceId = log.ResourceID
	}

	if log.ErrorMessage != nil {
		entry.ErrorMessage = log.ErrorMessage
	}

	// Resolve user information (name and email) from user_id
	if log.UserID != "" && log.UserID != "system" {
		resolver := organizations.GetUserProfileResolver()
		if resolver != nil && resolver.IsConfigured() {
			if userProfile, err := resolver.Resolve(ctx, log.UserID); err == nil && userProfile != nil {
				if userProfile.Name != "" {
					name := userProfile.Name
					entry.UserName = &name
				}
				if userProfile.Email != "" {
					email := userProfile.Email
					entry.UserEmail = &email
				}
			}
		}
	}

	return entry
}
