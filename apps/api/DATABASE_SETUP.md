# Database Setup

This document describes how to set up PostgreSQL and Redis for the Obiente Cloud API.

## Quick Start

1. **Start PostgreSQL and Redis with Docker Compose:**

```bash
docker compose up -d
```

2. **Create a `.env` file (optional):**

```bash
# Copy this into a .env file in the project root
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=obiente
REDIS_URL=redis://localhost:6379
PORT=3001
```

The API will automatically load environment variables from `.env` file or use these defaults:

- `DB_HOST=localhost`
- `DB_PORT=5432`
- `DB_USER=obiente-postgres`
- `DB_PASSWORD=obiente-postgres`
- `DB_NAME=obiente`
- `REDIS_URL=redis://localhost:6379`

3. **Run the API:**

```bash
nx dev api
```

The database schema will be automatically migrated on first run.

## Docker Compose Services

- **PostgreSQL**: Running on port 5432
- **Redis**: Running on port 6379

## Database Schema

The `deployments` table is automatically created with the following columns:

- `id` - Primary key
- `name` - Deployment name
- `domain` - Deployment domain
- `custom_domains` - JSON array of custom domains
- `type` - Deployment type enum
- `status` - Deployment status enum
- `environment` - Environment enum
- `organization_id` - Links deployments to organizations
- `created_by` - User who created the deployment
- And various other fields...

## Redis Caching

- Deployment lookups are cached for 5 minutes
- Cache is automatically invalidated on updates/deletes
- If Redis is unavailable, the app continues without caching

## Manual Setup (without Docker)

### PostgreSQL

```bash
createdb obiente
psql obiente < schema.sql
```

### Redis

```bash
redis-server
```
