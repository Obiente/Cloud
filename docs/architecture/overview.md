# Obiente Cloud - Distributed Architecture

## Overview

Obiente Cloud is a **Platform-as-a-Service (PaaS)** similar to Vercel, designed to run user deployments across a distributed Docker Swarm cluster. The platform automatically orchestrates, routes, and monitors user applications across multiple nodes.

## Architecture Components

### 1. Control Plane (Obiente Cloud Services)

These are the core services that manage the platform:

#### API Services

- **API (`api`)**: ConnectRPC service handling deployment operations
  - Manages Docker containers via Docker API
  - Tracks deployment locations across nodes
  - Handles deployment lifecycle (create, update, delete)
  - User authentication and authorization (Zitadel)
  - Project and deployment management
  - One instance per node (global mode) for direct Docker access

#### Deployment Orchestrator

- Decides which node should host new deployments
- Strategies: least-loaded, round-robin, resource-based
- Monitors node health and capacity
- Handles deployment migration when needed
- Integrates with Docker Swarm API for placement

### 2. Data Plane (High Availability Storage)

#### PostgreSQL Cluster (Patroni + etcd)

- **3-node PostgreSQL cluster** with automatic failover
- **Patroni**: Manages PostgreSQL replication and failover
- **etcd**: Distributed consensus for leader election
- **PgPool**: Connection pooling and load balancing
- Stores:
  - User accounts and organizations
  - Projects and deployments metadata
  - Deployment locations (node_id, container_id, status)
  - Node resource usage and capacity
  - Routing configuration

#### Metrics Database (TimescaleDB)

- **Separate TimescaleDB instance** optimized for time-series data
- **Production HA**: 3-node TimescaleDB cluster with Patroni + etcd (mirrors PostgreSQL setup)
- **PgPool**: `metrics-pgpool` for connection pooling and load balancing
- Stores:
  - Container metrics (CPU, memory, network, disk I/O)
  - Aggregated hourly usage statistics
  - Historical deployment performance data
- **Benefits**:
  - Isolated from main database workload
  - Optimized for time-series queries and aggregations
  - Automatic hypertable partitioning for performance
  - Falls back to main PostgreSQL if TimescaleDB unavailable

#### Redis Cluster

- **3-node Redis cluster** for distributed caching
- Use cases:
  - Session storage
  - API response caching
  - Job queue for deployment operations
  - Real-time deployment status updates
  - Rate limiting state

### 3. User Deployments (Data Plane)

User applications run as Docker containers distributed across worker nodes:

```
Node 1:              Node 2:              Node 3:
- user-app-a-v1     - user-app-b-v1     - user-app-c-v1
- user-app-d-v2     - user-app-e-v1     - user-app-f-v1
- user-app-g-v1     - user-app-h-v2     - user-app-i-v1
```

Each deployment is tracked in the database:

```sql
deployment_locations table:
- deployment_id
- node_id (which Swarm node)
- container_id (Docker container ID)
- status (running/stopped/failed)
- port (assigned port)
- domain (custom domain)
- health_status
- resource_usage
```

### 4. Traffic Routing (Traefik)

**Traefik** acts as the dynamic reverse proxy:

- **Service Discovery**: Automatically discovers services via Docker Swarm API
- **Dynamic Routing**: Routes traffic based on domains and paths
- **SSL/TLS**: Automatic HTTPS via Let's Encrypt
- **Load Balancing**: Distributes traffic across deployment replicas
- **Middleware**: Rate limiting, authentication, compression

**Domain-Based Routing:**

Obiente Cloud uses domain-based routing for service-to-service communication, enabling:

- **Cross-node communication**: Services on different nodes communicate via domains
- **Cross-network communication**: Works with VPNs, service meshes, and custom networks
- **Multi-cluster support**: Multiple Swarm clusters can share the same domain
- **Automatic load balancing**: Traefik load balances across all healthy replicas

By default, services communicate via HTTPS through Traefik (e.g., `https://auth-service.${DOMAIN}`) instead of direct service-to-service HTTP. This enables distributed deployments across nodes, networks, and clusters.

See [Domain-Based Routing Guide](../guides/domain-based-routing.md) for detailed configuration.

Routing flow:

```
User Request (app.example.com)
       ↓
Traefik (checks routing table)
       ↓
Looks up deployment_routing table in DB
       ↓
Routes to correct node + container
       ↓
User's deployed application
```

Service-to-service communication:

```
Service A (Node 1)
       ↓
Requests: https://auth-service.${DOMAIN}
       ↓
Traefik (load balances across all nodes)
       ↓
Service B (Node 2 or Node 3)
```

### 5. Monitoring & Observability

#### Metrics Collection System

The platform uses a production-ready metrics system with:

- **Live Metrics Streaming**: Real-time container stats collection (5-second intervals)
- **In-Memory Caching**: Fast access to recent metrics for UI streaming
- **Batch Storage**: Aggregated metrics written to TimescaleDB every minute
- **Resilience Features**:
  - Circuit breaker pattern for Docker API protection
  - Exponential backoff retry mechanism
  - Automatic graceful degradation under load
  - Health monitoring with failure detection
  - Backpressure handling for slow subscribers

#### Metrics Flow

```
Container Stats (Docker API)
       ↓
Metrics Streamer (Parallel Collection)
       ↓
Live Cache (Memory) → Subscribers (UI streaming)
       ↓
Aggregation (Every 60s)
       ↓
TimescaleDB (Historical Storage)
```

#### Observability Endpoints

- **`/health`**: Health check including metrics system status
- **`/metrics/observability`**: Real-time metrics collection statistics
  - Collection rates and error counts
  - Database write success/failure rates
  - Circuit breaker state
  - Subscriber and cache metrics

#### External Monitoring

- **Prometheus**: Scrapes metrics from all services and nodes (optional)
- **Grafana**: Visualizes metrics and creates dashboards (optional)
- Metrics tracked:
  - Node resource usage (CPU, memory, disk)
  - Deployment resource consumption (real-time and historical)
  - API request latency and throughput
  - Database performance
  - Network traffic per deployment
  - Metrics collection health and performance

## How Deployments Work

### Deployment Flow

1. **User initiates deployment** via API

   ```
   POST /api/v1/deployments
   { project_id, git_repo, branch, env_vars }
   ```

2. **Orchestrator selects target node**

   ```go
   func SelectNode(strategy string) (*Node, error) {
       nodes := GetAvailableNodes()
       switch strategy {
       case "least-loaded":
           return nodes.WithLowestCPU()
       case "round-robin":
           return nodes.NextInRotation()
       }
   }
   ```

3. **Go API on target node creates container**

   ```go
   container := dockerClient.CreateContainer(deployment.Image, deployment.Config)
   dockerClient.StartContainer(container.ID)
   ```

4. **Location is recorded in database**

   ```go
   location := DeploymentLocation{
       DeploymentID: deployment.ID,
       NodeID:       currentNode.ID,
       ContainerID:  container.ID,
       Status:       "running",
       Port:         assignedPort,
   }
   db.RecordDeploymentLocation(location)
   ```

5. **Routing is configured**

   ```go
   routing := DeploymentRouting{
       DeploymentID: deployment.ID,
       Domain:       deployment.Domain,
       TargetPort:   assignedPort,
       SSLEnabled:   true,
   }
   db.UpsertDeploymentRouting(routing)
   ```

6. **Traefik discovers new container** via Docker labels
   ```yaml
   labels:
     - "traefik.enable=true"
     - "traefik.http.routers.{deployment-id}.rule=Host(`{domain}`)"
     - "traefik.http.services.{deployment-id}.loadbalancer.server.port={port}"
   ```

### Tracking Deployments Across Nodes

The system maintains several tracking mechanisms:

#### 1. Database Tables

**`deployment_locations`**: Real-time deployment locations

```sql
SELECT * FROM deployment_locations WHERE deployment_id = 'dep_123';
-- Result: node_id='node-worker-2', container_id='abc123', status='running'
```

**`node_metadata`**: Cluster node information

```sql
SELECT * FROM node_metadata WHERE availability='active' ORDER BY deployment_count ASC;
-- Returns nodes sorted by current load
```

**`deployment_routing`**: Traffic routing configuration

```sql
SELECT * FROM deployment_routing WHERE domain = 'myapp.com';
-- Returns: deployment_id, target_port, load_balancer_algo
```

#### 2. Docker Swarm API

The Go API queries Docker Swarm directly:

```go
// Get all nodes in the cluster
nodes, _ := dockerClient.NodeList(ctx, types.NodeListOptions{})

// Get containers on current node
containers, _ := dockerClient.ContainerList(ctx, types.ContainerListOptions{
    Filters: filters.NewArgs(
        filters.Arg("label", "cloud.obiente.deployment=true"),
    ),
})
```

#### 3. Periodic Reconciliation

Background job runs every minute:

```go
func ReconcileDeployments() {
    // 1. Query actual containers from Docker
    actualContainers := getAllContainersFromAllNodes()

    // 2. Compare with database records
    dbRecords := db.GetAllDeploymentLocations()

    // 3. Update discrepancies
    for _, container := range actualContainers {
        if !existsInDB(container) {
            db.RecordDeploymentLocation(container)
        }
    }

    // 4. Clean up stale records
    for _, record := range dbRecords {
        if !existsInCluster(record) {
            db.RemoveDeploymentLocation(record.ContainerID)
        }
    }
}
```

## Node Labeling Strategy

Nodes are labeled for deployment placement:

```bash
# Label nodes for PostgreSQL replicas
docker node update --label-add postgres.replica=1 node-1
docker node update --label-add postgres.replica=2 node-2
docker node update --label-add postgres.replica=3 node-3

# Label nodes for Redis
docker node update --label-add redis=1 node-1
docker node update --label-add redis=2 node-2
docker node update --label-add redis=3 node-3

# Label compute nodes for user deployments
docker node update --label-add compute=true node-4
docker node update --label-add compute=true node-5
docker node update --label-add compute=true node-6
```

## Scaling Strategy

### Horizontal Scaling

**Control Plane Services:**

- API services: Scale replicas based on request load
- Orchestrator: 2-3 replicas for redundancy

**Data Plane:**

- PostgreSQL: 3-5 replicas (1 primary + 2-4 replicas)
- Redis: 3-6 nodes for cluster

**User Deployments:**

- Add more worker nodes as capacity increases
- Each node can handle 50-100 deployments (configurable)

### Vertical Scaling

Increase resources per node based on workload:

- Manager nodes: 4-8 CPU, 8-16GB RAM
- Worker nodes: 8-16 CPU, 16-32GB RAM
- Database nodes: 4-8 CPU, 16-32GB RAM

## High Availability Features

1. **PostgreSQL**: Automatic failover within seconds via Patroni
2. **Redis**: Cluster mode with automatic resharding
3. **API Services**: Multiple replicas behind load balancer
4. **Deployments**: Can be replicated across multiple nodes
5. **Traefik**: Multiple instances for routing redundancy

## Security Considerations

1. **Network Isolation**: Overlay network isolates services
2. **TLS Everywhere**: Automatic HTTPS via Let's Encrypt
3. **Resource Limits**: CPU/memory limits prevent resource exhaustion
4. **Authentication**: Zitadel for user authentication
5. **API Security**: Rate limiting, CORS, helmet middleware
6. **Database**: Connection pooling, prepared statements
7. **Secrets Management**: Docker secrets for sensitive data

## Disaster Recovery

### Backup Strategy

- **PostgreSQL**: Continuous archiving + daily snapshots
- **Redis**: AOF persistence + periodic snapshots
- **Deployment metadata**: Replicated across 3 nodes

### Recovery Procedures

1. Database failover: Automatic via Patroni (< 30 seconds)
2. Node failure: Swarm reschedules containers automatically
3. Complete cluster failure: Restore from backups

## Monitoring Dashboards

### Node Health Dashboard

- CPU usage per node
- Memory usage per node
- Deployment count per node
- Network throughput

### Deployment Dashboard

- Total deployments
- Deployments per project
- Resource usage per deployment
- Request latency per deployment

### Platform Health Dashboard

- API response times
- Database query performance
- Cache hit rates
- Error rates

## Performance Optimization

1. **Database Connection Pooling**: PgPool with 100 connections
2. **Caching**: Redis for frequently accessed data (5min TTL)
3. **CDN**: Serve static assets via CDN
4. **Image Optimization**: Use multi-stage Docker builds
5. **Lazy Loading**: Load deployment metadata on-demand
6. **Batch Operations**: Bulk update deployment status

## Cost Optimization

1. **Resource Limits**: Prevent over-provisioning
2. **Auto-scaling**: Scale down during low usage
3. **Spot Instances**: Use for non-critical worker nodes
4. **Caching**: Reduce database queries
5. **Compression**: Enable gzip for API responses

## Future Enhancements

1. **Multi-region deployments**: Deploy to multiple geographic regions
2. **Edge computing**: Run deployments closer to users
3. **Serverless functions**: Support for FaaS workloads
4. **Auto-scaling**: Automatically scale deployments based on traffic
5. **Blue-green deployments**: Zero-downtime deployment updates
6. **A/B testing**: Traffic splitting between deployment versions
7. **Cost analytics**: Per-deployment resource usage and billing
