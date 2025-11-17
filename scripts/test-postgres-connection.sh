#!/bin/bash
# Test PostgreSQL connection from overlay network with authentication

set -e

STACK_NAME="${STACK_NAME:-obiente}"
NETWORK_NAME="${STACK_NAME}_obiente-network"
DB_USER="${POSTGRES_USER:-obiente_postgres}"
DB_PASSWORD="${POSTGRES_PASSWORD:-obiente_postgres}"
DB_NAME="${POSTGRES_DB:-obiente}"

echo "🔍 Testing PostgreSQL connection from overlay network..."
echo "   Network: $NETWORK_NAME"
echo "   User: $DB_USER"
echo "   Database: $DB_NAME"
echo ""

if ! docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
  echo "❌ Network $NETWORK_NAME not found"
  exit 1
fi

echo "1. Testing connection with psql..."
if docker run --rm --network "$NETWORK_NAME" \
  -e PGPASSWORD="$DB_PASSWORD" \
  postgres:16-alpine \
  psql -h postgres -U "$DB_USER" -d "$DB_NAME" -c "SELECT version();" 2>&1; then
  echo "   ✅ Connection successful!"
else
  echo "   ❌ Connection failed"
  echo ""
  echo "2. Testing connection without authentication (to see error)..."
  docker run --rm --network "$NETWORK_NAME" \
    postgres:16-alpine \
    psql -h postgres -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" 2>&1 || true
fi

echo ""
echo "✅ Connection test complete!"

