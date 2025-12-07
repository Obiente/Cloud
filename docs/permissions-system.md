# Permissions System

This document explains how the permissions and role-based access control (RBAC) system works in Obiente Cloud.

## Overview

Obiente Cloud uses a flexible role-based access control (RBAC) system with:
- **System Roles**: Predefined roles (Owner, Admin, Member, Viewer, None) with hardcoded permissions
- **Custom Roles**: Organization-specific roles with configurable permissions
- **Role Bindings**: Assign roles to users, optionally scoped to specific resources
- **Wildcard Permissions**: Use `*` to grant all permissions for a resource type (e.g., `deployment.*`)
- **Resource Scoping**: Role bindings can be scoped to specific resources, while direct role permissions are always organization-wide

## System Roles

System roles are predefined and cannot be modified or deleted. They are defined in code (`apps/shared/pkg/auth/system_roles.go`) and automatically available to all organizations.

### Owner
- **ID**: `system:owner`
- **Permissions**: Full access to everything
  - `deployment.*`
  - `gameservers.*`
  - `vps.*`
  - `organization.*`
  - `admin.*`
- **Capabilities**: Can manage all resources, billing, delete organization, assign any role

### Admin
- **ID**: `system:admin`
- **Permissions**: Full resource management, limited organization control
  - `deployment.*`
  - `gameservers.*`
  - `vps.*`
  - `organization.read`
  - `organization.update`
  - `organization.members.*`
  - `admin.*`
- **Capabilities**: Can manage all resources, manage members, but cannot delete organization

### Member
- **ID**: `system:member`
- **Permissions**: Create and manage resources, read-only organization access
  - Deployments: `read`, `create`, `update`, `start`, `stop`, `restart`, `scale`, `logs` (no delete)
  - Game Servers: `read`, `create`, `update`, `start`, `stop`, `restart` (no delete)
  - VPS: `read`, `create`, `update`, `start`, `stop`, `reboot` (no delete or manage)
  - Organization: `read`, `members.read`
- **Capabilities**: Can create and manage resources but cannot delete them or manage organization settings

### Viewer
- **ID**: `system:viewer`
- **Permissions**: Read-only access
  - `deployment.read`, `deployment.logs`
  - `gameservers.read`
  - `vps.read`
  - `organization.read`, `organization.members.read`
- **Capabilities**: Can view resources, metrics, and logs but cannot create, update, or delete anything

### None
- **ID**: `system:none`
- **Permissions**: No permissions
- **Capabilities**: Users with this role must have permissions granted via role bindings. Useful for fine-grained permission control.

## Custom Roles

Custom roles are organization-specific and can be created, modified, and deleted by users with `admin.roles.*` permissions.

### Creating Custom Roles

1. Navigate to **Admin > Roles** in your organization
2. Click **Create Role**
3. Enter role name and optional description
4. Select permissions from the permission tree
5. Save the role

### Custom Role Permissions

Custom roles can have any combination of permissions:
- **Specific permissions**: `deployment.create`, `gameservers.read`, `vps.update`
- **Wildcard permissions**: `deployment.*` (grants all deployment permissions)
- **Resource-specific**: Permissions can be scoped to specific resources via role bindings

### Managing Custom Roles

- **Update**: Modify permissions or name (requires `admin.roles.update` or `admin.roles.*`)
- **Delete**: Remove the role (requires `admin.roles.delete` or `admin.roles.*`)
- **System roles cannot be modified**: System roles are defined in code and cannot be changed

## Role Bindings

Role bindings assign roles to users, optionally scoped to specific resources. This allows fine-grained access control.

### Organization-Wide Bindings

A role binding without resource scoping grants permissions organization-wide:

```
User: john@example.com
Role: "Deployment Manager" (custom role with deployment.* permissions)
Scope: Organization-wide
Result: User can manage all deployments in the organization
```

### Resource-Scoped Bindings

Role bindings can be scoped to:
- **Specific Resources**: A specific deployment, VPS, or game server
- **Resource Types**: All resources of a type (e.g., all deployments)
- **Environments**: All deployments in specific environments

Examples:

```
User: jane@example.com
Role: "Production Manager" (custom role with deployment.* permissions)
Scope: Environment "production"
Result: User has additional deployment.* permissions for production deployments only.
Note: If the user also has a base role (e.g., Member) with deployment permissions,
those permissions remain organization-wide.
```

```
User: bob@example.com
Role: "Deployment Viewer" (custom role with deployment.read permissions)
Scope: Specific deployment "my-app-prod"
Result: User has additional deployment.read permission for this specific deployment.
Note: If the user also has a base role (e.g., Viewer) with deployment.read,
that permission remains organization-wide.
```

### Creating Role Bindings

1. Navigate to **Admin > Bindings** in your organization
2. Select a member and role
3. (Optional) Select resource type and specific resource
4. Click **Bind**

### Resource Types

Supported resource types for scoping:
- **Deployment**: Scope to specific deployments or environments
- **Environment**: Scope to specific environments (applies to deployments in those environments)
- **VPS**: Scope to specific VPS instances
- **Game Server**: Scope to specific game servers

## Permission Format

Permissions follow the format: `<resource>.<action>`

### Resource Types
- `deployment` - Deployments
- `gameservers` - Game servers
- `vps` - VPS instances
- `organization` - Organization settings
- `admin` - Admin operations (roles, bindings, quotas)
- `superadmin` - Superadmin operations (global management)

### Actions
- `read` - View/list resources
- `create` - Create new resources
- `update` - Modify existing resources
- `delete` - Delete resources
- `start` - Start resources
- `stop` - Stop resources
- `restart` - Restart resources
- `scale` - Scale resources
- `logs` - View logs
- `manage` - Full management (all actions)
- `*` - Wildcard (all actions for the resource)

### Examples
- `deployment.read` - View deployments
- `deployment.create` - Create deployments
- `deployment.*` - All deployment permissions
- `gameservers.manage` - Full game server management
- `admin.roles.read` - View roles
- `admin.bindings.create` - Create role bindings

## Wildcard Permissions

Wildcard permissions use `*` to grant all permissions for a resource type:

- `deployment.*` - All deployment permissions (read, create, update, delete, start, stop, restart, scale, logs, etc.)
- `gameservers.*` - All game server permissions
- `vps.*` - All VPS permissions
- `admin.*` - All admin permissions (roles, bindings, quotas)
- `organization.*` - All organization permissions
- `*` - All permissions (superadmin only)

Wildcards are useful for:
- System roles (Owner, Admin) that need broad access
- Custom roles that should have full access to a resource type
- Simplifying permission management

## How Permissions Are Checked

When a user performs an action, the system checks permissions in this order:

1. **Superadmin Check**: If user is a superadmin, grant access
2. **Direct Role Assignment**: Check the role assigned directly in `organization_members.role`
   - If system role: Check hardcoded permissions (always grants org-wide access if permission matches)
   - If custom role: Look up role in database and check permissions (always grants org-wide access if permission matches)
   - If the direct role has the permission, access is granted organization-wide and the check stops here
3. **Role Bindings**: Check all role bindings for the user (only if direct role doesn't have the permission)
   - Evaluate permissions from bound roles
   - Check resource scoping (organization-wide, resource-specific, environment-based)
   - Role bindings can add additional permissions but cannot remove or limit direct role permissions
4. **Permission Match**: Check if permission matches (exact match or wildcard match)
5. **Grant or Deny**: Allow or deny the action

**Important**: Direct role permissions are always organization-wide. Role bindings are checked only if the direct role doesn't grant the permission, and they can add scoped permissions but cannot restrict your base role permissions.

### Permission Matching

Permissions are matched using:
- **Exact match**: `deployment.create` matches `deployment.create`
- **Wildcard match**: `deployment.*` matches `deployment.create`, `deployment.read`, etc.
- **Prefix match**: `deployment.*` matches any permission starting with `deployment.`

## Adding New Permissions

### For Backend Developers

1. **Register the permission** in `apps/shared/pkg/auth/register_procedures.go`:
   ```go
   func RegisterYourServiceProcedures() {
       registry := GetPermissionRegistry()
       registry.RegisterProcedureWithFlags(
           "/obiente.cloud.yourservice.v1.YourService/YourMethod",
           "yourservice.action",  // Permission string
           "yourservice",          // Resource type
           "action",               // Action
           "Description",          // Human-readable description
           false,                  // Is public?
           false,                  // Is superadmin-only?
       )
   }
   ```

2. **Add to RegisterAllServices()** in the same file:
   ```go
   func RegisterAllServices() {
       // ... existing services
       RegisterYourServiceProcedures()
   }
   ```

3. **Use in service handlers**:
   ```go
   // Automatic via middleware (recommended)
   // Or manual check:
   pc := auth.NewPermissionChecker()
   if err := pc.CheckScopedPermission(ctx, orgID, auth.ScopedPermission{
       Permission:   "yourservice.action",
       ResourceType: "yourservice",
       ResourceID:   resourceID, // Optional, for resource-scoped checks
   }); err != nil {
       return nil, connect.NewError(connect.CodePermissionDenied, err)
   }
   ```

### Permission Naming Convention

- Use lowercase with dots: `resource.action`
- Resource type should match the service/resource name
- Actions should be verbs: `read`, `create`, `update`, `delete`, `start`, `stop`, etc.
- Use `*` for wildcard permissions: `resource.*`

## Permission Registry

The permission registry automatically:
- Maps RPC procedures to permissions
- Tracks public endpoints (no auth required)
- Maintains backward compatibility when procedures change
- Auto-discovers permissions from procedure paths

### Service Registration

All services register their procedures at startup via `auth.RegisterAllServices()`. This is called automatically in `auth-service/main.go`.

Each service has a registration function (e.g., `RegisterDeploymentServiceProcedures()`) that:
- Maps procedure paths to method names
- Specifies which procedures are public
- Defines explicit permissions (recommended) or auto-generates from service/method names

### Permission Mapping Rules (Auto-Discovery)

If a permission isn't explicitly registered, the system infers it from the procedure path:

**Service → Resource Type:**
- `DeploymentService` → `deployment`
- `VPSService` → `vps`
- `GameServerService` → `gameserver`
- `OrganizationService` → `organization`
- `AdminService` → `admin`
- `SuperadminService` → `superadmin`

**RPC → Action:**
- `Create*`, `Add*` → `create`
- `List*`, `Get*`, `Query*`, `Stream*` → `read`
- `Update*`, `Set*`, `Upsert*` → `update`
- `Delete*`, `Remove*` → `delete`
- `Start*` → `start`
- `Stop*` → `stop`
- `Restart*` → `restart`
- `Scale*` → `scale`
- `*Log*` → `logs`
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

## Manual Permission Checks

You can still manually check permissions in service handlers when you need custom logic:

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
        Permission:   "deployment.create",
        ResourceType: "deployment",
        ResourceID:   "", // Empty for org-wide check
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

1. **Use system roles for common access patterns**: Owner, Admin, Member, Viewer cover most use cases
2. **Create custom roles for specific needs**: Use custom roles when you need fine-grained permissions
3. **Use wildcard permissions sparingly**: Prefer specific permissions unless you truly need all actions
4. **Scope permissions appropriately**: Use resource scoping in role bindings to grant additional permissions for specific resources. Note that direct role permissions are always organization-wide and cannot be limited by role bindings.
5. **Test permissions**: Create test roles and verify permissions work as expected
6. **Document custom roles**: Add descriptions to custom roles explaining their purpose
7. **Register all procedures**: Explicitly register all procedures when adding new services
8. **Use middleware**: Prefer automatic permission checking via middleware over manual checks

## Troubleshooting

### Permission Denied Errors

If you're getting permission denied errors:

1. **Check your role**: Verify your role in the organization (Owner, Admin, Member, Viewer, or custom)
2. **Check role permissions**: If using a custom role, verify it has the required permissions
3. **Check role bindings**: Verify you have role bindings with the required permissions
4. **Check resource scoping**: If permissions are scoped to specific resources, ensure you're accessing the correct resource
5. **Check admin permissions**: Some actions require `admin.*` permissions (e.g., creating roles, bindings)

### Common Permission Issues

- **"I can't create deployments"**: You need `deployment.create` or `deployment.*` permission
- **"I can't see other users' resources"**: You need `resource.read` permission and may need org-wide access
- **"I can't manage roles"**: You need `admin.roles.*` or `admin.*` permission
- **"I can't assign roles"**: You need `admin.bindings.create` or `admin.bindings.*` permission

## API Reference

### GetMyPermissions

Get the current user's permissions for an organization:

```protobuf
rpc GetMyPermissions(GetMyPermissionsRequest) returns (GetMyPermissionsResponse);

message GetMyPermissionsRequest {
  string organization_id = 1;
}

message GetMyPermissionsResponse {
  repeated string permissions = 1;
}
```

Returns all permissions the user has in the organization, including:
- Permissions from system roles
- Permissions from custom roles (assigned directly or via bindings)
- Wildcard permissions (e.g., `deployment.*`)
- Superadmin wildcard (`*`) for superadmins

## Future Improvements

- Permission inheritance and hierarchies
- Permission versioning and migration tools
- Resource-level permission scoping improvements
- Permission audit logging
- Permission templates for common use cases
