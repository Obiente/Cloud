# Runtime Permission Registry System

This document explains how the runtime permission registry system works and how to use it.

## Overview

The permissions system automatically:
1. **Registers permissions** from service procedure definitions at runtime
2. **Maps RPC calls to permissions** automatically via middleware
3. **Checks permissions** before RPC execution
4. **Maintains backward compatibility** when procedures change

## How It Works

### Permission Registry

The `PermissionRegistry` is a central registry that maps RPC procedures to permissions. It:
- Stores procedure-to-permission mappings
- Tracks public endpoints (no auth required)
- Maintains backward compatibility mappings
- Auto-discovers permissions from procedure paths

### Service Registration

All services register their procedures at startup via `auth.RegisterAllServices()`. This is called automatically in `auth-service/main.go`.

Each service has a registration function (e.g., `RegisterDeploymentServiceProcedures()`) that:
- Maps procedure paths to method names
- Specifies which procedures are public
- Automatically generates permissions from service/method names

### Permission Mapping Rules

**Service → Resource Type:**
- `DeploymentService` → `deployment`
- `VPSService` → `vps`
- `GameServerService` → `gameserver`
- `BillingService` → `billing`
- `OrganizationService` → `organization`
- `SupportService` → `support`
- `NotificationService` → `notification`
- `AdminService` → `admin`
- `SuperadminService` → `superadmin`
- `AuditService` → `audit`

**RPC → Action:**
- `Create*`, `Add*` → `create`
- `List*`, `Get*`, `Query*`, `Stream*` → `read`
- `Update*`, `Set*`, `Upsert*` → `update`
- `Delete*`, `Remove*` → `delete`
- `Start*` → `start`
- `Stop*` → `stop`
- `Restart*` → `restart`
- `Scale*` → `scale`
- `Trigger*` → `trigger`
- `*Log*` → `logs`
- `*Metric*`, `*Usage*` → `read`
- Default → `manage`

**Example:**
- `DeploymentService/CreateDeployment` → `deployment.create`
- `VPSService/ListVPS` → `vps.read`
- `BillingService/UpdateBillingAccount` → `billing.update`

## Using Permission Middleware

### Basic Usage

Add the permission middleware to your service handlers:

```go
import (
    "github.com/obiente/cloud/apps/shared/pkg/auth"
    "connectrpc.com/connect"
)

// In your main.go
authInterceptor := auth.MiddlewareInterceptor(authConfig)
permissionInterceptor := auth.PermissionMiddleware() // Add this

path, handler := servicev1connect.NewServiceHandler(
    service,
    connect.WithInterceptors(
        auditInterceptor,
        authInterceptor,
        permissionInterceptor, // Add permission checking
    ),
)
```

### How It Works

1. **Extracts RPC information** from `req.Spec().Procedure`
2. **Looks up permission** in the registry
3. **Falls back to inference** if not registered (auto-registers for future use)
4. **Extracts organization ID** from request message
5. **Checks permission** using existing `CheckScopedPermission` logic
6. **Allows or denies** the request

### Public Endpoints

The middleware automatically skips permission checks for:
- `/obiente.cloud.auth.v1.AuthService/Login`
- `/obiente.cloud.auth.v1.AuthService/GetPublicConfig`
- `/obiente.cloud.superadmin.v1.SuperadminService/GetPricing`

You can register additional public procedures in the service registration functions.

### Superadmin Bypass

Superadmins automatically bypass all permission checks.

## Adding New Services

When you add a new service:

1. **Add registration function** in `apps/shared/pkg/auth/register_procedures.go`:
   ```go
   func RegisterNewServiceProcedures() {
       public := []string{} // or list public procedures
       procedures := map[string]string{
           "/obiente.cloud.newservice.v1.NewService/Method1": "Method1",
           "/obiente.cloud.newservice.v1.NewService/Method2": "Method2",
       }
       RegisterServiceProcedures("NewService", procedures, public)
   }
   ```

2. **Add to RegisterAllServices()**:
   ```go
   func RegisterAllServices() {
       // ... existing services
       RegisterNewServiceProcedures()
   }
   ```

3. **Ensure service is registered** - `RegisterAllServices()` is called in `auth-service/main.go`

## Backward Compatibility

The registry maintains backward compatibility:

- **Existing permissions** continue to work even if procedure paths change
- **Permission strings** remain stable (e.g., `deployment.create`)
- **Multiple procedures** can map to the same permission
- Use `EnsureBackwardCompatibility()` to migrate old procedures to new ones

## Manual Permission Checks

You can still manually check permissions in service handlers:

```go
import (
    "github.com/obiente/cloud/apps/shared/pkg/auth"
)

func (s *Service) CreateDeployment(ctx context.Context, req *connect.Request[deploymentsv1.CreateDeploymentRequest]) (*connect.Response[deploymentsv1.CreateDeploymentResponse], error) {
    user, err := auth.GetUserFromContext(ctx)
    if err != nil {
        return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
    }

    orgID := req.Msg.GetOrganizationId()
    
    // Manual permission check
    pc := auth.NewPermissionChecker()
    sp := auth.ScopedPermission{
        Permission:   auth.PermDeploymentsCreate,
        ResourceType: "deployment",
        ResourceID:   "",
    }
    if err := pc.CheckScopedPermission(ctx, orgID, sp); err != nil {
        return nil, connect.NewError(connect.CodePermissionDenied, err)
    }

    // ... rest of handler
}
```

## Listing Permissions

The `ListPermissions` RPC automatically includes:
- All registered permissions from the registry
- Manual permissions (from `scopes.go`)
- Auto-discovered permissions (from procedure inference)

All permissions are grouped by resource type and sorted for display in the UI.

## Best Practices

1. **Register all procedures** when adding new services
2. **Use middleware** for automatic permission checking (recommended)
3. **Manual checks** only when you need custom logic
4. **Test permissions** by creating roles with specific permissions
5. **Maintain backward compatibility** when renaming procedures

## Future Improvements

- Reflection-based auto-discovery of Procedure constants
- Permission inheritance and hierarchies
- Resource-level permission scoping
- Permission versioning and migration tools
