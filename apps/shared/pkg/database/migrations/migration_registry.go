package migrations

import (
	"fmt"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"

	"gorm.io/gorm"
)

// RegisterMigrations adds all migrations to the registry
func RegisterMigrations(registry *MigrationRegistry) {
	// Register migrations in order
	// Format: YYYY_MM_DD_###_description
	registry.Register("2025_10_28_001", "Initial schema setup", initialSchema)
	registry.Register("2025_10_28_002", "Add health status to deployments", addHealthStatus)
	registry.Register("2025_10_28_003", "Add custom domains column", addCustomDomains)
	registry.Register("2025_11_02_001", "Create credit_transactions table", createCreditTransactions)
	registry.Register("2025_12_20_001", "Drop redundant usage tables", dropRedundantUsageTables)
	registry.Register("2025_12_20_002", "Rename storage_usage to storage_bytes", renameStorageUsageToStorageBytes)
	registry.Register("2025_12_20_003", "Add group column to deployments", addGroupColumnToDeployments)
	registry.Register("2025_12_20_004", "Migrate group to groups JSONB array", migrateGroupToGroupsArray)
	registry.Register("2025_12_20_005", "Add start_command column to deployments", addStartCommandColumn)
	registry.Register("2025_12_28_001", "Create build_history and build_logs tables", createBuildHistoryTables)
	registry.Register("2025_01_03_001", "Add configurable build paths and nginx config to deployments", addBuildPathsAndNginxConfig)
	registry.Register("2025_01_03_002", "Add region column to node_metadata table", addRegionToNodeMetadata)
	registry.Register("2025_01_03_003", "Create support_tickets and ticket_comments tables", createSupportTicketsTables)
	registry.Register("2025_11_07_001", "Create deployment_metrics table", createDeploymentMetricsTable)
	registry.Register("2025_11_07_002", "Create deployment_usage_hourly table", createDeploymentUsageHourlyTable)
	registry.Register("2025_11_07_003", "Ensure idx_domain_type constraint exists on delegated_dns_records", ensureDelegatedDNSRecordsConstraint)
	registry.Register("2025_11_07_004", "Create audit_logs table", createAuditLogsTable)
	registry.Register("2025_11_13_001", "Create vps_bastion_keys table", createVPSBastionKeysTable)
	registry.Register("2025_11_14_001", "Add ssh_alias column to vps_instances", addSSHAliasToVPSInstances)

	// Add new migrations here
}

// initialSchema creates the initial database schema
func initialSchema(db *gorm.DB) error {
	// This migration creates the initial tables
	// Since we use GORM AutoMigrate for initial setup, this is mainly for tracking purposes
	return nil
}

// addHealthStatus adds the health_status field to deployments
func addHealthStatus(db *gorm.DB) error {
	// Check if column exists
	if db.Migrator().HasColumn("deployments", "health_status") {
		return nil
	}

	// Add column
	return db.Exec("ALTER TABLE deployments ADD COLUMN health_status VARCHAR(255) DEFAULT 'unknown'").Error
}

// addCustomDomains adds the custom_domains JSONB column
func addCustomDomains(db *gorm.DB) error {
	// Check if column exists
	if db.Migrator().HasColumn("deployments", "custom_domains") {
		return nil
	}

	// Add column
	return db.Exec("ALTER TABLE deployments ADD COLUMN custom_domains JSONB DEFAULT '[]'::jsonb").Error
}

// createCreditTransactions creates the credit_transactions table for tracking credit history
func createCreditTransactions(db *gorm.DB) error {
	// Check if table exists
	if db.Migrator().HasTable("credit_transactions") {
		return nil
	}

	// Create table with proper indexes
	return db.Exec(`
		CREATE TABLE credit_transactions (
			id VARCHAR(255) PRIMARY KEY,
			organization_id VARCHAR(255) NOT NULL,
			amount_cents BIGINT NOT NULL,
			balance_after BIGINT NOT NULL,
			type VARCHAR(255) NOT NULL,
			source VARCHAR(255) NOT NULL,
			note TEXT,
			created_by VARCHAR(255),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_organization_id (organization_id),
			INDEX idx_created_by (created_by),
			INDEX idx_created_at (created_at)
		)
	`).Error
}

// dropRedundantUsageTables drops the redundant usage tables (usage_monthly, usage_weekly, deployment_usage)
// These are no longer needed as usage is calculated on-demand from deployment_usage_hourly
func dropRedundantUsageTables(db *gorm.DB) error {
	tables := []string{"deployment_usage", "usage_weekly", "usage_monthly"}
	for _, table := range tables {
		if db.Migrator().HasTable(table) {
			if err := db.Exec("DROP TABLE IF EXISTS " + table).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// renameStorageUsageToStorageBytes renames the storage_usage column to storage_bytes for better clarity
func renameStorageUsageToStorageBytes(db *gorm.DB) error {
	// Check if column already renamed
	if db.Migrator().HasColumn("deployments", "storage_bytes") {
		return nil
	}

	// Check if old column exists
	if !db.Migrator().HasColumn("deployments", "storage_usage") {
		// Neither column exists - might be a fresh install, nothing to do
		return nil
	}

	// Rename the column
	return db.Exec("ALTER TABLE deployments RENAME COLUMN storage_usage TO storage_bytes").Error
}

// addGroupColumnToDeployments adds the group column to deployments table for organizing deployments
func addGroupColumnToDeployments(db *gorm.DB) error {
	// Check if column already exists
	if db.Migrator().HasColumn("deployments", "group") {
		return nil
	}

	// Add column with index
	if err := db.Exec("ALTER TABLE deployments ADD COLUMN \"group\" VARCHAR(255)").Error; err != nil {
		return err
	}

	// Add index
	return db.Exec("CREATE INDEX IF NOT EXISTS idx_deployments_group ON deployments(\"group\")").Error
}

// migrateGroupToGroupsArray migrates from single group column to groups JSONB array
func migrateGroupToGroupsArray(db *gorm.DB) error {
	// Check if groups column already exists
	if db.Migrator().HasColumn("deployments", "groups") {
		return nil
	}

	// Add groups column as JSONB
	// Use GORM's Migrator to add column without default first, then set default
	// This avoids SQL escaping issues
	if !db.Migrator().HasColumn("deployments", "groups") {
		// Add column without default first (GORM handles this better)
		if err := db.Exec(`ALTER TABLE deployments ADD COLUMN groups JSONB`).Error; err != nil {
			// Check if column was added anyway (maybe by AutoMigrate concurrently)
			if !db.Migrator().HasColumn("deployments", "groups") {
				return err
			}
		}
		// Now set the default value separately to avoid quote escaping issues
		if err := db.Exec(`ALTER TABLE deployments ALTER COLUMN groups SET DEFAULT '[]'::jsonb`).Error; err != nil {
			// Default might already be set, continue
		}
	}

	// Migrate existing group values to groups array
	// If group is not null and not empty, convert to JSON array
	if err := db.Exec(`
		UPDATE deployments 
		SET groups = CASE 
			WHEN "group" IS NOT NULL AND "group" != '' THEN jsonb_build_array("group")
			ELSE '[]'::jsonb
		END
	`).Error; err != nil {
		return err
	}

	// Drop old group column and its index
	if db.Migrator().HasColumn("deployments", "group") {
		if err := db.Exec("DROP INDEX IF EXISTS idx_deployments_group").Error; err != nil {
			return err
		}
		if err := db.Exec("ALTER TABLE deployments DROP COLUMN IF EXISTS \"group\"").Error; err != nil {
			return err
		}
	}

	return nil
}

// addStartCommandColumn adds the start_command column to deployments table
func addStartCommandColumn(db *gorm.DB) error {
	// Check if column already exists
	if db.Migrator().HasColumn("deployments", "start_command") {
		return nil
	}

	// Add column
	return db.Exec("ALTER TABLE deployments ADD COLUMN start_command VARCHAR(500)").Error
}

// createBuildHistoryTables creates the build_history and build_logs tables
func createBuildHistoryTables(db *gorm.DB) error {
	// Check if tables already exist
	if db.Migrator().HasTable("build_history") && db.Migrator().HasTable("build_logs") {
		return nil
	}

	// Create build_history table
	if !db.Migrator().HasTable("build_history") {
		if err := db.Exec(`
			CREATE TABLE build_history (
				id VARCHAR(255) PRIMARY KEY,
				deployment_id VARCHAR(255) NOT NULL,
				organization_id VARCHAR(255) NOT NULL,
				build_number INTEGER NOT NULL,
				status INTEGER NOT NULL DEFAULT 0,
				started_at TIMESTAMP NOT NULL,
				completed_at TIMESTAMP,
				build_time INTEGER NOT NULL DEFAULT 0,
				triggered_by VARCHAR(255) NOT NULL,
				repository_url VARCHAR(500),
				branch VARCHAR(255) NOT NULL,
				commit_sha VARCHAR(255),
				build_command VARCHAR(500),
				install_command VARCHAR(500),
				start_command VARCHAR(500),
				dockerfile_path VARCHAR(500),
				compose_file_path VARCHAR(500),
				build_strategy INTEGER NOT NULL DEFAULT 0,
				image_name VARCHAR(500),
				compose_yaml TEXT,
				size VARCHAR(50),
				error TEXT,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				INDEX idx_deployment_id (deployment_id),
				INDEX idx_organization_id (organization_id),
				INDEX idx_build_number (build_number),
				INDEX idx_started_at (started_at),
				INDEX idx_triggered_by (triggered_by),
				UNIQUE INDEX idx_deployment_build_number (deployment_id, build_number)
			)
		`).Error; err != nil {
			return err
		}
	}

	return nil
}

// addBuildPathsAndNginxConfig adds configurable build paths and nginx configuration fields
func addBuildPathsAndNginxConfig(db *gorm.DB) error {
	// Add build_path column
	if !db.Migrator().HasColumn("deployments", "build_path") {
		if err := db.Exec("ALTER TABLE deployments ADD COLUMN build_path VARCHAR(500)").Error; err != nil {
			return err
		}
	}

	// Add build_output_path column
	if !db.Migrator().HasColumn("deployments", "build_output_path") {
		if err := db.Exec("ALTER TABLE deployments ADD COLUMN build_output_path VARCHAR(500)").Error; err != nil {
			return err
		}
	}

	// Add use_nginx column
	if !db.Migrator().HasColumn("deployments", "use_nginx") {
		if err := db.Exec("ALTER TABLE deployments ADD COLUMN use_nginx BOOLEAN DEFAULT false").Error; err != nil {
			return err
		}
	}

	// Add nginx_config column
	if !db.Migrator().HasColumn("deployments", "nginx_config") {
		if err := db.Exec("ALTER TABLE deployments ADD COLUMN nginx_config TEXT").Error; err != nil {
			return err
		}
	}

	return nil
}

// addRegionToNodeMetadata adds the region column to node_metadata table for multi-region DNS routing
func addRegionToNodeMetadata(db *gorm.DB) error {
	// Check if column already exists
	if db.Migrator().HasColumn("node_metadata", "region") {
		return nil
	}

	// Add column with index
	if err := db.Exec("ALTER TABLE node_metadata ADD COLUMN region VARCHAR(255)").Error; err != nil {
		return err
	}

	// Add index for faster region lookups
	return db.Exec("CREATE INDEX IF NOT EXISTS idx_node_metadata_region ON node_metadata(region)").Error
}

// createSupportTicketsTables creates the support_tickets and ticket_comments tables
func createSupportTicketsTables(db *gorm.DB) error {
	// Check if tables already exist
	if db.Migrator().HasTable("support_tickets") && db.Migrator().HasTable("ticket_comments") {
		return nil
	}

	// Create support_tickets table
	if !db.Migrator().HasTable("support_tickets") {
		if err := db.Exec(`
			CREATE TABLE support_tickets (
				id VARCHAR(255) PRIMARY KEY,
				subject VARCHAR(500) NOT NULL,
				description TEXT NOT NULL,
				status INTEGER NOT NULL DEFAULT 1,
				priority INTEGER NOT NULL DEFAULT 2,
				category INTEGER NOT NULL DEFAULT 0,
				created_by VARCHAR(255) NOT NULL,
				assigned_to VARCHAR(255),
				organization_id VARCHAR(255),
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				resolved_at TIMESTAMP,
				INDEX idx_created_by (created_by),
				INDEX idx_assigned_to (assigned_to),
				INDEX idx_organization_id (organization_id),
				INDEX idx_status (status),
				INDEX idx_priority (priority),
				INDEX idx_category (category),
				INDEX idx_created_at (created_at)
			)
		`).Error; err != nil {
			return err
		}
	}

	// Create ticket_comments table
	if !db.Migrator().HasTable("ticket_comments") {
		if err := db.Exec(`
			CREATE TABLE ticket_comments (
				id VARCHAR(255) PRIMARY KEY,
				ticket_id VARCHAR(255) NOT NULL,
				content TEXT NOT NULL,
				created_by VARCHAR(255) NOT NULL,
				internal BOOLEAN NOT NULL DEFAULT false,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				INDEX idx_ticket_id (ticket_id),
				INDEX idx_created_by (created_by),
				INDEX idx_created_at (created_at)
			)
		`).Error; err != nil {
			return err
		}
	}

	return nil
}

// createDeploymentMetricsTable creates the deployment_metrics table for storing deployment metrics
// This only creates the table in the metrics database, never in the main database
func createDeploymentMetricsTable(db *gorm.DB) error {
	// Initialize metrics database if not already initialized
	if database.MetricsDB == nil {
		if err := database.InitMetricsDatabase(); err != nil {
			logger.Warn("Failed to initialize metrics database for migration: %v", err)
			// If metrics DB is not available, skip this migration
			// The tables will be created by InitMetricsTables() when the app starts
			return nil
		}
	}

	// Only create in metrics database
	if database.MetricsDB == nil {
		logger.Warn("Metrics database not available, skipping deployment_metrics table creation")
		return nil
	}

	if database.MetricsDB.Migrator().HasTable("deployment_metrics") {
		return nil
	}

	return createDeploymentMetricsTableInDB(database.MetricsDB)
}

// createDeploymentMetricsTableInDB creates the deployment_metrics table in a specific database
func createDeploymentMetricsTableInDB(db *gorm.DB) error {
	// Create table with proper indexes
	if err := db.Exec(`
		CREATE TABLE deployment_metrics (
			id SERIAL PRIMARY KEY,
			deployment_id VARCHAR(255) NOT NULL,
			container_id VARCHAR(255),
			node_id VARCHAR(255),
			cpu_usage DOUBLE PRECISION,
			memory_usage BIGINT,
			network_rx_bytes BIGINT,
			network_tx_bytes BIGINT,
			disk_read_bytes BIGINT,
			disk_write_bytes BIGINT,
			request_count BIGINT,
			error_count BIGINT,
			timestamp TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_deployment_id (deployment_id),
			INDEX idx_container_id (container_id),
			INDEX idx_node_id (node_id),
			INDEX idx_timestamp (timestamp)
		)
	`).Error; err != nil {
		return err
	}

	// Create composite indexes for better query performance
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_deployment_metrics_deployment_timestamp 
		ON deployment_metrics(deployment_id, timestamp DESC)
	`).Error; err != nil {
		logger.Warn("Failed to create deployment_timestamp index: %v", err)
	}

	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_deployment_metrics_timestamp_deployment 
		ON deployment_metrics(timestamp DESC, deployment_id)
	`).Error; err != nil {
		logger.Warn("Failed to create timestamp_deployment index: %v", err)
	}

	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_deployment_metrics_container_timestamp 
		ON deployment_metrics(container_id, timestamp DESC)
	`).Error; err != nil {
		logger.Warn("Failed to create container_timestamp index: %v", err)
	}

	return nil
}

// createDeploymentUsageHourlyTable creates the deployment_usage_hourly table for storing hourly aggregated metrics
// This only creates the table in the metrics database, never in the main database
func createDeploymentUsageHourlyTable(db *gorm.DB) error {
	// Initialize metrics database if not already initialized
	if database.MetricsDB == nil {
		if err := database.InitMetricsDatabase(); err != nil {
			logger.Warn("Failed to initialize metrics database for migration: %v", err)
			// If metrics DB is not available, skip this migration
			// The tables will be created by InitMetricsTables() when the app starts
			return nil
		}
	}

	// Only create in metrics database
	if database.MetricsDB == nil {
		logger.Warn("Metrics database not available, skipping deployment_usage_hourly table creation")
		return nil
	}

	if database.MetricsDB.Migrator().HasTable("deployment_usage_hourly") {
		return nil
	}

	return createDeploymentUsageHourlyTableInDB(database.MetricsDB)
}

// createDeploymentUsageHourlyTableInDB creates the deployment_usage_hourly table in a specific database
func createDeploymentUsageHourlyTableInDB(db *gorm.DB) error {
	// Create table with proper indexes
	if err := db.Exec(`
		CREATE TABLE deployment_usage_hourly (
			id SERIAL PRIMARY KEY,
			deployment_id VARCHAR(255) NOT NULL,
			organization_id VARCHAR(255) NOT NULL,
			hour TIMESTAMP NOT NULL,
			avg_cpu_usage DOUBLE PRECISION,
			avg_memory_usage DOUBLE PRECISION,
			bandwidth_rx_bytes BIGINT,
			bandwidth_tx_bytes BIGINT,
			disk_read_bytes BIGINT,
			disk_write_bytes BIGINT,
			request_count BIGINT,
			error_count BIGINT,
			sample_count BIGINT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_deployment_id (deployment_id),
			INDEX idx_organization_id (organization_id),
			INDEX idx_hour (hour)
		)
	`).Error; err != nil {
		return err
	}

	// Create composite index for better query performance
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_deployment_usage_hourly_deployment_hour 
		ON deployment_usage_hourly(deployment_id, hour DESC)
	`).Error; err != nil {
		logger.Warn("Failed to create usage_hourly index: %v", err)
	}

	return nil
}

// ensureDelegatedDNSRecordsConstraint ensures the idx_domain_type unique constraint exists on delegated_dns_records
func ensureDelegatedDNSRecordsConstraint(db *gorm.DB) error {
	// Check if table exists
	if !db.Migrator().HasTable("delegated_dns_records") {
		// Table doesn't exist, nothing to do (will be created by AutoMigrate)
		return nil
	}

	// Check if constraint already exists
	var constraintExists bool
	if err := db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM pg_constraint 
			WHERE conname = 'idx_domain_type' 
			AND conrelid = 'delegated_dns_records'::regclass
		)
	`).Scan(&constraintExists).Error; err != nil {
		return fmt.Errorf("failed to check constraint existence: %w", err)
	}

	if constraintExists {
		return nil
	}

	// Check if columns exist
	if !db.Migrator().HasColumn("delegated_dns_records", "domain") || 
	   !db.Migrator().HasColumn("delegated_dns_records", "record_type") {
		// Columns don't exist yet, will be created by AutoMigrate
		return nil
	}

	// Create the unique constraint
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_domain_type 
		ON delegated_dns_records(domain, record_type)
	`).Error; err != nil {
		// If it fails, it might already exist with a different name, try to find it
		var existingConstraint string
		if err2 := db.Raw(`
			SELECT conname FROM pg_constraint 
			WHERE conrelid = 'delegated_dns_records'::regclass 
			AND contype = 'u'
			AND array_length(conkey, 1) = 2
		`).Scan(&existingConstraint).Error; err2 == nil && existingConstraint != "" {
			// Constraint exists with different name, that's fine
			logger.Debug("Unique constraint already exists with name: %s", existingConstraint)
			return nil
		}
		return fmt.Errorf("failed to create idx_domain_type constraint: %w", err)
	}

	return nil
}

// createAuditLogsTable creates the audit_logs table for storing audit log entries
// This only creates the table in the metrics database, never in the main database
func createAuditLogsTable(db *gorm.DB) error {
	// Initialize metrics database if not already initialized
	if database.MetricsDB == nil {
		if err := database.InitMetricsDatabase(); err != nil {
			logger.Warn("Failed to initialize metrics database for migration: %v", err)
			// If metrics DB is not available, skip this migration
			// The tables will be created by InitMetricsTables() when the app starts
			return nil
		}
	}

	// Only create in metrics database
	if database.MetricsDB == nil {
		logger.Warn("Metrics database not available, skipping audit_logs table creation")
		return nil
	}

	if database.MetricsDB.Migrator().HasTable("audit_logs") {
		return nil
	}

	return createAuditLogsTableInDB(database.MetricsDB)
}

// createAuditLogsTableInDB creates the audit_logs table in a specific database
func createAuditLogsTableInDB(db *gorm.DB) error {
	// Create table with proper indexes
	if err := db.Exec(`
		CREATE TABLE audit_logs (
			id VARCHAR(255) NOT NULL,
			user_id VARCHAR(255) NOT NULL,
			organization_id VARCHAR(255),
			action VARCHAR(255) NOT NULL,
			service VARCHAR(255) NOT NULL,
			resource_type VARCHAR(255),
			resource_id VARCHAR(255),
			ip_address VARCHAR(255),
			user_agent TEXT,
			request_data JSONB,
			response_status INTEGER,
			error_message TEXT,
			duration_ms BIGINT,
			created_at TIMESTAMP NOT NULL,
			PRIMARY KEY (id, created_at),
			INDEX idx_user_id (user_id),
			INDEX idx_organization_id (organization_id),
			INDEX idx_action (action),
			INDEX idx_service (service),
			INDEX idx_resource_type (resource_type),
			INDEX idx_resource_id (resource_id),
			INDEX idx_created_at (created_at)
		)
	`).Error; err != nil {
		return err
	}

	return nil
}

// createVPSBastionKeysTable creates the vps_bastion_keys table for storing SSH keys used by the bastion host
func createVPSBastionKeysTable(db *gorm.DB) error {
	// Check if table already exists
	if db.Migrator().HasTable("vps_bastion_keys") {
		return nil
	}

	// Create table with proper indexes
	return db.Exec(`
		CREATE TABLE vps_bastion_keys (
			id VARCHAR(255) PRIMARY KEY,
			vps_id VARCHAR(255) NOT NULL UNIQUE,
			organization_id VARCHAR(255) NOT NULL,
			public_key TEXT NOT NULL,
			private_key TEXT NOT NULL,
			fingerprint VARCHAR(255),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_vps_id (vps_id),
			INDEX idx_organization_id (organization_id),
			INDEX idx_fingerprint (fingerprint)
		)
	`).Error
}

// addSSHAliasToVPSInstances adds the ssh_alias column to vps_instances table
func addSSHAliasToVPSInstances(db *gorm.DB) error {
	// Check if column already exists
	if db.Migrator().HasColumn("vps_instances", "ssh_alias") {
		return nil
	}

	// Add column with unique index
	// First add the column
	if err := db.Exec("ALTER TABLE vps_instances ADD COLUMN ssh_alias VARCHAR(255)").Error; err != nil {
		return err
	}

	// Add unique index (allowing NULL values - multiple NULLs are allowed in unique indexes)
	return db.Exec("CREATE UNIQUE INDEX idx_vps_instances_ssh_alias ON vps_instances(ssh_alias) WHERE ssh_alias IS NOT NULL").Error
}

// Template for creating a new migration:
/*
func yourNewMigration(db *gorm.DB) error {
	// Check if necessary to avoid re-applying
	if db.Migrator().HasColumn("your_table", "your_column") {
		return nil
	}

	// Run SQL or GORM operations
	return db.Exec("YOUR SQL HERE").Error

	// Alternatively, use GORM operations:
	// return db.Model(&YourModel{}).AddColumn("column_name", "column_type").Error
}
*/
