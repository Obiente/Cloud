# Support Service

Standalone microservice for support ticket management.

## Overview

This is the second microservice extracted from the monolithic API. It handles:
- Creating support tickets
- Listing tickets with filtering
- Getting individual tickets
- Updating tickets (status, priority, assignee)
- Adding comments to tickets
- Listing comments for a ticket

## Port

Default port: `3009` (configurable via `PORT` environment variable)

## Environment Variables

Required:
- `DB_HOST` - PostgreSQL host (default: `postgres`)
- `DB_PORT` - PostgreSQL port (default: `5432`)
- `DB_USER` - Database user
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name

Optional:
- `PORT` - Service port (default: `3009`)
- `LOG_LEVEL` - Logging level (default: `info`)
- `CORS_ORIGIN` - CORS origin (default: `*`)
- `ZITADEL_URL` - Zitadel URL for authentication
- `ZITADEL_CLIENT_ID` - Zitadel client ID
- `DISABLE_AUTH` - Disable authentication (default: `false`)

## Building

```bash
docker build -t support-service:latest -f apps/support-service/Dockerfile .
```

## Running

```bash
# Set environment variables
export DB_HOST=postgres
export DB_PORT=5432
export DB_USER=obiente_postgres
export DB_PASSWORD=your_password
export DB_NAME=obiente

# Run
./support-service
```

## Health Check

The service exposes a health check endpoint at `/health`:

```bash
curl http://localhost:3009/health
```

## API Endpoints

The service exposes Connect RPC endpoints:
- `obiente.cloud.support.v1.SupportService/CreateTicket`
- `obiente.cloud.support.v1.SupportService/ListTickets`
- `obiente.cloud.support.v1.SupportService/GetTicket`
- `obiente.cloud.support.v1.SupportService/UpdateTicket`
- `obiente.cloud.support.v1.SupportService/AddComment`
- `obiente.cloud.support.v1.SupportService/ListComments`

## Migration Status

✅ **Phase 2 Complete**: Support service extracted and running independently

The existing monolithic API still has the support service registered, so both can run in parallel during migration.

