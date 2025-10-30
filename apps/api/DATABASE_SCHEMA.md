# Database Schema Management

This document outlines our approach to managing database schemas, migrations, and validation against protocol buffer definitions.

## Core Principles

1. **GORM-First Approach**: We use GORM models as the primary source of truth for database structure
2. **Version-Controlled Migrations**: All schema changes are tracked as versioned migrations
3. **Proto-Compatibility**: GORM models are validated against protocol buffer definitions
4. **Automated Testing**: Schema validation runs in CI to prevent compatibility regressions

## Directory Structure

- `internal/database/` - Database models and connection code
- `internal/database/migrations/` - Migration system and individual migrations
- `internal/database/protovalidation/` - Proto compatibility validation
- `cmd/migrate/` - Command-line tool for running migrations

## Workflow

### Development Workflow

1. Update GORM models in `internal/database/models.go` to reflect new database structure
2. Create a new migration in `internal/database/migrations/migration_registry.go`
3. Test the migration with `make db-dry-run` and `make db-validate`
4. Apply the migration with `make db-migrate`

### CI/CD Workflow

1. Run `make ci-db-validate` to validate models against proto definitions
2. Verify migration dry-run succeeds
3. In deployment, run migrations before deploying new code

## Creating a New Migration

Use the helper command:

```bash
make new-migration name=add_column_xyz
```

Then implement the migration in the generated file and register it in `migration_registry.go`.

## Migration Strategies

1. **Simple Migrations**: Use direct SQL or GORM operations for simple changes
2. **Data Migrations**: For complex data transformations, break into multiple steps
3. **Rollback Support**: Include rollback logic where possible

Example:

```go
func addCustomFields(db *gorm.DB) error {
    // Check if we need to apply
    if db.Migrator().HasColumn("deployments", "custom_field") {
        return nil
    }

    // Add the column
    return db.Exec(`
        ALTER TABLE deployments
        ADD COLUMN custom_field TEXT DEFAULT '';
    `).Error
}
```

## Proto Compatibility

The `protovalidation` package ensures that GORM models remain compatible with proto definitions.
This guarantees that:

1. Fields in proto messages have corresponding fields in GORM models
2. Field types are compatible between proto and GORM
3. JSON field names match proto field names (snake_case)

## Commands Reference

- `make db-migrate` - Run pending migrations
- `make db-status` - Show migration status
- `make db-validate` - Validate schemas against proto definitions
- `make db-dry-run` - Test migrations without applying them
- `make db-auto-migrate` - Use GORM's AutoMigrate (development only)

## Best Practices

1. **Never use AutoMigrate in production** - Always use versioned migrations
2. **Migrations should be idempotent** - Safe to run multiple times
3. **Validate before deploying** - Always run validation before applying migrations
4. **Keep migrations small** - Easier to test and roll back
5. **Document complex migrations** - Add comments explaining the purpose and approach
