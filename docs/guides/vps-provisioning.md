# VPS Provisioning Guide

This guide explains how to provision and manage Virtual Private Server (VPS) instances on Obiente Cloud.

## Overview

VPS instances provide dedicated virtual machines with full root access and complete control over the operating system. VPS instances are provisioned via Proxmox and can be accessed through:

- **Web Terminal**: Browser-based terminal access via WebSocket
- **SSH Proxy**: SSH access through gateway (no dedicated IP required)

## Prerequisites

### For Self-Hosters

To enable VPS provisioning, you need:

1. **Proxmox VE** installed and configured
2. **Proxmox API Access** configured with appropriate credentials
3. **Environment Variables** set in your deployment

### Required Environment Variables

```bash
# Proxmox API Configuration
PROXMOX_API_URL=https://your-proxmox-server:8006
PROXMOX_USERNAME=root@pam
PROXMOX_PASSWORD=your-password

# Or use API token (recommended for production)
PROXMOX_TOKEN_ID=your-token-id
PROXMOX_TOKEN_SECRET=your-token-secret

# Optional: Storage pool (defaults to local-lvm)
PROXMOX_STORAGE_POOL=local-lvm

# SSH Proxy Configuration (optional)
SSH_PROXY_PORT=2222
SSH_PROXY_HOST_KEY_PATH=/var/lib/obiente/ssh_proxy_host_key

# VPS Gateway Configuration (optional, for DHCP and SSH proxying)
# If set, enables centralized DHCP management and SSH proxying via dedicated gateway service
VPS_GATEWAY_API_SECRET=your-shared-secret  # Must match GATEWAY_API_SECRET in vps-gateway
VPS_GATEWAY_BRIDGE=OCvpsnet  # SDN bridge name for gateway network
```

## Setting Up Proxmox

### 1. Install Proxmox VE

Follow the [official Proxmox installation guide](https://pve.proxmox.com/pve-docs/pve-admin-guide.html#_installation) to install Proxmox VE on your server.

### 2. Create API Token (Recommended)

For production use, create an API token instead of using password authentication:

1. Log into Proxmox web interface
2. Go to **Datacenter** → **Permissions** → **API Tokens**
3. Click **Add** → **API Token**
4. Configure:
   - **User**: `root@pam` (or create a dedicated user)
   - **Token ID**: `obiente-cloud` (or your preferred name)
   - **Privilege Separation**: Enable if using a non-root user
5. Save the **Token Secret** securely

### 3. Configure API Token Permissions

The API token requires specific permissions to create and manage VMs. After creating the token, configure permissions:

1. Go to **Datacenter** → **Permissions** → **Users**
2. Select the user (e.g., `root@pam`) or create a dedicated user
3. Go to **Permissions** tab
4. Add the following permissions:

**Required Permissions:**

- **VM.Allocate** - Create new VMs
- **VM.Clone** - Clone VM templates (if using templates)
- **VM.Config.Disk** - Configure VM disk storage
- **VM.Config.Network** - Configure VM network settings
- **VM.Config.Options** - Configure VM options (cloud-init, etc.)
- **VM.Config.CPU** - Configure VM CPU settings
- **VM.Config.Memory** - Configure VM memory settings
- **VM.PowerMgmt** - Start, stop, reboot VMs
- **VM.Monitor** - Monitor VM status and metrics
- **Datastore.Allocate** - Allocate storage for VMs
- **Datastore.AllocateSpace** - Allocate disk space
- **Sys.Audit** - Read system information (optional, for node listing)

**Quick Setup (Full Access):**

For testing or if you want full access, you can grant the **Administrator** role at the **Datacenter** level:

1. Go to **Datacenter** → **Permissions**
2. Click **Add** → **Permission**
3. Select:
   - **User/Token**: Your API token user
   - **Role**: `Administrator`
   - **Path**: `/` (entire datacenter)
4. Click **Add**

**Minimal Permissions (Production):**

For production with minimal permissions, grant only the required permissions at the datacenter level:

1. Go to **Datacenter** → **Permissions**
2. Click **Add** → **Permission**
3. Select:
   - **User/Token**: Your API token user
   - **Path**: `/`
4. Check only the required permissions listed above
5. Click **Add**

**Note:** If you're using a non-root user, you may need to grant permissions on specific paths (nodes, storage pools) instead of the entire datacenter.

### 4. Configure VPS Gateway (Required for SSH Proxy)

The SSH proxy allows users to connect to VPS instances even without public IP addresses. The gateway handles SSH proxying, eliminating the need for jump hosts.

**Why This Is Needed:**

- The SSH proxy allows users to connect to VPS instances even without public IP addresses
- The gateway handles routing and proxying of SSH connections
- No SSH keys or jump host configuration is required - the gateway manages connections automatically
- The gateway provides better security and isolation than jump hosts

**Setup Steps:**

1. **Set up the VPS Gateway**:

   Follow the [VPS Gateway Setup Guide](./vps-gateway-setup.md) to configure the gateway service.

2. **Configure Gateway Environment Variables**:

   Add to your `docker-compose.yml` or environment:

   ```bash
   VPS_GATEWAY_URL=http://gateway-public-ip:1537  # Gateway gRPC server URL
   VPS_GATEWAY_API_SECRET=your-shared-secret       # Must match GATEWAY_API_SECRET in gateway
   ```

3. **Verify Gateway Connection**:

   The API will automatically use the gateway for SSH proxying when `VPS_GATEWAY_URL` is configured. No additional SSH key setup is required.

**Verification:**

Test the SSH connection from the API container:

Replace `your-proxmox-host` with your actual Proxmox hostname or IP address:

```bash
# If using SSH agent
docker compose exec api ssh -o StrictHostKeyChecking=no root@your-proxmox-host "echo 'SSH connection successful'"

# Or test from host
ssh -o StrictHostKeyChecking=no root@your-proxmox-host "echo 'SSH connection successful'"
```

**Troubleshooting:**

- **"unable to authenticate"**: SSH key is not properly configured or not accessible
- **"Connection refused"**: Proxmox node SSH service is not running or firewall is blocking
- **"Host key verification failed"**: Add Proxmox host to known_hosts or disable strict checking

### 6. Download ISO Images for VPS Provisioning

For VPS provisioning, you need to download ISO images to Proxmox's ISO storage. The system will use these ISO files when templates are not available.

**Two Options:**

1. **Templates (Recommended - Faster):** Create VM templates with cloud-init support (see "Creating VM Templates" section below)
2. **ISO Files (Fallback):** Download ISO images to Proxmox ISO storage for manual installation

**Download ISO Files to Proxmox:**

You can download ISO files directly through the Proxmox web interface or via command line:

#### Via Proxmox Web Interface

1. Log into Proxmox web interface (e.g., `https://main.obiente.cloud:8006`)
2. Navigate to **Datacenter** → **Storage** → Select your ISO storage (e.g., `local`)
3. Click **ISO Images** tab
4. Click **Upload** or **Download from URL**
5. Enter the ISO download URL (see URLs below)
6. Wait for download to complete

#### Via Command Line (SSH to Proxmox Node)

Find your ISO storage path first:

```bash
# List storage pools and find ISO storage
pvesm status

# Common ISO storage paths:
# - /var/lib/vz/template/iso (for 'local' storage)
# - Or check in Proxmox web UI: Storage → local → Content → ISO images
```

Then download ISO files:

#### Required ISO Files

The following ISO files are needed for VPS provisioning when templates are not available. Download them to your Proxmox ISO storage:

**Ubuntu 22.04 LTS:**

- **URL:** `https://releases.ubuntu.com/22.04/ubuntu-22.04.6-live-server-amd64.iso`
- **Expected filename in Proxmox:** `ubuntu-22.04-server-amd64.iso`
- **Rename after download if needed**

**Ubuntu 24.04 LTS:**

- **URL:** `https://releases.ubuntu.com/24.04/ubuntu-24.04.1-live-server-amd64.iso`
- **Expected filename in Proxmox:** `ubuntu-24.04-server-amd64.iso`
- **Rename after download if needed**

**Debian 12:**

- **URL:** `https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-12.x.x-amd64-netinst.iso`
- **Expected filename in Proxmox:** `debian-12-netinst-amd64.iso`
- **Note:** Replace `x.x` with the latest version number from [Debian downloads](https://www.debian.org/CD/http-ftp/)

**Debian 13:**

- **URL:** `https://cdimage.debian.org/cdimage/daily-builds/daily/iso-cd/debian-testing-amd64-netinst.iso`
- **Expected filename in Proxmox:** `debian-13-netinst-amd64.iso`
- **Note:** Debian 13 (Trixie) may still be in testing - check [Debian daily builds](https://cdimage.debian.org/cdimage/daily-builds/)

**Rocky Linux 9:**

- **URL:** `https://download.rockylinux.org/pub/rocky/9/isos/x86_64/Rocky-9.x-x86_64-minimal.iso`
- **Expected filename in Proxmox:** `Rocky-9-x86_64-minimal.iso`
- **Note:** Replace version numbers with latest from [Rocky Linux downloads](https://rockylinux.org/download)

**AlmaLinux 9:**

- **URL:** `https://repo.almalinux.org/almalinux/9/isos/x86_64/AlmaLinux-9.x-x86_64-minimal.iso`
- **Expected filename in Proxmox:** `AlmaLinux-9-x86_64-minimal.iso`
- **Note:** Replace version numbers with latest from [AlmaLinux downloads](https://almalinux.org/download)

**Quick Download via Proxmox Web UI:**

1. Go to **Datacenter** → **Storage** → Select your ISO storage (usually `local`)
2. Click **ISO Images** tab
3. Click **Download from URL**
4. Paste the ISO URL above
5. Click **Download**
6. Rename the file if needed to match the expected filename

**Note:** ISO installation is slower than template-based provisioning, but it works without pre-configured templates.

**Example: Download Ubuntu 22.04 ISO via Proxmox Web UI**

1. Go to `https://your-proxmox-host:8006` (or your Proxmox URL)
2. Navigate to **Datacenter** → **Storage** → `local` (or your ISO storage)
3. Click **ISO Images** tab
4. Click **Download from URL**
5. Enter: `https://releases.ubuntu.com/22.04/ubuntu-22.04.6-live-server-amd64.iso`
6. Click **Download**
7. After download completes, rename the file to `ubuntu-22.04-server-amd64.iso` if needed

Repeat this process for all the ISO files listed above.

---

### 7. Creating VM Templates (Optional - For Faster Provisioning)

**Note:** This section is optional. If you only download ISO files (section 4), VPS provisioning will work but will be slower. Templates allow VMs to be provisioned in seconds rather than minutes.

**How Templates Work:**

1. Create a VM with the cloud image
2. Configure it with cloud-init support
3. Convert the VM to a template (this makes it read-only and reusable)
4. Future VMs are cloned from this template (fast provisioning)

**Important:** Template names must match exactly:

- `ubuntu-22.04-standard`
- `ubuntu-24.04-standard`
- `debian-12-standard`
- `debian-13-standard`
- `rockylinux-9-standard`
- `almalinux-9-standard`

#### Quick Setup (Recommended)

**Automated Template Setup Script:**

The easiest way to set up all templates is using the provided setup script. This script will:
- Auto-detect available storage pools
- Detect storage type (directory vs LVM/ZFS)
- Create or update all templates
- Handle disk path formatting automatically

**Prerequisites:**
- Access to Proxmox node (SSH or direct console)
- `wget` installed
- Storage pool configured in Proxmox

**Usage:**

**Option 1: Direct execution from GitHub (Recommended)**

Run the script directly from GitHub without cloning the repository:

```bash
curl -fsSL https://raw.githubusercontent.com/obiente/cloud/main/scripts/setup-proxmox-templates.sh | bash
```

This is the recommended method as it:
- Works from any machine with internet access
- Always uses the latest version
- No need to clone the entire repository

**Option 2: Clone and run locally**

If you prefer to clone the repository first:

```bash
git clone https://github.com/obiente/cloud.git
cd cloud
./scripts/setup-proxmox-templates.sh
```

**After running the script:**

1. **Follow the prompts**:
   - The script will auto-detect available storage pools
   - It will suggest a storage pool based on common defaults
   - You can accept the suggestion or choose a different one
   - The script will show storage types (Directory, LVM, ZFS) for each option

2. **Wait for completion**:
   - The script downloads cloud images (this may take a while)
   - Creates/updates each template
   - Verifies disk attachment before converting to template

**Example Output:**
```
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
```

**Manual Setup (Alternative)**

If you prefer to set up templates manually, see the sections below for step-by-step instructions for each operating system.

#### Ubuntu 22.04 LTS Template

```bash
# Step 1: Download Ubuntu 22.04 cloud image
cd /tmp
wget https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04-server-cloudimg-amd64.img

# Step 2: Create a VM (we'll convert it to a template later)
# Replace 'local-lvm' with your storage pool name (e.g., 'local', 'local-zfs')
STORAGE="local-lvm"  # Change this to your storage pool
qm create 9000 --name ubuntu-22.04-standard --memory 2048 --net0 virtio,bridge=vmbr0

# Step 3: Import the cloud image disk
qm importdisk 9000 ubuntu-22.04-server-cloudimg-amd64.img $STORAGE

# Step 4: Determine the correct disk path based on storage type
# For directory storage (local), the path includes the vmID subdirectory
# For LVM/ZFS storage, the path is simpler
if [ "$STORAGE" = "local" ]; then
    # Directory storage: path is storage:vmID/vm-XXX-disk-0.qcow2
    DISK_PATH="$STORAGE:9000/vm-9000-disk-0.qcow2"
else
    # LVM/ZFS storage: path is storage:vm-XXX-disk-0
    DISK_PATH="$STORAGE:vm-9000-disk-0"
fi

# Step 5: Configure the VM with cloud-init support
# IMPORTANT: Set scsi0 BEFORE setting bootdisk, otherwise bootdisk will be 0 B
qm set 9000 --scsihw virtio-scsi-pci --scsi0 $DISK_PATH
qm set 9000 --ide2 $STORAGE:cloudinit
qm set 9000 --boot c --bootdisk scsi0
qm set 9000 --serial0 socket --vga serial0
qm set 9000 --agent enabled=1

# Step 6: Verify the disk is attached correctly
# Check that scsi0 shows a size > 0
qm config 9000 | grep scsi0
# You should see: scsi0: local:9000/vm-9000-disk-0.qcow2 (or similar with size)

# Step 7: Convert the VM to a template (this makes it reusable)
qm template 9000

# Step 8: Clean up downloaded image
rm ubuntu-22.04-server-cloudimg-amd64.img
```

**Note:** The `qm create` command creates a VM, which is then configured and converted to a template. This is the standard Proxmox workflow - you cannot create a template directly without first creating a VM.

#### Ubuntu 24.04 LTS Template

```bash
# Step 1: Download Ubuntu 24.04 cloud image
cd /tmp
wget https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-amd64.img

# Step 2: Create VM and configure (replace 'local-lvm' with your storage pool)
STORAGE="local-lvm"  # Change this to your storage pool
qm create 9001 --name ubuntu-24.04-standard --memory 2048 --net0 virtio,bridge=vmbr0
qm importdisk 9001 ubuntu-24.04-server-cloudimg-amd64.img $STORAGE

# Step 3: Determine the correct disk path based on storage type
if [ "$STORAGE" = "local" ]; then
    DISK_PATH="$STORAGE:9001/vm-9001-disk-0.qcow2"
else
    DISK_PATH="$STORAGE:vm-9001-disk-0"
fi

# Step 4: Configure the VM with cloud-init support
qm set 9001 --scsihw virtio-scsi-pci --scsi0 $DISK_PATH
qm set 9001 --ide2 $STORAGE:cloudinit
qm set 9001 --boot c --bootdisk scsi0
qm set 9001 --serial0 socket --vga serial0
qm set 9001 --agent enabled=1

# Step 5: Verify the disk is attached correctly
qm config 9001 | grep scsi0

# Step 6: Convert to template
qm template 9001

# Clean up
rm ubuntu-24.04-server-cloudimg-amd64.img
```

#### Debian 12 Template

```bash
# Step 1: Download Debian 12 cloud image
cd /tmp
wget https://cloud.debian.org/images/cloud/bookworm/latest/debian-12-generic-amd64.qcow2

# Step 2: Create VM and configure (replace 'local-lvm' with your storage pool)
STORAGE="local-lvm"  # Change this to your storage pool
qm create 9002 --name debian-12-standard --memory 2048 --net0 virtio,bridge=vmbr0
qm importdisk 9002 debian-12-generic-amd64.qcow2 $STORAGE

# Step 3: Determine the correct disk path based on storage type
if [ "$STORAGE" = "local" ]; then
    DISK_PATH="$STORAGE:9002/vm-9002-disk-0.qcow2"
else
    DISK_PATH="$STORAGE:vm-9002-disk-0"
fi

# Step 4: Configure the VM with cloud-init support
qm set 9002 --scsihw virtio-scsi-pci --scsi0 $DISK_PATH
qm set 9002 --ide2 $STORAGE:cloudinit
qm set 9002 --boot c --bootdisk scsi0
qm set 9002 --serial0 socket --vga serial0
qm set 9002 --agent enabled=1

# Step 5: Verify the disk is attached correctly
qm config 9002 | grep scsi0

# Step 6: Convert to template
qm template 9002

# Clean up
rm debian-12-generic-amd64.qcow2
```

#### Debian 13 Template

```bash
# Step 1: Download Debian 13 cloud image
cd /tmp
wget https://cloud.debian.org/images/cloud/trixie/latest/debian-13-generic-amd64.qcow2

# Step 2: Create VM and configure (replace 'local-lvm' with your storage pool)
STORAGE="local-lvm"  # Change this to your storage pool
qm create 9003 --name debian-13-standard --memory 2048 --net0 virtio,bridge=vmbr0
qm importdisk 9003 debian-13-generic-amd64.qcow2 $STORAGE

# Step 3: Determine the correct disk path based on storage type
if [ "$STORAGE" = "local" ]; then
    DISK_PATH="$STORAGE:9003/vm-9003-disk-0.qcow2"
else
    DISK_PATH="$STORAGE:vm-9003-disk-0"
fi

# Step 4: Configure the VM with cloud-init support
qm set 9003 --scsihw virtio-scsi-pci --scsi0 $DISK_PATH
qm set 9003 --ide2 $STORAGE:cloudinit
qm set 9003 --boot c --bootdisk scsi0
qm set 9003 --serial0 socket --vga serial0
qm set 9003 --agent enabled=1

# Step 5: Verify the disk is attached correctly
qm config 9003 | grep scsi0

# Step 6: Convert to template
qm template 9003

# Clean up
rm debian-13-generic-amd64.qcow2
```

#### Rocky Linux 9 Template

```bash
# Step 1: Download Rocky Linux 9 cloud image
cd /tmp
wget https://download.rockylinux.org/pub/rocky/9/images/x86_64/Rocky-9-GenericCloud-Base.latest.x86_64.qcow2

# Step 2: Create VM and configure (replace 'local-lvm' with your storage pool)
STORAGE="local-lvm"  # Change this to your storage pool
qm create 9004 --name rockylinux-9-standard --memory 2048 --net0 virtio,bridge=vmbr0
qm importdisk 9004 Rocky-9-GenericCloud-Base.latest.x86_64.qcow2 $STORAGE

# Step 3: Determine the correct disk path based on storage type
if [ "$STORAGE" = "local" ]; then
    DISK_PATH="$STORAGE:9004/vm-9004-disk-0.qcow2"
else
    DISK_PATH="$STORAGE:vm-9004-disk-0"
fi

# Step 4: Configure the VM with cloud-init support
qm set 9004 --scsihw virtio-scsi-pci --scsi0 $DISK_PATH
qm set 9004 --ide2 $STORAGE:cloudinit
qm set 9004 --boot c --bootdisk scsi0
qm set 9004 --serial0 socket --vga serial0
qm set 9004 --agent enabled=1

# Step 5: Verify the disk is attached correctly
qm config 9004 | grep scsi0

# Step 6: Convert to template
qm template 9004

# Clean up
rm Rocky-9-GenericCloud-Base.latest.x86_64.qcow2
```

#### AlmaLinux 9 Template

```bash
# Step 1: Download AlmaLinux 9 cloud image
cd /tmp
wget https://repo.almalinux.org/almalinux/9/cloud/x86_64/images/AlmaLinux-9-GenericCloud-latest.x86_64.qcow2

# Step 2: Create VM and configure (replace 'local-lvm' with your storage pool)
STORAGE="local-lvm"  # Change this to your storage pool
qm create 9005 --name almalinux-9-standard --memory 2048 --net0 virtio,bridge=vmbr0
qm importdisk 9005 AlmaLinux-9-GenericCloud-latest.x86_64.qcow2 $STORAGE

# Step 3: Determine the correct disk path based on storage type
if [ "$STORAGE" = "local" ]; then
    DISK_PATH="$STORAGE:9005/vm-9005-disk-0.qcow2"
else
    DISK_PATH="$STORAGE:vm-9005-disk-0"
fi

# Step 4: Configure the VM with cloud-init support
qm set 9005 --scsihw virtio-scsi-pci --scsi0 $DISK_PATH
qm set 9005 --ide2 $STORAGE:cloudinit
qm set 9005 --boot c --bootdisk scsi0
qm set 9005 --serial0 socket --vga serial0
qm set 9005 --agent enabled=1

# Step 5: Verify the disk is attached correctly
qm config 9005 | grep scsi0

# Step 6: Convert to template
qm template 9005

# Clean up
rm AlmaLinux-9-GenericCloud-latest.x86_64.qcow2
```

#### Template Setup Notes

**Storage Pool:** Replace `local-lvm` with your storage pool name if different. Common values:

- `local-lvm` - LVM thin provisioning (default)
- `local` - Directory storage
- `local-zfs` - ZFS storage
- Custom storage pools

**VM IDs:** The examples use VM IDs 9000-9005. You can use any available VM IDs, but ensure they don't conflict with your `PROXMOX_VM_ID_START` range.

**Storage Type Detection:**

The setup script automatically detects storage type, but if setting up manually, note the differences:

- **Directory storage (`local`)**: Disk path format is `local:9000/vm-9000-disk-0.qcow2` (includes vmID subdirectory and `.qcow2` extension)
- **LVM/ZFS storage (`local-lvm`, `local-zfs`)**: Disk path format is `storage:vm-9000-disk-0` (no vmID subdirectory, no extension)

The manual setup scripts below handle this automatically based on the `$STORAGE` variable.

**Verification:** After creating templates, verify they appear in Proxmox:

1. Go to **Datacenter** → **Your Node** → **VMs**
2. Look for templates (they'll have a template icon)
3. Ensure template names match exactly (case-sensitive)

**Cloud-Init Configuration:** All templates are configured with:

- Cloud-init support via `ide2` (CD-ROM drive)
- QEMU guest agent enabled (for IP address retrieval)
- Serial console for better logging
- VirtIO SCSI for better disk performance
- VirtIO network adapter

**Alternative: Using Different Storage**

If your storage pool name is different (e.g., `local-zfs`), replace `local-lvm` in all commands:

```bash
# Example for ZFS storage
qm importdisk 9000 ubuntu-22.04-server-cloudimg-amd64.img local-zfs
qm set 9000 --scsi0 local-zfs:vm-9000-disk-0
qm set 9000 --ide2 local-zfs:cloudinit
```

**Updating Templates**

To update a template with the latest cloud image:

1. Delete the old template (or use a new VM ID)
2. Follow the creation steps above with the new image
3. The new template will be used for future VM provisioning

**Troubleshooting Template Issues**

If VMs are not being created from templates:

1. **Check template names:** Template names must match exactly (case-sensitive). Verify in Proxmox web UI.
2. **Verify template status:** Templates should show as "Template" in Proxmox, not as regular VMs.
3. **Check storage pool:** Ensure the storage pool used for templates matches your `PROXMOX_STORAGE_POOL` environment variable.
4. **Verify permissions:** The API token needs `VM.Clone` permission to clone templates.
5. **Check logs:** If template cloning fails, the system will fall back to ISO installation (which requires ISO files to be uploaded).
6. **Verify template has boot disk:** The template must have a boot disk configured (e.g., `scsi0=local:vm-9000-disk-0`). If the template only has a cloud-init disk (`ide2`), the system will try to create a boot disk automatically, but this may fail if the storage format is incorrect. To verify:
   ```bash
   qm config 9000 | grep scsi0
   ```
   This should show a boot disk, not just a cloud-init disk. If it only shows `ide2=local:cloudinit`, the template was created incorrectly and should be recreated following the steps above.

**Fallback to ISO Installation**

If templates are not available, the system will automatically fall back to ISO installation. However, this requires:

- ISO files to be uploaded to Proxmox ISO storage
- Manual installation (slower provisioning)
- ISO files must match expected names (see VPS Configuration guide for details)

## VPS Instance Sizes

Obiente Cloud provides several pre-configured VPS sizes:

| Size ID  | CPU Cores | RAM  | Storage |
| -------- | --------- | ---- | ------- |
| `small`  | 1         | 1 GB | 10 GB   |
| `medium` | 2         | 2 GB | 20 GB   |
| `large`  | 4         | 4 GB | 40 GB   |
| `xlarge` | 8         | 8 GB | 80 GB   |

Custom sizes can be configured in the dashboard by superadmins.

## Supported Operating Systems

The following operating systems are supported:

- **Ubuntu 22.04 LTS** - Long-term support release
- **Ubuntu 24.04 LTS** - Latest LTS release
- **Debian 12** - Stable release
- **Debian 13** - Latest stable release
- **Rocky Linux 9** - Enterprise Linux
- **AlmaLinux 9** - Enterprise Linux
- **Custom Images** - Upload your own VM templates

## Creating a VPS Instance

### Via Dashboard

1. Navigate to **VPS** in the dashboard
2. Click **New VPS Instance**
3. Configure:
   - **Name**: Descriptive name for your VPS
   - **Region**: Select deployment region
   - **Size**: Choose instance size
   - **Operating System**: Select OS image
   - **SSH Key** (optional): Add SSH key for initial access
4. Click **Create VPS**

### Via API

```bash
curl -X POST https://your-instance/api/v1/vps \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "organization_id": "org-123",
    "name": "my-vps",
    "region": "us-east-1",
    "image": "UBUNTU_24_04",
    "size": "medium"
  }'
```

## Accessing Your VPS

### Web Terminal

1. Navigate to your VPS instance in the dashboard
2. Click **Terminal** tab
3. The terminal will connect automatically via WebSocket

### SSH Access

VPS instances can be accessed via SSH without requiring a dedicated IP address. The SSH proxy routes connections through the vps-gateway service, which uses gRPC to proxy SSH connections.

**Prerequisites:**

- **With Gateway**: Gateway service must be running and accessible (see "Configure VPS Gateway" section above)
- The VPS instance must be running
- Network must be configured on the VPS (even if no public IP is available)

**Connection Steps:**

1. Get proxy connection info:

   ```bash
   curl https://your-instance/api/v1/vps/{vps_id}/proxy-info \
     -H "Authorization: Bearer YOUR_TOKEN"
   ```

2. Use the provided SSH proxy command:

   ```bash
   ssh -p 2222 vps-{vps_id}@your-instance.com
   ```

   When prompted for password, enter your API token (or use SSH key authentication).

3. Or configure SSH config:
   ```ssh-config
   Host vps-{vps_id}
     HostName your-instance.com
     Port 2222
     User vps-{vps_id}
     PreferredAuthentications publickey,password
     PasswordAuthentication yes
     StrictHostKeyChecking no
   ```

**How It Works:**

**With vps-gateway:**
1. User connects to API server on port 2222 (SSH proxy)
2. API server authenticates user (SSH key or API token)
3. API server queries gateway for VPS IP address
4. API server proxies SSH connection via gateway (gRPC)
5. Gateway routes connection to VPS instance
6. User gets interactive SSH session on the VPS


**Troubleshooting SSH Proxy:**

- **"VPS IP address not available"**: 
  - Check gateway service is running and VPS has allocated IP
  - VPS may not have network configured or guest agent not ready. The proxy will attempt to connect using hostname directly.
- **"failed to connect to gateway"**: Check `VPS_GATEWAY_API_SECRET` matches `GATEWAY_API_SECRET` in gateway service. Ensure gateway can reach API at `GATEWAY_API_URL`.
- **"failed to connect via gateway"**: Gateway not configured or unreachable (see "Configure VPS Gateway" section above)
- **"Connection reset"**: Check if SSH proxy service is running and port 2222 is accessible

## Managing VPS Instances

### Start/Stop/Reboot

- **Start**: Powers on the VPS instance
- **Stop**: Gracefully shuts down the VPS
- **Reboot**: Restarts the VPS instance

### Resize

VPS instances can be resized (requires reboot):

1. Navigate to VPS settings
2. Select new size
3. Confirm resize (VPS will reboot)

### Delete

VPS instances can be soft-deleted (recoverable) or force-deleted (permanent):

- **Soft Delete**: Instance is marked as deleted but can be recovered
- **Force Delete**: Permanently removes the VPS and all data

⚠️ **Warning**: Force deletion cannot be undone!

## Monitoring and Metrics

### Real-Time Metrics

View real-time resource usage:

- CPU usage percentage
- Memory usage (used/total)
- Disk usage (used/total)
- Network I/O (RX/TX bytes)
- Disk IOPS

### Usage Tracking

Track resource usage over time:

- Hourly usage aggregation
- Monthly usage summaries
- Cost estimation based on usage

## Quota Management

VPS instances are subject to organization quotas:

- **Plan Limits**: Maximum VPS instances per plan
- **Organization Overrides**: Custom limits per organization
- **Current Usage**: Track active VPS count

Check your quota:

```bash
curl https://your-instance/api/v1/organizations/{org_id}/quota/vps \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Troubleshooting

### VPS Won't Start

1. Check Proxmox logs: `journalctl -u pve-cluster`
2. Verify VM exists in Proxmox: `qm list`
3. Check VM status: `qm status {vmid}`
4. Review API logs for provisioning errors

### Can't Access VPS

1. **Web Terminal**: Check WebSocket connection and authentication
2. **SSH Proxy**: Verify SSH proxy service is running
3. **Network**: Ensure VPS has network connectivity
4. **Guest Agent**: Verify QEMU guest agent is installed and running

### Provisioning Fails

1. Check Proxmox API connectivity
2. Verify API credentials are correct
3. Ensure sufficient resources (CPU, RAM, storage)
4. Check for template availability
5. Review API error logs

## Best Practices

### Security

- ✅ Use SSH keys instead of passwords
- ✅ Keep VPS instances updated
- ✅ Configure firewall rules
- ✅ Use strong root passwords
- ✅ Enable automatic security updates

### Performance

- ✅ Choose appropriate instance size
- ✅ Monitor resource usage
- ✅ Scale up before hitting limits
- ✅ Use SSD storage for better I/O

### Cost Optimization

- ✅ Stop unused VPS instances
- ✅ Use appropriate instance sizes
- ✅ Monitor usage and costs
- ✅ Clean up unused instances

## Advanced Configuration

### Custom Images

To use custom VM templates:

1. Create template in Proxmox
2. Name template with pattern: `{os}-{version}-standard`
3. Ensure template supports cloud-init
4. Reference by `image_id` when creating VPS

### Network Configuration

#### Without VPS Gateway (Default)

VPS instances are connected to the default bridge (`vmbr0`). For custom networking:

1. Configure Proxmox network bridges
2. Update VM configuration via Proxmox API
3. Configure routing as needed

#### With VPS Gateway (Recommended for DHCP Management)

When using the vps-gateway service, VPS instances are connected to the SDN bridge (typically `OCvpsnet`) where the gateway manages DHCP. This provides:

- **Centralized IP Management**: Gateway allocates and tracks IP addresses for all VPS instances
- **DHCP Automation**: VPS instances automatically receive IP addresses via DHCP
- **SSH Proxying**: Gateway can proxy SSH connections without requiring SSH keys on the Proxmox node
- **Network Isolation**: Gateway network can be isolated from the main Proxmox network

**Setup Steps:**

1. **Create Network Bridge in Proxmox**:
   - Configure SDN VNet (see [VPS Gateway Setup Guide](vps-gateway-setup.md))
   - Connect it to your gateway network or create a dedicated network segment
   - See the [VPS Gateway Setup Guide](vps-gateway-setup.md) for detailed instructions

2. **Deploy vps-gateway Service**:
   - See the [VPS Gateway Setup Guide](vps-gateway-setup.md) for detailed deployment instructions
   - Configure DHCP pool, gateway IP, DNS servers
   - Set `GATEWAY_API_SECRET` to match `VPS_GATEWAY_API_SECRET` in the API
   - Set `GATEWAY_API_URL` to point to your API service (e.g., `http://api:3001`)

3. **Configure API Environment Variables**:
   ```bash
   # API Secret (must match GATEWAY_API_SECRET in gateway service)
   VPS_GATEWAY_API_SECRET=your-shared-secret
   # SDN VNet bridge name
   VPS_GATEWAY_BRIDGE=OCvpsnet
   ```
   
   See the [VPS Gateway Setup Guide](vps-gateway-setup.md) for complete configuration details.

4. **VPS Provisioning**:
   - When creating VPS instances, the API will automatically:
     - Allocate an IP address from the gateway
     - Configure the VM to use the gateway bridge
     - Store the allocated IP in the database

**Network Configuration for VPS Instances**

For detailed instructions on setting up the network bridge, gateway VM, and vps-gateway service, see the [VPS Gateway Setup Guide](vps-gateway-setup.md).

Quick summary:
1. **Configure SDN in Proxmox**: Set up SDN Zone and VNet (see [VPS Gateway Setup Guide](vps-gateway-setup.md) for details)
2. **Create Gateway LXC**: Create an LXC container with access to the SDN VNet
3. **Deploy vps-gateway Service**: Deploy the gateway service on the gateway VM
4. **Configure API**: Set `VPS_GATEWAY_API_SECRET` and `VPS_GATEWAY_BRIDGE` in API environment variables. The gateway will automatically connect to the API.

### Storage Pools

Specify storage pool via `PROXMOX_STORAGE_POOL` environment variable:

```bash
PROXMOX_STORAGE_POOL=local-lvm  # Default
PROXMOX_STORAGE_POOL=local-zfs  # ZFS pool
PROXMOX_STORAGE_POOL=ceph-pool  # Ceph storage
```

## Related Documentation

- [VPS Gateway Setup Guide](vps-gateway-setup.md) - Detailed guide for setting up the vps-gateway service
- [VPS Configuration](vps-configuration.md) - Advanced configuration options
- [Troubleshooting Guide](troubleshooting.md) - Common issues and solutions
- [Environment Variables](../reference/environment-variables.md) - Complete variable reference

---

[← Back to Guides](index.md)
