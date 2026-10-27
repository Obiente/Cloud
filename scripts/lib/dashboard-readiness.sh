#!/usr/bin/env bash

# A running task alone does not prove the dashboard can be reached through its
# public route. Require the current image, all replicas and an HTTP 200 response.
wait_for_dashboard_ready() {
  local service="$1" url="$2"
  local deadline="${DASHBOARD_READY_TIMEOUT_SECONDS:-300}"
  local started=$SECONDS remaining budget snapshot update image desired tasks
  local state task_image running total status reason="waiting for service state"

  if ! [[ "$deadline" =~ ^[1-9][0-9]*$ ]]; then
    echo "DASHBOARD_READY_TIMEOUT_SECONDS must be a positive integer." >&2
    return 1
  fi
  if [[ "$url" != http://* && "$url" != https://* ]]; then
    echo "DASHBOARD_URL must be an HTTP or HTTPS URL." >&2
    return 1
  fi
  if ! command -v curl >/dev/null 2>&1 || ! command -v timeout >/dev/null 2>&1; then
    echo "curl and timeout are required to verify dashboard readiness." >&2
    return 1
  fi

  while (( SECONDS - started < deadline )); do
    remaining=$((deadline - (SECONDS - started)))
    budget=$remaining
    (( budget > 10 )) && budget=10
    if snapshot="$(timeout --kill-after=1 "$budget" docker service inspect "$service" --format '{{if .UpdateStatus}}{{.UpdateStatus.State}}{{end}}|{{.Spec.TaskTemplate.ContainerSpec.Image}}|{{if .Spec.Mode.Replicated}}{{.Spec.Mode.Replicated.Replicas}}{{else}}0{{end}}' 2>/dev/null)"; then
      IFS='|' read -r update image desired <<< "$snapshot"
      case "$update" in
        paused|rollback_*)
          echo "Dashboard rollout is $update. Inspect docker service ps $service before retrying." >&2
          return 1
          ;;
      esac
      if ! [[ "$desired" =~ ^[1-9][0-9]*$ ]] || [ -z "$image" ]; then
        echo "Dashboard service has no valid active replica target." >&2
        return 1
      fi
      remaining=$((deadline - (SECONDS - started)))
      if (( remaining <= 0 )); then break; fi
      budget=$remaining
      (( budget > 10 )) && budget=10
      if tasks="$(timeout --kill-after=1 "$budget" docker service ps "$service" --no-trunc --filter desired-state=running --format '{{.CurrentState}}|{{.Image}}' 2>/dev/null)"; then
        running=0
        total=0
        while IFS='|' read -r state task_image; do
          [ -n "$state" ] || continue
          total=$((total + 1))
          if [[ "$state" == Running\ * && "$task_image" == "$image" ]]; then
            running=$((running + 1))
          fi
        done <<< "$tasks"
        reason="$running/$desired replicas running the requested image (rollout ${update:-initial})"
        if [[ "$update" == completed || -z "$update" ]] && (( running == desired && total == desired )); then
          remaining=$((deadline - (SECONDS - started)))
          if (( remaining <= 0 )); then break; fi
          budget=$remaining
          (( budget > 5 )) && budget=5
          if status="$(curl --silent --output /dev/null --write-out '%{http_code}' --connect-timeout "$budget" --max-time "$budget" "$url")" && [ "$status" = 200 ]; then
            echo "Dashboard replicas converged and its HTTP route is responding."
            return 0
          fi
          reason="dashboard HTTP route did not return 200"
        fi
      else
        reason="Docker could not list dashboard tasks"
      fi
    else
      reason="Docker could not inspect the dashboard service"
    fi
    remaining=$((deadline - (SECONDS - started)))
    if (( remaining > 0 )); then sleep 1; fi
  done
  echo "Dashboard readiness timed out after ${deadline}s: $reason." >&2
  echo "Inspect docker service ps $service and its logs before retrying." >&2
  return 1
}
