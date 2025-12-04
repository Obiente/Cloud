// Map routes to required permissions
const routePermissions: Record<string, string> = {
  "/superadmin": "superadmin.overview.read",
  "/superadmin/audit-logs": "superadmin.overview.read",
  "/superadmin/organizations": "superadmin.overview.read",
  "/superadmin/plans": "superadmin.plans.read",
  "/superadmin/deployments": "superadmin.deployments.read",
  "/superadmin/vps": "superadmin.vps.read",
  "/superadmin/nodes": "superadmin.nodes.read",
  "/superadmin/users": "superadmin.users.read",
  "/superadmin/usage": "superadmin.overview.read",
  "/superadmin/dns": "superadmin.dns.read",
  "/superadmin/abuse": "superadmin.abuse.read",
  "/superadmin/income": "superadmin.income.read",
  "/superadmin/invoices": "superadmin.invoices.read",
  "/superadmin/webhook-events": "superadmin.webhooks.read",
  "/superadmin/roles": "admin.roles.read",
  "/superadmin/role-bindings": "admin.bindings.read",
};

export default defineNuxtRouteMiddleware(async (to) => {
  // Only run on client side - server side auth check happens in auth middleware
  if (import.meta.server) {
    return;
  }

  const superAdmin = useSuperAdmin();
  await superAdmin.fetchOverview();

  // Check if user has any superadmin access
  if (superAdmin.allowed.value === false) {
    return navigateTo("/dashboard");
  }

  // Check specific permission for this route
  const route = to.path;
  const requiredPerm = routePermissions[route];
  
  if (requiredPerm && !superAdmin.hasPermission(requiredPerm)) {
    // User doesn't have permission for this specific page
    // Redirect to overview if they have that, otherwise to dashboard
    if (superAdmin.hasPermission("superadmin.overview.read")) {
      return navigateTo("/superadmin");
    }
    return navigateTo("/dashboard");
  }
});
