package database

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

var DB *gorm.DB
var RedisClient *RedisCache

// customGormLogger is a GORM logger that filters out "record not found" errors at warn level
type customGormLogger struct {
	gormlogger.Interface
	logLevel string
}

func (l *customGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &customGormLogger{
		Interface: l.Interface.LogMode(level),
		logLevel: l.logLevel,
	}
}

func (l *customGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	// Filter out "record not found" errors at warn level or higher
	if l.logLevel == "warn" || l.logLevel == "warning" || l.logLevel == "error" {
		for _, d := range data {
			if err, ok := d.(error); ok {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// Suppress "record not found" errors at warn/error level
					return
				}
			}
		}
	}
	l.Interface.Error(ctx, msg, data...)
}

// getGormLogger returns a custom GORM logger that respects DB_LOG_LEVEL (or falls back to LOG_LEVEL)
func getGormLogger() gormlogger.Interface {
	// Get DB_LOG_LEVEL, fallback to LOG_LEVEL if not set
	dbLogLevel := os.Getenv("DB_LOG_LEVEL")
	if dbLogLevel == "" {
		// Fallback to application LOG_LEVEL
		logger.Init()
		dbLogLevel = logger.GetLevel()
	} else {
		dbLogLevel = strings.ToLower(strings.TrimSpace(dbLogLevel))
	}
	
	var gormLevel gormlogger.LogLevel
	switch dbLogLevel {
	case "error":
		gormLevel = gormlogger.Error // Only errors
	case "warn", "warning":
		gormLevel = gormlogger.Error // Errors only (suppress SQL queries and record not found)
	case "info":
		gormLevel = gormlogger.Error // Suppress SQL queries at info level
	case "debug", "trace":
		gormLevel = gormlogger.Info // All SQL queries (only for debug/trace)
	default:
		gormLevel = gormlogger.Error // Default to error to suppress SQL queries
	}
	
	logger.Debug("[DB] Database logging level: %s (GORM level: %v)", dbLogLevel, gormLevel)
	
	// Create custom logger that filters out "record not found" at warn level
	return &customGormLogger{
		Interface: gormlogger.Default.LogMode(gormLevel),
		logLevel:  dbLogLevel,
	}
}

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

	// Build DSN with increased connection timeout for Docker Swarm overlay networks
	// connect_timeout: Time to wait for initial connection (default 5s, increased to 60s)
	// statement_timeout: Maximum time for a query to run (30s) - prevents hanging queries
	// This helps with overlay network initialization delays and slow DNS resolution
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable connect_timeout=60 statement_timeout=30000",
		host, port, user, password, dbname)

	logger.Info("Attempting to connect to database at %s:%s (database: %s, user: %s)", host, port, dbname, user)
	
	// Diagnostic: Log network information for troubleshooting worker node connectivity
	if host != "localhost" && host != "127.0.0.1" {
		logger.Debug("[DB] Resolving database hostname: %s", host)
		if addrs, err := net.LookupHost(host); err == nil {
			logger.Debug("[DB] Database hostname resolves to: %v", addrs)
		} else {
			logger.Warn("[DB] Failed to resolve database hostname %s: %v", host, err)
		}
	}

	// Retry database connection with exponential backoff
	// This handles cases where DNS resolution isn't ready yet (common in Docker Swarm)
	maxRetries := 10
	retryDelay := 2 * time.Second
	var db *gorm.DB
	var err error
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Note: gorm.Open doesn't accept context directly, but the underlying driver
		// will respect the connect_timeout in the DSN (set to 60s above)
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: getGormLogger(),
		})
		
		if err == nil {
			// Configure sql.DB connection pool settings for better reliability
			sqlDB, err := db.DB()
			if err == nil {
				// Set connection pool timeouts
				sqlDB.SetConnMaxLifetime(5 * time.Minute)
				sqlDB.SetConnMaxIdleTime(1 * time.Minute)
				sqlDB.SetMaxOpenConns(25)
				sqlDB.SetMaxIdleConns(5)
			}
			break
		}
		
		if attempt < maxRetries {
			logger.Warn("Database connection attempt %d/%d failed: %v. Retrying in %v...", attempt, maxRetries, err, retryDelay)
			time.Sleep(retryDelay)
			retryDelay = time.Duration(float64(retryDelay) * 1.5) // Exponential backoff, max 30s
			if retryDelay > 30*time.Second {
				retryDelay = 30 * time.Second
			}
		}
	}
	
	if err != nil {
		return fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
	}

	DB = db
	logger.Info("Database connection established")

	// Pre-create groups column if it doesn't exist to avoid GORM AutoMigrate syntax issues
	// This prevents GORM from trying to add it with incorrect default value syntax
	if !db.Migrator().HasColumn("deployments", "groups") {
		logger.Debug("Creating groups column before AutoMigrate to avoid syntax issues...")
		if err := db.Exec(`ALTER TABLE deployments ADD COLUMN groups JSONB`).Error; err != nil {
			// Column might have been created by another process, check again
			if !db.Migrator().HasColumn("deployments", "groups") {
				logger.Warn("Failed to pre-create groups column: %v. AutoMigrate may fail, but migration will handle it.", err)
			}
		} else {
			// Set default after adding column to avoid quote escaping issues
			if err := db.Exec(`ALTER TABLE deployments ALTER COLUMN groups SET DEFAULT '[]'::jsonb`).Error; err != nil {
				logger.Warn("Failed to set groups default: %v", err)
			}
		}
	}

	// Auto-migrate the schema (build_logs is stored in TimescaleDB, not here)
	if err := db.AutoMigrate(
		&Deployment{},
		&BuildHistory{},
		&DelegatedDNSRecord{},
		&DNSDelegationAPIKey{},
		&OrganizationPlan{},
		&OrgQuota{},
		&MonthlyCreditGrant{},
		&MonthlyBill{},
		&StrayContainer{},
		&VPSInstance{},
		&VPSSizeCatalog{},
		&VPSRegionCatalog{},
		&SSHKey{},
		&VPSTerminalKey{},
		&VPSBastionKey{},
	); err != nil {
		return fmt.Errorf("failed to auto-migrate: %w", err)
	}

	// Initialize VPS catalog with default sizes and regions
	if err := InitVPSCatalog(); err != nil {
		logger.Warn("Failed to initialize VPS catalog: %v", err)
		// Non-fatal - catalog can be initialized later
	}

	logger.Info("Database schema migrated")

	// Initialize deployment tracking tables
	if err := InitDeploymentTracking(); err != nil {
		return fmt.Errorf("failed to initialize deployment tracking: %w", err)
	}

	logger.Info("Deployment tracking initialized")

	// Initialize metrics database (separate connection for metrics)
	if err := InitMetricsDatabase(); err != nil {
		logger.Warn("Metrics database initialization failed: %v. Metrics may not work correctly.", err)
		// Don't fail main initialization if metrics DB fails
	}

	return nil
}

// InitAuditLogsTimescaleDB converts audit_logs table to TimescaleDB hypertable if available
// This provides better performance for time-series audit log data
// Note: This should be called with MetricsDB (TimescaleDB connection), not the main DB
func InitAuditLogsTimescaleDB(db *gorm.DB) error {
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
		logger.Info("TimescaleDB extension enabled for audit_logs")
	}

	// Check if audit_logs table exists
	if !db.Migrator().HasTable("audit_logs") {
		logger.Debug("audit_logs table does not exist yet, skipping TimescaleDB conversion")
		return nil
	}

	// Check if table is already a hypertable
	var isHypertable bool
	if err := db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM timescaledb_information.hypertables 
			WHERE hypertable_name = 'audit_logs'
		)
	`).Scan(&isHypertable).Error; err != nil {
		return fmt.Errorf("failed to check if audit_logs is hypertable: %w", err)
	}

	if isHypertable {
		logger.Debug("audit_logs is already a TimescaleDB hypertable")
		return nil
	}

	// Check if table has data
	var rowCount int64
	if err := db.Table("audit_logs").Count(&rowCount).Error; err != nil {
		// Table might not exist yet, which is fine - it will be created by AutoMigrate
		logger.Debug("Could not check audit_logs row count (table may not exist): %v", err)
		rowCount = 0
	}

	// TimescaleDB requires that unique indexes include the partitioning column (created_at)
	// We need to drop the primary key constraint and recreate it as a unique index with created_at
	// First, check if there's a primary key constraint
	var hasPK bool
	if err := db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM pg_constraint 
			WHERE conrelid = 'audit_logs'::regclass 
			AND contype = 'p'
		)
	`).Scan(&hasPK).Error; err == nil && hasPK {
		logger.Debug("Dropping primary key constraint on audit_logs to prepare for TimescaleDB conversion...")
		// Drop the primary key constraint
		if err := db.Exec("ALTER TABLE audit_logs DROP CONSTRAINT audit_logs_pkey").Error; err != nil {
			logger.Warn("Failed to drop primary key constraint (may not exist): %v", err)
		}
		// Create a unique index that includes created_at (required for TimescaleDB)
		if err := db.Exec(`
			CREATE UNIQUE INDEX IF NOT EXISTS audit_logs_id_created_at_unique 
			ON audit_logs (id, created_at)
		`).Error; err != nil {
			logger.Warn("Failed to create unique index with created_at: %v", err)
		} else {
			logger.Debug("Created unique index on (id, created_at) for TimescaleDB compatibility")
		}
	}

	// Convert to hypertable with created_at as time column
	// Use 1 day chunk interval for audit logs (audit logs are queried by time ranges)
	// If table has data, use migrate_data => TRUE to convert it
	if rowCount > 0 {
		logger.Info("Converting existing audit_logs table (%d rows) to TimescaleDB hypertable...", rowCount)
		if err := db.Exec(`
			SELECT create_hypertable('audit_logs', 'created_at', 
				chunk_time_interval => INTERVAL '1 day',
				if_not_exists => TRUE,
				migrate_data => TRUE)
		`).Error; err != nil {
			return fmt.Errorf("failed to create hypertable for audit_logs (with data migration): %w", err)
		}
		logger.Info("✓ Converted existing audit_logs table to TimescaleDB hypertable (migrated %d rows)", rowCount)
	} else {
		// Table is empty, can create hypertable normally
		if err := db.Exec(`
			SELECT create_hypertable('audit_logs', 'created_at', 
				chunk_time_interval => INTERVAL '1 day',
				if_not_exists => TRUE)
		`).Error; err != nil {
			return fmt.Errorf("failed to create hypertable for audit_logs: %w", err)
		}
		logger.Info("✓ Converted audit_logs to TimescaleDB hypertable (optimized for time-series queries)")
	}

	// Create additional indexes for common query patterns
	// Composite index for querying logs by organization and time range
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_audit_logs_org_created_at 
		ON audit_logs (organization_id, created_at DESC)
		WHERE organization_id IS NOT NULL
	`).Error; err != nil {
		logger.Warn("Failed to create organization index: %v", err)
	}

	// Composite index for querying logs by resource type and ID
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_created_at 
		ON audit_logs (resource_type, resource_id, created_at DESC)
		WHERE resource_type IS NOT NULL AND resource_id IS NOT NULL
	`).Error; err != nil {
		logger.Warn("Failed to create resource index: %v", err)
	}

	// Index for querying by user
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_audit_logs_user_created_at 
		ON audit_logs (user_id, created_at DESC)
	`).Error; err != nil {
		logger.Warn("Failed to create user index: %v", err)
	}

	// Index for querying by service and action
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_audit_logs_service_action_created_at 
		ON audit_logs (service, action, created_at DESC)
	`).Error; err != nil {
		logger.Warn("Failed to create service/action index: %v", err)
	}

	return nil
}

// InitRedis initializes the Redis connection
func InitRedis() error {
	// Will be implemented separately
	// For now, return nil if Redis is not configured
	if os.Getenv("REDIS_URL") == "" {
		logger.Info("Redis not configured, running without cache")
		return nil
	}

	client := NewRedisCache()
	if err := client.Connect(); err != nil {
		logger.Warn("Failed to connect to Redis: %v", err)
		return nil // Don't fail if Redis is unavailable
	}

	RedisClient = client
	logger.Info("Redis connection established")
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
		logger.Info("TimescaleDB extension enabled for build_logs")
	}

	// Check if build_logs table exists
	if !db.Migrator().HasTable("build_logs") {
		logger.Debug("build_logs table does not exist yet, skipping TimescaleDB conversion")
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
		logger.Debug("build_logs is already a TimescaleDB hypertable")
		return nil
	}

	// TimescaleDB requires that unique indexes include the partitioning column (timestamp)
	// We need to drop the primary key constraint and recreate it as a unique index with timestamp
	// First, check if there's a primary key constraint
	var hasPK bool
	if err := db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM pg_constraint 
			WHERE conrelid = 'build_logs'::regclass 
			AND contype = 'p'
		)
	`).Scan(&hasPK).Error; err == nil && hasPK {
		logger.Debug("Dropping primary key constraint on build_logs to prepare for TimescaleDB conversion...")
		// Drop the primary key constraint
		if err := db.Exec("ALTER TABLE build_logs DROP CONSTRAINT build_logs_pkey").Error; err != nil {
			logger.Warn("Failed to drop primary key constraint (may not exist): %v", err)
		}
		// Create a unique index that includes timestamp (required for TimescaleDB)
		if err := db.Exec(`
			CREATE UNIQUE INDEX IF NOT EXISTS build_logs_id_timestamp_unique 
			ON build_logs (id, timestamp)
		`).Error; err != nil {
			logger.Warn("Failed to create unique index with timestamp: %v", err)
		} else {
			logger.Debug("Created unique index on (id, timestamp) for TimescaleDB compatibility")
		}
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

	logger.Info("✓ Converted build_logs to TimescaleDB hypertable (optimized for time-series queries)")

	// Create additional indexes for common query patterns
	// Composite index for querying logs by build_id and timestamp range
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_build_logs_build_id_timestamp 
		ON build_logs (build_id, timestamp DESC)
	`).Error; err != nil {
		logger.Warn("Failed to create composite index: %v", err)
	}

	// Index for line_number ordering within a build
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_build_logs_build_id_line_number 
		ON build_logs (build_id, line_number ASC)
	`).Error; err != nil {
		logger.Warn("Failed to create line_number index: %v", err)
	}

	return nil
}
