package database

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type BuildHistoryRepository struct {
	db *gorm.DB
}

func NewBuildHistoryRepository(db *gorm.DB) *BuildHistoryRepository {
	return &BuildHistoryRepository{
		db: db,
	}
}

// CreateBuild creates a new build record
func (r *BuildHistoryRepository) CreateBuild(ctx context.Context, build *BuildHistory) error {
	return r.db.WithContext(ctx).Create(build).Error
}

// GetBuildByID retrieves a build by ID
func (r *BuildHistoryRepository) GetBuildByID(ctx context.Context, buildID string) (*BuildHistory, error) {
	var build BuildHistory
	if err := r.db.WithContext(ctx).Where("id = ?", buildID).First(&build).Error; err != nil {
		return nil, err
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
	
	return r.db.WithContext(ctx).
		Model(&BuildHistory{}).
		Where("id = ?", buildID).
		Updates(update).Error
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
	
	return r.db.WithContext(ctx).
		Model(&BuildHistory{}).
		Where("id = ?", buildID).
		Updates(update).Error
}

// NOTE: Build log methods (AddBuildLog, AddBuildLogsBatch, GetBuildLogs) have been moved
// to BuildLogsRepository which uses TimescaleDB. Use database.NewBuildLogsRepository(MetricsDB)
// to get a repository instance for build logs.

// GetLatestSuccessfulBuild returns the most recent successful build for a deployment
func (r *BuildHistoryRepository) GetLatestSuccessfulBuild(ctx context.Context, deploymentID string) (*BuildHistory, error) {
	var build BuildHistory
	err := r.db.WithContext(ctx).
		Where("deployment_id = ? AND status = ?", deploymentID, 3). // BUILD_SUCCESS = 3
		Order("build_number DESC").
		First(&build).Error
	
	if err != nil {
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
	return r.db.WithContext(ctx).Where("id = ?", buildID).Delete(&BuildHistory{}).Error
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
