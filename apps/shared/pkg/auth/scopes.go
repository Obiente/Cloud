package auth

// Permission constants per domain
const (
    // Deployments
    PermDeploymentsCreate = "deployments.create"
    PermDeploymentsRead   = "deployments.read"
    PermDeploymentsUpdate = "deployments.update"
    PermDeploymentsDelete = "deployments.delete"
    PermDeploymentsLogs   = "deployments.logs"
    PermDeploymentsScale  = "deployments.scale"

    // Environments
    PermEnvironmentsManage = "environments.manage"
    PermEnvironmentsDeploy = "environments.deploy"

    // Admin
    PermAdminRolesRead    = "admin.roles.read"
    PermAdminRolesWrite   = "admin.roles.write"
    PermAdminBindingsRead = "admin.bindings.read"
    PermAdminBindingsWrite= "admin.bindings.write"
    PermAdminQuotasUpdate = "admin.quotas.update"
)

var ScopeDescriptions = map[string]string{
    PermDeploymentsCreate: "Create new deployments",
    PermDeploymentsRead:   "View deployments and details",
    PermDeploymentsUpdate: "Edit deployment configuration",
    PermDeploymentsDelete: "Delete deployments",
    PermDeploymentsLogs:   "View deployment logs",
    PermDeploymentsScale:  "Scale deployment replicas",

    PermEnvironmentsManage: "Manage environments and settings",
    PermEnvironmentsDeploy: "Deploy to environment",

    PermAdminRolesRead:     "View roles",
    PermAdminRolesWrite:    "Create/update roles",
    PermAdminBindingsRead:  "View role bindings",
    PermAdminBindingsWrite: "Create/update role bindings",
    PermAdminQuotasUpdate:  "Update organization quotas",
}
