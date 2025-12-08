package auth

import (
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
)

// Standard role definitions
const (
	// System roles
	RoleSuperAdmin = "superadmin"
	RoleAdmin      = "admin"
	RoleOwner      = "owner"
	RoleMember     = "member"
	RoleViewer     = "viewer"

)

// PermissionChecker handles checking permissions for users
type PermissionChecker struct {
	// We could add database connection here for more complex checks
}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker() *PermissionChecker {
	return &PermissionChecker{}
}

// Note: Old permission checking methods (CheckPermission, CheckOwnership, etc.) have been removed.
// All permission checks should now use CheckScopedPermission or CheckResourcePermission from roles.go

// HasRole is a helper function to check if a user has a specific role
func HasRole(userInfo *authv1.User, role string) bool {
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
