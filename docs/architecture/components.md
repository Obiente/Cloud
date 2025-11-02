# Architecture Components

This page documents the core components of Obiente Cloud.

## Control Plane Components

### Go API

The main API service that handles all deployment operations.

- **ConnectRPC** service for type-safe API calls
- **Deployment management**: Create, update, delete deployments
- **Docker integration**: Direct container management via Docker API
- **Authentication**: Zitadel OIDC integration
- **Metrics streaming**: Real-time metrics delivery to clients
- **Global mode**: One instance per node for direct Docker access

**Endpoints:**
- `/health` - Health check including metrics system status
- `/metrics/observability` - Metrics collection statistics

### Orchestrator

The orchestration service handles deployment placement and metrics collection.

- **Node selection**: Strategies (least-loaded, round-robin, resource-based)
- **Metrics collection**: Parallel Docker stats collection
- **Health monitoring**: Container health checks
- **Cleanup tasks**: Old metrics aggregation and deletion

### Metrics Streamer

Production-ready metrics collection and streaming system.

**Features:**
- Live metrics streaming (5-second collection intervals)
- In-memory caching for fast UI access
- Batch storage to TimescaleDB (60-second aggregation)
- Circuit breaker for Docker API protection
- Exponential backoff retry mechanism
- Graceful degradation under load
- Health monitoring and alerting
- Backpressure handling for slow subscribers

**Architecture:**
```
MetricsStreamer
├── Parallel Workers (configurable, default: 50)
├── Live Cache (in-memory, 5-minute retention)
├── Subscribers (channel-based streaming)
├── Circuit Breaker (protects Docker API)
├── Retry Queue (failed database writes)
└── Health Monitor (failure detection)
```

## Data Plane Components

### PostgreSQL

Main relational database for application metadata.

**Single Instance:**
- Standard PostgreSQL for development/small deployments

**High Availability:**
- 3-node PostgreSQL cluster with Patroni + etcd
- PgPool for connection pooling and load balancing
- Automatic failover (< 30 seconds)

**Stores:**
- User accounts and organizations
- Projects and deployments
- Deployment locations
- Routing configuration

### TimescaleDB (Metrics Database)

Separate time-series database optimized for metrics storage.

**Single Instance:**
- TimescaleDB for development/small deployments

**High Availability:**
- 3-node TimescaleDB cluster with Patroni + etcd
- `metrics-pgpool` for connection pooling
- Automatic failover
- Mirrors PostgreSQL HA setup

**Features:**
- Hypertable partitioning for performance
- Time-series optimized queries
- Automatic data retention policies
- Aggregated hourly metrics storage

**Stores:**
- Container metrics (CPU, memory, network, disk)
- Aggregated hourly usage statistics
- Historical performance data

**Fallback:**
- Uses main PostgreSQL if TimescaleDB unavailable

### Redis

Distributed caching and session storage.

**Single Instance:**
- Standard Redis for development

**High Availability:**
- 3-node Redis cluster
- Automatic resharding
- Replication and failover

**Use Cases:**
- Session storage
- API response caching
- Job queues
- Rate limiting state

### Traefik

Dynamic reverse proxy and load balancer.

**Features:**
- Service discovery via Docker Swarm
- Automatic HTTPS (Let's Encrypt)
- Dynamic routing configuration
- Load balancing
- Middleware support (rate limiting, auth, compression)

## Service Registry

Tracks deployment locations across the cluster.

- Real-time container tracking
- Node metadata and health
- Periodic reconciliation with Docker
- Deployment location queries

---

[← Back to Architecture](index.md)
