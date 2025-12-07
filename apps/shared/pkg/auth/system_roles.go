package auth

import (
	"strings"
)

// System role IDs - these are the IDs used in organization_members.role field
const (
	SystemRoleIDOwner  = "system:owner"
	SystemRoleIDAdmin  = "system:admin"
	SystemRoleIDMember = "system:member"
	SystemRoleIDViewer = "system:viewer"
	SystemRoleIDNone   = "system:none"
)

// SystemRolePermissions defines the permissions for each system role
// These are hardcoded and not stored in the database
// Key is the role name (lowercase), value is the permissions list
var SystemRolePermissions = map[string][]string{
	"owner": {
		"deployment.*",
		"gameservers.*",
		"vps.*",
		"organization.*",
		"admin.*",
	},
	"admin": {
		"deployment.*",
		"gameservers.*",
		"vps.*",
		"organization.read",
		"organization.update",
		"organization.members.*",
		"admin.*",
	},
	"member": {
		// Deployments (all except delete)
		"deployment.read",
		"deployment.create",
		"deployment.update",
		"deployment.start",
		"deployment.stop",
		"deployment.restart",
		"deployment.scale",
		"deployment.logs",
		// Game Servers (all except delete)
		"gameservers.read",
		"gameservers.create",
		"gameservers.update",
		"gameservers.start",
		"gameservers.stop",
		"gameservers.restart",
		// VPS (all except delete and manage)
		"vps.read",
		"vps.create",
		"vps.update",
		"vps.start",
		"vps.stop",
		"vps.reboot",
		// Organization (read-only)
		"organization.read",
		"organization.members.read",
	},
	"viewer": {
		"deployment.read",
		"deployment.logs",
		"gameservers.read",
		"vps.read",
		"organization.read",
		"organization.members.read",
	},
	"none": {
		// No permissions - users with this role must have permissions granted via role bindings
	},
}

// GetSystemRolePermissions returns the permissions for a system role
func GetSystemRolePermissions(roleName string) []string {
	// Normalize role name to lowercase
	roleName = strings.ToLower(roleName)
	if perms, ok := SystemRolePermissions[roleName]; ok {
		return perms
	}
	return nil
}

// IsSystemRole checks if a role name is a system role
// This function accepts both role names (e.g., "owner") and role IDs (e.g., "system:owner")
func IsSystemRole(roleNameOrID string) bool {
	// First check if it's a system role ID
	if IsSystemRoleID(roleNameOrID) {
		return true
	}
	// Then check if it's a system role name
	roleName := strings.ToLower(roleNameOrID)
	_, ok := SystemRolePermissions[roleName]
	return ok
}

// CheckSystemRolePermission checks if a system role has a specific permission
func CheckSystemRolePermission(roleName, permission string) bool {
	perms := GetSystemRolePermissions(roleName)
	if perms == nil {
		return false
	}

	for _, perm := range perms {
		if perm == permission || matchesPermission(perm, permission) {
			return true
		}
	}
	return false
}

// GetSystemRoleID returns the system role ID for a given role name
// Returns empty string if the role name is not a system role
func GetSystemRoleID(roleName string) string {
	roleName = strings.ToLower(roleName)
	switch roleName {
	case "owner":
		return SystemRoleIDOwner
	case "admin":
		return SystemRoleIDAdmin
	case "member":
		return SystemRoleIDMember
	case "viewer":
		return SystemRoleIDViewer
	case "none":
		return SystemRoleIDNone
	default:
		return ""
	}
}

// GetSystemRoleNameFromID returns the system role name for a given role ID
// Returns empty string if the role ID is not a system role ID
func GetSystemRoleNameFromID(roleID string) string {
	switch roleID {
	case SystemRoleIDOwner:
		return "owner"
	case SystemRoleIDAdmin:
		return "admin"
	case SystemRoleIDMember:
		return "member"
	case SystemRoleIDViewer:
		return "viewer"
	case SystemRoleIDNone:
		return "none"
	default:
		return ""
	}
}

// IsSystemRoleID checks if a role ID is a system role ID
func IsSystemRoleID(roleID string) bool {
	return GetSystemRoleNameFromID(roleID) != ""
}

// CheckSystemRolePermissionByID checks if a system role (by ID) has a specific permission
func CheckSystemRolePermissionByID(roleID, permission string) bool {
	roleName := GetSystemRoleNameFromID(roleID)
	if roleName == "" {
		return false
	}
	return CheckSystemRolePermission(roleName, permission)
}
