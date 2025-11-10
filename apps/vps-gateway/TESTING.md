# Testing VPS Gateway Service

This guide explains how to test the vps-gateway service implementation.

## Prerequisites

1. **Proto files generated**: Run `cd packages/proto && npm run build` (or `buf generate`)
2. **dnsmasq installed**: Required for DHCP functionality
3. **Network interface configured**: You need a network interface for dnsmasq to bind to

## Quick Start Testing

### Option 1: Run Locally (Recommended for Initial Testing)

1. **Set environment variables**:
```bash
export GATEWAY_API_SECRET="test-secret-key-change-in-production"
export GATEWAY_DHCP_POOL_START="192.168.100.10"
export GATEWAY_DHCP_POOL_END="192.168.100.254"
export GATEWAY_DHCP_SUBNET="255.255.255.0"
export GATEWAY_DHCP_GATEWAY="192.168.100.1"
export GATEWAY_DHCP_DNS="8.8.8.8,8.8.4.4"
export GATEWAY_DHCP_INTERFACE="eth0"  # Change to your network interface
export LOG_LEVEL="debug"
```

2. **Build and run**:
```bash
cd apps/vps-gateway
go build -o vps-gateway ./main.go
sudo ./vps-gateway  # Requires root for dnsmasq
```

**Note**: Running locally requires root privileges because dnsmasq needs to bind to network interfaces. For production, use Docker or a dedicated VM.


## Testing the Service

### 1. Check Service is Running

The service exposes two ports (accessible from within the container):
- **Port 1537**: gRPC server (OCG - Obiente Cloud Gateway, configurable via `GATEWAY_GRPC_PORT`)
- **Port 9091**: Prometheus metrics (configurable via `GATEWAY_METRICS_PORT`)

**All testing is done by attaching to the container**:
```bash
docker exec -it vps-gateway-test /bin/sh
# Then inside the container:
curl http://localhost:9091/metrics
```

**If running locally**, check directly:
```bash
curl http://localhost:9091/metrics
```

You should see Prometheus metrics including:
- `vps_gateway_dhcp_pool_size`
- `vps_gateway_dhcp_allocations_active`
- `vps_gateway_ssh_proxy_connections_active`

### 2. Test gRPC API with grpcurl

**If testing in Docker**, you'll need grpcurl inside the container. Install it:
```bash
# Install grpcurl in the container
docker exec vps-gateway-test apk add --no-cache curl
# Or install Go and grpcurl
docker exec vps-gateway-test sh -c "apk add --no-cache go && go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest"
```

**List available services** (from inside container):
```bash
docker exec vps-gateway-test /root/go/bin/grpcurl -plaintext localhost:18080 list
```

**Get gateway info** (from inside container):
```bash
docker exec vps-gateway-test /root/go/bin/grpcurl -plaintext \
  -H "x-api-secret: test-secret-key-change-in-production" \
  localhost:18080 \
  obiente.cloud.vpsgateway.v1.VPSGatewayService/GetGatewayInfo
```

**Or exec into the container and run commands interactively**:
```bash
docker exec -it vps-gateway-test /bin/sh
# Then inside the container:
grpcurl -plaintext -H "x-api-secret: test-secret-key-change-in-production" localhost:1537 list
```

**Allocate an IP**:
```bash
grpcurl -plaintext \
  -H "x-api-secret: test-secret-key-change-in-production" \
  -d '{
    "vps_id": "vps-test-001",
    "organization_id": "org-test-001",
    "mac_address": "00:11:22:33:44:55"
  }' \
  localhost:1537 \
  obiente.cloud.vpsgateway.v1.VPSGatewayService/AllocateIP
```

Expected response:
```json
{
  "ipAddress": "192.168.100.10",
  "subnetMask": "255.255.255.0",
  "gateway": "192.168.100.1",
  "dnsServers": ["8.8.8.8", "8.8.4.4"],
  "leaseExpires": "2025-11-11T05:33:00Z"
}
```

**List allocated IPs**:
```bash
grpcurl -plaintext \
  -H "x-api-secret: test-secret-key-change-in-production" \
  localhost:1537 \
  obiente.cloud.vpsgateway.v1.VPSGatewayService/ListIPs
```

**Release an IP**:
```bash
grpcurl -plaintext \
  -H "x-api-secret: test-secret-key-change-in-production" \
  -d '{
    "vps_id": "vps-test-001"
  }' \
  localhost:1537 \
  obiente.cloud.vpsgateway.v1.VPSGatewayService/ReleaseIP
```

### 3. Run Automated Test Scripts

**For Docker testing** (from host):
```bash
cd apps/vps-gateway
export GATEWAY_API_SECRET="test-secret-key-change-in-production"
./test-docker.sh
```

**For local testing** (inside container or locally):
```bash
cd apps/vps-gateway
export GATEWAY_API_SECRET="test-secret-key-change-in-production"
# If in container:
./test.sh
# Or from host:
docker exec -it vps-gateway-test /bin/sh -c "cd /app && ./test.sh"
```

### 4. Test Prometheus Metrics

View all metrics:
```bash
curl http://localhost:9091/metrics | grep vps_gateway
```

Key metrics to verify:
- `vps_gateway_dhcp_allocations_total{organization_id="org-test-001"}` - Should increment after allocating IPs
- `vps_gateway_dhcp_allocations_active{organization_id="org-test-001"}` - Should show current active allocations
- `vps_gateway_dhcp_pool_size` - Total IPs in pool
- `vps_gateway_dhcp_pool_available` - Available IPs
- `vps_gateway_dhcp_server_status` - Should be 1 (running)

### 5. Test dnsmasq Integration

Check if dnsmasq is running:
```bash
ps aux | grep dnsmasq
```

Check dnsmasq logs (if running in foreground):
```bash
# dnsmasq should log DHCP requests
tail -f /var/log/syslog | grep dnsmasq
```

Verify hosts file was created:
```bash
cat /var/lib/vps-gateway/dnsmasq.hosts
```

### 6. Test Authentication

Test with wrong secret (should fail):
```bash
grpcurl -plaintext \
  -H "x-api-secret: wrong-secret" \
  localhost:1537 \
  obiente.cloud.vpsgateway.v1.VPSGatewayService/GetGatewayInfo
```

Expected error:
```
rpc error: code = Unauthenticated desc = invalid x-api-secret
```

Test without secret (should fail):
```bash
grpcurl -plaintext \
  localhost:1537 \
  obiente.cloud.vpsgateway.v1.VPSGatewayService/GetGatewayInfo
```

Expected error:
```
rpc error: code = Unauthenticated desc = missing x-api-secret header
```

## Troubleshooting

### Service won't start

1. **Check dnsmasq is installed**:
```bash
which dnsmasq
# If not found:
# Ubuntu/Debian: sudo apt-get install dnsmasq
# Alpine: apk add dnsmasq
```

2. **Check network interface exists**:
```bash
ip addr show eth0  # Replace eth0 with your interface
```

3. **Check permissions**: dnsmasq requires root or NET_ADMIN capability

### IP allocation fails

1. **Check IP pool configuration**: Ensure pool start/end are valid and in the same subnet
2. **Check subnet mask**: Should match your network configuration
3. **Check dnsmasq is running**: `ps aux | grep dnsmasq`
4. **Check dnsmasq logs**: Look for errors in syslog

### gRPC connection refused

1. **Check service is listening**: `netstat -tlnp | grep 8080`
2. **Check firewall**: Ensure port 8080 is not blocked
3. **Check logs**: Look for startup errors

### Metrics not appearing

1. **Check metrics endpoint**: `curl http://localhost:9091/metrics`
2. **Check service is running**: Metrics are only exported when service is active
3. **Check LOG_LEVEL**: Some metrics updates are logged at debug level

## Next Steps

After verifying the service works:

1. **Integrate with API**: Update the API to use vps-gateway for IP allocation
2. **Update SSH proxy**: Modify SSH proxy to use vps-gateway for proxying
3. **Configure production**: Set up proper network configuration and secrets management
4. **Set up monitoring**: Configure Prometheus scraping and Grafana dashboards

