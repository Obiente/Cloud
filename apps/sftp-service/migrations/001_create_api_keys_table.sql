-- Migration to add api_keys table for SFTP service
-- This table stores API keys for authentication with scoped permissions

CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    key_hash TEXT UNIQUE NOT NULL,
    user_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    scopes TEXT NOT NULL, -- Comma-separated scopes (e.g., "sftp:read,sftp:write")
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    -- Foreign key to organizations table
    CONSTRAINT fk_organization
        FOREIGN KEY (organization_id)
        REFERENCES organizations(id)
        ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_organization_id ON api_keys(organization_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_deleted_at ON api_keys(deleted_at);

-- Comments for documentation
COMMENT ON TABLE api_keys IS 'API keys for service authentication with scoped permissions';
COMMENT ON COLUMN api_keys.key_hash IS 'SHA-256 hash of the API key';
COMMENT ON COLUMN api_keys.scopes IS 'Comma-separated list of permission scopes (e.g., sftp:read, sftp:write, sftp:*)';
COMMENT ON COLUMN api_keys.last_used_at IS 'Timestamp of last successful authentication';
COMMENT ON COLUMN api_keys.expires_at IS 'Optional expiration timestamp for the key';
COMMENT ON COLUMN api_keys.revoked_at IS 'Timestamp when the key was revoked (if revoked)';

-- Example scopes:
-- sftp:read     - Read-only access to SFTP (download, list)
-- sftp:write    - Write access to SFTP (upload, delete, mkdir)
-- sftp:*        - Full SFTP access (read + write)
-- sftp          - Full SFTP access (read + write)
