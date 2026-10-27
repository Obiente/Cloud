#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_DIR="$(mktemp -d)"
trap 'rm -rf "$TEST_DIR"' EXIT
mkdir -p "$TEST_DIR/bin" "$TEST_DIR/repo/scripts/lib"
cp "$SCRIPT_DIR"/{deploy-swarm.sh,deploy-swarm-dev.sh,build-swarm.sh} "$TEST_DIR/repo/scripts/"
cp "$SCRIPT_DIR"/lib/{common.sh,database-owner.sh} "$TEST_DIR/repo/scripts/lib/"

fail() { printf 'FAIL: %s\n' "$1" >&2; exit 1; }

cat > "$TEST_DIR/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
printf '%s\n' "$*" >> "$MOCK_DOCKER_LOG"
case "$1" in
  pull)
    [[ "$2" != *"${MOCK_FAIL_IMAGE:-no-image-fails}"* ]]
    ;;
  build) [[ "$*" != *"${MOCK_FAIL_BUILD:-no-build-fails}"* ]] ;;
  *) exit 91 ;;
esac
EOF
# Fail before any real host filesystem mutation, including on regression.
for command in mkdir chmod; do
  cat > "$TEST_DIR/bin/$command" <<'EOF'
#!/usr/bin/env bash
printf '%s %s\n' "${0##*/}" "$*" >> "$MOCK_MUTATION_LOG"
exit 92
EOF
done
chmod +x "$TEST_DIR/bin/"*
export MOCK_DOCKER_LOG="$TEST_DIR/docker.log"
export MOCK_MUTATION_LOG="$TEST_DIR/mutations.log"
export PATH="$TEST_DIR/bin:$PATH"

for script in deploy-swarm.sh deploy-swarm-dev.sh; do
  for failed_image in api-gateway gameservers-service vps-service; do
    : > "$MOCK_DOCKER_LOG"
    : > "$MOCK_MUTATION_LOG"
    if output="$(cd "$TEST_DIR/repo"; MOCK_FAIL_IMAGE="$failed_image" PULL_CONCURRENCY=4 \
      bash "scripts/$script" --pull < /dev/null 2>&1)"; then
      fail "$script accepted a failed pull of $failed_image"
    fi
    [[ "$output" == *"Deployment stopped before changing the stack"* ]] || fail "$script did not report the pull failure"
    [[ ! -s "$MOCK_MUTATION_LOG" ]] || fail "$script changed host directories before validating images"
    if grep -Ev '^pull ' "$MOCK_DOCKER_LOG"; then
      fail "$script called Docker mutation commands after a failed pull"
    fi
  done
done

# Registry/dashboard settings from .env must be applied before constructing tags.
printf 'REGISTRY=registry.example.test/team\nDASHBOARD_IMAGE=registry.example.test/ui:release\n' > "$TEST_DIR/repo/.env"
: > "$MOCK_DOCKER_LOG"
if (cd "$TEST_DIR/repo"; MOCK_FAIL_IMAGE=databases-service bash scripts/deploy-swarm.sh --pull < /dev/null > "$TEST_DIR/main.log" 2>&1); then
  fail "database image pull failure was ignored"
fi
grep -Fxq 'pull registry.example.test/team/cloud-databases-service:latest' "$MOCK_DOCKER_LOG" || fail "database service was omitted"
grep -Fxq 'pull registry.example.test/ui:release' "$MOCK_DOCKER_LOG" || fail "dashboard override was ignored"
if grep -q ghcr.io "$MOCK_DOCKER_LOG"; then fail "registry override was ignored"; fi

printf 'REGISTRY=registry.example.test/team\n' > "$TEST_DIR/repo/.env"
: > "$MOCK_DOCKER_LOG"
if (cd "$TEST_DIR/repo"; MOCK_FAIL_IMAGE=databases-service bash scripts/deploy-swarm.sh --pull < /dev/null > "$TEST_DIR/default-dashboard.log" 2>&1); then
  fail "database image pull failure was ignored"
fi
grep -Fxq 'pull registry.example.test/team/cloud-dashboard:latest' "$MOCK_DOCKER_LOG" || fail "default dashboard image used the wrong registry"

: > "$MOCK_DOCKER_LOG"
: > "$MOCK_MUTATION_LOG"
if (cd "$TEST_DIR/repo"; BUILD_LOCAL=true MOCK_FAIL_BUILD=databases-service bash scripts/deploy-swarm.sh < /dev/null > "$TEST_DIR/local-build.log" 2>&1); then
  fail "local database build failure was ignored"
fi
grep -Fxq 'build -f apps/databases-service/Dockerfile -t registry.example.test/team/cloud-databases-service:latest .' "$MOCK_DOCKER_LOG" || fail "local deployment built the wrong database image"
[[ ! -s "$MOCK_MUTATION_LOG" ]] || fail "local build failure changed host directories"
if grep -Ev '^build ' "$MOCK_DOCKER_LOG"; then fail "local build failure changed Docker services"; fi

# All repository Dockerfiles must be built, including the database control plane.
: > "$MOCK_DOCKER_LOG"
(cd "$TEST_DIR/repo"; REGISTRY=registry.example.test/team bash scripts/build-swarm.sh > "$TEST_DIR/build.log")
for dockerfile in "$SCRIPT_DIR"/../apps/*/Dockerfile; do
  service="$(basename "$(dirname "$dockerfile")")"
  grep -Fxq "build -f apps/$service/Dockerfile -t registry.example.test/team/cloud-$service:latest ." "$MOCK_DOCKER_LOG" || fail "build omitted or mistagged $service"
done

# Exercise failure collection both while filling worker slots and during the final wait.
# shellcheck source=scripts/lib/common.sh
source "$SCRIPT_DIR/lib/common.sh"
for concurrency in 1 4 99; do
  for failed_image in first middle last; do
    if MOCK_FAIL_IMAGE="$failed_image" PULL_CONCURRENCY="$concurrency" \
      require_pulled_images first=registry.example.test/first middle=registry.example.test/middle last=registry.example.test/last > "$TEST_DIR/pull.log" 2>&1; then
      fail "pull failure was lost (concurrency $concurrency, image $failed_image)"
    fi
    [[ " ${PARALLEL_PULL_FAILURES[*]} " == *" $failed_image "* ]] || fail "failed image was not reported"
  done
done
MOCK_FAIL_IMAGE=none require_pulled_images first=registry.example.test/first last=registry.example.test/last > "$TEST_DIR/pull.log" 2>&1
printf 'deployment image preparation tests passed\n'
