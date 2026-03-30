package databases

import (
	"context"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/logger"

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

	// Update status to STARTING
	dbInstance.Status = 2 // STARTING
	if err := s.repo.Update(ctx, dbInstance); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start database: %w", err))
	}

	// Start the database container asynchronously
	go func() {
		startCtx, cancel := s.detachedContext(2 * time.Minute)
		defer cancel()

		if dbInstance.InstanceID != nil && *dbInstance.InstanceID != "" && s.provisioner != nil {
			if err := s.provisioner.StartDatabase(startCtx, *dbInstance.InstanceID); err != nil {
				logger.Error("Failed to start database container: %v", err)
				dbInstance.Status = 8 // FAILED
				s.repo.Update(startCtx, dbInstance)
				return
			}
		}

		// Update status to RUNNING
		dbInstance.Status = 3 // RUNNING
		dbInstance.LastStartedAt = timePtr(time.Now())
		if err := s.repo.Update(startCtx, dbInstance); err != nil {
			logger.Error("Failed to update database status: %v", err)
		}

		// Update route registry
		if s.routeRegistry != nil {
			containerName := fmt.Sprintf("obiente-%s", dbInstance.ID)
			s.routeRegistry.MarkRunning(dbInstance.ID, containerName)
		}
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

	// Update status to STOPPING
	dbInstance.Status = 4 // STOPPING
	if err := s.repo.Update(ctx, dbInstance); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stop database: %w", err))
	}

	// Stop the database container asynchronously
	go func() {
		stopCtx, cancel := s.detachedContext(2 * time.Minute)
		defer cancel()

		if dbInstance.InstanceID != nil && *dbInstance.InstanceID != "" && s.provisioner != nil {
			if err := s.provisioner.StopDatabase(stopCtx, *dbInstance.InstanceID); err != nil {
				logger.Error("Failed to stop database container: %v", err)
				dbInstance.Status = 8 // FAILED
				s.repo.Update(stopCtx, dbInstance)
				return
			}
		}

		// Update status to STOPPED
		dbInstance.Status = 5 // STOPPED
		if err := s.repo.Update(stopCtx, dbInstance); err != nil {
			logger.Error("Failed to update database status: %v", err)
		}

		// Update route registry - STOPPED means no auto-wake
		if s.routeRegistry != nil {
			s.routeRegistry.MarkStopped(dbInstance.ID, 5)
		}
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

	// Update status (using STARTING as an interim state)
	dbInstance.Status = 2 // STARTING
	if err := s.repo.Update(ctx, dbInstance); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to restart database: %w", err))
	}

	// Restart the database container asynchronously
	go func() {
		restartCtx, cancel := s.detachedContext(3 * time.Minute)
		defer cancel()

		if dbInstance.InstanceID != nil && *dbInstance.InstanceID != "" && s.provisioner != nil {
			if err := s.provisioner.RestartDatabase(restartCtx, *dbInstance.InstanceID); err != nil {
				logger.Error("Failed to restart database container: %v", err)
				dbInstance.Status = 8 // FAILED
				s.repo.Update(restartCtx, dbInstance)
				return
			}
		}

		// Update status to RUNNING
		dbInstance.Status = 3 // RUNNING
		dbInstance.LastStartedAt = timePtr(time.Now())
		if err := s.repo.Update(restartCtx, dbInstance); err != nil {
			logger.Error("Failed to update database status: %v", err)
		}
	}()

	protoDB := dbDatabaseToProto(dbInstance)

	res := connect.NewResponse(&databasesv1.RestartDatabaseResponse{
		Database: protoDB,
	})
	return res, nil
}

// SleepDatabase puts a database to sleep (auto-wakes on connection)
func (s *Service) SleepDatabase(ctx context.Context, req *connect.Request[databasesv1.SleepDatabaseRequest]) (*connect.Response[databasesv1.SleepDatabaseResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		if eff, ok := resolveUserDefaultOrgID(ctx); ok {
			orgID = eff
		}
	}

	// Check resource-level permission (reuse stop permission)
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

	// Update status to STOPPING temporarily
	dbInstance.Status = 4 // STOPPING
	if err := s.repo.Update(ctx, dbInstance); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to sleep database: %w", err))
	}

	// Stop the database container asynchronously
	go func() {
		sleepCtx, cancel := s.detachedContext(2 * time.Minute)
		defer cancel()

		if dbInstance.InstanceID != nil && *dbInstance.InstanceID != "" && s.provisioner != nil {
			if err := s.provisioner.StopDatabase(sleepCtx, *dbInstance.InstanceID); err != nil {
				logger.Error("Failed to stop database container for sleep: %v", err)
				dbInstance.Status = 8 // FAILED
				s.repo.Update(sleepCtx, dbInstance)
				return
			}
		}

		// Update status to SLEEPING (not STOPPED)
		dbInstance.Status = 12 // SLEEPING
		if err := s.repo.Update(sleepCtx, dbInstance); err != nil {
			logger.Error("Failed to update database status: %v", err)
		}

		// Update route registry - SLEEPING means auto-wake on connect
		if s.routeRegistry != nil {
			s.routeRegistry.MarkStopped(dbInstance.ID, 12)
		}
	}()

	protoDB := dbDatabaseToProto(dbInstance)

	res := connect.NewResponse(&databasesv1.SleepDatabaseResponse{
		Database: protoDB,
	})
	return res, nil
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}
