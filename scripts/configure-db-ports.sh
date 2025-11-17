#!/bin/bash
# Script to configure database port exposure for Netbird VPN access
# This script modifies docker-compose.swarm.yml to expose PostgreSQL ports

set -e

COMPOSE_FILE="${1:-docker-compose.swarm.yml}"
POSTGRES_BIND="${POSTGRES_PORT_BIND:-}"
METRICS_BIND="${METRICS_DB_PORT_BIND:-}"

if [ ! -f "$COMPOSE_FILE" ]; then
  echo "❌ Error: $COMPOSE_FILE not found"
  exit 1
fi

echo "🔧 Configuring database port exposure in $COMPOSE_FILE"
echo ""

# Parse POSTGRES_PORT_BIND format: HOST_IP:HOST_PORT:CONTAINER_PORT
if [ -n "$POSTGRES_BIND" ]; then
  IFS=':' read -r HOST_IP HOST_PORT CONTAINER_PORT <<< "$POSTGRES_BIND"
  if [ -z "$HOST_IP" ] || [ -z "$HOST_PORT" ] || [ -z "$CONTAINER_PORT" ]; then
    echo "❌ Error: POSTGRES_PORT_BIND format invalid. Expected: HOST_IP:HOST_PORT:CONTAINER_PORT"
    echo "   Example: POSTGRES_PORT_BIND=127.0.0.1:5432:5432"
    exit 1
  fi
  
  echo "📋 Configuring PostgreSQL port binding: $POSTGRES_BIND"
  echo "   Host IP: $HOST_IP"
  echo "   Host Port: $HOST_PORT"
  echo "   Container Port: $CONTAINER_PORT"
  
  # Check if ports section already exists for postgres
  if grep -q "^\s*ports:" "$COMPOSE_FILE" -A 5 | grep -q "postgres" -B 2; then
    echo "   ⚠️  Ports section already exists for postgres, skipping..."
  else
    # Add ports section after volumes in postgres service
    # This is a simplified approach - for production, consider using sed or a template
    echo "   ✅ Use the following configuration in docker-compose.swarm.yml:"
    echo ""
    echo "   Add to postgres service:"
    echo "   ports:"
    echo "     - target: $CONTAINER_PORT"
    echo "       published: $HOST_PORT"
    echo "       protocol: tcp"
    echo "       mode: host"
    echo ""
    echo "   Note: For specific IP binding ($HOST_IP), configure firewall rules on the host"
    echo "   or use iptables to forward traffic from $HOST_IP:$HOST_PORT to 0.0.0.0:$HOST_PORT"
  fi
fi

# Parse METRICS_DB_PORT_BIND format: HOST_IP:HOST_PORT:CONTAINER_PORT
if [ -n "$METRICS_BIND" ]; then
  IFS=':' read -r HOST_IP HOST_PORT CONTAINER_PORT <<< "$METRICS_BIND"
  if [ -z "$HOST_IP" ] || [ -z "$HOST_PORT" ] || [ -z "$CONTAINER_PORT" ]; then
    echo "❌ Error: METRICS_DB_PORT_BIND format invalid. Expected: HOST_IP:HOST_PORT:CONTAINER_PORT"
    echo "   Example: METRICS_DB_PORT_BIND=127.0.0.1:5433:5432"
    exit 1
  fi
  
  echo "📋 Configuring TimescaleDB port binding: $METRICS_BIND"
  echo "   Host IP: $HOST_IP"
  echo "   Host Port: $HOST_PORT"
  echo "   Container Port: $CONTAINER_PORT"
  
  echo "   ✅ Use the following configuration in docker-compose.swarm.yml:"
  echo ""
  echo "   Add to timescaledb service:"
  echo "   ports:"
  echo "     - target: $CONTAINER_PORT"
  echo "       published: $HOST_PORT"
  echo "       protocol: tcp"
  echo "       mode: host"
  echo ""
fi

if [ -z "$POSTGRES_BIND" ] && [ -z "$METRICS_BIND" ]; then
  echo "ℹ️  No port bindings configured"
  echo "   Set POSTGRES_PORT_BIND and/or METRICS_DB_PORT_BIND environment variables"
  echo "   Example: POSTGRES_PORT_BIND=127.0.0.1:5432:5432"
fi

echo ""
echo "✅ Configuration complete!"
echo ""
echo "📝 Next steps:"
echo "   1. Uncomment and configure the ports sections in docker-compose.swarm.yml"
echo "   2. Update pg_hba.conf to allow connections from Netbird VPN subnet"
echo "   3. Restart the database services: docker service update --force <service-name>"

