#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="$(mktemp -d)"
server_pid=""
cleanup() {
  if [ -n "$server_pid" ]; then kill "$server_pid" 2>/dev/null || true; wait "$server_pid" 2>/dev/null || true; fi
  rm -rf "$TEST_DIR"
}
trap cleanup EXIT
mkdir -p "$TEST_DIR/bin"
# shellcheck source=scripts/lib/dashboard-readiness.sh
source "$SCRIPT_DIR/lib/dashboard-readiness.sh"
fail() { printf 'FAIL: %s\n' "$1" >&2; exit 1; }

cat > "$TEST_DIR/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
if [ "${MOCK_DOCKER_HANG:-false}" = true ]; then sleep 5; fi
if [ "${MOCK_DOCKER_FAIL:-false}" = true ]; then exit 1; fi
case "$1 $2" in
  'service inspect') printf '%s|%s|%s\n' "${MOCK_UPDATE:-completed}" "$MOCK_IMAGE" "${MOCK_REPLICAS:-1}" ;;
  'service ps')
    if [ "${MOCK_DELAYED:-false}" = true ] && [ ! -f "$MOCK_POLLED" ]; then
      touch "$MOCK_POLLED"
      printf 'Preparing 1 second ago|%s\n' "$MOCK_IMAGE"
    else
      printf '%s\n' "$MOCK_TASKS"
    fi
    ;;
  *) exit 93 ;;
esac
EOF
chmod +x "$TEST_DIR/bin/docker"
export PATH="$TEST_DIR/bin:$PATH"
export MOCK_IMAGE='registry.example.test/dashboard:release@sha256:1111111111111111111111111111111111111111111111111111111111111111'
export MOCK_TASKS="Running 5 seconds ago|$MOCK_IMAGE"
export MOCK_POLLED="$TEST_DIR/polled"

# Use real HTTP responses, including a stalled request, rather than mocking curl.
python3 - "$TEST_DIR/port" <<'PY' &
import http.server, pathlib, sys, time
class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/hang': time.sleep(5)
        status = 503 if self.path == '/bad' else 302 if self.path == '/redirect' else 200
        self.send_response(status)
        self.end_headers()
    def log_message(self, *args): pass
server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), Handler)
pathlib.Path(sys.argv[1]).write_text(str(server.server_port))
server.serve_forever()
PY
server_pid=$!
for _ in {1..100}; do [ -f "$TEST_DIR/port" ] && break; sleep 0.05; done
[ -f "$TEST_DIR/port" ] || fail "HTTP fixture did not start"
url="http://127.0.0.1:$(cat "$TEST_DIR/port")"

wait_for_dashboard_ready fixture_dashboard "$url/ok" > "$TEST_DIR/result"
DASHBOARD_READY_TIMEOUT_SECONDS=5 MOCK_DELAYED=true wait_for_dashboard_ready fixture_dashboard "$url/ok" > "$TEST_DIR/result"

for update in paused rollback_started rollback_completed; do
  if MOCK_UPDATE="$update" wait_for_dashboard_ready fixture_dashboard "$url/ok" > "$TEST_DIR/result" 2>&1; then fail "$update was accepted"; fi
done
if MOCK_REPLICAS=0 wait_for_dashboard_ready fixture_dashboard "$url/ok" > "$TEST_DIR/result" 2>&1; then fail "zero replicas were considered ready"; fi
if DASHBOARD_READY_TIMEOUT_SECONDS=invalid wait_for_dashboard_ready fixture_dashboard "$url/ok" > "$TEST_DIR/result" 2>&1; then fail "invalid deadline was accepted"; fi

export DASHBOARD_READY_TIMEOUT_SECONDS=1
if MOCK_REPLICAS=2 wait_for_dashboard_ready fixture_dashboard "$url/ok" > "$TEST_DIR/result" 2>&1; then fail "missing replicas were accepted"; fi
if MOCK_TASKS='Running 5 seconds ago|registry.example.test/dashboard:old' wait_for_dashboard_ready fixture_dashboard "$url/ok" > "$TEST_DIR/result" 2>&1; then fail "old image was accepted"; fi
if MOCK_UPDATE=updating wait_for_dashboard_ready fixture_dashboard "$url/ok" > "$TEST_DIR/result" 2>&1; then fail "unfinished rollout was accepted"; fi
if MOCK_DOCKER_FAIL=true wait_for_dashboard_ready fixture_dashboard "$url/ok" > "$TEST_DIR/result" 2>&1; then fail "Docker failure was accepted"; fi
started=$SECONDS
if MOCK_DOCKER_HANG=true wait_for_dashboard_ready fixture_dashboard "$url/ok" > "$TEST_DIR/result" 2>&1; then fail "Docker request was not bounded"; fi
(( SECONDS - started < 4 )) || fail "Docker timeout left a child holding the request open"
for route in bad redirect hang; do
  if wait_for_dashboard_ready fixture_dashboard "$url/$route" > "$TEST_DIR/result" 2>&1; then fail "HTTP $route was accepted"; fi
done
printf 'Dashboard readiness tests passed\n'
