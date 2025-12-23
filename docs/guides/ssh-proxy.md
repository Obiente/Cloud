# SSH Proxy Configuration

The VPS service includes an integrated SSH proxy server that allows users to connect to their VPS instances via SSH. This proxy runs on port 2222 (configurable via `SSH_PROXY_PORT`) and routes connections to the appropriate VPS instances.

## Supported Features

The SSH proxy supports most standard SSH features:

- **Interactive shell sessions** - Standard terminal access ✅
- **Port forwarding (local)** - Forward local ports to VPS with `ssh -L` ✅
- **Port forwarding (remote)** - Forward VPS ports to your machine with `ssh -R` ✅
- **SSH agent forwarding** - Use your local SSH keys on the VPS with `ssh -A` ✅
- **X11 forwarding** - Run graphical applications with `ssh -X` or `ssh -Y` ✅

### Examples

```bash
# Standard connection
ssh -p 2222 root@vps-xxx@your-domain.com

# Local port forwarding (access VPS port 80 on localhost:8080)
ssh -p 2222 -L 8080:localhost:80 root@vps-xxx@your-domain.com

# Remote port forwarding (expose your localhost:3000 on VPS port 8080)
ssh -p 2222 -R 8080:localhost:3000 root@vps-xxx@your-domain.com

# Agent forwarding (use your local SSH keys on the VPS)
ssh -p 2222 -A root@vps-xxx@your-domain.com

# X11 forwarding (run graphical applications)
ssh -p 2222 -X root@vps-xxx@your-domain.com xeyes

# Trusted X11 forwarding (less secure, but faster)
ssh -p 2222 -Y root@vps-xxx@your-domain.com firefox

# Combined features
ssh -p 2222 -A -X -L 8080:localhost:80 root@vps-xxx@your-domain.com
```

## Host Key Management

When running multiple replicas of vps-service in Docker Swarm, they must share the same SSH host key to prevent host key verification warnings and enable SSH port forwarding.

### Configuration

The vps-service supports loading a shared SSH host key from multiple sources (in priority order):

1. **SSH_PROXY_HOST_KEY_FILE** - Docker secret file path (recommended for Swarm)
2. **SSH_PROXY_HOST_KEY** - Environment variable with PEM-encoded key
3. **SSH_PROXY_HOST_KEY_PATH** - File path on container filesystem
4. **Auto-generated** - Creates new key if none of the above are found (not recommended for production)

## Docker Swarm Setup (Production/HA)

### 1. Generate the SSH Host Key

Generate a new RSA key for the SSH proxy:

```bash
# Generate the key (no passphrase)
ssh-keygen -t rsa -b 2048 -f ssh_proxy_host_key -N ""

# Verify the key was created
ls -l ssh_proxy_host_key*
```

This creates two files:
- `ssh_proxy_host_key` - Private key (this is what we'll use)
- `ssh_proxy_host_key.pub` - Public key (informational only)

### 2. Create Docker Secret

Create a Docker secret with the private key:

```bash
# Create the secret
docker secret create ssh_proxy_host_key ssh_proxy_host_key

# Verify it was created
docker secret ls | grep ssh_proxy_host_key

# Remove the local files for security
rm ssh_proxy_host_key ssh_proxy_host_key.pub
```

### 3. Deploy Stack

The `docker-compose.swarm.ha.yml` file is already configured to use this secret. Deploy your stack:

```bash
docker stack deploy -c docker-compose.swarm.ha.yml obiente
```

All replicas of the vps-service will use the same SSH host key from the secret.

### 4. Verify

Check that the service loaded the key:

```bash
# View logs from all replicas
docker service logs obiente_vps-service | grep "SSH host key"
```

You should see logs like:
```
[SSHProxy] Loading SSH host key from SSH_PROXY_HOST_KEY_FILE: /run/secrets/ssh_proxy_host_key
[SSHProxy] Successfully loaded SSH host key from Docker secret
[SSHProxy] Key fingerprint: SHA256:6rv7yl5CcWumP0JVIZOheXlXUOhPUrW63o9/Zskdn7Y
```

All replicas should show the same fingerprint.

## Docker Compose Setup (Development)

For local development with docker-compose, the SSH host key is persisted using a Docker volume:

```bash
docker-compose up -d vps-service
```

The volume `vps_ssh_host_key` persists the key at `/var/lib/obiente/ssh_proxy_host_key` inside the container. This ensures the same key is used across container restarts.

To regenerate the key (for testing):

```bash
# Stop the service
docker-compose stop vps-service

# Remove the volume
docker volume rm cloud_vps_ssh_host_key

# Start the service (a new key will be generated)
docker-compose up -d vps-service

# Check the new fingerprint
docker-compose logs vps-service | grep fingerprint
```

## Advanced Configuration

### Using Environment Variable

Instead of a Docker secret or volume, you can provide the key directly as an environment variable:

```yaml
environment:
  SSH_PROXY_HOST_KEY: |
    -----BEGIN RSA PRIVATE KEY-----
    MIIEpAIBAAKCAQEA...
    -----END RSA PRIVATE KEY-----
```

**Note:** This is not recommended for production as it exposes the key in environment variables.

### Custom Key Path

To use a different file path:

```yaml
environment:
  SSH_PROXY_HOST_KEY_PATH: /custom/path/to/key
volumes:
  - /host/path/to/key:/custom/path/to/key:ro
```

## Rotating the SSH Host Key

To rotate the key in production:

### 1. Generate New Key

```bash
ssh-keygen -t rsa -b 2048 -f ssh_proxy_host_key_new -N ""
```

### 2. Update Secret

```bash
# Remove old secret
docker secret rm ssh_proxy_host_key

# Create new secret
docker secret create ssh_proxy_host_key ssh_proxy_host_key_new

# Clean up
rm ssh_proxy_host_key_new ssh_proxy_host_key_new.pub
```

### 3. Redeploy Service

```bash
docker service update --force obiente_vps-service
```

**Warning:** Users who have the old host key in their `known_hosts` file will see the "REMOTE HOST IDENTIFICATION HAS CHANGED" warning after rotation. They will need to remove the old key:

```bash
ssh-keygen -R "[your-domain]:2222"
```

## Troubleshooting

### Different Fingerprints Across Replicas

If you see different fingerprints, check:

1. **Secret exists:**
   ```bash
   docker secret ls | grep ssh_proxy_host_key
   ```

2. **Service has secret mounted:**
   ```bash
   docker service inspect obiente_vps-service --format '{{json .Spec.TaskTemplate.ContainerSpec.Secrets}}' | jq
   ```

3. **Logs show secret loading:**
   ```bash
   docker service logs obiente_vps-service | grep "SSH_PROXY_HOST_KEY_FILE"
   ```

### Port Forwarding Still Disabled

If clients still see "Port forwarding is disabled", their `known_hosts` file contains the old fingerprint. They need to:

1. Remove the old key:
   ```bash
   ssh-keygen -R "[your-domain]:2222"
   ```

2. Connect again to accept the new key

### Checking Current Fingerprint

To see what fingerprint clients will see:

```bash
# Get the public key from the secret
docker secret inspect ssh_proxy_host_key --format '{{.Spec.Data}}' | base64 -d > temp_key

# Generate public key
ssh-keygen -y -f temp_key > temp_key.pub

# Show fingerprint
ssh-keygen -lf temp_key.pub

# Clean up
rm temp_key temp_key.pub
```

### X11 Forwarding Not Working

X11 forwarding is fully supported through the SSH proxy with bidirectional channel forwarding. If X11 applications show "Can't open display" errors:

1. **Ensure X11 forwarding is enabled on the VPS:**
   ```bash
   # Check sshd_config on the VPS
   ssh -p 2222 root@vps-xxx@your-domain.com
   grep X11Forwarding /etc/ssh/sshd_config
   ```

   Should show:
   ```
   X11Forwarding yes
   X11DisplayOffset 10
   X11UseLocalhost yes
   ```

2. **Install xauth on the VPS:**
   ```bash
   # Ubuntu/Debian
   apt-get install xauth

   # RHEL/Rocky/Alma
   yum install xorg-x11-xauth
   ```

3. **Ensure your local X server is running:**
   - **Linux/macOS**: X11 should be installed (XQuartz on macOS)
   - **Windows**: Install VcXsrv or Xming

4. **Use the `-X` or `-Y` flag when connecting:**
   ```bash
   ssh -p 2222 -X root@vps-xxx@your-domain.com xeyes
   # or for trusted X11 forwarding (less secure, but faster):
   ssh -p 2222 -Y root@vps-xxx@your-domain.com firefox
   ```

5. **Verify DISPLAY is set on the VPS:**
   ```bash
   echo $DISPLAY
   # Should show something like: localhost:10.0
   ```

6. **Test with a simple X11 application:**
   ```bash
   # Install xeyes if not present
   apt-get install x11-apps
   
   # Test X11 forwarding
   ssh -p 2222 -X root@vps-xxx@your-domain.com xeyes
   ```

### Port Forwarding Not Working

If port forwarding isn't working:

1. **Verify the command syntax:**
   ```bash
   # Local forwarding
   ssh -p 2222 -L local_port:destination:destination_port user@vps
   
   # Remote forwarding
   ssh -p 2222 -R remote_port:localhost:local_port user@vps
   ```

2. **Check if the port is already in use:**
   ```bash
   # On your local machine
   netstat -tuln | grep <port>
   ```

3. **For remote forwarding, check GatewayPorts on VPS:**
   ```bash
   grep GatewayPorts /etc/ssh/sshd_config
   ```

   To allow remote hosts to connect to forwarded ports:
   ```
   GatewayPorts yes
   ```

## Security Considerations

1. **Never commit the private key** to version control
2. **Restrict secret access** to only services that need it
3. **Rotate keys periodically** (e.g., annually)
4. **Monitor for unauthorized changes** to the secret
5. **Use strong key sizes** (minimum 2048-bit RSA, 4096-bit recommended)

## Related Configuration

- **SSH_PROXY_PORT**: Port the SSH proxy listens on (default: 2222)
- **SSH_PROXY_IDLE_TIMEOUT**: Timeout for idle SSH connections
- **SSH_PROXY_MAX_CONN_PER_VPS**: Maximum concurrent connections per VPS

See [VPS Configuration](vps-configuration.md) for complete VPS service configuration.
