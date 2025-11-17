package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type BuildHistoryRepository struct {
	db    *gorm.DB
	cache *RedisCache
}

func NewBuildHistoryRepository(db *gorm.DB) *BuildHistoryRepository {
	return &BuildHistoryRepository{
		db:    db,
		cache: nil, // Builds are less frequently accessed, caching optional
	}
}

func NewBuildHistoryRepositoryWithCache(db *gorm.DB, cache *RedisCache) *BuildHistoryRepository {
	return &BuildHistoryRepository{
		db:    db,
		cache: cache,
	}
}

// CreateBuild creates a new build record
func (r *BuildHistoryRepository) CreateBuild(ctx context.Context, build *BuildHistory) error {
	if err := r.db.WithContext(ctx).Create(build).Error; err != nil {
		return err
	}

	// Cache the newly created build if cache is available
	if r.cache != nil {
		cacheKey := fmt.Sprintf("build:%s", build.ID)
		r.cache.Set(ctx, cacheKey, build, 2*time.Minute)
	}

	return nil
}

// GetBuildByID retrieves a build by ID
func (r *BuildHistoryRepository) GetBuildByID(ctx context.Context, buildID string) (*BuildHistory, error) {
	cacheKey := fmt.Sprintf("build:%s", buildID)

	// Try cache first if available
	if r.cache != nil {
		if cachedData, err := r.cache.Get(ctx, cacheKey); err == nil && cachedData != "" {
			var build BuildHistory
			if err := json.Unmarshal([]byte(cachedData), &build); err == nil {
				return &build, nil
			}
		}
	}

	var build BuildHistory
	if err := r.db.WithContext(ctx).Where("id = ?", buildID).First(&build).Error; err != nil {
		return nil, err
	}

	// Cache the result if cache is available
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, build, 2*time.Minute) // Shorter TTL for builds
	}

	return &build, nil
}

// ListBuilds retrieves builds for a deployment with pagination
func (r *BuildHistoryRepository) ListBuilds(ctx context.Context, deploymentID, organizationID string, limit, offset int) ([]*BuildHistory, int64, error) {
	query := r.db.WithContext(ctx).
		Where("deployment_id = ? AND organization_id = ?", deploymentID, organizationID)

	// Get total count
	var total int64
	if err := query.Model(&BuildHistory{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// Order by build number descending (newest first)
	query = query.Order("build_number DESC")

	var builds []*BuildHistory
	if err := query.Find(&builds).Error; err != nil {
		return nil, 0, err
	}

	return builds, total, nil
}

// GetNextBuildNumber returns the next build number for a deployment
func (r *BuildHistoryRepository) GetNextBuildNumber(ctx context.Context, deploymentID string) (int32, error) {
	var maxBuildNumber int32
	err := r.db.WithContext(ctx).
		Model(&BuildHistory{}).
		Where("deployment_id = ?", deploymentID).
		Select("COALESCE(MAX(build_number), 0)").
		Scan(&maxBuildNumber).Error
	
	if err != nil {
		return 0, err
	}
	return maxBuildNumber + 1, nil
}

// UpdateBuildStatus updates the build status and completion time
func (r *BuildHistoryRepository) UpdateBuildStatus(ctx context.Context, buildID string, status int32, buildTime int32, errorMsg *string) error {
	now := time.Now()
	update := map[string]interface{}{
		"status":      status,
		"build_time":  buildTime,
		"updated_at":  now,
	}
	
	if status == 3 || status == 4 { // BUILD_SUCCESS or BUILD_FAILED
		update["completed_at"] = &now
	}
	
	if errorMsg != nil {
		update["error"] = *errorMsg
	}
	
	if err := r.db.WithContext(ctx).
		Model(&BuildHistory{}).
		Where("id = ?", buildID).
		Updates(update).Error; err != nil {
		return err
	}

	// Clear cache AFTER successful update
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("build:%s", buildID))
	}

	return nil
}

// UpdateBuildResults updates build result fields (image name, compose yaml, size)
func (r *BuildHistoryRepository) UpdateBuildResults(ctx context.Context, buildID string, imageName, composeYaml, size *string) error {
	update := map[string]interface{}{
		"updated_at": time.Now(),
	}
	
	if imageName != nil {
		update["image_name"] = *imageName
	}
	if composeYaml != nil {
		update["compose_yaml"] = *composeYaml
	}
	if size != nil {
		update["size"] = *size
	}
	
	if err := r.db.WithContext(ctx).
		Model(&BuildHistory{}).
		Where("id = ?", buildID).
		Updates(update).Error; err != nil {
		return err
	}

	// Clear cache AFTER successful update
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("build:%s", buildID))
	}

	return nil
}

// NOTE: Build log methods (AddBuildLog, AddBuildLogsBatch, GetBuildLogs) have been moved
// to BuildLogsRepository which uses TimescaleDB. Use database.NewBuildLogsRepository(MetricsDB)
// to get a repository instance for build logs.

// GetLatestSuccessfulBuild returns the most recent successful build for a deployment
// Returns nil, nil if no successful build is found (this is a normal condition, not an error)
func (r *BuildHistoryRepository) GetLatestSuccessfulBuild(ctx context.Context, deploymentID string) (*BuildHistory, error) {
	var build BuildHistory
	err := r.db.WithContext(ctx).
		Where("deployment_id = ? AND status = ?", deploymentID, 3). // BUILD_SUCCESS = 3
		Order("build_number DESC").
		First(&build).Error
	
	if err != nil {
		// "Record not found" is a normal condition - deployment simply has no successful builds yet
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		// Return other errors (database issues, etc.)
		return nil, err
	}
	return &build, nil
}

// DeleteBuild deletes a build from PostgreSQL
// Note: Build logs are stored in TimescaleDB and must be deleted separately
func (r *BuildHistoryRepository) DeleteBuild(ctx context.Context, buildID, organizationID string) error {
	// First verify the build belongs to the organization
	var build BuildHistory
	if err := r.db.WithContext(ctx).Where("id = ? AND organization_id = ?", buildID, organizationID).First(&build).Error; err != nil {
		return err
	}

	// Delete the build (logs are handled separately in the service layer using TimescaleDB)
	if err := r.db.WithContext(ctx).Where("id = ?", buildID).Delete(&BuildHistory{}).Error; err != nil {
		return err
	}

	// Clear cache AFTER successful delete
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("build:%s", buildID))
	}

	return nil
}

// DeleteBuildsByDeployment deletes all builds for a deployment
// Returns the build IDs that were deleted and the count, so the caller can delete logs from TimescaleDB
// The caller should use BuildLogsRepository to delete logs for the returned build IDs
func (r *BuildHistoryRepository) DeleteBuildsByDeployment(ctx context.Context, deploymentID string) ([]string, int64, error) {
	// Get build IDs that will be deleted so caller can delete logs from TimescaleDB
	var builds []BuildHistory
	if err := r.db.WithContext(ctx).
		Where("deployment_id = ?", deploymentID).
		Select("id").
		Find(&builds).Error; err != nil {
		return nil, 0, err
	}

	buildIDs := make([]string, len(builds))
	for i, build := range builds {
		buildIDs[i] = build.ID
	}

	// Delete builds from PostgreSQL
	result := r.db.WithContext(ctx).
		Where("deployment_id = ?", deploymentID).
		Delete(&BuildHistory{})

	if result.Error != nil {
		return nil, 0, result.Error
	}

	return buildIDs, result.RowsAffected, nil
}

// DeleteBuildsOlderThan deletes all builds older than the specified duration
// Note: This function does NOT delete logs - logs are stored in TimescaleDB and must be deleted separately
// The caller should use BuildLogsRepository to delete logs for the returned build IDs
func (r *BuildHistoryRepository) DeleteBuildsOlderThan(ctx context.Context, olderThan time.Duration) ([]string, int64, error) {
	cutoff := time.Now().Add(-olderThan)
	
	// Get build IDs that will be deleted so caller can delete logs from TimescaleDB
	var buildIDs []string
	err := r.db.WithContext(ctx).
		Model(&BuildHistory{}).
		Where("started_at < ?", cutoff).
		Pluck("id", &buildIDs).Error
	
	if err != nil {
		return nil, 0, err
	}

	// Delete builds from PostgreSQL
	result := r.db.WithContext(ctx).
		Where("started_at < ?", cutoff).
		Delete(&BuildHistory{})
	
	return buildIDs, result.RowsAffected, result.Error
}
