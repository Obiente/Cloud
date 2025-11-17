#!/bin/bash
# Init script to copy pg_hba.conf to PGDATA
# This runs after database initialization via /docker-entrypoint-initdb.d/
# Ensures pg_hba.conf is always up to date with overlay network rules and allowed hosts

PGDATA=${PGDATA:-/var/lib/postgresql/data}
HBA_SOURCE="/docker-entrypoint-initdb.d/pg_hba.conf"
HBA_TARGET="$PGDATA/pg_hba.conf"

# Wait for pg_hba.conf to be created by initdb (if not already initialized)
if [ ! -f "$HBA_TARGET" ]; then
  echo "⏳ Waiting for PostgreSQL to create pg_hba.conf..."
  MAX_WAIT=30
  WAITED=0
  while [ ! -f "$HBA_TARGET" ] && [ $WAITED -lt $MAX_WAIT ]; do
    sleep 1
    WAITED=$((WAITED + 1))
  done
fi

# Copy pg_hba.conf if source exists
if [ -f "$HBA_SOURCE" ]; then
  if [ ! -f "$HBA_TARGET" ] || ! cmp -s "$HBA_SOURCE" "$HBA_TARGET"; then
    echo "📋 Copying custom pg_hba.conf to $PGDATA..."
    cp "$HBA_SOURCE" "$HBA_TARGET"
    chmod 0600 "$HBA_TARGET"
    chown postgres:postgres "$HBA_TARGET" 2>/dev/null || true
    
    # Verify overlay network rule is present
    if grep -qE "^host\s+all\s+all\s+10\.15\.3\.0/24\s+md5" "$HBA_TARGET" 2>/dev/null; then
      echo "✅ Custom pg_hba.conf installed with overlay network rule"
      echo "   Rule: $(grep -E '^host\s+all\s+all\s+10\.15\.3\.0/24\s+md5' "$HBA_TARGET")"
    else
      echo "⚠️  Warning: Overlay network rule not found in pg_hba.conf!"
    fi
  else
    echo "✅ pg_hba.conf is already up to date"
  fi
else
  echo "⚠️  Custom pg_hba.conf not found at $HBA_SOURCE"
fi

# Add allowed hosts from environment variable
# Format: comma-separated list of IPs or subnets (e.g., "10.10.10.1,10.0.0.0/8")
# Supports both POSTGRES_ALLOWED_HOSTS and METRICS_DB_ALLOWED_HOSTS
ALLOWED_HOSTS="${POSTGRES_ALLOWED_HOSTS:-}"

if [ -n "$ALLOWED_HOSTS" ]; then
  echo "🔧 Adding allowed hosts to pg_hba.conf..."
  
  # Convert comma-separated list to array
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
    if grep -qE "^host\s+all\s+all\s+${escaped_host}\s+md5" "$HBA_TARGET" 2>/dev/null; then
      echo "   ✅ Rule already exists for $normalized_host"
      continue
    fi
    
    # Add rule after the overlay network rule (or at the end if not found)
    if grep -qE "^host\s+all\s+all\s+10\.15\.3\.0/24\s+md5" "$HBA_TARGET" 2>/dev/null; then
      # Insert after overlay network rule
      sed -i "/^host\s+all\s+all\s+10\.15\.3\.0\/24\s+md5/a host    all    all    $normalized_host    md5" "$HBA_TARGET"
    else
      # Append at the end
      echo "host    all    all    $normalized_host    md5" >> "$HBA_TARGET"
    fi
    
    echo "   ✅ Added rule for $normalized_host"
  done
  
  echo "✅ Allowed hosts configuration complete"
  
  # Reload PostgreSQL configuration to apply changes
  if [ -f "$PGDATA/postmaster.pid" ]; then
    echo "🔄 Reloading PostgreSQL configuration..."
    PID=$(head -1 "$PGDATA/postmaster.pid" 2>/dev/null || echo "")
    if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null; then
      kill -HUP "$PID" 2>/dev/null && echo "   ✅ Configuration reloaded" || echo "   ⚠️  Failed to reload (may need restart)"
    else
      echo "   ⚠️  PostgreSQL not running, changes will apply on next start"
    fi
  else
    echo "ℹ️  PostgreSQL not running, changes will apply on next start"
  fi
else
  echo "ℹ️  No allowed hosts configured (POSTGRES_ALLOWED_HOSTS not set)"
fi

