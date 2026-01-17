# SFTP Service Implementation Summary

## Overview

Successfully implemented a complete SFTP data plane microservice for the Obiente Cloud platform. The service provides secure SFTP file transfer capabilities with API key authentication, scoped permissions, comprehensive audit logging, and organization-based isolation.

## Key Features

### 1. Security
- **API Key Authentication**: No password-based auth; all authentication via API keys
- **SHA-256 Hashing**: API keys are hashed before storage and comparison
- **Scoped Permissions**: Granular control with `sftp:read`, `sftp:write`, `sftp:*`, and `sftp` scopes
- **Organization Isolation**: Files organized by org/user, preventing cross-org access
- **Path Traversal Protection**: All paths validated to stay within user's directory
- **No Symlinks**: Symlink operations disabled for security

### 2. Architecture
- **Microservice Pattern**: Follows existing patterns (audit-service, auth-service, etc.)
- **Graceful Shutdown**: Proper cleanup of SFTP and HTTP servers
- **Health Checks**: HTTP endpoint on port 3020 for monitoring
- **Database Integration**: PostgreSQL for API key storage
- **Audit Logging**: TimescaleDB for operation logs

### 3. Operations
- **Read Operations**: Download, list, stat (requires `sftp:read`)
- **Write Operations**: Upload, delete, mkdir, rename (requires `sftp:write`)
- **Wildcard Scopes**: `sftp:*` and `sftp` grant both read and write
- **Audit Trail**: All operations logged with user, org, path, and result

## Files Created

### Core Package (`apps/shared/pkg/sftp/`)
- `server.go`: SFTP server implementation with SSH integration
- `handler.go`: File operation handlers with permission checking

### Microservice (`apps/sftp-service/`)
- `main.go`: Service entry point with HTTP and SFTP servers
- `internal/service/auth.go`: API key validator and audit logger
- `Dockerfile`: Multi-stage build following existing patterns
- `README.md`: Comprehensive documentation
- `go.mod`: Dependency management

### Database (`apps/shared/pkg/database/`)
- `api_keys.go`: APIKey model with GORM annotations

### Infrastructure
- `migrations/001_create_api_keys_table.sql`: Database schema
- `scripts/create-api-key.sh`: Utility for creating API keys
- Updated `docker-compose.yml`: Local development config
- Updated `docker-compose.swarm.yml`: Production swarm config
- Updated `go.work`: Workspace configuration

## Configuration

### Environment Variables
- `SFTP_PORT`: SFTP server port (default: 2222)
- `SFTP_BASE_PATH`: Base directory for files (default: /var/lib/sftp)
- `SFTP_HOST_KEY_PATH`: SSH host key location (default: /var/lib/sftp/host_key)
- `PORT`: HTTP health check port (default: 3020)
- Standard database and auth variables from existing services

### Docker
- **Port 2222**: SFTP access
- **Port 3020**: HTTP health checks
- **Volume**: `sftp-data` for persistent storage
- **Networks**: `obiente-network` overlay

## API Key Scopes

| Scope | Read | Write | Description |
|-------|------|-------|-------------|
| `sftp:read` | ✅ | ❌ | Download and list files |
| `sftp:write` | ❌ | ✅ | Upload, delete, modify files |
| `sftp:*` | ✅ | ✅ | Full access |
| `sftp` | ✅ | ✅ | Full access |

## Directory Structure

Files are organized as:
```
/var/lib/sftp/
  ├── org-123/
  │   ├── user-456/
  │   │   ├── file1.txt
  │   │   └── subdir/
  │   └── user-789/
  │       └── file2.txt
  └── org-abc/
      └── user-def/
          └── file3.txt
```

## Usage Examples

### Creating an API Key
```bash
./apps/sftp-service/scripts/create-api-key.sh \
  "My SFTP Key" \
  user-123 \
  org-456 \
  "sftp:read,sftp:write"
```

### Connecting via SFTP
```bash
# Using command-line client
sftp -P 2222 user@hostname
# When prompted for password, enter your API key

# Using FileZilla
Host: sftp://hostname
Port: 2222
User: any_username
Password: your-api-key
```

### Testing Health
```bash
curl http://localhost:3020/health
```

## Audit Logging

All operations are logged to TimescaleDB with:
- User ID and Organization ID
- Operation type (upload, download, delete, etc.)
- File path
- Success/failure status
- Bytes transferred
- Timestamp

## Deployment

### Docker Compose (Development)
```bash
docker compose up -d sftp-service
```

### Docker Swarm (Production)
```bash
docker stack deploy -c docker-compose.swarm.yml obiente
```

## Security Considerations

1. **API Keys**: 
   - Stored as SHA-256 hashes
   - Never logged in plain text
   - Include expiration and revocation support

2. **File Isolation**:
   - Each user restricted to their directory
   - Path traversal attempts blocked
   - No symlink support

3. **Audit Trail**:
   - All operations logged
   - Failed attempts recorded
   - User and organization tracked

4. **Network Security**:
   - SSH protocol for transport encryption
   - Host key verification
   - No password fallback

## Code Review Fixes

1. ✅ Replaced custom string splitting with `strings.Split`
2. ✅ Implemented SHA-256 hashing for API keys
3. ✅ Fixed wildcard scope permissions (`sftp:*` and `sftp` now grant both read/write)

## Testing Checklist

- [x] Service builds successfully
- [x] Code review completed and issues fixed
- [x] Security scan passed (no CodeQL issues)
- [ ] Manual testing with SFTP client
- [ ] API key authentication verification
- [ ] Permission scope enforcement testing
- [ ] Audit log verification
- [ ] Cross-org isolation testing

## Future Enhancements

1. **Rate Limiting**: Implement per-user/org rate limits
2. **Quota Management**: Enforce storage quotas per user/org
3. **Web UI**: Admin interface for API key management
4. **Metrics**: Prometheus metrics for SFTP operations
5. **WebDAV**: Add WebDAV support for web-based file access

## References

- SFTP Protocol: RFC 4251-4254
- Go SSH Package: golang.org/x/crypto/ssh
- Go SFTP Package: github.com/pkg/sftp
- Existing Microservices: audit-service, auth-service, etc.
