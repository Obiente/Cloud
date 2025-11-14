export default defineNuxtRouteMiddleware(async () => {
  // Only run on client side - server side auth check happens in auth middleware
  if (import.meta.server) {
    return;
  }

  const superAdmin = useSuperAdmin();
  await superAdmin.fetchOverview();

  if (superAdmin.allowed.value === false) {
    return navigateTo("/dashboard");
  }
});
