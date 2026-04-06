#!/bin/bash

trim_whitespace() {
  local value="$1"
  value="${value#"${value%%[![:space:]]*}"}"
  value="${value%"${value##*[![:space:]]}"}"
  printf '%s' "$value"
}

load_env_file() {
  local env_file="${1:-.env}"
  local line=""
  local key=""
  local value=""

  if [ ! -f "$env_file" ]; then
    return 0
  fi

  echo "📝 Loading environment variables from ${env_file}..."

  while IFS= read -r line || [ -n "$line" ]; do
    line="${line%$'\r'}"
    line="$(trim_whitespace "$line")"

    case "$line" in
      "" | \#*)
        continue
        ;;
    esac

    line="${line#export }"

    if [[ "$line" != *=* ]]; then
      continue
    fi

    key="$(trim_whitespace "${line%%=*}")"
    value="${line#*=}"
    value="${value#"${value%%[![:space:]]*}"}"

    if [[ ! "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]]; then
      echo "⚠️  Skipping invalid environment key in ${env_file}: ${key}" >&2
      continue
    fi

    if [[ "$value" == \"*\" && "$value" == *\" ]]; then
      value="${value:1:${#value}-2}"
    elif [[ "$value" == \'*\' && "$value" == *\' ]]; then
      value="${value:1:${#value}-2}"
    fi

    export "${key}=${value}"
  done < "$env_file"
}

PARALLEL_PULL_FAILURES=()

wait_for_pull_slot() {
  local max_jobs="$1"
  local -n pids_ref="$2"
  local -n names_ref="$3"
  local -n images_ref="$4"
  local -n logs_ref="$5"
  local found_completed=0
  local i=0
  local pid=0

  while [ "${#pids_ref[@]}" -ge "$max_jobs" ]; do
    found_completed=0
    for i in "${!pids_ref[@]}"; do
      pid="${pids_ref[$i]}"
      if ! kill -0 "$pid" 2>/dev/null; then
        found_completed=1
        if wait "$pid"; then
          echo "   ✅ Pulled ${images_ref[$i]}"
        else
          echo "   ⚠️  Failed to pull ${images_ref[$i]}" >&2
          sed -n '1,120p' "${logs_ref[$i]}" >&2
          PARALLEL_PULL_FAILURES+=("${names_ref[$i]}")
        fi
        unset 'pids_ref[$i]' 'names_ref[$i]' 'images_ref[$i]' 'logs_ref[$i]'
        pids_ref=("${pids_ref[@]}")
        names_ref=("${names_ref[@]}")
        images_ref=("${images_ref[@]}")
        logs_ref=("${logs_ref[@]}")
        break
      fi
    done

    if [ "$found_completed" -eq 0 ]; then
      sleep 0.2
    fi
  done
}

pull_images_in_parallel() {
  local concurrency="${PULL_CONCURRENCY:-4}"
  local work_dir=""
  local spec=""
  local name=""
  local image=""
  local log_file=""
  local -a pids=()
  local -a names=()
  local -a images=()
  local -a logs=()
  local i=0
  local pid=0

  if ! [[ "$concurrency" =~ ^[1-9][0-9]*$ ]]; then
    echo "⚠️  Invalid PULL_CONCURRENCY=${concurrency}, defaulting to 4" >&2
    concurrency=4
  fi

  PARALLEL_PULL_FAILURES=()
  work_dir="$(mktemp -d)"

  for spec in "$@"; do
    name="${spec%%=*}"
    image="${spec#*=}"
    log_file="${work_dir}/${name}.log"

    echo "📦 Pulling ${image}..."
    (
      docker pull "${image}" >"${log_file}" 2>&1
    ) &

    pids+=("$!")
    names+=("${name}")
    images+=("${image}")
    logs+=("${log_file}")

    wait_for_pull_slot "$concurrency" pids names images logs
  done

  for i in "${!pids[@]}"; do
    pid="${pids[$i]}"
    if wait "$pid"; then
      echo "   ✅ Pulled ${images[$i]}"
    else
      echo "   ⚠️  Failed to pull ${images[$i]}" >&2
      sed -n '1,120p' "${logs[$i]}" >&2
      PARALLEL_PULL_FAILURES+=("${names[$i]}")
    fi
  done

  rm -rf "${work_dir}"

  if [ "${#PARALLEL_PULL_FAILURES[@]}" -gt 0 ]; then
    return 1
  fi
}
