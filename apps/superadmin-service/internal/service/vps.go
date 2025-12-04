package superadmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/services/organizations"
	vpsorch "github.com/obiente/cloud/apps/vps-service/orchestrator"
	vpsservice "github.com/obiente/cloud/apps/vps-service/pkg/service"

	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	superadminv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/superadmin/v1"
	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// Helper functions for VPS operations
func boolPtr(b bool) *bool {
	return &b
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// protoToCloudInitConfigForSuperadmin converts proto CloudInitConfig to orchestrator format
func protoToCloudInitConfigForSuperadmin(proto *vpsv1.CloudInitConfig) *vpsorch.CloudInitConfig {
	if proto == nil {
		return nil
	}

	config := &vpsorch.CloudInitConfig{
		Users:      make([]vpsorch.CloudInitUser, 0, len(proto.Users)),
		Packages:   proto.Packages,
		Runcmd:     proto.Runcmd,
		WriteFiles: make([]vpsorch.CloudInitWriteFile, 0, len(proto.WriteFiles)),
	}

	// Convert users
	for _, userProto := range proto.Users {
		user := vpsorch.CloudInitUser{
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
		file := vpsorch.CloudInitWriteFile{
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
			append := fileProto.GetAppend()
			file.Append = &append
		}
		if fileProto.Defer != nil {
			deferVal := fileProto.GetDefer()
			file.Defer = &deferVal
		}

		config.WriteFiles = append(config.WriteFiles, file)
	}

	return config
}

// ListAllVPS lists all VPS instances across all organizations (superadmin only)
func (s *Service) ListAllVPS(ctx context.Context, req *connect.Request[superadminv1.ListAllVPSRequest]) (*connect.Response[superadminv1.ListAllVPSResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	// Parse pagination
	page := int(req.Msg.GetPage())
	if page < 1 {
		page = 1
	}
	perPage := int(req.Msg.GetPerPage())
	if perPage < 1 {
		perPage = 50
	}
	if perPage > 100 {
		perPage = 100
	}
	offset := (page - 1) * perPage

	// Build query
	query := database.DB.Table("vps_instances v").
		Select(`
			v.*,
			o.name as organization_name
		`).
		Joins("LEFT JOIN organizations o ON o.id = v.organization_id").
		Where("v.deleted_at IS NULL")

	// Apply filters
	if orgID := req.Msg.GetOrganizationId(); orgID != "" {
		query = query.Where("v.organization_id = ?", orgID)
	}

	if req.Msg.Status != nil {
		status := req.Msg.GetStatus()
		if status != vpsv1.VPSStatus_VPS_STATUS_UNSPECIFIED {
			query = query.Where("v.status = ?", int32(status))
		}
	}

	// Apply search filter
	if search := strings.TrimSpace(req.Msg.GetSearch()); search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where(
			"v.name ILIKE ? OR v.id ILIKE ? OR v.organization_id ILIKE ? OR v.region ILIKE ? OR v.size ILIKE ? OR o.name ILIKE ?",
			searchTerm, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm,
		)
	}

	// Get total count
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to count VPS instances: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to count VPS instances: %w", err))
	}

	// Apply pagination and ordering
	var vpsRows []struct {
		database.VPSInstance
		OrganizationName string `gorm:"column:organization_name"`
	}

	if err := query.Order("v.created_at DESC").Limit(perPage).Offset(offset).Find(&vpsRows).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to list VPS instances: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list VPS instances: %w", err))
	}

	// Convert to proto
	vpsOverviews := make([]*superadminv1.VPSOverview, 0, len(vpsRows))
	for _, row := range vpsRows {
		// Convert database model to proto
		vpsProto := convertVPSInstanceToProto(&row.VPSInstance)
		vpsOverviews = append(vpsOverviews, &superadminv1.VPSOverview{
			Vps:              vpsProto,
			OrganizationName: row.OrganizationName,
		})
	}

	// Calculate pagination
	totalPages := int(math.Ceil(float64(totalCount) / float64(perPage)))

	return connect.NewResponse(&superadminv1.ListAllVPSResponse{
		VpsInstances: vpsOverviews,
		Pagination: &commonv1.Pagination{
			Page:       int32(page),
			PerPage:    int32(perPage),
			Total:      int32(totalCount),
			TotalPages: int32(totalPages),
		},
	}), nil
}

// convertVPSInstanceToProto converts a database.VPSInstance to vpsv1.VPSInstance
func convertVPSInstanceToProto(vps *database.VPSInstance) *vpsv1.VPSInstance {
	protoVPS := &vpsv1.VPSInstance{
		Id:             vps.ID,
		Name:           vps.Name,
		Description:    vps.Description,
		Status:         vpsv1.VPSStatus(vps.Status),
		Region:         vps.Region,
		Image:          vpsv1.VPSImage(vps.Image),
		ImageId:        vps.ImageID,
		Size:           vps.Size,
		CpuCores:       vps.CPUCores,
		MemoryBytes:    vps.MemoryBytes,
		DiskBytes:      vps.DiskBytes,
		InstanceId:     vps.InstanceID,
		NodeId:         vps.NodeID,
		SshKeyId:       vps.SSHKeyID,
		CreatedAt:      timestamppb.New(vps.CreatedAt),
		UpdatedAt:      timestamppb.New(vps.UpdatedAt),
		OrganizationId: vps.OrganizationID,
		CreatedBy:      vps.CreatedBy,
	}

	if vps.LastStartedAt != nil {
		protoVPS.LastStartedAt = timestamppb.New(*vps.LastStartedAt)
	}
	if vps.DeletedAt != nil {
		protoVPS.DeletedAt = timestamppb.New(*vps.DeletedAt)
	}

	// Unmarshal JSON fields
	if vps.IPv4Addresses != "" {
		var ipv4s []string
		if err := json.Unmarshal([]byte(vps.IPv4Addresses), &ipv4s); err == nil {
			protoVPS.Ipv4Addresses = ipv4s
		}
	}
	if vps.IPv6Addresses != "" {
		var ipv6s []string
		if err := json.Unmarshal([]byte(vps.IPv6Addresses), &ipv6s); err == nil {
			protoVPS.Ipv6Addresses = ipv6s
		}
	}
	if vps.Metadata != "" {
		var metadata map[string]string
		if err := json.Unmarshal([]byte(vps.Metadata), &metadata); err == nil {
			protoVPS.Metadata = metadata
		}
	}

	return protoVPS
}

// SuperadminGetVPS gets a VPS instance by ID (superadmin - bypasses organization checks)
func (s *Service) SuperadminGetVPS(ctx context.Context, req *connect.Request[superadminv1.SuperadminGetVPSRequest]) (*connect.Response[superadminv1.SuperadminGetVPSResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	vpsID := req.Msg.GetVpsId()
	if vpsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vps_id is required"))
	}

	var vps database.VPSInstance
	if err := database.DB.Where("id = ?", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	// Get organization name
	var org database.Organization
	orgName := "Unknown"
	if err := database.DB.Where("id = ?", vps.OrganizationID).First(&org).Error; err == nil {
		orgName = org.Name
	}

	// Get creator user information
	var createdByUser *superadminv1.UserInfo
	if vps.CreatedBy != "" {
		resolver := organizations.GetUserProfileResolver()
		userInfo := &superadminv1.UserInfo{
			Id: vps.CreatedBy,
		}

		// Try to resolve profile from Zitadel
		if resolver != nil && resolver.IsConfigured() {
			if profile, err := resolver.Resolve(ctx, vps.CreatedBy); err == nil && profile != nil {
				userInfo.Id = profile.Id
				userInfo.Email = profile.Email
				userInfo.Name = profile.Name
				userInfo.PreferredUsername = profile.PreferredUsername
				userInfo.Locale = profile.Locale
				userInfo.EmailVerified = profile.EmailVerified
				if profile.AvatarUrl != "" {
					userInfo.AvatarUrl = &profile.AvatarUrl
				}
				if profile.UpdatedAt != nil {
					userInfo.UpdatedAt = profile.UpdatedAt
				}
				if profile.CreatedAt != nil {
					userInfo.CreatedAt = profile.CreatedAt
				}
			}
		}

		createdByUser = userInfo
	}

	// Convert to proto
	protoVPS := convertVPSInstanceToProto(&vps)

	return connect.NewResponse(&superadminv1.SuperadminGetVPSResponse{
		Vps:              protoVPS,
		OrganizationName: orgName,
		CreatedBy:        createdByUser,
	}), nil
}

// SuperadminResizeVPS resizes a VPS instance to a new size (superadmin only)
func (s *Service) SuperadminResizeVPS(ctx context.Context, req *connect.Request[superadminv1.SuperadminResizeVPSRequest]) (*connect.Response[superadminv1.SuperadminResizeVPSResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	vpsID := req.Msg.GetVpsId()
	newSize := req.Msg.GetNewSize()
	growDisk := req.Msg.GetGrowDisk()
	applyCloudInit := req.Msg.GetApplyCloudinit()

	if vpsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vps_id is required"))
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

	// Determine new size specifications
	var newCPUCores int32
	var newMemoryBytes int64
	var newDiskBytes int64
	var finalSizeID string

	// Check if using custom size (access fields directly to check if they're set)
	customCPUCoresPtr := req.Msg.CustomCpuCores
	customMemoryBytesPtr := req.Msg.CustomMemoryBytes
	customDiskBytesPtr := req.Msg.CustomDiskBytes

	if newSize == "custom" || (newSize == "" && customCPUCoresPtr != nil) {
		// Validate custom size parameters
		if customCPUCoresPtr == nil || *customCPUCoresPtr <= 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("custom_cpu_cores is required and must be greater than 0"))
		}
		if customMemoryBytesPtr == nil || *customMemoryBytesPtr <= 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("custom_memory_bytes is required and must be greater than 0"))
		}
		if customDiskBytesPtr == nil || *customDiskBytesPtr <= 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("custom_disk_bytes is required and must be greater than 0"))
		}

		newCPUCores = *customCPUCoresPtr
		newMemoryBytes = *customMemoryBytesPtr
		newDiskBytes = *customDiskBytesPtr
		finalSizeID = "custom"
	} else {
		// Use predefined size from catalog
		if newSize == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("new_size is required"))
		}

		newSizeCatalog, err := database.GetVPSSizeCatalog(newSize, vps.Region)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid VPS size: %w", err))
		}

		newCPUCores = newSizeCatalog.CPUCores
		newMemoryBytes = newSizeCatalog.MemoryBytes
		newDiskBytes = newSizeCatalog.DiskBytes
		finalSizeID = newSize
	}

	// Check if resize is needed
	if vps.Size == finalSizeID && vps.CPUCores == newCPUCores && vps.MemoryBytes == newMemoryBytes && vps.DiskBytes == newDiskBytes {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("VPS is already at the requested size"))
	}

	// Get Proxmox configuration
	proxmoxConfig, err := vpsorch.GetProxmoxConfig()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get Proxmox config: %w", err))
	}

	// Create Proxmox client
	proxmoxClient, err := vpsorch.NewProxmoxClient(proxmoxConfig)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create Proxmox client: %w", err))
	}

	vmIDInt := 0
	fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
	if vmIDInt == 0 {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invalid VM ID: %s", *vps.InstanceID))
	}

	nodes, err := proxmoxClient.ListNodes(ctx)
	if err != nil || len(nodes) == 0 {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find Proxmox node: %w", err))
	}
	nodeName := nodes[0]

	// Stop VM if running (required for resize)
	status, err := proxmoxClient.GetVMStatus(ctx, nodeName, vmIDInt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VM status: %w", err))
	}

	wasRunning := status == "running"
	if wasRunning {
		logger.Info("[SuperAdmin] Stopping VM %d for resize", vmIDInt)
		if err := proxmoxClient.StopVM(ctx, nodeName, vmIDInt); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to stop VM: %w", err))
		}
		// Wait for VM to stop
		for i := 0; i < 30; i++ {
			time.Sleep(2 * time.Second)
			status, _ := proxmoxClient.GetVMStatus(ctx, nodeName, vmIDInt)
			if status == "stopped" {
				break
			}
		}
	}

	// Update CPU and memory
	vmConfig := make(map[string]interface{})
	if vps.CPUCores != newCPUCores {
		vmConfig["cores"] = newCPUCores
		logger.Info("[SuperAdmin] Resizing CPU from %d to %d cores", vps.CPUCores, newCPUCores)
	}
	if vps.MemoryBytes != newMemoryBytes {
		vmConfig["memory"] = int(newMemoryBytes / (1024 * 1024)) // Convert to MB
		logger.Info("[SuperAdmin] Resizing memory from %d to %d bytes", vps.MemoryBytes, newMemoryBytes)
	}

	if len(vmConfig) > 0 {
		// Update VM config via Proxmox API
		if err := proxmoxClient.UpdateVMConfig(ctx, nodeName, vmIDInt, vmConfig); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update VM config: %w", err))
		}
		logger.Info("[SuperAdmin] Successfully updated VM config: %v", vmConfig)
	}

	// Resize disk if needed and requested
	if growDisk && vps.DiskBytes != newDiskBytes {
		// Find disk key
		vmConfigAfter, err := proxmoxClient.GetVMConfig(ctx, nodeName, vmIDInt)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VM config: %w", err))
		}

		diskKeys := []string{"scsi0", "virtio0", "sata0", "ide0"}
		var diskKey string
		for _, key := range diskKeys {
			if disk, ok := vmConfigAfter[key].(string); ok && disk != "" {
				diskKey = key
				break
			}
		}

		if diskKey == "" {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("could not find disk to resize"))
		}

		newDiskSizeGB := newDiskBytes / (1024 * 1024 * 1024)
		logger.Info("[SuperAdmin] Resizing disk %s from %dGB to %dGB", diskKey, vps.DiskBytes/(1024*1024*1024), newDiskSizeGB)

		// Resize disk via Proxmox API
		if err := proxmoxClient.ResizeDisk(ctx, nodeName, vmIDInt, diskKey, newDiskSizeGB); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to resize disk: %w", err))
		}
		logger.Info("[SuperAdmin] Successfully resized disk %s to %dGB", diskKey, newDiskSizeGB)

		// If cloud-init should be applied, update it to grow the filesystem
		if applyCloudInit {
			// Load existing cloud-init config
			configService := vpsservice.NewConfigService(nil)
			cloudInitConfig, err := configService.LoadCloudInitConfig(ctx, &vps)
			if err != nil {
				logger.Warn("[SuperAdmin] Failed to load cloud-init config: %v, using default", err)
				cloudInitConfig = &vpsorch.CloudInitConfig{
					PackageUpdate:    boolPtr(true),
					PackageUpgrade:   boolPtr(false),
					SSHInstallServer: boolPtr(true),
					SSHAllowPW:       boolPtr(true),
					Runcmd:           []string{},
				}
			}

			// Add growpart and resize2fs commands to runcmd if not already present
			growPartCmd := "growpart /dev/sda 1 || growpart /dev/vda 1 || true"
			resizeFsCmd := "resize2fs /dev/sda1 || resize2fs /dev/vda1 || true"

			// Check if commands already exist
			hasGrowPart := false
			hasResizeFs := false
			for _, cmd := range cloudInitConfig.Runcmd {
				if strings.Contains(cmd, "growpart") {
					hasGrowPart = true
				}
				if strings.Contains(cmd, "resize2fs") {
					hasResizeFs = true
				}
			}

			if !hasGrowPart {
				cloudInitConfig.Runcmd = append([]string{growPartCmd}, cloudInitConfig.Runcmd...)
			}
			if !hasResizeFs {
				cloudInitConfig.Runcmd = append([]string{resizeFsCmd}, cloudInitConfig.Runcmd...)
			}

			// Save cloud-init config
			if err := configService.SaveCloudInitConfig(ctx, &vps, cloudInitConfig); err != nil {
				logger.Warn("[SuperAdmin] Failed to update cloud-init config: %v", err)
			} else {
				logger.Info("[SuperAdmin] Updated cloud-init config with disk growth commands")
			}
		}
	}

	// Update database
	vps.Size = finalSizeID
	vps.CPUCores = newCPUCores
	vps.MemoryBytes = newMemoryBytes
	if growDisk {
		vps.DiskBytes = newDiskBytes
	}
	vps.UpdatedAt = time.Now()

	if err := database.DB.Save(&vps).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update VPS in database: %w", err))
	}

	// Restart VM if it was running
	if wasRunning {
		logger.Info("[SuperAdmin] Starting VM %d after resize", vmIDInt)
		if err := proxmoxClient.StartVM(ctx, nodeName, vmIDInt); err != nil {
			logger.Warn("[SuperAdmin] Failed to start VM after resize: %v", err)
		}
	}

	message := fmt.Sprintf("VPS resized successfully. CPU: %d cores, Memory: %s, Disk: %s",
		newCPUCores,
		formatBytes(newMemoryBytes),
		formatBytes(newDiskBytes))
	if growDisk && applyCloudInit {
		message += " Disk will be grown on next boot via cloud-init."
	}

	return connect.NewResponse(&superadminv1.SuperadminResizeVPSResponse{
		Vps:     convertVPSInstanceToProto(&vps),
		Message: message,
	}), nil
}

// SuperadminSuspendVPS suspends a VPS instance (superadmin only)
func (s *Service) SuperadminSuspendVPS(ctx context.Context, req *connect.Request[superadminv1.SuperadminSuspendVPSRequest]) (*connect.Response[superadminv1.SuperadminSuspendVPSResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	vpsID := req.Msg.GetVpsId()
	if vpsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vps_id is required"))
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

	// Update status to SUSPENDED
	vps.Status = int32(vpsv1.VPSStatus_SUSPENDED)
	vps.UpdatedAt = time.Now()

	// Store suspension reason in metadata if provided
	if reason := req.Msg.GetReason(); reason != "" {
		var metadata map[string]string
		if vps.Metadata != "" {
			json.Unmarshal([]byte(vps.Metadata), &metadata)
		}
		if metadata == nil {
			metadata = make(map[string]string)
		}
		metadata["suspended_reason"] = reason
		metadata["suspended_at"] = time.Now().Format(time.RFC3339)
		metadataJSON, _ := json.Marshal(metadata)
		vps.Metadata = string(metadataJSON)
	}

	if err := database.DB.Save(&vps).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to suspend VPS: %w", err))
	}

	message := "VPS suspended successfully"
	if reason := req.Msg.GetReason(); reason != "" {
		message += fmt.Sprintf(" (reason: %s)", reason)
	}

	return connect.NewResponse(&superadminv1.SuperadminSuspendVPSResponse{
		Vps:     convertVPSInstanceToProto(&vps),
		Message: message,
	}), nil
}

// SuperadminUnsuspendVPS unsuspends a VPS instance (superadmin only)
func (s *Service) SuperadminUnsuspendVPS(ctx context.Context, req *connect.Request[superadminv1.SuperadminUnsuspendVPSRequest]) (*connect.Response[superadminv1.SuperadminUnsuspendVPSResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	vpsID := req.Msg.GetVpsId()
	if vpsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vps_id is required"))
	}

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	// Update status to STOPPED (was suspended)
	vps.Status = int32(vpsv1.VPSStatus_STOPPED)
	vps.UpdatedAt = time.Now()

	// Remove suspension metadata
	if vps.Metadata != "" {
		var metadata map[string]string
		if err := json.Unmarshal([]byte(vps.Metadata), &metadata); err == nil {
			delete(metadata, "suspended_reason")
			delete(metadata, "suspended_at")
			metadataJSON, _ := json.Marshal(metadata)
			vps.Metadata = string(metadataJSON)
		}
	}

	if err := database.DB.Save(&vps).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to unsuspend VPS: %w", err))
	}

	return connect.NewResponse(&superadminv1.SuperadminUnsuspendVPSResponse{
		Vps:     convertVPSInstanceToProto(&vps),
		Message: "VPS unsuspended successfully",
	}), nil
}

// SuperadminUpdateVPSCloudInit updates the cloud-init configuration for a VPS (superadmin only)
func (s *Service) SuperadminUpdateVPSCloudInit(ctx context.Context, req *connect.Request[superadminv1.SuperadminUpdateVPSCloudInitRequest]) (*connect.Response[superadminv1.SuperadminUpdateVPSCloudInitResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	vpsID := req.Msg.GetVpsId()
	if vpsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vps_id is required"))
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

	// Convert proto cloud-init to orchestrator format
	cloudInitProto := req.Msg.GetCloudInit()
	if cloudInitProto == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cloud_init is required"))
	}

	// Convert proto to orchestrator format
	cloudInitConfig := protoToCloudInitConfigForSuperadmin(cloudInitProto)

	// Save cloud-init config
	configService := vpsservice.NewConfigService(nil)
	if err := configService.SaveCloudInitConfig(ctx, &vps, cloudInitConfig); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update cloud-init config: %w", err))
	}

	logger.Info("[SuperAdmin] Successfully updated cloud-init config for VPS %s", vpsID)

	message := "Cloud-init configuration updated. Changes will take effect on the next reboot or when cloud-init is re-run."
	if req.Msg.GetGrowDiskIfNeeded() {
		message += " Disk growth commands have been added if needed."
	}

	return connect.NewResponse(&superadminv1.SuperadminUpdateVPSCloudInitResponse{
		Vps:     convertVPSInstanceToProto(&vps),
		Message: message,
	}), nil
}

// SuperadminForceStopVPS force stops a VPS instance (superadmin only)
func (s *Service) SuperadminForceStopVPS(ctx context.Context, req *connect.Request[superadminv1.SuperadminForceStopVPSRequest]) (*connect.Response[superadminv1.SuperadminForceStopVPSResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	vpsID := req.Msg.GetVpsId()
	if vpsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vps_id is required"))
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

	// Get VPS manager and force stop
	vpsManager, err := vpsorch.NewVPSManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create VPS manager: %w", err))
	}

	// Force stop (use force=true for forceful stop)
	if err := vpsManager.StopVPS(ctx, vpsID, true); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to force stop VPS: %w", err))
	}

	// Refresh VPS status
	if err := database.DB.Where("id = ?", vpsID).First(&vps).Error; err == nil {
		vps.Status = int32(vpsv1.VPSStatus_STOPPED)
		vps.UpdatedAt = time.Now()
		database.DB.Save(&vps)
	}

	return connect.NewResponse(&superadminv1.SuperadminForceStopVPSResponse{
		Vps:     convertVPSInstanceToProto(&vps),
		Message: "VPS force stopped successfully",
	}), nil
}

// SuperadminForceDeleteVPS force deletes a VPS instance (superadmin only)
func (s *Service) SuperadminForceDeleteVPS(ctx context.Context, req *connect.Request[superadminv1.SuperadminForceDeleteVPSRequest]) (*connect.Response[superadminv1.SuperadminForceDeleteVPSResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	vpsID := req.Msg.GetVpsId()
	if vpsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vps_id is required"))
	}

	// Get VPS instance
	var vps database.VPSInstance
	if err := database.DB.Where("id = ?", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	// Get VPS manager and delete
	vpsManager, err := vpsorch.NewVPSManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create VPS manager: %w", err))
	}
	defer vpsManager.Close()

	if err := vpsManager.DeleteVPS(ctx, vpsID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to force delete VPS: %w", err))
	}

	message := fmt.Sprintf("VPS %s force deleted successfully", vpsID)

	return connect.NewResponse(&superadminv1.SuperadminForceDeleteVPSResponse{
		Success: true,
		Message: message,
	}), nil
}
