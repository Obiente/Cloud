#!/usr/bin/env bash
# Usage: ./scripts/rollback-swarm.sh [stack-name] [service-name]
set -euo pipefail

STACK_NAME="${1:-obiente}"
SERVICE_NAME="${2:-}"
ROLLBACK_TIMEOUT="${SWARM_ROLLBACK_TIMEOUT_SECONDS:-300}"

if ! [[ "$ROLLBACK_TIMEOUT" =~ ^[1-9][0-9]*$ ]]; then
  echo "SWARM_ROLLBACK_TIMEOUT_SECONDS must be a positive integer." >&2
  exit 1
fi
if ! command -v timeout >/dev/null 2>&1; then
  echo "The timeout command is required for bounded rollbacks." >&2
  exit 1
fi
if ! MAIN_SERVICES="$(docker stack services "$STACK_NAME" --format '{{.Name}}')"; then
  echo "Could not list stack services. No rollback started." >&2
  exit 1
fi
if [ -z "$MAIN_SERVICES" ]; then
  echo "No services found in stack $STACK_NAME." >&2
  exit 1
fi

rollback_service() {
  local service="$1" output
  echo "Rolling back $service..."
  if output="$(timeout --kill-after=1 "$ROLLBACK_TIMEOUT" docker service rollback --detach=false "$service" 2>&1)"; then
    echo "Rollback completed for $service."
    return 0
  fi
  echo "Rollback failed or timed out for $service:" >&2
  printf '%s\n' "$output" >&2
  echo "Inspect docker service ps $service before retrying." >&2
  return 1
}

if [ -n "$SERVICE_NAME" ]; then
  FULL_SERVICE_NAME="${STACK_NAME}_${SERVICE_NAME}"
  if ! grep -Fxq -- "$FULL_SERVICE_NAME" <<< "$MAIN_SERVICES"; then
    echo "Service $FULL_SERVICE_NAME was not found in stack $STACK_NAME." >&2
    exit 1
  fi
  rollback_service "$FULL_SERVICE_NAME"
  exit 0
fi

printf 'Services in stack %s:\n%s\n' "$STACK_NAME" "$MAIN_SERVICES"
echo "This rolls back every listed service to its previous specification."
read -r -p 'Continue with rollback? (y/N) ' -n 1 reply || reply='n'
echo
if [[ ! "$reply" =~ ^[Yy]$ ]]; then
  echo "Rollback cancelled."
  exit 0
fi

failed=0
while IFS= read -r service; do
  [ -n "$service" ] || continue
  if ! rollback_service "$service"; then failed=1; fi
done <<< "$MAIN_SERVICES"

if [ "$failed" -ne 0 ]; then
  echo "Rollback incomplete: one or more services failed. Review the errors above." >&2
  exit 1
fi
echo "Rollback completed for all services."
