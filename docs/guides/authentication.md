# Authentication Setup Guide

## How Authentication Works

The API validates bearer tokens by calling Zitadel's **userinfo endpoint**, just like your frontend does. This is simple and reliable:

1. Frontend gets token from Zitadel (during login)
2. Frontend sends token to API in `Authorization: Bearer <token>` header
3. API validates token by calling `{ZITADEL_URL}/oidc/v1/userinfo`
4. If valid, API extracts user info and processes the request

No JWKS, no local JWT validation - just a simple userinfo check.

## Error: TLS Certificate Verification Failed

If you're seeing this error:

```
failed to fetch user info: x509: certificate is valid for [traefik-default], not obiente.cloud
```

This means the API is trying to call Zitadel's userinfo endpoint over HTTPS, but there's a TLS certificate issue.

## Solutions

### Option 1: Development Mode - Disable Authentication (Easiest)

**⚠️ ONLY for local development - NEVER in production!**

```bash
# In your .env file:
DISABLE_AUTH=true
```

When `DISABLE_AUTH=true`, authentication is completely bypassed and a mock development user is automatically provided:

- **User ID**: `mem-development`
- **Email**: `dev@obiente.local`
- **Name**: `Development User`
- **Roles**: `admin` and `owner` (full permissions)

This allows you to develop and test all features without needing Zitadel configured. All API endpoints work as if a fully authenticated dev user is logged in, including:
- Creating and managing deployments
- Accessing all organizations and resources
- Full admin permissions for testing

**Important**: Both the API and Dashboard must have `DISABLE_AUTH=true` set in their environment for this to work correctly.

### Option 2: Skip TLS Verification (Development)

**⚠️ ONLY for development - NEVER in production!**

```bash
# In your .env file:
SKIP_TLS_VERIFY=true
```

This allows HTTPS connections even with invalid certificates.

### Option 3: Use HTTP for Zitadel (Development)

If your Zitadel is on HTTP (local development):

```bash
ZITADEL_URL=http://localhost:8080
```

### Option 4: Configure Proper SSL (Production)

For production, ensure your domain has proper SSL:

#### 4a. Configure DNS

```bash
# Point your domain to your server
A    obiente.cloud    -> YOUR_SERVER_IP
A    auth.example.com -> YOUR_AUTH_SERVER_IP
```

#### 4b. Wait for Let's Encrypt

Traefik will automatically request certificates via Let's Encrypt. Check logs:

```bash
docker service logs obiente_traefik | grep -i certificate
```

#### 4c. Configure Environment

```bash
# .env file
ZITADEL_URL=https://auth.obiente.cloud
DISABLE_AUTH=false
SKIP_TLS_VERIFY=false
```

## Testing Authentication

### 1. Check Auth Configuration

Logs will show on startup:

```
🔐 Auth Configuration:
  Zitadel URL: https://auth.obiente.cloud
  UserInfo URL: https://auth.obiente.cloud/oidc/v1/userinfo
  Skip TLS Verify: false
```

### 2. Test Without Auth (Development)

```bash
# Set in .env
DISABLE_AUTH=true

# Rebuild and restart
docker service update --force obiente_api

# Test API
curl http://localhost:3001/api.deployments.v1.DeploymentService/ListDeployments
```

### 3. Test With Auth

```bash
# Get a token from your auth provider
TOKEN="your-jwt-token-here"

# Test API with token
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:3001/api.deployments.v1.DeploymentService/ListDeployments
```

## Common Issues

### Issue: "Token validated for user" not appearing in logs

**Problem**: Userinfo fetch is failing

**Solution**: Check the userinfo URL is accessible:

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  https://auth.obiente.cloud/oidc/v1/userinfo
```

### Issue: "invalid token" error

**Problem**: Token validation failed

**Causes**:

- Token expired
- Token revoked
- Wrong Zitadel URL
- Network/firewall issue

**Debug**:

```bash
# Enable detailed logging
LOG_LEVEL=debug

# Check logs for auth errors
docker service logs obiente_api | grep -E "(Auth|token|validated)"
```

### Issue: Can't connect to Zitadel

**Problem**: Network/firewall blocking JWKS fetch

**Solution**:

1. Check Zitadel is accessible from the API (if running in a container):

```bash
docker exec -it <api-container> wget -O- https://auth.obiente.cloud/oidc/v1/userinfo
# Replace <api-container> with your container ID or name
```

2. If it fails, check firewall/network rules

## Environment Variables Reference

```bash
# Required for Production
ZITADEL_URL=https://auth.obiente.cloud

# Development Only (NEVER in production!)
DISABLE_AUTH=true          # Skip all authentication
SKIP_TLS_VERIFY=true       # Skip SSL certificate verification
```

## Security Best Practices

### ✅ Production Checklist

- [ ] `DISABLE_AUTH=false` (or not set)
- [ ] `SKIP_TLS_VERIFY=false` (or not set)
- [ ] Valid SSL certificates on all domains
- [ ] Firewall rules configured
- [ ] Userinfo URL accessible from API
- [ ] Token expiration configured

### ❌ Never in Production

- ❌ `DISABLE_AUTH=true`
- ❌ `SKIP_TLS_VERIFY=true`
- ❌ HTTP URLs for JWKS
- ❌ Default/hardcoded credentials
- ❌ Publicly accessible without auth

## Quick Start for Development

```bash
# 1. Edit .env
DISABLE_AUTH=true
LOG_LEVEL=debug
CORS_ORIGIN=*

# 2. For API (Docker)
docker build -f apps/api/Dockerfile -t obiente/cloud-api:latest .
docker service update --force obiente_api

# 3. For Dashboard (if running separately)
# Make sure DISABLE_AUTH=true is set in your environment
export DISABLE_AUTH=true
pnpm --filter dashboard dev

# 4. Test
curl http://localhost:3001/health

# The dashboard will automatically show you as "Development User" (mem-development)
# All API calls will work without tokens
```

## Quick Start for Production

```bash
# 1. Ensure Zitadel is running and accessible
# - Your frontend already authenticates with it
# - API just needs to validate tokens

# 2. Edit .env
ZITADEL_URL=https://auth.obiente.cloud
DISABLE_AUTH=false
SKIP_TLS_VERIFY=false

# 3. Ensure DNS is configured
dig auth.your-domain.com

# 4. Deploy
docker stack deploy -c docker-compose.swarm.ha.yml obiente

# 5. Wait for SSL certificates
docker service logs obiente_traefik | grep certificate

# 6. Test
curl https://api.your-domain.com/health
```
