package vps

import (
	"context"
	"errors"
	"fmt"
	"os"

	vpsv1 "api/gen/proto/obiente/cloud/vps/v1"
	"api/internal/database"

	"connectrpc.com/connect"
	"gorm.io/gorm"
)

// StartVPS starts a stopped VPS instance
func (s *Service) StartVPS(ctx context.Context, req *connect.Request[vpsv1.StartVPSRequest]) (*connect.Response[vpsv1.StartVPSResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.manage"); err != nil {
		return nil, err
	}

	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	// Start VM via Proxmox
	if err := s.vpsManager.StartVPS(ctx, vpsID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to start VPS: %w", err))
	}

	// Refresh VPS instance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to refresh VPS: %w", err))
	}

	return connect.NewResponse(&vpsv1.StartVPSResponse{
		Vps: vpsToProto(&vps),
	}), nil
}

// StopVPS stops a running VPS instance
func (s *Service) StopVPS(ctx context.Context, req *connect.Request[vpsv1.StopVPSRequest]) (*connect.Response[vpsv1.StopVPSResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.manage"); err != nil {
		return nil, err
	}

	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	// Stop VM via Proxmox
	if err := s.vpsManager.StopVPS(ctx, vpsID, false); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stop VPS: %w", err))
	}

	// Refresh VPS instance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to refresh VPS: %w", err))
	}

	return connect.NewResponse(&vpsv1.StopVPSResponse{
		Vps: vpsToProto(&vps),
	}), nil
}

// RebootVPS reboots a VPS instance
func (s *Service) RebootVPS(ctx context.Context, req *connect.Request[vpsv1.RebootVPSRequest]) (*connect.Response[vpsv1.RebootVPSResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.manage"); err != nil {
		return nil, err
	}

	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	// Reboot VM via Proxmox
	if err := s.vpsManager.RebootVPS(ctx, vpsID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to reboot VPS: %w", err))
	}

	// Refresh VPS instance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to refresh VPS: %w", err))
	}

	return connect.NewResponse(&vpsv1.RebootVPSResponse{
		Vps: vpsToProto(&vps),
	}), nil
}

// GetVPSProxyInfo returns proxy connection information for accessing a VPS without dedicated IP
func (s *Service) GetVPSProxyInfo(ctx context.Context, req *connect.Request[vpsv1.GetVPSProxyInfoRequest]) (*connect.Response[vpsv1.GetVPSProxyInfoResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.view"); err != nil {
		return nil, err
	}

	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	// Get API base URL from environment or use default
	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		apiBaseURL = "https://obiente.cloud"
	}

	// Construct WebSocket URL for terminal access
	wsURL := fmt.Sprintf("%s/vps/%s/terminal/ws", apiBaseURL, vpsID)
	// Convert http/https to ws/wss
	if len(wsURL) > 4 && wsURL[:5] == "https" {
		wsURL = "wss" + wsURL[5:]
	} else if len(wsURL) > 3 && wsURL[:4] == "http" {
		wsURL = "ws" + wsURL[4:]
	}

	// Get SSH proxy port from environment
	sshProxyPort := os.Getenv("SSH_PROXY_PORT")
	if sshProxyPort == "" {
		sshProxyPort = "2222"
	}

	// Construct SSH proxy command
	sshProxyCommand := fmt.Sprintf("ssh -J proxy@%s -p %s vps-%s@%s", apiBaseURL, sshProxyPort, vpsID, apiBaseURL)

	instructions := fmt.Sprintf(`To access your VPS instance "%s" without a dedicated IP:

1. Web Terminal (Browser):
   - Open the WebSocket URL in your browser or terminal client
   - URL: %s

2. SSH Access (via Jump Host):
   - Use the SSH proxy command:
   %s
   - Or configure your SSH config:
     Host vps-%s
       ProxyJump proxy@%s:2222
       User root

Note: The VPS must be running to access it.`, vps.Name, wsURL, sshProxyCommand, vpsID, apiBaseURL, sshProxyPort, vpsID)

	return connect.NewResponse(&vpsv1.GetVPSProxyInfoResponse{
		VpsId:                  vpsID,
		TerminalWsUrl:          wsURL,
		SshProxyCommand:        sshProxyCommand,
		ConnectionInstructions: instructions,
	}), nil
}
