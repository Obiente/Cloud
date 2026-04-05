package databases

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"databases-service/internal/provisioner"
	"databases-service/internal/proxy"
	"databases-service/internal/secrets"

	"github.com/google/uuid"
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
	totalCount, err := s.repo.CountAll(ctx, orgID, filters)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to count databases: %w", err))
	}

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
	if s.provisioner == nil {
		return nil, connect.NewError(
			connect.CodeFailedPrecondition,
			fmt.Errorf("database provisioning is unavailable on this node; no provisioner is configured"),
		)
	}

	id := fmt.Sprintf("db-%s", uuid.NewString())

	// Parse metadata
	metadataJSON := "{}"
	if len(req.Msg.Metadata) > 0 {
		metadataBytes, err := json.Marshal(req.Msg.Metadata)
		if err == nil {
			metadataJSON = string(metadataBytes)
		}
	}

	size := req.Msg.GetSize()
	if size == "" {
		size = "small"
	}
	sizeSpec, ok := lookupDatabaseSize(req.Msg.GetType(), size)
	if !ok {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("unknown database size %q for type %s", size, req.Msg.GetType().String()),
		)
	}
	cpuCores := sizeSpec.CpuCores
	memoryBytes := sizeSpec.MemoryBytes
	diskBytes := sizeSpec.DiskBytes
	maxConnections := sizeSpec.MaxConnections

	// Auto-sleep configuration
	var autoSleepSeconds int32
	if req.Msg.AutoSleepSeconds != nil {
		autoSleepSeconds = *req.Msg.AutoSleepSeconds
	}

	// Create database instance record first
	dbInstance := &database.DatabaseInstance{
		ID:               id,
		Name:             req.Msg.GetName(),
		Description:      req.Msg.Description,
		Status:           1, // CREATING
		Type:             int32(req.Msg.GetType()),
		Version:          req.Msg.Version,
		Size:             size,
		CPUCores:         cpuCores,
		MemoryBytes:      memoryBytes,
		DiskBytes:        diskBytes,
		MaxConnections:   maxConnections,
		AutoSleepSeconds: autoSleepSeconds,
		Metadata:         metadataJSON,
		OrganizationID:   orgID,
		CreatedBy:        userInfo.Id,
	}

	if err := s.repo.Create(ctx, dbInstance); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create database record: %w", err))
	}

	initialUsername := req.Msg.GetInitialUsername()
	if initialUsername == "" {
		initialUsername = "admin"
	}
	initialPassword := req.Msg.GetInitialPassword()
	if initialPassword == "" {
		// Use secure password generation
		var err error
		initialPassword, err = secrets.GenerateSecurePassword(32)
		if err != nil {
			logger.Warn("Failed to generate secure password: %v, using fallback", err)
			initialPassword = generateRandomPassword(32)
		}
	}

	// Determine port based on database type
	port := int32(5432) // Default PostgreSQL port
	if req.Msg.GetType() == databasesv1.DatabaseType_MYSQL || req.Msg.GetType() == databasesv1.DatabaseType_MARIADB {
		port = 3306
	} else if req.Msg.GetType() == databasesv1.DatabaseType_MONGODB {
		port = 27017
	} else if req.Msg.GetType() == databasesv1.DatabaseType_REDIS {
		port = 6379
	}

	// Provision the actual database container asynchronously
	go func() {
		provisionCtx, cancel := s.detachedContext(5 * time.Minute)
		defer cancel()

		version := ""
		if req.Msg.Version != nil {
			version = *req.Msg.Version
		}
		provCfg := &provisioner.DatabaseConfig{
			DatabaseID:  id,
			Type:        provisioner.DatabaseType(req.Msg.GetType()),
			Version:     version,
			Username:    initialUsername,
			Password:    initialPassword,
			Port:        int(port),
			CPUCores:    float64(cpuCores),
			MemoryBytes: memoryBytes,
			DiskBytes:   diskBytes,
		}

		result, err := s.provisioner.ProvisionDatabase(provisionCtx, provCfg)
		if err != nil {
			logger.Error("Failed to provision database container: %v", err)
			// Mark as failed
			dbInstance.Status = 8 // FAILED
			s.repo.Update(provisionCtx, dbInstance)
			return
		}

		// Update database instance with container info
		dbInstance.InstanceID = &result.ContainerID
		dbInstance.Host = &result.Host
		dbInstance.Port = &port
		dbInstance.Status = 3 // RUNNING

		// Create connection record
		connID := fmt.Sprintf("conn-%s", uuid.NewString())

		// Encrypt password before storing
		encryptedPassword := initialPassword
		if s.secretManager != nil {
			if encrypted, err := s.secretManager.EncryptPassword(initialPassword); err == nil {
				encryptedPassword = encrypted
			} else {
				logger.Warn("Failed to encrypt password: %v, storing plaintext", err)
			}
		}

		dbConn := &database.DatabaseConnection{
			ID:           connID,
			DatabaseID:   id,
			DatabaseName: id,
			Username:     initialUsername,
			Password:     encryptedPassword,
			Host:         *dbInstance.Host,
			Port:         port,
			SSLRequired:  true,
		}

		if err := s.connRepo.Create(provisionCtx, dbConn); err != nil {
			logger.Warn("Failed to create connection record: %v", err)
		}

		// Update database with final status
		if err := s.repo.Update(provisionCtx, dbInstance); err != nil {
			logger.Error("Failed to update database instance: %v", err)
		}

		// Register route in proxy
		if s.routeRegistry != nil && dbInstance.Status == 3 {
			dbType := provisioner.DatabaseType(req.Msg.GetType()).String()
			route := &proxy.Route{
				DatabaseID:       id,
				DatabaseType:     dbType,
				InternalPort:     int(port),
				Username:         initialUsername,
				Password:         encryptedPassword,
				OrganizationID:   orgID,
				AutoSleepSeconds: autoSleepSeconds,
				LastConnectionAt: time.Now(),
			}
			if dbInstance.InstanceID != nil {
				route.ContainerID = *dbInstance.InstanceID
			}
			if dbInstance.Host != nil {
				route.ContainerIP = *dbInstance.Host
			}

			// Allocate Redis port if needed
			if dbType == "redis" {
				if redisPort, err := s.routeRegistry.AllocateRedisPort(id); err == nil {
					route.RedisPort = redisPort
				} else {
					logger.Error("Failed to allocate Redis port: %v", err)
				}
			}

			s.routeRegistry.Register(route)

			// Start Redis listener if needed
			if route.RedisPort > 0 && s.proxy != nil {
				if err := s.proxy.StartRedisListener(route); err != nil {
					logger.Error("Failed to start Redis listener: %v", err)
				}
			}
		}

		logger.Info("Database provisioning complete: %s (host: %s)", id, *dbInstance.Host)
	}()

	// Return immediately with CREATING status
	protoDB := dbDatabaseToProto(dbInstance)

	res := connect.NewResponse(&databasesv1.CreateDatabaseResponse{
		Database: protoDB,
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

	// Update auto-sleep seconds
	if req.Msg.AutoSleepSeconds != nil {
		dbInstance.AutoSleepSeconds = *req.Msg.AutoSleepSeconds
		// Update route registry too
		if s.routeRegistry != nil {
			if route, ok := s.routeRegistry.LookupByID(dbInstance.ID); ok {
				route.AutoSleepSeconds = *req.Msg.AutoSleepSeconds
			}
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

	// Delete the actual database container asynchronously
	go func() {
		deleteCtx, cancel := s.detachedContext(2 * time.Minute)
		defer cancel()

		// Update status to DELETING
		dbInstance.Status = 9 // DELETING
		s.repo.Update(deleteCtx, dbInstance)

		// Unregister route from proxy
		if s.routeRegistry != nil {
			if route, ok := s.routeRegistry.LookupByID(req.Msg.GetDatabaseId()); ok {
				if route.RedisPort > 0 && s.proxy != nil {
					s.proxy.StopRedisListener(route.RedisPort)
				}
			}
			s.routeRegistry.Unregister(req.Msg.GetDatabaseId())
		}

		// Deprovision Docker container if it exists
		if dbInstance.InstanceID != nil && *dbInstance.InstanceID != "" && s.provisioner != nil {
			if err := s.provisioner.DeprovisionDatabase(deleteCtx, *dbInstance.InstanceID); err != nil {
				logger.Error("Failed to deprovision database container: %v", err)
				// Continue with soft delete anyway
			}
		}

		// Soft delete the database record
		if err := s.repo.Delete(deleteCtx, req.Msg.GetDatabaseId()); err != nil {
			logger.Error("Failed to soft delete database: %v", err)
		} else {
			logger.Info("Database deleted: %s", req.Msg.GetDatabaseId())
		}
	}()

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
	if _, err := cryptorand.Read(b); err != nil {
		return strings.Repeat("x", length)
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}
