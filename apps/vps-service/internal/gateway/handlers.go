package gateway

// ARCHITECTURE: VPS→Gateway Bidirectional Stream Request Handlers
//
// This file contains request handlers for the VPS service's OUTBOUND bidirectional
// stream connections to gateway nodes. This is the PRIMARY and ACTIVE communication path.
//
// Connection Flow:
//   1. VPS service connects TO each gateway node (VPS_NODE_GATEWAY_ENDPOINTS env var)
//   2. Bidirectional stream established (client.go)
//   3. Gateway sends requests over the stream (e.g., FindVPSByLease, AllocateIP)
//   4. Handlers in this file process those requests and send responses
//
// DO NOT confuse with:
//   - internal/gateway/service.go: RegisterGateway() - DEPRECATED inbound stream (Gateway→VPS)
//   - internal/service/leases.go: FindVPSByLease() - Connect RPC endpoint (not used by gateways)
//   - orchestrator/vps_gateway_client.go: VPSGatewayClient - DEPRECATED unary RPC client
//
// Active handlers in this file:
//   - FindVPSByLeaseHandler: Resolves MAC/IP to VPS ID using dhcp_leases → vps_instances → Proxmox API

import (
	"context"
	"fmt"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"

	"strings"

	"google.golang.org/protobuf/proto"
)

// FindVPSByLeaseHandler handles FindVPSByLease requests from the gateway
type FindVPSByLeaseHandler struct{
	vpsManager VPSManagerInterface
}

// VPSManagerInterface defines the methods needed from VPSManager
type VPSManagerInterface interface {
	FindVPSByMAC(ctx context.Context, macAddress string) (*database.VPSInstance, error)
}

// NewFindVPSByLeaseHandler creates a new FindVPSByLease handler
func NewFindVPSByLeaseHandler() *FindVPSByLeaseHandler {
	return &FindVPSByLeaseHandler{}
}

// SetVPSManager sets the VPS manager for Proxmox lookups
func (h *FindVPSByLeaseHandler) SetVPSManager(vm VPSManagerInterface) {
	h.vpsManager = vm
}

// HandleRequest implements RequestHandler interface
// This is the ACTUAL handler called when gateway sends FindVPSByLease requests over the bidirectional stream
func (h *FindVPSByLeaseHandler) HandleRequest(ctx context.Context, method string, payload []byte) ([]byte, error) {
	// Unmarshal request
	var req vpsv1.FindVPSByLeaseRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal FindVPSByLease request: %w", err)
	}

	// Query for VPS by lease (database + Proxmox fallback)
	mac := strings.ToLower(strings.TrimSpace(req.GetMac()))
	ip := strings.TrimSpace(req.GetIp())

	logger.Info("[FindVPSByLeaseHandler] ========== REQUEST: MAC=%s IP=%s ==========", mac, ip)

	var lease database.DHCPLease
	var found bool
	var vpsID, orgID string

	// Try MAC address first - check dhcp_leases table
	if mac != "" {
		logger.Debug("[FindVPSByLeaseHandler] Step 1: Checking dhcp_leases for MAC=%s", mac)
		if err := database.DB.WithContext(ctx).Where("mac_address = ?", mac).First(&lease).Error; err == nil {
			found = true
			vpsID = lease.VPSID
			orgID = lease.OrganizationID
			logger.Info("[FindVPSByLeaseHandler] ✓ Found VPS %s by MAC in dhcp_leases", vpsID)
		} else {
			logger.Debug("[FindVPSByLeaseHandler] dhcp_leases lookup failed: %v", err)
			
			// Not in dhcp_leases - try vps_instances table
			logger.Debug("[FindVPSByLeaseHandler] Step 2: Checking vps_instances for MAC=%s", mac)
			var vps database.VPSInstance
			if err := database.DB.WithContext(ctx).Where("mac_address = ? AND deleted_at IS NULL", mac).First(&vps).Error; err == nil {
				found = true
				vpsID = vps.ID
				orgID = vps.OrganizationID
				logger.Info("[FindVPSByLeaseHandler] ✓ Found VPS %s by MAC in vps_instances", vpsID)
			} else {
				logger.Debug("[FindVPSByLeaseHandler] vps_instances lookup failed: %v", err)
				
				// Database lookups failed - query Proxmox API
				if h.vpsManager != nil {
					logger.Info("[FindVPSByLeaseHandler] Step 3: Database lookups failed, querying Proxmox API for MAC=%s...", mac)
					vpsFromProxmox, err := h.vpsManager.FindVPSByMAC(ctx, mac)
					if err != nil {
						logger.Error("[FindVPSByLeaseHandler] ✗ Proxmox API lookup failed for MAC %s: %v", mac, err)
					} else if vpsFromProxmox != nil {
						found = true
						vpsID = vpsFromProxmox.ID
						orgID = vpsFromProxmox.OrganizationID
						logger.Info("[FindVPSByLeaseHandler] ✓ Found VPS %s by MAC via Proxmox API", vpsID)
					} else {
						logger.Warn("[FindVPSByLeaseHandler] ✗ Proxmox API returned nil for MAC %s", mac)
					}
				} else {
					logger.Error("[FindVPSByLeaseHandler] ✗ vpsManager is nil, cannot query Proxmox!")
				}
			}
		}
	}

	// Try IP address as fallback
	if !found && ip != "" {
		logger.Debug("[FindVPSByLeaseHandler] Step 4: Trying IP fallback for IP=%s", ip)
		if err := database.DB.WithContext(ctx).Where("ip_address = ?", ip).First(&lease).Error; err == nil {
			found = true
			vpsID = lease.VPSID
			orgID = lease.OrganizationID
			logger.Info("[FindVPSByLeaseHandler] ✓ Found VPS %s by IP in dhcp_leases", vpsID)
		}
	}

	// Create response
	resp := &vpsv1.FindVPSByLeaseResponse{}
	if found {
		resp.VpsId = vpsID
		resp.OrganizationId = orgID
		logger.Info("[FindVPSByLeaseHandler] ========== RESPONSE: VPS=%s Org=%s ==========", vpsID, orgID)
	} else {
		logger.Warn("[FindVPSByLeaseHandler] ========== RESPONSE: NOT FOUND (MAC=%s IP=%s) ==========", mac, ip)
	}

	// Marshal response
	respPayload, err := proto.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal FindVPSByLease response: %w", err)
	}

	return respPayload, nil
}
