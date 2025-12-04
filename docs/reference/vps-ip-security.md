# VPS IP Security

## Overview

The VPS gateway service automatically implements security measures to prevent IP address hijacking. When IP addresses are allocated to VPS instances, the gateway automatically:

- **Binds IPs to MAC addresses** - Only the assigned VPS can use its allocated IP
- **Applies firewall rules** - Prevents unauthorized IP usage automatically
- **Configures ARP protection** - Prevents ARP spoofing attacks
- **Validates IP ownership** - Ensures only authorized VPSs can use their IPs

**No manual configuration is required** - all security measures are applied automatically by the gateway service.

## How It Works

### Automatic Security Measures

When a VPS is created or a public IP is assigned:

1. **IP Allocation**: The gateway allocates an IP address and records the VPS's MAC address
2. **Firewall Rules**: Automatic iptables rules are added to only allow traffic from the correct MAC address
3. **ARP Protection**: Static ARP entries are created to prevent ARP spoofing
4. **Ongoing Validation**: The gateway continuously validates that IPs are only used by their assigned VPSs

### For DHCP Pool IPs

- IPs are allocated via DHCP with MAC address binding
- dnsmasq automatically enforces MAC-to-IP bindings
- Only the assigned VPS can receive its allocated IP via DHCP

### For Public IPs

- IPs are allocated with explicit MAC address binding
- Firewall rules automatically prevent IP hijacking
- Static ARP entries prevent ARP spoofing
- All security measures are applied automatically when the IP is assigned

## Configuration

### Gateway Requirements

The gateway service must run with the following privileges to apply security measures:

- `network_mode: host` - Required for network access
- `privileged: true` - Required for iptables and ARP management

These are already configured in the default `docker-compose.vps-gateway.yml`.

### Environment Variables

- `GATEWAY_UPLINK_INTERFACE`: Optional - Uplink interface (e.g., `vmbr0`). Auto-detected if not set.
- `GATEWAY_API_SECRET`: Required - API secret for authentication

## Troubleshooting

### IP Not Working

If a VPS cannot use its allocated IP:

1. **Check gateway logs**:
   ```bash
   docker logs vps-gateway | grep -i "security\|firewall\|arp"
   ```

2. **Verify gateway is running**:
   ```bash
   docker ps | grep vps-gateway
   ```

3. **Check gateway privileges**:
   - Ensure container has `privileged: true`
   - Ensure container has `network_mode: host`

### Security Rules Not Applied

If security measures aren't working:

1. **Verify gateway has privileges**:
   - Container must run with `privileged: true`
   - Container must have `network_mode: host`

2. **Check gateway logs for errors**:
   ```bash
   docker logs vps-gateway | grep -i error
   ```

3. **Verify iptables access**:
   ```bash
   docker exec vps-gateway iptables -L FORWARD -n -v
   ```

### IP Conflicts

If you suspect an IP conflict:

1. **Check gateway allocations**:
   - Use the gateway API to list allocated IPs
   - Verify each IP is only allocated to one VPS

2. **Review gateway logs**:
   ```bash
   docker logs vps-gateway | grep -i "allocated\|conflict"
   ```

## Best Practices

1. **Use gateway allocation**: Always allocate IPs through the gateway service - don't manually configure IPs
2. **Monitor gateway logs**: Regularly check gateway logs for security-related warnings
3. **Keep gateway updated**: Ensure the gateway service is running the latest version
4. **Verify container privileges**: Ensure the gateway container has required privileges

## Technical Details

### Firewall Rules

The gateway automatically adds iptables rules when IPs are allocated:

- **ACCEPT rule**: Allows traffic from the correct MAC address using the allocated IP
- **DROP rule**: Blocks traffic from other MAC addresses using the allocated IP

These rules are automatically removed when IPs are released.

### ARP Protection

Static ARP entries are automatically created to prevent ARP spoofing:

- Each allocated IP has a static ARP entry mapping to the VPS's MAC address
- This prevents other VPSs from claiming the IP via ARP

### MAC Address Binding

- DHCP pool IPs: Bound via dnsmasq hosts file
- Public IPs: Bound via firewall rules and ARP entries

All bindings are automatically managed by the gateway service.

## Security Guarantees

The gateway service provides the following security guarantees:

1. **IP Ownership**: Only the assigned VPS can use its allocated IP
2. **MAC Validation**: Traffic is validated against the assigned MAC address
3. **ARP Protection**: ARP spoofing is prevented via static entries
4. **Automatic Cleanup**: Security rules are automatically removed when IPs are released

These measures work together to prevent IP hijacking, MAC spoofing, and ARP spoofing attacks.
