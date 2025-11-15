#!/bin/bash
# Script to create required directories on all Docker Swarm nodes
# Run this on each node before deploying the stack

set -e

echo "Creating Obiente Cloud directories..."

# Create main directories
sudo mkdir -p /var/lib/obiente/volumes
sudo mkdir -p /var/lib/obiente/deployments
sudo mkdir -p /var/lib/obiente/builds
sudo mkdir -p /var/obiente/tmp/obiente-volumes
sudo mkdir -p /var/obiente/tmp/obiente-deployments

# Set appropriate permissions (adjust as needed)
sudo chmod 755 /var/lib/obiente
sudo chmod 755 /var/lib/obiente/volumes
sudo chmod 755 /var/lib/obiente/deployments
sudo chmod 755 /var/lib/obiente/builds
sudo chmod 755 /var/obiente
sudo chmod 755 /var/obiente/tmp
sudo chmod 755 /var/obiente/tmp/obiente-volumes
sudo chmod 755 /var/obiente/tmp/obiente-deployments

echo "✓ Directories created successfully"
echo ""
echo "Directories created:"
echo "  - /var/lib/obiente/volumes"
echo "  - /var/lib/obiente/deployments"
echo "  - /var/lib/obiente/builds"
echo "  - /var/obiente/tmp/obiente-volumes"
echo "  - /var/obiente/tmp/obiente-deployments"

