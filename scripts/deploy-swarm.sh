#!/bin/bash
# Quick deploy script for Obiente Cloud Docker Swarm
# Builds images and deploys the stack

set -e

STACK_NAME="${1:-obiente}"
COMPOSE_FILE="${2:-docker-compose.swarm.yml}"

echo "🔨 Building Obiente Cloud images..."

# Enable BuildKit for faster builds
export DOCKER_BUILDKIT=1

# Build the API image (used by both api and dns services)
echo "📦 Building obiente/cloud-api:latest..."
docker build -f apps/api/Dockerfile -t obiente/cloud-api:latest .

echo "✅ Build complete!"
echo ""
echo "🚀 Deploying stack '$STACK_NAME'..."

# Deploy the stack
docker stack deploy -c "$COMPOSE_FILE" "$STACK_NAME"

echo ""
echo "✅ Deployment started!"
echo ""
echo "📋 Useful commands:"
echo "  View services:     docker stack services $STACK_NAME"
echo "  View logs:         docker service logs -f ${STACK_NAME}_api"
echo "  Remove stack:      docker stack rm $STACK_NAME"
echo "  List tasks:        docker stack ps $STACK_NAME"

