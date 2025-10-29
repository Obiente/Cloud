# Development Setup

This guide will help you set up Obiente Cloud for local development.

## Prerequisites

- Docker and Docker Compose
- Go 1.21+
- Node.js 18+ (for frontend development)
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
docker-compose up -d postgres

# Optional: enable Redis in docker-compose.yml first, then start it
# docker-compose up -d redis
```

### 3. Run API Locally

```bash
cd apps/api
go run main.go
```

The API will be available at `http://localhost:3001`.

## Development Workflow

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests in specific package
go test ./internal/database/...

# Run with coverage
go test -cover ./...
```

### Building

```bash
# Build API binary
cd apps/api
go build -o bin/api main.go

# Build Docker image
docker build -f apps/api/Dockerfile -t obiente/cloud-api:latest .
```

### Rebuilding Docker Images

**Important:** Docker Compose does NOT automatically rebuild images when code changes. After making code changes, you must rebuild:

```bash
# Build and restart in one command (recommended)
docker-compose up -d --build api

# Or rebuild separately
docker-compose build api
docker-compose restart api

# Force full rebuild (ignores cache)
docker-compose build --no-cache api
docker-compose restart api
```

**Note:** If running the API locally (not in Docker), code changes are picked up automatically on restart.

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
docker-compose logs postgres
```

### Database Access

```bash
# Connect to PostgreSQL (container name from docker-compose.yml)
docker exec -it obiente-postgres psql -U obiente-postgres -d obiente
```

## Troubleshooting

See [Troubleshooting Guide](../guides/troubleshooting.md) for common issues.

---

[← Back to Getting Started](index.md)

