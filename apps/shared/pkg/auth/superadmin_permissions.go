package auth

import (
	"context"
	"encoding/json"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
)

// IsSuperadmin checks if a user is a superadmin (has any superadmin access).
// This checks both the hardcoded RoleSuperAdmin role and whether the user has
// any superadmin role bindings.
// Returns true if the user is a superadmin, false otherwise.
//
// Use this when you need to check if a user has general superadmin access,
// rather than a specific permission.
func IsSuperadmin(ctx context.Context, user *authv1.User) bool {
	if user == nil {
		return false
	}

	// Check if user has the hardcoded superadmin role
	if HasRole(user, RoleSuperAdmin) {
		return true
	}

	// Check if user has any superadmin role bindings
	var count int64
	if err := database.DB.Model(&database.SuperadminRoleBinding{}).
		Where("user_id = ?", user.Id).
		Count(&count).Error; err != nil {
		return false
	}

	return count > 0
}

// HasSuperadminPermission checks if a user has a specific superadmin permission.
// This checks both the hardcoded RoleSuperAdmin role and superadmin role bindings.
// Returns true if the user has the permission, false otherwise.
//
// Examples:
//   - HasSuperadminPermission(ctx, user, "superadmin.support.read") - checks for specific permission
//   - HasSuperadminPermission(ctx, user, "superadmin.support.*") - checks for wildcard permission
//   - Users with RoleSuperAdmin always return true
//   - Users with superadmin role bindings are checked against their bound roles
func HasSuperadminPermission(ctx context.Context, user *authv1.User, permission string) bool {
	if user == nil {
		return false
	}

	// Check if user has the hardcoded superadmin role
	if HasRole(user, RoleSuperAdmin) {
		return true
	}

	// Check superadmin role bindings
	var bindings []database.SuperadminRoleBinding
	if err := database.DB.Where("user_id = ?", user.Id).Find(&bindings).Error; err != nil {
		logger.Debug("[HasSuperadminPermission] Failed to load superadmin role bindings for user %s: %v", user.Id, err)
		return false
	}

	if len(bindings) == 0 {
		return false
	}

	// Load roles
	var roles []database.SuperadminRole
	roleIDs := make([]string, 0, len(bindings))
	for _, b := range bindings {
		roleIDs = append(roleIDs, b.RoleID)
	}
	if err := database.DB.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
		logger.Debug("[HasSuperadminPermission] Failed to load superadmin roles: %v", err)
		return false
	}

	// Check if any role has the required permission
	for _, r := range roles {
		var perms []string
		if err := json.Unmarshal([]byte(r.Permissions), &perms); err != nil {
			continue
		}
		for _, perm := range perms {
			// Check exact match or wildcard match
			// Use the matchesPermission function from roles.go
			if perm == permission || matchesPermission(perm, permission) {
				return true
			}
		}
	}

	return false
}
