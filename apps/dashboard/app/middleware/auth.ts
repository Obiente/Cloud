export default defineNuxtRouteMiddleware(async () => {
  // The SSR payload owns initial authentication. Refresh it after hydration,
  // when changing the authenticated navigation cannot invalidate the DOM.
  if (import.meta.client && useNuxtApp().isHydrating) return;
  // Check if auth is disabled via public config
  const appConfig = useConfig();
  // Fetch config if not already loaded (with timeout to avoid blocking)
  if (appConfig.disableAuth.value === null && !appConfig.loading.value) {
    const fetchPromise = appConfig.fetchConfig();
    const timeoutPromise = new Promise(resolve => setTimeout(resolve, 1000)); // 1s timeout
    await Promise.race([fetchPromise, timeoutPromise]);
  }
  
  // If auth is disabled, skip all authentication checks
  if (appConfig.disableAuth.value === true) {
    return;
  }

  const user = useAuth();
  // SSR must settle the session before rendering. Later client navigation can
  // refresh in the background. The request itself has a bounded timeout.
  if (import.meta.server) {
    // Resolve token refresh before the page makes authenticated API calls.
    await user.fetch();
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

    // Try silent auth in the background (non-blocking)
    // Start the iframe auth process but don't wait for it - let the page load normally
    // The silent auth will update the session when it completes
    if (import.meta.client) {
      user.trySilentAuth().catch(() => {
        // Silent auth failed silently in iframe - allow page to load normally
        // User can click login button to authenticate
      });
    }
    // Allow page to load immediately without waiting for silent auth
    return;
  }
});
