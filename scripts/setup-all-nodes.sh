#!/bin/bash
# Setup script to run on ALL Swarm nodes
# This script can be copied to and run on each worker node
# Usage: Copy this script to each node and run it before deploying

set -e

echo "🔧 Setting up Obiente Cloud directories on $(hostname)..."

# Create required directories
mkdir -p /var/lib/obiente/volumes
mkdir -p /tmp/obiente-volumes
mkdir -p /tmp/obiente-deployments

# Set permissions (ensure Docker can access)
chmod 755 /var/lib/obiente
chmod 755 /tmp/obiente-volumes
chmod 755 /tmp/obiente-deployments

echo "✅ Directories created successfully!"
echo ""
echo "Created directories:"
echo "  - /var/lib/obiente/volumes"
echo "  - /tmp/obiente-volumes"
echo "  - /tmp/obiente-deployments"
echo ""
echo "This node is ready for deployment."

