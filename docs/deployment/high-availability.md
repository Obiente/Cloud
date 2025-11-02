# High Availability Deployment

Guidance for running Obiente Cloud with full HA in production (multi-node, failover, redundancy).

## Overview

The High Availability (HA) deployment provides:

- ✅ **3-node PostgreSQL cluster** with automatic failover (Patroni + etcd + PgPool)
- ✅ **3-node TimescaleDB cluster** for metrics with automatic failover
- ✅ **3-node Redis cluster** with automatic resharding
- ✅ **Multiple API instances** across nodes
- ✅ **Automatic failover** within seconds
- ✅ **Zero-downtime deployments** for platform services

## Architecture

### Database Layer

#### PostgreSQL Cluster

- **Primary**: 1 node (Patroni-managed leader)
- **Replicas**: 2 nodes (standby replicas)
- **Patroni**: Manages replication and automatic failover
- **etcd**: Distributed consensus for leader election (3-node etcd cluster)
- **PgPool**: Connection pooling and load balancing (2 replicas)
- **Failover Time**: < 30 seconds

#### Metrics Database (TimescaleDB) Cluster

- **Primary**: 1 node (Patroni-managed leader)
- **Replicas**: 2 nodes (standby replicas)
- **Patroni**: Manages replication and automatic failover
- **etcd**: Shared with PostgreSQL (3-node etcd cluster)
- **metrics-pgpool**: Dedicated connection pooler (2 replicas)
- **Failover Time**: < 30 seconds

#### Redis Cluster

- **3 Redis nodes** in cluster mode
- **Automatic resharding** on node failure
- **Replication**: Each shard replicated across nodes
- **Failover Time**: < 10 seconds

### Application Layer

#### API Services

- **Multiple replicas** across nodes
- **Global mode**: One instance per node for Docker API access
- **Load balanced**: Via Traefik
- **Health checks**: Automatic container restart on failure

#### Orchestrator Services

- **Single instance** per cluster (runs on manager node)
- **Metrics collection**: Continues even if orchestrator restarts
- **Cleanup tasks**: Resume on restart

### Infrastructure Requirements

**Minimum Nodes:**
- 5 nodes total
  - 3 manager nodes (etcd, PostgreSQL, TimescaleDB, orchestrator)
  - 2 worker nodes (API, deployments)

**Recommended Nodes:**
- 7+ nodes
  - 3 manager nodes (etcd, databases, orchestrator)
  - 4+ worker nodes (API, deployments, load distribution)

**Node Labels Required:**

```bash
# PostgreSQL replicas (manager nodes)
docker node update --label-add postgres.replica=1 <node-1>
docker node update --label-add postgres.replica=2 <node-2>
docker node update --label-add postgres.replica=3 <node-3>

# TimescaleDB replicas (manager nodes)
docker node update --label-add metrics.replica=1 <node-1>
docker node update --label-add metrics.replica=2 <node-2>
docker node update --label-add metrics.replica=3 <node-3>

# Redis nodes
docker node update --label-add redis.replica=1 <node-1>
docker node update --label-add redis.replica=2 <node-2>
docker node update --label-add redis.replica=3 <node-3>
```

## Deployment

### 1. Prepare Nodes

Ensure all nodes meet requirements:
- Docker Engine 20.10+
- Swarm mode enabled
- Network connectivity between nodes
- Persistent volumes configured

### 2. Deploy Stack

```bash
# Deploy HA stack
docker stack deploy -c docker-compose.swarm.ha.yml obiente

# Monitor deployment
docker service ls
docker stack ps obiente
```

### 3. Verify HA Setup

**Check PostgreSQL:**
```bash
docker service ps obiente_pgpool
docker service ps obiente_patroni-1
docker service ps obiente_patroni-2
docker service ps obiente_patroni-3
```

**Check TimescaleDB:**
```bash
docker service ps obiente_metrics-pgpool
docker service ps obiente_metrics-patroni-1
docker service ps obiente_metrics-patroni-2
docker service ps obiente_metrics-patroni-3
```

**Check Redis:**
```bash
docker service ps obiente_redis-1
docker service ps obiente_redis-2
docker service ps obiente_redis-3
```

## Failover Scenarios

### PostgreSQL Primary Failure

1. Patroni detects primary failure
2. etcd coordinates leader election
3. One replica promotes to primary (< 30 seconds)
4. PgPool redirects connections to new primary
5. Applications reconnect automatically

### TimescaleDB Primary Failure

1. Patroni detects primary failure
2. etcd coordinates leader election
3. One replica promotes to primary (< 30 seconds)
4. metrics-pgpool redirects connections to new primary
5. Metrics collection continues without interruption

### Redis Node Failure

1. Cluster detects node failure
2. Automatic resharding redistributes data
3. Remaining nodes continue serving requests
4. No data loss (with replication)

### API Node Failure

1. Docker Swarm detects node failure
2. Containers automatically rescheduled to healthy nodes
3. Traefik updates routing
4. New deployments routed to available nodes

## Monitoring HA Health

### Health Checks

**API Health:**
```bash
curl http://<api-url>/health
```

Returns:
- Database connectivity status
- Metrics system health
- Overall service status

**Metrics Observability:**
```bash
curl http://<api-url>/metrics/observability
```

Returns:
- Metrics collection statistics
- Circuit breaker state
- Health status

### Database Health

**PostgreSQL:**
```bash
docker exec obiente_patroni-1_1 patronictl list
```

**TimescaleDB:**
```bash
docker exec obiente_metrics-patroni-1_1 patronictl list
```

## Backup and Recovery

### PostgreSQL Backups

- Continuous archiving via Patroni
- Daily snapshots recommended
- Point-in-time recovery available

### TimescaleDB Backups

- Same as PostgreSQL (Patroni-managed)
- Metrics retention: Configurable (default: aggregates only)
- Historical data can be exported

### Redis Backups

- AOF persistence enabled
- Periodic snapshots recommended
- Replication provides redundancy

## Configuration

All HA configuration is handled via environment variables in `docker-compose.swarm.ha.yml`.

**Key Variables:**
- `POSTGRES_USER`, `POSTGRES_PASSWORD`
- `METRICS_DB_USER`, `METRICS_DB_PASSWORD`
- `REPLICATION_PASSWORD`
- `PATRONI_ADMIN_PASSWORD`
- `METRICS_REPLICATION_PASSWORD`
- `METRICS_PATRONI_ADMIN_PASSWORD`

See: [Environment Variables Reference](../reference/environment-variables.md)

## Troubleshooting

### Patroni Failover Issues

```bash
# Check Patroni status
docker exec <patroni-container> patronictl list

# Manual failover (if needed)
docker exec <patroni-container> patronictl switchover
```

### Connection Pool Issues

```bash
# Check PgPool status
docker logs obiente_pgpool_1

# Check metrics-pgpool status
docker logs obiente_metrics-pgpool_1
```

### Metrics Collection Issues

```bash
# Check metrics streamer health
curl http://<api-url>/metrics/observability

# Check circuit breaker state
# (see observability endpoint response)
```

## Best Practices

1. **Monitor Failover Events**: Set up alerts for database failovers
2. **Regular Backups**: Daily backups of PostgreSQL and TimescaleDB
3. **Test Failover**: Periodically test failover scenarios
4. **Resource Monitoring**: Monitor node resources and database performance
5. **Network Latency**: Ensure low latency between database nodes (< 10ms)
6. **Disk I/O**: Use fast storage (SSD) for database nodes

## Performance Tuning

### Database Connections

- **PgPool**: Adjust `PGPOOL_MAX_CONNECTIONS` based on load
- **metrics-pgpool**: Separate pool for metrics queries

### Metrics Collection

- Adjust `METRICS_MAX_WORKERS` for parallel collection
- Tune `METRICS_COLLECTION_INTERVAL` based on needs
- Monitor via `/metrics/observability` endpoint

See: [Environment Variables Reference](../reference/environment-variables.md#metrics-collection-configuration)

---

See also: [Docker Swarm Deployment](docker-swarm.md)

[← Back to Deployment](index.md)
