#!/bin/bash
# Quick deploy script for Obiente Cloud Docker Swarm
# Deploys both the main stack and dashboard stack
# Set BUILD_LOCAL=true to build images locally instead
# Set DEPLOY_DASHBOARD=false to skip dashboard deployment

set -e

STACK_NAME="${1:-obiente}"
COMPOSE_FILE="${2:-docker-compose.swarm.yml}"
BUILD_LOCAL="${BUILD_LOCAL:-false}"
API_IMAGE="${API_IMAGE:-ghcr.io/obiente/cloud-api:latest}"
DASHBOARD_IMAGE="${DASHBOARD_IMAGE:-ghcr.io/obiente/cloud-dashboard:latest}"
DEPLOY_DASHBOARD="${DEPLOY_DASHBOARD:-true}"

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
  "/tmp/obiente-volumes"
  "/tmp/obiente-deployments"
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
  echo "✅ Directories created!"
  echo ""
  echo "⚠️  IMPORTANT: The API service runs on ALL nodes (mode: global)."
  echo "   You must create these directories on ALL worker nodes before deployment:"
  echo ""
  echo "   Run this on each worker node:"
  echo "   ./scripts/setup-all-nodes.sh"
  echo ""
  echo "   Or manually:"
  echo "   mkdir -p /var/lib/obiente/volumes"
  echo "   mkdir -p /tmp/obiente-volumes"
  echo "   mkdir -p /tmp/obiente-deployments"
  echo ""
  read -p "Continue with deployment? (y/N) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Deployment cancelled. Please create directories on all nodes first."
    exit 1
  fi
fi

if [ "$BUILD_LOCAL" = "true" ]; then
  echo "🔨 Building Obiente Cloud images locally..."
  
  # Enable BuildKit for faster builds
  export DOCKER_BUILDKIT=1
  
  # Build the API image (used by both api and dns services)
  echo "📦 Building obiente/cloud-api:latest..."
  docker build -f apps/api/Dockerfile -t obiente/cloud-api:latest .
  
  # Build the Dashboard image
  echo "📦 Building obiente/cloud-dashboard:latest..."
  docker build -f apps/dashboard/Dockerfile -t obiente/cloud-dashboard:latest .
  
  # Use local image names
  export API_IMAGE="obiente/cloud-api:latest"
  export DASHBOARD_IMAGE="obiente/cloud-dashboard:latest"
  
  echo "✅ Build complete!"
else
  echo "📥 Pulling Obiente Cloud images from GitHub Container Registry..."
  
  # Pull the API image from ghcr.io
  echo "📦 Pulling $API_IMAGE..."
  docker pull "$API_IMAGE" || {
    echo "⚠️  Warning: Failed to pull image. Make sure you're authenticated to ghcr.io:"
    echo "   docker login ghcr.io"
    echo "   Or set BUILD_LOCAL=true to build locally"
    exit 1
  }
  
  # Pull the Dashboard image if deploying dashboard
  if [ "$DEPLOY_DASHBOARD" = "true" ]; then
    echo "📦 Pulling $DASHBOARD_IMAGE..."
    docker pull "$DASHBOARD_IMAGE" || {
      echo "⚠️  Warning: Failed to pull dashboard image. Make sure you're authenticated to ghcr.io:"
      echo "   docker login ghcr.io"
      echo "   Or set BUILD_LOCAL=true to build locally"
      exit 1
    }
  fi
  
  echo "✅ Image pull complete!"
fi

echo ""
echo "🚀 Deploying main stack '$STACK_NAME'..."

# Deploy the main stack with environment variables loaded from .env
# Use --resolve-image always to force pulling latest images
docker stack deploy --resolve-image always -c "$COMPOSE_FILE" "$STACK_NAME"

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
  STACK_NAME="$STACK_NAME" DASHBOARD_IMAGE="$DASHBOARD_IMAGE" docker stack deploy --resolve-image always -c "$TEMP_DASHBOARD_COMPOSE" "${STACK_NAME}"
  rm -f "$TEMP_DASHBOARD_COMPOSE"
  
  echo ""
  echo "✅ Dashboard stack deployment started!"
fi

echo ""
echo "✅ All deployments started!"
echo ""
echo "📋 Useful commands:"
echo "  View services:     docker stack services $STACK_NAME"
echo "  View logs:         docker service logs -f ${STACK_NAME}_api"
if [ "$DEPLOY_DASHBOARD" = "true" ]; then
  echo "  Dashboard logs:    docker service logs -f ${STACK_NAME}_dashboard"
fi
echo "  Remove stacks:     docker stack rm $STACK_NAME${DEPLOY_DASHBOARD:+ ${STACK_NAME}_dashboard}"
echo "  List tasks:        docker stack ps $STACK_NAME"
echo ""
echo "⚠️  If you see mount errors on worker nodes, ensure directories exist:"
echo "   Run on each worker: ./scripts/setup-all-nodes.sh"
