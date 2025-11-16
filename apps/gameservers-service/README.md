# Game Servers Service

Microservice for managing game server instances.

## Features

- Game server CRUD operations
- Server lifecycle management (start/stop/restart)
- Log streaming
- Terminal WebSocket access
- Metrics collection
- Storage management

## Port

Default port: `3006`

## Environment Variables

See shared configuration in `docker-compose.yml` for common variables.

### Service-Specific Variables

- `PORT` - Service port (default: 3006)

## Endpoints

- `/obiente.cloud.gameservers.v1.GameServerService/*` - Connect RPC endpoints
- `/terminal/ws` - WebSocket terminal endpoint
- `/health` - Health check endpoint
- `/` - Service info

## Dependencies

- PostgreSQL (main database)
- TimescaleDB (metrics database)
- Docker (for container management)
- Orchestrator Service (for game server management)

## Notes

- This service requires Docker access to manage game server containers
- The orchestrator service should be running for full functionality
- Game servers are managed via the orchestrator's GameServerManager

