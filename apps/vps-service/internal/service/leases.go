package vps

import (
	"context"
	"fmt"

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

	// Get the VPS to determine which node it's on
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", msg.VpsId).First(&vps).Error; err != nil {
		logger.Error("Failed to get VPS for lease registration",
			"vps_id", msg.VpsId,
			"error", err,
		)
		return connect.NewResponse(&vpsv1.RegisterLeaseResponse{
			Success: false,
			Message: "VPS not found",
		}), nil
	}

	// Get the node name from VPS (source of truth for which node this is on)
	gatewayNode := ""
	if vps.NodeID != nil {
		gatewayNode = *vps.NodeID
	}

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
