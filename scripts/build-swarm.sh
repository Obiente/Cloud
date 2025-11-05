#!/bin/bash
# Build script for Obiente Cloud Docker Swarm deployment
# This script builds the required images before deploying the stack
# For production, use images from GitHub Container Registry instead:
#   ghcr.io/obiente/cloud-api:latest
#   ghcr.io/obiente/cloud-dashboard:latest

set -e

echo "🔨 Building Obiente Cloud images..."

# Enable BuildKit for faster builds
export DOCKER_BUILDKIT=1

# Build the API image (used by both api and dns services)
echo "📦 Building obiente/cloud-api:latest..."
docker build -f apps/api/Dockerfile -t obiente/cloud-api:latest .

echo "✅ Build complete!"
echo ""
echo "📋 Next steps:"
echo "  1. Push to GitHub Container Registry (recommended for production):"
echo "     docker tag obiente/cloud-api:latest ghcr.io/obiente/cloud-api:latest"
echo "     docker login ghcr.io"
echo "     docker push ghcr.io/obiente/cloud-api:latest"
echo ""
echo "     # Dashboard image"
echo "     docker tag obiente/cloud-dashboard:latest ghcr.io/obiente/cloud-dashboard:latest"
echo "     docker push ghcr.io/obiente/cloud-dashboard:latest"
echo ""
echo "  2. Or push to another registry:"
echo "     docker tag obiente/cloud-api:latest your-registry/obiente/cloud-api:latest"
echo "     docker push your-registry/obiente/cloud-api:latest"
echo ""
echo "  3. Or use docker save/load to transfer images to worker nodes:"
echo "     docker save obiente/cloud-api:latest | gzip > cloud-api.tar.gz"
echo "     # Transfer to worker node, then:"
echo "     docker load < cloud-api.tar.gz"
echo ""
echo "  4. Deploy the stack:"
echo "     docker stack deploy -c docker-compose.swarm.yml obiente"

