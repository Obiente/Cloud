package organizations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/services/common"
	adminv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/admin/v1"
	adminv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/admin/v1/adminv1connect"

	"connectrpc.com/connect"
	"gorm.io/gorm"
)

// AdminService implements AdminServiceHandler for role and permission management
type AdminService struct {
	adminv1connect.UnimplementedAdminServiceHandler
}

func NewAdminService() adminv1connect.AdminServiceHandler {
	return &AdminService{}
}

// ListPermissions returns the catalog of available permissions
func (s *AdminService) ListPermissions(ctx context.Context, req *connect.Request[adminv1.ListPermissionsRequest]) (*connect.Response[adminv1.ListPermissionsResponse], error) {
	// Build permission catalog from auth package definitions
	permissions := []*adminv1.PermissionDefinition{
		// Deployments
		{Id: auth.PermDeploymentsCreate, Description: auth.ScopeDescriptions[auth.PermDeploymentsCreate], ResourceType: "deployment"},
		{Id: auth.PermDeploymentsRead, Description: auth.ScopeDescriptions[auth.PermDeploymentsRead], ResourceType: "deployment"},
		{Id: auth.PermDeploymentsUpdate, Description: auth.ScopeDescriptions[auth.PermDeploymentsUpdate], ResourceType: "deployment"},
		{Id: auth.PermDeploymentsDelete, Description: auth.ScopeDescriptions[auth.PermDeploymentsDelete], ResourceType: "deployment"},
		{Id: auth.PermDeploymentsLogs, Description: auth.ScopeDescriptions[auth.PermDeploymentsLogs], ResourceType: "deployment"},
		{Id: auth.PermDeploymentsScale, Description: auth.ScopeDescriptions[auth.PermDeploymentsScale], ResourceType: "deployment"},
		// Environments
		{Id: auth.PermEnvironmentsManage, Description: auth.ScopeDescriptions[auth.PermEnvironmentsManage], ResourceType: "environment"},
		{Id: auth.PermEnvironmentsDeploy, Description: auth.ScopeDescriptions[auth.PermEnvironmentsDeploy], ResourceType: "environment"},
		// Admin
		{Id: auth.PermAdminRolesRead, Description: auth.ScopeDescriptions[auth.PermAdminRolesRead], ResourceType: "admin"},
		{Id: auth.PermAdminRolesWrite, Description: auth.ScopeDescriptions[auth.PermAdminRolesWrite], ResourceType: "admin"},
		{Id: auth.PermAdminBindingsRead, Description: auth.ScopeDescriptions[auth.PermAdminBindingsRead], ResourceType: "admin"},
		{Id: auth.PermAdminBindingsWrite, Description: auth.ScopeDescriptions[auth.PermAdminBindingsWrite], ResourceType: "admin"},
		{Id: auth.PermAdminQuotasUpdate, Description: auth.ScopeDescriptions[auth.PermAdminQuotasUpdate], ResourceType: "admin"},
	}

	return connect.NewResponse(&adminv1.ListPermissionsResponse{
		Permissions: permissions,
	}), nil
}

// ListRoles returns all roles for an organization
func (s *AdminService) ListRoles(ctx context.Context, req *connect.Request[adminv1.ListRolesRequest]) (*connect.Response[adminv1.ListRolesResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Check authorization - user must be owner/admin of the organization or superadmin
	isSuperAdmin := auth.HasRole(user, auth.RoleSuperAdmin)
	if !isSuperAdmin {
		if err := common.AuthorizeOrgRoles(ctx, orgID, user, "owner", "admin"); err != nil {
			return nil, err
		}
	}

	var roles []database.OrgRole
	if err := database.DB.Where("organization_id = ?", orgID).Find(&roles).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list roles: %w", err))
	}

	protoRoles := make([]*adminv1.Role, 0, len(roles))
	for _, r := range roles {
		protoRoles = append(protoRoles, &adminv1.Role{
			Id:             r.ID,
			Name:           r.Name,
			PermissionsJson: r.Permissions,
		})
	}

	return connect.NewResponse(&adminv1.ListRolesResponse{
		Roles: protoRoles,
	}), nil
}

// CreateRole creates a new role in an organization
func (s *AdminService) CreateRole(ctx context.Context, req *connect.Request[adminv1.CreateRoleRequest]) (*connect.Response[adminv1.CreateRoleResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("role name is required"))
	}

	permissionsJSON := strings.TrimSpace(req.Msg.GetPermissionsJson())
	if permissionsJSON == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("permissions_json is required"))
	}

	// Validate JSON format
	var perms []string
	if err := json.Unmarshal([]byte(permissionsJSON), &perms); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid permissions_json: %w", err))
	}

	// Check authorization - user must be owner/admin of the organization or superadmin
	isSuperAdmin := auth.HasRole(user, auth.RoleSuperAdmin)
	if !isSuperAdmin {
		if err := common.AuthorizeOrgRoles(ctx, orgID, user, "owner", "admin"); err != nil {
			return nil, err
		}
	}

	// Check if role name already exists in this organization
	var existing database.OrgRole
	if err := database.DB.Where("organization_id = ? AND name = ?", orgID, name).First(&existing).Error; err == nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("role with name %s already exists", name))
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check existing role: %w", err))
	}

	role := &database.OrgRole{
		ID:             generateID("role"),
		OrganizationID: orgID,
		Name:           name,
		Permissions:    permissionsJSON,
	}

	if err := database.DB.Create(role).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create role: %w", err))
	}

	return connect.NewResponse(&adminv1.CreateRoleResponse{
		Role: &adminv1.Role{
			Id:             role.ID,
			Name:           role.Name,
			PermissionsJson: role.Permissions,
		},
	}), nil
}

// UpdateRole updates an existing role
func (s *AdminService) UpdateRole(ctx context.Context, req *connect.Request[adminv1.UpdateRoleRequest]) (*connect.Response[adminv1.UpdateRoleResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	roleID := strings.TrimSpace(req.Msg.GetId())
	if roleID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("role id is required"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Check authorization - user must be owner/admin of the organization or superadmin
	isSuperAdmin := auth.HasRole(user, auth.RoleSuperAdmin)
	if !isSuperAdmin {
		if err := common.AuthorizeOrgRoles(ctx, orgID, user, "owner", "admin"); err != nil {
			return nil, err
		}
	}

	var role database.OrgRole
	if err := database.DB.Where("id = ? AND organization_id = ?", roleID, orgID).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("role not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get role: %w", err))
	}

	// Update name if provided
	if name := strings.TrimSpace(req.Msg.GetName()); name != "" {
		// Check if new name conflicts with existing role
		var existing database.OrgRole
		if err := database.DB.Where("organization_id = ? AND name = ? AND id != ?", orgID, name, roleID).First(&existing).Error; err == nil {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("role with name %s already exists", name))
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check existing role: %w", err))
		}
		role.Name = name
	}

	// Update permissions if provided
	if permissionsJSON := strings.TrimSpace(req.Msg.GetPermissionsJson()); permissionsJSON != "" {
		// Validate JSON format
		var perms []string
		if err := json.Unmarshal([]byte(permissionsJSON), &perms); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid permissions_json: %w", err))
		}
		role.Permissions = permissionsJSON
	}

	if err := database.DB.Save(&role).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update role: %w", err))
	}

	return connect.NewResponse(&adminv1.UpdateRoleResponse{
		Role: &adminv1.Role{
			Id:             role.ID,
			Name:           role.Name,
			PermissionsJson: role.Permissions,
		},
	}), nil
}

// DeleteRole deletes a role
func (s *AdminService) DeleteRole(ctx context.Context, req *connect.Request[adminv1.DeleteRoleRequest]) (*connect.Response[adminv1.DeleteRoleResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	roleID := strings.TrimSpace(req.Msg.GetId())
	if roleID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("role id is required"))
	}

	var role database.OrgRole
	if err := database.DB.First(&role, "id = ?", roleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("role not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get role: %w", err))
	}

	// Check authorization - user must be owner/admin of the organization or superadmin
	isSuperAdmin := auth.HasRole(user, auth.RoleSuperAdmin)
	if !isSuperAdmin {
		if err := common.AuthorizeOrgRoles(ctx, role.OrganizationID, user, "owner", "admin"); err != nil {
			return nil, err
		}
	}

	// Check if role is in use by any bindings
	var bindingCount int64
	if err := database.DB.Model(&database.OrgRoleBinding{}).Where("role_id = ?", roleID).Count(&bindingCount).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check role bindings: %w", err))
	}
	if bindingCount > 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("cannot delete role: %d role binding(s) still reference it", bindingCount))
	}

	if err := database.DB.Delete(&role).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete role: %w", err))
	}

	return connect.NewResponse(&adminv1.DeleteRoleResponse{Success: true}), nil
}

// ListRoleBindings returns all role bindings for an organization
func (s *AdminService) ListRoleBindings(ctx context.Context, req *connect.Request[adminv1.ListRoleBindingsRequest]) (*connect.Response[adminv1.ListRoleBindingsResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Check authorization - user must be owner/admin of the organization or superadmin
	isSuperAdmin := auth.HasRole(user, auth.RoleSuperAdmin)
	if !isSuperAdmin {
		if err := common.AuthorizeOrgRoles(ctx, orgID, user, "owner", "admin"); err != nil {
			return nil, err
		}
	}

	var bindings []database.OrgRoleBinding
	if err := database.DB.Where("organization_id = ?", orgID).Find(&bindings).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list role bindings: %w", err))
	}

	protoBindings := make([]*adminv1.RoleBinding, 0, len(bindings))
	for _, b := range bindings {
		protoBindings = append(protoBindings, &adminv1.RoleBinding{
			Id:             b.ID,
			OrganizationId: b.OrganizationID,
			UserId:         b.UserID,
			RoleId:         b.RoleID,
			ResourceType:   b.ResourceType,
			ResourceId:     b.ResourceID,
			ResourceSelector: b.ResourceSelector,
		})
	}

	return connect.NewResponse(&adminv1.ListRoleBindingsResponse{
		Bindings: protoBindings,
	}), nil
}

// CreateRoleBinding creates a new role binding
func (s *AdminService) CreateRoleBinding(ctx context.Context, req *connect.Request[adminv1.CreateRoleBindingRequest]) (*connect.Response[adminv1.CreateRoleBindingResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	userID := strings.TrimSpace(req.Msg.GetUserId())
	if userID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id is required"))
	}

	roleID := strings.TrimSpace(req.Msg.GetRoleId())
	if roleID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("role_id is required"))
	}

	// Check authorization - user must be owner/admin of the organization or superadmin
	isSuperAdmin := auth.HasRole(user, auth.RoleSuperAdmin)
	if !isSuperAdmin {
		if err := common.AuthorizeOrgRoles(ctx, orgID, user, "owner", "admin"); err != nil {
			return nil, err
		}
	}

	// Verify role exists and belongs to the organization
	var role database.OrgRole
	if err := database.DB.Where("id = ? AND organization_id = ?", roleID, orgID).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("role not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get role: %w", err))
	}

	// Check if binding already exists
	var existing database.OrgRoleBinding
	query := database.DB.Where("organization_id = ? AND user_id = ? AND role_id = ?", orgID, userID, roleID)
	if resourceType := strings.TrimSpace(req.Msg.GetResourceType()); resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	} else {
		query = query.Where("resource_type = ? OR resource_type = ''", "")
	}
	if resourceID := strings.TrimSpace(req.Msg.GetResourceId()); resourceID != "" {
		query = query.Where("resource_id = ?", resourceID)
	} else {
		query = query.Where("resource_id = ? OR resource_id = ''", "")
	}
	if err := query.First(&existing).Error; err == nil {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("role binding already exists"))
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check existing binding: %w", err))
	}

	binding := &database.OrgRoleBinding{
		ID:             generateID("rb"),
		OrganizationID: orgID,
		UserID:         userID,
		RoleID:         roleID,
		ResourceType:   strings.TrimSpace(req.Msg.GetResourceType()),
		ResourceID:     strings.TrimSpace(req.Msg.GetResourceId()),
		ResourceSelector: strings.TrimSpace(req.Msg.GetResourceSelector()),
	}

	if err := database.DB.Create(binding).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create role binding: %w", err))
	}

	return connect.NewResponse(&adminv1.CreateRoleBindingResponse{
		Binding: &adminv1.RoleBinding{
			Id:             binding.ID,
			OrganizationId: binding.OrganizationID,
			UserId:         binding.UserID,
			RoleId:         binding.RoleID,
			ResourceType:   binding.ResourceType,
			ResourceId:     binding.ResourceID,
			ResourceSelector: binding.ResourceSelector,
		},
	}), nil
}

// DeleteRoleBinding deletes a role binding
func (s *AdminService) DeleteRoleBinding(ctx context.Context, req *connect.Request[adminv1.DeleteRoleBindingRequest]) (*connect.Response[adminv1.DeleteRoleBindingResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	bindingID := strings.TrimSpace(req.Msg.GetId())
	if bindingID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("binding id is required"))
	}

	var binding database.OrgRoleBinding
	if err := database.DB.First(&binding, "id = ?", bindingID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("role binding not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get role binding: %w", err))
	}

	// Check authorization - user must be owner/admin of the organization or superadmin
	isSuperAdmin := auth.HasRole(user, auth.RoleSuperAdmin)
	if !isSuperAdmin {
		if err := common.AuthorizeOrgRoles(ctx, binding.OrganizationID, user, "owner", "admin"); err != nil {
			return nil, err
		}
	}

	if err := database.DB.Delete(&binding).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete role binding: %w", err))
	}

	return connect.NewResponse(&adminv1.DeleteRoleBindingResponse{Success: true}), nil
}

// UpsertOrgQuota updates organization quota overrides
func (s *AdminService) UpsertOrgQuota(ctx context.Context, req *connect.Request[adminv1.UpsertOrgQuotaRequest]) (*connect.Response[adminv1.UpsertOrgQuotaResponse], error) {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	orgID := strings.TrimSpace(req.Msg.GetOrganizationId())
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id is required"))
	}

	// Only superadmins can update quotas
	if !auth.HasRole(user, auth.RoleSuperAdmin) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("superadmin access required"))
	}

	var quota database.OrgQuota
	if err := database.DB.First(&quota, "organization_id = ?", orgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new quota - need to get plan ID first
			var org database.Organization
			if err := database.DB.First(&org, "id = ?", orgID).Error; err != nil {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("organization not found"))
			}
			// Get plan ID from org quota or use default
			var existingQuota database.OrgQuota
			if err := database.DB.First(&existingQuota, "organization_id = ?", orgID).Error; err != nil {
				// Need to ensure plan is assigned
				if err := EnsurePlanAssigned(orgID); err != nil {
					return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("ensure plan assigned: %w", err))
				}
				if err := database.DB.First(&existingQuota, "organization_id = ?", orgID).Error; err != nil {
					return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get quota after plan assignment: %w", err))
				}
			}
			quota = database.OrgQuota{
				OrganizationID: orgID,
				PlanID:         existingQuota.PlanID,
			}
		} else {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get quota: %w", err))
		}
	}

	// Update overrides (0 means unlimited, so we check if the value is set)
	if req.Msg.DeploymentsMaxOverride != 0 {
		val := int(req.Msg.DeploymentsMaxOverride)
		quota.DeploymentsMaxOverride = &val
	}
	if req.Msg.CpuCoresOverride != 0 {
		val := int(req.Msg.CpuCoresOverride)
		quota.CPUCoresOverride = &val
	}
	if req.Msg.MemoryBytesOverride != 0 {
		val := req.Msg.MemoryBytesOverride
		quota.MemoryBytesOverride = &val
	}
	if req.Msg.BandwidthBytesMonthOverride != 0 {
		val := req.Msg.BandwidthBytesMonthOverride
		quota.BandwidthBytesMonthOverride = &val
	}
	if req.Msg.StorageBytesOverride != 0 {
		val := req.Msg.StorageBytesOverride
		quota.StorageBytesOverride = &val
	}

	if err := database.DB.Save(&quota).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("upsert quota: %w", err))
	}

	return connect.NewResponse(&adminv1.UpsertOrgQuotaResponse{Success: true}), nil
}

