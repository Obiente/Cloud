#!/bin/bash
# Cleanup script for Docker Swarm
# Removes old stopped containers, unused images, and cleans up Swarm tasks
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

# 1. Remove old stopped containers (on all nodes if requested)
echo -e "${BLUE}1️⃣  Cleaning up old containers...${NC}"

if [ "$ALL_NODES" = "--all-nodes" ]; then
  echo -e "${BLUE}   Running on all nodes...${NC}"
  # Get all nodes
  NODES=$(docker node ls --format "{{.Hostname}}")
  for NODE in $NODES; do
    echo -e "${BLUE}   Node: ${NODE}${NC}"
    execute "docker node update --availability drain \"\$NODE\" 2>/dev/null || true" "Draining node (if needed)"
    
    # Remove stopped containers on this node
    execute "docker ps -a --filter 'status=exited' --filter 'status=dead' --format '{{.ID}}' | xargs -r docker rm -f 2>/dev/null || true" "Removing stopped containers"
    
    execute "docker node update --availability active \"\$NODE\" 2>/dev/null || true" "Reactivating node"
  done
else
  # Just clean up on current node
  echo -e "${BLUE}   Running on current node only (use --all-nodes for all nodes)...${NC}"
  execute "docker ps -a --filter 'status=exited' --filter 'status=dead' --format '{{.ID}}' | xargs -r docker rm -f 2>/dev/null || true" "Removing stopped containers"
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

# 3. Prune unused images
echo -e "${BLUE}3️⃣  Cleaning up unused images...${NC}"
execute "docker image prune -a -f --filter 'until=24h'" "Removing unused images older than 24h"

echo ""

# 4. Prune unused networks
echo -e "${BLUE}4️⃣  Cleaning up unused networks...${NC}"
execute "docker network prune -f" "Removing unused networks"

echo ""

# 5. Prune build cache (optional, but can free up space)
echo -e "${BLUE}5️⃣  Cleaning up build cache...${NC}"
execute "docker builder prune -f --filter 'until=168h'" "Removing build cache older than 7 days"

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

