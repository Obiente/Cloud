# Multi-Node Gateway IP Configuration

## Overview

For multi-node deployments, each Proxmox node needs its own gateway service instance. Since all gateways are on the same VXLAN (10.15.3.0/24), each gateway needs a **unique IP address** on that VXLAN.

## Network Architecture

```
VXLAN Network: 10.15.3.0/24
├── VXLAN Gateway/Router: 10.15.3.1 (managed by Proxmox SDN)
├── Node 1 Gateway: 10.15.3.10 (host IP on VXLAN)
├── Node 2 Gateway: 10.15.3.11 (host IP on VXLAN)
├── Node 3 Gateway: 10.15.3.12 (host IP on VXLAN)
└── VPS IP Pool: 10.15.3.20-254 (allocated by gateways)
```

## Important Distinctions

1. **VXLAN Gateway (10.15.3.1)**: This is the network gateway/router managed by Proxmox SDN. All VPSs use this as their default gateway.
2. **Gateway Service IPs (10.15.3.10, 10.15.3.11, etc.)**: These are the IP addresses of each gateway service instance on the VXLAN. Each node's gateway needs a unique IP.

## Configuration for Docker with `network_mode: host`

When using Docker with `network_mode: host`, the container uses the host's network namespace. This means:

- The **host** must have an IP address on the VXLAN interface
- The container (using host networking) can bind to that IP
- Each node's host needs a different IP on the VXLAN

### Step 1: Configure Host IP on VXLAN

On each Proxmox node, assign a unique IP to the VXLAN interface:

**Node 1:**
```bash
# On the Proxmox HOST (not inside container):
# Find the SDN bridge (created by Proxmox SDN)
ip addr show | grep -E "vnet|sdn|OCvpsnet"

# Assign IP to the SDN bridge on the host (OCvpsnet is the default SDN bridge name)
sudo ip addr add 10.15.3.10/24 dev OCvpsnet

# Note: Inside the container/VM, this bridge appears as eth0 (or eth1)
# The container's eth0 interface is connected to the host's OCvpsnet bridge
# Or configure via Proxmox network configuration
```

**Node 2:**
```bash
# On Node 2 host:
sudo ip addr add 10.15.3.11/24 dev OCvpsnet
```

**Node 3:**
```bash
# On Node 3 host:
sudo ip addr add 10.15.3.12/24 dev OCvpsnet
```

### Step 2: Configure Gateway Service

The gateway service needs to know which IP to listen on. Currently, the service listens on the VXLAN gateway IP (10.15.3.1) via the `GATEWAY_DHCP_GATEWAY` setting, but it should listen on the host's IP instead.

**Option A: Use Host IP for DHCP Listening (Recommended)**

Add a new environment variable `GATEWAY_DHCP_LISTEN_IP` to specify the IP the gateway should listen on:

```yaml
# Node 1
GATEWAY_DHCP_LISTEN_IP: 10.15.3.10  # Host's IP on VXLAN (assigned to OCvpsnet bridge on host)
GATEWAY_DHCP_GATEWAY: 10.15.3.1      # VXLAN gateway (for VPSs)
GATEWAY_DHCP_INTERFACE: eth0         # Interface inside container (connected to OCvpsnet bridge on host)
```

**Important**: The `GATEWAY_DHCP_INTERFACE` is the interface name **inside the container/VM** (typically `eth0`), not the bridge name on the host (`OCvpsnet`). The container's `eth0` interface is connected to the host's `OCvpsnet` bridge.

**Option B: Use Interface-Based Binding**

The gateway can bind to the interface inside the container (e.g., `eth0`) directly, and the host's IP on the OCvpsnet bridge will be used automatically. The container's `eth0` interface is connected to the host's `OCvpsnet` bridge.

## Recommended IP Allocation

Reserve specific IP ranges for infrastructure:

- **10.15.3.1**: VXLAN Gateway/Router (Proxmox SDN)
- **10.15.3.2-9**: Reserved for future infrastructure
- **10.15.3.10-19**: Gateway service IPs (one per node)
  - 10.15.3.10: Node 1 Gateway
  - 10.15.3.11: Node 2 Gateway
  - 10.15.3.12: Node 3 Gateway
  - etc.
- **10.15.3.20-254**: VPS IP pool (allocated by gateways)

## Current Issue

The current implementation uses `GATEWAY_DHCP_GATEWAY` (10.15.3.1) as the listen address for dnsmasq. This means:

1. All gateways would try to listen on 10.15.3.1 (conflict!)
2. Only one gateway can actually bind to that IP
3. Other gateways would fail to start DHCP service

## Solution

The gateway service should:
1. Listen on the **host's IP** on the VXLAN (e.g., 10.15.3.10, 10.15.3.11)
2. Configure VPSs to use **10.15.3.1** as their gateway (via DHCP options)

This requires either:
- Adding `GATEWAY_DHCP_LISTEN_IP` environment variable, OR
- Auto-detecting the host's IP on the VXLAN interface

## Example Configuration per Node

**Node 1 docker-compose.vps-gateway.yml:**
```yaml
services:
  vps-gateway:
    network_mode: host
    privileged: true
    environment:
      GATEWAY_DHCP_LISTEN_IP: 10.15.3.10  # Host IP on VXLAN
      GATEWAY_DHCP_GATEWAY: 10.15.3.1     # VXLAN gateway for VPSs
      GATEWAY_DHCP_INTERFACE: vmbr100     # VXLAN interface
      GATEWAY_DHCP_POOL_START: 10.15.3.20
      GATEWAY_DHCP_POOL_END: 10.15.3.254
      # ... other config
```

**Node 2 docker-compose.vps-gateway.yml:**
```yaml
services:
  vps-gateway:
    network_mode: host
    privileged: true
    environment:
      GATEWAY_DHCP_LISTEN_IP: 10.15.3.11  # Different IP for Node 2
      GATEWAY_DHCP_GATEWAY: 10.15.3.1      # Same VXLAN gateway
      GATEWAY_DHCP_INTERFACE: vmbr100      # Same VXLAN interface
      # ... same pool config
```

## Verification

After configuration, verify each gateway is listening on its own IP:

```bash
# On Node 1
ss -tulpn | grep :67
# Should show dnsmasq listening on 10.15.3.10:67

# On Node 2
ss -tulpn | grep :67
# Should show dnsmasq listening on 10.15.3.11:67
```

