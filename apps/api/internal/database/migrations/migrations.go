package migrations

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"api/internal/database"

	"gorm.io/gorm"
)

// MigrationVersion represents a database schema version
type MigrationVersion struct {
	Version     string    `gorm:"primaryKey" json:"version"`
	Description string    `json:"description"`
	Applied     bool      `json:"applied"`
	CreatedAt   time.Time `json:"created_at"`
	AppliedAt   time.Time `json:"applied_at"`
	Hash        string    `json:"hash"`        // Hash of migration SQL for verification
	Duration    int64     `json:"duration_ms"` // Time taken to apply in milliseconds
}

// MigrationFunc is a function that applies a migration
type MigrationFunc func(*gorm.DB) error

// MigrationRegistry holds all migrations
type MigrationRegistry struct {
	migrations       map[string]MigrationFunc
	migrationDetails map[string]string // version -> description
	db               *gorm.DB
	dryRun           bool
}

// NewMigrationRegistry creates a new migration registry
func NewMigrationRegistry(db *gorm.DB) *MigrationRegistry {
	return &MigrationRegistry{
		migrations:       make(map[string]MigrationFunc),
		migrationDetails: make(map[string]string),
		db:               db,
		dryRun:           false,
	}
}

// SetDryRun sets whether migrations should be applied in dry run mode
func (r *MigrationRegistry) SetDryRun(dryRun bool) {
	r.dryRun = dryRun
}

// Register adds a new migration
func (r *MigrationRegistry) Register(version string, description string, fn MigrationFunc) {
	r.migrations[version] = fn
	r.migrationDetails[version] = description
}

// Setup creates the migrations table if it doesn't exist
func (r *MigrationRegistry) Setup() error {
	return r.db.AutoMigrate(&MigrationVersion{})
}

// GetStatus returns the current migration status
func (r *MigrationRegistry) GetStatus() ([]MigrationVersion, error) {
	if err := r.Setup(); err != nil {
		return nil, fmt.Errorf("failed to setup migrations table: %w", err)
	}

	// Get all applied migrations
	var applied []MigrationVersion
	if err := r.db.Find(&applied).Error; err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedMap := make(map[string]MigrationVersion)
	for _, m := range applied {
		appliedMap[m.Version] = m
	}

	// Create status for all migrations
	var status []MigrationVersion
	var migrationIDs []string
	for id := range r.migrations {
		migrationIDs = append(migrationIDs, id)
	}
	sort.Strings(migrationIDs)

	for _, id := range migrationIDs {
		if applied, ok := appliedMap[id]; ok {
			status = append(status, applied)
		} else {
			status = append(status, MigrationVersion{
				Version:     id,
				Description: r.migrationDetails[id],
				Applied:     false,
				CreatedAt:   time.Now(),
			})
		}
	}

	return status, nil
}

// Apply runs all pending migrations
func (r *MigrationRegistry) Apply() error {
	if err := r.Setup(); err != nil {
		return fmt.Errorf("failed to setup migrations table: %w", err)
	}

	// Get applied migrations
	var applied []MigrationVersion
	if err := r.db.Find(&applied).Error; err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedMap := make(map[string]bool)
	for _, m := range applied {
		appliedMap[m.Version] = m.Applied
	}

	// Sort migration IDs to ensure they run in order
	var migrationIDs []string
	for id := range r.migrations {
		migrationIDs = append(migrationIDs, id)
	}
	sort.Strings(migrationIDs)

	// Apply pending migrations
	for _, id := range migrationIDs {
		if appliedMap[id] {
			log.Printf("Migration %s already applied", id)
			continue
		}

		log.Printf("Applying migration: %s - %s", id, r.migrationDetails[id])
		startTime := time.Now()

		err := r.db.Transaction(func(tx *gorm.DB) error {
			// Apply the migration
			if err := r.migrations[id](tx); err != nil {
				return fmt.Errorf("failed to apply migration %s: %w", id, err)
			}

			// In dry run mode, don't record the migration
			if r.dryRun {
				return nil
			}

			// Calculate a hash for the migration (simplified for now)
			// In a real implementation, you'd calculate a hash of the SQL generated
			migrationHash := fmt.Sprintf("%x", id)

			// Record the migration
			return tx.Create(&MigrationVersion{
				Version:     id,
				Description: r.migrationDetails[id],
				Applied:     true,
				CreatedAt:   time.Now(),
				AppliedAt:   time.Now(),
				Hash:        migrationHash,
				Duration:    time.Since(startTime).Milliseconds(),
			}).Error
		})

		if err != nil {
			return fmt.Errorf("migration %s failed: %w", id, err)
		}

		duration := time.Since(startTime)
		log.Printf("Migration %s applied successfully in %v", id, duration)
	}

	return nil
}

// IsMigrationMode returns true if the application is running in migration mode
func IsMigrationMode() bool {
	return len(os.Args) > 1 && os.Args[1] == "migrate"
}

// MigrateModels automatically creates or updates database tables based on GORM models
func MigrateModels(db *gorm.DB) error {
	log.Println("Running auto-migration for models...")

	// Register all models that should be auto-migrated
	models := []interface{}{
		&database.Deployment{},
		&MigrationVersion{}, // Make sure our migrations table is included
		// Add other models here as you create them
	}

	// Run AutoMigrate for each model
	for _, model := range models {
		modelName := fmt.Sprintf("%T", model)
		modelName = strings.TrimPrefix(modelName, "*")
		log.Printf("Auto-migrating model: %s", modelName)

		if err := db.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to auto-migrate %s: %w", modelName, err)
		}
	}

	log.Println("Auto-migration completed successfully")
	return nil
}
