import { createError } from "h3";

export default defineNuxtRouteMiddleware(async () => {
  const superAdmin = useSuperAdmin();
  await superAdmin.fetchOverview();

  if (superAdmin.allowed.value === false) {
    return navigateTo("/dashboard");
  }
});
