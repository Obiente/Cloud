package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

var DB *gorm.DB
var RedisClient *RedisCache

var (
	registeredModelsMu     sync.RWMutex
	registeredModels       []interface{}
	registeredModelTypeSet = make(map[reflect.Type]struct{})
)

// Use two-parameter advisory lock to avoid uint32 overflow in pg_locks.objid
// Split "obiente" (0x6f6269656e7465) into two 32-bit parts:
// classid: 0x006f6269 (first 3 bytes "obi", padded to 4) = 7302761
// objid: 0x656e7465 (last 4 bytes "ente") = 1702258789
const migrationAdvisoryLockClassID int32 = 0x006f6269 // "obi" - first 3 bytes (padded to 4 bytes)
const migrationAdvisoryLockObjID int32 = 0x656e7465  // "ente" - last 4 bytes
const migrationLockMaxAge = 10 * time.Minute           // Consider lock stuck if held for > 10 minutes
const migrationLockIdleMaxAge = 2 * time.Minute        // Consider lock stuck if held by idle connection for > 2 minutes

// RegisterModels allows services to register the GORM models they depend on so
// they can be migrated when the service initializes its database connection.
// This keeps migrations service-scoped, ensuring we only create the tables
// needed by the binaries that actually run.
func RegisterModels(models ...interface{}) {
	registeredModelsMu.Lock()
	defer registeredModelsMu.Unlock()

	for _, model := range models {
		if model == nil {
			continue
		}

		modelType := reflect.TypeOf(model)
		if modelType.Kind() != reflect.Pointer {
			logger.Warn("RegisterModels expects pointer types, got %T", model)
			continue
		}

		if _, exists := registeredModelTypeSet[modelType]; exists {
			continue
		}

		registeredModelTypeSet[modelType] = struct{}{}
		registeredModels = append(registeredModels, model)
	}
}

func getRegisteredModels() []interface{} {
	registeredModelsMu.RLock()
	defer registeredModelsMu.RUnlock()

	if len(registeredModels) == 0 {
		return nil
	}

	out := make([]interface{}, len(registeredModels))
	copy(out, registeredModels)
	return out
}

// checkAndReleaseStuckLock checks if the migration lock is held by a dead/stale connection
// and attempts to release it. Returns true if the lock was released, false otherwise.
func checkAndReleaseStuckLock(db *gorm.DB) bool {
	// Check if lock is held and by which backend PID
	var lockHolderPID *int64
	err := db.Raw(`
		SELECT l.pid
		FROM pg_locks l
		WHERE l.locktype = 'advisory' 
		AND l.classid = $1
		AND l.objid = $2
		AND l.granted = true
		LIMIT 1
	`, migrationAdvisoryLockClassID, migrationAdvisoryLockObjID).Scan(&lockHolderPID).Error
	
	if err != nil || lockHolderPID == nil {
		// Lock is not held or query failed
		return false
	}
	
	// Check if the backend process is still active
	// If the PID doesn't exist in pg_stat_activity, the connection is definitely dead
	var backendActive bool
	err = db.Raw(`
		SELECT EXISTS(
			SELECT 1 FROM pg_stat_activity 
			WHERE pid = $1
		)
	`, *lockHolderPID).Scan(&backendActive).Error
	
	if err != nil {
		logger.Warn("Failed to check if lock holder backend is active: %v", err)
		return false
	}
	
		if !backendActive {
		logger.Warn("Migration lock (%d, %d) is held by dead backend PID %d, attempting to terminate...", 
			migrationAdvisoryLockClassID, migrationAdvisoryLockObjID, *lockHolderPID)
		// Try to terminate the dead backend (requires appropriate privileges)
		if err := db.Exec("SELECT pg_terminate_backend($1)", *lockHolderPID).Error; err != nil {
			logger.Warn("Failed to terminate dead backend %d: %v (may require superuser privileges)", *lockHolderPID, err)
			return false
		}
		logger.Info("Terminated dead backend %d holding migration lock", *lockHolderPID)
		// Give PostgreSQL a moment to release the lock
		time.Sleep(500 * time.Millisecond)
		return true
	}
	
	// Backend is active - check if it's idle or actively running
	// If idle for too long, it's likely stuck. If running for too long, also consider it stuck.
	type backendInfo struct {
		State       string     `gorm:"column:state"`
		QueryStart  *time.Time `gorm:"column:query_start"`
		StateChange *time.Time `gorm:"column:state_change"`
	}
	
	var info backendInfo
	err = db.Raw(`
		SELECT state, query_start, state_change
		FROM pg_stat_activity
		WHERE pid = $1
		LIMIT 1
	`, *lockHolderPID).Scan(&info).Error
	
	if err != nil {
		logger.Warn("Failed to get backend info for PID %d: %v", *lockHolderPID, err)
		return false
	}
	
	// Determine how long the connection has been in its current state
	var stateAge time.Duration
	var referenceTime *time.Time
	
	if info.State == "idle" || info.State == "idle in transaction" {
		// For idle connections, use state_change (when it became idle)
		if info.StateChange != nil {
			stateAge = time.Since(*info.StateChange)
			referenceTime = info.StateChange
		}
	} else {
		// For active connections, use query_start
		if info.QueryStart != nil {
			stateAge = time.Since(*info.QueryStart)
			referenceTime = info.QueryStart
		}
	}
	
	// If connection is idle for too long, it's definitely stuck
	if (info.State == "idle" || info.State == "idle in transaction") && stateAge > migrationLockIdleMaxAge {
		logger.Warn("Migration lock (%d, %d) is held by idle backend PID %d (idle for %v), terminating...", 
			migrationAdvisoryLockClassID, migrationAdvisoryLockObjID, *lockHolderPID, stateAge)
		if err := db.Exec("SELECT pg_terminate_backend($1)", *lockHolderPID).Error; err != nil {
			logger.Warn("Failed to terminate idle backend %d: %v", *lockHolderPID, err)
			return false
		}
		logger.Info("Terminated idle backend %d holding migration lock", *lockHolderPID)
		time.Sleep(500 * time.Millisecond)
		return true
	}
	
	// If actively running for too long, also consider it stuck (migrations shouldn't take > 10 minutes)
	if info.State != "idle" && info.State != "idle in transaction" && stateAge > migrationLockMaxAge {
		logger.Warn("Migration lock (%d, %d) has been held for %v by backend PID %d (state: %s), may be stuck. Terminating...", 
			migrationAdvisoryLockClassID, migrationAdvisoryLockObjID, stateAge, *lockHolderPID, info.State)
		if err := db.Exec("SELECT pg_terminate_backend($1)", *lockHolderPID).Error; err != nil {
			logger.Warn("Failed to terminate long-running backend %d: %v", *lockHolderPID, err)
			return false
		}
		logger.Info("Terminated long-running backend %d holding migration lock", *lockHolderPID)
		time.Sleep(500 * time.Millisecond)
		return true
	}
	
	// Lock is legitimately held by an active, recent operation
	if referenceTime != nil {
		logger.Debug("Migration lock (%d, %d) is held by backend PID %d (state: %s, age: %v)", 
			migrationAdvisoryLockClassID, migrationAdvisoryLockObjID, *lockHolderPID, info.State, stateAge)
	}
	
	return false
}

// ReleaseStuckMigrationLock attempts to release a stuck migration lock by terminating
// the backend process holding it. This can be called manually if needed.
// Returns true if a stuck lock was found and released, false otherwise.
func ReleaseStuckMigrationLock(db *gorm.DB) bool {
	return checkAndReleaseStuckLock(db)
}

func acquireMigrationLock(db *gorm.DB) (func(), error) {
	logger.Debug("Acquiring advisory lock (%d, %d) for database migrations...", migrationAdvisoryLockClassID, migrationAdvisoryLockObjID)
	
	// Get a single connection from the pool to ensure we use the same connection
	// for both acquire and release, since PostgreSQL advisory locks are session-scoped
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	
	// Get a single connection that we'll use for both acquire and release
	var conn *sql.Conn
	conn, err = sqlDB.Conn(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}
	
	// Use pg_try_advisory_lock instead of pg_advisory_lock to avoid blocking
	// and being canceled by statement_timeout. Retry with exponential backoff.
	maxRetries := 30
	retryDelay := 1 * time.Second
	var acquired bool
	stuckLockCheckInterval := 5 // Check for stuck locks every 5 attempts
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		var result bool
		if err := conn.QueryRowContext(context.Background(), "SELECT pg_try_advisory_lock($1, $2)", migrationAdvisoryLockClassID, migrationAdvisoryLockObjID).Scan(&result); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to try advisory lock: %w", err)
		}
		
		if result {
			acquired = true
			logger.Debug("Successfully acquired advisory lock (%d, %d) on attempt %d", migrationAdvisoryLockClassID, migrationAdvisoryLockObjID, attempt)
			break
		}
		
		// Periodically check if the lock is held by a dead connection
		if attempt%stuckLockCheckInterval == 0 {
			if checkAndReleaseStuckLock(db) {
				// Lock was released, try again immediately
				logger.Debug("Stuck lock was released, retrying lock acquisition...")
				continue
			}
		}
		
		if attempt < maxRetries {
			logger.Debug("Advisory lock (%d, %d) is held by another process, retrying in %v (attempt %d/%d)...", 
				migrationAdvisoryLockClassID, migrationAdvisoryLockObjID, retryDelay, attempt, maxRetries)
			time.Sleep(retryDelay)
			// Exponential backoff, max 5 seconds
			retryDelay = time.Duration(float64(retryDelay) * 1.5)
			if retryDelay > 5*time.Second {
				retryDelay = 5 * time.Second
			}
		}
	}
	
	if !acquired {
		// Final aggressive attempt: after all retries failed, be more aggressive
		// Check if lock has been held for > 1 minute and terminate regardless of state
		type lockInfo struct {
			PID         *int64     `gorm:"column:pid"`
			State       string     `gorm:"column:state"`
			QueryStart  *time.Time `gorm:"column:query_start"`
			StateChange *time.Time `gorm:"column:state_change"`
		}
		
		var info lockInfo
		err := db.Raw(`
			SELECT l.pid, a.state, a.query_start, a.state_change
			FROM pg_locks l
			LEFT JOIN pg_stat_activity a ON l.pid = a.pid
			WHERE l.locktype = 'advisory' 
			AND l.classid = $1
			AND l.objid = $2
			AND l.granted = true
			LIMIT 1
		`, migrationAdvisoryLockClassID, migrationAdvisoryLockObjID).Scan(&info).Error
		
		if err == nil && info.PID != nil {
			// Calculate how long lock has been held
			var lockAge time.Duration
			if info.State == "idle" || info.State == "idle in transaction" {
				if info.StateChange != nil {
					lockAge = time.Since(*info.StateChange)
				}
			} else {
				if info.QueryStart != nil {
					lockAge = time.Since(*info.QueryStart)
				}
			}
			
			// If held for > 1 minute, terminate it (we've already waited long enough)
			if lockAge > 1*time.Minute {
				logger.Warn("After %d failed attempts, migration lock (%d, %d) has been held for %v by backend PID %d (state: %s). Force terminating...", 
					maxRetries, migrationAdvisoryLockClassID, migrationAdvisoryLockObjID, lockAge, *info.PID, info.State)
				if err := db.Exec("SELECT pg_terminate_backend($1)", *info.PID).Error; err == nil {
					logger.Info("Force terminated backend %d holding migration lock", *info.PID)
					time.Sleep(1 * time.Second)
					// Try one more time
					var result bool
					if err := conn.QueryRowContext(context.Background(), "SELECT pg_try_advisory_lock($1, $2)", migrationAdvisoryLockClassID, migrationAdvisoryLockObjID).Scan(&result); err == nil && result {
						acquired = true
						logger.Info("Acquired migration lock after force-terminating stuck lock")
					}
				} else {
					logger.Warn("Failed to force-terminate backend %d: %v", *info.PID, err)
				}
			}
		}
		
		// Also try the regular stuck lock check
		if !acquired && checkAndReleaseStuckLock(db) {
			// Try one more time
			var result bool
			if err := conn.QueryRowContext(context.Background(), "SELECT pg_try_advisory_lock($1, $2)", migrationAdvisoryLockClassID, migrationAdvisoryLockObjID).Scan(&result); err == nil && result {
				acquired = true
				logger.Info("Acquired migration lock after releasing stuck lock")
			}
		}
		
		if !acquired {
			conn.Close()
			return nil, fmt.Errorf("failed to acquire advisory lock (%d, %d) after %d attempts. "+
				"The lock may be held by an active migration. If it's truly stuck, the system will automatically release it on the next attempt", 
				migrationAdvisoryLockClassID, migrationAdvisoryLockObjID, maxRetries)
		}
	}

	released := false
	release := func() {
		if released {
			return
		}
		released = true
		// Use the same connection for unlock to ensure we own the lock
		if _, err := conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1, $2)", migrationAdvisoryLockClassID, migrationAdvisoryLockObjID); err != nil {
			logger.Warn("Failed to release migration advisory lock: %v", err)
		} else {
			logger.Debug("Released advisory lock (%d, %d) after migrations", migrationAdvisoryLockClassID, migrationAdvisoryLockObjID)
		}
		// Return the connection to the pool
		conn.Close()
	}

	return release, nil
}

// customGormLogger is a GORM logger that filters out "record not found" errors at warn level
type customGormLogger struct {
	gormlogger.Interface
	logLevel string
}

func (l *customGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &customGormLogger{
		Interface: l.Interface.LogMode(level),
		logLevel:  l.logLevel,
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

	// Build DSN with reasonable connection timeout for Docker Swarm overlay networks
	// connect_timeout: Time to wait for initial connection (10s - faster failure for debugging)
	// statement_timeout: Maximum time for a query to run (30s) - prevents hanging queries
	// This helps with overlay network initialization delays and slow DNS resolution
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable connect_timeout=10 statement_timeout=30000",
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

	releaseLock, err := acquireMigrationLock(db)
	if err != nil {
		return fmt.Errorf("failed to acquire migration lock: %w", err)
	}
	defer releaseLock()

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
		&VPSPublicIP{},
		&DHCPLease{},
		&SSHKey{},
		&VPSTerminalKey{},
		&VPSBastionKey{},
		&Notification{},
	); err != nil {
		return fmt.Errorf("failed to auto-migrate: %w", err)
	}

	if extraModels := getRegisteredModels(); len(extraModels) > 0 {
		logger.Debug("Auto-migrating %d service-registered models", len(extraModels))
		if err := db.AutoMigrate(extraModels...); err != nil {
			return fmt.Errorf("failed to auto-migrate registered models: %w", err)
		}
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
	// Only initialize if explicitly requested via METRICS_DB_NAME or if service needs it
	// This prevents services that don't need metrics from trying to connect to metrics DB
	metricsDBName := os.Getenv("METRICS_DB_NAME")
	shouldInitMetrics := metricsDBName != "" || os.Getenv("INIT_METRICS_DB") == "true"

	if shouldInitMetrics {
		if err := InitMetricsDatabase(); err != nil {
			logger.Warn("Metrics database initialization failed: %v. Metrics may not work correctly.", err)
			// Don't fail main initialization if metrics DB fails
		}
	} else {
		logger.Debug("Skipping metrics database initialization (not required for this service)")
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
	// Check if Redis is configured - either via REDIS_URL or REDIS_HOST
	redisURL := os.Getenv("REDIS_URL")
	redisHost := os.Getenv("REDIS_HOST")

	// If neither REDIS_URL nor REDIS_HOST is set, Redis is not configured
	if redisURL == "" && redisHost == "" {
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
