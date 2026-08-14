# Self-Hosting Guide

Welcome to the Obiente Cloud self-hosting guide! This guide will help you deploy and operate Obiente Cloud on your own infrastructure.

## What is Self-Hosting?

Self-hosting means running Obiente Cloud on your own servers rather than using a managed service. This gives you:

- ✅ **Complete Control** - Your data, your rules
- ✅ **Cost Efficiency** - No recurring hosting fees
- ✅ **Privacy** - Data never leaves your infrastructure
- ✅ **Customization** - Configure to your exact needs
- ✅ **Learning** - Understand distributed systems

## Who Should Self-Host?

### Perfect For:

- 🏠 **Home Lab Enthusiasts** - Personal projects and learning
- 🏢 **Small Businesses** - Internal development and testing
- 🎓 **Educational Institutions** - Teaching distributed systems
- 🔧 **Developers** - Local development and testing

### Consider Managed Service If:

- You need 24/7 support
- You prefer someone else managing infrastructure
- You don't have time for maintenance
- You need enterprise features immediately

## Quick Start

### Option 1: Local Development (Single Server)

Perfect for development and testing:

```bash
git clone https://github.com/obiente/cloud.git
cd cloud
docker compose up -d
```

See [Installation Guide](../getting-started/installation.md) for details.

### Option 2: Small Production (2-3 Nodes)

For small deployments with basic redundancy:

```bash
docker swarm init
./scripts/deploy-swarm.sh obiente docker-compose.swarm.yml
```

See [Docker Swarm Deployment](../deployment/docker-swarm.md).

### Option 3: Production HA (5+ Nodes)

For maximum uptime and reliability:

```bash
docker swarm init
docker stack deploy -c docker-compose.swarm.ha.yml obiente
```

See [High Availability Setup](../deployment/high-availability.md).

## Requirements

### Minimum Requirements (Single Server)

- **CPU**: 2 cores
- **RAM**: 4GB
- **Storage**: 50GB
- **OS**: Linux (Ubuntu 22.04+)

### Recommended (Small Production)

- **CPU**: 4 cores per server
- **RAM**: 8GB per server
- **Storage**: 100GB per server
- **Network**: 100 Mbps

### Production HA

- **CPU**: 8+ cores per server
- **RAM**: 16GB+ per server
- **Storage**: 200GB+ SSD per server
- **Network**: 1 Gbps

See [Detailed Requirements](requirements.md).

## Installation Steps

1. **[Install Docker](requirements.md#docker-installation)**
2. **[Choose Deployment Method](requirements.md#deployment-method)**
3. **[Configure Environment](../getting-started/configuration.md)**
4. **[Deploy Services](../getting-started/installation.md)**
5. **[Set Up Authentication](../guides/authentication.md)**
6. **[Configure Domains](../guides/routing.md)**

## Self-Hosting Considerations

### Networking

- **Port Forwarding**: Expose ports 80 and 443
- **Dynamic DNS**: Use if you don't have static IP
- **Firewall**: Configure ufw or iptables
- **SSL**: Let's Encrypt handles this automatically

### Security

- **Change Default Passwords**: First thing after installation
- **Keep Systems Updated**: Regular OS and Docker updates
- **Monitor Logs**: Check for unusual activity
- **Backups**: Automated backups are essential

### Maintenance

- **Monitor Resource Usage**: CPU, memory, disk
- **Check Logs Regularly**: API, database, proxy logs
- **Update Regularly**: Keep Docker images current
- **Test Backups**: Verify your backup strategy works

## Common Use Cases

### Home Lab

Deploy personal projects on your home server:

```bash
docker compose up -d
```

### Development Team

Internal staging environment:

```bash
./scripts/deploy-swarm.sh obiente docker-compose.swarm.yml
```

### Small Business

Customer-facing deployments:

```bash
docker stack deploy -c docker-compose.swarm.ha.yml obiente
```

## Guides

- [Requirements](requirements.md) - Hardware and software needs
- [Configuration](configuration.md) - Environment configuration
- [Upgrading](upgrading.md) - How to upgrade your deployment
- [Troubleshooting](../guides/troubleshooting.md) - Common issues

## Getting Help

- 📖 [Documentation](../README.md)
- 💬 [GitHub Discussions](https://github.com/obiente/cloud/discussions)
- 🐛 [Issue Tracker](https://github.com/obiente/cloud/issues)

---

[← Back to Documentation](../README.md)
