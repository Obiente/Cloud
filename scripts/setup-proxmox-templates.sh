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
declare -A TEMPLATES=(
    ["ubuntu-22.04-standard"]="9000|https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04-server-cloudimg-amd64.img|ubuntu-22.04-server-cloudimg-amd64.img"
    ["ubuntu-24.04-standard"]="9001|https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-amd64.img|ubuntu-24.04-server-cloudimg-amd64.img"
    ["debian-12-standard"]="9002|https://cloud.debian.org/images/cloud/bookworm/latest/debian-12-generic-amd64.qcow2|debian-12-generic-amd64.qcow2"
    ["debian-13-standard"]="9003|https://cloud.debian.org/images/cloud/trixie/latest/debian-13-generic-amd64.qcow2|debian-13-generic-amd64.qcow2"
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
        *lvm*|*lvm-thin*)
            echo "lvm"
            ;;
        *zfs*|*zfspool*)
            echo "zfs"
            ;;
        *)
            # Default detection based on storage name
            if [ "$storage" = "local" ]; then
                echo "dir"
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
                    type_display="Directory (files)"
                    ;;
                lvm)
                    type_display="LVM (block device)"
                    ;;
                zfs)
                    type_display="ZFS (block device)"
                    ;;
                *)
                    type_display="Unknown"
                    ;;
            esac
            
            # Auto-detect: prefer local-lvm or local-zfs, fallback to first available
            if [ -z "$detected_storage" ]; then
                if [ "$storage" = "local-lvm" ] || [ "$storage" = "local-zfs" ]; then
                    detected_storage="$storage"
                    detected_type="$storage_type"
                elif [ "$index" -eq 1 ]; then
                    detected_storage="$storage"
                    detected_type="$storage_type"
                fi
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
            
            # Delete existing template
            print_info "Deleting existing template..."
            qm destroy "$existing_vmid" --purge 2>/dev/null || true
            print_success "Deleted existing template"
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
    
    # Create VM
    print_info "Creating VM $vmid..."
    if ! qm create "$vmid" --name "$template_name" --memory 2048 --net0 virtio,bridge=vmbr0 2>/dev/null; then
        print_error "Failed to create VM $vmid (may already exist)"
        # Try to destroy and recreate
        qm destroy "$vmid" --purge 2>/dev/null || true
        qm create "$vmid" --name "$template_name" --memory 2048 --net0 virtio,bridge=vmbr0
    fi
    
    # Import disk
    print_info "Importing disk to storage: $storage..."
    if ! qm importdisk "$vmid" "$image_path" "$storage"; then
        print_error "Failed to import disk"
        qm destroy "$vmid" --purge 2>/dev/null || true
        return 1
    fi
    
    # Get the actual imported disk path
    # qm importdisk creates an unused disk (unused0) that we need to find
    print_info "Detecting imported disk..."
    local disk_path=""
    
    # Get the unused disk from VM config (qm importdisk creates unused0)
    local unused_disk=$(qm config "$vmid" 2>/dev/null | grep "^unused0:" | sed 's/unused0:[[:space:]]*//' | sed 's/[[:space:]]*$//' || echo "")
    
    if [ -n "$unused_disk" ]; then
        # Use the actual imported disk path
        disk_path="$unused_disk"
        print_info "Found imported disk: $disk_path"
    else
        # Fallback: construct path based on storage type
        if [ "$storage_type" = "dir" ]; then
            # For directory storage, check both .raw and .qcow2
            # Try .raw first (common for Debian images)
            local raw_path="$storage:$vmid/vm-$vmid-disk-0.raw"
            local qcow2_path="$storage:$vmid/vm-$vmid-disk-0.qcow2"
            
            # Check which file actually exists on disk
            local storage_path=$(pvesm path "$raw_path" 2>/dev/null || echo "")
            if [ -n "$storage_path" ] && [ -f "$storage_path" ]; then
                disk_path="$raw_path"
            else
                storage_path=$(pvesm path "$qcow2_path" 2>/dev/null || echo "")
                if [ -n "$storage_path" ] && [ -f "$storage_path" ]; then
                    disk_path="$qcow2_path"
                else
                    # Last resort: try raw (most common for cloud images)
                    disk_path="$raw_path"
                fi
            fi
        else
            # For LVM/ZFS, format is storage:vm-vmid-disk-0
            disk_path="$storage:vm-$vmid-disk-0"
        fi
        print_warning "Could not find unused0 in config, using constructed path: $disk_path"
    fi
    
    if [ -z "$disk_path" ]; then
        print_error "Could not detect imported disk path"
        qm destroy "$vmid" --purge 2>/dev/null || true
        return 1
    fi
    
    print_info "Using disk: $disk_path"
    
    # Configure VM
    print_info "Configuring VM..."
    qm set "$vmid" --scsihw virtio-scsi-pci --scsi0 "$disk_path"
    qm set "$vmid" --ide2 "$storage:cloudinit"
    qm set "$vmid" --boot c --bootdisk scsi0
    qm set "$vmid" --serial0 socket --vga serial0
    qm set "$vmid" --agent enabled=1
    
    # Set kernel boot arguments to use device names instead of UUIDs
    # This prevents boot failures when cloning (cloned VMs have different disk UUIDs)
    # We use /dev/sda1 for scsi0 disks (cloud images typically use scsi0)
    print_info "Setting kernel boot arguments to use device names (fixes UUID issues on clone)..."
    qm set "$vmid" --args "root=/dev/sda1 rootdelay=10"
    
    # Verify disk is attached
    print_info "Verifying disk attachment..."
    local disk_config=$(qm config "$vmid" | grep "^scsi0:" || echo "")
    if [ -z "$disk_config" ]; then
        print_error "Disk not attached correctly!"
        qm destroy "$vmid" --purge 2>/dev/null || true
        return 1
    fi
    
    print_success "Disk attached: $disk_config"
    
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

