# Database Routing with Protocol-Aware TCP Proxy

## Architecture

Databases-service embeds a protocol-aware TCP proxy that routes client connections to the correct database container. This replaces the previous Traefik SNI-based approach, eliminating the TLS requirement.

```
Client → databases-service:5432 (PostgreSQL)
       → databases-service:3306 (MySQL/MariaDB)
       → databases-service:27017 (MongoDB)
       → databases-service:16XXX (Redis, port-per-instance)
         │
         ├─ PostgreSQL: Parse StartupMessage, extract "database" field
         ├─ MySQL/MariaDB: Send greeting, read HandshakeResponse, extract DB name
         ├─ MongoDB: Parse OP_MSG, extract $db field from BSON
         ├─ Redis: Port-per-instance (no protocol parsing)
         │
         ├─ Route lookup: database name → container IP on overlay network
         └─ Bidirectional TCP forwarding (io.Copy)
```

## How It Works

### 1. Route Registry

An in-memory `RouteRegistry` maps database names (the database ID, e.g. `db-1234567890`) to container IPs on the Docker overlay network.

- Populated on startup from `database_instances` table
- Updated on create/delete via gRPC service methods
- Background goroutine refreshes container IPs every 30s via Docker inspect API

### 2. Protocol Handlers

#### PostgreSQL (port 5432)
1. Client sends `SSLRequest` → proxy responds `N` (no SSL)
2. Client sends `StartupMessage` with `database=db-{id}`
3. Proxy extracts database name, looks up route
4. Connects to backend container, replays StartupMessage
5. Bidirectional TCP forwarding

#### MySQL/MariaDB (port 3306)
1. Proxy sends fake server greeting with auth challenge
2. Client sends `HandshakeResponse41` with database name + credentials
3. Proxy extracts database name, looks up route
4. Connects to backend, reads real greeting
5. Re-authenticates using stored credentials with `mysql_native_password`
6. Forwards auth result to client, then bidirectional forwarding

#### MongoDB (port 27017)
1. Client sends `OP_MSG` with `$db` field in BSON body
2. Proxy extracts `$db`, looks up route
3. Connects to backend, replays original message
4. Bidirectional TCP forwarding

#### Redis (port 16379-16478)
- Each Redis instance gets a dedicated port from range 16379-16478
- Plain TCP forwarding (no protocol parsing needed)
- Port allocated on create, released on delete

### 3. Client Connection

```bash
# PostgreSQL (no SSL required)
psql -h proxy-host -p 5432 -d db-1234567890 -U admin

# MySQL
mysql -h proxy-host -P 3306 -u admin -p db-1234567890

# MongoDB
mongosh "mongodb://admin:pass@proxy-host:27017/db-1234567890"

# Redis (dedicated port)
redis-cli -h proxy-host -p 16379
```

## Benefits

- No TLS requirement for routing (TLS optional)
- No DNS wildcard needed (connect directly to proxy host)
- No Traefik dependency for database traffic
- Protocol-level routing (not SNI-based)
- In-process proxy (no separate service to manage)
- Port-per-instance for Redis (protocol doesn't carry DB name)

## Implementation

### Files

- `apps/databases-service/internal/proxy/proxy.go` - Main proxy server
- `apps/databases-service/internal/proxy/router.go` - Route registry with IP refresh
- `apps/databases-service/internal/proxy/postgres.go` - PostgreSQL protocol parser
- `apps/databases-service/internal/proxy/mysql.go` - MySQL handshake proxy
- `apps/databases-service/internal/proxy/mongodb.go` - MongoDB OP_MSG parser
- `apps/databases-service/internal/proxy/redis.go` - Redis port-per-instance manager

### Docker Compose Ports

Database TCP ports are exposed on the databases-service container (not Traefik):

```yaml
databases-service:
  ports:
    - "5432:5432"      # PostgreSQL
    - "3306:3306"      # MySQL/MariaDB
    - "27017:27017"    # MongoDB
    - "16379-16478:16379-16478"  # Redis instances
```
