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
env_uid_before="$(stat -c '%u' "$env_file")"
env_gid_before="$(stat -c '%g' "$env_file")"
write_env_secret_names "$env_file" certificate_v2 key_v2
assert_equals "1" "$(grep -c '^PREVIEW_TLS_CERT_SECRET=' "$env_file")" "certificate secret was not de-duplicated"
assert_equals "1" "$(grep -c '^PREVIEW_TLS_KEY_SECRET=' "$env_file")" "key secret was not de-duplicated"
assert_equals "certificate_v2" "$(sed -n 's/^PREVIEW_TLS_CERT_SECRET=//p' "$env_file")" "certificate secret was not updated"
assert_equals "key_v2" "$(sed -n 's/^PREVIEW_TLS_KEY_SECRET=//p' "$env_file")" "key secret was not updated"
assert_equals "640" "$(stat -c '%a' "$env_file")" "environment file permissions changed"
assert_equals "$env_uid_before" "$(stat -c '%u' "$env_file")" "environment file owner changed"
assert_equals "$env_gid_before" "$(stat -c '%g' "$env_file")" "environment file group changed"

loader_env="${TEST_DIR}/loader.env"
cat > "$loader_env" <<'EOF'
PATH=/tmp/untrusted-bin
LD_PRELOAD=/tmp/untrusted-library.so
PREVIEW_TLS_RENEW_DAYS=99
ENABLE_DNS=false
UNRELATED_SETTING=ignored
EOF
original_path="$PATH"
unset LD_PRELOAD
PREVIEW_TLS_RENEW_DAYS=12
export PREVIEW_TLS_RENEW_DAYS
unset ENABLE_DNS
load_env_file "$loader_env"
assert_equals "$original_path" "$PATH" "deployment env changed PATH"
assert_equals "" "${LD_PRELOAD:-}" "deployment env loaded LD_PRELOAD"
assert_equals "12" "$PREVIEW_TLS_RENEW_DAYS" "deployment env overrode an installed setting"
assert_equals "false" "$ENABLE_DNS" "allowlisted deployment setting was not loaded"
assert_equals "" "${UNRELATED_SETTING:-}" "unrelated deployment setting was loaded"
unset PREVIEW_TLS_RENEW_DAYS ENABLE_DNS

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
mock_docker_args="${TEST_DIR}/docker-run.args"
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
    printf '%s\n' "$*" > "${MOCK_DOCKER_ARGS_FILE:-/dev/null}"
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
        if [ "${MOCK_FAIL_CERT_SECRET:-false}" = "true" ] && [[ "${3:-}" == preview_tls_cert_* ]]; then
          exit 1
        fi
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
    case "${2:-}" in
      inspect)
        [ "${MOCK_TRAEFIK_EXISTS:-false}" = "true" ] || exit 1
        if printf '%s\n' "$@" | grep -q '^--format$' && [ -f "${MOCK_ACTIVE_SECRETS_FILE:-/dev/null}" ]; then
          cat "$MOCK_ACTIVE_SECRETS_FILE"
        fi
        ;;
      update)
        if [ -n "${MOCK_SERVICE_FAIL_ONCE_FILE:-}" ] && [ -f "$MOCK_SERVICE_FAIL_ONCE_FILE" ]; then
          rm -f "$MOCK_SERVICE_FAIL_ONCE_FILE"
          exit 1
        fi
        : > "$MOCK_ACTIVE_SECRETS_FILE"
        shift 2
        while [ "$#" -gt 0 ]; do
          if [ "$1" = "--secret-add" ]; then
            source_name="$(printf '%s' "$2" | sed -n 's/^source=\([^,]*\),.*$/\1/p')"
            target_name="$(printf '%s' "$2" | sed -n 's/^.*target=\([^,]*\),.*$/\1/p')"
            printf '%s|%s\n' "$source_name" "$target_name" >> "$MOCK_ACTIVE_SECRETS_FILE"
            shift 2
          else
            shift
          fi
        done
        ;;
      rollback)
        ;;
      *) exit 1 ;;
    esac
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
MOCK_DOCKER_ARGS_FILE="$mock_docker_args" \
"${TEST_SCRIPT_DIR}/manage-preview-tls.sh" setup \
  --provider test \
  --credentials-file "$mock_credentials" \
  --email admin@example.com \
  --env-file "$mock_env" \
  --state-dir "$mock_state" \
  --accept-tos \
  --no-activate \
  >/dev/null

assert_equals "run" "$(awk '{print $NF}' "$mock_docker_args")" "lego command appeared before its global options"

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
MOCK_DOCKER_ARGS_FILE="$mock_docker_args" \
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
grep -q -- '--renew-days 36500' "$mock_docker_args" || fail "issue-only mode did not force a fresh DNS challenge"

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

failed_secret_state="${TEST_DIR}/failed-secret-state"
failed_secret_store="${TEST_DIR}/failed-secret-store"
failed_secret_env="${TEST_DIR}/failed-secret.env"
mkdir -p "$failed_secret_state" "$failed_secret_store"
printf 'DOMAIN=obiente.cloud\nENABLE_DNS=false\n' > "$failed_secret_env"
assert_fails "certificate-secret creation failure was ignored" env \
  PATH="${mock_bin}:${PATH}" \
  MOCK_STATE_DIR="$failed_secret_state" \
  MOCK_SECRETS_DIR="$failed_secret_store" \
  MOCK_FAIL_CERT_SECRET=true \
  "${TEST_SCRIPT_DIR}/manage-preview-tls.sh" setup \
  --provider test \
  --credentials-file "$mock_credentials" \
  --email admin@example.com \
  --env-file "$failed_secret_env" \
  --state-dir "$failed_secret_state" \
  --accept-tos \
  --no-activate
assert_equals "0" "$(find "$failed_secret_store" -type f | wc -l)" "key secret was created after certificate-secret failure"
assert_equals $'DOMAIN=obiente.cloud\nENABLE_DNS=false' "$(cat "$failed_secret_env")" "failed secret creation changed the environment file"

retry_state="${TEST_DIR}/retry-state"
retry_secrets="${TEST_DIR}/retry-secrets"
retry_env="${TEST_DIR}/retry.env"
retry_active="${TEST_DIR}/retry-active-secrets"
retry_fail_once="${TEST_DIR}/retry-fail-once"
mkdir -p "$retry_state" "$retry_secrets"
printf 'DOMAIN=obiente.cloud\nENABLE_DNS=false\n' > "$retry_env"
touch "$retry_fail_once"
assert_fails "failed Traefik activation unexpectedly succeeded" env \
  PATH="${mock_bin}:${PATH}" \
  MOCK_STATE_DIR="$retry_state" \
  MOCK_SECRETS_DIR="$retry_secrets" \
  MOCK_TRAEFIK_EXISTS=true \
  MOCK_ACTIVE_SECRETS_FILE="$retry_active" \
  MOCK_SERVICE_FAIL_ONCE_FILE="$retry_fail_once" \
  "${TEST_SCRIPT_DIR}/manage-preview-tls.sh" setup \
  --provider test \
  --credentials-file "$mock_credentials" \
  --email admin@example.com \
  --env-file "$retry_env" \
  --state-dir "$retry_state" \
  --accept-tos
[ -f "${retry_state}/pending-secret-activation" ] || fail "failed activation did not retain retry metadata"
retry_secret_count="$(find "$retry_secrets" -type f | wc -l)"

PATH="${mock_bin}:${PATH}" \
MOCK_STATE_DIR="$retry_state" \
MOCK_SECRETS_DIR="$retry_secrets" \
MOCK_TRAEFIK_EXISTS=true \
MOCK_ACTIVE_SECRETS_FILE="$retry_active" \
MOCK_SERVICE_FAIL_ONCE_FILE="$retry_fail_once" \
"${TEST_SCRIPT_DIR}/manage-preview-tls.sh" renew \
  --provider test \
  --credentials-file "$mock_credentials" \
  --email admin@example.com \
  --env-file "$retry_env" \
  --state-dir "$retry_state" \
  --accept-tos \
  >/dev/null

assert_equals "$retry_secret_count" "$(find "$retry_secrets" -type f | wc -l)" "activation retry created duplicate secrets"
[ ! -f "${retry_state}/pending-secret-activation" ] || fail "successful retry retained pending activation metadata"
retry_certificate_secret="$(sed -n 's/^PREVIEW_TLS_CERT_SECRET=//p' "$retry_env")"
retry_key_secret="$(sed -n 's/^PREVIEW_TLS_KEY_SECRET=//p' "$retry_env")"
grep -q "^${retry_certificate_secret}|preview_tls_cert$" "$retry_active" || fail "retry did not activate the certificate secret"
grep -q "^${retry_key_secret}|preview_tls_key$" "$retry_active" || fail "retry did not activate the key secret"

printf 'Preview TLS script tests passed.\n'
