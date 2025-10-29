# Obiente Cloud

> **A distributed Platform-as-a-Service (PaaS) for deploying and managing applications across multiple nodes**

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
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

```bash
# Clone the repository
git clone https://github.com/obiente/cloud.git
cd cloud

# Start services
docker-compose up -d

# Check status
docker-compose ps
```

### Docker Swarm (Production)

```bash
# Initialize Swarm
docker swarm init

# Deploy
docker stack deploy -c docker-compose.swarm.yml obiente

# Verify
docker service ls
```

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
┌─────────────────────────────────────────────────────────┐
│                    Obiente Cloud                        │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │   Go API     │  │   Traefik    │  │ Orchestrator │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│         │                 │                  │         │
│  ┌──────────────────────────────────────────────────┐  │
│  │    PostgreSQL Cluster (HA) / Single Instance     │  │
│  └──────────────────────────────────────────────────┘  │
│                                                         │
└─────────────────────────────────────────────────────────┘
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

[MIT License](LICENSE)

## Support

- 📖 [Documentation](docs/README.md)
- 💬 [GitHub Discussions](https://github.com/obiente/cloud/discussions)
- 🐛 [Issue Tracker](https://github.com/obiente/cloud/issues)

---

**Made with ❤️ by the Obiente Team**
