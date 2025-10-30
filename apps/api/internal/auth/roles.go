package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"api/internal/database"
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
func isOrgOwner(ctx context.Context, orgID string) bool {
    user, err := GetUserFromContext(ctx)
    if err != nil || user == nil {
        return false
    }
    var cnt int64
    _ = database.DB.Model(&database.OrganizationMember{}).
        Where("organization_id = ? AND user_id = ? AND role = ?", orgID, user.Id, "owner").
        Count(&cnt).Error
    return cnt > 0
}

// CheckScopedPermission verifies user has permission in org, optionally scoped to resource
func (p *PermissionChecker) CheckScopedPermission(ctx context.Context, orgID string, sp ScopedPermission) error {
	user, err := GetUserFromContext(ctx)
	if err != nil {
		return fmt.Errorf("unauthenticated")
	}
    // Global admin or org-level owners/managers/admins permitted
    if HasRole(user, RoleAdmin) || isOrgOwner(ctx, orgID) || HasOrgRole(ctx, orgID, RoleOrgManager) || HasOrgRole(ctx, orgID, RoleOrgAdmin) {
		return nil
	}
	// Load bindings and roles
	var bindings []database.OrgRoleBinding
	if err := database.DB.Where("organization_id = ? AND user_id = ?", orgID, user.Id).Find(&bindings).Error; err != nil {
		return fmt.Errorf("permission lookup failed")
	}
	if len(bindings) == 0 {
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
		var perms []string
		_ = json.Unmarshal([]byte(r.Permissions), &perms)
		for _, perm := range perms {
			if perm == sp.Permission {
				// Resource scoping resolution
				for _, b := range bindings {
					if b.RoleID != r.ID {
						continue
					}
					// Org-wide
					if b.ResourceType == "" && b.ResourceID == "" && b.ResourceSelector == "" {
						return nil
					}
					// Exact or wildcard id match
					if b.ResourceType == sp.ResourceType {
						if b.ResourceID == "*" || b.ResourceID == sp.ResourceID || b.ResourceID == "" {
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
	return fmt.Errorf("permission denied: %s", sp.Permission)
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
