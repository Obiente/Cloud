#!/bin/bash
# Script to check Docker Swarm network connectivity and DNS resolution
# Usage: ./scripts/check-swarm-network.sh [stack-name]

set -e

STACK_NAME="${1:-obiente}"
NETWORK_NAME="${STACK_NAME}_obiente-network"

echo "=== Docker Swarm Network Diagnostic ==="
echo "Stack: $STACK_NAME"
echo "Network: $NETWORK_NAME"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "1. Checking network exists:"
if docker network ls | grep -q "$NETWORK_NAME"; then
  echo -e "${GREEN}✅ Network $NETWORK_NAME exists${NC}"
else
  echo -e "${RED}❌ Network $NETWORK_NAME not found!${NC}"
  exit 1
fi
echo ""

echo "2. Network details:"
docker network inspect "$NETWORK_NAME" --format 'Driver: {{.Driver}}, Scope: {{.Scope}}, Attachable: {{.Attachable}}' 2>/dev/null || echo "Cannot inspect network"
echo ""

echo "3. Checking required ports (overlay network):"
echo "   UDP 4789: Overlay network traffic"
echo "   TCP/UDP 7946: Node discovery"
echo "   Run on each node: sudo ufw status | grep -E '4789|7946'"
echo ""

echo "4. Services on network:"
docker network inspect "$NETWORK_NAME" --format '{{range .Services}}{{.Name}} (VIP: {{.VIP}}){{"\n"}}{{end}}' 2>/dev/null | head -10 || echo "Cannot list services"
echo ""

echo "5. Testing DNS resolution from a test container:"
TEST_CONTAINER=$(docker run -d --rm --network "$NETWORK_NAME" alpine sleep 60 2>/dev/null || echo "")
if [ -n "$TEST_CONTAINER" ]; then
  echo "   Testing postgres resolution:"
  docker exec "$TEST_CONTAINER" nslookup postgres 2>&1 | head -5 || echo "   ❌ DNS test failed"
  echo ""
  echo "   Testing superadmin-service resolution:"
  docker exec "$TEST_CONTAINER" nslookup superadmin-service 2>&1 | head -5 || echo "   ❌ DNS test failed"
  echo ""
  echo "   Checking /etc/resolv.conf:"
  docker exec "$TEST_CONTAINER" cat /etc/resolv.conf 2>&1 || echo "   ❌ Cannot read resolv.conf"
  docker rm -f "$TEST_CONTAINER" >/dev/null 2>&1 || true
else
  echo "   ⚠️  Cannot create test container"
fi
echo ""

echo "6. Checking postgres service:"
POSTGRES_SERVICE="${STACK_NAME}_postgres"
if docker service ls | grep -q "$POSTGRES_SERVICE"; then
  echo -e "${GREEN}✅ Postgres service exists${NC}"
  echo "   Service VIP:"
  docker service inspect "$POSTGRES_SERVICE" --format '{{range .Endpoint.VirtualIPs}}{{.Addr}}{{end}}' 2>/dev/null || echo "   Cannot get VIP"
  echo "   Service networks:"
  docker service inspect "$POSTGRES_SERVICE" --format '{{range .Spec.TaskTemplate.Networks}}{{.Target}}{{end}}' 2>/dev/null || echo "   Cannot get networks"
else
  echo -e "${RED}❌ Postgres service not found!${NC}"
fi
echo ""

echo "7. Checking service health:"
echo "   Postgres tasks:"
docker service ps "$POSTGRES_SERVICE" --format '{{.Name}}: {{.CurrentState}}' --filter 'desired-state=running' 2>/dev/null | head -3 || echo "   Cannot check tasks"
echo ""

echo "8. Network connectivity test (ping postgres VIP):"
if [ -n "$TEST_CONTAINER" ]; then
  POSTGRES_VIP=$(docker service inspect "$POSTGRES_SERVICE" --format '{{range .Endpoint.VirtualIPs}}{{.Addr}}{{end}}' 2>/dev/null | cut -d'/' -f1)
  if [ -n "$POSTGRES_VIP" ]; then
    docker exec "$TEST_CONTAINER" ping -c 2 "$POSTGRES_VIP" 2>&1 | head -5 || echo "   ❌ Ping failed"
  fi
fi
echo ""

echo "=== Diagnostic Complete ==="
echo ""
echo "If DNS resolution fails, check:"
echo "  1. Firewall rules (UDP 4789, TCP/UDP 7946)"
echo "  2. Network initialization on worker nodes"
echo "  3. Service health and readiness"
echo "  4. Docker daemon logs: journalctl -u docker.service"

