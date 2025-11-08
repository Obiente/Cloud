# Configuration Guide

Configure your Obiente Cloud deployment to fit your needs.

## Environment File

The primary configuration is through the `.env` file.

Copy the example file:

```bash
cp .env.example .env
```

## Basic Configuration

### Database

```bash
POSTGRES_USER=obiente
POSTGRES_PASSWORD=your_secure_password
POSTGRES_DB=obiente

# Metrics database (TimescaleDB)
METRICS_DB_USER=obiente
METRICS_DB_PASSWORD=your_secure_password
METRICS_DB_NAME=obiente_metrics
```

**Note:** Metrics are stored in a separate TimescaleDB instance for optimal time-series performance. See [Environment Variables Reference](../reference/environment-variables.md#metrics-database-configuration) for all metrics configuration options.

### Authentication

```bash
# Production
ZITADEL_URL=https://auth.obiente.cloud

# Development (disable auth)
DISABLE_AUTH=true
```

### Domain & SSL

```bash
DOMAIN=obiente.cloud
ACME_EMAIL=admin@obiente.cloud
```

### DNS Configuration

```bash
# Traefik IPs per region (required)
# Format: "region1:ip1,ip2;region2:ip3,ip4"
TRAEFIK_IPS="us-east-1:1.2.3.4,1.2.3.5;eu-west-1:5.6.7.8,5.6.7.9"

# DNS server IPs (optional, for nameserver configuration)
# Comma-separated list of public IPs where DNS servers run
DNS_IPS="1.2.3.4,5.6.7.8,9.10.11.12"

# DNS server port (optional, default: 53)
# Use a different port if 53 is already in use (e.g., systemd-resolved)
DNS_PORT=53
```

**Note:** See [DNS Configuration](../deployment/dns.md) for detailed setup instructions.

## Security Configuration

### CORS

```bash
# Allow specific origins
CORS_ORIGIN=https://obiente.cloud

# Development only
CORS_ORIGIN=*
```

### Secrets

Generate secure secrets:

```bash
openssl rand -hex 32
```

Add to `.env`:

```bash
SECRET=<generated_value>
```

**Note:** `SECRET` is required and used for cryptographic operations like domain verification token generation.

### Metrics Configuration (Optional)

Fine-tune metrics collection behavior:

```bash
# Collection intervals
METRICS_COLLECTION_INTERVAL=5s
METRICS_STORAGE_INTERVAL=60s

# Performance tuning
METRICS_MAX_WORKERS=50
METRICS_BATCH_SIZE=100

# Resilience settings
METRICS_CIRCUIT_BREAKER_FAILURE_THRESHOLD=5
METRICS_HEALTH_CHECK_INTERVAL=30s
```

See [Environment Variables Reference](../reference/environment-variables.md#metrics-collection-configuration) for complete metrics configuration options.

## Production Configuration

For production deployments:

1. **Change all defaults** - Update all default passwords
2. **Set LOG_LEVEL=info** - Reduce application log verbosity
3. **Set DB_LOG_LEVEL=error** - Suppress database query logs (optional, defaults to LOG_LEVEL)
4. **Configure CORS properly** - Use specific origins
5. **Enable authentication** - Set `DISABLE_AUTH=false`
6. **Set up SSL** - Configure domain and ACME email
7. **Configure metrics database** - Set up separate TimescaleDB credentials
8. **Tune metrics collection** - Adjust workers and intervals based on load

## Reference

See [Environment Variables Reference](../reference/environment-variables.md) for complete configuration options.

---

[← Back to Getting Started](index.md)
