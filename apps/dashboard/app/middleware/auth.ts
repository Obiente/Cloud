export default defineNuxtRouteMiddleware(async (to, from) => {
  const user = useAuth();
  // Don't block navigation - fetch auth in background
  // On SSR: use timeout to prevent blocking page render, but ensure token refresh completes
  // On client: no timeout - let slow connections complete
  if (import.meta.server) {
    // On SSR, we need to ensure auth is fetched (including token refresh if needed)
    // before API calls are made. Use a longer timeout to allow token refresh to complete.
    const fetchPromise = user.fetch();
    const timeoutPromise = new Promise(resolve => setTimeout(resolve, 2000)); // Increased to 2s to allow token refresh
    await Promise.race([fetchPromise, timeoutPromise]);
    // Don't return early - continue to check if user is authenticated
  } else {
    // Client-side: fetch without timeout, don't block navigation
    user.fetch().catch(() => null);
  }
  if (!user.session || !user.user) {
    // Check if we just logged out (prevent silent auth immediately after logout)
    if (import.meta.client) {
      const logoutTime = sessionStorage.getItem("obiente_logout_time");
      if (logoutTime) {
        const timeSinceLogout = Date.now() - parseInt(logoutTime, 10);
        // If we logged out recently (within 1 minute), skip silent auth
        // Allow page to load normally - user can login when ready
        if (timeSinceLogout < 60000) {
          return;
        }
        // Clear the flag after timeout
        sessionStorage.removeItem("obiente_logout_time");
      }
    }

    // Try silent auth first (background iframe)
    // If it fails, don't show popup - let user login normally on the page
    const silentAuthSuccess = await user.trySilentAuth();
    if (!silentAuthSuccess) {
      // Silent auth failed silently in iframe - allow page to load normally
      // User can click login button to authenticate
      return;
    }
  }
});
