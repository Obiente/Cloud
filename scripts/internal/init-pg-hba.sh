#!/bin/bash
# Init script to copy pg_hba.conf to PGDATA
# This runs during database initialization AND on container start
# Ensures pg_hba.conf is always up to date with overlay network rules

set -e

PGDATA=${PGDATA:-/var/lib/postgresql/data}
HBA_SOURCE="/docker-entrypoint-initdb.d/pg_hba.conf"
HBA_TARGET="$PGDATA/pg_hba.conf"

# Function to copy and verify pg_hba.conf
copy_pg_hba() {
  if [ -f "$HBA_SOURCE" ]; then
    echo "📋 Copying custom pg_hba.conf to $PGDATA..."
    cp "$HBA_SOURCE" "$HBA_TARGET"
    chmod 0600 "$HBA_TARGET"
    chown postgres:postgres "$HBA_TARGET" 2>/dev/null || true
    
    # Verify overlay network rule is present
    if grep -qE "^host\s+all\s+all\s+10\.15\.3\.0/24\s+md5" "$HBA_TARGET" 2>/dev/null; then
      echo "✅ Custom pg_hba.conf installed with overlay network rule"
      echo "   Rule: $(grep -E '^host\s+all\s+all\s+10\.15\.3\.0/24\s+md5' "$HBA_TARGET")"
      return 0
    else
      echo "⚠️  Warning: Overlay network rule not found in pg_hba.conf!"
      return 1
    fi
  else
    echo "⚠️  Custom pg_hba.conf not found at $HBA_SOURCE"
    return 1
  fi
}

# Copy pg_hba.conf
copy_pg_hba

# If PostgreSQL is already running, reload configuration
if [ -f "$PGDATA/postmaster.pid" ]; then
  echo "🔄 PostgreSQL is running, reloading configuration..."
  # Use pg_ctl reload if available, otherwise use kill -HUP
  if command -v pg_ctl >/dev/null 2>&1; then
    pg_ctl reload -D "$PGDATA" 2>/dev/null || true
  else
    # Try to reload by sending SIGHUP to postmaster
    PID=$(head -1 "$PGDATA/postmaster.pid" 2>/dev/null || echo "")
    if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null; then
      kill -HUP "$PID" 2>/dev/null || true
      echo "✅ Configuration reload signal sent"
    fi
  fi
fi

