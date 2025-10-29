# Database migration and schema management commands
.PHONY: db-migrate db-status db-validate db-dry-run db-auto-migrate

# Default database connection for local development
DB_CONNECTION ?= "host=localhost port=5432 user=postgres dbname=obiente sslmode=disable"

# Run migrations
db-migrate:
	@echo "Running database migrations..."
	@go run cmd/migrate/main.go

# Show migration status
db-status:
	@echo "Checking migration status..."
	@go run cmd/migrate/main.go --status

# Validate schemas against proto definitions
db-validate:
	@echo "Validating database schemas against proto definitions..."
	@go run cmd/migrate/main.go --validate-proto --dry-run

# Dry run migrations (validate without applying)
db-dry-run:
	@echo "Running migration dry run..."
	@go run cmd/migrate/main.go --dry-run

# Run GORM auto-migration (development only)
db-auto-migrate:
	@echo "Running auto-migration (DEVELOPMENT ONLY)..."
	@go run cmd/migrate/main.go --auto

# Run CI validation (schema + migration dry run)
ci-db-validate: db-validate db-dry-run
	@echo "CI database validation complete"

# Generate a new migration file (usage: make new-migration name=add_column_xyz)
new-migration:
	@if [ -z "$(name)" ]; then \
		echo "Error: Migration name required. Usage: make new-migration name=your_migration_name"; \
		exit 1; \
	fi
	@timestamp=$$(date +%Y_%m_%d_%H%M); \
	filename="migrations/$${timestamp}_$(name).go"; \
	echo "Creating new migration: $$filename"; \
	echo "package migrations\n\nimport (\n\t\"gorm.io/gorm\"\n)\n\nfunc $${timestamp}_$(name)(db *gorm.DB) error {\n\t// TODO: Implement migration\n\treturn nil\n}" > $$filename; \
	echo "Migration file created at $$filename"

# Help command
help:
	@echo "Database Migration Commands:"
	@echo "  make db-migrate         - Run all pending migrations"
	@echo "  make db-status          - Show migration status"
	@echo "  make db-validate        - Validate schemas against proto definitions"
	@echo "  make db-dry-run         - Dry run migrations (validate without applying)"
	@echo "  make db-auto-migrate    - Run GORM auto-migration (DEVELOPMENT ONLY)"
	@echo "  make ci-db-validate     - Run CI validation (schema + migration dry run)"
	@echo "  make new-migration name=xyz - Generate a new migration file"
