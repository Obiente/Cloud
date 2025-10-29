# Configuration Guide

Configure your Obiente Cloud deployment to fit your needs.

## Environment File

The primary configuration is through the `.env` file.

Copy the example file:

```bash
cp env.swarm.example .env
```

## Basic Configuration

### Database

```bash
POSTGRES_USER=obiente
POSTGRES_PASSWORD=your_secure_password
POSTGRES_DB=obiente
```

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

## Security Configuration

### CORS

```bash
# Allow specific origins
CORS_ORIGIN=https://app.obiente.cloud,https://dashboard.obiente.cloud

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
JWT_SECRET=<generated_value>
SESSION_SECRET=<generated_value>
```

## Production Configuration

For production deployments:

1. **Change all defaults** - Update all default passwords
2. **Set LOG_LEVEL=info** - Reduce log verbosity
3. **Configure CORS properly** - Use specific origins
4. **Enable authentication** - Set `DISABLE_AUTH=false`
5. **Set up SSL** - Configure domain and ACME email

## Reference

See [Environment Variables Reference](../reference/environment-variables.md) for complete configuration options.

---

[← Back to Getting Started](index.md)

