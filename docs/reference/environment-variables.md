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

| Variable                   | Type   | Default        | Required |
| -------------------------- | ------ | -------------- | -------- |
| `DEPLOYMENT_STRATEGY`      | string | `least-loaded` | ❌       |
| `MAX_DEPLOYMENTS_PER_NODE` | number | `50`           | ❌       |

**Deployment Strategies:**

- `least-loaded` - Deploy to node with least resources
- `round-robin` - Cycle through nodes
- `resource-based` - Match resources to node capacity

### Metrics Database Configuration

| Variable              | Type   | Default                           | Required |
| --------------------- | ------ | --------------------------------- | -------- |
| `METRICS_DB_HOST`         | string | `timescaledb`                     | ❌       |
| `METRICS_DB_PORT`         | number | `5432`                            | ❌       |
| `METRICS_DB_EXPOSE_PORT`  | number | `5432`                            | ❌       | Port to expose TimescaleDB on host (default: 5432, localhost only) |
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

TimescaleDB port is **exposed by default on localhost only** (127.0.0.1:5432) for security, similar to PostgreSQL.

**Default Configuration:**
- Port exposed: `5432` (configurable via `METRICS_DB_EXPOSE_PORT`)
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
METRICS_DB_EXPOSE_PORT=5432
METRICS_DB_PORT_MODE=host

# Expose on all interfaces (for Netbird VPN access)
METRICS_DB_EXPOSE_PORT=5432
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
| `TRAEFIK_IPS` | string | -       | ✅       | Traefik IPs per region (format: `"region1:ip1,ip2;region2:ip3,ip4"` or simple `"ip1,ip2"`) |
| `DNS_IPS`     | string | -       | ❌       | DNS server IPs (comma-separated) for nameserver configuration                              |
| `DNS_PORT`    | number | `53`    | ❌       | DNS server port (use different port if 53 is in use)                                       |

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

**TRAEFIK_IPS Format:**

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
TRAEFIK_IPS="1.2.3.4"

# Simple format (multiple IPs, defaults to "default" region)
TRAEFIK_IPS="1.2.3.4,1.2.3.5"

# Single region
TRAEFIK_IPS="us-east-1:1.2.3.4,1.2.3.5"

# Multiple regions
TRAEFIK_IPS="us-east-1:1.2.3.4,1.2.3.5;eu-west-1:5.6.7.8,5.6.7.9"

# Explicit default region
TRAEFIK_IPS="default:1.2.3.4"
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
