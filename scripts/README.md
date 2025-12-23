# Obiente Cloud Scripts

This directory contains utility scripts for managing and deploying Obiente Cloud.

## Deployment Scripts

### `setup-ssh-proxy-key.sh`

Generates and creates a Docker secret for the SSH proxy host key in Docker Swarm deployments.

**Purpose:**
- Ensures all vps-service replicas use the same SSH host key
- Prevents "REMOTE HOST IDENTIFICATION HAS CHANGED" warnings for users
- Enables SSH port forwarding (TCP forwarding) to work correctly

**Usage:**

```bash
./scripts/setup-ssh-proxy-key.sh
```

**What it does:**
1. Checks if Docker is in Swarm mode
2. Checks if the secret already exists (offers to rotate if it does)
3. Generates a new 2048-bit RSA host key
4. Creates a Docker secret named `ssh_proxy_host_key`
5. Cleans up temporary files

**After running:**
```bash
# Deploy your stack
docker stack deploy -c docker-compose.swarm.ha.yml obiente

# Verify the key is loaded
docker service logs obiente_vps-service | grep 'SSH host key'
```

**See also:** [SSH Proxy Configuration](../docs/guides/ssh-proxy.md)

---

## Proxmox Setup Scripts

### `setup-proxmox-templates.sh`

Automated script to create or update all VM templates required for VPS provisioning.

**Features:**
- Auto-detects available storage pools
- Detects storage type (Directory, LVM, ZFS)
- Creates or updates all required templates
- Handles disk path formatting automatically
- Verifies disk attachment before template conversion

**Usage:**

**Option 1: Direct execution from GitHub (Recommended)**

Run the script directly from GitHub without cloning the repository:

```bash
curl -fsSL https://raw.githubusercontent.com/obiente/cloud/main/scripts/setup-proxmox-templates.sh | bash
```

**Option 2: Clone and run locally**

```bash
# Clone the repository
git clone https://github.com/obiente/cloud.git
cd cloud

# Run the script
./scripts/setup-proxmox-templates.sh
```

**Prerequisites:**
- Must be run on a Proxmox node (has access to `qm` command)
- `wget` must be installed
- Storage pools must be configured in Proxmox

**What it does:**

1. Checks prerequisites (`qm` command, `wget`)
2. Detects available storage pools that support VM images
3. Prompts you to select a storage pool (with auto-detection)
4. Shows storage type for each pool (Directory, LVM, ZFS)
5. Creates or updates the following templates:
   - `ubuntu-22.04-standard` (VMID 9000)
   - `ubuntu-24.04-standard` (VMID 9001)
   - `debian-12-standard` (VMID 9002)
   - `debian-13-standard` (VMID 9003)
   - `rockylinux-9-standard` (VMID 9004)
   - `almalinux-9-standard` (VMID 9005)

**Example Session:**

```bash
$ ./scripts/setup-proxmox-templates.sh
==========================================
  Proxmox VM Template Setup Script
==========================================

[INFO] Checking prerequisites...
[SUCCESS] All prerequisites met
[INFO] Using node: main
[INFO] Detecting available storage pools on node: main
[INFO] Available storage pools:

  [1] local (Directory (files))
  [2] local-lvm (LVM (block device))
  [3] local-zfs (ZFS (block device))

[INFO] Auto-detected storage: local-lvm (type: lvm)
Use detected storage? [Y/n]: Y

[INFO] Selected storage: local-lvm (type: lvm)

This will create/update the following templates:
  - ubuntu-22.04-standard
  - ubuntu-24.04-standard
  - debian-12-standard
  - debian-13-standard
  - rockylinux-9-standard
  - almalinux-9-standard

Continue? [Y/n]: Y

[INFO] Processing template: ubuntu-22.04-standard (VMID: 9000)
[INFO] Downloading cloud image...
[SUCCESS] Downloaded image: ubuntu-22.04-server-cloudimg-amd64.img
[INFO] Creating VM 9000...
[INFO] Importing disk to storage: local-lvm...
[INFO] Configuring VM...
[INFO] Verifying disk attachment...
[SUCCESS] Disk attached: scsi0: local-lvm:vm-9000-disk-0,size=2362232012
[INFO] Converting VM to template...
[SUCCESS] Template 'ubuntu-22.04-standard' created successfully!

...

==========================================
[SUCCESS] Completed: 6 templates created/updated
==========================================
```

**Updating Existing Templates:**

If templates already exist, the script will:
1. Detect the existing template
2. Prompt you to update it
3. Delete the old template (with purge)
4. Create a new template with the latest cloud image

**Storage Type Handling:**

The script automatically handles different storage types:
- **Directory storage (`local`)**: Uses path format `local:9000/vm-9000-disk-0.qcow2`
- **LVM/ZFS storage**: Uses path format `storage:vm-9000-disk-0`

**Troubleshooting:**

- **"qm command not found"**: This script must be run on a Proxmox node, not from a remote machine
- **"No storage pools found"**: Ensure storage pools are configured in Proxmox (Datacenter → Storage)
- **"Failed to import disk"**: Check storage pool has sufficient space and supports VM images
- **"Disk not attached correctly"**: This usually indicates a storage type mismatch - the script should handle this automatically, but verify your storage configuration

**See Also:**

- [VPS Provisioning Guide](../docs/guides/vps-provisioning.md) - Complete guide on VPS provisioning
- [VPS Configuration Guide](../docs/guides/vps-configuration.md) - Advanced configuration options
