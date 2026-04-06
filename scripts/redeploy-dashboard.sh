#!/bin/bash
# Quick script to redeploy dashboard with proper DOMAIN substitution
# Usage: ./scripts/redeploy-dashboard.sh [domain]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
source "${SCRIPT_DIR}/lib/common.sh"

TMP_FILES=()
cleanup() {
  if [ ${#TMP_FILES[@]} -gt 0 ]; then
    rm -f "${TMP_FILES[@]}"
  fi
}
trap cleanup EXIT

STACK_NAME="${STACK_NAME:-obiente}"

if [ -f "${REPO_ROOT}/.env" ]; then
  load_env_file "${REPO_ROOT}/.env"
fi

# Resolve domain after loading env so self-hosted .env values win.
DOMAIN="${1:-${DOMAIN:-localhost}}"
export DOMAIN="$DOMAIN"

if [ -z "${DASHBOARD_URL:-}" ]; then
  DASHBOARD_URL="$(derive_dashboard_url "$DOMAIN")"
  export DASHBOARD_URL
fi

if [ -z "${API_URL:-}" ]; then
  API_URL="$(derive_api_url "$DOMAIN")"
  export API_URL
fi

echo "🚀 Redeploying dashboard with DOMAIN=$DOMAIN..."

# Merge docker-compose.base.yml with docker-compose.dashboard.yml
TEMP_DASHBOARD_COMPOSE=$(mktemp)
TMP_FILES+=("$TEMP_DASHBOARD_COMPOSE")
./scripts/merge-compose-files.sh docker-compose.dashboard.yml "$TEMP_DASHBOARD_COMPOSE"

# Substitute __STACK_NAME__ placeholder and DOMAIN variables
sed -i "s/__STACK_NAME__/${STACK_NAME}/g" "$TEMP_DASHBOARD_COMPOSE"
sed -i "s/\${DOMAIN:-localhost}/${DOMAIN}/g" "$TEMP_DASHBOARD_COMPOSE"
sed -i "s/\${DOMAIN}/${DOMAIN}/g" "$TEMP_DASHBOARD_COMPOSE"

# Deploy dashboard service in the same stack (not a separate stack)
docker stack deploy --resolve-image always -c "$TEMP_DASHBOARD_COMPOSE" "$STACK_NAME"

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
