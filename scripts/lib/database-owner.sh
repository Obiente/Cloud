#!/usr/bin/env bash

database_owner_required() {
  grep -q '^  databases-service:' "$1"
}

ensure_database_owner_label() {
  local stack_name="$1"
  local control_service="${stack_name}_databases-service"
  local -a labeled_nodes=()
  local -a postgres_nodes=()
  local owner_node=""

  mapfile -t labeled_nodes < <(docker node ls -q --filter node.label=databases.enabled=true | awk 'NF')
  if [ "${#labeled_nodes[@]}" -gt 1 ]; then
    echo "Error: multiple Swarm nodes have databases.enabled=true; exactly one control-plane owner is required." >&2
    return 1
  fi
  if [ "${#labeled_nodes[@]}" -eq 1 ]; then
    echo "Database control-plane owner label already exists on node ${labeled_nodes[0]}."
    return 0
  fi

  if docker service inspect "$control_service" >/dev/null 2>&1; then
    owner_node="$(docker service ps "$control_service" \
      --filter desired-state=running \
      --format '{{.Node}}' 2>/dev/null | awk 'NF { print; exit }')"
  fi

  if [ -z "$owner_node" ]; then
    mapfile -t postgres_nodes < <(docker node ls -q --filter node.label=postgres.enabled=true | awk 'NF')
    if [ "${#postgres_nodes[@]}" -ne 1 ]; then
      echo "Error: cannot bootstrap the database owner; expected exactly one postgres.enabled=true node." >&2
      return 1
    fi
    owner_node="${postgres_nodes[0]}"
  fi

  docker node update --label-add databases.enabled=true "$owner_node" >/dev/null
  echo "Pinned the database control plane to its owner node ${owner_node}."
}
