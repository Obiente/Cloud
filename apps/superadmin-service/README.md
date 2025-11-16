# Superadmin Service

Microservice for superadmin operations and system management.

## Features

- Organization management
- User management
- System configuration
- DNS management
- Webhook events viewing
- Invoice management
- System overview and statistics

## Port

Default port: `3011`

## Environment Variables

See shared configuration in `docker-compose.yml` for common variables.

### Service-Specific Variables

- `PORT` - Service port (default: 3009)

## Endpoints

- `/obiente.cloud.superadmin.v1.SuperadminService/*` - Connect RPC endpoints
- `/health` - Health check endpoint
- `/` - Service info

## Dependencies

- PostgreSQL (main database)
- TimescaleDB (metrics database)
- Stripe API (for invoice management)

## Notes

- This service requires superadmin role for all operations
- Accesses data from all other services for system-wide operations

