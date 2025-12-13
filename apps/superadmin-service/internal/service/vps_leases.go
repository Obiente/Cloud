package superadmin

import (
	"context"
	"fmt"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"

	vpsgatewayv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vpsgateway/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GetOrgLeases retrieves DHCP leases for a specific organization
// This is a superadmin-only endpoint that reads from the database (populated by gateway sync)
func (s *Service) GetOrgLeases(ctx context.Context, req *connect.Request[vpsgatewayv1.GetOrgLeasesRequest]) (*connect.Response[vpsgatewayv1.GetOrgLeasesResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	// Check superadmin permission
	if !auth.IsSuperadmin(ctx, user) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	organizationID := req.Msg.GetOrganizationId()
	if organizationID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	logger.Info("Fetching organization leases",
		"organization_id", organizationID,
		"vps_id", req.Msg.VpsId,
	)

	if database.DB == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("database not initialised"))
	}

	query := database.DB.Where("organization_id = ?", organizationID)
	if vpsID := req.Msg.GetVpsId(); vpsID != "" {
		query = query.Where("vps_id = ?", vpsID)
	}

	var rows []database.DHCPLease
	if err := query.Order("expires_at DESC").Find(&rows).Error; err != nil {
		logger.Error("Failed to fetch leases from database",
			"organization_id", organizationID,
			"vps_id", req.Msg.VpsId,
			"error", err,
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch leases: %w", err))
	}

	leases := make([]*vpsgatewayv1.OrgLeaseRecord, 0, len(rows))
	for _, row := range rows {
		leases = append(leases, &vpsgatewayv1.OrgLeaseRecord{
			VpsId:          row.VPSID,
			OrganizationId: row.OrganizationID,
			MacAddress:     row.MACAddress,
			IpAddress:      row.IPAddress,
			ExpiresAt:      timestamppb.New(row.ExpiresAt),
			IsPublic:       row.IsPublic,
		})
	}

	return connect.NewResponse(&vpsgatewayv1.GetOrgLeasesResponse{Leases: leases}), nil
}
