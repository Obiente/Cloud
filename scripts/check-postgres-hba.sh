#!/bin/bash
# Diagnostic script to check PostgreSQL pg_hba.conf configuration

set -e

STACK_NAME="${STACK_NAME:-obiente}"
POSTGRES_SERVICE="${STACK_NAME}_postgres"
SUBNET="${OVERLAY_SUBNET:-10.15.3.0/24}"

echo "🔍 Checking PostgreSQL pg_hba.conf configuration"
echo "   Service: $POSTGRES_SERVICE"
echo "   Expected subnet: $SUBNET"
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
  echo "   Container may still be initializing."
  exit 1
fi

echo "2. Checking pg_hba.conf for overlay network rule..."
echo ""

# Check if rule exists
if docker exec "$CONTAINER_ID" grep -qE "^host\s+all\s+all\s+${SUBNET//\//\\/}\s+md5" "$PG_HBA_CONF" 2>/dev/null; then
  echo "✅ Overlay network rule EXISTS in pg_hba.conf"
  docker exec "$CONTAINER_ID" grep -E "^host\s+all\s+all\s+${SUBNET//\//\\/}\s+md5" "$PG_HBA_CONF"
else
  echo "❌ Overlay network rule NOT FOUND in pg_hba.conf"
  echo ""
  echo "   Current pg_hba.conf rules:"
  docker exec "$CONTAINER_ID" grep -E "^host|^local" "$PG_HBA_CONF" | head -10 || echo "   (no host/local rules found)"
fi

echo ""
echo "3. Checking PostgreSQL listen_addresses..."
LISTEN_ADDR=$(docker exec "$CONTAINER_ID" psql -U obiente_postgres -d obiente -t -c "SHOW listen_addresses;" 2>/dev/null | tr -d ' ' || echo "unknown")
echo "   listen_addresses: $LISTEN_ADDR"
if [ "$LISTEN_ADDR" = "*" ]; then
  echo "   ✅ PostgreSQL is listening on all interfaces"
else
  echo "   ⚠️  PostgreSQL is NOT listening on all interfaces"
fi

echo ""
echo "4. Checking PostgreSQL network interfaces..."
echo "   Container IPs:"
docker exec "$CONTAINER_ID" ip addr show | grep -E "inet " | grep -v "127.0.0.1" || echo "   (could not get IPs)"

echo ""
echo "5. Testing connection from overlay network..."
NETWORK_NAME="${STACK_NAME}_obiente-network"
if docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
  echo "   Testing connection from a test container..."
  if docker run --rm --network "$NETWORK_NAME" alpine sh -c "nc -zv postgres 5432 2>&1 || echo 'Connection failed'" 2>&1 | head -3; then
    echo "   (connection test completed)"
  else
    echo "   ⚠️  Could not test connection"
  fi
else
  echo "   ⚠️  Network $NETWORK_NAME not found"
fi

echo ""
echo "✅ Diagnostic complete!"

