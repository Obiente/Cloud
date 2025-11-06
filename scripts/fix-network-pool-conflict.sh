#!/bin/bash
# Fix Docker Swarm network IP pool conflicts
# This script identifies and removes conflicting networks, then recreates the Obiente network with a specific subnet
# Run on a Docker Swarm manager node
# Usage: ./scripts/fix-network-pool-conflict.sh [--subnet <subnet>] [--force]

set -e

STACK_NAME="${STACK_NAME:-obiente}"
NETWORK_NAME="${STACK_NAME}_obiente-network"
SUBNET="${SUBNET:-10.0.9.0/24}"
FORCE="${FORCE:-false}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --subnet)
      SUBNET="$2"
      shift 2
      ;;
    --force)
      FORCE=true
      shift
      ;;
    *)
      echo -e "${RED}❌ Unknown option: $1${NC}"
      echo "Usage: ./scripts/fix-network-pool-conflict.sh [--subnet <subnet>] [--force]"
      exit 1
      ;;
  esac
done

echo -e "${BLUE}🔧 Fixing Docker Swarm Network IP Pool Conflict${NC}"
echo ""
echo -e "${BLUE}  Target network: ${NETWORK_NAME}${NC}"
echo -e "${BLUE}  Subnet: ${SUBNET}${NC}"
echo ""

# Check if we're on a manager node
if ! docker node ls &>/dev/null; then
  echo -e "${RED}❌ Error: This script must be run on a Docker Swarm manager node${NC}"
  exit 1
fi

# Step 1: Find all overlay networks and their subnets
echo -e "${BLUE}1️⃣  Analyzing network subnets...${NC}"
echo ""

ALL_NETWORKS=$(docker network ls --filter driver=overlay --format "{{.ID}}" 2>/dev/null || echo "")
CONFLICTING_NETWORKS=()
NETWORK_SUBNETS=()

for NET_ID in $ALL_NETWORKS; do
  NET_NAME=$(docker network inspect "$NET_ID" --format '{{.Name}}' 2>/dev/null || echo "unknown")
  IPAM=$(docker network inspect "$NET_ID" --format '{{json .IPAM}}' 2>/dev/null || echo "{}")
  
  NET_SUBNET=$(echo "$IPAM" | jq -r '.Config[0].Subnet // "none"' 2>/dev/null || echo "none")
  
  if [ "$NET_SUBNET" != "none" ] && [ "$NET_SUBNET" != "null" ]; then
    echo -e "${BLUE}  Network: ${NET_NAME}${NC}"
    echo "    ID: $NET_ID"
    echo "    Subnet: $NET_SUBNET"
    
    # Check if subnet overlaps with our target subnet
    # Simple check: same subnet = conflict
    if [ "$NET_SUBNET" = "$SUBNET" ] && [ "$NET_NAME" != "$NETWORK_NAME" ]; then
      CONFLICTING_NETWORKS+=("$NET_ID|$NET_NAME|$NET_SUBNET")
      echo -e "${RED}    ⚠️  CONFLICT: Same subnet as target!${NC}"
    fi
    
    NETWORK_SUBNETS+=("$NET_NAME|$NET_SUBNET")
    echo ""
  fi
done

# Step 2: Check if target network exists and its current subnet
echo -e "${BLUE}2️⃣  Checking target network...${NC}"
echo ""

if docker network inspect "$NETWORK_NAME" &>/dev/null; then
  CURRENT_SUBNET=$(docker network inspect "$NETWORK_NAME" --format '{{json .IPAM}}' 2>/dev/null | jq -r '.Config[0].Subnet // "none"' 2>/dev/null || echo "none")
  echo -e "${YELLOW}  Target network exists: ${NETWORK_NAME}${NC}"
  echo "    Current subnet: $CURRENT_SUBNET"
  
  if [ "$CURRENT_SUBNET" = "$SUBNET" ]; then
    echo -e "${GREEN}    Subnet matches target - no change needed${NC}"
  else
    echo -e "${YELLOW}    Subnet differs - will recreate${NC}"
  fi
else
  echo -e "${BLUE}  Target network does not exist - will create${NC}"
fi

echo ""

# Step 3: Identify conflicting networks
if [ ${#CONFLICTING_NETWORKS[@]} -gt 0 ]; then
  echo -e "${RED}3️⃣  Found ${#CONFLICTING_NETWORKS[@]} conflicting network(s):${NC}"
  echo ""
  
  for conflict in "${CONFLICTING_NETWORKS[@]}"; do
    IFS='|' read -r NET_ID NET_NAME NET_SUBNET <<< "$conflict"
    echo -e "${RED}    - ${NET_NAME} (${NET_ID})${NC}"
    echo "      Subnet: $NET_SUBNET"
    
    # Check if network is in use
    USED_BY=$(docker service ls --format "{{.Name}}" | while read svc; do
      SERVICE_NETWORKS=$(docker service inspect "$svc" --format '{{range .Spec.TaskTemplate.Networks}}{{.Target}}{{"\n"}}{{end}}' 2>/dev/null || echo "")
      if echo "$SERVICE_NETWORKS" | grep -q "^${NET_ID}$"; then
        echo "$svc"
      fi
    done | tr '\n' ' ')
    
    if [ -n "$USED_BY" ]; then
      echo -e "${YELLOW}      ⚠️  IN USE by services: ${USED_BY}${NC}"
    else
      echo -e "${GREEN}      ✅ Not in use - safe to remove${NC}"
    fi
    echo ""
  done
else
  echo -e "${GREEN}3️⃣  No conflicting networks found${NC}"
  echo ""
fi

# Step 4: Apply fixes
if [ "$FORCE" != "true" ]; then
  echo -e "${YELLOW}💡 Review the information above${NC}"
  echo ""
  echo "To apply fixes, run:"
  echo "  ./scripts/fix-network-pool-conflict.sh --subnet ${SUBNET} --force"
  echo ""
  echo -e "${RED}WARNING: This will:${NC}"
  echo "  - Remove conflicting networks (if safe)"
  echo "  - Remove and recreate ${NETWORK_NAME}"
  echo "  - May cause brief service interruption"
  echo ""
  exit 0
fi

echo -e "${BLUE}4️⃣  Applying fixes...${NC}"
echo ""

# Step 4a: Remove conflicting networks (only if not in use)
REMOVED_COUNT=0
for conflict in "${CONFLICTING_NETWORKS[@]}"; do
  IFS='|' read -r NET_ID NET_NAME NET_SUBNET <<< "$conflict"
  
  # Check if network is in use
  USED_BY=$(docker service ls --format "{{.Name}}" | while read svc; do
    SERVICE_NETWORKS=$(docker service inspect "$svc" --format '{{range .Spec.TaskTemplate.Networks}}{{.Target}}{{"\n"}}{{end}}' 2>/dev/null || echo "")
    if echo "$SERVICE_NETWORKS" | grep -q "^${NET_ID}$"; then
      echo "$svc"
    fi
  done | tr '\n' ' ')
  
  if [ -z "$USED_BY" ]; then
    echo -e "${BLUE}  Removing conflicting network: ${NET_NAME}${NC}"
    docker network rm "$NET_ID" 2>/dev/null && {
      echo -e "${GREEN}    ✅ Removed${NC}"
      REMOVED_COUNT=$((REMOVED_COUNT + 1))
    } || {
      echo -e "${YELLOW}    ⚠️  Failed to remove (may be in use)${NC}"
    }
  else
    echo -e "${YELLOW}  Skipping ${NET_NAME} - in use by: ${USED_BY}${NC}"
    echo -e "${YELLOW}    Remove services first or use a different subnet${NC}"
  fi
done

echo ""

# Step 4b: Remove target network if it exists
if docker network inspect "$NETWORK_NAME" &>/dev/null; then
  echo -e "${BLUE}  Removing target network: ${NETWORK_NAME}${NC}"
  
  # Check if network is in use
  USED_BY=$(docker service ls --format "{{.Name}}" | while read svc; do
    SERVICE_NETWORKS=$(docker service inspect "$svc" --format '{{range .Spec.TaskTemplate.Networks}}{{.Target}}{{"\n"}}{{end}}' 2>/dev/null || echo "")
    if echo "$SERVICE_NETWORKS" | grep -q "$NETWORK_NAME"; then
      echo "$svc"
    fi
  done | tr '\n' ' ')
  
  if [ -n "$USED_BY" ]; then
    echo -e "${YELLOW}    ⚠️  Network is in use by services: ${USED_BY}${NC}"
    echo -e "${YELLOW}    Services must be removed or updated first${NC}"
    echo ""
    echo -e "${BLUE}    Removing stacks to free network...${NC}"
    docker stack rm "$STACK_NAME" "${STACK_NAME}_dashboard" 2>/dev/null || true
    echo -e "${BLUE}    Waiting for stacks to remove...${NC}"
    sleep 10
  fi
  
  docker network rm "$NETWORK_NAME" 2>/dev/null && {
    echo -e "${GREEN}    ✅ Removed${NC}"
  } || {
    echo -e "${YELLOW}    ⚠️  Failed to remove (may need to remove services first)${NC}"
    echo -e "${YELLOW}    Try: docker stack rm ${STACK_NAME} ${STACK_NAME}_dashboard${NC}"
  }
else
  echo -e "${GREEN}  Target network does not exist - skipping removal${NC}"
fi

echo ""

# Step 4c: Create network with specific subnet
echo -e "${BLUE}  Creating network with subnet ${SUBNET}...${NC}"
if docker network create --driver overlay --subnet "$SUBNET" "$NETWORK_NAME" 2>/dev/null; then
  echo -e "${GREEN}    ✅ Network created successfully${NC}"
else
  echo -e "${RED}    ❌ Failed to create network${NC}"
  echo ""
  echo -e "${YELLOW}    Possible causes:${NC}"
  echo "      - Subnet still conflicts with existing network"
  echo "      - Network name already exists"
  echo "      - Insufficient permissions"
  echo ""
  exit 1
fi

echo ""

# Step 5: Verify network
echo -e "${BLUE}5️⃣  Verifying network...${NC}"
docker network inspect "$NETWORK_NAME" --format '{{json .}}' | jq '{
  name: .Name,
  driver: .Driver,
  scope: .Scope,
  subnet: .IPAM.Config[0].Subnet,
  gateway: .IPAM.Config[0].Gateway
}' 2>/dev/null || docker network inspect "$NETWORK_NAME"

echo ""
echo -e "${GREEN}✅ Network conflict fixed!${NC}"
echo ""
echo -e "${BLUE}📋 Next steps:${NC}"
echo "  1. Deploy the stacks:"
echo "     ./scripts/deploy-swarm.sh"
echo ""
echo -e "${BLUE}💡 To prevent future conflicts, specify subnet in docker-compose.swarm.yml:${NC}"
echo ""
echo "networks:"
echo "  obiente-network:"
echo "    driver: overlay"
echo "    ipam:"
echo "      config:"
echo "        - subnet: ${SUBNET}"
echo ""

