# VPS Service

Microservice for managing Virtual Private Server instances.

## Features

- VPS instance management
- SSH proxy server
- Terminal WebSocket access
- Proxmox integration
- Firewall management
- SSH key management

## Port

Default port: `3008`

## Environment Variables

See shared configuration in `docker-compose.yml` for common variables.

### Service-Specific Variables

- `PORT` - Service port (default: 3008)

## Endpoints

- `/obiente.cloud.vps.v1.VPSService/*` - Connect RPC endpoints
- `/terminal/ws` - WebSocket terminal endpoint
- `/ssh/` - SSH proxy endpoint
- `/health` - Health check endpoint
- `/` - Service info

## Dependencies

- PostgreSQL (main database)
- Orchestrator Service (for VPS management)

## Notes

- This service requires access to Proxmox API for VPS operations
- The orchestrator service should be running for full functionality
- SSH proxy requires proper network configuration

