package database

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB
var RedisClient *RedisCache

// InitDatabase initializes the PostgreSQL database connection
func InitDatabase() error {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}

	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "obiente"
	}

	// Build DSN
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Open database connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	DB = db
	log.Println("Database connection established")

	// Pre-create groups column if it doesn't exist to avoid GORM AutoMigrate syntax issues
	// This prevents GORM from trying to add it with incorrect default value syntax
	if !db.Migrator().HasColumn("deployments", "groups") {
		log.Println("Creating groups column before AutoMigrate to avoid syntax issues...")
		if err := db.Exec(`ALTER TABLE deployments ADD COLUMN groups JSONB`).Error; err != nil {
			// Column might have been created by another process, check again
			if !db.Migrator().HasColumn("deployments", "groups") {
				log.Printf("Warning: Failed to pre-create groups column: %v. AutoMigrate may fail, but migration will handle it.", err)
			}
		} else {
			// Set default after adding column to avoid quote escaping issues
			if err := db.Exec(`ALTER TABLE deployments ALTER COLUMN groups SET DEFAULT '[]'::jsonb`).Error; err != nil {
				log.Printf("Warning: Failed to set groups default: %v", err)
			}
		}
	}

	// Auto-migrate the schema (build_logs is stored in TimescaleDB, not here)
	if err := db.AutoMigrate(
		&Deployment{},
		&BuildHistory{},
	); err != nil {
		return fmt.Errorf("failed to auto-migrate: %w", err)
	}

	log.Println("Database schema migrated")

	// Initialize deployment tracking tables
	if err := InitDeploymentTracking(); err != nil {
		return fmt.Errorf("failed to initialize deployment tracking: %w", err)
	}

	log.Println("Deployment tracking initialized")

	// Initialize metrics database (separate connection for metrics)
	if err := InitMetricsDatabase(); err != nil {
		log.Printf("Warning: Metrics database initialization failed: %v. Metrics may not work correctly.", err)
		// Don't fail main initialization if metrics DB fails
	}

	return nil
}

// InitRedis initializes the Redis connection
func InitRedis() error {
	// Will be implemented separately
	// For now, return nil if Redis is not configured
	if os.Getenv("REDIS_URL") == "" {
		log.Println("Redis not configured, running without cache")
		return nil
	}

	client := NewRedisCache()
	if err := client.Connect(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
		return nil // Don't fail if Redis is unavailable
	}

	RedisClient = client
	log.Println("Redis connection established")
	return nil
}

// InitBuildLogsTimescaleDB converts build_logs table to TimescaleDB hypertable if available
// This provides better performance for time-series log data
// Note: This should be called with MetricsDB (TimescaleDB connection), not the main DB
func InitBuildLogsTimescaleDB(db *gorm.DB) error {
	// Check if TimescaleDB extension is available
	var extensionExists bool
	if err := db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM pg_extension WHERE extname = 'timescaledb'
		)
	`).Scan(&extensionExists).Error; err != nil {
		return fmt.Errorf("failed to check TimescaleDB extension: %w", err)
	}

	if !extensionExists {
		// Try to create the extension
		if err := db.Exec("CREATE EXTENSION IF NOT EXISTS timescaledb").Error; err != nil {
			return fmt.Errorf("TimescaleDB extension not available: %w", err)
		}
		log.Println("TimescaleDB extension enabled for build_logs")
	}

	// Check if build_logs table exists
	if !db.Migrator().HasTable("build_logs") {
		log.Println("build_logs table does not exist yet, skipping TimescaleDB conversion")
		return nil
	}

	// Check if table is already a hypertable
	var isHypertable bool
	if err := db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM timescaledb_information.hypertables 
			WHERE hypertable_name = 'build_logs'
		)
	`).Scan(&isHypertable).Error; err != nil {
		return fmt.Errorf("failed to check if build_logs is hypertable: %w", err)
	}

	if isHypertable {
		log.Println("build_logs is already a TimescaleDB hypertable")
		return nil
	}

	// Convert to hypertable with timestamp as time column and build_id as partition dimension
	// This allows efficient querying by both time and build_id
	// Use 1 hour chunk interval for build logs (builds typically last minutes to hours)
	if err := db.Exec(`
		SELECT create_hypertable('build_logs', 'timestamp', 
			chunk_time_interval => INTERVAL '1 hour',
			if_not_exists => TRUE)
	`).Error; err != nil {
		return fmt.Errorf("failed to create hypertable for build_logs: %w", err)
	}

	log.Println("✓ Converted build_logs to TimescaleDB hypertable (optimized for time-series queries)")

	// Create additional indexes for common query patterns
	// Composite index for querying logs by build_id and timestamp range
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_build_logs_build_id_timestamp 
		ON build_logs (build_id, timestamp DESC)
	`).Error; err != nil {
		log.Printf("Warning: Failed to create composite index: %v", err)
	}

	// Index for line_number ordering within a build
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_build_logs_build_id_line_number 
		ON build_logs (build_id, line_number ASC)
	`).Error; err != nil {
		log.Printf("Warning: Failed to create line_number index: %v", err)
	}

	return nil
}
