#!/usr/bin/env bash

# Wait for each requested service to converge. Never turn failed or timed-out
# background updates into a successful restart report.
restart_stack_services() {
  local stack="$1"
  shift
  local deadline="${SWARM_UPDATE_TIMEOUT_SECONDS:-300}"
  local available service full_name index
  local -a pids=() names=() failures=()

  if ! [[ "$deadline" =~ ^[1-9][0-9]*$ ]]; then
    echo "SWARM_UPDATE_TIMEOUT_SECONDS must be a positive integer." >&2
    return 1
  fi
  if ! command -v timeout >/dev/null 2>&1; then
    echo "The timeout command is required for bounded service updates." >&2
    return 1
  fi
  if ! available="$(docker stack services "$stack" --format '{{.Name}}')"; then
    echo "Could not read services for stack $stack. No forced updates started." >&2
    return 1
  fi

  # No selection means every service in this stack, including its dashboard.
  if [ "$#" -eq 0 ]; then
    while IFS= read -r full_name; do
      if [ -n "$full_name" ]; then
        set -- "$@" "${full_name#"${stack}_"}"
      fi
    done <<< "$available"
  fi
  for service in "$@"; do
    full_name="${stack}_${service}"
    if ! grep -Fxq -- "$full_name" <<< "$available"; then
      continue
    fi
    echo "Restarting $full_name..."
    if [ "${SWARM_UPDATE_SERIAL:-false}" = true ]; then
      # Keep infrastructure restarts sequential and stop at the first failure.
      if ! timeout --foreground "$deadline" docker service update --detach=false --force "$full_name"; then
        echo "Forced update failed or did not converge within ${deadline}s: $full_name" >&2
        return 1
      fi
      names+=("$full_name")
      continue
    fi
    timeout --foreground "$deadline" docker service update --detach=false --force "$full_name" &
    pids+=("$!")
    names+=("$full_name")
  done
  if [ "${#names[@]}" -eq 0 ]; then
    echo "No matching services found in stack $stack. No forced updates started." >&2
    return 1
  fi

  for index in "${!pids[@]}"; do
    if ! wait "${pids[$index]}"; then
      failures+=("${names[$index]}")
    fi
  done
  if [ "${#failures[@]}" -gt 0 ]; then
    echo "Forced updates failed or did not converge within ${deadline}s:" >&2
    printf '  - %s\n' "${failures[@]}" >&2
    echo "Inspect these services with docker service ps before retrying." >&2
    return 1
  fi
}
