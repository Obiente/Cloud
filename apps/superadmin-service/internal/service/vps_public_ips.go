package superadmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	vpsorch "github.com/obiente/cloud/apps/vps-service/orchestrator"

	superadminv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/superadmin/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// ListVPSPublicIPs lists all public IPs with optional filters (superadmin only)
func (s *Service) ListVPSPublicIPs(ctx context.Context, req *connect.Request[superadminv1.ListVPSPublicIPsRequest]) (*connect.Response[superadminv1.ListVPSPublicIPsResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.vps_public_ips.read") {
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
	query := database.DB.Table("vps_public_ips ip").
		Select(`
			ip.*,
			vps.name as vps_name,
			o.name as organization_name
		`).
		Joins("LEFT JOIN vps_instances vps ON vps.id = ip.vps_id").
		Joins("LEFT JOIN organizations o ON o.id = ip.organization_id")

	// Apply filters
	if vpsID := req.Msg.GetVpsId(); vpsID != "" {
		query = query.Where("ip.vps_id = ?", vpsID)
	}
	if orgID := req.Msg.GetOrganizationId(); orgID != "" {
		query = query.Where("ip.organization_id = ?", orgID)
	}
	if !req.Msg.GetIncludeUnassigned() {
		query = query.Where("ip.vps_id IS NOT NULL")
	}

	// Get total count
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to count public IPs: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to count public IPs: %w", err))
	}

	// Apply pagination and ordering
	var ipRows []struct {
		database.VPSPublicIP
		VPSName         *string `gorm:"column:vps_name"`
		OrganizationName *string `gorm:"column:organization_name"`
	}
	if err := query.Order("ip.created_at DESC").Offset(offset).Limit(perPage).Scan(&ipRows).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to list public IPs: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list public IPs: %w", err))
	}

	// Convert to proto
	ips := make([]*superadminv1.VPSPublicIP, 0, len(ipRows))
	for _, row := range ipRows {
		ip := &superadminv1.VPSPublicIP{
			Id:              row.ID,
			IpAddress:       row.IPAddress,
			MonthlyCostCents: row.MonthlyCostCents,
			CreatedAt:       timestamppb.New(row.CreatedAt),
			UpdatedAt:       timestamppb.New(row.UpdatedAt),
		}
		if row.VPSID != nil {
			ip.VpsId = row.VPSID
		}
		if row.OrganizationID != nil {
			ip.OrganizationId = row.OrganizationID
		}
		if row.VPSName != nil {
			ip.VpsName = row.VPSName
		}
		if row.OrganizationName != nil {
			ip.OrganizationName = row.OrganizationName
		}
		if row.Gateway != nil {
			ip.Gateway = row.Gateway
		}
		if row.Netmask != nil {
			ip.Netmask = row.Netmask
		}
		if row.AssignedAt != nil {
			ip.AssignedAt = timestamppb.New(*row.AssignedAt)
		}
		ips = append(ips, ip)
	}

	return connect.NewResponse(&superadminv1.ListVPSPublicIPsResponse{
		Ips:        ips,
		TotalCount: totalCount,
	}), nil
}

// CreateVPSPublicIP creates a new public IP (superadmin only)
func (s *Service) CreateVPSPublicIP(ctx context.Context, req *connect.Request[superadminv1.CreateVPSPublicIPRequest]) (*connect.Response[superadminv1.CreateVPSPublicIPResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.vps_public_ips.create") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	ipAddress := strings.TrimSpace(req.Msg.GetIpAddress())
	if ipAddress == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("ip_address is required"))
	}

	// Validate IP address format
	parsedIP := net.ParseIP(ipAddress)
	if parsedIP == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid IP address format: %s", ipAddress))
	}

	monthlyCostCents := req.Msg.GetMonthlyCostCents()
	if monthlyCostCents < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("monthly_cost_cents must be non-negative"))
	}

	// Check if IP already exists
	var existing database.VPSPublicIP
	if err := database.DB.Where("ip_address = ?", ipAddress).First(&existing).Error; err == nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("IP address %s already exists", ipAddress))
	}

	// Create IP record
	ipID := fmt.Sprintf("ip-%d", time.Now().UnixNano())
	ip := &database.VPSPublicIP{
		ID:               ipID,
		IPAddress:        ipAddress,
		MonthlyCostCents: monthlyCostCents,
		VPSID:            nil,
		OrganizationID:   nil,
		AssignedAt:       nil,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Set gateway if provided
	if req.Msg.Gateway != nil && req.Msg.GetGateway() != "" {
		gateway := req.Msg.GetGateway()
		ip.Gateway = &gateway
	}

	// Set netmask if provided
	if req.Msg.Netmask != nil && req.Msg.GetNetmask() != "" {
		netmask := req.Msg.GetNetmask()
		ip.Netmask = &netmask
	}

	if err := database.DB.Create(ip).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to create public IP: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create public IP: %w", err))
	}

	responseIP := &superadminv1.VPSPublicIP{
		Id:               ip.ID,
		IpAddress:        ip.IPAddress,
		MonthlyCostCents: ip.MonthlyCostCents,
		CreatedAt:        timestamppb.New(ip.CreatedAt),
		UpdatedAt:        timestamppb.New(ip.UpdatedAt),
	}
	if ip.Gateway != nil {
		responseIP.Gateway = ip.Gateway
	}
	if ip.Netmask != nil {
		responseIP.Netmask = ip.Netmask
	}

	return connect.NewResponse(&superadminv1.CreateVPSPublicIPResponse{
		Ip: responseIP,
	}), nil
}

// UpdateVPSPublicIP updates a public IP's monthly cost (superadmin only)
func (s *Service) UpdateVPSPublicIP(ctx context.Context, req *connect.Request[superadminv1.UpdateVPSPublicIPRequest]) (*connect.Response[superadminv1.UpdateVPSPublicIPResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.vps_public_ips.update") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	ipID := req.Msg.GetId()
	if ipID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}

	// Get existing IP
	var ip database.VPSPublicIP
	if err := database.DB.Where("id = ?", ipID).First(&ip).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("public IP %s not found", ipID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get public IP: %w", err))
	}

	// Update monthly cost if provided
	if req.Msg.MonthlyCostCents != nil {
		monthlyCostCents := req.Msg.GetMonthlyCostCents()
		if monthlyCostCents < 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("monthly_cost_cents must be non-negative"))
		}
		ip.MonthlyCostCents = monthlyCostCents
	}

	// Update gateway if provided
	if req.Msg.Gateway != nil {
		gateway := req.Msg.GetGateway()
		if gateway == "" {
			ip.Gateway = nil // Clear gateway if empty string
		} else {
			ip.Gateway = &gateway
		}
	}

	// Update netmask if provided
	if req.Msg.Netmask != nil {
		netmask := req.Msg.GetNetmask()
		if netmask == "" {
			ip.Netmask = nil // Clear netmask if empty string
		} else {
			ip.Netmask = &netmask
		}
	}

	ip.UpdatedAt = time.Now()
	if err := database.DB.Save(&ip).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to update public IP: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update public IP: %w", err))
	}

	// Get VPS and organization names for response
	var vpsName, orgName *string
	if ip.VPSID != nil {
		var vps database.VPSInstance
		if err := database.DB.Where("id = ?", ip.VPSID).First(&vps).Error; err == nil {
			vpsName = &vps.Name
		}
	}
	if ip.OrganizationID != nil {
		var org database.Organization
		if err := database.DB.Where("id = ?", ip.OrganizationID).First(&org).Error; err == nil {
			orgName = &org.Name
		}
	}

	responseIP := &superadminv1.VPSPublicIP{
		Id:               ip.ID,
		IpAddress:        ip.IPAddress,
		MonthlyCostCents: ip.MonthlyCostCents,
		CreatedAt:        timestamppb.New(ip.CreatedAt),
		UpdatedAt:        timestamppb.New(ip.UpdatedAt),
	}
	if ip.VPSID != nil {
		responseIP.VpsId = ip.VPSID
	}
	if ip.OrganizationID != nil {
		responseIP.OrganizationId = ip.OrganizationID
	}
	if vpsName != nil {
		responseIP.VpsName = vpsName
	}
	if orgName != nil {
		responseIP.OrganizationName = orgName
	}
	if ip.Gateway != nil {
		responseIP.Gateway = ip.Gateway
	}
	if ip.Netmask != nil {
		responseIP.Netmask = ip.Netmask
	}
	if ip.AssignedAt != nil {
		responseIP.AssignedAt = timestamppb.New(*ip.AssignedAt)
	}

	return connect.NewResponse(&superadminv1.UpdateVPSPublicIPResponse{
		Ip: responseIP,
	}), nil
}

// DeleteVPSPublicIP deletes a public IP (superadmin only)
func (s *Service) DeleteVPSPublicIP(ctx context.Context, req *connect.Request[superadminv1.DeleteVPSPublicIPRequest]) (*connect.Response[superadminv1.DeleteVPSPublicIPResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.vps_public_ips.delete") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	ipID := req.Msg.GetId()
	if ipID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id is required"))
	}

	// Check if IP exists and is assigned
	var ip database.VPSPublicIP
	if err := database.DB.Where("id = ?", ipID).First(&ip).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("public IP %s not found", ipID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get public IP: %w", err))
	}

	if ip.VPSID != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("cannot delete public IP: it is currently assigned to VPS %s. Unassign it first", *ip.VPSID))
	}

	// Delete IP
	if err := database.DB.Delete(&ip).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to delete public IP: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete public IP: %w", err))
	}

	return connect.NewResponse(&superadminv1.DeleteVPSPublicIPResponse{
		Success: true,
	}), nil
}

// AssignVPSPublicIP assigns a public IP to a VPS (superadmin only)
func (s *Service) AssignVPSPublicIP(ctx context.Context, req *connect.Request[superadminv1.AssignVPSPublicIPRequest]) (*connect.Response[superadminv1.AssignVPSPublicIPResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.vps_public_ips.update") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	ipID := req.Msg.GetIpId()
	vpsID := req.Msg.GetVpsId()
	if ipID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("ip_id is required"))
	}
	if vpsID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vps_id is required"))
	}

	// Get IP
	var ip database.VPSPublicIP
	if err := database.DB.Where("id = ?", ipID).First(&ip).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("public IP %s not found", ipID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get public IP: %w", err))
	}

	// Check if IP is already assigned
	if ip.VPSID != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("public IP %s is already assigned to VPS %s", ipID, *ip.VPSID))
	}

	// Get VPS
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS %s not found", vpsID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	// Assign IP to VPS
	now := time.Now()
	ip.VPSID = &vpsID
	ip.OrganizationID = &vps.OrganizationID
	ip.AssignedAt = &now
	ip.UpdatedAt = now

	if err := database.DB.Save(&ip).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to assign public IP: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to assign public IP: %w", err))
	}

	// Update VPS IPv4Addresses to include the assigned IP
	var ipv4Addresses []string
	if vps.IPv4Addresses != "" && vps.IPv4Addresses != "[]" {
		if err := json.Unmarshal([]byte(vps.IPv4Addresses), &ipv4Addresses); err != nil {
			logger.Warn("[SuperAdmin] Failed to unmarshal existing IPv4 addresses: %v", err)
		}
	}
	// Add IP if not already present
	found := false
	for _, addr := range ipv4Addresses {
		if addr == ip.IPAddress {
			found = true
			break
		}
	}
	if !found {
		ipv4Addresses = append(ipv4Addresses, ip.IPAddress)
		ipv4JSON, _ := json.Marshal(ipv4Addresses)
		vps.IPv4Addresses = string(ipv4JSON)
		if err := database.DB.Model(&vps).Update("ipv4_addresses", vps.IPv4Addresses).Error; err != nil {
			logger.Warn("[SuperAdmin] Failed to update VPS IPv4 addresses: %v", err)
		}
	}

	// Update cloud-init to add the static IP to the VPS
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
						// Get gateway and netmask from IP record (set by superadmin)
						publicGateway := ""
						if ip.Gateway != nil && *ip.Gateway != "" {
							publicGateway = *ip.Gateway
						} else {
							// Fallback: calculate default gateway from public IP (typically .1 in the subnet)
							parsedIP := net.ParseIP(ip.IPAddress)
							if parsedIP != nil {
								ip4 := parsedIP.To4()
								if ip4 != nil {
									ip4[3] = 1 // Set last octet to 1 for gateway
									publicGateway = ip4.String()
								}
							}
							// If we still don't have a gateway, return an error
							if publicGateway == "" {
								logger.Error("[SuperAdmin] Failed to calculate gateway for public IP %s: invalid IP format", ip.IPAddress)
								return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid IP address format: %s", ip.IPAddress))
							}
						}
						
						netmask := "24" // Default netmask /24
						if ip.Netmask != nil && *ip.Netmask != "" {
							netmask = *ip.Netmask
						}
						
						// Update cloud-init userData with public IP (alongside existing DHCP)
						if err := proxmoxClient.UpdateCloudInitUserDataWithStaticIP(ctx, nodes[0], vmIDInt, ip.IPAddress, publicGateway, netmask); err != nil {
							logger.Warn("[SuperAdmin] Failed to update cloud-init with public IP %s for VPS %s: %v. The IP has been assigned in the database but may not be configured on the VPS until it reboots or cloud-init runs.", ip.IPAddress, vpsID, err)
						} else {
							logger.Info("[SuperAdmin] Successfully updated cloud-init with public IP %s (gateway: %s, netmask: %s) for VPS %s", ip.IPAddress, publicGateway, netmask, vpsID)
							
							// Try to configure the IP immediately on the running VPS (if VM is running)
							if err := proxmoxClient.ConfigurePublicIPOnVM(ctx, nodes[0], vmIDInt, ip.IPAddress, publicGateway, netmask); err != nil {
								logger.Warn("[SuperAdmin] Failed to configure public IP %s immediately on VPS %s: %v. The IP will be configured on next boot via cloud-init.", ip.IPAddress, vpsID, err)
							} else {
								// Verify the IP is actually configured
								configured, err := proxmoxClient.VerifyPublicIPOnVM(ctx, nodes[0], vmIDInt, ip.IPAddress)
								if err != nil {
									logger.Warn("[SuperAdmin] Failed to verify public IP %s on VPS %s: %v", ip.IPAddress, vpsID, err)
								} else if !configured {
									logger.Warn("[SuperAdmin] Public IP %s was configured but verification failed. The IP may not be active yet.", ip.IPAddress)
								} else {
									logger.Info("[SuperAdmin] Public IP %s successfully configured and verified on VPS %s", ip.IPAddress, vpsID)
								}
							}
						}
					} else if err != nil {
						logger.Warn("[SuperAdmin] Failed to list Proxmox nodes for VPS %s: %v", vpsID, err)
					}
				}
			} else {
				logger.Warn("[SuperAdmin] Failed to create Proxmox client for VPS %s: %v", vpsID, err)
			}
		} else {
			logger.Warn("[SuperAdmin] Failed to get Proxmox config for VPS %s: %v", vpsID, err)
		}
	}

	// Get VPS and organization names for response
	vpsName := vps.Name
	var orgName *string
	var org database.Organization
	if err := database.DB.Where("id = ?", vps.OrganizationID).First(&org).Error; err == nil {
		orgName = &org.Name
	}

	responseIP := &superadminv1.VPSPublicIP{
		Id:               ip.ID,
		IpAddress:        ip.IPAddress,
		MonthlyCostCents: ip.MonthlyCostCents,
		VpsId:            &vpsID,
		OrganizationId:   &vps.OrganizationID,
		VpsName:          &vpsName,
		CreatedAt:        timestamppb.New(ip.CreatedAt),
		UpdatedAt:        timestamppb.New(ip.UpdatedAt),
		AssignedAt:       timestamppb.New(now),
	}
	if orgName != nil {
		responseIP.OrganizationName = orgName
	}
	if ip.Gateway != nil {
		responseIP.Gateway = ip.Gateway
	}
	if ip.Netmask != nil {
		responseIP.Netmask = ip.Netmask
	}

	return connect.NewResponse(&superadminv1.AssignVPSPublicIPResponse{
		Ip:      responseIP,
		Message: fmt.Sprintf("Public IP %s assigned to VPS %s", ip.IPAddress, vpsID),
	}), nil
}

// UnassignVPSPublicIP unassigns a public IP from a VPS (superadmin only)
func (s *Service) UnassignVPSPublicIP(ctx context.Context, req *connect.Request[superadminv1.UnassignVPSPublicIPRequest]) (*connect.Response[superadminv1.UnassignVPSPublicIPResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if !auth.HasSuperadminPermission(ctx, user, "superadmin.vps_public_ips.update") {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	ipID := req.Msg.GetIpId()
	if ipID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("ip_id is required"))
	}

	// Get IP
	var ip database.VPSPublicIP
	if err := database.DB.Where("id = ?", ipID).First(&ip).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("public IP %s not found", ipID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get public IP: %w", err))
	}

	// Check if IP is assigned
	if ip.VPSID == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("public IP %s is not assigned to any VPS", ipID))
	}

	vpsID := *ip.VPSID

	// Get VPS to remove IP from IPv4Addresses
	var vps database.VPSInstance
	if err := database.DB.Where("id = ?", vpsID).First(&vps).Error; err == nil {
		var ipv4Addresses []string
		if vps.IPv4Addresses != "" && vps.IPv4Addresses != "[]" {
			if err := json.Unmarshal([]byte(vps.IPv4Addresses), &ipv4Addresses); err == nil {
				// Remove IP from list
				newAddresses := make([]string, 0, len(ipv4Addresses))
				for _, addr := range ipv4Addresses {
					if addr != ip.IPAddress {
						newAddresses = append(newAddresses, addr)
					}
				}
				ipv4JSON, _ := json.Marshal(newAddresses)
				vps.IPv4Addresses = string(ipv4JSON)
				if err := database.DB.Model(&vps).Update("ipv4_addresses", vps.IPv4Addresses).Error; err != nil {
					logger.Warn("[SuperAdmin] Failed to update VPS IPv4 addresses: %v", err)
				}
			}
		}
	}

	// Unassign IP
	ip.VPSID = nil
	ip.OrganizationID = nil
	ip.AssignedAt = nil
	ip.UpdatedAt = time.Now()

	if err := database.DB.Save(&ip).Error; err != nil {
		logger.Error("[SuperAdmin] Failed to unassign public IP: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to unassign public IP: %w", err))
	}

	responseIP := &superadminv1.VPSPublicIP{
		Id:               ip.ID,
		IpAddress:        ip.IPAddress,
		MonthlyCostCents: ip.MonthlyCostCents,
		CreatedAt:        timestamppb.New(ip.CreatedAt),
		UpdatedAt:        timestamppb.New(ip.UpdatedAt),
	}

	return connect.NewResponse(&superadminv1.UnassignVPSPublicIPResponse{
		Ip:      responseIP,
		Message: fmt.Sprintf("Public IP %s unassigned from VPS %s", ip.IPAddress, vpsID),
	}), nil
}

