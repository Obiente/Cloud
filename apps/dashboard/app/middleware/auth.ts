export default defineNuxtRouteMiddleware(async (to, from) => {
  const user = useAuth();
  await user.fetch();
  if (import.meta.server) return;
  if (!user.session || !user.user) {
    // Check if we just logged out (prevent silent auth immediately after logout)
    if (import.meta.client) {
      const logoutTime = sessionStorage.getItem("obiente_logout_time");
      if (logoutTime) {
        const timeSinceLogout = Date.now() - parseInt(logoutTime, 10);
        // If we logged out recently (within 1 minute), skip silent auth
        if (timeSinceLogout < 60000) {
          return navigateTo("/auth/login");
        }
        // Clear the flag after timeout
        sessionStorage.removeItem("obiente_logout_time");
      }
    }

    // Try silent auth first (background iframe)
    const silentAuthSuccess = await user.trySilentAuth();
    if (!silentAuthSuccess) {
      // Silent auth failed, redirect to our custom login page
      return navigateTo("/auth/login");
    }
  }
});
