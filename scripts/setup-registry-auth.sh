#!/bin/bash
# Setup script for Docker Registry authentication
# Generates htpasswd file for registry authentication

set -e

REGISTRY_USERNAME="${REGISTRY_USERNAME:-obiente}"
REGISTRY_PASSWORD="${REGISTRY_PASSWORD:-}"
AUTH_DIR="${AUTH_DIR:-/var/lib/obiente/registry-auth}"
HTPASSWD_FILE="${AUTH_DIR}/htpasswd"

# Create auth directory if it doesn't exist
mkdir -p "$AUTH_DIR"

# Check if htpasswd file already exists
if [ -f "$HTPASSWD_FILE" ]; then
    echo "✅ Registry htpasswd file already exists at $HTPASSWD_FILE"
    exit 0
fi

# Check if htpasswd command is available
if ! command -v htpasswd &> /dev/null; then
    echo "❌ Error: htpasswd command not found"
    echo "   Install apache2-utils (Debian/Ubuntu) or httpd-tools (RHEL/CentOS):"
    echo "   Debian/Ubuntu: sudo apt-get install apache2-utils"
    echo "   RHEL/CentOS: sudo yum install httpd-tools"
    exit 1
fi

# Generate password if not provided
if [ -z "$REGISTRY_PASSWORD" ]; then
    echo "⚠️  Warning: REGISTRY_PASSWORD not set, generating random password..."
    REGISTRY_PASSWORD=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
    echo "📝 Generated password: $REGISTRY_PASSWORD"
    echo "   Save this password securely! You'll need it to push/pull images."
fi

# Create htpasswd file
echo "🔐 Creating htpasswd file for user: $REGISTRY_USERNAME"
htpasswd -Bbn "$REGISTRY_USERNAME" "$REGISTRY_PASSWORD" > "$HTPASSWD_FILE"

# Set proper permissions
chmod 600 "$HTPASSWD_FILE"
chown root:root "$HTPASSWD_FILE"

echo "✅ Registry authentication configured successfully!"
echo "   Username: $REGISTRY_USERNAME"
echo "   Password: $REGISTRY_PASSWORD"
echo "   htpasswd file: $HTPASSWD_FILE"
echo ""
echo "📋 To use the registry:"
echo "   docker login registry:5000 -u $REGISTRY_USERNAME -p '$REGISTRY_PASSWORD'"
echo ""
echo "💡 To set a custom password, use:"
echo "   REGISTRY_USERNAME=myuser REGISTRY_PASSWORD=mypassword ./scripts/setup-registry-auth.sh"

