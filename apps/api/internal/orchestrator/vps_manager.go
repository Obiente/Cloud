package orchestrator

import (
	"context"
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
	dockerClient client.APIClient
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

	return &VPSManager{
		dockerClient: cli,
	}, nil
}

// CreateVPS provisions a new VPS instance via Proxmox
func (vm *VPSManager) CreateVPS(ctx context.Context, config *VPSConfig) (*database.VPSInstance, error) {
	logger.Info("[VPSManager] Creating VPS instance %s", config.VPSID)

	// Get organization settings to check if inter-VM communication is allowed
	var org database.Organization
	if err := database.DB.Where("id = ?", config.OrganizationID).First(&org).Error; err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	// Get Proxmox configuration from environment variables
	proxmoxConfig, err := GetProxmoxConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get Proxmox config: %w", err)
	}

	// Create Proxmox client
	proxmoxClient, err := NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	// Provision VM via Proxmox API
	vmID, err := proxmoxClient.CreateVM(ctx, config, org.AllowInterVMCommunication)
	if err != nil {
		return nil, fmt.Errorf("failed to provision VM via Proxmox: %w", err)
	}

	// Get actual VM status from Proxmox and map to our status enum
	vmIDInt := 0
	fmt.Sscanf(vmID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return nil, fmt.Errorf("invalid VM ID: %s", vmID)
	}

	nodes, err := proxmoxClient.ListNodes(ctx)
	if err != nil || len(nodes) == 0 {
		return nil, fmt.Errorf("failed to find Proxmox node: %w", err)
	}

	// Verify VM actually exists before creating VPS record
	// If GetVMStatus fails with "does not exist", the VM creation failed
	proxmoxStatus, err := proxmoxClient.GetVMStatus(ctx, nodes[0], vmIDInt)
	if err != nil {
		errorMsg := err.Error()
		// Check if the error indicates the VM config doesn't exist (VM creation failed)
		if strings.Contains(errorMsg, "does not exist") || strings.Contains(errorMsg, "Configuration file") {
			return nil, fmt.Errorf("VM creation failed: VM %d does not exist in Proxmox. The VM may not have been created properly: %w", vmIDInt, err)
		}
		// For other errors, still fail - we need to verify the VM exists
		return nil, fmt.Errorf("failed to verify VM exists after creation: %w", err)
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
	if len(config.IPv4Addresses) > 0 {
		ipv4JSON, err := json.Marshal(config.IPv4Addresses)
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
		return nil, fmt.Errorf("failed to create VPS instance record: %w", err)
	}

	logger.Info("[VPSManager] Created VPS instance %s (VM ID: %s)",
		config.VPSID, vmID)

	return vpsInstance, nil
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

	nodes, err := proxmoxClient.ListNodes(ctx)
	if err != nil || len(nodes) == 0 {
		return fmt.Errorf("failed to find Proxmox node: %w", err)
	}

	// Get actual status from Proxmox
	proxmoxStatus, err := proxmoxClient.GetVMStatus(ctx, nodes[0], vmIDInt)
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
