#!/bin/bash
# Helper script to merge docker-compose.base.yml with a compose file
# Usage: merge-compose-files.sh <compose-file> [output-file]
# If output-file is not provided, outputs to stdout

set -e

BASE_FILE="docker-compose.base.yml"
COMPOSE_FILE="$1"
OUTPUT_FILE="${2:-}"

if [ -z "$COMPOSE_FILE" ]; then
  echo "Error: Compose file argument required" >&2
  echo "Usage: merge-compose-files.sh <compose-file> [output-file]" >&2
  exit 1
fi

if [ ! -f "$BASE_FILE" ]; then
  echo "Error: $BASE_FILE not found" >&2
  exit 1
fi

if [ ! -f "$COMPOSE_FILE" ]; then
  echo "Error: $COMPOSE_FILE not found" >&2
  exit 1
fi

# Merge base file with compose file
# Remove 'include' lines and related comments from compose file
{
  cat "$BASE_FILE"
  echo ""
  grep -v "^include:" "$COMPOSE_FILE" | \
    grep -v "^  - docker-compose.base.yml" | \
    grep -v "^# This file includes\|^# Usage:.*include\|^# The 'include'"
} > "${OUTPUT_FILE:-/dev/stdout}"

