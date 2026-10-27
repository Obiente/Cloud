package auth

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/obiente/cloud/apps/shared/pkg/database"
	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"
	"gorm.io/gorm"
)

// ModerationError preserves a machine-readable restriction for clients without
// turning database outages into bans or ordinary permission failures.
func ModerationError(err error, orgID string) error {
	if err == nil {
		return nil
	}
	code := connect.CodeUnavailable
	scope := ""
	switch {
	case errors.Is(err, database.ErrUserRestricted):
		code, scope = connect.CodePermissionDenied, "account"
	case errors.Is(err, database.ErrOrganizationRestricted):
		code, scope = connect.CodePermissionDenied, "organization"
	case errors.Is(err, gorm.ErrRecordNotFound):
		code = connect.CodeNotFound
	}
	rpcErr := connect.NewError(code, err)
	if scope != "" {
		rpcErr.Meta().Set("X-Obiente-Restriction", scope)
		if scope == "organization" {
			rpcErr.Meta().Set("X-Obiente-Organization-ID", orgID)
		}
	}
	return rpcErr
}

func EnsureActiveUser(ctx context.Context) error {
	user, err := GetUserFromContext(ctx)
	if err != nil || user == nil {
		return connect.NewError(connect.CodeUnauthenticated, ErrNoToken)
	}
	if ctx.Value(systemUserKey) == true {
		return nil
	}
	return ModerationError(database.EnsureUserActive(ctx, database.DB, user.Id), "")
}

// CanInspectOrganization grants only read access to platform administrators.
// Organization owners/admins never qualify through their tenant roles.
func CanInspectOrganization(ctx context.Context, user *authv1.User, permission string) bool {
	permission = normalizePermission("", permission)
	if !strings.HasSuffix(permission, ".read") && !strings.HasSuffix(permission, ".logs") {
		return false
	}
	if HasRole(user, RoleAdmin) || HasRole(user, RoleSuperAdmin) {
		return true
	}
	domain := strings.SplitN(permission, ".", 2)[0]
	platformDomain := map[string]string{
		"deployment": "deployments", "gameservers": "gameservers",
		"database": "databases", "vps": "vps", "organization": "organizations",
	}[domain]
	return platformDomain != "" && HasSuperadminPermission(ctx, user, "superadmin."+platformDomain+".read")
}

// CheckOrganizationAccess enforces moderation before ownership, membership, or
// list filtering. A banned administrator is still a banned account. This does
// not replace the caller's normal resource or membership authorization.
func CheckOrganizationAccess(ctx context.Context, orgID, permission string) error {
	if err := EnsureActiveUser(ctx); err != nil {
		return err
	}
	err := database.EnsureOrganizationActive(ctx, database.DB, orgID)
	if errors.Is(err, database.ErrOrganizationRestricted) {
		user, _ := GetUserFromContext(ctx)
		if ctx.Value(systemUserKey) != true && CanInspectOrganization(ctx, user, permission) {
			return nil
		}
		// A denied administrative mutation must not lock its read-only view.
		readPermission := strings.SplitN(permission, ".", 2)[0] + ".read"
		if ctx.Value(systemUserKey) != true && CanInspectOrganization(ctx, user, readPermission) {
			return connect.NewError(connect.CodePermissionDenied, err)
		}
	}
	return ModerationError(err, orgID)
}

func checkRequestOrganizationAccess(ctx context.Context, req connect.AnyRequest) error {
	// Dedicated platform moderation methods authorize their own operations, such
	// as lifting a ban. They still pass through current account moderation.
	if strings.HasPrefix(req.Spec().Procedure, "/obiente.cloud.superadmin.v1.SuperadminService/") {
		return nil
	}
	msg, ok := req.Any().(interface{ GetOrganizationId() string })
	if !ok || msg.GetOrganizationId() == "" {
		return nil
	}
	permission, _, _ := inferPermissionFromProcedure(req.Spec().Procedure)
	return CheckOrganizationAccess(ctx, msg.GetOrganizationId(), permission)
}
