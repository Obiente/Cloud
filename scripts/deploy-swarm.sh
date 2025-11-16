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

# Check if required overlay network exists (must be created manually on manager node)
NETWORK_NAME="${STACK_NAME}_obiente-network"
if ! docker network inspect "$NETWORK_NAME" &>/dev/null; then
  echo "❌ Error: Required overlay network '$NETWORK_NAME' does not exist!"
  echo ""
  echo "📋 Create the network first on a Swarm manager node:"
  echo ""
  echo "  Option 1: Use the script (recommended):"
  echo "    ./scripts/create-swarm-network.sh --subnet 10.0.9.0/24"
  echo ""
  echo "  Option 2: Manual creation:"
  echo "    docker network create --driver overlay --subnet 10.0.9.0/24 $NETWORK_NAME"
  echo ""
  echo "💡 Note: Overlay networks can only be created on Swarm manager nodes"
  echo ""
  exit 1
fi

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

# Deploy the main stack with environment variables loaded from .env
# Use --resolve-image always to force pulling latest images
docker stack deploy --resolve-image always -c "$MERGED_COMPOSE" "$STACK_NAME"
rm -f "$MERGED_COMPOSE"

echo ""
echo "✅ Main stack deployment started!"

# Deploy dashboard stack if enabled
if [ "$DEPLOY_DASHBOARD" = "true" ]; then
  echo ""
  echo "🚀 Deploying dashboard stack..."
  
  # Wait a moment for the network to be created
  sleep 2
  
  # Ensure the network exists (Docker Swarm creates it with stack prefix)
  NETWORK_NAME="${STACK_NAME}_obiente-network"
  if ! docker network ls | grep -q "$NETWORK_NAME"; then
    echo "⚠️  Warning: Network $NETWORK_NAME not found. Waiting for main stack to create it..."
    sleep 3
  fi
  
  # Deploy dashboard stack (uses external network from main stack)
  # The network name in docker-compose.dashboard.yml references: ${STACK_NAME}_obiente-network
  # Substitute DOMAIN variable in labels (Docker Swarm doesn't expand env vars in labels)
  export DOMAIN="${DOMAIN:-obiente.cloud}"
  TEMP_DASHBOARD_COMPOSE=$(mktemp)
  sed "s/\${DOMAIN:-localhost}/${DOMAIN}/g; s/\${DOMAIN}/${DOMAIN}/g" docker-compose.dashboard.yml > "$TEMP_DASHBOARD_COMPOSE"
  # Use --resolve-image always to force pulling latest images
  docker stack deploy --resolve-image always -c "$TEMP_DASHBOARD_COMPOSE" "${STACK_NAME}"
  rm -f "$TEMP_DASHBOARD_COMPOSE"
  
  echo ""
  echo "✅ Dashboard stack deployment started!"
fi

echo ""
echo "✅ All deployments started!"
echo ""
echo "📋 Useful commands:"
echo "  View services:     docker stack services $STACK_NAME"
echo "  View logs:         docker service logs -f ${STACK_NAME}_api-gateway"
if [ "$DEPLOY_DASHBOARD" = "true" ]; then
  echo "  Dashboard logs:    docker service logs -f ${STACK_NAME}_dashboard"
fi
echo "  Remove stacks:     docker stack rm $STACK_NAME${DEPLOY_DASHBOARD:+ ${STACK_NAME}_dashboard}"
echo "  List tasks:        docker stack ps $STACK_NAME"
echo ""
echo "📦 Microservices deployed:"
for service in "${MICROSERVICES[@]}"; do
  echo "   - ${service}"
done
echo ""
echo "⚠️  If you see mount errors on worker nodes, ensure directories exist:"
echo "   Run on each worker: ./scripts/setup-all-nodes.sh"
