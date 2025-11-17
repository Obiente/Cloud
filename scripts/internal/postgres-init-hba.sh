#!/bin/sh
# PostgreSQL initialization script for Docker Swarm overlay network
# This script runs in /docker-entrypoint-initdb.d/ (standard PostgreSQL mechanism)
# It configures pg_hba.conf to allow connections from the overlay network

set -e

PGDATA=${PGDATA:-/var/lib/postgresql/data}
PG_HBA_CONF="$PGDATA/pg_hba.conf"
SUBNET="${OVERLAY_SUBNET:-10.15.3.0/24}"

echo "🔧 Configuring pg_hba.conf for overlay network: $SUBNET"

# Wait for pg_hba.conf to be created by PostgreSQL initialization
MAX_WAIT=60
WAITED=0
while [ ! -f "$PG_HBA_CONF" ] && [ $WAITED -lt $MAX_WAIT ]; do
  sleep 1
  WAITED=$((WAITED + 1))
done

if [ ! -f "$PG_HBA_CONF" ]; then
  echo "⚠️  pg_hba.conf not found after waiting ${MAX_WAIT}s"
  exit 1
fi

# Check if rule already exists
if grep -qE "^host\s+all\s+all\s+${SUBNET//\//\\/}\s+md5" "$PG_HBA_CONF" 2>/dev/null; then
  echo "✅ Overlay network rule already exists in pg_hba.conf"
  exit 0
fi

echo "➕ Adding overlay network rule to pg_hba.conf"

# Create backup
cp "$PG_HBA_CONF" "$PG_HBA_CONF.backup.$(date +%Y%m%d_%H%M%S)" 2>/dev/null || true

# Insert rule after IPv4 localhost connections (standard PostgreSQL location)
if grep -qE "^host\s+all\s+all\s+127\.0\.0\.1" "$PG_HBA_CONF"; then
  sed -i "/^host\s+all\s+all\s+127\.0\.0\.1/a host    all    all    $SUBNET    md5" "$PG_HBA_CONF"
elif grep -q "^local" "$PG_HBA_CONF"; then
  sed -i "/^local/a host    all    all    $SUBNET    md5" "$PG_HBA_CONF"
else
  echo "host    all    all    $SUBNET    md5" >> "$PG_HBA_CONF"
fi

echo "✅ Added rule: host    all    all    $SUBNET    md5"
echo "✅ PostgreSQL pg_hba.conf configuration complete"

