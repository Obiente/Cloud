# Deployment Guide

Obiente Cloud can be deployed in several ways, from simple local development to fully distributed production environments.

## Deployment Options

### 1. Docker Compose (Development)

**Best for**: Local development, testing, single-server deployments

- Single PostgreSQL instance
- Single Redis instance
- Simple to set up
- Good for development

[→ Docker Compose Guide](docker-compose.md)

### 2. Docker Swarm - Simple (Small Production)

**Best for**: Small production deployments, minimal HA requirements

- Single PostgreSQL (Swarm-managed)
- Single Redis (Swarm-managed)
- Distributed across nodes
- Basic redundancy

[→ Docker Swarm Guide](docker-swarm.md)

### 3. Docker Swarm - High Availability (Production)

**Best for**: Production environments requiring maximum uptime

- 3-node PostgreSQL cluster (Patroni + etcd)
- 3-node Redis cluster
- Automatic failover
- Full HA capabilities

[→ High Availability Guide](high-availability.md)

## Choosing Your Deployment Method

### Use Docker Compose if:

- ✅ You're developing locally
- ✅ You have a single server
- ✅ You don't need redundancy
- ✅ You want the simplest setup

### Use Docker Swarm Simple if:

- ✅ You have 2-3 nodes
- ✅ You want basic redundancy
- ✅ You don't need full HA
- ✅ You want distributed deployments

### Use Docker Swarm HA if:

- ✅ You have 5+ nodes
- ✅ You need maximum uptime
- ✅ You require automatic failover
- ✅ You're running production workloads

## Quick Comparison

| Feature          | Docker Compose | Swarm Simple | Swarm HA       |
| ---------------- | -------------- | ------------ | -------------- |
| Nodes Required   | 1              | 2+           | 5+             |
| PostgreSQL       | Single         | Single       | 3-node cluster |
| TimescaleDB      | Single         | Single       | 3-node cluster |
| Redis            | Single         | Single       | 3-node cluster |
| Auto Failover    | ❌             | ❌           | ✅             |
| Load Balancing   | ❌             | ✅           | ✅             |
| Metrics HA       | ❌             | ❌           | ✅             |
| Setup Complexity | Low            | Medium       | High           |
| Resource Usage   | Low            | Medium       | High           |

## Prerequisites

All deployment methods require:

- Docker 24.0+
- Linux OS (Ubuntu 22.04+ recommended)
- 4GB+ RAM
- 20GB+ disk space

For specific requirements, see [Requirements](../self-hosting/requirements.md).

## Next Steps

Choose your deployment method and follow the respective guide:

1. [Docker Compose](docker-compose.md)
2. [Docker Swarm](docker-swarm.md)
3. [High Availability](high-availability.md)
4. [DNS Configuration](dns.md)

---

[← Back to Documentation](../README.md)
