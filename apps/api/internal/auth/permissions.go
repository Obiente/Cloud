package auth

import (
	"context"
	"errors"
)

// Standard role definitions
const (
	// System roles
	RoleAdmin  = "admin"
	RoleOwner  = "owner"
	RoleMember = "member"
	RoleViewer = "viewer"

	// Resource-specific permissions
	PermissionCreate = "create"
	PermissionRead   = "read"
	PermissionUpdate = "update"
	PermissionDelete = "delete"
	PermissionAdmin  = "admin"
)

var (
	// Permission errors
	ErrInsufficientPermission = errors.New("insufficient permissions")
	ErrResourceAccessDenied   = errors.New("access denied for this resource")
	ErrNotResourceOwner       = errors.New("user is not the owner of this resource")
)

// ResourceType represents different types of resources that can be protected
type ResourceType string

const (
	ResourceTypeDeployment   ResourceType = "deployment"
	ResourceTypeOrganization ResourceType = "organization"
	ResourceTypeUser         ResourceType = "user"
	ResourceTypeVPS          ResourceType = "vps"
	ResourceTypeDatabase     ResourceType = "database"
)

// PermissionChecker handles checking permissions for users
type PermissionChecker struct {
	// We could add database connection here for more complex checks
}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker() *PermissionChecker {
	return &PermissionChecker{}
}

// CheckPermission checks if the user has the required permission for a resource
func (pc *PermissionChecker) CheckPermission(ctx context.Context, resourceType ResourceType, resourceID string, permission string) error {
	// Get user info from context
	userInfo, err := GetUserFromContext(ctx)
	if err != nil {
		return err
	}

	// Admin role has all permissions
	if pc.HasRole(userInfo, RoleAdmin) {
		return nil
	}

	// Check resource-specific permissions
	switch resourceType {
	case ResourceTypeDeployment:
		return pc.checkDeploymentPermission(ctx, userInfo, resourceID, permission)
	case ResourceTypeOrganization:
		return pc.checkOrganizationPermission(ctx, userInfo, resourceID, permission)
	default:
		// Default to checking role-based permissions
		return pc.checkRolePermission(userInfo, permission)
	}
}

// CheckOwnership verifies if the user is the owner of a resource
func (pc *PermissionChecker) CheckOwnership(ctx context.Context, resourceType ResourceType, resourceID string, ownerID string) error {
	// Get user info from context
	userInfo, err := GetUserFromContext(ctx)
	if err != nil {
		return err
	}

	// Admin role bypasses ownership check
	if pc.HasRole(userInfo, RoleAdmin) {
		return nil
	}

	// Check if the user is the owner
	if userInfo.ID == ownerID {
		return nil
	}

	// If not the direct owner, check if they have admin permissions on the resource
	// This could involve checking organization membership, etc.

	return ErrNotResourceOwner
}

// HasRole checks if the user has a specific role
func (pc *PermissionChecker) HasRole(userInfo *UserInfo, role string) bool {
	for _, r := range userInfo.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// checkDeploymentPermission checks permissions for deployment resources
func (pc *PermissionChecker) checkDeploymentPermission(ctx context.Context, userInfo *UserInfo, deploymentID, permission string) error {
	// In a real implementation, you would query the database to check:
	// 1. If user is the deployment creator (full access)
	// 2. If user is organization admin/owner with access to this deployment
	// 3. If user has been granted specific access to this deployment

	// For now, implement a simplified version:
	if pc.HasRole(userInfo, RoleOwner) || pc.HasRole(userInfo, RoleAdmin) {
		return nil // Owners and admins have full access
	}

	if pc.HasRole(userInfo, RoleMember) {
		// Members can read, update, but not delete
		if permission == PermissionRead || permission == PermissionUpdate || permission == PermissionCreate {
			return nil
		}
	}

	if pc.HasRole(userInfo, RoleViewer) {
		// Viewers can only read
		if permission == PermissionRead {
			return nil
		}
	}

	return ErrInsufficientPermission
}

// checkOrganizationPermission checks permissions for organization resources
func (pc *PermissionChecker) checkOrganizationPermission(ctx context.Context, userInfo *UserInfo, orgID, permission string) error {
	// Check if user has the required role within this organization
	// This would require a database query in a real implementation

	if pc.HasRole(userInfo, RoleOwner) {
		return nil // Owners have full access
	}

	if pc.HasRole(userInfo, RoleAdmin) {
		// Admins can do everything except delete the organization
		if permission != PermissionDelete {
			return nil
		}
	}

	if pc.HasRole(userInfo, RoleMember) {
		// Members can read and create within the organization
		if permission == PermissionRead || permission == PermissionCreate {
			return nil
		}
	}

	if pc.HasRole(userInfo, RoleViewer) {
		// Viewers can only read
		if permission == PermissionRead {
			return nil
		}
	}

	return ErrInsufficientPermission
}

// checkRolePermission does a general role-based permission check
func (pc *PermissionChecker) checkRolePermission(userInfo *UserInfo, permission string) error {
	// Define which roles have which permissions by default
	switch permission {
	case PermissionCreate:
		if pc.HasRole(userInfo, RoleOwner) || pc.HasRole(userInfo, RoleAdmin) || pc.HasRole(userInfo, RoleMember) {
			return nil
		}
	case PermissionRead:
		if pc.HasRole(userInfo, RoleOwner) || pc.HasRole(userInfo, RoleAdmin) || pc.HasRole(userInfo, RoleMember) || pc.HasRole(userInfo, RoleViewer) {
			return nil
		}
	case PermissionUpdate:
		if pc.HasRole(userInfo, RoleOwner) || pc.HasRole(userInfo, RoleAdmin) || pc.HasRole(userInfo, RoleMember) {
			return nil
		}
	case PermissionDelete:
		if pc.HasRole(userInfo, RoleOwner) || pc.HasRole(userInfo, RoleAdmin) {
			return nil
		}
	case PermissionAdmin:
		if pc.HasRole(userInfo, RoleOwner) || pc.HasRole(userInfo, RoleAdmin) {
			return nil
		}
	}

	return ErrInsufficientPermission
}

// RequirePermission is a helper that returns an error if the user doesn't have the required permission
func RequirePermission(ctx context.Context, resourceType ResourceType, resourceID string, permission string) error {
	pc := NewPermissionChecker()
	return pc.CheckPermission(ctx, resourceType, resourceID, permission)
}

// RequireOwnership is a helper that returns an error if the user is not the owner
func RequireOwnership(ctx context.Context, resourceType ResourceType, resourceID string, ownerID string) error {
	pc := NewPermissionChecker()
	return pc.CheckOwnership(ctx, resourceType, resourceID, ownerID)
}

// GetUserRole gets the user's highest role
func GetUserRole(userInfo *UserInfo) string {
	if userInfo == nil {
		return ""
	}

	// Check from highest to lowest privilege
	if HasRole(userInfo, RoleAdmin) {
		return RoleAdmin
	}

	if HasRole(userInfo, RoleOwner) {
		return RoleOwner
	}

	if HasRole(userInfo, RoleMember) {
		return RoleMember
	}

	if HasRole(userInfo, RoleViewer) {
		return RoleViewer
	}

	return ""
}

// HasRole is a helper function to check if a user has a specific role
func HasRole(userInfo *UserInfo, role string) bool {
	if userInfo == nil {
		return false
	}

	for _, r := range userInfo.Roles {
		if r == role {
			return true
		}
	}

	return false
}

// HasAnyRole checks if a user has any of the specified roles
func HasAnyRole(userInfo *UserInfo, roles ...string) bool {
	if userInfo == nil {
		return false
	}

	for _, role := range roles {
		if HasRole(userInfo, role) {
			return true
		}
	}

	return false
}
