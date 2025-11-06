#!/bin/bash
# Troubleshooting script for Dashboard routing with Traefik
# Run this on a Docker Swarm manager node

set -e

echo "🔍 Checking Dashboard routing configuration..."
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Check if dashboard service exists
echo -e "${BLUE}1. Checking dashboard service...${NC}"
DASHBOARD_SERVICE=$(docker service ls --format "{{.Name}}" | grep -i dashboard || echo "")
if [ -z "$DASHBOARD_SERVICE" ]; then
  echo -e "${RED}❌ Dashboard service not found!${NC}"
  echo "   Deploy it with: docker stack deploy -c docker-compose.dashboard.yml obiente-dashboard"
  exit 1
else
  echo -e "${GREEN}✅ Dashboard service found: ${DASHBOARD_SERVICE}${NC}"
fi

# Check service status
echo ""
echo -e "${BLUE}2. Checking dashboard service status...${NC}"
docker service ps "$DASHBOARD_SERVICE" --no-trunc | head -5

# Check labels
echo ""
echo -e "${BLUE}3. Checking dashboard service labels...${NC}"
docker service inspect "$DASHBOARD_SERVICE" --format '{{json .Spec.Labels}}' | jq '.' | grep -E "(traefik|cloud.obiente)" || echo "No Traefik labels found!"

# Check network
echo ""
echo -e "${BLUE}4. Checking network configuration...${NC}"
DASHBOARD_NETWORKS=$(docker service inspect "$DASHBOARD_SERVICE" --format '{{range .Spec.TaskTemplate.Networks}}{{.Target}}{{end}}')
echo "Dashboard networks: $DASHBOARD_NETWORKS"

TRAEFIK_NETWORK=$(docker network ls --format "{{.Name}}" | grep obiente-network | head -1)
if [ -z "$TRAEFIK_NETWORK" ]; then
  echo -e "${RED}❌ obiente-network not found!${NC}"
else
  echo -e "${GREEN}✅ Network found: ${TRAEFIK_NETWORK}${NC}"
fi

# Check Traefik service
echo ""
echo -e "${BLUE}5. Checking Traefik service...${NC}"
TRAEFIK_SERVICE=$(docker service ls --format "{{.Name}}" | grep traefik | head -1)
if [ -z "$TRAEFIK_SERVICE" ]; then
  echo -e "${RED}❌ Traefik service not found!${NC}"
else
  echo -e "${GREEN}✅ Traefik service found: ${TRAEFIK_SERVICE}${NC}"
fi

# Check Traefik routes
echo ""
echo -e "${BLUE}6. Checking Traefik HTTP routers...${NC}"
echo "Querying Traefik API for routes..."
curl -s http://localhost:8080/api/http/routers | jq -r '.[] | select(.name | contains("dashboard")) | {name: .name, rule: .rule, service: .service}' 2>/dev/null || echo -e "${YELLOW}⚠️  Could not query Traefik API. Is Traefik accessible on port 8080?${NC}"

# Check if services are on same network
echo ""
echo -e "${BLUE}7. Verifying network connectivity...${NC}"
DASHBOARD_TASK=$(docker service ps "$DASHBOARD_SERVICE" --no-trunc --format "{{.Name}}" | head -1)
if [ -n "$DASHBOARD_TASK" ]; then
  echo "Dashboard task: $DASHBOARD_TASK"
  # Get container ID
  CONTAINER_ID=$(docker ps --filter "name=$DASHBOARD_TASK" --format "{{.ID}}" | head -1)
  if [ -n "$CONTAINER_ID" ]; then
    echo "Container ID: $CONTAINER_ID"
    docker inspect "$CONTAINER_ID" --format '{{range $key, $value := .NetworkSettings.Networks}}{{$key}}{{end}}' 2>/dev/null || echo "Could not inspect container"
  fi
fi

# Summary
echo ""
echo -e "${BLUE}📋 Summary:${NC}"
echo ""
echo "To fix dashboard routing, try:"
echo "1. Ensure DOMAIN environment variable is set when deploying:"
echo "   DOMAIN=obiente.cloud docker stack deploy -c docker-compose.dashboard.yml obiente-dashboard"
echo ""
echo "2. Force update Traefik to rediscover services:"
echo "   docker service update --force ${TRAEFIK_SERVICE}"
echo ""
echo "3. Force update dashboard to apply new labels:"
echo "   docker service update --force ${DASHBOARD_SERVICE}"
echo ""
echo "4. Check Traefik logs for discovery issues:"
echo "   docker service logs ${TRAEFIK_SERVICE} --tail 50 | grep -i dashboard"
echo ""


