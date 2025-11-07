import { appendResponseHeader } from "h3";
import type { User, UserSession } from "@obiente/types";
import { useOrganizationsStore } from "~/stores/organizations";

export const useAuth = () => {
  const serverEvent = import.meta.server ? useRequestEvent() : null;

  // Reactive state
  const sessionState = useState<UserSession | null>(
    "obiente-session",
    () => null
  );
  const authReadyState = useState("obiente-auth-ready", () => false);
  const accessToken = useState<string | null>("auth-token", () => null);
  const tokenExpiry = useState<number | null>("token-expiry", () => null);
  const isRefreshing = useState<boolean>("is-refreshing", () => false);
  const user = computed(() => sessionState.value?.user || null);
  const isAuthenticated = computed(
    () => Boolean(sessionState.value && user.value)
  );
  const isLoading = ref(false);

  const orgStore = useOrganizationsStore();
  orgStore.hydrate();

  // Get current user session
  const fetch = async () => {
    try {
      isLoading.value = true;
      sessionState.value = await useRequestFetch()<UserSession>(
        "/auth/session",
        {
          headers: {
            accept: "application/json",
          },
          retry: false,
        }
      ).catch(() => null);

      if (!authReadyState.value) {
        authReadyState.value = true;
      }
    } catch (error) {
      console.error("Failed to get current user:", error);
      sessionState.value = null;
    } finally {
      isLoading.value = false;
    }
  };

  // Logout function
  const logout = async () => {
    // Set flag to prevent silent auth immediately after logout
    if (import.meta.client) {
      sessionStorage.setItem("obiente_logout_time", Date.now().toString());
    }

    // Get id_token from session BEFORE clearing it (needed for logout)
    const idToken = sessionState.value?.secure?.id_token;
    
    await useRequestFetch()("/auth/session", {
      method: "DELETE",
      onResponse({ response: { headers } }) {
        if (import.meta.server && serverEvent) {
          for (const setCookie of headers.getSetCookie()) {
            appendResponseHeader(serverEvent, "Set-Cookie", setCookie);
          }
        }
      },
    });
    sessionState.value = null;
    authReadyState.value = false;
    orgStore.reset();

    // Redirect to Zitadel logout endpoint to clear Zitadel session
    if (import.meta.client) {
      const config = useRuntimeConfig();
      // Redirect to homepage after logout using requestHost from config
      // The post_logout_redirect_uri must be registered in Zitadel client configuration
      // IMPORTANT: The URI must match EXACTLY (including protocol, domain, and path)
      // Use requestHost from config to ensure consistency across environments
      const homepageUrl = config.public.requestHost || window.location.origin;
      
      console.log("[Auth] Logging out - redirect URI:", homepageUrl);
      
      // Get the end_session endpoint
      // Zitadel's end_session endpoint is at /oidc/v1/end_session
      // According to OIDC spec: https://openid.net/specs/openid-connect-rpinitiated-1_0.html
      const endSessionEndpoint = `${config.public.oidcBase}/oidc/v1/end_session`;
      
      // Build parameters according to OIDC RP-Initiated Logout spec
      // See: https://zitadel.com/docs/guides/integrate/login/oidc/logout
      // Reference: https://zitadel.com/docs/apis/openidoauth/endpoints
      const params = new URLSearchParams();
      
      // post_logout_redirect_uri is REQUIRED and must be registered in Zitadel
      // This must match EXACTLY one of the URIs in "Post Logout Redirect URIs" in Zitadel console
      params.set("post_logout_redirect_uri", homepageUrl);
      
      // client_id is REQUIRED if id_token_hint is not provided
      // Zitadel needs either id_token_hint OR client_id to identify the client and validate the redirect URI
      // Always include client_id to ensure Zitadel can identify the client
      params.set("client_id", config.public.oidcClientId);
      
      // id_token_hint helps Zitadel identify which session to terminate
      // This is recommended but optional - client_id can be used instead
      if (idToken) {
        params.set("id_token_hint", idToken);
        console.log("[Auth] Using id_token_hint for logout");
      } else {
        console.log("[Auth] No id_token available, using client_id for logout");
      }
      
      // Optional: state parameter for CSRF protection
      const state = crypto.randomUUID();
      params.set("state", state);
      // Store state to verify on redirect (optional, for security)
      sessionStorage.setItem("logout_state", state);
      
      const logoutUrl = `${endSessionEndpoint}?${params.toString()}`;
      console.log("[Auth] Redirecting to Zitadel logout:", logoutUrl);
      
      // Redirect to Zitadel's end_session endpoint
      // Zitadel will clear the session and redirect back to post_logout_redirect_uri
      // If the redirect URI is not registered, Zitadel will show its UI logout page instead
      window.location.href = logoutUrl;
    }
  };

  // Silent authentication using iframe (Zitadel allows iframes when configured)
  const trySilentAuth = async (): Promise<boolean> => {
    if (import.meta.server) return false;

    // Check if we just logged out (prevent silent auth for 1 minute after logout)
    if (import.meta.client) {
      const logoutTime = sessionStorage.getItem("obiente_logout_time");
      if (logoutTime) {
        const timeSinceLogout = Date.now() - parseInt(logoutTime, 10);
        // Prevent silent auth for 1 minute after logout
        if (timeSinceLogout < 60000) {
          return false;
        }
        // Clear the flag after timeout
        sessionStorage.removeItem("obiente_logout_time");
      }
    }

    return new Promise((resolve) => {
      const iframe = document.createElement("iframe");
      iframe.style.display = "none";
      iframe.style.width = "0";
      iframe.style.height = "0";
      iframe.style.border = "none";
      iframe.style.position = "absolute";
      iframe.style.visibility = "hidden";

      let resolved = false;
      const cleanup = () => {
        if (resolved) return;
        resolved = true;
        try {
          if (iframe.parentNode) {
            document.body.removeChild(iframe);
          }
        } catch (e) {
          // Iframe might already be removed
        }
        window.removeEventListener("message", messageHandler);
      };

      const timeout = setTimeout(() => {
        cleanup();
        resolve(false);
      }, 5000); // 5 second timeout for iframe

      const messageHandler = (e: MessageEvent) => {
        if (resolved) return;
        if (e.origin !== window.location.origin) return;

        if (e.data?.type === "silent-auth-success") {
          cleanup();
          clearTimeout(timeout);
          // Refresh session after successful silent auth
          fetch().then(() => resolve(true));
        } else if (e.data?.type === "silent-auth-error") {
          cleanup();
          clearTimeout(timeout);
          resolve(false);
        }
      };

      window.addEventListener("message", messageHandler);
      iframe.src = "/auth/silent-check";
      document.body.appendChild(iframe);
    });
  };

  // Popup authentication support
  const popupListener = (e: StorageEvent) => {
    if (e.key === "auth-completed") {
      fetch();
      window.removeEventListener("storage", popupListener);
    }
  };

  // Message listener for OAuth errors and signup success from popup (not used for silent auth iframe)
  const messageListener = (e: MessageEvent) => {
    if (e.origin !== window.location.origin) return;
    
    if (e.data?.type === "oauth-error") {
      console.log("[Auth] OAuth error from popup:", e.data.error);
      // Only handle popup errors, not silent auth iframe errors
      // Silent auth errors are handled in trySilentAuth message handler
      if (e.data.error === "login_required" || e.data.error === "interaction_required" || e.data.error === "no_session") {
        window.removeEventListener("message", messageListener);
        // Open Zitadel login popup (only for explicit popup auth, not silent auth)
        popupLogin("/auth/oauth-login");
      }
    } else if (e.data?.type === "signup-success") {
      console.log("[Auth] Signup successful:", e.data.message);
      // Signup completed - user needs to login manually
      // Refresh the page to show login options
      window.removeEventListener("message", messageListener);
      // Optionally show a notification or refresh auth state
      fetch();
    }
  };

  const popupLogin = (
    route: string = "/auth/oauth-login",
    size: { width?: number; height?: number } = {}
  ) => {
    const width = size.width ?? 500;
    const height = size.height ?? 700;
    const top =
      (window.top?.outerHeight ?? 0) / 2 +
      (window.top?.screenY ?? 0) -
      height / 2;
    const left =
      (window.top?.outerWidth ?? 0) / 2 +
      (window.top?.screenX ?? 0) -
      width / 2;

    // Add message listener when opening popup
    window.addEventListener("message", messageListener);

    window.open(
      route,
      "_blank",
      `width=${width}, height=${height}, top=${top}, left=${left}, toolbar=no, location=no, directories=no, status=no, menubar=no, scrollbars=no, resizable=no, copyhistory=no`
    );
    window.addEventListener("storage", popupListener);
  };

  const popupSignup = (
    route: string = "/auth/oauth-signup",
    size: { width?: number; height?: number } = {}
  ) => {
    const width = size.width ?? 500;
    const height = size.height ?? 700;
    const top =
      (window.top?.outerHeight ?? 0) / 2 +
      (window.top?.screenY ?? 0) -
      height / 2;
    const left =
      (window.top?.outerWidth ?? 0) / 2 +
      (window.top?.screenX ?? 0) -
      width / 2;

    // Add message listener when opening popup (same as popupLogin)
    window.addEventListener("message", messageListener);

    console.log("[Auth] Opening signup popup to:", route);
    const popup = window.open(
      route,
      "zitadel-signup",
      `width=${width}, height=${height}, top=${top}, left=${left}, toolbar=no, location=no, directories=no, status=no, menubar=no, scrollbars=no, resizable=no, copyhistory=no`
    );
    
    if (!popup || popup.closed || typeof popup.closed === "undefined") {
      console.error("[Auth] Popup blocked or failed to open - please allow popups for this site");
      alert("Please allow popups for this site to sign up. The popup was blocked by your browser.");
      return;
    }
    
    console.log("[Auth] Signup popup opened successfully");
    window.addEventListener("storage", popupListener);
  };

  // Get the current access token with caching to prevent excessive fetches
  let getAccessTokenPromise: Promise<string | null> | null = null;
  const getAccessToken = async (
    forceRefresh = false
  ): Promise<string | null> => {
    if (import.meta.server) {
      return sessionState.value?.secure?.access_token || null;
    }

    // If we have a valid cached token and not forcing refresh, return it immediately
    if (
      !forceRefresh &&
      accessToken.value &&
      tokenExpiry.value &&
      Date.now() < tokenExpiry.value
    ) {
      return accessToken.value;
    }

    // If there's already a refresh in progress, wait for it instead of starting a new one
    if (!forceRefresh && getAccessTokenPromise) {
      return await getAccessTokenPromise;
    }

    // Only refresh if needed
    if (
      !accessToken.value ||
      forceRefresh ||
      (tokenExpiry.value && Date.now() >= tokenExpiry.value)
    ) {
      getAccessTokenPromise = refreshAccessToken(forceRefresh).then(() => {
        getAccessTokenPromise = null;
        return accessToken.value;
      });
      return await getAccessTokenPromise;
    }

    return accessToken.value;
  };

  // Refresh the access token
  const refreshAccessToken = async (force = false): Promise<void> => {
    try {
      if (isRefreshing.value && !force) {
        return;
      }

      isRefreshing.value = true;

      if (!force && tokenExpiry.value && Date.now() < tokenExpiry.value) {
        return;
      }

      try {
        const response = await $fetch<{
          accessToken: string;
          expiresIn?: number;
        }>("/auth/token");
        if (response?.accessToken) {
          accessToken.value = response.accessToken;
          if (response.expiresIn) {
            tokenExpiry.value = Date.now() + response.expiresIn * 1000 - 30000;
          }
          return;
        }
      } catch {}

        const response = await $fetch<{
          accessToken: string;
          expiresIn?: number;
      }>("/auth/refresh", { method: "POST" });
        if (response.accessToken) {
          accessToken.value = response.accessToken;
          if (response.expiresIn) {
            tokenExpiry.value = Date.now() + response.expiresIn * 1000 - 30000;
        }
      }
    } catch (error) {
      console.error("Failed to refresh access token:", error);
      accessToken.value = null;
      tokenExpiry.value = null;

      if (
        error &&
        typeof error === "object" &&
        "statusCode" in error &&
        (error as any).statusCode === 401
      ) {
        logout();
      }
    } finally {
      isRefreshing.value = false;
    }
  };

  watch(
    () => sessionState.value,
    async (newSession, oldSession) => {
      // Only fetch token if session actually changed (not just on mount)
      if (newSession && newSession !== oldSession) {
        try {
          if (import.meta.client) {
            // Only fetch if we don't have a valid token already
            if (
              !accessToken.value ||
              (tokenExpiry.value && Date.now() >= tokenExpiry.value)
            ) {
              const response = await $fetch<{
                accessToken: string;
                expiresIn?: number;
              }>("/auth/token");
              if (response.accessToken) {
                accessToken.value = response.accessToken;
                if (response.expiresIn) {
                  tokenExpiry.value =
                    Date.now() + response.expiresIn * 1000 - 30000;
                }
              }
            }
          }
        } catch (e) {
          console.error("Failed to fetch token after session update:", e);
        }
      } else if (!newSession && oldSession) {
        // Session was cleared
        accessToken.value = null;
        tokenExpiry.value = null;
      }
    },
    { immediate: false } // Don't run immediately - let onMounted handle initial fetch
  );

  onMounted(() => {
    fetch();

    if (import.meta.client) {
      // Try silent auth first if no session exists
      if (!sessionState.value || !user.value) {
        trySilentAuth().then((success) => {
          if (!success) {
            // Silent auth failed, but don't auto-open popup here
            // Let the middleware handle it if needed
          }
        });
      }

      const tokenCheckInterval = setInterval(() => {
        if (tokenExpiry.value && Date.now() >= tokenExpiry.value) {
          refreshAccessToken(true).catch(console.error);
        }
      }, 60 * 1000);

      onUnmounted(() => {
        clearInterval(tokenCheckInterval);
      });
    }
  });

  return reactive({
    user: user,
    currentOrganization: computed(() => orgStore.currentOrg),
    organizations: computed(() => orgStore.orgs),
    currentOrganizationId: computed(() => orgStore.currentOrgId),
    session: readonly(sessionState),
    ready: computed(() => authReadyState.value),
    isAuthenticated,
    isLoading: readonly(isLoading),

    fetch,
    logout,
    switchOrganization: orgStore.switchOrganization,
    setOrganizations: orgStore.setOrganizations,
    notifyOrganizationsUpdated: orgStore.notifyOrganizationsUpdated,
    getCurrentUser: fetch,
    trySilentAuth,
    popupLogin,
    popupSignup,

    getAccessToken,
    refreshAccessToken,
  });
};
