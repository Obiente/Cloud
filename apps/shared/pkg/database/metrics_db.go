package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

var MetricsDB *gorm.DB
var metricsInitMu sync.Mutex

// InitMetricsDatabase initializes a separate TimescaleDB/PostgreSQL connection for metrics
func InitMetricsDatabase() error {
	metricsInitMu.Lock()
	defer metricsInitMu.Unlock()
	if MetricsDB != nil {
		return nil
	}
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
	if password == "" {
		return fmt.Errorf("METRICS_DB_PASSWORD or DB_PASSWORD environment variable is required but not set")
	}

	dbname := os.Getenv("METRICS_DB_NAME")
	if dbname == "" {
		dbname = "obiente_metrics" // Default metrics database name
	}

	// Build DSN with reasonable connection timeout for Docker Swarm overlay networks
	// connect_timeout: Time to wait for initial connection (10s - faster failure for debugging)
	// statement_timeout: Maximum time for a query to run (30s) - prevents hanging queries
	// This helps with overlay network initialization delays and slow DNS resolution
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable connect_timeout=10 statement_timeout=30000",
		host, port, user, password, dbname)

	logger.Info("Attempting to connect to metrics database at %s:%s (database: %s, user: %s)", host, port, dbname, user)

	// Diagnostic: Log network information for troubleshooting
	if host != "localhost" && host != "127.0.0.1" {
		logger.Debug("[MetricsDB] Resolving metrics database hostname: %s", host)
		if addrs, err := net.LookupHost(host); err == nil {
			logger.Debug("[MetricsDB] Metrics database hostname resolves to: %v", addrs)
		} else {
			logger.Warn("[MetricsDB] Failed to resolve metrics database hostname %s: %v", host, err)
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
			logger.Warn("Metrics database connection attempt %d/%d failed: %v. Retrying in %v...", attempt, maxRetries, err, retryDelay)
			time.Sleep(retryDelay)
			retryDelay = time.Duration(float64(retryDelay) * 1.5) // Exponential backoff, max 30s
			if retryDelay > 30*time.Second {
				retryDelay = 30 * time.Second
			}
		}
	}

	if err != nil {
		return fmt.Errorf("failed to connect to metrics database (TimescaleDB required) after %d attempts: %w", maxRetries, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := withMetricsSchemaLock(ctx, db, func(conn *gorm.DB) error {
		var installed bool
		if err := conn.Raw("SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb')").Scan(&installed).Error; err != nil {
			return err
		}
		if !installed {
			if err := conn.Exec("CREATE EXTENSION IF NOT EXISTS timescaledb").Error; err != nil {
				logger.Warn("TimescaleDB extension unavailable; using PostgreSQL: %v", err)
			}
		}
		return initMetricsTables(conn)
	}); err != nil {
		if pool, poolErr := db.DB(); poolErr == nil {
			_ = pool.Close()
		}
		return fmt.Errorf("initialize metrics schema: %w", err)
	}
	// Publish only a fully initialized connection; repeated service initialization is a no-op.
	MetricsDB = db

	logger.Info("Metrics database initialized")
	return nil
}

// InitMetricsTables creates and configures metrics tables
func initMetricsTables(db *gorm.DB) error {
	// Check which tables are already hypertables before AutoMigrate
	hypertableMap := make(map[string]bool)
	var hypertableNames []string
	var hasTimescale bool
	if err := db.Raw("SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb')").Scan(&hasTimescale).Error; err != nil {
		return err
	}
	if hasTimescale {
		if err := db.Raw("SELECT hypertable_name FROM timescaledb_information.hypertables WHERE hypertable_schema = 'public'").Scan(&hypertableNames).Error; err != nil {
			return fmt.Errorf("read Timescale schema: %w", err)
		}
	}

	for _, name := range hypertableNames {
		hypertableMap[name] = true
	}

	// Auto-migrate metrics tables, but skip tables that are already hypertables
	// GORM AutoMigrate will fail if it tries to modify hypertable partitioning columns
	tablesToMigrate := []interface{}{}

	if !hypertableMap["deployment_metrics"] {
		tablesToMigrate = append(tablesToMigrate, &DeploymentMetrics{})
	}
	if !hypertableMap["deployment_usage_hourly"] {
		tablesToMigrate = append(tablesToMigrate, &DeploymentUsageHourly{})
	}
	if !hypertableMap["game_server_metrics"] {
		tablesToMigrate = append(tablesToMigrate, &GameServerMetrics{})
	}
	if !hypertableMap["game_server_usage_hourly"] {
		tablesToMigrate = append(tablesToMigrate, &GameServerUsageHourly{})
	}
	if !hypertableMap["build_logs"] {
		tablesToMigrate = append(tablesToMigrate, &BuildLog{})
	}
	if !hypertableMap["deployment_runtime_logs"] {
		tablesToMigrate = append(tablesToMigrate, &DeploymentRuntimeLog{})
	}
	if !hypertableMap["audit_logs"] {
		tablesToMigrate = append(tablesToMigrate, &AuditLog{})
	}
	if !hypertableMap["vps_metrics"] {
		tablesToMigrate = append(tablesToMigrate, &VPSMetrics{})
	}
	if !hypertableMap["vps_usage_hourly"] {
		tablesToMigrate = append(tablesToMigrate, &VPSUsageHourly{})
	}
	if !hypertableMap["database_metrics"] {
		tablesToMigrate = append(tablesToMigrate, &DatabaseMetrics{})
	}
	if !hypertableMap["database_usage_hourly"] {
		tablesToMigrate = append(tablesToMigrate, &DatabaseUsageHourly{})
	}

	if len(tablesToMigrate) > 0 {
		if err := db.AutoMigrate(tablesToMigrate...); err != nil {
			return fmt.Errorf("migrate metrics tables: %w", err)
		}
	}

	// Initialize TimescaleDB hypertable for build_logs
	if err := InitBuildLogsTimescaleDB(db); err != nil {
		logger.Warn("Failed to initialize TimescaleDB hypertable for build_logs: %v", err)
		// Continue anyway - standard PostgreSQL will work fine
	}

	// Initialize TimescaleDB hypertable for audit_logs
	if err := InitAuditLogsTimescaleDB(db); err != nil {
		logger.Warn("Failed to initialize TimescaleDB hypertable for audit_logs: %v", err)
		// Continue anyway - standard PostgreSQL will work fine
	}

	// Create composite indexes for better query performance
	if err := createMetricsIndexes(db); err != nil {
		return fmt.Errorf("failed to create metrics indexes: %w", err)
	}

	// Convert to hypertable if TimescaleDB is available
	// Check if table is already a hypertable
	var isHypertable bool
	if err := db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM timescaledb_information.hypertables 
			WHERE hypertable_name = 'deployment_metrics'
		)
	`).Scan(&isHypertable).Error; err == nil && !isHypertable {
		// Check if primary key already includes timestamp
		var pkIncludesTimestamp bool
		db.Raw(`
			SELECT EXISTS (
				SELECT 1 FROM pg_constraint c
				JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY(c.conkey)
				WHERE c.conrelid = 'deployment_metrics'::regclass
				AND c.contype = 'p'
				AND a.attname = 'timestamp'
			)
		`).Scan(&pkIncludesTimestamp)

		if !pkIncludesTimestamp {
			// Only try to modify primary key if table is empty, otherwise skip hypertable creation
			var rowCount int64
			db.Raw(`SELECT COUNT(*) FROM deployment_metrics`).Scan(&rowCount)

			if rowCount > 0 {
				// Table has data - check for duplicates before modifying primary key
				var duplicateCount int64
				db.Raw(`
					SELECT COUNT(*) FROM (
						SELECT id, timestamp, COUNT(*) as cnt
						FROM deployment_metrics
						GROUP BY id, timestamp
						HAVING COUNT(*) > 1
					) duplicates
				`).Scan(&duplicateCount)

				if duplicateCount > 0 {
					logger.Warn("deployment_metrics has duplicate (id, timestamp) pairs - skipping hypertable creation to preserve functionality")
					isHypertable = true
				} else {
					// Safe to modify primary key
					if err := db.Exec(`ALTER TABLE deployment_metrics DROP CONSTRAINT IF EXISTS deployment_metrics_pkey`).Error; err != nil {
						logger.Warn("Failed to drop primary key for deployment_metrics: %v", err)
						isHypertable = true
					} else {
						if err := db.Exec(`ALTER TABLE deployment_metrics ADD PRIMARY KEY (id, timestamp)`).Error; err != nil {
							logger.Warn("Failed to create composite primary key for deployment_metrics: %v", err)
							// CRITICAL: Restore original primary key - metrics won't work without it
							if restoreErr := db.Exec(`ALTER TABLE deployment_metrics ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
								logger.Error("CRITICAL: Failed to restore primary key for deployment_metrics: %v", restoreErr)
								_ = db.Exec(`ALTER TABLE deployment_metrics ADD CONSTRAINT deployment_metrics_pkey PRIMARY KEY (id)`).Error
							}
							isHypertable = true
						}
					}
				}
			} else {
				// Empty table - safe to modify primary key
				if err := db.Exec(`ALTER TABLE deployment_metrics DROP CONSTRAINT IF EXISTS deployment_metrics_pkey`).Error; err != nil {
					logger.Warn("Failed to drop primary key for deployment_metrics: %v", err)
					isHypertable = true
				} else {
					if err := db.Exec(`ALTER TABLE deployment_metrics ADD PRIMARY KEY (id, timestamp)`).Error; err != nil {
						logger.Warn("Failed to create composite primary key for deployment_metrics: %v", err)
						if restoreErr := db.Exec(`ALTER TABLE deployment_metrics ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
							logger.Error("CRITICAL: Failed to restore primary key for deployment_metrics: %v", restoreErr)
							_ = db.Exec(`ALTER TABLE deployment_metrics ADD CONSTRAINT deployment_metrics_pkey PRIMARY KEY (id)`).Error
						}
						isHypertable = true
					}
				}
			}
		}

		if !isHypertable {
			// Convert to hypertable with 1 hour chunk interval
			if err := db.Exec(`
			SELECT create_hypertable('deployment_metrics', 'timestamp', 
				chunk_time_interval => INTERVAL '1 hour',
					if_not_exists => TRUE,
					migrate_data => TRUE)
		`).Error; err != nil {
				logger.Warn("Failed to create hypertable for deployment_metrics: %v", err)
				// Restore original primary key if hypertable creation failed and we changed it
				if !pkIncludesTimestamp {
					if err := db.Exec(`ALTER TABLE deployment_metrics DROP CONSTRAINT IF EXISTS deployment_metrics_pkey`).Error; err == nil {
						_ = db.Exec(`ALTER TABLE deployment_metrics ADD PRIMARY KEY (id)`).Error
					}
				}
			} else {
				logger.Info("Created TimescaleDB hypertable for deployment_metrics")
			}
		}
	}

	// Convert game_server_metrics to hypertable if available
	if err := db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM timescaledb_information.hypertables 
			WHERE hypertable_name = 'game_server_metrics'
		)
	`).Scan(&isHypertable).Error; err == nil && !isHypertable {
		// Check if primary key already includes timestamp
		var pkIncludesTimestamp bool
		db.Raw(`
			SELECT EXISTS (
				SELECT 1 FROM pg_constraint c
				JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY(c.conkey)
				WHERE c.conrelid = 'game_server_metrics'::regclass
				AND c.contype = 'p'
				AND a.attname = 'timestamp'
			)
		`).Scan(&pkIncludesTimestamp)

		if !pkIncludesTimestamp {
			var rowCount int64
			db.Raw(`SELECT COUNT(*) FROM game_server_metrics`).Scan(&rowCount)

			if rowCount > 0 {
				var duplicateCount int64
				db.Raw(`
					SELECT COUNT(*) FROM (
						SELECT id, timestamp, COUNT(*) as cnt
						FROM game_server_metrics
						GROUP BY id, timestamp
						HAVING COUNT(*) > 1
					) duplicates
				`).Scan(&duplicateCount)

				if duplicateCount > 0 {
					logger.Warn("game_server_metrics has duplicate (id, timestamp) pairs - skipping hypertable creation")
					isHypertable = true
				} else {
					if err := db.Exec(`ALTER TABLE game_server_metrics DROP CONSTRAINT IF EXISTS game_server_metrics_pkey`).Error; err != nil {
						logger.Warn("Failed to drop primary key for game_server_metrics: %v", err)
						isHypertable = true
					} else {
						if err := db.Exec(`ALTER TABLE game_server_metrics ADD PRIMARY KEY (id, timestamp)`).Error; err != nil {
							logger.Warn("Failed to create composite primary key for game_server_metrics: %v", err)
							if restoreErr := db.Exec(`ALTER TABLE game_server_metrics ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
								logger.Error("CRITICAL: Failed to restore primary key for game_server_metrics: %v", restoreErr)
								_ = db.Exec(`ALTER TABLE game_server_metrics ADD CONSTRAINT game_server_metrics_pkey PRIMARY KEY (id)`).Error
							}
							isHypertable = true
						}
					}
				}
			} else {
				if err := db.Exec(`ALTER TABLE game_server_metrics DROP CONSTRAINT IF EXISTS game_server_metrics_pkey`).Error; err != nil {
					logger.Warn("Failed to drop primary key for game_server_metrics: %v", err)
					isHypertable = true
				} else {
					if err := db.Exec(`ALTER TABLE game_server_metrics ADD PRIMARY KEY (id, timestamp)`).Error; err != nil {
						logger.Warn("Failed to create composite primary key for game_server_metrics: %v", err)
						if restoreErr := db.Exec(`ALTER TABLE game_server_metrics ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
							logger.Error("CRITICAL: Failed to restore primary key for game_server_metrics: %v", restoreErr)
							_ = db.Exec(`ALTER TABLE game_server_metrics ADD CONSTRAINT game_server_metrics_pkey PRIMARY KEY (id)`).Error
						}
						isHypertable = true
					}
				}
			}
		}

		if !isHypertable {
			if err := db.Exec(`
				SELECT create_hypertable('game_server_metrics', 'timestamp', 
					chunk_time_interval => INTERVAL '1 hour',
					if_not_exists => TRUE,
					migrate_data => TRUE)
			`).Error; err != nil {
				logger.Warn("Failed to create hypertable for game_server_metrics: %v", err)
				// Restore original primary key if hypertable creation failed and we changed it
				if !pkIncludesTimestamp {
					if err := db.Exec(`ALTER TABLE game_server_metrics DROP CONSTRAINT IF EXISTS game_server_metrics_pkey`).Error; err == nil {
						_ = db.Exec(`ALTER TABLE game_server_metrics ADD PRIMARY KEY (id)`).Error
					}
				}
			} else {
				logger.Info("Created TimescaleDB hypertable for game_server_metrics")
			}
		}
	}

	// Convert hourly aggregates to hypertable if available
	if err := db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM timescaledb_information.hypertables 
			WHERE hypertable_name = 'deployment_usage_hourly'
		)
	`).Scan(&isHypertable).Error; err == nil && !isHypertable {
		// Check if primary key already includes hour
		var pkIncludesHour bool
		db.Raw(`
			SELECT EXISTS (
				SELECT 1 FROM pg_constraint c
				JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY(c.conkey)
				WHERE c.conrelid = 'deployment_usage_hourly'::regclass
				AND c.contype = 'p'
				AND a.attname = 'hour'
			)
		`).Scan(&pkIncludesHour)

		if !pkIncludesHour {
			var rowCount int64
			db.Raw(`SELECT COUNT(*) FROM deployment_usage_hourly`).Scan(&rowCount)

			if rowCount > 0 {
				var duplicateCount int64
				db.Raw(`
					SELECT COUNT(*) FROM (
						SELECT id, hour, COUNT(*) as cnt
						FROM deployment_usage_hourly
						GROUP BY id, hour
						HAVING COUNT(*) > 1
					) duplicates
				`).Scan(&duplicateCount)

				if duplicateCount > 0 {
					logger.Warn("deployment_usage_hourly has duplicate (id, hour) pairs - skipping hypertable creation to preserve functionality")
					isHypertable = true
				} else {
					if err := db.Exec(`ALTER TABLE deployment_usage_hourly DROP CONSTRAINT IF EXISTS deployment_usage_hourly_pkey`).Error; err != nil {
						logger.Warn("Failed to drop primary key for deployment_usage_hourly: %v", err)
						isHypertable = true
					} else {
						if err := db.Exec(`ALTER TABLE deployment_usage_hourly ADD PRIMARY KEY (id, hour)`).Error; err != nil {
							logger.Warn("Failed to create composite primary key for deployment_usage_hourly: %v", err)
							if restoreErr := db.Exec(`ALTER TABLE deployment_usage_hourly ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
								logger.Error("CRITICAL: Failed to restore primary key for deployment_usage_hourly: %v", restoreErr)
								_ = db.Exec(`ALTER TABLE deployment_usage_hourly ADD CONSTRAINT deployment_usage_hourly_pkey PRIMARY KEY (id)`).Error
							}
							isHypertable = true
						}
					}
				}
			} else {
				if err := db.Exec(`ALTER TABLE deployment_usage_hourly DROP CONSTRAINT IF EXISTS deployment_usage_hourly_pkey`).Error; err != nil {
					logger.Warn("Failed to drop primary key for deployment_usage_hourly: %v", err)
					isHypertable = true
				} else {
					if err := db.Exec(`ALTER TABLE deployment_usage_hourly ADD PRIMARY KEY (id, hour)`).Error; err != nil {
						logger.Warn("Failed to create composite primary key for deployment_usage_hourly: %v", err)
						if restoreErr := db.Exec(`ALTER TABLE deployment_usage_hourly ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
							logger.Error("CRITICAL: Failed to restore primary key for deployment_usage_hourly: %v", restoreErr)
							_ = db.Exec(`ALTER TABLE deployment_usage_hourly ADD CONSTRAINT deployment_usage_hourly_pkey PRIMARY KEY (id)`).Error
						}
						isHypertable = true
					}
				}
			}
		}

		if !isHypertable {
			if err := db.Exec(`
			SELECT create_hypertable('deployment_usage_hourly', 'hour', 
				chunk_time_interval => INTERVAL '7 days',
					if_not_exists => TRUE,
					migrate_data => TRUE)
		`).Error; err != nil {
				logger.Warn("Failed to create hypertable for deployment_usage_hourly: %v", err)
				// Restore original primary key if hypertable creation failed and we changed it
				if !pkIncludesHour {
					if err := db.Exec(`ALTER TABLE deployment_usage_hourly DROP CONSTRAINT IF EXISTS deployment_usage_hourly_pkey`).Error; err == nil {
						_ = db.Exec(`ALTER TABLE deployment_usage_hourly ADD PRIMARY KEY (id)`).Error
					}
				}
			} else {
				logger.Info("Created TimescaleDB hypertable for deployment_usage_hourly")
			}
		}
	}

	// Convert game_server_usage_hourly to hypertable if available
	if err := db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM timescaledb_information.hypertables 
			WHERE hypertable_name = 'game_server_usage_hourly'
		)
	`).Scan(&isHypertable).Error; err == nil && !isHypertable {
		// Check if primary key already includes hour
		var pkIncludesHour bool
		db.Raw(`
			SELECT EXISTS (
				SELECT 1 FROM pg_constraint c
				JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY(c.conkey)
				WHERE c.conrelid = 'game_server_usage_hourly'::regclass
				AND c.contype = 'p'
				AND a.attname = 'hour'
			)
		`).Scan(&pkIncludesHour)

		if !pkIncludesHour {
			var rowCount int64
			db.Raw(`SELECT COUNT(*) FROM game_server_usage_hourly`).Scan(&rowCount)

			if rowCount > 0 {
				var duplicateCount int64
				db.Raw(`
					SELECT COUNT(*) FROM (
						SELECT id, hour, COUNT(*) as cnt
						FROM game_server_usage_hourly
						GROUP BY id, hour
						HAVING COUNT(*) > 1
					) duplicates
				`).Scan(&duplicateCount)

				if duplicateCount > 0 {
					logger.Warn("game_server_usage_hourly has duplicate (id, hour) pairs - skipping hypertable creation")
					isHypertable = true
				} else {
					if err := db.Exec(`ALTER TABLE game_server_usage_hourly DROP CONSTRAINT IF EXISTS game_server_usage_hourly_pkey`).Error; err != nil {
						logger.Warn("Failed to drop primary key for game_server_usage_hourly: %v", err)
						isHypertable = true
					} else {
						if err := db.Exec(`ALTER TABLE game_server_usage_hourly ADD PRIMARY KEY (id, hour)`).Error; err != nil {
							logger.Warn("Failed to create composite primary key for game_server_usage_hourly: %v", err)
							if restoreErr := db.Exec(`ALTER TABLE game_server_usage_hourly ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
								logger.Error("CRITICAL: Failed to restore primary key for game_server_usage_hourly: %v", restoreErr)
								_ = db.Exec(`ALTER TABLE game_server_usage_hourly ADD CONSTRAINT game_server_usage_hourly_pkey PRIMARY KEY (id)`).Error
							}
							isHypertable = true
						}
					}
				}
			} else {
				if err := db.Exec(`ALTER TABLE game_server_usage_hourly DROP CONSTRAINT IF EXISTS game_server_usage_hourly_pkey`).Error; err != nil {
					logger.Warn("Failed to drop primary key for game_server_usage_hourly: %v", err)
					isHypertable = true
				} else {
					if err := db.Exec(`ALTER TABLE game_server_usage_hourly ADD PRIMARY KEY (id, hour)`).Error; err != nil {
						logger.Warn("Failed to create composite primary key for game_server_usage_hourly: %v", err)
						if restoreErr := db.Exec(`ALTER TABLE game_server_usage_hourly ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
							logger.Error("CRITICAL: Failed to restore primary key for game_server_usage_hourly: %v", restoreErr)
							_ = db.Exec(`ALTER TABLE game_server_usage_hourly ADD CONSTRAINT game_server_usage_hourly_pkey PRIMARY KEY (id)`).Error
						}
						isHypertable = true
					}
				}
			}
		}

		if !isHypertable {
			if err := db.Exec(`
				SELECT create_hypertable('game_server_usage_hourly', 'hour', 
					chunk_time_interval => INTERVAL '7 days',
					if_not_exists => TRUE,
					migrate_data => TRUE)
			`).Error; err != nil {
				logger.Warn("Failed to create hypertable for game_server_usage_hourly: %v", err)
				if !pkIncludesHour {
					if err := db.Exec(`ALTER TABLE game_server_usage_hourly DROP CONSTRAINT IF EXISTS game_server_usage_hourly_pkey`).Error; err == nil {
						if restoreErr := db.Exec(`ALTER TABLE game_server_usage_hourly ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
							logger.Error("CRITICAL: Failed to restore primary key after hypertable creation failure: %v", restoreErr)
							_ = db.Exec(`ALTER TABLE game_server_usage_hourly ADD CONSTRAINT game_server_usage_hourly_pkey PRIMARY KEY (id)`).Error
						}
					}
				}
			} else {
				logger.Info("Created TimescaleDB hypertable for game_server_usage_hourly")
			}
		}
	}

	return nil
}

// InitMetricsTables is retained for callers that explicitly request schema reconciliation.
func InitMetricsTables() error {
	metricsInitMu.Lock()
	defer metricsInitMu.Unlock()
	if MetricsDB == nil {
		return fmt.Errorf("metrics database is not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	return withMetricsSchemaLock(ctx, MetricsDB, initMetricsTables)
}

// A session advisory lock must be held on the same physical connection as the DDL.
// Separate transactions avoid holding every hypertable/chunk lock until the entire schema finishes.
func withMetricsSchemaLock(ctx context.Context, db *gorm.DB, migrate func(*gorm.DB) error) error {
	return db.WithContext(ctx).Connection(func(conn *gorm.DB) (retErr error) {
		const lockID int64 = 734681092
		for {
			var acquired bool
			if err := conn.Raw("SELECT pg_try_advisory_lock(?)", lockID).Scan(&acquired).Error; err != nil {
				return err
			}
			if acquired {
				break
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(250 * time.Millisecond):
			}
		}
		defer func() {
			cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := conn.WithContext(cleanup).Exec("SELECT pg_advisory_unlock(?)", lockID).Error; err != nil {
				if pinned, ok := conn.Statement.ConnPool.(*sql.Conn); ok {
					_ = pinned.Raw(func(any) error { return driver.ErrBadConn })
				}
				retErr = errors.Join(retErr, fmt.Errorf("release metrics schema lock: %w", err))
			}
		}()
		return migrate(conn)
	})
}
