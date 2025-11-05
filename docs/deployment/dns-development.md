# DNS Development Guide

## Overview

This guide covers DNS configuration for Obiente Cloud, including:
- Local development DNS setup
- DNS delegation for self-hosters
- Production DNS configuration

## How DNS Resolution Works

The DNS server **queries the database** to resolve domains. It doesn't "create" records - it reads from the database where deployments are stored.

### Local Development DNS (Default)

By default, the development setup runs a **local DNS server** inside Docker that:
- Runs on port **53** inside Docker (not exposed to host - no port conflict!)
- Queries the **dev database** (so dev deployments resolve correctly)
- Is accessible to all containers via Docker's internal DNS
- Allows testing dev deployment domains without port conflicts or nameserver configuration

### DNS Delegation (For Self-Hosters)

DNS delegation allows self-hosted Obiente Cloud instances to use the main `my.obiente.cloud` DNS service while keeping their deployments in their own database. See [DNS Delegation Guide](dns-delegation.md) for detailed setup instructions.

### The Problem with Production DNS

If your **dev environment** uses the **production DNS server** without delegation:
- Production DNS queries the **production database**
- Your dev deployments are stored in the **dev database**
- Result: Production DNS **cannot resolve** dev deployment domains
- **Solution**: Use DNS delegation (see above)

### DNS Server IP: Internal vs Public

The DNS server IP can be:
- **Internal** (recommended): e.g., `10.0.9.10` - faster, more secure, works within your network
- **Public**: e.g., `203.0.113.1` - accessible from internet, but requires firewall rules and is less secure

**However**, the real issue is **which database** the DNS server queries, not whether the IP is internal or public.

## Solutions for Development

### Option 1: Use Local Dev DNS (Default - Recommended)

The default setup runs a DNS server inside Docker that queries the dev database:

```bash
# Use docker-compose.yml (includes local DNS on port 53 inside Docker)
docker compose up -d

# Your dev deployments will resolve via dev DNS → dev database
# DNS runs on port 53 inside Docker - no host port 53 conflict!
# All containers automatically use this DNS server
```

**Pros**: 
- Dev deployments resolve correctly
- No port 53 conflict on host (runs on port 53 inside Docker, not exposed)
- No nameserver configuration needed (containers use it automatically)
- Isolated from production

**Cons**: None - this is the default setup!

### Option 2: Share Database (Not Recommended)

Point dev DNS to production database:

```yaml
# In docker-compose.yml, modify DNS service:
dns:
  environment:
    DB_HOST: production-db-host  # Point to production DB
    DB_NAME: obiente  # Production database
```

**Pros**: Single DNS server, dev deployments resolve  
**Cons**: Dev and prod share database (risky!), dev data mixed with prod

### Option 3: Use Production DNS (Not Recommended for Dev)

Use production DNS but accept that dev deployments won't resolve:

```bash
export MAIN_DNS_IP=10.0.9.10  # Production DNS IP
export MAIN_DNS_PORT=53        # Production DNS port
docker compose up -d --scale dns=0  # Skip local DNS

# Dev deployments won't resolve via DNS
# But you can still access them via IP:port directly
```

**Pros**: Simple, no local DNS needed  
**Cons**: Dev deployment domains won't work (production DNS queries production DB)

### Option 4: Use Production DNS for Testing Production Domains

Use production DNS to test how production domains resolve:

```bash
export MAIN_DNS_IP=10.0.9.10  # Production DNS
docker compose up -d --scale dns=0

# Your dev API can query production DNS to test production domain resolution
# But dev deployments still won't resolve
```

## Recommended Setup

**For Development (Default):**
- Use **local dev DNS** (Option 1) - runs automatically on port 53 inside Docker
- Dev DNS queries dev database
- Dev deployments resolve correctly
- No port 53 conflict (not exposed to host), no nameserver configuration needed
- Isolated from production

**For Testing Production DNS:**
- Set `MAIN_DNS_IP` and `MAIN_DNS_PORT` to production DNS
- Use `--scale dns=0` to skip local DNS
- Test production domain resolution from dev environment
- Note: Production DNS won't resolve dev deployments (queries production DB)

## Finding Production DNS IP

**Internal IP** (recommended if on same network):
```bash
# On manager node
docker service ps obiente_dns --format "{{.Node}}"
docker node inspect <node-name> --format "{{.Status.Addr}}"

# Or check overlay network
docker network inspect obiente_obiente-network | grep -A 5 dns
```

**Public IP** (if DNS is exposed):
- Check your firewall/load balancer configuration
- Usually not recommended for security reasons


