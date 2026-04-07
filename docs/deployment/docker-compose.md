# Docker Compose Deployments

Use Docker Compose deployments when your app needs multiple containers that should run together, such as an app plus Redis, workers, or background services.

## What Obiente Expects

Your Compose file should describe the services your app needs. Obiente will run the stack and expose the services you route publicly.

Good Compose candidates:

- app + Redis
- API + worker + queue
- CMS + Redis
- web + scheduler + background processor

## Important Networking Rules

There are two kinds of hostnames in Compose deployments:

### 1. Compose Service Names

Use service names for containers inside the same Compose deployment.

Example:

```yaml
services:
  cache:
    image: redis:7

  app:
    environment:
      REDIS_URL: redis://cache:6379
```

Inside the deployment, `cache` resolves because it is a Compose service name.

### 2. Managed Obiente Service Hostnames

Use Obiente-managed hostnames for resources outside the Compose stack, such as managed databases.

Example:

```yaml
services:
  app:
    environment:
      DB_HOST: db-8e98bdb7f10a47ec.my.obiente.cloud
      DB_PORT: "5432"
```

Do not expect Compose service discovery to resolve managed database names, and do not expect managed database names to behave like local Compose service names.

## A Good Starting Pattern

```yaml
services:
  cache:
    image: redis:7
    restart: always
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  app:
    image: your-image:latest
    restart: always
    depends_on:
      - cache
    environment:
      REDIS_URL: "redis://cache:6379"
      DB_HOST: "db-8e98bdb7f10a47ec.my.obiente.cloud"
      DB_PORT: "5432"
      DB_NAME: "db-8e98bdb7-f10a-47ec-a637-4d6540eeba0f"
      DB_USER: "admin"
      DB_PASSWORD: "your-password"
```

## Use Quoted Environment Values

Quote environment values in Compose YAML, especially:

- booleans
- numbers
- URLs
- secrets containing special characters

Example:

```yaml
environment:
  PORT: "8055"
  CORS_ENABLED: "true"
  REDIS_URL: "redis://cache:6379"
```

This avoids YAML coercion surprises.

## `depends_on` Is Not A Readiness Guarantee

`depends_on` helps startup ordering, but it does not guarantee the dependency is ready to accept traffic.

You should still use:

- health checks on supporting services
- retries in your application
- entrypoint wait logic when the app is sensitive to startup timing

This matters for both:

- Compose-local services like Redis
- external managed services like databases

## Common Failure Modes

### `getaddrinfo ENOTFOUND cache`

Your app is trying to resolve `cache`, but there is no service with that name in the Compose file visible to that container.

Check:

- the service is actually named `cache`
- the app container is in the same Compose deployment
- the app uses the service name, not some stale hostname like `redis` or `valkey`

### `getaddrinfo ENOTFOUND db-....my.obiente.cloud`

The managed database hostname is not resolving from the container.

Check:

- the managed database is running
- Obiente DNS is deployed and healthy
- public DNS delegation for `my.obiente.cloud` is correct in self-hosted setups
- the app container has working DNS resolution

If needed, test inside the container:

```bash
getent hosts db-8e98bdb7f10a47ec.my.obiente.cloud
nslookup db-8e98bdb7f10a47ec.my.obiente.cloud 1.1.1.1
```

### App Starts Before Redis Or Database Is Ready

Symptoms:

- connection refused
- initial crash loops
- intermittent startup success

Fixes:

- add health checks
- make the app retry on startup
- avoid assuming `depends_on` means “ready”

## Managed Databases In Compose Apps

When using an Obiente managed database from a Compose app:

- use the hostname shown in the database connection panel
- use the generated connection string when possible
- remember that passwords are only shown at creation time or after a password reset

If you rotate the password in the dashboard, update the Compose deployment secret or env var too.

## Recommended Checklist

Before shipping a Compose deployment:

- service-to-service URLs use Compose service names
- managed services use Obiente hostnames
- env values are quoted
- health checks are defined
- the app retries dependency connections
- secrets are updated after password resets
- only the intended service is exposed publicly through routing

## Related Docs

- [DNS](dns.md)
- [High Availability](high-availability.md)
- [GitHub Integration](../guides/github-integration.md)
- [Troubleshooting](../guides/troubleshooting.md)
