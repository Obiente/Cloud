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

    case "$line" in
      "" | [[:space:]]*"#")
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
