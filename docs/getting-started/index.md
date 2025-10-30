# Getting Started

Welcome to Obiente Cloud! These guides will help you get up and running quickly.

## Installation

- [Installation Guide](installation.md) - Install Obiente Cloud
- [Development Setup](development.md) - Set up for development
- [Configuration](configuration.md) - Configure your deployment

## Quick Start

### Local Development

```bash
git clone https://github.com/obiente/cloud.git
cd cloud
docker compose up -d
```

### Docker Swarm

```bash
docker swarm init
docker stack deploy -c docker-compose.swarm.yml obiente
```

See [Installation Guide](installation.md) for detailed instructions.

---

[← Back to Documentation](../README.md)
