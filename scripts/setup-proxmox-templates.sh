#!/bin/bash

# Proxmox VM Template Setup Script
# This script creates or updates VM templates for Obiente Cloud
#
# Usage:
#   Direct execution (from GitHub):
#     curl -fsSL https://raw.githubusercontent.com/obiente/cloud/main/scripts/setup-proxmox-templates.sh | bash
#
#   Or clone and run:
#     git clone https://github.com/obiente/cloud.git
#     cd cloud
#     ./scripts/setup-proxmox-templates.sh
#
# Options:
#   --recreate-all, -y, --yes    Automatically recreate all templates without prompts
#                                 (uses cached images if available, skips all confirmations)
#   --help, -h                    Show help message
#
# Note: Templates can be created on any storage type. When VMs are cloned to LVM thin storage,
# the system automatically uses qemu-img convert to preserve partition table integrity.
#
# Examples:
#   ./scripts/setup-proxmox-templates.sh              # Interactive mode
#   ./scripts/setup-proxmox-templates.sh --recreate-all # Auto-recreate all templates

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Template configurations
# Using the same image URLs as Proxmox community helper scripts (tteck/community-scripts)
# These are tested and known to work with UEFI boot
declare -A TEMPLATES=(
    ["ubuntu-22.04-standard"]="9000|https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img|jammy-server-cloudimg-amd64.img"
    ["ubuntu-24.04-standard"]="9001|https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img|noble-server-cloudimg-amd64.img"
    ["debian-12-standard"]="9002|https://cloud.debian.org/images/cloud/bookworm/latest/debian-12-genericcloud-amd64.qcow2|debian-12-genericcloud-amd64.qcow2"
    ["debian-13-standard"]="9003|https://cloud.debian.org/images/cloud/trixie/latest/debian-13-genericcloud-amd64.qcow2|debian-13-genericcloud-amd64.qcow2"
    ["rockylinux-9-standard"]="9004|https://download.rockylinux.org/pub/rocky/9/images/x86_64/Rocky-9-GenericCloud-Base.latest.x86_64.qcow2|Rocky-9-GenericCloud-Base.latest.x86_64.qcow2"
    ["almalinux-9-standard"]="9005|https://repo.almalinux.org/almalinux/9/cloud/x86_64/images/AlmaLinux-9-GenericCloud-latest.x86_64.qcow2|AlmaLinux-9-GenericCloud-latest.x86_64.qcow2"
)

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    if ! command_exists qm; then
        print_error "Proxmox 'qm' command not found. This script must be run on a Proxmox node."
        exit 1
    fi
    
    if ! command_exists wget; then
        print_error "wget is required but not installed. Please install it first."
        exit 1
    fi
    
    print_success "All prerequisites met"
}

# Get available storage pools
get_available_storages() {
    local node_name="$1"
    if [ -z "$node_name" ]; then
        # Try to get first node
        node_name=$(pvecm nodes 2>/dev/null | awk 'NF > 0 {print $1; exit}' || echo "")
        if [ -z "$node_name" ]; then
            # Fallback: try to detect from hostname
            node_name=$(hostname -s 2>/dev/null || echo "localhost")
        fi
    fi
    
    print_info "Detecting available storage pools on node: $node_name" >&2
    
    # Get storage pools that support images
    # pvesm status format varies, but typically: name type status content avail used
    # Try to find storages where any column contains "images"
    local storages=$(pvesm status 2>/dev/null | awk 'NR>1 {
        for(i=1; i<=NF; i++) {
            if ($i ~ /images/) {
                print $1
                next
            }
        }
    }' | sed 's/^[ \t]*//;s/[ \t]*$//' | awk 'NF > 0' || echo "")
    
    if [ -z "$storages" ]; then
        # Fallback: Get all storages (skip header line)
        # Most storages support images by default, so list all
        print_info "Listing all available storage pools (most support images by default)..." >&2
        storages=$(pvesm status 2>/dev/null | awk 'NR>1 {print $1}' | sed 's/^[ \t]*//;s/[ \t]*$//' | awk 'NF > 0' || echo "")
    fi
    
    echo "$storages"
}

# Detect storage type
detect_storage_type() {
    local storage="$1"
    local node_name="$2"
    
    if [ -z "$node_name" ]; then
        node_name=$(hostname -s 2>/dev/null || echo "localhost")
    fi
    
    # Try to get storage info from pvesm status
    # Format: storage_name type status content ...
    local storage_info=$(pvesm status 2>/dev/null | awk -v s="$storage" '$1 == s {print $2}' || echo "")
    
    if [ -z "$storage_info" ]; then
        # Try alternative: check storage config using awk for portability
        storage_info=$(pvesm status 2>/dev/null | awk -v s="$storage" '$1 == s {print $2; exit}' || echo "")
    fi
    
    case "$storage_info" in
        *dir*|*directory*)
            echo "dir"
            ;;
        *lvmthin*)
            echo "lvmthin"
            ;;
        *lvm*)
            echo "lvm"
            ;;
        *zfs*|*zfspool*)
            echo "zfs"
            ;;
        *)
            # Default detection based on storage name
            if [ "$storage" = "local" ]; then
                echo "dir"
            elif [[ "$storage" == *"lvmthin"* ]]; then
                echo "lvmthin"
            elif [[ "$storage" == *"lvm"* ]]; then
                echo "lvm"
            elif [[ "$storage" == *"zfs"* ]]; then
                echo "zfs"
            else
                echo "unknown"
            fi
            ;;
    esac
}

# Fix boot configuration to use device names instead of PARTUUID
# Cloud images use PARTUUID which can cause boot issues
fix_boot_config() {
    local vmid="$1"
    local storage_type="$2"
    local volume_id="$3"
    
    local disk_path
    disk_path=$(get_volume_path "$volume_id")
    
    if [ -z "$disk_path" ]; then
        print_warning "Unable to resolve disk path for volume '$volume_id'"
        return 0
    fi
    
    local mount_point="/tmp/fix-boot-$$"
    local root_part=""
    local cleanup_cmd=""
    
    if { [ "$storage_type" = "lvmthin" ] || [ "$storage_type" = "lvm" ]; } && [ -b "$disk_path" ]; then
        print_info "Fixing boot config on block device: $disk_path"
        
        if ! command -v kpartx >/dev/null 2>&1; then
            print_warning "kpartx not available; skipping boot fix on $disk_path"
            return 0
        fi
        
        local kpartx_output
        kpartx_output=$(kpartx -av "$disk_path" 2>&1)
        if [ $? -ne 0 ]; then
            print_warning "kpartx failed: $kpartx_output"
            return 0
        fi
        
        print_info "kpartx output: $kpartx_output"
        sleep 1
        
        local mapper_name
        mapper_name=$(echo "$disk_path" | sed 's|/dev/||; s|-|--|g; s|/|-|g')
        root_part="/dev/mapper/${mapper_name}p1"
        
        if [ ! -b "$root_part" ]; then
            print_warning "Could not find mapper partition for $disk_path"
            ls -la /dev/mapper/ | grep -E "(vm|base).*$vmid" || true
            kpartx -dv "$disk_path" 2>/dev/null || true
            return 0
        fi
        
        cleanup_cmd="kpartx -dv $disk_path"
        
    elif [ "$storage_type" = "dir" ] && [ -f "$disk_path" ]; then
        print_info "Fixing boot config on image file: $disk_path"
        local disk_format="${disk_path##*.}"
        
        if [ "$disk_format" = "$disk_path" ]; then
            disk_format="raw"
        fi
        
        if [ "$disk_format" = "raw" ]; then
            local loop_device
            loop_device=$(losetup -fP --show "$disk_path" 2>&1)
            if [ $? -ne 0 ] || [ -z "$loop_device" ]; then
                print_warning "losetup failed: $loop_device"
                return 0
            fi
            sleep 1
            partprobe "$loop_device" 2>/dev/null || true
            sleep 1
            root_part="${loop_device}p1"
            if [ ! -b "$root_part" ]; then
                print_warning "Could not detect partition on $loop_device"
                losetup -d "$loop_device" 2>/dev/null || true
                return 0
            fi
            cleanup_cmd="losetup -d $loop_device"
        else
            if ! command -v qemu-nbd >/dev/null 2>&1; then
                print_warning "qemu-nbd not available; cannot fix $disk_path"
                return 0
            fi
            modprobe nbd max_part=16 2>/dev/null || true
            sleep 1
            local nbd_device=""
            for i in $(seq 0 15); do
                if [ ! -e "/sys/block/nbd${i}/pid" ] || [ ! -s "/sys/block/nbd${i}/pid" ]; then
                    nbd_device="/dev/nbd${i}"
                    break
                fi
            done
            if [ -z "$nbd_device" ]; then
                print_warning "No free NBD device found"
                return 0
            fi
            local nbd_output
            nbd_output=$(qemu-nbd --connect="$nbd_device" --format="$disk_format" "$disk_path" 2>&1)
            if [ $? -ne 0 ]; then
                print_warning "Failed to connect $disk_path via NBD: $nbd_output"
                return 0
            fi
            sleep 2
            partprobe "$nbd_device" 2>/dev/null || true
            sleep 1
            root_part="${nbd_device}p1"
            if [ ! -b "$root_part" ]; then
                print_warning "Could not find partition on $nbd_device"
                qemu-nbd --disconnect "$nbd_device" 2>/dev/null || true
                return 0
            fi
            cleanup_cmd="qemu-nbd --disconnect $nbd_device"
        fi
    else
        print_warning "Unsupported storage type '$storage_type' or invalid disk path '$disk_path'"
        return 0
    fi
    
    print_info "Found root partition: $root_part"
    
    # Create mount point and mount
    mkdir -p "$mount_point"
    
    local mount_output
    mount_output=$(mount "$root_part" "$mount_point" 2>&1)
    if [ $? -ne 0 ]; then
        print_warning "Failed to mount $root_part: $mount_output"
        eval "$cleanup_cmd" 2>/dev/null || true
        rmdir "$mount_point" 2>/dev/null || true
        return 0
    fi
    
    print_info "Mounted root partition at $mount_point"
    
    local fixed_something=false
    
    # Fix /etc/fstab if it uses PARTUUID or LABEL, and remove /boot/efi (we use BIOS)
    if [ -f "$mount_point/etc/fstab" ]; then
        if grep -q "PARTUUID=" "$mount_point/etc/fstab" 2>/dev/null; then
            print_info "Fixing /etc/fstab..."
            # Replace PARTUUID=xxx with /dev/sda1 for root
            sed -i 's|PARTUUID=[^ \t]*|/dev/sda1|g' "$mount_point/etc/fstab"
            fixed_something=true
        fi
        # Also fix LABEL=cloudimg-rootfs which some images use
        if grep -q "LABEL=cloudimg-rootfs" "$mount_point/etc/fstab" 2>/dev/null; then
            print_info "Fixing LABEL=cloudimg-rootfs in /etc/fstab..."
            sed -i 's|LABEL=cloudimg-rootfs|/dev/sda1|g' "$mount_point/etc/fstab"
            fixed_something=true
        fi
        # Many cloud images ship a /boot/efi entry even when booting in BIOS mode.
        # On Proxmox with SeaBIOS, there is no EFI system partition, so this mount fails
        # and drops the system into emergency mode. Safest is to comment out /boot/efi.
        if grep -q "[[:space:]]/boot/efi[[:space:]]" "$mount_point/etc/fstab" 2>/dev/null; then
            print_info "Commenting out /boot/efi entry in /etc/fstab (no EFI partition in BIOS VMs)..."
            # Comment any non-comment line that mounts /boot/efi
            sed -i 's/^\([^#].*[[:space:]]\/boot\/efi[[:space:]].*\)$/#\1/' "$mount_point/etc/fstab"
            fixed_something=true
        fi
    fi
    
    # Fix GRUB configuration
    local grub_cfg="$mount_point/boot/grub/grub.cfg"
    if [ -f "$grub_cfg" ]; then
        if grep -q "PARTUUID=" "$grub_cfg" 2>/dev/null; then
            print_info "Fixing GRUB configuration (PARTUUID -> /dev/sda1)..."
            sed -i 's|root=PARTUUID=[^ ]*|root=/dev/sda1|g' "$grub_cfg"
            fixed_something=true
        fi
        # Some images (e.g. Ubuntu cloud) use LABEL=cloudimg-rootfs in GRUB
        if grep -q "LABEL=cloudimg-rootfs" "$grub_cfg" 2>/dev/null; then
            print_info "Fixing GRUB configuration (LABEL=cloudimg-rootfs -> /dev/sda1)..."
            sed -i 's|root=LABEL=[^ ]*|root=/dev/sda1|g' "$grub_cfg"
            fixed_something=true
        fi
    fi
    
    # Also fix /etc/default/grub for future kernel updates
    local grub_default="$mount_point/etc/default/grub"
    if [ -f "$grub_default" ]; then
        if grep -q "PARTUUID=" "$grub_default" 2>/dev/null; then
            print_info "Fixing /etc/default/grub (PARTUUID -> /dev/sda1)..."
            sed -i 's|PARTUUID=[^ "]*|/dev/sda1|g' "$grub_default"
            fixed_something=true
        fi
        # Handle LABEL=cloudimg-rootfs or other LABEL-based roots in GRUB_CMDLINE_LINUX
        if grep -q "LABEL=cloudimg-rootfs" "$grub_default" 2>/dev/null; then
            print_info "Fixing /etc/default/grub (LABEL=cloudimg-rootfs -> /dev/sda1)..."
            sed -i 's|LABEL=cloudimg-rootfs|/dev/sda1|g' "$grub_default"
            fixed_something=true
        fi
    fi

    # Also fix any additional GRUB config snippets (Ubuntu cloud images often use /etc/default/grub.d)
    local grub_d_dir="$mount_point/etc/default/grub.d"
    if [ -d "$grub_d_dir" ]; then
        for cfg in "$grub_d_dir"/*.cfg; do
            [ -f "$cfg" ] || continue
            if grep -q "PARTUUID=" "$cfg" 2>/dev/null; then
                print_info "Fixing GRUB snippet $(basename "$cfg") (PARTUUID -> /dev/sda1)..."
                sed -i 's|PARTUUID=[^ "]*|/dev/sda1|g' "$cfg"
                fixed_something=true
            fi
            if grep -q "LABEL=cloudimg-rootfs" "$cfg" 2>/dev/null; then
                print_info "Fixing GRUB snippet $(basename "$cfg") (LABEL=cloudimg-rootfs -> /dev/sda1)..."
                sed -i 's|LABEL=cloudimg-rootfs|/dev/sda1|g' "$cfg"
                fixed_something=true
            fi
        done
    fi
    
    # Sync and unmount
    sync
    umount "$mount_point" 2>/dev/null || true
    rmdir "$mount_point" 2>/dev/null || true
    
    # Cleanup (disconnect loop device or NBD)
    eval "$cleanup_cmd" 2>/dev/null || true
    
    if [ "$fixed_something" = true ]; then
        print_success "Boot configuration fixed"
    else
        print_info "No PARTUUID references found (already using device names or labels)"
    fi
    
    return 0
}

# Resolve Proxmox volume ID (e.g., local-lvmthin:vm-100-disk-0) to an absolute path
get_volume_path() {
    local volume_id="$1"
    if command -v pvesm >/dev/null 2>&1; then
        pvesm path "$volume_id" 2>/dev/null || true
    else
        echo ""
    fi
}

# Prompt for storage selection
# Outputs result to stdout, prompts/info to stderr
prompt_storage() {
    local storages="$1"
    local node_name="$2"
    
    # All user-facing output goes to stderr
    print_info "Available storage pools:" >&2
    echo "" >&2
    
    # Create array of storages
    local storage_array=()
    local index=1
    local detected_storage=""
    local detected_type=""
    
    # Process storages line by line
    while IFS= read -r storage || [ -n "$storage" ]; do
        # Trim whitespace - use sed with simple patterns for portability
        storage=$(echo "$storage" | sed 's/^[ \t]*//;s/[ \t]*$//')
        
        if [ -n "$storage" ] && [ "$storage" != "" ]; then
            storage_array+=("$storage")
            local storage_type=$(detect_storage_type "$storage" "$node_name")
            local type_display=""
            
            case "$storage_type" in
                dir)
                    type_display="Directory (files) - RECOMMENDED for templates"
                    ;;
                lvm)
                    type_display="LVM (block device)"
                    ;;
                lvmthin)
                    type_display="LVM-thin (block device) - NOT recommended for templates"
                    ;;
                zfs)
                    type_display="ZFS (block device)"
                    ;;
                *)
                    type_display="Unknown"
                    ;;
            esac
            
            # Auto-detect: prefer 'local' (directory storage) for templates
            # Directory storage preserves PARTUUID correctly when VMs are cloned
            # Priority: 1) local dir, 2) any dir, 3) lvm/zfs (never unknown/lvmthin)
            if [ "$storage" = "local" ] && [ "$storage_type" = "dir" ]; then
                # Best choice: local directory storage
                detected_storage="$storage"
                detected_type="$storage_type"
            elif [ -z "$detected_storage" ] && [ "$storage_type" = "dir" ]; then
                # Fallback: any directory storage
                detected_storage="$storage"
                detected_type="$storage_type"
            elif [ -z "$detected_storage" ] && [ "$storage_type" != "unknown" ] && [ "$storage_type" != "lvmthin" ]; then
                # Last resort: lvm or zfs (but not unknown or lvmthin)
                detected_storage="$storage"
                detected_type="$storage_type"
            fi
            
            echo "  [$index] $storage ($type_display)" >&2
            ((index++))
        fi
    done <<< "$storages"
    
    echo "" >&2
    
    if [ -n "$detected_storage" ]; then
        print_info "Auto-detected storage: $detected_storage (type: $detected_type)" >&2
        
        
        echo -n "Use detected storage? [Y/n]: " >&2
        # Read from terminal if available, otherwise stdin
        # Use || true to prevent read failure from exiting script (set -e)
        if [ -t 0 ]; then
            read -r use_detected || use_detected="Y"
        elif [ -c /dev/tty ]; then
            read -r use_detected < /dev/tty || use_detected="Y"
        else
            # Fallback: try stdin anyway (might work in some environments)
            read -r use_detected || use_detected="Y"
        fi
        use_detected=${use_detected:-Y}
        
        if [[ "$use_detected" =~ ^[Yy]$ ]]; then
            # Output result to stdout
            echo "$detected_storage|$detected_type"
            return
        fi
    fi
    
    echo -n "Select storage pool [1-$((index-1))]: " >&2
    # Read from terminal if available, otherwise stdin
    # Use || true to prevent read failure from exiting script (set -e)
    if [ -t 0 ]; then
        read -r selection || selection=""
    elif [ -c /dev/tty ]; then
        read -r selection < /dev/tty || selection=""
    else
        # Fallback: try stdin anyway (might work in some environments)
        read -r selection || selection=""
    fi
    
    if ! [[ "$selection" =~ ^[0-9]+$ ]] || [ "$selection" -lt 1 ] || [ "$selection" -gt $((index-1)) ]; then
        print_error "Invalid selection" >&2
        exit 1
    fi
    
    local selected_storage="${storage_array[$((selection-1))]}"
    local selected_type=$(detect_storage_type "$selected_storage" "$node_name")
    
    
    # Output result to stdout
    echo "$selected_storage|$selected_type"
}

# Get disk path based on storage type
get_disk_path() {
    local storage="$1"
    local storage_type="$2"
    local vmid="$3"
    
    if [ "$storage_type" = "dir" ]; then
        echo "$storage:$vmid/vm-$vmid-disk-0.qcow2"
    else
        echo "$storage:vm-$vmid-disk-0"
    fi
}

# Check if template exists
template_exists() {
    local template_name="$1"
    qm list | grep -q "$template_name" 2>/dev/null
}

# Check if template has linked clones (VMs using the template's base volume)
check_linked_clones() {
    local template_vmid="$1"
    local linked_vms=()
    
    # Get all VMs and check if they reference the template's base volume
    # Linked clones will have disk references like "local:9001/base-9001-disk-0.raw/300/vm-300-disk-0.qcow2"
    # or for LVM: "local-lvmthin:base-9001-disk-0,backing=local-lvmthin:base-9001-disk-0"
    while IFS= read -r line; do
        local vmid=$(echo "$line" | awk '{print $1}')
        if [ -z "$vmid" ] || ! [[ "$vmid" =~ ^[0-9]+$ ]]; then
            continue
        fi
        
        # Skip the template itself
        if [ "$vmid" = "$template_vmid" ]; then
            continue
        fi
        
        # Get VM config and check for references to template's base volume
        local config=$(qm config "$vmid" 2>/dev/null)
        if echo "$config" | grep -q "base-$template_vmid-disk-0"; then
            local vm_name=$(echo "$line" | awk '{print $2}')
            linked_vms+=("$vmid:$vm_name")
        fi
    done < <(qm list 2>/dev/null | tail -n +2)
    
    echo "${linked_vms[@]}"
}

# Convert linked clone to full clone
convert_to_full_clone() {
    local vm_id="$1"
    local vm_name="$2"
    local template_vmid="$3"
    
    print_info "Converting VM $vm_id ($vm_name) from linked clone to full clone..."
    
    # Get VM config to find the disk
    local config=$(qm config "$vm_id" 2>/dev/null)
    local disk_key=""
    local disk_value=""
    local storage=""
    
    # Find the disk that references the template
    for key in scsi0 virtio0 sata0 ide0; do
        local disk=$(echo "$config" | grep "^$key:" | cut -d' ' -f2- | tr -d ' ')
        if [ -n "$disk" ] && echo "$disk" | grep -q "base-$template_vmid-disk-0"; then
            disk_key="$key"
            disk_value="$disk"
            
            # Extract storage from disk value
            # Format: storage:path or storage:vm-XXX-disk-0
            if echo "$disk" | grep -q ":"; then
                storage=$(echo "$disk" | cut -d':' -f1)
            fi
            break
        fi
    done
    
    if [ -z "$disk_key" ] || [ -z "$storage" ]; then
        print_error "Could not determine disk or storage for VM $vm_id"
        return 1
    fi
    
    # Stop VM if it's running (required for disk conversion)
    local vm_status=$(qm status "$vm_id" 2>/dev/null | awk '{print $2}')
    local was_running=false
    if [ "$vm_status" = "running" ]; then
        print_info "Stopping VM $vm_id (required for disk conversion)..."
        qm shutdown "$vm_id" 2>/dev/null || true
        # Wait for VM to stop (max 30 seconds)
        local wait_count=0
        while [ $wait_count -lt 30 ]; do
            sleep 1
            vm_status=$(qm status "$vm_id" 2>/dev/null | awk '{print $2}')
            if [ "$vm_status" != "running" ]; then
                break
            fi
            ((wait_count++))
        done
        
        # Force stop if still running
        if [ "$vm_status" = "running" ]; then
            print_info "Force stopping VM $vm_id..."
            qm stop "$vm_id" 2>/dev/null || true
            sleep 2
        fi
        was_running=true
    fi
    
    # Convert linked clone to full clone by moving disk to same storage
    # This forces Proxmox to create a full copy, breaking the link to the template
    print_info "Converting disk $disk_key to full clone (moving to same storage: $storage)..."
    if qm disk move "$vm_id" "$disk_key" --storage "$storage" 2>/dev/null; then
        print_success "Successfully converted VM $vm_id to full clone"
        
        # Restart VM if it was running
        if [ "$was_running" = true ]; then
            print_info "Restarting VM $vm_id..."
            qm start "$vm_id" 2>/dev/null || true
        fi
        return 0
    else
        print_error "Failed to convert VM $vm_id to full clone"
        # Try to restart VM if it was running
        if [ "$was_running" = true ]; then
            print_info "Attempting to restart VM $vm_id..."
            qm start "$vm_id" 2>/dev/null || true
        fi
        return 1
    fi
}

# Create or update template
create_or_update_template() {
    local template_name="$1"
    local vmid="$2"
    local image_url="$3"
    local image_filename="$4"
    local storage="$5"
    local storage_type="$6"
    local skip_prompts="${7:-false}"  # Optional: skip all prompts (default: false)
    
    print_info "Processing template: $template_name (VMID: $vmid)"
    
    # Check if template already exists
    local exists=false
    local existing_vmid=""
    
    if template_exists "$template_name"; then
        existing_vmid=$(qm list | grep "$template_name" | awk '{print $1}' | head -n1)
        if [ -n "$existing_vmid" ]; then
            exists=true
            print_warning "Template '$template_name' already exists (VMID: $existing_vmid)"
            
            if [ "$skip_prompts" = true ]; then
                print_info "Auto-updating existing template (--recreate-all mode)"
            else
                echo -n "Update existing template? [Y/n]: "
                # Read from terminal if available, otherwise stdin
                # Use || true to prevent read failure from exiting script (set -e)
                if [ -t 0 ]; then
                    read -r update || update="Y"
                elif [ -c /dev/tty ]; then
                    read -r update < /dev/tty || update="Y"
                else
                    # Fallback: try stdin anyway (might work in some environments)
                    read -r update || update="Y"
                fi
                update=${update:-Y}
                
                if [[ ! "$update" =~ ^[Yy]$ ]]; then
                    print_info "Skipping template: $template_name"
                    return 0
                fi
            fi
            
            # Check for linked clones before deleting template
            print_info "Checking for VMs using this template (linked clones)..."
            local linked_clones=($(check_linked_clones "$existing_vmid"))
            
            if [ ${#linked_clones[@]} -gt 0 ]; then
                print_warning "Found ${#linked_clones[@]} VM(s) using this template as linked clones:"
                for linked_vm in "${linked_clones[@]}"; do
                    local vm_id=$(echo "$linked_vm" | cut -d':' -f1)
                    local vm_name=$(echo "$linked_vm" | cut -d':' -f2)
                    print_warning "  - VM $vm_id: $vm_name"
                done
                
                if [ "$skip_prompts" = true ]; then
                    print_info "Auto-converting linked clones to full clones (--recreate-all mode)..."
                else
                    echo ""
                    echo -n "Convert these VMs to full clones to proceed? [Y/n]: "
                    # Read from terminal if available, otherwise stdin
                    if [ -t 0 ]; then
                        read -r convert_vms || convert_vms="Y"
                    elif [ -c /dev/tty ]; then
                        read -r convert_vms < /dev/tty || convert_vms="Y"
                    else
                        read -r convert_vms || convert_vms="Y"
                    fi
                    convert_vms=${convert_vms:-Y}
                    
                    if [[ ! "$convert_vms" =~ ^[Yy]$ ]]; then
                        print_warning "Cannot update template while linked clones exist"
                        print_info "Skipping template: $template_name"
                        return 1
                    fi
                fi
                
                # Convert each linked clone to full clone
                local convert_success=true
                for linked_vm in "${linked_clones[@]}"; do
                    local vm_id=$(echo "$linked_vm" | cut -d':' -f1)
                    local vm_name=$(echo "$linked_vm" | cut -d':' -f2)
                    if ! convert_to_full_clone "$vm_id" "$vm_name" "$existing_vmid"; then
                        convert_success=false
                    fi
                done
                
                if [ "$convert_success" != true ]; then
                    print_error "Failed to convert some linked clones to full clones"
                    print_info "Skipping template: $template_name"
                    return 1
                fi
                
                # Wait a moment for disk operations to complete
                sleep 2
                
                print_success "All linked clones converted to full clones"
            fi
            
            # Delete existing template (only if no linked clones exist)
            print_info "Deleting existing template..."
            if qm destroy "$existing_vmid" --purge 2>/dev/null; then
                print_success "Deleted existing template"
            else
                print_error "Failed to delete template (may still be in use)"
                print_info "Skipping template: $template_name"
                return 1
            fi
        fi
    fi
    
    # Use cache directory for images (persists across runs)
    # Default to /tmp/proxmox-images, but allow override via PROXMOX_IMAGE_CACHE env var
    local cache_dir="${PROXMOX_IMAGE_CACHE:-/tmp/proxmox-images}"
    mkdir -p "$cache_dir"
    
    local image_path="$cache_dir/$image_filename"
    local need_download=true
    
    # Check if image already exists in cache
    if [ -f "$image_path" ]; then
        print_info "Found cached image: $image_filename ($(du -h "$image_path" | cut -f1))"
        
        if [ "$skip_prompts" = true ]; then
            need_download=false
            print_info "Using cached image (--recreate-all mode): $image_path"
        else
            echo -n "Use cached image? [Y/n]: " >&2
            if [ -t 0 ]; then
                read -r use_cached || use_cached="Y"
            elif [ -c /dev/tty ]; then
                read -r use_cached < /dev/tty || use_cached="Y"
            else
                read -r use_cached || use_cached="Y"
            fi
            use_cached=${use_cached:-Y}
            
            if [[ "$use_cached" =~ ^[Yy]$ ]]; then
                need_download=false
                print_info "Using cached image: $image_path"
            else
                print_info "Re-downloading image..."
                rm -f "$image_path"
            fi
        fi
    fi
    
    # Download image if needed
    if [ "$need_download" = true ]; then
        print_info "Downloading cloud image from $image_url..."
        if ! wget -q --show-progress -O "$image_path" "$image_url"; then
            print_error "Failed to download image from $image_url"
            return 1
        fi
        print_success "Downloaded and cached image: $image_filename ($(du -h "$image_path" | cut -f1))"
    fi
    
    # Create VM with SeaBIOS (standard BIOS - cloud images are designed for this)
    print_info "Creating VM $vmid..."
    if ! qm create "$vmid" --name "$template_name" --memory 2048 --cores 2 \
        --net0 virtio,bridge=vmbr0 \
        --ostype l26 \
        --scsihw virtio-scsi-pci --agent 1 2>/dev/null; then
        print_error "Failed to create VM $vmid (may already exist)"
        # Try to destroy and recreate
        qm destroy "$vmid" --purge 2>/dev/null || true
        qm create "$vmid" --name "$template_name" --memory 2048 --cores 2 \
            --net0 virtio,bridge=vmbr0 \
            --ostype l26 \
            --scsihw virtio-scsi-pci --agent 1
    fi
    
    # Import the cloud image disk
    print_info "Importing disk to storage: $storage..."
    if ! qm importdisk "$vmid" "$image_path" "$storage"; then
        print_error "Failed to import disk"
        qm destroy "$vmid" --purge 2>/dev/null || true
        return 1
    fi
    
    # Find the imported disk from VM config
    local imported_disk=$(qm config "$vmid" 2>/dev/null | grep "^unused" | head -1 | sed 's/unused[0-9]*:[[:space:]]*//' | tr -d ' ')
    
    if [ -z "$imported_disk" ]; then
        print_error "Could not find imported disk in VM config"
        qm destroy "$vmid" --purge 2>/dev/null || true
        return 1
    fi
    print_info "Found imported disk: $imported_disk"
    
    # Configure VM with SeaBIOS
    print_info "Configuring VM..."
    qm set "$vmid" \
        --scsi0 "${imported_disk},discard=on,ssd=1" \
        --ide2 "${storage}:cloudinit" \
        --boot c --bootdisk scsi0 \
        --serial0 socket --vga std
    
    # Verify disk is attached
    print_info "Verifying disk attachment..."
    local scsi0_config=$(qm config "$vmid" | grep "^scsi0:" || echo "")
    local cloudinit_config=$(qm config "$vmid" | grep "^ide2:" || echo "")
    
    if [ -z "$scsi0_config" ]; then
        print_error "Main disk (scsi0) not attached correctly!"
        print_info "Full VM config:"
        qm config "$vmid"
        qm destroy "$vmid" --purge 2>/dev/null || true
        return 1
    fi
    
    print_success "Main disk: $scsi0_config"
    [ -n "$cloudinit_config" ] && print_success "Cloud-init: $cloudinit_config"
    
    # Fix boot configuration to use device names instead of PARTUUID
    print_info "Fixing boot configuration..."
    fix_boot_config "$vmid" "$storage_type" "$imported_disk"
    
    # Convert to template
    print_info "Converting VM to template..."
    if ! qm template "$vmid"; then
        print_error "Failed to convert VM to template"
        return 1
    fi
    
    print_success "Template '$template_name' created successfully!"
    echo ""
    return 0
}

# Main function
main() {
    # Parse command-line arguments
    local recreate_all=false
    local skip_cache_prompt=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --recreate-all|-y|--yes)
                recreate_all=true
                skip_cache_prompt=true
                shift
                ;;
            --help|-h)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --recreate-all, -y, --yes    Automatically recreate all templates without prompts"
                echo "                                (uses cached images if available, skips all confirmations)"
                echo "  --help, -h                    Show this help message"
                echo ""
                echo "Examples:"
                echo "  $0                           Interactive mode (prompts for each template)"
                echo "  $0 --recreate-all            Recreate all templates automatically"
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done
    
    echo "=========================================="
    echo "  Proxmox VM Template Setup Script"
    echo "=========================================="
    echo ""
    
    if [ "$recreate_all" = true ]; then
        print_info "Auto-recreate mode: All templates will be recreated automatically"
        echo ""
    fi
    
    # Check prerequisites
    check_prerequisites
    
    # Get node name
    local node_name=$(hostname -s 2>/dev/null || echo "localhost")
    print_info "Using node: $node_name"
    echo ""
    
    # Get available storages
    local storages=$(get_available_storages "$node_name")
    
    if [ -z "$storages" ]; then
        print_error "No storage pools found. Please configure storage in Proxmox first."
        exit 1
    fi
    
    # Prompt for storage
    local storage_info=$(prompt_storage "$storages" "$node_name")
    local storage=$(echo "$storage_info" | cut -d'|' -f1)
    local storage_type=$(echo "$storage_info" | cut -d'|' -f2)
    
    echo ""
    print_info "Selected storage: $storage (type: $storage_type)"
    echo ""
    
    # Confirm (skip if recreate_all is true)
    if [ "$recreate_all" != true ]; then
        echo "This will create/update the following templates:"
        for template_name in "${!TEMPLATES[@]}"; do
            echo "  - $template_name"
        done
        echo ""
        echo -n "Continue? [Y/n]: "
        # Read from terminal if available, otherwise stdin
        # Use || true to prevent read failure from exiting script (set -e)
        if [ -t 0 ]; then
            read -r confirm || confirm="Y"
        elif [ -c /dev/tty ]; then
            read -r confirm < /dev/tty || confirm="Y"
        else
            # Fallback: try stdin anyway (might work in some environments)
            read -r confirm || confirm="Y"
        fi
        confirm=${confirm:-Y}
        
        if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
            print_info "Aborted by user"
            exit 0
        fi
        
        echo ""
    fi
    
    # Process each template
    local success_count=0
    local fail_count=0
    
    # Temporarily disable set -e for the loop to continue on errors
    set +e
    
    for template_name in "${!TEMPLATES[@]}"; do
        local template_info="${TEMPLATES[$template_name]}"
        local vmid=$(echo "$template_info" | cut -d'|' -f1)
        local image_url=$(echo "$template_info" | cut -d'|' -f2)
        local image_filename=$(echo "$template_info" | cut -d'|' -f3)
        
        echo ""
        print_info "=========================================="
        print_info "Processing: $template_name"
        print_info "=========================================="
        echo ""
        
        if create_or_update_template "$template_name" "$vmid" "$image_url" "$image_filename" "$storage" "$storage_type" "$recreate_all"; then
            ((success_count++))
            print_success "Successfully processed: $template_name"
        else
            ((fail_count++))
            print_error "Failed to process: $template_name"
        fi
    done
    
    # Re-enable set -e
    set -e
    
    echo ""
    echo "=========================================="
    print_success "Completed: $success_count templates created/updated"
    if [ "$fail_count" -gt 0 ]; then
        print_error "Failed: $fail_count templates"
    fi
    echo "=========================================="
}

# Run main function
main "$@"

