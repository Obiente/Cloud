#!/bin/bash
# Quick deploy script for Obiente Cloud Docker Swarm
# Pulls images from GitHub Container Registry and deploys the stack
# Set BUILD_LOCAL=true to build images locally instead

set -e

STACK_NAME="${1:-obiente}"
COMPOSE_FILE="${2:-docker-compose.swarm.yml}"
BUILD_LOCAL="${BUILD_LOCAL:-false}"
API_IMAGE="${API_IMAGE:-ghcr.io/obiente/cloud-api:latest}"

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

if [ "$BUILD_LOCAL" = "true" ]; then
  echo "🔨 Building Obiente Cloud images locally..."
  
  # Enable BuildKit for faster builds
  export DOCKER_BUILDKIT=1
  
  # Build the API image (used by both api and dns services)
  echo "📦 Building obiente/cloud-api:latest..."
  docker build -f apps/api/Dockerfile -t obiente/cloud-api:latest .
  
  # Use local image name
  export API_IMAGE="obiente/cloud-api:latest"
  
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
  
  echo "✅ Image pull complete!"
fi

echo ""
echo "🚀 Deploying stack '$STACK_NAME'..."

# Deploy the stack with environment variables loaded from .env
docker stack deploy -c "$COMPOSE_FILE" "$STACK_NAME"

echo ""
echo "✅ Deployment started!"
echo ""
echo "📋 Useful commands:"
echo "  View services:     docker stack services $STACK_NAME"
echo "  View logs:         docker service logs -f ${STACK_NAME}_api"
echo "  Remove stack:      docker stack rm $STACK_NAME"
echo "  List tasks:        docker stack ps $STACK_NAME"

