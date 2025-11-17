#!/bin/bash
# Fix PostgreSQL user - create obiente_postgres if it doesn't exist

set -e

# Load .env file if it exists
if [ -f .env ]; then
  echo "📝 Loading environment variables from .env file..."
  # Export variables from .env file (handles comments and empty lines)
  set -a
  source .env
  set +a
fi

STACK_NAME="${STACK_NAME:-obiente}"
POSTGRES_SERVICE="${STACK_NAME}_postgres"
POSTGRES_USER="${POSTGRES_USER:-obiente_postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-obiente_postgres}"
POSTGRES_DB="${POSTGRES_DB:-obiente}"

# Try to get the actual postgres password from the service
# The default postgres user might have a different password
# Check if POSTGRES_PASSWORD is set, otherwise try to get it from the service
if [ -z "$POSTGRES_PASSWORD" ] || [ "$POSTGRES_PASSWORD" = "obiente_postgres" ]; then
  # Try to get password from service environment
  SERVICE_ENV=$(docker service inspect "$POSTGRES_SERVICE" --format '{{range .Spec.TaskTemplate.ContainerSpec.Env}}{{println .}}{{end}}' 2>/dev/null || echo "")
  if echo "$SERVICE_ENV" | grep -q "POSTGRES_PASSWORD"; then
    ACTUAL_PASSWORD=$(echo "$SERVICE_ENV" | grep "POSTGRES_PASSWORD" | cut -d= -f2- | head -1)
    if [ -n "$ACTUAL_PASSWORD" ] && [ "$ACTUAL_PASSWORD" != "obiente_postgres" ]; then
      POSTGRES_PASSWORD="$ACTUAL_PASSWORD"
      echo "ℹ️  Using POSTGRES_PASSWORD from service environment"
    fi
  fi
fi

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
# First, try to connect as postgres user (default superuser)
# The postgres user might not have a password, or might use trust authentication locally
USER_EXISTS=""
ERROR_OUTPUT=""

# Try without password first (trust authentication for local connections)
if docker exec "$CONTAINER_ID" psql -U postgres -d postgres -t -c "SELECT 1 FROM pg_roles WHERE rolname='$POSTGRES_USER';" 2>/dev/null | tr -d ' \n\r' | grep -q "1"; then
  USER_EXISTS="1"
elif docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$CONTAINER_ID" psql -U postgres -d postgres -t -c "SELECT 1 FROM pg_roles WHERE rolname='$POSTGRES_USER';" 2>/dev/null | tr -d ' \n\r' | grep -q "1"; then
  USER_EXISTS="1"
else
  # Check what users actually exist
  echo "   Checking existing users..."
  EXISTING_USERS=$(docker exec "$CONTAINER_ID" psql -U postgres -d postgres -t -c "SELECT rolname FROM pg_roles WHERE rolname NOT LIKE 'pg_%' ORDER BY rolname;" 2>/dev/null | tr -d ' \n\r' || echo "")
  if [ -z "$EXISTING_USERS" ]; then
    # Try with password
    EXISTING_USERS=$(docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$CONTAINER_ID" psql -U postgres -d postgres -t -c "SELECT rolname FROM pg_roles WHERE rolname NOT LIKE 'pg_%' ORDER BY rolname;" 2>/dev/null | tr -d ' \n\r' || echo "")
  fi
  if [ -n "$EXISTING_USERS" ]; then
    echo "   Existing users: $EXISTING_USERS"
  fi
fi

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
  
  # Try to create the user - try without password first, then with password
  CREATE_SUCCESS=false
  
  # Try without password (trust authentication)
  if docker exec "$CONTAINER_ID" psql -U postgres -d postgres -c "CREATE USER $POSTGRES_USER WITH PASSWORD '$POSTGRES_PASSWORD' CREATEDB;" 2>&1; then
    CREATE_SUCCESS=true
  elif docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$CONTAINER_ID" psql -U postgres -d postgres -c "CREATE USER $POSTGRES_USER WITH PASSWORD '$POSTGRES_PASSWORD' CREATEDB;" 2>&1; then
    CREATE_SUCCESS=true
  else
    # Show the actual error
    echo "   Attempting to connect as postgres user to diagnose..."
    ERROR_MSG=$(docker exec "$CONTAINER_ID" psql -U postgres -d postgres -c "SELECT 1;" 2>&1 || true)
    if echo "$ERROR_MSG" | grep -q "password authentication failed\|FATAL.*password"; then
      echo "   ⚠️  Password authentication required for postgres user"
      echo "   💡 Try setting POSTGRES_PASSWORD in your .env file to match the service password"
      echo ""
      echo "   Or, if you know the postgres user password, run:"
      echo "   PGPASSWORD=your_password docker exec -e PGPASSWORD=your_password $CONTAINER_ID psql -U postgres -d postgres -c \"CREATE USER $POSTGRES_USER WITH PASSWORD '$POSTGRES_PASSWORD' CREATEDB;\""
    else
      echo "   Error: $ERROR_MSG"
    fi
    echo "❌ Failed to create user"
    exit 1
  fi
  
  if [ "$CREATE_SUCCESS" = true ]; then
    echo "✅ User created"
    echo ""
    
    # Grant privileges
    echo "4. Granting privileges..."
    # Try without password first, then with password
    if ! docker exec "$CONTAINER_ID" psql -U postgres -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE $POSTGRES_DB TO $POSTGRES_USER;" 2>/dev/null; then
      docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$CONTAINER_ID" psql -U postgres -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE $POSTGRES_DB TO $POSTGRES_USER;" 2>/dev/null || true
    fi
    if ! docker exec "$CONTAINER_ID" psql -U postgres -d postgres -c "ALTER USER $POSTGRES_USER WITH SUPERUSER;" 2>/dev/null; then
      docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$CONTAINER_ID" psql -U postgres -d postgres -c "ALTER USER $POSTGRES_USER WITH SUPERUSER;" 2>/dev/null || true
    fi
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
fi

echo ""
echo "✅ User setup complete!"
echo ""
echo "💡 If services still can't connect, restart them:"
echo "   docker service update --force obiente_auth-service"
echo "   docker service update --force obiente_audit-service"
echo "   # ... etc"

