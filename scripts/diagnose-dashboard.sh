#!/bin/bash
# Diagnostic script for dashboard deployment issues
# Run this on a Docker Swarm manager node

set -e

STACK_NAME="${STACK_NAME:-obiente}"
DASHBOARD_SERVICE="${STACK_NAME}_dashboard"

echo "🔍 Dashboard Deployment Diagnostics"
echo ""

# Check service status
echo "1. Service Status:"
docker service ps "$DASHBOARD_SERVICE" --no-trunc --format "table {{.ID}}\t{{.Name}}\t{{.Node}}\t{{.DesiredState}}\t{{.CurrentState}}\t{{.Error}}"
echo ""

# Check service configuration
echo "2. Service Configuration:"
docker service inspect "$DASHBOARD_SERVICE" --format '{{json .Spec}}' | jq '{
  replicas: .Mode.Replicated.Replicas,
  healthcheck: .TaskTemplate.ContainerSpec.Healthcheck,
  resources: .TaskTemplate.Resources,
  restart_policy: .TaskTemplate.RestartPolicy,
  placement: {
    constraints: .TaskTemplate.Placement.Constraints,
    max_replicas_per_node: .TaskTemplate.Placement.MaxReplicas
  }
}'
echo ""

# Check recent logs for errors
echo "3. Recent Logs (last 50 lines):"
docker service logs "$DASHBOARD_SERVICE" --tail 50 2>&1 | tail -20
echo ""

# Check if containers are actually running
echo "4. Container Status:"
for TASK_ID in $(docker service ps "$DASHBOARD_SERVICE" --filter "desired-state=running" --format "{{.ID}}" | head -2); do
  echo "Task: $TASK_ID"
  docker inspect "$TASK_ID" --format '{{json .Status}}' | jq '{state: .State, health: .Health}' 2>/dev/null || echo "  Could not inspect task"
done
echo ""

# Check network connectivity
echo "5. Network Configuration:"
docker service inspect "$DASHBOARD_SERVICE" --format '{{json .Spec.TaskTemplate.Networks}}' | jq '.'
echo ""

# Check resource usage
echo "6. Available Resources on Nodes:"
docker node ls --format "{{.Hostname}}" | while read NODE; do
  echo "Node: $NODE"
  docker node inspect "$NODE" --format '{{json .Description.Resources}}' | jq '{cpus: .NanoCPUs, memory: .MemoryBytes}' 2>/dev/null || echo "  Could not inspect"
done
echo ""

# Check if health check is passing
echo "7. Health Check Status:"
docker service ps "$DASHBOARD_SERVICE" --filter "desired-state=running" --format "{{.ID}}" | head -1 | while read TASK_ID; do
  if [ -n "$TASK_ID" ]; then
    CONTAINER_ID=$(docker inspect "$TASK_ID" --format '{{.Status.ContainerStatus.ContainerID}}' 2>/dev/null)
    if [ -n "$CONTAINER_ID" ]; then
      echo "Container: $CONTAINER_ID"
      docker inspect "$CONTAINER_ID" --format '{{json .State.Health}}' | jq '.' 2>/dev/null || echo "  No health check configured or container not found"
    fi
  fi
done
echo ""

echo "💡 Common Issues to Check:"
echo "  - Health check failing (wget not available in container)"
echo "  - Resource constraints (CPU/Memory limits too low)"
echo "  - Network connectivity issues"
echo "  - Container crashing after startup"
echo "  - Port conflicts"
echo ""

