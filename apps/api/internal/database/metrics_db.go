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
		return fmt.Errorf("failed to connect to metrics database (TimescaleDB required): %w", err)
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
	// Check which tables are already hypertables before AutoMigrate
	hypertableMap := make(map[string]bool)
	var hypertableNames []string
	MetricsDB.Raw(`
		SELECT hypertable_name 
		FROM timescaledb_information.hypertables 
		WHERE hypertable_schema = 'public'
	`).Scan(&hypertableNames)

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
	if !hypertableMap["audit_logs"] {
		tablesToMigrate = append(tablesToMigrate, &AuditLog{})
	}

	if len(tablesToMigrate) > 0 {
		if err := MetricsDB.AutoMigrate(tablesToMigrate...); err != nil {
			logger.Warn("Failed to auto-migrate some metrics tables (may already be hypertables): %v", err)
			// Continue anyway - hypertables don't need AutoMigrate
		}
	}

	// Initialize TimescaleDB hypertable for build_logs
	if err := InitBuildLogsTimescaleDB(MetricsDB); err != nil {
		logger.Warn("Failed to initialize TimescaleDB hypertable for build_logs: %v", err)
		// Continue anyway - standard PostgreSQL will work fine
	}

	// Initialize TimescaleDB hypertable for audit_logs
	if err := InitAuditLogsTimescaleDB(MetricsDB); err != nil {
		logger.Warn("Failed to initialize TimescaleDB hypertable for audit_logs: %v", err)
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
		// Check if primary key already includes timestamp
		var pkIncludesTimestamp bool
		MetricsDB.Raw(`
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
			MetricsDB.Raw(`SELECT COUNT(*) FROM deployment_metrics`).Scan(&rowCount)

			if rowCount > 0 {
				// Table has data - check for duplicates before modifying primary key
				var duplicateCount int64
				MetricsDB.Raw(`
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
					if err := MetricsDB.Exec(`ALTER TABLE deployment_metrics DROP CONSTRAINT IF EXISTS deployment_metrics_pkey`).Error; err != nil {
						logger.Warn("Failed to drop primary key for deployment_metrics: %v", err)
						isHypertable = true
					} else {
						if err := MetricsDB.Exec(`ALTER TABLE deployment_metrics ADD PRIMARY KEY (id, timestamp)`).Error; err != nil {
							logger.Warn("Failed to create composite primary key for deployment_metrics: %v", err)
							// CRITICAL: Restore original primary key - metrics won't work without it
							if restoreErr := MetricsDB.Exec(`ALTER TABLE deployment_metrics ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
								logger.Error("CRITICAL: Failed to restore primary key for deployment_metrics: %v", restoreErr)
								_ = MetricsDB.Exec(`ALTER TABLE deployment_metrics ADD CONSTRAINT deployment_metrics_pkey PRIMARY KEY (id)`).Error
							}
							isHypertable = true
						}
					}
				}
			} else {
				// Empty table - safe to modify primary key
				if err := MetricsDB.Exec(`ALTER TABLE deployment_metrics DROP CONSTRAINT IF EXISTS deployment_metrics_pkey`).Error; err != nil {
					logger.Warn("Failed to drop primary key for deployment_metrics: %v", err)
					isHypertable = true
				} else {
					if err := MetricsDB.Exec(`ALTER TABLE deployment_metrics ADD PRIMARY KEY (id, timestamp)`).Error; err != nil {
						logger.Warn("Failed to create composite primary key for deployment_metrics: %v", err)
						if restoreErr := MetricsDB.Exec(`ALTER TABLE deployment_metrics ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
							logger.Error("CRITICAL: Failed to restore primary key for deployment_metrics: %v", restoreErr)
							_ = MetricsDB.Exec(`ALTER TABLE deployment_metrics ADD CONSTRAINT deployment_metrics_pkey PRIMARY KEY (id)`).Error
						}
						isHypertable = true
					}
				}
			}
		}

		if !isHypertable {
			// Convert to hypertable with 1 hour chunk interval
			if err := MetricsDB.Exec(`
			SELECT create_hypertable('deployment_metrics', 'timestamp', 
				chunk_time_interval => INTERVAL '1 hour',
					if_not_exists => TRUE,
					migrate_data => TRUE)
		`).Error; err != nil {
				logger.Warn("Failed to create hypertable for deployment_metrics: %v", err)
				// Restore original primary key if hypertable creation failed and we changed it
				if !pkIncludesTimestamp {
					if err := MetricsDB.Exec(`ALTER TABLE deployment_metrics DROP CONSTRAINT IF EXISTS deployment_metrics_pkey`).Error; err == nil {
						_ = MetricsDB.Exec(`ALTER TABLE deployment_metrics ADD PRIMARY KEY (id)`).Error
					}
				}
			} else {
				logger.Info("Created TimescaleDB hypertable for deployment_metrics")
			}
		}
	}

	// Convert game_server_metrics to hypertable if available
	if err := MetricsDB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM timescaledb_information.hypertables 
			WHERE hypertable_name = 'game_server_metrics'
		)
	`).Scan(&isHypertable).Error; err == nil && !isHypertable {
		// Check if primary key already includes timestamp
		var pkIncludesTimestamp bool
		MetricsDB.Raw(`
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
			MetricsDB.Raw(`SELECT COUNT(*) FROM game_server_metrics`).Scan(&rowCount)

			if rowCount > 0 {
				var duplicateCount int64
				MetricsDB.Raw(`
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
					if err := MetricsDB.Exec(`ALTER TABLE game_server_metrics DROP CONSTRAINT IF EXISTS game_server_metrics_pkey`).Error; err != nil {
						logger.Warn("Failed to drop primary key for game_server_metrics: %v", err)
						isHypertable = true
					} else {
						if err := MetricsDB.Exec(`ALTER TABLE game_server_metrics ADD PRIMARY KEY (id, timestamp)`).Error; err != nil {
							logger.Warn("Failed to create composite primary key for game_server_metrics: %v", err)
							if restoreErr := MetricsDB.Exec(`ALTER TABLE game_server_metrics ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
								logger.Error("CRITICAL: Failed to restore primary key for game_server_metrics: %v", restoreErr)
								_ = MetricsDB.Exec(`ALTER TABLE game_server_metrics ADD CONSTRAINT game_server_metrics_pkey PRIMARY KEY (id)`).Error
							}
							isHypertable = true
						}
					}
				}
			} else {
				if err := MetricsDB.Exec(`ALTER TABLE game_server_metrics DROP CONSTRAINT IF EXISTS game_server_metrics_pkey`).Error; err != nil {
					logger.Warn("Failed to drop primary key for game_server_metrics: %v", err)
					isHypertable = true
				} else {
					if err := MetricsDB.Exec(`ALTER TABLE game_server_metrics ADD PRIMARY KEY (id, timestamp)`).Error; err != nil {
						logger.Warn("Failed to create composite primary key for game_server_metrics: %v", err)
						if restoreErr := MetricsDB.Exec(`ALTER TABLE game_server_metrics ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
							logger.Error("CRITICAL: Failed to restore primary key for game_server_metrics: %v", restoreErr)
							_ = MetricsDB.Exec(`ALTER TABLE game_server_metrics ADD CONSTRAINT game_server_metrics_pkey PRIMARY KEY (id)`).Error
						}
						isHypertable = true
					}
				}
			}
		}

		if !isHypertable {
			if err := MetricsDB.Exec(`
				SELECT create_hypertable('game_server_metrics', 'timestamp', 
					chunk_time_interval => INTERVAL '1 hour',
					if_not_exists => TRUE,
					migrate_data => TRUE)
			`).Error; err != nil {
				logger.Warn("Failed to create hypertable for game_server_metrics: %v", err)
				// Restore original primary key if hypertable creation failed and we changed it
				if !pkIncludesTimestamp {
					if err := MetricsDB.Exec(`ALTER TABLE game_server_metrics DROP CONSTRAINT IF EXISTS game_server_metrics_pkey`).Error; err == nil {
						_ = MetricsDB.Exec(`ALTER TABLE game_server_metrics ADD PRIMARY KEY (id)`).Error
					}
				}
			} else {
				logger.Info("Created TimescaleDB hypertable for game_server_metrics")
			}
		}
	}

	// Convert hourly aggregates to hypertable if available
	if err := MetricsDB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM timescaledb_information.hypertables 
			WHERE hypertable_name = 'deployment_usage_hourly'
		)
	`).Scan(&isHypertable).Error; err == nil && !isHypertable {
		// Check if primary key already includes hour
		var pkIncludesHour bool
		MetricsDB.Raw(`
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
			MetricsDB.Raw(`SELECT COUNT(*) FROM deployment_usage_hourly`).Scan(&rowCount)

			if rowCount > 0 {
				var duplicateCount int64
				MetricsDB.Raw(`
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
					if err := MetricsDB.Exec(`ALTER TABLE deployment_usage_hourly DROP CONSTRAINT IF EXISTS deployment_usage_hourly_pkey`).Error; err != nil {
						logger.Warn("Failed to drop primary key for deployment_usage_hourly: %v", err)
						isHypertable = true
					} else {
						if err := MetricsDB.Exec(`ALTER TABLE deployment_usage_hourly ADD PRIMARY KEY (id, hour)`).Error; err != nil {
							logger.Warn("Failed to create composite primary key for deployment_usage_hourly: %v", err)
							if restoreErr := MetricsDB.Exec(`ALTER TABLE deployment_usage_hourly ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
								logger.Error("CRITICAL: Failed to restore primary key for deployment_usage_hourly: %v", restoreErr)
								_ = MetricsDB.Exec(`ALTER TABLE deployment_usage_hourly ADD CONSTRAINT deployment_usage_hourly_pkey PRIMARY KEY (id)`).Error
							}
							isHypertable = true
						}
					}
				}
			} else {
				if err := MetricsDB.Exec(`ALTER TABLE deployment_usage_hourly DROP CONSTRAINT IF EXISTS deployment_usage_hourly_pkey`).Error; err != nil {
					logger.Warn("Failed to drop primary key for deployment_usage_hourly: %v", err)
					isHypertable = true
				} else {
					if err := MetricsDB.Exec(`ALTER TABLE deployment_usage_hourly ADD PRIMARY KEY (id, hour)`).Error; err != nil {
						logger.Warn("Failed to create composite primary key for deployment_usage_hourly: %v", err)
						if restoreErr := MetricsDB.Exec(`ALTER TABLE deployment_usage_hourly ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
							logger.Error("CRITICAL: Failed to restore primary key for deployment_usage_hourly: %v", restoreErr)
							_ = MetricsDB.Exec(`ALTER TABLE deployment_usage_hourly ADD CONSTRAINT deployment_usage_hourly_pkey PRIMARY KEY (id)`).Error
						}
						isHypertable = true
					}
				}
			}
		}

		if !isHypertable {
			if err := MetricsDB.Exec(`
			SELECT create_hypertable('deployment_usage_hourly', 'hour', 
				chunk_time_interval => INTERVAL '7 days',
					if_not_exists => TRUE,
					migrate_data => TRUE)
		`).Error; err != nil {
				logger.Warn("Failed to create hypertable for deployment_usage_hourly: %v", err)
				// Restore original primary key if hypertable creation failed and we changed it
				if !pkIncludesHour {
					if err := MetricsDB.Exec(`ALTER TABLE deployment_usage_hourly DROP CONSTRAINT IF EXISTS deployment_usage_hourly_pkey`).Error; err == nil {
						_ = MetricsDB.Exec(`ALTER TABLE deployment_usage_hourly ADD PRIMARY KEY (id)`).Error
					}
				}
			} else {
				logger.Info("Created TimescaleDB hypertable for deployment_usage_hourly")
			}
		}
	}

	// Convert game_server_usage_hourly to hypertable if available
	if err := MetricsDB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM timescaledb_information.hypertables 
			WHERE hypertable_name = 'game_server_usage_hourly'
		)
	`).Scan(&isHypertable).Error; err == nil && !isHypertable {
		// Check if primary key already includes hour
		var pkIncludesHour bool
		MetricsDB.Raw(`
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
			MetricsDB.Raw(`SELECT COUNT(*) FROM game_server_usage_hourly`).Scan(&rowCount)

			if rowCount > 0 {
				var duplicateCount int64
				MetricsDB.Raw(`
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
					if err := MetricsDB.Exec(`ALTER TABLE game_server_usage_hourly DROP CONSTRAINT IF EXISTS game_server_usage_hourly_pkey`).Error; err != nil {
						logger.Warn("Failed to drop primary key for game_server_usage_hourly: %v", err)
						isHypertable = true
					} else {
						if err := MetricsDB.Exec(`ALTER TABLE game_server_usage_hourly ADD PRIMARY KEY (id, hour)`).Error; err != nil {
							logger.Warn("Failed to create composite primary key for game_server_usage_hourly: %v", err)
							if restoreErr := MetricsDB.Exec(`ALTER TABLE game_server_usage_hourly ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
								logger.Error("CRITICAL: Failed to restore primary key for game_server_usage_hourly: %v", restoreErr)
								_ = MetricsDB.Exec(`ALTER TABLE game_server_usage_hourly ADD CONSTRAINT game_server_usage_hourly_pkey PRIMARY KEY (id)`).Error
							}
							isHypertable = true
						}
					}
				}
			} else {
				if err := MetricsDB.Exec(`ALTER TABLE game_server_usage_hourly DROP CONSTRAINT IF EXISTS game_server_usage_hourly_pkey`).Error; err != nil {
					logger.Warn("Failed to drop primary key for game_server_usage_hourly: %v", err)
					isHypertable = true
				} else {
					if err := MetricsDB.Exec(`ALTER TABLE game_server_usage_hourly ADD PRIMARY KEY (id, hour)`).Error; err != nil {
						logger.Warn("Failed to create composite primary key for game_server_usage_hourly: %v", err)
						if restoreErr := MetricsDB.Exec(`ALTER TABLE game_server_usage_hourly ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
							logger.Error("CRITICAL: Failed to restore primary key for game_server_usage_hourly: %v", restoreErr)
							_ = MetricsDB.Exec(`ALTER TABLE game_server_usage_hourly ADD CONSTRAINT game_server_usage_hourly_pkey PRIMARY KEY (id)`).Error
						}
						isHypertable = true
					}
				}
			}
		}

		if !isHypertable {
			if err := MetricsDB.Exec(`
				SELECT create_hypertable('game_server_usage_hourly', 'hour', 
					chunk_time_interval => INTERVAL '7 days',
					if_not_exists => TRUE,
					migrate_data => TRUE)
			`).Error; err != nil {
				logger.Warn("Failed to create hypertable for game_server_usage_hourly: %v", err)
				if !pkIncludesHour {
					if err := MetricsDB.Exec(`ALTER TABLE game_server_usage_hourly DROP CONSTRAINT IF EXISTS game_server_usage_hourly_pkey`).Error; err == nil {
						if restoreErr := MetricsDB.Exec(`ALTER TABLE game_server_usage_hourly ADD PRIMARY KEY (id)`).Error; restoreErr != nil {
							logger.Error("CRITICAL: Failed to restore primary key after hypertable creation failure: %v", restoreErr)
							_ = MetricsDB.Exec(`ALTER TABLE game_server_usage_hourly ADD CONSTRAINT game_server_usage_hourly_pkey PRIMARY KEY (id)`).Error
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
