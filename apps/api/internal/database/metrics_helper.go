package database

import (
	"gorm.io/gorm"
)

// GetMetricsDB returns the metrics database connection, falling back to main DB if not available
func GetMetricsDB() *gorm.DB {
	if MetricsDB != nil {
		return MetricsDB
	}
	return DB
}
