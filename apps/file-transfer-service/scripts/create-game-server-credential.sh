#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 4 ]; then
  echo "Usage: $0 <name> <game-server-id> <user-id> <organization-id> [scopes] [expires-at]"
  echo "Example: $0 'World upload' gs-123 user-123 org-123 read,write"
  exit 1
fi

NAME="$1"
GAME_SERVER_ID="$2"
USER_ID="$3"
ORGANIZATION_ID="$4"
SCOPES="${5:-read,write}"
EXPIRES_AT="${6:-}"

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-obiente_postgres}"
DB_PASSWORD="${DB_PASSWORD:-obiente_postgres}"
DB_NAME="${DB_NAME:-obiente}"

if command -v openssl >/dev/null 2>&1; then
  SECRET="oft_$(openssl rand -base64 32 | tr '+/' '-_' | tr -d '=')"
else
  SECRET="oft_$(head -c 32 /dev/urandom | base64 | tr '+/' '-_' | tr -d '=')"
fi

KEY_HASH="$(printf '%s' "$SECRET" | sha256sum | awk '{print $1}')"
CREDENTIAL_ID="ftc-$(date +%s)-$(head -c 6 /dev/urandom | od -An -tx1 | tr -d ' \n')"

export PGPASSWORD="$DB_PASSWORD"

if [ -n "$EXPIRES_AT" ]; then
  EXPIRES_SQL="'$EXPIRES_AT'::timestamptz"
else
  EXPIRES_SQL="NULL"
fi

psql \
  --host="$DB_HOST" \
  --port="$DB_PORT" \
  --username="$DB_USER" \
  --dbname="$DB_NAME" \
  --set=ON_ERROR_STOP=1 \
  --command="
    INSERT INTO file_transfer_credentials (
      id, name, key_hash, user_id, organization_id, resource_type, resource_id, scopes, expires_at, created_at, updated_at
    ) VALUES (
      '$CREDENTIAL_ID',
      '$NAME',
      '$KEY_HASH',
      '$USER_ID',
      '$ORGANIZATION_ID',
      'gameserver',
      '$GAME_SERVER_ID',
      '$SCOPES',
      $EXPIRES_SQL,
      now(),
      now()
    );
  "

cat <<EOF
Created file transfer credential:
  id:       $CREDENTIAL_ID
  resource: gameserver:$GAME_SERVER_ID
  scopes:   $SCOPES

SFTP connection:
  host:     <node-host>
  port:     \${FILE_TRANSFER_SFTP_PUBLIC_PORT:-2223}
  username: any value
  password: $SECRET

Save the password now. Only its SHA-256 hash is stored.
EOF
