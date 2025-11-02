package migrations

import (
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
