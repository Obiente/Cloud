#!/bin/bash
# Cleanup script for Docker Swarm - Obiente Cloud Only
# Removes old stopped Obiente containers, unused Obiente images, and cleans up Obiente Swarm tasks
# Only targets Obiente Cloud resources - other Docker resources are safe
# Usage: ./scripts/cleanup-swarm.sh [--dry-run] [--all-nodes]

set -e

DRY_RUN="${1:-}"
ALL_NODES="${2:-}"
STACK_NAME="${STACK_NAME:-obiente}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧹 Docker Swarm Cleanup${NC}"
echo ""

if [ "$DRY_RUN" = "--dry-run" ]; then
  echo -e "${YELLOW}🔍 DRY RUN MODE - No changes will be made${NC}"
  echo ""
fi

# Function to execute command (with dry-run check)
execute() {
  local cmd="$1"
  local description="$2"
  
  if [ "$DRY_RUN" = "--dry-run" ]; then
    echo -e "${YELLOW}  [DRY RUN] Would run: ${cmd}${NC}"
  else
    echo -e "${BLUE}  ${description}...${NC}"
    eval "$cmd" || {
      echo -e "${RED}  ⚠️  Error: ${description}${NC}"
      return 1
    }
  fi
}

# 1. Remove old stopped Obiente containers (on all nodes if requested)
echo -e "${BLUE}1️⃣  Cleaning up old Obiente containers...${NC}"

if [ "$ALL_NODES" = "--all-nodes" ]; then
  echo -e "${BLUE}   Running on all nodes...${NC}"
  # Get all nodes
  NODES=$(docker node ls --format "{{.Hostname}}")
  for NODE in $NODES; do
    echo -e "${BLUE}   Node: ${NODE}${NC}"
    execute "docker node update --availability drain \"\$NODE\" 2>/dev/null || true" "Draining node (if needed)"
    
    # Remove only stopped Obiente containers
    execute "docker ps -a --filter 'status=exited' --filter 'status=dead' --filter \"label=com.docker.stack.namespace=${STACK_NAME}\" --format '{{.ID}}' | xargs -r docker rm -f 2>/dev/null || true" "Removing stopped Obiente containers"
    execute "docker ps -a --filter 'status=exited' --filter 'status=dead' --filter \"label=com.docker.stack.namespace=${STACK_NAME}_dashboard\" --format '{{.ID}}' | xargs -r docker rm -f 2>/dev/null || true" "Removing stopped Obiente dashboard containers"
    execute "docker ps -a --filter 'status=exited' --filter 'status=dead' --filter \"name=${STACK_NAME}_\" --format '{{.ID}}' | xargs -r docker rm -f 2>/dev/null || true" "Removing stopped Obiente containers by name"
    
    execute "docker node update --availability active \"\$NODE\" 2>/dev/null || true" "Reactivating node"
  done
else
  # Just clean up on current node
  echo -e "${BLUE}   Running on current node only (use --all-nodes for all nodes)...${NC}"
  execute "docker ps -a --filter 'status=exited' --filter 'status=dead' --filter \"label=com.docker.stack.namespace=${STACK_NAME}\" --format '{{.ID}}' | xargs -r docker rm -f 2>/dev/null || true" "Removing stopped Obiente containers"
  execute "docker ps -a --filter 'status=exited' --filter 'status=dead' --filter \"label=com.docker.stack.namespace=${STACK_NAME}_dashboard\" --format '{{.ID}}' | xargs -r docker rm -f 2>/dev/null || true" "Removing stopped Obiente dashboard containers"
  execute "docker ps -a --filter 'status=exited' --filter 'status=dead' --filter \"name=${STACK_NAME}_\" --format '{{.ID}}' | xargs -r docker rm -f 2>/dev/null || true" "Removing stopped Obiente containers by name"
fi

echo ""

# 2. Remove old Swarm tasks (shutdown/failed)
echo -e "${BLUE}2️⃣  Cleaning up old Swarm tasks...${NC}"

# Get all services in stacks
if docker stack ls --format "{{.Name}}" | grep -q "^${STACK_NAME}$"; then
  SERVICES=$(docker stack services "$STACK_NAME" --format "{{.Name}}" 2>/dev/null || echo "")
  
  if [ -n "$SERVICES" ]; then
    while IFS= read -r service; do
      if [ -n "$service" ]; then
        echo -e "${BLUE}   Service: ${service}${NC}"
        # Count old tasks
        OLD_TASKS=$(docker service ps "$service" --filter "desired-state=shutdown" --filter "desired-state=rejected" --filter "desired-state=failed" --format "{{.ID}}" 2>/dev/null | wc -l)
        if [ "$OLD_TASKS" -gt 0 ]; then
          echo -e "${YELLOW}     Found ${OLD_TASKS} old tasks${NC}"
          if [ "$DRY_RUN" != "--dry-run" ]; then
            # Force update to clean up old tasks
            docker service update --force "$service" 2>/dev/null || true
          fi
        fi
      fi
    done <<< "$SERVICES"
  fi
fi

# Check dashboard stack if it exists
if docker stack ls --format "{{.Name}}" | grep -q "^${STACK_NAME}_dashboard$"; then
  DASHBOARD_SERVICES=$(docker stack services "${STACK_NAME}_dashboard" --format "{{.Name}}" 2>/dev/null || echo "")
  
  if [ -n "$DASHBOARD_SERVICES" ]; then
    while IFS= read -r service; do
      if [ -n "$service" ]; then
        echo -e "${BLUE}   Service: ${service}${NC}"
        OLD_TASKS=$(docker service ps "$service" --filter "desired-state=shutdown" --filter "desired-state=rejected" --filter "desired-state=failed" --format "{{.ID}}" 2>/dev/null | wc -l)
        if [ "$OLD_TASKS" -gt 0 ]; then
          echo -e "${YELLOW}     Found ${OLD_TASKS} old tasks${NC}"
          if [ "$DRY_RUN" != "--dry-run" ]; then
            docker service update --force "$service" 2>/dev/null || true
          fi
        fi
      fi
    done <<< "$DASHBOARD_SERVICES"
  fi
fi

echo ""

# 3. Remove Obiente-related unused images only
echo -e "${BLUE}3️⃣  Cleaning up unused Obiente images...${NC}"
# Only remove Obiente-related images that are unused
OBIENTE_IMAGES=$(docker images --format "{{.Repository}}:{{.Tag}}" --filter "dangling=true" 2>/dev/null | grep -E "(obiente|ghcr.io/obiente)" || echo "")
if [ -n "$OBIENTE_IMAGES" ]; then
  while IFS= read -r img; do
    if [ -n "$img" ]; then
      execute "docker rmi \"$img\" 2>/dev/null || true" "Removing unused Obiente image: $img"
    fi
  done <<< "$OBIENTE_IMAGES"
else
  echo -e "${GREEN}  No unused Obiente images found${NC}"
fi

echo ""

# 4. Remove only Obiente-related unused networks
echo -e "${BLUE}4️⃣  Cleaning up unused Obiente networks...${NC}"
# Only remove networks that match Obiente stack naming
OBIENTE_NETWORKS=$(docker network ls --filter driver=overlay --format "{{.Name}}" 2>/dev/null | grep -E "(^${STACK_NAME}_|${STACK_NAME})" || echo "")
if [ -n "$OBIENTE_NETWORKS" ]; then
  while IFS= read -r net; do
    if [ -n "$net" ]; then
      # Check if network is unused
      if ! docker network inspect "$net" --format '{{.Containers}}' 2>/dev/null | grep -q "."; then
        execute "docker network rm \"$net\" 2>/dev/null || true" "Removing unused Obiente network: $net"
      fi
    fi
  done <<< "$OBIENTE_NETWORKS"
else
  echo -e "${GREEN}  No unused Obiente networks found${NC}"
fi

echo ""

# 5. Note: We don't prune build cache as it's shared across all projects
echo -e "${BLUE}5️⃣  Build cache cleanup skipped${NC}"
echo -e "${GREEN}  Build cache is shared across all projects - skipping cleanup${NC}"
echo -e "${YELLOW}  To clean build cache manually: docker builder prune -f${NC}"

echo ""

# 6. Show system info
echo -e "${BLUE}📊 System Information:${NC}"
echo ""
echo -e "${BLUE}  Disk usage:${NC}"
docker system df
echo ""

echo -e "${BLUE}  Recent tasks (last 10):${NC}"
docker service ps "$STACK_NAME"_api --no-trunc --format "table {{.ID}}\t{{.Name}}\t{{.Node}}\t{{.DesiredState}}\t{{.CurrentState}}\t{{.Error}}" 2>/dev/null | head -11 || echo "  No tasks found"
echo ""

echo -e "${GREEN}✅ Cleanup complete!${NC}"
echo ""
echo -e "${BLUE}📋 Useful commands:${NC}"
echo "  View all tasks:        docker stack ps $STACK_NAME"
echo "  View disk usage:      docker system df"
echo "  Full system prune:     docker system prune -a --volumes --force"
echo "  Clean specific service: docker service update --force <service-name>"
echo ""
echo -e "${YELLOW}⚠️  Note: Use --dry-run first to see what would be cleaned${NC}"
echo ""

