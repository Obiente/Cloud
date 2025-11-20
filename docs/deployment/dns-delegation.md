# DNS Delegation for Self-Hosters

## Overview

DNS delegation allows self-hosted Obiente Cloud instances to use the main `my.obiente.cloud` DNS service while keeping their deployments in their own database. This enables:

- **Unified DNS**: All deployments resolve via `my.obiente.cloud` regardless of where they're hosted
- **Secure Communication**: API key authentication ensures only authorized APIs can push DNS records
- **Automatic Expiration**: DNS records expire if not refreshed, preventing stale records
- **No Port Conflicts**: No need to expose DNS port 53 on your host
- **No Nameserver Configuration**: Works automatically without DNS configuration changes
- **Subscription-Based**: Simple $2/month subscription for self-service access

## Pricing

DNS delegation is available via a **$2/month subscription**. This subscription:
- Allows self-service API key creation
- Automatically creates an API key when you subscribe
- Automatically revokes API keys if subscription is cancelled
- Can be managed via the Stripe Customer Portal

## How It Works

1. **Subscribe**: Purchase a DNS delegation subscription ($2/month) via your organization's billing page
2. **API Key Created**: An API key is automatically created when your subscription becomes active
3. **Self-Hosted API** periodically pushes DNS records to production API (every 2 minutes by default)
4. **Production API** stores these records with TTL (5 minutes by default)
5. **Production DNS Server** queries delegated records when local DB lookup fails
6. **Records Expire** automatically if not refreshed within TTL
7. **Production DNS** returns the result to the client

## Setup Instructions

### Step 1: Subscribe to DNS Delegation

1. **Go to your organization's billing page** in the dashboard
2. **Click "Subscribe to DNS Delegation"** or navigate to the DNS Delegation section
3. **Complete the Stripe checkout** ($2/month)
4. **Wait for subscription activation** - Your API key will be automatically created when the subscription becomes active

**Note**: The API key will be automatically created via webhook. If you don't see it immediately, wait a few moments and refresh.

### Step 2: Retrieve Your API Key

After subscribing, you can create or retrieve your API key:

1. **Via Dashboard**: Navigate to your organization settings → DNS Delegation
2. **Via API**: Use the `CreateDNSDelegationAPIKey` endpoint (will automatically use your organization's subscription)

**Important**: If you already have an active subscription, you can create an API key directly without superadmin access.

### Step 3: Configure Your Self-Hosted API

Add the API key and production API URL to your `docker-compose.yml` or environment:

```yaml
services:
  api:
    environment:
      # Production API URL
      DNS_DELEGATION_PRODUCTION_API_URL: "https://api.obiente.cloud"
      # API key received from production
      DNS_DELEGATION_API_KEY: "your-api-key-from-production"
      # Optional: How often to push DNS records (default: 2m)
      DNS_DELEGATION_PUSH_INTERVAL: "2m"
      # Optional: TTL for pushed DNS records (default: 300s = 5 minutes)
      DNS_DELEGATION_TTL: "300s"
```

Or set as environment variables:

```bash
export DNS_DELEGATION_PRODUCTION_API_URL="https://api.obiente.cloud"
export DNS_DELEGATION_API_KEY="your-api-key-from-production"
export DNS_DELEGATION_PUSH_INTERVAL="2m"
export DNS_DELEGATION_TTL="300s"
docker compose up -d
```

### Step 3: Verify DNS Records Are Being Pushed

Check your API logs to verify DNS records are being pushed:

```bash
docker logs obiente-api | grep -i "dns pusher"
```

You should see logs like:
```
[DNS Pusher] Successfully pushed 5 DNS records
```

### Step 4: Test DNS Resolution

Test that your deployments resolve via production DNS:

```bash
# Query DNS for your deployment
dig deploy-123.my.obiente.cloud

# Should return the IP from your self-hosted database
```

## How to Get an API Key

**For Self-Hosters:** DNS delegation is now subscription-based ($2/month). To get an API key:

1. **Subscribe**: Purchase a DNS delegation subscription via your organization's billing page
2. **API Key Created Automatically**: An API key is automatically created when your subscription becomes active
3. **Access Your Key**: View your API key in the dashboard or create a new one via the API

**For Development:** If you're developing locally and need an API key for testing, you can:
- Subscribe to DNS delegation for your development organization
- Contact superadmins for a temporary key

**For Superadmins:** Superadmins can create API keys manually for any organization without requiring a subscription.

### Creating API Keys (Self-Service)

Users with active DNS delegation subscriptions can create API keys directly:

```typescript
import { createPromiseClient } from "@connectrpc/connect";
import { SuperadminService } from "@obiente/proto/superadmin/v1/superadmin_service";

const client = createPromiseClient(SuperadminService, transport);
const response = await client.createDNSDelegationAPIKey({
  description: "Self-hosted instance at example.com",
  sourceApi: "https://selfhosted-api.example.com"
});

console.log("API Key:", response.apiKey); // Save this immediately!
```

**Note**: If your organization already has an active API key, you'll need to revoke it first before creating a new one.

### Creating API Keys (Superadmin)

Superadmins can create API keys for any organization without requiring a subscription:

```typescript
import { createPromiseClient } from "@connectrpc/connect";
import { SuperadminService } from "@obiente/proto/superadmin/v1/superadmin_service";

const client = createPromiseClient(SuperadminService, transport);
const response = await client.createDNSDelegationAPIKey({
  description: "Self-hosted instance at example.com",
  sourceApi: "https://selfhosted-api.example.com"
});

console.log("API Key:", response.apiKey); // Save this immediately!
```

### Revoking API Keys

Revoke an API key if it's compromised or no longer needed:

```typescript
await client.revokeDNSDelegationAPIKey({
  apiKey: "abc123..."
});
```

**Note**: When a subscription is cancelled, API keys are automatically revoked.

1. **API Key Security**:
   - Use strong, randomly generated API keys (32 bytes, base64 encoded)
   - Never commit API keys to version control
   - Rotate API keys periodically
   - Revoke compromised keys immediately

2. **HTTPS Only**:
   - Always use HTTPS for DNS delegation
   - Use valid SSL certificates (Let's Encrypt recommended)
   - Verify SSL certificate chain

3. **TTL Configuration**:
   - Set appropriate TTL values (default: 300 seconds)
   - Shorter TTL = faster expiration but more frequent pushes
   - Longer TTL = less frequent pushes but slower expiration

4. **Push Interval**:
   - Default: 2 minutes (records refreshed every 2 minutes)
   - Must be less than TTL to prevent expiration
   - Recommended: Push interval < TTL / 2

## Troubleshooting

### DNS Records Not Pushing

1. **Check API Key**:
   ```bash
   # Verify API key is set
   echo $DNS_DELEGATION_API_KEY
   ```

2. **Check Production API URL**:
   ```bash
   # Verify production API URL is correct
   echo $DNS_DELEGATION_PRODUCTION_API_URL
   ```

3. **Check API Logs**:
   ```bash
   docker logs obiente-api | grep -i "dns pusher"
   ```

4. **Test Push Manually**:
   ```bash
   # Test pushing a single record
   curl -X POST https://api.obiente.cloud/dns/push \
     -H "Authorization: Bearer $DNS_DELEGATION_API_KEY" \
     -H "Content-Type: application/json" \
     -d '{
       "domain": "deploy-123.my.obiente.cloud",
       "record_type": "A",
       "records": ["203.0.113.1"],
       "ttl": 300
     }'
   ```

### DNS Records Not Resolving

1. **Check if Records Are Pushed**:
   ```bash
   # Check production API logs
   docker logs obiente-api | grep -i "dns/push"
   ```

2. **Check Record Expiration**:
   - Records expire if not refreshed within TTL
   - Ensure push interval < TTL
   - Check that pusher is running (should see periodic logs)

3. **Check DNS Server Logs**:
   ```bash
   docker logs obiente-dns | grep -i "delegated"
   ```

### Common Issues

**Issue**: "DNS pusher not configured"
- **Solution**: Set `DNS_DELEGATION_PRODUCTION_API_URL` and `DNS_DELEGATION_API_KEY`

**Issue**: "Invalid API key"
- **Solution**: Verify API key matches the one created on production
- **Solution**: Check if API key was revoked

**Issue**: "API returned status 401"
- **Solution**: API key is invalid or revoked
- **Solution**: Request a new API key from production

**Issue**: "DNS records expire too quickly"
- **Solution**: Increase `DNS_DELEGATION_TTL` (e.g., `600s` for 10 minutes)
- **Solution**: Decrease `DNS_DELEGATION_PUSH_INTERVAL` (e.g., `1m` for faster refresh)

## Example Configuration

### Self-Hosted API (docker-compose.yml)

```yaml
services:
  api:
    environment:
      DNS_DELEGATION_PRODUCTION_API_URL: "https://api.obiente.cloud"
      DNS_DELEGATION_API_KEY: "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6"
      DNS_DELEGATION_PUSH_INTERVAL: "2m"
      DNS_DELEGATION_TTL: "300s"
      NODE_IPS: "default:203.0.113.1"  # Your node IP
```

### Production API (docker-compose.yml)

```yaml
services:
  api:
    environment:
      # API keys are managed via /dns/delegation/api-keys/create endpoint
      # No environment variables needed for accepting pushed records
```

## API Endpoints

### Push DNS Record (Single)

```
POST /dns/push
Authorization: Bearer <api-key>
Content-Type: application/json

{
  "domain": "deploy-123.my.obiente.cloud",
  "record_type": "A",
  "records": ["203.0.113.1"],
  "ttl": 300
}
```

### Push DNS Records (Batch)

```
POST /dns/push/batch
Authorization: Bearer <api-key>
Content-Type: application/json

{
  "records": [
    {
      "domain": "deploy-123.my.obiente.cloud",
      "record_type": "A",
      "records": ["203.0.113.1"],
      "ttl": 300
    },
    {
      "domain": "gameserver-123.my.obiente.cloud",
      "record_type": "A",
      "records": ["203.0.113.2"],
      "ttl": 300
    }
  ]
}
```

### Create API Key (Superadmin Only)

**Connect RPC Endpoint:**
```
POST /obiente.cloud.superadmin.v1.SuperadminService/CreateDNSDelegationAPIKey
Authorization: Bearer <superadmin-token>
Content-Type: application/json

{
  "description": "Self-hosted instance at example.com",
  "source_api": "https://selfhosted-api.example.com"
}
```

### Revoke API Key (Superadmin Only)

**Connect RPC Endpoint:**
```
POST /obiente.cloud.superadmin.v1.SuperadminService/RevokeDNSDelegationAPIKey
Authorization: Bearer <superadmin-token>
Content-Type: application/json

{
  "api_key": "abc123..."
}
```

## Record Expiration

DNS records automatically expire if not refreshed within the TTL period:

- **Default TTL**: 300 seconds (5 minutes)
- **Default Push Interval**: 120 seconds (2 minutes)
- **Records expire** if not refreshed within TTL
- **Expired records** are automatically cleaned up every 5 minutes

This ensures that:
- Stopped deployments automatically stop resolving (records expire)
- No manual cleanup needed (records expire automatically)
- Self-hosters don't need to explicitly remove records

## Development Setup

For development environments, you can use the same push-based delegation:

```bash
# Set production API URL
export DNS_DELEGATION_PRODUCTION_API_URL="https://api.obiente.cloud"

# Set API key from production
export DNS_DELEGATION_API_KEY="your-dev-api-key"

# Start services
docker compose up -d
```

Your dev deployments will automatically push DNS records to production, allowing them to resolve via `my.obiente.cloud` DNS.
