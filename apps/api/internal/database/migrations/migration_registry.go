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
