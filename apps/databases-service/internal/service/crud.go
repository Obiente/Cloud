package databases

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"

	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"

	"connectrpc.com/connect"
)

// Helper function to resolve user's default organization ID
func resolveUserDefaultOrgID(ctx context.Context) (string, bool) {
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil || userInfo == nil {
		return "", false
	}
	type row struct{ OrganizationID string }
	var r row
	if err := database.DB.Raw(`
        SELECT m.organization_id
        FROM organization_members m
        JOIN organizations o ON o.id = m.organization_id
        WHERE m.user_id = ?
        ORDER BY o.created_at DESC
        LIMIT 1
    `, userInfo.Id).Scan(&r).Error; err != nil {
		return "", false
	}
	if r.OrganizationID == "" {
		return "", false
	}
	return r.OrganizationID, true
}

// ListDatabases lists all databases for an organization
func (s *Service) ListDatabases(ctx context.Context, req *connect.Request[databasesv1.ListDatabasesRequest]) (*connect.Response[databasesv1.ListDatabasesResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	// Check organization permission
	if err := s.checkOrganizationPermission(ctx, orgID); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Create filters
	filters := &database.DatabaseFilters{}

	// Add status filter if provided
	if req.Msg.Status != nil {
		statusVal := int32(*req.Msg.Status)
		filters.Status = &statusVal
	}

	// Add type filter if provided
	if req.Msg.Type != nil {
		typeVal := int32(*req.Msg.Type)
		filters.Type = &typeVal
	}

	// Pagination
	page := int32(1)
	if req.Msg.Page > 0 {
		page = req.Msg.Page
	}
	perPage := int32(50)
	if req.Msg.PerPage > 0 {
		perPage = req.Msg.PerPage
	}
	filters.Offset = int64((page - 1) * perPage)
	filters.Limit = int64(perPage)

	// Get databases
	dbDatabases, err := s.repo.GetAll(ctx, orgID, filters)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list databases: %w", err))
	}

	// Convert DB models to proto models
	items := make([]*databasesv1.DatabaseInstance, 0, len(dbDatabases))
	for _, dbDB := range dbDatabases {
		database := dbDatabaseToProto(dbDB)
		items = append(items, database)
	}

	// Calculate total count for pagination
	var totalCount int64
	database.DB.WithContext(ctx).Model(&database.DatabaseInstance{}).
		Where("organization_id = ? AND deleted_at IS NULL", orgID).
		Count(&totalCount)

	res := connect.NewResponse(&databasesv1.ListDatabasesResponse{
		Databases: items,
		Pagination: &commonv1.Pagination{
			Page:    page,
			PerPage: perPage,
			Total:   int32(totalCount),
		},
	})
	return res, nil
}

// CreateDatabase creates a new database instance
func (s *Service) CreateDatabase(ctx context.Context, req *connect.Request[databasesv1.CreateDatabaseRequest]) (*connect.Response[databasesv1.CreateDatabaseResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	// Permission: org-level
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{Permission: auth.PermissionDatabaseCreate}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user authentication required: %w", err))
	}

	id := fmt.Sprintf("db-%d", time.Now().UnixNano())

	// Parse metadata
	metadataJSON := "{}"
	if len(req.Msg.Metadata) > 0 {
		metadataBytes, err := json.Marshal(req.Msg.Metadata)
		if err == nil {
			metadataJSON = string(metadataBytes)
		}
	}

	// Get database size specs (defaults)
	cpuCores := int32(1)
	memoryBytes := int64(2147483648) // 2GB default
	diskBytes := int64(10737418240)  // 10GB default
	maxConnections := int64(100)

	// TODO: Look up size from catalog
	size := req.Msg.GetSize()
	if size == "" {
		size = "small"
	}

	// Create database instance
	dbInstance := &database.DatabaseInstance{
		ID:             id,
		Name:           req.Msg.GetName(),
		Description:    req.Msg.Description,
		Status:         1, // CREATING
		Type:           int32(req.Msg.GetType()),
		Version:        req.Msg.Version,
		Size:           size,
		CPUCores:       cpuCores,
		MemoryBytes:    memoryBytes,
		DiskBytes:      diskBytes,
		MaxConnections: maxConnections,
		Metadata:       metadataJSON,
		OrganizationID: orgID,
		CreatedBy:      userInfo.Id,
	}

	if err := s.repo.Create(ctx, dbInstance); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create database: %w", err))
	}

	// Generate connection credentials
	initialDBName := req.Msg.GetInitialDatabaseName()
	if initialDBName == "" {
		initialDBName = "default"
	}
	initialUsername := req.Msg.GetInitialUsername()
	if initialUsername == "" {
		initialUsername = "admin"
	}
	initialPassword := req.Msg.GetInitialPassword()
	if initialPassword == "" {
		initialPassword = generateRandomPassword(32)
	}

	// TODO: Actually provision the database container/service
	// For now, we'll just create the connection record
	host := fmt.Sprintf("db-%s.internal", id)
	port := int32(5432) // Default PostgreSQL port
	if req.Msg.GetType() == databasesv1.DatabaseType_MYSQL || req.Msg.GetType() == databasesv1.DatabaseType_MARIADB {
		port = 3306
	} else if req.Msg.GetType() == databasesv1.DatabaseType_MONGODB {
		port = 27017
	} else if req.Msg.GetType() == databasesv1.DatabaseType_REDIS {
		port = 6379
	}

	connID := fmt.Sprintf("conn-%d", time.Now().UnixNano())
	dbConn := &database.DatabaseConnection{
		ID:           connID,
		DatabaseID:   id,
		DatabaseName: initialDBName,
		Username:     initialUsername,
		Password:     initialPassword, // TODO: Encrypt this
		Host:         host,
		Port:         port,
		SSLRequired:  true,
	}

	if err := s.connRepo.Create(ctx, dbConn); err != nil {
		logger.Warn("Failed to create connection record: %v", err)
		// Continue anyway
	}

	// Update database instance with connection info
	dbInstance.Host = &host
	dbInstance.Port = &port
	s.repo.Update(ctx, dbInstance)

	// Convert to proto
	protoDB := dbDatabaseToProto(dbInstance)
	connInfo := dbConnectionToProto(dbConn, id)

	res := connect.NewResponse(&databasesv1.CreateDatabaseResponse{
		Database:       protoDB,
		ConnectionInfo: connInfo,
	})
	return res, nil
}

// GetDatabase gets a database instance by ID
func (s *Service) GetDatabase(ctx context.Context, req *connect.Request[databasesv1.GetDatabaseRequest]) (*connect.Response[databasesv1.GetDatabaseResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	// Check organization permission
	if err := s.checkOrganizationPermission(ctx, orgID); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Get database
	dbInstance, err := s.repo.GetByID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found: %w", err))
	}

	// Verify organization ownership
	if dbInstance.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}

	// Check resource-level permission
	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseRead); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	protoDB := dbDatabaseToProto(dbInstance)

	res := connect.NewResponse(&databasesv1.GetDatabaseResponse{
		Database: protoDB,
	})
	return res, nil
}

// UpdateDatabase updates a database instance
func (s *Service) UpdateDatabase(ctx context.Context, req *connect.Request[databasesv1.UpdateDatabaseRequest]) (*connect.Response[databasesv1.UpdateDatabaseResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	// Check resource-level permission
	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseUpdate); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Get existing database
	dbInstance, err := s.repo.GetByID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found: %w", err))
	}

	// Verify organization ownership
	if dbInstance.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}

	// Update fields
	if req.Msg.Name != nil {
		dbInstance.Name = *req.Msg.Name
	}
	if req.Msg.Description != nil {
		dbInstance.Description = req.Msg.Description
	}

	// Update metadata
	if len(req.Msg.Metadata) > 0 {
		metadataBytes, err := json.Marshal(req.Msg.Metadata)
		if err == nil {
			dbInstance.Metadata = string(metadataBytes)
		}
	}

	if err := s.repo.Update(ctx, dbInstance); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update database: %w", err))
	}

	protoDB := dbDatabaseToProto(dbInstance)

	res := connect.NewResponse(&databasesv1.UpdateDatabaseResponse{
		Database: protoDB,
	})
	return res, nil
}

// DeleteDatabase deletes a database instance
func (s *Service) DeleteDatabase(ctx context.Context, req *connect.Request[databasesv1.DeleteDatabaseRequest]) (*connect.Response[databasesv1.DeleteDatabaseResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	// Check resource-level permission
	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseDelete); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Get existing database
	dbInstance, err := s.repo.GetByID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found: %w", err))
	}

	// Verify organization ownership
	if dbInstance.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}

	// TODO: Stop and delete the actual database container/service
	// For now, just soft delete the record
	if err := s.repo.Delete(ctx, req.Msg.GetDatabaseId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete database: %w", err))
	}

	res := connect.NewResponse(&databasesv1.DeleteDatabaseResponse{
		Success: true,
	})
	return res, nil
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

