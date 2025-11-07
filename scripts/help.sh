#!/usr/bin/env bash
# Obiente Cloud - Deployment Scripts Usage Guide
# This script provides help and usage information for all deployment scripts
# Usage: ./scripts/help.sh [script-name] or ./scripts/help.sh

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

SCRIPT_NAME="${1:-}"

# Print header
print_header() {
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "${CYAN}   Obiente Cloud - Deployment Scripts Usage Guide${NC}"
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo ""
}

# Print script info
print_script() {
  local name=$1
  local category=$2
  local description=$3
  local usage=$4
  local when=$5
  
  echo -e "${BLUE}📄 ${name}${NC}"
  echo -e "   ${YELLOW}Category:${NC} ${category}"
  echo -e "   ${YELLOW}Description:${NC} ${description}"
  echo -e "   ${YELLOW}Usage:${NC} ${usage}"
  if [ -n "$when" ]; then
    echo -e "   ${YELLOW}When to use:${NC} ${when}"
  fi
  echo ""
}

# Show overview
show_overview() {
  print_header
  
  echo -e "${GREEN}📚 Quick Start${NC}"
  echo ""
  echo "1. Setup all nodes (run on each worker node):"
  echo "   ./scripts/setup-all-nodes.sh"
  echo ""
  echo "2. Deploy the stack:"
  echo "   ./scripts/deploy-swarm.sh"
  echo ""
  echo "3. Check deployment status:"
  echo "   docker stack services obiente"
  echo ""
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo ""
  
  echo -e "${GREEN}🚀 Deployment Scripts${NC}"
  echo ""
  print_script \
    "deploy-swarm.sh" \
    "Deployment" \
    "Main deployment script - deploys both main stack and dashboard" \
    "./scripts/deploy-swarm.sh [stack-name] [compose-file]" \
    "Initial deployment or updates"
  
  print_script \
    "build-swarm.sh" \
    "Deployment" \
    "Build Docker images locally before deployment" \
    "./scripts/build-swarm.sh" \
    "When building images locally instead of pulling from registry"
  
  print_script \
    "force-deploy.sh" \
    "Deployment" \
    "Force update all services to ensure latest configuration" \
    "./scripts/force-deploy.sh [stack-name] [compose-file]" \
    "After config changes or when services aren't updating"
  
  print_script \
    "rollback-swarm.sh" \
    "Deployment" \
    "Rollback services to their previous version" \
    "./scripts/rollback-swarm.sh [stack-name] [service-name]" \
    "When a deployment causes issues and you need to revert"
  
  print_script \
    "redeploy-dashboard.sh" \
    "Deployment" \
    "Quick dashboard-only redeployment" \
    "./scripts/redeploy-dashboard.sh [domain]" \
    "Quick dashboard updates without full redeploy"
  
  print_script \
    "calculate-replicas.sh" \
    "Configuration" \
    "Calculate dashboard replica counts based on cluster size" \
    "./scripts/calculate-replicas.sh" \
    "When configuring dashboard replicas for your cluster size"
  
  echo -e "${GREEN}🔧 Setup & Maintenance${NC}"
  echo ""
  print_script \
    "setup-all-nodes.sh" \
    "Setup" \
    "Create required directories on Swarm nodes" \
    "./scripts/setup-all-nodes.sh" \
    "Run on each worker node before first deployment"
  
  print_script \
    "create-swarm-network.sh" \
    "Setup" \
    "Create Docker Swarm overlay network for Obiente Cloud" \
    "./scripts/create-swarm-network.sh [--subnet <subnet>]" \
    "Create network before deployment (optional - stacks will create it automatically)"
  
  print_script \
    "cleanup-swarm.sh" \
    "Maintenance" \
    "Clean up old Obiente containers, tasks, and unused Obiente resources (SAFE: only targets Obiente resources)" \
    "./scripts/cleanup-swarm.sh [--dry-run] [--all-nodes]" \
    "Regular maintenance, use --dry-run first to preview"
  
  print_script \
    "cleanup-swarm-complete.sh" \
    "Maintenance" \
    "Complete Obiente cleanup - removes ALL Obiente stacks, services, volumes, networks (SAFE: only targets Obiente resources)" \
    "./scripts/cleanup-swarm-complete.sh [--confirm]" \
    "Fresh start - WARNING: Deletes all Obiente data including volumes!"
  
  echo -e "${GREEN}🔍 Diagnostic Scripts${NC}"
  echo ""
  print_script \
    "diagnose-dashboard.sh" \
    "Diagnostic" \
    "Comprehensive dashboard deployment diagnostics" \
    "./scripts/diagnose-dashboard.sh" \
    "When dashboard isn't working - comprehensive check"
  
  print_script \
    "check-dashboard-deployment.sh" \
    "Diagnostic" \
    "Check dashboard deployment across nodes" \
    "./scripts/check-dashboard-deployment.sh" \
    "Verify dashboard replicas are distributed correctly"
  
  print_script \
    "check-dashboard-nodes.sh" \
    "Diagnostic" \
    "Check why dashboard replicas aren't deploying on multiple nodes" \
    "./scripts/check-dashboard-nodes.sh" \
    "When dashboard isn't deploying on all manager nodes"
  
  print_script \
    "check-dashboard-traefik.sh" \
    "Diagnostic" \
    "Check Traefik discovery configuration for dashboard" \
    "./scripts/check-dashboard-traefik.sh" \
    "When Traefik isn't discovering the dashboard"
  
  print_script \
    "check-dashboard-routing.sh" \
    "Diagnostic" \
    "Check dashboard routing configuration" \
    "./scripts/check-dashboard-routing.sh [stack-name]" \
    "When dashboard isn't accessible via Traefik"
  
  echo -e "${GREEN}🔧 Troubleshooting Scripts${NC}"
  echo ""
  print_script \
    "fix-swarm-networks.sh" \
    "Troubleshooting" \
    "Diagnose and fix Docker Swarm network IP pool conflicts" \
    "./scripts/fix-swarm-networks.sh [--fix]" \
    "When seeing 'Pool overlaps with other one' errors"
  
  print_script \
    "fix-network-pool-conflict.sh" \
    "Troubleshooting" \
    "Fix network IP pool conflicts by removing conflicting networks and recreating with specific subnet" \
    "./scripts/fix-network-pool-conflict.sh [--subnet <subnet>] [--force]" \
    "When getting 'Pool overlaps' errors - use --force to apply fixes"
  
  print_script \
    "fix-dashboard-network.sh" \
    "Troubleshooting" \
    "Fix dashboard service network configuration" \
    "./scripts/fix-dashboard-network.sh" \
    "When dashboard is on wrong network or has network issues"
  
  echo ""
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo ""
  
  echo -e "${GREEN}💡 Common Workflows${NC}"
  echo ""
  echo -e "${BLUE}First-time deployment:${NC}"
  echo "  1. ./scripts/setup-all-nodes.sh  # On each worker node"
  echo "  2. ./scripts/deploy-swarm.sh"
  echo ""
  echo -e "${BLUE}Update deployment:${NC}"
  echo "  1. ./scripts/deploy-swarm.sh     # Normal update"
  echo "  2. ./scripts/force-deploy.sh     # If updates aren't applying"
  echo ""
  echo -e "${BLUE}Rollback deployment:${NC}"
  echo "  1. ./scripts/rollback-swarm.sh                    # Rollback all services"
  echo "  2. ./scripts/rollback-swarm.sh obiente api        # Rollback specific service"
  echo ""
  echo -e "${BLUE}Dashboard issues:${NC}"
  echo "  1. ./scripts/diagnose-dashboard.sh"
  echo "  2. ./scripts/check-dashboard-traefik.sh"
  echo "  3. ./scripts/fix-dashboard-network.sh"
  echo ""
  echo -e "${BLUE}Dashboard replica configuration:${NC}"
  echo "  1. ./scripts/calculate-replicas.sh  # Calculate values for your cluster"
  echo "  2. Add output to .env file"
  echo "  3. Redeploy dashboard"
  echo ""
  echo -e "${BLUE}Network conflicts:${NC}"
  echo "  1. ./scripts/fix-swarm-networks.sh"
  echo "  2. ./scripts/fix-swarm-networks.sh --fix"
  echo ""
  echo -e "${BLUE}Cleanup and maintenance:${NC}"
  echo "  1. ./scripts/cleanup-swarm.sh --dry-run  # Preview"
  echo "  2. ./scripts/cleanup-swarm.sh             # Regular cleanup"
  echo "  3. ./scripts/cleanup-swarm-complete.sh --confirm  # Full reset"
  echo ""
  
  echo -e "${GREEN}📋 Environment Variables${NC}"
  echo ""
  echo "  STACK_NAME          - Stack name (default: obiente)"
  echo "  DOMAIN              - Domain for dashboard (default: obiente.cloud)"
  echo "  BUILD_LOCAL         - Build images locally (default: false)"
  echo "  DEPLOY_DASHBOARD    - Deploy dashboard stack (default: true)"
  echo "  API_IMAGE           - API image tag (default: ghcr.io/obiente/cloud-api:latest)"
  echo "  DASHBOARD_IMAGE     - Dashboard image tag (default: ghcr.io/obiente/cloud-dashboard:latest)"
  echo ""
  echo -e "${GREEN}📊 Dashboard Replica Configuration${NC}"
  echo ""
  echo "  DASHBOARD_REPLICAS            - Number of dashboard replicas (default: 5)"
  echo "  DASHBOARD_MAX_REPLICAS_PER_NODE - Max replicas per node (default: 2)"
  echo ""
  echo "  Use ./scripts/calculate-replicas.sh to calculate values based on cluster size"
  echo "  Ensure: DASHBOARD_REPLICAS <= (cluster_size * DASHBOARD_MAX_REPLICAS_PER_NODE)"
  echo ""
  
  echo -e "${YELLOW}💡 Tip:${NC} Run './scripts/help.sh <script-name>' for detailed info about a specific script"
  echo ""
}

# Show specific script details
show_script_details() {
  case "$SCRIPT_NAME" in
    deploy-swarm.sh|deploy)
      print_header
      echo -e "${BLUE}deploy-swarm.sh${NC} - Main deployment script"
      echo ""
      echo -e "${YELLOW}Description:${NC}"
      echo "  Deploys the Obiente Cloud stack to Docker Swarm. Handles both"
      echo "  the main stack and dashboard stack. Can pull images from registry"
      echo "  or build locally."
      echo ""
      echo -e "${YELLOW}Usage:${NC}"
      echo "  ./scripts/deploy-swarm.sh [stack-name] [compose-file]"
      echo ""
      echo -e "${YELLOW}Options:${NC}"
      echo "  stack-name      - Stack name (default: obiente)"
      echo "  compose-file    - Compose file (default: docker-compose.swarm.yml)"
      echo ""
      echo -e "${YELLOW}Environment Variables:${NC}"
      echo "  BUILD_LOCAL=true        - Build images locally instead of pulling"
      echo "  DEPLOY_DASHBOARD=false  - Skip dashboard deployment"
      echo "  DOMAIN                  - Dashboard domain (default: obiente.cloud)"
      echo ""
      echo -e "${YELLOW}Examples:${NC}"
      echo "  ./scripts/deploy-swarm.sh"
      echo "  BUILD_LOCAL=true ./scripts/deploy-swarm.sh"
      echo "  DEPLOY_DASHBOARD=false ./scripts/deploy-swarm.sh"
      ;;
    
    rollback-swarm.sh|rollback)
      print_header
      echo -e "${BLUE}rollback-swarm.sh${NC} - Rollback services to previous version"
      echo ""
      echo -e "${YELLOW}Description:${NC}"
      echo "  Rolls back Docker Swarm services to their previous version. Can rollback"
      echo "  all services in a stack or a specific service. Uses Docker Swarm's built-in"
      echo "  rollback functionality to revert to the previous service configuration."
      echo ""
      echo -e "${YELLOW}Usage:${NC}"
      echo "  ./scripts/rollback-swarm.sh [stack-name] [service-name]"
      echo ""
      echo -e "${YELLOW}Options:${NC}"
      echo "  stack-name      - Stack name (default: obiente)"
      echo "  service-name    - Optional: specific service to rollback (e.g., api, postgres)"
      echo "                    If omitted, rolls back all services in the stack"
      echo ""
      echo -e "${YELLOW}How it works:${NC}"
      echo "  1. Lists all services in the specified stack"
      echo "  2. For each service, uses 'docker service rollback' to revert to previous version"
      echo "  3. Shows service status after rollback"
      echo ""
      echo -e "${YELLOW}Examples:${NC}"
      echo "  ./scripts/rollback-swarm.sh                    # Rollback all services in 'obiente' stack"
      echo "  ./scripts/rollback-swarm.sh obiente            # Rollback all services in 'obiente' stack"
      echo "  ./scripts/rollback-swarm.sh obiente api        # Rollback only the 'api' service"
      echo "  ./scripts/rollback-swarm.sh obiente postgres   # Rollback only the 'postgres' service"
      echo ""
      echo -e "${YELLOW}Notes:${NC}"
      echo "  - Rollback only works if services have a previous version"
      echo "  - Services are rolled back one at a time"
      echo "  - Rollback is confirmed before execution (unless service-name is specified)"
      echo "  - Use 'docker service ps <service>' to check rollback status"
      echo ""
      ;;
    
    cleanup-swarm-complete.sh|cleanup-complete)
      print_header
      echo -e "${RED}cleanup-swarm-complete.sh${NC} - Complete cleanup (DESTRUCTIVE)"
      echo ""
      echo -e "${YELLOW}Description:${NC}"
      echo "  ${RED}WARNING: This removes ALL stacks, services, containers, volumes,"
      echo "  and networks. All data will be lost!${NC}"
      echo ""
      echo "  Completely removes all Docker Swarm resources for a fresh start."
      echo ""
      echo -e "${YELLOW}Usage:${NC}"
      echo "  ./scripts/cleanup-swarm-complete.sh [--confirm]"
      echo ""
      echo -e "${YELLOW}Options:${NC}"
      echo "  --confirm  - Skip confirmation prompt"
      echo ""
      echo -e "${YELLOW}What it removes:${NC}"
      echo "  - All stacks (obiente and obiente_dashboard)"
      echo "  - All services"
      echo "  - All containers"
      echo "  - All volumes (DATA LOSS!)"
      echo "  - All networks"
      echo "  - Unused images"
      echo ""
      echo -e "${RED}⚠️  Use with extreme caution!${NC}"
      ;;
    
    diagnose-dashboard.sh|diagnose)
      print_header
      echo -e "${BLUE}diagnose-dashboard.sh${NC} - Dashboard diagnostics"
      echo ""
      echo -e "${YELLOW}Description:${NC}"
      echo "  Comprehensive diagnostic tool for dashboard deployment issues."
      echo "  Checks service status, configuration, logs, network, and resources."
      echo ""
      echo -e "${YELLOW}Usage:${NC}"
      echo "  ./scripts/diagnose-dashboard.sh"
      echo ""
      echo -e "${YELLOW}What it checks:${NC}"
      echo "  - Service status and tasks"
      echo "  - Service configuration"
      echo "  - Recent logs"
      echo "  - Container health"
      echo "  - Network configuration"
      echo "  - Node resources"
      echo ""
      ;;
    
    calculate-replicas.sh|replicas)
      print_header
      echo -e "${BLUE}calculate-replicas.sh${NC} - Calculate dashboard replica configuration"
      echo ""
      echo -e "${YELLOW}Description:${NC}"
      echo "  Calculates appropriate dashboard replica counts and max_replicas_per_node"
      echo "  based on your Docker Swarm cluster size. Helps avoid 'max replicas per"
      echo "  node limit exceeded' errors."
      echo ""
      echo -e "${YELLOW}Usage:${NC}"
      echo "  ./scripts/calculate-replicas.sh"
      echo ""
      echo -e "${YELLOW}How it works:${NC}"
      echo "  1. Detects cluster size automatically (or prompts for manual input)"
      echo "  2. Calculates replicas based on percentage of nodes (default: 50%)"
      echo "  3. Calculates max_replicas_per_node for even distribution"
      echo "  4. Ensures minimum 2 replicas for HA"
      echo "  5. Outputs recommended .env configuration"
      echo ""
      echo -e "${YELLOW}Environment Variables:${NC}"
      echo "  DASHBOARD_REPLICAS_PERCENT      - Percentage of nodes for replicas (default: 50)"
      echo "  DASHBOARD_MAX_REPLICAS_PERCENT  - Max replicas per node percentage (default: 40)"
      echo ""
      echo -e "${YELLOW}Example Output:${NC}"
      echo "  Detected cluster size: 3 nodes"
      echo "  Recommended Configuration:"
      echo "    DASHBOARD_REPLICAS=2"
      echo "    DASHBOARD_MAX_REPLICAS_PER_NODE=1"
      echo ""
      echo -e "${YELLOW}Common Issues:${NC}"
      echo "  Error: 'max replicas per node limit exceeded'"
      echo "  Solution: Run this script and update your .env file with the recommended values"
      echo ""
      ;;
    
    *)
      echo -e "${RED}❌ Unknown script: ${SCRIPT_NAME}${NC}"
      echo ""
      echo "Available scripts:"
      echo "  - deploy-swarm.sh"
      echo "  - rollback-swarm.sh"
      echo "  - cleanup-swarm-complete.sh"
      echo "  - diagnose-dashboard.sh"
      echo "  - calculate-replicas.sh"
      echo ""
      echo "Run './scripts/help.sh' for full list"
      exit 1
      ;;
  esac
}

# Main
if [ -z "$SCRIPT_NAME" ]; then
  show_overview
else
  show_script_details
fi

