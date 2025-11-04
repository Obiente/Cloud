# Troubleshooting Guide

Common issues and solutions when using Obiente Cloud.

## Quick Diagnostics

### Check Service Status

```bash
# Docker Compose
docker compose ps

# Docker Swarm
docker service ls
```

### View Logs

```bash
# All compose services
docker compose logs

# Specific service (compose)
docker compose logs postgres

# API logs
# If running API locally: view the terminal running `go run`
# If running API in Swarm:
# docker service logs -f obiente_api
```

## Common Issues

### Services Won't Start

**Problem**: Services crash on startup

**Solution**:

```bash
# Check logs for errors
# If API is local, view the terminal output; if in Swarm, use docker service logs

# Check resource usage
docker stats

# Verify environment variables
cat .env
```

### Database Connection Failed

**Problem**: API can't connect to database

**Solution**:

```bash
# Verify PostgreSQL is running
docker compose ps postgres

# Check PostgreSQL logs
docker compose logs postgres

# Test connection
docker exec -it obiente-postgres psql -U obiente-postgres -d obiente

# Verify TimescaleDB (metrics database) is running
docker compose ps timescaledb

# Check TimescaleDB logs
docker compose logs timescaledb

# Test metrics database connection
docker exec -it obiente-timescaledb psql -U postgres -d obiente_metrics
```

### Authentication Errors

**Problem**: Getting authentication failures

**Solution**:

```bash
# Disable auth for development
echo "DISABLE_AUTH=true" >> .env
# Restart your API process (if running locally) or the Swarm service
# docker service update --force obiente_api

# Check Zitadel configuration
echo $ZITADEL_URL

# Test userinfo endpoint
curl -H "Authorization: Bearer TOKEN" https://auth.obiente.cloud/oidc/v1/userinfo
```

See [Authentication Guide](authentication.md) for detailed troubleshooting.

### CORS Errors

**Problem**: Browser shows "CORS Failed" or "Response body is not available to scripts"

**Common Causes:**

1. Origin mismatch - Browser sends exact origin including port
2. Missing CORS_ORIGIN env variable
3. API not rebuilt after CORS_ORIGIN change

**Solutions:**

1. **Check what origin browser is sending:**

   - Open browser DevTools → Network tab
   - Look at request headers for `Origin: http://localhost:3000`
   - This must exactly match an entry in `CORS_ORIGIN`

2. **Set CORS_ORIGIN with exact origin including port:**

   ```bash
   # For frontend on localhost:3000
   echo "CORS_ORIGIN=http://localhost:3000" >> .env

   # Multiple origins (comma-separated)
   echo "CORS_ORIGIN=http://localhost:3000,https://app.example.com" >> .env
   ```

3. **Rebuild API after changing CORS_ORIGIN:**

   ```bash
   docker compose up -d --build api
   ```

4. **Check CORS logs:**

   ```bash
   docker compose logs api | grep CORS
   ```

   You should see:

   - `[CORS] Origin http://localhost:3000 matched allowed origin`
   - `[CORS] Origin ... NOT in allowed list` (if mismatch)

5. **Temporary: Allow all origins (development only):**
   ```bash
   echo "CORS_ORIGIN=*" >> .env
   docker compose up -d --build api
   ```
   **Warning:** Only use `*` in development. With credentials enabled, the API will echo the origin anyway.

**Example for local development:**

```bash
# .env file
CORS_ORIGIN=http://localhost:3000
PUBLIC_HTTPS_PORT=2443
```

### Port Already in Use

**Problem**: Port 3001 or 5432 already in use

**Solution**:

```bash
# Find process using port
sudo lsof -i :3001

# Kill process
sudo kill -9 <PID>

# Or change port in docker-compose.yml
```

### Out of Disk Space

**Problem**: Docker runs out of disk space

**Solution**:

```bash
# Clean up Docker
docker system prune -a

# Remove unused volumes
docker volume prune

# Remove specific images
docker image prune
```

## Advanced Troubleshooting

### Network Issues

```bash
# Inspect network
docker network inspect obiente-network

# Test connectivity between containers (replace <api-container>)
docker exec -it <api-container> ping postgres
```

### Resource Limits

```bash
# Check container stats
docker stats

# Adjust limits in docker-compose.yml
services:
  postgres:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
```

### Database Performance

```bash
# Check database size
docker exec obiente-postgres psql -U obiente-postgres -c "SELECT pg_database_size('obiente');"

# Analyze tables
docker exec obiente-postgres psql -U obiente-postgres -d obiente -c "VACUUM ANALYZE;"

# Check TimescaleDB size (metrics database)
docker exec obiente-timescaledb psql -U postgres -d obiente_metrics -c "SELECT pg_database_size('obiente_metrics');"
```

### Metrics Collection Issues

**Problem**: Metrics not being collected or displayed

**Solution**:

```bash
# Check metrics system health
curl http://localhost:3001/health

# Check detailed metrics observability
curl http://localhost:3001/metrics/observability | jq

# Look for:
# - "healthy": should be true
# - "circuit_breaker_state": should be 0 (Closed)
# - "consecutive_failures": should be 0
# - "collection_errors": check error rate
```

**Common Issues:**

1. **Circuit Breaker Open (state = 1)**
   - Docker API is having issues
   - Check Docker daemon connectivity
   - Review Docker API permissions
   - Wait for cooldown period or restart API

2. **High Error Rates**
   ```bash
   # Check Docker API connectivity
   docker ps
   
   # Check API logs for errors
   docker service logs obiente_api | grep -i "metrics\|docker"
   ```

3. **Metrics Database Connection Failed**
   ```bash
   # Verify TimescaleDB is running
   docker compose ps timescaledb
   
   # Check connection from API
   docker exec obiente-api psql -h timescaledb -U postgres -d obiente_metrics
   ```

4. **Slow Metrics Collection**
   - Increase `METRICS_MAX_WORKERS` environment variable
   - Check Docker API response times
   - Monitor container count (may need more workers)

See [Monitoring Guide](monitoring.md) for more details.

## Getting Help

If you can't resolve the issue:

1. Check the logs
2. Review [GitHub Issues](https://github.com/obiente/cloud/issues)
3. Create a new issue with details
4. Join [GitHub Discussions](https://github.com/obiente/cloud/discussions)

## Debug Mode

Enable debug logging:

```bash
# Enable application debug logging
echo "LOG_LEVEL=debug" >> .env

# Enable database query logging (optional, falls back to LOG_LEVEL if not set)
echo "DB_LOG_LEVEL=debug" >> .env

# If running API locally, restart your local process
# If running API in Swarm:
# docker service update --force obiente_api
# docker service logs -f obiente_api
```

**Note:** Database logging can be very verbose. Use `DB_LOG_LEVEL=debug` only when debugging database issues. For production, use `DB_LOG_LEVEL=error` to suppress SQL query logs.

---

[← Back to Guides](index.md)
