#!/bin/bash
# Init script to copy pg_hba.conf to PGDATA
# This runs after database initialization via /docker-entrypoint-initdb.d/
# Ensures pg_hba.conf is always up to date with overlay network rules

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

