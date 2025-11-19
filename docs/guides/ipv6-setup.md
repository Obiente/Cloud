# IPv6 Setup for Traefik and Docker Swarm

This guide explains how to enable IPv6 support for Traefik in Docker Swarm, which is required for Let's Encrypt ACME validation via IPv6.

## Prerequisites

1. **Docker must have IPv6 enabled** in the daemon configuration
2. **Host must have IPv6 connectivity** configured
3. **Firewall must allow IPv6 traffic** on ports 80 and 443

## Step 1: Enable IPv6 in Docker Daemon

Edit `/etc/docker/daemon.json` (create if it doesn't exist):

```json
{
  "ipv6": true,
  "fixed-cidr-v6": "fd00:0b1e:c10d::/64"
}
```

**Note**: This uses the Obiente Cloud IPv6 subnet `fd00:0b1e:c10d::/64` (ULA - Unique Local Address range).

Restart Docker:
```bash
sudo systemctl restart docker
```

## Step 2: Verify Docker IPv6 Support

```bash
docker info | grep -i ipv6
```

You should see IPv6-related information if IPv6 is enabled.

## Step 3: Configure Docker Swarm Network

The `docker-compose.swarm.yml` file has been configured with:

```yaml
networks:
  obiente-network:
    ipam:
      config:
        - subnet: 10.15.3.0/24  # IPv4
        - subnet: fd00:0b1e:c10d::/64  # IPv6 for Obiente Cloud
```

**Note**: 
- IPv6 is automatically enabled when you include an IPv6 subnet in the `ipam.config` section
- The IPv6 subnet (`fd00:0b1e:c10d::/64`) must match the range configured in Docker daemon (`/etc/docker/daemon.json`)

## Step 4: Traefik Configuration

Traefik entryPoints are configured to listen on both IPv4 and IPv6:

```yaml
- --entryPoints.web.address=:80
- --entryPoints.websecure.address=:443
```

The `:port` notation binds to all interfaces (IPv4 and IPv6) when IPv6 is enabled in Docker. Traefik automatically handles both protocols when Docker has IPv6 support enabled.

## Step 5: Verify IPv6 Connectivity

### Test IPv6 connectivity from outside:

```bash
# Test IPv6 HTTP connectivity
curl -6 -v http://[2602:f9ab:4:1b00::3]/.well-known/acme-challenge/test

# Test IPv6 HTTPS connectivity  
curl -6 -v https://[2602:f9ab:4:1b00::3]/.well-known/acme-challenge/test
```

### Test from Let's Encrypt's perspective:

```bash
# Use Let's Encrypt's test tool or check DNS
dig +short audit-service.obiente.cloud AAAA
```

## Step 6: Firewall Configuration

Ensure your firewall allows IPv6 traffic on ports 80 and 443:

### UFW (Ubuntu):
```bash
sudo ufw allow 80/tcp comment "HTTP IPv6"
sudo ufw allow 443/tcp comment "HTTPS IPv6"
```

### iptables (IPv6):
```bash
sudo ip6tables -A INPUT -p tcp --dport 80 -j ACCEPT
sudo ip6tables -A INPUT -p tcp --dport 443 -j ACCEPT
```

## Troubleshooting

### Issue: IPv6 connectivity fails

1. **Check Docker IPv6 status**:
   ```bash
   docker info | grep -i ipv6
   ```

2. **Check network IPv6 configuration**:
   ```bash
   docker network inspect obiente_obiente-network | grep -i ipv6
   ```

3. **Check host IPv6 connectivity**:
   ```bash
   ping6 -c 3 2001:4860:4860::8888  # Google IPv6 DNS
   ```

4. **Check Traefik logs**:
   ```bash
   docker service logs obiente_traefik | grep -i ipv6
   ```

### Issue: Let's Encrypt still times out on IPv6

1. **Verify DNS AAAA records**:
   ```bash
   dig +short audit-service.obiente.cloud AAAA
   ```

2. **Test IPv6 connectivity from external tool**:
   Use online IPv6 connectivity testers to verify your IPv6 address is reachable from the internet.

3. **Check if IPv6 is properly routed**:
   Ensure your hosting provider/network has IPv6 properly configured and routed.

## Important Notes

- **IPv6 subnet**: The IPv6 subnet in `docker-compose.swarm.yml` must match the subnet configured in Docker daemon (`/etc/docker/daemon.json`)
- **Dual-stack**: When IPv6 is enabled, Traefik will accept both IPv4 and IPv6 connections
- **Let's Encrypt**: Let's Encrypt will try both IPv4 and IPv6. If IPv6 times out, it will fall back to IPv4, but this may cause delays
- **Network recreation**: If you enable IPv6 on an existing network, you may need to recreate the network (this will cause brief downtime)

## Alternative: Disable IPv6 AAAA Records

If IPv6 connectivity cannot be properly configured, you can remove AAAA records from DNS to force Let's Encrypt to use IPv4 only. However, this is not recommended as IPv6 is becoming increasingly important.

