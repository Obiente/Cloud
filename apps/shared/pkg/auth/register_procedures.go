package auth

import (
	"fmt"
	"strings"
)

// AutoRegisterProceduresFromPackage registers all procedures from a Connect-generated package
// This uses reflection to find all Procedure constants and register them
// Note: This is a placeholder for future reflection-based auto-discovery
func AutoRegisterProceduresFromPackage(pkg interface{}, serviceName string, publicProcedures []string) {
	_ = pkg
	_ = serviceName
	_ = publicProcedures
	// For now, we use explicit registration via RegisterServiceProcedures
	// Future: Use reflection to discover Procedure constants from Connect packages
}

// RegisterServiceProcedures explicitly registers procedures for a service
// This should be called at service startup
func RegisterServiceProcedures(serviceName string, procedures map[string]string, publicProcedures []string) {
	registry := GetPermissionRegistry()
	publicMap := make(map[string]bool)
	for _, proc := range publicProcedures {
		publicMap[proc] = true
	}

	resourceType := serviceToResourceType(serviceName)
	if resourceType == "" {
		return
	}

	for procedure, methodName := range procedures {
		isPublic := publicMap[procedure]
		action := methodToAction(methodName)
		if action == "" {
			continue
		}

		permission := fmt.Sprintf("%s.%s", resourceType, action)
		description := generatePermissionDescription(methodName, resourceType, action)

		registry.RegisterProcedure(procedure, permission, resourceType, action, description, isPublic)
	}
}

// RegisterDeploymentServiceProcedures registers all DeploymentService procedures
// Note: Deployment operations are manually registered with semantic permission names
// for better UX (e.g., "deployment.logs" instead of "deployment.read" for log operations)
func RegisterDeploymentServiceProcedures() {
	registry := GetPermissionRegistry()
	public := []string{}

	// Register with explicit, semantic permission names
	deploymentProcedures := []struct {
		procedure    string
		permission   string
		resourceType string
		action       string
		description  string
	}{
		// Basic CRUD
		{"/obiente.cloud.deployments.v1.DeploymentService/ListDeployments", "deployment.read", "deployment", "read", "View deployments"},
		{"/obiente.cloud.deployments.v1.DeploymentService/CreateDeployment", "deployment.create", "deployment", "create", "Create deployment"},
		{"/obiente.cloud.deployments.v1.DeploymentService/GetDeployment", "deployment.read", "deployment", "read", "View deployment details"},
		{"/obiente.cloud.deployments.v1.DeploymentService/UpdateDeployment", "deployment.update", "deployment", "update", "Update deployment configuration"},
		{"/obiente.cloud.deployments.v1.DeploymentService/DeleteDeployment", "deployment.delete", "deployment", "delete", "Delete deployment"},

		// Lifecycle operations
		{"/obiente.cloud.deployments.v1.DeploymentService/TriggerDeployment", "deployment.update", "deployment", "update", "Trigger deployment (redeploy)"},
		{"/obiente.cloud.deployments.v1.DeploymentService/StartDeployment", "deployment.start", "deployment", "start", "Start deployment"},
		{"/obiente.cloud.deployments.v1.DeploymentService/StopDeployment", "deployment.stop", "deployment", "stop", "Stop deployment"},
		{"/obiente.cloud.deployments.v1.DeploymentService/RestartDeployment", "deployment.restart", "deployment", "restart", "Restart deployment"},
		{"/obiente.cloud.deployments.v1.DeploymentService/ScaleDeployment", "deployment.scale", "deployment", "scale", "Scale deployment"},

		// Logs and monitoring
		{"/obiente.cloud.deployments.v1.DeploymentService/GetDeploymentLogs", "deployment.logs", "deployment", "logs", "View deployment logs"},
		{"/obiente.cloud.deployments.v1.DeploymentService/StreamDeploymentLogs", "deployment.logs", "deployment", "logs", "Stream deployment logs"},
		{"/obiente.cloud.deployments.v1.DeploymentService/StreamBuildLogs", "deployment.logs", "deployment", "logs", "Stream build logs"},
		{"/obiente.cloud.deployments.v1.DeploymentService/GetDeploymentMetrics", "deployment.read", "deployment", "read", "View deployment metrics"},
		{"/obiente.cloud.deployments.v1.DeploymentService/StreamDeploymentMetrics", "deployment.read", "deployment", "read", "Stream deployment metrics"},
		{"/obiente.cloud.deployments.v1.DeploymentService/GetDeploymentUsage", "deployment.read", "deployment", "read", "View deployment usage"},
		{"/obiente.cloud.deployments.v1.DeploymentService/StreamDeploymentStatus", "deployment.read", "deployment", "read", "Stream deployment status"},

		// Environment variables
		{"/obiente.cloud.deployments.v1.DeploymentService/GetDeploymentEnvVars", "deployment.read", "deployment", "read", "View deployment environment variables"},
		{"/obiente.cloud.deployments.v1.DeploymentService/UpdateDeploymentEnvVars", "deployment.update", "deployment", "update", "Update deployment environment variables"},

		// Compose files
		{"/obiente.cloud.deployments.v1.DeploymentService/GetDeploymentCompose", "deployment.read", "deployment", "read", "View deployment compose file"},
		{"/obiente.cloud.deployments.v1.DeploymentService/ValidateDeploymentCompose", "deployment.read", "deployment", "read", "Validate deployment compose file"},
		{"/obiente.cloud.deployments.v1.DeploymentService/UpdateDeploymentCompose", "deployment.update", "deployment", "update", "Update deployment compose file"},

		// GitHub integration
		{"/obiente.cloud.deployments.v1.DeploymentService/ListGitHubRepos", "deployment.read", "deployment", "read", "View GitHub repositories"},
		{"/obiente.cloud.deployments.v1.DeploymentService/GetGitHubBranches", "deployment.read", "deployment", "read", "View GitHub branches"},
		{"/obiente.cloud.deployments.v1.DeploymentService/GetGitHubFile", "deployment.read", "deployment", "read", "View GitHub file"},
		{"/obiente.cloud.deployments.v1.DeploymentService/ListAvailableGitHubIntegrations", "deployment.read", "deployment", "read", "View available GitHub integrations"},

		// Builds
		{"/obiente.cloud.deployments.v1.DeploymentService/ListBuilds", "deployment.read", "deployment", "read", "View builds"},
		{"/obiente.cloud.deployments.v1.DeploymentService/GetBuild", "deployment.read", "deployment", "read", "View build details"},
		{"/obiente.cloud.deployments.v1.DeploymentService/GetBuildLogs", "deployment.logs", "deployment", "logs", "View build logs"},
		{"/obiente.cloud.deployments.v1.DeploymentService/RevertToBuild", "deployment.update", "deployment", "update", "Revert to previous build"},
		{"/obiente.cloud.deployments.v1.DeploymentService/DeleteBuild", "deployment.delete", "deployment", "delete", "Delete build"},

		// Terminal and file operations
		{"/obiente.cloud.deployments.v1.DeploymentService/StreamTerminal", "deployment.update", "deployment", "update", "Access deployment terminal"},
		{"/obiente.cloud.deployments.v1.DeploymentService/StreamTerminalOutput", "deployment.read", "deployment", "read", "View terminal output"},
		{"/obiente.cloud.deployments.v1.DeploymentService/SendTerminalInput", "deployment.update", "deployment", "update", "Send terminal input"},
		{"/obiente.cloud.deployments.v1.DeploymentService/ListContainerFiles", "deployment.read", "deployment", "read", "View container files"},
		{"/obiente.cloud.deployments.v1.DeploymentService/GetContainerFile", "deployment.read", "deployment", "read", "View container file"},
		{"/obiente.cloud.deployments.v1.DeploymentService/UploadContainerFiles", "deployment.update", "deployment", "update", "Upload container files"},
		{"/obiente.cloud.deployments.v1.DeploymentService/DownloadContainerFiles", "deployment.read", "deployment", "read", "Download container files"},
		{"/obiente.cloud.deployments.v1.DeploymentService/DeleteContainerFile", "deployment.update", "deployment", "update", "Delete container file"},
	}

	for _, proc := range deploymentProcedures {
		isPublic := false
		for _, pub := range public {
			if pub == proc.procedure {
				isPublic = true
				break
			}
		}
		registry.RegisterProcedure(proc.procedure, proc.permission, proc.resourceType, proc.action, proc.description, isPublic)
	}
}

// RegisterVPSServiceProcedures registers all VPSService procedures
func RegisterVPSServiceProcedures() {
	// ListVPSSizes and ListVPSRegions are catalog/pricing endpoints - no auth required
	public := []string{
		"/obiente.cloud.vps.v1.VPSService/ListVPSSizes",
		"/obiente.cloud.vps.v1.VPSService/ListVPSRegions",
	}
	procedures := map[string]string{
		"/obiente.cloud.vps.v1.VPSService/ListVPS":            "ListVPS",
		"/obiente.cloud.vps.v1.VPSService/CreateVPS":          "CreateVPS",
		"/obiente.cloud.vps.v1.VPSService/GetVPS":             "GetVPS",
		"/obiente.cloud.vps.v1.VPSService/UpdateVPS":          "UpdateVPS",
		"/obiente.cloud.vps.v1.VPSService/DeleteVPS":          "DeleteVPS",
		"/obiente.cloud.vps.v1.VPSService/StartVPS":           "StartVPS",
		"/obiente.cloud.vps.v1.VPSService/StopVPS":            "StopVPS",
		"/obiente.cloud.vps.v1.VPSService/RebootVPS":          "RebootVPS",
		"/obiente.cloud.vps.v1.VPSService/StreamVPSStatus":    "StreamVPSStatus",
		"/obiente.cloud.vps.v1.VPSService/GetVPSMetrics":      "GetVPSMetrics",
		"/obiente.cloud.vps.v1.VPSService/StreamVPSMetrics":   "StreamVPSMetrics",
		"/obiente.cloud.vps.v1.VPSService/GetVPSUsage":        "GetVPSUsage",
		"/obiente.cloud.vps.v1.VPSService/ListVPSSizes":       "ListVPSSizes",
		"/obiente.cloud.vps.v1.VPSService/ListVPSRegions":     "ListVPSRegions",
		"/obiente.cloud.vps.v1.VPSService/GetVPSProxyInfo":    "GetVPSProxyInfo",
		"/obiente.cloud.vps.v1.VPSService/ListFirewallRules":  "ListFirewallRules",
		"/obiente.cloud.vps.v1.VPSService/GetFirewallRule":    "GetFirewallRule",
		"/obiente.cloud.vps.v1.VPSService/CreateFirewallRule": "CreateFirewallRule",
		"/obiente.cloud.vps.v1.VPSService/UpdateFirewallRule": "UpdateFirewallRule",
		"/obiente.cloud.vps.v1.VPSService/DeleteFirewallRule": "DeleteFirewallRule",
	}

	RegisterServiceProcedures("VPSService", procedures, public)
}

// RegisterGameServerServiceProcedures registers all GameServerService procedures
func RegisterGameServerServiceProcedures() {
	public := []string{}
	procedures := map[string]string{
		"/obiente.cloud.gameservers.v1.GameServerService/ListGameServers":         "ListGameServers",
		"/obiente.cloud.gameservers.v1.GameServerService/CreateGameServer":        "CreateGameServer",
		"/obiente.cloud.gameservers.v1.GameServerService/GetGameServer":           "GetGameServer",
		"/obiente.cloud.gameservers.v1.GameServerService/UpdateGameServer":        "UpdateGameServer",
		"/obiente.cloud.gameservers.v1.GameServerService/DeleteGameServer":        "DeleteGameServer",
		"/obiente.cloud.gameservers.v1.GameServerService/StartGameServer":         "StartGameServer",
		"/obiente.cloud.gameservers.v1.GameServerService/StopGameServer":          "StopGameServer",
		"/obiente.cloud.gameservers.v1.GameServerService/RestartGameServer":       "RestartGameServer",
		"/obiente.cloud.gameservers.v1.GameServerService/StreamGameServerStatus":  "StreamGameServerStatus",
		"/obiente.cloud.gameservers.v1.GameServerService/GetGameServerMetrics":    "GetGameServerMetrics",
		"/obiente.cloud.gameservers.v1.GameServerService/StreamGameServerMetrics": "StreamGameServerMetrics",
		"/obiente.cloud.gameservers.v1.GameServerService/GetGameServerUsage":      "GetGameServerUsage",
	}

	RegisterServiceProcedures("GameServerService", procedures, public)
}

// RegisterBillingServiceProcedures registers all BillingService procedures
// Note: Billing operations are manually registered with semantic permission names
// to avoid confusing permissions like "billing.attach" or "billing.detach"
func RegisterBillingServiceProcedures() {
	registry := GetPermissionRegistry()
	public := []string{}

	// Register with explicit, semantic permission names
	billingProcedures := []struct {
		procedure    string
		permission   string
		resourceType string
		action       string
		description  string
	}{
		{"/obiente.cloud.billing.v1.BillingService/CreateCheckoutSession", "billing.create", "billing", "create", "Create checkout session"},
		{"/obiente.cloud.billing.v1.BillingService/CreatePaymentIntent", "billing.create", "billing", "create", "Create payment intent"},
		{"/obiente.cloud.billing.v1.BillingService/CreatePortalSession", "billing.read", "billing", "read", "Create billing portal session"},
		{"/obiente.cloud.billing.v1.BillingService/CreateSetupIntent", "billing.update", "billing", "update", "Setup payment method"},
		{"/obiente.cloud.billing.v1.BillingService/GetBillingAccount", "billing.read", "billing", "read", "View billing account"},
		{"/obiente.cloud.billing.v1.BillingService/UpdateBillingAccount", "billing.update", "billing", "update", "Update billing account"},
		{"/obiente.cloud.billing.v1.BillingService/ListPaymentMethods", "billing.read", "billing", "read", "View payment methods"},
		{"/obiente.cloud.billing.v1.BillingService/AttachPaymentMethod", "billing.update", "billing", "update", "Add payment method"},
		{"/obiente.cloud.billing.v1.BillingService/DetachPaymentMethod", "billing.update", "billing", "update", "Remove payment method"},
		{"/obiente.cloud.billing.v1.BillingService/SetDefaultPaymentMethod", "billing.update", "billing", "update", "Set default payment method"},
		{"/obiente.cloud.billing.v1.BillingService/GetPaymentStatus", "billing.read", "billing", "read", "View payment status"},
		{"/obiente.cloud.billing.v1.BillingService/ListInvoices", "billing.read", "billing", "read", "View invoices"},
	}

	for _, proc := range billingProcedures {
		isPublic := false
		for _, pub := range public {
			if pub == proc.procedure {
				isPublic = true
				break
			}
		}
		registry.RegisterProcedure(proc.procedure, proc.permission, proc.resourceType, proc.action, proc.description, isPublic)
	}
}

// RegisterOrganizationServiceProcedures registers all OrganizationService procedures
// Note: Organization operations are manually registered with semantic permission names
// for better UX (e.g., "organization.members.invite" instead of "organization.invite")
func RegisterOrganizationServiceProcedures() {
	registry := GetPermissionRegistry()
	// ListOrganizations and CreateOrganization are user-based operations:
	// - ListOrganizations: users see their own orgs (filtered by membership), superadmins see all
	// - CreateOrganization: any authenticated user can create an org (they become owner)
	// - GetOrganization: currently has no auth check in service (may need review)
	public := []string{
		"/obiente.cloud.organizations.v1.OrganizationService/ListOrganizations",
		"/obiente.cloud.organizations.v1.OrganizationService/CreateOrganization",
		"/obiente.cloud.organizations.v1.OrganizationService/GetOrganization", // No auth check in service - service handles access
	}

	// Register with explicit, semantic permission names
	orgProcedures := []struct {
		procedure    string
		permission   string
		resourceType string
		action       string
		description  string
	}{
		// Basic org operations (user-based, marked as public)
		{"/obiente.cloud.organizations.v1.OrganizationService/ListOrganizations", "organization.read", "organization", "read", "View organizations"},
		{"/obiente.cloud.organizations.v1.OrganizationService/CreateOrganization", "organization.create", "organization", "create", "Create organization"},
		{"/obiente.cloud.organizations.v1.OrganizationService/GetOrganization", "organization.read", "organization", "read", "View organization details"},

		// Org management (requires org admin/owner)
		{"/obiente.cloud.organizations.v1.OrganizationService/UpdateOrganization", "organization.update", "organization", "update", "Update organization"},
		{"/obiente.cloud.organizations.v1.OrganizationService/DeleteOrganization", "organization.delete", "organization", "delete", "Delete organization"},

		// Member management (requires org admin/owner)
		{"/obiente.cloud.organizations.v1.OrganizationService/ListMembers", "organization.members.read", "organization", "members.read", "View organization members"},
		{"/obiente.cloud.organizations.v1.OrganizationService/InviteMember", "organization.members.invite", "organization", "members.invite", "Invite member to organization"},
		{"/obiente.cloud.organizations.v1.OrganizationService/UpdateMember", "organization.members.update", "organization", "members.update", "Update member role"},
		{"/obiente.cloud.organizations.v1.OrganizationService/RemoveMember", "organization.members.delete", "organization", "members.delete", "Remove member from organization"},

		// Usage and billing
		{"/obiente.cloud.organizations.v1.OrganizationService/GetUsage", "organization.read", "organization", "read", "View organization usage"},
		{"/obiente.cloud.organizations.v1.OrganizationService/GetCreditLog", "organization.read", "organization", "read", "View credit log"},

		// Admin operations (superadmin only) - hierarchical permissions
		// These are marked as superadmin-only and won't appear in organization permission trees
		{"/obiente.cloud.organizations.v1.OrganizationService/AdminAddCredits", "organization.admin.add_credits", "organization", "admin.add_credits", "Add credits (admin)"},
		{"/obiente.cloud.organizations.v1.OrganizationService/AdminRemoveCredits", "organization.admin.remove_credits", "organization", "admin.remove_credits", "Remove credits (admin)"},
	}

	for _, proc := range orgProcedures {
		isPublic := false
		for _, pub := range public {
			if pub == proc.procedure {
				isPublic = true
				break
			}
		}
		// Mark admin operations as superadmin-only
		isSuperadminOnly := strings.HasPrefix(proc.permission, "organization.admin.") || strings.HasPrefix(proc.permission, "admin.")
		registry.RegisterProcedureWithFlags(proc.procedure, proc.permission, proc.resourceType, proc.action, proc.description, isPublic, isSuperadminOnly)
	}
}

// RegisterSupportServiceProcedures registers all SupportService procedures
// Note: Support tickets are user-based, not organization-based. The service handles
// access control internally (users see their own tickets, superadmins see all).
// These procedures don't require organization-level permissions.
func RegisterSupportServiceProcedures() {
	// Mark all support procedures as public (no permission check needed)
	// The service itself enforces: users can only access their own tickets,
	// superadmins can access all tickets
	public := []string{
		"/obiente.cloud.support.v1.SupportService/CreateTicket",
		"/obiente.cloud.support.v1.SupportService/ListTickets",
		"/obiente.cloud.support.v1.SupportService/GetTicket",
		"/obiente.cloud.support.v1.SupportService/UpdateTicket", // Only superadmins can update, but service handles this
		"/obiente.cloud.support.v1.SupportService/AddComment",
		"/obiente.cloud.support.v1.SupportService/ListComments",
	}
	procedures := map[string]string{
		"/obiente.cloud.support.v1.SupportService/CreateTicket": "CreateTicket",
		"/obiente.cloud.support.v1.SupportService/ListTickets":  "ListTickets",
		"/obiente.cloud.support.v1.SupportService/GetTicket":    "GetTicket",
		"/obiente.cloud.support.v1.SupportService/UpdateTicket": "UpdateTicket",
		"/obiente.cloud.support.v1.SupportService/AddComment":   "AddComment",
		"/obiente.cloud.support.v1.SupportService/ListComments": "ListComments",
	}

	RegisterServiceProcedures("SupportService", procedures, public)
}

// RegisterNotificationServiceProcedures registers all NotificationService procedures
// Note: Notifications are user-based, not organization-based. The service handles
// access control internally (users can only access their own notifications).
// Admin operations (CreateNotification, CreateOrganizationNotification) are protected
// by the service itself (require superadmin or internal service).
func RegisterNotificationServiceProcedures() {
	// Mark all notification procedures as public (no permission check needed)
	// The service itself enforces: users can only access their own notifications,
	// admin operations require superadmin or internal service auth
	public := []string{
		"/obiente.cloud.notifications.v1.NotificationService/ListNotifications",
		"/obiente.cloud.notifications.v1.NotificationService/GetNotification",
		"/obiente.cloud.notifications.v1.NotificationService/GetUnreadCount",
		"/obiente.cloud.notifications.v1.NotificationService/MarkAsRead",
		"/obiente.cloud.notifications.v1.NotificationService/MarkAllAsRead",
		"/obiente.cloud.notifications.v1.NotificationService/DeleteNotification",
		"/obiente.cloud.notifications.v1.NotificationService/DeleteAllNotifications",
		"/obiente.cloud.notifications.v1.NotificationService/CreateNotification",             // Admin-only, but service handles auth
		"/obiente.cloud.notifications.v1.NotificationService/CreateOrganizationNotification", // Admin-only, but service handles auth
		"/obiente.cloud.notifications.v1.NotificationService/GetNotificationTypes",
		"/obiente.cloud.notifications.v1.NotificationService/GetNotificationPreferences",
		"/obiente.cloud.notifications.v1.NotificationService/UpdateNotificationPreferences",
	}
	procedures := map[string]string{
		"/obiente.cloud.notifications.v1.NotificationService/ListNotifications":              "ListNotifications",
		"/obiente.cloud.notifications.v1.NotificationService/GetNotification":                "GetNotification",
		"/obiente.cloud.notifications.v1.NotificationService/GetUnreadCount":                 "GetUnreadCount",
		"/obiente.cloud.notifications.v1.NotificationService/MarkAsRead":                     "MarkAsRead",
		"/obiente.cloud.notifications.v1.NotificationService/MarkAllAsRead":                  "MarkAllAsRead",
		"/obiente.cloud.notifications.v1.NotificationService/DeleteNotification":             "DeleteNotification",
		"/obiente.cloud.notifications.v1.NotificationService/DeleteAllNotifications":         "DeleteAllNotifications",
		"/obiente.cloud.notifications.v1.NotificationService/CreateNotification":             "CreateNotification",
		"/obiente.cloud.notifications.v1.NotificationService/CreateOrganizationNotification": "CreateOrganizationNotification",
		"/obiente.cloud.notifications.v1.NotificationService/GetNotificationTypes":           "GetNotificationTypes",
		"/obiente.cloud.notifications.v1.NotificationService/GetNotificationPreferences":     "GetNotificationPreferences",
		"/obiente.cloud.notifications.v1.NotificationService/UpdateNotificationPreferences":  "UpdateNotificationPreferences",
	}

	RegisterServiceProcedures("NotificationService", procedures, public)
}

// RegisterAdminServiceProcedures registers all AdminService procedures
// Note: Admin operations are manually registered with hierarchical permission names
// for better UX (e.g., "admin.roles.read" instead of "admin.read")
func RegisterAdminServiceProcedures() {
	registry := GetPermissionRegistry()
	public := []string{}

	// Register with explicit, hierarchical permission names
	adminProcedures := []struct {
		procedure    string
		permission   string
		resourceType string
		action       string
		description  string
	}{
		// Permissions management
		{"/obiente.cloud.admin.v1.AdminService/ListPermissions", "admin.permissions.read", "admin", "permissions.read", "View available permissions"},

		// Role management
		{"/obiente.cloud.admin.v1.AdminService/ListRoles", "admin.roles.read", "admin", "roles.read", "View roles"},
		{"/obiente.cloud.admin.v1.AdminService/CreateRole", "admin.roles.create", "admin", "roles.create", "Create role"},
		{"/obiente.cloud.admin.v1.AdminService/UpdateRole", "admin.roles.update", "admin", "roles.update", "Update role"},
		{"/obiente.cloud.admin.v1.AdminService/DeleteRole", "admin.roles.delete", "admin", "roles.delete", "Delete role"},

		// Role binding management
		{"/obiente.cloud.admin.v1.AdminService/ListRoleBindings", "admin.bindings.read", "admin", "bindings.read", "View role bindings"},
		{"/obiente.cloud.admin.v1.AdminService/CreateRoleBinding", "admin.bindings.create", "admin", "bindings.create", "Create role binding"},
		{"/obiente.cloud.admin.v1.AdminService/DeleteRoleBinding", "admin.bindings.delete", "admin", "bindings.delete", "Delete role binding"},

		// Quota management
		{"/obiente.cloud.admin.v1.AdminService/UpsertOrgQuota", "admin.quotas.update", "admin", "quotas.update", "Update organization quota"},
	}

	for _, proc := range adminProcedures {
		isPublic := false
		for _, pub := range public {
			if pub == proc.procedure {
				isPublic = true
				break
			}
		}
		// All admin.* permissions are superadmin-only
		registry.RegisterProcedureWithFlags(proc.procedure, proc.permission, proc.resourceType, proc.action, proc.description, isPublic, true)
	}
}

// RegisterSuperadminServiceProcedures registers all SuperadminService procedures
// All superadmin procedures are marked as superadmin-only and use hierarchical permission names
func RegisterSuperadminServiceProcedures() {
	registry := GetPermissionRegistry()
	public := []string{
		"/obiente.cloud.superadmin.v1.SuperadminService/GetPricing", // Public pricing endpoint
	}

	superadminProcedures := []struct {
		procedure    string
		permission   string
		resourceType string
		action       string
		description  string
	}{
		// Overview
		{"/obiente.cloud.superadmin.v1.SuperadminService/GetOverview", "superadmin.overview.read", "superadmin", "overview.read", "View system overview"},

		// DNS management
		{"/obiente.cloud.superadmin.v1.SuperadminService/QueryDNS", "superadmin.dns.read", "superadmin", "dns.read", "Query DNS records"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/ListDNSRecords", "superadmin.dns.read", "superadmin", "dns.read", "List DNS records"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/GetDNSConfig", "superadmin.dns.read", "superadmin", "dns.read", "View DNS configuration"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/ListDelegatedDNSRecords", "superadmin.dns.read", "superadmin", "dns.read", "List delegated DNS records"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/HasDelegatedDNS", "superadmin.dns.read", "superadmin", "dns.read", "Check delegated DNS status"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/CreateDNSDelegationAPIKey", "superadmin.dns.create", "superadmin", "dns.create", "Create DNS delegation API key"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/ListDNSDelegationAPIKeys", "superadmin.dns.read", "superadmin", "dns.read", "List DNS delegation API keys"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/RevokeDNSDelegationAPIKey", "superadmin.dns.delete", "superadmin", "dns.delete", "Revoke DNS delegation API key"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/RevokeDNSDelegationAPIKeyForOrganization", "superadmin.dns.delete", "superadmin", "dns.delete", "Revoke DNS delegation API key for organization"},

		// Abuse detection
		{"/obiente.cloud.superadmin.v1.SuperadminService/GetAbuseDetection", "superadmin.abuse.read", "superadmin", "abuse.read", "View abuse detection data"},

		// Income and billing
		{"/obiente.cloud.superadmin.v1.SuperadminService/GetIncomeOverview", "superadmin.income.read", "superadmin", "income.read", "View income overview"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/ListAllInvoices", "superadmin.invoices.read", "superadmin", "invoices.read", "List all invoices"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/SendInvoiceReminder", "superadmin.invoices.update", "superadmin", "invoices.update", "Send invoice reminder"},

		// Plan management
		{"/obiente.cloud.superadmin.v1.SuperadminService/ListPlans", "superadmin.plans.read", "superadmin", "plans.read", "List plans"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/CreatePlan", "superadmin.plans.create", "superadmin", "plans.create", "Create plan"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/UpdatePlan", "superadmin.plans.update", "superadmin", "plans.update", "Update plan"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/DeletePlan", "superadmin.plans.delete", "superadmin", "plans.delete", "Delete plan"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/AssignPlanToOrganization", "superadmin.plans.update", "superadmin", "plans.update", "Assign plan to organization"},

		// User management
		{"/obiente.cloud.superadmin.v1.SuperadminService/ListUsers", "superadmin.users.read", "superadmin", "users.read", "List all users"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/GetUser", "superadmin.users.read", "superadmin", "users.read", "View user details"},

		// VPS management
		{"/obiente.cloud.superadmin.v1.SuperadminService/ListAllVPS", "superadmin.vps.read", "superadmin", "vps.read", "List all VPS instances"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/SuperadminGetVPS", "superadmin.vps.read", "superadmin", "vps.read", "View VPS instance"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/SuperadminResizeVPS", "superadmin.vps.update", "superadmin", "vps.update", "Resize VPS instance"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/SuperadminSuspendVPS", "superadmin.vps.update", "superadmin", "vps.update", "Suspend VPS instance"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/SuperadminUnsuspendVPS", "superadmin.vps.update", "superadmin", "vps.update", "Unsuspend VPS instance"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/SuperadminUpdateVPSCloudInit", "superadmin.vps.update", "superadmin", "vps.update", "Update VPS cloud-init"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/SuperadminForceStopVPS", "superadmin.vps.update", "superadmin", "vps.update", "Force stop VPS instance"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/SuperadminForceDeleteVPS", "superadmin.vps.delete", "superadmin", "vps.delete", "Force delete VPS instance"},

		// VPS size catalog
		{"/obiente.cloud.superadmin.v1.SuperadminService/ListVPSSizes", "superadmin.vps_sizes.read", "superadmin", "vps_sizes.read", "List VPS sizes"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/CreateVPSSize", "superadmin.vps_sizes.create", "superadmin", "vps_sizes.create", "Create VPS size"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/UpdateVPSSize", "superadmin.vps_sizes.update", "superadmin", "vps_sizes.update", "Update VPS size"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/DeleteVPSSize", "superadmin.vps_sizes.delete", "superadmin", "vps_sizes.delete", "Delete VPS size"},

		// VPS public IPs
		{"/obiente.cloud.superadmin.v1.SuperadminService/ListVPSPublicIPs", "superadmin.vps_public_ips.read", "superadmin", "vps_public_ips.read", "List VPS public IPs"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/CreateVPSPublicIP", "superadmin.vps_public_ips.create", "superadmin", "vps_public_ips.create", "Create VPS public IP"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/UpdateVPSPublicIP", "superadmin.vps_public_ips.update", "superadmin", "vps_public_ips.update", "Update VPS public IP"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/DeleteVPSPublicIP", "superadmin.vps_public_ips.delete", "superadmin", "vps_public_ips.delete", "Delete VPS public IP"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/AssignVPSPublicIP", "superadmin.vps_public_ips.update", "superadmin", "vps_public_ips.update", "Assign VPS public IP"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/UnassignVPSPublicIP", "superadmin.vps_public_ips.update", "superadmin", "vps_public_ips.update", "Unassign VPS public IP"},

		// Stripe webhook events
		{"/obiente.cloud.superadmin.v1.SuperadminService/ListStripeWebhookEvents", "superadmin.webhooks.read", "superadmin", "webhooks.read", "List Stripe webhook events"},

		// Node management
		{"/obiente.cloud.superadmin.v1.SuperadminService/ListNodes", "superadmin.nodes.read", "superadmin", "nodes.read", "List nodes"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/GetNode", "superadmin.nodes.read", "superadmin", "nodes.read", "View node details"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/UpdateNodeConfig", "superadmin.nodes.update", "superadmin", "nodes.update", "Update node configuration"},

		// Superadmin permissions catalog
		{"/obiente.cloud.superadmin.v1.SuperadminService/ListSuperadminPermissions", "admin.permissions.read", "admin", "permissions.read", "View superadmin permissions"},

		// Superadmin role management
		{"/obiente.cloud.superadmin.v1.SuperadminService/ListSuperadminRoles", "admin.roles.read", "admin", "roles.read", "View superadmin roles"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/CreateSuperadminRole", "admin.roles.create", "admin", "roles.create", "Create superadmin role"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/UpdateSuperadminRole", "admin.roles.update", "admin", "roles.update", "Update superadmin role"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/DeleteSuperadminRole", "admin.roles.delete", "admin", "roles.delete", "Delete superadmin role"},

		// Superadmin role bindings
		{"/obiente.cloud.superadmin.v1.SuperadminService/ListSuperadminRoleBindings", "admin.bindings.read", "admin", "bindings.read", "View superadmin role bindings"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/CreateSuperadminRoleBinding", "admin.bindings.create", "admin", "bindings.create", "Create superadmin role binding"},
		{"/obiente.cloud.superadmin.v1.SuperadminService/DeleteSuperadminRoleBinding", "admin.bindings.delete", "admin", "bindings.delete", "Delete superadmin role binding"},
	}

	// Register superadmin permissions for Support Service (all ticket access)
	supportProcedures := []struct {
		procedure    string
		permission   string
		resourceType string
		action       string
		description  string
	}{
		{"/obiente.cloud.support.v1.SupportService/ListTickets", "superadmin.support.read", "superadmin", "support.read", "View all support tickets"},
		{"/obiente.cloud.support.v1.SupportService/GetTicket", "superadmin.support.read", "superadmin", "support.read", "View any support ticket"},
		{"/obiente.cloud.support.v1.SupportService/UpdateTicket", "superadmin.support.update", "superadmin", "support.update", "Update any support ticket"},
		{"/obiente.cloud.support.v1.SupportService/AddComment", "superadmin.support.update", "superadmin", "support.update", "Add comment to any ticket (including internal)"},
		{"/obiente.cloud.support.v1.SupportService/ListComments", "superadmin.support.read", "superadmin", "support.read", "View all comments (including internal)"},
	}

	for _, proc := range supportProcedures {
		// All superadmin support procedures are superadmin-only
		registry.RegisterProcedureWithFlags(proc.procedure, proc.permission, proc.resourceType, proc.action, proc.description, false, true)
	}

	// Register superadmin permissions for Deployment Service (view all deployments)
	deploymentProcedures := []struct {
		procedure    string
		permission   string
		resourceType string
		action       string
		description  string
	}{
		{"/obiente.cloud.deployments.v1.DeploymentService/ListDeployments", "superadmin.deployments.read", "superadmin", "deployments.read", "View all deployments"},
		{"/obiente.cloud.deployments.v1.DeploymentService/GetDeployment", "superadmin.deployments.read", "superadmin", "deployments.read", "View any deployment"},
		{"/obiente.cloud.deployments.v1.DeploymentService/GetDeploymentLogs", "superadmin.deployments.read", "superadmin", "deployments.read", "View logs for any deployment"},
		{"/obiente.cloud.deployments.v1.DeploymentService/StreamDeploymentLogs", "superadmin.deployments.read", "superadmin", "deployments.read", "Stream logs for any deployment"},
		{"/obiente.cloud.deployments.v1.DeploymentService/GetBuildLogs", "superadmin.deployments.read", "superadmin", "deployments.read", "View build logs for any deployment"},
		{"/obiente.cloud.deployments.v1.DeploymentService/GetDeploymentMetrics", "superadmin.deployments.read", "superadmin", "deployments.read", "View metrics for any deployment"},
		{"/obiente.cloud.deployments.v1.DeploymentService/StreamDeploymentMetrics", "superadmin.deployments.read", "superadmin", "deployments.read", "Stream metrics for any deployment"},
		{"/obiente.cloud.deployments.v1.DeploymentService/GetDeploymentUsage", "superadmin.deployments.read", "superadmin", "deployments.read", "View usage for any deployment"},
	}

	for _, proc := range deploymentProcedures {
		// All superadmin deployment procedures are superadmin-only
		registry.RegisterProcedureWithFlags(proc.procedure, proc.permission, proc.resourceType, proc.action, proc.description, false, true)
	}

	for _, proc := range superadminProcedures {
		isPublic := false
		for _, pub := range public {
			if pub == proc.procedure {
				isPublic = true
				break
			}
		}
		// All superadmin procedures are superadmin-only (except public ones)
		registry.RegisterProcedureWithFlags(proc.procedure, proc.permission, proc.resourceType, proc.action, proc.description, isPublic, true)
	}
}

// RegisterAllServices registers procedures for all services
// This should be called at application startup
func RegisterAllServices() {
	RegisterDeploymentServiceProcedures()
	RegisterVPSServiceProcedures()
	RegisterGameServerServiceProcedures()
	RegisterBillingServiceProcedures()
	RegisterOrganizationServiceProcedures()
	RegisterSupportServiceProcedures()
	RegisterNotificationServiceProcedures()
	RegisterAdminServiceProcedures()
	RegisterSuperadminServiceProcedures()
}
