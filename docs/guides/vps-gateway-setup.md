# VPS Gateway Setup Guide

This guide explains how to set up the vps-gateway service for centralized DHCP management and SSH proxying for VPS instances.

## Overview

The vps-gateway service provides:

- **DHCP Management**: Automatically allocates and tracks IP addresses for VPS instances
- **SSH Proxying**: Routes SSH connections to VPS instances without requiring SSH keys on the Proxmox node
- **Network Isolation**: Gateway network can be isolated from the main Proxmox network
- **Centralized Management**: Single service manages all VPS networking

### Connection Architecture (Forward Connection Pattern)

The vps-gateway uses a **forward connection pattern** where:
- **Gateway exposes gRPC server** on port **1537** (OCG - Obiente Cloud Gateway)
- **API instances connect to gateway** via the gateway's public IP (configured via DNAT)
- Gateway is the server, API is the client
- Port **1537** maps to "O 15 C 3 G" = "OCG" (Obiente Cloud Gateway), similar to how `10.15.3` maps to "O 15 C 3"

When running multiple API instances (e.g., in Docker Swarm), each API instance connects to the gateway independently. The gateway can handle multiple concurrent connections from different API instances. For operations like IP allocation and SSH proxying, any API instance can communicate with the gateway directly.

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
   - **IPv4/CIDR**: `10.15.3.10/24`
   - **Gateway**: `10.15.3.1` (VNet gateway)
   - **IPv6**: `static` empty, (disabled)
   - **Firewall**: Enable if using Proxmox firewall

4. **Start Container**:

   - Click **Start** to boot the container
   - The container will be ready for configuration

### Verify Network Configuration

On the gateway container:

```bash
# Check interfaces
ip addr show
# Should show:
# - eth0: 10.15.3.10/24

# Test connectivity
ping -c 3 8.8.8.8  # Should work via eth0
ping -c 3 10.15.3.1  # Should work via eth1 (local)
```

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
      # API Connection
      GATEWAY_API_URL: http://api:3001  # API URL for gateway to connect to
      GATEWAY_API_SECRET: ${GATEWAY_API_SECRET:-change-me-in-production}
      # DHCP Configuration
      GATEWAY_DHCP_POOL_START: 10.15.3.20
      GATEWAY_DHCP_POOL_END: 10.15.3.254
      GATEWAY_DHCP_SUBNET: 10.15.3.0
      GATEWAY_DHCP_SUBNET_MASK: 255.255.255.0
      GATEWAY_DHCP_GATEWAY: 10.15.3.1
      GATEWAY_DHCP_DNS: 1.1.1.1,1.0.0.1
      GATEWAY_DHCP_INTERFACE: eth0 # Interface on SDN VNet
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
# API URL (gateway connects to API, not the other way around)
GATEWAY_API_URL=http://api:3001
# Or if API is on a different host:
# GATEWAY_API_URL=http://your-api-host:3001

# Shared secret (must match VPS_GATEWAY_API_SECRET in API service)
GATEWAY_API_SECRET=your-secure-random-secret-here
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
export GATEWAY_API_URL=http://api:3001  # Or your API hostname/IP
export GATEWAY_API_SECRET=$(openssl rand -hex 32)  # Must match VPS_GATEWAY_API_SECRET in API

# DHCP Configuration
export GATEWAY_DHCP_POOL_START=10.15.3.10
export GATEWAY_DHCP_POOL_END=10.15.3.254
export GATEWAY_DHCP_SUBNET=10.15.3.0
export GATEWAY_DHCP_SUBNET_MASK=255.255.255.0
export GATEWAY_DHCP_GATEWAY=10.15.3.1
export GATEWAY_DHCP_DNS=1.1.1.1,1.0.0.1
export GATEWAY_DHCP_INTERFACE=eth0
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
      Environment="GATEWAY_API_URL=http://api:3001"
      Environment="GATEWAY_API_SECRET=your-secure-random-secret-here"
Environment="GATEWAY_DHCP_POOL_START=10.15.3.10"
Environment="GATEWAY_DHCP_POOL_END=10.15.3.254"
Environment="GATEWAY_DHCP_SUBNET=10.15.3.0"
Environment="GATEWAY_DHCP_SUBNET_MASK=255.255.255.0"
Environment="GATEWAY_DHCP_GATEWAY=10.15.3.1"
Environment="GATEWAY_DHCP_DNS=8.8.8.8,8.8.4.4"
Environment="GATEWAY_DHCP_INTERFACE=eth1"
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
      # API Secret (must match GATEWAY_API_SECRET in gateway service)
      VPS_GATEWAY_API_SECRET: ${VPS_GATEWAY_API_SECRET}
      # SDN VNet bridge name
      VPS_GATEWAY_BRIDGE: OCvpsnet # SDN bridge name for OCvps-vnet
```

### Update Docker Swarm

If using Docker Swarm, the environment variables are already configured in `docker-compose.swarm.yml`. Set them in your environment:

```bash
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
   - API instances use `VPS_GATEWAY_URL` environment variable (e.g., `http://gateway-public-ip:1537`)
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

### Best Practices for Swarm Deployments

- **Gateway Discovery**: API instances discover gateway via `VPS_GATEWAY_URL` environment variable or node metadata
- **Multiple Gateways**: If you have multiple gateways (e.g., one per Proxmox node), configure each API instance to connect to the appropriate gateway based on which node hosts the VPS
- **High Availability**: If a gateway goes down, API instances will fail requests to that gateway. Consider implementing gateway health checks and failover
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

3. **Verify Gateway Connection to API**:

```bash
# Check gateway logs for connection status
docker compose logs vps-gateway | grep -i "connected\|registered\|error\|api"

# Should see messages like:
# "[APIClient] Connecting to API at http://api:3001"
# "[APIClient] Successfully registered with API"
```

4. **Check API Logs for Gateway Registration**:

```bash
# Check API logs for gateway registration
docker compose logs api | grep -i "gateway.*registered"

# Should see:
# "[GatewayRegistry] Gateway <gateway-id> registered"
```

5. **Check Metrics via API**:

```bash
# Gateway metrics are forwarded to API's /metrics endpoint
curl http://api:3001/metrics | grep -i "vps_gateway"

# Should see gateway metrics with "# Gateway: <gateway-id>" comments
```

### Check DHCP Service

1. **Check dnsmasq Status**:

   ```bash
   # On gateway container
   ps aux | grep dnsmasq
   # Should show dnsmasq process

   # Check dnsmasq config
   cat /var/lib/obiente/vps-gateway/dnsmasq.conf
   # Should show DHCP pool configuration
   ```

2. **Check DHCP Leases**:

   ```bash
   # On gateway container
   cat /var/lib/obiente/vps-gateway/dnsmasq.leases
   # Should show allocated IPs (empty initially)

   # Or from the host (if using bind mount)
   cat /var/lib/obiente/vps-gateway/dnsmasq.leases
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
   - **"GATEWAY_API_URL not set"**: Ensure `GATEWAY_API_URL` points to your API service
   - **"Failed to connect to API"**: Check that API is accessible from gateway container
   - **"Failed to initialize DHCP manager"**: Check network interface name (`GATEWAY_DHCP_INTERFACE`)
   - **"Permission denied"**: Ensure container has `privileged: true` or `network_mode: host`

### DHCP Not Working

1. **Check dnsmasq Process**:

   ```bash
   ps aux | grep dnsmasq
   ```

2. **Check Interface**:

   ```bash
   ip addr show eth1
   # Should show 10.15.3.1/24
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

### Gateway Can't Connect to API

**Note**: In Swarm deployments, the gateway connects to the API service name (e.g., `http://api:3001`), which is load-balanced by Docker Swarm DNS. The gateway will connect to one API instance and maintain that connection.

1. **Test Connectivity**:

   ```bash
   # From gateway container
   curl http://api:3001/health
   # Or test from gateway host
   curl http://your-api-host:3001/health
   ```

2. **Check API URL**:

   - Ensure `GATEWAY_API_URL` points to the correct API hostname/IP
   - In Swarm, use the service name: `http://api:3001` (resolves via Swarm DNS)
   - Ensure API port (default: 3001) is accessible from gateway
   - Check gateway logs: `docker compose logs vps-gateway` or `docker logs vps-gateway`

3. **Check API Secret**:
   - Ensure `GATEWAY_API_SECRET` in gateway matches `VPS_GATEWAY_API_SECRET` in API service
   - Both must be identical

4. **Check Gateway Registration** (Swarm):

   In Swarm, check logs from all API instances:
   
   ```bash
   # Check all API instances for gateway connection
   docker service logs obiente_api | grep -i "gateway\|register"
   
   # Check specific API instance
   docker service ps obiente_api --format "{{.Name}}" | head -1 | xargs docker service logs
   ```
   
   For Docker Compose:
   
   ```bash
   docker compose logs api | grep -i "gateway\|register"
   ```
   
   - Gateway should automatically register with API on startup
   - Check API logs for "Gateway registered" messages
   - If gateway disconnects, it will automatically reconnect to another instance

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
  - eth0 (vmbr0): 10.15.3.10
  - eth1 (SDN VNet bridge): 10.15.3.1

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

Gateway Container (on Node 1):
  - eth0 (vmbr0): 10.15.3.10
  - eth1 (SDN VNet): 10.15.3.1

VPS VMs:
  - Can be created on any node
  - All connected to same SDN VNet
  - All receive IPs from gateway DHCP pool
  - Can communicate across nodes via SDN
```

**Note**: With SDN, SNAT is handled automatically when enabled in the subnet configuration. You don't need to manually configure NAT on the gateway container unless you need custom routing rules.

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
- [Environment Variables Reference](../reference/environment-variables.md) - Complete variable reference
- [Monitoring Guide](monitoring.md) - Prometheus and Grafana setup
- [Proxmox SDN Documentation](https://pve.proxmox.com/wiki/Software_Defined_Networking) - Official Proxmox SDN guide
