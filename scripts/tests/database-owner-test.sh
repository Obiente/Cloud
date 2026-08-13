#!/usr/bin/env bash

set -Eeuo pipefail

TEST_DIR="$(mktemp -d)"
readonly TEST_DIR
TEST_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly TEST_SCRIPT_DIR

cleanup() {
  rm -rf "$TEST_DIR"
}
trap cleanup EXIT

# shellcheck source=scripts/lib/database-owner.sh
source "${TEST_SCRIPT_DIR}/lib/database-owner.sh"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

mock_bin="${TEST_DIR}/bin"
mock_log="${TEST_DIR}/docker.log"
mkdir -p "$mock_bin"

cat > "${mock_bin}/docker" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail

if [ "${1:-}" = "node" ] && [ "${2:-}" = "ls" ]; then
  case "${*:3}" in
    *databases.enabled=true*) printf '%s\n' "${MOCK_DATABASE_NODES:-}" ;;
    *postgres.enabled=true*) printf '%s\n' "${MOCK_POSTGRES_NODES:-}" ;;
  esac
  exit 0
fi
if [ "${1:-}" = "service" ] && [ "${2:-}" = "inspect" ]; then
  [ "${MOCK_SERVICE_EXISTS:-false}" = "true" ]
  exit
fi
if [ "${1:-}" = "service" ] && [ "${2:-}" = "ps" ]; then
  printf '%s\n' "${MOCK_SERVICE_NODE:-}"
  exit 0
fi
if [ "${1:-}" = "node" ] && [ "${2:-}" = "update" ]; then
  printf '%s\n' "$*" >> "${MOCK_DOCKER_LOG}"
  exit 0
fi

printf 'unexpected docker command: %s\n' "$*" >&2
exit 1
EOF
chmod 700 "${mock_bin}/docker"

PATH="${mock_bin}:${PATH}" \
MOCK_DOCKER_LOG="$mock_log" \
MOCK_SERVICE_EXISTS=true \
MOCK_SERVICE_NODE="existing-owner" \
ensure_database_owner_label "test-stack"
grep -q 'databases.enabled=true existing-owner' "$mock_log" || fail "existing service owner was not labeled"

: > "$mock_log"
PATH="${mock_bin}:${PATH}" \
MOCK_DOCKER_LOG="$mock_log" \
MOCK_POSTGRES_NODES="fresh-owner" \
ensure_database_owner_label "test-stack"
grep -q 'databases.enabled=true fresh-owner' "$mock_log" || fail "fresh install did not use the PostgreSQL owner"

if PATH="${mock_bin}:${PATH}" \
  MOCK_DOCKER_LOG="$mock_log" \
  MOCK_DATABASE_NODES=$'owner-one\nowner-two' \
  ensure_database_owner_label "test-stack"; then
  fail "multiple database owners were accepted"
fi

printf 'database owner deployment helper tests passed\n'
