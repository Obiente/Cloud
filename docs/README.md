# Obiente Cloud Documentation

Welcome to the Obiente Cloud documentation! This comprehensive guide will help you understand, deploy, and operate Obiente Cloud.

## What is Obiente Cloud?

Obiente Cloud is a **Platform-as-a-Service (PaaS)** that allows you to deploy and manage applications across distributed infrastructure, similar to Vercel. Whether you're self-hosting at home or running a production cluster, Obiente Cloud makes it easy to deploy and scale your applications.

## Quick Links

### 🚀 Getting Started

- [Installation](getting-started/installation.md) - Install Obiente Cloud
- [Quick Start Guide](getting-started/quickstart.md) - Get up and running in minutes
- [Development Setup](getting-started/development.md) - Set up for local development

### 🏗️ Architecture

- [Overview](architecture/overview.md) - System architecture overview
- [Components](architecture/components.md) - Detailed component descriptions (coming soon)
- [Deployment Model](architecture/deployment-model.md) - How deployments work (coming soon)

### 🚢 Deployment

- [Deployment Methods](deployment/index.md) - Overview of deployment options
- [Docker Compose](deployment/docker-compose.md) - Simple local deployment (coming soon)
- [Docker Swarm](deployment/docker-swarm.md) - Distributed deployment
- [High Availability](deployment/high-availability.md) - HA production setup (coming soon)

### 🏠 Self-Hosting

- [Self-Hosting Guide](self-hosting/index.md) - Introduction to self-hosting

### 📚 Guides

- [Authentication](guides/authentication.md) - Setting up authentication with Zitadel
- [Routing & Domains](guides/routing.md) - Configuring domains and routing
- [VPS Provisioning](guides/vps-provisioning.md) - Provision and manage VPS instances on Proxmox
- [Proxmox SSH Setup](guides/proxmox-ssh-user-setup.md) - Configure SSH access for cloud-init snippets
- [VPS Gateway Setup](guides/vps-gateway-setup.md) - Set up gateway service for DHCP and SSH proxying
- [Monitoring](guides/monitoring.md) - Setting up monitoring and alerts (coming soon)
- [Troubleshooting](guides/troubleshooting.md) - Common issues and solutions

### 📖 Reference

- [Environment Variables](reference/environment-variables.md) - Complete environment variable reference
- [Reference Index](reference/index.md) - API and CLI docs (coming soon)

## Documentation Structure

This documentation is organized into several sections:

```
docs/
├── getting-started/      # Installation and setup guides
├── architecture/         # System design and components
├── deployment/           # Deployment methods and configurations
├── guides/               # Step-by-step guides for common tasks
├── self-hosting/         # Guides specific to self-hosting
└── reference/            # API docs, environment variables, etc.
```

## Who is this for?

### Self-Hosters

Looking to run Obiente Cloud at home? Start with:

1. [Self-Hosting Guide](self-hosting/index.md)
2. [Requirements](self-hosting/requirements.md)
3. [Installation](getting-started/installation.md)

### Developers

Working on Obiente Cloud itself? Check out:

1. [Development Setup](getting-started/development.md)
2. [Architecture Overview](architecture/overview.md)
3. Reference docs: [Reference Index](reference/index.md)

### DevOps Engineers

Deploying in production? Start here:

1. [Deployment Methods](deployment/index.md)
2. [High Availability Setup](deployment/high-availability.md)
3. [Troubleshooting](guides/troubleshooting.md)

## Getting Help

- **Issues**: [GitHub Issues](https://github.com/obiente/cloud/issues)
- **Discussions**: [GitHub Discussions](https://github.com/obiente/cloud/discussions)
- **Documentation Issues**: Found an error? [File an issue](https://github.com/obiente/cloud/issues/new)

## License

[GNU Affero General Public License v3.0](https://www.gnu.org/licenses/agpl-3.0)

---

**Last updated**: [Current date]
