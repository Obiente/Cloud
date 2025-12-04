# Proxmox SSH User Setup for Snippet Writing and Disk Operations

This guide explains how to set up a dedicated SSH user (`obiente-cloud`) on your Proxmox node for writing cloud-init snippet files and performing disk operations (e.g., `qemu-img convert` for LVM thin storage).

## Why a Dedicated User?

Using a dedicated user with minimal permissions is a security best practice:

- **Principle of Least Privilege**: The user only has permissions for required operations (snippet writing and disk operations)
- **Separation of Concerns**: API token permissions are separate from file system permissions
- **Audit Trail**: All operations are traceable to the `obiente-cloud` user
- **Security Isolation**: If the SSH key is compromised, the attacker has limited access scope

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

## Step 3.5: Configure Permissions for Disk Operations

The `obiente-cloud` user needs permissions to execute `qemu-img convert` for disk operations on LVM thin storage. The code automatically constructs disk paths from volume IDs, so no additional tools are required. The user needs access to:
- Read disk images from directory storage (e.g., `/var/lib/vz/images/`)
- Write to LVM volumes (e.g., `/dev/pve/vm-XXX-disk-0`)

**Option A: Add User to Disk Group and Configure Directory Access with ACLs**

This is the recommended approach when sudo is not available. We use both group ownership and ACLs to ensure new files get correct permissions:

```bash
# 1. Install ACL tools if not already installed (required for default permissions)
apt-get update && apt-get install -y acl

# 2. Create a group for vz images access (if it doesn't exist)
groupadd -f vz-images || true

# 3. Add obiente-cloud to both disk and vz-images groups
usermod -a -G disk,vz-images obiente-cloud

# 4. Change group ownership of /var/lib/vz/images to vz-images
# Note: Some template/base images may have immutable attributes and cannot be changed
# Use -f flag to suppress errors for files that cannot be changed
chgrp -Rf vz-images /var/lib/vz/images

# 5. Set group read/write permissions on /var/lib/vz/images
# Note: Template files may fail, but new VM disk images will have correct permissions
chmod -Rf g+rw /var/lib/vz/images

# 6. Set setgid bit so new files inherit the group
find /var/lib/vz/images -type d -exec chmod g+s {} \;

# 7. Set ACLs with default permissions (CRITICAL - ensures new files get group write)
# This overrides Proxmox's umask and ensures new files are created with correct permissions
# -m sets ACLs on existing files/directories, -d sets default ACLs for new files/directories
setfacl -R -m g:vz-images:rwx /var/lib/vz/images
setfacl -R -d -m g:vz-images:rwx /var/lib/vz/images  # Default ACL for new files/directories

# 8. Fix permissions for all existing VM directories (important for VMs created before this setup)
for dir in /var/lib/vz/images/*/; do
    if [ -d "$dir" ] && [[ "$dir" =~ /[0-9]+/$ ]]; then  # Only process VM ID directories
        chgrp vz-images "$dir" 2>/dev/null || true
        chmod g+w "$dir" 2>/dev/null || true
        chmod g+s "$dir" 2>/dev/null || true
        setfacl -m g:vz-images:rwx "$dir" 2>/dev/null || true
        setfacl -d -m g:vz-images:rwx "$dir" 2>/dev/null || true
        chgrp vz-images "$dir"/* 2>/dev/null || true
        chmod g+r "$dir"/* 2>/dev/null || true
        chmod g+w "$dir"/*.raw "$dir"/*.qcow2 2>/dev/null || true
    fi
done

# 9. Verify group membership
groups obiente-cloud
# Should show: obiente-cloud : obiente-cloud disk vz-images

# 10. Verify permissions
ls -ld /var/lib/vz/images
# Should show: drwxrwsr-x or drwxrwxr-x root:vz-images (with setgid bit)
stat -c "%a" /var/lib/vz/images
# Should show a number ending in 2, 3, 6, or 7 (e.g., 2775) indicating setgid is set

# 11. Verify ACLs
getfacl /var/lib/vz/images | grep vz-images
# Should show ACL entries for vz-images group (obiente-cloud user inherits via group membership)
```

**Important**: 
- After adding the user to groups, **restart your API service** so it establishes a new SSH connection with updated group membership. Groups are only applied to new sessions.
- If you see "Operation not permitted" errors when changing group ownership, this is normal for template/base images that may have immutable attributes or be in use. The `-f` flag suppresses these errors.
- **ACLs with default permissions are critical** - they ensure that new files created by Proxmox will have group write permissions, even if Proxmox uses a restrictive umask. Without default ACLs, new VMs may be created with `-rw-r-----` instead of `-rw-rw----`.
- The loop in step 8 fixes permissions for existing VM directories that may have been created before this setup was applied.

**Option B: Direct Ownership (Less Secure, Not Recommended)**

As a last resort, you can change ownership directly (not recommended for production):

```bash
# Change ownership of /var/lib/vz/images to obiente-cloud
chown -R obiente-cloud:obiente-cloud /var/lib/vz/images
```

**Note**: Option A (disk group + vz-images group with ACLs) is the recommended approach as it provides the necessary permissions while maintaining security boundaries and ensures new files are created with correct permissions via default ACLs.

**Testing Permissions**

After configuring permissions, test that the user can access LVM volumes and `qemu-img`:

```bash
# Test qemu-img access (if a test volume exists)
ssh -i obiente-cloud-key obiente-cloud@your-proxmox-node "/usr/bin/qemu-img info /dev/pve/vm-100-disk-0"

# Should output disk information without permission errors

# Test that the user can read LVM volumes
ssh -i obiente-cloud-key obiente-cloud@your-proxmox-node "ls -la /dev/pve/ | head -5"

# Should list LVM volumes without permission errors
```

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

# Test disk operation permissions (if you have a test VM with disk)
# Replace vm-100-disk-0 with an actual volume ID from your Proxmox
ssh -i obiente-cloud-key obiente-cloud@your-proxmox-node "/usr/bin/qemu-img info /dev/pve/vm-100-disk-0"

# Should output a path like: /dev/pve/vm-100-disk-0 (or show permission error if not configured)
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

### Disk Operation Permission Errors

**Error**: `failed to convert disk: permission denied` or `qemu-img: Could not open ... Permission denied`

**Solutions**:
- Verify user is in both `disk` and `vz-images` groups: `groups obiente-cloud` (should include both `disk` and `vz-images`)
- Verify ACLs are set with default permissions: `getfacl /var/lib/vz/images | grep vz-images`
- Check that new VM directories have correct permissions: `ls -la /var/lib/vz/images/VM_ID/` (should show `-rw-rw----` for disk files, not `-rw-r-----`)
- If new VMs are created with wrong permissions, ensure default ACLs are set: `setfacl -R -d -m g:vz-images:rwx /var/lib/vz/images`
- Fix permissions for existing VM directories by running the loop from step 8 in Option A
- Test direct access to LVM volume: `ssh -i key obiente-cloud@host "ls -l /dev/pve/vm-100-disk-0"`
- After adding to groups or changing permissions, **restart your API service** so it establishes a new SSH connection with updated group membership

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

