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

# shellcheck source=scripts/lib/common.sh
source "${TEST_SCRIPT_DIR}/lib/common.sh"

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

assert_contains() {
  local value="$1"
  local expected="$2"
  local message="$3"

  [[ "$value" == *"$expected"* ]] || fail "$message"
}

mock_bin="${TEST_DIR}/bin"
mock_secrets="${TEST_DIR}/secrets"
compose_file="${TEST_DIR}/docker-compose.yml"
mkdir -p "$mock_bin" "$mock_secrets"

cat > "${mock_bin}/docker" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail

if [ "${1:-}" = "secret" ] && [ "${2:-}" = "inspect" ]; then
  [ -f "${MOCK_SECRETS_DIR}/${3:-}" ]
  exit
fi

printf 'unexpected docker command: %s\n' "$*" >&2
exit 1
EOF
chmod 700 "${mock_bin}/docker"

cat > "$compose_file" <<'EOF'
services:
  traefik:
    secrets:
      - source: preview_tls_cert
        target: preview_tls_cert
      - source: preview_tls_key
        target: preview_tls_key
EOF

if output="$(
  PATH="${mock_bin}:${PATH}" \
  MOCK_SECRETS_DIR="$mock_secrets" \
  require_preview_tls_configuration "$compose_file" 2>&1
)"; then
  fail "missing preview TLS secrets were accepted"
fi

assert_contains "$output" "preview_tls_cert" "missing certificate secret was not reported"
assert_contains "$output" "preview_tls_key" "missing key secret was not reported"
assert_contains "$output" "sudo ./scripts/manage-preview-tls.sh bootstrap" "bootstrap recovery command was not shown"
assert_contains "$output" "seven-day self-signed certificate" "bootstrap certificate limitation was not explained"
assert_contains "$output" "sudo ./scripts/manage-preview-tls.sh setup" "trusted certificate command was not shown"
assert_contains "$output" "--credentials-file /etc/obiente/preview-dns.env" "protected provider credential path was not shown"
assert_contains "$output" "docs/deployment/preview-tls.md" "full setup documentation was not linked"

if [[ "$output" == *"/secure/path/"* ]]; then
  fail "obsolete placeholder secret commands were still shown"
fi

touch "${mock_secrets}/preview_tls_cert" "${mock_secrets}/preview_tls_key"
PATH="${mock_bin}:${PATH}" \
MOCK_SECRETS_DIR="$mock_secrets" \
require_preview_tls_configuration "$compose_file"

printf 'common deployment helper tests passed\n'
