# Public IP Provisioning with Automatic Security

## Overview

The VPS gateway service automatically handles public IP provisioning with comprehensive security measures. When a superadmin assigns a public IP to a VPS, the system:

1. **Allocates the IP** via the gateway service
2. **Applies security measures** automatically:
   - Firewall rules to prevent IP hijacking
   - Static ARP entries to prevent ARP spoofing
   - MAC address validation
3. **Configures routing** automatically (as long as IP goes to vmbr0/uplink)
4. **Updates VPS configuration** via cloud-init

## Architecture

### Components

1. **Gateway Service** (`vps-gateway`):
   - `AllocatePublicIP`: Allocates public IP with security measures
   - `ReleasePublicIP`: Releases public IP and removes security measures
   - Security Manager: Manages firewall rules and ARP entries

2. **Superadmin Service**:
   - `AssignVPSPublicIP`: Assigns public IP and triggers gateway provisioning
   - Retrieves VPS MAC address from Proxmox
   - Calls gateway service for allocation and security

3. **Security Manager**:
   - Firewall rules: Only allow traffic from correct MAC address
   - ARP entries: Static ARP to prevent spoofing
   - Route validation: Ensures IP is routable

## Security Measures

### 1. Firewall Rules

When a public IP is allocated, the gateway automatically adds iptables rules:

```bash
# Allow traffic from correct MAC
iptables -A FORWARD -s <public-ip> -m mac --mac-source <vps-mac> -j ACCEPT

# Block traffic from other MACs (prevent hijacking)
iptables -A FORWARD -s <public-ip> ! -m mac --mac-source <vps-mac> -j DROP
```

### 2. Static ARP Entries

Static ARP entries prevent ARP spoofing:

```bash
arp -s <public-ip> <vps-mac>
```

### 3. MAC Address Validation

The gateway validates that only the allocated MAC address can use the public IP.

## Usage

### Superadmin Workflow

1. **Create Public IP** (via superadmin UI/API):
   ```protobuf
   CreateVPSPublicIP {
     ip_address: "203.0.113.10"
     gateway: "203.0.113.1"  // Optional, auto-calculated if not provided
     netmask: "24"            // Optional, defaults to /24
     monthly_cost_cents: 500
   }
   ```

2. **Assign Public IP** (via superadmin UI/API):
   ```protobuf
   AssignVPSPublicIP {
     ip_id: "ip-123456789"
     vps_id: "vps-123456789"
   }
   ```

3. **System Automatically**:
   - Retrieves VPS MAC address from Proxmox
   - Calls gateway `AllocatePublicIP` with security measures
   - Gateway applies firewall rules and ARP entries
   - Updates VPS cloud-init configuration
   - Configures IP on running VPS (if VM is running)

### Gateway API

#### AllocatePublicIP

```protobuf
AllocatePublicIPRequest {
  vps_id: "vps-123456789"
  organization_id: "org-123456789"
  mac_address: "00:16:3e:aa:bb:cc"  // Required for security
  public_ip: "203.0.113.10"
  gateway: "203.0.113.1"             // Optional
  netmask: "24"                      // Optional
}
```

#### ReleasePublicIP

```protobuf
ReleasePublicIPRequest {
  vps_id: "vps-123456789"
  public_ip: "203.0.113.10"
  mac_address: "00:16:3e:aa:bb:cc"  // Required for cleanup
}
```

## Configuration

### Gateway Environment Variables

- `GATEWAY_UPLINK_INTERFACE`: Uplink interface (e.g., `vmbr0`). Auto-detected if not set.
- `GATEWAY_API_SECRET`: API secret for authentication (required)

### Network Requirements

- Public IPs must be routable on the uplink interface (vmbr0)
- Gateway container must have `network_mode: host` and `privileged: true`
- Gateway must have access to iptables and arp commands

## Troubleshooting

### IP Not Working

1. **Check firewall rules**:
   ```bash
   iptables -L FORWARD -n -v | grep <public-ip>
   ```

2. **Check ARP entry**:
   ```bash
   arp -n <public-ip>
   ```

3. **Check gateway logs**:
   ```bash
   docker logs vps-gateway
   ```

### Security Rules Not Applied

1. **Verify gateway has privileges**:
   - Container must run with `privileged: true`
   - Container must have `network_mode: host`

2. **Check gateway logs** for errors:
   ```bash
   docker logs vps-gateway | grep -i "security\|firewall\|arp"
   ```

### MAC Address Issues

1. **Get MAC from Proxmox**:
   ```bash
   # Via Proxmox API or CLI
   qm config <vmid> | grep net0
   ```

2. **Verify MAC matches**:
   - Check gateway allocation logs
   - Verify firewall rules use correct MAC

## Best Practices

1. **Always use gateway allocation**: Don't manually configure public IPs
2. **Monitor security rules**: Regularly verify firewall and ARP entries
3. **Document IP assignments**: Keep track of which IPs are assigned to which VPSs
4. **Test before production**: Verify security measures work in test environment

## Related Documentation

- [VPS IP Security](../reference/vps-ip-security.md) - Automatic IP hijacking prevention
- [VPS Gateway Setup](vps-gateway-setup.md) - Gateway service configuration

## Future Enhancements

- Automatic route announcement (BGP)
- RPKI validation for public IPs
- Network monitoring and alerting
- Automatic IP conflict detection

