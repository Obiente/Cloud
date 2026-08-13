package databases

import (
	"context"
	"fmt"
	"time"

	"databases-service/internal/proxy"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
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
	updateDatabaseLocationStatus(ctx, dbInstance.ID, "starting")

	// Start the database container asynchronously
	go func() {
		startCtx, cancel := s.detachedContext(2 * time.Minute)
		defer cancel()

		if dbInstance.InstanceID != nil && *dbInstance.InstanceID != "" && s.provisioner != nil {
			if err := s.provisioner.StartDatabase(startCtx, *dbInstance.InstanceID); err != nil {
				logger.Error("Failed to start database container: %v", err)
				dbInstance.Status = 8 // FAILED
				s.repo.Update(startCtx, dbInstance)
				updateDatabaseLocationStatus(startCtx, dbInstance.ID, "failed")
				return
			}
		}

		// Update status to RUNNING
		dbInstance.Status = 3 // RUNNING
		dbInstance.LastStartedAt = timePtr(time.Now())
		if err := s.persistDatabaseInstance(startCtx, dbInstance); err != nil {
			logger.Error("Failed to persist running database status: %v", err)
			return
		}
		updateDatabaseLocationStatus(startCtx, dbInstance.ID, "running")

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
	updateDatabaseLocationStatus(ctx, dbInstance.ID, "stopping")

	// Stop the database container asynchronously
	go func() {
		stopCtx, cancel := s.detachedContext(2 * time.Minute)
		defer cancel()

		if dbInstance.InstanceID != nil && *dbInstance.InstanceID != "" && s.provisioner != nil {
			if err := s.provisioner.StopDatabase(stopCtx, *dbInstance.InstanceID); err != nil {
				logger.Error("Failed to stop database container: %v", err)
				s.restoreDatabaseAfterFailedStop(stopCtx, dbInstance)
				return
			}
		}

		// Update status to STOPPED
		dbInstance.Status = 5 // STOPPED
		if err := s.persistDatabaseInstance(stopCtx, dbInstance); err != nil {
			logger.Error("Failed to persist stopped database status: %v", err)
			return
		}
		updateDatabaseLocationStatus(stopCtx, dbInstance.ID, "stopped")

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
	updateDatabaseLocationStatus(ctx, dbInstance.ID, "starting")

	// Restart the database container asynchronously
	go func() {
		restartCtx, cancel := s.detachedContext(3 * time.Minute)
		defer cancel()

		if dbInstance.InstanceID != nil && *dbInstance.InstanceID != "" && s.provisioner != nil {
			if err := s.provisioner.RestartDatabase(restartCtx, *dbInstance.InstanceID); err != nil {
				logger.Error("Failed to restart database container: %v", err)
				dbInstance.Status = 8 // FAILED
				s.repo.Update(restartCtx, dbInstance)
				updateDatabaseLocationStatus(restartCtx, dbInstance.ID, "failed")
				return
			}
		}

		// Update status to RUNNING
		dbInstance.Status = 3 // RUNNING
		dbInstance.LastStartedAt = timePtr(time.Now())
		if err := s.persistDatabaseInstance(restartCtx, dbInstance); err != nil {
			logger.Error("Failed to persist running database status after restart: %v", err)
			return
		}
		updateDatabaseLocationStatus(restartCtx, dbInstance.ID, "running")
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

	// Report STOPPING to the caller, but retain durable RUNNING until Docker has
	// stopped and the wakeable SLEEPING transition can be committed.
	dbInstance.Status = 4 // STOPPING
	updateDatabaseLocationStatus(ctx, dbInstance.ID, "stopping")

	// Stop the database container asynchronously
	go func() {
		sleepCtx, cancel := s.detachedContext(2 * time.Minute)
		defer cancel()

		if dbInstance.InstanceID != nil && *dbInstance.InstanceID != "" && s.provisioner != nil {
			if err := s.provisioner.StopDatabase(sleepCtx, *dbInstance.InstanceID); err != nil {
				logger.Error("Failed to stop database container for sleep: %v", err)
				s.restoreDatabaseAfterFailedStop(sleepCtx, dbInstance)
				return
			}
		}

		// Update status to SLEEPING (not STOPPED)
		dbInstance.Status = 12 // SLEEPING
		if err := s.persistDatabaseInstance(sleepCtx, dbInstance); err != nil {
			logger.Error("Failed to persist sleeping database status: %v", s.recoverAfterSleepingWriteFailure(dbInstance, &proxy.Route{
				DatabaseID:  dbInstance.ID,
				ContainerID: valueOrEmpty(dbInstance.InstanceID),
			}, err))
			return
		}
		updateDatabaseLocationStatus(sleepCtx, dbInstance.ID, "sleeping")

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

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (s *Service) restoreDatabaseAfterFailedStop(ctx context.Context, dbInstance *database.DatabaseInstance) {
	if dbInstance == nil {
		return
	}
	dbInstance.Status = 3 // RUNNING: a failed stop must keep proxy, metrics, and uptime tracking active.
	if err := s.repo.Update(ctx, dbInstance); err != nil {
		logger.Error("Failed to restore database status after stop failure: %v", err)
	}
	updateDatabaseLocationStatus(ctx, dbInstance.ID, "running")
}

func (s *Service) persistDatabaseInstance(ctx context.Context, dbInstance *database.DatabaseInstance) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("database repository is unavailable")
	}
	return retryDatabaseWrite(ctx, 5, 250*time.Millisecond, func() error {
		return s.repo.Update(ctx, dbInstance)
	})
}

func (s *Service) persistDatabaseConnection(ctx context.Context, connection *database.DatabaseConnection) error {
	if s == nil || s.connRepo == nil {
		return fmt.Errorf("database connection repository is unavailable")
	}
	return retryDatabaseWrite(ctx, 5, 250*time.Millisecond, func() error {
		return s.connRepo.CreateOrUpdate(ctx, connection)
	})
}

func retryDatabaseWrite(ctx context.Context, attempts int, delay time.Duration, update func() error) error {
	if attempts < 1 {
		attempts = 1
	}
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err = update(); err == nil {
			return nil
		}
		if attempt == attempts {
			break
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return fmt.Errorf("persist database status: %w", ctx.Err())
		case <-timer.C:
		}
	}
	return fmt.Errorf("persist database status after %d attempts: %w", attempts, err)
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}

func updateDatabaseLocationStatus(ctx context.Context, databaseID, status string) {
	if err := database.UpdateDatabaseLocationStatus(ctx, databaseID, status); err != nil {
		logger.Warn("Failed to update database %s location status to %s: %v", databaseID, status, err)
	}
}
