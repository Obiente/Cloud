# Docker Swarm Deployment

This guide explains how to deploy Obiente Cloud using Docker Swarm for distributed, production-ready deployments.

## Overview

Docker Swarm deployment provides:

- ✅ Distributed architecture across multiple nodes
- ✅ Automatic load balancing
- ✅ Service discovery
- ✅ Rolling updates with zero downtime
- ✅ Simple high availability setup

## Prerequisites

- Docker Engine 20.10+ with Swarm mode
- At least 1 manager node (recommended: 3 managers for HA)
- Worker nodes as needed for load distribution
- Domain name with DNS pointing to your nodes
- SSL certificates (auto-handled by Traefik)

## Initial Setup

### 1. Initialize Docker Swarm

On your first manager node:

```bash
docker swarm init --advertise-addr <MANAGER-IP>
```

To get the join token for additional managers:

```bash
docker swarm join-token manager
```

To get the join token for workers:

```bash
docker swarm join-token worker
```

### 2. Join Additional Nodes

On additional manager nodes:

```bash
docker swarm join --token <MANAGER-TOKEN> <MANAGER-IP>:2377
```

On worker nodes:

```bash
docker swarm join --token <WORKER-TOKEN> <MANAGER-IP>:2377
```

### 3. Create Environment File

Copy the example environment file:

```bash
cp .env.example .env
```

Edit `.env` with your configuration:

- Update database passwords
- Configure Zitadel authentication
- Set your domain name
- Generate secrets for JWT and sessions

### 4. Prepare All Nodes (Required)

The API service runs on **ALL nodes** (mode: global). You **must** create required directories on each node before deploying:

**On Manager Node:**
```bash
./scripts/setup-swarm-nodes.sh
```

**On Each Worker Node:**
```bash
# SSH to worker node
ssh worker-node-1

# Copy and run the setup script
scp scripts/setup-swarm-nodes.sh worker-node-1:/tmp/
ssh worker-node-1 "bash /tmp/setup-swarm-nodes.sh"

# Or manually create directories
mkdir -p /var/lib/obiente/volumes
mkdir -p /var/obiente/tmp/obiente-volumes
mkdir -p /var/obiente/tmp/obiente-deployments
chmod 755 /var/lib/obiente /var/obiente /var/obiente/tmp /var/obiente/tmp/obiente-volumes /var/obiente/tmp/obiente-deployments
```

**Alternative: Run on all nodes via SSH:**
```bash
# If you have SSH access to all nodes
for node in manager-1 worker-1 worker-2; do
  echo "Setting up $node..."
  ssh $node "mkdir -p /var/lib/obiente/volumes /var/obiente/tmp/obiente-volumes /var/obiente/tmp/obiente-deployments && chmod 755 /var/lib/obiente /var/obiente /var/obiente/tmp /var/obiente/tmp/obiente-volumes /var/obiente/tmp/obiente-deployments"
done
```

### 4.5. Configure Node Labels (Required for Non-HA)

For non-HA deployments, you must label nodes to specify where database services should run. This gives you control over which nodes host PostgreSQL, TimescaleDB, and Redis.

**Label a node for PostgreSQL:**
```bash
# Find your node name or ID
docker node ls

# Label a node to run PostgreSQL
docker node update --label-add postgres.enabled=true <node-name-or-id>
```

**Label a node for TimescaleDB (metrics database):**
```bash
docker node update --label-add metrics.enabled=true <node-name-or-id>
```

**Label a node for Redis:**
```bash
docker node update --label-add redis.enabled=true <node-name-or-id>
```

**Example: Label a single node for all databases:**
```bash
# Get node name
NODE_NAME=$(docker node ls --format "{{.Hostname}}" | head -n 1)

# Label the node for all database services
docker node update --label-add postgres.enabled=true $NODE_NAME
docker node update --label-add metrics.enabled=true $NODE_NAME
docker node update --label-add redis.enabled=true $NODE_NAME
```

**Verify labels:**
```bash
docker node inspect <node-name-or-id> --pretty
```

**Note:** You can use the same node for all database services, or distribute them across different nodes. The labels allow you to control placement precisely.

### 5. Deploy the Stack

Deploy to the Swarm cluster:

```bash
docker stack deploy -c docker-compose.swarm.yml obiente
```

Or use the deploy script (recommended):

```bash
./scripts/deploy-swarm.sh
```

## Development Deployment

For development environments, use the development deployment script which uses existing images by default (use `-b` to build locally):

### Quick Start (Development)

```bash
# Deploy using existing images (default - no build/pull)
./scripts/deploy-swarm-dev.sh

# Build images locally and deploy
./scripts/deploy-swarm-dev.sh -b

# Or specify custom stack name
./scripts/deploy-swarm-dev.sh my-dev-stack

# Or use custom compose file
./scripts/deploy-swarm-dev.sh obiente-dev docker-compose.swarm.dev.yml
```

### Build Options

The development script supports building images locally or pulling from registry:

```bash
# Build images locally (default)
./scripts/deploy-swarm-dev.sh -b
# or
./scripts/deploy-swarm-dev.sh --build

# Pull images from registry instead
./scripts/deploy-swarm-dev.sh -p
# or
./scripts/deploy-swarm-dev.sh --pull
```

### Development vs Production

| Feature | Development Script | Production Script |
|---------|-------------------|-------------------|
| Default compose file | `docker-compose.swarm.dev.yml` | `docker-compose.swarm.yml` |
| Default stack name | `obiente-dev` | `obiente` |
| Image building | Uses existing images by default (use `-b` to build) | Pulls from registry by default |
| Dashboard | Not included | Included by default |
| Registry auth | Optional | Required |

### Development Workflow

1. **Make code changes** in your local repository

2. **Build and deploy**:
   ```bash
   ./scripts/deploy-swarm-dev.sh -b
   ```

3. **View logs**:
   ```bash
   docker service logs -f obiente-dev_api-gateway
   ```

4. **Update after changes**:
   ```bash
   # Rebuild and redeploy
   ./scripts/deploy-swarm-dev.sh -b
   ```

### Development Stack Management

```bash
# View services
docker stack services obiente-dev

# View logs
docker service logs -f obiente-dev_api-gateway

# Remove stack
docker stack rm obiente-dev

# List tasks
docker stack ps obiente-dev
```

**Note**: The development script uses `docker-compose.swarm.dev.yml` which includes build configurations for all microservices. Images are built locally and tagged with the registry prefix (e.g., `ghcr.io/obiente/cloud-api-gateway:latest`).

### 6. Verify Deployment

Check service status:

```bash
docker service ls
```

Expected output:

```
ID             NAME                MODE         REPLICAS   IMAGE
abc123         obiente_api        global       3/3        obiente/cloud-api:latest
def456         obiente_postgres      replicated   1/1        postgres:16-alpine
ghi789         obiente_timescaledb  replicated   1/1        timescale/timescaledb:latest-pg16
jkl012         obiente_redis         replicated   1/1        redis:8-alpine
mno345         obiente_traefik       global       3/3        traefik:v2.11
```

## Service Configuration

API domain

- The API is exposed via Traefik at `api.${DOMAIN}` (from your `.env`).

### API Service

The Go API runs in **global mode**, meaning one instance per node:

```bash
# Check API replicas per node
docker service ps obiente_api

# View logs
docker service logs obiente_api
```

### Database Configuration

**PostgreSQL** runs as a single replica with health checks:

```bash
# Check database status
docker service ps obiente_postgres

# Check logs
docker service logs obiente_postgres
```

**TimescaleDB** (metrics database) runs as a separate service:

```bash
# Check metrics database status
docker service ps obiente_timescaledb

# Check logs
docker service logs obiente_timescaledb
```

### Scaling Services

Scale services as needed:

```bash
# Scale a specific service
docker service scale obiente_postgres=2

# Note: API runs in global mode, so scaling requires adding more nodes
```

## Service Updates

### Update Services

To update after code changes:

```bash
# Redeploy the stack
docker stack deploy -c docker-compose.swarm.yml obiente

# Force immediate update
docker service update --force obiente_api
```

### Rolling Updates

Docker Swarm handles rolling updates automatically:

- Starts new containers before stopping old ones
- Waits for health checks to pass
- Gradually replaces instances

## Health Checks

All services include health checks:

- **PostgreSQL**: `pg_isready`
- **TimescaleDB**: `pg_isready` (metrics database)
- **Redis**: `redis-cli ping`
- **Go API**: HTTP endpoint at `/health` (includes metrics system health)

Check health status:

```bash
docker service ps obiente_api
```

## Resource Limits

Default resource allocations:

| Service     | CPU | Memory |
| ----------- | --- | ------ |
| PostgreSQL  | 2.0 | 2GB    |
| TimescaleDB | 2.0 | 2GB    |
| Redis       | 0.5 | 256MB  |
| Go API      | 2.0 | 1GB    |
| Traefik     | 1.0 | 256MB  |

Adjust these in `docker-compose.swarm.yml` based on your cluster.

## Networking

All services communicate over the `obiente-network` overlay network, enabling:

- Service discovery by name (e.g., `postgres`, `redis`)
- Automatic load balancing across replicas
- Secure communication between services

## Volumes and Persistence

Data persists through Docker volumes:

- `postgres_data`: PostgreSQL database data
- `timescaledb_data`: TimescaleDB metrics data
- `redis_data`: Redis persistence
- `traefik_letsencrypt`: SSL certificates

Volumes are created automatically and persist across restarts.

## Traefik Configuration

Traefik automatically:

- Discovers services in the Swarm
- Provides HTTPS via Let's Encrypt
- Load balances across replicas
- Exposes routes based on service labels

Configure custom routes by adding Traefik labels in the compose file.

## Security

1. **Change Default Passwords**: Update all passwords in `.env`
2. **Use Secrets**: For sensitive data, use Docker secrets
3. **Network Security**: Configure firewall rules
4. **SSL Certificates**: Let's Encrypt provides automatic HTTPS
5. **Resource Limits**: Prevent resource exhaustion

## Troubleshooting

### Service Won't Start

```bash
# Check service logs
docker service logs -f obiente_api

# Check service status
docker service ps obiente_api
```

### Database Connection Issues

```bash
# Verify PostgreSQL is healthy
docker service ps obiente_postgres
docker service logs obiente_postgres

# Test connection from within network
docker exec -it $(docker ps -qf name=postgres) psql -U obiente-postgres -c "\l"
```

### Health Check Failures

Ensure health endpoints respond:

- API: `GET /health`

Test manually:

```bash
curl http://<node-ip>:3001/health
```

### High Memory Usage

Monitor resource usage:

```bash
docker stats
```

Adjust resource limits in `docker-compose.swarm.yml`.

## Backup and Recovery

### Backing Up PostgreSQL

```bash
# Create backup
docker exec $(docker ps -qf name=postgres) \
  pg_dump -U obiente-postgres obiente > backup_$(date +%Y%m%d).sql

# Restore backup
docker exec -i $(docker ps -qf name=postgres) \
  psql -U obiente-postgres obiente < backup_20240101.sql
```

### Backing Up Redis

```bash
# Create backup
docker exec $(docker ps -qf name=redis) redis-cli BGSAVE
docker cp $(docker ps -qf name=redis):/data/dump.rdb ./redis-backup.rdb
```

## Maintenance

### Zero-Downtime Updates

Updates are zero-downtime by default:

```bash
docker stack deploy -c docker-compose.swarm.yml obiente
```

### Graceful Shutdown

All services handle shutdown gracefully:

- API: HTTP server with proper timeouts
- Databases: PostgreSQL and Redis handle connections properly

## Production Checklist

Before deploying to production:

- [ ] Change all default passwords
- [ ] Configure proper CORS origins
- [ ] Set up SSL certificates
- [ ] Configure authentication (Zitadel)
- [ ] Set up monitoring
- [ ] Configure backups
- Rolling updates tested
- [ ] Document recovery procedures
- [ ] Set up log aggregation

## Next Steps

After deployment:

1. [Configure Authentication](../guides/authentication.md)
2. [Set up Monitoring](../guides/monitoring.md)
3. [Configure Custom Domains](../guides/routing.md)

---

[← Back to Deployment Guide](index.md)
