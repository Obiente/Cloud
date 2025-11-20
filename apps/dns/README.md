# DNS Server for my.obiente.cloud
# Resolves *.my.obiente.cloud domains to node IPs based on deployment and game server locations

## Deployment Modes

The DNS server works in both **single-node** and **multi-node HA** deployments:

### Single-Node (`docker-compose.yml`)
- Runs one DNS server instance
- Suitable for development and small deployments
- Configure one NS record pointing to the node IP

### Multi-Node HA (`docker-compose.swarm.yml`)
- Runs in **global mode** on all API nodes
- High availability across all nodes
- Automatic failover if nodes fail
- Load distribution for DNS queries
- Configure multiple NS records (one per node) for redundancy
- Same deployment pattern as API service

## Environment Variables:
# - DB_HOST: PostgreSQL host
# - DB_PORT: PostgreSQL port
# - DB_USER: PostgreSQL user
# - DB_PASSWORD: PostgreSQL password
# - DB_NAME: PostgreSQL database name
# - NODE_IPS: Node IPs per region (format: "region1:ip1,ip2;region2:ip3,ip4")

## Example Configuration:
# NODE_IPS="us-east-1:1.2.3.4,1.2.3.5;eu-west-1:5.6.7.8,5.6.7.9"

## Nameserver Configuration

### Single-Node:
- Configure one NS record pointing to your node IP

### Multi-Node:
- Configure multiple nameservers (one per node) in your DNS provider for redundancy
- Each node running the DNS service can act as a nameserver
- Configure NS records pointing to node IPs
- DNS clients will automatically failover between nameservers

See DNS_DELEGATION.md for detailed setup instructions.
