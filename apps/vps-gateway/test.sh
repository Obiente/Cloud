#!/bin/bash

# Test script for vps-gateway service
# This script tests the basic functionality of the vps-gateway service

set -e

GATEWAY_URL="localhost:8080"
METRICS_URL="localhost:9091"
API_SECRET="${GATEWAY_API_SECRET:-test-secret-key-change-in-production}"

echo "=== Testing VPS Gateway Service ==="
echo ""

# Check if grpcurl is installed, try common locations
GRPCURL_CMD=""
if command -v grpcurl &> /dev/null; then
    GRPCURL_CMD="grpcurl"
elif [ -f "/root/go/bin/grpcurl" ]; then
    GRPCURL_CMD="/root/go/bin/grpcurl"
    export PATH=$PATH:/root/go/bin
elif ! command -v grpcurl &> /dev/null; then
    echo "Error: grpcurl is not installed"
    echo "Install it with: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest"
    exit 1
fi

# Use grpcurl command
GRPCURL_CMD="${GRPCURL_CMD:-grpcurl}"

# Test 1: Check metrics endpoint
echo "1. Testing metrics endpoint..."
if ${CURL_CMD:-curl} -s -f "${METRICS_URL}/metrics" > /dev/null 2>&1; then
    echo "   ✓ Metrics endpoint is accessible"
else
    echo "   ✗ Metrics endpoint is not accessible"
    exit 1
fi

# Test 2: List services
echo "2. Listing gRPC services..."
if ${GRPCURL_CMD} -plaintext "${GATEWAY_URL}" list > /dev/null 2>&1; then
    echo "   ✓ gRPC server is responding"
    ${GRPCURL_CMD} -plaintext "${GATEWAY_URL}" list
else
    echo "   ✗ gRPC server is not responding"
    echo "   Note: If running from host, ensure grpcurl is installed in container"
    echo "   Trying to install grpcurl..."
    if command -v go &> /dev/null; then
        go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
        export PATH=$PATH:/root/go/bin
        GRPCURL_CMD="/root/go/bin/grpcurl"
        ${GRPCURL_CMD} -plaintext "${GATEWAY_URL}" list
    else
        exit 1
    fi
fi

# Test 3: Get gateway info
echo ""
echo "3. Getting gateway info..."
RESPONSE=$(${GRPCURL_CMD} -plaintext \
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
ALLOC_RESPONSE=$(${GRPCURL_CMD} -plaintext \
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
LIST_RESPONSE=$(${GRPCURL_CMD} -plaintext \
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
RELEASE_RESPONSE=$(${GRPCURL_CMD} -plaintext \
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
METRICS=$(${CURL_CMD:-curl} -s "${METRICS_URL}/metrics")
if echo "${METRICS}" | grep -q "vps_gateway_dhcp_allocations_total"; then
    echo "   ✓ Metrics are being exported"
    echo "   Sample metrics:"
    echo "${METRICS}" | grep "vps_gateway" | head -5
else
    echo "   ✗ Metrics not found"
fi

echo ""
echo "=== All tests completed ==="

