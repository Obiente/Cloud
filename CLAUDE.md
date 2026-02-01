# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Obiente Cloud is a distributed Platform-as-a-Service (PaaS) built as an Nx monorepo with 14 Go microservices and a Nuxt 4 dashboard. Services communicate via ConnectRPC (protocol buffers). Deployed on Docker Swarm.

## Build & Development Commands

### Package Management
```bash
pnpm install                    # Install all JS/TS dependencies
```

### Dashboard (Nuxt 4 frontend)
```bash
nx serve dashboard              # Dev server on port 3000
nx nuxt:build dashboard         # Production build
nx lint dashboard               # ESLint
nx typecheck dashboard          # Type checking
```

### Go Services
```bash
cd apps/<service-name>
go run main.go                  # Run locally
go build                        # Build binary
go test ./...                   # Run tests
```

### Protocol Buffers
```bash
cd packages/proto
pnpm build                      # Regenerate all proto code (buf generate)
```
Generated Go code goes to `apps/shared/proto/`, TypeScript to `packages/proto/src/generated/`.

### Docker
```bash
docker compose up -d                          # Local dev (all services)
docker build -f apps/<svc>/Dockerfile -t ghcr.io/obiente/cloud-<svc>:latest .  # Build image
./scripts/deploy-swarm-dev.sh                 # Swarm dev deploy
./scripts/deploy-swarm-dev.sh -b              # Build + deploy
```

### Nx
Always prefer running tasks through `nx` rather than underlying tooling directly. Use `nx run`, `nx run-many`, `nx affected`.

## Architecture

### Service Ports
| Service | Port |
|---------|------|
| Dashboard | 3000 |
| API Gateway | 3001 |
| Auth | 3002 |
| Organizations | 3003 |
| Billing | 3004 |
| Deployments | 3005 |
| GameServers | 3006 |
| Orchestrator | 3007 |
| VPS | 3008 |
| Support | 3009 |
| Audit | 3010 |
| Superadmin | 3011 |
| Notifications | 3012 |
| DNS | 8053 |

### Key Architectural Patterns

- **API Gateway** routes all external requests to backend services. Supports both direct service routing and Traefik-based routing.
- **ConnectRPC** is used for all inter-service communication. Proto definitions live in `packages/proto/proto/obiente/cloud/`. Buf generates both Go and TypeScript clients.
- **Go workspace** (`go.work`) links all 15 Go modules. Shared code is in `apps/shared/` with packages for auth, database, docker, middleware, orchestrator, quota, etc.
- **Auth** is handled via Zitadel integration with RBAC. The auth-service validates tokens and manages roles/permissions.
- **Orchestrator** handles intelligent node selection and load balancing across the Docker Swarm cluster.
- **Database**: PostgreSQL (primary), TimescaleDB (metrics/audit), Redis (cache, build logs).
- **Dashboard** uses Nuxt 4, Vue 3, Tailwind CSS v4, Pinia for state, Ark UI for components, and `@connectrpc/connect-web` for API calls.

### Monorepo Structure
- `apps/` - All microservices + dashboard
- `packages/proto/` - Protobuf definitions and generated code
- `packages/database/` - Drizzle ORM schemas and migrations
- `packages/config/` - Shared ESLint, Prettier, TypeScript configs
- `packages/types/` - Shared TypeScript types
- `tools/nxsh/` - Custom Nx shell executor
- `monitoring/` - Prometheus & Grafana configs
- `scripts/` - Deployment and operational scripts

### Docker Compose Files
- `docker-compose.yml` - Local development
- `docker-compose.base.yml` - Shared env vars (YAML anchors)
- `docker-compose.swarm.yml` - Production swarm
- `docker-compose.swarm.dev.yml` - Dev swarm (must use `docker stack deploy`, not `docker compose`)
- `docker-compose.swarm.ha.yml` - HA production with PostgreSQL cluster
