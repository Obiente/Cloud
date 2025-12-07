package common

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"

	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"

	"connectrpc.com/connect"
	"gorm.io/gorm"
)

// AuthorizeOrgRoles checks if a user has the required role(s) in an organization.
// Superadmins bypass all checks. If no allowedRoles are specified, any active member is allowed.
func AuthorizeOrgRoles(ctx context.Context, orgID string, user *authv1.User, allowedRoles ...string) error {
	if user == nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	if auth.IsSuperadmin(ctx, user) {
		return nil
	}

	var member database.OrganizationMember
	if err := database.DB.First(&member, "organization_id = ? AND user_id = ?", orgID, user.Id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not a member of this organization"))
		}
		return connect.NewError(connect.CodeInternal, fmt.Errorf("membership lookup: %w", err))
	}

	if !strings.EqualFold(member.Status, "active") {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("inactive members cannot perform this action"))
	}

	if len(allowedRoles) == 0 {
		return nil
	}

	// member.Role should always contain a role ID (e.g., "system:owner" for system roles, or custom role ID)
	roleID := member.Role
	
	// Check if member's role ID matches any of the allowed roles
	// allowedRoles can be either system role names (e.g., "owner", "admin") or role IDs
	for _, allowed := range allowedRoles {
		allowedLower := strings.ToLower(allowed)
		
		// Check if allowed is a system role name - convert to ID
		if systemRoleID := auth.GetSystemRoleID(allowedLower); systemRoleID != "" {
			if roleID == systemRoleID {
				return nil
			}
		}
		
		// Check if allowed is already a role ID (matches exactly)
		if roleID == allowed {
			return nil
		}
	}

	return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("insufficient role to perform this action"))
}

// AuthorizeOrgAdmin checks if a user is an admin or owner of an organization.
// Superadmins bypass all checks.
func AuthorizeOrgAdmin(ctx context.Context, orgID string, user *authv1.User) error {
	return AuthorizeOrgRoles(ctx, orgID, user, "owner", "admin")
}

// GetOrganizationMember retrieves an organization member record for a user.
// Returns nil, nil if the user is a superadmin (they don't need a member record).
func GetOrganizationMember(ctx context.Context, orgID string, user *authv1.User) (*database.OrganizationMember, error) {
	if user == nil {
		return nil, fmt.Errorf("user is nil")
	}

	if auth.IsSuperadmin(ctx, user) {
		return nil, nil // Superadmins don't need member records
	}

	var member database.OrganizationMember
	if err := database.DB.First(&member, "organization_id = ? AND user_id = ?", orgID, user.Id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("membership lookup: %w", err)
	}

	return &member, nil
}

// VerifyOrgAccess checks if a user has access to an organization (is a member or superadmin).
// This is a lighter check than AuthorizeOrgRoles - it just verifies membership.
func VerifyOrgAccess(ctx context.Context, orgID string, user *authv1.User) error {
	if user == nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}

	if auth.IsSuperadmin(ctx, user) {
		return nil
	}

	member, err := GetOrganizationMember(ctx, orgID, user)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("check organization access: %w", err))
	}

	if member == nil {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("organization not found or access denied"))
	}

	return nil
}

