# Installation Guide

This guide will help you install Obiente Cloud on your infrastructure. Choose the installation method that best fits your needs.

## Prerequisites

Before installing, ensure you have:

- **Docker** 24.0 or higher installed
- **Docker Compose** 2.0 or higher (for local development)
- **Linux OS** (Ubuntu 22.04+ recommended)
- **Minimum 4GB RAM** (8GB+ recommended)
- **20GB+ free disk space**
- **Network access** for pulling Docker images

### Verify Docker Installation

```bash
docker --version
docker compose version  # Or docker compose --version
```

## Installation Methods

Choose the method that best suits your needs:

1. **[Docker Compose](docker-compose.md)** - Simple, single-server deployment
2. **[Docker Swarm - Simple](docker-swarm.md)** - Distributed deployment with basic HA
3. **[Docker Swarm - HA](high-availability.md)** - Full high availability setup

## Quick Install (Docker Compose)

For local development or testing:

```bash
# Clone the repository
git clone https://github.com/obiente/cloud.git
cd cloud

# Copy environment template
cp env.swarm.example .env

# Edit .env with your configuration
nano .env

# Start services
docker compose up -d

# Verify installation
docker compose ps
```

## Production Install (Docker Swarm)

For production environments:

```bash
# Initialize Docker Swarm
docker swarm init

# Deploy the stack
docker stack deploy -c docker-compose.swarm.yml obiente

# Check status
docker service ls
docker stack services obiente
```

See [Production Deployment](deployment/docker-swarm.md) for detailed instructions.

## Verify Installation

After installation, verify everything is working:

### 1. Check Services

```bash
# Docker Compose (databases/services)
docker compose ps

# Docker Swarm (if using Swarm deployment)
docker service ls
```

### 2. Check Logs

```bash
# If running API locally: view the terminal output running `go run`

# Docker Swarm (API in Swarm)
docker service logs obiente_api
```

### 3. Test API

```bash
# Health check
curl http://localhost:3001/health

# Should return: {"status":"ok"}
```

### 4. Test Database Connection

```bash
docker exec -it obiente-postgres psql -U obiente-postgres -d obiente -c "\dt"
```

## Next Steps

After successful installation:

1. **[Configure Authentication](guides/authentication.md)** - Set up Zitadel
2. **[Configure Domains](guides/routing.md)** - Set up custom domains
3. **[Set up Monitoring](../guides/index.md)** - Configure Prometheus and Grafana
   See the [Troubleshooting](../guides/troubleshooting.md) guide

## Common Issues

### Docker Compose Not Found

```bash
# Ubuntu/Debian
sudo apt-get install docker-compose-plugin

# Or install via Docker Desktop
```

### Permission Denied

```bash
# Add user to docker group
sudo usermod -aG docker $USER
# Log out and back in
```

### Port Already in Use

Edit `docker-compose.yml` or `docker-compose.swarm.yml` and change the port mappings.

## Uninstall

### Docker Compose

```bash
docker compose down -v
```

### Docker Swarm

```bash
docker stack rm obiente
```

This removes all services and data volumes. **Warning**: This will delete all data!

## Support

Having trouble? Check out:

- [Troubleshooting Guide](guides/troubleshooting.md)
- [GitHub Issues](https://github.com/obiente/cloud/issues)
- [Documentation](../README.md)

---

Next: [Configuration](configuration.md)
