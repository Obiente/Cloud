#!/bin/bash
# Check dashboard routing configuration
# Run this on a Docker Swarm manager node

set -e

STACK_NAME="${1:-obiente}"
DOMAIN="${DOMAIN:-obiente.cloud}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "🔍 Checking Dashboard routing configuration..."
echo ""

# 1. Check if dashboard service exists (try both possible names)
echo -e "${BLUE}1. Checking dashboard service...${NC}"

# Try to find dashboard service - check both possible names
DASHBOARD_SERVICE=$(docker service ls --format "{{.Name}}" | grep -E "^${STACK_NAME}_dashboard$" | head -n 1)

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
docker service ps "$DASHBOARD_SERVICE" --no-trunc | head -5 || echo "Could not get service status"

# Check labels
echo ""
echo -e "${BLUE}3. Checking dashboard service labels...${NC}"
DASHBOARD_LABELS=$(docker service inspect "$DASHBOARD_SERVICE" --format '{{json .Spec.Labels}}' 2>/dev/null || echo "{}")
echo "$DASHBOARD_LABELS" | jq '.' | grep -E "(traefik|cloud.obiente)" || echo "No Traefik labels found!"
echo ""

# Check network
echo -e "${BLUE}4. Checking network configuration...${NC}"
DASHBOARD_NETWORKS=$(docker service inspect "$DASHBOARD_SERVICE" --format '{{json .Spec.TaskTemplate.Networks}}' 2>/dev/null || echo "[]")
echo "Dashboard networks:"
echo "$DASHBOARD_NETWORKS" | jq '.'
echo ""

TRAEFIK_SERVICE="${STACK_NAME}_traefik"
TRAEFIK_NETWORK=$(docker service inspect "$TRAEFIK_SERVICE" --format '{{json .Spec.TaskTemplate.Networks}}' 2>/dev/null || echo "[]")
echo "Traefik networks:"
echo "$TRAEFIK_NETWORK" | jq '.'
echo ""

# Check Traefik service
echo -e "${BLUE}5. Checking Traefik service...${NC}"
docker service ps "$TRAEFIK_SERVICE" --no-trunc | head -5 || echo "Traefik service not found or not running"

# Check Traefik discovered routers
echo ""
echo -e "${BLUE}6. Checking Traefik discovered routers...${NC}"
echo "Looking for dashboard routes in Traefik..."
curl -s http://localhost:8080/api/http/routers 2>/dev/null | jq '.[] | select(.name | contains("dashboard")) | {name: .name, rule: .rule, service: .service}' || echo "Could not query Traefik API or parse response"

# Summary
echo ""
echo -e "${BLUE}📊 Summary:${NC}"
echo "  Dashboard service: $DASHBOARD_SERVICE"
echo "  Traefik service: $TRAEFIK_SERVICE"
echo ""
echo -e "${YELLOW}💡 To fix routing issues:${NC}"
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
