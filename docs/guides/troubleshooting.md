# Troubleshooting Guide

Common issues and solutions when using Obiente Cloud.

## Quick Diagnostics

### Check Service Status

```bash
# Docker Compose
docker-compose ps

# Docker Swarm
docker service ls
```

### View Logs

```bash
# All compose services
docker-compose logs

# Specific service (compose)
docker-compose logs postgres

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
docker-compose ps postgres

# Check PostgreSQL logs
docker-compose logs postgres

# Test connection
docker exec -it obiente-postgres psql -U obiente-postgres -d obiente
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

**Problem**: Browser CORS errors

**Solution**:
```bash
# Allow all origins (development only)
echo "CORS_ORIGIN=*" >> .env
# Restart your API process (if running locally) or the Swarm service
# docker service update --force obiente_api

# Or specify exact origin
echo "CORS_ORIGIN=https://example.com" >> .env
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
```

## Getting Help

If you can't resolve the issue:

1. Check the logs
2. Review [GitHub Issues](https://github.com/obiente/cloud/issues)
3. Create a new issue with details
4. Join [GitHub Discussions](https://github.com/obiente/cloud/discussions)

## Debug Mode

Enable debug logging:

```bash
echo "LOG_LEVEL=debug" >> .env
# If running API locally, restart your local process
# If running API in Swarm:
# docker service update --force obiente_api
# docker service logs -f obiente_api
```

---

[← Back to Guides](index.md)

