#!/bin/bash
# Example script to create an API key for SFTP access
# This script generates a random API key, hashes it, and inserts it into the database

set -e

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-obiente_postgres}"
DB_NAME="${DB_NAME:-obiente}"

# Check if required arguments are provided
if [ "$#" -lt 3 ]; then
    echo "Usage: $0 <name> <user_id> <organization_id> [scopes]"
    echo ""
    echo "Arguments:"
    echo "  name              - Friendly name for the API key (e.g., 'SFTP Upload Key')"
    echo "  user_id           - User ID who owns the key"
    echo "  organization_id   - Organization ID"
    echo "  scopes            - Comma-separated scopes (default: 'sftp:*')"
    echo ""
    echo "Scope options:"
    echo "  sftp:read         - Read-only access (download, list)"
    echo "  sftp:write        - Write access (upload, delete, mkdir)"
    echo "  sftp:*            - Full access (read + write)"
    echo "  sftp              - Full access (read + write)"
    echo ""
    echo "Example:"
    echo "  $0 'My SFTP Key' user-123 org-456 'sftp:read,sftp:write'"
    exit 1
fi

NAME="$1"
USER_ID="$2"
ORG_ID="$3"
SCOPES="${4:-sftp:*}"

# Generate a random API key (32 characters)
API_KEY=$(openssl rand -hex 16)

# Generate UUID for the key ID
KEY_ID=$(uuidgen | tr '[:upper:]' '[:lower:]')

# Hash the API key using SHA-256
KEY_HASH=$(echo -n "$API_KEY" | sha256sum | awk '{print $1}')

echo "=========================================="
echo "API Key Created Successfully!"
echo "=========================================="
echo ""
echo "API Key ID:      $KEY_ID"
echo "API Key:         $API_KEY"
echo ""
echo "⚠️  IMPORTANT: Save this API key securely!"
echo "⚠️  It will not be shown again."
echo ""
echo "Details:"
echo "  Name:            $NAME"
echo "  User ID:         $USER_ID"
echo "  Organization ID: $ORG_ID"
echo "  Scopes:          $SCOPES"
echo ""

# Insert into database
PGPASSWORD="$POSTGRES_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" << EOF
INSERT INTO api_keys (
    id,
    name,
    key_hash,
    user_id,
    organization_id,
    scopes,
    created_at,
    updated_at
) VALUES (
    '$KEY_ID',
    '$NAME',
    '$KEY_HASH',
    '$USER_ID',
    '$ORG_ID',
    '$SCOPES',
    NOW(),
    NOW()
);
EOF

if [ $? -eq 0 ]; then
    echo "✓ API key inserted into database"
    echo ""
    echo "To use this key with SFTP:"
    echo "  sftp -P 2222 user@your-hostname"
    echo "  Password: $API_KEY"
    echo ""
else
    echo "✗ Failed to insert API key into database"
    exit 1
fi
