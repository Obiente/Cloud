# Docker Registry Authentication Setup

The Obiente Cloud Docker Registry uses HTTP Basic Authentication (htpasswd) to secure private repositories.

## Initial Setup

Before deploying the Swarm stack, you need to generate the htpasswd file:

### Option 1: Use the Setup Script (Recommended)

```bash
# Generate htpasswd with default username (obiente) and random password
./scripts/setup-registry-auth.sh

# Or specify custom credentials
REGISTRY_USERNAME=myuser REGISTRY_PASSWORD=mypassword ./scripts/setup-registry-auth.sh
```

The script will:
- Create `/var/lib/obiente/registry-auth/htpasswd`
- Generate a random password if `REGISTRY_PASSWORD` is not set
- Set proper file permissions

### Option 2: Manual Setup

```bash
# Install htpasswd (if not already installed)
# Debian/Ubuntu:
sudo apt-get install apache2-utils

# RHEL/CentOS:
sudo yum install httpd-tools

# Create auth directory
sudo mkdir -p /var/lib/obiente/registry-auth

# Generate htpasswd file
sudo htpasswd -Bbn obiente your-password > /var/lib/obiente/registry-auth/htpasswd

# Set proper permissions
sudo chmod 600 /var/lib/obiente/registry-auth/htpasswd
sudo chown root:root /var/lib/obiente/registry-auth/htpasswd
```

### Option 3: Use Docker Volume (Swarm)

If using Docker Swarm, you can create the htpasswd file in a volume:

```bash
# Create a temporary container to generate htpasswd
docker run --rm -v registry_auth:/auth httpd:2.4-alpine \
  htpasswd -Bbn obiente your-password > /tmp/htpasswd

# Copy to volume (requires access to Docker volume)
# This is more complex - use Option 1 or 2 instead
```

## Environment Variables

Set these environment variables in your `.env` file or deployment:

```bash
# Registry authentication credentials
REGISTRY_USERNAME=obiente          # Default: obiente
REGISTRY_PASSWORD=your-secure-password  # Required for pushing/pulling images
```

**Important**: These credentials must match the username/password in the htpasswd file.

## Using the Registry

The registry is accessible via HTTPS at `https://registry.yourdomain.com` (replace `yourdomain.com` with your `DOMAIN` environment variable).

### Login

```bash
# Login to the registry
docker login https://registry.yourdomain.com -u obiente -p your-password

# Or using environment variable
echo $REGISTRY_PASSWORD | docker login https://registry.yourdomain.com -u $REGISTRY_USERNAME --password-stdin
```

### Push Images

```bash
# Tag your image
docker tag myimage:latest registry.yourdomain.com/myimage:latest

# Push to registry
docker push registry.yourdomain.com/myimage:latest
```

### Pull Images

```bash
# Pull from registry
docker pull registry.yourdomain.com/myimage:latest
```

## Registry Security

The registry is configured to use HTTPS via Traefik with Let's Encrypt certificates. This means:

- **No insecure registry configuration needed**: Docker can connect securely without requiring `/etc/docker/daemon.json` changes
- **Automatic certificate management**: Traefik handles Let's Encrypt certificate generation and renewal
- **Secure by default**: All registry communication is encrypted

The registry is accessible at `https://registry.yourdomain.com` (where `yourdomain.com` is your `DOMAIN` environment variable).

## Swarm Deployment

When deploying with Docker Swarm:

1. **Generate htpasswd file** (see Initial Setup above)
2. **Set environment variables** in your `.env` file:
   ```bash
   REGISTRY_USERNAME=obiente
   REGISTRY_PASSWORD=your-password
   DOMAIN=yourdomain.com  # Used to construct registry URL (https://registry.yourdomain.com)
   ```
3. **Deploy the stack** - the registry service will automatically use the htpasswd file from the volume

The API service will automatically:
- Authenticate before pushing images
- Authenticate before pulling images
- Pass credentials to Swarm services via `--with-registry-auth=true`

## Security Considerations

1. **Password Storage**: Store `REGISTRY_PASSWORD` securely (e.g., in a secrets manager, not in version control)
2. **File Permissions**: The htpasswd file should be readable only by root (600)
3. **HTTPS Only**: The registry is accessible only via HTTPS at `https://registry.yourdomain.com` with Let's Encrypt certificates
4. **No Insecure Registry Required**: Docker can connect securely without requiring insecure registry configuration

## Troubleshooting

### Authentication Fails

- Verify htpasswd file exists: `ls -la /var/lib/obiente/registry-auth/htpasswd`
- Check file permissions: Should be `600` and owned by `root:root`
- Verify credentials match: `htpasswd -v /var/lib/obiente/registry-auth/htpasswd obiente`
- Check environment variables: `echo $REGISTRY_USERNAME` and `echo $REGISTRY_PASSWORD`

### Cannot Push/Pull

- Ensure you're logged in: `docker login https://registry.yourdomain.com`
- Check registry logs: `docker service logs obiente_registry`
- Verify registry is running: `docker service ps obiente_registry`
- Verify HTTPS endpoint is accessible: `curl -k https://registry.yourdomain.com/v2/`

### Swarm Services Can't Pull Images

- Ensure `--with-registry-auth=true` is set (already configured in deployment code)
- Verify credentials are available on all Swarm nodes
- Check if registry is accessible: `docker exec -it <container> curl -k https://registry.yourdomain.com/v2/`
- Verify `REGISTRY_URL` environment variable is set correctly in the API service

### "dial tcp: lookup registry" Error

This error indicates that Docker cannot resolve the registry hostname. Possible causes:

1. **Registry service not running**: Check with `docker service ps obiente_registry`
2. **DNS resolution issue**: Ensure `DOMAIN` environment variable is set correctly
3. **Certificate issue**: Verify Let's Encrypt certificate was issued for `registry.yourdomain.com`

### Certificate Issues

If you see certificate errors:

1. **Check Traefik logs**: `docker service logs obiente_traefik | grep -i certificate`
2. **Verify domain is accessible**: Ensure `registry.yourdomain.com` resolves to your Traefik IP
3. **Check Let's Encrypt rate limits**: If you've made many certificate requests, you may need to wait

