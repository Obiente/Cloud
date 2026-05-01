#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

MODE="${1:-all}"

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

contains_line() {
  local needle="$1"
  local file="$2"
  grep -Fxq "$needle" "$file"
}

validate_compose_file() {
  local compose_file="$1"
  shift
  local required_services=("$@")
  local merged_file
  local services_file

  merged_file="$(mktemp)"
  services_file="$(mktemp)"

  scripts/merge-compose-files.sh "$compose_file" "$merged_file" >/dev/null

  docker compose --project-directory "$ROOT_DIR" -f "$merged_file" config --quiet
  docker compose --project-directory "$ROOT_DIR" -f "$merged_file" config --services >"$services_file"

  for service in "${required_services[@]}"; do
    if ! contains_line "$service" "$services_file"; then
      echo "Compose file $compose_file is missing required service: $service" >&2
      echo "Services found:" >&2
      sed 's/^/  - /' "$services_file" >&2
      exit 1
    fi
  done

  echo "validated $compose_file ($(wc -l <"$services_file" | tr -d ' ') services)"
  rm -f "$merged_file" "$services_file"
}

validate_compose() {
  validate_compose_file docker-compose.yml \
    postgres \
    timescaledb \
    redis \
    traefik \
    api-gateway \
    auth-service \
    organizations-service \
    deployments-service \
    orchestrator-service \
    vps-service
}

validate_swarm_config() {
  local core_services=(
    postgres
    timescaledb
    redis
    traefik
    registry
    api-gateway
    auth-service
    organizations-service
    deployments-service
    orchestrator-service
    vps-service
  )

  validate_compose_file docker-compose.swarm.yml "${core_services[@]}"
  validate_compose_file docker-compose.swarm.dev.yml "${core_services[@]}"
  validate_compose_file docker-compose.swarm.ha.yml \
    etcd \
    pgpool \
    metrics-pgpool \
    redis-1 \
    redis-2 \
    redis-3 \
    traefik \
    registry \
    api-gateway \
    auth-service \
    organizations-service \
    deployments-service \
    orchestrator-service \
    vps-service
  validate_compose_file docker-compose.dashboard.yml dashboard
  validate_compose_file docker-compose.vps-gateway.yml vps-gateway
}

port_in_use() {
  local protocol="$1"
  local port="$2"

  case "$protocol" in
    tcp)
      ss -H -ltn | awk '{print $4}' | grep -Eq "(^|:)$port$"
      ;;
    udp)
      ss -H -lun | awk '{print $4}' | grep -Eq "(^|:)$port$"
      ;;
    *)
      echo "Unsupported port protocol in compose file: $protocol" >&2
      exit 1
      ;;
  esac
}

assert_published_ports_available() {
  local compose_file="$1"
  local ports_file

  ports_file="$(mktemp)"
  docker compose --project-directory "$ROOT_DIR" -f "$compose_file" config --format json |
    jq -r '.services | to_entries[] | select(.value.ports != null) | .key as $service | .value.ports[] | [$service, (.published | tostring), .protocol] | @tsv' >"$ports_file"

  while IFS=$'\t' read -r service published protocol; do
    if [ -z "$published" ] || [ "$published" = "null" ]; then
      continue
    fi

    if port_in_use "$protocol" "$published"; then
      echo "Cannot run compose runtime smoke: $service needs $protocol port $published, but it is already in use." >&2
      echo "Stop the conflicting process or override the published port before running the smoke." >&2
      rm -f "$ports_file"
      exit 1
    fi
  done <"$ports_file"

  rm -f "$ports_file"
}

print_runtime_diagnostics() {
  local project_name="$1"
  local compose_file="$2"

  echo "Compose runtime diagnostics for project $project_name:" >&2
  docker compose --project-name "$project_name" --project-directory "$ROOT_DIR" -f "$compose_file" ps >&2 || true
  docker compose --project-name "$project_name" --project-directory "$ROOT_DIR" -f "$compose_file" logs --tail=120 >&2 || true
}

assert_no_project_containers_remain() {
  local project_name="$1"
  local remaining

  remaining="$(docker ps -aq --filter "label=com.docker.compose.project=$project_name")"
  if [ -n "$remaining" ]; then
    echo "Compose runtime smoke cleanup left containers behind for project $project_name:" >&2
    docker ps -a --filter "label=com.docker.compose.project=$project_name" >&2
    exit 1
  fi
}

validate_runtime_compose() {
  local merged_file
  local project_name
  local timeout
  local exit_code=0

  require_command jq
  require_command ss
  require_command curl

  merged_file="$(mktemp)"
  project_name="${OBIENTE_COMPOSE_SMOKE_PROJECT:-obiente-compose-smoke}"
  timeout="${OBIENTE_COMPOSE_SMOKE_TIMEOUT:-420}"

  scripts/merge-compose-files.sh docker-compose.yml "$merged_file" >/dev/null
  docker compose --project-directory "$ROOT_DIR" -f "$merged_file" config --quiet
  assert_published_ports_available "$merged_file"

  cleanup_runtime() {
    docker compose --project-name "$project_name" --project-directory "$ROOT_DIR" -f "$merged_file" down -v --remove-orphans >/dev/null 2>&1 || true
    rm -f "$merged_file"
  }
  trap cleanup_runtime EXIT

  docker compose --project-name "$project_name" --project-directory "$ROOT_DIR" -f "$merged_file" up -d --build --wait --wait-timeout "$timeout" || exit_code=$?
  if [ "$exit_code" -ne 0 ]; then
    print_runtime_diagnostics "$project_name" "$merged_file"
    exit "$exit_code"
  fi

  curl -fsS -H 'Host: api.localhost' http://127.0.0.1/health >/dev/null
  curl -fsS http://127.0.0.1:18080/ping >/dev/null

  docker compose --project-name "$project_name" --project-directory "$ROOT_DIR" -f "$merged_file" down -v --remove-orphans
  trap - EXIT
  rm -f "$merged_file"
  assert_no_project_containers_remain "$project_name"

  echo "validated docker-compose.yml runtime smoke"
}

require_command docker
docker compose version >/dev/null

case "$MODE" in
  all)
    validate_compose
    validate_swarm_config
    ;;
  compose)
    validate_compose
    ;;
  runtime-compose)
    validate_runtime_compose
    ;;
  swarm-config)
    validate_swarm_config
    ;;
  *)
    echo "Usage: $0 [all|compose|runtime-compose|swarm-config]" >&2
    exit 2
    ;;
esac
