#!/bin/bash
# Diagnostic script for "server misbehaving" DNS errors in Docker Swarm

STACK_NAME="${STACK_NAME:-obiente}"

echo "=========================================="
echo "DNS 'Server Misbehaving' Diagnostics"
echo "=========================================="
echo ""

echo "1. Check if services are running:"
echo ""
docker service ls --format "table {{.Name}}\t{{.Replicas}}\t{{.Image}}" | grep -E "NAME|api-gateway|auth-service|superadmin-service"
echo ""

echo "2. Check network connectivity:"
echo ""
NETWORK_NAME="${STACK_NAME}_obiente-network"
echo "Network: $NETWORK_NAME"
docker network inspect "$NETWORK_NAME" --format 'Name: {{.Name}}
Driver: {{.Driver}}
Scope: {{.Scope}}
Containers: {{len .Containers}}
Services: {{len .Services}}' 2>&1
echo ""

echo "3. Check if services are on the same network:"
echo ""
echo "API Gateway network:"
docker service inspect "${STACK_NAME}_api-gateway" --format '{{range .Endpoint.VirtualIPs}}{{.NetworkID}} {{.Addr}}{{end}}' 2>&1
echo ""
echo "Auth Service network:"
docker service inspect "${STACK_NAME}_auth-service" --format '{{range .Endpoint.VirtualIPs}}{{.NetworkID}} {{.Addr}}{{end}}' 2>&1
echo ""

echo "4. Test DNS resolution from a container:"
echo ""
API_CONTAINER=$(docker ps --filter 'name=obiente_api-gateway' --format '{{.ID}}' | head -1)
if [ -n "$API_CONTAINER" ]; then
  echo "Testing from api-gateway container ($API_CONTAINER):"
  echo ""
  echo "nslookup auth-service:"
  docker exec "$API_CONTAINER" nslookup auth-service 127.0.0.11 2>&1 || echo "nslookup failed"
  echo ""
  echo "ping auth-service (to get IP):"
  docker exec "$API_CONTAINER" ping -c 1 auth-service 2>&1 | head -3 || echo "ping failed"
  echo ""
  echo "/etc/resolv.conf:"
  docker exec "$API_CONTAINER" cat /etc/resolv.conf 2>&1
else
  echo "No api-gateway container found (might be on worker node)"
fi
echo ""

echo "5. Check overlay network ports (required for DNS across nodes):"
echo ""
echo "Checking if overlay network ports are open:"
echo "  UDP 4789: Overlay network traffic"
echo "  TCP/UDP 7946: Node discovery"
echo ""
echo "Run on each node to check:"
echo "  sudo netstat -tuln | grep -E '4789|7946'"
echo ""

echo "6. Check Docker Swarm node status:"
echo ""
docker node ls 2>&1
echo ""

echo "7. Check for network issues:"
echo ""
echo "Recent network events:"
docker events --since 5m --filter type=network 2>&1 | tail -10 || echo "No recent network events"
echo ""

echo "=========================================="
echo "Potential Fixes:"
echo "=========================================="
echo ""
echo "1. Restart services to re-register in DNS:"
echo "   docker service update --force ${STACK_NAME}_api-gateway"
echo "   docker service update --force ${STACK_NAME}_auth-service"
echo ""
echo "2. Check firewall rules on all nodes:"
echo "   sudo ufw status"
echo "   sudo iptables -L -n | grep -E '4789|7946'"
echo ""
echo "3. Restart Docker daemon on problematic nodes:"
echo "   sudo systemctl restart docker"
echo "   (This will restart all containers on that node)"
echo ""
echo "4. Redeploy the stack:"
echo "   ./scripts/deploy-swarm.sh"
echo ""

