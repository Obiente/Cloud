package vps

import (
	"context"
	"errors"
	"fmt"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	orchestrator "github.com/obiente/cloud/apps/vps-service/orchestrator"

	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"

	"connectrpc.com/connect"
	"gorm.io/gorm"
)

// ResetVPSPassword resets the root password for a VPS instance
// The new password is generated, updated in Proxmox cloud-init, and returned once
// Password is NEVER stored in the database
func (s *Service) ResetVPSPassword(ctx context.Context, req *connect.Request[vpsv1.ResetVPSPasswordRequest]) (*connect.Response[vpsv1.ResetVPSPasswordResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, auth.PermissionVPSUpdate); err != nil {
		return nil, err
	}

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	if vps.InstanceID == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no instance ID (not provisioned yet)"))
	}

	// Parse VM ID
	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid VM ID: %s", *vps.InstanceID))
	}

	// Get node name from VPS (required)
	nodeName := ""
	if vps.NodeID != nil && *vps.NodeID != "" {
		nodeName = *vps.NodeID
	} else {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("VPS has no node ID - cannot determine which Proxmox node to use"))
	}

	// Get VPS manager to get Proxmox client for the node
	vpsManager, err := orchestrator.NewVPSManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create VPS manager: %w", err))
	}
	defer vpsManager.Close()

	// Get Proxmox client for the node where VPS is running
	proxmoxClient, err := vpsManager.GetProxmoxClientForNode(nodeName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get Proxmox client for node %s: %w", nodeName, err))
	}

	// Generate new random password
	newPassword := orchestrator.GenerateRandomPassword(16)

	// Update password in Proxmox cloud-init configuration
	if err := proxmoxClient.UpdateVMCloudInitPassword(ctx, nodeName, vmIDInt, newPassword); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update password in Proxmox: %w", err))
	}

	logger.Info("[VPS Service] Reset root password for VPS %s (VM %d). Password will take effect after VM reboot or cloud-init re-run.", vpsID, vmIDInt)

	return connect.NewResponse(&vpsv1.ResetVPSPasswordResponse{
		VpsId:        vpsID,
		RootPassword: newPassword,
		Message:      "Password has been reset. The new password will take effect after the VM is rebooted or cloud-init is re-run. Please note this password down as it will not be shown again.",
	}), nil
}

