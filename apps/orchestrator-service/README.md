# Orchestrator Service

Microservice for orchestrating Docker containers, managing deployments, game servers, and collecting metrics.

## Features

- Docker container orchestration
- Deployment management
- Game server management
- Metrics collection and aggregation
- Health checks
- Node coordination
- Usage statistics aggregation

## Port

Default port: `3007`

## Environment Variables

See shared configuration in `docker-compose.yml` for common variables.

### Service-Specific Variables

- `PORT` - Service port (default: 3007)
- `ORCHESTRATOR_SYNC_INTERVAL` - Interval for syncing node state (default: 30s)
- `REDIS_URL` - Redis connection URL (for caching)

## Endpoints

- `/health` - Health check endpoint
- `/` - Service info

## Dependencies

- PostgreSQL (main database)
- TimescaleDB (metrics database)
- Redis (for caching)
- Docker (for container management)

## Notes

- This service requires Docker socket access to manage containers
- It coordinates with deployment and game server services
- Metrics collection runs in the background
- Health checks monitor container status across nodes

