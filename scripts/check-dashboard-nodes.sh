#!/bin/bash
# Check why dashboard replicas aren't deploying on multiple nodes
# Run this on a Docker Swarm manager node

set -e

STACK_NAME="${STACK_NAME:-obiente}"

# Find dashboard service
DASHBOARD_SERVICE=$(docker service ls --format "{{.Name}}" | grep -E "^${STACK_NAME}_dashboard$" | head -n 1)

if [ -z "$DASHBOARD_SERVICE" ]; then
  echo "❌ Dashboard service not found: ${STACK_NAME}_dashboard"
  exit 1
fi

echo "🔍 Analyzing Dashboard Deployment"
echo "Dashboard service: $DASHBOARD_SERVICE"
echo ""

# Check service configuration
echo "📋 Service Configuration:"
docker service inspect "$DASHBOARD_SERVICE" --format '{{json .Spec}}' | jq '{
  replicas: .Mode.Replicated.Replicas,
  placement: .TaskTemplate.Placement,
  constraints: .TaskTemplate.Placement.Constraints,
  max_replicas_per_node: .TaskTemplate.Placement.MaxReplicas
}'
echo ""

# Check available nodes
echo "🌐 Available Nodes:"
docker node ls --format "table {{.Hostname}}\t{{.Status}}\t{{.Availability}}\t{{.Role}}"
echo ""

# Check which nodes have tasks
echo "📍 Dashboard Tasks by Node:"
docker service ps "$DASHBOARD_SERVICE" --format "{{.Node}}\t{{.DesiredState}}\t{{.CurrentState}}" | sort | uniq -c
echo ""

# Check node resources
echo "💻 Node Resources:"
for NODE in $(docker node ls --format "{{.Hostname}}"); do
  echo "Node: $NODE"
  docker node inspect "$NODE" --format '{{json .Status}}' | jq '{cpu: .Capacity.CPU, memory: .Capacity.Memory, availability: .Availability}' 2>/dev/null || echo "  Could not inspect node"
  echo ""
done

# Check if there are enough nodes
NODE_COUNT=$(docker node ls --format "{{.Hostname}}" | wc -l)
REPLICAS=$(docker service inspect "$DASHBOARD_SERVICE" --format '{{.Spec.Mode.Replicated.Replicas}}')

echo "📊 Summary:"
echo "  Total nodes: $NODE_COUNT"
echo "  Desired replicas: $REPLICAS"
echo "  Max replicas per node: $(docker service inspect "$DASHBOARD_SERVICE" --format '{{.Spec.TaskTemplate.Placement.MaxReplicas}}')"
echo ""

if [ "$NODE_COUNT" -lt "$REPLICAS" ]; then
  echo "⚠️  Warning: You have fewer nodes ($NODE_COUNT) than desired replicas ($REPLICAS)"
  echo "   Dashboard will only deploy $NODE_COUNT replica(s)"
fi

