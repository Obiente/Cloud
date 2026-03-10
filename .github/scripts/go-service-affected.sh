#!/usr/bin/env bash
set -euo pipefail

service_name="${1:?service name required}"
pr_base_sha="${2:-}"
push_before_sha="${3:-}"
head_sha="${4:-}"

emit_output() {
  local affected="$1"
  local reason="${2:-}"
  if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    {
      echo "affected=${affected}"
      echo "reason=${reason}"
    } >>"${GITHUB_OUTPUT}"
  else
    echo "affected=${affected}"
    echo "reason=${reason}"
  fi
}

if [[ "${GITHUB_EVENT_NAME:-}" == "workflow_dispatch" ]]; then
  emit_output "true" "manual dispatch"
  exit 0
fi

if [[ "${GITHUB_REF:-}" == refs/tags/* ]]; then
  emit_output "true" "tag build"
  exit 0
fi

base_sha="${pr_base_sha}"
if [[ -z "${base_sha}" ]]; then
  base_sha="${push_before_sha}"
fi

if [[ -z "${base_sha}" || "${base_sha}" =~ ^0+$ ]]; then
  if git rev-parse "${head_sha}^" >/dev/null 2>&1; then
    base_sha="$(git rev-parse "${head_sha}^")"
  else
    emit_output "true" "no diff base available"
    exit 0
  fi
fi

mapfile -t changed_files < <(git diff --name-only "${base_sha}" "${head_sha}")

if [[ "${#changed_files[@]}" -eq 0 ]]; then
  emit_output "false" "no changed files"
  exit 0
fi

for path in "${changed_files[@]}"; do
  if [[ "${path}" == "apps/${service_name}/"* ]]; then
    emit_output "true" "service source changed: ${path}"
    exit 0
  fi
done

# Global build-definition changes should still rebuild this service.
for path in "${changed_files[@]}"; do
  case "${path}" in
    apps/shared/go.mod|apps/shared/go.sum|.github/scripts/go-service-affected.sh)
      emit_output "true" "shared build metadata changed: ${path}"
      exit 0
      ;;
  esac
done

service_dir="apps/${service_name}"
if [[ ! -d "${service_dir}" ]]; then
  emit_output "true" "service directory missing locally"
  exit 0
fi

mapfile -t dependency_dirs < <(
  cd "${service_dir}" &&
    go list -deps -f '{{if .Dir}}{{.Dir}}{{end}}' ./... 2>/dev/null |
    sed '/^$/d' |
    sort -u
)

if [[ "${#dependency_dirs[@]}" -eq 0 ]]; then
  emit_output "true" "failed to resolve dependencies"
  exit 0
fi

repo_root="$(pwd)"
declare -a repo_relative_dep_dirs=()
for dep_dir in "${dependency_dirs[@]}"; do
  case "${dep_dir}" in
    "${repo_root}"/*)
      repo_relative_dep_dirs+=("${dep_dir#${repo_root}/}")
      ;;
  esac
done

for path in "${changed_files[@]}"; do
  candidate_path="${path}"
  if [[ "${path}" == packages/proto/proto/* ]]; then
    candidate_path="apps/shared/proto/${path#packages/proto/proto/}"
  fi

  for dep_dir in "${repo_relative_dep_dirs[@]}"; do
    if [[ "${candidate_path}" == "${dep_dir}" || "${candidate_path}" == "${dep_dir}/"* ]]; then
      emit_output "true" "dependency changed: ${path}"
      exit 0
    fi
  done
done

emit_output "false" "no imported dependency changed"
