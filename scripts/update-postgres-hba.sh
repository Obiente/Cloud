#!/bin/bash
# Manual script to update pg_hba.conf with allowed hosts
# Usage: ./scripts/update-postgres-hba.sh [service-name]
#   service-name: 'postgres' or 'timescaledb' (default: 'postgres')

set -e

STACK_NAME="${STACK_NAME:-obiente}"
SERVICE_NAME="${1:-postgres}"
FULL_SERVICE_NAME="${STACK_NAME}_${SERVICE_NAME}"

echo "🔧 Updating pg_hba.conf for service: $SERVICE_NAME"
echo ""

# Find the running container
TASK=$(docker service ps "$FULL_SERVICE_NAME" --filter "desired-state=running" --format "{{.ID}}" | head -1)

if [ -z "$TASK" ]; then
  echo "❌ Service '$FULL_SERVICE_NAME' not found or not running"
  exit 1
fi

CONTAINER_ID=$(docker inspect --format '{{.Status.ContainerStatus.ContainerID}}' "$TASK" 2>/dev/null || echo "")

if [ -z "$CONTAINER_ID" ]; then
  echo "❌ Could not get container ID for service"
  exit 1
fi

echo "✅ Found container: ${CONTAINER_ID:0:12}"
echo ""

# Get environment variables from the service
SERVICE_ENV=$(docker service inspect "$FULL_SERVICE_NAME" --format '{{range .Spec.TaskTemplate.ContainerSpec.Env}}{{println .}}{{end}}' 2>/dev/null || echo "")

# Extract database user and password from service environment
if [ "$SERVICE_NAME" = "timescaledb" ]; then
  ALLOWED_HOSTS_ENV="METRICS_DB_ALLOWED_HOSTS"
  FALLBACK_ENV="POSTGRES_ALLOWED_HOSTS"
  DB_USER_ENV="METRICS_DB_USER"
  DB_PASSWORD_ENV="METRICS_DB_PASSWORD"
  DB_NAME_ENV="METRICS_DB_NAME"
  FALLBACK_USER_ENV="POSTGRES_USER"
  FALLBACK_PASSWORD_ENV="POSTGRES_PASSWORD"
  FALLBACK_DB_ENV="POSTGRES_DB"
else
  ALLOWED_HOSTS_ENV="POSTGRES_ALLOWED_HOSTS"
  FALLBACK_ENV=""
  DB_USER_ENV="DB_USER"
  DB_PASSWORD_ENV="DB_PASSWORD"
  DB_NAME_ENV="DB_NAME"
  FALLBACK_USER_ENV="POSTGRES_USER"
  FALLBACK_PASSWORD_ENV="POSTGRES_PASSWORD"
  FALLBACK_DB_ENV="POSTGRES_DB"
fi

# Get database user from service environment (check common-database anchor)
DB_USER=""
if echo "$SERVICE_ENV" | grep -q "^${DB_USER_ENV}="; then
  DB_USER=$(echo "$SERVICE_ENV" | grep "^${DB_USER_ENV}=" | cut -d'=' -f2- | tr -d '\n\r' | sed 's/[{}]//g' | xargs)
elif echo "$SERVICE_ENV" | grep -q "^${FALLBACK_USER_ENV}="; then
  DB_USER=$(echo "$SERVICE_ENV" | grep "^${FALLBACK_USER_ENV}=" | cut -d'=' -f2- | tr -d '\n\r' | sed 's/[{}]//g' | xargs)
fi

# Get database password
DB_PASSWORD=""
if echo "$SERVICE_ENV" | grep -q "^${DB_PASSWORD_ENV}="; then
  DB_PASSWORD=$(echo "$SERVICE_ENV" | grep "^${DB_PASSWORD_ENV}=" | cut -d'=' -f2- | tr -d '\n\r' | sed 's/[{}]//g' | xargs)
elif echo "$SERVICE_ENV" | grep -q "^${FALLBACK_PASSWORD_ENV}="; then
  DB_PASSWORD=$(echo "$SERVICE_ENV" | grep "^${FALLBACK_PASSWORD_ENV}=" | cut -d'=' -f2- | tr -d '\n\r' | sed 's/[{}]//g' | xargs)
fi

# Get database name
DB_NAME=""
if echo "$SERVICE_ENV" | grep -q "^${DB_NAME_ENV}="; then
  DB_NAME=$(echo "$SERVICE_ENV" | grep "^${DB_NAME_ENV}=" | cut -d'=' -f2- | tr -d '\n\r' | sed 's/[{}]//g' | xargs)
elif echo "$SERVICE_ENV" | grep -q "^${FALLBACK_DB_ENV}="; then
  DB_NAME=$(echo "$SERVICE_ENV" | grep "^${FALLBACK_DB_ENV}=" | cut -d'=' -f2- | tr -d '\n\r' | sed 's/[{}]//g' | xargs)
fi

# Fallback to defaults if not found
DB_USER="${DB_USER:-obiente_postgres}"
DB_PASSWORD="${DB_PASSWORD:-obiente_postgres}"
DB_NAME="${DB_NAME:-obiente}"

# Get allowed hosts from service environment
ALLOWED_HOSTS=""
if echo "$SERVICE_ENV" | grep -q "^${ALLOWED_HOSTS_ENV}="; then
  ALLOWED_HOSTS=$(echo "$SERVICE_ENV" | grep "^${ALLOWED_HOSTS_ENV}=" | cut -d'=' -f2- | tr -d '\n\r' | sed 's/[{}]//g' | xargs)
elif [ -n "$FALLBACK_ENV" ] && echo "$SERVICE_ENV" | grep -q "^${FALLBACK_ENV}="; then
  ALLOWED_HOSTS=$(echo "$SERVICE_ENV" | grep "^${FALLBACK_ENV}=" | cut -d'=' -f2- | tr -d '\n\r' | sed 's/[{}]//g' | xargs)
fi

# Also check current environment (in case set locally)
if [ -z "$ALLOWED_HOSTS" ]; then
  if [ "$SERVICE_NAME" = "timescaledb" ]; then
    ALLOWED_HOSTS="${METRICS_DB_ALLOWED_HOSTS:-${POSTGRES_ALLOWED_HOSTS:-}}"
  else
    ALLOWED_HOSTS="${POSTGRES_ALLOWED_HOSTS:-}"
  fi
fi

echo "📋 Database configuration:"
echo "   User: $DB_USER"
echo "   Database: $DB_NAME"
echo ""

# Ensure database user exists and has correct password
echo "🔧 Ensuring database user '$DB_USER' exists..."
USER_EXISTS=$(docker exec "$CONTAINER_ID" psql -U postgres -d postgres -t -c "SELECT 1 FROM pg_roles WHERE rolname='$DB_USER';" 2>/dev/null | tr -d ' \n\r' || echo "")

if [ "$USER_EXISTS" = "1" ]; then
  echo "   ✅ User '$DB_USER' exists"
  echo "   Updating password..."
  docker exec "$CONTAINER_ID" psql -U postgres -d postgres -c "ALTER USER $DB_USER WITH PASSWORD '$DB_PASSWORD';" 2>/dev/null || true
  echo "   ✅ Password updated"
else
  echo "   Creating user '$DB_USER'..."
  docker exec "$CONTAINER_ID" psql -U postgres -d postgres -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD' CREATEDB SUPERUSER;" 2>/dev/null || true
  docker exec "$CONTAINER_ID" psql -U postgres -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;" 2>/dev/null || true
  echo "   ✅ User '$DB_USER' created"
fi

echo ""

if [ -z "$ALLOWED_HOSTS" ]; then
  echo "⚠️  No allowed hosts configured (${ALLOWED_HOSTS_ENV} not set)"
  echo ""
  echo "Usage:"
  echo "  export ${ALLOWED_HOSTS_ENV}=10.10.10.1,10.0.0.0/8"
  echo "  docker service update --env-add ${ALLOWED_HOSTS_ENV}=\"10.10.10.1,10.0.0.0/8\" $FULL_SERVICE_NAME"
  echo "  ./scripts/update-postgres-hba.sh $SERVICE_NAME"
  exit 0
fi

echo "📋 Allowed hosts: $ALLOWED_HOSTS"
echo ""

PGDATA="/var/lib/postgresql/data"
HBA_TARGET="$PGDATA/pg_hba.conf"

# Check if pg_hba.conf exists in container
if ! docker exec "$CONTAINER_ID" test -f "$HBA_TARGET" 2>/dev/null; then
  echo "❌ pg_hba.conf not found in container at $HBA_TARGET"
  echo "   Make sure PostgreSQL is initialized"
  exit 1
fi

echo "🔧 Updating pg_hba.conf..."

# Convert comma-separated list to array and process each host
IFS=',' read -ra HOSTS <<< "$ALLOWED_HOSTS"

for host in "${HOSTS[@]}"; do
  # Trim whitespace
  host=$(echo "$host" | xargs)
  
  if [ -z "$host" ]; then
    continue
  fi
  
  # Normalize host format (add /32 for single IPs if not CIDR)
  if [[ "$host" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    # Single IP, add /32
    normalized_host="${host}/32"
  elif [[ "$host" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+/[0-9]+$ ]]; then
    # Already CIDR format
    normalized_host="$host"
  else
    echo "⚠️  Warning: Invalid host format '$host', skipping..."
    continue
  fi
  
  # Escape dots and slashes for grep
  escaped_host=$(echo "$normalized_host" | sed 's/\./\\./g' | sed 's/\//\\\//g')
  
  # Check if rule already exists
  if docker exec "$CONTAINER_ID" grep -qE "^host\s+all\s+all\s+${escaped_host}\s+md5" "$HBA_TARGET" 2>/dev/null; then
    echo "   ✅ Rule already exists for $normalized_host"
    continue
  fi
  
  # Add rule after the overlay network rule (or at the end if not found)
  # Use awk for reliable insertion (works in Alpine Linux)
  if docker exec "$CONTAINER_ID" grep -qE "^host\s+all\s+all\s+10\.15\.3\.0/24\s+md5" "$HBA_TARGET" 2>/dev/null; then
    # Insert after overlay network rule using awk
    docker exec "$CONTAINER_ID" sh -c "
      awk -v new_rule=\"host    all    all    $normalized_host    md5\" '
        /^host[[:space:]]+all[[:space:]]+all[[:space:]]+10\\.15\\.3\\.0\\/24[[:space:]]+md5/ {
          print
          print new_rule
          next
        }
        {print}
      ' $HBA_TARGET > $HBA_TARGET.tmp && \
      mv $HBA_TARGET.tmp $HBA_TARGET && \
      chmod 0600 $HBA_TARGET && \
      chown postgres:postgres $HBA_TARGET
    "
  else
    # Append at the end
    docker exec "$CONTAINER_ID" sh -c "echo 'host    all    all    $normalized_host    md5' >> $HBA_TARGET && chmod 0600 $HBA_TARGET && chown postgres:postgres $HBA_TARGET"
  fi
  
  # Verify the rule was actually added
  if docker exec "$CONTAINER_ID" grep -qE "^host\s+all\s+all\s+${escaped_host}\s+md5" "$HBA_TARGET" 2>/dev/null; then
    echo "   ✅ Added rule for $normalized_host"
  else
    echo "   ❌ Failed to add rule for $normalized_host (check container logs)"
    echo "      Attempted to add: host    all    all    $normalized_host    md5"
  fi
done

echo ""
echo "✅ pg_hba.conf updated"

# Reload PostgreSQL configuration
echo ""
echo "🔄 Reloading PostgreSQL configuration..."
PID=$(docker exec "$CONTAINER_ID" head -1 "$PGDATA/postmaster.pid" 2>/dev/null || echo "")
if [ -n "$PID" ]; then
  if docker exec "$CONTAINER_ID" kill -0 "$PID" 2>/dev/null; then
    docker exec "$CONTAINER_ID" kill -HUP "$PID" 2>/dev/null && echo "✅ Configuration reloaded" || echo "⚠️  Failed to reload (may need restart)"
  else
    echo "⚠️  PostgreSQL process not found"
  fi
else
  echo "⚠️  Could not find PostgreSQL PID"
fi

echo ""
echo "✅ Update complete!"
echo ""
echo "📝 Verifying pg_hba.conf rules:"
echo "   Allowed host rules:"
docker exec "$CONTAINER_ID" grep -E '^host[[:space:]]+all[[:space:]]+all' "$HBA_TARGET" 2>/dev/null | grep -v '127.0.0.1\|::1' || echo "   (no remote host rules found)"
echo ""
echo "   To view full file:"
echo "   docker exec $CONTAINER_ID cat $HBA_TARGET"

