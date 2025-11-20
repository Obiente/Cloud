# DNS Delegation Configuration for my.obiente.cloud

## Overview

The DNS server resolves `*.my.obiente.cloud` domains to Traefik IP addresses based on where deployments are actually running. This requires DNS delegation from the parent `obiente.cloud` domain.

## DNS Zone Delegation

To enable DNS resolution for `my.obiente.cloud`, you need to delegate the zone to your DNS servers.

### Step 1: Get DNS Server IPs

Get the public IP addresses of all your nodes (since DNS runs on all nodes):

```bash
# Get all node IPs
docker node ls --format "{{.Hostname}}: {{.ID}}"

# Or get specific node IPs
docker node inspect <node-id> --format '{{.Status.Addr}}'
```

Since DNS runs on **all nodes** in global mode, you can configure **multiple nameservers** (one per node) for maximum redundancy.

### Step 2: Configure Nameserver Records

In your DNS provider for `obiente.cloud`, add NS records for **all nodes** (or as many as you want for redundancy):

```
my.obiente.cloud.    IN    NS    ns1.my.obiente.cloud.
my.obiente.cloud.    IN    NS    ns2.my.obiente.cloud.
my.obiente.cloud.    IN    NS    ns3.my.obiente.cloud.
# ... add more NS records for each node
```

Then add A records for each nameserver pointing to node IPs:

```
ns1.my.obiente.cloud.    IN    A    <NODE_1_IP>
ns2.my.obiente.cloud.    IN    A    <NODE_2_IP>
ns3.my.obiente.cloud.    IN    A    <NODE_3_IP>
# ... add more A records for each node
```

**Best Practice**: Configure at least 2-3 nameservers, but you can configure one per node for maximum redundancy.

### Step 3: Update DNS Server Configuration

The DNS servers are configured via environment variables in `docker-compose.swarm.yml`:

- `NODE_IPS`: Node IPs per region (format: `"region1:ip1,ip2;region2:ip3,ip4"`)

Example:
```bash
NODE_IPS="us-east-1:1.2.3.4,1.2.3.5;eu-west-1:5.6.7.8,5.6.7.9"
```

### Step 4: Verify DNS Resolution

Test DNS resolution:

```bash
# Query a deployment domain
dig deploy-123.my.obiente.cloud @<DNS_SERVER_IP>

# Or use nslookup
nslookup deploy-123.my.obiente.cloud <DNS_SERVER_IP>
```

## Deployment Modes

The DNS server works in both deployment scenarios:

### Single-Node Deployment (`docker-compose.yml`)
- Runs **one DNS server instance** on the single node
- Port 53 directly bound to the host
- Suitable for development and small deployments
- Configure **one NS record** pointing to the node IP

### Multi-Node HA Deployment (`docker-compose.swarm.yml`)
- Runs in **global mode** on **all nodes**
- Port 53 available on every node
- Automatic failover if nodes fail
- Configure **multiple NS records** (one per node) for redundancy
- DNS clients automatically distribute queries across all nodes

## High Availability

- **Automatic Distribution**: DNS server runs on every node automatically
- **Node Failure Tolerance**: If a node fails, DNS continues working on other nodes
- **Load Distribution**: DNS queries are distributed across all nodes
- **No Single Point of Failure**: Multiple nameservers ensure redundancy
- **Same as API Service**: Runs alongside API service on all nodes for consistency

### Nameserver Failover

When you configure multiple nameservers:
1. DNS clients will try nameservers in order
2. If one nameserver fails, clients automatically try the next one
3. All nameservers query the same PostgreSQL database (single source of truth)
4. All nameservers return consistent results
5. Each node has its own DNS server instance, so node-level failures don't affect DNS service
6. You can configure NS records for all nodes, providing maximum redundancy

### Configuration Strategy

Since DNS runs on all nodes, you can:
- Configure multiple NS records (one per node IP)
- DNS clients will automatically distribute queries across all available nameservers
- Failed nodes are automatically skipped by DNS clients
- No manual failover configuration needed - DNS protocol handles it automatically

## DNS Caching

DNS responses are cached for 60 seconds (TTL) to reduce database load while still providing relatively real-time updates when deployments move between nodes.

## Monitoring

Monitor DNS server health:
- Check Docker service status: `docker service ps dns`
- View DNS server logs: `docker service logs dns`
- Test DNS resolution: `dig @<DNS_SERVER_IP> deploy-123.my.obiente.cloud`
