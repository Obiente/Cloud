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

	// Auto-migrate the schema
	if err := db.AutoMigrate(&Deployment{}); err != nil {
		return fmt.Errorf("failed to auto-migrate: %w", err)
	}

	log.Println("Database schema migrated")

	// Initialize deployment tracking tables
	if err := InitDeploymentTracking(); err != nil {
		return fmt.Errorf("failed to initialize deployment tracking: %w", err)
	}

	log.Println("Deployment tracking initialized")

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
