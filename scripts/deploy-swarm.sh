#!/bin/bash
# Quick deploy script for Obiente Cloud Docker Swarm
# Deploys both the main stack and dashboard stack
# Set BUILD_LOCAL=true to build images locally instead
# Set DEPLOY_DASHBOARD=false to skip dashboard deployment

set -e

STACK_NAME="${1:-obiente}"
COMPOSE_FILE="${2:-docker-compose.swarm.yml}"
BUILD_LOCAL="${BUILD_LOCAL:-false}"
DEPLOY_DASHBOARD="${DEPLOY_DASHBOARD:-true}"

# Define all microservice images
REGISTRY="${REGISTRY:-ghcr.io/obiente}"
MICROSERVICES=(
  "api-gateway"
  "audit-service"
  "auth-service"
  "billing-service"
  "deployments-service"
  "dns-service"
  "gameservers-service"
  "orchestrator-service"
  "organizations-service"
  "superadmin-service"
  "support-service"
  "vps-gateway"
  "vps-service"
)
DASHBOARD_IMAGE="${DASHBOARD_IMAGE:-${REGISTRY}/cloud-dashboard:latest}"

# Load .env file if it exists
if [ -f .env ]; then
  echo "📝 Loading environment variables from .env file..."
  # Export variables from .env file (handles comments and empty lines)
  set -a
  source .env
  set +a
elif [ -f .env.example ]; then
  echo "⚠️  Warning: .env file not found. Using .env.example as reference."
  echo "   Copy .env.example to .env and configure it: cp .env.example .env"
fi

# Check if required directories exist on this node
REQUIRED_DIRS=(
  "/var/lib/obiente"
  "/var/obiente/tmp/obiente-volumes"
  "/var/obiente/tmp/obiente-deployments"
  "/var/lib/obiente/registry-auth"
)

MISSING_DIRS=()
for dir in "${REQUIRED_DIRS[@]}"; do
  if [ ! -d "$dir" ]; then
    MISSING_DIRS+=("$dir")
  fi
done

if [ ${#MISSING_DIRS[@]} -gt 0 ]; then
  echo "⚠️  Warning: Required directories missing on this node:"
  for dir in "${MISSING_DIRS[@]}"; do
    echo "   - $dir"
  done
  echo ""
  echo "📋 Creating directories on this node..."
  mkdir -p "${MISSING_DIRS[@]}"
  chmod 755 "${MISSING_DIRS[@]}"
  
  # Special handling for registry-auth directory
  if [[ " ${MISSING_DIRS[@]} " =~ " /var/lib/obiente/registry-auth " ]]; then
    echo ""
    echo "🔐 Registry auth directory created. Setting up authentication..."
    echo "   Run: ./scripts/setup-registry-auth.sh"
    echo "   Or manually create htpasswd file in /var/lib/obiente/registry-auth/"
  fi
  echo "✅ Directories created!"
  echo ""
  echo "⚠️  IMPORTANT: Some services run on ALL nodes (mode: global)."
  echo "   You must create these directories on ALL worker nodes before deployment:"
  echo ""
  echo "   Run this on each worker node:"
  echo "   ./scripts/setup-all-nodes.sh"
  echo ""
  echo "   Or manually:"
  echo "   mkdir -p /var/lib/obiente/volumes"
  echo "   mkdir -p /var/obiente/tmp/obiente-volumes"
  echo "   mkdir -p /var/obiente/tmp/obiente-deployments"
  echo ""
  read -p "Continue with deployment? (y/N) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Deployment cancelled. Please create directories on all nodes first."
    exit 1
  fi
fi

if [ "$BUILD_LOCAL" = "true" ]; then
  echo "🔨 Building Obiente Cloud microservice images locally..."
  
  # Enable BuildKit for faster builds
  export DOCKER_BUILDKIT=1
  
  # Build all microservice images
  for service in "${MICROSERVICES[@]}"; do
    echo "📦 Building obiente/cloud-${service}:latest..."
    docker build -f apps/${service}/Dockerfile -t obiente/cloud-${service}:latest . || {
      echo "❌ Failed to build ${service}"
      exit 1
    }
  done
  
  # Build the Dashboard image
  if [ "$DEPLOY_DASHBOARD" = "true" ]; then
    echo "📦 Building obiente/cloud-dashboard:latest..."
    docker build -f apps/dashboard/Dockerfile -t obiente/cloud-dashboard:latest .
  fi
  
  echo "✅ Build complete!"
else
  echo "📥 Pulling Obiente Cloud microservice images from GitHub Container Registry..."
  
  # Pull all microservice images
  FAILED_PULLS=()
  for service in "${MICROSERVICES[@]}"; do
    IMAGE="${REGISTRY}/cloud-${service}:latest"
    echo "📦 Pulling ${IMAGE}..."
    if ! docker pull "${IMAGE}"; then
      echo "⚠️  Warning: Failed to pull ${service} image"
      FAILED_PULLS+=("${service}")
    fi
  done
  
  # Pull the Dashboard image if deploying dashboard
  if [ "$DEPLOY_DASHBOARD" = "true" ]; then
    echo "📦 Pulling ${DASHBOARD_IMAGE}..."
    if ! docker pull "${DASHBOARD_IMAGE}"; then
      echo "⚠️  Warning: Failed to pull dashboard image"
      FAILED_PULLS+=("dashboard")
    fi
  fi
  
  if [ ${#FAILED_PULLS[@]} -gt 0 ]; then
    echo ""
    echo "❌ Failed to pull the following images:"
    for service in "${FAILED_PULLS[@]}"; do
      echo "   - ${service}"
    done
    echo ""
    echo "Make sure you're authenticated to ghcr.io:"
    echo "   docker login ghcr.io"
    echo ""
    echo "Or set BUILD_LOCAL=true to build locally"
    exit 1
  fi
  
  echo "✅ Image pull complete!"
fi

echo ""
echo "🚀 Deploying main stack '$STACK_NAME'..."

# Network will be created automatically by Swarm when the stack is deployed
# No need to check for pre-existing network - Swarm handles it
NETWORK_NAME="${STACK_NAME}_obiente-network"
echo "ℹ️  Network '$NETWORK_NAME' will be created automatically by Docker Swarm"

# Check if registry auth is configured
if [ ! -f "/var/lib/obiente/registry-auth/htpasswd" ]; then
  echo "⚠️  Warning: Registry authentication not configured!"
  echo ""
  echo "📋 Setup registry authentication:"
  echo "   ./scripts/setup-registry-auth.sh"
  echo ""
  echo "   Or set custom credentials:"
  echo "   REGISTRY_USERNAME=myuser REGISTRY_PASSWORD=mypassword ./scripts/setup-registry-auth.sh"
  echo ""
  read -p "Continue without registry auth? (y/N) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Deployment cancelled. Please setup registry authentication first."
    exit 1
  fi
  echo "⚠️  Continuing without registry authentication (registry will reject push/pull requests)"
fi

# Merge docker-compose.base.yml with the compose file
# YAML anchors don't work across files, so we merge them first
MERGED_COMPOSE=$(mktemp)
./scripts/merge-compose-files.sh "$COMPOSE_FILE" "$MERGED_COMPOSE"

# Substitute __STACK_NAME__ placeholder with actual stack name
# This makes network names dynamic (e.g., __STACK_NAME___obiente-network → obiente_obiente-network)
sed -i "s/__STACK_NAME__/${STACK_NAME}/g" "$MERGED_COMPOSE"

# Note: We no longer pre-create the network as external
# External networks break DNS resolution in Docker Swarm
# Docker Swarm will create the network automatically when the stack deploys
# This ensures proper DNS configuration and service name resolution

# Deploy the main stack with environment variables loaded from .env
# Use --resolve-image always to force pulling latest images
docker stack deploy --resolve-image always -c "$MERGED_COMPOSE" "$STACK_NAME"
rm -f "$MERGED_COMPOSE"

echo ""
echo "✅ Main stack deployment started!"
echo ""

# Deploy dashboard in the same stack if enabled
if [ "$DEPLOY_DASHBOARD" = "true" ]; then
  echo "🚀 Adding dashboard service to main stack '$STACK_NAME'..."
  
  # Ensure DOMAIN is set for label substitution
  export DOMAIN="${DOMAIN:-obiente.cloud}"
  
  # Merge docker-compose.base.yml with docker-compose.dashboard.yml
  TEMP_DASHBOARD_COMPOSE=$(mktemp)
  ./scripts/merge-compose-files.sh docker-compose.dashboard.yml "$TEMP_DASHBOARD_COMPOSE"
  
  # Substitute __STACK_NAME__ placeholder and DOMAIN variables
  sed -i "s/__STACK_NAME__/${STACK_NAME}/g" "$TEMP_DASHBOARD_COMPOSE"
  sed -i "s/\${DOMAIN:-localhost}/${DOMAIN}/g" "$TEMP_DASHBOARD_COMPOSE"
  sed -i "s/\${DOMAIN}/${DOMAIN}/g" "$TEMP_DASHBOARD_COMPOSE"
  
  # Deploy dashboard service in the same stack (not a separate stack)
  docker stack deploy --resolve-image always -c "$TEMP_DASHBOARD_COMPOSE" "$STACK_NAME"
  rm -f "$TEMP_DASHBOARD_COMPOSE"
  
  echo "✅ Dashboard service added to stack '$STACK_NAME'!"
  echo ""
fi

echo "✅ All deployments started!"
echo ""
echo "📋 Useful commands:"
echo "  View services:     docker stack services $STACK_NAME"
echo "  View logs:         docker service logs -f ${STACK_NAME}_api-gateway"
if [ "$DEPLOY_DASHBOARD" = "true" ]; then
  echo "  Dashboard logs:    docker service logs -f ${STACK_NAME}_dashboard"
  echo "  Remove stack:      docker stack rm $STACK_NAME"
else
  echo "  Remove stack:      docker stack rm $STACK_NAME"
fi
echo "  List tasks:        docker stack ps $STACK_NAME"
echo ""
echo "📦 Microservices deployed:"
for service in "${MICROSERVICES[@]}"; do
  echo "   - ${service}"
done
echo ""
echo "⚠️  If you see mount errors on worker nodes, ensure directories exist:"
echo "   Run on each worker: ./scripts/setup-all-nodes.sh"
