#!/bin/bash
# Custom entrypoint wrapper for PostgreSQL that configures postgresql.conf
# to use our custom pg_hba.conf file and set listen_addresses=*
#
# We mount a custom pg_hba.conf file as a Docker config to prevent PostgreSQL
# from regenerating it. This wrapper ensures postgresql.conf points to it.

set -e

PGDATA=${PGDATA:-/var/lib/postgresql/data}
POSTGRESQL_CONF="$PGDATA/postgresql.conf"
CUSTOM_HBA="/etc/postgresql/pg_hba.conf"

# Function to configure postgresql.conf
configure_postgresql_conf() {
  echo "🔧 Configuring postgresql.conf..."
  
  # Wait for postgresql.conf to exist
  local max_wait=60
  local waited=0
  while [ ! -f "$POSTGRESQL_CONF" ] && [ $waited -lt $max_wait ]; do
    sleep 1
    waited=$((waited + 1))
  done
  
  if [ ! -f "$POSTGRESQL_CONF" ]; then
    echo "⚠️  postgresql.conf not found, will be configured via command line"
    return 0
  fi
  
  # Copy custom pg_hba.conf to PGDATA if it exists
  if [ -f "$CUSTOM_HBA" ]; then
    echo "📋 Copying custom pg_hba.conf to $PGDATA..."
    cp "$CUSTOM_HBA" "$PGDATA/pg_hba.conf"
    chmod 0600 "$PGDATA/pg_hba.conf"
    echo "✅ Custom pg_hba.conf installed"
  fi
  
  # Configure listen_addresses=* if not already set
  if ! grep -qE "^listen_addresses\s*=" "$POSTGRESQL_CONF" 2>/dev/null; then
    echo "📝 Adding listen_addresses=* to postgresql.conf..."
    echo "listen_addresses = '*'" >> "$POSTGRESQL_CONF"
    echo "✅ listen_addresses configured"
  elif ! grep -qE "^listen_addresses\s*=\s*'\*'" "$POSTGRESQL_CONF" 2>/dev/null; then
    echo "📝 Updating listen_addresses to '*' in postgresql.conf..."
    sed -i "s/^listen_addresses\s*=.*/listen_addresses = '*'/" "$POSTGRESQL_CONF"
    echo "✅ listen_addresses updated"
  else
    echo "✅ listen_addresses already set to '*'"
  fi
  
  return 0
}

# Configure postgresql.conf before starting PostgreSQL
configure_postgresql_conf

# Call the original PostgreSQL entrypoint
# This will initialize the database if needed, then start PostgreSQL
exec /usr/local/bin/docker-entrypoint.sh "$@"

