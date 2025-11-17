#!/bin/bash
# Custom entrypoint wrapper for PostgreSQL that properly configures pg_hba.conf
# This extends the official postgres entrypoint with overlay network configuration
# Works for both new database initialization and existing databases
#
# IMPORTANT: PostgreSQL's official entrypoint may regenerate pg_hba.conf during initialization.
# This wrapper configures pg_hba.conf AFTER PostgreSQL has initialized it, ensuring our
# overlay network rule persists.

set -e

PGDATA=${PGDATA:-/var/lib/postgresql/data}
PG_HBA_CONF="$PGDATA/pg_hba.conf"

# Function to configure pg_hba.conf for overlay network
# This should be called AFTER PostgreSQL has initialized the file
configure_pg_hba() {
  local subnet="${OVERLAY_SUBNET:-10.15.3.0/24}"
  
  # Wait for pg_hba.conf to exist
  local max_wait=60
  local waited=0
  while [ ! -f "$PG_HBA_CONF" ] && [ $waited -lt $max_wait ]; do
    sleep 1
    waited=$((waited + 1))
  done
  
  if [ ! -f "$PG_HBA_CONF" ]; then
    echo "⚠️  pg_hba.conf not found after waiting ${max_wait}s"
    return 1
  fi
  
  # Check if rule already exists
  if grep -qE "^host\s+all\s+all\s+${subnet//\//\\/}\s+md5" "$PG_HBA_CONF" 2>/dev/null; then
    echo "✅ Overlay network rule already exists in pg_hba.conf"
    return 0
  fi
  
  echo "🔧 Configuring pg_hba.conf for overlay network: $subnet"
  
  # Create backup
  cp "$PG_HBA_CONF" "$PG_HBA_CONF.backup.$(date +%Y%m%d_%H%M%S)" 2>/dev/null || true
  
  # Remove any conflicting "host all all all" rules that might interfere
  sed -i '/^host\s\+all\s\+all\s\+all\s/d' "$PG_HBA_CONF" 2>/dev/null || true
  
  # Insert rule after IPv4 localhost connections (standard PostgreSQL location)
  if grep -qE "^host\s+all\s+all\s+127\.0\.0\.1" "$PG_HBA_CONF"; then
    sed -i "/^host\s+all\s+all\s+127\.0\.0\.1/a host    all    all    $subnet    md5" "$PG_HBA_CONF"
  elif grep -q "^local" "$PG_HBA_CONF"; then
    sed -i "/^local/a host    all    all    $subnet    md5" "$PG_HBA_CONF"
  else
    echo "host    all    all    $subnet    md5" >> "$PG_HBA_CONF"
  fi
  
  echo "✅ Added overlay network rule to pg_hba.conf"
  
  # Reload configuration if PostgreSQL is running
  if pg_isready -U "${POSTGRES_USER:-obiente_postgres}" >/dev/null 2>&1; then
    echo "🔄 Reloading PostgreSQL configuration..."
    psql -U "${POSTGRES_USER:-obiente_postgres}" -d "${POSTGRES_DB:-obiente}" -c "SELECT pg_reload_conf();" >/dev/null 2>&1 && echo "✅ Configuration reloaded" || echo "⚠️  Could not reload (will be applied on next restart)"
  fi
  
  return 0
}

# The issue: PostgreSQL's entrypoint may regenerate pg_hba.conf during initialization.
# Solution: Configure pg_hba.conf AFTER PostgreSQL has fully initialized and started.
# We use a background process that retries configuration multiple times.

# Function that runs after PostgreSQL starts (with retries)
post_start_configure() {
  # Wait for PostgreSQL to be ready
  local max_wait=120
  local waited=0
  while ! pg_isready -U "${POSTGRES_USER:-obiente_postgres}" >/dev/null 2>&1 && [ $waited -lt $max_wait ]; do
    sleep 1
    waited=$((waited + 1))
  done
  
  if ! pg_isready -U "${POSTGRES_USER:-obiente_postgres}" >/dev/null 2>&1; then
    echo "⚠️  PostgreSQL did not become ready, skipping pg_hba.conf configuration"
    return 1
  fi
  
  # Wait a bit more to ensure pg_hba.conf is finalized
  sleep 3
  
  # Try to configure multiple times (PostgreSQL might regenerate it during startup)
  for attempt in 1 2 3 4 5; do
    if configure_pg_hba; then
      return 0
    fi
    if [ $attempt -lt 5 ]; then
      echo "⚠️  Configuration attempt $attempt failed, retrying in 2s..."
      sleep 2
    fi
  done
  
  echo "⚠️  Could not configure pg_hba.conf automatically after 5 attempts"
  echo "   You may need to run the fix script manually: ./scripts/fix-postgres-hba.sh"
  return 1
}

# Start configuration monitor in background
# This will configure pg_hba.conf after PostgreSQL has started
post_start_configure &

# Call the original PostgreSQL entrypoint
# This will initialize the database if needed, then start PostgreSQL
exec /usr/local/bin/docker-entrypoint.sh "$@"

