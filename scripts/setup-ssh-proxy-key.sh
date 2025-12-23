#!/bin/bash
# Script to generate and create SSH proxy host key for Docker Swarm
# This ensures all vps-service replicas use the same SSH host key

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SECRET_NAME="ssh_proxy_host_key"
KEY_FILE="/tmp/ssh_proxy_host_key_$$"

echo "=================================================="
echo "SSH Proxy Host Key Setup for Docker Swarm"
echo "=================================================="
echo ""

# Check if running in Swarm mode
if ! docker info 2>/dev/null | grep -q "Swarm: active"; then
    echo "❌ Error: Docker is not in Swarm mode"
    echo "   Initialize Swarm first with: docker swarm init"
    exit 1
fi

# Check if secret already exists
if docker secret inspect "$SECRET_NAME" >/dev/null 2>&1; then
    echo "⚠️  Secret '$SECRET_NAME' already exists!"
    echo ""
    read -p "Do you want to rotate it? (yes/no): " -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        echo "Aborted. Secret not changed."
        exit 0
    fi
    
    echo "🔄 Rotating SSH host key..."
    echo "   Users will see 'host key changed' warnings and need to:"
    echo "   ssh-keygen -R \"[your-domain]:2222\""
    echo ""
    read -p "Continue? (yes/no): " -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        echo "Aborted."
        exit 0
    fi
    
    ROTATING=true
else
    ROTATING=false
fi

# Generate new SSH host key
echo "🔑 Generating new RSA host key (2048-bit)..."
ssh-keygen -t rsa -b 2048 -f "$KEY_FILE" -N "" -C "obiente-ssh-proxy" >/dev/null 2>&1

if [ ! -f "$KEY_FILE" ]; then
    echo "❌ Error: Failed to generate SSH key"
    exit 1
fi

# Display fingerprint
echo "✅ Key generated successfully"
echo ""
echo "📋 Key fingerprint:"
ssh-keygen -lf "$KEY_FILE.pub"
echo ""

# Remove old secret if rotating
if [ "$ROTATING" = true ]; then
    echo "🗑️  Removing old secret..."
    docker secret rm "$SECRET_NAME" >/dev/null 2>&1 || true
fi

# Create Docker secret
echo "📦 Creating Docker secret '$SECRET_NAME'..."
docker secret create "$SECRET_NAME" "$KEY_FILE"

if [ $? -eq 0 ]; then
    echo "✅ Secret created successfully"
else
    echo "❌ Error: Failed to create Docker secret"
    rm -f "$KEY_FILE" "$KEY_FILE.pub"
    exit 1
fi

# Clean up temporary files
echo "🧹 Cleaning up temporary files..."
rm -f "$KEY_FILE" "$KEY_FILE.pub"

echo ""
echo "=================================================="
echo "✨ Setup Complete!"
echo "=================================================="
echo ""

if [ "$ROTATING" = true ]; then
    echo "⚠️  Important: After redeploying vps-service, users will need to:"
    echo "   ssh-keygen -R \"[your-domain]:2222\""
    echo ""
    echo "Next steps:"
    echo "1. Redeploy vps-service:"
    echo "   docker service update --force obiente_vps-service"
    echo ""
    echo "2. Verify all replicas use the same key:"
    echo "   docker service logs obiente_vps-service | grep fingerprint"
else
    echo "Next steps:"
    echo "1. Deploy your stack:"
    echo "   docker stack deploy -c docker-compose.swarm.ha.yml obiente"
    echo ""
    echo "2. Verify the key is loaded:"
    echo "   docker service logs obiente_vps-service | grep 'SSH host key'"
    echo ""
    echo "All vps-service replicas will now use the same SSH host key!"
fi

echo ""
echo "📖 For more information, see: docs/deployment/ssh-proxy.md"
echo ""
