#!/usr/bin/env bash

set -Eeuo pipefail

readonly TEST_DIR="$(mktemp -d)"
readonly TEST_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cleanup() {
  rm -rf "$TEST_DIR"
}
trap cleanup EXIT

# shellcheck source=scripts/manage-preview-tls.sh
source "${TEST_SCRIPT_DIR}/manage-preview-tls.sh"

assert_equals() {
  local expected="$1"
  local actual="$2"
  local message="$3"
  if [ "$expected" != "$actual" ]; then
    printf 'FAIL: %s\nexpected: %s\nactual:   %s\n' "$message" "$expected" "$actual" >&2
    exit 1
  fi
}

assert_fails() {
  local message="$1"
  shift
  if ("$@") >/dev/null 2>&1; then
    printf 'FAIL: %s\n' "$message" >&2
    exit 1
  fi
}

validate_dns_name my.obiente.cloud
assert_fails "single-label DNS name was accepted" validate_dns_name localhost
assert_fails "double-dot DNS name was accepted" validate_dns_name my..example.com
assert_fails "leading hyphen was accepted" validate_dns_name -my.example.com

env_file="${TEST_DIR}/cloud.env"
cat > "$env_file" <<'EOF'
DOMAIN=obiente.cloud
PREVIEW_TLS_CERT_SECRET=old_certificate
PREVIEW_TLS_KEY_SECRET=old_key
PREVIEW_TLS_CERT_SECRET=duplicate_certificate
EOF
chmod 640 "$env_file"
write_env_secret_names "$env_file" certificate_v2 key_v2
assert_equals "1" "$(grep -c '^PREVIEW_TLS_CERT_SECRET=' "$env_file")" "certificate secret was not de-duplicated"
assert_equals "1" "$(grep -c '^PREVIEW_TLS_KEY_SECRET=' "$env_file")" "key secret was not de-duplicated"
assert_equals "certificate_v2" "$(sed -n 's/^PREVIEW_TLS_CERT_SECRET=//p' "$env_file")" "certificate secret was not updated"
assert_equals "key_v2" "$(sed -n 's/^PREVIEW_TLS_KEY_SECRET=//p' "$env_file")" "key secret was not updated"
assert_equals "640" "$(stat -c '%a' "$env_file")" "environment file permissions changed"

credentials_file="${TEST_DIR}/provider.env"
printf 'TEST_TOKEN=secret\n' > "$credentials_file"
chmod 600 "$credentials_file"
validate_credentials_file "$credentials_file"
chmod 640 "$credentials_file"
assert_fails "group-readable credentials were accepted" validate_credentials_file "$credentials_file"
chmod 600 "$credentials_file"
printf 'LEGO_DEBUG_DNS_API_HTTP_CLIENT=true\n' > "$credentials_file"
assert_fails "credential-leaking debug mode was accepted" validate_credentials_file "$credentials_file"

openssl req -x509 -newkey rsa:2048 -nodes -days 2 \
  -keyout "${TEST_DIR}/preview.key" \
  -out "${TEST_DIR}/preview.crt" \
  -subj '/CN=my.obiente.cloud' \
  -addext 'subjectAltName=DNS:my.obiente.cloud,DNS:*.my.obiente.cloud' \
  >/dev/null 2>&1
validate_certificate_pair "${TEST_DIR}/preview.crt" "${TEST_DIR}/preview.key" my.obiente.cloud

openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out "${TEST_DIR}/wrong.key" >/dev/null 2>&1
assert_fails "mismatched certificate key was accepted" validate_certificate_pair "${TEST_DIR}/preview.crt" "${TEST_DIR}/wrong.key" my.obiente.cloud

ENABLE_DNS=true PREVIEW_ACME_CHALLENGE_CNAME=_acme-preview.example.net validate_challenge_delegation
assert_fails "in-zone ACME challenge delegation was accepted" env \
  ENABLE_DNS=true PREVIEW_ACME_CHALLENGE_CNAME=my.obiente.cloud bash -c \
  "source '${TEST_SCRIPT_DIR}/manage-preview-tls.sh'; validate_challenge_delegation"
assert_fails "invalid ACME challenge delegation was accepted" env \
  ENABLE_DNS=true PREVIEW_ACME_CHALLENGE_CNAME='invalid name' bash -c \
  "source '${TEST_SCRIPT_DIR}/manage-preview-tls.sh'; validate_challenge_delegation"

mock_bin="${TEST_DIR}/bin"
mock_state="${TEST_DIR}/state"
mock_secrets="${TEST_DIR}/secrets"
mock_env="${TEST_DIR}/integration.env"
mock_credentials="${TEST_DIR}/integration-provider.env"
mkdir -p "$mock_bin" "$mock_state" "$mock_secrets"
printf 'TEST_TOKEN=secret\n' > "$mock_credentials"
chmod 600 "$mock_credentials"
printf 'DOMAIN=obiente.cloud\nENABLE_DNS=false\n' > "$mock_env"

cat > "${mock_bin}/docker" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail

case "${1:-}" in
  info)
    printf 'active true\n'
    ;;
  run)
    mkdir -p "${MOCK_STATE_DIR}/certificates"
    if [ ! -f "${MOCK_STATE_DIR}/certificates/preview-wildcard.crt" ]; then
      openssl req -x509 -newkey rsa:2048 -nodes -days 90 \
        -keyout "${MOCK_STATE_DIR}/certificates/preview-wildcard.key" \
        -out "${MOCK_STATE_DIR}/certificates/preview-wildcard.crt" \
        -subj '/CN=my.obiente.cloud' \
        -addext 'subjectAltName=DNS:my.obiente.cloud,DNS:*.my.obiente.cloud' \
        >/dev/null 2>&1
    fi
    ;;
  secret)
    case "${2:-}" in
      create)
        touch "${MOCK_SECRETS_DIR}/${3}"
        ;;
      inspect)
        [ -f "${MOCK_SECRETS_DIR}/${3}" ]
        ;;
      rm)
        rm -f "${MOCK_SECRETS_DIR}/${3}"
        ;;
      *) exit 1 ;;
    esac
    ;;
  service)
    exit 1
    ;;
  *)
    printf 'unexpected docker command: %s\n' "$*" >&2
    exit 1
    ;;
esac
EOF
chmod 700 "${mock_bin}/docker"

PATH="${mock_bin}:${PATH}" \
MOCK_STATE_DIR="$mock_state" \
MOCK_SECRETS_DIR="$mock_secrets" \
"${TEST_SCRIPT_DIR}/manage-preview-tls.sh" setup \
  --provider test \
  --credentials-file "$mock_credentials" \
  --email admin@example.com \
  --env-file "$mock_env" \
  --state-dir "$mock_state" \
  --accept-tos \
  --no-activate \
  >/dev/null

selected_certificate_secret="$(sed -n 's/^PREVIEW_TLS_CERT_SECRET=//p' "$mock_env")"
selected_key_secret="$(sed -n 's/^PREVIEW_TLS_KEY_SECRET=//p' "$mock_env")"
[ -f "${mock_secrets}/${selected_certificate_secret}" ] || fail "setup did not create the selected certificate secret"
[ -f "${mock_secrets}/${selected_key_secret}" ] || fail "setup did not create the selected key secret"
secret_count_before="$(find "$mock_secrets" -type f | wc -l)"

PATH="${mock_bin}:${PATH}" \
MOCK_STATE_DIR="$mock_state" \
MOCK_SECRETS_DIR="$mock_secrets" \
"${TEST_SCRIPT_DIR}/manage-preview-tls.sh" renew \
  --provider test \
  --credentials-file "$mock_credentials" \
  --email admin@example.com \
  --env-file "$mock_env" \
  --state-dir "$mock_state" \
  --accept-tos \
  --no-activate \
  >/dev/null

assert_equals "$secret_count_before" "$(find "$mock_secrets" -type f | wc -l)" "unchanged renewal created duplicate secrets"

issue_only_state="${TEST_DIR}/issue-only-state"
issue_only_secrets="${TEST_DIR}/issue-only-secrets"
issue_only_env="${TEST_DIR}/issue-only.env"
mkdir -p "$issue_only_state" "$issue_only_secrets"
printf 'DOMAIN=obiente.cloud\nENABLE_DNS=false\n' > "$issue_only_env"
PATH="${mock_bin}:${PATH}" \
MOCK_STATE_DIR="${issue_only_state}/issue-only" \
MOCK_SECRETS_DIR="$issue_only_secrets" \
"${TEST_SCRIPT_DIR}/manage-preview-tls.sh" setup \
  --provider test \
  --credentials-file "$mock_credentials" \
  --email admin@example.com \
  --env-file "$issue_only_env" \
  --state-dir "$issue_only_state" \
  --ca-server https://acme-staging-v02.api.letsencrypt.org/directory \
  --accept-tos \
  --issue-only \
  >/dev/null
assert_equals "0" "$(find "$issue_only_secrets" -type f | wc -l)" "issue-only mode created Swarm secrets"
assert_equals $'DOMAIN=obiente.cloud\nENABLE_DNS=false' "$(cat "$issue_only_env")" "issue-only mode changed the environment file"

assert_fails "staging activation was accepted" env \
  PATH="${mock_bin}:${PATH}" \
  MOCK_STATE_DIR="$issue_only_state" \
  MOCK_SECRETS_DIR="$issue_only_secrets" \
  "${TEST_SCRIPT_DIR}/manage-preview-tls.sh" setup \
  --provider test \
  --credentials-file "$mock_credentials" \
  --email admin@example.com \
  --env-file "$issue_only_env" \
  --state-dir "$issue_only_state" \
  --ca-server https://acme-staging-v02.api.letsencrypt.org/directory \
  --accept-tos

bootstrap_state="${TEST_DIR}/bootstrap-state"
bootstrap_secrets="${TEST_DIR}/bootstrap-secrets"
bootstrap_env="${TEST_DIR}/bootstrap.env"
mkdir -p "$bootstrap_state" "$bootstrap_secrets"
printf 'DOMAIN=obiente.cloud\n' > "$bootstrap_env"
PATH="${mock_bin}:${PATH}" \
MOCK_STATE_DIR="$bootstrap_state" \
MOCK_SECRETS_DIR="$bootstrap_secrets" \
"${TEST_SCRIPT_DIR}/manage-preview-tls.sh" bootstrap \
  --env-file "$bootstrap_env" \
  --state-dir "$bootstrap_state" \
  >/dev/null
assert_equals "2" "$(find "$bootstrap_secrets" -type f | wc -l)" "bootstrap did not create two Swarm secrets"
grep -q '^PREVIEW_TLS_CERT_SECRET=preview_tls_cert_' "$bootstrap_env" || fail "bootstrap did not select its certificate secret"
grep -q '^PREVIEW_TLS_KEY_SECRET=preview_tls_key_' "$bootstrap_env" || fail "bootstrap did not select its key secret"

printf 'Preview TLS script tests passed.\n'
