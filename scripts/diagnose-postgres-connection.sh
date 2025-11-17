#!/bin/bash
# Comprehensive diagnostic script for PostgreSQL connection issues

set -e

STACK_NAME="${STACK_NAME:-obiente}"
POSTGRES_SERVICE="${STACK_NAME}_postgres"

echo "🔍 Comprehensive PostgreSQL Connection Diagnostic"
echo "=================================================="
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

# Check what PostgreSQL is actually listening on
echo "2. Checking what PostgreSQL is listening on..."
echo "   Listening ports:"
docker exec "$CONTAINER_ID" netstat -tlnp 2>/dev/null | grep 5432 || docker exec "$CONTAINER_ID" ss -tlnp | grep 5432 || echo "   (could not check)"
echo ""

# Check container IPs
echo "3. Container network interfaces:"
docker exec "$CONTAINER_ID" ip addr show | grep -E "inet " | grep -v "127.0.0.1" || echo "   (could not get IPs)"
echo ""

# Check listen_addresses
echo "4. PostgreSQL listen_addresses:"
LISTEN=$(docker exec "$CONTAINER_ID" psql -U obiente_postgres -d obiente -t -c "SHOW listen_addresses;" 2>/dev/null | tr -d ' \n\r' || echo "unknown")
echo "   SHOW listen_addresses: '$LISTEN'"
echo ""

# Check pg_hba.conf
echo "5. pg_hba.conf overlay network rule:"
PG_HBA="/var/lib/postgresql/data/pg_hba.conf"
if docker exec "$CONTAINER_ID" grep -qE "^host\s+all\s+all\s+10\.15\.3\.0/24\s+md5" "$PG_HBA" 2>/dev/null; then
  echo "   ✅ Rule exists:"
  docker exec "$CONTAINER_ID" grep -E "^host\s+all\s+all\s+10\.15\.3\.0/24\s+md5" "$PG_HBA"
else
  echo "   ❌ Rule NOT found!"
  echo "   Current rules:"
  docker exec "$CONTAINER_ID" grep -E "^host|^local" "$PG_HBA" | head -10 || echo "   (no rules found)"
fi
echo ""

# Check PostgreSQL logs for connection attempts
echo "6. Recent PostgreSQL logs (connection-related):"
docker exec "$CONTAINER_ID" tail -20 /var/lib/postgresql/data/log/*.log 2>/dev/null | grep -iE "connection|auth|reject|timeout" | tail -10 || echo "   (no relevant logs found)"
echo ""

# Test connection from within the container
echo "7. Testing connection from PostgreSQL container itself:"
# First check if user exists
POSTGRES_USER="${POSTGRES_USER:-obiente_postgres}"
POSTGRES_DB="${POSTGRES_DB:-obiente}"
USER_EXISTS=$(docker exec "$CONTAINER_ID" psql -U postgres -d postgres -t -c "SELECT 1 FROM pg_roles WHERE rolname='$POSTGRES_USER';" 2>/dev/null | tr -d ' \n\r' || echo "")

if [ "$USER_EXISTS" != "1" ]; then
  echo "   ⚠️  User '$POSTGRES_USER' does NOT exist in PostgreSQL!"
  echo "   💡 Run: ./scripts/fix-postgres-user.sh"
  echo ""
  echo "   Trying to connect as 'postgres' user instead..."
  if docker exec "$CONTAINER_ID" psql -U postgres -d postgres -c "SELECT current_user;" >/dev/null 2>&1; then
    echo "   ✅ Can connect as 'postgres' user"
    echo ""
    echo "   Listing all users:"
    docker exec "$CONTAINER_ID" psql -U postgres -d postgres -c "\du" 2>/dev/null || true
  else
    echo "   ❌ Cannot connect as 'postgres' user either"
  fi
else
  if docker exec "$CONTAINER_ID" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT 1;" >/dev/null 2>&1; then
    echo "   ✅ Local connection works"
  else
    echo "   ❌ Local connection failed"
    docker exec "$CONTAINER_ID" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT 1;" 2>&1 | head -3 || true
  fi
fi
echo ""

# Check if we can connect using the overlay IP with password
OVERLAY_IP=$(docker exec "$CONTAINER_ID" ip addr show eth0 | grep "inet " | awk '{print $2}' | cut -d/ -f1)
if [ -n "$OVERLAY_IP" ]; then
  echo "8. Testing connection using overlay IP ($OVERLAY_IP) with password:"
  PGPASSWORD="${POSTGRES_PASSWORD:-obiente_postgres}" docker exec -e PGPASSWORD="$PGPASSWORD" "$CONTAINER_ID" psql -h "$OVERLAY_IP" -U obiente_postgres -d obiente -c "SELECT 1;" >/dev/null 2>&1
  if [ $? -eq 0 ]; then
    echo "   ✅ Connection via overlay IP works with authentication"
  else
    echo "   ❌ Connection via overlay IP failed"
    PGPASSWORD="${POSTGRES_PASSWORD:-obiente_postgres}" docker exec -e PGPASSWORD="$PGPASSWORD" "$CONTAINER_ID" psql -h "$OVERLAY_IP" -U obiente_postgres -d obiente -c "SELECT 1;" 2>&1 | head -5 || true
  fi
fi
echo ""

# Check PostgreSQL process and what it's actually listening on
echo "9. PostgreSQL process details:"
docker exec "$CONTAINER_ID" ps aux | grep postgres | head -3 || echo "   (could not get process info)"
echo ""

# Check if there are any connection rejections in recent logs
echo "10. Checking for connection rejections in PostgreSQL logs:"
docker exec "$CONTAINER_ID" find /var/lib/postgresql/data -name "*.log" -type f -exec tail -50 {} \; 2>/dev/null | grep -iE "connection|auth|reject|fatal|timeout" | tail -10 || echo "   (no relevant log entries found)"
echo ""

echo "✅ Diagnostic complete!"
echo ""
echo "💡 If connections are timing out but port is open, check:"
echo "   1. PostgreSQL logs for connection rejections"
echo "   2. Firewall rules on worker nodes"
echo "   3. Network routing between nodes"

