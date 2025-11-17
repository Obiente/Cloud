#!/bin/bash
# Remove PostgreSQL Docker configs to allow redeployment with updated content
# Docker configs can't be updated, only their labels can be changed

set -e

STACK_NAME="${STACK_NAME:-obiente}"

CONFIG_NAMES=(
  "${STACK_NAME}_postgres_init_hba"
  "${STACK_NAME}_postgres_entrypoint_wrapper"
)

echo "🔧 Removing PostgreSQL Docker configs for stack '$STACK_NAME'..."
echo ""

for config_name in "${CONFIG_NAMES[@]}"; do
  if docker config ls --format "{{.Name}}" | grep -q "^${config_name}$"; then
    echo "   Removing: $config_name"
    if docker config rm "$config_name" 2>/dev/null; then
      echo "   ✅ Removed: $config_name"
    else
      echo "   ⚠️  Could not remove $config_name (may be in use)"
      echo "      You may need to update services first or remove the stack"
    fi
  else
    echo "   ℹ️  Config not found: $config_name (already removed or never created)"
  fi
done

echo ""
echo "✅ Config removal complete!"
echo ""
echo "📝 You can now redeploy the stack:"
echo "   ./scripts/deploy-swarm.sh"

