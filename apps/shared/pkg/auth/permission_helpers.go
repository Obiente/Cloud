package auth

import (
	"context"
	"strings"

	"connectrpc.com/connect"
)

// CheckResourcePermissionWithError is a reusable helper that checks resource permissions
// and converts errors to Connect errors with appropriate codes.
// This eliminates the need for service-specific permission helpers.
//
// Usage:
//   if err := auth.CheckResourcePermissionWithError(ctx, pc, "deployment", deploymentID, "read"); err != nil {
//       return nil, err
//   }
func CheckResourcePermissionWithError(ctx context.Context, pc *PermissionChecker, resourceType, resourceID, permission string) error {
	if err := pc.CheckResourcePermission(ctx, resourceType, resourceID, permission); err != nil {
		// Convert to Connect error with appropriate code
		errStr := err.Error()
		if strings.Contains(errStr, "not found") {
			return connect.NewError(connect.CodeNotFound, err)
		}
		if strings.Contains(errStr, "unauthenticated") {
			return connect.NewError(connect.CodeUnauthenticated, err)
		}
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return nil
}

// CheckScopedPermissionWithError is a reusable helper that checks scoped permissions
// and converts errors to Connect errors with appropriate codes.
//
// Usage:
//   if err := auth.CheckScopedPermissionWithError(ctx, pc, orgID, auth.ScopedPermission{
//       Permission: auth.PermissionDeploymentRead,
//       ResourceType: "deployment",
//       ResourceID: deploymentID,
//   }); err != nil {
//       return nil, err
//   }
func CheckScopedPermissionWithError(ctx context.Context, pc *PermissionChecker, orgID string, sp ScopedPermission) error {
	if err := pc.CheckScopedPermission(ctx, orgID, sp); err != nil {
		// Convert to Connect error with appropriate code
		errStr := err.Error()
		if strings.Contains(errStr, "unauthenticated") {
			return connect.NewError(connect.CodeUnauthenticated, err)
		}
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return nil
}

