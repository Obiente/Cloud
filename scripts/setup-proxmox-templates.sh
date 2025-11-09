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
    ["almalinux-9-standard"]="9005|https://repo.almalinux.org/almalinux/9/cloud/x86_64/images/AlmaLinux-9-GenericCloud-Base-latest.x86_64.qcow2|AlmaLinux-9-GenericCloud-Base-latest.x86_64.qcow2"
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
        node_name=$(pvecm nodes | grep -v "^$" | head -n1 | awk '{print $1}' 2>/dev/null || echo "")
        if [ -z "$node_name" ]; then
            # Fallback: try to detect from hostname
            node_name=$(hostname -s 2>/dev/null || echo "localhost")
        fi
    fi
    
    print_info "Detecting available storage pools on node: $node_name"
    
    # Get storage pools that support images
    local storages=$(pvesm status -content images 2>/dev/null | awk 'NR>1 {print $1}' | grep -v "^$" || echo "")
    
    if [ -z "$storages" ]; then
        print_warning "Could not auto-detect storage pools. Trying alternative method..."
        storages=$(pvesm status 2>/dev/null | awk 'NR>1 {print $1}' | grep -v "^$" || echo "")
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
    
    # Try to get storage info
    local storage_info=$(pvesm status -storage "$storage" 2>/dev/null | grep "$storage" | awk '{print $2}' || echo "")
    
    if [ -z "$storage_info" ]; then
        # Try alternative method
        storage_info=$(pvesm status 2>/dev/null | grep "^$storage" | awk '{print $2}' || echo "")
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
prompt_storage() {
    local storages="$1"
    local node_name="$2"
    
    print_info "Available storage pools:"
    echo ""
    
    # Create array of storages
    local storage_array=()
    local index=1
    local detected_storage=""
    local detected_type=""
    
    while IFS= read -r storage; do
        if [ -n "$storage" ]; then
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
            
            echo "  [$index] $storage ($type_display)"
            ((index++))
        fi
    done <<< "$storages"
    
    echo ""
    
    if [ -n "$detected_storage" ]; then
        print_info "Auto-detected storage: $detected_storage (type: $detected_type)"
        echo -n "Use detected storage? [Y/n]: "
        read -r use_detected
        use_detected=${use_detected:-Y}
        
        if [[ "$use_detected" =~ ^[Yy]$ ]]; then
            echo "$detected_storage|$detected_type"
            return
        fi
    fi
    
    echo -n "Select storage pool [1-$((index-1))]: "
    read -r selection
    
    if ! [[ "$selection" =~ ^[0-9]+$ ]] || [ "$selection" -lt 1 ] || [ "$selection" -gt $((index-1)) ]; then
        print_error "Invalid selection"
        exit 1
    fi
    
    local selected_storage="${storage_array[$((selection-1))]}"
    local selected_type=$(detect_storage_type "$selected_storage" "$node_name")
    
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
    
    print_info "Processing template: $template_name (VMID: $vmid)"
    
    # Check if template already exists
    local exists=false
    local existing_vmid=""
    
    if template_exists "$template_name"; then
        existing_vmid=$(qm list | grep "$template_name" | awk '{print $1}' | head -n1)
        if [ -n "$existing_vmid" ]; then
            exists=true
            print_warning "Template '$template_name' already exists (VMID: $existing_vmid)"
            echo -n "Update existing template? [Y/n]: "
            read -r update
            update=${update:-Y}
            
            if [[ ! "$update" =~ ^[Yy]$ ]]; then
                print_info "Skipping template: $template_name"
                return
            fi
            
            # Delete existing template
            print_info "Deleting existing template..."
            qm destroy "$existing_vmid" --purge 2>/dev/null || true
            print_success "Deleted existing template"
        fi
    fi
    
    # Create temporary directory for downloads
    local tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT
    
    # Download image
    print_info "Downloading cloud image..."
    local image_path="$tmp_dir/$image_filename"
    
    if ! wget -q --show-progress -O "$image_path" "$image_url"; then
        print_error "Failed to download image from $image_url"
        return 1
    fi
    
    print_success "Downloaded image: $image_filename"
    
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
    
    # Get disk path
    local disk_path=$(get_disk_path "$storage" "$storage_type" "$vmid")
    
    # Configure VM
    print_info "Configuring VM..."
    qm set "$vmid" --scsihw virtio-scsi-pci --scsi0 "$disk_path"
    qm set "$vmid" --ide2 "$storage:cloudinit"
    qm set "$vmid" --boot c --bootdisk scsi0
    qm set "$vmid" --serial0 socket --vga serial0
    qm set "$vmid" --agent enabled=1
    
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
}

# Main function
main() {
    echo "=========================================="
    echo "  Proxmox VM Template Setup Script"
    echo "=========================================="
    echo ""
    
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
    
    # Confirm
    echo "This will create/update the following templates:"
    for template_name in "${!TEMPLATES[@]}"; do
        echo "  - $template_name"
    done
    echo ""
    echo -n "Continue? [Y/n]: "
    read -r confirm
    confirm=${confirm:-Y}
    
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        print_info "Aborted by user"
        exit 0
    fi
    
    echo ""
    
    # Process each template
    local success_count=0
    local fail_count=0
    
    for template_name in "${!TEMPLATES[@]}"; do
        local template_info="${TEMPLATES[$template_name]}"
        local vmid=$(echo "$template_info" | cut -d'|' -f1)
        local image_url=$(echo "$template_info" | cut -d'|' -f2)
        local image_filename=$(echo "$template_info" | cut -d'|' -f3)
        
        if create_or_update_template "$template_name" "$vmid" "$image_url" "$image_filename" "$storage" "$storage_type"; then
            ((success_count++))
        else
            ((fail_count++))
        fi
    done
    
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

