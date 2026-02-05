# SFTP Service

A secure SFTP data plane microservice with API key authentication, permission scoping, and comprehensive audit logging.

## Features

- **API Key Authentication**: Authenticate using API keys instead of passwords
- **Permission Scoping**: Separate read and write permissions (sftp:read, sftp:write)
- **User Isolation**: Each user has their own isolated directory (organized by org/user)
- **Audit Logging**: All operations are logged to the audit service
- **Secure by Design**: No symlinks, path traversal protection, permission enforcement

## Configuration

Environment variables:

- `SFTP_PORT`: SFTP server port (default: 2222)
- `SFTP_BASE_PATH`: Base directory for SFTP files (default: /var/lib/sftp)
- `SFTP_HOST_KEY_PATH`: Path to SSH host key (default: /var/lib/sftp/host_key)
- `PORT`: HTTP health check port (default: 3020)
- `DATABASE_URL`: PostgreSQL database URL
- `METRICS_DATABASE_URL`: TimescaleDB URL for audit logs
- `LOG_LEVEL`: Logging level (debug, info, warn, error)

## API Key Scopes

API keys must have one or more of these scopes:

- `sftp:read` - Allow reading/downloading files and listing directories
- `sftp:write` - Allow uploading, deleting, and modifying files
- `sftp:*` or `sftp` - Grant both read and write permissions

## Usage

### Creating an API Key

API keys must be created through the auth service or admin interface with the appropriate SFTP scopes.

Example:
```
Scopes: sftp:read,sftp:write
```

### Connecting via SFTP

```bash
# Using command-line SFTP client
sftp -P 2222 -o User=any_username -o IdentityFile=/path/to/key user@hostname

# When prompted for password, enter your API key

# Using FileZilla or other GUI clients
Host: sftp://hostname
Port: 2222
User: any_username (username doesn't matter, authentication is via API key)
Password: your-api-key
```

### Directory Structure

Files are organized by organization and user:

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

Each user can only access their own directory within their organization.

## Operations

All operations are audited and logged:

- **upload**: Upload a file
- **download**: Download a file
- **delete**: Delete a file or directory
- **mkdir**: Create a directory
- **rename**: Rename/move a file
- **list**: List directory contents
- **stat**: Get file information

## Security

- **Path Traversal Protection**: Users cannot escape their directory
- **No Symlinks**: Symlink creation and reading is disabled
- **Permission Enforcement**: Operations are checked against API key scopes
- **Audit Trail**: All operations are logged with user, org, and result
- **API Key Tracking**: Last used timestamp is updated on each connection

## Health Check

HTTP endpoint available at `http://localhost:3020/health`

Returns:
```json
{
  "status": "healthy",
  "service": "sftp-service",
  "timestamp": "2024-01-17T20:00:00Z",
  "details": {
    "sftp_address": "0.0.0.0:2222",
    "base_path": "/var/lib/sftp"
  }
}
```

## Development

```bash
# Build
go build -o sftp-service

# Run
./sftp-service

# Test connection
sftp -P 2222 test@localhost
# Enter your API key when prompted for password
```

## Architecture

The service consists of three main components:

1. **SFTP Server** (`pkg/sftp/server.go`): Handles SSH/SFTP protocol
2. **Auth Validator** (`internal/service/auth.go`): Validates API keys against database
3. **Audit Logger** (`internal/service/auth.go`): Logs operations to TimescaleDB

The server uses the standard Go SSH and SFTP libraries with custom handlers for permission checking and audit logging.
