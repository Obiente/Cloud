export default defineNuxtRouteMiddleware(async (to, from) => {
  const user = useAuth();
  await user.fetch();
  if (import.meta.server) return;
  if (!user.session || !user.user) {
    // Try silent auth first (background popup)
    const silentAuthSuccess = await user.trySilentAuth();
    if (!silentAuthSuccess) {
      // Silent auth failed, redirect to our custom login page
      return navigateTo("/auth/login");
    }
  }
});
