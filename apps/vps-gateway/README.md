# VPS Gateway Service

The VPS Gateway service provides DHCP management and SSH proxying for VPS instances in the Obiente Cloud platform.

## Features

- **DHCP Management**: Allocates and manages IP addresses for VPS instances using dnsmasq
- **SSH Proxy**: Proxies SSH connections to VPS instances via bidirectional gRPC streams
- **Prometheus Metrics**: Exposes metrics for monitoring DHCP and SSH proxy operations
- **gRPC API**: Provides a gRPC API for IP allocation, release, and SSH proxying

## Prerequisites

- Go 1.25+
- dnsmasq installed on the host system
- Network interface configured for DHCP management
- Docker and Docker Compose (for containerized deployment)

## Configuration

The service is configured via environment variables:

### Required Variables

- `GATEWAY_API_SECRET`: Shared secret for authenticating API requests
- `GATEWAY_DHCP_POOL_START`: Starting IP address of the DHCP pool (e.g., `192.168.100.10`)
- `GATEWAY_DHCP_POOL_END`: Ending IP address of the DHCP pool (e.g., `192.168.100.254`)
- `GATEWAY_DHCP_SUBNET`: Subnet mask (e.g., `255.255.255.0`)
- `GATEWAY_DHCP_GATEWAY`: Gateway IP address (e.g., `192.168.100.1`)
- `GATEWAY_DHCP_INTERFACE`: Network interface name for dnsmasq (e.g., `eth0`)

### Optional Variables

- `GATEWAY_GRPC_PORT`: gRPC server port (defaults to `1537` - OCG - Obiente Cloud Gateway)
- `GATEWAY_DHCP_DNS`: Comma-separated list of DNS servers (defaults to gateway IP)
- `GATEWAY_DHCP_LEASES_DIR`: Directory for storing DHCP lease files (defaults to `/var/lib/vps-gateway`)
- `GATEWAY_PUBLIC_IP`: Public IP for DNAT configuration (optional, for documentation)
- `LOG_LEVEL`: Logging level (`debug`, `info`, `warn`, `error`) - defaults to `info`

## Building

```bash
cd apps/vps-gateway
go build -o vps-gateway ./main.go
```

## Running Locally

1. Ensure dnsmasq is installed:
```bash
# On Debian/Ubuntu
sudo apt-get install dnsmasq

# On Alpine
apk add dnsmasq
```

2. Set environment variables (see Configuration section)

3. Run the service:
```bash
./vps-gateway
```

## Running with Docker Compose

```bash
cd apps/vps-gateway
docker-compose -f docker-compose.test.yml up --build
```

**Note**: The service requires `network_mode: host` and `privileged: true` to manage network interfaces and run dnsmasq. This is only suitable for testing. For production, you'll need to configure the network differently.

## Testing

### 1. Check Service Health

The service exposes a Prometheus metrics endpoint:

```bash
curl http://localhost:9091/metrics
```

### 2. Test gRPC API with grpcurl

Install grpcurl:
```bash
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

List available services:
```bash
grpcurl -plaintext localhost:1537 list
```

Get gateway info:
```bash
grpcurl -plaintext \
  -H "x-api-secret: test-secret-key-change-in-production" \
  localhost:1537 \
  obiente.cloud.vpsgateway.v1.VPSGatewayService/GetGatewayInfo
```

Allocate an IP:
```bash
grpcurl -plaintext \
  -H "x-api-secret: test-secret-key-change-in-production" \
  -d '{
    "vps_id": "vps-test-123",
    "organization_id": "org-test-456",
    "mac_address": "00:11:22:33:44:55"
  }' \
  localhost:1537 \
  obiente.cloud.vpsgateway.v1.VPSGatewayService/AllocateIP
```

List allocated IPs:
```bash
grpcurl -plaintext \
  -H "x-api-secret: test-secret-key-change-in-production" \
  localhost:1537 \
  obiente.cloud.vpsgateway.v1.VPSGatewayService/ListIPs
```

Release an IP:
```bash
grpcurl -plaintext \
  -H "x-api-secret: test-secret-key-change-in-production" \
  -d '{
    "vps_id": "vps-test-123"
  }' \
  localhost:1537 \
  obiente.cloud.vpsgateway.v1.VPSGatewayService/ReleaseIP
```

### 3. Test Prometheus Metrics

View metrics:
```bash
curl http://localhost:9091/metrics | grep vps_gateway
```

Key metrics to check:
- `vps_gateway_dhcp_allocations_total`: Total number of IP allocations
- `vps_gateway_dhcp_allocations_active`: Current active allocations
- `vps_gateway_dhcp_pool_size`: Total IP pool size
- `vps_gateway_dhcp_pool_available`: Available IPs in pool
- `vps_gateway_ssh_proxy_connections_total`: Total SSH proxy connections
- `vps_gateway_ssh_proxy_connections_active`: Active SSH proxy connections

## Development

### Generate Proto Files

Proto files are generated using buf. From the repository root:

```bash
cd packages/proto
npm run build  # or: buf generate
```

This will generate Go code in both `apps/api/gen/proto` and `apps/vps-gateway/gen/proto`.

### Running Tests

```bash
go test ./...
```

## Architecture

The service uses a **forward connection pattern** where:
- Gateway exposes a gRPC server on port **1537** (OCG - Obiente Cloud Gateway)
- API instances connect to the gateway's public IP (configured via DNAT)
- Port **1537** maps to "O 15 C 3 G" = "OCG" (Obiente Cloud Gateway), similar to how `10.15.3` maps to "O 15 C 3"

The service consists of:

- **DHCP Manager** (`internal/dhcp/`): Manages IP allocations using dnsmasq
- **SSH Proxy** (`internal/sshproxy/`): Handles SSH connection proxying
- **gRPC Server** (`internal/server/`): Implements the VPSGatewayService API (listens on port 1537)
- **Authentication** (`internal/auth/`): Validates shared secret for API requests
- **Metrics** (`internal/metrics/`): Exposes Prometheus metrics

## Troubleshooting

### dnsmasq fails to start

- Ensure the network interface exists and is up
- Check that the interface name matches `GATEWAY_DHCP_INTERFACE`
- Verify you have permissions to bind to the interface (may require root or NET_ADMIN capability)

### IP allocation fails

- Check that the IP pool range is valid and not overlapping with existing network configuration
- Verify the subnet mask matches your network configuration
- Ensure dnsmasq is running: `ps aux | grep dnsmasq`

### gRPC connection refused

- Verify the service is listening on port 1537: `netstat -tlnp | grep 1537`
- Check firewall rules (port 1537 should be accessible from API instances)
- Ensure DNAT is configured if using public IP access
- Ensure the `x-api-secret` header matches `GATEWAY_API_SECRET`

## Production Deployment

For production deployment:

1. Use a dedicated network interface or VLAN for VPS instances
2. Configure proper firewall rules
3. Use a secrets management system for `GATEWAY_API_SECRET`
4. Set up monitoring and alerting based on Prometheus metrics
5. Configure log aggregation
6. Use a process manager (systemd, supervisor, etc.) for service management

