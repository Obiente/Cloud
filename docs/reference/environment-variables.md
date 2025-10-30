# Environment Variables Reference

Complete reference for all Obiente Cloud environment variables.

## Quick Reference

| Variable            | Default                      | Required | Description          |
| ------------------- | ---------------------------- | -------- | -------------------- |
| `POSTGRES_USER`     | `obiente`                    | ❌       | PostgreSQL username  |
| `POSTGRES_PASSWORD` | -                            | ✅       | PostgreSQL password  |
| `ZITADEL_URL`       | `https://auth.obiente.cloud` | ❌       | Zitadel instance URL |
| `LOG_LEVEL`         | `debug`                      | ❌       | Logging level        |
| `CORS_ORIGIN`       | `*`                          | ❌       | Allowed CORS origins |

## Configuration Sections

### Database Configuration

| Variable                 | Type   | Default    | Required     |
| ------------------------ | ------ | ---------- | ------------ |
| `POSTGRES_USER`          | string | `obiente`  | ❌           |
| `POSTGRES_PASSWORD`      | string | -          | ✅           |
| `POSTGRES_DB`            | string | `obiente`  | ❌           |
| `DB_HOST`                | string | `postgres` | ❌           |
| `DB_PORT`                | number | `5432`     | ❌           |
| `REPLICATION_PASSWORD`   | string | -          | ❌ (HA only) |
| `PATRONI_ADMIN_PASSWORD` | string | -          | ❌ (HA only) |

**Example:**

```bash
POSTGRES_USER=obiente
POSTGRES_PASSWORD=secure_random_password_here
POSTGRES_DB=obiente
```

### API Configuration

| Variable      | Type   | Default | Required |
| ------------- | ------ | ------- | -------- |
| `GO_API_PORT` | number | `3001`  | ❌       |
| `LOG_LEVEL`   | string | `debug` | ❌       |

**Log Levels:**

- `debug` - Verbose logging for development
- `info` - Standard production logging
- `warn` - Only warnings and errors
- `error` - Only errors

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

### Orchestration

| Variable                   | Type   | Default        | Required |
| -------------------------- | ------ | -------------- | -------- |
| `DEPLOYMENT_STRATEGY`      | string | `least-loaded` | ❌       |
| `MAX_DEPLOYMENTS_PER_NODE` | number | `50`           | ❌       |

**Deployment Strategies:**

- `least-loaded` - Deploy to node with least resources
- `round-robin` - Cycle through nodes
- `resource-based` - Match resources to node capacity

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

### Security

| Variable               | Type   | Default | Required |
| ---------------------- | ------ | ------- | -------- |
| `JWT_SECRET`           | string | -       | ❌       |
| `SESSION_SECRET`       | string | -       | ❌       |
| `RATE_LIMIT_WINDOW_MS` | number | `60000` | ❌       |
| `RATE_LIMIT_MAX`       | number | `100`   | ❌       |

**Generate Secrets:**

```bash
# Generate a secure secret
openssl rand -hex 32

# Add to .env
JWT_SECRET=<generated_value>
SESSION_SECRET=<generated_value>
```

### Monitoring

| Variable           | Type   | Default | Required     |
| ------------------ | ------ | ------- | ------------ |
| `GRAFANA_PASSWORD` | string | -       | ❌ (HA only) |

## Environment File Templates

### Local Development (.env)

```bash
POSTGRES_USER=obiente
POSTGRES_PASSWORD=local_dev_password
POSTGRES_DB=obiente
LOG_LEVEL=debug
CORS_ORIGIN=*
DISABLE_AUTH=true
```

### Production (.env)

```bash
POSTGRES_USER=obiente
POSTGRES_PASSWORD=<strong_random_password>
POSTGRES_DB=obiente
LOG_LEVEL=info
CORS_ORIGIN=https://obiente.cloud
ZITADEL_URL=https://auth.obiente.cloud
DOMAIN=obiente.cloud
ACME_EMAIL=admin@obiente.cloud
JWT_SECRET=<generated_secret>
SESSION_SECRET=<generated_secret>
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
