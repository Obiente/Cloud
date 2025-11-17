#!/bin/bash
# Fix PostgreSQL user - create obiente_postgres if it doesn't exist

set -e

STACK_NAME="${STACK_NAME:-obiente}"
POSTGRES_SERVICE="${STACK_NAME}_postgres"
POSTGRES_USER="${POSTGRES_USER:-obiente_postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-obiente_postgres}"
POSTGRES_DB="${POSTGRES_DB:-obiente}"

echo "🔧 Fixing PostgreSQL User: $POSTGRES_USER"
echo "=========================================="
echo ""

# Find postgres container
echo "1. Finding PostgreSQL container..."
TASK=$(docker service ps "$POSTGRES_SERVICE" --filter "desired-state=running" --format "{{.ID}}" | head -1)

if [ -z "$TASK" ]; then
  echo "❌ PostgreSQL service not found"
  exit 1
fi

CONTAINER_ID=$(docker inspect --format '{{.Status.ContainerStatus.ContainerID}}' "$TASK" 2>/dev/null || echo "")

if [ -z "$CONTAINER_ID" ]; then
  echo "❌ Could not get container ID"
  exit 1
fi

echo "✅ Container: ${CONTAINER_ID:0:12}"
echo ""

# Check if user exists
echo "2. Checking if user '$POSTGRES_USER' exists..."
USER_EXISTS=$(docker exec "$CONTAINER_ID" psql -U postgres -d postgres -t -c "SELECT 1 FROM pg_roles WHERE rolname='$POSTGRES_USER';" 2>/dev/null | tr -d ' \n\r' || echo "")

if [ "$USER_EXISTS" = "1" ]; then
  echo "✅ User '$POSTGRES_USER' already exists"
  echo ""
  echo "3. Verifying user can connect..."
  if docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$CONTAINER_ID" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT current_user;" >/dev/null 2>&1; then
    echo "✅ User can connect successfully"
  else
    echo "⚠️  User exists but cannot connect. Updating password..."
    docker exec "$CONTAINER_ID" psql -U postgres -d postgres -c "ALTER USER $POSTGRES_USER WITH PASSWORD '$POSTGRES_PASSWORD';" 2>/dev/null || true
    echo "✅ Password updated"
  fi
else
  echo "❌ User '$POSTGRES_USER' does NOT exist"
  echo ""
  echo "3. Creating user '$POSTGRES_USER'..."
  
  # Create the user
  docker exec "$CONTAINER_ID" psql -U postgres -d postgres -c "CREATE USER $POSTGRES_USER WITH PASSWORD '$POSTGRES_PASSWORD' CREATEDB;" 2>/dev/null || {
    echo "❌ Failed to create user"
    exit 1
  }
  
  echo "✅ User created"
  echo ""
  
  # Grant privileges
  echo "4. Granting privileges..."
  docker exec "$CONTAINER_ID" psql -U postgres -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE $POSTGRES_DB TO $POSTGRES_USER;" 2>/dev/null || true
  docker exec "$CONTAINER_ID" psql -U postgres -d postgres -c "ALTER USER $POSTGRES_USER WITH SUPERUSER;" 2>/dev/null || true
  echo "✅ Privileges granted"
  echo ""
  
  # Test connection
  echo "5. Testing connection..."
  if docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$CONTAINER_ID" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT current_user;" >/dev/null 2>&1; then
    echo "✅ Connection test successful"
  else
    echo "⚠️  Connection test failed, but user was created"
  fi
fi

echo ""
echo "✅ User setup complete!"
echo ""
echo "💡 If services still can't connect, restart them:"
echo "   docker service update --force obiente_auth-service"
echo "   docker service update --force obiente_audit-service"
echo "   # ... etc"

