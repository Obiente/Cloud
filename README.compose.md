# Docker Compose Files

This project uses multiple Docker Compose files for different deployment scenarios. All compose files share common environment variable blocks defined in `docker-compose.base.yml`.

## File Structure

- **`docker-compose.base.yml`** - Contains all `x-common-*` environment variable anchors (REQUIRED)
- **`docker-compose.yml`** - Main compose file for local development
- **`docker-compose.swarm.yml`** - Docker Swarm deployment
- **`docker-compose.swarm.ha.yml`** - High Availability Docker Swarm deployment
- **`docker-compose.swarm.dev.yml`** - Development Docker Swarm deployment

## Usage

**IMPORTANT:** YAML anchors don't work across files with Docker Compose's `-f` flag or `include` feature. We provide a wrapper script to merge files.

### Local Development

```bash
# Option 1: Use the wrapper script (recommended)
./scripts/docker-compose-wrapper.sh docker-compose.yml up

# Option 2: Manual merge (for one-time use)
cat docker-compose.base.yml docker-compose.yml > /tmp/merged.yml
docker compose -f /tmp/merged.yml up
```

### Docker Swarm

For Swarm deployments, you'll need to merge files manually or use a CI/CD script:

```bash
# Merge and deploy
cat docker-compose.base.yml docker-compose.swarm.yml > /tmp/swarm-merged.yml
docker stack deploy -c /tmp/swarm-merged.yml obiente
```

**Note:** This is a Docker Compose limitation - YAML anchors are processed per-file before merging, so anchors from one file aren't available in another. The wrapper script merges files before passing to Docker Compose.

## Common Environment Variables

All common environment variable blocks are defined in `docker-compose.base.yml`:

- `x-common-database` - Database connection settings
- `x-common-metrics-db` - Metrics database (TimescaleDB) settings
- `x-common-auth` - Authentication and authorization settings
- `x-common-smtp` - Email delivery settings
- `x-common-dashboard` - Dashboard and UI settings
- `x-common-stripe` - Stripe payment processing
- `x-common-redis` - Redis connection settings
- `x-common-orchestrator` - Orchestration and deployment settings
- `x-common-vps` - VPS provisioning settings
- `x-common-dns-delegation` - DNS delegation settings
- `x-common-dns` - DNS service settings
- `x-common-security` - Security settings
- `x-common-github` - GitHub integration settings

## Updating Common Variables

When adding or modifying common environment variables:

1. Update `docker-compose.base.yml`
2. All other compose files will automatically use the updated anchors when loaded with `-f docker-compose.base.yml`
