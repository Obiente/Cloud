# DNS Configuration

The DNS server resolves `*.my.obiente.cloud` domains to Traefik IP addresses based on where deployments are actually running.

## Overview

The DNS server is integrated into the API service and runs alongside it. It queries the database to determine which region a deployment is running in, then returns the appropriate Traefik IP addresses for that region.

## Environment Variables

### Required

- **`TRAEFIK_IPS`**: Traefik IPs per region
  - **Multi-region format**: `"region1:ip1,ip2;region2:ip3,ip4"`
  - **Simple format**: `"ip1,ip2"` (defaults to "default" region)
  - **Examples**:
    - Simple: `TRAEFIK_IPS="1.2.3.4"` or `TRAEFIK_IPS="1.2.3.4,1.2.3.5"`
    - Multi-region: `TRAEFIK_IPS="us-east-1:1.2.3.4,1.2.3.5;eu-west-1:5.6.7.8,5.6.7.9"`
  - Maps regions to Traefik IP addresses
  - Multiple IPs per region enable load balancing
  - If using simple format, deployments will use the "default" region

### Optional

- **`DNS_IPS`**: DNS server IPs (comma-separated list)
  - **Example**: `DNS_IPS="1.2.3.4,5.6.7.8,9.10.11.12"`
  - Used for documentation and configuring nameserver records
  - Does not affect server operation
  - Set to your node public IPs for easier DNS delegation setup

- **`DNS_PORT`**: DNS server port (default: `53`)
  - **Example**: `DNS_PORT=5353`
  - Use a different port if port 53 is already in use (e.g., systemd-resolved)
  - Must match the port mapping in docker-compose.yml

### Database Configuration

The DNS server uses the same database configuration as the API:
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`

## Deployment Modes

### Single-Node Deployment (`docker-compose.yml`)

- Runs one DNS server instance
- Suitable for development and small deployments
- Configure one NS record pointing to the node IP

**Example `.env`:**

```bash
TRAEFIK_IPS="default:1.2.3.4"
DNS_IPS="1.2.3.4"
```

### Multi-Node HA Deployment (`docker-compose.swarm.yml`)

- Runs in **global mode** on all API nodes
- High availability across all nodes
- Automatic failover if nodes fail
- Load distribution for DNS queries
- Configure multiple NS records (one per node) for redundancy

**Example `.env`:**

```bash
TRAEFIK_IPS="us-east-1:1.2.3.4,1.2.3.5;eu-west-1:5.6.7.8,5.6.7.9"
DNS_IPS="1.2.3.4,5.6.7.8,9.10.11.12"
```

## DNS Zone Delegation

To enable DNS resolution for `my.obiente.cloud`, you need to delegate the zone to your DNS servers.

### Step 1: Get DNS Server IPs

Get the public IP addresses of all your nodes (since DNS runs on all nodes in global mode):

```bash
# Get all node IPs
docker node ls --format "{{.Hostname}}: {{.ID}}"

# Or get specific node IPs
docker node inspect <node-id> --format '{{.Status.Addr}}'
```

Alternatively, set `DNS_IPS` in your `.env` file with comma-separated node IPs.

### Step 2: Configure Nameserver Records

In your DNS provider for `obiente.cloud`, add NS records for all nodes (or as many as you want for redundancy):

```
my.obiente.cloud.    IN    NS    ns1.my.obiente.cloud.
my.obiente.cloud.    IN    NS    ns2.my.obiente.cloud.
my.obiente.cloud.    IN    NS    ns3.my.obiente.cloud.
```

Then add A records for each nameserver pointing to node IPs:

```
ns1.my.obiente.cloud.    IN    A    <NODE_1_IP>
ns2.my.obiente.cloud.    IN    A    <NODE_2_IP>
ns3.my.obiente.cloud.    IN    A    <NODE_3_IP>
```

**Best Practice**: Configure at least 2-3 nameservers, but you can configure one per node for maximum redundancy.

### Step 3: Verify DNS Resolution

Test DNS resolution:

```bash
# Query a deployment domain
dig deploy-123.my.obiente.cloud @<DNS_SERVER_IP>

# Or use nslookup
nslookup deploy-123.my.obiente.cloud <DNS_SERVER_IP>
```

## High Availability

### Automatic Distribution

- DNS server runs on every node automatically (global mode)
- Each node has its own DNS server instance
- Node-level failures don't affect DNS service

### Nameserver Failover

When you configure multiple nameservers:
1. DNS clients try nameservers in order
2. If one nameserver fails, clients automatically try the next one
3. All nameservers query the same PostgreSQL database (single source of truth)
4. All nameservers return consistent results
5. No manual failover configuration needed - DNS protocol handles it automatically

### Load Distribution

DNS queries are distributed across all nodes:
- DNS clients automatically distribute queries across all available nameservers
- Failed nodes are automatically skipped by DNS clients
- Multiple nameservers ensure redundancy

## DNS Caching

DNS responses are cached for 60 seconds (TTL) to reduce database load while still providing relatively real-time updates when deployments move between nodes.

## Monitoring

Monitor DNS server health:

```bash
# Check Docker service status
docker service ps dns

# View DNS server logs
docker service logs dns

# Test DNS resolution
dig @<DNS_SERVER_IP> deploy-123.my.obiente.cloud
```

## Troubleshooting

### Port 53 already in use

Port 53 is often used by systemd-resolved or other DNS services on Linux. You have two options:

**Option 1: Use a different port (Recommended for development)**

1. Set `DNS_PORT` in your `.env` file:
   ```bash
   DNS_PORT=5353
   ```

2. Update docker-compose.yml port mapping:
   ```yaml
   ports:
     - "5353:5353/udp"
     - "5353:5353/tcp"
   ```

3. Test DNS resolution using the custom port:
   ```bash
   dig @localhost -p 5353 deploy-123.my.obiente.cloud
   ```

**Option 2: Stop systemd-resolved (Production only)**

If you need to use port 53 directly:

```bash
# Stop systemd-resolved
sudo systemctl stop systemd-resolved
sudo systemctl disable systemd-resolved

# Edit /etc/systemd/resolved.conf to disable DNS stub listener
sudo sed -i 's/#DNSStubListener=yes/DNSStubListener=no/' /etc/systemd/resolved.conf

# Restart systemd-resolved
sudo systemctl restart systemd-resolved
```

**Note:** Only do this if you understand the implications. You may need to configure alternative DNS resolution.

### DNS not resolving

1. **Check DNS server is running:**
   ```bash
   docker service ps dns
   ```

2. **Check DNS server logs:**
   ```bash
   docker service logs dns
   ```

3. **Verify TRAEFIK_IPS is set:**
   ```bash
   docker service inspect dns --format '{{range .Spec.TaskTemplate.ContainerSpec.Env}}{{println .}}{{end}}' | grep TRAEFIK_IPS
   ```

4. **Test DNS resolution directly:**
   ```bash
   dig @<DNS_SERVER_IP> deploy-123.my.obiente.cloud
   ```

### Deployment not found

- Ensure the deployment is running: `docker service ps <deployment-id>`
- Check deployment status in database
- Verify node has a region configured in `node_metadata` table

### Wrong IP returned

- Verify `TRAEFIK_IPS` matches your actual Traefik IPs
- Check deployment region matches Traefik IP region mapping
- Ensure node has correct region set in `node_metadata` table

---

[← Back to Deployment](../deployment/index.md)
