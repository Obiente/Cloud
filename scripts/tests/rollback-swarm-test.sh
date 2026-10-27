#!/usr/bin/env bash
set -Eeuo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="$(mktemp -d)"
trap 'rm -rf "$TEST_DIR"' EXIT
mkdir -p "$TEST_DIR/bin"
fail() { printf 'FAIL: %s\n' "$1" >&2; exit 1; }
cat > "$TEST_DIR/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
case "$1 $2" in
  'stack ls') printf 'fixture\n' ;;
  'stack services'|'service ls')
    if [ "${MOCK_LIST_FAIL:-false}" = true ]; then exit 1; fi
    printf 'fixture_api-gateway\nfixture_dashboard\n'
    ;;
  'service rollback')
    service="${!#}"
    printf '%s\n' "$service" >> "$MOCK_CALLS"
    if [ "$service" = "${MOCK_FAIL:-none}" ]; then echo 'injected rollback failure' >&2; exit 1; fi
    if [ "${MOCK_HANG:-false}" = true ]; then sleep 10; fi
    ;;
  'service ps') exit 0 ;;
  *) exit 93 ;;
esac
EOF
chmod +x "$TEST_DIR/bin/docker"
export PATH="$TEST_DIR/bin:$PATH" MOCK_CALLS="$TEST_DIR/calls"
export SWARM_ROLLBACK_TIMEOUT_SECONDS=1

if printf 'y\n' | MOCK_FAIL=fixture_dashboard bash "$SCRIPT_DIR/rollback-swarm.sh" fixture > "$TEST_DIR/result" 2>&1; then
  fail "bulk rollback returned success after a service failed"
fi
grep -Fq 'injected rollback failure' "$TEST_DIR/result" || fail "rollback error was hidden"
[[ "$(wc -l < "$MOCK_CALLS")" -eq 2 ]] || fail "bulk rollback did not visit both services"

: > "$MOCK_CALLS"
if MOCK_FAIL=fixture_api-gateway bash "$SCRIPT_DIR/rollback-swarm.sh" fixture api-gateway > "$TEST_DIR/result" 2>&1; then fail "single rollback failure was ignored"; fi
grep -Fq 'injected rollback failure' "$TEST_DIR/result" || fail "single rollback error was hidden"

: > "$MOCK_CALLS"
printf 'y\n' | bash "$SCRIPT_DIR/rollback-swarm.sh" fixture > "$TEST_DIR/result" 2>&1
[[ "$(wc -l < "$MOCK_CALLS")" -eq 2 ]] || fail "successful bulk rollback missed a service"
: > "$MOCK_CALLS"
bash "$SCRIPT_DIR/rollback-swarm.sh" fixture dashboard > "$TEST_DIR/result" 2>&1
[[ "$(cat "$MOCK_CALLS")" = fixture_dashboard ]] || fail "specific rollback touched another service"

: > "$MOCK_CALLS"
if MOCK_LIST_FAIL=true bash "$SCRIPT_DIR/rollback-swarm.sh" fixture dashboard > "$TEST_DIR/result" 2>&1; then fail "Docker listing failure was ignored"; fi
if bash "$SCRIPT_DIR/rollback-swarm.sh" fixture '.*' > "$TEST_DIR/result" 2>&1; then fail "service selection used a regular expression"; fi
[[ ! -s "$MOCK_CALLS" ]] || fail "invalid selection started a rollback"
printf 'n\n' | bash "$SCRIPT_DIR/rollback-swarm.sh" fixture > "$TEST_DIR/result" 2>&1
[[ ! -s "$MOCK_CALLS" ]] || fail "cancelled rollback changed services"
if SWARM_ROLLBACK_TIMEOUT_SECONDS=invalid bash "$SCRIPT_DIR/rollback-swarm.sh" fixture dashboard > "$TEST_DIR/result" 2>&1; then fail "invalid timeout was accepted"; fi
started=$SECONDS
if MOCK_HANG=true bash "$SCRIPT_DIR/rollback-swarm.sh" fixture dashboard > "$TEST_DIR/result" 2>&1; then fail "stalled rollback returned success"; fi
(( SECONDS - started < 4 )) || fail "rollback did not respect its deadline"
printf 'Swarm rollback tests passed\n'
