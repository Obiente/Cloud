#!/bin/bash
# Helper script to calculate replica counts and max_replicas_per_node
# based on cluster size and desired percentage

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to calculate ceil division
ceil() {
  local dividend=$1
  local divisor=$2
  echo $(( (dividend + divisor - 1) / divisor ))
}

# Function to calculate percentage-based value
calculate_percentage() {
  local total=$1
  local percent=$2
  local result=$(( total * percent / 100 ))
  # Ensure minimum of 1 if percentage would result in 0
  if [ $result -eq 0 ] && [ $total -gt 0 ]; then
    result=1
  fi
  echo $result
}

# Get cluster node count
echo -e "${GREEN}Calculating replica configuration for Docker Swarm cluster...${NC}\n"

# Try to get node count from Docker Swarm
if command -v docker &> /dev/null; then
  NODE_COUNT=$(docker node ls --format '{{.ID}}' | wc -l 2>/dev/null || echo "0")
  if [ "$NODE_COUNT" -gt 0 ]; then
    echo -e "${GREEN}Detected cluster size: ${NODE_COUNT} nodes${NC}\n"
  else
    echo -e "${YELLOW}Could not detect cluster size. Please enter manually:${NC}"
    read -p "Number of nodes: " NODE_COUNT
  fi
else
  echo -e "${YELLOW}Docker not found. Please enter cluster size manually:${NC}"
  read -p "Number of nodes: " NODE_COUNT
fi

if [ -z "$NODE_COUNT" ] || [ "$NODE_COUNT" -lt 1 ]; then
  echo -e "${RED}Error: Invalid node count${NC}"
  exit 1
fi

# Configuration
REPLICAS_PERCENT=${DASHBOARD_REPLICAS_PERCENT:-50}  # Default 50% of nodes
MAX_REPLICAS_PERCENT=${DASHBOARD_MAX_REPLICAS_PERCENT:-40}  # Default 40% per node

echo -e "${GREEN}Configuration:${NC}"
echo "  Replicas percentage: ${REPLICAS_PERCENT}%"
echo "  Max replicas per node percentage: ${MAX_REPLICAS_PERCENT}%"
echo ""

# Calculate desired replicas
DESIRED_REPLICAS=$(calculate_percentage $NODE_COUNT $REPLICAS_PERCENT)
# Ensure minimum of 2 for HA
if [ $DESIRED_REPLICAS -lt 2 ]; then
  DESIRED_REPLICAS=2
fi

# Calculate max replicas per node
# Formula: ceil(replicas * max_percent / 100 / node_count)
# This ensures even distribution
MAX_PER_NODE=$(ceil $(( DESIRED_REPLICAS * MAX_REPLICAS_PERCENT / 100 )) $NODE_COUNT)
# Ensure at least 1 per node
if [ $MAX_PER_NODE -lt 1 ]; then
  MAX_PER_NODE=1
fi

# Verify feasibility
MAX_POSSIBLE_REPLICAS=$(( NODE_COUNT * MAX_PER_NODE ))
if [ $DESIRED_REPLICAS -gt $MAX_POSSIBLE_REPLICAS ]; then
  echo -e "${RED}Warning: Desired replicas (${DESIRED_REPLICAS}) exceeds maximum possible (${MAX_POSSIBLE_REPLICAS})${NC}"
  echo -e "${YELLOW}Adjusting max_replicas_per_node to accommodate desired replicas...${NC}\n"
  MAX_PER_NODE=$(ceil $DESIRED_REPLICAS $NODE_COUNT)
fi

echo -e "${GREEN}Recommended Configuration:${NC}"
echo "  DASHBOARD_REPLICAS=${DESIRED_REPLICAS}"
echo "  DASHBOARD_MAX_REPLICAS_PER_NODE=${MAX_PER_NODE}"
echo ""
echo -e "${GREEN}Add to your .env file:${NC}"
echo "DASHBOARD_REPLICAS=${DESIRED_REPLICAS}"
echo "DASHBOARD_MAX_REPLICAS_PER_NODE=${MAX_PER_NODE}"
echo ""
echo -e "${GREEN}Verification:${NC}"
echo "  Cluster size: ${NODE_COUNT} nodes"
echo "  Desired replicas: ${DESIRED_REPLICAS}"
echo "  Max per node: ${MAX_PER_NODE}"
echo "  Maximum possible: $(( NODE_COUNT * MAX_PER_NODE )) replicas"
echo "  Distribution: ~$(( (DESIRED_REPLICAS + NODE_COUNT - 1) / NODE_COUNT )) replicas per node (average)"

