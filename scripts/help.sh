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
    "redeploy-dashboard.sh" \
    "Deployment" \
    "Quick dashboard-only redeployment" \
    "./scripts/redeploy-dashboard.sh [domain]" \
    "Quick dashboard updates without full redeploy"
  
  echo -e "${GREEN}🔧 Setup & Maintenance${NC}"
  echo ""
  print_script \
    "setup-all-nodes.sh" \
    "Setup" \
    "Create required directories on Swarm nodes" \
    "./scripts/setup-all-nodes.sh" \
    "Run on each worker node before first deployment"
  
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
  echo -e "${BLUE}Dashboard issues:${NC}"
  echo "  1. ./scripts/diagnose-dashboard.sh"
  echo "  2. ./scripts/check-dashboard-traefik.sh"
  echo "  3. ./scripts/fix-dashboard-network.sh"
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
    
    *)
      echo -e "${RED}❌ Unknown script: ${SCRIPT_NAME}${NC}"
      echo ""
      echo "Available scripts:"
      echo "  - deploy-swarm.sh"
      echo "  - cleanup-swarm-complete.sh"
      echo "  - diagnose-dashboard.sh"
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

