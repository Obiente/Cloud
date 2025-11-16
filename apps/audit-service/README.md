# Audit Service

Standalone microservice for audit log management.

## Overview

This is the first microservice extracted from the monolithic API. It handles:
- Listing audit logs with filtering
- Getting individual audit log entries
- Querying TimescaleDB for audit log data

## Port

Default port: `3010` (configurable via `PORT` environment variable)

## Environment Variables

Required:
- `DB_HOST` - PostgreSQL host (default: `postgres`)
- `DB_PORT` - PostgreSQL port (default: `5432`)
- `DB_USER` - Database user
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name
- `METRICS_DB_HOST` - TimescaleDB host (default: `timescaledb`)
- `METRICS_DB_PORT` - TimescaleDB port (default: `5432`)
- `METRICS_DB_USER` - TimescaleDB user
- `METRICS_DB_PASSWORD` - TimescaleDB password
- `METRICS_DB_NAME` - TimescaleDB database name

Optional:
- `PORT` - Service port (default: `3010`)
- `LOG_LEVEL` - Logging level (default: `info`)
- `CORS_ORIGIN` - CORS origin (default: `*`)
- `ZITADEL_URL` - Zitadel URL for authentication
- `ZITADEL_CLIENT_ID` - Zitadel client ID
- `DISABLE_AUTH` - Disable authentication (default: `false`)

## Building

```bash
docker build -t audit-service:latest -f apps/audit-service/Dockerfile .
```

## Running

```bash
# Set environment variables
export DB_HOST=postgres
export DB_PORT=5432
export DB_USER=obiente_postgres
export DB_PASSWORD=your_password
export DB_NAME=obiente
export METRICS_DB_HOST=timescaledb
export METRICS_DB_PORT=5432
export METRICS_DB_USER=obiente_postgres
export METRICS_DB_PASSWORD=your_password
export METRICS_DB_NAME=obiente_metrics

# Run
./audit-service
```

## Health Check

The service exposes a health check endpoint at `/health`:

```bash
curl http://localhost:3010/health
```

## API Endpoints

The service exposes Connect RPC endpoints:
- `obiente.cloud.audit.v1.AuditService/ListAuditLogs`
- `obiente.cloud.audit.v1.AuditService/GetAuditLog`

## Migration Status

✅ **Phase 1 Complete**: Audit service extracted and running independently

The existing monolithic API still has the audit service registered, so both can run in parallel during migration.

