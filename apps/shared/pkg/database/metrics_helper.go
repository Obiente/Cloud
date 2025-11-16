package database

import (
	"gorm.io/gorm"
)

// GetMetricsDB returns the metrics database connection (TimescaleDB)
// Returns nil if MetricsDB is not initialized - callers must check for nil
// DO NOT fallback to main DB - TimescaleDB is required for metrics, audit logs, and build logs
func GetMetricsDB() *gorm.DB {
	return MetricsDB
}
