# Obiente Cloud - Routing & Traffic Management

This guide explains how traffic routing works in Obiente Cloud for user deployments.

## Overview

Obiente Cloud uses a **multi-tier routing system** to direct traffic to user deployments:

1. **Traefik** - Entry point, handles SSL/TLS, service discovery
2. **Deployment Router** - Application-level routing logic
3. **Load Balancing** - Distributes traffic across replicas
4. **Health Checking** - Routes only to healthy instances

## Routing Architecture

```
User Request (app.example.com)
         ↓
    [DNS Resolution]
         ↓
  [Traefik (Port 80/443)]
    - SSL/TLS termination
    - Docker labels discovery
    - Automatic HTTPS
         ↓
  [Deployment Router]
    - Domain → Deployment lookup
    - Load balancing algorithm
    - Health filtering
         ↓
  [Target Container]
    - User's deployed application
    - Running on specific node
    - Port mapping
```

## Routing Flow

### 1. Domain Configuration

When a deployment is created:

```go
// Recorded in database
routing := &DeploymentRouting{
    DeploymentID: "dep_abc123",
    Domain: "app.example.com",
    TargetPort: 3000,
    LoadBalancerAlgo: "round-robin",
    SSLEnabled: true,
}
```

### 2. Container Labels

Containers are created with Traefik labels:

```yaml
labels:
  traefik.enable: "true"
  traefik.http.routers.dep_abc123.rule: "Host(`app.example.com`)"
  traefik.http.routers.dep_abc123.entrypoints: "websecure"
  traefik.http.routers.dep_abc123.tls.certresolver: "letsencrypt"
  traefik.http.services.dep_abc123.loadbalancer.server.port: "3000"
```

### 3. Request Routing

When a request arrives at `https://app.example.com`:

1. **Traefik** receives the request
2. Looks up routing rule by domain
3. Forwards to appropriate service
4. **Deployment Router** handles application logic:
   - Queries database for deployment
   - Gets all active replicas
   - Filters healthy instances
   - Selects target based on load balancing
   - Proxies request to container

## Load Balancing Algorithms

### Round-Robin (Default)

Distributes requests evenly across all instances:

```go
func roundRobin(locations []Location, domain string) Location {
    idx := getNextIndex(domain)
    return locations[idx % len(locations)]
}
```

**Use case**: General purpose, predictable distribution

### Least Connections

Routes to the instance with lowest CPU usage:

```go
func leastConnections(locations []Location) Location {
    minCPU := 100.0
    for _, loc := range locations {
        if loc.CPUUsage < minCPU {
            selected = loc
        }
    }
    return selected
}
```

**Use case**: CPU-intensive applications

### IP Hash

Consistent hashing based on client IP:

```go
func ipHash(locations []Location, clientIP string) Location {
    hash := hash(clientIP)
    return locations[hash % len(locations)]
}
```

**Use case**: Session affinity, stateful applications

## Health-Based Routing

Only healthy instances receive traffic:

```go
healthyInstances := filterHealthy(allInstances)
if len(healthyInstances) == 0 {
    return 503 // Service Unavailable
}
target := selectTarget(healthyInstances, algorithm)
```

Health status is checked:

- Every 30 seconds via `/health` endpoint
- On request failure (marked unhealthy immediately)
- After 3 consecutive successes (marked healthy again)

## Custom Domains

### Setting Up a Custom Domain

1. **User adds domain** via API:

```bash
POST /api/v1/deployments/dep_abc123/domains
{
  "domain": "myapp.com"
}
```

2. **System creates routing**:

```go
routing := &DeploymentRouting{
    DeploymentID: "dep_abc123",
    Domain: "myapp.com",
    TargetPort: 3000,
}
database.UpsertDeploymentRouting(routing)
```

3. **Traefik discovers via labels**:
   The deployment container gets updated labels automatically

4. **User configures DNS**:

```
A    myapp.com           →  <swarm-manager-ip>
CNAME www.myapp.com      →  myapp.com
```

5. **Let's Encrypt certificate** issued automatically

## Wildcard Domains

Support for subdomain routing:

```yaml
traefik.http.routers.app.rule: "Host(`*.example.com`)"
```

Database routing:

```go
routing := &DeploymentRouting{
    Domain: "*.app.example.com",
    PathPrefix: "/",
}
```

All requests to `*.app.example.com` route to the same deployment.

## Path-Based Routing

Route different paths to different deployments:

```yaml
# API deployment
traefik.http.routers.api.rule: "Host(`example.com`) && PathPrefix(`/api`)"

# Frontend deployment
traefik.http.routers.frontend.rule: "Host(`example.com`) && PathPrefix(`/`)"
```

## SSL/TLS Management

### Automatic HTTPS

Traefik + Let's Encrypt provides automatic HTTPS:

1. Client requests `https://app.example.com`
2. Traefik checks if certificate exists
3. If not, initiates ACME challenge
4. Certificate issued and cached
5. Auto-renewal before expiration

### Custom Certificates

For custom SSL certificates:

```bash
# Upload certificate
docker secret create app-cert /path/to/cert.pem
docker secret create app-key /path/to/key.pem

# Configure Traefik
traefik.http.routers.app.tls.certificates[0].certFile: /run/secrets/app-cert
traefik.http.routers.app.tls.certificates[0].keyFile: /run/secrets/app-key
```

## Middleware

Traefik supports middleware for request processing:

### Rate Limiting

```yaml
traefik.http.middlewares.ratelimit.ratelimit.average: "100"
traefik.http.middlewares.ratelimit.ratelimit.burst: "50"
traefik.http.routers.app.middlewares: "ratelimit"
```

### Authentication

```yaml
traefik.http.middlewares.auth.basicauth.users: "user:hashedpassword"
traefik.http.routers.app.middlewares: "auth"
```

### Headers

```yaml
traefik.http.middlewares.headers.headers.customresponseheaders.X-Custom-Header: "value"
traefik.http.routers.app.middlewares: "headers"
```

### Compression

```yaml
traefik.http.middlewares.compress.compress: "true"
traefik.http.routers.app.middlewares: "compress"
```

## Routing API

### Get Deployment Routes

```bash
GET /api/v1/deployments/:id/routing
```

Response:

```json
{
  "deployment_id": "dep_abc123",
  "domain": "app.example.com",
  "target_port": 3000,
  "protocol": "http",
  "load_balancer_algo": "round-robin",
  "ssl_enabled": true,
  "ssl_cert_resolver": "letsencrypt"
}
```

### Update Routing

```bash
PUT /api/v1/deployments/:id/routing
{
  "domain": "newdomain.com",
  "load_balancer_algo": "least-conn"
}
```

### Get Routing Stats

```bash
GET /api/v1/routing/stats
```

Response:

```json
{
  "total_routes": 1523,
  "cached_proxies": 842,
  "timestamp": "2025-10-29T12:00:00Z"
}
```

## Monitoring & Debugging

### Check Traefik Routes

```bash
# View Traefik dashboard
https://traefik.obiente.example.com

# API endpoint
curl https://traefik.obiente.example.com/api/http/routers
```

### Check Deployment Routing

```sql
-- Query database
SELECT deployment_id, domain, target_port, load_balancer_algo
FROM deployment_routing
WHERE deployment_id = 'dep_abc123';
```

### View Request Logs

```bash
# Traefik access logs
docker service logs obiente_traefik | grep "app.example.com"

# Deployment Router logs (Swarm)
docker service logs obiente_api | grep Router
```

### Test Routing

```bash
# Direct request
curl -H "Host: app.example.com" http://<swarm-ip>

# With SSL
curl https://app.example.com
```

## Performance Optimization

### Connection Pooling

```go
proxy.Transport = &http.Transport{
    MaxIdleConns:        100,
    IdleConnTimeout:     90 * time.Second,
    MaxIdleConnsPerHost: 10,
}
```

### Proxy Caching

Reverse proxies are cached per target:

```go
cacheKey := fmt.Sprintf("%s:%d", nodeIP, port)
proxyCache.Store(cacheKey, proxy)
```

### DNS Caching

Traefik caches DNS lookups for service discovery.

## Troubleshooting

### 404 Not Found

**Symptom**: `404 Not Found` for a deployed app

**Checks**:

1. Verify routing exists in database
2. Check Traefik discovered the route
3. Verify DNS points to cluster
4. Check container labels are correct

```bash
# Check routing
docker exec -it $(docker ps -qf name=patroni-1) psql -U obiente -d obiente \
  -c "SELECT * FROM deployment_routing WHERE domain = 'app.example.com';"

# Check Traefik
curl http://traefik.obiente.example.com/api/http/routers
```

### 502 Bad Gateway

**Symptom**: `502 Bad Gateway` error

**Causes**:

1. Container not running
2. Container unhealthy
3. Wrong port mapping
4. Network issues

```bash
# Check container status
docker ps | grep dep_abc123

# Check health
docker exec <api-container> \
  wget -O- http://localhost:3000/health
# Replace <api-container> with your container ID or name, or run the health check directly against your app

# Check logs
docker logs <container-id>
```

### 503 Service Unavailable

**Symptom**: `503 Service Unavailable`

**Causes**:

1. No healthy instances
2. All replicas down
3. Deployment not found

```bash
# Check deployment locations
curl https://api.obiente.example.com/api/v1/deployments/dep_abc123/locations
```

### SSL Certificate Issues

**Symptom**: SSL handshake errors

**Fixes**:

1. Check Let's Encrypt rate limits
2. Verify DNS is correct
3. Check Traefik logs
4. Manually trigger certificate renewal

```bash
docker service logs obiente_traefik | grep acme
```

## Best Practices

1. **Use Health Checks**: Always implement `/health` endpoint
2. **Set Timeouts**: Configure appropriate timeouts for your app
3. **Monitor Metrics**: Track request latency and error rates
4. **Use CDN**: For static assets, use a CDN in front
5. **Load Test**: Test routing under high load before production
6. **Log Analysis**: Aggregate and analyze logs for insights
7. **Gradual Rollouts**: Use multiple replicas for zero-downtime deploys

## Future Enhancements

- **Geographic routing**: Route to nearest data center
- **A/B testing**: Split traffic between versions
- **Canary deployments**: Gradual traffic shifting
- **Circuit breakers**: Automatic failover on errors
- **Request retry**: Automatic retry on failures
- **WebSocket support**: Long-lived connection handling
- **gRPC routing**: Support for gRPC services
