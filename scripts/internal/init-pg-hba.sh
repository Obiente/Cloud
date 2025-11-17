#!/bin/bash
# Simple init script to copy pg_hba.conf to PGDATA
# This runs after PostgreSQL initializes the database

set -e

PGDATA=${PGDATA:-/var/lib/postgresql/data}
HBA_SOURCE="/docker-entrypoint-initdb.d/pg_hba.conf"
HBA_TARGET="$PGDATA/pg_hba.conf"

if [ -f "$HBA_SOURCE" ]; then
  echo "📋 Copying custom pg_hba.conf to $PGDATA..."
  cp "$HBA_SOURCE" "$HBA_TARGET"
  chmod 0600 "$HBA_TARGET"
  chown postgres:postgres "$HBA_TARGET" 2>/dev/null || true
  echo "✅ Custom pg_hba.conf installed"
else
  echo "⚠️  Custom pg_hba.conf not found at $HBA_SOURCE"
fi

