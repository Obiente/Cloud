# VPS Gateway Setup Guide

This guide explains how to set up the vps-gateway service for per-node DHCP management and SSH proxying for VPS instances.

## Overview

The vps-gateway service provides:

- **DHCP Management**: Automatically allocates and tracks IP addresses for VPS instances
- **SSH Proxying**: Routes SSH connections to VPS instances without requiring SSH keys on the Proxmox node
- **Network Isolation**: Gateway network can be isolated from the main Proxmox network
- **Per-Node Management**: Each Proxmox node has its own gateway service instance for optimal network routing

### Connection Architecture (Forward Connection Pattern)

The vps-gateway uses a **forward connection pattern** where:
- **Gateway exposes gRPC server** on port **1537** (OCG - Obiente Cloud Gateway)
- **API instances connect to gateway** via the gateway's public IP (configured via DNAT)
- Gateway is the server, API is the client
- Port **1537** maps to "O 15 C 3 G" = "OCG" (Obiente Cloud Gateway), similar to how `10.15.3` maps to "O 15 C 3"

For multi-node deployments, each Proxmox node has its own gateway service instance. The API service automatically routes gateway requests to the gateway on the same node as each VPS, ensuring optimal network connectivity. Each gateway handles DHCP and SSH proxying for VPSs on its node, and routes outbound traffic through that node's network interface for low latency.

This guide uses **Proxmox SDN (Software-Defined Networking)** for network topology management, while the vps-gateway service handles DHCP allocation and SSH proxying. SDN is recommended for **all deployments** (both single-node and multi-node clusters) because it provides:

- **Automatic SNAT**: Handles source NAT for internet access automatically
- **Centralized Management**: Configure networks at the datacenter level
- **Automatic Bridge Creation**: Proxmox creates and manages bridges automatically
- **Better Scalability**: Easy to add nodes without manual bridge configuration
- **Network Isolation**: Built-in support for network segmentation
- **Consistent Configuration**: Same setup works for single-node and multi-node deployments

## Architecture

```
┌────────────────────────────────────────────────────────┐
│                Proxmox Datacenter (SDN)                │
│                                                        │
│  ┌──────────────────────────────────────────────────┐  │
│  │                 SDN Zone: OCvps                  │  │
│  │                                                  │  │
│  │  ┌────────────────────────────────────────────┐  │  │
│  │  │              VNet: OCvps-vnet              │  │  │
│  │  │  Subnet: 10.15.3.0/24                      │  │  │
│  │  │  Gateway: 10.15.3.1                        │  │  │
│  │  │  SNAT: Enabled (automatic internet access) │  │  │
│  │  │  (DHCP managed by vps-gateway service)     │  │  │
│  │  └────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────┘  │
│                                                        │
│      ┌────────────┐            ┌──────────────┐        │
│      │   vmbr0    │            │  SDN Bridge  │        │
│      │ (Main Net) │            │(auto-created)│        │
│      └────────────┘            └──────────────┘        │
│             │                         │                │
│             │                         │                │
│      ┌────────────┐            ┌─────────────┐         │
│      │  Proxmox   │            │ Gateway LXC │         │
│      │  Services  │            │(vps-gateway)│         │
│      └────────────┘            └─────────────┘         │
│             │                         │                │
│             │                         │                │
│             │                  ┌─────────────┐         │
│             │                  │  SDN VNet   │         │
│             │                  │  Interface  │         │
│             │                  └─────────────┘         │
│             │                         │                │
│             └────────────┬────────────┘                │
│                          │                             │
│                   ┌─────────────┐                      │
│                   │  VPS VMs    │                      │
│                   │(on SDN VNet)│                      │
│                   └─────────────┘                      │
└────────────────────────────────────────────────────────┘
```

## Prerequisites

- Proxmox VE 7.0+ (SDN support requires Proxmox VE 7.0 or later)
- Proxmox cluster or single node (SDN works for both)
- Access to Proxmox web interface or SSH
- Understanding of network configuration and SDN concepts

## Step 1: Configure Proxmox SDN

Proxmox SDN (Software-Defined Networking) provides centralized network management at the datacenter level. We'll create an SDN zone named "OCvps" and VNet for VPS instances, while the vps-gateway service manages DHCP. SDN works for both single-node and multi-node deployments, and automatically handles SNAT for internet access.

### Prerequisites: Install SDN Components

1. **Ensure SDN is Available**:

   - SDN is included in Proxmox VE 7.0+
   - Verify SDN is enabled: Go to **Datacenter** → **SDN** in the web interface
   - If SDN is not visible, ensure you're running Proxmox VE 7.0 or later

2. **Enable SDN in Network Configuration**:
   ```bash
   # On Proxmox nodes, ensure /etc/network/interfaces includes:
   echo "source /etc/network/interfaces.d/*" >> /etc/network/interfaces
   ```

### Create SDN Zone

1. **Log into Proxmox Web Interface**:

   - Navigate to **Datacenter** → **SDN** → **Zones**

2. **Create Simple Zone**:
   - Click **Add** → **Simple**
   - Configure:
     - **ID**: `OCvps` (recommended name)
   - Click **Create**

### Create VNet (Virtual Network)

1. **Navigate to VNets**:

   - Go to **Datacenter** → **SDN** → **VNets**

2. **Create VNet**:
   - Click **Create**
   - Configure:
     - **VNet**: `OCvpsnet` (recommended name)
     - **Zone**: Select the zone created above (`OCvps`)
     - **Alias**: "Obiente Cloud VPS Virtual Network" (optional)
   - Click **Create**

### Create Subnet

1. **Navigate to Subnets**:

   - Go to **Datacenter** → **SDN** → **VNets** → **OCvpsvn** → **Subnets**

2. **Create Subnet**:
   - Click **Create**
   - Configure:
     - **Subnet**: `10.15.3.0/24` (maps to "O 15 C 3" - Obiente Cloud, uses private IP space)
     - **Gateway**: `10.15.3.1` (this will be the gateway container's IP)
     - **SNAT**: Enable if VPS instances need internet access (recommended)
   - Click **Create**

### Apply SDN Configuration

1. **Apply Changes**:

   - Go to **Datacenter** → **SDN**
   - Click **Apply**
   - Wait for the configuration to be applied to all nodes
   - Verify no errors appear

2. **Verify SDN Configuration**:

```bash
# On Proxmox nodes, check SDN bridges are created
ip addr show | grep -E "vnet|sdn"
# Should show SDN-created bridges

# Check SDN status
pvesh get /cluster/sdn/zones
pvesh get /cluster/sdn/vnets
# Should show your zone and VNet configuration
```

## Step 2: Create Gateway LXC Container

The gateway LXC container will run the vps-gateway service and manage DHCP for VPS instances. It needs access to both the main network (for API access) and the SDN VNet (for DHCP management). The SDN VNet automatically handles SNAT for internet access, so no manual NAT configuration is needed.

LXC containers are more lightweight and native to Proxmox compared to VMs, making them ideal for the gateway service.

### Create LXC Container in Proxmox

1. **Create New CT (Container)**:

   - In Proxmox web interface, click **Create CT**
   - **CT ID**: Choose an available ID (e.g., `200`)
   - **Hostname**: `vps-gateway` (or your preferred name)
   - **Password**: Set root password (or use SSH keys)
   - **Template**: Select a Linux template (Ubuntu, Debian, etc.)

2. **Configure Resources**:

   - **CPU**: 1-2 cores (sufficient for gateway service)
   - **Memory**: 512MB-1GB (512MB minimum)
   - **Disk**: 10GB (minimal, gateway doesn't store much data)
   - **Swap**: 512MB (optional)

3. **Configure Network**:

   - **Bridge**: `OCvpsnet` (your created VNet)
   - **IPv4/CIDR**: `10.15.3.10/24` (for Node 1 - use different IPs for other nodes)
   - **Gateway**: `10.15.3.1` (VNet gateway - same for all nodes)
   - **IPv6**: `static` empty, (disabled)
   - **Firewall**: Enable if using Proxmox firewall
   
   **Important for Multi-Node**: Each node's gateway container needs a **unique IP** on the VXLAN:
   - Node 1: `10.15.3.10`
   - Node 2: `10.15.3.11`
   - Node 3: `10.15.3.12`
   - etc.

4. **Start Container**:

   - Click **Start** to boot the container
   - The container will be ready for configuration

### Verify Network Configuration

On the gateway container:

```bash
# Check interfaces
ip addr show
# Should show:
# - eth0: 10.15.3.10/24 (connected to OCvpsnet bridge on host)

# Test connectivity
ping -c 3 8.8.8.8  # Should work via eth0
ping -c 3 10.15.3.1  # Should work via eth0 (local VXLAN gateway)
```

**Important**: Inside the container/VM, the SDN bridge `OCvpsnet` appears as `eth0` (or `eth1` depending on configuration). The `GATEWAY_DHCP_INTERFACE` should be set to the interface name **inside the container** (e.g., `eth0`), not the bridge name on the host (`OCvpsnet`).

## Step 3: Install Prerequisites on Gateway Container

**For Docker deployments**: dnsmasq is already included in the Docker image (installed in the Dockerfile), so no manual installation is needed. Skip to Step 4.

**For systemd/native deployments**: You need to install dnsmasq on the gateway container/VM:

### Ubuntu/Debian

```bash
sudo apt update
sudo apt install -y dnsmasq curl
```

### CentOS/Rocky/AlmaLinux

```bash
sudo yum install -y dnsmasq curl
# Or for newer versions:
sudo dnf install -y dnsmasq curl
```

**Note**: The vps-gateway service runs its own dnsmasq instance. For Docker deployments, dnsmasq runs **inside the container** (with `network_mode: host` to access host interfaces). For systemd deployments, it runs as a subprocess of the service. You should **disable the system dnsmasq service** on the host/container to avoid conflicts:

```bash
# Disable system dnsmasq (if installed)
sudo systemctl stop dnsmasq
sudo systemctl disable dnsmasq
```

The vps-gateway service will manage dnsmasq automatically - you don't need to configure or start it manually.

**Important DNS Configuration:**

The gateway's dnsmasq acts as both a DHCP server and a DNS server. For SSH proxying to work with VPS hostnames (e.g., `vps-1762797183902442010`), the gateway must be able to resolve these hostnames. The gateway automatically:

1. **Configures dnsmasq as a DNS server** on port 53
2. **Sets up a local domain** (default: `vps.local`, configurable via `GATEWAY_DHCP_DOMAIN`)
3. **Maps VPS hostnames to IP addresses** in the `dnsmasq.hosts` file
4. **Resolves hostnames** like `vps-1762797183902442010` or `vps-1762797183902442010.vps.local`

**DNS Domain Configuration:**

- **Default domain**: `vps.local` (if `GATEWAY_DHCP_DOMAIN` is not set)
- **Custom domain**: Set `GATEWAY_DHCP_DOMAIN` to your preferred domain (e.g., `vps.internal`, `vps.obiente.local`)
- **Hostname resolution**: VPS hostnames can be resolved as:
  - `vps-1762797183902442010` (short name, requires `expand-hosts`)
  - `vps-1762797183902442010.vps.local` (FQDN)

**Note**: The gateway itself uses the configured DNS servers (from `GATEWAY_DHCP_DNS`) for upstream DNS resolution, while also serving DNS for the local VPS network.

## Step 4: Deploy vps-gateway Service

Prerequisites:

- Docker
  - https://docs.docker.com/engine/install/

You can deploy the vps-gateway service in several ways:

### Option A: Docker Compose

1. **Create Directory**:

```bash
mkdir -p /opt/vps-gateway
cd /opt/vps-gateway
```

2. **Create `docker-compose.yml`**:

```yaml
services:
  vps-gateway:
    image: ghcr.io/obiente/cloud-vps-gateway:latest
    # Or build locally:
    # build:
    #   context: /path/to/cloud/apps/vps-gateway
    #   dockerfile: Dockerfile
    container_name: vps-gateway
    restart: unless-stopped
    network_mode: host # Required for DHCP management
    privileged: true # Required for network management
    environment:
      # API Connection (gateway connects to API)
      GATEWAY_API_SECRET: ${GATEWAY_API_SECRET:-change-me-in-production}
      # DHCP Configuration
      GATEWAY_DHCP_POOL_START: 10.15.3.20
      GATEWAY_DHCP_POOL_END: 10.15.3.254
      GATEWAY_DHCP_SUBNET: 10.15.3.0
      GATEWAY_DHCP_SUBNET_MASK: 255.255.255.0
      GATEWAY_DHCP_GATEWAY: 10.15.3.1
      GATEWAY_DHCP_DNS: 1.1.1.1,1.0.0.1  # Upstream DNS servers
      GATEWAY_DHCP_DOMAIN: vps.local      # DNS domain for VPS hostname resolution (optional, defaults to vps.local)
      GATEWAY_DHCP_INTERFACE: eth0 # Interface inside container (connected to OCvpsnet bridge on host)
      LOG_LEVEL: info
    volumes:
      - /var/lib/obiente/vps-gateway:/var/lib/obiente/vps-gateway
    healthcheck:
      test: ["CMD-SHELL", "pgrep -f vps-gateway || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

3. **Create `.env` File**:

```bash
cat > .env << EOF
# Shared secret (must match VPS_GATEWAY_API_SECRET in API service)
GATEWAY_API_SECRET=your-secure-random-secret-here

# Optional: Dedicated outbound IP for traffic isolation
GATEWAY_OUTBOUND_IP=203.0.113.10
EOF
```

**Important**: Generate a secure random secret:

```bash
# Generate a secure random secret
openssl rand -hex 32
```


4. **Start Service**:

```bash
docker compose up -d
```

### Option B: Docker Swarm (Production)

If you're using Docker Swarm, add the vps-gateway service to your `docker-compose.swarm.yml`:

The service is already configured in `docker-compose.swarm.yml`. Ensure:

1. **Set Environment Variables**:

```bash
# API Connection
export GATEWAY_API_SECRET=$(openssl rand -hex 32)  # Must match VPS_GATEWAY_API_SECRET in API

# DHCP Configuration
export GATEWAY_DHCP_POOL_START=10.15.3.10
export GATEWAY_DHCP_POOL_END=10.15.3.254
export GATEWAY_DHCP_SUBNET=10.15.3.0
export GATEWAY_DHCP_SUBNET_MASK=255.255.255.0
export GATEWAY_DHCP_GATEWAY=10.15.3.1
export GATEWAY_DHCP_DNS=1.1.1.1,1.0.0.1
export GATEWAY_DHCP_INTERFACE=eth0  # Interface inside container (connected to OCvpsnet bridge)
```

2. **Deploy Service**:

```bash
docker stack deploy -c docker-compose.swarm.yml obiente
```

**Note**: For production, the gateway container should be deployed with `network_mode: host` or have direct access to the SDN bridge. You may need to deploy the service directly on the gateway container rather than in the Swarm cluster.

### Option C: Systemd Service (Native)

1. **Build Binary** (on gateway container or build machine):

```bash
cd /path/to/cloud/apps/vps-gateway
go build -o vps-gateway ./main.go
```

2. **Install Binary**:

```bash
sudo cp vps-gateway /usr/local/bin/
sudo chmod +x /usr/local/bin/vps-gateway
```

3. **Create Systemd Service**:

```bash
sudo tee /etc/systemd/system/vps-gateway.service > /dev/null << 'EOF'
[Unit]
Description=VPS Gateway Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/vps-gateway
Restart=always
RestartSec=5
Environment="GATEWAY_API_SECRET=your-secure-random-secret-here"
Environment="GATEWAY_DHCP_POOL_START=10.15.3.10"
Environment="GATEWAY_DHCP_POOL_END=10.15.3.254"
Environment="GATEWAY_DHCP_SUBNET=10.15.3.0"
Environment="GATEWAY_DHCP_SUBNET_MASK=255.255.255.0"
Environment="GATEWAY_DHCP_GATEWAY=10.15.3.1"
Environment="GATEWAY_DHCP_DNS=8.8.8.8,8.8.4.4"
Environment="GATEWAY_DHCP_INTERFACE=eth0"  # Interface inside container (connected to OCvpsnet bridge)
Environment="LOG_LEVEL=info"

[Install]
WantedBy=multi-user.target
EOF
```

4. **Start Service**:

```bash
sudo systemctl daemon-reload
sudo systemctl enable vps-gateway
sudo systemctl start vps-gateway
```

## Step 5: Configure API to Use Gateway

Configure the API service to connect to the gateway:

### Environment Variables

Add to your API service configuration (`.env` or `docker-compose.yml`):

```bash
# Map Proxmox node names to gateway URLs (required for multi-node)
VPS_NODE_GATEWAY_ENDPOINTS="node1:http://gateway1:1537,node2:http://gateway2:1537"

# API Secret (must match GATEWAY_API_SECRET in gateway service)
VPS_GATEWAY_API_SECRET=your-secure-random-secret-here

# SDN VNet bridge name (find this from Proxmox SDN configuration)
# This is the bridge name created by SDN for your VNet
# Check: Datacenter → SDN → VNets → your-vnet → check bridge name
# Or run: ip addr show | grep -E "vnet|sdn" on Proxmox nodes
VPS_GATEWAY_BRIDGE=OCvpsnet  # SDN bridge name for OCvps-vnet
```

### Update Docker Compose

If using Docker Compose, add to your `docker-compose.yml`:

```yaml
services:
  api:
    environment:
      # Map Proxmox node names to gateway URLs (required for multi-node)
      VPS_NODE_GATEWAY_ENDPOINTS: "node1:http://gateway1:1537,node2:http://gateway2:1537"
      # API Secret (must match GATEWAY_API_SECRET in gateway service)
      VPS_GATEWAY_API_SECRET: ${VPS_GATEWAY_API_SECRET}
      # SDN VNet bridge name
      VPS_GATEWAY_BRIDGE: OCvpsnet # SDN bridge name for OCvps-vnet
```

### Update Docker Swarm

If using Docker Swarm, the environment variables are already configured in `docker-compose.swarm.yml`. Set them in your environment:

```bash
# Map Proxmox node names to gateway URLs (required for multi-node)
export VPS_NODE_GATEWAY_ENDPOINTS="node1:http://gateway1:1537,node2:http://gateway2:1537"
# API Secret (must match GATEWAY_API_SECRET in gateway service)
export VPS_GATEWAY_API_SECRET=your-secure-random-secret-here
# SDN VNet bridge name
export VPS_GATEWAY_BRIDGE=OCvpsnet  # SDN bridge name for OCvps-vnet
```

## Step 6: Understanding Multi-Instance Behavior (Docker Swarm)

When running multiple API instances in Docker Swarm, it's important to understand how the gateway connects and how requests are handled:

### How Gateway Connection Works (Forward Connection Pattern)

1. **Gateway Exposes gRPC Server**: The gateway runs a gRPC server on port **1537** (OCG - Obiente Cloud Gateway):
   - Port **1537** maps to "O 15 C 3 G" = "OCG" (Obiente Cloud Gateway)
   - Gateway listens on all interfaces (0.0.0.0:1537) or specific interface
   - Gateway is accessible via public IP configured with DNAT

2. **API Instances Connect to Gateway**: Each API instance connects to the gateway independently:
   - API instances use `VPS_NODE_GATEWAY_ENDPOINTS` environment variable to map nodes to gateway URLs (e.g., `"node1:http://gateway1:1537,node2:http://gateway2:1537"`)
   - Each API instance maintains its own connection to the gateway
   - Multiple API instances can connect to the same gateway concurrently

3. **Request Handling**: 
   - Operations like IP allocation and SSH proxying are handled directly via gRPC calls
   - Each API instance communicates with the gateway independently
   - No shared registry needed - gateway handles all requests directly

4. **DNAT Configuration**: For public IP access:
   - Configure DNAT on your router/firewall to forward port 1537 to the gateway's internal IP
   - Gateway's public IP should be accessible from API instances
   - Example: `iptables -t nat -A PREROUTING -p tcp --dport 1537 -j DNAT --to-destination <gateway-internal-ip>:1537`

### Multi-Node Gateway Configuration

For multi-node deployments, each Proxmox node should have its own gateway service instance. The API service automatically routes gateway requests to the gateway on the same node as each VPS.

**Required Configuration:**

1. **Deploy gateway on each node**: Each Proxmox node needs its own vps-gateway service instance
2. **Configure VPS_NODE_GATEWAY_ENDPOINTS**: Map Proxmox node names to gateway URLs in the API service

**Example API Service Configuration:**

```bash
# Map Proxmox node names to gateway URLs
VPS_NODE_GATEWAY_ENDPOINTS="node1:http://gateway1.example.com:1537,node2:http://gateway2.example.com:1537,node3:http://gateway3.example.com:1537"
VPS_GATEWAY_API_SECRET=your-shared-secret
```

**Gateway Service Configuration per Node:**

Each gateway should be configured with:
- **Unique listen IP** on the VXLAN (required for multi-node - each node needs different IP)
- Node-specific outbound IP (optional but recommended for traffic isolation)
- Correct network interface for the node
- Same DHCP pool configuration (all gateways share the same VXLAN)

**Example Gateway Configuration (Node 1):**

```yaml
GATEWAY_DHCP_LISTEN_IP: 10.15.3.10  # Node 1's unique IP on VXLAN
GATEWAY_DHCP_GATEWAY: 10.15.3.1     # VXLAN gateway (same for all nodes)
GATEWAY_DHCP_INTERFACE: eth0     # Interface inside container (connected to OCvpsnet bridge)
GATEWAY_OUTBOUND_IP: 203.0.113.10  # Node 1's dedicated outbound IP
GATEWAY_API_SECRET: your-shared-secret
```

**Example Gateway Configuration (Node 2):**

```yaml
GATEWAY_DHCP_LISTEN_IP: 10.15.3.11  # Node 2's unique IP on VXLAN (different!)
GATEWAY_DHCP_GATEWAY: 10.15.3.1     # VXLAN gateway (same for all nodes)
GATEWAY_DHCP_INTERFACE: eth0     # Interface inside container (connected to OCvpsnet bridge)
GATEWAY_OUTBOUND_IP: 203.0.113.11  # Node 2's dedicated outbound IP
GATEWAY_API_SECRET: your-shared-secret
```

**IP Allocation Recommendations:**

- `10.15.3.1`: VXLAN Gateway/Router (Proxmox SDN)
- `10.15.3.2-9`: Reserved for infrastructure
- `10.15.3.10-19`: Gateway service IPs (one per node)
- `10.15.3.20-254`: VPS IP pool (allocated by gateways)

### Best Practices for Multi-Node Deployments

- **Per-Node Gateways**: Each Proxmox node should have its own gateway service instance
- **Node Mapping**: Configure `VPS_NODE_GATEWAY_ENDPOINTS` to map each node to its gateway URL
- **Outbound IP Isolation**: Use `GATEWAY_OUTBOUND_IP` to isolate VPS traffic from other infrastructure
- **Network Routing**: Ensure each gateway routes outbound traffic through its node's network interface
- **High Availability**: If a gateway goes down, only VPSs on that node are affected
- **DNAT Setup**: Configure DNAT rules to expose each gateway's port 1537 on a public IP accessible to API instances

## Step 7: Verify Setup

### Check Gateway Service

1. **Check Service Status**:

```bash
# Docker Compose
docker compose ps

# Systemd
sudo systemctl status vps-gateway

# Docker Swarm
docker service ps vps-gateway
```

2. **Check Logs**:

```bash
# Docker Compose
docker compose logs vps-gateway

# Systemd
sudo journalctl -u vps-gateway -f

# Docker Swarm
docker service logs vps-gateway -f
```

3. **Verify Gateway is Listening**:

```bash
# Check gateway logs for startup
docker compose logs vps-gateway | grep -i "listening\|started\|gRPC"

# Should see messages like:
# "[GatewayServer] Starting gRPC server on :1537"
# "[GatewayServer] Gateway server started"
```

4. **Verify API Can Connect to Gateway**:

```bash
# Check API logs for gateway connections
docker compose logs api | grep -i "gateway.*node\|connected.*gateway"

# Should see successful gateway client creation for each node
```

5. **Check Gateway Metrics**:

```bash
# Gateway exposes metrics on port 9091 (if enabled)
curl http://gateway-host:9091/metrics | grep -i "vps_gateway"

# Should see gateway metrics
```

### Check DHCP Service

1. **Check dnsmasq Status**:

   ```bash
   # On gateway container
   ps aux | grep dnsmasq
   # Should show dnsmasq process

   # Check dnsmasq config
   cat /var/lib/obiente/vps-gateway/dnsmasq.conf
   # Should show DHCP pool configuration and DNS server settings
   ```

2. **Check DHCP Leases**:

   ```bash
   # On gateway container
   cat /var/lib/obiente/vps-gateway/dnsmasq.leases
   # Should show allocated IPs (empty initially)

   # Or from the host (if using bind mount)
   cat /var/lib/obiente/vps-gateway/dnsmasq.leases
   ```

3. **Check DNS Resolution**:

   ```bash
   # On gateway container, test DNS resolution
   # The gateway's dnsmasq acts as a DNS server for VPS hostnames
   nslookup vps-1762797183902442010.vps.local 127.0.0.1
   # Or using the gateway IP (if configured as DNS server)
   nslookup vps-1762797183902442010.vps.local 10.15.3.1
   
   # Check hosts file (used by dnsmasq for DNS resolution)
   cat /var/lib/obiente/vps-gateway/dnsmasq.hosts
   # Should show IP-to-hostname mappings for allocated VPSes
   ```

### Test VPS Creation

1. **Create a Test VPS**:

   - Use the dashboard or API to create a VPS instance
   - The API should allocate an IP from the gateway

2. **Check IP Allocation**:

```bash
# List allocated IPs via gRPC
grpcurl -plaintext \
  -H "x-api-secret: your-secure-random-secret-here" \
  10.15.3.10:1537 \
  obiente.cloud.vpsgateway.v1.VPSGatewayService/ListIPs
```

3. **Verify VPS Network**:
   - Check that the VPS receives an IP from the DHCP pool
   - Verify the VPS can reach the gateway (10.15.3.1)
   - Verify the VPS can reach the internet (via gateway)

## Step 8: Configure Firewall (Optional)

If using a firewall on the gateway container, allow necessary ports:

### UFW (Ubuntu/Debian)

```bash
sudo ufw allow 67/udp   # DHCP server
sudo ufw allow 68/udp   # DHCP client
sudo ufw allow 53/udp   # DNS (if dnsmasq provides DNS)
sudo ufw allow 53/tcp   # DNS (if dnsmasq provides DNS)
```

### firewalld (CentOS/Rocky/AlmaLinux)

```bash
sudo firewall-cmd --permanent --add-port=67/udp
sudo firewall-cmd --permanent --add-port=68/udp
sudo firewall-cmd --permanent --add-port=53/udp
sudo firewall-cmd --permanent --add-port=53/tcp
sudo firewall-cmd --reload
```

## Troubleshooting

### Gateway Service Won't Start

1. **Check Logs**:

   ```bash
   docker compose logs vps-gateway
   # Or
   sudo journalctl -u vps-gateway -n 50
   ```

2. **Common Issues**:
   - **"GATEWAY_API_SECRET not set"**: Ensure environment variable is set
   - **"Failed to initialize DHCP manager"**: Check network interface name (`GATEWAY_DHCP_INTERFACE`)
   - **"Permission denied"**: Ensure container has `privileged: true` or `network_mode: host`
   - **"VPS_NODE_GATEWAY_ENDPOINTS not configured"**: Ensure node-to-gateway mapping is set in API service
   - **"No gateway endpoint configured for node"**: Verify the node name matches the mapping in `VPS_NODE_GATEWAY_ENDPOINTS`

### DHCP Not Working

1. **Check dnsmasq Process**:

   ```bash
   ps aux | grep dnsmasq
   ```

2. **Check Interface**:

   ```bash
   ip addr show eth0  # Inside container, OCvpsnet bridge appears as eth0
   # Should show the gateway container's IP (e.g., 10.15.3.10/24 for Node 1)
   # The IP depends on which node the gateway is on (10.15.3.10, 10.15.3.11, etc.)
   ```

3. **Check dnsmasq Config**:

   ```bash
   cat /var/lib/obiente/vps-gateway/dnsmasq.conf
   # Verify DHCP pool settings
   ```

4. **Check Firewall**:
   ```bash
   sudo ufw status
   # Or
   sudo firewall-cmd --list-all
   ```

### API Can't Connect to Gateway

**Note**: The API connects to gateways (forward connection pattern). Each API instance routes requests to the gateway on the same node as each VPS.

1. **Check Node-to-Gateway Mapping**:

   ```bash
   # Verify VPS_NODE_GATEWAY_ENDPOINTS is configured
   echo $VPS_NODE_GATEWAY_ENDPOINTS
   # Should show: "node1:http://gateway1:1537,node2:http://gateway2:1537"
   ```

2. **Verify Gateway URLs**:

   - Ensure each gateway URL in `VPS_NODE_GATEWAY_ENDPOINTS` is accessible from API instances
   - Test connectivity: `curl http://gateway1:1537` (should connect)
   - Check DNAT rules if gateways are behind NAT

3. **Check API Secret**:
   - Ensure `VPS_GATEWAY_API_SECRET` in API matches `GATEWAY_API_SECRET` in gateway service
   - Both must be identical

4. **Check Gateway Status**:

   ```bash
   # Check gateway logs
   docker compose logs vps-gateway | grep -i "listening\|started\|error"
   
   # Verify gateway is listening on port 1537
   netstat -tlnp | grep 1537
   # Or
   ss -tlnp | grep 1537
   ```

5. **Check API Logs for Gateway Errors**:

   ```bash
   # Check API logs for gateway connection errors
   docker compose logs api | grep -i "gateway.*error\|failed.*gateway\|node.*gateway"
   
   # Should see successful connections or specific error messages
   ```

### VPS Not Getting IP Address

1. **Check Gateway Connection** (Swarm):

   In a Swarm deployment, verify which API instance the gateway is connected to:
   
   ```bash
   # Check API logs for gateway registration
   docker service logs obiente_api | grep -i "gateway.*registered"
   
   # Check which API instance has the gateway
   docker service ps obiente_api --format "table {{.Name}}\t{{.Node}}\t{{.CurrentState}}"
   ```

2. **Check Gateway Allocations via API**:

   Check allocations through the API:
   
   ```bash
   # Docker Compose
   docker compose logs api | grep -i "allocated.*ip\|vps.*ip"
   
   # Docker Swarm - check all API instances
   docker service logs obiente_api | grep -i "allocated.*ip\|vps.*ip"
   
   # Note: In Swarm, the gateway may be connected to a different API instance
   # than the one handling the request. Check logs from all API instances.
   ```

3. **Check VPS Network Configuration**:

   - Verify VPS is connected to the SDN VNet bridge (check in Proxmox VM configuration)
   - Check VPS network interface is configured for DHCP
   - Verify VPS can reach gateway (10.15.3.1)

4. **Check DHCP Leases**:

   ```bash
   # On gateway container
   cat /var/lib/obiente/vps-gateway/dnsmasq.leases

   # Or from the host (using bind mount)
   cat /var/lib/obiente/vps-gateway/dnsmasq.leases
   ```

## Network Configuration Examples

### Example 1: Single-Node SDN Configuration

```
Main Network (vmbr0): 192.168.1.0/24
SDN VNet (OCvpsnet): 10.15.3.0/24

SDN Configuration:
  - Zone: OCvps
  - VNet: OCvps-vnet
  - Subnet: 10.15.3.0/24 (maps to "O 15 C 3" - Obiente Cloud, uses private IP space)
  - Gateway: 10.15.3.1
  - SNAT: Enabled (automatic internet access - no manual NAT needed)

Gateway Container:
  - eth0 (connected to OCvpsnet bridge on host): 10.15.3.10
  - Gateway IP: 10.15.3.1 (VXLAN gateway)

VPS VMs:
  - Connected to SDN VNet bridge (OCvpsnet)
  - Receive IPs: 10.15.3.10-254 (via vps-gateway DHCP)
  - Automatic internet access via SNAT
```

### Example 2: Multi-Node SDN Cluster

When using SDN with multiple Proxmox nodes:

```
Proxmox Cluster:
  - Node 1: Main node
  - Node 2: Additional node
  - Node 3: Additional node

SDN Configuration (applied to all nodes):
  - Zone: OCvps (datacenter-wide)
  - VNet: OCvps-vnet (spans all nodes)
  - Subnet: 10.15.3.0/24 (maps to "O 15 C 3" - Obiente Cloud, uses private IP space)
  - SNAT: Enabled (automatic routing)

Gateway Containers:
  - Node 1 Gateway: eth0 (connected to OCvpsnet bridge): 10.15.3.10
  - Node 2 Gateway: eth0 (connected to OCvpsnet bridge): 10.15.3.11
  - Node 3 Gateway: eth0 (connected to OCvpsnet bridge): 10.15.3.12
  - Each gateway manages DHCP for VPSs on its node
  - Each gateway routes outbound traffic through its node's network interface

VPS VMs:
  - Can be created on any node
  - All connected to same SDN VNet (VXLAN)
  - Each VPS uses the gateway on its node for DHCP and routing
  - Can communicate across nodes via SDN
  - Outbound traffic routes through the gateway on the same node
```

**Note**: With SDN, SNAT is handled automatically when enabled in the subnet configuration. However, for multi-node deployments, each gateway should route outbound traffic through its node's network interface, and optionally use a dedicated outbound IP for traffic isolation.

## Multi-Node Network Routing

### Critical Requirements

For multi-node deployments, it's critical that:

1. **Each node's gateway routes outbound traffic through that node's own network interface** (ensures low latency)
2. **Each gateway can use a dedicated outbound IP for SNAT** (allows traffic isolation and prevents abuse/blocking from affecting other services)

### VXLAN Architecture

- All VPSs are on the same VXLAN (shared network segment)
- VPSs can communicate with each other across nodes via VXLAN
- Each node has its own gateway service instance
- Each gateway handles DHCP and SSH proxying for VPSs on its node

### Outbound Traffic Routing

- VPSs on node1 must route outbound traffic through node1's network interface
- VPSs on node2 must route outbound traffic through node2's network interface
- This ensures traffic doesn't cross nodes unnecessarily, reducing latency

### Outbound IP Configuration

Each gateway can be configured with `GATEWAY_OUTBOUND_IP` environment variable:

- This IP is used for SNAT on all outbound traffic from VPSs on that node
- Allows isolation: VPS traffic uses dedicated IP, other infrastructure uses different IP
- Prevents abuse/blocking on VPS traffic from affecting other services
- Each node can have a different outbound IP

**Configuration:**

```yaml
GATEWAY_OUTBOUND_IP: 203.0.113.10  # Dedicated IP for VPS traffic on this node
```

**Network Setup Requirements:**

1. The outbound IP must be assigned to the gateway's network interface (or the host's primary interface)
2. The gateway container runs with `network_mode: host` and `privileged: true` to allow iptables configuration
3. **iptables SNAT rules are automatically configured** by the gateway service on startup
4. Rules are automatically cleaned up on gateway shutdown

**Automatic SNAT Configuration:**

The gateway service automatically:
- Detects the VPS subnet from DHCP configuration
- Auto-detects the outbound interface (from default route) or uses `GATEWAY_OUTBOUND_INTERFACE` if set
- Configures iptables SNAT rules on startup
- Removes SNAT rules on shutdown

**Manual Interface Selection (Optional):**

If you need to specify a specific outbound interface, set `GATEWAY_OUTBOUND_INTERFACE`:

```yaml
GATEWAY_OUTBOUND_IP: 203.0.113.10
GATEWAY_OUTBOUND_INTERFACE: eth0  # Optional: specify outbound interface manually (usually vmbr0, not OCvpsnet)
```

If not set, the gateway will auto-detect the interface from the default route.

### Gateway Configuration per Node

Each gateway service must be configured with:

1. **Network Interface Binding**: Gateway should bind to the node's primary network interface
2. **SNAT Configuration**: Gateway automatically configures iptables SNAT rules if `GATEWAY_OUTBOUND_IP` is set, otherwise uses node's default IP
3. **DHCP Interface**: Gateway's DHCP should listen on the VXLAN interface
4. **Outbound IP**: Configure specific IP address for outbound traffic (optional but recommended)

**Example Configuration:**

For Node 1:
```yaml
GATEWAY_DHCP_INTERFACE: eth0  # Interface inside container (connected to OCvpsnet bridge)
GATEWAY_OUTBOUND_IP: 203.0.113.10  # Dedicated IP for VPS traffic
# Gateway routes outbound through node1's network interface (eth0)
# All outbound traffic from VPSs on node1 will use 203.0.113.10
```

For Node 2:
```yaml
GATEWAY_DHCP_INTERFACE: eth0  # Interface inside container (connected to OCvpsnet bridge)
GATEWAY_OUTBOUND_IP: 203.0.113.11  # Different dedicated IP for VPS traffic
# Gateway routes outbound through node2's network interface (eth0)
# All outbound traffic from VPSs on node2 will use 203.0.113.11
```

### Network Interface Setup

Each gateway container should:

- Have access to the VXLAN interface (for DHCP and VPS communication)
- Use the node's primary network interface for outbound routing (auto-detected or via `GATEWAY_OUTBOUND_INTERFACE`)
- Automatically configure iptables SNAT rules if `GATEWAY_OUTBOUND_IP` is set
- Ensure the outbound IP is assigned to the host's network interface (or gateway container's interface)

### Verification

To verify outbound routing and IP configuration:

1. SSH into a VPS on node1
2. Run: `curl ifconfig.me` (should show node1's outbound IP, or `GATEWAY_OUTBOUND_IP` if configured)
3. SSH into a VPS on node2
4. Run: `curl ifconfig.me` (should show node2's outbound IP, or `GATEWAY_OUTBOUND_IP` if configured)
5. Verify that other infrastructure services use different IPs (not affected by VPS traffic)

### Finding SDN Bridge Names

After creating an SDN VNet, you need to find the bridge name for API configuration:

1. **Via Proxmox Web Interface**:

   - Go to **Datacenter** → **SDN** → **VNets**
   - Click on your VNet (`OCvps-vnet`)
   - The bridge name may be shown in the details or you can check the network interfaces

2. **Via Command Line**:

   ```bash
   # On Proxmox nodes
   ip addr show | grep -E "vnet|sdn"
   # Or check all bridges
   brctl show | grep -v "vmbr0"
   # Or check SDN-generated configs
   cat /etc/network/interfaces.d/sdn-*
   ```

3. **Via Proxmox API**:

   ```bash
   curl -H "Authorization: PVEAPIToken=USER@REALM!TOKENID=SECRET" \
     https://your-proxmox:8006/api2/json/cluster/sdn/vnets
   ```

4. **Check Network Interfaces**:
   ```bash
   # On Proxmox nodes
   ls -la /etc/network/interfaces.d/
   # SDN configs are auto-generated here
   cat /etc/network/interfaces.d/sdn-*
   ```

Once you have the bridge name, set it in `VPS_GATEWAY_BRIDGE` environment variable in your API configuration.

## Security Considerations

1. **API Secret**:

   - Use a strong, random secret
   - Never commit secrets to version control
   - Rotate secrets periodically

2. **Network Isolation**:

   - SDN VNet provides network isolation from main network
   - Use SDN firewall rules and Proxmox firewall to restrict access
   - Only allow necessary ports
   - SDN zones can be configured for additional isolation

3. **Gateway Container Security**:

   - Keep gateway container updated
   - Use SSH keys instead of passwords
   - Restrict SSH access to necessary IPs
   - Monitor gateway logs for suspicious activity
   - Consider using unprivileged containers for additional security

4. **DHCP Security**:
   - Gateway manages DHCP, preventing IP conflicts
   - MAC address binding ensures IP consistency
   - Lease tracking prevents IP exhaustion

## Next Steps

After setting up the gateway:

1. **Create Test VPS**: Create a VPS instance and verify it receives an IP from the gateway
2. **Test SSH Proxy**: Connect to the VPS via SSH proxy to verify SSH proxying works
3. **Monitor Metrics**: Check Grafana dashboard for gateway metrics
4. **Scale**: Add more VPS instances and monitor DHCP pool utilization

## Why Use SDN?

SDN is recommended for **all deployments** (single-node and multi-node) because it provides:

**Key Advantages:**

- **Automatic SNAT**: Handles source NAT for internet access automatically - no manual iptables configuration needed
- **Centralized Management**: Configure networks at datacenter level
- **Automatic Bridge Creation**: Proxmox creates and manages bridges automatically
- **Better Scalability**: Easy to add nodes without manual bridge configuration
- **Network Isolation**: Built-in support for network segmentation
- **Consistent Configuration**: Same setup works for single-node and multi-node deployments
- **Multi-Node Support**: SDN spans entire Proxmox cluster automatically when you scale

**SDN Zone and VNet Naming**

We recommend using:

- **Zone**: `OCvps` (Obiente Cloud VPS)
- **VNet**: `OCvps-vnet` (Obiente Cloud VPS Virtual Network)

This provides clear naming and easy identification in the Proxmox interface.

## Related Documentation

- [VPS Provisioning Guide](vps-provisioning.md) - General VPS setup
- [VPS IP Security](../reference/vps-ip-security.md) - Automatic IP hijacking prevention
- [Environment Variables Reference](../reference/environment-variables.md) - Complete variable reference
- [Monitoring Guide](monitoring.md) - Prometheus and Grafana setup
- [Proxmox SDN Documentation](https://pve.proxmox.com/wiki/Software_Defined_Networking) - Official Proxmox SDN guide
