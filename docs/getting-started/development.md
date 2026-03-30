# Development Setup

This guide will help you set up Obiente Cloud for local development.

## Prerequisites

- Docker and Docker Compose
- Go 1.21+
- Node.js 18+ (for frontend development)
- pnpm 10.x
- Git

## Local Development

### 1. Clone Repository

```bash
git clone https://github.com/obiente/cloud.git
cd cloud
```

### 2. Start Dependencies

Start database and other dependencies:

```bash
pnpm install

# Start PostgreSQL (main database)
docker compose up -d postgres

# Start TimescaleDB (metrics database)
docker compose up -d timescaledb

# Optional: enable Redis in docker-compose.yml first, then start it
# docker compose up -d redis
```

### 3. Run Services Locally

```bash
# Start the dashboard
pnpm exec nx serve dashboard

# In another terminal, run a Go service from its own directory
cd apps/api-gateway
go run .
```

The dashboard will be available through the Nuxt dev server, and Go services run on their configured service ports.

## Development Workflow

### Running Tests

```bash
# Show inferred Nx projects
pnpm exec nx show projects

# Run frontend lint/typecheck through Nx
pnpm exec nx run dashboard:lint
pnpm exec nx run dashboard:typecheck

# Run Go tests from a specific service directory
cd apps/deployments-service
go test ./...

# Run with coverage for a specific Go service
go test -cover ./...
```

### Building

```bash
# Build the dashboard
pnpm exec nx run dashboard:nuxt:build

# Build a service binary
cd apps/api-gateway
go build ./...

# Build a service Docker image
docker build -f apps/api-gateway/Dockerfile -t ghcr.io/obiente/cloud-api-gateway:latest .
```

### Rebuilding Docker Images

**Important:** Docker Compose does NOT automatically rebuild images when code changes. After making code changes, you must rebuild:

```bash
# Build and restart in one command (recommended)
docker compose up -d --build api-gateway

# Or rebuild separately
docker compose build api-gateway
docker compose restart api-gateway

# Force full rebuild (ignores cache)
docker compose build --no-cache api-gateway
docker compose restart api-gateway
```

**Note:** If running a Go service locally (not in Docker), code changes are picked up when you restart that service.

### Hot Reload

Use air for automatic reloading:

```bash
go install github.com/cosmtrek/air@latest
air
```

## Configuration

Create a `.env` file in the project root:

```bash
LOG_LEVEL=debug
# For local development with frontend on localhost:3000
CORS_ORIGIN=http://localhost:3000
DISABLE_AUTH=true

# If using non-standard ports for API
PUBLIC_HTTPS_PORT=2443
```

**Important for CORS:**

- When your frontend runs on `http://localhost:3000`, set `CORS_ORIGIN=http://localhost:3000` (with port)
- The browser sends the exact origin including port in cross-origin requests
- Multiple origins: `CORS_ORIGIN=http://localhost:3000,https://app.example.com`

## Debugging

### View Logs

```bash
# Container logs
docker compose logs postgres
```

### Database Access

```bash
# Connect to PostgreSQL (main database)
docker exec -it obiente-postgres psql -U obiente-postgres -d obiente

# Connect to TimescaleDB (metrics database)
docker exec -it obiente-timescaledb psql -U postgres -d obiente_metrics
```

## Troubleshooting

See [Troubleshooting Guide](../guides/troubleshooting.md) for common issues.

---

[← Back to Getting Started](index.md)
