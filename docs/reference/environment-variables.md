# Environment Variables Reference

Complete reference for all Obiente Cloud environment variables.

## Quick Reference

| Variable                             | Default                      | Required      | Description                                                                                                         |
| ------------------------------------ | ---------------------------- | ------------- | ------------------------------------------------------------------------------------------------------------------- |
| `POSTGRES_USER`                      | `obiente`                    | ❌            | PostgreSQL username                                                                                                 |
| `POSTGRES_PASSWORD`                  | -                            | ✅            | PostgreSQL password                                                                                                 |
| `ZITADEL_URL`                        | `https://auth.obiente.cloud` | ❌            | Zitadel instance URL                                                                                                |
| `LOG_LEVEL`                          | `info`                       | ❌            | Application logging level                                                                                           |
| `DB_LOG_LEVEL`                       | (uses LOG_LEVEL)             | ❌            | Database query logging level                                                                                        |
| `CORS_ORIGIN`                        | `*`                          | ❌            | Allowed CORS origins                                                                                                |
| `SMTP_HOST`                          | -                            | ❌            | SMTP server host (required to enable email)                                                                         |
| `SMTP_FROM_ADDRESS`                  | -                            | ❌            | From address used for outbound email                                                                                |
| `DASHBOARD_URL`                      | `https://obiente.cloud`      | ❌            | Dashboard URL used in invitation call-to-action and billing redirects                                               |
| `SUPPORT_EMAIL`                      | -                            | ❌            | Support contact displayed in email footers                                                                          |
| `SUPERADMIN_EMAILS`                  | -                            | ❌            | Comma-separated list of emails with global access (superadmins for self-hosted, The Obiente Cloud Team for managed) |
| `SELF_HOSTED`                        | `false`                      | ❌            | Set to `true` if this is a self-hosted deployment (affects terminology in UI/docs)                                  |
| `STRIPE_SECRET_KEY`                  | -                            | ✅ (billing)  | Stripe secret API key for payment processing                                                                        |
| `STRIPE_WEBHOOK_SECRET`              | -                            | ✅ (webhooks) | Stripe webhook signing secret for webhook verification                                                              |
| `NUXT_PUBLIC_STRIPE_PUBLISHABLE_KEY` | -                            | ✅ (frontend) | Stripe publishable key for client-side Stripe.js                                                                    |
| `USE_TRAEFIK_ROUTING`                | `true`                       | ❌            | Route API gateway requests via Traefik (HTTPS) instead of direct service-to-service (HTTP)                          |
| `USE_DOMAIN_ROUTING`                 | `true`                       | ❌            | Use domain-based routing for service-to-service communication (works across nodes/networks)                          |

## Configuration Sections

### Database Configuration

| Variable                 | Type   | Default    | Required     |
| ------------------------ | ------ | ---------- | ------------ |
| `POSTGRES_USER`          | string | `obiente`  | ❌           |
| `POSTGRES_PASSWORD`      | string | -          | ✅           |
| `POSTGRES_DB`            | string | `obiente`  | ❌           |
| `DB_HOST`                | string | `postgres` | ❌           |
| `DB_PORT`                | number | `5432`     | ❌           |
| `POSTGRES_EXPOSE_PORT`   | number | `5432`     | ❌           | Port to expose PostgreSQL on host (default: 5432, localhost only) |
| `POSTGRES_PORT_MODE`     | string | `host`     | ❌           | Port mode: `host` (default, for localhost binding) or `ingress` |
| `POSTGRES_ALLOWED_HOSTS` | string | -          | ❌           | Comma-separated IPs/subnets to allow in pg_hba.conf (e.g., "10.10.10.1,10.0.0.0/8") |
| `REPLICATION_PASSWORD`   | string | -          | ❌ (HA only) |
| `PATRONI_ADMIN_PASSWORD` | string | -          | ❌ (HA only) |

**Database Host Configuration (`DB_HOST`):**

The `DB_HOST` variable supports different networking configurations:

- **Docker Swarm**: Use the service name (default: `postgres`)
- **Netbird VPN**: Use the Netbird internal domain (e.g., `postgres.example.netbird`)
- **Custom**: Set to any hostname or IP address

**Examples:**

```bash
# Docker Swarm (default)
DB_HOST=postgres

# Netbird VPN
DB_HOST=postgres.example.netbird

# Custom hostname/IP
DB_HOST=db.example.com
DB_HOST=10.0.0.5

# Full example
POSTGRES_USER=obiente
POSTGRES_PASSWORD=secure_random_password_here
POSTGRES_DB=obiente
DB_HOST=postgres.example.netbird
```

**Database Port Exposure (`POSTGRES_EXPOSE_PORT`):**

PostgreSQL port is **exposed by default on localhost only** (127.0.0.1:5432) for security. This allows local access while preventing external connections.

**Default Configuration:**
- Port exposed: `5432` (configurable via `POSTGRES_EXPOSE_PORT`)
- Mode: `host` (for localhost binding)
- Binding: All interfaces (restrict via firewall for localhost-only)

**To restrict to localhost only**, configure firewall rules on the host:
```bash
# Using iptables (restrict PostgreSQL to localhost only)
sudo iptables -A INPUT -p tcp --dport 5432 ! -s 127.0.0.1 -j DROP

# Or using ufw (if installed)
sudo ufw deny 5432
sudo ufw allow from 127.0.0.1 to any port 5432
```

**Examples:**

```bash
# Default: Exposed on localhost only (requires firewall rules for true localhost-only)
POSTGRES_EXPOSE_PORT=5432
POSTGRES_PORT_MODE=host

# Expose on all interfaces (for Netbird VPN access)
POSTGRES_EXPOSE_PORT=5432
POSTGRES_PORT_MODE=host
# Then configure pg_hba.conf to allow Netbird VPN subnet

# Use ingress mode (Docker Swarm load balancing)
POSTGRES_EXPOSE_PORT=5432
POSTGRES_PORT_MODE=ingress
```

**Note:** The port is exposed by default. To disable, comment out the `ports:` section in `docker-compose.swarm.yml`.

**Database Allowed Hosts (`POSTGRES_ALLOWED_HOSTS`):**

Configure additional IP addresses or subnets that are allowed to connect to PostgreSQL. This adds entries to `pg_hba.conf` automatically.

**Format:** Comma-separated list of IP addresses or CIDR subnets

**Examples:**

```bash
# Allow specific IP address
POSTGRES_ALLOWED_HOSTS=10.10.10.1

# Allow multiple IPs
POSTGRES_ALLOWED_HOSTS=10.10.10.1,192.168.1.100

# Allow subnet (CIDR notation)
POSTGRES_ALLOWED_HOSTS=10.0.0.0/8

# Mix of IPs and subnets
POSTGRES_ALLOWED_HOSTS=10.10.10.1,10.0.0.0/8,192.168.1.0/24
```

**Note:** Single IPs are automatically converted to `/32` CIDR format. 

**To apply allowed hosts changes:**

1. **Update the environment variable** in your deployment:
   ```bash
   # Update the service with new allowed hosts
   docker service update --env-add POSTGRES_ALLOWED_HOSTS="10.10.10.1,10.0.0.0/8" obiente_postgres
   ```

2. **Run the manual update script** to apply changes to pg_hba.conf:
   ```bash
   # For postgres service
   ./scripts/update-postgres-hba.sh postgres
   
   # For timescaledb service
   ./scripts/update-postgres-hba.sh timescaledb
   ```

The script will:
- Read the `POSTGRES_ALLOWED_HOSTS` (or `METRICS_DB_ALLOWED_HOSTS` for timescaledb) from the service environment
- Add missing rules to `pg_hba.conf`
- Reload PostgreSQL configuration automatically

### API Configuration

| Variable       | Type   | Default | Required |
| -------------- | ------ | ------- | -------- |
| `GO_API_PORT`  | number | `3001`  | ❌       |
| `LOG_LEVEL`    | string | `info`  | ❌       |
| `DB_LOG_LEVEL` | string | -       | ❌       |

**Application Log Levels (`LOG_LEVEL`):**

- `debug` - Verbose logging for development
- `info` - Standard production logging
- `warn` - Only warnings and errors
- `error` - Only errors

**Database Log Levels (`DB_LOG_LEVEL`):**

Controls GORM database query logging. If not set, falls back to `LOG_LEVEL`.

- `debug` / `trace` - Show all SQL queries and parameters
- `info` - Show SQL queries only (no parameters)
- `warn` / `warning` - Only database errors (suppresses "record not found")
- `error` - Only database errors

**Examples:**

```bash
# Application logs at info, database logs at error (no SQL queries)
LOG_LEVEL=info
DB_LOG_LEVEL=error

# Both at debug for development
LOG_LEVEL=debug
DB_LOG_LEVEL=debug

# Application at warn, database at debug (useful for debugging slow queries)
LOG_LEVEL=warn
DB_LOG_LEVEL=debug
```

### Redis Configuration

| Variable            | Type   | Default | Required |
| ------------------- | ------ | ------- | -------- |
| `REDIS_HOST`        | string | `redis` | ❌       |
| `REDIS_PORT`        | number | `6379`  | ❌       |
| `REDIS_PASSWORD`    | string | -       | ❌       | Redis password (required if port is exposed) |
| `REDIS_EXPOSE_PORT` | number | `6379`  | ❌       | Port to expose Redis on host (default: 6379, localhost only) |
| `REDIS_PORT_MODE`   | string | `host`  | ❌       | Port mode: `host` (default, for localhost binding) or `ingress` |
| `REDIS_URL`         | string | -       | ❌       | Full Redis URL (constructed from REDIS_HOST, REDIS_PORT, and REDIS_PASSWORD if not set) |

**Redis Host Configuration (`REDIS_HOST`):**

The `REDIS_HOST` variable supports different networking configurations, similar to database configuration:

- **Docker Swarm**: Use the service name (default: `redis`)
- **Netbird VPN**: Use the Netbird internal domain (e.g., `redis.example.netbird`)
- **Custom**: Set to any hostname or IP address

**Examples:**

```bash
# Docker Swarm (default)
REDIS_HOST=redis

# Netbird VPN
REDIS_HOST=redis.example.netbird

# Custom hostname/IP
REDIS_HOST=redis.example.com
REDIS_HOST=10.0.0.5

# Full example
REDIS_HOST=redis.example.netbird
REDIS_PORT=6379
```

**Redis Password (`REDIS_PASSWORD`):**

**⚠️ Security Note:** If Redis port is exposed (via `REDIS_EXPOSE_PORT`), you **must** set a password to secure Redis. Without a password, Redis will be accessible to anyone who can reach the exposed port.

**Examples:**

```bash
# Set a strong password (required if port is exposed)
REDIS_PASSWORD=your_secure_random_password_here

# Generate a secure password
openssl rand -base64 32
```

**Redis URL (`REDIS_URL`):**

If `REDIS_URL` is not explicitly set, it is automatically constructed from `REDIS_HOST`, `REDIS_PORT`, and `REDIS_PASSWORD`:

```bash
# Automatically constructed (default, no password)
REDIS_HOST=redis
REDIS_PORT=6379
# Results in: redis://redis:6379

# Automatically constructed (with password)
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=my_secure_password
# Results in: redis://:my_secure_password@redis:6379

# Explicit URL (overrides REDIS_HOST, REDIS_PORT, and REDIS_PASSWORD)
REDIS_URL=redis://:password@redis.example.netbird:6379
```

**Redis Port Exposure (`REDIS_EXPOSE_PORT`):**

Redis port is **exposed by default on localhost only** (127.0.0.1:6379) for security. This allows local access while preventing external connections.

**Default Configuration:**
- Port exposed: `6379` (configurable via `REDIS_EXPOSE_PORT`)
- Mode: `host` (for localhost binding)
- Binding: All interfaces (restrict via firewall for localhost-only)

**To restrict to localhost only**, configure firewall rules on the host:
```bash
# Using iptables (restrict Redis to localhost only)
sudo iptables -A INPUT -p tcp --dport 6379 ! -s 127.0.0.1 -j DROP

# Or using ufw (if installed)
sudo ufw deny 6379
sudo ufw allow from 127.0.0.1 to any port 6379
```

**Examples:**

```bash
# Default: Exposed on localhost only (requires firewall rules for true localhost-only)
REDIS_EXPOSE_PORT=6379
REDIS_PORT_MODE=host

# Expose on all interfaces (for Netbird VPN access)
REDIS_EXPOSE_PORT=6379
REDIS_PORT_MODE=host
# Then configure firewall to allow Netbird VPN subnet

# Use ingress mode (Docker Swarm load balancing)
REDIS_EXPOSE_PORT=6379
REDIS_PORT_MODE=ingress
```

**Note:** The port is exposed by default. To disable, comment out the `ports:` section in `docker-compose.swarm.yml`.

**Note:** Redis is an internal service and typically communicates within the Docker Swarm overlay network or via VPN. Port exposure is optional and mainly for external access scenarios.

### API Gateway Routing Configuration

| Variable                    | Type    | Default | Required | Description                                                                                                                                    |
| --------------------------- | ------- | ------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `USE_TRAEFIK_ROUTING`       | boolean | `true`  | ❌       | Route API gateway requests via Traefik (HTTPS) instead of direct service-to-service (HTTP). Defaults to `true` for cross-node compatibility. |
| `USE_DOMAIN_ROUTING`        | boolean | `true`  | ❌       | Use domain-based routing for service-to-service communication. When `true`, services communicate via domains (works across nodes/networks).   |
| `SKIP_TLS_VERIFY`           | boolean | `false` | ❌       | Skip TLS certificate verification when using Traefik routing (for internal certs).                                                         |

**Service Routing Modes:**

The API Gateway can route to backend services in two ways:

1. **Internal Routing**: Direct service-to-service communication using Docker Swarm service names
   - Uses HTTP: `http://auth-service:3002`
   - Faster, no TLS overhead
   - Requires services to be on the same Docker network
   - Only works within a single Docker network

2. **Traefik Routing (default)**: Routes through Traefik reverse proxy with HTTPS
   - Uses HTTPS: `https://auth-service.${DOMAIN}`
   - All services must have Traefik labels configured
   - Provides TLS termination and load balancing
   - Works across nodes, networks, and clusters
   - Better for distributed deployments and VPN access

**Domain-Based Routing:**

When `USE_DOMAIN_ROUTING=true` (default), services communicate via fully qualified domain names (FQDNs) instead of internal service names. This enables:

- **Cross-node communication**: Services on different nodes can communicate
- **Cross-network communication**: Works with VPNs, service meshes, and custom networks
- **Multi-cluster support**: Multiple Swarm clusters can share the same domain
- **Load balancing**: Traefik automatically load balances across all healthy replicas

**Node-Specific Domains:**

Node-specific domain configuration is managed via node labels in the database (configured through the Superadmin dashboard). This allows:

- **Direct node routing**: API Gateway can route to specific nodes
- **Node isolation**: Each node has its own service domains
- **Custom routing logic**: Applications can target specific nodes

**Node Configuration via Labels:**

Node-specific domains are configured using node labels stored in the database:

- **`obiente.subdomain`**: Node subdomain identifier (e.g., `node1`, `us-east-1`)
- **`obiente.use_node_specific_domains`**: Boolean to enable node-specific domains
- **`obiente.service_domain_pattern`**: Domain pattern (`node-service` or `service-node`)

**Domain Patterns:**

- **`node-service` (default)**: `node1-auth-service.obiente.cloud`
  - Node identifier comes first
  - Example: `node1-auth-service.obiente.cloud`, `us-east-1-billing-service.obiente.cloud`

- **`service-node`**: `auth-service.node1.obiente.cloud`
  - Service name comes first, node identifier is a subdomain
  - Example: `auth-service.node1.obiente.cloud`, `billing-service.us-east-1.obiente.cloud`

**Important Notes:**

- **API Gateway and Dashboard ALWAYS use shared domains** (`api.${DOMAIN}` and `${DOMAIN}`) for load balancing, regardless of node-specific domain settings
- **Node subdomain detection**: 
  - **Swarm deployments**: Extracted from node labels (`obiente.subdomain` or `subdomain`) configured via Superadmin dashboard, or hostname if not configured
  - **Compose deployments**: Use `NODE_SUBDOMAIN` environment variable
- **Configuration**: 
  - **Swarm deployments**: Use the Superadmin dashboard to configure node-specific domains per node
  - **Compose deployments**: Use environment variables `NODE_SUBDOMAIN`, `USE_NODE_SPECIFIC_DOMAINS`, and `SERVICE_DOMAIN_PATTERN`

**Examples:**

```bash
# Internal routing (single network only)
USE_TRAEFIK_ROUTING=false
USE_DOMAIN_ROUTING=false
# Routes: http://auth-service:3002

# Traefik routing with shared domains (default, load balanced)
USE_TRAEFIK_ROUTING=true
USE_DOMAIN_ROUTING=true
DOMAIN=obiente.cloud
SKIP_TLS_VERIFY=true
# Routes: https://auth-service.obiente.cloud (load balanced across all nodes)

# Traefik routing with node-specific domains
USE_TRAEFIK_ROUTING=true
USE_DOMAIN_ROUTING=true
DOMAIN=obiente.cloud
SKIP_TLS_VERIFY=true
# For Swarm: Configure via node labels (obiente.subdomain=node1, obiente.use_node_specific_domains=true, obiente.service_domain_pattern=node-service)
# For Compose: Use environment variables (NODE_SUBDOMAIN=node1, USE_NODE_SPECIFIC_DOMAINS=true, SERVICE_DOMAIN_PATTERN=node-service)
# Routes: https://node1-auth-service.obiente.cloud (specific node)
```

**Load Balancing:**

Traefik automatically load balances services with the same router rule across all healthy replicas:

- **Shared domains** (default): All nodes register with the same domain (e.g., `auth-service.obiente.cloud`). Traefik distributes requests across all healthy replicas using round-robin.
- **Node-specific domains**: Each node registers with its own domain (e.g., `node1-auth-service.obiente.cloud`) when configured via node labels. API Gateway routes to specific nodes based on node subdomain.
- **Multi-cluster**: When multiple Swarm clusters share the same domain (via external DNS/LB), Traefik load balances across all clusters.

**Note:** When using Traefik routing, ensure all services have Traefik labels configured. The orchestrator service automatically syncs Traefik labels for microservices every 30 seconds. DNS service may need special handling as it uses a different port (8053).

### Authentication

| Variable          | Type    | Default                      | Required |
| ----------------- | ------- | ---------------------------- | -------- |
| `ZITADEL_URL`     | string  | `https://auth.obiente.cloud` | ❌       |
| `DISABLE_AUTH`    | boolean | `false`                      | ❌       |
| `SKIP_TLS_VERIFY` | boolean | `false`                      | ❌       |

**Development Options:**

```bash
# Disable authentication completely (development only!)
DISABLE_AUTH=true

# Skip TLS certificate verification (development only!)
SKIP_TLS_VERIFY=true
```

### CORS Configuration

| Variable      | Type   | Default | Required |
| ------------- | ------ | ------- | -------- |
| `CORS_ORIGIN` | string | `*`     | ❌       |

**Examples:**

```bash
# Allow all origins (development only)
CORS_ORIGIN=*

# Allow specific origins
CORS_ORIGIN=https://example.com,https://app.example.com

# Single origin
CORS_ORIGIN=https://obiente.cloud
```

### Email Delivery

| Variable               | Type    | Default             | Required   |
| ---------------------- | ------- | ------------------- | ---------- |
| `SMTP_HOST`            | string  | -                   | ✅ (email) |
| `SMTP_PORT`            | number  | `587`               | ❌         |
| `SMTP_USERNAME`        | string  | -                   | ❌         |
| `SMTP_PASSWORD`        | string  | -                   | ❌         |
| `SMTP_FROM_ADDRESS`    | string  | -                   | ✅ (email) |
| `SMTP_FROM_NAME`       | string  | `Obiente Cloud`     | ❌         |
| `SMTP_REPLY_TO`        | string  | -                   | ❌         |
| `SMTP_USE_STARTTLS`    | boolean | `true`              | ❌         |
| `SMTP_SKIP_TLS_VERIFY` | boolean | `false`             | ❌         |
| `SMTP_TIMEOUT_SECONDS` | number  | `10`                | ❌         |
| `SMTP_LOCAL_NAME`      | string  | `api.obiente.local` | ❌         |

**Notes:**

- `SMTP_HOST` and `SMTP_FROM_ADDRESS` must be set for outbound email. When missing, email delivery is disabled gracefully.
- Set `SMTP_USERNAME` and `SMTP_PASSWORD` for authenticated SMTP relays.
- Use `SMTP_REPLY_TO` to redirect replies to a shared inbox (e.g. `support@yourdomain`).

**Example:**

```bash
SMTP_HOST=smtp.mailprovider.com
SMTP_PORT=587
SMTP_USERNAME=obiente-api
SMTP_PASSWORD=<strong_password>
SMTP_FROM_ADDRESS=no-reply@obiente.cloud
SMTP_FROM_NAME="Obiente Cloud"
SMTP_REPLY_TO=support@obiente.cloud
```

### Dashboard & Support

| Variable            | Type   | Default                 | Required |
| ------------------- | ------ | ----------------------- | -------- |
| `DASHBOARD_URL`     | string | `https://obiente.cloud` | ❌       |
| `SUPPORT_EMAIL`     | string | -                       | ❌       |
| `SUPERADMIN_EMAILS` | string | -                       | ❌       |
| `SELF_HOSTED`       | bool   | `false`                 | ❌       |
| `BILLING_ENABLED`   | bool   | `true`                  | ❌       |

The API uses `DASHBOARD_URL` to build links in transactional emails and billing redirects. Configure `SUPPORT_EMAIL` to surface a contact address in email footers. `SUPERADMIN_EMAILS` grants system-wide access to the Superadmin API and dashboard (provide a comma-separated list of email addresses matching your identity provider). For self-hosted deployments, these are superadmins. For Obiente Cloud managed deployments, this refers to The Obiente Cloud Team. Set `SELF_HOSTED=true` to indicate this is a self-hosted deployment. Set `BILLING_ENABLED=false` to disable all billing functionality (hides billing pages, disables payment processing, and ignores webhooks).

### Orchestration

| Variable                   | Type    | Default        | Required | Description                                                                                                                                    |
| -------------------------- | ------- | -------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `DEPLOYMENT_STRATEGY`      | string  | `least-loaded` | ❌       | Deployment strategy for node selection                                                                                                         |
| `MAX_DEPLOYMENTS_PER_NODE` | number  | `50`           | ❌       | Maximum number of deployments allowed per node                                                                                                |
| `API_BASE_URL`             | string  | `http://api-gateway:3001` | ❌   | Base URL for API service communication. Automatically uses domain-based URL (`https://api.${DOMAIN}`) when `USE_DOMAIN_ROUTING=true`.         |
| `USE_DOMAIN_ROUTING`       | boolean | `true`         | ❌       | Use domain-based routing for service-to-service communication. When `true`, services use domain-based URLs (works across nodes/networks).    |
| `NODE_SUBDOMAIN`           | string  | -              | ❌       | Node subdomain identifier for compose deployments (e.g., `node1`, `us-east-1`). Used for node-specific domains when `USE_NODE_SPECIFIC_DOMAINS=true`. For Swarm deployments, configure via Superadmin dashboard instead. |
| `USE_NODE_SPECIFIC_DOMAINS`| boolean | `false`        | ❌       | Enable node-specific domains for compose deployments. When `true`, microservices use node-specific subdomains (e.g., `node1-auth-service.domain`). For Swarm deployments, configure via Superadmin dashboard instead. |
| `SERVICE_DOMAIN_PATTERN`   | string  | `node-service` | ❌       | Domain pattern for node-specific domains (`node-service` or `service-node`). For Swarm deployments, configure via Superadmin dashboard instead. |

**Deployment Strategies:**

- `least-loaded` - Deploy to node with least resources
- `round-robin` - Cycle through nodes
- `resource-based` - Match resources to node capacity

**API Base URL Configuration:**

The `API_BASE_URL` variable controls how services communicate with the API Gateway:

- **Default (internal)**: `http://api-gateway:3001` - Direct service-to-service communication within Docker network
- **Domain-based (auto)**: When `USE_DOMAIN_ROUTING=true` and `DOMAIN` is set (not `localhost`), automatically uses `https://api.${DOMAIN}`
- **Explicit**: Set `API_BASE_URL` explicitly to override automatic detection

**Examples:**

```bash
# Internal routing (single network)
USE_DOMAIN_ROUTING=false
API_BASE_URL=http://api-gateway:3001
# Services connect via: http://api-gateway:3001

# Domain-based routing (default, cross-node)
USE_DOMAIN_ROUTING=true
DOMAIN=obiente.cloud
# Services automatically connect via: https://api.obiente.cloud

# Explicit override
API_BASE_URL=https://api.example.com
# Services connect via: https://api.example.com
```

**Node-Specific Domain Configuration (Compose Deployments):**

For non-Swarm (compose) deployments, node-specific domains are configured via environment variables:

```bash
# Enable node-specific domains for compose deployment
NODE_SUBDOMAIN=node1
USE_NODE_SPECIFIC_DOMAINS=true
SERVICE_DOMAIN_PATTERN=node-service
DOMAIN=obiente.cloud
# Results in: node1-auth-service.obiente.cloud

# Alternative pattern (service-node)
SERVICE_DOMAIN_PATTERN=service-node
# Results in: auth-service.node1.obiente.cloud
```

**Note:** For Swarm deployments, node-specific domains are configured via the Superadmin dashboard (node labels), not environment variables.

### Metrics Database Configuration

| Variable              | Type   | Default                           | Required |
| --------------------- | ------ | --------------------------------- | -------- |
| `METRICS_DB_HOST`         | string | `timescaledb`                     | ❌       |
| `METRICS_DB_PORT`         | number | `5432`                            | ❌       |
| `METRICS_DB_EXPOSE_PORT`  | number | `5433`                            | ❌       | Port to expose TimescaleDB on host (default: 5433, localhost only) |
| `METRICS_DB_PORT_MODE`    | string | `host`                             | ❌       | Port mode: `host` (default, for localhost binding) or `ingress` |
| `METRICS_DB_ALLOWED_HOSTS` | string | -                                 | ❌       | Comma-separated IPs/subnets to allow in pg_hba.conf (falls back to POSTGRES_ALLOWED_HOSTS) |
| `METRICS_DB_USER`         | string | `POSTGRES_USER` or `postgres`     | ❌       |
| `METRICS_DB_PASSWORD`     | string | `POSTGRES_PASSWORD` or `postgres` | ❌       |
| `METRICS_DB_NAME`         | string | `obiente_metrics`                 | ❌       |

**Metrics Database Host Configuration (`METRICS_DB_HOST`):**

Similar to `DB_HOST`, supports different networking configurations:

- **Docker Swarm**: Use the service name (default: `timescaledb`)
- **Netbird VPN**: Use the Netbird internal domain (e.g., `timescaledb.example.netbird`)
- **Custom**: Set to any hostname or IP address

**Examples:**

```bash
# Docker Swarm (default)
METRICS_DB_HOST=timescaledb

# Netbird VPN
METRICS_DB_HOST=timescaledb.example.netbird

# Custom hostname/IP
METRICS_DB_HOST=metrics-db.example.com
```

**Metrics Database Port Exposure (`METRICS_DB_EXPOSE_PORT`):**

TimescaleDB port is **exposed by default on localhost only** (127.0.0.1:5433) for security, similar to PostgreSQL.

**Default Configuration:**
- Port exposed: `5433` (configurable via `METRICS_DB_EXPOSE_PORT`)
- Mode: `host` (for localhost binding)
- Binding: All interfaces (restrict via firewall for localhost-only)

**To restrict to localhost only**, configure firewall rules (same as PostgreSQL):
```bash
# Using iptables (restrict TimescaleDB to localhost only)
sudo iptables -A INPUT -p tcp --dport 5432 ! -s 127.0.0.1 -j DROP
```

**Examples:**

```bash
# Default: Exposed on localhost only
METRICS_DB_EXPOSE_PORT=5433
METRICS_DB_PORT_MODE=host

# Expose on all interfaces (for Netbird VPN access)
METRICS_DB_EXPOSE_PORT=5433
METRICS_DB_PORT_MODE=host
```

**Note:** The port is exposed by default. To disable, comment out the `ports:` section in `docker-compose.swarm.yml`.

**Metrics Database Allowed Hosts (`METRICS_DB_ALLOWED_HOSTS`):**

Similar to `POSTGRES_ALLOWED_HOSTS`, configure allowed hosts for TimescaleDB. Falls back to `POSTGRES_ALLOWED_HOSTS` if not set.

**Examples:**

```bash
# Allow specific IP for metrics database
METRICS_DB_ALLOWED_HOSTS=10.10.10.1

# Allow subnet
METRICS_DB_ALLOWED_HOSTS=10.0.0.0/8

# Falls back to POSTGRES_ALLOWED_HOSTS if not set
POSTGRES_ALLOWED_HOSTS=10.10.10.1,10.0.0.0/8
```

**Notes:**

- Metrics are stored in a separate TimescaleDB instance for optimal time-series performance
- Falls back to main PostgreSQL if TimescaleDB is not available
- In HA deployments, connects via `metrics-pgpool` load balancer

### Metrics Collection Configuration

| Variable                          | Type     | Default | Required | Description                                  |
| --------------------------------- | -------- | ------- | -------- | -------------------------------------------- |
| `METRICS_COLLECTION_INTERVAL`     | duration | `5s`    | ❌       | How often to collect metrics from containers |
| `METRICS_STORAGE_INTERVAL`        | duration | `60s`   | ❌       | How often to batch write metrics to database |
| `METRICS_LIVE_RETENTION`          | duration | `5m`    | ❌       | How long to keep live metrics in memory      |
| `METRICS_MAX_WORKERS`             | number   | `50`    | ❌       | Max parallel workers for stats collection    |
| `METRICS_BATCH_SIZE`              | number   | `100`   | ❌       | Batch size for database writes               |
| `METRICS_MAX_LIVE_PER_DEPLOYMENT` | number   | `1000`  | ❌       | Max metrics to keep in memory per deployment |
| `METRICS_MAX_PREVIOUS_STATS`      | number   | `10000` | ❌       | Max container stats to cache for delta calc  |

**Duration Format:**

All duration values use Go's duration format: `5s` (5 seconds), `1m` (1 minute), `2h` (2 hours).

**Example:**

```bash
# Collect metrics every 3 seconds
METRICS_COLLECTION_INTERVAL=3s

# Store aggregated metrics every 2 minutes
METRICS_STORAGE_INTERVAL=2m

# Keep 10 minutes of live metrics in memory
METRICS_LIVE_RETENTION=10m

# Use 100 parallel workers for high-throughput scenarios
METRICS_MAX_WORKERS=100
```

### Metrics Docker API Configuration

| Variable                                   | Type     | Default | Required | Description                             |
| ------------------------------------------ | -------- | ------- | -------- | --------------------------------------- |
| `METRICS_DOCKER_API_TIMEOUT`               | duration | `10s`   | ❌       | Timeout for Docker API calls            |
| `METRICS_DOCKER_API_RETRY_MAX`             | number   | `3`     | ❌       | Max retry attempts for failed API calls |
| `METRICS_DOCKER_API_RETRY_BACKOFF_INITIAL` | duration | `1s`    | ❌       | Initial backoff delay for retries       |
| `METRICS_DOCKER_API_RETRY_BACKOFF_MAX`     | duration | `30s`   | ❌       | Maximum backoff delay for retries       |

**Example:**

```bash
# Increase timeout for slow Docker hosts
METRICS_DOCKER_API_TIMEOUT=30s

# More aggressive retry strategy
METRICS_DOCKER_API_RETRY_MAX=5
METRICS_DOCKER_API_RETRY_BACKOFF_INITIAL=500ms
```

### Metrics Circuit Breaker Configuration

| Variable                                    | Type     | Default | Required | Description                                 |
| ------------------------------------------- | -------- | ------- | -------- | ------------------------------------------- |
| `METRICS_CIRCUIT_BREAKER_FAILURE_THRESHOLD` | number   | `5`     | ❌       | Failures before opening circuit             |
| `METRICS_CIRCUIT_BREAKER_COOLDOWN`          | duration | `1m`    | ❌       | Cooldown period before attempting half-open |
| `METRICS_CIRCUIT_BREAKER_HALFOPEN_MAX`      | number   | `3`     | ❌       | Successful calls needed to close circuit    |

**Circuit Breaker States:**

- **Closed**: Normal operation, requests pass through
- **Open**: Too many failures, requests immediately fail
- **Half-Open**: Testing if service recovered, limited requests allowed

**Example:**

```bash
# More sensitive circuit breaker (opens after 3 failures)
METRICS_CIRCUIT_BREAKER_FAILURE_THRESHOLD=3

# Longer cooldown for unstable Docker hosts
METRICS_CIRCUIT_BREAKER_COOLDOWN=5m
```

### Metrics Health Check Configuration

| Variable                                 | Type     | Default | Required | Description                           |
| ---------------------------------------- | -------- | ------- | -------- | ------------------------------------- |
| `METRICS_HEALTH_CHECK_INTERVAL`          | duration | `30s`   | ❌       | How often to run health checks        |
| `METRICS_HEALTH_CHECK_FAILURE_THRESHOLD` | number   | `3`     | ❌       | Consecutive failures before unhealthy |

**Example:**

```bash
# Check health every 10 seconds
METRICS_HEALTH_CHECK_INTERVAL=10s

# More sensitive health checks
METRICS_HEALTH_CHECK_FAILURE_THRESHOLD=2
```

### Metrics Backpressure & Subscriber Configuration

| Variable                              | Type     | Default | Required | Description                            |
| ------------------------------------- | -------- | ------- | -------- | -------------------------------------- |
| `METRICS_SUBSCRIBER_BUFFER_SIZE`      | number   | `100`   | ❌       | Buffer size for subscriber channels    |
| `METRICS_SUBSCRIBER_SLOW_THRESHOLD`   | duration | `5s`    | ❌       | Time before marking subscriber as slow |
| `METRICS_SUBSCRIBER_CLEANUP_INTERVAL` | duration | `1m`    | ❌       | How often to cleanup dead subscribers  |

**Example:**

```bash
# Larger buffers for high-throughput streaming
METRICS_SUBSCRIBER_BUFFER_SIZE=500

# More aggressive cleanup of slow subscribers
METRICS_SUBSCRIBER_SLOW_THRESHOLD=2s
METRICS_SUBSCRIBER_CLEANUP_INTERVAL=30s
```

### Metrics Retry Queue Configuration

| Variable                       | Type     | Default | Required | Description                            |
| ------------------------------ | -------- | ------- | -------- | -------------------------------------- |
| `METRICS_RETRY_MAX_RETRIES`    | number   | `5`     | ❌       | Max retries for failed database writes |
| `METRICS_RETRY_INTERVAL`       | duration | `1m`    | ❌       | Interval between retry attempts        |
| `METRICS_RETRY_MAX_QUEUE_SIZE` | number   | `10000` | ❌       | Max size of retry queue                |

**Example:**

```bash
# More persistent retry strategy
METRICS_RETRY_MAX_RETRIES=10
METRICS_RETRY_INTERVAL=30s

# Larger queue for high-volume scenarios
METRICS_RETRY_MAX_QUEUE_SIZE=50000
```

### Domain & SSL

| Variable     | Type   | Default               | Required               |
| ------------ | ------ | --------------------- | ---------------------- |
| `DOMAIN`     | string | `obiente.example.com` | ❌                     |
| `ACME_EMAIL` | string | -                     | ❌ (for Let's Encrypt) |

**Example:**

```bash
DOMAIN=obiente.cloud
ACME_EMAIL=admin@obiente.cloud
```

### DNS Configuration

| Variable      | Type   | Default | Required | Description                                                                                |
| ------------- | ------ | ------- | -------- | ------------------------------------------------------------------------------------------ |
| `NODE_IPS` | string | -       | ✅       | Node IPs per region (format: `"region1:ip1,ip2;region2:ip3,ip4"` or simple `"ip1,ip2"`). Used for DNS resolution of deployments and game servers. |
| `DNS_IPS`     | string | -       | ❌       | DNS server IPs (comma-separated) for nameserver configuration                              |
| `DNS_PORT`    | number | `53`    | ❌       | DNS server port (use different port if 53 is in use)                                       |

### Traefik Port Configuration

| Variable                  | Type   | Default | Required | Description                                                                                |
| ------------------------- | ------ | ------- | -------- | ------------------------------------------------------------------------------------------ |
| `TRAEFIK_HTTP_PORT`       | number | `80`    | ❌       | Published HTTP port for Traefik (default: 80)                                              |
| `TRAEFIK_HTTPS_PORT`      | number | `443`   | ❌       | Published HTTPS port for Traefik (default: 443)                                            |
| `TRAEFIK_DEPLOYMENTS_PORT`| number | `8000`  | ❌       | Published port for user deployments (default: 8000)                                        |
| `TRAEFIK_DASHBOARD_PORT`  | number | `8080`  | ❌       | Published port for Traefik dashboard (default: 8080)                                       |
| `TRAEFIK_SSH_PORT`        | number | `2222`  | ❌       | Published port for SSH proxy (default: 2222)                                              |

**Examples:**

```bash
# Use default ports (80, 443, 8000, 8080, 2222)
# No configuration needed

# Use custom ports (e.g., when running behind another reverse proxy)
TRAEFIK_HTTP_PORT=8080
TRAEFIK_HTTPS_PORT=8443
TRAEFIK_DEPLOYMENTS_PORT=8000
TRAEFIK_DASHBOARD_PORT=8080
TRAEFIK_SSH_PORT=2222

# Use non-standard ports for development
TRAEFIK_HTTP_PORT=8080
TRAEFIK_HTTPS_PORT=8443
TRAEFIK_DASHBOARD_PORT=9090
```

**Note:** These variables control the **published** (host) ports that Traefik listens on. The **target** ports (inside the container) remain fixed at 80, 443, 8000, 8080, and the SSH proxy port. This allows you to map Traefik to different host ports if needed (e.g., when running behind another reverse proxy or when standard ports are already in use).

### VPS Configuration

| Variable                 | Type   | Default     | Required | Description                                                                                                                                                                                                                                                                                                                                             |
| ------------------------ | ------ | ----------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `VPS_REGIONS`            | string | -           | ✅       | VPS regions configuration (format: `"region1:Name 1;region2:Name 2"` or simple `"region1"`)                                                                                                                                                                                                                                                             |
| `PROXMOX_API_URL`        | string | -           | ✅       | Proxmox API URL (e.g., `https://proxmox.example.com:8006`)                                                                                                                                                                                                                                                                                              |
| `PROXMOX_USERNAME`       | string | `root@pam`  | ❌       | Proxmox username (default: `root@pam`)                                                                                                                                                                                                                                                                                                                  |
| `PROXMOX_PASSWORD`       | string | -           | ✅\*     | Proxmox password (required if not using token)                                                                                                                                                                                                                                                                                                          |
| `PROXMOX_TOKEN_ID`       | string | -           | ✅\*     | Proxmox API token ID (required if not using password)                                                                                                                                                                                                                                                                                                   |
| `PROXMOX_TOKEN_SECRET`   | string | -           | ✅\*     | Proxmox API token secret (required if not using password)                                                                                                                                                                                                                                                                                               |
| `PROXMOX_STORAGE_POOL`   | string | `local-lvm` | ❌       | Proxmox storage pool for VM disks                                                                                                                                                                                                                                                                                                                       |
| `PROXMOX_VM_ID_START`    | number | -           | ❌       | Starting VM ID range (e.g., `300`). If set, VMs will be created starting from this ID. If not set, Proxmox auto-generates the next available ID.                                                                                                                                                                                                        |
| `PROXMOX_VLAN_ID`        | number | -           | ❌       | Optional VLAN tag for VM network isolation (e.g., `100`). If set, all VMs will be placed on this VLAN. This provides Layer 2 isolation and helps prevent IP spoofing and network attacks.                                                                                                                                                               |
| `PROXMOX_SSH_HOST`       | string | -           | ❌       | Proxmox node hostname/IP for SSH snippet writing (defaults to `PROXMOX_API_URL` host if not set). Required for cloud-init snippet writing via SSH. See [Proxmox SSH User Setup Guide](../guides/proxmox-ssh-user-setup.md) for details.                                                              |
| `PROXMOX_SSH_USER`       | string | `obiente-cloud` | ❌       | SSH user for snippet writing. Defaults to `obiente-cloud` if not set. See [Proxmox SSH User Setup Guide](../guides/proxmox-ssh-user-setup.md) for setup instructions.                                                                                                                                                                                  |
| `PROXMOX_SSH_KEY_PATH`   | string | -           | ❌       | Path to SSH private key file for snippet writing. Either this or `PROXMOX_SSH_KEY_DATA` must be set if using SSH method.                                                                                                                                                                                                                                  |
| `PROXMOX_SSH_KEY_DATA`   | string | -           | ❌       | SSH private key content (alternative to `PROXMOX_SSH_KEY_PATH`). Supports both raw key data and base64-encoded keys. Useful when using secrets managers. Either this or `PROXMOX_SSH_KEY_PATH` must be set if using SSH method.                                                                                                                                                                              |
| `SSH_PROXY_PORT`         | number | `2222`      | ❌       | SSH proxy port for VPS access                                                                                                                                                                                                                                                                                                                           |
| `VPS_GATEWAY_API_SECRET` | string | -           | ❌       | Shared secret for authenticating with vps-gateway service. Must match `GATEWAY_API_SECRET` configured in vps-gateway. Required when using gateway service.                                                                                                                                                                                              |
| `VPS_GATEWAY_URL`        | string | -           | ❌       | Gateway gRPC server URL (e.g., `http://gateway-public-ip:1537`). API instances connect to this URL to communicate with the gateway. Port 1537 = OCG (Obiente Cloud Gateway).                                                                                                                                                                            |
| `VPS_GATEWAY_BRIDGE`     | string | `OCvpsnet`  | ❌       | Bridge name for gateway network in Proxmox. When using SDN, this should be the SDN VNet bridge name (auto-created by Proxmox, e.g., `OCvpsnet` for the OCvps-vnet VNet). VPS instances will be connected to this bridge when gateway is enabled. See [VPS Gateway Setup Guide](../guides/vps-gateway-setup.md) for details on finding SDN bridge names. |

### VPS Gateway Service Configuration

These environment variables are used by the `vps-gateway` service itself (not the API):

| Variable                   | Type   | Default | Required | Description                                                                                                                                                                                                            |
| -------------------------- | ------ | ------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GATEWAY_GRPC_PORT`        | int    | `1537`  | ⚪       | gRPC server port for the gateway. Default is **1537** which maps to "O 15 C 3 G" = "OCG" (Obiente Cloud Gateway), similar to how `10.15.3` maps to "O 15 C 3". Gateway exposes this port for API instances to connect. |
| `GATEWAY_API_SECRET`       | string | -       | ✅       | Shared secret for authenticating API connections. Must match `VPS_GATEWAY_API_SECRET` in API service. Both must be identical.                                                                                          |
| `GATEWAY_PUBLIC_IP`        | string | -       | ⚪       | Public IP address for DNAT configuration. Used for documentation purposes - actual DNAT is configured on router/firewall. Example: `203.0.113.1`.                                                                      |
| `GATEWAY_DHCP_POOL_START`  | string | -       | ✅       | Starting IP address for DHCP pool (e.g., `10.15.3.20`)                                                                                                                                                                 |
| `GATEWAY_DHCP_POOL_END`    | string | -       | ✅       | Ending IP address for DHCP pool (e.g., `10.15.3.254`)                                                                                                                                                                  |
| `GATEWAY_DHCP_SUBNET`      | string | -       | ✅       | Subnet address (e.g., `10.15.3.0`)                                                                                                                                                                                     |
| `GATEWAY_DHCP_SUBNET_MASK` | string | -       | ✅       | Subnet mask (e.g., `255.255.255.0`)                                                                                                                                                                                    |
| `GATEWAY_DHCP_GATEWAY`     | string | -       | ✅       | Gateway IP address (e.g., `10.15.3.1`)                                                                                                                                                                                 |
| `GATEWAY_DHCP_DNS`         | string | -       | ✅       | Comma-separated DNS servers for upstream DNS resolution (e.g., `1.1.1.1,1.0.0.1`)                                                                                                                                      |
| `GATEWAY_DHCP_DOMAIN`      | string | `vps.local` | ❌       | DNS domain for VPS hostname resolution (e.g., `vps.local`). The gateway's dnsmasq will resolve VPS hostnames within this domain.                          |
| `GATEWAY_DHCP_INTERFACE`   | string | -       | ✅       | Network interface name for DHCP (e.g., `eth0`, `eth1`)                                                                                                                                                                 |
| `LOG_LEVEL`                | string | `info`  | ❌       | Logging level (`debug`, `info`, `warn`, `error`)                                                                                                                                                                       |

**NODE_IPS Format:**

Two formats are supported:

**Simple format** (single IP or comma-separated IPs, defaults to "default" region):

```
ip1,ip2
```

**Multi-region format**:

```
region1:ip1,ip2;region2:ip3,ip4
```

**Examples:**

```bash
# Simple format (single IP, defaults to "default" region)
NODE_IPS="1.2.3.4"

# Simple format (multiple IPs, defaults to "default" region)
NODE_IPS="1.2.3.4,1.2.3.5"

# Single region
NODE_IPS="us-east-1:1.2.3.4,1.2.3.5"

# Multiple regions
NODE_IPS="us-east-1:1.2.3.4,1.2.3.5;eu-west-1:5.6.7.8,5.6.7.9"

# Explicit default region
NODE_IPS="default:1.2.3.4"
```

**DNS_IPS Format:**

Comma-separated list of public IP addresses where DNS servers run. Used for configuring nameserver records in your DNS provider.

```bash
# Single node
DNS_IPS="1.2.3.4"

# Multiple nodes (HA)
DNS_IPS="1.2.3.4,5.6.7.8,9.10.11.12"
```

**DNS_PORT:**

Default DNS port is 53. If port 53 is already in use (e.g., systemd-resolved), configure a different port:

```bash
# Use port 5353 instead
DNS_PORT=5353
```

**VPS_REGIONS Format:**

Two formats are supported:

**Simple format** (single region ID or comma-separated region IDs):

```
us-illinois
```

```
us-illinois,us-east-1
```

**Multi-region format with names**:

```
region1:Name 1;region2:Name 2
```

**Examples:**

```bash
# Single region (ID only - name auto-generated)
VPS_REGIONS="us-illinois"

# Multiple regions (IDs only)
VPS_REGIONS="us-illinois,us-east-1"

# Single region with custom name
VPS_REGIONS="us-illinois:US Illinois"

# Multiple regions with custom names
VPS_REGIONS="us-illinois:US Illinois;us-east-1:US East (N. Virginia)"
```

**Proxmox Configuration:**

VPS provisioning requires Proxmox VE to be configured. You can use either password or API token authentication:

```bash
# Password authentication
PROXMOX_API_URL=https://proxmox.example.com:8006
PROXMOX_USERNAME=root@pam
PROXMOX_PASSWORD=your-secure-password

# Or API token authentication (recommended)
PROXMOX_API_URL=https://proxmox.example.com:8006
PROXMOX_USERNAME=root@pam
PROXMOX_TOKEN_ID=obiente-cloud
PROXMOX_TOKEN_SECRET=your-token-secret

# Optional: Custom storage pool (must exist in Proxmox)
PROXMOX_STORAGE_POOL=local-zfs

# Optional: VLAN tag for network isolation (recommended for security)
PROXMOX_VLAN_ID=100

# Optional: SSH configuration for cloud-init snippet writing
# See docs/guides/proxmox-ssh-user-setup.md for detailed setup instructions
PROXMOX_SSH_HOST=proxmox.example.com
PROXMOX_SSH_USER=obiente-cloud
PROXMOX_SSH_KEY_PATH=/path/to/obiente-cloud-key
# Or use key data from secrets manager:
# PROXMOX_SSH_KEY_DATA="-----BEGIN OPENSSH PRIVATE KEY-----\n..."
```

**Note:** The storage pool specified in `PROXMOX_STORAGE_POOL` must exist in your Proxmox installation and support VM disk images. Common values are `local-lvm` (default), `local`, `local-zfs`, or custom storage pools. If the storage pool doesn't exist, VPS creation will fail with a clear error message listing available storage pools. See the [VPS Configuration Guide](../guides/vps-configuration.md#storage-configuration) for details on checking available storage pools.

**Required Proxmox API Token Permissions:**

The API token must have the following permissions to create and manage VMs:

- `VM.Allocate` - Create new VMs
- `VM.Clone` - Clone VM templates (if using templates)
- `VM.Config.Disk` - Configure VM disk storage
- `VM.Config.Network` - Configure VM network settings
- `VM.Config.Options` - Configure VM options (cloud-init, etc.)
- `VM.Config.CPU` - Configure VM CPU settings
- `VM.Config.Memory` - Configure VM memory settings
- `VM.PowerMgmt` - Start, stop, reboot VMs
- `VM.Monitor` - Monitor VM status and metrics
- `Datastore.Allocate` - Allocate storage for VMs
- `Datastore.AllocateSpace` - Allocate disk space
- `Datastore.AllocateTemplate` - Upload cloud-init snippets and templates (required for user management and cloud-init configuration)

See the [VPS Provisioning Guide](../guides/vps-provisioning.md#3-configure-api-token-permissions) for detailed permission setup instructions.

**Note:** See [DNS Configuration](../deployment/dns.md) for detailed setup instructions.

### Security

| Variable               | Type   | Default | Required |
| ---------------------- | ------ | ------- | -------- |
| `SECRET`               | string | -       | ✅       |
| `RATE_LIMIT_WINDOW_MS` | number | `60000` | ❌       |
| `RATE_LIMIT_MAX`       | number | `100`   | ❌       |

**Generate Secrets:**

```bash
# Generate a secure secret
openssl rand -hex 32

# Add to .env
SECRET=<generated_value>
```

**Note:** `SECRET` is used for cryptographic operations like domain verification token generation. It should be a strong, random value and kept secure.

### Stripe Payment Processing

| Variable                             | Type   | Default | Required      |
| ------------------------------------ | ------ | ------- | ------------- |
| `STRIPE_SECRET_KEY`                  | string | -       | ✅ (billing)  |
| `STRIPE_WEBHOOK_SECRET`              | string | -       | ✅ (webhooks) |
| `NUXT_PUBLIC_STRIPE_PUBLISHABLE_KEY` | string | -       | ✅ (frontend) |

**Setup:**

1. Create a Stripe account at https://stripe.com
2. Get your API keys from the Stripe Dashboard (API keys section)
   - **Secret key** (starts with `sk_`) → Set as `STRIPE_SECRET_KEY`
   - **Publishable key** (starts with `pk_`) → Set as `NUXT_PUBLIC_STRIPE_PUBLISHABLE_KEY`
3. Create a webhook endpoint in Stripe Dashboard pointing to `https://your-domain.com/webhooks/stripe`
   - **Important:** Set the API version to `2025-10-29.clover` to match the SDK version
   - Go to **Developers** > **Webhooks** > **Add endpoint**
   - Enter your endpoint URL
   - Select events: `checkout.session.completed`, `payment_intent.succeeded`, `payment_intent.payment_failed`
   - In the **Version** dropdown, select `2025-10-29.clover`
4. Set `STRIPE_WEBHOOK_SECRET` to the webhook signing secret (starts with `whsec_`)

**Development:**

For local development, use Stripe CLI to forward webhooks:

```bash
# Install Stripe CLI
https://docs.stripe.com/stripe-cli/install

# Login
stripe login

# Forward webhooks to local server
stripe listen --forward-to localhost:3001/webhooks/stripe
```

The webhook secret will be shown in the CLI output. Use that for `STRIPE_WEBHOOK_SECRET` in development.

**Example:**

```bash
# Backend (API)
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...

# Frontend (Dashboard)
NUXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_test_...
```

**Note:** The publishable key (`NUXT_PUBLIC_STRIPE_PUBLISHABLE_KEY`) is safe to expose publicly as it's used client-side for Stripe.js initialization. The secret key (`STRIPE_SECRET_KEY`) must be kept secure and never exposed to the client.

### Monitoring

| Variable                  | Type   | Default                                        | Required | Description                                         |
| ------------------------- | ------ | ---------------------------------------------- | -------- | --------------------------------------------------- |
| `GRAFANA_PASSWORD`        | string | `admin`                                        | ❌       | Grafana admin password                              |
| `GRAFANA_POSTGRES_HOST`   | string | `postgres` (Swarm)<br>`pgpool` (HA)            | ❌       | PostgreSQL service hostname for Grafana datasource  |
| `GRAFANA_METRICS_DB_HOST` | string | `timescaledb` (Swarm)<br>`metrics-pgpool` (HA) | ❌       | TimescaleDB service hostname for Grafana datasource |
| `ALERT_EMAIL`             | string | `admin@example.com`                            | ❌       | Email address for Grafana alert notifications       |

**Grafana Configuration:**

Grafana automatically provisions datasources using environment variables. The `GRAFANA_POSTGRES_HOST` and `GRAFANA_METRICS_DB_HOST` variables determine which database services Grafana connects to:

- **Swarm deployments**: Use direct service names (`postgres`, `timescaledb`)
- **HA deployments**: Use pgpool service names (`pgpool`, `metrics-pgpool`)

### Dashboard Replica Configuration

| Variable                          | Type   | Default | Required | Description                            |
| --------------------------------- | ------ | ------- | -------- | -------------------------------------- |
| `DASHBOARD_REPLICAS`              | number | `5`     | ❌       | Number of dashboard replicas to deploy |
| `DASHBOARD_MAX_REPLICAS_PER_NODE` | number | `2`     | ❌       | Maximum replicas allowed per node      |

**Replica Configuration:**

Docker Swarm doesn't natively support percentage-based replica configuration. Use these variables to configure replica counts based on your cluster size.

**Calculating Values:**

Use the helper script to calculate appropriate values:

```bash
# Run the calculation script
./scripts/calculate-replicas.sh

# Or manually calculate:
# - Replicas: ceil(cluster_size * desired_percentage / 100)
#   Example: 3 nodes * 50% = 1.5 → 2 replicas (minimum 2 for HA)
# - Max per node: ceil(replicas / cluster_size)
#   Example: 5 replicas / 3 nodes = 1.67 → 2 per node
```

**Example Configurations:**

```bash
# Small cluster (2-3 nodes)
DASHBOARD_REPLICAS=2
DASHBOARD_MAX_REPLICAS_PER_NODE=1

# Medium cluster (4-6 nodes)
DASHBOARD_REPLICAS=3
DASHBOARD_MAX_REPLICAS_PER_NODE=1

# Large cluster (7+ nodes)
DASHBOARD_REPLICAS=5
DASHBOARD_MAX_REPLICAS_PER_NODE=2
```

**Important:** Ensure `DASHBOARD_REPLICAS <= (cluster_size * DASHBOARD_MAX_REPLICAS_PER_NODE)` to avoid deployment errors.

**Example:**

```bash
# Swarm deployment
GRAFANA_POSTGRES_HOST=postgres
GRAFANA_METRICS_DB_HOST=timescaledb
ALERT_EMAIL=alerts@obiente.cloud

# HA deployment
GRAFANA_POSTGRES_HOST=pgpool
GRAFANA_METRICS_DB_HOST=metrics-pgpool
ALERT_EMAIL=alerts@obiente.cloud
```

**Metrics Observability:**

The API exposes metrics observability at `/metrics/observability` (no authentication required). This endpoint provides real-time statistics about metrics collection, including:

- Collection rates and error counts
- Container processing statistics
- Database write success/failure rates
- Retry queue status
- Subscriber counts and backpressure metrics
- Circuit breaker state
- Health status

Access via:

```bash
curl http://localhost:3001/metrics/observability
```

## Environment File Templates

### Local Development (.env)

```bash
POSTGRES_USER=obiente
POSTGRES_PASSWORD=local_dev_password
POSTGRES_DB=obiente
LOG_LEVEL=debug
DB_LOG_LEVEL=debug
CORS_ORIGIN=*
DISABLE_AUTH=true
# Stripe configuration (optional for local dev)
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
NUXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_test_...
```

### Production (.env)

```bash
POSTGRES_USER=obiente
POSTGRES_PASSWORD=<strong_random_password>
POSTGRES_DB=obiente
METRICS_DB_HOST=metrics-pgpool
METRICS_DB_USER=obiente
METRICS_DB_PASSWORD=<strong_random_password>
METRICS_DB_NAME=obiente_metrics
LOG_LEVEL=info
DB_LOG_LEVEL=error
CORS_ORIGIN=https://obiente.cloud
ZITADEL_URL=https://auth.obiente.cloud
DOMAIN=obiente.cloud
ACME_EMAIL=admin@obiente.cloud
SECRET=<generated_secret>
# Stripe configuration (required for billing)
STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...
# Frontend Stripe publishable key
NUXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_live_...
# Metrics configuration (optional, uses defaults if not set)
METRICS_COLLECTION_INTERVAL=5s
METRICS_STORAGE_INTERVAL=60s
METRICS_MAX_WORKERS=50
# Grafana configuration
GRAFANA_PASSWORD=<strong_random_password>
GRAFANA_POSTGRES_HOST=postgres
GRAFANA_METRICS_DB_HOST=timescaledb
ALERT_EMAIL=alerts@obiente.cloud
```

## Loading Environment Variables

### Docker Compose

```bash
# Automatically loads from .env file
docker compose up
```

### Docker Swarm

```bash
# Load from .env file
docker stack deploy --env-file .env -c docker-compose.swarm.yml obiente
```

## Security Best Practices

1. **Never commit `.env` files** to version control
2. **Use strong passwords** for all credentials
3. **Rotate secrets** regularly
4. **Use Docker secrets** in production
5. **Set `LOG_LEVEL=info`** in production

## Troubleshooting

### Environment variable not taking effect

```bash
# Verify variable is set
docker exec <container> env | grep VARIABLE_NAME

# Restart service to pick up changes
docker compose restart <service>
```

### .env file not loading

```bash
# Check file is in project root
ls -la .env

# Verify syntax (no spaces around =)
cat .env | grep VARIABLE_NAME
```

---

[← Back to Reference](../reference/index.md)
