package vps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	vpsorch "vps-service/orchestrator"

	commonv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1"
	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// ListVPS lists VPS instances for an organization
func (s *Service) ListVPS(ctx context.Context, req *connect.Request[vpsv1.ListVPSRequest]) (*connect.Response[vpsv1.ListVPSResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrganizationId()
	if err := s.checkOrganizationPermission(ctx, orgID); err != nil {
		return nil, err
	}

	page := int(req.Msg.GetPage())
	if page < 1 {
		page = 1
	}
	perPage := int(req.Msg.GetPerPage())
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	offset := (page - 1) * perPage

	query := database.DB.Model(&database.VPSInstance{}).
		Where("organization_id = ? AND deleted_at IS NULL", orgID)

	// Filter by status if provided
	if req.Msg.Status != nil {
		status := req.Msg.GetStatus()
		query = query.Where("status = ?", int32(status))
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to count VPS instances: %w", err))
	}

	// Get instances
	var instances []database.VPSInstance
	if err := query.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&instances).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list VPS instances: %w", err))
	}

	// Convert to proto
	vpsList := make([]*vpsv1.VPSInstance, len(instances))
	for i, vps := range instances {
		vpsList[i] = vpsToProto(&vps)
	}

	totalPages := int((total + int64(perPage) - 1) / int64(perPage))

	return connect.NewResponse(&vpsv1.ListVPSResponse{
		VpsInstances: vpsList,
		Pagination: &commonv1.Pagination{
			Page:       int32(page),
			PerPage:    int32(perPage),
			Total:      int32(total),
			TotalPages: int32(totalPages),
		},
	}), nil
}

// CreateVPS creates a new VPS instance
func (s *Service) CreateVPS(ctx context.Context, req *connect.Request[vpsv1.CreateVPSRequest]) (*connect.Response[vpsv1.CreateVPSResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	orgID := req.Msg.GetOrganizationId()
	if err := s.checkOrganizationPermission(ctx, orgID); err != nil {
		return nil, err
	}

	// Check quota
	if err := s.quotaChecker.CanAllocateVPS(ctx, orgID); err != nil {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("quota exceeded: %w", err))
	}

	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}

	// Generate VPS ID
	vpsID := fmt.Sprintf("vps-%d", time.Now().UnixNano())

	// Convert proto to VPSConfig
	config := &vpsorch.VPSConfig{
		VPSID:          vpsID,
		Name:           req.Msg.GetName(),
		Description:    req.Msg.Description,
		Region:         req.Msg.GetRegion(),
		Image:          int(req.Msg.GetImage()),
		ImageID:        req.Msg.ImageId,
		Size:           req.Msg.GetSize(),
		SSHKeyID:       req.Msg.SshKeyId,
		OrganizationID: orgID,
		CreatedBy:      userInfo.Id,
		Metadata:       req.Msg.GetMetadata(),
	}
	
	// Set creator name if available
	if userInfo.Name != "" {
		config.CreatorName = &userInfo.Name
	}
	
	// Handle root password (custom or auto-generated)
	if req.Msg.RootPassword != nil && req.Msg.GetRootPassword() != "" {
		rootPass := req.Msg.GetRootPassword()
		config.RootPassword = &rootPass
	}
	
	// Convert cloud-init configuration from proto
	if req.Msg.CloudInit != nil {
		cloudInitProto := req.Msg.GetCloudInit()
		cloudInit := &vpsorch.CloudInitConfig{
			Users:            make([]vpsorch.CloudInitUser, 0, len(cloudInitProto.Users)),
			Packages:         cloudInitProto.Packages,
			Runcmd:           cloudInitProto.Runcmd,
			WriteFiles:       make([]vpsorch.CloudInitWriteFile, 0, len(cloudInitProto.WriteFiles)),
		}
		
		// Convert users
		for _, userProto := range cloudInitProto.Users {
			user := vpsorch.CloudInitUser{
				Name:              userProto.GetName(),
				SSHAuthorizedKeys: userProto.SshAuthorizedKeys,
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
			user.Groups = userProto.Groups
			
			cloudInit.Users = append(cloudInit.Users, user)
		}
		
		// Convert system configuration
		if cloudInitProto.Hostname != nil {
			hostname := cloudInitProto.GetHostname()
			cloudInit.Hostname = &hostname
		}
		if cloudInitProto.Timezone != nil {
			timezone := cloudInitProto.GetTimezone()
			cloudInit.Timezone = &timezone
		}
		if cloudInitProto.Locale != nil {
			locale := cloudInitProto.GetLocale()
			cloudInit.Locale = &locale
		}
		
		// Convert package management
		if cloudInitProto.PackageUpdate != nil {
			packageUpdate := cloudInitProto.GetPackageUpdate()
			cloudInit.PackageUpdate = &packageUpdate
		}
		if cloudInitProto.PackageUpgrade != nil {
			packageUpgrade := cloudInitProto.GetPackageUpgrade()
			cloudInit.PackageUpgrade = &packageUpgrade
		}
		
		// Convert SSH configuration
		if cloudInitProto.SshInstallServer != nil {
			sshInstallServer := cloudInitProto.GetSshInstallServer()
			cloudInit.SSHInstallServer = &sshInstallServer
		}
		if cloudInitProto.SshAllowPw != nil {
			sshAllowPW := cloudInitProto.GetSshAllowPw()
			cloudInit.SSHAllowPW = &sshAllowPW
		}
		
		// Convert write files
		for _, fileProto := range cloudInitProto.WriteFiles {
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
			
			cloudInit.WriteFiles = append(cloudInit.WriteFiles, file)
		}
		
		config.CloudInit = cloudInit
	}

	// Get size from catalog
	sizeCatalog, err := database.GetVPSSizeCatalog(req.Msg.GetSize(), req.Msg.GetRegion())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid VPS size: %w", err))
	}

	// Check minimum payment requirement
	if sizeCatalog.MinimumPaymentCents > 0 {
		var org database.Organization
		if err := database.DB.First(&org, "id = ?", orgID).Error; err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get organization: %w", err))
		}

		if org.TotalPaidCents < sizeCatalog.MinimumPaymentCents {
			return nil, connect.NewError(
				connect.CodePermissionDenied,
				fmt.Errorf(
					"insufficient payment history: this VPS size requires a minimum payment of $%.2f, but your organization has only paid $%.2f. Please make additional payments to unlock this VPS size",
					float64(sizeCatalog.MinimumPaymentCents)/100.0,
					float64(org.TotalPaidCents)/100.0,
				),
			)
		}
	}

	config.CPUCores = sizeCatalog.CPUCores
	config.MemoryBytes = sizeCatalog.MemoryBytes
	config.DiskBytes = sizeCatalog.DiskBytes

	// Create VPS via manager
	vpsInstance, rootPassword, err := s.vpsManager.CreateVPS(ctx, config)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create VPS: %w", err))
	}

	// Convert to proto and include password (one-time only)
	protoVPS := vpsToProto(vpsInstance)
	if rootPassword != "" {
		protoVPS.RootPassword = &rootPassword
		logger.Info("[VPS Service] Including root password in CreateVPS response for VPS %s (length: %d)", vpsInstance.ID, len(rootPassword))
	} else {
		logger.Warn("[VPS Service] WARNING: rootPassword is empty for VPS %s - password will not be returned", vpsInstance.ID)
	}

	return connect.NewResponse(&vpsv1.CreateVPSResponse{
		Vps: protoVPS,
	}), nil
}

// GetVPS retrieves a VPS instance by ID
func (s *Service) GetVPS(ctx context.Context, req *connect.Request[vpsv1.GetVPSRequest]) (*connect.Response[vpsv1.GetVPSResponse], error) {
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

	// Sync status from Proxmox to ensure we show current status
	// This prevents showing stale status like REBOOTING when VPS has actually finished rebooting
	if vps.InstanceID != nil {
		s.syncVPSStatusFromProxmox(ctx, vpsID)
		// Refresh VPS after sync to get updated status
		if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
			// If refresh fails, continue with original vps - sync is best-effort
			logger.Warn("[VPS Service] Failed to refresh VPS after status sync: %v", err)
		}
	}

	// If VPS has an instance ID, try to fetch latest disk size and IP addresses from Proxmox
	if vps.InstanceID != nil {
		proxmoxConfig, err := vpsorch.GetProxmoxConfig()
		if err == nil {
			proxmoxClient, err := vpsorch.NewProxmoxClient(proxmoxConfig)
			if err == nil {
				vmIDInt := 0
				fmt.Sscanf(*vps.InstanceID, "%d", &vmIDInt)
				if vmIDInt > 0 {
					nodes, err := proxmoxClient.ListNodes(ctx)
					if err == nil && len(nodes) > 0 {
						// Try to get disk size from Proxmox
						if diskSize, err := proxmoxClient.GetVMDiskSize(ctx, nodes[0], vmIDInt); err == nil && diskSize > 0 {
							// Update database if disk size is different
							if vps.DiskBytes != diskSize {
								vps.DiskBytes = diskSize
								database.DB.Model(&vps).Update("disk_bytes", diskSize)
							}
						}

						// Try to get IP addresses from Proxmox (requires guest agent)
						// Only update IP addresses if guest agent is available and returns valid IPs
						ipv4, ipv6, err := proxmoxClient.GetVMIPAddresses(ctx, nodes[0], vmIDInt)
						if err == nil && (len(ipv4) > 0 || len(ipv6) > 0) {
							// Update database with IP addresses from guest agent
							if len(ipv4) > 0 {
								ipv4JSON, _ := json.Marshal(ipv4)
								vps.IPv4Addresses = string(ipv4JSON)
							}
							if len(ipv6) > 0 {
								ipv6JSON, _ := json.Marshal(ipv6)
								vps.IPv6Addresses = string(ipv6JSON)
							}
							if len(ipv4) > 0 || len(ipv6) > 0 {
								database.DB.Model(&vps).Updates(map[string]interface{}{
									"ipv4_addresses": vps.IPv4Addresses,
									"ipv6_addresses": vps.IPv6Addresses,
								})
							}
						} else if err != nil {
							// Guest agent not available - clear IP addresses to avoid showing stale/incorrect IPs
							logger.Info("[VPS Service] Guest agent not available for VPS %s, clearing IP addresses", vpsID)
							vps.IPv4Addresses = "[]"
							vps.IPv6Addresses = "[]"
							database.DB.Model(&vps).Updates(map[string]interface{}{
								"ipv4_addresses": vps.IPv4Addresses,
								"ipv6_addresses": vps.IPv6Addresses,
							})
						}
					}
				}
			}
		}
	}

	return connect.NewResponse(&vpsv1.GetVPSResponse{
		Vps: vpsToProto(&vps),
	}), nil
}

// UpdateVPS updates a VPS instance
func (s *Service) UpdateVPS(ctx context.Context, req *connect.Request[vpsv1.UpdateVPSRequest]) (*connect.Response[vpsv1.UpdateVPSResponse], error) {
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

	// Update fields
	if req.Msg.Name != nil {
		vps.Name = req.Msg.GetName()
	}
	if req.Msg.Description != nil {
		vps.Description = req.Msg.Description
	}
	if req.Msg.Metadata != nil {
		metadataJSON, err := json.Marshal(req.Msg.GetMetadata())
		if err == nil {
			vps.Metadata = string(metadataJSON)
		}
	}

	vps.UpdatedAt = time.Now()

	if err := database.DB.Save(&vps).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update VPS: %w", err))
	}

	return connect.NewResponse(&vpsv1.UpdateVPSResponse{
		Vps: vpsToProto(&vps),
	}), nil
}

// DeleteVPS deletes a VPS instance (soft delete)
func (s *Service) DeleteVPS(ctx context.Context, req *connect.Request[vpsv1.DeleteVPSRequest]) (*connect.Response[vpsv1.DeleteVPSResponse], error) {
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

	// Always delete from Proxmox if InstanceID exists (VM was provisioned)
	// This ensures the VM is removed from Proxmox when deleted from the dashboard
	if vps.InstanceID != nil {
		if err := s.vpsManager.DeleteVPS(ctx, vpsID); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete VPS from Proxmox: %w", err))
		}
	}

	// Delete from database (hard delete)
	if err := database.DB.Delete(&vps).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete VPS: %w", err))
	}

	return connect.NewResponse(&vpsv1.DeleteVPSResponse{
		Success: true,
	}), nil
}

// vpsToProto converts a database VPSInstance to proto
func vpsToProto(vps *database.VPSInstance) *vpsv1.VPSInstance {
	proto := &vpsv1.VPSInstance{
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
		proto.LastStartedAt = timestamppb.New(*vps.LastStartedAt)
	}
	if vps.DeletedAt != nil {
		proto.DeletedAt = timestamppb.New(*vps.DeletedAt)
	}

	// Parse JSON arrays for IP addresses
	if vps.IPv4Addresses != "" {
		var ipv4 []string
		if err := json.Unmarshal([]byte(vps.IPv4Addresses), &ipv4); err == nil {
			proto.Ipv4Addresses = ipv4
		}
	}
	if vps.IPv6Addresses != "" {
		var ipv6 []string
		if err := json.Unmarshal([]byte(vps.IPv6Addresses), &ipv6); err == nil {
			proto.Ipv6Addresses = ipv6
		}
	}

	// Parse JSON object for metadata
	if vps.Metadata != "" {
		var metadata map[string]string
		if err := json.Unmarshal([]byte(vps.Metadata), &metadata); err == nil {
			proto.Metadata = metadata
		}
	}

	// NOTE: Root password is NEVER returned in GetVPS or ListVPS responses
	// Password is only shown once during creation, then discarded for security

	return proto
}
