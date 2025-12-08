package auth

// Permission constants - single source of truth for all permission strings
// All permissions are automatically normalized, but these constants ensure consistency

// Resource prefixes (canonical forms)
const (
	ResourcePrefixDeployment  = "deployment"  // singular
	ResourcePrefixGameServers = "gameservers" // plural
	ResourcePrefixVPS         = "vps"         // singular
	ResourcePrefixOrganization = "organization"
	ResourcePrefixAdmin        = "admin"
	ResourcePrefixSuperadmin   = "superadmin"
)

// Common actions
const (
	ActionRead   = "read"
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionStart  = "start"
	ActionStop   = "stop"
	ActionRestart = "restart"
	ActionScale  = "scale"
	ActionLogs   = "logs"
	ActionManage = "manage"
	ActionDeploy = "deploy"
)

// Deployment permissions
const (
	PermissionDeploymentRead   = ResourcePrefixDeployment + "." + ActionRead
	PermissionDeploymentCreate = ResourcePrefixDeployment + "." + ActionCreate
	PermissionDeploymentUpdate = ResourcePrefixDeployment + "." + ActionUpdate
	PermissionDeploymentDelete = ResourcePrefixDeployment + "." + ActionDelete
	PermissionDeploymentStart  = ResourcePrefixDeployment + "." + ActionStart
	PermissionDeploymentStop   = ResourcePrefixDeployment + "." + ActionStop
	PermissionDeploymentRestart = ResourcePrefixDeployment + "." + ActionRestart
	PermissionDeploymentScale  = ResourcePrefixDeployment + "." + ActionScale
	PermissionDeploymentLogs   = ResourcePrefixDeployment + "." + ActionLogs
	PermissionDeploymentDeploy = ResourcePrefixDeployment + "." + ActionDeploy
	PermissionDeploymentManage = ResourcePrefixDeployment + "." + ActionManage
	PermissionDeploymentAll    = ResourcePrefixDeployment + ".*"
)

// Game server permissions
const (
	PermissionGameServersRead   = ResourcePrefixGameServers + "." + ActionRead
	PermissionGameServersCreate = ResourcePrefixGameServers + "." + ActionCreate
	PermissionGameServersUpdate = ResourcePrefixGameServers + "." + ActionUpdate
	PermissionGameServersDelete = ResourcePrefixGameServers + "." + ActionDelete
	PermissionGameServersStart  = ResourcePrefixGameServers + "." + ActionStart
	PermissionGameServersStop   = ResourcePrefixGameServers + "." + ActionStop
	PermissionGameServersRestart = ResourcePrefixGameServers + "." + ActionRestart
	PermissionGameServersManage = ResourcePrefixGameServers + "." + ActionManage
	PermissionGameServersAll    = ResourcePrefixGameServers + ".*"
)

// VPS permissions
const (
	PermissionVPSRead   = ResourcePrefixVPS + "." + ActionRead
	PermissionVPSCreate = ResourcePrefixVPS + "." + ActionCreate
	PermissionVPSUpdate = ResourcePrefixVPS + "." + ActionUpdate
	PermissionVPSDelete = ResourcePrefixVPS + "." + ActionDelete
	PermissionVPSStart  = ResourcePrefixVPS + "." + ActionStart
	PermissionVPSStop   = ResourcePrefixVPS + "." + ActionStop
	PermissionVPSReboot = ResourcePrefixVPS + ".reboot"
	PermissionVPSWrite  = ResourcePrefixVPS + ".write" // Legacy alias for update
	PermissionVPSManage = ResourcePrefixVPS + "." + ActionManage
	PermissionVPSAll    = ResourcePrefixVPS + ".*"
)

// Organization permissions
const (
	PermissionOrganizationRead   = ResourcePrefixOrganization + "." + ActionRead
	PermissionOrganizationUpdate = ResourcePrefixOrganization + "." + ActionUpdate
	PermissionOrganizationDelete = ResourcePrefixOrganization + "." + ActionDelete
	PermissionOrganizationMembersRead   = ResourcePrefixOrganization + ".members." + ActionRead
	PermissionOrganizationMembersUpdate = ResourcePrefixOrganization + ".members." + ActionUpdate
	PermissionOrganizationMembersAll    = ResourcePrefixOrganization + ".members.*"
	PermissionOrganizationAll           = ResourcePrefixOrganization + ".*"
)

// Admin permissions
const (
	PermissionAdminPermissionsRead = ResourcePrefixAdmin + ".permissions." + ActionRead
	PermissionAdminRolesRead       = ResourcePrefixAdmin + ".roles." + ActionRead
	PermissionAdminRolesCreate      = ResourcePrefixAdmin + ".roles." + ActionCreate
	PermissionAdminRolesUpdate      = ResourcePrefixAdmin + ".roles." + ActionUpdate
	PermissionAdminRolesDelete      = ResourcePrefixAdmin + ".roles." + ActionDelete
	PermissionAdminBindingsRead     = ResourcePrefixAdmin + ".bindings." + ActionRead
	PermissionAdminBindingsCreate   = ResourcePrefixAdmin + ".bindings." + ActionCreate
	PermissionAdminBindingsDelete    = ResourcePrefixAdmin + ".bindings." + ActionDelete
	PermissionAdminQuotasUpdate     = ResourcePrefixAdmin + ".quotas." + ActionUpdate
	PermissionAdminAll              = ResourcePrefixAdmin + ".*"
)

// Helper function to build permission strings dynamically (for custom permissions)
func BuildPermission(resourcePrefix, action string) string {
	return resourcePrefix + "." + action
}

