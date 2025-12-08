package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/obiente/cloud/apps/shared/pkg/logger"

	authv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/auth/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// PermissionMiddleware creates a Connect interceptor that automatically checks permissions
// based on the RPC being called. It uses the permission registry to map RPCs to permissions.
func PermissionMiddleware() connect.UnaryInterceptorFunc {
	registry := GetPermissionRegistry()

	// Ensure registry is initialized
	_ = registry.AutoDiscoverProcedures()

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := req.Spec().Procedure

			// Skip permission check for public endpoints
			if registry.IsPublic(procedure) {
				return next(ctx, req)
			}

			// Get user from context
			user, err := GetUserFromContext(ctx)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
			}

			// Superadmins bypass all permission checks
			if HasRole(user, RoleSuperAdmin) {
				return next(ctx, req)
			}

			// Get permission from registry
			permission, resourceType, action, isPublic := registry.GetPermission(procedure)
			if isPublic {
				return next(ctx, req)
			}

			if permission == "" {
				// No permission mapping found - infer from procedure
				permission, resourceType, action = inferPermissionFromProcedure(procedure)
				if permission == "" {
					// Still no permission - allow for now (can be made stricter later)
					logger.Debug("[Permission] No permission mapping for procedure: %s", procedure)
					return next(ctx, req)
				}
				// Auto-register the inferred permission for future use
				description := generatePermissionDescription(procedure, resourceType, action)
				registry.RegisterProcedure(procedure, permission, resourceType, action, description, false)
			}

			// Skip permission check for support service - it handles access control internally
			// (users see their own tickets, superadmins see all)
			if strings.Contains(procedure, "SupportService") {
				return next(ctx, req)
			}

			// Skip permission check for notification service - it handles access control internally
			// (users can only access their own notifications, admin operations require superadmin)
			if strings.Contains(procedure, "NotificationService") {
				return next(ctx, req)
			}

			// Extract organization ID from request
			orgID := extractOrgID(req)
			resourceID := extractResourceID(req, procedure, resourceType)

			// Check if user has the required permission
			if orgID != "" {
				// Organization-scoped permission check
				if err := checkOrgPermission(ctx, user, orgID, permission, resourceID); err != nil {
					logger.Debug("[Permission] Permission denied for %s: %v", procedure, err)
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
			} else {
				// Global permission check (for services that don't require org context)
				if err := checkGlobalPermission(ctx, user, permission); err != nil {
					logger.Debug("[Permission] Permission denied for %s: %v", procedure, err)
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
			}

			return next(ctx, req)
		}
	}
}

// isPublicProcedure checks if a procedure is public (no auth/permission required)
func isPublicProcedure(procedure string) bool {
	publicProcedures := []string{
		"/obiente.cloud.auth.v1.AuthService/Login",
		"/obiente.cloud.auth.v1.AuthService/GetPublicConfig",
		"/obiente.cloud.superadmin.v1.SuperadminService/GetPricing",
	}

	for _, publicProc := range publicProcedures {
		if procedure == publicProc {
			return true
		}
	}

	return false
}

// procedureToPermission maps a procedure path to a permission string
// Returns: permission, organizationID, resourceID
func procedureToPermission(procedure string, req connect.AnyRequest) (string, string, string) {
	// Parse procedure: /package.Service/Method
	parts := strings.Split(strings.TrimPrefix(procedure, "/"), "/")
	if len(parts) != 2 {
		return "", "", ""
	}

	servicePath := parts[0]
	method := parts[1]

	// Extract service name from path (e.g., "obiente.cloud.deployments.v1.DeploymentService" -> "DeploymentService")
	serviceParts := strings.Split(servicePath, ".")
	serviceName := serviceParts[len(serviceParts)-1]

	// Map service name to resource type
	resourceType := serviceToResourceType(serviceName)
	if resourceType == "" {
		return "", "", ""
	}

	// Map method name to action
	action := methodToAction(method)
	if action == "" {
		return "", "", ""
	}

	// Build permission string
	permission := fmt.Sprintf("%s.%s", resourceType, action)

	// Try to extract organization ID and resource ID from request
	orgID := extractOrgID(req)
	resourceID := extractResourceID(req, method, resourceType)

	return permission, orgID, resourceID
}

// serviceToResourceType maps service names to resource types
func serviceToResourceType(serviceName string) string {
	mapping := map[string]string{
		"DeploymentService":   "deployment",
		"VPSService":          "vps",
		"GameServerService":   "gameserver",
		"BillingService":      "billing",
		"OrganizationService": "organization",
		"SupportService":      "support",
		"NotificationService": "notification",
		"AdminService":        "admin",
		"SuperadminService":   "superadmin",
		"AuditService":        "audit",
		"VPSConfigService":    "vps",
	}

	if rt, ok := mapping[serviceName]; ok {
		return rt
	}

	// Try to infer from service name
	if strings.HasSuffix(serviceName, "Service") {
		base := strings.TrimSuffix(serviceName, "Service")
		return strings.ToLower(base)
	}

	return ""
}

// methodToAction maps RPC method names to permission actions
func methodToAction(method string) string {
	// Check prefixes
	actionMap := map[string]string{
		"Create":   "create",
		"List":     "read",
		"Get":      "read",
		"Update":   "update",
		"Delete":   "delete",
		"Start":    "start",
		"Stop":     "stop",
		"Restart":  "restart",
		"Scale":    "scale",
		"Trigger":  "trigger",
		"Stream":   "read",
		"Attach":   "attach",
		"Detach":   "detach",
		"Set":      "update",
		"Query":    "read",
		"Revert":   "revert",
		"Upload":   "upload",
		"Download": "download",
		"Validate": "validate",
		"Upsert":   "update",
		"Revoke":   "revoke",
		"Invite":   "invite",
		"Remove":   "delete",
		"Leave":    "leave",
		"Add":      "create",
	}

	for prefix, action := range actionMap {
		if strings.HasPrefix(method, prefix) {
			return action
		}
	}

	// Special cases
	if strings.Contains(method, "Log") {
		return "logs"
	}
	if strings.Contains(method, "Metric") {
		return "read"
	}
	if strings.Contains(method, "Usage") {
		return "read"
	}

	// Default to "manage" for unknown methods
	return "manage"
}

// extractOrgID tries to extract organization ID from request
func extractOrgID(req connect.AnyRequest) string {
	msg := req.Any()
	if msg == nil {
		return ""
	}

	// Try to extract using reflection via protojson
	// This is a common pattern - most requests have organizationId or organization_id
	if protoMsg, ok := msg.(proto.Message); ok {
		jsonBytes, err := protojson.Marshal(protoMsg)
		if err == nil {
			var jsonData map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &jsonData); err == nil {
				// Try common field names
				if orgID, ok := jsonData["organizationId"].(string); ok && orgID != "" {
					return orgID
				}
				if orgID, ok := jsonData["organization_id"].(string); ok && orgID != "" {
					return orgID
				}
			}
		}
	}

	return ""
}

// extractResourceID tries to extract resource ID from request
func extractResourceID(req connect.AnyRequest, procedure, resourceType string) string {
	msg := req.Any()
	if msg == nil {
		return ""
	}

	// Try common resource ID field names based on resource type
	if protoMsg, ok := msg.(proto.Message); ok {
		jsonBytes, err := protojson.Marshal(protoMsg)
		if err == nil {
			var jsonData map[string]interface{}
			if err := json.Unmarshal(jsonBytes, &jsonData); err == nil {
				// Try common field names for resource IDs
				fieldNames := []string{
					resourceType + "Id",
					resourceType + "_id",
					strings.TrimSuffix(resourceType, "s") + "Id", // deployments -> deploymentId
					strings.TrimSuffix(resourceType, "s") + "_id",
					"id",
				}

				for _, fieldName := range fieldNames {
					if id, ok := jsonData[fieldName].(string); ok && id != "" {
						return id
					}
				}
			}
		}
	}

	return ""
}

// checkOrgPermission checks if user has permission in an organization
// Uses the unified CheckScopedPermission for consistency
func checkOrgPermission(ctx context.Context, user *authv1.User, orgID, permission, resourceID string) error {
	// Normalize permission and extract resource type
	normalizedPerm := normalizePermission("", permission)
	
	// Extract resource type from normalized permission using helper
	resourceType := resourceToResourceType(normalizedPerm)

	// Use unified permission checking
	pc := NewPermissionChecker()
	sp := ScopedPermission{
		Permission:   normalizedPerm,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}

	return pc.CheckScopedPermission(ctx, orgID, sp)
}

// checkGlobalPermission checks global permissions (for services without org context)
// Uses the shared HasSuperadminPermission helper for consistency
func checkGlobalPermission(ctx context.Context, user *authv1.User, permission string) error {
	// Normalize permission first
	normalizedPerm := normalizePermission("", permission)
	
	if HasSuperadminPermission(ctx, user, normalizedPerm) {
		return nil
	}
	
	return fmt.Errorf("permission denied: %s", normalizedPerm)
}

// resourceToResourceType extracts resource type from permission string
// Maps permission prefixes back to resource types for scoping
func resourceToResourceType(permission string) string {
	parts := strings.Split(permission, ".")
	if len(parts) == 0 {
		return ""
	}
	
	prefix := parts[0]
	
	// Map permission prefix to resource type
	switch prefix {
	case ResourcePrefixDeployment:
		return "deployment"
	case ResourcePrefixGameServers:
		return "gameserver" // Use singular for resource type
	case ResourcePrefixVPS:
		return "vps"
	case ResourcePrefixOrganization:
		return "organization"
	case ResourcePrefixAdmin:
		return "admin"
	case ResourcePrefixSuperadmin:
		return "superadmin"
	default:
		return prefix // Use as-is if unknown
	}
}
