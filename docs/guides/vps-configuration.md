# VPS Configuration Guide

Advanced configuration options for VPS instances on Obiente Cloud.

## Proxmox Configuration

### API Authentication

#### Password Authentication

```bash
PROXMOX_API_URL=https://proxmox.example.com:8006
PROXMOX_USERNAME=root@pam
PROXMOX_PASSWORD=your-secure-password
```

#### Token Authentication (Recommended)

```bash
PROXMOX_API_URL=https://proxmox.example.com:8006
PROXMOX_USERNAME=root@pam
PROXMOX_TOKEN_ID=obiente-cloud
PROXMOX_TOKEN_SECRET=your-token-secret
```

**Important:** The API token must have the following permissions:

- **VM.Allocate** - Create new VMs
- **VM.Clone** - Clone VM templates (if using templates)
- **VM.Config.Disk** - Configure VM disk storage
- **VM.Config.Network** - Configure VM network settings
- **VM.Config.Options** - Configure VM options (cloud-init, etc.)
- **VM.Config.CPU** - Configure VM CPU settings
- **VM.Config.Memory** - Configure VM memory settings
- **VM.PowerMgmt** - Start, stop, reboot VMs
- **VM.Monitor** - Monitor VM status and metrics
- **Datastore.Allocate** - Allocate storage for VMs
- **Datastore.AllocateSpace** - Allocate disk space
- **Datastore.AllocateTemplate** - Upload cloud-init snippets and templates (required for user management and cloud-init configuration)

See the [VPS Provisioning Guide](./vps-provisioning.md#3-configure-api-token-permissions) for detailed permission setup instructions.

### Storage Configuration

```bash
# Default storage pool
PROXMOX_STORAGE_POOL=local-lvm

# Alternative storage pools
PROXMOX_STORAGE_POOL=local-zfs    # ZFS storage
PROXMOX_STORAGE_POOL=ceph-pool    # Ceph distributed storage
PROXMOX_STORAGE_POOL=nfs-storage  # NFS storage
```

**Important:** The storage pool specified in `PROXMOX_STORAGE_POOL` must exist in your Proxmox installation and support VM disk images. Common storage pools include:

- `local-lvm` - Default LVM thin pool (most common)
- `local` - Local directory storage
- `local-zfs` - ZFS storage pool (if ZFS is configured)
- Custom storage pools you've created

**Template-Based VMs (Linked Clones):**

⚠️ **Important Limitation:** For VMs created from templates (linked clones), `PROXMOX_STORAGE_POOL` does **not** affect the cloned disk. Linked clones inherit the template's storage pool and cannot be changed during cloning. The `PROXMOX_STORAGE_POOL` setting only applies to:

1. **VMs created without templates** (ISO installation) - uses the specified storage pool
2. **New disks created when templates have no disk** - uses the specified storage pool
3. **Additional disks** beyond what's cloned from the template - uses the specified storage pool

**To use a different storage pool for template-based VMs:**
- Create your template on the desired storage pool before cloning
- The cloned VMs will inherit the template's storage pool

**Checking Available Storage Pools:**

To see available storage pools in your Proxmox installation:

1. **Via Proxmox Web UI:**
   - Go to **Datacenter** → **Storage**
   - Look for storages with "Disk image" content type

2. **Via Proxmox CLI:**
   ```bash
   # List all storage pools
   pvesm status
   
   # List storage pools on a specific node
   pvesm list <node-name>
   ```

3. **Via Proxmox API:**
   ```bash
   curl -H "Authorization: PVEAPIToken=USER@REALM!TOKENID=SECRET" \
     https://your-proxmox:8006/api2/json/nodes/<node-name>/storage
   ```

If you get an error that a storage pool doesn't exist, check the available pools using one of the methods above and update `PROXMOX_STORAGE_POOL` accordingly.

### Network Configuration

#### Default Configuration

By default, VPS instances use the `vmbr0` bridge with Proxmox firewall enabled (`firewall=1`). This provides basic security but VMs are on the same Layer 2 network segment.

**⚠️ Important: Inter-VM Communication**

**By default, VMs are configured to block inter-VM communication** for security. However, organizations can enable inter-VM communication if they need VMs to communicate with each other.

**Default Behavior (Inter-VM Blocked):**
- ❌ VMs cannot ping each other
- ❌ VMs cannot access services on other VMs
- ✅ VMs can access the internet
- ✅ VMs are isolated from devices on different VLANs (if VLAN is configured)
- ✅ Each organization can enable inter-VM communication for their VMs

**Enabling Inter-VM Communication:**

Organizations can enable inter-VM communication by setting `allow_inter_vm_communication=true` in the database. When enabled, VMs in that organization can communicate with each other.

**Note:** Firewall rules are automatically configured when VMs are created, but for production deployments, you may need to fine-tune firewall rules manually in Proxmox for optimal security.

#### VLAN Isolation (Recommended)

For better security and network isolation, you can configure VMs to use a VLAN tag. This provides:

- **Layer 2 Isolation**: VMs are isolated from other devices on the network (but NOT from each other)
- **IP Spoofing Prevention**: VLANs help prevent unauthorized IP usage between VLANs
- **Network Segmentation**: Separate VMs from management and other services
- **Firewall Integration**: Works with Proxmox firewall rules

**⚠️ Note:** VLANs isolate VMs from devices on different VLANs, but **VMs on the same VLAN can still access each other** unless firewall rules prevent it.

**Configuration:**

```bash
# Set VLAN ID in environment variables
PROXMOX_VLAN_ID=100
```

When `PROXMOX_VLAN_ID` is set, all VMs will be configured with the VLAN tag:
```
virtio,bridge=vmbr0,tag=100,firewall=1
```

**Proxmox Configuration:**

The VLAN tag works through `vmbr0` - no separate bridge is needed. However, you need to ensure:

1. **Physical Switch/Router Support**: Your network infrastructure must support VLAN tagging (802.1Q)
2. **Proxmox Bridge Configuration**: `vmbr0` must be configured to handle VLAN tags (usually automatic)
3. **Router/Firewall Rules**: Configure your router/firewall to route traffic for the VLAN

**Proxmox Bridge Setup:**

If you want to use a separate bridge that works through `vmbr0`, you can create `vmbr1` and add `vmbr0` as a port:

```bash
# Edit /etc/network/interfaces on Proxmox host
auto vmbr1
iface vmbr1 inet manual
    bridge-ports vmbr0
    bridge-stp off
    bridge-fd 0
```

Then configure VMs to use `vmbr1` with VLAN tags. However, using VLAN tags directly on `vmbr0` is simpler and more common.

**Security Benefits:**

- Prevents IP spoofing between VLANs
- Isolates VM traffic from management network
- Allows firewall rules per VLAN
- Enables network monitoring and logging per VLAN

**Alternative: Separate Bridge**

If you prefer a completely separate bridge instead of VLANs:

1. Create a new bridge in Proxmox (e.g., `vmbr1`)
2. Configure it with appropriate network settings
3. Update the VM network configuration in the code to use the new bridge

#### Inter-VM Communication Control

**Automatic Configuration:**

By default, Obiente Cloud automatically configures firewall rules to block inter-VM communication when VMs are created. This is controlled by the `allow_inter_vm_communication` setting on the organization.

**Organization Setting:**

To enable inter-VM communication for an organization's VMs, update the organization record:

```sql
UPDATE organizations 
SET allow_inter_vm_communication = true 
WHERE id = 'org-xxx';
```

When `allow_inter_vm_communication` is:
- `false` (default): VMs cannot communicate with each other
- `true`: VMs in the same organization can communicate with each other

**Manual Firewall Configuration:**

For advanced firewall configuration, you can manually configure Proxmox firewall rules:

**Via Proxmox Web UI:**

1. Go to **Datacenter** → **Firewall** → **Options**
2. Enable **Firewall** if not already enabled
3. Go to **Firewall** → **Security Groups** (or **Firewall** → **Rules**)
4. Create a rule to block inter-VM communication:

**Option 1: Block all inter-VM traffic (most restrictive)**

Create a firewall rule that blocks traffic between VMs:
- **Action**: `REJECT` or `DROP`
- **Source**: VM network range (e.g., `10.0.100.0/24`)
- **Destination**: VM network range (e.g., `10.0.100.0/24`)
- **Interface**: `vmbr0` (or your VLAN interface)

**Option 2: Allow only specific inter-VM communication**

Create allow rules for specific services, then add a default deny rule:
- Allow rules for specific ports/services between VMs
- Default deny rule for all other inter-VM traffic

**Via Proxmox CLI:**

```bash
# Block all inter-VM traffic on vmbr0
pvesh create /nodes/{node}/firewall/rules \
  --action REJECT \
  --source 10.0.100.0/24 \
  --dest 10.0.100.0/24 \
  --iface vmbr0 \
  --comment "Block inter-VM communication"

# Or use iptables directly (not recommended, use Proxmox firewall)
iptables -A FORWARD -i vmbr0 -o vmbr0 -j REJECT
```

**Per-VM Firewall Rules:**

You can also configure firewall rules per VM:

1. Go to **VM** → **Firewall** → **Options**
2. Enable **Firewall** for the VM
3. Add rules to block/allow specific traffic

**Security Groups:**

For more advanced isolation, use Proxmox Security Groups:

1. Go to **Datacenter** → **Firewall** → **Security Groups**
2. Create a security group (e.g., "isolated-vms")
3. Add rules to the security group
4. Assign VMs to the security group

**Example: Complete VM Isolation**

To completely isolate VMs from each other while allowing internet access:

```bash
# Allow outbound internet traffic
# Allow inbound traffic from internet (if needed)
# Block all inter-VM traffic
```

**Testing Inter-VM Communication:**

To test if VMs can access each other:

```bash
# From VM1, ping VM2
ping <VM2_IP>

# From VM1, try to connect to a service on VM2
curl http://<VM2_IP>:<port>
```

If you want VMs to be isolated, configure firewall rules as described above.

## SSH Proxy Configuration

### Host Key

The SSH proxy generates a host key automatically. To use a custom key:

```bash
SSH_PROXY_HOST_KEY_PATH=/path/to/ssh_host_rsa_key
```

### Port Configuration

```bash
SSH_PROXY_PORT=2222  # Default SSH proxy port
```

### Firewall Rules

Allow SSH proxy port in your firewall:

```bash
# UFW
sudo ufw allow 2222/tcp

# iptables
sudo iptables -A INPUT -p tcp --dport 2222 -j ACCEPT
```

## VPS Size Catalog

### Default Sizes

The system includes default VPS sizes. To customize:

1. Connect to database
2. Update `vps_size_catalog` table
3. Add/modify sizes as needed

### Custom Sizes

```sql
INSERT INTO vps_size_catalog (
  id, name, description,
  cpu_cores, memory_bytes, disk_bytes,
  bandwidth_bytes_month, price_cents_per_month,
  available, region
) VALUES (
  'custom-1',
  'Custom Size 1',
  'Custom configuration',
  4,  # CPU cores
  8589934592,  # 8 GB RAM
  107374182400,  # 100 GB disk
  0,  # Unlimited bandwidth
  3000,  # $30/month
  true,
  ''  # Available in all regions
);
```

## Region Configuration

### Adding Regions

```sql
INSERT INTO vps_region_catalog (
  id, name, location, country,
  features, available
) VALUES (
  'us-west-2',
  'US West (Oregon)',
  'Portland, Oregon, USA',
  'US',
  '["nvme_storage", "low_latency"]',
  true
);
```

## Cloud-Init Configuration

VPS instances use cloud-init for initial configuration. Customize via:

1. **SSH Keys**: Add via `ssh_key_id` parameter
2. **User Data**: Configured automatically
3. **Network**: DHCP by default

### Custom Cloud-Init

To add custom cloud-init configuration, modify the `generateCloudInitUserData` function in `proxmox_client.go`.

## Monitoring Configuration

### Metrics Collection

VPS metrics are collected via:
- **QEMU Guest Agent**: For resource usage
- **Proxmox API**: For VM status
- **TimescaleDB**: For historical metrics

### Metrics Retention

Configure retention in TimescaleDB:

```sql
-- Set retention policy for VPS metrics
SELECT add_retention_policy('vps_metrics', INTERVAL '30 days');
SELECT add_retention_policy('vps_usage_hourly', INTERVAL '90 days');
```

## Security Configuration

### SSH Key Management

SSH keys should be stored securely. Consider:

1. **Database Storage**: Store encrypted SSH keys
2. **Key Rotation**: Implement key rotation policy
3. **Access Control**: Limit who can add SSH keys

### Network Security

**Inter-VM Communication Control:**

**By default, inter-VM communication is blocked** for security. This means:
- VMs cannot ping each other
- VMs cannot access services on other VMs
- VMs can still access the internet
- Each organization can enable inter-VM communication if needed

**Enabling Inter-VM Communication:**

Organizations can enable inter-VM communication by setting `allow_inter_vm_communication=true` in the database. See [Inter-VM Communication Control](#inter-vm-communication-control) for details.

**VLAN Isolation (Recommended):**

Configure VLAN tags via `PROXMOX_VLAN_ID` environment variable to isolate VMs at Layer 2. This prevents:
- IP spoofing between VLANs
- Unauthorized network access from other network segments
- ARP spoofing attacks from other VLANs
- Network scanning from VMs to other network segments

**⚠️ Important:** VLANs isolate VMs from devices on different VLANs, but **VMs on the same VLAN can still access each other** unless firewall rules prevent it.

See [Network Configuration](#network-configuration) for detailed VLAN setup instructions.

**Additional Security Measures:**

- **Configure Proxmox firewall rules** to block inter-VM communication (required for isolation)
- Use firewall rules to restrict outbound access if needed
- Implement network policies and access controls
- Monitor network traffic per VLAN
- Use MAC address filtering for additional security
- Consider using Proxmox Security Groups for advanced isolation

## Performance Tuning

### CPU Pinning

For high-performance VPS instances, consider CPU pinning:

1. Configure in Proxmox VM settings
2. Pin to specific CPU cores
3. Reserve cores for host system

### Memory Ballooning

Enable memory ballooning for better resource utilization:

```bash
# Configure in Proxmox
qm set {vmid} --balloon 1024  # Enable ballooning with 1GB
```

### Storage Optimization

- Use SSD storage for better I/O
- Configure storage cache mode
- Use thin provisioning for disk images

## Backup Configuration

### Proxmox Backups

Configure Proxmox backup storage:

1. Set up backup storage in Proxmox
2. Schedule automatic backups
3. Configure retention policies

### Application-Level Backups

For application data:
- Use backup tools within VPS
- Schedule regular backups
- Store backups externally

## High Availability

### Proxmox Cluster

For HA VPS provisioning:

1. Set up Proxmox cluster
2. Configure shared storage
3. Enable VM migration
4. Configure quorum

### Load Balancing

Distribute VPS instances across nodes:

1. Configure node selection logic
2. Monitor node resources
3. Implement load balancing

## Troubleshooting Configuration

### Enable Debug Logging

```bash
# API service
LOG_LEVEL=debug

# Proxmox client
PROXMOX_DEBUG=true
```

### API Timeout Configuration

```bash
# Increase timeout for large VPS operations
PROXMOX_API_TIMEOUT=300  # 5 minutes
```

## Related Documentation

- [VPS Provisioning Guide](vps-provisioning.md) - Getting started with VPS
- [Environment Variables](../reference/environment-variables.md) - Complete variable reference
- [Troubleshooting Guide](troubleshooting.md) - Common issues

---

[← Back to Guides](index.md)

