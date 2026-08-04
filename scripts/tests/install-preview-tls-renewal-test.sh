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

# shellcheck source=scripts/install-preview-tls-renewal.sh
source "${TEST_SCRIPT_DIR}/install-preview-tls-renewal.sh"

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

credentials_file="${TEST_DIR}/provider.env"
printf 'TEST_TOKEN=secret\n' > "$credentials_file"
chmod 600 "$credentials_file"
install_managed_credentials "$credentials_file" "$credentials_file"
assert_equals "TEST_TOKEN=secret" "$(cat "$credentials_file")" "same-path credential reinstall changed the file"
assert_equals "600" "$(stat -c '%a' "$credentials_file")" "same-path credential reinstall changed permissions"

credential_copy="${TEST_DIR}/managed/provider.env"
mkdir -p "$(dirname "$credential_copy")"
install_managed_credentials "$credentials_file" "$credential_copy"
assert_equals "TEST_TOKEN=secret" "$(cat "$credential_copy")" "credential copy lost its contents"
assert_equals "600" "$(stat -c '%a' "$credential_copy")" "credential copy has unsafe permissions"

paths_override="${TEST_DIR}/write-paths.conf"
write_service_paths_override "$paths_override" "/etc/obiente-cloud" "/var/lib/obiente/preview-tls"
grep -q '^\[Service\]$' "$paths_override" || fail "service override has no Service section"
grep -q '^ReadWritePaths="/etc/obiente-cloud" "/var/lib/obiente/preview-tls"$' "$paths_override" || \
  fail "service override does not contain the configured write paths"
assert_equals "644" "$(stat -c '%a' "$paths_override")" "service override has unexpected permissions"
assert_fails "broad /etc service write access was accepted" validate_service_write_directory /etc
assert_fails "relative service write path was accepted" validate_service_write_directory relative/path

printf 'Preview TLS installer tests passed.\n'
