package vps

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"api/internal/database"
	"api/internal/logger"

	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"

	"connectrpc.com/connect"
	"gorm.io/gorm"
)

// syncVPSStatusFromProxmox syncs the VPS status from Proxmox to the database
// This ensures we have the current status before performing actions
func (s *Service) syncVPSStatusFromProxmox(ctx context.Context, vpsID string) error {
	// Try to sync status from Proxmox, but don't fail if it errors
	// This is best-effort to keep status accurate
	if err := s.vpsManager.SyncVPSStatusFromProxmox(ctx, vpsID); err != nil {
		// Log warning but don't fail - we'll still try to perform the action
		// The action itself will update the status after completion
		logger.Warn("[VPS Service] Failed to sync VPS %s status from Proxmox before action: %v", vpsID, err)
	}
	return nil
}

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

	// Sync status from Proxmox before performing action to ensure we have current status
	// This prevents issues where VPS is stuck in REBOOTING or other transitional states
	s.syncVPSStatusFromProxmox(ctx, vpsID)

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

	// Sync status from Proxmox before performing action to ensure we have current status
	// This prevents issues where VPS is stuck in REBOOTING or other transitional states
	s.syncVPSStatusFromProxmox(ctx, vpsID)

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

	// Sync status from Proxmox before performing action to ensure we have current status
	// This prevents issues where VPS is stuck in REBOOTING or other transitional states
	s.syncVPSStatusFromProxmox(ctx, vpsID)

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

// ReinitializeVPS reinitializes a VPS instance by deleting the VM and recreating it
func (s *Service) ReinitializeVPS(ctx context.Context, req *connect.Request[vpsv1.ReinitializeVPSRequest]) (*connect.Response[vpsv1.ReinitializeVPSResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.manage"); err != nil {
		return nil, err
	}

	// Reinitialize VPS via manager
	vpsInstance, rootPassword, err := s.vpsManager.ReinitializeVPS(ctx, vpsID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to reinitialize VPS: %w", err))
	}

	// Convert to proto
	protoVPS := vpsToProto(vpsInstance)

	// Build response message
	message := "VPS has been reinitialized. The operating system has been reinstalled and cloud-init will be reapplied. "
	if rootPassword != "" {
		message += "Please save the root password as it will not be shown again."
	}

	response := &vpsv1.ReinitializeVPSResponse{
		Vps:     protoVPS,
		Message: message,
	}
	if rootPassword != "" {
		response.RootPassword = &rootPassword
	}

	return connect.NewResponse(response), nil
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

	// Get API base URL from environment or construct from DOMAIN
	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		// Try to construct from DOMAIN environment variable (used in docker-compose)
		domain := os.Getenv("DOMAIN")
		if domain != "" && domain != "localhost" {
			// Use HTTPS for production domains
			apiBaseURL = fmt.Sprintf("https://api.%s", domain)
		} else {
			// For localhost/dev, use HTTP
			apiBaseURL = "http://localhost:3001"
		}
	}

	// Construct WebSocket URL for terminal access
	wsURL := fmt.Sprintf("%s/vps/%s/terminal/ws", apiBaseURL, vpsID)
	// Convert http/https to ws/wss
	if len(wsURL) > 4 && wsURL[:5] == "https" {
		wsURL = "wss" + wsURL[5:]
	} else if len(wsURL) > 3 && wsURL[:4] == "http" {
		wsURL = "ws" + wsURL[4:]
	}

	// SSH proxy is exposed directly on port 2222 (bypassing Traefik)
	// Get API server hostname for SSH connection
	// Use DOMAIN environment variable directly, or extract from API base URL
	sshHost := ""
	domain := os.Getenv("DOMAIN")
	if domain != "" && domain != "localhost" {
		// Use DOMAIN env variable directly
		sshHost = domain
	} else if apiBaseURL != "" {
		// Fallback: extract from API base URL
		if u, err := url.Parse(apiBaseURL); err == nil {
			apiHost := u.Hostname()
			// If hostname is api.domain.com, use just domain.com for SSH
			if strings.HasPrefix(apiHost, "api.") {
				sshHost = strings.TrimPrefix(apiHost, "api.")
			} else if apiHost != "localhost" && apiHost != "127.0.0.1" {
				sshHost = apiHost
			}
		}
	}
	if sshHost == "" {
		// Final fallback: use localhost for dev
		sshHost = "localhost"
	}
	
	sshProxyPort := os.Getenv("SSH_PROXY_PORT")
	if sshProxyPort == "" {
		sshProxyPort = "2222"
	}
	sshPort := sshProxyPort

	// Construct SSH connection instructions
	// Users connect directly to API server on port 2222 (SSH proxy)
	// Username format: root@{vps_id}@{host} (standard SSH user@host format)
	// Authentication: SSH public key (recommended) or API token as password
	// Default user is "root", but users can specify different users like: user@vps-xxx@domain
	defaultUser := "root"
	sshCommand := fmt.Sprintf("ssh -p %s %s@%s@%s", sshPort, defaultUser, vpsID, sshHost)
	sshConfig := fmt.Sprintf(`Host %s
  HostName %s
  Port %s
  User %s@%s
  PreferredAuthentications publickey,password
  PasswordAuthentication yes
  StrictHostKeyChecking no
  # Use SSH key (recommended) or API token as password
  # To connect as a different user, use: ssh -p %s user@%s@%s`, vpsID, sshHost, sshPort, defaultUser, vpsID, sshPort, vpsID, sshHost)

	instructions := fmt.Sprintf(`To access your VPS instance "%s":

1. Web Terminal (Browser):
   - Use the built-in web terminal in the dashboard
   - Or connect via WebSocket: %s

2. SSH Access (via SSH Proxy):
   - Connect via SSH using the standard format: user@vps-id@domain
   - Default user is "root", but you can specify any user:
     %s
   
   - Examples:
     * Connect as root: ssh -p %s root@%s@%s
     * Connect as a different user: ssh -p %s username@%s@%s
   
   - Authentication options:
     * SSH public key (recommended): Add your SSH key in account settings
     * Password: Use the VPS user's password (if password auth is enabled)
   
   - Or add this to your ~/.ssh/config:
%s

Note: 
- The VPS must be running to access it
- SSH keys are automatically added to new VPS instances via cloud-init
- The SSH proxy handles the connection to your VPS securely
- Agent forwarding is supported: use -A flag (ssh -A -p %s root@%s@%s)`, vps.Name, wsURL, sshCommand, sshPort, vpsID, sshHost, sshPort, vpsID, sshHost, sshConfig, sshPort, vpsID, sshHost)

	return connect.NewResponse(&vpsv1.GetVPSProxyInfoResponse{
		VpsId:                  vpsID,
		TerminalWsUrl:          wsURL,
		SshProxyCommand:        sshCommand,
		ConnectionInstructions: instructions,
	}), nil
}


