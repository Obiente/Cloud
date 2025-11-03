package database

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// BuildLogsRepository handles build logs stored in TimescaleDB
type BuildLogsRepository struct {
	db *gorm.DB // TimescaleDB connection
}

// NewBuildLogsRepository creates a new repository for build logs using TimescaleDB
func NewBuildLogsRepository(timescaleDB *gorm.DB) *BuildLogsRepository {
	return &BuildLogsRepository{
		db: timescaleDB,
	}
}

// AddBuildLog adds a log line to a build
func (r *BuildLogsRepository) AddBuildLog(ctx context.Context, buildID string, line string, stderr bool, lineNumber int32) error {
	buildLog := &BuildLog{
		BuildID:    buildID,
		Line:       line,
		Timestamp:  time.Now(),
		Stderr:     stderr,
		LineNumber: lineNumber,
	}
	return r.db.WithContext(ctx).Create(buildLog).Error
}

// AddBuildLogsBatch adds multiple log lines to a build in a single transaction
// This is much more efficient for TimescaleDB than individual inserts
func (r *BuildLogsRepository) AddBuildLogsBatch(ctx context.Context, buildID string, logs []struct {
	Line      string
	Stderr    bool
	LineNumber int32
	Timestamp time.Time
}) error {
	if len(logs) == 0 {
		return nil
	}

	// Convert to BuildLog structs
	buildLogs := make([]*BuildLog, len(logs))
	for i, logEntry := range logs {
		buildLogs[i] = &BuildLog{
			BuildID:    buildID,
			Line:       logEntry.Line,
			Timestamp:  logEntry.Timestamp,
			Stderr:     logEntry.Stderr,
			LineNumber: logEntry.LineNumber,
		}
	}

	// Batch insert using CreateInBatches (GORM handles chunking automatically)
	// TimescaleDB benefits greatly from batch inserts
	return r.db.WithContext(ctx).CreateInBatches(buildLogs, 100).Error
}

// GetBuildLogs retrieves logs for a build with pagination
func (r *BuildLogsRepository) GetBuildLogs(ctx context.Context, buildID string, limit, offset int) ([]*BuildLog, int64, error) {
	query := r.db.WithContext(ctx).Where("build_id = ?", buildID)

	// Get total count
	var total int64
	if err := query.Model(&BuildLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	// Order by line number ascending
	query = query.Order("line_number ASC")

	var logs []*BuildLog
	if err := query.Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

