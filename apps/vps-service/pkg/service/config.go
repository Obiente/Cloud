package service

// This package provides a public API for VPS service operations
// that can be used by other services like superadmin-service.

import (
	"context"
	"fmt"
	"os"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	orchestrator "github.com/obiente/cloud/apps/vps-service/orchestrator"
)

// ConfigService provides endpoints for managing VPS configuration
// This is a minimal public implementation for use by other services
type ConfigService struct {
	vpsManager *orchestrator.VPSManager
}

// NewConfigService creates a new VPS config service
// This is a public wrapper that can be used by other services
func NewConfigService(vpsManager *orchestrator.VPSManager) *ConfigService {
	return &ConfigService{
		vpsManager: vpsManager,
	}
}

// LoadCloudInitConfig loads cloud-init configuration for a VPS
func (s *ConfigService) LoadCloudInitConfig(ctx context.Context, vps *database.VPSInstance) (*orchestrator.CloudInitConfig, error) {
	// If VPS is not provisioned yet, return default config
	if vps.InstanceID == nil {
		return &orchestrator.CloudInitConfig{
			Users:            []orchestrator.CloudInitUser{},
			PackageUpdate:    boolPtr(true),
			PackageUpgrade:   boolPtr(false),
			SSHInstallServer: boolPtr(true),
			SSHAllowPW:       boolPtr(true),
		}, nil
	}

	// Parse VM ID
	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return nil, fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	// Get Proxmox configuration
	proxmoxConfig, err := orchestrator.GetProxmoxConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get Proxmox config: %w", err)
	}

	// Create Proxmox client
	proxmoxClient, err := orchestrator.NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	// Get node name from VPS metadata or environment
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	} else {
		nodeName = os.Getenv("PROXMOX_NODE")
		if nodeName == "" && proxmoxConfig.SSHHost != "" {
			nodeName = proxmoxConfig.SSHHost
		}
		if nodeName == "" {
			return nil, fmt.Errorf("no Proxmox node specified")
		}
	}

	// Get VM config to check for cicustom
	vmConfig, err := proxmoxClient.GetVMConfig(ctx, nodeName, vmIDInt)
	if err != nil {
		// If we can't get VM config, return default
		return &orchestrator.CloudInitConfig{
			Users:            []orchestrator.CloudInitUser{},
			PackageUpdate:    boolPtr(true),
			PackageUpgrade:   boolPtr(false),
			SSHInstallServer: boolPtr(true),
			SSHAllowPW:       boolPtr(true),
		}, nil
	}

	// Check if cicustom is set (indicates custom cloud-init)
	cicustom, _ := vmConfig["cicustom"].(string)
	if cicustom == "" {
		// No custom cloud-init, return default config
		return &orchestrator.CloudInitConfig{
			Users:            []orchestrator.CloudInitUser{},
			PackageUpdate:    boolPtr(true),
			PackageUpgrade:   boolPtr(false),
			SSHInstallServer: boolPtr(true),
			SSHAllowPW:       boolPtr(true),
		}, nil
	}

	// For now, return default config
	// Full implementation would read from Proxmox snippet via SSH
	return &orchestrator.CloudInitConfig{
		Users:            []orchestrator.CloudInitUser{},
		PackageUpdate:    boolPtr(true),
		PackageUpgrade:   boolPtr(false),
		SSHInstallServer: boolPtr(true),
		SSHAllowPW:       boolPtr(true),
	}, nil
}

// SaveCloudInitConfig saves cloud-init configuration for a VPS
func (s *ConfigService) SaveCloudInitConfig(ctx context.Context, vps *database.VPSInstance, config *orchestrator.CloudInitConfig) error {
	if config == nil {
		return nil
	}

	if vps.InstanceID == nil {
		return fmt.Errorf("VPS has no instance ID (not provisioned yet)")
	}

	// Parse VM ID
	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return fmt.Errorf("invalid VM ID: %s", *vps.InstanceID)
	}

	// Get Proxmox configuration
	proxmoxConfig, err := orchestrator.GetProxmoxConfig()
	if err != nil {
		return fmt.Errorf("failed to get Proxmox config: %w", err)
	}

	// Create Proxmox client
	proxmoxClient, err := orchestrator.NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return fmt.Errorf("failed to create Proxmox client: %w", err)
	}

	// Get node name from VPS metadata or environment
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	} else {
		nodeName = os.Getenv("PROXMOX_NODE")
		if nodeName == "" && proxmoxConfig.SSHHost != "" {
			nodeName = proxmoxConfig.SSHHost
		}
		if nodeName == "" {
			return fmt.Errorf("no Proxmox node specified")
		}
	}

	// Generate cloud-init user data
	userData := orchestrator.GenerateCloudInitUserData(&orchestrator.VPSConfig{
		VPSID:     vps.ID,
		CloudInit: config,
	})

	// Get storage (default to "local" or from environment)
	storage := os.Getenv("PROXMOX_STORAGE")
	if storage == "" {
		storage = "local"
	}

	// Create cloud-init snippet
	_, err = proxmoxClient.CreateCloudInitSnippet(ctx, nodeName, storage, vmIDInt, userData)
	if err != nil {
		return fmt.Errorf("failed to create cloud-init snippet: %w", err)
	}

	return nil
}

func boolPtr(b bool) *bool {
	return &b
}
