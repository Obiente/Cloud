package vps

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"api/internal/auth"
	"api/internal/database"
	"api/internal/logger"
	"api/internal/orchestrator"
	"api/internal/services/common"

	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"
	vpsv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1/vpsv1connect"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v3"
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

// GetCloudInitUserData retrieves the actual generated cloud-init userData for a VPS
// This includes bastion and terminal keys that are dynamically added
func (s *ConfigService) GetCloudInitUserData(ctx context.Context, req *connect.Request[vpsv1.GetCloudInitUserDataRequest]) (*connect.Response[vpsv1.GetCloudInitUserDataResponse], error) {
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

	// Load cloud-init config
	cloudInitConfig, err := s.loadCloudInitConfig(ctx, &vps)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load cloud-init config: %w", err))
	}

	// Generate the actual cloud-init userData (includes bastion/terminal keys)
	vpsConfig := &orchestrator.VPSConfig{
		VPSID:          vps.ID,
		OrganizationID: vps.OrganizationID,
		CloudInit:      cloudInitConfig,
	}
	userData := orchestrator.GenerateCloudInitUserData(vpsConfig)

	return connect.NewResponse(&vpsv1.GetCloudInitUserDataResponse{
		UserData: userData,
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

// LoadCloudInitConfig loads cloud-init configuration for a VPS (public method for superadmin use)
func (s *ConfigService) LoadCloudInitConfig(ctx context.Context, vps *database.VPSInstance) (*orchestrator.CloudInitConfig, error) {
	return s.loadCloudInitConfig(ctx, vps)
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

	// Parse cicustom parameter (format: "user=local:snippets/vm-301-user-data")
	// Extract storage and filename
	var storage, filename string
	if strings.HasPrefix(cicustom, "user=") {
		parts := strings.SplitN(cicustom[5:], ":", 2)
		if len(parts) == 2 {
			storage = parts[0]
			snippetPath := parts[1]
			// Extract filename from path (e.g., "snippets/vm-301-user-data" -> "vm-301-user-data")
			if lastSlash := strings.LastIndex(snippetPath, "/"); lastSlash >= 0 {
				filename = snippetPath[lastSlash+1:]
			} else {
				filename = snippetPath
			}
		}
	}

	if storage == "" || filename == "" {
		logger.Warn("[VPSConfigService] Failed to parse cicustom parameter '%s' for VPS %s. Returning default config.", cicustom, vps.ID)
		return &orchestrator.CloudInitConfig{
			Users:            []orchestrator.CloudInitUser{},
			PackageUpdate:    boolPtr(true),
			PackageUpgrade:   boolPtr(false),
			SSHInstallServer: boolPtr(true),
			SSHAllowPW:       boolPtr(true),
		}, nil
	}

	// Read snippet file via SSH
	userData, err := proxmoxClient.ReadSnippetViaSSH(ctx, nodeName, storage, filename)
	if err != nil {
		logger.Warn("[VPSConfigService] Failed to read cloud-init snippet for VPS %s: %v. Returning default config.", vps.ID, err)
		return &orchestrator.CloudInitConfig{
			Users:            []orchestrator.CloudInitUser{},
			PackageUpdate:    boolPtr(true),
			PackageUpgrade:   boolPtr(false),
			SSHInstallServer: boolPtr(true),
			SSHAllowPW:       boolPtr(true),
		}, nil
	}

	// Parse YAML to extract cloud-init config
	config, err := parseCloudInitYAML(userData)
	if err != nil {
		logger.Warn("[VPSConfigService] Failed to parse cloud-init YAML for VPS %s: %v. This may indicate malformed YAML in the snippet. Returning default config. Note: Bastion and terminal keys will still be included when regenerating cloud-init.", vps.ID, err)
		// Return default config - GenerateCloudInitUserData will still add bastion/terminal keys from DB
		return &orchestrator.CloudInitConfig{
			Users:            []orchestrator.CloudInitUser{},
			PackageUpdate:    boolPtr(true),
			PackageUpgrade:   boolPtr(false),
			SSHInstallServer: boolPtr(true),
			SSHAllowPW:       boolPtr(true),
		}, nil
	}

	logger.Info("[VPSConfigService] Successfully parsed cloud-init config for VPS %s from snippet", vps.ID)
	return config, nil
}

// SaveCloudInitConfig saves cloud-init configuration for a VPS (public method for superadmin use)
func (s *ConfigService) SaveCloudInitConfig(ctx context.Context, vps *database.VPSInstance, config *orchestrator.CloudInitConfig) error {
	return s.saveCloudInitConfig(ctx, vps, config)
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

	// Load existing config and merge with new config
	existingConfig, err := s.loadCloudInitConfig(ctx, vps)
	if err != nil {
		logger.Warn("[VPSConfigService] Failed to load existing cloud-init config for VPS %s: %v. Using new config only.", vps.ID, err)
		existingConfig = &orchestrator.CloudInitConfig{}
	}

	// Merge configs (new config takes precedence, but preserve fields not in new config)
	mergedConfig := mergeCloudInitConfig(existingConfig, config)

	// Create VPSConfig for cloud-init generation
	vpsConfig := &orchestrator.VPSConfig{
		VPSID:          vps.ID,
		OrganizationID: vps.OrganizationID,
		CloudInit:      mergedConfig,
	}

	// Generate cloud-init userData
	userData := orchestrator.GenerateCloudInitUserData(vpsConfig)
	
	// Log bastion key inclusion for debugging
	bastionKey, err := database.GetVPSBastionKey(vps.ID)
	if err == nil {
		logger.Debug("[VPSConfigService] Cloud-init userData includes bastion key for VPS %s (fingerprint: %s)", vps.ID, bastionKey.Fingerprint)
	} else {
		logger.Warn("[VPSConfigService] Cloud-init userData does NOT include bastion key for VPS %s: %v", vps.ID, err)
	}

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

	logger.Info("[VPSConfigService] Updated cloud-init config for VPS %s (VM %d). Changes will be applied on next boot.", vps.ID, vmIDInt)

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

// parseCloudInitYAML parses cloud-init YAML userData into CloudInitConfig
func parseCloudInitYAML(userData string) (*orchestrator.CloudInitConfig, error) {
	// Remove #cloud-config header if present
	content := userData
	if strings.HasPrefix(content, "#cloud-config") {
		lines := strings.SplitN(content, "\n", 2)
		if len(lines) > 1 {
			content = lines[1]
		} else {
			content = ""
		}
		// Also remove empty lines after header
		content = strings.TrimLeft(content, "\n\r")
	}
	
	// Clean up any potential encoding issues (BOM, carriage returns, etc.)
	// Remove BOM if present
	if len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF {
		content = content[3:]
	}
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	var yamlData map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &yamlData); err != nil {
		// Try to extract line number from error message
		errorMsg := err.Error()
		logger.Warn("[VPSConfigService] YAML parse error: %s", errorMsg)
		
		// Extract line number from error message if present (format: "yaml: line X: ...")
		lines := strings.Split(content, "\n")
		errorLineNum := -1
		if strings.Contains(errorMsg, "line ") {
			// Try to extract line number
			parts := strings.Split(errorMsg, "line ")
			if len(parts) > 1 {
				linePart := strings.Split(parts[1], ":")[0]
				fmt.Sscanf(linePart, "%d", &errorLineNum)
			}
		}
		
		// Log the problematic area (use WARN level so it's always visible)
		if errorLineNum > 0 && len(lines) >= errorLineNum {
			logger.Warn("[VPSConfigService] YAML parse error at line %d. Content around error:", errorLineNum)
			start := errorLineNum - 3
			if start < 0 {
				start = 0
			}
			end := errorLineNum + 3
			if end > len(lines) {
				end = len(lines)
			}
			for i := start; i < end; i++ {
				lineContent := lines[i]
				// Show first 120 chars to avoid huge logs
				if len(lineContent) > 120 {
					lineContent = lineContent[:120] + "..."
				}
				marker := ""
				if i+1 == errorLineNum {
					marker = " <-- ERROR HERE"
				}
				logger.Warn("[VPSConfigService]   Line %d: %q%s", i+1, lineContent, marker)
			}
		} else if len(lines) > 0 {
			// Fallback: log first few lines if we couldn't extract line number
			logger.Warn("[VPSConfigService] YAML content preview (first 10 lines):")
			for i := 0; i < len(lines) && i < 10; i++ {
				lineContent := lines[i]
				if len(lineContent) > 120 {
					lineContent = lineContent[:120] + "..."
				}
				logger.Warn("[VPSConfigService]   Line %d: %q", i+1, lineContent)
			}
		}
		
		// Return error with more context
		return nil, fmt.Errorf("failed to parse YAML (this may indicate corrupted or malformed cloud-init snippet): %w", err)
	}

	config := &orchestrator.CloudInitConfig{
		Users:            []orchestrator.CloudInitUser{},
		PackageUpdate:    boolPtr(true),
		PackageUpgrade:   boolPtr(false),
		SSHInstallServer: boolPtr(true),
		SSHAllowPW:       boolPtr(true),
	}

	// Parse SSH configuration
	if sshVal, ok := yamlData["ssh"].(map[string]interface{}); ok {
		if installServer, ok := sshVal["install-server"].(bool); ok {
			config.SSHInstallServer = &installServer
		}
		if allowPW, ok := sshVal["allow-pw"].(bool); ok {
			config.SSHAllowPW = &allowPW
		}
	}

	// Parse hostname
	if hostname, ok := yamlData["hostname"].(string); ok {
		config.Hostname = &hostname
	}

	// Parse timezone
	if timezone, ok := yamlData["timezone"].(string); ok {
		config.Timezone = &timezone
	}

	// Parse locale
	if locale, ok := yamlData["locale"].(string); ok {
		config.Locale = &locale
	}

	// Parse packages
	if packages, ok := yamlData["packages"].([]interface{}); ok {
		config.Packages = make([]string, 0, len(packages))
		for _, pkg := range packages {
			if pkgStr, ok := pkg.(string); ok {
				config.Packages = append(config.Packages, pkgStr)
			}
		}
	}

	// Parse package_update
	if packageUpdate, ok := yamlData["package_update"].(bool); ok {
		config.PackageUpdate = &packageUpdate
	}

	// Parse package_upgrade
	if packageUpgrade, ok := yamlData["package_upgrade"].(bool); ok {
		config.PackageUpgrade = &packageUpgrade
	}

	// Parse users (skip root user as it is handled separately)
	if users, ok := yamlData["users"].([]interface{}); ok {
		for _, userVal := range users {
			if userMap, ok := userVal.(map[string]interface{}); ok {
				name, _ := userMap["name"].(string)
				if name == "root" {
					continue
				}

				user := orchestrator.CloudInitUser{Name: name}
				if passwd, ok := userMap["passwd"].(string); ok {
					user.Password = &passwd
				}
				if sshKeys, ok := userMap["ssh_authorized_keys"].([]interface{}); ok {
					user.SSHAuthorizedKeys = make([]string, 0, len(sshKeys))
					for _, key := range sshKeys {
						if keyStr, ok := key.(string); ok {
							user.SSHAuthorizedKeys = append(user.SSHAuthorizedKeys, keyStr)
						}
					}
				}
				if sudo, ok := userMap["sudo"].(string); ok {
					sudoBool := sudo == "ALL" || sudo == "NOPASSWD: ALL"
					user.Sudo = &sudoBool
					if strings.Contains(sudo, "NOPASSWD") {
						nopasswd := true
						user.SudoNopasswd = &nopasswd
					}
				}
				if groups, ok := userMap["groups"].([]interface{}); ok {
					user.Groups = make([]string, 0, len(groups))
					for _, group := range groups {
						if groupStr, ok := group.(string); ok {
							user.Groups = append(user.Groups, groupStr)
						}
					}
				}
				if shell, ok := userMap["shell"].(string); ok {
					user.Shell = &shell
				}
				if lockPasswd, ok := userMap["lock_passwd"].(bool); ok {
					user.LockPasswd = &lockPasswd
				}
				if gecos, ok := userMap["gecos"].(string); ok {
					user.Gecos = &gecos
				}
				config.Users = append(config.Users, user)
			}
		}
	}

	// Parse runcmd
	if runcmd, ok := yamlData["runcmd"].([]interface{}); ok {
		config.Runcmd = make([]string, 0, len(runcmd))
		for _, cmd := range runcmd {
			if cmdStr, ok := cmd.(string); ok {
				config.Runcmd = append(config.Runcmd, cmdStr)
			} else if cmdList, ok := cmd.([]interface{}); ok {
				cmdParts := make([]string, 0, len(cmdList))
				for _, part := range cmdList {
					if partStr, ok := part.(string); ok {
						cmdParts = append(cmdParts, partStr)
					}
				}
				if len(cmdParts) > 0 {
					config.Runcmd = append(config.Runcmd, strings.Join(cmdParts, " "))
				}
			}
		}
	}

	// Parse write_files
	if writeFiles, ok := yamlData["write_files"].([]interface{}); ok {
		config.WriteFiles = make([]orchestrator.CloudInitWriteFile, 0, len(writeFiles))
		for _, fileVal := range writeFiles {
			if fileMap, ok := fileVal.(map[string]interface{}); ok {
				file := orchestrator.CloudInitWriteFile{}
				if path, ok := fileMap["path"].(string); ok {
					file.Path = path
				}
				if content, ok := fileMap["content"].(string); ok {
					file.Content = content
				}
				if owner, ok := fileMap["owner"].(string); ok {
					file.Owner = &owner
				}
				if permissions, ok := fileMap["permissions"].(string); ok {
					file.Permissions = &permissions
				}
				if appendVal, ok := fileMap["append"].(bool); ok {
					file.Append = &appendVal
					file.Append = &appendVal
				}
				if deferVal, ok := fileMap["defer"].(bool); ok {
					file.Defer = &deferVal
					file.Defer = &deferVal
				}
				config.WriteFiles = append(config.WriteFiles, file)
			}
		}
	}

	return config, nil
}

// mergeCloudInitConfig merges existing config with new config
func mergeCloudInitConfig(existing, new *orchestrator.CloudInitConfig) *orchestrator.CloudInitConfig {
	merged := &orchestrator.CloudInitConfig{}
	if new.Hostname != nil {
		merged.Hostname = new.Hostname
	} else {
		merged.Hostname = existing.Hostname
	}
	if new.Timezone != nil {
		merged.Timezone = new.Timezone
	} else {
		merged.Timezone = existing.Timezone
	}
	if new.Locale != nil {
		merged.Locale = new.Locale
	} else {
		merged.Locale = existing.Locale
	}
	if len(new.Packages) > 0 {
		merged.Packages = new.Packages
	} else {
		merged.Packages = existing.Packages
	}
	if new.PackageUpdate != nil {
		merged.PackageUpdate = new.PackageUpdate
	} else {
		merged.PackageUpdate = existing.PackageUpdate
	}
	if new.PackageUpgrade != nil {
		merged.PackageUpgrade = new.PackageUpgrade
	} else {
		merged.PackageUpgrade = existing.PackageUpgrade
	}
	if new.SSHInstallServer != nil {
		merged.SSHInstallServer = new.SSHInstallServer
	} else {
		merged.SSHInstallServer = existing.SSHInstallServer
	}
	if new.SSHAllowPW != nil {
		merged.SSHAllowPW = new.SSHAllowPW
	} else {
		merged.SSHAllowPW = existing.SSHAllowPW
	}
	if len(new.Users) > 0 {
		merged.Users = new.Users
	} else {
		merged.Users = existing.Users
	}
	if len(new.Runcmd) > 0 {
		merged.Runcmd = new.Runcmd
	} else {
		merged.Runcmd = existing.Runcmd
	}
	if len(new.WriteFiles) > 0 {
		merged.WriteFiles = new.WriteFiles
	} else {
		merged.WriteFiles = existing.WriteFiles
	}
	return merged
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

// GetTerminalKey gets the web terminal SSH key status for a VPS
func (s *ConfigService) GetTerminalKey(ctx context.Context, req *connect.Request[vpsv1.GetTerminalKeyRequest]) (*connect.Response[vpsv1.GetTerminalKeyResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.read"); err != nil {
		return nil, err
	}

	// Get terminal key
	terminalKey, err := database.GetVPSTerminalKey(vpsID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("terminal key not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get terminal key: %w", err))
	}

	return connect.NewResponse(&vpsv1.GetTerminalKeyResponse{
		Fingerprint: terminalKey.Fingerprint,
		CreatedAt:   timestamppb.New(terminalKey.CreatedAt),
		UpdatedAt:   timestamppb.New(terminalKey.UpdatedAt),
	}), nil
}

// RotateBastionKey rotates the bastion SSH key for a VPS
func (s *ConfigService) RotateBastionKey(ctx context.Context, req *connect.Request[vpsv1.RotateBastionKeyRequest]) (*connect.Response[vpsv1.RotateBastionKeyResponse], error) {
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

	// Rotate the bastion key (generates new key pair, or creates if it doesn't exist)
	bastionKey, err := database.RotateVPSBastionKey(vpsID, vps.OrganizationID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to rotate bastion key: %w", err))
	}

	// Get current cloud-init config to preserve existing settings
	cloudInitConfig, err := s.loadCloudInitConfig(ctx, &vps)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load cloud-init config: %w", err))
	}

	// Regenerate cloud-init config (this will automatically include the new bastion key)
	// saveCloudInitConfig will also trigger cloud-init regeneration
	if err := s.saveCloudInitConfig(ctx, &vps, cloudInitConfig); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update cloud-init config: %w", err))
	}

	logger.Info("[ConfigService] Rotated bastion key for VPS %s (new fingerprint: %s)", vpsID, bastionKey.Fingerprint)

	return connect.NewResponse(&vpsv1.RotateBastionKeyResponse{
		Fingerprint: bastionKey.Fingerprint,
		Message:     "Bastion key rotated. The new key will take effect on the next reboot or when cloud-init is re-run. If the connection still fails, try running 'cloud-init clean' on the VPS and rebooting.",
	}), nil
}

// GetBastionKey gets the bastion SSH key status for a VPS
func (s *ConfigService) GetBastionKey(ctx context.Context, req *connect.Request[vpsv1.GetBastionKeyRequest]) (*connect.Response[vpsv1.GetBastionKeyResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.read"); err != nil {
		return nil, err
	}

	// Get bastion key
	bastionKey, err := database.GetVPSBastionKey(vpsID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("bastion key not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get bastion key: %w", err))
	}

	return connect.NewResponse(&vpsv1.GetBastionKeyResponse{
		Fingerprint: bastionKey.Fingerprint,
		CreatedAt:   timestamppb.New(bastionKey.CreatedAt),
		UpdatedAt:   timestamppb.New(bastionKey.UpdatedAt),
	}), nil
}

// GetSSHAlias gets the SSH alias for a VPS
func (s *ConfigService) GetSSHAlias(ctx context.Context, req *connect.Request[vpsv1.GetSSHAliasRequest]) (*connect.Response[vpsv1.GetSSHAliasResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.read"); err != nil {
		return nil, err
	}

	// Get VPS to retrieve alias
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	var alias *string
	if vps.SSHAlias != nil && *vps.SSHAlias != "" {
		alias = vps.SSHAlias
	}

	return connect.NewResponse(&vpsv1.GetSSHAliasResponse{
		Alias: alias,
	}), nil
}

// SetSSHAlias sets the SSH alias for a VPS
func (s *ConfigService) SetSSHAlias(ctx context.Context, req *connect.Request[vpsv1.SetSSHAliasRequest]) (*connect.Response[vpsv1.SetSSHAliasResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	alias := req.Msg.GetAlias()

	if err := s.checkVPSPermission(ctx, vpsID, "vps.write"); err != nil {
		return nil, err
	}

	// Validate alias format (alphanumeric, hyphens, underscores, 1-63 chars)
	if alias == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("alias cannot be empty"))
	}
	if len(alias) > 63 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("alias must be 63 characters or less"))
	}
	// Check if alias contains only allowed characters
	for _, r := range alias {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("alias can only contain alphanumeric characters, hyphens, and underscores"))
		}
	}
	// Alias cannot start with "vps-" to avoid confusion with VPS IDs
	if len(alias) >= 4 && alias[:4] == "vps-" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("alias cannot start with 'vps-'"))
	}

	// Get VPS to check organization
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	// Check if alias is already in use by another VPS
	var existingVPS database.VPSInstance
	if err := database.DB.Where("ssh_alias = ? AND id != ? AND deleted_at IS NULL", alias, vpsID).First(&existingVPS).Error; err == nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("SSH alias '%s' is already in use by VPS %s", alias, existingVPS.ID))
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check alias availability: %w", err))
	}

	// Update VPS with new alias
	vps.SSHAlias = &alias
	if err := database.DB.Save(&vps).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to set SSH alias: %w", err))
	}

	logger.Info("[ConfigService] Set SSH alias '%s' for VPS %s", alias, vpsID)

	return connect.NewResponse(&vpsv1.SetSSHAliasResponse{
		Alias:   alias,
		Message: fmt.Sprintf("SSH alias '%s' has been set. You can now connect using: ssh -p 2323 root@%s@localhost", alias, alias),
	}), nil
}

// RemoveSSHAlias removes the SSH alias for a VPS
func (s *ConfigService) RemoveSSHAlias(ctx context.Context, req *connect.Request[vpsv1.RemoveSSHAliasRequest]) (*connect.Response[vpsv1.RemoveSSHAliasResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, "vps.write"); err != nil {
		return nil, err
	}

	// Get VPS
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	// Remove alias
	vps.SSHAlias = nil
	if err := database.DB.Save(&vps).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to remove SSH alias: %w", err))
	}

	logger.Info("[ConfigService] Removed SSH alias for VPS %s", vpsID)

	return connect.NewResponse(&vpsv1.RemoveSSHAliasResponse{
		Message: "SSH alias has been removed. You can still connect using the full VPS ID.",
	}), nil
}
