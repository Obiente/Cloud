#!/bin/bash

# Test script for vps-gateway service running in Docker
# This script runs tests from the host by executing commands in the container

set -e

CONTAINER_NAME="${CONTAINER_NAME:-vps-gateway-test}"
API_SECRET="${GATEWAY_API_SECRET:-test-secret-key-change-in-production}"
GATEWAY_URL="${GATEWAY_URL:-localhost:1537}"
METRICS_URL="${METRICS_URL:-localhost:9091}"

echo "=== Testing VPS Gateway Service (Docker) ==="
echo "Container: ${CONTAINER_NAME}"
echo ""

# Check if container is running
if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "Error: Container ${CONTAINER_NAME} is not running"
    echo "Start it with: docker-compose -f docker-compose.test.yml up -d"
    exit 1
fi

# Check if grpcurl is installed in container
if ! docker exec "${CONTAINER_NAME}" sh -c "command -v /root/go/bin/grpcurl" > /dev/null 2>&1; then
    echo "Installing grpcurl in container..."
    docker exec "${CONTAINER_NAME}" sh -c "apk add --no-cache go git && go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest"
fi

# Test 1: Check metrics endpoint
echo "1. Testing metrics endpoint..."
if docker exec "${CONTAINER_NAME}" curl -s -f "http://${METRICS_URL}/metrics" > /dev/null 2>&1; then
    echo "   ✓ Metrics endpoint is accessible"
else
    echo "   ✗ Metrics endpoint is not accessible"
    exit 1
fi

# Test 2: List services
echo "2. Listing gRPC services..."
if docker exec "${CONTAINER_NAME}" /root/go/bin/grpcurl -plaintext "${GATEWAY_URL}" list > /dev/null 2>&1; then
    echo "   ✓ gRPC server is responding"
    docker exec "${CONTAINER_NAME}" /root/go/bin/grpcurl -plaintext "${GATEWAY_URL}" list
else
    echo "   ✗ gRPC server is not responding"
    exit 1
fi

# Test 3: Get gateway info
echo ""
echo "3. Getting gateway info..."
RESPONSE=$(docker exec "${CONTAINER_NAME}" /root/go/bin/grpcurl -plaintext \
  -H "x-api-secret: ${API_SECRET}" \
  "${GATEWAY_URL}" \
  obiente.cloud.vpsgateway.v1.VPSGatewayService/GetGatewayInfo 2>&1)

if echo "${RESPONSE}" | grep -q "dhcpPoolStart\|dhcp_pool_start"; then
    echo "   ✓ GetGatewayInfo succeeded"
    echo "${RESPONSE}" | head -20
else
    echo "   ✗ GetGatewayInfo failed"
    echo "${RESPONSE}"
    exit 1
fi

# Test 4: Allocate IP
echo ""
echo "4. Allocating IP address..."
ALLOC_RESPONSE=$(docker exec "${CONTAINER_NAME}" /root/go/bin/grpcurl -plaintext \
  -H "x-api-secret: ${API_SECRET}" \
  -d '{
    "vps_id": "vps-test-001",
    "organization_id": "org-test-001",
    "mac_address": "00:11:22:33:44:55"
  }' \
  "${GATEWAY_URL}" \
  obiente.cloud.vpsgateway.v1.VPSGatewayService/AllocateIP 2>&1)

# Check if response contains IP address (either camelCase or snake_case)
if echo "${ALLOC_RESPONSE}" | grep -qE "(ipAddress|ip_address)"; then
    echo "   ✓ IP allocation succeeded"
    # Extract IP address - try multiple patterns
    ALLOCATED_IP=$(echo "${ALLOC_RESPONSE}" | grep -oE '"ipAddress":\s*"[^"]+"' | grep -oE '[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    if [ -z "${ALLOCATED_IP}" ]; then
        ALLOCATED_IP=$(echo "${ALLOC_RESPONSE}" | grep -oE '"ip_address":\s*"[^"]+"' | grep -oE '[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    fi
    if [ -z "${ALLOCATED_IP}" ]; then
        # Fallback: just find first IP address in response
        ALLOCATED_IP=$(echo "${ALLOC_RESPONSE}" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    fi
    if [ -n "${ALLOCATED_IP}" ]; then
        echo "   Allocated IP: ${ALLOCATED_IP}"
    fi
else
    echo "   ✗ IP allocation failed"
    echo "${ALLOC_RESPONSE}"
    exit 1
fi

# Test 5: List IPs
echo ""
echo "5. Listing allocated IPs..."
LIST_RESPONSE=$(docker exec "${CONTAINER_NAME}" /root/go/bin/grpcurl -plaintext \
  -H "x-api-secret: ${API_SECRET}" \
  "${GATEWAY_URL}" \
  obiente.cloud.vpsgateway.v1.VPSGatewayService/ListIPs 2>&1)

if echo "${LIST_RESPONSE}" | grep -q "vps-test-001"; then
    echo "   ✓ ListIPs succeeded"
    echo "${LIST_RESPONSE}" | head -10
else
    echo "   ✗ ListIPs failed or no IPs found"
    echo "${LIST_RESPONSE}"
fi

# Test 6: Release IP
echo ""
echo "6. Releasing IP address..."
RELEASE_RESPONSE=$(docker exec "${CONTAINER_NAME}" /root/go/bin/grpcurl -plaintext \
  -H "x-api-secret: ${API_SECRET}" \
  -d '{
    "vps_id": "vps-test-001"
  }' \
  "${GATEWAY_URL}" \
  obiente.cloud.vpsgateway.v1.VPSGatewayService/ReleaseIP 2>&1)

if echo "${RELEASE_RESPONSE}" | grep -q "success"; then
    echo "   ✓ IP release succeeded"
else
    echo "   ✗ IP release failed"
    echo "${RELEASE_RESPONSE}"
fi

# Test 7: Check metrics
echo ""
echo "7. Checking Prometheus metrics..."
METRICS=$(docker exec "${CONTAINER_NAME}" curl -s "http://${METRICS_URL}/metrics")
if echo "${METRICS}" | grep -q "vps_gateway_dhcp_allocations_total"; then
    echo "   ✓ Metrics are being exported"
    echo "   Sample metrics:"
    echo "${METRICS}" | grep "vps_gateway" | head -5
else
    echo "   ✗ Metrics not found"
fi

echo ""
echo "=== All tests completed ==="
echo ""
echo "To test interactively, exec into the container:"
echo "  docker exec -it ${CONTAINER_NAME} /bin/sh"

