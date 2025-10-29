package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"api/internal/database"
	"api/internal/database/migrations"
	"api/internal/database/protovalidation"

	"gorm.io/gorm"
)

func main() {
	startTime := time.Now()

	// Parse flags
	autoMigrate := flag.Bool("auto", false, "Use GORM's AutoMigrate instead of running migrations")
	dryRun := flag.Bool("dry-run", false, "Check migrations without applying them")
	validateProto := flag.Bool("validate-proto", true, "Validate GORM models against proto definitions")
	showStatus := flag.Bool("status", false, "Show migration status")
	flag.Parse()

	// Validate GORM models against proto definitions if requested
	if *validateProto {
		fmt.Println("Validating GORM models against proto definitions...")
		validator := protovalidation.ValidateAllModels()
		if !validator.IsValid() {
			log.Fatalf("Schema validation failed:\n%s", validator.ErrorsString())
		}
		fmt.Println("Schema validation passed")
	}

	// Initialize database
	if err := database.InitDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Configure dry run if requested
	if *dryRun {
		// Replace DB with a cloned connection that uses a transaction
		tx := database.DB.Session(&gorm.Session{DryRun: true})
		database.DB = tx
		fmt.Println("Running in dry-run mode (no changes will be applied)")
	}

	if *autoMigrate {
		// Run auto-migration
		if err := migrations.MigrateModels(database.DB); err != nil {
			log.Fatalf("Auto-migration failed: %v", err)
		}
		fmt.Println("Auto-migration plan generated successfully")
	} else {
		// Run migrations
		registry := migrations.NewMigrationRegistry(database.DB)
		migrations.RegisterMigrations(registry)

		if *showStatus {
			// Show migration status
			status, err := registry.GetStatus()
			if err != nil {
				log.Fatalf("Failed to get migration status: %v", err)
			}

			fmt.Println("Migration Status:")
			fmt.Println("=====================================")
			fmt.Printf("%-20s %-30s %s\n", "VERSION", "DESCRIPTION", "STATUS")
			fmt.Println("-------------------------------------")

			for _, m := range status {
				statusStr := "PENDING"
				if m.Applied {
					statusStr = fmt.Sprintf("APPLIED (%s)", m.AppliedAt.Format("2006-01-02 15:04:05"))
				}
				fmt.Printf("%-20s %-30s %s\n", m.Version, m.Description, statusStr)
			}
			fmt.Println("=====================================")
		} else {
			// Set dry run mode if requested
			if *dryRun {
				registry.SetDryRun(true)
			}

			// Apply migrations
			if err := registry.Apply(); err != nil {
				log.Fatalf("Migration failed: %v", err)
			}

			if *dryRun {
				fmt.Println("Migration plan looks good (dry run successful)")
			} else {
				fmt.Println("Migrations completed successfully")
			}
		}
	}

	duration := time.Since(startTime)
	fmt.Printf("Migration completed in %v\n", duration)
	os.Exit(0)
}
