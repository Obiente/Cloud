#!/bin/bash
# Complete cleanup script for Obiente Cloud Docker Swarm deployment
# Removes all Obiente Cloud stacks, services, containers, volumes, and networks
# SAFE: Only targets Obiente Cloud resources - other Docker resources are NOT touched
# WARNING: This will delete all Obiente Cloud data including volumes!
# Usage: ./scripts/cleanup-swarm-complete.sh [--confirm]

set -e

STACK_NAME="${STACK_NAME:-obiente}"
CONFIRM="${1:-}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${RED}⚠️  WARNING: This will completely remove all Obiente Cloud resources!${NC}"
echo ""
echo -e "${YELLOW}This includes:${NC}"
echo "  - Obiente Cloud stacks (${STACK_NAME} and ${STACK_NAME}_dashboard)"
echo "  - Obiente Cloud services"
echo "  - Obiente Cloud containers and tasks"
echo "  - Obiente Cloud volumes (DATA WILL BE LOST)"
echo "  - Obiente Cloud networks"
echo "  - Obiente Cloud related images"
echo ""
echo -e "${GREEN}✅ This script ONLY removes Obiente Cloud resources${NC}"
echo -e "${GREEN}   Other Docker resources are safe and will NOT be touched${NC}"
echo ""

# Check if we're on a manager node
if ! docker node ls &>/dev/null; then
  echo -e "${RED}❌ Error: This script must be run on a Docker Swarm manager node${NC}"
  exit 1
fi

# Require confirmation
if [ "$CONFIRM" != "--confirm" ]; then
  echo -e "${YELLOW}To proceed, run:${NC}"
  echo "  ./scripts/cleanup-swarm-complete.sh --confirm"
  echo ""
  read -p "Are you sure you want to continue? (type 'yes' to confirm): " -r
  echo
  if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    echo -e "${BLUE}Cleanup cancelled.${NC}"
    exit 0
  fi
fi

echo -e "${BLUE}🧹 Starting complete cleanup...${NC}"
echo ""

# Function to safely remove resources
safe_remove() {
  local resource_type=$1
  local resource_name=$2
  local description=$3
  
  if docker $resource_type ls --format "{{.Name}}" 2>/dev/null | grep -q "^${resource_name}$"; then
    echo -e "${BLUE}  Removing ${description}: ${resource_name}${NC}"
    docker $resource_type rm "$resource_name" 2>/dev/null || {
      echo -e "${YELLOW}    ⚠️  Failed to remove ${resource_name} (may already be removed)${NC}"
    }
  else
    echo -e "${YELLOW}    ${description} not found: ${resource_name}${NC}"
  fi
}

# 1. Remove stacks
echo -e "${BLUE}1️⃣  Removing stacks...${NC}"
echo ""

# Main stack
safe_remove "stack" "$STACK_NAME" "Stack"

# Dashboard stack
safe_remove "stack" "${STACK_NAME}_dashboard" "Dashboard stack"

# Wait a moment for stacks to remove
sleep 5

# 2. Remove any remaining Obiente services
echo ""
echo -e "${BLUE}2️⃣  Removing remaining Obiente services...${NC}"
echo ""

# Only remove services that belong to Obiente stacks
ALL_SERVICES=$(docker service ls --format "{{.Name}}" 2>/dev/null || echo "")
if [ -n "$ALL_SERVICES" ]; then
  while IFS= read -r service; do
    if [ -n "$service" ]; then
      # Check if service belongs to Obiente stacks
      if [[ "$service" == "${STACK_NAME}_"* ]] || [[ "$service" == *"_${STACK_NAME}_"* ]]; then
        echo -e "${BLUE}  Removing Obiente service: ${service}${NC}"
        docker service rm "$service" 2>/dev/null || true
      else
        echo -e "${GREEN}  Skipping non-Obiente service: ${service}${NC}"
      fi
    fi
  done <<< "$ALL_SERVICES"
else
  echo -e "${GREEN}  No services found${NC}"
fi

echo ""

# 3. Stop and remove only Obiente containers
echo -e "${BLUE}3️⃣  Removing Obiente containers...${NC}"
echo ""

# Get containers that belong to Obiente services
OBIENTE_CONTAINERS=$(docker ps -a --filter "label=com.docker.stack.namespace=${STACK_NAME}" --format "{{.ID}}" 2>/dev/null || echo "")
OBIENTE_CONTAINERS="$OBIENTE_CONTAINERS $(docker ps -a --filter "label=com.docker.stack.namespace=${STACK_NAME}_dashboard" --format "{{.ID}}" 2>/dev/null || echo "")"

# Also check by container name pattern
OBIENTE_CONTAINERS="$OBIENTE_CONTAINERS $(docker ps -a --filter "name=${STACK_NAME}_" --format "{{.ID}}" 2>/dev/null || echo "")"

# Remove duplicates and empty lines
OBIENTE_CONTAINERS=$(echo "$OBIENTE_CONTAINERS" | tr ' ' '\n' | grep -v '^$' | sort -u | tr '\n' ' ')

if [ -n "$OBIENTE_CONTAINERS" ]; then
  CONTAINER_COUNT=$(echo "$OBIENTE_CONTAINERS" | wc -w)
  echo -e "${BLUE}  Found ${CONTAINER_COUNT} Obiente container(s)...${NC}"
  echo -e "${BLUE}  Stopping containers...${NC}"
  docker stop $OBIENTE_CONTAINERS 2>/dev/null || true
  
  echo -e "${BLUE}  Removing containers...${NC}"
  docker rm -f $OBIENTE_CONTAINERS 2>/dev/null || true
else
  echo -e "${GREEN}  No Obiente containers found${NC}"
fi

echo ""

# 4. Remove volumes
echo -e "${BLUE}4️⃣  Removing volumes...${NC}"
echo -e "${YELLOW}  ⚠️  WARNING: This will delete all data in volumes!${NC}"
echo ""

# List volumes to be removed
VOLUMES=$(docker volume ls --format "{{.Name}}" 2>/dev/null | grep -E "(obiente|postgres|timescale|redis|traefik|prometheus|grafana)" || echo "")
if [ -n "$VOLUMES" ]; then
  echo -e "${BLUE}  Volumes to be removed:${NC}"
  echo "$VOLUMES" | while read vol; do
    echo "    - $vol"
  done
  echo ""
  
  echo -e "${BLUE}  Removing volumes...${NC}"
  echo "$VOLUMES" | while read vol; do
    docker volume rm "$vol" 2>/dev/null || echo -e "${YELLOW}    ⚠️  Failed to remove volume: $vol${NC}"
  done
else
  echo -e "${GREEN}  No matching volumes found${NC}"
fi

# Only remove Obiente-related volumes (already filtered above)
# Do NOT prune all volumes - this would remove other projects' volumes

echo ""

# 5. Remove networks
echo -e "${BLUE}5️⃣  Removing networks...${NC}"
echo ""

# Remove stack networks
NETWORKS=$(docker network ls --format "{{.Name}}" --filter driver=overlay 2>/dev/null | grep -E "(obiente|${STACK_NAME})" || echo "")
if [ -n "$NETWORKS" ]; then
  echo "$NETWORKS" | while read net; do
    echo -e "${BLUE}  Removing network: ${net}${NC}"
    docker network rm "$net" 2>/dev/null || echo -e "${YELLOW}    ⚠️  Failed to remove network: $net${NC}"
  done
else
  echo -e "${GREEN}  No matching networks found${NC}"
fi

echo ""

# 6. Remove Obiente-related images only
echo -e "${BLUE}6️⃣  Removing Obiente-related images...${NC}"
echo ""

# Only remove images that are clearly Obiente-related
OBIENTE_IMAGES=$(docker images --format "{{.Repository}}:{{.Tag}}" 2>/dev/null | grep -E "(obiente|ghcr.io/obiente)" || echo "")
if [ -n "$OBIENTE_IMAGES" ]; then
  echo -e "${BLUE}  Obiente images found:${NC}"
  echo "$OBIENTE_IMAGES" | while read img; do
    if [ -n "$img" ]; then
      echo "    - $img"
      docker rmi "$img" 2>/dev/null || echo -e "${YELLOW}      ⚠️  Failed to remove (may be in use)${NC}"
    fi
  done
else
  echo -e "${GREEN}  No Obiente images found${NC}"
fi

# Do NOT prune all images/build cache/networks - this would affect other projects

echo ""

# 7. Show final status
echo -e "${BLUE}📊 Final Status:${NC}"
echo ""

echo -e "${BLUE}  Stacks:${NC}"
docker stack ls 2>/dev/null || echo "    No stacks found"

echo ""
echo -e "${BLUE}  Services:${NC}"
docker service ls 2>/dev/null || echo "    No services found"

echo ""
echo -e "${BLUE}  Containers:${NC}"
docker ps -a --format "table {{.ID}}\t{{.Names}}\t{{.Status}}" 2>/dev/null || echo "    No containers found"

echo ""
echo -e "${BLUE}  Volumes:${NC}"
docker volume ls 2>/dev/null || echo "    No volumes found"

echo ""
echo -e "${BLUE}  Networks:${NC}"
docker network ls --filter driver=overlay 2>/dev/null || echo "    No overlay networks found"

echo ""
echo -e "${GREEN}✅ Cleanup complete!${NC}"
echo ""
echo -e "${BLUE}📋 Next steps:${NC}"
echo "  1. Recreate directories if needed:"
echo "     mkdir -p /var/lib/obiente /var/obiente/tmp/obiente-volumes /var/obiente/tmp/obiente-deployments"
echo ""
echo "  2. Redeploy stacks:"
echo "     ./scripts/deploy-swarm.sh"
echo ""
echo -e "${YELLOW}💡 Note: All data has been removed. You'll need to redeploy everything.${NC}"
echo ""

