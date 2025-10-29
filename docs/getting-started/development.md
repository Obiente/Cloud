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
CORS_ORIGIN=*
DISABLE_AUTH=true
```

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

