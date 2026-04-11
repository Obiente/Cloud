package database

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// DeploymentRuntimeLogsRepository handles persisted deployment runtime logs stored in MetricsDB.
type DeploymentRuntimeLogsRepository struct {
	db *gorm.DB
}

func NewDeploymentRuntimeLogsRepository(db *gorm.DB) *DeploymentRuntimeLogsRepository {
	return &DeploymentRuntimeLogsRepository{db: db}
}

func (r *DeploymentRuntimeLogsRepository) AddLogsBatch(ctx context.Context, logs []DeploymentRuntimeLog) error {
	if r == nil || r.db == nil || len(logs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(logs, 100).Error
}

func (r *DeploymentRuntimeLogsRepository) GetRecentLogs(ctx context.Context, deploymentID string, limit int) ([]*DeploymentRuntimeLog, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 200
	}

	var logs []*DeploymentRuntimeLog
	if err := r.db.WithContext(ctx).
		Where("deployment_id = ?", deploymentID).
		Order("timestamp DESC, id DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	// Return chronological order for UI consumption.
	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}
	return logs, nil
}

func (r *DeploymentRuntimeLogsRepository) GetRecentLogsExcludingSources(ctx context.Context, deploymentID string, limit int, excludedSources []string) ([]*DeploymentRuntimeLog, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 200
	}

	query := r.db.WithContext(ctx).
		Where("deployment_id = ?", deploymentID)

	if len(excludedSources) > 0 {
		query = query.Where("source NOT IN ?", excludedSources)
	}

	var logs []*DeploymentRuntimeLog
	if err := query.
		Order("timestamp DESC, id DESC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}
	return logs, nil
}

func (r *DeploymentRuntimeLogsRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.WithContext(ctx).Where("timestamp < ?", cutoff).Delete(&DeploymentRuntimeLog{}).Error
}
