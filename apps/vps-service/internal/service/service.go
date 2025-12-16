package vps

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/quota"
	"github.com/obiente/cloud/apps/shared/pkg/services/common"
	orchestrator "github.com/obiente/cloud/apps/vps-service/orchestrator"

	vpsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1"
	vpsv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1/vpsv1connect"

	"connectrpc.com/connect"
	"gorm.io/gorm"
)

type Service struct {
	vpsv1connect.UnimplementedVPSServiceHandler
	permissionChecker *auth.PermissionChecker
	quotaChecker      *quota.Checker
	vpsManager        *orchestrator.VPSManager
}

func NewService(vpsManager *orchestrator.VPSManager, qc *quota.Checker) *Service {
	return &Service{
		permissionChecker: auth.NewPermissionChecker(),
		quotaChecker:      qc,
		vpsManager:        vpsManager,
	}
}

// ensureAuthenticated ensures the user is authenticated for streaming RPCs
func (s *Service) ensureAuthenticated(ctx context.Context, req connect.AnyRequest) (context.Context, error) {
	return common.EnsureAuthenticated(ctx, req)
}

// checkVPSPermission verifies user permissions for a VPS
// Uses the reusable CheckResourcePermissionWithError helper
func (s *Service) checkVPSPermission(ctx context.Context, vpsID string, permission string) error {
	return auth.CheckResourcePermissionWithError(ctx, s.permissionChecker, "vps", vpsID, permission)
}

// checkOrganizationPermission verifies user has access to an organization
// Uses the reusable CheckScopedPermissionWithError helper
func (s *Service) checkOrganizationPermission(ctx context.Context, organizationID string) error {
	return auth.CheckScopedPermissionWithError(ctx, s.permissionChecker, organizationID, auth.ScopedPermission{
		Permission: auth.PermissionOrganizationRead,
	})
}

// --- DHCP Public IP Assignment ---
// AssignVPSPublicIP assigns a public IP to a VPS and triggers DHCP lease creation
func (s *Service) AssignVPSPublicIP(ctx context.Context, req *connect.Request[vpsv1.AssignVPSPublicIPRequest]) (*connect.Response[vpsv1.AssignVPSPublicIPResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	vpsID := req.Msg.GetVpsId()
	publicIP := req.Msg.GetPublicIp()

	// Permission check
	if err := s.checkVPSPermission(ctx, vpsID, "vps:update"); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	// Fetch VPS from DB
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND organization_id = ? AND deleted_at IS NULL", vpsID, orgID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("VPS not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if vps.NodeID == nil || *vps.NodeID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("VPS node not set"))
	}

	// Fetch MAC address from DHCPLease
	var lease database.DHCPLease
	if err := database.DB.Where("vps_id = ? AND organization_id = ? AND is_public = ?", vpsID, orgID, false).First(&lease).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("VPS MAC address not set (no DHCP lease found)"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	err := s.vpsManager.AssignVPSPublicIP(ctx, *vps.NodeID, vps.ID, vps.OrganizationID, lease.MACAddress, publicIP)
	if err != nil {
		return connect.NewResponse(&vpsv1.AssignVPSPublicIPResponse{
			Success: false,
			Message: err.Error(),
		}), nil
	}

	// Ensure a record exists in vps_public_ips so billing can pick up the assigned IP
	now := time.Now()
	var ip database.VPSPublicIP
	if err := database.DB.Where("ip_address = ?", publicIP).First(&ip).Error; err == nil {
		ip.VPSID = &vpsID
		ip.OrganizationID = &orgID
		ip.AssignedAt = &now
		ip.UpdatedAt = now
		if err := database.DB.Save(&ip).Error; err != nil {
			log.Printf("[VPS Service] Warning: failed to update vps_public_ips for %s: %v", publicIP, err)
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		newID := fmt.Sprintf("ip-%d", time.Now().UnixNano())
		// Determine default monthly cost (env-configurable; fallback $1.00 = 100 cents)
		defaultCostCents := int64(100)
		if v := os.Getenv("DEFAULT_PUBLIC_IP_MONTHLY_COST_CENTS"); v != "" {
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				defaultCostCents = parsed
			}
		}
		newIP := &database.VPSPublicIP{
			ID:               newID,
			IPAddress:        publicIP,
			VPSID:            &vpsID,
			OrganizationID:   &orgID,
			MonthlyCostCents: defaultCostCents,
			AssignedAt:       &now,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := database.DB.Create(newIP).Error; err != nil {
			log.Printf("[VPS Service] Warning: failed to create vps_public_ips record for %s: %v", publicIP, err)
		}
	} else {
		log.Printf("[VPS Service] Warning: failed to query vps_public_ips for %s: %v", publicIP, err)
	}

	return connect.NewResponse(&vpsv1.AssignVPSPublicIPResponse{
		Success: true,
		Message: "Assigned public IP via DHCP",
	}), nil
}
