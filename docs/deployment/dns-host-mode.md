# DNS Host Mode Port Publishing

## Overview

The DNS service is configured to run **only on designated nodes** and expose port 53 using **host mode** port publishing. This prevents other nodes from even attempting to bind to port 53.

## How It Works

1. **DNS runs ONLY on nodes labeled with `dns.enabled=true`** (replicated mode with placement constraints)
2. **Port 53 is published in host mode**, which means:
   - Only designated nodes will run DNS and bind to port 53
   - Other nodes will not run DNS at all (no binding attempt)
   - This prevents port conflicts and ensures only the DNS IP node exposes port 53

## Configuration

The DNS service uses host mode port publishing:

```yaml
ports:
  - target: 53
    published: 53
    protocol: udp
    mode: host
  - target: 53
    published: 53
    protocol: tcp
    mode: host
```

## Setup Steps

### 1. Label the DNS IP Node

Label the node(s) where you want DNS to run (typically the node with your DNS IP, e.g., 209.205.228.173):

```bash
# Find your node name
docker node ls

# Label the DNS IP node
docker node update --label-add dns.enabled=true <node-name>

# Verify the label
docker node inspect <node-name> | grep -A 5 Labels
```

### 2. Disable systemd-resolved on DNS IP Node

On the node with your DNS IP, disable systemd-resolved:

```bash
# Disable DNS stub listener
sudo sed -i 's/#DNSStubListener=yes/DNSStubListener=no/' /etc/systemd/resolved.conf
sudo sed -i 's/DNSStubListener=yes/DNSStubListener=no/' /etc/systemd/resolved.conf
sudo systemctl restart systemd-resolved

# Verify port 53 is free
ss -tuln | grep :53
```

### 3. Configure DNS Replicas (Optional)

If you want DNS to run on multiple nodes for redundancy, set the number of replicas:

```bash
# In your .env file
DNS_REPLICAS=2  # Run DNS on 2 nodes (both must have dns.enabled=true label)
```

### 4. Deploy the Stack

```bash
docker stack deploy -c docker-compose.swarm.yml obiente
```

## Verification

### Check DNS Service Status

```bash
# List DNS tasks on all nodes
docker service ps obiente_dns

# Check which nodes have port 53 exposed
# On each node, run:
ss -tuln | grep :53
```

### Test DNS Resolution

```bash
# From external network
dig @209.205.228.173 deploy-123.my.obiente.cloud

# From within the cluster
dig @<dns-node-ip> deploy-123.my.obiente.cloud
```

## Benefits

1. **No Unnecessary Binding**: Other nodes don't even attempt to bind to port 53
2. **Selective Deployment**: Only designated nodes run DNS
3. **No Port Conflicts**: Nodes without the label won't run DNS at all
4. **Redundancy**: Can run DNS on multiple labeled nodes for high availability
5. **Internal Access**: All nodes can resolve DNS internally via the Docker network

## Troubleshooting

### Port 53 Not Exposed on DNS IP Node

1. Check if systemd-resolved is using port 53:
   ```bash
   ss -tuln | grep :53
   ```

2. Check DNS service logs:
   ```bash
   docker service logs obiente_dns
   ```

3. Verify DNS service is running on the DNS IP node:
   ```bash
   docker service ps obiente_dns --filter "node=<dns-node-name>"
   ```

### DNS Not Resolving

1. Verify DNS server is listening on the DNS IP node:
   ```bash
   # On DNS IP node
   ss -tuln | grep :53
   # Should show 0.0.0.0:53 or the node's IP:53
   ```

2. Check firewall rules:
   ```bash
   # Allow UDP and TCP port 53
   sudo ufw allow 53/udp
   sudo ufw allow 53/tcp
   ```

3. Test from within the cluster:
   ```bash
   docker run --rm --network obiente_obiente-network alpine nslookup deploy-123.my.obiente.cloud
   ```

