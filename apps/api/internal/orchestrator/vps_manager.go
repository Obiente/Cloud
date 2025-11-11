package orchestrator

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"api/internal/database"
	"api/internal/logger"

	"github.com/moby/moby/client"
)

// VPSManager manages the lifecycle of VPS instances via Proxmox
type VPSManager struct {
	dockerClient   client.APIClient
	gatewayClient *VPSGatewayClient
}

// GetProxmoxConfig gets Proxmox configuration from environment variables
func GetProxmoxConfig() (*ProxmoxConfig, error) {
	config := &ProxmoxConfig{}

	// Get Proxmox API URL from environment
	config.APIURL = os.Getenv("PROXMOX_API_URL")
	if config.APIURL == "" {
		return nil, fmt.Errorf("PROXMOX_API_URL environment variable is required")
	}

	// Get username (default: root@pam)
	config.Username = os.Getenv("PROXMOX_USERNAME")
	if config.Username == "" {
		config.Username = "root@pam"
	}

	// Get password (optional if using token)
	config.Password = os.Getenv("PROXMOX_PASSWORD")

	// Get token (alternative to password)
	config.TokenID = os.Getenv("PROXMOX_TOKEN_ID")
	config.Secret = os.Getenv("PROXMOX_TOKEN_SECRET")

	// Validate that either password or token is provided
	if config.Password == "" && (config.TokenID == "" || config.Secret == "") {
		return nil, fmt.Errorf("either PROXMOX_PASSWORD or both PROXMOX_TOKEN_ID and PROXMOX_TOKEN_SECRET must be provided")
	}

	// If using token, clear password
	if config.TokenID != "" && config.Secret != "" {
		config.Password = "" // Clear password if using token
	}

	return config, nil
}

// ProxmoxConfig holds Proxmox API configuration
type ProxmoxConfig struct {
	APIURL   string
	Username string
	Realm    string
	Password string
	TokenID  string // Alternative: use API token instead of password
	Secret   string // Token secret
}

// NewVPSManager creates a new VPS manager
func NewVPSManager() (*VPSManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Initialize gateway client (optional - will be nil if gateway is not configured)
	// Uses VPS_GATEWAY_URL from environment or can be discovered from node metadata
	gatewayClient, err := NewVPSGatewayClient("")
	if err != nil {
		logger.Warn("[VPSManager] Failed to initialize VPS gateway client (gateway may not be configured): %v", err)
		gatewayClient = nil // Continue without gateway - IP allocation will be skipped
	}

	return &VPSManager{
		dockerClient:   cli,
		gatewayClient: gatewayClient,
	}, nil
}

// CreateVPS provisions a new VPS instance via Proxmox
// CreateVPS creates a new VPS instance
// Returns: VPS instance, root password (one-time only, not stored), error
func (vm *VPSManager) CreateVPS(ctx context.Context, config *VPSConfig) (*database.VPSInstance, string, error) {
	logger.Info("[VPSManager] Creating VPS instance %s", config.VPSID)

	// Get organization settings to check if inter-VM communication is allowed
	var org database.Organization
	if err := database.DB.Where("id = ?", config.OrganizationID).First(&org).Error; err != nil {
		return nil, "", fmt.Errorf("failed to get organization: %w", err)
	}

	// Get Proxmox configuration from environment variables
	proxmoxConfig, err := GetProxmoxConfig()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get Proxmox config: %w", err)
	}

	// Create Proxmox client
	proxmoxClient, err := NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	// Allocate IP address from gateway if available
	var allocatedIP string
	var macAddress string
	if vm.gatewayClient != nil {
		// Generate MAC address for the VM (Proxmox will assign one, but we need it for DHCP)
		// Format: 00:16:3e:XX:XX:XX (QEMU/KVM standard prefix)
		macAddress = generateMACAddress()
		
		// Request IP allocation from gateway
		allocResp, err := vm.gatewayClient.AllocateIP(ctx, config.VPSID, config.OrganizationID, macAddress)
		if err != nil {
			logger.Warn("[VPSManager] Failed to allocate IP from gateway for VPS %s: %v (continuing without gateway IP)", config.VPSID, err)
			// Continue without gateway IP - VM will use DHCP or static IP from Proxmox
		} else {
			allocatedIP = allocResp.IpAddress
			logger.Info("[VPSManager] Allocated IP %s for VPS %s from gateway", allocatedIP, config.VPSID)
		}
	}

	// Provision VM via Proxmox API
	createResult, err := proxmoxClient.CreateVM(ctx, config, org.AllowInterVMCommunication)
		if err != nil {
			// If VM creation fails, release the allocated IP
			if vm.gatewayClient != nil && allocatedIP != "" {
				if releaseErr := vm.gatewayClient.ReleaseIP(ctx, config.VPSID); releaseErr != nil {
					logger.Warn("[VPSManager] Failed to release IP %s after VM creation failure: %v", allocatedIP, releaseErr)
				}
			}
			return nil, "", fmt.Errorf("failed to provision VM via Proxmox: %w", err)
		}

		vmID := createResult.VMID
		rootPassword := createResult.Password

		// Get actual VM status from Proxmox and map to our status enum
		vmIDInt := 0
		fmt.Sscanf(vmID, "%d", &vmIDInt)
		if vmIDInt == 0 {
			return nil, "", fmt.Errorf("invalid VM ID: %s", vmID)
		}

		nodes, err := proxmoxClient.ListNodes(ctx)
		if err != nil || len(nodes) == 0 {
			return nil, "", fmt.Errorf("failed to find Proxmox node: %w", err)
		}

	// Verify VM actually exists before creating VPS record
	// If GetVMStatus fails with "does not exist", the VM creation failed
	proxmoxStatus, err := proxmoxClient.GetVMStatus(ctx, nodes[0], vmIDInt)
	if err != nil {
		errorMsg := err.Error()
		// Check if the error indicates the VM config doesn't exist (VM creation failed)
		if strings.Contains(errorMsg, "does not exist") || strings.Contains(errorMsg, "Configuration file") {
			return nil, "", fmt.Errorf("VM creation failed: VM %d does not exist in Proxmox. The VM may not have been created properly: %w", vmIDInt, err)
		}
		// For other errors, still fail - we need to verify the VM exists
		return nil, "", fmt.Errorf("failed to verify VM exists after creation: %w", err)
	}

	// Map Proxmox status to our VPSStatus enum
	vpsStatus := mapProxmoxStatusToVPSStatus(proxmoxStatus)

	// Create VPS instance record in database
	vpsInstance := &database.VPSInstance{
		ID:             config.VPSID,
		Name:           config.Name,
		Description:    config.Description,
		Status:         vpsStatus,
		Region:         config.Region,
		Image:          int32(config.Image),
		ImageID:        config.ImageID,
		Size:           config.Size,
		CPUCores:       config.CPUCores,
		MemoryBytes:    config.MemoryBytes,
		DiskBytes:      config.DiskBytes,
		InstanceID:     &vmID,
		NodeID:         nil, // Node ID not needed - any node can access Proxmox via API
		SSHKeyID:       config.SSHKeyID,
		OrganizationID: config.OrganizationID,
		CreatedBy:      config.CreatedBy,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	// NOTE: Root password is NOT stored in database for security
	// Password is only returned once in CreateVPS response, then discarded

	// Store metadata as JSON (must be valid JSON or NULL for JSONB columns)
	if len(config.Metadata) > 0 {
		metadataJSON, err := json.Marshal(config.Metadata)
		if err == nil {
			vpsInstance.Metadata = string(metadataJSON)
		} else {
			vpsInstance.Metadata = "{}" // Default to empty object if marshaling fails
		}
	} else {
		vpsInstance.Metadata = "{}" // Empty object for JSONB, not empty string
	}

	// Store IP addresses as JSON arrays (must be valid JSON or NULL for JSONB columns)
	// Include gateway-allocated IP if available
	ipv4Addresses := config.IPv4Addresses
	if allocatedIP != "" {
		// Add gateway-allocated IP to the list (prepend it)
		ipv4Addresses = append([]string{allocatedIP}, ipv4Addresses...)
	}
	if len(ipv4Addresses) > 0 {
		ipv4JSON, err := json.Marshal(ipv4Addresses)
		if err == nil {
			vpsInstance.IPv4Addresses = string(ipv4JSON)
		} else {
			vpsInstance.IPv4Addresses = "[]" // Default to empty array if marshaling fails
		}
	} else {
		vpsInstance.IPv4Addresses = "[]" // Empty array for JSONB, not empty string
	}
	if len(config.IPv6Addresses) > 0 {
		ipv6JSON, err := json.Marshal(config.IPv6Addresses)
		if err == nil {
			vpsInstance.IPv6Addresses = string(ipv6JSON)
		} else {
			vpsInstance.IPv6Addresses = "[]" // Default to empty array if marshaling fails
		}
	} else {
		vpsInstance.IPv6Addresses = "[]" // Empty array for JSONB, not empty string
	}

	// Save to database
	if err := database.DB.Create(vpsInstance).Error; err != nil {
		return nil, "", fmt.Errorf("failed to create VPS instance record: %w", err)
	}

	logger.Info("[VPSManager] Created VPS instance %s (VM ID: %s)",
		config.VPSID, vmID)

	// Return VPS instance and root password (password is NOT stored in database)
	// Password is only returned once in CreateVPS response, then discarded
	return vpsInstance, rootPassword, nil
}

// VPSConfig holds configuration for creating a VPS instance
type VPSConfig struct {
	VPSID          string
	Name           string
	Description    *string
	Region         string
	Image          int // VPSImage enum
	ImageID        *string
	Size           string
	CPUCores       int32
	MemoryBytes    int64
	DiskBytes      int64
	SSHKeyID       *string
	Metadata       map[string]string
	IPv4Addresses  []string
	IPv6Addresses  []string
	OrganizationID string
	CreatedBy      string
}

// StartVPS starts a VPS instance
func (vm *VPSManager) StartVPS(ctx context.Context, vpsID string) error {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return fmt.Errorf("VPS has no instance ID")
	}

	proxmoxConfig, err := GetProxmoxConfig()
	if err != nil {
		return fmt.Errorf("failed to get Proxmox config: %w", err)
	}

	proxmoxClient, err := NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	// Parse VM ID and find node
	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	// Find node (for now, use first available)
	nodes, err := proxmoxClient.ListNodes(ctx)
	if err != nil || len(nodes) == 0 {
		return fmt.Errorf("failed to find Proxmox node: %w", err)
	}

	if err := proxmoxClient.startVM(ctx, nodes[0], vmIDInt); err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}

	// Get actual status from Proxmox and update
	proxmoxStatus, err := proxmoxClient.GetVMStatus(ctx, nodes[0], vmIDInt)
	if err != nil {
		logger.Warn("[VPSManager] Failed to get VM status after start, defaulting to RUNNING: %v", err)
		vps.Status = 3 // RUNNING
	} else {
		vps.Status = mapProxmoxStatusToVPSStatus(proxmoxStatus)
	}
	vps.UpdatedAt = time.Now()
	if err := database.DB.Save(&vps).Error; err != nil {
		logger.Warn("[VPSManager] Failed to update VPS status: %v", err)
	}

	return nil
}

// StopVPS stops a VPS instance
func (vm *VPSManager) StopVPS(ctx context.Context, vpsID string, force bool) error {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return fmt.Errorf("VPS has no instance ID")
	}

	proxmoxConfig, err := GetProxmoxConfig()
	if err != nil {
		return fmt.Errorf("failed to get Proxmox config: %w", err)
	}

	proxmoxClient, err := NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	nodes, err := proxmoxClient.ListNodes(ctx)
	if err != nil || len(nodes) == 0 {
		return fmt.Errorf("failed to find Proxmox node: %w", err)
	}

	if err := proxmoxClient.StopVM(ctx, nodes[0], vmIDInt); err != nil {
		return fmt.Errorf("failed to stop VM: %w", err)
	}

	// Get actual status from Proxmox and update
	proxmoxStatus, err := proxmoxClient.GetVMStatus(ctx, nodes[0], vmIDInt)
	if err != nil {
		logger.Warn("[VPSManager] Failed to get VM status after stop, defaulting to STOPPED: %v", err)
		vps.Status = 5 // STOPPED
	} else {
		vps.Status = mapProxmoxStatusToVPSStatus(proxmoxStatus)
	}
	vps.UpdatedAt = time.Now()
	if err := database.DB.Save(&vps).Error; err != nil {
		logger.Warn("[VPSManager] Failed to update VPS status: %v", err)
	}

	return nil
}

// RebootVPS reboots a VPS instance
func (vm *VPSManager) RebootVPS(ctx context.Context, vpsID string) error {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return fmt.Errorf("VPS has no instance ID")
	}

	proxmoxConfig, err := GetProxmoxConfig()
	if err != nil {
		return fmt.Errorf("failed to get Proxmox config: %w", err)
	}

	proxmoxClient, err := NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	nodes, err := proxmoxClient.ListNodes(ctx)
	if err != nil || len(nodes) == 0 {
		return fmt.Errorf("failed to find Proxmox node: %w", err)
	}

	if err := proxmoxClient.RebootVM(ctx, nodes[0], vmIDInt); err != nil {
		return fmt.Errorf("failed to reboot VM: %w", err)
	}

	// Get actual status from Proxmox and update
	// Note: Reboot is async, so status might be "running" or transitioning
	proxmoxStatus, err := proxmoxClient.GetVMStatus(ctx, nodes[0], vmIDInt)
	if err != nil {
		logger.Warn("[VPSManager] Failed to get VM status after reboot, defaulting to REBOOTING: %v", err)
		vps.Status = 6 // REBOOTING
	} else {
		// If VM is still running, it's rebooting; if stopped, it might be starting
		if proxmoxStatus == "running" {
			vps.Status = 6 // REBOOTING
		} else {
			vps.Status = mapProxmoxStatusToVPSStatus(proxmoxStatus)
		}
	}
	vps.UpdatedAt = time.Now()
	if err := database.DB.Save(&vps).Error; err != nil {
		logger.Warn("[VPSManager] Failed to update VPS status: %v", err)
	}

	return nil
}

// GetVPSStatus retrieves the current status of a VPS from Proxmox
func (vm *VPSManager) GetVPSStatus(ctx context.Context, vpsID string) (string, error) {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return "", fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return "", fmt.Errorf("VPS has no instance ID")
	}

	proxmoxConfig, err := GetProxmoxConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get Proxmox config: %w", err)
	}

	proxmoxClient, err := NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return "", fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	nodes, err := proxmoxClient.ListNodes(ctx)
	if err != nil || len(nodes) == 0 {
		return "", fmt.Errorf("failed to find Proxmox node: %w", err)
	}

	status, err := proxmoxClient.GetVMStatus(ctx, nodes[0], vmIDInt)
	if err != nil {
		return "", fmt.Errorf("failed to get VM status: %w", err)
	}

	return status, nil
}

// GetVPSIPAddresses retrieves IP addresses of a VPS from Proxmox
func (vm *VPSManager) GetVPSIPAddresses(ctx context.Context, vpsID string) ([]string, []string, error) {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return nil, nil, fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return nil, nil, fmt.Errorf("VPS has no instance ID")
	}

	proxmoxConfig, err := GetProxmoxConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get Proxmox config: %w", err)
	}

	proxmoxClient, err := NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return nil, nil, fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	nodes, err := proxmoxClient.ListNodes(ctx)
	if err != nil || len(nodes) == 0 {
		return nil, nil, fmt.Errorf("failed to find Proxmox node: %w", err)
	}

	ipv4, ipv6, err := proxmoxClient.GetVMIPAddresses(ctx, nodes[0], vmIDInt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get VM IP addresses: %w", err)
	}

	return ipv4, ipv6, nil
}

// DeleteVPS deletes a VPS instance from Proxmox
// SECURITY: Only deletes VMs that were created by our API
func (vm *VPSManager) DeleteVPS(ctx context.Context, vpsID string) error {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return fmt.Errorf("VPS has no instance ID")
	}

	proxmoxConfig, err := GetProxmoxConfig()
	if err != nil {
		return fmt.Errorf("failed to get Proxmox config: %w", err)
	}

	proxmoxClient, err := NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	nodes, err := proxmoxClient.ListNodes(ctx)
	if err != nil || len(nodes) == 0 {
		return fmt.Errorf("failed to find Proxmox node: %w", err)
	}

	// Release IP address from gateway if available
	if vm.gatewayClient != nil {
		if err := vm.gatewayClient.ReleaseIP(ctx, vpsID); err != nil {
			logger.Warn("[VPSManager] Failed to release IP from gateway for VPS %s: %v (continuing with VM deletion)", vpsID, err)
			// Continue with VM deletion even if IP release fails
		} else {
			logger.Info("[VPSManager] Released IP from gateway for VPS %s", vpsID)
		}
	}

	// DeleteVM will validate that the VM was created by our API by checking VM name matches VPS ID
	if err := proxmoxClient.DeleteVM(ctx, nodes[0], vmIDInt, vpsID); err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	logger.Info("[VPSManager] Successfully deleted VPS %s (VM ID: %d)", vpsID, vmIDInt)
	return nil
}

// Close closes the Docker client
func (vm *VPSManager) Close() error {
	return vm.dockerClient.Close()
}

// SyncVPSStatusFromProxmox updates the VPS status in the database based on the actual Proxmox VM status
func (vm *VPSManager) SyncVPSStatusFromProxmox(ctx context.Context, vpsID string) error {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return fmt.Errorf("VPS has no instance ID")
	}

	proxmoxConfig, err := GetProxmoxConfig()
	if err != nil {
		return fmt.Errorf("failed to get Proxmox config: %w", err)
	}

	proxmoxClient, err := NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	// Find the node where the VM is running (for multi-node clusters)
	nodeName, err := proxmoxClient.FindVMNode(ctx, vmIDInt)
	if err != nil {
		return fmt.Errorf("failed to find VM node: %w", err)
	}

	// Get actual status from Proxmox
	proxmoxStatus, err := proxmoxClient.GetVMStatus(ctx, nodeName, vmIDInt)
	if err != nil {
		return fmt.Errorf("failed to get VM status: %w", err)
	}

	// Map and update status
	vps.Status = mapProxmoxStatusToVPSStatus(proxmoxStatus)
	vps.UpdatedAt = time.Now()
	if err := database.DB.Save(&vps).Error; err != nil {
		return fmt.Errorf("failed to update VPS status: %w", err)
	}

	logger.Info("[VPSManager] Synced VPS %s status from Proxmox: %s -> %d", vpsID, proxmoxStatus, vps.Status)
	return nil
}

// UpdateOrganizationVPSSSHKeys updates SSH keys in cloud-init for all VPS instances in an organization
// This is called when SSH keys are added or removed
func (vm *VPSManager) UpdateOrganizationVPSSSHKeys(ctx context.Context, organizationID string) error {
	return vm.UpdateOrganizationVPSSSHKeysExcluding(ctx, organizationID, "")
}

// UpdateOrganizationVPSSSHKeysExcluding updates SSH keys for all VPS instances in an organization,
// excluding a specific key ID (e.g., when deleting an org-wide key)
func (vm *VPSManager) UpdateOrganizationVPSSSHKeysExcluding(ctx context.Context, organizationID string, excludeKeyID string) error {
	// Get all VPS instances for this organization
	var vpsInstances []database.VPSInstance
	if err := database.DB.Where("organization_id = ? AND deleted_at IS NULL AND instance_id IS NOT NULL", organizationID).Find(&vpsInstances).Error; err != nil {
		return fmt.Errorf("failed to get VPS instances: %w", err)
	}

	if len(vpsInstances) == 0 {
		logger.Info("[VPSManager] No VPS instances found for organization %s, skipping SSH key update", organizationID)
		return nil
	}

	// Get Proxmox configuration
	proxmoxConfig, err := GetProxmoxConfig()
	if err != nil {
		return fmt.Errorf("failed to get Proxmox config: %w", err)
	}

	// Create Proxmox client
	proxmoxClient, err := NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	// Update SSH keys for each VPS instance
	successCount := 0
	for _, vps := range vpsInstances {
		if vps.InstanceID == nil {
			continue
		}

		vmIDInt := 0
		fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
		if vmIDInt == 0 {
			logger.Warn("[VPSManager] Invalid VM ID for VPS %s: %s", vps.ID, *vps.InstanceID)
			continue
		}

		// Find the node where the VM is running
		nodeName, err := proxmoxClient.FindVMNode(ctx, vmIDInt)
		if err != nil {
			logger.Warn("[VPSManager] Failed to find node for VM %d (VPS %s): %v", vmIDInt, vps.ID, err)
			continue
		}

		// Update SSH keys (includes VPS-specific + org-wide), excluding the specified key if provided
		if err := proxmoxClient.UpdateVMSSHKeys(ctx, nodeName, vmIDInt, organizationID, vps.ID, excludeKeyID); err != nil {
			logger.Warn("[VPSManager] Failed to update SSH keys for VM %d (VPS %s): %v", vmIDInt, vps.ID, err)
			continue
		}

		successCount++
	}

	logger.Info("[VPSManager] Updated SSH keys for %d/%d VPS instances in organization %s", successCount, len(vpsInstances), organizationID)
	return nil
}

// UpdateVPSSSHKeys updates SSH keys in cloud-init for a specific VPS instance
// This includes both VPS-specific keys and organization-wide keys
func (vm *VPSManager) UpdateVPSSSHKeys(ctx context.Context, vpsID string) error {
	return vm.UpdateVPSSSHKeysExcluding(ctx, vpsID, "")
}

// UpdateVPSSSHKeysExcluding updates SSH keys in cloud-init for a specific VPS instance,
// excluding a specific key ID (e.g., when deleting a key)
func (vm *VPSManager) UpdateVPSSSHKeysExcluding(ctx context.Context, vpsID string, excludeKeyID string) error {
	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return fmt.Errorf("VPS has no instance ID")
	}

	// Get Proxmox configuration
	proxmoxConfig, err := GetProxmoxConfig()
	if err != nil {
		return fmt.Errorf("failed to get Proxmox config: %w", err)
	}

	// Create Proxmox client
	proxmoxClient, err := NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	// Find the node where the VM is running
	nodeName, err := proxmoxClient.FindVMNode(ctx, vmIDInt)
	if err != nil {
		return fmt.Errorf("failed to find VM node: %w", err)
	}

	// Update SSH keys (includes VPS-specific and org-wide), excluding the specified key if provided
	if err := proxmoxClient.UpdateVMSSHKeys(ctx, nodeName, vmIDInt, vps.OrganizationID, vpsID, excludeKeyID); err != nil {
		return fmt.Errorf("failed to update SSH keys: %w", err)
	}

	logger.Info("[VPSManager] Updated SSH keys for VPS %s (VM %d)", vpsID, vmIDInt)
	return nil
}

// EnableVPSGuestAgent enables QEMU guest agent for a specific VPS instance
func (vm *VPSManager) EnableVPSGuestAgent(ctx context.Context, vpsID string) error {
	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return fmt.Errorf("VPS has no instance ID")
	}

	// Get Proxmox configuration
	proxmoxConfig, err := GetProxmoxConfig()
	if err != nil {
		return fmt.Errorf("failed to get Proxmox config: %w", err)
	}

	// Create Proxmox client
	proxmoxClient, err := NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	// Find the node where the VM is running
	nodeName, err := proxmoxClient.FindVMNode(ctx, vmIDInt)
	if err != nil {
		return fmt.Errorf("failed to find VM node: %w", err)
	}

	// Enable guest agent in VM config
	if err := proxmoxClient.EnableVMGuestAgent(ctx, nodeName, vmIDInt); err != nil {
		return fmt.Errorf("failed to enable guest agent: %w", err)
	}

	logger.Info("[VPSManager] Enabled guest agent for VPS %s (VM %d)", vpsID, vmIDInt)
	return nil
}

// RecoverVPSGuestAgent recovers QEMU guest agent for a specific VPS instance
// This updates both the VM config and cloud-init to ensure guest agent is properly configured
func (vm *VPSManager) RecoverVPSGuestAgent(ctx context.Context, vpsID string) error {
	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return fmt.Errorf("VPS not found: %w", err)
	}

	if vps.InstanceID == nil {
		return fmt.Errorf("VPS has no instance ID")
	}

	// Get Proxmox configuration
	proxmoxConfig, err := GetProxmoxConfig()
	if err != nil {
		return fmt.Errorf("failed to get Proxmox config: %w", err)
	}

	// Create Proxmox client
	proxmoxClient, err := NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	// Find the node where the VM is running
	nodeName, err := proxmoxClient.FindVMNode(ctx, vmIDInt)
	if err != nil {
		return fmt.Errorf("failed to find VM node: %w", err)
	}

	// Recover guest agent (updates both VM config and cloud-init)
	if err := proxmoxClient.RecoverVMGuestAgent(ctx, nodeName, vmIDInt, vps.OrganizationID, vpsID); err != nil {
		return fmt.Errorf("failed to recover guest agent: %w", err)
	}

	logger.Info("[VPSManager] Recovered guest agent for VPS %s (VM %d). VM should be rebooted for changes to take effect.", vpsID, vmIDInt)
	return nil
}

// generateMACAddress generates a random MAC address for a VM
// Format: 00:16:3e:XX:XX:XX (QEMU/KVM standard prefix)
func generateMACAddress() string {
	// Generate random bytes for the last 3 octets
	randBytes := make([]byte, 3)
	rand.Read(randBytes)
	return fmt.Sprintf("00:16:3e:%02x:%02x:%02x", randBytes[0], randBytes[1], randBytes[2])
}

// mapProxmoxStatusToVPSStatus maps Proxmox VM status strings to VPSStatus enum values
// Proxmox status values: "running", "stopped", "paused", "suspended", "unknown"
// VPSStatus enum: CREATING=1, STARTING=2, RUNNING=3, STOPPING=4, STOPPED=5, REBOOTING=6, FAILED=7, DELETING=8, DELETED=9
func mapProxmoxStatusToVPSStatus(proxmoxStatus string) int32 {
	switch strings.ToLower(proxmoxStatus) {
	case "running":
		return 3 // RUNNING
	case "stopped":
		return 5 // STOPPED
	case "paused", "suspended":
		return 5 // STOPPED (treat paused/suspended as stopped)
	default:
		// For unknown or other statuses, default to CREATING
		// This handles cases where VM is still initializing
		return 1 // CREATING
	}
}
