#!/bin/bash
# Quick script to redeploy dashboard with proper DOMAIN substitution
# Usage: ./scripts/redeploy-dashboard.sh [domain]

set -e

DOMAIN="${1:-${DOMAIN:-obiente.cloud}}"
STACK_NAME="${STACK_NAME:-obiente}"

echo "🚀 Redeploying dashboard with DOMAIN=$DOMAIN..."

# Load .env file if it exists
if [ -f .env ]; then
  echo "📝 Loading environment variables from .env file..."
  set -a
  source .env
  set +a
fi

# Override DOMAIN if provided as argument
export DOMAIN="$DOMAIN"

# Substitute __STACK_NAME__ placeholder and DOMAIN variables in labels and network name
TEMP_DASHBOARD_COMPOSE=$(mktemp)
sed "s/__STACK_NAME__/${STACK_NAME}/g; s/\${DOMAIN:-localhost}/${DOMAIN}/g; s/\${DOMAIN}/${DOMAIN}/g" docker-compose.dashboard.yml > "$TEMP_DASHBOARD_COMPOSE"

# Deploy dashboard stack
docker stack deploy --resolve-image always -c "$TEMP_DASHBOARD_COMPOSE" "${STACK_NAME}"
rm -f "$TEMP_DASHBOARD_COMPOSE"

echo "✅ Dashboard redeployed!"

# Force update Traefik to rediscover
echo ""
echo "🔄 Forcing Traefik to rediscover services..."
docker service update --force "${STACK_NAME}_traefik" || echo "⚠️  Could not update Traefik (may not be running or may need to run on manager node)"

echo ""
echo "✅ Done! Wait ~10 seconds for Traefik to discover the dashboard."
echo "📋 Check dashboard labels:"
echo "   # Find dashboard service name first:"
echo "   DASHBOARD_SERVICE=\$(docker service ls --format '{{.Name}}' | grep -i dashboard | head -1)"
echo "   docker service inspect \$DASHBOARD_SERVICE --format '{{json .Spec.Labels}}' | jq 'to_entries | map(select(.key | startswith(\"traefik\")))'"
echo ""

