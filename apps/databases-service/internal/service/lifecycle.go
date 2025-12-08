package databases

import (
	"context"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"

	databasesv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/databases/v1"

	"connectrpc.com/connect"
)

// StartDatabase starts a stopped database instance
func (s *Service) StartDatabase(ctx context.Context, req *connect.Request[databasesv1.StartDatabaseRequest]) (*connect.Response[databasesv1.StartDatabaseResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	// Check resource-level permission
	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseStart); err != nil {
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

	// TODO: Actually start the database container/service
	// For now, just update status
	dbInstance.Status = 2 // STARTING
	if err := s.repo.Update(ctx, dbInstance); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start database: %w", err))
	}

	// Simulate async start
	go func() {
		time.Sleep(5 * time.Second) // Simulate startup time
		dbInstance.Status = 3       // RUNNING
		dbInstance.LastStartedAt = timePtr(time.Now())
		s.repo.Update(context.Background(), dbInstance)
	}()

	protoDB := dbDatabaseToProto(dbInstance)

	res := connect.NewResponse(&databasesv1.StartDatabaseResponse{
		Database: protoDB,
	})
	return res, nil
}

// StopDatabase stops a running database instance
func (s *Service) StopDatabase(ctx context.Context, req *connect.Request[databasesv1.StopDatabaseRequest]) (*connect.Response[databasesv1.StopDatabaseResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	// Check resource-level permission
	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseStop); err != nil {
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

	// TODO: Actually stop the database container/service
	dbInstance.Status = 4 // STOPPING
	if err := s.repo.Update(ctx, dbInstance); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stop database: %w", err))
	}

	// Simulate async stop
	go func() {
		time.Sleep(2 * time.Second) // Simulate shutdown time
		dbInstance.Status = 5       // STOPPED
		s.repo.Update(context.Background(), dbInstance)
	}()

	protoDB := dbDatabaseToProto(dbInstance)

	res := connect.NewResponse(&databasesv1.StopDatabaseResponse{
		Database: protoDB,
	})
	return res, nil
}

// RestartDatabase restarts a database instance
func (s *Service) RestartDatabase(ctx context.Context, req *connect.Request[databasesv1.RestartDatabaseRequest]) (*connect.Response[databasesv1.RestartDatabaseResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	// Check resource-level permission
	if err := s.checkDatabasePermission(ctx, req.Msg.GetDatabaseId(), auth.PermissionDatabaseRestart); err != nil {
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

	// TODO: Actually restart the database container/service
	dbInstance.Status = 6 // REBOOTING (using similar status)
	if err := s.repo.Update(ctx, dbInstance); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to restart database: %w", err))
	}

	// Simulate async restart
	go func() {
		time.Sleep(3 * time.Second) // Simulate restart time
		dbInstance.Status = 3       // RUNNING
		dbInstance.LastStartedAt = timePtr(time.Now())
		s.repo.Update(context.Background(), dbInstance)
	}()

	protoDB := dbDatabaseToProto(dbInstance)

	res := connect.NewResponse(&databasesv1.RestartDatabaseResponse{
		Database: protoDB,
	})
	return res, nil
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}

