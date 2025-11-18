# Obiente Cloud

> **A distributed Platform-as-a-Service (PaaS) for deploying and managing applications across multiple nodes**

[![License](https://img.shields.io/badge/license-AGPL--3.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-24.0+-blue.svg)](https://www.docker.com/)

## Features

- 🚀 **Multi-Node Deployments** - Distribute applications across multiple servers
- 🌐 **Dynamic Routing** - Automatic traffic routing with custom domains
- 🔒 **Integrated Auth** - Built-in Zitadel authentication support
- 📊 **Real-Time Monitoring** - Track deployments with Prometheus and Grafana
- 🔄 **Auto Scaling** - Automatic scaling based on load
- 💾 **High Availability** - Optional HA setup for production
- 🎯 **Smart Orchestration** - Intelligent node selection and load balancing

## Quick Start

### Docker Compose (Local Development)

**For development on worker nodes or single-machine setups:**

```bash
# Clone the repository
git clone https://github.com/obiente/cloud.git
cd cloud

# Option 1: Use local DNS (default)
docker compose up -d

# Option 2: Use production DNS (recommended if you have a production deployment)
# Set production DNS server IP
export MAIN_DNS_IP=10.0.9.10  # Replace with your production DNS server IP

# Disable local DNS service (comment out lines 186-217 in docker-compose.yml)
# Or skip it: docker compose up -d --scale dns=0
docker compose up -d

# Check status
docker compose ps
```

**DNS Configuration**: By default, a local DNS server runs inside Docker on port 53 (not exposed to host, so no port conflict). It queries the dev database, so dev deployments resolve correctly. All containers automatically use this DNS server. To use production DNS instead, set `MAIN_DNS_IP` and `MAIN_DNS_PORT` - but note that production DNS queries the production database, so it won't resolve dev deployments. See [DNS Development Guide](docs/deployment/dns-development.md) for details.

**Note**: Worker nodes cannot deploy Docker Swarm stacks. Use `docker compose` for development on worker nodes.

### Docker Swarm (Development - Uses Main Deployment DNS)

**For development on manager nodes only** (worker nodes cannot deploy stacks):

**Recommended: Use the development deployment script:**

```bash
# Deploy using existing images (default - no build/pull)
./scripts/deploy-swarm-dev.sh

# Build images locally and deploy
./scripts/deploy-swarm-dev.sh -b
# or
./scripts/deploy-swarm-dev.sh --build

# Pull images from registry and deploy
./scripts/deploy-swarm-dev.sh -p
# or
./scripts/deploy-swarm-dev.sh --pull

# Custom stack name
./scripts/deploy-swarm-dev.sh my-dev-stack
```

**Manual deployment (alternative):**

```bash
# Verify you're a manager node
docker node ls  # Should work, not show "not a manager" error

# Build images first (required before deploying)
export DOCKER_BUILDKIT=1
for service in api-gateway auth-service organizations-service billing-service deployments-service gameservers-service orchestrator-service vps-service support-service audit-service superadmin-service dns-service vps-gateway; do
  docker build -f apps/$service/Dockerfile -t ghcr.io/obiente/cloud-$service:latest .
done

# Merge and deploy
cat docker-compose.base.yml docker-compose.swarm.dev.yml | docker stack deploy -c - obiente-dev

# View logs
docker service logs -f obiente-dev_api-gateway

# Remove stack
docker stack rm obiente-dev
```

**Note**: The `docker-compose.swarm.dev.yml` file uses Swarm-specific features (overlay networks) and **must** be deployed with `docker stack deploy`, not `docker compose`. Only manager nodes can deploy stacks. Worker nodes should use regular `docker compose` (see above). See [Development Deployment Guide](docs/deployment/docker-swarm.md#development-deployment) for more details.

### Docker Swarm (Production)

```bash
# Initialize Docker Swarm (if not already initialized)
docker swarm init

# Build and deploy (recommended - uses helper script)
./scripts/deploy-swarm.sh obiente docker-compose.swarm.yml

# Or build manually, then deploy:
export DOCKER_BUILDKIT=1
docker build -f apps/api/Dockerfile -t obiente/cloud-api:latest .
docker stack deploy -c docker-compose.swarm.yml obiente

# For multi-node deployments, push images to a registry or use docker save/load
# docker tag obiente/cloud-api:latest your-registry/obiente/cloud-api:latest
# docker push your-registry/obiente/cloud-api:latest

# Check status
docker service ls
docker stack services obiente
```

**Important**: Docker Swarm doesn't support building images during deployment. You must build images first, then deploy. On multi-node setups, ensure images are available on all nodes (use a registry or `docker save/load`).

See the [Installation Guide](docs/getting-started/installation.md) for detailed instructions.

## Documentation

📚 **[Full Documentation](docs/README.md)**

### Quick Links

- [Getting Started](docs/getting-started/installation.md)
- [Architecture](docs/architecture/overview.md)
- [Deployment Guide](docs/deployment/index.md)
- [Self-Hosting](docs/self-hosting/index.md)
- [Reference](docs/reference/index.md)

## Architecture

```
┌────────────────────────────────────────────────────────┐
│                      Obiente Cloud                     │
├────────────────────────────────────────────────────────┤
│                                                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │    Go API    │  │    Traefik    │  │ Orchestrator │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│         │                 │                  │         │
│  ┌──────────────────────────────────────────────────┐  │
│  │    PostgreSQL Cluster (HA) / Single Instance     │  │
│  └──────────────────────────────────────────────────┘  │
│                                                        │
└────────────────────────────────────────────────────────┘
```

Learn more: [Architecture Overview](docs/architecture/overview.md)

## Use Cases

### 🏠 Self-Hosting for Hobbyists

Deploy your personal projects on your own infrastructure. Perfect for:

- Home lab deployments
- Personal project hosting
- Learning distributed systems

### 🏢 Production IaaS

Sell Obiente Cloud as Infrastructure-as-a-Service:

- Multi-tenant deployments
- Custom domains per customer
- Resource management and billing

### 🚀 Development Team

Use Obiente Cloud internally:

- Staging and production environments
- Testing distributed applications
- CI/CD integration

## Requirements

- **Docker**: 24.0+
- **OS**: Linux (Ubuntu 22.04+ recommended)
- **RAM**: 4GB minimum (8GB+ recommended)
- **Storage**: 20GB minimum

See [Requirements](docs/self-hosting/requirements.md) for detailed specifications.

## Project Structure

```
cloud/
├── apps/
│   └── api/                 # Go ConnectRPC API
├── docs/                    # Documentation
├── monitoring/              # Prometheus & Grafana configs
├── docker-compose.yml       # Local development
├── docker-compose.swarm.yml # Simple swarm deployment
└── docker-compose.swarm.ha.yml # HA production deployment
```

## Development

```bash
# Install dependencies
go mod download

# Run API locally
cd apps/api
go run main.go

# Run tests
go test ./...

# Build Docker image
docker build -f apps/api/Dockerfile -t obiente/cloud-api:latest .
```

See [Development Guide](docs/getting-started/development.md) for more details.

## Contributing

We welcome contributions! Please see our contributing guidelines:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

See [Contributing Guide](docs/reference/contributing.md) for details.

## License

[GNU Affero General Public License v3.0](LICENSE)

## Support

- 📖 [Documentation](docs/README.md)
- 💬 [GitHub Discussions](https://github.com/obiente/cloud/discussions)
- 🐛 [Issue Tracker](https://github.com/obiente/cloud/issues)

---

**Made with ❤️ by the Obiente Team**
