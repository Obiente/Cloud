#!/bin/bash
# Check dashboard deployment across nodes
# Run this on a Docker Swarm manager node

set -e

echo "🔍 Checking Dashboard Multi-Node Deployment"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Check manager nodes
echo -e "${BLUE}1. Checking Swarm manager nodes...${NC}"
MANAGER_NODES=$(docker node ls --filter "role=manager" --format "{{.Hostname}}" | wc -l)
echo "Manager nodes: $MANAGER_NODES"

if [ "$MANAGER_NODES" -lt 2 ]; then
  echo -e "${YELLOW}⚠️  Only $MANAGER_NODES manager node(s) found${NC}"
  echo "   Dashboard is configured for:"
  echo "   - replicas: 2"
  echo "   - placement: node.role == manager"
  echo "   - max_replicas_per_node: 1"
  echo ""
  echo "   With only 1 manager node, only 1 replica can be deployed."
  echo "   To deploy on multiple nodes, you need 2+ manager nodes."
else
  echo -e "${GREEN}✅ $MANAGER_NODES manager nodes available${NC}"
fi

# Check dashboard service
echo ""
echo -e "${BLUE}2. Checking dashboard service deployment...${NC}"
DASHBOARD_SERVICE=$(docker service ls --format "{{.Name}}" | grep -i dashboard || echo "")
if [ -z "$DASHBOARD_SERVICE" ]; then
  echo -e "${RED}❌ Dashboard service not found!${NC}"
  exit 1
fi

echo "Dashboard service: $DASHBOARD_SERVICE"
docker service ls --filter "name=$DASHBOARD_SERVICE" --format "table {{.Name}}\t{{.Replicas}}\t{{.Image}}"

# Check which nodes are running dashboard
echo ""
echo -e "${BLUE}3. Checking dashboard task distribution...${NC}"
docker service ps "$DASHBOARD_SERVICE" --no-trunc --format "table {{.Name}}\t{{.Node}}\t{{.DesiredState}}\t{{.CurrentState}}"

# Count tasks per node
echo ""
echo -e "${BLUE}4. Dashboard tasks per node:${NC}"
docker service ps "$DASHBOARD_SERVICE" --format "{{.Node}}" | sort | uniq -c | while read count node; do
  echo "  $count replica(s) on: $node"
done

# Check if max_replicas_per_node is limiting deployment
TOTAL_REPLICAS=$(docker service ps "$DASHBOARD_SERVICE" --format "{{.Name}}" | wc -l)
DESIRED_REPLICAS=$(docker service inspect "$DASHBOARD_SERVICE" --format '{{.Spec.Mode.Replicated.Replicas}}' 2>/dev/null || echo "N/A")

echo ""
echo -e "${BLUE}5. Replica Summary:${NC}"
echo "  Desired: $DESIRED_REPLICAS"
echo "  Running: $TOTAL_REPLICAS"
echo "  Manager nodes: $MANAGER_NODES"

if [ "$TOTAL_REPLICAS" -lt "$DESIRED_REPLICAS" ] && [ "$MANAGER_NODES" -ge "$DESIRED_REPLICAS" ]; then
  echo -e "${YELLOW}⚠️  Not all replicas are running!${NC}"
  echo "   Check: docker service ps $DASHBOARD_SERVICE --no-trunc"
fi

# Recommendations
echo ""
echo -e "${BLUE}📋 Recommendations:${NC}"
if [ "$MANAGER_NODES" -lt 2 ]; then
  echo ""
  echo "To deploy dashboard on multiple nodes, you have options:"
  echo ""
  echo "Option 1: Add more manager nodes"
  echo "  docker node promote <node-name>"
  echo ""
  echo "Option 2: Allow dashboard on worker nodes"
  echo "  Remove 'node.role == manager' constraint from docker-compose.dashboard.yml"
  echo ""
  echo "Option 3: Increase replicas if you have enough nodes"
  echo "  Set replicas: $MANAGER_NODES (or higher if using workers)"
fi

echo ""


