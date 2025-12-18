package vps

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"
)

// GetVPSLeases retrieves DHCP lease information from the database
func (s *Service) GetVPSLeases(ctx context.Context, req *connect.Request[vpsv1.GetVPSLeasesRequest]) (*connect.Response[vpsv1.GetVPSLeasesResponse], error) {
	organizationID := req.Msg.GetOrganizationId()
	vpsID := req.Msg.VpsId // This is *string

	// Check organization permission
	if err := s.checkOrganizationPermission(ctx, organizationID); err != nil {
		return nil, err
	}

	// If specific VPS ID is provided, check VPS permission
	if vpsID != nil && *vpsID != "" {
		if err := s.checkVPSPermission(ctx, *vpsID, "vps:read"); err != nil {
			return nil, err
		}
	}

	logger.Info("Fetching VPS leases",
		"organization_id", organizationID,
		"vps_id", vpsID,
	)

	// Get leases from the VPS manager which queries the database
	leases, err := s.vpsManager.GetVPSLeases(ctx, organizationID, vpsID)
	if err != nil {
		logger.Error("Failed to fetch VPS leases",
			"organization_id", organizationID,
			"vps_id", vpsID,
			"error", err,
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch leases: %w", err))
	}

	return connect.NewResponse(&vpsv1.GetVPSLeasesResponse{
		Leases: leases,
	}), nil
}

// RegisterLease handles lease registration from gateway nodes (called via persistent connection)
func (s *Service) RegisterLease(ctx context.Context, req *connect.Request[vpsv1.RegisterLeaseRequest]) (*connect.Response[vpsv1.RegisterLeaseResponse], error) {
	msg := req.Msg

	// Get gateway node from the request (gateway tells us which node it is)
	gatewayNode := msg.GatewayNode
	
	logger.Info("Registering DHCP lease",
		"vps_id", msg.VpsId,
		"organization_id", msg.OrganizationId,
		"mac_address", msg.MacAddress,
		"ip_address", msg.IpAddress,
		"gateway_node", gatewayNode,
	)

	// Delegate to VPS manager with the node information
	if err := s.vpsManager.RegisterLease(ctx, msg, gatewayNode); err != nil {
		logger.Error("Failed to register lease",
			"vps_id", msg.VpsId,
			"error", err,
		)
		return connect.NewResponse(&vpsv1.RegisterLeaseResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to register lease: %v", err),
		}), nil
	}

	return connect.NewResponse(&vpsv1.RegisterLeaseResponse{
		Success: true,
		Message: "Lease registered successfully",
	}), nil
}

// FindVPSByLease looks up a DHCP lease by IP or MAC and returns the associated VPS
// if found.
//
// NOTE: This Connect RPC endpoint is NOT USED by gateways in production.
// Gateways use the bidirectional stream handler in internal/gateway/handlers.go instead.
//
// This endpoint exists for:
//   - Direct RPC calls from other services (if needed)
//   - API/debugging purposes
//   - Legacy compatibility
//
// For gateway requests, see: internal/gateway/handlers.go → FindVPSByLeaseHandler
//
// Lookup order for MAC address:
// 1. DHCP leases table (fast, authoritative for registered leases)
// 2. VPS instances table (database fallback for self-healing when lease not registered yet)
// 3. Proxmox API (queries all nodes to find VM with matching MAC - CRITICAL for initial lease creation)
//
// The Proxmox API lookup is essential to break the circular dependency:
// - Gateway needs VPS ID to populate hosts file
// - Gateway needs VPS ID to upsert lease to dhcp_leases table
// - But dhcp_leases is empty until VPS ID is known!
// Solution: Query Proxmox directly to find which VM owns this MAC address
func (s *Service) FindVPSByLease(ctx context.Context, req *connect.Request[vpsv1.FindVPSByLeaseRequest]) (*connect.Response[vpsv1.FindVPSByLeaseResponse], error) {
	mac := strings.ToLower(strings.TrimSpace(req.Msg.GetMac()))
	ip := strings.TrimSpace(req.Msg.GetIp())

	logger.Info("[FindVPSByLease] Request received: MAC=%s IP=%s", mac, ip)

	// Try MAC address first - most reliable
	if mac != "" {
		// 1. Check DHCP leases table (fast path)
		logger.Debug("[FindVPSByLease] Step 1: Checking dhcp_leases table for MAC=%s", mac)
		var lease database.DHCPLease
		if err := database.DB.WithContext(ctx).Where("mac_address = ?", mac).First(&lease).Error; err == nil {
			logger.Info("[FindVPSByLease] ✓ Found VPS %s by MAC in dhcp_leases", lease.VPSID)
			return connect.NewResponse(&vpsv1.FindVPSByLeaseResponse{VpsId: lease.VPSID, OrganizationId: lease.OrganizationID}), nil
		} else {
			logger.Debug("[FindVPSByLease] dhcp_leases lookup failed: %v", err)
		}
		
		// 2. Check VPS instances table (self-healing path - MAC synced from Proxmox)
		logger.Debug("[FindVPSByLease] Step 2: Checking vps_instances table for MAC=%s", mac)
		var vps database.VPSInstance
		if err := database.DB.WithContext(ctx).Where("mac_address = ? AND deleted_at IS NULL", mac).First(&vps).Error; err == nil {
			logger.Info("[FindVPSByLease] ✓ Found VPS %s by MAC in vps_instances (self-healing mode)", vps.ID)
			return connect.NewResponse(&vpsv1.FindVPSByLeaseResponse{VpsId: vps.ID, OrganizationId: vps.OrganizationID}), nil
		} else {
			logger.Debug("[FindVPSByLease] vps_instances lookup failed: %v", err)
		}

		// 3. Query Proxmox API to find the VM with this MAC (CRITICAL: breaks circular dependency)
		logger.Info("[FindVPSByLease] Step 3: Database lookups failed, attempting Proxmox API query for MAC=%s", mac)
		if s.vpsManager != nil {
			logger.Debug("[FindVPSByLease] Calling vpsManager.FindVPSByMAC()...")
			vpsFromProxmox, err := s.vpsManager.FindVPSByMAC(ctx, mac)
			if err != nil {
				logger.Error("[FindVPSByLease] ✗ Proxmox API lookup failed for MAC %s: %v", mac, err)
			} else if vpsFromProxmox != nil {
				logger.Info("[FindVPSByLease] ✓ Found VPS %s by MAC via Proxmox API (initial lease creation)", vpsFromProxmox.ID)
				return connect.NewResponse(&vpsv1.FindVPSByLeaseResponse{
					VpsId:          vpsFromProxmox.ID,
					OrganizationId: vpsFromProxmox.OrganizationID,
				}), nil
			} else {
				logger.Warn("[FindVPSByLease] ✗ Proxmox API returned nil (no VPS found with MAC %s)", mac)
			}
		} else {
			logger.Error("[FindVPSByLease] ✗ vpsManager is nil, cannot query Proxmox!")
		}
	}
	
	// Try IP address as fallback
	if ip != "" {
		var lease database.DHCPLease
		if err := database.DB.WithContext(ctx).Where("ip_address = ?", ip).First(&lease).Error; err == nil {
			logger.Debug("[FindVPSByLease] Found VPS %s by IP in dhcp_leases", lease.VPSID)
			return connect.NewResponse(&vpsv1.FindVPSByLeaseResponse{VpsId: lease.VPSID, OrganizationId: lease.OrganizationID}), nil
		}
	}

	// Not found: return empty response
	logger.Warn("[FindVPSByLease] ✗ No VPS found after all lookup attempts (MAC=%s IP=%s) - returning empty response", mac, ip)
	return connect.NewResponse(&vpsv1.FindVPSByLeaseResponse{}), nil
}

// ReleaseLease handles lease release from gateway nodes (called via persistent connection)
func (s *Service) ReleaseLease(ctx context.Context, req *connect.Request[vpsv1.ReleaseLeaseRequest]) (*connect.Response[vpsv1.ReleaseLeaseResponse], error) {
	msg := req.Msg

	// Get the VPS to determine which node it's on
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", msg.VpsId).First(&vps).Error; err != nil {
		logger.Error("Failed to get VPS for lease release",
			"vps_id", msg.VpsId,
			"error", err,
		)
		return connect.NewResponse(&vpsv1.ReleaseLeaseResponse{
			Success: false,
			Message: "VPS not found",
		}), nil
	}

	// Get the node name from VPS (source of truth for which node this is on)
	gatewayNode := ""
	if vps.NodeID != nil {
		gatewayNode = *vps.NodeID
	}

	logger.Info("Releasing DHCP lease",
		"vps_id", msg.VpsId,
		"mac_address", msg.MacAddress,
		"gateway_node", gatewayNode,
	)

	// Delegate to VPS manager with the node information
	if err := s.vpsManager.ReleaseLease(ctx, msg, gatewayNode); err != nil {
		logger.Error("Failed to release lease",
			"vps_id", msg.VpsId,
			"error", err,
		)
		return connect.NewResponse(&vpsv1.ReleaseLeaseResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to release lease: %v", err),
		}), nil
	}

	return connect.NewResponse(&vpsv1.ReleaseLeaseResponse{
		Success: true,
		Message: "Lease released successfully",
	}), nil
}
