#!/bin/bash
# Rollback script for Obiente Cloud Docker Swarm
# Rolls back services to their previous version
# Usage: ./scripts/rollback-swarm.sh [stack-name] [service-name]

set -e

STACK_NAME="${1:-obiente}"
SERVICE_NAME="${2:-}"
DEPLOY_DASHBOARD="${DEPLOY_DASHBOARD:-true}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to get all services in a stack
get_stack_services() {
  local stack=$1
  docker stack services --format "{{.Name}}" "$stack" 2>/dev/null || echo ""
}

# Function to rollback a service
rollback_service() {
  local service=$1
  if docker service ls --format "{{.Name}}" | grep -q "^${service}$"; then
    echo -e "${YELLOW}  🔄 Rolling back service: ${service}${NC}"
    
    # Perform rollback and capture output
    local rollback_output
    rollback_output=$(docker service rollback "$service" 2>&1)
    local rollback_exit=$?
    
    if [ $rollback_exit -eq 0 ]; then
      echo -e "${GREEN}  ✅ ${service} rollback initiated${NC}"
      
      # Wait a moment and show status
      sleep 2
      echo -e "${BLUE}  📊 Current status:${NC}"
      docker service ps "$service" --no-trunc --format "table {{.Name}}\t{{.CurrentState}}\t{{.Image}}" | head -n 3
      return 0
    else
      # Check if rollback failed because there's no previous version
      if echo "$rollback_output" | grep -qi "no rollback\|does not have a previous version"; then
        echo -e "${YELLOW}  ⚠️  ${service}: No previous version to rollback to${NC}"
      else
        echo -e "${RED}  ❌ Failed to rollback ${service}${NC}"
        echo -e "${RED}     Error: ${rollback_output}${NC}"
      fi
      return 1
    fi
  else
    echo -e "${YELLOW}  ⚠️  Service ${service} not found, skipping...${NC}"
    return 1
  fi
}

# Print header
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}   Obiente Cloud - Docker Swarm Rollback${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Check if stack exists
if ! docker stack ls --format "{{.Name}}" | grep -q "^${STACK_NAME}$"; then
  echo -e "${RED}❌ Stack '${STACK_NAME}' not found!${NC}"
  echo ""
  echo "Available stacks:"
  docker stack ls --format "  - {{.Name}}"
  exit 1
fi

# Get all services in the stack
MAIN_SERVICES=$(get_stack_services "$STACK_NAME")

if [ -z "$MAIN_SERVICES" ]; then
  echo -e "${RED}❌ No services found in stack '${STACK_NAME}'${NC}"
  exit 1
fi

# If specific service is provided, rollback only that service
if [ -n "$SERVICE_NAME" ]; then
  # Check if service exists (might be in main stack or dashboard)
  FULL_SERVICE_NAME="${STACK_NAME}_${SERVICE_NAME}"
  
  if docker service ls --format "{{.Name}}" | grep -q "^${FULL_SERVICE_NAME}$"; then
    echo -e "${BLUE}🔄 Rolling back service: ${FULL_SERVICE_NAME}${NC}"
    echo ""
    rollback_service "$FULL_SERVICE_NAME"
    echo ""
    echo -e "${GREEN}✅ Rollback complete!${NC}"
    exit 0
  else
    echo -e "${RED}❌ Service '${FULL_SERVICE_NAME}' not found!${NC}"
    echo ""
    echo "Available services in stack '${STACK_NAME}':"
    echo "$MAIN_SERVICES" | sed 's/^/  - /'
    exit 1
  fi
fi

# Show current services
echo -e "${BLUE}📋 Services in stack '${STACK_NAME}':${NC}"
echo ""
echo "$MAIN_SERVICES" | while read -r service; do
  echo -e "  - ${service}"
done
echo ""

# Confirm rollback
echo -e "${YELLOW}⚠️  This will rollback ALL services in the stack to their previous version.${NC}"
echo ""
read -p "Continue with rollback? (y/N) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  echo "Rollback cancelled."
  exit 0
fi

echo ""
echo -e "${BLUE}🔄 Starting rollback...${NC}"
echo ""

# Rollback all services
ROLLBACK_FAILED=0
for service in $MAIN_SERVICES; do
  if ! rollback_service "$service"; then
    ROLLBACK_FAILED=1
  fi
  echo ""
done

# Summary
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
if [ $ROLLBACK_FAILED -eq 0 ]; then
  echo -e "${GREEN}✅ Rollback complete for all services!${NC}"
else
  echo -e "${YELLOW}⚠️  Rollback completed with some failures. Check the output above.${NC}"
fi
echo ""
echo -e "${BLUE}📋 Useful commands:${NC}"
echo "  View services:     docker stack services $STACK_NAME"
echo "  View logs:         docker service logs -f ${STACK_NAME}_api"
echo "  Check status:      docker service ps ${STACK_NAME}_api"
echo "  List tasks:        docker stack ps $STACK_NAME"
echo ""

