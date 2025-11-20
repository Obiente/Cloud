# DNS Service

Microservice for DNS resolution of deployments and game servers.

## Features

- DNS record resolution for `my.obiente.cloud` zone
- A record handling for deployments and game servers
- SRV record handling for game servers
- Delegated DNS record support
- Redis caching for performance

## Port

Default port: `53` (UDP and TCP)

## Environment Variables

See shared configuration in `docker-compose.yml` for common variables.

### Service-Specific Variables

- `NODE_IPS` - Node IP addresses by region (required, used for DNS resolution of deployments and game servers)
- `DNS_IPS` - DNS server IP addresses (optional, for documentation)
- `DNS_PORT` - DNS server port (default: 53)
- `REDIS_URL` - Redis connection URL (for caching)

## Endpoints

- DNS queries on port 53 (UDP/TCP)
- Handles queries for `*.my.obiente.cloud` domain

## Dependencies

- PostgreSQL (main database)
- Redis (for caching)

## Notes

- Requires `NET_BIND_SERVICE` capability to bind to port 53
- Must be accessible on port 53 for DNS queries
- Caches DNS responses for 60 seconds

