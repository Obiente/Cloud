# DNS Server Configuration for Docker Swarm

## Overview

The Obiente Cloud DNS server resolves `*.my.obiente.cloud` domains for deployments and game servers. By default, it runs on all nodes in the swarm (global mode), but you can configure it to run only on specific nodes to delegate DNS to dedicated instances.

## Configuration Options

### Environment Variables

- `DNS_MODE`: Deployment mode for DNS service
  - `global` (default): Runs on all nodes in the swarm
  - `replicated`: Runs on specific nodes based on constraints
- `DNS_REPLICAS`: Number of DNS instances when using replicated mode (default: 1)
- `DNS_PORT`: Port for DNS server (default: 53)

## Delegating DNS to Specific Nodes

### Step 1: Label Nodes

Label the nodes where you want DNS to run:

```bash
# Enable DNS on specific nodes
docker node update --label-add dns.enabled=true <node-name>

# Disable DNS on specific nodes (useful for excluding certain nodes)
docker node update --label-add dns.enabled=false <node-name>
```

### Step 2: Configure Deployment Mode

Edit your `docker-compose.swarm.yml` or `docker-compose.swarm.ha.yml`:

1. Set DNS mode to `replicated`:
   ```yaml
   deploy:
     mode: replicated  # or set DNS_MODE=replicated in .env
     replicas: 2  # or set DNS_REPLICAS=2 in .env
   ```

2. Add placement constraints to restrict DNS to specific nodes:
   ```yaml
   deploy:
     placement:
       constraints:
         # Run only on nodes with dns.enabled=true label
         - node.labels.dns.enabled == true
   ```

### Step 3: Deploy

```bash
docker stack deploy -c docker-compose.swarm.yml obiente
```

## Common Configurations

### Run DNS Only on Manager Nodes

```yaml
deploy:
  mode: replicated
  replicas: 2
  placement:
    constraints:
      - node.role == manager
```

### Run DNS Only on Labeled Nodes

```yaml
deploy:
  mode: replicated
  replicas: 2
  placement:
    constraints:
      - node.labels.dns.enabled == true
```

### Exclude Specific Nodes from Running DNS

```yaml
deploy:
  mode: global
  placement:
    constraints:
      - node.labels.dns.enabled != false
```

This runs DNS on all nodes except those explicitly labeled with `dns.enabled=false`.

## Examples

### Example 1: Dedicated DNS Node

Run DNS on a single dedicated node:

```bash
# Label the DNS node
docker node update --label-add dns.enabled=true dns-node-1

# Label other nodes to exclude them
docker node update --label-add dns.enabled=false worker-node-1
docker node update --label-add dns.enabled=false worker-node-2
```

Then in `docker-compose.swarm.yml`:

```yaml
dns:
  deploy:
    mode: replicated
    replicas: 1
    placement:
      constraints:
        - node.labels.dns.enabled == true
```

### Example 2: DNS on Multiple Manager Nodes

Run DNS on all manager nodes for redundancy:

```yaml
dns:
  deploy:
    mode: replicated
    replicas: 3
    placement:
      constraints:
        - node.role == manager
```

### Example 3: DNS on Specific Region

If you have nodes labeled by region:

```yaml
dns:
  deploy:
    mode: replicated
    replicas: 2
    placement:
      constraints:
        - node.labels.region == us-east-1
        - node.labels.dns.enabled == true
```

## Verification

After deployment, verify DNS is running on the correct nodes:

```bash
# List DNS service tasks
docker service ps obiente_dns

# Check DNS resolution
dig @<dns-node-ip> deploy-123.my.obiente.cloud
nslookup deploy-123.my.obiente.cloud <dns-node-ip>
```

## Troubleshooting

### DNS Not Starting on Expected Nodes

1. Check node labels:
   ```bash
   docker node ls
   docker node inspect <node-name> | grep -A 10 Labels
   ```

2. Verify placement constraints match your labels:
   ```bash
   docker service inspect obiente_dns --pretty
   ```

3. Check service logs:
   ```bash
   docker service logs obiente_dns
   ```

### Port Conflicts

If port 53 is already in use on a node, DNS won't start. Check for conflicts:

```bash
# Check if port 53 is in use
sudo netstat -tulpn | grep :53
sudo ss -tulpn | grep :53
```

### DNS Not Resolving

1. Verify DNS containers are running:
   ```bash
   docker service ps obiente_dns
   ```

2. Check DNS server logs:
   ```bash
   docker service logs obiente_dns --tail 100
   ```

3. Test DNS resolution from within the network:
   ```bash
   docker run --rm --network obiente_obiente-network alpine nslookup deploy-123.my.obiente.cloud
   ```

## Notes

- The DNS server requires `NET_BIND_SERVICE` capability to bind to port 53
- In Docker Swarm, port 53 must be available on each node where DNS runs
- DNS queries are load-balanced across all running DNS instances
- DNS caches responses for 60 seconds to reduce database load

