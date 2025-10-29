#!/bin/bash
set -e

# Script to validate database schemas against proto definitions
# Designed to be run in CI environments

echo "Validating database schemas against proto definitions..."

# Navigate to project root
cd "$(dirname "$0")/.."

# Run the validation tests
go test -v ./internal/database/protovalidation/...

# Check if any schema validation errors occur
if [ $? -ne 0 ]; then
  echo "❌ Schema validation failed! Database models don't match proto definitions."
  exit 1
fi

echo "✅ Schema validation passed. Database models match proto definitions."

# Optionally run a dry-run migration to check for schema issues
echo "Running migration dry run to verify database structure..."
go run ./cmd/migrate/main.go --dry-run

if [ $? -ne 0 ]; then
  echo "❌ Migration dry run failed! Database schema has issues."
  exit 1
fi

echo "✅ Migration dry run passed. Database schema is valid."
echo "All schema validations completed successfully."
