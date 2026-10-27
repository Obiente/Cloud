#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="$(mktemp -d)"
trap 'rm -rf "$TEST_DIR"' EXIT
mkdir -p "$TEST_DIR/bin"
# shellcheck source=scripts/lib/swarm-updates.sh
source "$SCRIPT_DIR/lib/swarm-updates.sh"
fail() { printf 'FAIL: %s\n' "$1" >&2; exit 1; }

cat > "$TEST_DIR/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
if [ "$1 $2" = 'stack services' ]; then
  if [ "${MOCK_LIST_FAIL:-false}" = true ]; then exit 1; fi
  printf '%s\n' "$MOCK_SERVICES"
  exit 0
fi
if [ "$1 $2 $3 $4" = 'service update --detach=false --force' ]; then
  printf '%s\n' "$5" >> "$MOCK_UPDATES"
  if [ "$5" = "${MOCK_UPDATE_FAIL:-none}" ]; then exit 1; fi
  if [ "$5" = "${MOCK_UPDATE_HANG:-none}" ]; then sleep 30; fi
  sleep 0.1
  printf '%s\n' "$5" >> "$MOCK_COMPLETED"
  exit 0
fi
exit 93
EOF
chmod +x "$TEST_DIR/bin/docker"
export PATH="$TEST_DIR/bin:$PATH"
export MOCK_UPDATES="$TEST_DIR/updates" MOCK_COMPLETED="$TEST_DIR/completed"
export MOCK_SERVICES=$'fixture_api-gateway\nfixture_databases-service\nfixture_databases-proxy\nfixture_dashboard'

for failed_service in fixture_api-gateway fixture_databases-proxy fixture_dashboard; do
  : > "$MOCK_UPDATES"; : > "$MOCK_COMPLETED"
  if MOCK_UPDATE_FAIL="$failed_service" restart_stack_services fixture > "$TEST_DIR/result" 2>&1; then
    fail "failed update was reported as successful"
  fi
  grep -Fq "$failed_service" "$TEST_DIR/result" || fail "failed service was not identified"
  [[ "$(wc -l < "$MOCK_COMPLETED")" -eq 3 ]] || fail "returned without waiting for other updates"
done

: > "$MOCK_UPDATES"; : > "$MOCK_COMPLETED"
restart_stack_services fixture > "$TEST_DIR/result" 2>&1
[[ "$(sort -u "$MOCK_UPDATES" | wc -l)" -eq 4 ]] || fail "a stack service was missed"
[[ "$(grep -Fxc fixture_dashboard "$MOCK_UPDATES")" -eq 1 ]] || fail "dashboard was restarted twice"
[[ "$(wc -l < "$MOCK_COMPLETED")" -eq 4 ]] || fail "success was reported before updates completed"

: > "$MOCK_UPDATES"
restart_stack_services fixture api-gateway databases-proxy absent > "$TEST_DIR/result" 2>&1
[[ "$(wc -l < "$MOCK_UPDATES")" -eq 2 ]] || fail "explicit service selection was ignored"

: > "$MOCK_UPDATES"; : > "$MOCK_COMPLETED"
if SWARM_UPDATE_SERIAL=true MOCK_UPDATE_FAIL=fixture_databases-service restart_stack_services fixture > "$TEST_DIR/result" 2>&1; then
  fail "sequential update failure was ignored"
fi
[[ "$(wc -l < "$MOCK_UPDATES")" -eq 2 ]] || fail "sequential restart continued after failure"
[[ "$(wc -l < "$MOCK_COMPLETED")" -eq 1 ]] || fail "sequential restart did not wait for completion"

: > "$MOCK_UPDATES"
if MOCK_LIST_FAIL=true restart_stack_services fixture > "$TEST_DIR/result" 2>&1; then fail "Docker listing failure was ignored"; fi
[[ ! -s "$MOCK_UPDATES" ]] || fail "updates started after listing failed"
if MOCK_SERVICES='' restart_stack_services fixture > "$TEST_DIR/result" 2>&1; then fail "empty stack was reported successful"; fi
if SWARM_UPDATE_TIMEOUT_SECONDS=invalid restart_stack_services fixture > "$TEST_DIR/result" 2>&1; then fail "invalid deadline was accepted"; fi
if SWARM_UPDATE_TIMEOUT_SECONDS=1 MOCK_UPDATE_HANG=fixture_api-gateway restart_stack_services fixture api-gateway > "$TEST_DIR/result" 2>&1; then fail "stalled update did not time out"; fi
grep -Fq 'did not converge within 1s' "$TEST_DIR/result" || fail "timeout was not reported"
printf 'Swarm restart completion tests passed\n'
