# Architecture Documentation

Obiente Cloud's architecture is designed for distributed, production-grade deployments while remaining simple enough for home lab setups.

## Overview

- [System Architecture](overview.md) - High-level overview of the entire system
- [Components](components.md) - Detailed breakdown of each component
- [Deployment Model](deployment-model.md) - How deployments work across nodes

## Core Principles

### 1. Distributed by Default

- Deployments run across multiple nodes
- No single point of failure
- Automatic load balancing

### 2. Simple to Operate

- Docker-first approach
- Standard Docker Compose and Swarm
- No complex orchestration needed

### 3. Self-Hostable Abilities

- Runs on any Docker-capable infrastructure
- From single server to multi-node clusters
- Support for both HA and simple setups

### 4. Developer-Friendly

- Open source and extensible
- RESTful ConnectRPC API
- Comprehensive documentation

## Architecture Layers

```
┌──────────────────▼──────────────────┐
│          User Applications          │
│       (Deployed Applications)       │
└──────────────────┬──────────────────┘
                   │
┌──────────────────▼──────────────────┐
│           Routing Layer             │
│        (Traefik Reverse Proxy)       │
└──────────────────┬──────────────────┘
                   │
┌──────────────────▼──────────────────┐
│           Control Plane             │
│   ┌────────────┐  ┌────────────┐    │
│   │  Go API    │  │Orchestrator│    │
│   └────────────┘  └────────────┘    │
└──────────────────┬──────────────────┘
                   │
┌──────────────────▼──────────────────┐
│             Data Plane              │
│   ┌────────────┐  ┌────────────┐    │
│   │PostgreSQL  │  │TimescaleDB │    │
│   └────────────┘  └────────────┘    │
│   ┌────────────┐                    │
│   │   Redis    │                    │
│   └────────────┘                    │
└─────────────────────────────────────┘
```

## Key Components

### Control Plane

Services that manage the platform:

- **Go API** - Main application API
- **Orchestrator** - Node selection and deployment management
- **Service Registry** - Tracking deployment locations

### Data Plane

Storage and caching layer:

- **PostgreSQL** - Primary database for application metadata
- **TimescaleDB** - Time-series database for metrics storage
- **Redis** - Caching and sessions

### Routing

Traffic management:

- **Traefik** - Reverse proxy and load balancer

Learn more: [Architecture Overview](overview.md)

---

[← Back to Documentation](../README.md)
