# Domain-Based Routing Guide

This guide explains how Obiente Cloud uses domain-based routing to enable service-to-service communication across nodes, networks, and clusters.

## Overview

Domain-based routing allows Obiente Cloud services to communicate using fully qualified domain names (FQDNs) instead of internal service names. This enables:

- **Cross-node communication**: Services on different nodes can communicate
- **Cross-network communication**: Works with VPNs, service meshes, and custom networks
- **Multi-cluster support**: Multiple Swarm clusters can share the same domain
- **Load balancing**: Traefik automatically load balances across all healthy replicas

## Architecture

### Service Communication Modes

Obiente Cloud supports two service communication modes:

1. **Internal Routing** (legacy): Direct service-to-service communication using Docker service names
   - Uses HTTP: `http://auth-service:3002`
   - Only works within the same Docker network
   - Faster, no TLS overhead
   - Limited to single network deployments

2. **Domain-Based Routing** (default): Communication via Traefik using HTTPS
   - Uses HTTPS: `https://auth-service.${DOMAIN}`
   - Works across nodes, networks, and clusters
   - Provides TLS termination and load balancing
   - Better for distributed deployments

### How It Works

```
┌─────────────────────────────────────────────────────────────┐
│                    Service A (Node 1)                       │
│  Makes request to: https://auth-service.obiente.cloud       │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                    Traefik (Load Balancer)                  │
│  - Discovers services via Docker labels                    │
│  - Routes to healthy replicas across all nodes               │
│  - Provides TLS termination                                  │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│              Service B (Node 2 or Node 3)                   │
│              auth-service:3002                               │
└─────────────────────────────────────────────────────────────┘
```

## Configuration

### Basic Setup

Domain-based routing is enabled by default. Configure the following environment variables:

```bash
# Required: Your domain
DOMAIN=obiente.cloud

# Enable Traefik routing (default: true)
USE_TRAEFIK_ROUTING=true

# Enable domain routing (default: true)
USE_DOMAIN_ROUTING=true

# Skip TLS verification for internal certificates (if needed)
SKIP_TLS_VERIFY=true
```

### Node-Specific Domains

For advanced scenarios where you need to route to specific nodes, configure node-specific domains via node labels in the database (using the Superadmin dashboard):

**Node Label Configuration:**

- **`obiente.subdomain`**: Node subdomain identifier (e.g., `node1`, `us-east-1`)
- **`obiente.use_node_specific_domains`**: Boolean to enable node-specific domains (`true` or `false`)
- **`obiente.service_domain_pattern`**: Domain pattern (`node-service` or `service-node`)

**Domain Patterns:**

- **`node-service` (default)**: `node1-auth-service.obiente.cloud`
  - Node identifier comes first
  - Example: `node1-auth-service.obiente.cloud`, `us-east-1-billing-service.obiente.cloud`

- **`service-node`**: `auth-service.node1.obiente.cloud`
  - Service name comes first, node identifier is a subdomain
  - Example: `auth-service.node1.obiente.cloud`, `billing-service.us-east-1.obiente.cloud`

### Node Subdomain Detection

Node subdomain is automatically extracted from:

1. Node labels in database (`obiente.subdomain` or `subdomain`)
2. Node hostname (fallback, sanitized for DNS)

## Load Balancing

### Shared Domains (Default)

When node-specific domains are not enabled (default), all nodes register services with the same domain:

```
All nodes register: auth-service.obiente.cloud
                    ↓
Traefik load balances across all healthy replicas
                    ↓
Node 1, Node 2, Node 3 (round-robin)
```

**Benefits:**
- Automatic load balancing across all nodes
- High availability (if one node fails, others continue serving)
- Simple configuration

**Use Cases:**
- Standard multi-node deployments
- High availability requirements
- Automatic failover

### Node-Specific Domains

When node-specific domains are enabled via node labels, each node registers services with its own domain:

```
Node 1: node1-auth-service.obiente.cloud
Node 2: node2-auth-service.obiente.cloud
Node 3: node3-auth-service.obiente.cloud
```

**Benefits:**
- Direct node routing (API Gateway can target specific nodes)
- Node isolation
- Custom routing logic

**Use Cases:**
- Geographic routing (route to nearest node)
- Node-specific workloads
- A/B testing across nodes

### API Gateway and Dashboard

**Important**: API Gateway and Dashboard **always** use shared domains for load balancing:

- **API Gateway**: Always uses `api.${DOMAIN}` (e.g., `api.obiente.cloud`)
- **Dashboard**: Always uses `${DOMAIN}` (e.g., `obiente.cloud`)

This ensures proper load balancing across all nodes/clusters, regardless of node-specific domain settings.

## Multi-Cluster Deployments

Domain-based routing enables multiple Swarm clusters to share the same domain:

```
Cluster 1 (US East):
  - api.obiente.cloud → Load balanced across Cluster 1 nodes
  - auth-service.obiente.cloud → Load balanced across Cluster 1 nodes

Cluster 2 (EU West):
  - api.obiente.cloud → Load balanced across Cluster 2 nodes
  - auth-service.obiente.cloud → Load balanced across Cluster 2 nodes

External DNS/LB:
  - api.obiente.cloud → Load balanced across Cluster 1 + Cluster 2
```

**Requirements:**

1. **External DNS/Load Balancer**: Configure DNS to point to all cluster Traefik IPs
2. **Shared Domain**: All clusters use the same `DOMAIN` value
3. **Traefik Discovery**: Traefik automatically discovers services via Docker labels

**How Traefik Discovers Services:**

Traefik uses Docker labels for service discovery:

```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.auth-service.rule=Host(`auth-service.obiente.cloud`)"
  - "traefik.http.services.auth-service.loadbalancer.server.port=3002"
```

The orchestrator service automatically syncs these labels every 30 seconds.

## Networking Configurations

### Docker Swarm (Default)

Works out of the box with Docker Swarm overlay networks:

```bash
# Services communicate via Traefik
USE_TRAEFIK_ROUTING=true
USE_DOMAIN_ROUTING=true
DOMAIN=obiente.cloud
```

### VPN Networks (Netbird, Tailscale, etc.)

Works with VPN networks when services are accessible via domain:

```bash
# Services communicate via Traefik over VPN
USE_TRAEFIK_ROUTING=true
USE_DOMAIN_ROUTING=true
DOMAIN=obiente.cloud

# Database/Redis can use VPN domains
DB_HOST=postgres.example.netbird
REDIS_HOST=redis.example.netbird
```

### Custom Networks

Works with any network configuration where services are accessible via domain:

```bash
# Services communicate via Traefik
USE_TRAEFIK_ROUTING=true
USE_DOMAIN_ROUTING=true
DOMAIN=obiente.cloud

# Custom database/Redis hosts
DB_HOST=db.example.com
REDIS_HOST=redis.example.com
```

## Traefik Label Synchronization

The orchestrator service automatically synchronizes Traefik labels for all microservices every 30 seconds. This ensures:

- Labels are always up-to-date
- Node-specific domains are correctly configured
- Load balancing works across all nodes

**Supported Services:**

- `api-gateway` (always uses shared domain: `api.${DOMAIN}`)
- `auth-service`
- `organizations-service`
- `billing-service`
- `deployments-service`
- `gameservers-service`
- `orchestrator-service`
- `vps-service`
- `support-service`
- `audit-service`
- `superadmin-service`
- `dns-service`

## Troubleshooting

### Services Not Communicating

1. **Check Traefik routing is enabled:**
   ```bash
   docker service inspect obiente_api-gateway --format '{{range .Spec.TaskTemplate.ContainerSpec.Env}}{{println .}}{{end}}' | grep USE_TRAEFIK_ROUTING
   ```

2. **Verify Traefik labels are set:**
   ```bash
   docker service inspect obiente_auth-service --format '{{range .Spec.Labels}}{{println .}}{{end}}' | grep traefik
   ```

3. **Check domain resolution:**
   ```bash
   docker exec $(docker ps -q -f name=obiente_api-gateway) nslookup auth-service.obiente.cloud
   ```

### Load Balancing Not Working

1. **Verify all nodes register with same domain** (for shared domains):
   ```bash
   # Check labels on all nodes
   docker service inspect obiente_auth-service --format '{{range .Spec.Labels}}{{println .}}{{end}}'
   ```

2. **Check Traefik service discovery:**
   ```bash
   # Access Traefik dashboard
   curl http://localhost:8080/api/http/routers
   ```

3. **Verify health checks:**
   ```bash
   # Check service health
   docker service ps obiente_auth-service
   ```

### Node-Specific Domains Not Working

1. **Check node labels in database:**
   ```sql
   SELECT id, hostname, labels FROM node_metadata WHERE id = 'your-node-id';
   ```
   Verify that the labels contain:
   - `obiente.subdomain`: Node subdomain identifier
   - `obiente.use_node_specific_domains`: Set to `true`
   - `obiente.service_domain_pattern`: Either `node-service` or `service-node`

2. **Verify configuration via Superadmin dashboard:**
   - Navigate to Superadmin → Nodes
   - Click "Configure" on the node
   - Check that "Use Node-Specific Domains" is enabled
   - Verify the subdomain and domain pattern are set correctly

3. **Check Traefik labels on services:**
   ```bash
   docker service inspect obiente_auth-service --format '{{range .Spec.Labels}}{{println .}}{{end}}' | grep traefik
   ```
   Verify that the domain includes the node subdomain (e.g., `node1-auth-service.obiente.cloud`)

## Best Practices

1. **Use shared domains by default** for automatic load balancing
2. **Enable node-specific domains only when needed** for direct node routing (configure via Superadmin dashboard)
3. **Set `SKIP_TLS_VERIFY=true`** for internal Traefik certificates
4. **Monitor Traefik dashboard** to verify service discovery
5. **Use health checks** to ensure only healthy services receive traffic
6. **Configure external DNS/LB** for multi-cluster deployments
7. **Configure node labels via Superadmin dashboard** instead of environment variables for node-specific settings

## Related Documentation

- [Environment Variables Reference](../reference/environment-variables.md#api-gateway-routing-configuration) - Complete variable reference
- [Architecture Overview](../architecture/overview.md) - System architecture
- [Docker Swarm Deployment](../deployment/docker-swarm.md) - Deployment guide

---

[← Back to Guides](index.md)

