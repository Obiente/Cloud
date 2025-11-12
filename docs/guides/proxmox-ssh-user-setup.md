# Proxmox SSH User Setup for Snippet Writing

This guide explains how to set up a dedicated SSH user (`obiente-cloud`) on your Proxmox node for writing cloud-init snippet files.

## Why a Dedicated User?

Using a dedicated user with minimal permissions is a security best practice:

- **Principle of Least Privilege**: The user only has permissions to write snippet files, nothing else
- **Separation of Concerns**: API token permissions are separate from file system permissions
- **Audit Trail**: All snippet writes are traceable to the `obiente-cloud` user
- **Security Isolation**: If the SSH key is compromised, the attacker can only write snippet files, not access VMs or other Proxmox resources

## Prerequisites

- Root or sudo access to your Proxmox node
- SSH access to the Proxmox node
- A directory-type storage pool configured with snippets enabled (e.g., `local` storage)

## Step 1: Create the User

Create a dedicated user for snippet writing:

```bash
# Create the user with /bin/sh shell (non-interactive, allows command execution via SSH)
sudo useradd -r -s /bin/sh -d /var/lib/obiente-cloud obiente-cloud

# Create home directory (optional, but useful for SSH key storage)
sudo mkdir -p /var/lib/obiente-cloud/.ssh
sudo chown obiente-cloud:obiente-cloud /var/lib/obiente-cloud/.ssh
sudo chmod 700 /var/lib/obiente-cloud/.ssh
```

**Note**: The `-r` flag creates a system user (UID < 1000), and `-s /bin/sh` provides a non-interactive shell that allows command execution via SSH. This is secure because the user cannot log in interactively (no password set, key-based auth only) and can only execute commands via SSH with proper authentication.

## Step 2: Determine Snippets Directory Path

Find the path to your snippets directory based on your storage configuration:

```bash
# For 'local' storage (most common)
ls -la /var/lib/vz/snippets

# For other directory storage, check the storage path
# The snippets directory is typically: <storage-path>/snippets/
# You can find storage paths in: Datacenter → Storage → Select storage → Path
```

Common paths:
- **local storage**: `/var/lib/vz/snippets`
- **Other directory storage**: `/var/lib/vz/snippets` or `<storage-path>/snippets/`

## Step 3: Set Up Directory Permissions

Grant the `obiente-cloud` user write access to the snippets directory:

```bash
# Replace with your actual snippets path
SNIPPETS_DIR="/var/lib/vz/snippets"

# Create directory if it doesn't exist
sudo mkdir -p "$SNIPPETS_DIR"

# Set ownership to obiente-cloud user
sudo chown obiente-cloud:obiente-cloud "$SNIPPETS_DIR"

# Set permissions (755 allows read/execute for others, which Proxmox needs)
sudo chmod 755 "$SNIPPETS_DIR"
```

**Important**: The snippets directory must be readable by Proxmox (typically the `www-data` user or `root`), so we use `755` permissions. Files written to this directory will be created with `644` permissions (readable by all, writable by owner).

## Step 4: Generate SSH Key Pair

Generate an SSH key pair for the `obiente-cloud` user:

```bash
# Generate key pair (on your API server or a secure location)
ssh-keygen -t ed25519 -f obiente-cloud-key -N "" -C "obiente-cloud-snippet-writer"

# This creates:
# - obiente-cloud-key (private key) - Keep this SECRET
# - obiente-cloud-key.pub (public key) - Install this on Proxmox
```

**Security Note**: 
- Use `ed25519` keys (recommended) or `rsa` keys (minimum 2048 bits)
- Never share the private key
- Store the private key securely (e.g., in a secrets manager, environment variable, or encrypted file)

## Step 5: Install Public Key on Proxmox

Install the public key in the `obiente-cloud` user's authorized_keys:

```bash
# Copy public key to Proxmox node
scp obiente-cloud-key.pub root@your-proxmox-node:/tmp/

# On Proxmox node, install the key
sudo mkdir -p /var/lib/obiente-cloud/.ssh
sudo cp /tmp/obiente-cloud-key.pub /var/lib/obiente-cloud/.ssh/authorized_keys
sudo chown -R obiente-cloud:obiente-cloud /var/lib/obiente-cloud/.ssh
sudo chmod 700 /var/lib/obiente-cloud/.ssh
sudo chmod 600 /var/lib/obiente-cloud/.ssh/authorized_keys
```

## Step 6: Test SSH Connection

Test that the SSH connection works:

```bash
# Test SSH connection (from your API server)
ssh -i obiente-cloud-key obiente-cloud@your-proxmox-node "whoami"

# Should output: obiente-cloud

# Test writing to snippets directory
ssh -i obiente-cloud-key obiente-cloud@your-proxmox-node "/bin/sh -c 'echo test > /var/lib/vz/snippets/test.txt && cat /var/lib/vz/snippets/test.txt && rm /var/lib/vz/snippets/test.txt'"

# Should output: test
```

## Step 7: Configure Environment Variables

Configure the following environment variables in your API service:

```bash
# Optional: SSH user (defaults to "obiente-cloud" if not set)
PROXMOX_SSH_USER=obiente-cloud

# Optional: SSH host (defaults to PROXMOX_API_URL host if not set)
# Automatically extracts hostname and removes http:///https:// prefixes and ports
PROXMOX_SSH_HOST=your-proxmox-node.example.com
# Or with custom port:
# PROXMOX_SSH_HOST=your-proxmox-node.example.com:2222

# Required: SSH private key (choose one method)
# Method 1: Path to private key file
PROXMOX_SSH_KEY_PATH=/path/to/obiente-cloud-key

# Method 2: Private key content (supports both raw and base64-encoded)
# Raw key:
# PROXMOX_SSH_KEY_DATA="-----BEGIN OPENSSH PRIVATE KEY-----\n..."
# Base64-encoded key (useful for secrets managers):
# PROXMOX_SSH_KEY_DATA="LS0tLS1CRUdJTiBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0K..."
```

**Security Recommendations**:
- Use `PROXMOX_SSH_KEY_PATH` if the key file is stored securely on disk
- Use `PROXMOX_SSH_KEY_DATA` if using a secrets manager (e.g., Kubernetes secrets, HashiCorp Vault)
- Never commit private keys to version control
- Rotate keys periodically (e.g., every 90 days)

## Step 8: Verify Configuration

After configuring the environment variables, restart your API service and verify:

1. Check API logs for successful SSH connections:
   ```
   [ProxmoxClient] Successfully created snippet via SSH: user=local:snippets/vm-300-user-data
   ```

2. Verify snippet files are created:
   ```bash
   # On Proxmox node
   ls -la /var/lib/vz/snippets/
   ```

3. Test VPS creation with cloud-init configuration to ensure snippets are working.

## Troubleshooting

### SSH Connection Fails

**Error**: `failed to connect to Proxmox node via SSH`

**Solutions**:
- Verify SSH host and port are correct
- Check firewall rules allow SSH access from API server
- Verify SSH service is running: `sudo systemctl status ssh`
- Test manual SSH connection: `ssh -i key obiente-cloud@host`

### Permission Denied

**Error**: `failed to write file (exit code non-zero)`

**Solutions**:
- Verify `obiente-cloud` user owns the snippets directory: `ls -la /var/lib/vz/snippets`
- Check directory permissions: `stat /var/lib/vz/snippets`
- Ensure directory is writable: `sudo chmod 755 /var/lib/vz/snippets`

### Wrong Snippets Path

**Error**: Files written but not found by Proxmox

**Solutions**:
- Verify storage path in Proxmox: Datacenter → Storage → Select storage → Path
- Check if snippets directory exists at the expected path
- Review API logs for the detected snippets path: `[ProxmoxClient] Using storage-specific snippets path: ...`

### SSH Key Issues

**Error**: `failed to parse SSH key`

**Solutions**:
- Verify key format (should be OpenSSH format)
- Check key file permissions: `chmod 600 obiente-cloud-key`
- Ensure key is not password-protected (or handle passphrase if needed)
- For `PROXMOX_SSH_KEY_DATA`, ensure newlines are preserved (`\n`)

## Security Best Practices

1. **Use Ed25519 Keys**: More secure and faster than RSA
2. **Rotate Keys Regularly**: Change SSH keys every 90 days
3. **Monitor Access**: Review SSH logs for unauthorized access attempts
4. **Limit Network Access**: Restrict SSH access to API server IPs only (firewall rules)
5. **Use Key-Based Auth Only**: Disable password authentication for `obiente-cloud` user
6. **Audit Logging**: Monitor snippet file creation and modification
7. **Principle of Least Privilege**: User only has write access to snippets directory, nothing else

## Alternative: Using sudo (Not Recommended)

If you prefer using sudo instead of direct file ownership, you can configure sudo rules:

```bash
# Allow obiente-cloud to write to snippets directory via sudo
echo "obiente-cloud ALL=(ALL) NOPASSWD: /bin/mkdir -p /var/lib/vz/snippets, /bin/cat > /var/lib/vz/snippets/*, /bin/chmod 644 /var/lib/vz/snippets/*" | sudo tee /etc/sudoers.d/obiente-cloud
```

However, this is **not recommended** because:
- Requires sudo access (broader permissions)
- More complex to audit
- Potential security risk if sudo rules are misconfigured

Direct file ownership (as described in this guide) is the preferred approach.

## Related Documentation

- [VPS Provisioning Guide](./vps-provisioning.md) - General VPS setup
- [Environment Variables Reference](../reference/environment-variables.md) - All environment variables
- [VPS Configuration Guide](./vps-configuration.md) - Cloud-init configuration

