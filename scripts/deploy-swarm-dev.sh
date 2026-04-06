#!/bin/bash
# Development deployment script for Obiente Cloud Docker Swarm
# Deploys the development stack using existing images by default
# Use -b or --build to build images locally
# Use -p or --pull to pull images from registry

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/lib/common.sh"

TMP_FILES=()
cleanup() {
  if [ ${#TMP_FILES[@]} -gt 0 ]; then
    rm -f "${TMP_FILES[@]}"
  fi
}
trap cleanup EXIT

# Parse command-line arguments
BUILD_IMAGES=false
PULL_IMAGES=false
STACK_NAME=""
COMPOSE_FILE=""

while [[ $# -gt 0 ]]; do
  case $1 in
    -b|--build)
      BUILD_IMAGES=true
      PULL_IMAGES=false
      shift
      ;;
    -p|--pull)
      PULL_IMAGES=true
      BUILD_IMAGES=false
      shift
      ;;
    *)
      if [ -z "$STACK_NAME" ]; then
        STACK_NAME="$1"
      elif [ -z "$COMPOSE_FILE" ]; then
        COMPOSE_FILE="$1"
      fi
      shift
      ;;
  esac
done

# Set defaults if not provided
STACK_NAME="${STACK_NAME:-obiente-dev}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.swarm.dev.yml}"

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
  "notifications-service"
  "orchestrator-service"
  "organizations-service"
  "superadmin-service"
  "support-service"
  "vps-gateway"
  "vps-service"
)

if [ -f "${REPO_ROOT}/.env" ]; then
  load_env_file "${REPO_ROOT}/.env"
elif [ -f "${REPO_ROOT}/.env.example" ]; then
  echo "⚠️  Warning: .env file not found. Using .env.example as reference."
  echo "   Copy .env.example to .env and configure it: cp .env.example .env"
fi

# Check if required directories exist on this node
REQUIRED_DIRS=(
  "/var/lib/obiente"
  "/var/obiente/tmp/obiente-volumes"
  "/var/obiente/tmp/obiente-deployments"
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
  echo "⚠️  IMPORTANT: Some services run on ALL nodes (mode: global)."
  echo "   You must create these directories on ALL worker nodes before deployment:"
  echo ""
  echo "   Run this on each worker node:"
  echo "   ./scripts/setup-all-nodes.sh"
  echo ""
  echo "   Or manually:"
  echo "   mkdir -p /var/lib/obiente"
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

if [ "$BUILD_IMAGES" = "true" ]; then
  echo "🔨 Building Obiente Cloud microservice images locally..."
  
  # Enable BuildKit for faster builds
  export DOCKER_BUILDKIT=1
  
  # Build all microservice images
  for service in "${MICROSERVICES[@]}"; do
    echo "📦 Building ${REGISTRY}/cloud-${service}:latest..."
    docker build -f apps/${service}/Dockerfile -t ${REGISTRY}/cloud-${service}:latest . || {
      echo "❌ Failed to build ${service}"
      exit 1
    }
  done
  
  echo "✅ Build complete!"
elif [ "$PULL_IMAGES" = "true" ]; then
  echo "📥 Pulling Obiente Cloud microservice images from GitHub Container Registry..."

  PULL_TARGETS=()
  for service in "${MICROSERVICES[@]}"; do
    PULL_TARGETS+=("${service}=${REGISTRY}/cloud-${service}:latest")
  done

  if ! pull_images_in_parallel "${PULL_TARGETS[@]}"; then
    echo ""
    echo "⚠️  Failed to pull the following images:"
    for service in "${PARALLEL_PULL_FAILURES[@]}"; do
      echo "   - ${service}"
    done
    echo ""
    echo "Continuing deployment with cached/local images."
    echo "Services that use these images may keep their existing version or fail"
    echo "to schedule if no local image is available."
    echo ""
    echo "Or use -b or --build to build locally"
  fi
  
  echo "✅ Image pull step complete!"
else
  echo "ℹ️  Skipping image build/pull (use -b/--build to build locally or -p/--pull to pull from registry)"
  echo "   Using existing images if available"
fi

echo ""
echo "🚀 Deploying development stack '$STACK_NAME'..."

# Note: We let Docker Swarm create the network automatically from the compose file
# Pre-creating it manually causes conflicts. Docker Swarm will create it before services.
NETWORK_NAME="${STACK_NAME}_obiente-network"
echo "ℹ️  Network '$NETWORK_NAME' will be created automatically by Docker Swarm"

# Merge docker-compose.base.yml with the compose file
# YAML anchors don't work across files, so we merge them first
MERGED_COMPOSE=$(mktemp)
TMP_FILES+=("$MERGED_COMPOSE")
./scripts/merge-compose-files.sh "$COMPOSE_FILE" "$MERGED_COMPOSE"

# Substitute __STACK_NAME__ placeholder with actual stack name
# This makes network names dynamic (e.g., __STACK_NAME___obiente-network → obiente-dev_obiente-network)
sed -i "s/__STACK_NAME__/${STACK_NAME}/g" "$MERGED_COMPOSE"

# Convert relative config file paths to absolute paths
# Docker configs resolve file: paths relative to current working directory
# We need absolute paths so they work regardless of where docker stack deploy is run
sed -i "s|file: \\./scripts/internal/|file: ${REPO_ROOT}/scripts/internal/|g" "$MERGED_COMPOSE"

# Convert relative bind mount paths to absolute paths
# Bind mounts with relative paths must exist on every node, so we convert them to absolute paths
sed -i "s|\\./monitoring/|${REPO_ROOT}/monitoring/|g" "$MERGED_COMPOSE"

# Verify config files exist before deploying
echo "🔍 Verifying Docker config files exist..."
CONFIG_FILES=(
  "${REPO_ROOT}/scripts/internal/pg_hba.conf"
  "${REPO_ROOT}/scripts/internal/init-pg-hba.sh"
)
MISSING_CONFIGS=()
for config_file in "${CONFIG_FILES[@]}"; do
  if [ ! -f "$config_file" ]; then
    MISSING_CONFIGS+=("$config_file")
  fi
done

if [ ${#MISSING_CONFIGS[@]} -gt 0 ]; then
  echo "❌ Error: Required config files missing:"
  for config_file in "${MISSING_CONFIGS[@]}"; do
    echo "   - $config_file"
  done
  echo ""
  echo "Please ensure all scripts are present before deploying."
  exit 1
fi
echo "✅ All config files found"

# Check if existing configs have changed by comparing content
# Docker configs can't be updated, so we need to remove and recreate if content changed
echo ""
echo "🔍 Checking if config content has changed..."

NEW_CONFIG_NAMES=(
  "${STACK_NAME}_postgres_hba_conf"
  "${STACK_NAME}_postgres_hba_init"
)

# Function to get source file for a config name
get_config_source_file() {
  local config_name=$1
  case "$config_name" in
    "${STACK_NAME}_postgres_hba_conf")
      echo "${REPO_ROOT}/scripts/internal/pg_hba.conf"
      ;;
    "${STACK_NAME}_postgres_hba_init")
      echo "${REPO_ROOT}/scripts/internal/init-pg-hba.sh"
      ;;
    *)
      echo ""
      ;;
  esac
}

for config_name in "${NEW_CONFIG_NAMES[@]}"; do
  if docker config ls --format "{{.Name}}" | grep -q "^${config_name}$"; then
    SOURCE_FILE=$(get_config_source_file "$config_name")
    if [ -n "$SOURCE_FILE" ] && [ -f "$SOURCE_FILE" ]; then
      # Get existing config content
      # Docker configs store data as base64-encoded JSON
      EXISTING_CONTENT=$(docker config inspect "$config_name" --format '{{json .Spec.Data}}' 2>/dev/null | sed 's/^"//;s/"$//' | base64 -d 2>/dev/null || echo "")
      NEW_CONTENT=$(cat "$SOURCE_FILE")
      
      # Normalize content for comparison (remove trailing newlines)
      EXISTING_CONTENT=$(echo -n "$EXISTING_CONTENT")
      NEW_CONTENT=$(echo -n "$NEW_CONTENT")
      
      # Compare content (using a simple hash comparison)
      EXISTING_HASH=$(echo -n "$EXISTING_CONTENT" | sha256sum | cut -d' ' -f1)
      NEW_HASH=$(echo -n "$NEW_CONTENT" | sha256sum | cut -d' ' -f1)
      
      if [ "$EXISTING_HASH" != "$NEW_HASH" ]; then
        echo "   ⚠️  Config $config_name content has changed, will be recreated"
        
        # Remove old config (services will need to be updated first)
        echo "   🗑️  Removing old config: $config_name"
        # Find services using this config
        SERVICES_USING_CONFIG=()
        for service in $(docker service ls --format "{{.Name}}" | grep "^${STACK_NAME}_"); do
          SERVICE_CONFIGS=$(docker service inspect "$service" --format '{{range .Spec.TaskTemplate.ContainerSpec.Configs}}{{.ConfigName}} {{end}}' 2>/dev/null || echo "")
          if echo "$SERVICE_CONFIGS" | grep -q "$config_name"; then
            SERVICES_USING_CONFIG+=("$service")
          fi
        done
        
        # Remove config from services first
        for service in "${SERVICES_USING_CONFIG[@]}"; do
          echo "      Removing config from $service..."
          docker service update --config-rm "$config_name" "$service" 2>/dev/null || true
        done
        
        # Wait for services to update
        if [ ${#SERVICES_USING_CONFIG[@]} -gt 0 ]; then
          echo "   ⏳ Waiting for services to update..."
          sleep 3
        fi
        
        # Remove the config
        docker config rm "$config_name" 2>/dev/null || echo "      ⚠️  Could not remove $config_name (may still be in use)"
      else
        echo "   ✅ Config $config_name unchanged, skipping update"
      fi
    else
      echo "   ℹ️  Config $config_name exists (source file not found, skipping content check)"
    fi
  else
    echo "   ℹ️  Config $config_name does not exist, will be created"
  fi
done

echo "✅ Config cleanup complete"

# Deploy the main stack with environment variables loaded from .env
# Use --resolve-image never since we've already built/pulled images
docker stack deploy --resolve-image never -c "$MERGED_COMPOSE" "$STACK_NAME"
rm -f "$MERGED_COMPOSE"

echo ""
echo "✅ Development stack deployment started!"
echo ""
echo "📋 Useful commands:"
echo "  View services:     docker stack services $STACK_NAME"
echo "  View logs:         docker service logs -f ${STACK_NAME}_api-gateway"
echo "  Remove stack:      docker stack rm $STACK_NAME"
echo "  List tasks:        docker stack ps $STACK_NAME"
echo ""
echo "📦 Microservices deployed:"
for service in "${MICROSERVICES[@]}"; do
  echo "   - ${service}"
done
echo ""
echo "⚠️  If you see mount errors on worker nodes, ensure directories exist:"
echo "   Run on each worker: ./scripts/setup-all-nodes.sh"
