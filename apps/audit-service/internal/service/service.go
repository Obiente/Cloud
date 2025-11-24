package audit

import (
	"context"
	"fmt"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/services/common"
	"github.com/obiente/cloud/apps/shared/pkg/services/organizations"

	auditv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/audit/v1"
	auditv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/audit/v1/auditv1connect"

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
	// Superadmins and global admins can view all audit logs
	// Org admins and owners can view audit logs for their organizations
	isSuperAdminOrAdmin := false
	for _, role := range userInfo.Roles {
		if role == auth.RoleSuperAdmin || role == auth.RoleAdmin {
			isSuperAdminOrAdmin = true
			break
		}
	}

	// If not superadmin/admin, check if user is org admin/owner for the requested organization
	if !isSuperAdminOrAdmin {
		// If OrganizationId is provided, verify the user is admin/owner of that org
		if req.Msg.OrganizationId != nil && *req.Msg.OrganizationId != "" {
			if err := common.AuthorizeOrgAdmin(ctx, *req.Msg.OrganizationId, userInfo); err != nil {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied: only organization admins and owners can view audit logs for their organization"))
			}
		} else {
			// If no OrganizationId provided, org admins/owners must specify one
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required for organization admins and owners"))
		}
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

	// Build base query for counting (without pagination)
	countQuery := database.MetricsDB.Model(&database.AuditLog{})

	// Build query for fetching (with pagination)
	query := database.MetricsDB.Model(&database.AuditLog{})

	// Apply filters to both queries
	// Note: If OrganizationId is not provided, all audit logs from all organizations are returned
	// This allows superadmins/admins to view global audit logs
	// Org admins/owners must provide OrganizationId (enforced above)
	if req.Msg.OrganizationId != nil && *req.Msg.OrganizationId != "" {
		countQuery = countQuery.Where("organization_id = ?", *req.Msg.OrganizationId)
		query = query.Where("organization_id = ?", *req.Msg.OrganizationId)
	}

	if req.Msg.ResourceType != nil && *req.Msg.ResourceType != "" {
		countQuery = countQuery.Where("resource_type = ?", *req.Msg.ResourceType)
		query = query.Where("resource_type = ?", *req.Msg.ResourceType)
	}

	if req.Msg.ResourceId != nil && *req.Msg.ResourceId != "" {
		countQuery = countQuery.Where("resource_id = ?", *req.Msg.ResourceId)
		query = query.Where("resource_id = ?", *req.Msg.ResourceId)
	}

	if req.Msg.UserId != nil && *req.Msg.UserId != "" {
		countQuery = countQuery.Where("user_id = ?", *req.Msg.UserId)
		query = query.Where("user_id = ?", *req.Msg.UserId)
	}

	if req.Msg.Service != nil && *req.Msg.Service != "" {
		countQuery = countQuery.Where("service = ?", *req.Msg.Service)
		query = query.Where("service = ?", *req.Msg.Service)
	}

	if req.Msg.Action != nil && *req.Msg.Action != "" {
		countQuery = countQuery.Where("action = ?", *req.Msg.Action)
		query = query.Where("action = ?", *req.Msg.Action)
	}

	if req.Msg.StartTime != nil {
		countQuery = countQuery.Where("created_at >= ?", req.Msg.StartTime.AsTime())
		query = query.Where("created_at >= ?", req.Msg.StartTime.AsTime())
	}

	if req.Msg.EndTime != nil {
		countQuery = countQuery.Where("created_at <= ?", req.Msg.EndTime.AsTime())
		query = query.Where("created_at <= ?", req.Msg.EndTime.AsTime())
	}

	// Count total matching records (before pagination)
	var totalCount int64
	if err := countQuery.Count(&totalCount).Error; err != nil {
		logger.Error("[AuditService] Failed to count audit logs: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to count audit logs: %w", err))
	}

	// Handle pagination (only for the fetch query)
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
		TotalCount:    totalCount,
	}), nil
}

// GetAuditLog gets a specific audit log entry by ID
func (s *Service) GetAuditLog(ctx context.Context, req *connect.Request[auditv1.GetAuditLogRequest]) (*connect.Response[auditv1.GetAuditLogResponse], error) {
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}

	// Check if user has permission to view audit logs
	// Superadmins and global admins can view all audit logs
	// Org admins and owners can view audit logs for their organizations
	isSuperAdminOrAdmin := false
	for _, role := range userInfo.Roles {
		if role == auth.RoleSuperAdmin || role == auth.RoleAdmin {
			isSuperAdminOrAdmin = true
			break
		}
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

	// If not superadmin/admin, verify the user is admin/owner of the audit log's organization
	if !isSuperAdminOrAdmin {
		if auditLog.OrganizationID == nil || *auditLog.OrganizationID == "" {
			// Audit log has no organization (global log), only superadmins/admins can view
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied: only admins and superadmins can view global audit logs"))
		}
		if err := common.AuthorizeOrgAdmin(ctx, *auditLog.OrganizationID, userInfo); err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied: only organization admins and owners can view audit logs for their organization"))
		}
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
