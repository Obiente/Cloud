package vps

import (
	"context"
	"errors"
	"fmt"
	"os"

	"api/internal/auth"
	"api/internal/database"
	"api/internal/logger"
	"api/internal/orchestrator"
	"api/internal/services/common"

	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"
	vpsv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1/vpsv1connect"

	"connectrpc.com/connect"
	"gorm.io/gorm"
)

// ConfigService provides endpoints for managing VPS configuration
// including cloud-init settings and user management
type ConfigService struct {
	vpsv1connect.UnimplementedVPSConfigServiceHandler
	permissionChecker *auth.PermissionChecker
	vpsManager        *orchestrator.VPSManager
}

func NewConfigService(vpsManager *orchestrator.VPSManager) *ConfigService {
	return &ConfigService{
		permissionChecker: auth.NewPermissionChecker(),
		vpsManager:        vpsManager,
	}
}

// ensureAuthenticated ensures the user is authenticated
func (s *ConfigService) ensureAuthenticated(ctx context.Context, req connect.AnyRequest) (context.Context, error) {
	return common.EnsureAuthenticated(ctx, req)
}

// checkVPSPermission verifies user permissions for a VPS instance
func (s *ConfigService) checkVPSPermission(ctx context.Context, vpsID string, permission string) error {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}

	if auth.HasRole(userInfo, auth.RoleAdmin) {
		return nil
	}

	if vps.CreatedBy == userInfo.Id {
		return nil
	}

	err = s.permissionChecker.CheckPermission(ctx, auth.ResourceTypeVPS, vpsID, permission)
	if err != nil {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied: %w", err))
	}

	return nil
}

// GetCloudInitConfig retrieves the cloud-init configuration for a VPS
func (s *ConfigService) GetCloudInitConfig(ctx context.Context, req *connect.Request[vpsv1.GetCloudInitConfigRequest]) (*connect.Response[vpsv1.GetCloudInitConfigResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.read"); err != nil {
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

	// Load cloud-init config from database or Proxmox
	cloudInitConfig, err := s.loadCloudInitConfig(ctx, &vps)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load cloud-init config: %w", err))
	}

	// Convert to proto
	cloudInitProto := cloudInitConfigToProto(cloudInitConfig)

	return connect.NewResponse(&vpsv1.GetCloudInitConfigResponse{
		CloudInit: cloudInitProto,
	}), nil
}

// UpdateCloudInitConfig updates the cloud-init configuration for a VPS
func (s *ConfigService) UpdateCloudInitConfig(ctx context.Context, req *connect.Request[vpsv1.UpdateCloudInitConfigRequest]) (*connect.Response[vpsv1.UpdateCloudInitConfigResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.update"); err != nil {
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

	// Convert proto to orchestrator CloudInitConfig
	cloudInitProto := req.Msg.GetCloudInit()
	cloudInitConfig := protoToCloudInitConfig(cloudInitProto)

	// Save cloud-init config and update Proxmox
	if err := s.saveCloudInitConfig(ctx, &vps, cloudInitConfig); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update cloud-init config: %w", err))
	}

	// Convert back to proto for response
	responseConfig := cloudInitConfigToProto(cloudInitConfig)

	return connect.NewResponse(&vpsv1.UpdateCloudInitConfigResponse{
		CloudInit: responseConfig,
		Message:   "Cloud-init configuration updated. Changes will take effect on the next reboot or when cloud-init is re-run.",
	}), nil
}

// ListVPSUsers lists all users configured for a VPS
func (s *ConfigService) ListVPSUsers(ctx context.Context, req *connect.Request[vpsv1.ListVPSUsersRequest]) (*connect.Response[vpsv1.ListVPSUsersResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.read"); err != nil {
		return nil, err
	}

	// Get cloud-init config
	cloudInitConfig, err := s.getCloudInitConfigForVPS(ctx, vpsID)
	if err != nil {
		return nil, err
	}

	// Convert users to proto
	users := make([]*vpsv1.VPSUser, 0, len(cloudInitConfig.Users))
	for _, user := range cloudInitConfig.Users {
		hasPassword := user.Password != nil && *user.Password != ""
		users = append(users, &vpsv1.VPSUser{
			Name:              user.Name,
			HasPassword:       hasPassword,
			SshAuthorizedKeys: user.SSHAuthorizedKeys,
			Sudo:              user.Sudo != nil && *user.Sudo,
			SudoNopasswd:      user.SudoNopasswd != nil && *user.SudoNopasswd,
			Groups:            user.Groups,
			Shell:             user.Shell,
			LockPasswd:        user.LockPasswd != nil && *user.LockPasswd,
			Gecos:             user.Gecos,
		})
	}

	return connect.NewResponse(&vpsv1.ListVPSUsersResponse{
		Users: users,
	}), nil
}

// CreateVPSUser creates a new user on a VPS
func (s *ConfigService) CreateVPSUser(ctx context.Context, req *connect.Request[vpsv1.CreateVPSUserRequest]) (*connect.Response[vpsv1.CreateVPSUserResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.update"); err != nil {
		return nil, err
	}

	// Validate username
	username := req.Msg.GetName()
	if username == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("username is required"))
	}
	if username == "root" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cannot create root user (already exists)"))
	}

	// Get current cloud-init config
	cloudInitConfig, err := s.getCloudInitConfigForVPS(ctx, vpsID)
	if err != nil {
		return nil, err
	}

	// Check if user already exists
	for _, existingUser := range cloudInitConfig.Users {
		if existingUser.Name == username {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("user %s already exists", username))
		}
	}

	// Convert SSH key IDs to public keys
	sshKeys, err := s.resolveSSHKeyIDs(ctx, req.Msg.GetOrganizationId(), vpsID, req.Msg.SshAuthorizedKeys)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to resolve SSH keys: %w", err))
	}

	// Create new user
	newUser := orchestrator.CloudInitUser{
		Name:              username,
		SSHAuthorizedKeys: sshKeys,
		Groups:            req.Msg.Groups,
	}

	if req.Msg.Password != nil && req.Msg.GetPassword() != "" {
		password := req.Msg.GetPassword()
		newUser.Password = &password
	}
	if req.Msg.Sudo != nil {
		sudo := req.Msg.GetSudo()
		newUser.Sudo = &sudo
	}
	if req.Msg.SudoNopasswd != nil {
		sudoNopasswd := req.Msg.GetSudoNopasswd()
		newUser.SudoNopasswd = &sudoNopasswd
	}
	if req.Msg.Shell != nil {
		shell := req.Msg.GetShell()
		newUser.Shell = &shell
	}
	if req.Msg.LockPasswd != nil {
		lockPasswd := req.Msg.GetLockPasswd()
		newUser.LockPasswd = &lockPasswd
	}
	if req.Msg.Gecos != nil {
		gecos := req.Msg.GetGecos()
		newUser.Gecos = &gecos
	}

	// Add user to config
	cloudInitConfig.Users = append(cloudInitConfig.Users, newUser)

	// Save updated config
	if err := s.saveCloudInitConfigForVPS(ctx, vpsID, cloudInitConfig); err != nil {
		return nil, err
	}

	// Convert to proto for response
	hasPassword := newUser.Password != nil && *newUser.Password != ""
	userProto := &vpsv1.VPSUser{
		Name:              newUser.Name,
		HasPassword:       hasPassword,
		SshAuthorizedKeys: newUser.SSHAuthorizedKeys,
		Sudo:              newUser.Sudo != nil && *newUser.Sudo,
		SudoNopasswd:      newUser.SudoNopasswd != nil && *newUser.SudoNopasswd,
		Groups:            newUser.Groups,
		Shell:             newUser.Shell,
		LockPasswd:        newUser.LockPasswd != nil && *newUser.LockPasswd,
		Gecos:             newUser.Gecos,
	}

	return connect.NewResponse(&vpsv1.CreateVPSUserResponse{
		User:    userProto,
		Message: "User created. The user will be created on the next reboot or when cloud-init is re-run.",
	}), nil
}

// UpdateVPSUser updates an existing user on a VPS
func (s *ConfigService) UpdateVPSUser(ctx context.Context, req *connect.Request[vpsv1.UpdateVPSUserRequest]) (*connect.Response[vpsv1.UpdateVPSUserResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.update"); err != nil {
		return nil, err
	}

	username := req.Msg.GetName()
	if username == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("username is required"))
	}

	// Get current cloud-init config
	cloudInitConfig, err := s.getCloudInitConfigForVPS(ctx, vpsID)
	if err != nil {
		return nil, err
	}

	// Find user
	userIndex := -1
	for i, user := range cloudInitConfig.Users {
		if user.Name == username {
			userIndex = i
			break
		}
	}

	if userIndex == -1 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("user %s not found", username))
	}

	// Update user
	user := &cloudInitConfig.Users[userIndex]

	// Handle rename
	if req.Msg.NewName != nil && req.Msg.GetNewName() != "" && req.Msg.GetNewName() != username {
		// Check if new name already exists
		for _, existingUser := range cloudInitConfig.Users {
			if existingUser.Name == req.Msg.GetNewName() {
				return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("user %s already exists", req.Msg.GetNewName()))
			}
		}
		user.Name = req.Msg.GetNewName()
	}

	// Update SSH keys if provided
	if len(req.Msg.SshAuthorizedKeys) > 0 {
		sshKeys, err := s.resolveSSHKeyIDs(ctx, req.Msg.GetOrganizationId(), vpsID, req.Msg.SshAuthorizedKeys)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to resolve SSH keys: %w", err))
		}
		user.SSHAuthorizedKeys = sshKeys
	}

	if req.Msg.Sudo != nil {
		sudo := req.Msg.GetSudo()
		user.Sudo = &sudo
	}
	if req.Msg.SudoNopasswd != nil {
		sudoNopasswd := req.Msg.GetSudoNopasswd()
		user.SudoNopasswd = &sudoNopasswd
	}
	if req.Msg.Groups != nil {
		user.Groups = req.Msg.Groups
	}
	if req.Msg.Shell != nil {
		shell := req.Msg.GetShell()
		user.Shell = &shell
	}
	if req.Msg.LockPasswd != nil {
		lockPasswd := req.Msg.GetLockPasswd()
		user.LockPasswd = &lockPasswd
	}
	if req.Msg.Gecos != nil {
		gecos := req.Msg.GetGecos()
		user.Gecos = &gecos
	}

	// Save updated config
	if err := s.saveCloudInitConfigForVPS(ctx, vpsID, cloudInitConfig); err != nil {
		return nil, err
	}

	// Convert to proto for response
	hasPassword := user.Password != nil && *user.Password != ""
	userProto := &vpsv1.VPSUser{
		Name:              user.Name,
		HasPassword:       hasPassword,
		SshAuthorizedKeys: user.SSHAuthorizedKeys,
		Sudo:              user.Sudo != nil && *user.Sudo,
		SudoNopasswd:      user.SudoNopasswd != nil && *user.SudoNopasswd,
		Groups:            user.Groups,
		Shell:             user.Shell,
		LockPasswd:        user.LockPasswd != nil && *user.LockPasswd,
		Gecos:             user.Gecos,
	}

	return connect.NewResponse(&vpsv1.UpdateVPSUserResponse{
		User:    userProto,
		Message: "User updated. Changes will take effect on the next reboot or when cloud-init is re-run.",
	}), nil
}

// DeleteVPSUser deletes a user from a VPS
func (s *ConfigService) DeleteVPSUser(ctx context.Context, req *connect.Request[vpsv1.DeleteVPSUserRequest]) (*connect.Response[vpsv1.DeleteVPSUserResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.update"); err != nil {
		return nil, err
	}

	username := req.Msg.GetName()
	if username == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("username is required"))
	}
	if username == "root" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cannot delete root user"))
	}

	// Get current cloud-init config
	cloudInitConfig, err := s.getCloudInitConfigForVPS(ctx, vpsID)
	if err != nil {
		return nil, err
	}

	// Find and remove user
	userIndex := -1
	for i, user := range cloudInitConfig.Users {
		if user.Name == username {
			userIndex = i
			break
		}
	}

	if userIndex == -1 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("user %s not found", username))
	}

	// Remove user
	cloudInitConfig.Users = append(cloudInitConfig.Users[:userIndex], cloudInitConfig.Users[userIndex+1:]...)

	// Save updated config
	if err := s.saveCloudInitConfigForVPS(ctx, vpsID, cloudInitConfig); err != nil {
		return nil, err
	}

	return connect.NewResponse(&vpsv1.DeleteVPSUserResponse{
		Message: fmt.Sprintf("User %s deleted. The user will be removed on the next reboot or when cloud-init is re-run.", username),
	}), nil
}

// SetUserPassword sets or resets a user's password
func (s *ConfigService) SetUserPassword(ctx context.Context, req *connect.Request[vpsv1.SetUserPasswordRequest]) (*connect.Response[vpsv1.SetUserPasswordResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.update"); err != nil {
		return nil, err
	}

	username := req.Msg.GetUserName()
	password := req.Msg.GetPassword()

	if username == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("username is required"))
	}
	if password == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("password is required"))
	}

	// Get current cloud-init config
	cloudInitConfig, err := s.getCloudInitConfigForVPS(ctx, vpsID)
	if err != nil {
		return nil, err
	}

	// Find user
	userIndex := -1
	for i, user := range cloudInitConfig.Users {
		if user.Name == username {
			userIndex = i
			break
		}
	}

	if userIndex == -1 {
		// If user doesn't exist, create it
		newUser := orchestrator.CloudInitUser{
			Name:     username,
			Password: &password,
		}
		cloudInitConfig.Users = append(cloudInitConfig.Users, newUser)
	} else {
		// Update existing user's password
		cloudInitConfig.Users[userIndex].Password = &password
	}

	// Save updated config
	if err := s.saveCloudInitConfigForVPS(ctx, vpsID, cloudInitConfig); err != nil {
		return nil, err
	}

	return connect.NewResponse(&vpsv1.SetUserPasswordResponse{
		Message: fmt.Sprintf("Password for user %s updated. The password will take effect on the next reboot or when cloud-init is re-run.", username),
	}), nil
}

// UpdateUserSSHKeys updates SSH keys for a specific user
func (s *ConfigService) UpdateUserSSHKeys(ctx context.Context, req *connect.Request[vpsv1.UpdateUserSSHKeysRequest]) (*connect.Response[vpsv1.UpdateUserSSHKeysResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.update"); err != nil {
		return nil, err
	}

	username := req.Msg.GetUserName()
	if username == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("username is required"))
	}

	// Resolve SSH key IDs to public keys
	sshKeys, err := s.resolveSSHKeyIDs(ctx, req.Msg.GetOrganizationId(), vpsID, req.Msg.SshKeyIds)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("failed to resolve SSH keys: %w", err))
	}

	// Get current cloud-init config
	cloudInitConfig, err := s.getCloudInitConfigForVPS(ctx, vpsID)
	if err != nil {
		return nil, err
	}

	// Find user
	userIndex := -1
	for i, user := range cloudInitConfig.Users {
		if user.Name == username {
			userIndex = i
			break
		}
	}

	if userIndex == -1 {
		// If user doesn't exist, create it
		newUser := orchestrator.CloudInitUser{
			Name:              username,
			SSHAuthorizedKeys: sshKeys,
		}
		cloudInitConfig.Users = append(cloudInitConfig.Users, newUser)
		userIndex = len(cloudInitConfig.Users) - 1
	} else {
		// Update existing user's SSH keys
		cloudInitConfig.Users[userIndex].SSHAuthorizedKeys = sshKeys
	}

	// Save updated config
	if err := s.saveCloudInitConfigForVPS(ctx, vpsID, cloudInitConfig); err != nil {
		return nil, err
	}

	// Convert to proto for response
	user := &cloudInitConfig.Users[userIndex]
	hasPassword := user.Password != nil && *user.Password != ""
	userProto := &vpsv1.VPSUser{
		Name:              user.Name,
		HasPassword:       hasPassword,
		SshAuthorizedKeys: user.SSHAuthorizedKeys,
		Sudo:              user.Sudo != nil && *user.Sudo,
		SudoNopasswd:      user.SudoNopasswd != nil && *user.SudoNopasswd,
		Groups:            user.Groups,
		Shell:             user.Shell,
		LockPasswd:        user.LockPasswd != nil && *user.LockPasswd,
		Gecos:             user.Gecos,
	}

	return connect.NewResponse(&vpsv1.UpdateUserSSHKeysResponse{
		User:    userProto,
		Message: fmt.Sprintf("SSH keys for user %s updated. Changes will take effect on the next reboot or when cloud-init is re-run.", username),
	}), nil
}

// Helper functions

func (s *ConfigService) getCloudInitConfigForVPS(ctx context.Context, vpsID string) (*orchestrator.CloudInitConfig, error) {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return nil, err
	}
	return s.loadCloudInitConfig(ctx, &vps)
}

func (s *ConfigService) saveCloudInitConfigForVPS(ctx context.Context, vpsID string, config *orchestrator.CloudInitConfig) error {
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		return err
	}
	return s.saveCloudInitConfig(ctx, &vps, config)
}

func (s *ConfigService) loadCloudInitConfig(ctx context.Context, vps *database.VPSInstance) (*orchestrator.CloudInitConfig, error) {
	// If VPS is not provisioned yet, return empty config
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

	// Find the node where the VM is running
	nodeName, err := proxmoxClient.FindVMNode(ctx, vmIDInt)
	if err != nil {
		return nil, fmt.Errorf("failed to find VM node: %w", err)
	}

	// Try to get cloud-init config from Proxmox
	// Get VM config to check for cicustom parameter
	vmConfig, err := proxmoxClient.GetVMConfig(ctx, nodeName, vmIDInt)
	if err != nil {
		logger.Warn("[VPSConfigService] Failed to get VM config for VPS %s: %v. Returning default config.", vps.ID, err)
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

	// TODO: Parse the cloud-init snippet file to extract the actual config
	// For now, return a basic structure
	// In the future, we could:
	// 1. Store cloud-init config in database when saving
	// 2. Parse the snippet file from Proxmox storage
	// 3. Use Proxmox's cloud-init dump endpoint (if available)

	logger.Info("[VPSConfigService] VPS %s has custom cloud-init (cicustom: %s), but parsing not yet implemented. Returning default config.", vps.ID, cicustom)
	return &orchestrator.CloudInitConfig{
		Users:            []orchestrator.CloudInitUser{},
		PackageUpdate:    boolPtr(true),
		PackageUpgrade:   boolPtr(false),
		SSHInstallServer: boolPtr(true),
		SSHAllowPW:       boolPtr(true),
	}, nil
}

func (s *ConfigService) saveCloudInitConfig(ctx context.Context, vps *database.VPSInstance, config *orchestrator.CloudInitConfig) error {
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

	// Find the node where the VM is running
	nodeName, err := proxmoxClient.FindVMNode(ctx, vmIDInt)
	if err != nil {
		return fmt.Errorf("failed to find VM node: %w", err)
	}

	// Create VPSConfig for cloud-init generation
	vpsConfig := &orchestrator.VPSConfig{
		VPSID:          vps.ID,
		OrganizationID: vps.OrganizationID,
		CloudInit:      config,
	}

	// Generate cloud-init userData
	userData := orchestrator.GenerateCloudInitUserData(vpsConfig)

	// Get storage for snippets (use local storage by default)
	// Check if storage is configured in environment or use default
	storage := "local"
	if storageEnv := os.Getenv("PROXMOX_STORAGE"); storageEnv != "" {
		storage = storageEnv
	}

	// Upload cloud-init snippet
	snippetPath, err := proxmoxClient.CreateCloudInitSnippet(ctx, nodeName, storage, vmIDInt, userData)
	if err != nil {
		return fmt.Errorf("failed to create cloud-init snippet: %w", err)
	}

	// Update VM config with cicustom parameter
	if err := proxmoxClient.UpdateVMCicustom(ctx, nodeName, vmIDInt, snippetPath); err != nil {
		return fmt.Errorf("failed to update VM cicustom: %w", err)
	}

	logger.Info("[VPSConfigService] Updated cloud-init config for VPS %s (VM %d)", vps.ID, vmIDInt)

	return nil
}

func (s *ConfigService) resolveSSHKeyIDs(ctx context.Context, orgID, vpsID string, keyIDs []string) ([]string, error) {
	if len(keyIDs) == 0 {
		return []string{}, nil
	}

	// Get SSH keys from database
	var sshKeys []database.SSHKey
	query := database.DB.Where("organization_id = ?", orgID)

	// Build query for key IDs
	if len(keyIDs) > 0 {
		query = query.Where("id IN ?", keyIDs)
	}

	// Also include VPS-specific keys
	query = query.Where("(vps_id IS NULL OR vps_id = ?)", vpsID)

	if err := query.Find(&sshKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to query SSH keys: %w", err)
	}

	// Extract public keys
	publicKeys := make([]string, 0, len(sshKeys))
	for _, key := range sshKeys {
		if key.PublicKey != "" {
			publicKeys = append(publicKeys, key.PublicKey)
		}
	}

	return publicKeys, nil
}

// Conversion helpers

func protoToCloudInitConfig(proto *vpsv1.CloudInitConfig) *orchestrator.CloudInitConfig {
	if proto == nil {
		return nil
	}

	config := &orchestrator.CloudInitConfig{
		Users:      make([]orchestrator.CloudInitUser, 0, len(proto.Users)),
		Packages:   proto.Packages,
		Runcmd:     proto.Runcmd,
		WriteFiles: make([]orchestrator.CloudInitWriteFile, 0, len(proto.WriteFiles)),
	}

	// Convert users
	for _, userProto := range proto.Users {
		user := orchestrator.CloudInitUser{
			Name:              userProto.GetName(),
			SSHAuthorizedKeys: userProto.SshAuthorizedKeys,
			Groups:            userProto.Groups,
		}

		if userProto.Password != nil {
			pass := userProto.GetPassword()
			user.Password = &pass
		}
		if userProto.Sudo != nil {
			sudo := userProto.GetSudo()
			user.Sudo = &sudo
		}
		if userProto.SudoNopasswd != nil {
			sudoNopasswd := userProto.GetSudoNopasswd()
			user.SudoNopasswd = &sudoNopasswd
		}
		if userProto.Shell != nil {
			shell := userProto.GetShell()
			user.Shell = &shell
		}
		if userProto.LockPasswd != nil {
			lockPasswd := userProto.GetLockPasswd()
			user.LockPasswd = &lockPasswd
		}
		if userProto.Gecos != nil {
			gecos := userProto.GetGecos()
			user.Gecos = &gecos
		}

		config.Users = append(config.Users, user)
	}

	// Convert system configuration
	if proto.Hostname != nil {
		hostname := proto.GetHostname()
		config.Hostname = &hostname
	}
	if proto.Timezone != nil {
		timezone := proto.GetTimezone()
		config.Timezone = &timezone
	}
	if proto.Locale != nil {
		locale := proto.GetLocale()
		config.Locale = &locale
	}

	// Convert package management
	if proto.PackageUpdate != nil {
		packageUpdate := proto.GetPackageUpdate()
		config.PackageUpdate = &packageUpdate
	}
	if proto.PackageUpgrade != nil {
		packageUpgrade := proto.GetPackageUpgrade()
		config.PackageUpgrade = &packageUpgrade
	}

	// Convert SSH configuration
	if proto.SshInstallServer != nil {
		sshInstallServer := proto.GetSshInstallServer()
		config.SSHInstallServer = &sshInstallServer
	}
	if proto.SshAllowPw != nil {
		sshAllowPW := proto.GetSshAllowPw()
		config.SSHAllowPW = &sshAllowPW
	}

	// Convert write files
	for _, fileProto := range proto.WriteFiles {
		file := orchestrator.CloudInitWriteFile{
			Path:    fileProto.GetPath(),
			Content: fileProto.GetContent(),
		}

		if fileProto.Owner != nil {
			owner := fileProto.GetOwner()
			file.Owner = &owner
		}
		if fileProto.Permissions != nil {
			permissions := fileProto.GetPermissions()
			file.Permissions = &permissions
		}
		if fileProto.Append != nil {
			appendVal := fileProto.GetAppend()
			file.Append = &appendVal
		}
		if fileProto.Defer != nil {
			deferVal := fileProto.GetDefer()
			file.Defer = &deferVal
		}

		config.WriteFiles = append(config.WriteFiles, file)
	}

	return config
}

func cloudInitConfigToProto(config *orchestrator.CloudInitConfig) *vpsv1.CloudInitConfig {
	if config == nil {
		return nil
	}

	proto := &vpsv1.CloudInitConfig{
		Users:      make([]*vpsv1.CloudInitUser, 0, len(config.Users)),
		Packages:   config.Packages,
		Runcmd:     config.Runcmd,
		WriteFiles: make([]*vpsv1.CloudInitWriteFile, 0, len(config.WriteFiles)),
	}

	// Convert users (without passwords for security)
	for _, user := range config.Users {
		userProto := &vpsv1.CloudInitUser{
			Name:              user.Name,
			SshAuthorizedKeys: user.SSHAuthorizedKeys,
			Groups:            user.Groups,
		}

		// Don't include password in response (security)
		if user.Sudo != nil {
			sudo := *user.Sudo
			userProto.Sudo = &sudo
		}
		if user.SudoNopasswd != nil {
			sudoNopasswd := *user.SudoNopasswd
			userProto.SudoNopasswd = &sudoNopasswd
		}
		if user.Shell != nil {
			shell := *user.Shell
			userProto.Shell = &shell
		}
		if user.LockPasswd != nil {
			lockPasswd := *user.LockPasswd
			userProto.LockPasswd = &lockPasswd
		}
		if user.Gecos != nil {
			gecos := *user.Gecos
			userProto.Gecos = &gecos
		}

		proto.Users = append(proto.Users, userProto)
	}

	// Convert system configuration
	if config.Hostname != nil {
		hostname := *config.Hostname
		proto.Hostname = &hostname
	}
	if config.Timezone != nil {
		timezone := *config.Timezone
		proto.Timezone = &timezone
	}
	if config.Locale != nil {
		locale := *config.Locale
		proto.Locale = &locale
	}

	// Convert package management
	if config.PackageUpdate != nil {
		packageUpdate := *config.PackageUpdate
		proto.PackageUpdate = &packageUpdate
	}
	if config.PackageUpgrade != nil {
		packageUpgrade := *config.PackageUpgrade
		proto.PackageUpgrade = &packageUpgrade
	}

	// Convert SSH configuration
	if config.SSHInstallServer != nil {
		sshInstallServer := *config.SSHInstallServer
		proto.SshInstallServer = &sshInstallServer
	}
	if config.SSHAllowPW != nil {
		sshAllowPW := *config.SSHAllowPW
		proto.SshAllowPw = &sshAllowPW
	}

	// Convert write files
	for _, file := range config.WriteFiles {
		fileProto := &vpsv1.CloudInitWriteFile{
			Path:    file.Path,
			Content: file.Content,
		}

		if file.Owner != nil {
			owner := *file.Owner
			fileProto.Owner = &owner
		}
		if file.Permissions != nil {
			permissions := *file.Permissions
			fileProto.Permissions = &permissions
		}
		if file.Append != nil {
			appendVal := *file.Append
			fileProto.Append = &appendVal
		}
		if file.Defer != nil {
			deferVal := *file.Defer
			fileProto.Defer = &deferVal
		}

		proto.WriteFiles = append(proto.WriteFiles, fileProto)
	}

	return proto
}

func boolPtr(b bool) *bool {
	return &b
}

// RotateTerminalKey rotates the web terminal SSH key for a VPS
func (s *ConfigService) RotateTerminalKey(ctx context.Context, req *connect.Request[vpsv1.RotateTerminalKeyRequest]) (*connect.Response[vpsv1.RotateTerminalKeyResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.update"); err != nil {
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

	// Rotate the terminal key (generates new key pair, or creates if it doesn't exist)
	terminalKey, err := database.RotateVPSTerminalKey(vpsID, vps.OrganizationID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to rotate terminal key: %w", err))
	}

	// Get current cloud-init config to preserve existing settings
	cloudInitConfig, err := s.loadCloudInitConfig(ctx, &vps)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load cloud-init config: %w", err))
	}

	// Regenerate cloud-init config (this will automatically include the new terminal key)
	// The generateCloudInitUserData function automatically adds the terminal key to root's SSH keys
	if err := s.saveCloudInitConfig(ctx, &vps, cloudInitConfig); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update cloud-init config: %w", err))
	}

	logger.Info("[ConfigService] Rotated terminal key for VPS %s (new fingerprint: %s)", vpsID, terminalKey.Fingerprint)

	return connect.NewResponse(&vpsv1.RotateTerminalKeyResponse{
		Fingerprint: terminalKey.Fingerprint,
		Message:     "Terminal key rotated. The new key will take effect on the next reboot or when cloud-init is re-run.",
	}), nil
}

// RemoveTerminalKey removes the web terminal SSH key for a VPS
func (s *ConfigService) RemoveTerminalKey(ctx context.Context, req *connect.Request[vpsv1.RemoveTerminalKeyRequest]) (*connect.Response[vpsv1.RemoveTerminalKeyResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.update"); err != nil {
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

	// Check if terminal key exists
	_, err = database.GetVPSTerminalKey(vpsID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("terminal key not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get terminal key: %w", err))
	}

	// Delete terminal key from database
	if err := database.DeleteVPSTerminalKey(vpsID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete terminal key: %w", err))
	}

	// Get current cloud-init config to preserve existing settings
	cloudInitConfig, err := s.loadCloudInitConfig(ctx, &vps)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load cloud-init config: %w", err))
	}

	// Regenerate cloud-init config (terminal key will no longer be included since it's deleted)
	if err := s.saveCloudInitConfig(ctx, &vps, cloudInitConfig); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update cloud-init config: %w", err))
	}

	logger.Info("[ConfigService] Removed terminal key for VPS %s", vpsID)

	return connect.NewResponse(&vpsv1.RemoveTerminalKeyResponse{
		Message: "Terminal key removed. The key will be removed on the next reboot or when cloud-init is re-run. Web terminal access will no longer work until a new key is generated.",
	}), nil
}
