package auth

import (
	"fmt"
	"strings"
	"sync"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

// PermissionMapping represents a mapping from procedure to permission
type PermissionMapping struct {
	Procedure      string // Full procedure path, e.g., "/obiente.cloud.deployments.v1.DeploymentService/CreateDeployment"
	Permission     string // Permission string, e.g., "deployments.create"
	ResourceType   string // Resource type, e.g., "deployment"
	Action         string // Action, e.g., "create"
	Description    string // Human-readable description
	Public         bool   // Whether this is a public endpoint (no auth required)
	SuperadminOnly bool   // Whether this permission is superadmin-only (not assignable to org roles)
}

// PermissionRegistry maintains a registry of all RPC procedures and their permission mappings
type PermissionRegistry struct {
	mu          sync.RWMutex
	mappings    map[string]*PermissionMapping // procedure -> mapping
	byPerm      map[string][]string           // permission -> []procedures (for backward compatibility)
	public      map[string]bool               // procedure -> is public
	initialized bool
}

var globalRegistry = &PermissionRegistry{
	mappings: make(map[string]*PermissionMapping),
	byPerm:   make(map[string][]string),
	public:   make(map[string]bool),
}

// GetRegistry returns the global permission registry
func GetPermissionRegistry() *PermissionRegistry {
	return globalRegistry
}

// RegisterProcedure registers a procedure with its permission mapping
func (r *PermissionRegistry) RegisterProcedure(procedure, permission, resourceType, action, description string, public bool) {
	r.RegisterProcedureWithFlags(procedure, permission, resourceType, action, description, public, false)
}

// RegisterProcedureWithFlags registers a procedure with additional flags
func (r *PermissionRegistry) RegisterProcedureWithFlags(procedure, permission, resourceType, action, description string, public, superadminOnly bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	mapping := &PermissionMapping{
		Procedure:      procedure,
		Permission:     permission,
		ResourceType:   resourceType,
		Action:         action,
		Description:    description,
		Public:         public,
		SuperadminOnly: superadminOnly,
	}

	r.mappings[procedure] = mapping
	r.public[procedure] = public

	// Track by permission for backward compatibility
	if permission != "" {
		r.byPerm[permission] = append(r.byPerm[permission], procedure)
	}
}

// IsSuperadminOnly checks if a permission is superadmin-only
func (r *PermissionRegistry) IsSuperadminOnly(permission string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check all procedures that map to this permission
	if procedures, ok := r.byPerm[permission]; ok {
		for _, proc := range procedures {
			if mapping, ok := r.mappings[proc]; ok && mapping.SuperadminOnly {
				return true
			}
		}
	}

	// Also check by permission prefix patterns
	// All admin.* permissions are superadmin-only
	if strings.HasPrefix(permission, "admin.") {
		return true
	}
	// organization.admin.* permissions are superadmin-only
	if strings.HasPrefix(permission, "organization.admin.") {
		return true
	}
	// All superadmin.* permissions are superadmin-only
	if strings.HasPrefix(permission, "superadmin.") {
		return true
	}

	return false
}

// GetPermission returns the permission for a procedure
func (r *PermissionRegistry) GetPermission(procedure string) (string, string, string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if mapping, ok := r.mappings[procedure]; ok {
		return mapping.Permission, mapping.ResourceType, mapping.Action, mapping.Public
	}

	// Fallback: try to infer from procedure path
	perm, rt, action := inferPermissionFromProcedure(procedure)
	return perm, rt, action, false
}

// IsPublic checks if a procedure is public (no auth required)
func (r *PermissionRegistry) IsPublic(procedure string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if public, ok := r.public[procedure]; ok {
		return public
	}

	// Check hardcoded public procedures
	return isPublicProcedure(procedure)
}

// GetAllPermissions returns all unique permissions in the registry
// Excludes public procedures and user-based services (support, notifications)
// If excludeSuperadmin is true, also excludes superadmin-only permissions
func (r *PermissionRegistry) GetAllPermissions(excludeSuperadmin ...bool) []*PermissionDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	permMap := make(map[string]*PermissionDefinition)

	shouldExcludeSuperadmin := len(excludeSuperadmin) > 0 && excludeSuperadmin[0]

	// Add all registered permissions
	for _, mapping := range r.mappings {
		// Skip public procedures (they shouldn't appear in permission catalog)
		if mapping.Public {
			continue
		}
		// Skip user-based services (support, notifications)
		if mapping.ResourceType == "support" || mapping.ResourceType == "notification" {
			continue
		}
		// Skip superadmin-only permissions if requested
		if shouldExcludeSuperadmin && (mapping.SuperadminOnly || r.IsSuperadminOnly(mapping.Permission)) {
			continue
		}
		if mapping.Permission != "" {
			// When multiple procedures map to the same permission, prefer the most descriptive description
			if existing, ok := permMap[mapping.Permission]; !ok {
				permMap[mapping.Permission] = &PermissionDefinition{
					Permission:   mapping.Permission,
					ResourceType: mapping.ResourceType,
					Description:  mapping.Description,
				}
			} else {
				// Prefer longer, more descriptive descriptions
				if len(mapping.Description) > len(existing.Description) {
					existing.Description = mapping.Description
				}
			}
		}
	}

	// Add manual permissions (for backward compatibility)
	for perm, desc := range ScopeDescriptions {
		// Skip user-based permissions
		parts := strings.Split(perm, ".")
		resourceType := "other"
		if len(parts) > 0 {
			resourceType = parts[0]
		}
		if resourceType == "support" || resourceType == "notification" {
			continue
		}
		// Skip superadmin-only permissions if requested
		if shouldExcludeSuperadmin && r.IsSuperadminOnly(perm) {
			continue
		}

		if _, exists := permMap[perm]; !exists {
			permMap[perm] = &PermissionDefinition{
				Permission:   perm,
				ResourceType: resourceType,
				Description:  desc,
			}
		}
	}

	// Convert to slice
	result := make([]*PermissionDefinition, 0, len(permMap))
	for _, perm := range permMap {
		result = append(result, perm)
	}

	return result
}

// PermissionDefinition represents a permission for listing
type PermissionDefinition struct {
	Permission   string
	ResourceType string
	Description  string
}

// AutoDiscoverProcedures discovers procedures from generated Connect packages using reflection
// This is called at startup to populate the registry
func (r *PermissionRegistry) AutoDiscoverProcedures() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.initialized {
		return nil // Already initialized
	}

	logger.Info("[PermissionRegistry] Auto-discovering procedures from generated Connect code...")

	// Discover procedures from all Connect packages
	// We'll use a package-level init approach or explicit registration
	// For now, we'll discover from common patterns

	// Register known public procedures
	r.public["/obiente.cloud.auth.v1.AuthService/Login"] = true
	r.public["/obiente.cloud.auth.v1.AuthService/GetPublicConfig"] = true
	r.public["/obiente.cloud.superadmin.v1.SuperadminService/GetPricing"] = true

	// The actual procedure discovery will happen via explicit registration
	// when services are initialized, or we can use reflection to find all
	// Procedure constants in the connect packages

	r.initialized = true
	logger.Info("[PermissionRegistry] Registry initialized with %d procedures", len(r.mappings))
	return nil
}

// RegisterServiceProcedures registers all procedures for a service by reflecting on procedure constants
// This should be called for each service at startup
func (r *PermissionRegistry) RegisterServiceProcedures(servicePackage interface{}) error {
	// Use reflection to find all Procedure constants
	// This is a simplified version - in practice, you'd reflect on the package
	// For now, we'll use explicit registration per service

	return nil
}

// inferPermissionFromProcedure infers permission from procedure path when not explicitly registered
func inferPermissionFromProcedure(procedure string) (permission, resourceType, action string) {
	// Parse procedure: /package.Service/Method
	parts := strings.Split(strings.TrimPrefix(procedure, "/"), "/")
	if len(parts) != 2 {
		return "", "", ""
	}

	servicePath := parts[0]
	method := parts[1]

	// Extract service name
	serviceParts := strings.Split(servicePath, ".")
	serviceName := serviceParts[len(serviceParts)-1]

	// Map service to resource type
	resourceType = serviceToResourceType(serviceName)
	if resourceType == "" {
		return "", "", ""
	}

	// Map method to action
	action = methodToAction(method)
	if action == "" {
		return "", "", ""
	}

	permission = fmt.Sprintf("%s.%s", resourceType, action)
	return permission, resourceType, action
}

// RegisterProcedureFromSpec registers a procedure from a Connect spec
// This can be called automatically when handlers are created
func RegisterProcedureFromSpec(procedure, serviceName, methodName string, public bool) {
	resourceType := serviceToResourceType(serviceName)
	action := methodToAction(methodName)

	if resourceType == "" || action == "" {
		return // Skip if we can't determine
	}

	permission := fmt.Sprintf("%s.%s", resourceType, action)
	description := generatePermissionDescription(methodName, resourceType, action)

	globalRegistry.RegisterProcedure(procedure, permission, resourceType, action, description, public)
}

// generatePermissionDescription generates a human-readable description
func generatePermissionDescription(methodName, resourceType, action string) string {
	// Convert method name to readable format
	desc := methodName
	desc = strings.ReplaceAll(desc, "Deployment", "")
	desc = strings.ReplaceAll(desc, "VPS", "")
	desc = strings.ReplaceAll(desc, "GameServer", "")

	// Add action context
	actionDesc := action
	switch action {
	case "create":
		actionDesc = "Create"
	case "read":
		actionDesc = "View"
	case "update":
		actionDesc = "Update"
	case "delete":
		actionDesc = "Delete"
	case "start":
		actionDesc = "Start"
	case "stop":
		actionDesc = "Stop"
	case "restart":
		actionDesc = "Restart"
	case "scale":
		actionDesc = "Scale"
	case "logs":
		actionDesc = "View logs"
	default:
		if len(action) > 0 {
			actionDesc = strings.ToUpper(action[:1]) + action[1:]
		}
	}

	return fmt.Sprintf("%s %s", actionDesc, resourceType)
}

// EnsureBackwardCompatibility ensures existing permissions still work even if procedures change
func (r *PermissionRegistry) EnsureBackwardCompatibility(oldProcedure, newProcedure, permission string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// If we have a mapping for the old procedure, copy it to the new one
	if oldMapping, ok := r.mappings[oldProcedure]; ok {
		newMapping := *oldMapping
		newMapping.Procedure = newProcedure
		r.mappings[newProcedure] = &newMapping
	}

	// Also ensure the permission still maps to both procedures
	if permission != "" {
		procedures := r.byPerm[permission]
		found := false
		for _, proc := range procedures {
			if proc == newProcedure {
				found = true
				break
			}
		}
		if !found {
			r.byPerm[permission] = append(r.byPerm[permission], newProcedure)
		}
	}
}
