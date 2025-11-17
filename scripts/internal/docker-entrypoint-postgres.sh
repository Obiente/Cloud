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
  # We do this BEFORE PostgreSQL starts to prevent it from regenerating the file
  if [ -f "$CUSTOM_HBA" ]; then
    echo "📋 Copying custom pg_hba.conf to $PGDATA..."
    cp "$CUSTOM_HBA" "$PGDATA/pg_hba.conf"
    chmod 0600 "$PGDATA/pg_hba.conf"
    chown postgres:postgres "$PGDATA/pg_hba.conf" 2>/dev/null || true
    echo "✅ Custom pg_hba.conf installed"
    
    # Verify the rule is present
    if grep -qE "^host\s+all\s+all\s+10\.15\.3\.0/24\s+md5" "$PGDATA/pg_hba.conf" 2>/dev/null; then
      echo "✅ Verified overlay network rule in pg_hba.conf"
      echo "   Rule: $(grep -E '^host\s+all\s+all\s+10\.15\.3\.0/24\s+md5' "$PGDATA/pg_hba.conf")"
    else
      echo "⚠️  Warning: Overlay network rule not found in copied file!"
      echo "   Current rules:"
      grep -E "^host|^local" "$PGDATA/pg_hba.conf" | head -5 || echo "   (no rules found)"
    fi
  else
    echo "⚠️  Warning: Custom pg_hba.conf not found at $CUSTOM_HBA"
  fi
  
  # Configure listen_addresses=* in postgresql.conf
  # We need to set this in the config file, not just command line, to ensure it persists
  if ! grep -qE "^listen_addresses\s*=" "$POSTGRESQL_CONF" 2>/dev/null; then
    echo "📝 Adding listen_addresses=* to postgresql.conf..."
    echo "listen_addresses = '*'" >> "$POSTGRESQL_CONF"
    echo "✅ listen_addresses configured"
  else
    # Check current value
    CURRENT_VALUE=$(grep -E "^listen_addresses\s*=" "$POSTGRESQL_CONF" | head -1 | sed 's/.*=\s*//' | tr -d " '")
    if [ "$CURRENT_VALUE" != "*" ]; then
      echo "📝 Updating listen_addresses from '$CURRENT_VALUE' to '*' in postgresql.conf..."
      sed -i "s/^listen_addresses\s*=.*/listen_addresses = '*'/" "$POSTGRESQL_CONF"
      echo "✅ listen_addresses updated to '*'"
    else
      echo "✅ listen_addresses already set to '*'"
    fi
  fi
  
  # Also verify it's set correctly
  if grep -qE "^listen_addresses\s*=\s*'\*'" "$POSTGRESQL_CONF" 2>/dev/null; then
    echo "✅ Verified listen_addresses = '*' in postgresql.conf"
  else
    echo "⚠️  Warning: listen_addresses may not be set correctly"
  fi
  
  return 0
}

# Configure postgresql.conf before starting PostgreSQL
configure_postgresql_conf

# Ensure listen_addresses is in the command arguments if not already present
# This ensures it's set even if postgresql.conf doesn't have it
HAS_LISTEN_ARG=false
for arg in "$@"; do
  if echo "$arg" | grep -qE "listen_addresses"; then
    HAS_LISTEN_ARG=true
    break
  fi
done

# If not in arguments, add it
if [ "$HAS_LISTEN_ARG" = false ]; then
  echo "📝 Adding listen_addresses=* to command arguments..."
  set -- "$@" -c "listen_addresses=*"
fi

# Call the original PostgreSQL entrypoint
# This will initialize the database if needed, then start PostgreSQL
exec /usr/local/bin/docker-entrypoint.sh "$@"

