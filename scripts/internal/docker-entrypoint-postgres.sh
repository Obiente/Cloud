#!/bin/bash
# Custom entrypoint wrapper for PostgreSQL that properly configures pg_hba.conf
# This extends the official postgres entrypoint with overlay network configuration
# Works for both new database initialization and existing databases

set -e

PGDATA=${PGDATA:-/var/lib/postgresql/data}
PG_HBA_CONF="$PGDATA/pg_hba.conf"

# Function to configure pg_hba.conf for overlay network
configure_pg_hba() {
  local subnet="${OVERLAY_SUBNET:-10.15.3.0/24}"
  
  # Wait for pg_hba.conf to exist (may not exist during first init)
  local max_wait=30
  local waited=0
  while [ ! -f "$PG_HBA_CONF" ] && [ $waited -lt $max_wait ]; do
    sleep 1
    waited=$((waited + 1))
  done
  
  # If still doesn't exist, it will be configured by init script
  if [ ! -f "$PG_HBA_CONF" ]; then
    return 0
  fi
  
  # Check if rule already exists
  if grep -qE "^host\s+all\s+all\s+${subnet//\//\\/}\s+md5" "$PG_HBA_CONF" 2>/dev/null; then
    return 0
  fi
  
  echo "🔧 Configuring pg_hba.conf for overlay network: $subnet"
  
  # Create backup
  cp "$PG_HBA_CONF" "$PG_HBA_CONF.backup.$(date +%Y%m%d_%H%M%S)" 2>/dev/null || true
  
  # Insert rule after IPv4 localhost connections (standard PostgreSQL location)
  if grep -qE "^host\s+all\s+all\s+127\.0\.0\.1" "$PG_HBA_CONF"; then
    sed -i "/^host\s+all\s+all\s+127\.0\.0\.1/a host    all    all    $subnet    md5" "$PG_HBA_CONF"
  elif grep -q "^local" "$PG_HBA_CONF"; then
    sed -i "/^local/a host    all    all    $subnet    md5" "$PG_HBA_CONF"
  else
    echo "host    all    all    $subnet    md5" >> "$PG_HBA_CONF"
  fi
  
  echo "✅ Added overlay network rule to pg_hba.conf"
}

# Configure pg_hba.conf before starting PostgreSQL
configure_pg_hba

# Call the original PostgreSQL entrypoint
exec /usr/local/bin/docker-entrypoint.sh "$@"

