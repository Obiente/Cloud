#!/bin/bash
# Fix Docker Swarm network IP pool conflicts
# Run this on a Docker Swarm manager node
# Usage: ./scripts/fix-swarm-networks.sh [--fix]

set -e

FIX_MODE="${1:-}"
STACK_NAME="${STACK_NAME:-obiente}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 Docker Swarm Network Diagnostic${NC}"
echo ""

# Check if we're on a manager node
if ! docker node ls &>/dev/null; then
  echo -e "${RED}❌ Error: This script must be run on a Docker Swarm manager node${NC}"
  exit 1
fi

# 1. List all networks and their IP pools
echo -e "${BLUE}1️⃣  Current Networks and IP Pools:${NC}"
echo ""
docker network ls --format "table {{.ID}}\t{{.Name}}\t{{.Driver}}\t{{.Scope}}"
echo ""

echo -e "${BLUE}2️⃣  Network IP Pool Details:${NC}"
echo ""

# Check for overlapping networks
NETWORKS=$(docker network ls --format "{{.ID}}")
OVERLAPS=()

for NET_ID in $NETWORKS; do
  NET_NAME=$(docker network inspect "$NET_ID" --format '{{.Name}}' 2>/dev/null || echo "unknown")
  IPAM=$(docker network inspect "$NET_ID" --format '{{json .IPAM}}' 2>/dev/null || echo "{}")
  
  if echo "$IPAM" | grep -q "Subnet"; then
    SUBNET=$(echo "$IPAM" | jq -r '.Config[0].Subnet // "none"' 2>/dev/null || echo "none")
    if [ "$SUBNET" != "none" ] && [ "$SUBNET" != "null" ]; then
      echo -e "${BLUE}  Network: ${NET_NAME}${NC}"
      echo "    ID: $NET_ID"
      echo "    Subnet: $SUBNET"
      echo ""
    fi
  fi
done

# 2. Check for unused networks that might be causing conflicts
echo -e "${BLUE}3️⃣  Checking for unused networks:${NC}"
echo ""

USED_NETWORKS=$(docker service ls --format "{{.Name}}" | while read svc; do
  docker service inspect "$svc" --format '{{range .Spec.TaskTemplate.Networks}}{{.Target}}{{"\n"}}{{end}}' 2>/dev/null || true
done | sort -u)

ALL_NETWORKS=$(docker network ls --format "{{.ID}}")

UNUSED_NETWORKS=()
for NET_ID in $ALL_NETWORKS; do
  NET_NAME=$(docker network inspect "$NET_ID" --format '{{.Name}}' 2>/dev/null || echo "unknown")
  # Skip predefined networks
  if [[ "$NET_NAME" == "bridge" ]] || [[ "$NET_NAME" == "host" ]] || [[ "$NET_NAME" == "none" ]]; then
    continue
  fi
  
  # Check if network is used by any service
  if ! echo "$USED_NETWORKS" | grep -q "^${NET_ID}$"; then
    # Check if network has containers
    CONTAINERS=$(docker network inspect "$NET_ID" --format '{{len .Containers}}' 2>/dev/null || echo "0")
    if [ "$CONTAINERS" = "0" ] || [ "$CONTAINERS" = "0" ]; then
      UNUSED_NETWORKS+=("$NET_ID|$NET_NAME")
    fi
  fi
done

if [ ${#UNUSED_NETWORKS[@]} -gt 0 ]; then
  echo -e "${YELLOW}  Found ${#UNUSED_NETWORKS[@]} potentially unused networks:${NC}"
  for net_info in "${UNUSED_NETWORKS[@]}"; do
    IFS='|' read -r NET_ID NET_NAME <<< "$net_info"
    echo "    - $NET_NAME ($NET_ID)"
  done
else
  echo -e "${GREEN}  No unused networks found${NC}"
fi

echo ""

# 3. Check the specific failing network
echo -e "${BLUE}4️⃣  Checking stack networks:${NC}"
echo ""

STACK_NETWORK="${STACK_NAME}_obiente-network"
if docker network inspect "$STACK_NETWORK" &>/dev/null; then
  echo -e "${GREEN}  Stack network exists: ${STACK_NETWORK}${NC}"
  SUBNET=$(docker network inspect "$STACK_NETWORK" --format '{{json .IPAM}}' 2>/dev/null | jq -r '.Config[0].Subnet // "none"' 2>/dev/null || echo "none")
  echo "    Subnet: $SUBNET"
else
  echo -e "${YELLOW}  Stack network not found: ${STACK_NETWORK}${NC}"
fi

echo ""

# 4. Check service network configuration
echo -e "${BLUE}5️⃣  Service Network Configuration:${NC}"
echo ""

if docker service inspect "${STACK_NAME}_dashboard" &>/dev/null; then
  echo -e "${BLUE}  Dashboard service networks:${NC}"
  docker service inspect "${STACK_NAME}_dashboard" --format '{{json .Spec.TaskTemplate.Networks}}' | jq -r '.[] | "    - \(.Target)"' 2>/dev/null || echo "    Could not inspect"
fi

echo ""

# 5. Fix options
if [ "$FIX_MODE" = "--fix" ]; then
  echo -e "${YELLOW}🔧 Fix Mode Enabled${NC}"
  echo ""
  
  # Option 1: Remove unused networks
  if [ ${#UNUSED_NETWORKS[@]} -gt 0 ]; then
    echo -e "${BLUE}  Removing unused networks...${NC}"
    for net_info in "${UNUSED_NETWORKS[@]}"; do
      IFS='|' read -r NET_ID NET_NAME <<< "$net_info"
      # Skip stack networks
      if [[ "$NET_NAME" == *"${STACK_NAME}"* ]]; then
        echo -e "${YELLOW}    Skipping stack network: $NET_NAME${NC}"
        continue
      fi
      echo -e "${BLUE}    Removing: $NET_NAME${NC}"
      docker network rm "$NET_ID" 2>/dev/null || echo -e "${RED}    Failed to remove: $NET_NAME${NC}"
    done
  fi
  
  # Option 2: Update dashboard service to force network recreation
  echo ""
  echo -e "${BLUE}  Forcing dashboard service update...${NC}"
  docker service update --force "${STACK_NAME}_dashboard" || {
    echo -e "${RED}  Failed to update dashboard service${NC}"
    echo -e "${YELLOW}  Try manually: docker service update --force ${STACK_NAME}_dashboard${NC}"
  }
  
  echo ""
  echo -e "${GREEN}✅ Fix operations completed${NC}"
else
  echo -e "${YELLOW}💡 To apply fixes, run:${NC}"
  echo "   ./scripts/fix-swarm-networks.sh --fix"
  echo ""
  echo -e "${BLUE}📋 Manual fix options:${NC}"
  echo ""
  echo -e "${YELLOW}Option 1: Remove unused networks${NC}"
  echo "  docker network ls"
  echo "  docker network rm <network-id>"
  echo ""
  echo -e "${YELLOW}Option 2: Recreate the stack network${NC}"
  echo "  docker network rm ${STACK_NETWORK}"
  echo "  ./scripts/deploy-swarm.sh  # Will recreate network"
  echo ""
  echo -e "${YELLOW}Option 3: Force update dashboard service${NC}"
  echo "  docker service update --force ${STACK_NAME}_dashboard"
  echo ""
  echo -e "${YELLOW}Option 4: Specify custom IP pool (if needed)${NC}"
  echo "  docker network rm ${STACK_NETWORK}"
  echo "  docker network create --driver overlay --subnet=10.0.10.0/24 ${STACK_NETWORK}"
  echo "  docker service update --force ${STACK_NAME}_dashboard"
fi

echo ""
echo -e "${BLUE}📋 Useful commands:${NC}"
echo "  List all networks:     docker network ls"
echo "  Inspect network:       docker network inspect <network-name>"
echo "  Remove network:        docker network rm <network-name>"
echo "  Service logs:          docker service logs ${STACK_NAME}_dashboard --tail 50"
echo "  Service tasks:         docker service ps ${STACK_NAME}_dashboard --no-trunc"
echo ""

