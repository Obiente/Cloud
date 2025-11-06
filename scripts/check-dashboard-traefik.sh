#!/bin/bash
# Check dashboard Traefik discovery configuration
# Run this on a Docker Swarm manager node

set -e

STACK_NAME="${STACK_NAME:-obiente}"
DASHBOARD_STACK="${STACK_NAME}_dashboard"

echo "🔍 Checking Dashboard Traefik Discovery Configuration"
echo ""

# Find dashboard service (try both possible names)
DASHBOARD_SERVICE=$(docker service ls --format "{{.Name}}" | grep -iE "(dashboard|^${STACK_NAME}_dashboard$|^${DASHBOARD_STACK}_dashboard$)" | head -n 1)

if [ -z "$DASHBOARD_SERVICE" ]; then
  echo "❌ Dashboard service not found!"
  echo "   Searched for: ${STACK_NAME}_dashboard, ${DASHBOARD_STACK}_dashboard"
  exit 1
fi

echo "✅ Dashboard service found: $DASHBOARD_SERVICE"
echo ""

# Check all labels
echo "📋 All Dashboard Service Labels:"
docker service inspect "$DASHBOARD_SERVICE" --format '{{json .Spec.Labels}}' | jq '.'
echo ""

# Check cloud.obiente.traefik label
echo "🔍 Checking cloud.obiente.traefik label:"
TRAEFIK_LABEL=$(docker service inspect "$DASHBOARD_SERVICE" --format '{{index .Spec.Labels "cloud.obiente.traefik"}}')
if [ "$TRAEFIK_LABEL" = "true" ]; then
  echo "  ✅ cloud.obiente.traefik=true found"
else
  echo "  ❌ cloud.obiente.traefik label missing or incorrect: '$TRAEFIK_LABEL'"
fi
echo ""

# Check network
echo "🌐 Dashboard Network Configuration:"
docker service inspect "$DASHBOARD_SERVICE" --format '{{json .Spec.TaskTemplate.Networks}}' | jq '.'
echo ""

# Check Traefik network configuration
echo "🔍 Traefik Network Configuration:"
docker service inspect "${STACK_NAME}_traefik" --format '{{range .Spec.TaskTemplate.ContainerSpec.Args}}{{.}}{{"\n"}}{{end}}' | grep "swarm.network" || echo "  ⚠️  Network config not found in Traefik args"
echo ""

# Check if services are on the same network
DASHBOARD_NETWORK=$(docker service inspect "$DASHBOARD_SERVICE" --format '{{index (index .Spec.TaskTemplate.Networks 0) "Target"}}')
TRAEFIK_NETWORK=$(docker service inspect "${STACK_NAME}_traefik" --format '{{index (index .Spec.TaskTemplate.Networks 0) "Target"}}')

echo "Network Comparison:"
echo "  Dashboard network ID: $DASHBOARD_NETWORK"
echo "  Traefik network ID:   $TRAEFIK_NETWORK"

if [ "$DASHBOARD_NETWORK" = "$TRAEFIK_NETWORK" ]; then
  echo "  ✅ Services are on the same network"
else
  echo "  ❌ Services are on different networks!"
fi
echo ""

# Get network names
DASHBOARD_NETWORK_NAME=$(docker network inspect "$DASHBOARD_NETWORK" --format '{{.Name}}' 2>/dev/null || echo "unknown")
TRAEFIK_NETWORK_NAME=$(docker network inspect "$TRAEFIK_NETWORK" --format '{{.Name}}' 2>/dev/null || echo "unknown")

echo "Network Names:"
echo "  Dashboard network name: $DASHBOARD_NETWORK_NAME"
echo "  Traefik network name:   $TRAEFIK_NETWORK_NAME"
echo ""

# Check Traefik logs for dashboard discovery
echo "📋 Recent Traefik logs (filtered for dashboard):"
docker service logs "${STACK_NAME}_traefik" --tail 50 2>&1 | grep -i "dashboard" || echo "  (No dashboard-related logs found)"
echo ""

# Summary
echo "📊 Summary:"
echo "  - Dashboard service exists: ✅"
if [ "$TRAEFIK_LABEL" = "true" ]; then
  echo "  - cloud.obiente.traefik label: ✅"
else
  echo "  - cloud.obiente.traefik label: ❌"
fi
if [ "$DASHBOARD_NETWORK" = "$TRAEFIK_NETWORK" ]; then
  echo "  - Same network: ✅"
else
  echo "  - Same network: ❌"
fi
echo ""
echo "💡 If everything looks correct but Traefik isn't discovering the dashboard:"
echo "   1. Update Traefik with the correct network name"
echo "   2. Force update Traefik: docker service update --force ${STACK_NAME}_traefik"
echo "   3. Wait 10-15 seconds for Traefik to rediscover services"
echo "   4. Check Traefik routes: curl http://localhost:8080/api/http/routers | jq '.[] | select(.name | contains(\"dashboard\"))'"
echo ""

