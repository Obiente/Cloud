#!/bin/bash
# Fix dashboard service network configuration
# This ensures the dashboard service uses the correct network
# Run on a Docker Swarm manager node

set -e

STACK_NAME="${STACK_NAME:-obiente}"
DASHBOARD_SERVICE="${STACK_NAME}_dashboard"
EXPECTED_NETWORK="${STACK_NAME}_obiente-network"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🔧 Fixing Dashboard Service Network Configuration${NC}"
echo ""

# Check if we're on a manager node
if ! docker node ls &>/dev/null; then
  echo -e "${RED}❌ Error: Must run on a Docker Swarm manager node${NC}"
  exit 1
fi

# Check if dashboard service exists
if ! docker service inspect "$DASHBOARD_SERVICE" &>/dev/null; then
  echo -e "${RED}❌ Error: Service $DASHBOARD_SERVICE not found${NC}"
  exit 1
fi

# Check if expected network exists
if ! docker network inspect "$EXPECTED_NETWORK" &>/dev/null; then
  echo -e "${YELLOW}⚠️  Expected network $EXPECTED_NETWORK doesn't exist${NC}"
  echo -e "${BLUE}   Creating network...${NC}"
  docker network create --driver overlay "$EXPECTED_NETWORK" || {
    echo -e "${RED}❌ Failed to create network${NC}"
    exit 1
  }
fi

# Get current network configuration
echo -e "${BLUE}📊 Current Service Configuration:${NC}"
CURRENT_NETWORKS=$(docker service inspect "$DASHBOARD_SERVICE" --format '{{json .Spec.TaskTemplate.Networks}}' 2>/dev/null | jq -r '.[].Target' 2>/dev/null || echo "")

echo "  Current networks:"
for NET_ID in $CURRENT_NETWORKS; do
  NET_NAME=$(docker network inspect "$NET_ID" --format '{{.Name}}' 2>/dev/null || echo "unknown ($NET_ID)")
  echo "    - $NET_NAME"
done

EXPECTED_NET_ID=$(docker network inspect "$EXPECTED_NETWORK" --format '{{.Id}}' 2>/dev/null || echo "")

echo ""
echo -e "${BLUE}💡 Solution: Update service to use correct network${NC}"
echo ""

# Method 1: Update via docker-compose (recommended)
echo -e "${YELLOW}Option 1: Redeploy via docker-compose (RECOMMENDED)${NC}"
echo "  This ensures the service uses the correct network from the compose file:"
echo ""
# Get the directory containing this script, then go up to project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
echo "  cd \"$PROJECT_ROOT\""
echo "  export DOMAIN=\"\${DOMAIN:-obiente.cloud}\""
echo "  STACK_NAME=\"$STACK_NAME\" docker stack deploy --resolve-image always -c docker-compose.dashboard.yml $STACK_NAME"
echo ""

# Method 2: Manual service update
echo -e "${YELLOW}Option 2: Manual service update${NC}"
echo "  docker service update \\"
echo "    --network-rm <old-network-id> \\"
echo "    --network-add $EXPECTED_NETWORK \\"
echo "    $DASHBOARD_SERVICE"
echo ""

# Method 3: Remove and recreate
echo -e "${YELLOW}Option 3: Remove and recreate service${NC}"
echo "  docker service rm $DASHBOARD_SERVICE"
echo "  # Then redeploy dashboard stack"
echo ""

# Check for conflicting networks
echo -e "${BLUE}🔍 Checking for conflicting networks...${NC}"
ALL_NETWORKS=$(docker network ls --filter driver=overlay --format "{{.ID}}")
CONFLICTS=()

for NET_ID in $ALL_NETWORKS; do
  NET_NAME=$(docker network inspect "$NET_ID" --format '{{.Name}}' 2>/dev/null || echo "unknown")
  SUBNET=$(docker network inspect "$NET_ID" --format '{{json .IPAM}}' 2>/dev/null | jq -r '.Config[0].Subnet // "none"' 2>/dev/null || echo "none")
  
  if [ "$SUBNET" != "none" ] && [ "$SUBNET" != "null" ]; then
    # Check if subnet overlaps with expected network
    EXPECTED_SUBNET=$(docker network inspect "$EXPECTED_NETWORK" --format '{{json .IPAM}}' 2>/dev/null | jq -r '.Config[0].Subnet // "none"' 2>/dev/null || echo "none")
    
    if [ "$SUBNET" = "$EXPECTED_SUBNET" ] && [ "$NET_ID" != "$EXPECTED_NET_ID" ]; then
      CONFLICTS+=("$NET_ID|$NET_NAME|$SUBNET")
    fi
  fi
done

if [ ${#CONFLICTS[@]} -gt 0 ]; then
  echo -e "${RED}  ⚠️  Found networks with overlapping subnets:${NC}"
  for conflict in "${CONFLICTS[@]}"; do
    IFS='|' read -r NET_ID NET_NAME SUBNET <<< "$conflict"
    echo "    - $NET_NAME ($NET_ID): $SUBNET"
  done
  echo ""
  echo -e "${YELLOW}  Remove conflicting networks:${NC}"
  for conflict in "${CONFLICTS[@]}"; do
    IFS='|' read -r NET_ID NET_NAME SUBNET <<< "$conflict"
    echo "    docker network rm $NET_ID"
  done
else
  echo -e "${GREEN}  No obvious subnet conflicts found${NC}"
fi

echo ""
echo -e "${BLUE}📋 Quick Fix Command:${NC}"
echo ""
# Get the directory containing this script, then go up to project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
echo -e "${GREEN}  cd \"$PROJECT_ROOT\" && export DOMAIN=\"\${DOMAIN:-obiente.cloud}\" && STACK_NAME=\"$STACK_NAME\" docker stack deploy --resolve-image always -c docker-compose.dashboard.yml $STACK_NAME${NC}"
echo ""

