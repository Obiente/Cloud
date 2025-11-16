#!/bin/bash
# Create Docker Swarm network for Obiente Cloud
# Creates the overlay network that connects all Obiente services
# Run on a Docker Swarm manager node
# Usage: ./scripts/create-swarm-network.sh [--stack-name <name>] [--subnet <subnet>]

set -e

STACK_NAME="${STACK_NAME:-obiente}"
NETWORK_NAME="${STACK_NAME}_obiente-network"
SUBNET="${SUBNET:-}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --stack-name)
      STACK_NAME="$2"
      NETWORK_NAME="${STACK_NAME}_obiente-network"
      shift 2
      ;;
    --subnet)
      SUBNET="$2"
      shift 2
      ;;
    *)
      echo -e "${RED}❌ Unknown option: $1${NC}"
      echo "Usage: ./scripts/create-swarm-network.sh [--stack-name <name>] [--subnet <subnet>]"
      exit 1
      ;;
  esac
done

echo -e "${BLUE}🌐 Creating Docker Swarm Network for Obiente Cloud${NC}"
echo ""
echo -e "${BLUE}  Stack name: ${STACK_NAME}${NC}"
echo -e "${BLUE}  Network name: ${NETWORK_NAME}${NC}"
echo ""

# Check if we're on a manager node
if ! docker node ls &>/dev/null; then
  echo -e "${RED}❌ Error: This script must be run on a Docker Swarm manager node${NC}"
  echo ""
  echo "To initialize Swarm on this node:"
  echo "  docker swarm init"
  echo ""
  exit 1
fi

# Check if network already exists
if docker network inspect "$NETWORK_NAME" &>/dev/null; then
  echo -e "${YELLOW}⚠️  Network already exists: ${NETWORK_NAME}${NC}"
  echo ""
  
  # Show network details
  echo -e "${BLUE}📊 Current Network Details:${NC}"
  docker network inspect "$NETWORK_NAME" --format '{{json .}}' | jq '{
    name: .Name,
    driver: .Driver,
    scope: .Scope,
    subnet: .IPAM.Config[0].Subnet,
    gateway: .IPAM.Config[0].Gateway,
    ip_pool: .IPAM.Config[0].IPRange
  }' 2>/dev/null || docker network inspect "$NETWORK_NAME"
  
  echo ""
  echo -e "${GREEN}✅ Network is ready to use${NC}"
  echo ""
  echo -e "${BLUE}💡 To recreate the network:${NC}"
  echo "  1. Remove existing network: docker network rm ${NETWORK_NAME}"
  echo "  2. Run this script again: ./scripts/create-swarm-network.sh"
  echo ""
  exit 0
fi

# Create the network
echo -e "${BLUE}📦 Creating overlay network...${NC}"

NETWORK_CREATE_CMD="docker network create --driver overlay --attachable"
if [ -n "$SUBNET" ]; then
  echo -e "${BLUE}  Using custom subnet: ${SUBNET}${NC}"
  NETWORK_CREATE_CMD="$NETWORK_CREATE_CMD --subnet $SUBNET"
fi

NETWORK_CREATE_CMD="$NETWORK_CREATE_CMD $NETWORK_NAME"

if $NETWORK_CREATE_CMD; then
  echo -e "${GREEN}✅ Network created successfully: ${NETWORK_NAME}${NC}"
else
  echo -e "${RED}❌ Failed to create network${NC}"
  exit 1
fi

echo ""

# Show network details
echo -e "${BLUE}📊 Network Details:${NC}"
docker network inspect "$NETWORK_NAME" --format '{{json .}}' | jq '{
  name: .Name,
  driver: .Driver,
  scope: .Scope,
  subnet: .IPAM.Config[0].Subnet,
  gateway: .IPAM.Config[0].Gateway,
  ip_pool: .IPAM.Config[0].IPRange
}' 2>/dev/null || docker network inspect "$NETWORK_NAME"

echo ""
echo -e "${GREEN}✅ Network is ready for use${NC}"
echo ""
echo -e "${BLUE}📋 Next steps:${NC}"
echo "  1. Deploy the main stack:"
echo "     ./scripts/deploy-swarm.sh"
echo ""
echo "  2. Or use the network in docker-compose files:"
echo "     ./scripts/deploy-swarm.sh ${STACK_NAME} docker-compose.swarm.yml"
echo ""
echo -e "${BLUE}💡 Useful commands:${NC}"
echo "  View network:        docker network inspect ${NETWORK_NAME}"
echo "  List all networks:   docker network ls"
echo "  Remove network:     docker network rm ${NETWORK_NAME}"
echo ""

