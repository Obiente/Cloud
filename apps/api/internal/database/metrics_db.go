package database

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"api/internal/logger"
)

var MetricsDB *gorm.DB

// InitMetricsDatabase initializes a separate TimescaleDB/PostgreSQL connection for metrics
func InitMetricsDatabase() error {
	// Use separate environment variables for metrics database, with fallback to main DB
	host := os.Getenv("METRICS_DB_HOST")
	if host == "" {
		host = os.Getenv("DB_HOST") // Fallback to main DB host
		if host == "" {
			host = "localhost"
		}
	}

	port := os.Getenv("METRICS_DB_PORT")
	if port == "" {
		port = os.Getenv("DB_PORT") // Fallback to main DB port
		if port == "" {
			port = "5432"
		}
	}

	user := os.Getenv("METRICS_DB_USER")
	if user == "" {
		user = os.Getenv("DB_USER") // Fallback to main DB user
		if user == "" {
			user = "postgres"
		}
	}

	password := os.Getenv("METRICS_DB_PASSWORD")
	if password == "" {
		password = os.Getenv("DB_PASSWORD") // Fallback to main DB password
	}

	dbname := os.Getenv("METRICS_DB_NAME")
	if dbname == "" {
		dbname = "obiente_metrics" // Default metrics database name
	}

	// Build DSN
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Open database connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: getGormLogger(),
	})
	if err != nil {
		logger.Warn("Failed to connect to metrics database: %v. Falling back to main database.", err)
		// Fallback to main database if metrics DB is unavailable
		MetricsDB = DB
		return nil
	}

	MetricsDB = db
	logger.Info("Metrics database connection established")

	// Create TimescaleDB extension if available (will fail gracefully if not installed)
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS timescaledb").Error; err != nil {
		logger.Warn("TimescaleDB extension not available: %v. Using standard PostgreSQL.", err)
	} else {
		logger.Info("TimescaleDB extension enabled")
	}

	// Initialize metrics tables
	if err := InitMetricsTables(); err != nil {
		return fmt.Errorf("failed to initialize metrics tables: %w", err)
	}

	logger.Info("Metrics database initialized")
	return nil
}

// InitMetricsTables creates and configures metrics tables
func InitMetricsTables() error {
	// Auto-migrate metrics tables (including build_logs which is stored in TimescaleDB)
	if err := MetricsDB.AutoMigrate(
		&DeploymentMetrics{},
		&DeploymentUsageHourly{},
		&BuildLog{},
	); err != nil {
		return fmt.Errorf("failed to auto-migrate metrics tables: %w", err)
	}

	// Initialize TimescaleDB hypertable for build_logs
	if err := InitBuildLogsTimescaleDB(MetricsDB); err != nil {
		logger.Warn("Failed to initialize TimescaleDB hypertable for build_logs: %v", err)
		// Continue anyway - standard PostgreSQL will work fine
	}

	// Create composite indexes for better query performance
	if err := createMetricsIndexes(); err != nil {
		return fmt.Errorf("failed to create metrics indexes: %w", err)
	}

	// Convert to hypertable if TimescaleDB is available
	// Check if table is already a hypertable
	var isHypertable bool
	if err := MetricsDB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM timescaledb_information.hypertables 
			WHERE hypertable_name = 'deployment_metrics'
		)
	`).Scan(&isHypertable).Error; err == nil && !isHypertable {
		// Convert to hypertable with 1 hour chunk interval
		if err := MetricsDB.Exec(`
			SELECT create_hypertable('deployment_metrics', 'timestamp', 
				chunk_time_interval => INTERVAL '1 hour',
				if_not_exists => TRUE)
		`).Error; err != nil {
			logger.Warn("Failed to create hypertable for deployment_metrics: %v", err)
		} else {
			logger.Info("Created TimescaleDB hypertable for deployment_metrics")
		}
	}

	// Convert hourly aggregates to hypertable if available
	if err := MetricsDB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM timescaledb_information.hypertables 
			WHERE hypertable_name = 'deployment_usage_hourly'
		)
	`).Scan(&isHypertable).Error; err == nil && !isHypertable {
		if err := MetricsDB.Exec(`
			SELECT create_hypertable('deployment_usage_hourly', 'hour', 
				chunk_time_interval => INTERVAL '7 days',
				if_not_exists => TRUE)
		`).Error; err != nil {
			logger.Warn("Failed to create hypertable for deployment_usage_hourly: %v", err)
		} else {
			logger.Info("Created TimescaleDB hypertable for deployment_usage_hourly")
		}
	}

	return nil
}
