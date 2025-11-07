#!/bin/bash
# Script to verify DNS delegation records in production

set -e

API_KEY="${DNS_DELEGATION_API_KEY:-}"
PROD_URL="${DNS_DELEGATION_PRODUCTION_API_URL:-https://api.obiente.cloud}"

if [ -z "$API_KEY" ]; then
    echo "Error: DNS_DELEGATION_API_KEY environment variable not set"
    echo ""
    echo "Set it from your local API container:"
    echo "  docker exec cloud-api-1 env | grep DNS_DELEGATION_API_KEY"
    echo ""
    echo "Or export it:"
    echo "  export DNS_DELEGATION_API_KEY='your-api-key-here'"
    exit 1
fi

echo "=== Verifying DNS Delegation Records in Production ==="
echo "Production API: $PROD_URL"
echo ""

# Test API key validity
echo "Testing API key..."
TEST_RESPONSE=$(curl -s -w "\n%{http_code}" -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -X POST "$PROD_URL/dns/push" \
    -d '{"domain":"test-verify.my.obiente.cloud","record_type":"A","records":["127.0.0.1"],"ttl":60}' 2>&1)

HTTP_CODE=$(echo "$TEST_RESPONSE" | tail -1)
BODY=$(echo "$TEST_RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    echo "✓ API key is valid"
elif [ "$HTTP_CODE" = "401" ]; then
    echo "❌ API key is invalid or expired"
    echo "Response: $BODY"
    exit 1
else
    echo "⚠ Unexpected response: HTTP $HTTP_CODE"
    echo "Response: $BODY"
fi

echo ""
echo "=== Note ==="
echo "To view delegated DNS records, you need to:"
echo ""
echo "1. View them on the PRODUCTION dashboard:"
echo "   https://app.obiente.cloud/superadmin/dns"
echo ""
echo "2. Or query production API directly (requires authentication):"
echo "   curl -H 'Authorization: Bearer <your-jwt-token>' \\"
echo "        $PROD_URL/superadmin.v1.SuperadminService/ListDelegatedDNSRecords"
echo ""
echo "The local dashboard queries the LOCAL database, so it won't show"
echo "records that were pushed to production."

