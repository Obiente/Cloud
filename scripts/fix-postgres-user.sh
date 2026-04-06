#!/bin/bash
# Fix PostgreSQL user - create obiente_postgres if it doesn't exist

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/lib/common.sh"

trim_env_value() {
  local value="$1"
  value="${value%$'\r'}"
  value="${value//$'\n'/}"
  value="${value//[{}]/}"
  trim_whitespace "$value"
}

escape_sql_literal() {
  printf "%s" "$1" | sed "s/'/''/g"
}

assert_safe_sql_identifier() {
  local identifier="$1"
  local label="$2"
  if [[ ! "$identifier" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]]; then
    echo "❌ Unsafe ${label}: ${identifier}" >&2
    echo "   This script only supports simple PostgreSQL identifiers." >&2
    exit 1
  fi
}

run_psql() {
  local sql="$1"
  if [ -z "${SUPERUSER_PASSWORD:-}" ]; then
    docker exec "$CONTAINER_ID" psql -v ON_ERROR_STOP=1 -U "$SUPERUSER" -d postgres -c "$sql"
  else
    docker exec -e PGPASSWORD="$SUPERUSER_PASSWORD" "$CONTAINER_ID" psql -v ON_ERROR_STOP=1 -U "$SUPERUSER" -d postgres -c "$sql"
  fi
}

if [ -f "${REPO_ROOT}/.env" ]; then
  load_env_file "${REPO_ROOT}/.env"
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
    ACTUAL_PASSWORD=$(trim_env_value "$(echo "$SERVICE_ENV" | grep "POSTGRES_PASSWORD" | cut -d= -f2- | head -1)")
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

assert_safe_sql_identifier "$POSTGRES_USER" "PostgreSQL username"
assert_safe_sql_identifier "$POSTGRES_DB" "PostgreSQL database name"
ESCAPED_POSTGRES_PASSWORD="$(escape_sql_literal "$POSTGRES_PASSWORD")"

# Find the actual superuser in the database
# PostgreSQL might have been initialized with a custom user, so "postgres" might not exist
echo "2. Finding PostgreSQL superuser..."
SUPERUSER=""
SUPERUSER_PASSWORD=""

# Try to get the superuser from the service environment
SERVICE_ENV=$(docker service inspect "$POSTGRES_SERVICE" --format '{{range .Spec.TaskTemplate.ContainerSpec.Env}}{{println .}}{{end}}' 2>/dev/null || echo "")
SERVICE_POSTGRES_USER=$(trim_env_value "$(echo "$SERVICE_ENV" | grep "^POSTGRES_USER=" | cut -d= -f2- | head -1 || echo "")")

# Try different users to find one that works
CANDIDATE_USERS=("postgres" "$SERVICE_POSTGRES_USER" "$POSTGRES_USER")
CANDIDATE_PASSWORDS=("" "$POSTGRES_PASSWORD")

for candidate_user in "${CANDIDATE_USERS[@]}"; do
  if [ -z "$candidate_user" ]; then
    continue
  fi
  
  # Try without password first (trust authentication)
  if docker exec "$CONTAINER_ID" psql -U "$candidate_user" -d postgres -t -c "SELECT 1;" >/dev/null 2>&1; then
    SUPERUSER="$candidate_user"
    SUPERUSER_PASSWORD=""
    echo "   ✅ Found superuser: $SUPERUSER (trust authentication)"
    break
  fi
  
  # Try with password
  for candidate_password in "${CANDIDATE_PASSWORDS[@]}"; do
    if [ -n "$candidate_password" ] && docker exec -e PGPASSWORD="$candidate_password" "$CONTAINER_ID" psql -U "$candidate_user" -d postgres -t -c "SELECT 1;" >/dev/null 2>&1; then
      SUPERUSER="$candidate_user"
      SUPERUSER_PASSWORD="$candidate_password"
      echo "   ✅ Found superuser: $SUPERUSER (password authentication)"
      break 2
    fi
  done
done

if [ -z "$SUPERUSER" ]; then
  echo "   ❌ Could not find a working superuser"
  echo "   💡 Trying to list all users to find one..."
  
  # Try to connect using the database's own user (might work via peer auth)
  # List users by checking pg_authid directly (requires superuser, but we'll try)
  for candidate_user in "${CANDIDATE_USERS[@]}"; do
    if [ -z "$candidate_user" ]; then
      continue
    fi
    EXISTING_USERS=$(docker exec "$CONTAINER_ID" psql -U "$candidate_user" -d postgres -t -c "SELECT rolname FROM pg_roles WHERE rolname NOT LIKE 'pg_%' AND rolcanlogin ORDER BY rolname;" 2>/dev/null | tr '\n' ' ' || echo "")
    if [ -n "$EXISTING_USERS" ]; then
      echo "   Found users: $EXISTING_USERS"
      # Try the first user as superuser
      FIRST_USER=$(echo "$EXISTING_USERS" | awk '{print $1}')
      if [ -n "$FIRST_USER" ]; then
        SUPERUSER="$FIRST_USER"
        echo "   ⚠️  Attempting to use first user as superuser: $SUPERUSER"
        break
      fi
    fi
  done
  
  if [ -z "$SUPERUSER" ]; then
    echo "   ❌ Cannot proceed without a superuser connection"
    echo "   💡 You may need to:"
    echo "      1. Check your .env file for POSTGRES_USER and POSTGRES_PASSWORD"
    echo "      2. Or manually connect to PostgreSQL and create the user:"
    echo "         docker exec -it $CONTAINER_ID psql -U <existing_user> -d postgres"
    exit 1
  fi
fi

echo ""

# Check if user exists using the found superuser
echo "3. Checking if user '$POSTGRES_USER' exists..."
USER_EXISTS=""

if [ -z "$SUPERUSER_PASSWORD" ]; then
  # Trust authentication
  if docker exec "$CONTAINER_ID" psql -U "$SUPERUSER" -d postgres -t -c "SELECT 1 FROM pg_roles WHERE rolname='$POSTGRES_USER';" 2>/dev/null | tr -d ' \n\r' | grep -q "1"; then
    USER_EXISTS="1"
  fi
else
  # Password authentication
  if docker exec -e PGPASSWORD="$SUPERUSER_PASSWORD" "$CONTAINER_ID" psql -U "$SUPERUSER" -d postgres -t -c "SELECT 1 FROM pg_roles WHERE rolname='$POSTGRES_USER';" 2>/dev/null | tr -d ' \n\r' | grep -q "1"; then
    USER_EXISTS="1"
  fi
fi

if [ "$USER_EXISTS" != "1" ]; then
  # List existing users for debugging
  echo "   Checking existing users..."
  if [ -z "$SUPERUSER_PASSWORD" ]; then
    EXISTING_USERS=$(docker exec "$CONTAINER_ID" psql -U "$SUPERUSER" -d postgres -t -c "SELECT rolname FROM pg_roles WHERE rolname NOT LIKE 'pg_%' ORDER BY rolname;" 2>/dev/null | tr '\n' ' ' || echo "")
  else
    EXISTING_USERS=$(docker exec -e PGPASSWORD="$SUPERUSER_PASSWORD" "$CONTAINER_ID" psql -U "$SUPERUSER" -d postgres -t -c "SELECT rolname FROM pg_roles WHERE rolname NOT LIKE 'pg_%' ORDER BY rolname;" 2>/dev/null | tr '\n' ' ' || echo "")
  fi
  if [ -n "$EXISTING_USERS" ]; then
    echo "   Existing users: $EXISTING_USERS"
  fi
fi

if [ "$USER_EXISTS" = "1" ]; then
  echo "✅ User '$POSTGRES_USER' already exists"
  echo ""
  echo "4. Verifying user can connect..."
  if docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$CONTAINER_ID" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT current_user;" >/dev/null 2>&1; then
    echo "✅ User can connect successfully"
  else
    echo "⚠️  User exists but cannot connect. Updating password..."
    run_psql "ALTER USER ${POSTGRES_USER} WITH PASSWORD '${ESCAPED_POSTGRES_PASSWORD}';" >/dev/null
    echo "✅ Password updated"
  fi
else
  echo "❌ User '$POSTGRES_USER' does NOT exist"
  echo ""
  echo "4. Creating user '$POSTGRES_USER'..."
  
  # Create the user using the found superuser
  CREATE_SUCCESS=false

  if CREATE_OUTPUT=$(run_psql "CREATE USER ${POSTGRES_USER} WITH PASSWORD '${ESCAPED_POSTGRES_PASSWORD}' CREATEDB;" 2>&1); then
    CREATE_SUCCESS=true
  else
    echo "   Error: ${CREATE_OUTPUT}"
  fi
  
  if [ "$CREATE_SUCCESS" = true ]; then
    echo "✅ User created"
    echo ""
    
    # Grant privileges
    echo "5. Granting privileges..."
    run_psql "GRANT ALL PRIVILEGES ON DATABASE ${POSTGRES_DB} TO ${POSTGRES_USER};" >/dev/null
    run_psql "ALTER USER ${POSTGRES_USER} WITH SUPERUSER;" >/dev/null
    echo "✅ Privileges granted"
    echo ""
    
    # Test connection
    echo "6. Testing connection..."
    if docker exec -e PGPASSWORD="$POSTGRES_PASSWORD" "$CONTAINER_ID" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT current_user;" >/dev/null 2>&1; then
      echo "✅ Connection test successful"
    else
      echo "⚠️  Connection test failed, but user was created"
    fi
  else
    echo "❌ Failed to create user"
    echo ""
    echo "💡 Troubleshooting:"
    echo "   1. Check that POSTGRES_USER and POSTGRES_PASSWORD are set correctly in your .env file"
    echo "   2. The superuser '$SUPERUSER' might not have sufficient privileges"
    echo "   3. Try manually creating the user:"
    if [ -z "$SUPERUSER_PASSWORD" ]; then
      echo "      docker exec -it $CONTAINER_ID psql -U $SUPERUSER -d postgres"
    else
      echo "      docker exec -it -e PGPASSWORD='$SUPERUSER_PASSWORD' $CONTAINER_ID psql -U $SUPERUSER -d postgres"
    fi
    echo "      Then run: CREATE USER $POSTGRES_USER WITH PASSWORD '$POSTGRES_PASSWORD' CREATEDB SUPERUSER;"
    exit 1
  fi
fi

echo ""
echo "✅ User setup complete!"
echo ""
echo "💡 If services still can't connect, restart them:"
echo "   docker service update --force obiente_auth-service"
echo "   docker service update --force obiente_audit-service"
echo "   # ... etc"
