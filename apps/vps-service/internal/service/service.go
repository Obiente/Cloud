package vps

import (
	"context"

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
