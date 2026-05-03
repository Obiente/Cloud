package database

import (
	"context"
	"fmt"
	"strings"
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
	deduped := dedupeRuntimeLogs(logs)
	if len(deduped) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(deduped, 100).Error
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

func (r *DeploymentRuntimeLogsRepository) GetRecentLogsForServiceExcludingSources(ctx context.Context, deploymentID, serviceName string, limit int, excludedSources []string) ([]*DeploymentRuntimeLog, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	if strings.TrimSpace(serviceName) == "" {
		return r.GetRecentLogsExcludingSources(ctx, deploymentID, limit, excludedSources)
	}
	if limit <= 0 {
		limit = 200
	}

	query := r.db.WithContext(ctx).
		Where("deployment_id = ?", deploymentID).
		Where("service_name = ?", serviceName)

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

func (r *DeploymentRuntimeLogsRepository) GetRecentLogsForSources(ctx context.Context, deploymentID string, limit int, includedSources []string) ([]*DeploymentRuntimeLog, error) {
	if r == nil || r.db == nil || len(includedSources) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 200
	}

	var logs []*DeploymentRuntimeLog
	if err := r.db.WithContext(ctx).
		Where("deployment_id = ?", deploymentID).
		Where("source IN ?", includedSources).
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

func dedupeRuntimeLogs(logs []DeploymentRuntimeLog) []DeploymentRuntimeLog {
	seen := make(map[string]struct{}, len(logs))
	deduped := make([]DeploymentRuntimeLog, 0, len(logs))
	for _, entry := range logs {
		key := fmt.Sprintf("%s|%s|%s|%s|%d|%t|%d|%s",
			entry.DeploymentID,
			entry.ServiceName,
			entry.ContainerID,
			entry.Source,
			entry.Timestamp.UnixNano(),
			entry.Stderr,
			entry.LogLevel,
			entry.Line,
		)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, entry)
	}
	return deduped
}
