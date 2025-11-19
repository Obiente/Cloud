#!/bin/bash
# Setup script to run on ALL Swarm nodes (managers and workers)
# This script can be copied to and run on each node
# Usage: Copy this script to each node and run it before deploying

set -e

echo "🔧 Setting up Obiente Cloud on $(hostname)..."

# Create required directories
mkdir -p /var/lib/obiente/volumes
mkdir -p /var/obiente/tmp/obiente-volumes
mkdir -p /var/obiente/tmp/obiente-deployments

# Set permissions (ensure Docker can access)
chmod 755 /var/lib/obiente
chmod 755 /var/obiente
chmod 755 /var/obiente/tmp
chmod 755 /var/obiente/tmp/obiente-volumes
chmod 755 /var/obiente/tmp/obiente-deployments

# Configure Docker daemon for IPv6
DOCKER_DAEMON_JSON="/etc/docker/daemon.json"
IPV6_SUBNET="fd00:0b1e:c10d::/64"

echo ""
echo "🔧 Configuring Docker daemon for IPv6..."

# Check if daemon.json exists
if [ -f "$DOCKER_DAEMON_JSON" ]; then
    # Check if IPv6 is already configured
    if grep -q "\"ipv6\"" "$DOCKER_DAEMON_JSON" && grep -q "$IPV6_SUBNET" "$DOCKER_DAEMON_JSON"; then
        echo "✅ IPv6 already configured in Docker daemon"
    else
        echo "⚠️  Docker daemon.json exists but IPv6 not configured correctly"
        echo "   Please manually update $DOCKER_DAEMON_JSON with:"
        echo "   {"
        echo "     \"ipv6\": true,"
        echo "     \"fixed-cidr-v6\": \"$IPV6_SUBNET\""
        echo "   }"
        echo "   Then run: sudo systemctl restart docker"
    fi
else
    # Create daemon.json with IPv6 configuration
    echo "Creating $DOCKER_DAEMON_JSON with IPv6 configuration..."
    sudo tee "$DOCKER_DAEMON_JSON" > /dev/null <<EOF
{
  "ipv6": true,
  "fixed-cidr-v6": "$IPV6_SUBNET"
}
EOF
    echo "✅ Docker daemon.json created with IPv6 configuration"
    echo "⚠️  Restarting Docker daemon..."
    sudo systemctl restart docker
    echo "✅ Docker daemon restarted"
fi

# Verify IPv6 is enabled
echo ""
echo "🔍 Verifying IPv6 configuration..."
if docker info 2>/dev/null | grep -qi "ipv6"; then
    echo "✅ IPv6 is enabled in Docker"
else
    echo "❌ IPv6 is NOT enabled in Docker"
    echo "   Please check $DOCKER_DAEMON_JSON and restart Docker"
fi

echo ""
echo "✅ Setup complete!"
echo ""
echo "Created directories:"
echo "  - /var/lib/obiente/volumes"
echo "  - /var/obiente/tmp/obiente-volumes"
echo "  - /var/obiente/tmp/obiente-deployments"
echo ""
echo "This node is ready for deployment."

