#!/bin/bash
# PostgreSQL initialization script to ensure the user exists
# This runs after PostgreSQL is initialized but before it starts accepting connections
# Place this in /docker-entrypoint-initdb.d/ to run automatically

set -e

POSTGRES_USER="${POSTGRES_USER:-obiente_postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-obiente_postgres}"
POSTGRES_DB="${POSTGRES_DB:-obiente}"

echo "🔧 Ensuring PostgreSQL user '$POSTGRES_USER' exists..."

# Check if user exists
USER_EXISTS=$(psql -U postgres -d postgres -t -c "SELECT 1 FROM pg_roles WHERE rolname='$POSTGRES_USER';" 2>/dev/null | tr -d ' \n\r' || echo "")

if [ "$USER_EXISTS" != "1" ]; then
  echo "📝 Creating user '$POSTGRES_USER'..."
  psql -U postgres -d postgres -c "CREATE USER $POSTGRES_USER WITH PASSWORD '$POSTGRES_PASSWORD' CREATEDB;" 2>/dev/null || true
  
  # Grant privileges
  psql -U postgres -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE $POSTGRES_DB TO $POSTGRES_USER;" 2>/dev/null || true
  psql -U postgres -d postgres -c "ALTER USER $POSTGRES_USER WITH SUPERUSER;" 2>/dev/null || true
  
  echo "✅ User '$POSTGRES_USER' created"
else
  echo "✅ User '$POSTGRES_USER' already exists"
  
  # Update password in case it changed
  psql -U postgres -d postgres -c "ALTER USER $POSTGRES_USER WITH PASSWORD '$POSTGRES_PASSWORD';" 2>/dev/null || true
  echo "✅ Password updated for user '$POSTGRES_USER'"
fi

