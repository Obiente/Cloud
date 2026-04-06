package audit

import (
	"context"
	"fmt"
	"sort"

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
	defaultPageSize  = 50
	maxPageSize      = 1000
	sshProxyService  = "SSHProxyService"
	sshConnectAction = "SSHConnect"
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

func applyGlobalAuditNoiseFilters(query *gorm.DB, req *auditv1.ListAuditLogsRequest) *gorm.DB {
	if query == nil || req == nil {
		return query
	}

	// Global audit logs are intentionally higher-signal than org/resource scoped views.
	// Failed SSH probes generate a large amount of unactionable noise in the global feed,
	// so hide them by default there while preserving them in organization-scoped audit logs.
	if req.OrganizationId == nil || *req.OrganizationId == "" {
		query = query.Where(
			"NOT (service = ? AND action = ? AND response_status <> ?)",
			sshProxyService,
			sshConnectAction,
			200,
		)
	}

	return query
}

func authorizeAuditLogRead(ctx context.Context, reqOrgID *string) error {
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}

	isSuperAdminOrAdmin := false
	for _, role := range userInfo.Roles {
		if role == auth.RoleSuperAdmin || role == auth.RoleAdmin {
			isSuperAdminOrAdmin = true
			break
		}
	}

	if isSuperAdminOrAdmin {
		return nil
	}

	if reqOrgID != nil && *reqOrgID != "" {
		if err := common.AuthorizeOrgAdmin(ctx, *reqOrgID, userInfo); err != nil {
			return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied: only organization admins and owners can view audit logs for their organization"))
		}
		return nil
	}

	return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required for organization admins and owners"))
}

func applyAuditFilters(query *gorm.DB, organizationID, resourceType, resourceID, userID, service, action *string, startTime, endTime *timestamppb.Timestamp, responseStatus *int32, errorStatuses *bool) *gorm.DB {
	if query == nil {
		return nil
	}

	if organizationID != nil && *organizationID != "" {
		query = query.Where("organization_id = ?", *organizationID)
	}
	if resourceType != nil && *resourceType != "" {
		query = query.Where("resource_type = ?", *resourceType)
	}
	if resourceID != nil && *resourceID != "" {
		query = query.Where("resource_id = ?", *resourceID)
	}
	if userID != nil && *userID != "" {
		query = query.Where("user_id = ?", *userID)
	}
	if service != nil && *service != "" {
		query = query.Where("service = ?", *service)
	}
	if action != nil && *action != "" {
		query = query.Where("action = ?", *action)
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", startTime.AsTime())
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", endTime.AsTime())
	}
	if responseStatus != nil {
		query = query.Where("response_status = ?", *responseStatus)
	}
	if errorStatuses != nil && *errorStatuses {
		query = query.Where("response_status >= ?", 400)
	}

	return query
}

func resolveAuditUserOption(ctx context.Context, userID string) *auditv1.AuditLogUserOption {
	option := &auditv1.AuditLogUserOption{UserId: userID}
	if userID == "" || userID == "system" {
		return option
	}

	resolver := organizations.GetUserProfileResolver()
	if resolver == nil || !resolver.IsConfigured() {
		return option
	}

	userProfile, err := resolver.Resolve(ctx, userID)
	if err != nil || userProfile == nil {
		return option
	}
	if userProfile.Name != "" {
		option.UserName = &userProfile.Name
	}
	if userProfile.Email != "" {
		option.UserEmail = &userProfile.Email
	}
	return option
}

// ListAuditLogs lists audit logs with filtering options
func (s *Service) ListAuditLogs(ctx context.Context, req *connect.Request[auditv1.ListAuditLogsRequest]) (*connect.Response[auditv1.ListAuditLogsResponse], error) {
	if err := authorizeAuditLogRead(ctx, req.Msg.OrganizationId); err != nil {
		return nil, err
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

	countQuery = applyGlobalAuditNoiseFilters(countQuery, req.Msg)
	query = applyGlobalAuditNoiseFilters(query, req.Msg)
	countQuery = applyAuditFilters(
		countQuery,
		req.Msg.OrganizationId,
		req.Msg.ResourceType,
		req.Msg.ResourceId,
		req.Msg.UserId,
		req.Msg.Service,
		req.Msg.Action,
		req.Msg.StartTime,
		req.Msg.EndTime,
		req.Msg.ResponseStatus,
		req.Msg.ErrorStatuses,
	)
	query = applyAuditFilters(
		query,
		req.Msg.OrganizationId,
		req.Msg.ResourceType,
		req.Msg.ResourceId,
		req.Msg.UserId,
		req.Msg.Service,
		req.Msg.Action,
		req.Msg.StartTime,
		req.Msg.EndTime,
		req.Msg.ResponseStatus,
		req.Msg.ErrorStatuses,
	)

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

func (s *Service) GetAuditLogFilterOptions(ctx context.Context, req *connect.Request[auditv1.GetAuditLogFilterOptionsRequest]) (*connect.Response[auditv1.GetAuditLogFilterOptionsResponse], error) {
	if err := authorizeAuditLogRead(ctx, req.Msg.OrganizationId); err != nil {
		return nil, err
	}

	// Use MetricsDB (TimescaleDB) for querying audit logs - no fallback to main DB
	if database.MetricsDB == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("metrics database (TimescaleDB) not initialized - audit logs require TimescaleDB"))
	}

	baseQuery := database.MetricsDB.Model(&database.AuditLog{})
	noiseReq := &auditv1.ListAuditLogsRequest{
		OrganizationId: req.Msg.OrganizationId,
	}
	baseQuery = applyGlobalAuditNoiseFilters(baseQuery, noiseReq)

	serviceQuery := applyAuditFilters(
		baseQuery.Session(&gorm.Session{}),
		req.Msg.OrganizationId,
		req.Msg.ResourceType,
		req.Msg.ResourceId,
		req.Msg.UserId,
		nil,
		req.Msg.Action,
		req.Msg.StartTime,
		req.Msg.EndTime,
		req.Msg.ResponseStatus,
		req.Msg.ErrorStatuses,
	)
	actionQuery := applyAuditFilters(
		baseQuery.Session(&gorm.Session{}),
		req.Msg.OrganizationId,
		req.Msg.ResourceType,
		req.Msg.ResourceId,
		req.Msg.UserId,
		req.Msg.Service,
		nil,
		req.Msg.StartTime,
		req.Msg.EndTime,
		req.Msg.ResponseStatus,
		req.Msg.ErrorStatuses,
	)
	userQuery := applyAuditFilters(
		baseQuery.Session(&gorm.Session{}),
		req.Msg.OrganizationId,
		req.Msg.ResourceType,
		req.Msg.ResourceId,
		nil,
		req.Msg.Service,
		req.Msg.Action,
		req.Msg.StartTime,
		req.Msg.EndTime,
		req.Msg.ResponseStatus,
		req.Msg.ErrorStatuses,
	)
	statusQuery := applyAuditFilters(
		baseQuery.Session(&gorm.Session{}),
		req.Msg.OrganizationId,
		req.Msg.ResourceType,
		req.Msg.ResourceId,
		req.Msg.UserId,
		req.Msg.Service,
		req.Msg.Action,
		req.Msg.StartTime,
		req.Msg.EndTime,
		nil,
		nil,
	)

	var services []string
	if err := serviceQuery.
		Distinct("service").
		Where("service <> ''").
		Order("service ASC").
		Pluck("service", &services).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load audit filter services: %w", err))
	}

	var actions []string
	if err := actionQuery.
		Distinct("action").
		Where("action <> ''").
		Order("action ASC").
		Pluck("action", &actions).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load audit filter actions: %w", err))
	}

	var userIDs []string
	if err := userQuery.
		Distinct("user_id").
		Where("user_id <> ''").
		Order("user_id ASC").
		Pluck("user_id", &userIDs).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load audit filter users: %w", err))
	}

	users := make([]*auditv1.AuditLogUserOption, 0, len(userIDs))
	for _, userID := range userIDs {
		users = append(users, resolveAuditUserOption(ctx, userID))
	}
	sort.Slice(users, func(i, j int) bool {
		left := users[i].GetUserName()
		if left == "" {
			left = users[i].GetUserEmail()
		}
		if left == "" {
			left = users[i].GetUserId()
		}
		right := users[j].GetUserName()
		if right == "" {
			right = users[j].GetUserEmail()
		}
		if right == "" {
			right = users[j].GetUserId()
		}
		return left < right
	})

	var responseStatuses []int32
	if err := statusQuery.
		Distinct("response_status").
		Order("response_status ASC").
		Pluck("response_status", &responseStatuses).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load audit filter statuses: %w", err))
	}

	return connect.NewResponse(&auditv1.GetAuditLogFilterOptionsResponse{
		Services:         services,
		Actions:          actions,
		Users:            users,
		ResponseStatuses: responseStatuses,
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
