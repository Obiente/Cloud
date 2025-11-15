#!/bin/sh
set -e

# This entrypoint runs inside the container
# Note: Bind mounts require host directories to exist BEFORE container starts
# The host directories must be created on each node using scripts/create-obiente-directories.sh
# This entrypoint ensures container-side directories exist (for any container-internal operations)

# Create container-side directories if they don't exist (for internal operations)
# These are separate from the bind-mounted host directories
mkdir -p /tmp/obiente-volumes
mkdir -p /tmp/obiente-deployments

# Execute the main command
exec "$@"
