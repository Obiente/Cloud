#!/bin/bash
# Build script for Obiente Cloud Docker Swarm deployment
# This script builds the service images used by the Swarm stack.

set -euo pipefail

echo "🔨 Building Obiente Cloud images..."

# Enable BuildKit for faster builds
export DOCKER_BUILDKIT=1

MICROSERVICES=(
  "api-gateway"
  "audit-service"
  "auth-service"
  "billing-service"
  "deployments-service"
  "dns-service"
  "gameservers-service"
  "notifications-service"
  "orchestrator-service"
  "organizations-service"
  "superadmin-service"
  "support-service"
  "vps-gateway"
  "vps-service"
)

for service in "${MICROSERVICES[@]}"; do
  echo "📦 Building obiente/cloud-${service}:latest..."
  docker build -f "apps/${service}/Dockerfile" -t "obiente/cloud-${service}:latest" .
done

echo "📦 Building obiente/cloud-dashboard:latest..."
docker build -f apps/dashboard/Dockerfile -t obiente/cloud-dashboard:latest .

echo "✅ Build complete!"
echo ""
echo "📋 Next steps:"
echo "  1. Push to GitHub Container Registry (recommended for production):"
echo "     docker login ghcr.io"
echo "     docker tag obiente/cloud-api-gateway:latest ghcr.io/obiente/cloud-api-gateway:latest"
echo "     docker push ghcr.io/obiente/cloud-api-gateway:latest"
echo "     # Repeat for any additional images you want to publish"
echo ""
echo "  2. Or push selected images to another registry:"
echo "     docker tag obiente/cloud-api-gateway:latest your-registry/obiente/cloud-api-gateway:latest"
echo "     docker push your-registry/obiente/cloud-api-gateway:latest"
echo ""
echo "  3. Or use docker save/load to transfer images to worker nodes:"
echo "     docker save obiente/cloud-api-gateway:latest | gzip > cloud-api-gateway.tar.gz"
echo "     # Transfer to worker node, then:"
echo "     docker load < cloud-api-gateway.tar.gz"
echo ""
echo "  4. Deploy the stack (recommended - uses deploy script):"
echo "     ./scripts/deploy-swarm.sh obiente docker-compose.swarm.yml"
echo ""
echo "     Or build during deploy:"
echo "     BUILD_LOCAL=true ./scripts/deploy-swarm.sh obiente docker-compose.swarm.yml"
