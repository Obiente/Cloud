package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	adminv1 "api/gen/proto/obiente/cloud/admin/v1"
	adminv1connect "api/gen/proto/obiente/cloud/admin/v1/adminv1connect"
	"api/internal/auth"
	"api/internal/database"

	"connectrpc.com/connect"
)

type Service struct {
	adminv1connect.UnimplementedAdminServiceHandler
}

func NewService() adminv1connect.AdminServiceHandler { return &Service{} }

var reservedRoleNames = map[string]struct{}{
	auth.RoleOwner:      {},
	"org.owner":         {},
	auth.RoleOrgManager: {},
	auth.RoleOrgAdmin:   {},
	auth.RoleOrgMember:  {},
}

func isReservedRoleName(name string) bool {
	if _, ok := reservedRoleNames[name]; ok {
		return true
	}
	if len(name) >= 4 && name[:4] == "org." {
		return true
	}
	return false
}

func validatePermissionsJSON(permsJSON string) error {
	if permsJSON == "" {
		return nil
	}
	var perms []string
	if err := json.Unmarshal([]byte(permsJSON), &perms); err != nil {
		return fmt.Errorf("permissions_json must be JSON array of strings")
	}
	for _, p := range perms {
		if _, ok := auth.ScopeDescriptions[p]; !ok {
			return fmt.Errorf("unknown permission: %s", p)
		}
	}
	return nil
}

// isOrgOwner checks organization_members table for an 'owner' member
func isOrgOwner(ctx context.Context, orgID string) bool {
	user, err := auth.GetUserFromContext(ctx)
	if err != nil || user == nil {
		return false
	}
	var cnt int64
	_ = database.DB.Model(&database.OrganizationMember{}).
		Where("organization_id = ? AND user_id = ? AND role = ?", orgID, user.Id, "owner").
		Count(&cnt).Error
	return cnt > 0
}

func (s *Service) UpsertOrgQuota(ctx context.Context, req *connect.Request[adminv1.UpsertOrgQuotaRequest]) (*connect.Response[adminv1.UpsertOrgQuotaResponse], error) {
	// Require org manager or admin
	orgID := req.Msg.GetOrganizationId()
	if orgID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("organization_id required"))
	}
	user, _ := auth.GetUserFromContext(ctx)
	if !(auth.HasRole(user, auth.RoleAdmin) || auth.HasOrgRole(ctx, orgID, auth.RoleOrgManager) || auth.HasOrgRole(ctx, orgID, auth.RoleOrgAdmin)) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("forbidden"))
	}

	q := &database.OrgQuota{
		OrganizationID:              orgID,
		DeploymentsMaxOverride:      intPtrOrNil(req.Msg.GetDeploymentsMaxOverride()),
		CPUCoresOverride:            intPtrOrNil(req.Msg.GetCpuCoresOverride()),
		MemoryBytesOverride:         int64PtrOrNil(req.Msg.GetMemoryBytesOverride()),
		BandwidthBytesMonthOverride: int64PtrOrNil(req.Msg.GetBandwidthBytesMonthOverride()),
		StorageBytesOverride:        int64PtrOrNil(req.Msg.GetStorageBytesOverride()),
	}
	if err := database.DB.Save(q).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("save quota: %w", err))
	}
	return connect.NewResponse(&adminv1.UpsertOrgQuotaResponse{Success: true}), nil
}

func intPtrOrNil(v int32) *int {
	if v == 0 {
		return nil
	}
	x := int(v)
	return &x
}
func int64PtrOrNil(v int64) *int64 {
	if v == 0 {
		return nil
	}
	x := v
	return &x
}

// ListPermissions returns the server-defined permission catalog
func (s *Service) ListPermissions(ctx context.Context, _ *connect.Request[adminv1.ListPermissionsRequest]) (*connect.Response[adminv1.ListPermissionsResponse], error) {
	defs := make([]*adminv1.PermissionDefinition, 0, len(auth.ScopeDescriptions))
	for id, desc := range auth.ScopeDescriptions {
		rt := "admin"
		if len(id) >= 12 && id[:12] == "deployments." {
			rt = "deployment"
		}
		if len(id) >= 13 && id[:13] == "environments." {
			rt = "environment"
		}
		defs = append(defs, &adminv1.PermissionDefinition{Id: id, Description: desc, ResourceType: rt})
	}
	return connect.NewResponse(&adminv1.ListPermissionsResponse{Permissions: defs}), nil
}

// Roles CRUD
func (s *Service) ListRoles(ctx context.Context, req *connect.Request[adminv1.ListRolesRequest]) (*connect.Response[adminv1.ListRolesResponse], error) {
	var rows []database.OrgRole
	if err := database.DB.Where("organization_id = ?", req.Msg.GetOrganizationId()).Find(&rows).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list roles: %w", err))
	}
	out := make([]*adminv1.Role, 0, len(rows))
	for _, r := range rows {
		out = append(out, &adminv1.Role{Id: r.ID, Name: r.Name, PermissionsJson: r.Permissions})
	}
	return connect.NewResponse(&adminv1.ListRolesResponse{Roles: out}), nil
}

func (s *Service) CreateRole(ctx context.Context, req *connect.Request[adminv1.CreateRoleRequest]) (*connect.Response[adminv1.CreateRoleResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	user, _ := auth.GetUserFromContext(ctx)
	if !(auth.HasRole(user, auth.RoleAdmin) || isOrgOwner(ctx, orgID) || auth.HasOrgRole(ctx, orgID, auth.RoleOrgManager) || auth.HasOrgRole(ctx, orgID, auth.RoleOrgAdmin)) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("forbidden"))
	}
	if isReservedRoleName(req.Msg.GetName()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("reserved role name"))
	}
	if err := validatePermissionsJSON(req.Msg.GetPermissionsJson()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	role := &database.OrgRole{ID: generateID("role"), OrganizationID: orgID, Name: req.Msg.GetName(), Permissions: req.Msg.GetPermissionsJson()}
	if err := database.DB.Create(role).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create role: %w", err))
	}
	return connect.NewResponse(&adminv1.CreateRoleResponse{Role: &adminv1.Role{Id: role.ID, Name: role.Name, PermissionsJson: role.Permissions}}), nil
}

func (s *Service) UpdateRole(ctx context.Context, req *connect.Request[adminv1.UpdateRoleRequest]) (*connect.Response[adminv1.UpdateRoleResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	user, _ := auth.GetUserFromContext(ctx)
	if !(auth.HasRole(user, auth.RoleAdmin) || isOrgOwner(ctx, orgID) || auth.HasOrgRole(ctx, orgID, auth.RoleOrgManager) || auth.HasOrgRole(ctx, orgID, auth.RoleOrgAdmin)) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("forbidden"))
	}
	var role database.OrgRole
	if err := database.DB.First(&role, "id = ? AND organization_id = ?", req.Msg.GetId(), orgID).Error; err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("role not found"))
	}
	if n := req.Msg.GetName(); n != "" {
		if isReservedRoleName(n) {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("reserved role name"))
		}
		role.Name = n
	}
	if p := req.Msg.GetPermissionsJson(); p != "" {
		if err := validatePermissionsJSON(p); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		role.Permissions = p
	}
	if err := database.DB.Save(&role).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update role: %w", err))
	}
	return connect.NewResponse(&adminv1.UpdateRoleResponse{Role: &adminv1.Role{Id: role.ID, Name: role.Name, PermissionsJson: role.Permissions}}), nil
}

func (s *Service) DeleteRole(ctx context.Context, req *connect.Request[adminv1.DeleteRoleRequest]) (*connect.Response[adminv1.DeleteRoleResponse], error) {
	// We could enforce permission here if needed similar to UpdateRole
	if err := database.DB.Delete(&database.OrgRole{}, "id = ?", req.Msg.GetId()).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete role: %w", err))
	}
	return connect.NewResponse(&adminv1.DeleteRoleResponse{Success: true}), nil
}

// Role Binding CRUD
func (s *Service) ListRoleBindings(ctx context.Context, req *connect.Request[adminv1.ListRoleBindingsRequest]) (*connect.Response[adminv1.ListRoleBindingsResponse], error) {
	var rows []database.OrgRoleBinding
	if err := database.DB.Where("organization_id = ?", req.Msg.GetOrganizationId()).Find(&rows).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list bindings: %w", err))
	}
	out := make([]*adminv1.RoleBinding, 0, len(rows))
	for _, b := range rows {
		out = append(out, &adminv1.RoleBinding{
			Id: b.ID, OrganizationId: b.OrganizationID, UserId: b.UserID, RoleId: b.RoleID,
			ResourceType: b.ResourceType, ResourceId: b.ResourceID, ResourceSelector: b.ResourceSelector,
		})
	}
	return connect.NewResponse(&adminv1.ListRoleBindingsResponse{Bindings: out}), nil
}

func (s *Service) CreateRoleBinding(ctx context.Context, req *connect.Request[adminv1.CreateRoleBindingRequest]) (*connect.Response[adminv1.CreateRoleBindingResponse], error) {
	orgID := req.Msg.GetOrganizationId()
	user, _ := auth.GetUserFromContext(ctx)
	if !(auth.HasRole(user, auth.RoleAdmin) || isOrgOwner(ctx, orgID) || auth.HasOrgRole(ctx, orgID, auth.RoleOrgManager) || auth.HasOrgRole(ctx, orgID, auth.RoleOrgAdmin)) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("forbidden"))
	}
	// Validate role belongs to org and is not reserved
	var chk database.OrgRole
	if err := database.DB.First(&chk, "id = ? AND organization_id = ?", req.Msg.GetRoleId(), orgID).Error; err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid role_id"))
	}
	if isReservedRoleName(chk.Name) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("cannot bind reserved role"))
	}
	b := &database.OrgRoleBinding{
		ID:               generateID("bind"),
		OrganizationID:   orgID,
		UserID:           req.Msg.GetUserId(),
		RoleID:           req.Msg.GetRoleId(),
		ResourceType:     req.Msg.GetResourceType(),
		ResourceID:       req.Msg.GetResourceId(),
		ResourceSelector: req.Msg.GetResourceSelector(),
	}
	if err := database.DB.Create(b).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create binding: %w", err))
	}
	return connect.NewResponse(&adminv1.CreateRoleBindingResponse{Binding: &adminv1.RoleBinding{
		Id: b.ID, OrganizationId: b.OrganizationID, UserId: b.UserID, RoleId: b.RoleID,
		ResourceType: b.ResourceType, ResourceId: b.ResourceID, ResourceSelector: b.ResourceSelector,
	}}), nil
}

func (s *Service) DeleteRoleBinding(ctx context.Context, req *connect.Request[adminv1.DeleteRoleBindingRequest]) (*connect.Response[adminv1.DeleteRoleBindingResponse], error) {
	if err := database.DB.Delete(&database.OrgRoleBinding{}, "id = ?", req.Msg.GetId()).Error; err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("delete binding: %w", err))
	}
	return connect.NewResponse(&adminv1.DeleteRoleBindingResponse{Success: true}), nil
}

func generateID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
