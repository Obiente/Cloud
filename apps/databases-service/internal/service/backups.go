package databases

import (
	"context"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"

	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"

	"connectrpc.com/connect"
)

// ListBackups lists all backups for a database
func (s *Service) ListBackups(ctx context.Context, req *connect.Request[databasesv1.ListBackupsRequest]) (*connect.Response[databasesv1.ListBackupsResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	// Check resource-level permission
	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseRead); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Get database to verify ownership
	dbInstance, err := s.repo.GetByID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found: %w", err))
	}

	if dbInstance.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}

	// Get backups
	page := int32(1)
	if req.Msg.Page > 0 {
		page = req.Msg.Page
	}
	perPage := int32(50)
	if req.Msg.PerPage > 0 {
		perPage = req.Msg.PerPage
	}

	filters := &database.BackupFilters{
		Limit:  int64(perPage),
		Offset: int64((page - 1) * perPage),
	}

	backups, err := s.backupRepo.GetAll(ctx, req.Msg.GetDatabaseId(), filters)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list backups: %w", err))
	}

	// Convert to proto
	protoBackups := make([]*databasesv1.DatabaseBackup, 0, len(backups))
	for _, backup := range backups {
		protoBackups = append(protoBackups, dbBackupToProto(backup))
	}

	// Count total
	var totalCount int64
	database.DB.WithContext(ctx).Model(&database.DatabaseBackup{}).
		Where("database_id = ?", req.Msg.GetDatabaseId()).
		Count(&totalCount)

	res := connect.NewResponse(&databasesv1.ListBackupsResponse{
		Backups: protoBackups,
		Pagination: &commonv1.Pagination{
			Page:    page,
			PerPage: perPage,
			Total:   int32(totalCount),
		},
	})
	return res, nil
}

// CreateBackup creates a new backup for a database
func (s *Service) CreateBackup(ctx context.Context, req *connect.Request[databasesv1.CreateBackupRequest]) (*connect.Response[databasesv1.CreateBackupResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user authentication required: %w", err))
	}

	// Check resource-level permission
	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseUpdate); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Get database to verify ownership
	dbInstance, err := s.repo.GetByID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found: %w", err))
	}

	if dbInstance.OrganizationID != orgID {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found"))
	}

	// Create backup record
	backupID := fmt.Sprintf("backup-%d", time.Now().UnixNano())
	backupName := req.Msg.GetName()
	if backupName == "" {
		backupName = fmt.Sprintf("backup-%s", time.Now().Format("20060102-150405"))
	}

	backup := &database.DatabaseBackup{
		ID:             backupID,
		DatabaseID:     req.Msg.GetDatabaseId(),
		Name:           backupName,
		Description:    req.Msg.Description,
		Status:         1, // CREATING
		OrganizationID: orgID,
		CreatedBy:      userInfo.Id,
	}

	if err := s.backupRepo.Create(ctx, backup); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create backup: %w", err))
	}

	// TODO: Actually perform the backup operation
	// For now, simulate async backup
	go func() {
		time.Sleep(5 * time.Second) // Simulate backup time
		now := time.Now()
		backup.Status = 2 // COMPLETED
		backup.CompletedAt = &now
		backup.SizeBytes = 1024 * 1024 * 100 // 100MB placeholder
		s.backupRepo.Update(context.Background(), backup)
	}()

	res := connect.NewResponse(&databasesv1.CreateBackupResponse{
		Backup: dbBackupToProto(backup),
	})
	return res, nil
}

// GetBackup gets a backup by ID
func (s *Service) GetBackup(ctx context.Context, req *connect.Request[databasesv1.GetBackupRequest]) (*connect.Response[databasesv1.GetBackupResponse], error) {
	// Check resource-level permission
	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseRead); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	backup, err := s.backupRepo.GetByID(ctx, req.Msg.GetBackupId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("backup not found: %w", err))
	}

	// Verify database ownership
	dbInstance, err := s.repo.GetByID(ctx, req.Msg.GetDatabaseId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("database not found: %w", err))
	}

	if backup.DatabaseID != req.Msg.GetDatabaseId() || dbInstance.OrganizationID != req.Msg.GetOrganizationId() {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("backup not found"))
	}

	res := connect.NewResponse(&databasesv1.GetBackupResponse{
		Backup: dbBackupToProto(backup),
	})
	return res, nil
}

// DeleteBackup deletes a backup
func (s *Service) DeleteBackup(ctx context.Context, req *connect.Request[databasesv1.DeleteBackupRequest]) (*connect.Response[databasesv1.DeleteBackupResponse], error) {
	// Check resource-level permission
	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseDelete); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	backup, err := s.backupRepo.GetByID(ctx, req.Msg.GetBackupId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("backup not found: %w", err))
	}

	// Verify database ownership
	if backup.DatabaseID != req.Msg.GetDatabaseId() {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("backup not found"))
	}

	if err := s.backupRepo.Delete(ctx, req.Msg.GetBackupId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete backup: %w", err))
	}

	res := connect.NewResponse(&databasesv1.DeleteBackupResponse{
		Success: true,
	})
	return res, nil
}

// RestoreBackup restores a backup (placeholder)
func (s *Service) RestoreBackup(ctx context.Context, req *connect.Request[databasesv1.RestoreBackupRequest]) (*connect.Response[databasesv1.RestoreBackupResponse], error) {
	// Placeholder implementation
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("backup restoration not yet implemented"))
}

