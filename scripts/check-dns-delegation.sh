#!/bin/bash
# Script to check DNS delegation configuration and status

set -e

echo "=== DNS Delegation Configuration Check ==="
echo ""

# Find API container
API_CONTAINER=$(docker ps --format "{{.Names}}" | grep -E "obiente-api|api" | head -1)

if [ -z "$API_CONTAINER" ]; then
    echo "❌ API container not found"
    echo "   Make sure your containers are running: docker compose up -d"
    exit 1
fi

echo "Found API container: $API_CONTAINER"
echo ""

# Check environment variables
echo "=== Environment Variables ==="
PROD_URL=$(docker exec "$API_CONTAINER" env 2>/dev/null | grep "DNS_DELEGATION_PRODUCTION_API_URL" | cut -d'=' -f2- || echo "")
API_KEY=$(docker exec "$API_CONTAINER" env 2>/dev/null | grep "DNS_DELEGATION_API_KEY" | cut -d'=' -f2- || echo "")
PUSH_INTERVAL=$(docker exec "$API_CONTAINER" env 2>/dev/null | grep "DNS_DELEGATION_PUSH_INTERVAL" | cut -d'=' -f2- || echo "")
TTL=$(docker exec "$API_CONTAINER" env 2>/dev/null | grep "DNS_DELEGATION_TTL" | cut -d'=' -f2- || echo "")

if [ -z "$PROD_URL" ] || [ "$PROD_URL" = "" ]; then
    echo "❌ DNS_DELEGATION_PRODUCTION_API_URL is not set"
else
    echo "✓ DNS_DELEGATION_PRODUCTION_API_URL: $PROD_URL"
fi

if [ -z "$API_KEY" ] || [ "$API_KEY" = "" ]; then
    echo "❌ DNS_DELEGATION_API_KEY is not set"
else
    API_KEY_MASKED=$(echo "$API_KEY" | sed 's/\(.\{8\}\).*/\1.../')
    echo "✓ DNS_DELEGATION_API_KEY: $API_KEY_MASKED"
fi

if [ -n "$PUSH_INTERVAL" ] && [ "$PUSH_INTERVAL" != "" ]; then
    echo "✓ DNS_DELEGATION_PUSH_INTERVAL: $PUSH_INTERVAL"
else
    echo "ℹ DNS_DELEGATION_PUSH_INTERVAL: (default: 2m)"
fi

if [ -n "$TTL" ] && [ "$TTL" != "" ]; then
    echo "✓ DNS_DELEGATION_TTL: $TTL"
else
    echo "ℹ DNS_DELEGATION_TTL: (default: 300s)"
fi

echo ""

# Check API logs for DNS pusher status
echo "=== DNS Pusher Status (from logs) ==="
PUSHER_STARTED=$(docker logs "$API_CONTAINER" 2>&1 | grep -i "DNS pusher service started" | tail -1)
PUSHER_NOT_CONFIGURED=$(docker logs "$API_CONTAINER" 2>&1 | grep -i "DNS pusher not configured" | tail -1)
PUSHER_ERRORS=$(docker logs "$API_CONTAINER" 2>&1 | grep -i "DNS Pusher.*Failed\|DNS Pusher.*Error" | tail -5)

if [ -n "$PUSHER_STARTED" ]; then
    echo "✓ $PUSHER_STARTED"
elif [ -n "$PUSHER_NOT_CONFIGURED" ]; then
    echo "❌ $PUSHER_NOT_CONFIGURED"
    echo ""
    echo "   To fix:"
    echo "   1. Set DNS_DELEGATION_PRODUCTION_API_URL in your .env file"
    echo "   2. Set DNS_DELEGATION_API_KEY in your .env file"
    echo "   3. Restart the API: docker compose restart api"
else
    echo "ℹ DNS pusher status not found in logs"
fi

if [ -n "$PUSHER_ERRORS" ]; then
    echo ""
    echo "⚠ Recent DNS pusher errors:"
    echo "$PUSHER_ERRORS"
fi

echo ""

# Check recent push activity
echo "=== Recent DNS Push Activity ==="
RECENT_PUSHES=$(docker logs "$API_CONTAINER" 2>&1 | grep -i "DNS Pusher.*Successfully pushed\|DNS Pusher.*pushed.*DNS records" | tail -5)
if [ -n "$RECENT_PUSHES" ]; then
    echo "$RECENT_PUSHES"
else
    echo "ℹ No recent successful pushes found"
fi

echo ""

# Check if there are deployments to push
echo "=== Checking for Deployments ==="
DEPLOYMENT_COUNT=$(docker exec "$API_CONTAINER" psql -h postgres -U postgres -d obiente -t -c "SELECT COUNT(*) FROM deployment_locations WHERE status = 'running';" 2>/dev/null | tr -d ' ' || echo "0")
if [ "$DEPLOYMENT_COUNT" != "0" ] && [ -n "$DEPLOYMENT_COUNT" ]; then
    echo "✓ Found $DEPLOYMENT_COUNT running deployment(s) that should be pushed"
else
    echo "ℹ No running deployments found"
fi

echo ""
echo "=== Next Steps ==="
if [ -z "$PROD_URL" ] || [ -z "$API_KEY" ]; then
    echo "1. Add DNS delegation environment variables to your .env file:"
    echo "   DNS_DELEGATION_PRODUCTION_API_URL=https://api.obiente.cloud"
    echo "   DNS_DELEGATION_API_KEY=your-api-key-here"
    echo ""
    echo "2. Restart the API service:"
    echo "   docker compose restart api"
    echo ""
    echo "3. Check logs again:"
    echo "   docker logs obiente-api | grep -i 'dns pusher'"
else
    echo "1. Check API logs for DNS pusher activity:"
    echo "   docker logs obiente-api | grep -i 'dns pusher'"
    echo ""
    echo "2. If no pushes are happening, check for errors:"
    echo "   docker logs obiente-api | grep -i 'dns pusher.*fail\|dns pusher.*error'"
fi

