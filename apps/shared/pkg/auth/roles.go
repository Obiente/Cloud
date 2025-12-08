package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/obiente/cloud/apps/shared/pkg/database"
)

const (
	RoleOrgManager = "org.manager" // billing/admin
	RoleOrgAdmin   = "org.admin"   // manage projects/resources
	RoleOrgMember  = "org.member"  // default
)

// ScopedPermission is a structured permission: domain.action, optional resource
type ScopedPermission struct {
	Permission   string // e.g., deployments.create
	ResourceType string // e.g., deployment
	ResourceID   string
}

// HasOrgRole returns true if user has the specified org-level role
func HasOrgRole(ctx context.Context, orgID, role string) bool {
	user, err := GetUserFromContext(ctx)
	if err != nil {
		return false
	}
	// Global admin via token roles still wins
	if HasRole(user, RoleAdmin) {
		return true
	}
	// Check role bindings
	var bindings []database.OrgRoleBinding
	if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).Find(&bindings).Error; err != nil {
		return false
	}
	if len(bindings) == 0 {
		return false
	}
	var roles []database.OrgRole
	roleIDs := make([]string, 0, len(bindings))
	for _, b := range bindings {
		roleIDs = append(roleIDs, b.RoleID)
	}
	if err := database.DB.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
		return false
	}
	for _, r := range roles {
		if r.Name == role {
			return true
		}
	}
	return false
}

// isOrgOwner checks organization_members table for an 'owner' member
// The role field should contain a role ID (e.g., "system:owner" for system roles)
func isOrgOwner(ctx context.Context, orgID string) bool {
    user, err := GetUserFromContext(ctx)
    if err != nil || user == nil {
        return false
    }
    var cnt int64
    // Check for system role ID
    _ = database.DB.Model(&database.OrganizationMember{}).
        Where("organization_id = ? AND user_id = ? AND role = ?", orgID, user.Id, SystemRoleIDOwner).
        Count(&cnt).Error
    return cnt > 0
}

// CheckScopedPermission verifies user has permission in org, optionally scoped to resource
func (p *PermissionChecker) CheckScopedPermission(ctx context.Context, orgID string, sp ScopedPermission) error {
	// Normalize permission to canonical format automatically
	sp.Permission = normalizePermission(sp.ResourceType, sp.Permission)

	user, err := GetUserFromContext(ctx)
	if err != nil {
		return fmt.Errorf("unauthenticated")
	}
    // Global admin or org-level owners/managers/admins permitted
    if HasRole(user, RoleAdmin) || isOrgOwner(ctx, orgID) || HasOrgRole(ctx, orgID, RoleOrgManager) || HasOrgRole(ctx, orgID, RoleOrgAdmin) {
		return nil
	}

	// First check role from organization_members table (can be system role or custom role)
	// The role field should always contain a role ID
	var member database.OrganizationMember
	if err := database.DB.Where("organization_id = ? AND user_id = ? AND status = ?", orgID, user.Id, "active").First(&member).Error; err == nil {
		roleID := member.Role
		
		// Check if it's a system role ID
		if IsSystemRoleID(roleID) {
			if CheckSystemRolePermissionByID(roleID, sp.Permission) {
				// System role has permission, org-wide access is granted (no resource scoping)
				return nil
			}
		} else {
			// It's a custom role assigned directly - look it up in the database by ID
			var customRole database.OrgRole
			lookupErr := database.DB.Where("id = ? AND organization_id = ?", roleID, orgID).First(&customRole).Error
			if lookupErr == nil {
				// Found the custom role, check its permissions
				var perms []string
				if err := json.Unmarshal([]byte(customRole.Permissions), &perms); err == nil {
					for _, perm := range perms {
						// Check exact match or wildcard match
						if perm == sp.Permission || matchesPermission(perm, sp.Permission) {
							// Custom role assigned directly grants org-wide access (no resource scoping)
							return nil
						}
					}
				}
			}
		}
	}

	// Check custom roles via role bindings (even if user has a direct role assignment without permission)
	var bindings []database.OrgRoleBinding
	if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).Find(&bindings).Error; err != nil {
		return fmt.Errorf("permission lookup failed")
	}
	if len(bindings) == 0 {
		// No custom role bindings, and direct role assignment doesn't have permission
		return fmt.Errorf("permission denied: %s", sp.Permission)
	}
	var roles []database.OrgRole
	roleIDs := make([]string, 0, len(bindings))
	for _, b := range bindings {
		roleIDs = append(roleIDs, b.RoleID)
	}
	if err := database.DB.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
		return fmt.Errorf("permission lookup failed")
	}
	// Evaluate permissions JSON
	for _, r := range roles {
		// Skip system roles in database (they shouldn't be there, but just in case)
		if IsSystemRole(r.Name) {
			continue
		}
		var perms []string
		_ = json.Unmarshal([]byte(r.Permissions), &perms)
		for _, perm := range perms {
			// Check exact match or wildcard match
			if perm == sp.Permission || matchesPermission(perm, sp.Permission) {
				// Resource scoping resolution
				for _, b := range bindings {
					if b.RoleID != r.ID {
						continue
					}
					// Org-wide binding (all scoping fields empty)
					// ResourceSelector can be empty string or "{}" (empty JSON object)
					isSelectorEmpty := b.ResourceSelector == "" || b.ResourceSelector == "{}"
					if b.ResourceType == "" && b.ResourceID == "" && isSelectorEmpty {
						return nil
					}
					// If checking org-wide permission (sp.ResourceID is empty), allow if binding is org-wide or matches resource type with empty/wildcard ID
					if sp.ResourceID == "" {
						// Org-wide check: allow if binding is org-wide OR if binding matches resource type with empty/wildcard ID
						if b.ResourceType == "" || (b.ResourceType == sp.ResourceType && (b.ResourceID == "" || b.ResourceID == "*")) {
							return nil
						}
					} else {
						// Specific resource check: must match resource type and ID
						if b.ResourceType == sp.ResourceType {
							if b.ResourceID == "*" || b.ResourceID == sp.ResourceID {
							return nil
						}
						// Selector-based match (e.g., environment)
						if b.ResourceSelector != "" {
							if matchesSelector(sp.ResourceType, sp.ResourceID, b.ResourceSelector) {
								return nil
							}
						}
					}
					// Environment-to-deployment fallback: if binding is for environment and permission targets a deployment, resolve deployment's environment
					if b.ResourceType == "environment" && sp.ResourceType == "deployment" {
						if matchesEnvironmentForDeployment(sp.ResourceID, b) {
							return nil
							}
						}
					}
				}
			}
		}
	}
	return fmt.Errorf("permission denied: %s", sp.Permission)
}

// matchesPermission checks if a permission pattern (with wildcards) matches a specific permission
// Examples:
//   - "organization.members.*" matches "organization.members.invite"
//   - "organization.admin.*" matches "organization.admin.add_credits"
//   - "organization.*" matches auth.PermissionOrganizationUpdate
//   - "deployment.*" matches "deployment.create"
func matchesPermission(pattern, permission string) bool {
	// Exact match
	if pattern == permission {
		return true
	}
	
	// Wildcard match: pattern ends with ".*"
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(permission, prefix+".")
	}
	
	return false
}

// matchesSelector determines if a selector JSON matches the target resource's attributes
func matchesSelector(resourceType, resourceID, selectorJSON string) bool {
	// Minimal implementation: support {"environment":"name"}
	if resourceType == "environment" {
		// Exact match handled elsewhere
		return false
	}
	if resourceType == "deployment" {
		// Fallback handled in matchesEnvironmentForDeployment
		return false
	}
	return false
}

func matchesEnvironmentForDeployment(deploymentID string, b database.OrgRoleBinding) bool {
	// selector example: {"environment":"production"}
	if b.ResourceSelector == "" {
		return false
	}
	var sel map[string]string
	_ = json.Unmarshal([]byte(b.ResourceSelector), &sel)
	env, ok := sel["environment"]
	if !ok || env == "" {
		return false
	}
	// lookup deployment and compare environment
	var dep database.Deployment
	if err := database.DB.First(&dep, "id = ?", deploymentID).Error; err != nil {
		return false
	}
	// dep.Environment is int32; match by common names
	switch env {
	case "production":
		return dep.Environment == 1
	case "staging":
		return dep.Environment == 2
	case "development":
		return dep.Environment == 3
	default:
		return false
	}
}

// resourceTypeToPermissionPrefix maps resource types to their canonical permission prefixes
// This ensures consistency: deployment -> "deployment", gameserver -> "gameservers", etc.
var resourceTypeToPermissionPrefix = map[string]string{
	"deployment":  ResourcePrefixDeployment,  // singular, matches system roles
	"gameserver": ResourcePrefixGameServers, // plural, matches system roles
	"vps":        ResourcePrefixVPS,         // singular, matches system roles
	"database":   ResourcePrefixDatabase,    // singular
	// Also handle plural forms that might be passed in
	"deployments": ResourcePrefixDeployment,
	"gameservers": ResourcePrefixGameServers,
	"databases":   ResourcePrefixDatabase,
}

// normalizePermission converts any permission string to canonical format
// It handles:
//   - Legacy actions: "view" -> "read", "edit" -> "update", etc.
//   - Resource type normalization: "gameserver" -> "gameservers", "deployments" -> "deployment"
//   - Already normalized permissions: passes through unchanged
// Examples:
//   - normalizePermission("gameserver", "read") -> "gameservers.read"
//   - normalizePermission("gameserver", "view") -> "gameservers.read"
//   - normalizePermission("deployment", "read") -> "deployment.read"
//   - normalizePermission("deployments", "read") -> "deployment.read"
//   - normalizePermission("", "deployments.read") -> "deployment.read"
//   - normalizePermission("", "gameservers.read") -> "gameservers.read"
func normalizePermission(resourceType, permission string) string {
	// Action normalization map
	actionMap := map[string]string{
		"view":    "read",
		"edit":    "update",
		"manage":  "update",
		"write":   "update",
		"start":   "start",
		"stop":    "stop",
		"restart": "restart",
		"delete":  "delete",
		"create":  "create",
		"deploy":  "deploy",
	}

	// If permission already contains a dot, it's in format "resource.action"
	if strings.Contains(permission, ".") {
		parts := strings.SplitN(permission, ".", 2)
		if len(parts) == 2 {
			resourcePrefix := parts[0]
			action := parts[1]

			// Normalize resource prefix to canonical form
			if canonicalPrefix, ok := resourceTypeToPermissionPrefix[resourcePrefix]; ok {
				resourcePrefix = canonicalPrefix
			}

			// Normalize action
			if normalizedAction, ok := actionMap[action]; ok {
				action = normalizedAction
			}

			return fmt.Sprintf("%s.%s", resourcePrefix, action)
		}
		// Malformed, return as-is
		return permission
	}

	// Permission is just an action (legacy format), need resource type
	if resourceType == "" {
		// Can't normalize without resource type
		return permission
	}

	// Get canonical permission prefix for resource type
	permissionPrefix := resourceTypeToPermissionPrefix[resourceType]
	if permissionPrefix == "" {
		// Unknown resource type, use as-is
		permissionPrefix = resourceType
	}

	// Normalize action
	normalizedAction := actionMap[permission]
	if normalizedAction == "" {
		normalizedAction = permission // Use as-is if not in map
	}

	// Build canonical permission string
	return fmt.Sprintf("%s.%s", permissionPrefix, normalizedAction)
}

// CheckResourcePermission is a unified permission check for all resources
// It looks up the organization ID from the resource and uses CheckScopedPermission
// resourceType should be one of: "deployment", "gameserver", "vps"
// permission can be legacy format (e.g., "view", "edit") or proper format (e.g., "deployments.read", "gameservers.read")
// All permissions are automatically normalized to canonical format
func (p *PermissionChecker) CheckResourcePermission(ctx context.Context, resourceType, resourceID, permission string) error {
	// Normalize permission to canonical format automatically
	// normalizePermission will handle resource type mapping (e.g., "gameserver" -> "gameservers")
	normalizedPerm := normalizePermission(resourceType, permission)

	// Look up organization ID from resource
	var orgID string
	switch resourceType {
	case "deployment":
		var dep database.Deployment
		if err := database.DB.Select("organization_id").Where("id = ?", resourceID).First(&dep).Error; err != nil {
			return fmt.Errorf("resource not found: %s", resourceID)
		}
		orgID = dep.OrganizationID
	case "gameserver":
		var gs database.GameServer
		if err := database.DB.Select("organization_id").Where("id = ? AND deleted_at IS NULL", resourceID).First(&gs).Error; err != nil {
			return fmt.Errorf("resource not found: %s", resourceID)
		}
		orgID = gs.OrganizationID
	case "vps":
		var vps database.VPSInstance
		if err := database.DB.Select("organization_id").Where("id = ? AND deleted_at IS NULL", resourceID).First(&vps).Error; err != nil {
			return fmt.Errorf("resource not found: %s", resourceID)
		}
		orgID = vps.OrganizationID
	case "database":
		var db database.DatabaseInstance
		if err := database.DB.Select("organization_id").Where("id = ? AND deleted_at IS NULL", resourceID).First(&db).Error; err != nil {
			return fmt.Errorf("resource not found: %s", resourceID)
		}
		orgID = db.OrganizationID
	default:
		return fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	if orgID == "" {
		return fmt.Errorf("resource has no organization ID")
	}

	// Use CheckScopedPermission with resource scoping
	return p.CheckScopedPermission(ctx, orgID, ScopedPermission{
		Permission:   normalizedPerm,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
}
