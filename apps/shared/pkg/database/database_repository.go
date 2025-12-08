package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type DatabaseRepository struct {
	db    *gorm.DB
	cache *RedisCache
}

func NewDatabaseRepository(db *gorm.DB, cache *RedisCache) *DatabaseRepository {
	return &DatabaseRepository{
		db:    db,
		cache: cache,
	}
}

func (r *DatabaseRepository) Create(ctx context.Context, database *DatabaseInstance) error {
	if err := r.db.WithContext(ctx).Create(database).Error; err != nil {
		return err
	}

	// Cache the newly created database
	if r.cache != nil {
		cacheKey := fmt.Sprintf("database:%s", database.ID)
		r.cache.Set(ctx, cacheKey, database, 5*time.Minute)
	}

	return nil
}

func (r *DatabaseRepository) GetByID(ctx context.Context, id string) (*DatabaseInstance, error) {
	return r.GetByIDIncludeDeleted(ctx, id, false)
}

func (r *DatabaseRepository) GetByIDIncludeDeleted(ctx context.Context, id string, includeDeleted bool) (*DatabaseInstance, error) {
	cacheKey := fmt.Sprintf("database:%s", id)

	// Try cache first
	if r.cache != nil {
		if cachedData, err := r.cache.Get(ctx, cacheKey); err == nil && cachedData != "" {
			var database DatabaseInstance
			if err := json.Unmarshal([]byte(cachedData), &database); err == nil {
				return &database, nil
			}
		}
	}

	var database DatabaseInstance
	query := r.db.WithContext(ctx).Where("id = ?", id)
	if !includeDeleted {
		query = query.Where("deleted_at IS NULL")
	}
	if err := query.First(&database).Error; err != nil {
		return nil, err
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, database, 5*time.Minute)
	}

	return &database, nil
}

type DatabaseFilters struct {
	Status *int32
	Type   *int32
	Limit  int64
	Offset int64
}

func (r *DatabaseRepository) GetAll(ctx context.Context, organizationID string, filters *DatabaseFilters) ([]*DatabaseInstance, error) {
	query := r.db.WithContext(ctx).Where("organization_id = ? AND deleted_at IS NULL", organizationID)

	if filters != nil {
		// Apply status filter if provided
		if filters.Status != nil {
			query = query.Where("status = ?", *filters.Status)
		}

		// Apply type filter if provided
		if filters.Type != nil {
			query = query.Where("type = ?", *filters.Type)
		}

		// Apply pagination
		if filters.Limit > 0 {
			query = query.Limit(int(filters.Limit))
		}
		if filters.Offset > 0 {
			query = query.Offset(int(filters.Offset))
		}
	}

	var databases []*DatabaseInstance
	if err := query.Find(&databases).Error; err != nil {
		return nil, err
	}

	// Cache individual items for faster subsequent GetByID calls
	if r.cache != nil && len(databases) > 0 {
		pairs := make(map[string]interface{})
		for _, db := range databases {
			pairs[fmt.Sprintf("database:%s", db.ID)] = db
		}
		// Use MSet for batch caching (more efficient)
		r.cache.MSet(ctx, pairs, 5*time.Minute)
	}

	return databases, nil
}

func (r *DatabaseRepository) Update(ctx context.Context, database *DatabaseInstance) error {
	// Use Select to explicitly update fields
	if err := r.db.WithContext(ctx).
		Model(database).
		Select(
			"name", "description", "status", "type", "version",
			"size", "cpu_cores", "memory_bytes", "disk_bytes", "disk_used_bytes",
			"max_connections", "host", "port", "instance_id", "node_id",
			"metadata", "last_started_at", "updated_at",
		).
		Updates(database).Error; err != nil {
		return err
	}

	// Clear and re-populate cache AFTER successful update
	if r.cache != nil {
		cacheKey := fmt.Sprintf("database:%s", database.ID)
		r.cache.Delete(ctx, cacheKey)
		r.cache.Set(ctx, cacheKey, database, 5*time.Minute)
	}

	return nil
}

func (r *DatabaseRepository) Delete(ctx context.Context, id string) error {
	// Soft delete
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Model(&DatabaseInstance{}).
		Where("id = ?", id).
		Update("deleted_at", now).Error; err != nil {
		return err
	}

	// Clear cache
	if r.cache != nil {
		cacheKey := fmt.Sprintf("database:%s", id)
		r.cache.Delete(ctx, cacheKey)
	}

	return nil
}

func (r *DatabaseRepository) HardDelete(ctx context.Context, id string) error {
	// Hard delete
	if err := r.db.WithContext(ctx).Unscoped().Delete(&DatabaseInstance{}, "id = ?", id).Error; err != nil {
		return err
	}

	// Clear cache
	if r.cache != nil {
		cacheKey := fmt.Sprintf("database:%s", id)
		r.cache.Delete(ctx, cacheKey)
	}

	return nil
}

// DatabaseConnectionRepository handles database connection credentials
type DatabaseConnectionRepository struct {
	db *gorm.DB
}

func NewDatabaseConnectionRepository(db *gorm.DB) *DatabaseConnectionRepository {
	return &DatabaseConnectionRepository{
		db: db,
	}
}

func (r *DatabaseConnectionRepository) Create(ctx context.Context, conn *DatabaseConnection) error {
	return r.db.WithContext(ctx).Create(conn).Error
}

func (r *DatabaseConnectionRepository) GetByDatabaseID(ctx context.Context, databaseID string) (*DatabaseConnection, error) {
	var conn DatabaseConnection
	if err := r.db.WithContext(ctx).Where("database_id = ?", databaseID).First(&conn).Error; err != nil {
		return nil, err
	}
	return &conn, nil
}

func (r *DatabaseConnectionRepository) Update(ctx context.Context, conn *DatabaseConnection) error {
	return r.db.WithContext(ctx).
		Model(conn).
		Select("username", "password", "database_name", "host", "port", "ssl_required", "ssl_certificate", "updated_at").
		Updates(conn).Error
}

func (r *DatabaseConnectionRepository) Delete(ctx context.Context, databaseID string) error {
	return r.db.WithContext(ctx).Delete(&DatabaseConnection{}, "database_id = ?", databaseID).Error
}

// DatabaseBackupRepository handles database backups
type DatabaseBackupRepository struct {
	db *gorm.DB
}

func NewDatabaseBackupRepository(db *gorm.DB) *DatabaseBackupRepository {
	return &DatabaseBackupRepository{
		db: db,
	}
}

func (r *DatabaseBackupRepository) Create(ctx context.Context, backup *DatabaseBackup) error {
	return r.db.WithContext(ctx).Create(backup).Error
}

func (r *DatabaseBackupRepository) GetByID(ctx context.Context, id string) (*DatabaseBackup, error) {
	var backup DatabaseBackup
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&backup).Error; err != nil {
		return nil, err
	}
	return &backup, nil
}

type BackupFilters struct {
	Status *int32
	Limit  int64
	Offset int64
}

func (r *DatabaseBackupRepository) GetAll(ctx context.Context, databaseID string, filters *BackupFilters) ([]*DatabaseBackup, error) {
	query := r.db.WithContext(ctx).Where("database_id = ?", databaseID)

	if filters != nil {
		if filters.Status != nil {
			query = query.Where("status = ?", *filters.Status)
		}
		if filters.Limit > 0 {
			query = query.Limit(int(filters.Limit))
		}
		if filters.Offset > 0 {
			query = query.Offset(int(filters.Offset))
		}
	}

	var backups []*DatabaseBackup
	if err := query.Order("created_at DESC").Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *DatabaseBackupRepository) Update(ctx context.Context, backup *DatabaseBackup) error {
	return r.db.WithContext(ctx).
		Model(backup).
		Select("name", "description", "size_bytes", "status", "completed_at", "error_message", "backup_path").
		Updates(backup).Error
}

func (r *DatabaseBackupRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&DatabaseBackup{}, "id = ?", id).Error
}

