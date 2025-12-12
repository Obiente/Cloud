package vps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"github.com/obiente/cloud/apps/shared/pkg/redis"
	orchestrator "github.com/obiente/cloud/apps/vps-service/orchestrator"

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

	// Get authenticated user from context
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}

	// Check if user has organization-wide read permission for VPS
	// This allows users with custom roles (like "system admin") to see all VPS instances
	hasOrgWideRead := false
	if err := s.permissionChecker.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{
		Permission:   auth.PermissionVPSRead,
		ResourceType: "vps",
		ResourceID:   "", // Empty resource ID means org-wide permission
	}); err == nil {
		hasOrgWideRead = true
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

	// Build query with permission-based filtering
	query := database.DB.Model(&database.VPSInstance{}).
		Where("organization_id = ? AND deleted_at IS NULL", orgID)

	// Filter by status if provided
	if req.Msg.Status != nil {
		status := req.Msg.GetStatus()
		query = query.Where("status = ?", int32(status))
	}

	// Filter by user if they don't have org-wide read permission
	if !auth.IsSuperadmin(ctx, userInfo) && !hasOrgWideRead {
		query = query.Where("created_by = ?", userInfo.Id)
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

	// Create Redis log writer for this VPS using shared helper
	logWriter := redis.NewLogStreamer(vpsID).WithAutoExpiry(24 * time.Hour).AsLogWriter()

	// Convert proto to VPSConfig
	config := &orchestrator.VPSConfig{
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
		cloudInit := &orchestrator.CloudInitConfig{
			Users:      make([]orchestrator.CloudInitUser, 0, len(cloudInitProto.Users)),
			Packages:   cloudInitProto.Packages,
			Runcmd:     cloudInitProto.Runcmd,
			WriteFiles: make([]orchestrator.CloudInitWriteFile, 0, len(cloudInitProto.WriteFiles)),
		}

		// Convert users
		for _, userProto := range cloudInitProto.Users {
			user := orchestrator.CloudInitUser{
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
	// Use independent context to avoid HTTP request timeout/cancellation
	// VPS creation can take 1-2 minutes, but HTTP requests typically timeout at 30-60 seconds
	createCtx, createCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer createCancel()
	vpsInstance, rootPassword, err := s.vpsManager.CreateVPS(createCtx, config, logWriter)
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

	// Log response details before sending
	responseVPS := &vpsv1.CreateVPSResponse{
		Vps: protoVPS,
	}
	if responseVPS.Vps.RootPassword != nil {
		logger.Info("[VPS Service] Response contains root password (length: %d) for VPS %s", len(*responseVPS.Vps.RootPassword), vpsInstance.ID)
	} else {
		logger.Warn("[VPS Service] Response does NOT contain root password for VPS %s", vpsInstance.ID)
	}

	// Return response immediately to avoid context cancellation issues
	// Send notification in background (non-blocking, uses independent context)
	response := connect.NewResponse(responseVPS)

	// Send notification asynchronously with independent context to avoid blocking response
	go func() {
		// Use background context with timeout for notifications
		notifyCtx, notifyCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer notifyCancel()
		s.notifyVPSCreated(notifyCtx, vpsInstance)
	}()

	logger.Info("[VPS Service] Returning CreateVPS response for VPS %s", vpsInstance.ID)
	return response, nil
}

// GetVPS retrieves a VPS instance by ID
func (s *Service) GetVPS(ctx context.Context, req *connect.Request[vpsv1.GetVPSRequest]) (*connect.Response[vpsv1.GetVPSResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	// Keep external calls bounded so the API response doesn't hang for minutes
	const lookupTimeout = 5 * time.Second

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, auth.PermissionVPSRead); err != nil {
		return nil, err
	}

	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	// Return current (cached) details immediately
	resp := connect.NewResponse(&vpsv1.GetVPSResponse{Vps: vpsToProto(&vps)})

	// Best-effort async refresh: status, disk, IPs with bounded timeouts
	if vps.InstanceID != nil {
		cachedVPS := vps // capture for goroutine
		go func() {
			refreshCtx, refreshCancel := context.WithTimeout(context.Background(), lookupTimeout)
			defer refreshCancel()

			// Sync status (best effort)
			s.syncVPSStatusFromProxmox(refreshCtx, vpsID)

			// Fetch disk/IPs
			vpsManager, err := orchestrator.NewVPSManager()
			if err != nil {
				logger.Warn("[VPS Service] Failed to create VPS manager for IP/disk refresh: %v", err)
				return
			}
			defer vpsManager.Close()

			nodeName := ""
			if cachedVPS.NodeID != nil && *cachedVPS.NodeID != "" {
				nodeName = *cachedVPS.NodeID
			}

			if nodeName != "" {
				proxmoxClient, err := vpsManager.GetProxmoxClientForNode(nodeName)
				if err == nil {
					vmIDInt := 0
					fmt.Sscanf(*cachedVPS.InstanceID, "%d", &vmIDInt)
					if vmIDInt > 0 {
						if diskSize, err := proxmoxClient.GetVMDiskSize(refreshCtx, nodeName, vmIDInt); err == nil && diskSize > 0 && cachedVPS.DiskBytes != diskSize {
							database.DB.Model(&cachedVPS).Update("disk_bytes", diskSize)
						}
					}
				}
			}

			// IP refresh
			ipv4, ipv6, err := vpsManager.GetVPSIPAddresses(refreshCtx, vpsID)
			if err == nil && (len(ipv4) > 0 || len(ipv6) > 0) {
				updates := map[string]interface{}{}
				if len(ipv4) > 0 {
					if ipv4JSON, mErr := json.Marshal(ipv4); mErr == nil {
						updates["ipv4_addresses"] = string(ipv4JSON)
					}
				}
				if len(ipv6) > 0 {
					if ipv6JSON, mErr := json.Marshal(ipv6); mErr == nil {
						updates["ipv6_addresses"] = string(ipv6JSON)
					}
				}
				if len(updates) > 0 {
					database.DB.Model(&cachedVPS).Updates(updates)
				}
			} else if err != nil {
				logger.Debug("[VPS Service] Failed to get IP addresses for VPS %s (guest agent/gateway unavailable): %v", vpsID, err)
			}
		}()
	}

	return resp, nil
}

// UpdateVPS updates a VPS instance
func (s *Service) UpdateVPS(ctx context.Context, req *connect.Request[vpsv1.UpdateVPSRequest]) (*connect.Response[vpsv1.UpdateVPSResponse], error) {
	ctx, err := s.ensureAuthenticated(ctx, req)
	if err != nil {
		return nil, err
	}

	vpsID := req.Msg.GetVpsId()
	if err := s.checkVPSPermission(ctx, vpsID, auth.PermissionVPSManage); err != nil {
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
	if err := s.checkVPSPermission(ctx, vpsID, auth.PermissionVPSManage); err != nil {
		return nil, err
	}

	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	// Check if VPS is already marked as DELETED (status 9) or DELETING (status 8)
	// If so, skip Proxmox operations entirely - the VM is already gone or being deleted
	deletedStatus := int32(vpsv1.VPSStatus_DELETED)
	deletingStatus := int32(vpsv1.VPSStatus_DELETING)
	if vps.Status == deletedStatus {
		logger.Info("[VPS Service] VPS %s is already marked as DELETED, skipping Proxmox operations and deleting database record only", vpsID)
	} else if vps.Status == deletingStatus {
		logger.Info("[VPS Service] VPS %s is already being deleted, skipping duplicate deletion", vpsID)
		return connect.NewResponse(&vpsv1.DeleteVPSResponse{
			Success: true,
		}), nil
	} else {
		// Set status to DELETING immediately for frontend feedback
		if err := database.DB.Model(&vps).Update("status", deletingStatus).Error; err != nil {
			logger.Warn("[VPS Service] Failed to set VPS %s status to DELETING: %v", vpsID, err)
			// Continue with deletion anyway
		} else {
			logger.Info("[VPS Service] Set VPS %s status to DELETING", vpsID)
		}
		// Always delete from Proxmox if InstanceID exists (VM was provisioned)
		// This ensures the VM is removed from Proxmox when deleted from the dashboard
		// For VPS in CREATING status, we still attempt deletion but don't fail if it errors
		// (VM might not be fully created yet or might be in a transitional state)
		if vps.InstanceID != nil {
			if err := s.vpsManager.DeleteVPS(ctx, vpsID); err != nil {
				// If VPS is in CREATING status, log warning but continue with database deletion
				// The VM might not be fully created yet or might be in a bad state
				// Also allow deletion to proceed if VM doesn't exist (already deleted or never created)
				errStr := err.Error()
				isVMNotFound := strings.Contains(errStr, "does not exist") ||
					strings.Contains(errStr, "not found") ||
					strings.Contains(errStr, "already deleted") ||
					strings.Contains(errStr, "context canceled") // Context canceled often means VM doesn't exist

				creatingStatus := int32(vpsv1.VPSStatus_CREATING)
				if vps.Status == creatingStatus || isVMNotFound {
					if vps.Status == creatingStatus {
						logger.Warn("[VPS Service] Failed to delete VPS %s from Proxmox (status: CREATING): %v. Continuing with database deletion.", vpsID, err)
					} else {
						logger.Info("[VPS Service] VM for VPS %s does not exist in Proxmox (may have been deleted already). Continuing with database deletion.", vpsID)
					}
				} else {
					// For other statuses and errors, return error to prevent orphaned VMs
					return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete VPS from Proxmox: %w", err))
				}
			}
		}
	}

	// Send notification before deletion
	s.notifyVPSDeleted(ctx, &vps)

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
