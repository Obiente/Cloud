#!/bin/bash
# Wrapper script to merge docker-compose.base.yml with compose files
# This allows YAML anchors to work across files

set -e

BASE_FILE="docker-compose.base.yml"
COMPOSE_FILE="${1:-docker-compose.yml}"

if [ ! -f "$BASE_FILE" ]; then
    echo "Error: $BASE_FILE not found"
    exit 1
fi

if [ ! -f "$COMPOSE_FILE" ]; then
    echo "Error: $COMPOSE_FILE not found"
    exit 1
fi

# Create temporary merged file in /tmp
TMP_FILE=$(mktemp)
PROJECT_DIR="$(pwd)"
{
    cat "$BASE_FILE"
    echo ""
    # Remove 'include' line if present and comments about include
    grep -v "^include:" "$COMPOSE_FILE" | grep -v "^  - docker-compose.base.yml" | grep -v "^# This file includes\|^# Usage:.*include"
} > "$TMP_FILE"

# Run docker compose with merged file
# Use --project-directory to ensure relative paths resolve correctly
shift || true
docker compose --project-directory "$PROJECT_DIR" -f "$TMP_FILE" "$@"
EXIT_CODE=$?

# Cleanup
rm -f "$TMP_FILE"

exit $EXIT_CODE

