#!/bin/bash
# Quick deploy script for Obiente Cloud Docker Swarm
# Deploys both the main stack and dashboard stack
# Set BUILD_LOCAL=true to build images locally instead
# Set DEPLOY_DASHBOARD=false to skip dashboard deployment
# Use -p or --pull to pull images before deploying

set -e

# Parse command-line arguments
PULL_IMAGES=false
STACK_NAME=""
COMPOSE_FILE=""

while [[ $# -gt 0 ]]; do
  case $1 in
    -p|--pull)
      PULL_IMAGES=true
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
STACK_NAME="${STACK_NAME:-obiente}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.swarm.yml}"
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
elif [ "$PULL_IMAGES" = "true" ]; then
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
else
  echo "ℹ️  Skipping image pull (use -p or --pull to pull images)"
fi

echo ""
echo "🚀 Deploying main stack '$STACK_NAME'..."

# Note: We let Docker Swarm create the network automatically from the compose file
# Pre-creating it manually causes conflicts. Docker Swarm will create it before services.
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

# Convert relative config file paths to absolute paths
# Docker configs resolve file: paths relative to current working directory
# We need absolute paths so they work regardless of where docker stack deploy is run
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
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
  rm -f "$MERGED_COMPOSE"
  exit 1
fi
echo "✅ All config files found"

# Remove old Docker configs if they exist (Docker configs can't be updated, only labels)
# We need to update services first to remove config references, then remove the configs
echo ""
echo "🔧 Checking for existing Docker configs that need updating..."

OLD_CONFIG_NAMES=(
  "${STACK_NAME}_postgres_init_hba"
  "${STACK_NAME}_postgres_entrypoint_wrapper"
  "${STACK_NAME}_postgres_init_user"
)

NEW_CONFIG_NAMES=(
  "${STACK_NAME}_postgres_hba_conf"
  "${STACK_NAME}_postgres_hba_init"
)

# Check if we need to migrate from old config names to new ones
# Only migrate if old configs exist AND services are actually using them
NEEDS_MIGRATION=false
SERVICES_USING_OLD_CONFIGS=()

for old_config in "${OLD_CONFIG_NAMES[@]}"; do
  if docker config ls --format "{{.Name}}" | grep -q "^${old_config}$"; then
    # Check if any services are actually using this old config
    for service in $(docker service ls --format "{{.Name}}" | grep "^${STACK_NAME}_"); do
      SERVICE_CONFIGS=$(docker service inspect "$service" --format '{{range .Spec.TaskTemplate.ContainerSpec.Configs}}{{.ConfigName}} {{end}}' 2>/dev/null || echo "")
      if echo "$SERVICE_CONFIGS" | grep -q "$old_config"; then
        NEEDS_MIGRATION=true
        if [[ ! " ${SERVICES_USING_OLD_CONFIGS[@]} " =~ " ${service} " ]]; then
          SERVICES_USING_OLD_CONFIGS+=("$service")
        fi
      fi
    done
  fi
done

if [ "$NEEDS_MIGRATION" = true ] && [ ${#SERVICES_USING_OLD_CONFIGS[@]} -gt 0 ]; then
  echo "   ⚠️  Found old config names that need migration"
  echo "   📋 Services using old configs: ${SERVICES_USING_OLD_CONFIGS[*]}"
  echo "   💡 Removing old configs from services..."
  
  for service in "${SERVICES_USING_OLD_CONFIGS[@]}"; do
    echo "      Updating $service..."
    # Remove only the old configs that this service is actually using
    for old_config in "${OLD_CONFIG_NAMES[@]}"; do
      SERVICE_CONFIGS=$(docker service inspect "$service" --format '{{range .Spec.TaskTemplate.ContainerSpec.Configs}}{{.ConfigName}} {{end}}' 2>/dev/null || echo "")
      if echo "$SERVICE_CONFIGS" | grep -q "$old_config"; then
        docker service update --config-rm "$old_config" "$service" 2>/dev/null || true
      fi
    done
  done
  
  echo "   ⏳ Waiting for service updates to complete..."
  sleep 5
  
  # Now try to remove old configs (only if not in use)
  echo "   🗑️  Removing old configs..."
  for old_config in "${OLD_CONFIG_NAMES[@]}"; do
    if docker config ls --format "{{.Name}}" | grep -q "^${old_config}$"; then
      if docker config rm "$old_config" 2>/dev/null; then
        echo "      ✅ Removed: $old_config"
      else
        echo "      ⚠️  Could not remove $old_config (may still be in use)"
      fi
    fi
  done
else
  echo "   ✅ No old configs found or no services using them"
fi

# Check if existing configs have changed by comparing content
# Docker configs can't be updated, so we need to remove and recreate if content changed
echo ""
echo "🔍 Checking if config content has changed..."

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
        CONFIGS_NEED_UPDATE=true
        
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

# Note: We no longer pre-create the network as external
# External networks break DNS resolution in Docker Swarm
# Docker Swarm will create the network automatically when the stack deploys
# This ensures proper DNS configuration and service name resolution

# Deploy the main stack with environment variables loaded from .env
# Use --resolve-image always to force pulling latest images if PULL_IMAGES is true
if [ "$PULL_IMAGES" = "true" ]; then
  docker stack deploy --resolve-image always -c "$MERGED_COMPOSE" "$STACK_NAME"
else
  docker stack deploy --resolve-image never -c "$MERGED_COMPOSE" "$STACK_NAME"
fi
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
  
  # Process environment variables in the dashboard compose file
  # This is critical for Swarm deployments to properly substitute ${VAR} placeholders
  # We need to use envsubst to replace variables before docker stack deploy
  TEMP_DASHBOARD_COMPOSE_FINAL=$(mktemp)
  envsubst < "$TEMP_DASHBOARD_COMPOSE" > "$TEMP_DASHBOARD_COMPOSE_FINAL"
  
  # Deploy dashboard service in the same stack (not a separate stack)
  if [ "$PULL_IMAGES" = "true" ]; then
    docker stack deploy --resolve-image always -c "$TEMP_DASHBOARD_COMPOSE_FINAL" "$STACK_NAME"
  else
    docker stack deploy --resolve-image never -c "$TEMP_DASHBOARD_COMPOSE_FINAL" "$STACK_NAME"
  fi
  rm -f "$TEMP_DASHBOARD_COMPOSE" "$TEMP_DASHBOARD_COMPOSE_FINAL"
  
  echo "✅ Dashboard service added to stack '$STACK_NAME'!"
  echo ""
  
  # Wait for dashboard to be ready (Traefik needs to discover it before it can serve requests)
  echo "⏳ Waiting for dashboard service to be ready for Traefik routing..."
  MAX_WAIT=120
  ELAPSED=0
  DASHBOARD_READY=false
  
  while [ $ELAPSED -lt $MAX_WAIT ]; do
    # Check if dashboard service has running tasks
    RUNNING_TASKS=$(docker service ps "${STACK_NAME}_dashboard" --filter "desired-state=running" --format "{{.CurrentState}}" 2>/dev/null | grep -c "Running" || echo "0")
    
    if [ "$RUNNING_TASKS" -gt 0 ]; then
      # Check if Traefik has discovered the dashboard service (check Traefik logs)
      TRAEFIK_DISCOVERED=$(docker service logs "${STACK_NAME}_traefik" 2>/dev/null | grep -i "dashboard" | grep -i "discovered\|routing\|loadbalancer" | tail -1)
      
      if [ -n "$TRAEFIK_DISCOVERED" ]; then
        DASHBOARD_READY=true
        echo "✅ Dashboard is ready and discovered by Traefik!"
        break
      fi
      
      echo "   ✓ Dashboard tasks running, waiting for Traefik discovery..."
    else
      echo "   ⏳ Waiting for dashboard tasks to start (${ELAPSED}s)..."
    fi
    
    sleep 5
    ELAPSED=$((ELAPSED + 5))
  done
  
  if [ "$DASHBOARD_READY" = false ]; then
    echo "⚠️  Warning: Dashboard may not be fully ready yet (timeout after ${MAX_WAIT}s)"
    echo "   This is normal - service discovery can take time on large clusters"
    echo "   Monitor progress with: docker service logs -f ${STACK_NAME}_dashboard"
    echo "   And: docker service logs -f ${STACK_NAME}_traefik | grep -i dashboard"
  fi
else
  echo "✅ Main stack deployment started!"
  echo ""
fi

echo "✅ All deployments started!"
echo ""
echo "📋 Useful commands:"
echo "  View services:     docker stack services $STACK_NAME"
echo "  View logs:         docker service logs -f ${STACK_NAME}_api-gateway"
if [ "$DEPLOY_DASHBOARD" = "true" ]; then
  echo "  Dashboard logs:    docker service logs -f ${STACK_NAME}_dashboard"
  echo "  Traefik logs:       docker service logs -f ${STACK_NAME}_traefik | grep -i dashboard"
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
