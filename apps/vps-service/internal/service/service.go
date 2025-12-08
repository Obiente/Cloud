package vps

import (
	"context"
	"strings"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/quota"
	"github.com/obiente/cloud/apps/shared/pkg/services/common"
	orchestrator "github.com/obiente/cloud/apps/vps-service/orchestrator"

	vpsv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1/vpsv1connect"

	"connectrpc.com/connect"
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
// Uses the unified CheckResourcePermission which handles all permission logic
func (s *Service) checkVPSPermission(ctx context.Context, vpsID string, permission string) error {
	if err := s.permissionChecker.CheckResourcePermission(ctx, "vps", vpsID, permission); err != nil {
		// Convert to Connect error with appropriate code
		if strings.Contains(err.Error(), "not found") {
			return connect.NewError(connect.CodeNotFound, err)
		}
		if strings.Contains(err.Error(), "unauthenticated") {
			return connect.NewError(connect.CodeUnauthenticated, err)
		}
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return nil
}

// checkOrganizationPermission verifies user has access to an organization
// Uses CheckScopedPermission with organization.read permission
func (s *Service) checkOrganizationPermission(ctx context.Context, organizationID string) error {
	if err := s.permissionChecker.CheckScopedPermission(ctx, organizationID, auth.ScopedPermission{
		Permission: "organization.read",
	}); err != nil {
		if strings.Contains(err.Error(), "unauthenticated") {
			return connect.NewError(connect.CodeUnauthenticated, err)
		}
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return nil
}
