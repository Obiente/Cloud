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

### Login

```bash
# Login to the registry
docker login registry:5000 -u obiente -p your-password

# Or using environment variable
echo $REGISTRY_PASSWORD | docker login registry:5000 -u $REGISTRY_USERNAME --password-stdin
```

### Push Images

```bash
# Tag your image
docker tag myimage:latest registry:5000/myimage:latest

# Push to registry
docker push registry:5000/myimage:latest
```

### Pull Images

```bash
# Pull from registry
docker pull registry:5000/myimage:latest
```

## Swarm Deployment

When deploying with Docker Swarm:

1. **Generate htpasswd file** (see Initial Setup above)
2. **Set environment variables** in your `.env` file:
   ```bash
   REGISTRY_USERNAME=obiente
   REGISTRY_PASSWORD=your-password
   ```
3. **Deploy the stack** - the registry service will automatically use the htpasswd file from the volume

The API service will automatically:
- Authenticate before pushing images
- Authenticate before pulling images
- Pass credentials to Swarm services via `--with-registry-auth=true`

## Security Considerations

1. **Password Storage**: Store `REGISTRY_PASSWORD` securely (e.g., in a secrets manager, not in version control)
2. **File Permissions**: The htpasswd file should be readable only by root (600)
3. **Network**: The registry is accessible internally at `registry:5000` and externally via Traefik at `registry.yourdomain.com`
4. **HTTPS**: External access is secured via Traefik with Let's Encrypt certificates

## Troubleshooting

### Authentication Fails

- Verify htpasswd file exists: `ls -la /var/lib/obiente/registry-auth/htpasswd`
- Check file permissions: Should be `600` and owned by `root:root`
- Verify credentials match: `htpasswd -v /var/lib/obiente/registry-auth/htpasswd obiente`
- Check environment variables: `echo $REGISTRY_USERNAME` and `echo $REGISTRY_PASSWORD`

### Cannot Push/Pull

- Ensure you're logged in: `docker login registry:5000`
- Check registry logs: `docker service logs obiente_registry`
- Verify registry is running: `docker service ps obiente_registry`

### Swarm Services Can't Pull Images

- Ensure `--with-registry-auth=true` is set (already configured in deployment code)
- Verify credentials are available on all Swarm nodes
- Check if registry is accessible from worker nodes: `docker exec -it <container> ping registry`

