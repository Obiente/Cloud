#!/bin/bash
# Quick fix script to add overlay network rule to pg_hba.conf
# This is a one-time fix for existing databases before redeploying with configs

set -e

STACK_NAME="${STACK_NAME:-obiente}"
POSTGRES_SERVICE="${STACK_NAME}_postgres"
SUBNET="${OVERLAY_SUBNET:-10.15.3.0/24}"

echo "🔧 Fixing PostgreSQL pg_hba.conf for overlay network: $SUBNET"
echo ""

# Find postgres service task
echo "1. Finding PostgreSQL service task..."
TASK=$(docker service ps "$POSTGRES_SERVICE" --filter "desired-state=running" --format "{{.ID}}" | head -1)

if [ -z "$TASK" ]; then
  echo "❌ PostgreSQL service '$POSTGRES_SERVICE' not found or not running"
  exit 1
fi

# Get container ID from task
CONTAINER_ID=$(docker inspect --format '{{.Status.ContainerStatus.ContainerID}}' "$TASK" 2>/dev/null || echo "")

if [ -z "$CONTAINER_ID" ]; then
  echo "❌ Could not get container ID from task. Service may still be starting."
  echo "   Try again in a few seconds, or check: docker service ps $POSTGRES_SERVICE"
  exit 1
fi

echo "✅ Found PostgreSQL container: ${CONTAINER_ID:0:12}"
echo ""

PGDATA="/var/lib/postgresql/data"
PG_HBA_CONF="$PGDATA/pg_hba.conf"

# Check if pg_hba.conf exists
if ! docker exec "$CONTAINER_ID" test -f "$PG_HBA_CONF" 2>/dev/null; then
  echo "❌ pg_hba.conf not found at $PG_HBA_CONF"
  echo "   Container may still be initializing. Wait a moment and try again."
  exit 1
fi

# Check for old subnet rules and remove them
echo "2. Checking for old subnet rules..."
OLD_SUBNETS=("10.0.1.0/24" "10.0.0.0/24" "172.16.0.0/12")
HAS_OLD_RULE=false
for old_subnet in "${OLD_SUBNETS[@]}"; do
  if docker exec "$CONTAINER_ID" grep -qE "^host\s+all\s+all\s+${old_subnet//\//\\/}" "$PG_HBA_CONF" 2>/dev/null; then
    echo "   ⚠️  Found old subnet rule: $old_subnet (will be removed)"
    HAS_OLD_RULE=true
    docker exec "$CONTAINER_ID" sed -i "/^host\s\+all\s\+all\s\+${old_subnet//\//\\/}/d" "$PG_HBA_CONF"
  fi
done
if [ "$HAS_OLD_RULE" = true ]; then
  echo "   ✅ Removed old subnet rules"
fi

# Check if correct rule already exists
if docker exec "$CONTAINER_ID" grep -qE "^host\s+all\s+all\s+${SUBNET//\//\\/}\s+md5" "$PG_HBA_CONF" 2>/dev/null; then
  echo "✅ Overlay network rule already exists in pg_hba.conf"
  echo "   Rule: host    all    all    $SUBNET    md5"
else
  echo "3. Adding overlay network rule to pg_hba.conf..."
  
  # Backup
  docker exec "$CONTAINER_ID" cp "$PG_HBA_CONF" "$PG_HBA_CONF.backup.$(date +%Y%m%d_%H%M%S)" 2>/dev/null || true
  
  # Remove any existing "host all all all" rules that might conflict
  docker exec "$CONTAINER_ID" sed -i '/^host\s\+all\s\+all\s\+all\s/d' "$PG_HBA_CONF" 2>/dev/null || true
  
  # Add rule (insert after IPv4 localhost if it exists)
  if docker exec "$CONTAINER_ID" grep -qE "^host\s+all\s+all\s+127\.0\.0\.1" "$PG_HBA_CONF" 2>/dev/null; then
    docker exec "$CONTAINER_ID" sed -i "/^host\s+all\s+all\s+127\.0\.0\.1/a host    all    all    $SUBNET    md5" "$PG_HBA_CONF"
  elif docker exec "$CONTAINER_ID" grep -q "^local" "$PG_HBA_CONF" 2>/dev/null; then
    docker exec "$CONTAINER_ID" sed -i "/^local/a host    all    all    $SUBNET    md5" "$PG_HBA_CONF"
  else
    docker exec "$CONTAINER_ID" sh -c "echo 'host    all    all    $SUBNET    md5' >> $PG_HBA_CONF"
  fi
  
  echo "✅ Added rule: host    all    all    $SUBNET    md5"
fi

# Fix listen_addresses if not set to *
echo ""
echo "4. Checking and fixing listen_addresses..."
LISTEN_ADDR=$(docker exec "$CONTAINER_ID" psql -U obiente_postgres -d obiente -t -c "SHOW listen_addresses;" 2>/dev/null | tr -d ' ' || echo "unknown")
echo "   Current listen_addresses: $LISTEN_ADDR"

if [ "$LISTEN_ADDR" != "*" ]; then
  echo "   ⚠️  PostgreSQL is NOT listening on all interfaces"
  echo "   Setting listen_addresses=* in postgresql.conf..."
  
  POSTGRESQL_CONF="$PGDATA/postgresql.conf"
  if docker exec "$CONTAINER_ID" test -f "$POSTGRESQL_CONF" 2>/dev/null; then
    # Remove existing listen_addresses line
    docker exec "$CONTAINER_ID" sed -i '/^listen_addresses\s*=/d' "$POSTGRESQL_CONF" 2>/dev/null || true
    # Add new line
    docker exec "$CONTAINER_ID" sh -c "echo 'listen_addresses = '\''*'\''' >> $POSTGRESQL_CONF"
    echo "   ✅ Added listen_addresses=* to postgresql.conf"
    echo "   ⚠️  PostgreSQL needs to be restarted for this to take effect"
    echo "   Run: docker service update --force $POSTGRES_SERVICE"
  else
    echo "   ⚠️  postgresql.conf not found. This should be set via command line in docker-compose."
  fi
else
  echo "   ✅ PostgreSQL is listening on all interfaces"
fi

# Reload PostgreSQL configuration
echo ""
echo "5. Reloading PostgreSQL configuration..."
if docker exec "$CONTAINER_ID" psql -U obiente_postgres -d obiente -c "SELECT pg_reload_conf();" >/dev/null 2>&1; then
  echo "✅ PostgreSQL configuration reloaded"
else
  echo "⚠️  Could not reload configuration via SQL"
  echo "   Attempting to send SIGHUP to PostgreSQL process..."
  if docker exec "$CONTAINER_ID" killall -HUP postgres >/dev/null 2>&1; then
    echo "✅ Sent SIGHUP to PostgreSQL"
  else
    echo "⚠️  Could not send SIGHUP. Restarting service..."
    echo "   Run: docker service update --force $POSTGRES_SERVICE"
  fi
fi

# Verify the rule is actually in the file
echo ""
echo "6. Verifying configuration..."
if docker exec "$CONTAINER_ID" grep -qE "^host\s+all\s+all\s+${SUBNET//\//\\/}\s+md5" "$PG_HBA_CONF" 2>/dev/null; then
  echo "✅ Rule confirmed in pg_hba.conf"
  docker exec "$CONTAINER_ID" grep -E "^host\s+all\s+all\s+${SUBNET//\//\\/}\s+md5" "$PG_HBA_CONF"
else
  echo "❌ Rule NOT found in pg_hba.conf after adding!"
  echo "   This is unexpected. Please check the file manually."
fi


echo ""
echo "✅ Configuration complete!"
echo ""
echo "📝 Note: After redeploying with Docker configs, this will be automatic."

