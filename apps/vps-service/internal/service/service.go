package vps

import (
	"context"
	"errors"
	"fmt"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	vpsorch "vps-service/orchestrator"
	"github.com/obiente/cloud/apps/shared/pkg/quota"
	"github.com/obiente/cloud/apps/shared/pkg/services/common"

	vpsv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/vps/v1/vpsv1connect"

	"connectrpc.com/connect"
	"gorm.io/gorm"
)

type Service struct {
	vpsv1connect.UnimplementedVPSServiceHandler
	permissionChecker *auth.PermissionChecker
	quotaChecker      *quota.Checker
	vpsManager        *vpsorch.VPSManager
}

func NewService(vpsManager *vpsorch.VPSManager, qc *quota.Checker) *Service {
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

// checkVPSPermission verifies user permissions for a VPS instance
func (s *Service) checkVPSPermission(ctx context.Context, vpsID string, permission string) error {
	// Get VPS by ID to check ownership
	var vps database.VPSInstance
	if err := database.DB.Where("id = ? AND deleted_at IS NULL", vpsID).First(&vps).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return connect.NewError(connect.CodeNotFound, fmt.Errorf("VPS instance %s not found", vpsID))
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VPS: %w", err))
	}

	// Get user from context
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}

	// First check if user is admin (always has access)
	if auth.HasRole(userInfo, auth.RoleAdmin) {
		return nil
	}

	// Check if user is the resource owner
	if vps.CreatedBy == userInfo.Id {
		return nil // Resource owners have full access to their resources
	}

	// For more complex permissions (organization-based, team-based, etc.)
	err = s.permissionChecker.CheckPermission(ctx, auth.ResourceTypeVPS, vpsID, permission)
	if err != nil {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("permission denied: %w", err))
	}

	return nil
}

// checkOrganizationPermission verifies user has access to an organization
func (s *Service) checkOrganizationPermission(ctx context.Context, organizationID string) error {
	userInfo, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required: %w", err))
	}

	// Admins have access to all organizations
	if auth.HasRole(userInfo, auth.RoleAdmin) {
		return nil
	}

	// Check if user is a member of the organization
	var count int64
	if err := database.DB.Model(&database.OrganizationMember{}).
		Where("organization_id = ? AND user_id = ? AND status = ?", organizationID, userInfo.Id, "active").
		Count(&count).Error; err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check organization membership: %w", err))
	}

	if count == 0 {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("user is not a member of organization %s", organizationID))
	}

	return nil
}
