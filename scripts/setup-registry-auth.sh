#!/bin/bash
# Setup script for Docker Registry authentication
# Generates htpasswd file for registry authentication

set -e

# Parse command line arguments
FORCE_UPDATE=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --force)
            FORCE_UPDATE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--force]"
            echo "  --force    Update password even if htpasswd file exists"
            exit 1
            ;;
    esac
done

REGISTRY_USERNAME="${REGISTRY_USERNAME:-obiente}"
REGISTRY_PASSWORD="${REGISTRY_PASSWORD:-}"
AUTH_DIR="${AUTH_DIR:-/var/lib/obiente/registry-auth}"
HTPASSWD_FILE="${AUTH_DIR}/htpasswd"

# Create auth directory if it doesn't exist
mkdir -p "$AUTH_DIR"

# Check if htpasswd file already exists
if [ -f "$HTPASSWD_FILE" ] && [ "$FORCE_UPDATE" = false ]; then
    echo "✅ Registry htpasswd file already exists at $HTPASSWD_FILE"
    echo "💡 To update the password, use: $0 --force"
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

# Create or update htpasswd file
if [ -f "$HTPASSWD_FILE" ] && [ "$FORCE_UPDATE" = true ]; then
    echo "🔄 Updating htpasswd file for user: $REGISTRY_USERNAME"
    # Use htpasswd -Bb to update existing file (without -n flag, it updates in place)
    echo "$REGISTRY_PASSWORD" | htpasswd -Bi "$HTPASSWD_FILE" "$REGISTRY_USERNAME"
else
    echo "🔐 Creating htpasswd file for user: $REGISTRY_USERNAME"
    htpasswd -Bbn "$REGISTRY_USERNAME" "$REGISTRY_PASSWORD" > "$HTPASSWD_FILE"
fi

# Set proper permissions
chmod 600 "$HTPASSWD_FILE"
chown root:root "$HTPASSWD_FILE"

if [ "$FORCE_UPDATE" = true ]; then
    echo "✅ Registry authentication updated successfully!"
else
    echo "✅ Registry authentication configured successfully!"
fi
echo "   Username: $REGISTRY_USERNAME"
echo "   Password: $REGISTRY_PASSWORD"
echo "   htpasswd file: $HTPASSWD_FILE"
echo ""
echo "⚠️  IMPORTANT: Copy and paste these environment variables into your .env file:"
echo ""
echo "REGISTRY_USERNAME=$REGISTRY_USERNAME"
echo "REGISTRY_PASSWORD=$REGISTRY_PASSWORD"
echo ""
echo "   These variables are REQUIRED for the API service to authenticate with the registry"
echo "   and for multi-node Swarm deployments to pull images on worker nodes."
echo ""
echo "📋 To use the registry manually:"
echo "   docker login registry.yourdomain.com -u $REGISTRY_USERNAME -p '$REGISTRY_PASSWORD'"
echo ""
echo "💡 To set a custom password, use:"
echo "   REGISTRY_USERNAME=myuser REGISTRY_PASSWORD=mypassword ./scripts/setup-registry-auth.sh"
echo ""
echo "💡 To update an existing password, use:"
echo "   REGISTRY_PASSWORD=newpassword ./scripts/setup-registry-auth.sh --force"
echo "   (Don't forget to update REGISTRY_PASSWORD in your .env file and restart services!)"

