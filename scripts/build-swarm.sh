#!/bin/bash
# Build script for Obiente Cloud Docker Swarm deployment
# This script builds the service images used by the Swarm stack.

set -euo pipefail

echo "🔨 Building Obiente Cloud images..."

# Enable BuildKit for faster builds
export DOCKER_BUILDKIT=1
REGISTRY="${REGISTRY:-ghcr.io/obiente}"

MICROSERVICES=(
  "api-gateway"
  "audit-service"
  "auth-service"
  "billing-service"
  "databases-service"
  "deployments-service"
  "dns-service"
  "file-transfer-service"
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
  echo "📦 Building ${REGISTRY}/cloud-${service}:latest..."
  docker build -f "apps/${service}/Dockerfile" -t "${REGISTRY}/cloud-${service}:latest" .
done

echo "📦 Building ${DASHBOARD_IMAGE:-${REGISTRY}/cloud-dashboard:latest}..."
docker build -f apps/dashboard/Dockerfile -t "${DASHBOARD_IMAGE:-${REGISTRY}/cloud-dashboard:latest}" .

echo "✅ Build complete!"
echo ""
echo "Images are tagged for ${REGISTRY}. Make them available on all Swarm nodes before deployment."
echo "Deploy with: ./scripts/deploy-swarm.sh obiente docker-compose.swarm.yml"
