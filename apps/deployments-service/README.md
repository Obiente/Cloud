# Deployments Service

Microservice for managing application deployments, builds, and containers.

## Features

- Deployment CRUD operations
- Build management and history
- Container lifecycle management
- Log streaming
- Terminal WebSocket access
- Health monitoring
- Metrics collection
- Docker Compose support

## Port

Default port: `3005`

## Environment Variables

See shared configuration in `docker-compose.yml` for common variables.

### Service-Specific Variables

- `PORT` - Service port (default: 3005)
- `REDIS_URL` - Redis connection URL (for build logs)

## Endpoints

- `/obiente.cloud.deployments.v1.DeploymentService/*` - Connect RPC endpoints
- `/terminal/ws` - WebSocket terminal endpoint
- `/health` - Health check endpoint
- `/` - Service info

## Dependencies

- PostgreSQL (main database)
- TimescaleDB (metrics database)
- Redis (for build logs and caching)
- Docker (for container management)
- Orchestrator Service (for deployment management)

## Notes

- This service requires Docker access to manage containers
- The orchestrator service should be running for full functionality
- If orchestrator is not available, the service will attempt to create a deployment manager directly

