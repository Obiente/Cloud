#!/bin/bash
# Force deploy script for Obiente Cloud Docker Swarm
# Force updates all services to ensure they're running latest configuration
# Usage: ./scripts/force-deploy.sh [stack-name] [compose-file]

set -e

STACK_NAME="${1:-obiente}"
COMPOSE_FILE="${2:-docker-compose.swarm.yml}"
DEPLOY_DASHBOARD="${DEPLOY_DASHBOARD:-true}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Load .env file if it exists
if [ -f .env ]; then
  echo -e "${BLUE}📝 Loading environment variables from .env file...${NC}"
  set -a
  source .env
  set +a
fi

echo -e "${BLUE}🔄 Force deploying Obiente Cloud stack: ${STACK_NAME}${NC}"
echo ""

# Function to force update a service
force_update_service() {
  local service_name=$1
  if docker service ls --format "{{.Name}}" | grep -q "^${service_name}$"; then
    echo -e "${YELLOW}  ⚡ Force updating service: ${service_name}${NC}"
    docker service update --force "${service_name}" || {
      echo -e "${RED}  ❌ Failed to update ${service_name}${NC}"
      return 1
    }
    echo -e "${GREEN}  ✅ ${service_name} update initiated${NC}"
  else
    echo -e "${YELLOW}  ⚠️  Service ${service_name} not found, skipping...${NC}"
  fi
}

# Function to get all services in a stack
get_stack_services() {
  local stack=$1
  docker stack services --format "{{.Name}}" "$stack" 2>/dev/null || echo ""
}

# First, redeploy the stacks to ensure config is up to date
echo -e "${BLUE}📦 Step 1: Redeploying stacks with latest configuration...${NC}"
echo ""

# Redeploy main stack
echo -e "${BLUE}🚀 Redeploying main stack '${STACK_NAME}'...${NC}"
docker stack deploy --resolve-image always -c "$COMPOSE_FILE" "$STACK_NAME"
echo -e "${GREEN}✅ Main stack redeployed!${NC}"
echo ""

# Redeploy dashboard stack if enabled
if [ "$DEPLOY_DASHBOARD" = "true" ]; then
  echo -e "${BLUE}🚀 Redeploying dashboard stack...${NC}"
  # Ensure DOMAIN is set for label substitution
  export DOMAIN="${DOMAIN:-obiente.cloud}"
  # Substitute DOMAIN variable in labels (Docker Swarm doesn't expand env vars in labels)
  TEMP_DASHBOARD_COMPOSE=$(mktemp)
  sed "s/\${DOMAIN:-localhost}/${DOMAIN}/g; s/\${DOMAIN}/${DOMAIN}/g" docker-compose.dashboard.yml > "$TEMP_DASHBOARD_COMPOSE"
  STACK_NAME="$STACK_NAME" docker stack deploy --resolve-image always -c "$TEMP_DASHBOARD_COMPOSE" "${STACK_NAME}"
  rm -f "$TEMP_DASHBOARD_COMPOSE"
  echo -e "${GREEN}✅ Dashboard stack redeployed!${NC}"
  echo ""
fi

# Wait a moment for services to stabilize
echo -e "${BLUE}⏳ Waiting for services to stabilize...${NC}"
sleep 5

# Force update all services in main stack
echo ""
echo -e "${BLUE}📦 Step 2: Force updating all services in main stack...${NC}"
echo ""

MAIN_SERVICES=$(get_stack_services "$STACK_NAME")
if [ -z "$MAIN_SERVICES" ]; then
  echo -e "${YELLOW}⚠️  No services found in stack ${STACK_NAME}${NC}"
else
  while IFS= read -r service; do
    if [ -n "$service" ]; then
      force_update_service "$service"
    fi
  done <<< "$MAIN_SERVICES"
fi

# Force update dashboard services if enabled
if [ "$DEPLOY_DASHBOARD" = "true" ]; then
  echo ""
  echo -e "${BLUE}📦 Step 3: Force updating dashboard services...${NC}"
  echo ""
  
  DASHBOARD_SERVICES=$(get_stack_services "${STACK_NAME}_dashboard")
  if [ -z "$DASHBOARD_SERVICES" ]; then
    echo -e "${YELLOW}⚠️  No services found in dashboard stack${NC}"
  else
    while IFS= read -r service; do
      if [ -n "$service" ]; then
        force_update_service "$service"
      fi
    done <<< "$DASHBOARD_SERVICES"
  fi
fi

echo ""
echo -e "${GREEN}✅ Force deployment complete!${NC}"
echo ""

# Show service status
echo -e "${BLUE}📊 Current service status:${NC}"
echo ""
echo -e "${BLUE}Main stack services:${NC}"
docker stack services "$STACK_NAME" --format "table {{.Name}}\t{{.Replicas}}\t{{.Image}}"

if [ "$DEPLOY_DASHBOARD" = "true" ]; then
  echo ""
  echo -e "${BLUE}Dashboard stack services:${NC}"
  docker stack services "${STACK_NAME}_dashboard" --format "table {{.Name}}\t{{.Replicas}}\t{{.Image}}"
fi

echo ""
echo -e "${BLUE}📋 Useful commands:${NC}"
echo "  View all services:  docker stack services $STACK_NAME"
echo "  View service logs:  docker service logs -f ${STACK_NAME}_api"
if [ "$DEPLOY_DASHBOARD" = "true" ]; then
  echo "  Dashboard logs:     docker service logs -f ${STACK_NAME}_dashboard"
fi
echo "  Service status:     docker service ps ${STACK_NAME}_api"
echo "  Remove stacks:      docker stack rm $STACK_NAME${DEPLOY_DASHBOARD:+ ${STACK_NAME}_dashboard}"
echo ""
echo -e "${YELLOW}💡 Note: Services are being updated. Check status with: docker service ps <service-name>${NC}"
echo ""

