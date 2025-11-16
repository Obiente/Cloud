#!/bin/bash
# DNS Resolution Diagnostic Script for Docker Swarm
# Run this on a manager node to diagnose DNS resolution issues

set -e

STACK_NAME="${STACK_NAME:-obiente}"
NETWORK_NAME="${STACK_NAME}_obiente-network"

echo "=========================================="
echo "Docker Swarm DNS Resolution Diagnostics"
echo "=========================================="
echo ""
echo "Stack Name: $STACK_NAME"
echo "Network Name: $NETWORK_NAME"
echo ""
echo "=========================================="
echo "1. NETWORK STATUS"
echo "=========================================="
echo ""
echo "Checking if network exists:"
docker network ls | grep "$NETWORK_NAME" || echo "❌ Network not found!"
echo ""

echo "Network details:"
docker network inspect "$NETWORK_NAME" --format '{{json .}}' | jq -r '
  "Name: " + .Name,
  "Driver: " + .Driver,
  "Scope: " + .Scope,
  "Attachable: " + (.Attachable | tostring),
  "Internal: " + (.Internal | tostring),
  "IPAM Driver: " + .IPAM.Driver,
  "Subnets: " + ([.IPAM.Config[].Subnet] | join(", "))
' 2>/dev/null || docker network inspect "$NETWORK_NAME"
echo ""

echo "=========================================="
echo "2. SERVICE STATUS"
echo "=========================================="
echo ""
echo "All services in stack:"
docker stack services "$STACK_NAME" --format "table {{.Name}}\t{{.Replicas}}\t{{.Image}}"
echo ""

echo "Postgres service details:"
docker service inspect "${STACK_NAME}_postgres" --format '{{json .}}' | jq -r '
  "Service Name: " + .Spec.Name,
  "Network: " + (.Spec.TaskTemplate.Networks[0].Target // "none"),
  "Aliases: " + ([.Spec.TaskTemplate.Networks[0].Aliases[]?] | join(", ") // "none"),
  "VIP: " + (.Endpoint.VirtualIPs[0].Addr // "none")
' 2>/dev/null || docker service inspect "${STACK_NAME}_postgres"
echo ""

echo "API Gateway service details:"
docker service inspect "${STACK_NAME}_api-gateway" --format '{{json .}}' | jq -r '
  "Service Name: " + .Spec.Name,
  "Network: " + (.Spec.TaskTemplate.Networks[0].Target // "none"),
  "Aliases: " + ([.Spec.TaskTemplate.Networks[0].Aliases[]?] | join(", ") // "none"),
  "VIP: " + (.Endpoint.VirtualIPs[0].Addr // "none")
' 2>/dev/null || docker service inspect "${STACK_NAME}_api-gateway"
echo ""

echo "=========================================="
echo "3. DNS CONFIGURATION IN SERVICES"
echo "=========================================="
echo ""
echo "Postgres DNS config:"
docker service inspect "${STACK_NAME}_postgres" --format '{{json .Spec.TaskTemplate.ContainerSpec.DNSConfig}}' | jq '.' 2>/dev/null || echo "No DNS config"
echo ""

echo "API Gateway DNS config:"
docker service inspect "${STACK_NAME}_api-gateway" --format '{{json .Spec.TaskTemplate.ContainerSpec.DNSConfig}}' | jq '.' 2>/dev/null || echo "No DNS config"
echo ""

echo "Dashboard DNS config:"
docker service inspect "${STACK_NAME}_dashboard" --format '{{json .Spec.TaskTemplate.ContainerSpec.DNSConfig}}' | jq '.' 2>/dev/null || echo "No DNS config"
echo ""

echo "=========================================="
echo "4. DNS RESOLUTION FROM CONTAINERS"
echo "=========================================="
echo ""

# Get a running postgres task
POSTGRES_TASK=$(docker service ps "${STACK_NAME}_postgres" --filter "desired-state=running" --format "{{.ID}}" | head -1)
if [ -n "$POSTGRES_TASK" ]; then
  echo "Testing DNS resolution from postgres container:"
  docker exec "$POSTGRES_TASK" nslookup api-gateway 2>&1 || echo "nslookup failed"
  echo ""
fi

# Get a running api-gateway task
API_GATEWAY_TASK=$(docker service ps "${STACK_NAME}_api-gateway" --filter "desired-state=running" --format "{{.ID}}" | head -1)
if [ -n "$API_GATEWAY_TASK" ]; then
  echo "Testing DNS resolution from api-gateway container:"
  docker exec "$API_GATEWAY_TASK" nslookup postgres 2>&1 || echo "nslookup failed"
  echo ""
fi

# Get a running dashboard task
DASHBOARD_TASK=$(docker service ps "${STACK_NAME}_dashboard" --filter "desired-state=running" --format "{{.ID}}" | head -1)
if [ -n "$DASHBOARD_TASK" ]; then
  echo "Testing DNS resolution from dashboard container:"
  docker exec "$DASHBOARD_TASK" nslookup api-gateway 2>&1 || echo "nslookup failed"
  echo ""
  echo "Dashboard /etc/resolv.conf:"
  docker exec "$DASHBOARD_TASK" cat /etc/resolv.conf 2>&1 || echo "Failed to read resolv.conf"
  echo ""
fi

echo "=========================================="
echo "5. NETWORK CONNECTIVITY"
echo "=========================================="
echo ""

if [ -n "$API_GATEWAY_TASK" ]; then
  echo "Testing connectivity from api-gateway to postgres:"
  docker exec "$API_GATEWAY_TASK" ping -c 2 postgres 2>&1 || echo "ping failed"
  echo ""
fi

if [ -n "$DASHBOARD_TASK" ]; then
  echo "Testing connectivity from dashboard to api-gateway:"
  docker exec "$DASHBOARD_TASK" ping -c 2 api-gateway 2>&1 || echo "ping failed"
  echo ""
fi

echo "=========================================="
echo "6. SERVICE VIPs AND ALIASES"
echo "=========================================="
echo ""

echo "All services on network with VIPs:"
docker network inspect "$NETWORK_NAME" --format '{{range .Containers}}{{.Name}} - {{.IPv4Address}}{{"\n"}}{{end}}' 2>/dev/null || echo "No containers found"
echo ""

echo "Service endpoints:"
docker service ls --format "{{.Name}}" | while read service; do
  if [[ "$service" == "${STACK_NAME}_"* ]]; then
    echo "Service: $service"
    docker service inspect "$service" --format '  VIP: {{range .Endpoint.VirtualIPs}}{{.Addr}} {{end}}' 2>/dev/null || echo "  No VIP"
    docker service inspect "$service" --format '  Network Aliases: {{range .Spec.TaskTemplate.Networks}}{{range .Aliases}}{{.}} {{end}}{{end}}' 2>/dev/null || echo "  No aliases"
    echo ""
  fi
done

echo "=========================================="
echo "7. DOCKER EMBEDDED DNS STATUS"
echo "=========================================="
echo ""

if [ -n "$API_GATEWAY_TASK" ]; then
  echo "Testing Docker embedded DNS (127.0.0.11) from api-gateway:"
  docker exec "$API_GATEWAY_TASK" nslookup postgres 127.0.0.11 2>&1 || echo "Failed to query embedded DNS"
  echo ""
fi

echo "=========================================="
echo "8. HOST DNS CONFIGURATION"
echo "=========================================="
echo ""

echo "Host /etc/resolv.conf:"
cat /etc/resolv.conf 2>/dev/null || echo "Cannot read host resolv.conf"
echo ""

echo "Host search domain:"
grep -i "search" /etc/resolv.conf 2>/dev/null || echo "No search domain"
echo ""

echo "=========================================="
echo "9. RECENT SERVICE LOGS (DNS ERRORS)"
echo "=========================================="
echo ""

echo "Recent postgres logs (last 20 lines):"
docker service logs --tail 20 "${STACK_NAME}_postgres" 2>&1 | grep -i "dns\|resolve\|lookup" || echo "No DNS-related errors"
echo ""

echo "Recent api-gateway logs (last 20 lines):"
docker service logs --tail 20 "${STACK_NAME}_api-gateway" 2>&1 | grep -i "dns\|resolve\|lookup" || echo "No DNS-related errors"
echo ""

echo "Recent dashboard logs (last 20 lines):"
docker service logs --tail 20 "${STACK_NAME}_dashboard" 2>&1 | grep -i "dns\|resolve\|lookup\|api-gateway\|503\|timeout" || echo "No DNS-related errors"
echo ""

echo "=========================================="
echo "10. NETWORK NODES"
echo "=========================================="
echo ""

echo "Nodes in swarm:"
docker node ls --format "table {{.Hostname}}\t{{.Status}}\t{{.Availability}}\t{{.ManagerStatus}}"
echo ""

echo "Network on each node:"
docker node ls --format "{{.Hostname}}" | while read node; do
  echo "Node: $node"
  docker node inspect "$node" --format '  Networks: {{range $net, $config := .Spec.Labels}}{{$net}} {{end}}' 2>/dev/null || echo "  No network labels"
done
echo ""

echo "=========================================="
echo "Diagnostics Complete!"
echo "=========================================="

