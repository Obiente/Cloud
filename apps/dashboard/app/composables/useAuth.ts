import { appendResponseHeader } from "@nuxt/nitro-server/h3";
import type { User, UserSession } from "@obiente/types";
import { useOrganizationsStore } from "~/stores/organizations";

let clientAuthInitialized = false;
let clientAuthInitScheduled = false;
let clientTokenCheckInterval: ReturnType<typeof setInterval> | null = null;
let sessionWatcherStarted = false;
const clientAuthRequests = new Set<AbortController>();
let tokenRefreshRequest: { generation: number; promise: Promise<void> } | null =
  null;

export const useAuth = () => {
  const serverEvent = import.meta.server ? useRequestEvent() : null;
  const serverConfig = import.meta.server ? useRuntimeConfig() : undefined;

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
  const isAuthenticated = computed(() =>
    Boolean(sessionState.value && user.value)
  );
  const isLoading = useState("auth-loading", () => false);
  const generation = useState("auth-generation", () => 0);
  const sessionRequest = useState("auth-session-request", () => 0);
  const loggingOut = useState("auth-logging-out", () => false);
  const invalidateToken = () => {
    generation.value++;
    for (const controller of clientAuthRequests) controller.abort();
    clientAuthRequests.clear();
    accessToken.value = null;
    tokenExpiry.value = null;
    isRefreshing.value = false;
  };
  const statusCode = (error: unknown) => {
    const value = error as { statusCode?: number; status?: number } | null;
    return value?.statusCode ?? value?.status;
  };

  const orgStore = useOrganizationsStore();
  // Don't call hydrate() here - the store auto-hydrates after hydration completes
  // Calling it here would access localStorage during plugin initialization, causing Chrome freeze

  // Only the newest request for the current identity may change session state.
  const fetch = async () => {
    if (loggingOut.value) return;
    const request = ++sessionRequest.value;
    const startedGeneration = generation.value;
    const current = () =>
      request === sessionRequest.value &&
      startedGeneration === generation.value;
    isLoading.value = true;
    const controller = new AbortController();
    if (import.meta.client) clientAuthRequests.add(controller);
    try {
      let session: UserSession | null;
      if (import.meta.server && serverEvent) {
        const { readCurrentSession } = await import(
          "../../server/utils/current-session"
        );
        const timeout = setTimeout(() => controller.abort(), 5000);
        try {
          session = await readCurrentSession(
            serverEvent,
            controller.signal,
            serverConfig
          );
        } finally {
          clearTimeout(timeout);
        }
      } else {
        session = await useRequestFetch()<UserSession>("/auth/session", {
          headers: { accept: "application/json" },
          retry: false,
          timeout: 5000,
          signal: controller.signal,
        });
      }
      if (!current()) return;
      authReadyState.value = true;
      sessionState.value = session?.user ? session : null;
    } catch (error) {
      if (!current()) return;
      // An unavailable identity provider is not evidence that the user logged out.
      if (statusCode(error) === 401 || statusCode(error) === 403)
        sessionState.value = null;
      else console.error("[useAuth] Failed to get current user:", error);
    } finally {
      clientAuthRequests.delete(controller);
      if (request === sessionRequest.value) {
        authReadyState.value = true;
        isLoading.value = false;
      }
    }
  };

  // Logout function
  const logout = async () => {
    // Invalidate pending reads before awaiting the session deletion.
    const idToken = sessionState.value?.secure?.id_token;
    loggingOut.value = true;
    invalidateToken();
    sessionRequest.value++;
    sessionState.value = null;
    authReadyState.value = true;
    isLoading.value = false;
    orgStore.reset();
    const logoutGeneration = generation.value;
    if (import.meta.client) {
      try {
        sessionStorage.setItem("obiente_logout_time", Date.now().toString());
      } catch {
        /* Storage may be disabled. */
      }
    }

    try {
      await useRequestFetch()("/auth/session", {
        method: "DELETE",
        retry: false,
        timeout: 5000,
        onResponse({ response: { headers } }) {
          if (import.meta.server && serverEvent) {
            for (const setCookie of headers.getSetCookie()) {
              appendResponseHeader(serverEvent, "Set-Cookie", setCookie);
            }
          }
        },
      });
    } finally {
      if (generation.value === logoutGeneration) loggingOut.value = false;
    }
    if (generation.value !== logoutGeneration) return;

    // Redirect to Zitadel logout endpoint to clear Zitadel session
    if (import.meta.client) {
      const config = useRuntimeConfig();
      // Redirect to homepage after logout using requestHost from config
      // The post_logout_redirect_uri must be registered in Zitadel client configuration
      // IMPORTANT: The URI must match EXACTLY (including protocol, domain, and path)
      // Use requestHost from config to ensure consistency across environments
      const homepageUrl = config.public.requestHost || window.location.origin;

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
      }

      // Optional: state parameter for CSRF protection
      const state = crypto.randomUUID();
      params.set("state", state);
      // Store state to verify on redirect (optional, for security)
      try {
        sessionStorage.setItem("logout_state", state);
      } catch {
        /* Storage may be disabled. */
      }

      const logoutUrl = `${endSessionEndpoint}?${params.toString()}`;

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
        const timeSinceLogout = Date.now() - parseInt(logoutTime as string, 10);
        // Prevent silent auth for 1 minute after logout
        if (timeSinceLogout < 60000) {
          return false;
        }
        // Clear the flag after timeout
        sessionStorage.removeItem("obiente_logout_time");
      }
    }

    // Ensure document.body is ready (critical for Chrome during hydration)
    if (import.meta.client && !document.body) {
      await new Promise<void>((resolve) => {
        if (document.body) {
          resolve();
        } else {
          const checkBody = () => {
            if (document.body) {
              resolve();
            } else {
              requestAnimationFrame(checkBody);
            }
          };
          requestAnimationFrame(checkBody);
        }
      });
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
        } catch {
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
          fetch().then(() => {
            resolve(true);
          });
        } else if (e.data?.type === "silent-auth-error") {
          cleanup();
          clearTimeout(timeout);
          resolve(false);
        }
      };

      window.addEventListener("message", messageHandler);

      // Append iframe first without src to avoid blocking during hydration
      document.body.appendChild(iframe);

      // Defer setting src until after the next frame to avoid blocking hydration
      // This prevents Chrome from freezing during hydration
      requestAnimationFrame(() => {
        iframe.src = "/auth/silent-check";
      });
    });
  };

  // Popup authentication support - track listeners to prevent duplicates
  let popupListenerActive = false;
  let messageListenerActive = false;

  const popupListener = (e: StorageEvent) => {
    if (e.key === "auth-completed") {
      fetch();
      // Clean up listener after use
      if (popupListenerActive) {
        window.removeEventListener("storage", popupListener);
        popupListenerActive = false;
      }
    }
  };

  // Message listener for OAuth errors and signup success from popup (not used for silent auth iframe)
  const messageListener = (e: MessageEvent) => {
    if (e.origin !== window.location.origin) return;

    if (e.data?.type === "oauth-error") {
      // Only handle popup errors, not silent auth iframe errors
      // Silent auth errors are handled in trySilentAuth message handler
      if (
        e.data.error === "login_required" ||
        e.data.error === "interaction_required" ||
        e.data.error === "no_session"
      ) {
        // Clean up listener before opening new popup
        if (messageListenerActive) {
          window.removeEventListener("message", messageListener);
          messageListenerActive = false;
        }
        // Open Zitadel login popup (only for explicit popup auth, not silent auth)
        popupLogin("/auth/oauth-login");
      }
    } else if (e.data?.type === "signup-success") {
      // Signup completed - user needs to login manually
      // Refresh the page to show login options
      // Clean up listener after use
      if (messageListenerActive) {
        window.removeEventListener("message", messageListener);
        messageListenerActive = false;
      }
      // Optionally show a notification or refresh auth state
      fetch();
    }
  };

  const popupLogin = (
    route: string = "/auth/oauth-login",
    size: { width?: number; height?: number } = {}
  ) => {
    if (!import.meta.client) return;

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

    // Only add listeners if not already active (prevent duplicates)
    if (!messageListenerActive) {
      window.addEventListener("message", messageListener);
      messageListenerActive = true;
    }
    if (!popupListenerActive) {
      window.addEventListener("storage", popupListener);
      popupListenerActive = true;
    }

    window.open(
      route,
      "_blank",
      `width=${width}, height=${height}, top=${top}, left=${left}, toolbar=no, location=no, directories=no, status=no, menubar=no, scrollbars=no, resizable=no, copyhistory=no`
    );
  };

  const popupSignup = (
    route: string = "/auth/oauth-signup",
    size: { width?: number; height?: number } = {}
  ) => {
    if (!import.meta.client) return;

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

    // Only add listeners if not already active (prevent duplicates)
    if (!messageListenerActive) {
      window.addEventListener("message", messageListener);
      messageListenerActive = true;
    }
    if (!popupListenerActive) {
      window.addEventListener("storage", popupListener);
      popupListenerActive = true;
    }

    const popup = window.open(
      route,
      "zitadel-signup",
      `width=${width}, height=${height}, top=${top}, left=${left}, toolbar=no, location=no, directories=no, status=no, menubar=no, scrollbars=no, resizable=no, copyhistory=no`
    );

    if (!popup || popup.closed || typeof popup.closed === "undefined") {
      console.error(
        "[Auth] Popup blocked or failed to open - please allow popups for this site"
      );
      alert(
        "Please allow popups for this site to sign up. The popup was blocked by your browser."
      );
      // Clean up listeners if popup failed
      if (messageListenerActive) {
        window.removeEventListener("message", messageListener);
        messageListenerActive = false;
      }
      if (popupListenerActive) {
        window.removeEventListener("storage", popupListener);
        popupListenerActive = false;
      }
      return;
    }
  };

  const validToken = () =>
    accessToken.value && tokenExpiry.value && Date.now() < tokenExpiry.value;

  const getAccessToken = async (
    forceRefresh = false
  ): Promise<string | null> => {
    if (import.meta.server)
      return sessionState.value?.secure?.access_token || null;
    if (loggingOut.value || !sessionState.value?.user) return null;
    if (!forceRefresh && validToken()) return accessToken.value;
    const startedGeneration = generation.value;
    await refreshAccessToken(forceRefresh);
    return startedGeneration === generation.value && validToken()
      ? accessToken.value
      : null;
  };

  // Share a refresh across all composable instances. Rotating refresh tokens
  // must not be exchanged concurrently by the layout, page and RPC transport.
  const refreshAccessToken = async (force = false): Promise<void> => {
    if (loggingOut.value || !sessionState.value?.user) return;
    const startedGeneration = generation.value;
    if (tokenRefreshRequest?.generation === startedGeneration)
      return tokenRefreshRequest.promise;
    if (!force && validToken()) return;
    const current = () =>
      generation.value === startedGeneration && !loggingOut.value;
    const save = (response: {
      accessToken?: string | null;
      expiresIn?: number;
    }) => {
      if (!current() || !response?.accessToken) return false;
      const expiresIn = response.expiresIn ?? 60;
      if (expiresIn <= 0) return false;
      accessToken.value = response.accessToken;
      tokenExpiry.value =
        Date.now() +
        Math.max(1, expiresIn - Math.min(30, expiresIn / 2)) * 1000;
      return true;
    };
    isRefreshing.value = true;
    const controller = new AbortController();
    clientAuthRequests.add(controller);
    const promise = (async () => {
      try {
        if (!force) {
          try {
            const response = await $fetch<{
              accessToken: string | null;
              expiresIn?: number;
            }>("/auth/token", {
              retry: false,
              timeout: 5000,
              signal: controller.signal,
            });
            if (save(response)) return;
          } catch {
            /* The refresh endpoint can recover an expired session token. */
          }
        }
        if (!current()) return;
        save(
          await $fetch<{ accessToken: string; expiresIn?: number }>(
            "/auth/refresh",
            {
              method: "POST",
              retry: false,
              timeout: 5000,
              signal: controller.signal,
            }
          )
        );
      } catch (error) {
        if (!current()) return;
        console.error("Failed to refresh access token:", error);
        if (statusCode(error) === 401) {
          await logout().catch((logoutError) =>
            console.error("Failed to end expired session:", logoutError)
          );
        }
      } finally {
        clientAuthRequests.delete(controller);
        if (current()) isRefreshing.value = false;
      }
    })();
    const request = { generation: startedGeneration, promise };
    tokenRefreshRequest = request;
    try {
      await promise;
    } finally {
      if (tokenRefreshRequest === request) tokenRefreshRequest = null;
    }
  };

  const waitForAuthenticatedClient = async (
    options: { timeoutMs?: number; intervalMs?: number } = {}
  ): Promise<boolean> => {
    const timeoutMs = options.timeoutMs ?? 4000;
    const intervalMs = options.intervalMs ?? 100;

    const hasSession = () => Boolean(sessionState.value?.user);

    await fetch();

    if (import.meta.server) {
      return hasSession();
    }

    const startedAt = Date.now();

    while (Date.now() - startedAt < timeoutMs) {
      if (!hasSession()) {
        await fetch();
      }

      if (hasSession()) {
        const token = await getAccessToken().catch((error) => {
          console.warn(
            "[useAuth] Failed to prime access token after login:",
            error
          );
          return null;
        });

        if (token) {
          return true;
        }
      }

      await new Promise((resolve) => setTimeout(resolve, intervalMs));
    }

    return hasSession();
  };

  if (import.meta.client && !sessionWatcherStarted) {
    sessionWatcherStarted = true;
    watch(
      () => sessionState.value,
      (newSession, oldSession) => {
        const identity = (session: UserSession | null) =>
          session?.user
            ? `${session.id || ""}:${
                session.user.sub ||
                (session.user as User & { id?: string }).id ||
                ""
              }`
            : "";
        if (identity(newSession) !== identity(oldSession)) {
          invalidateToken();
          if (oldSession?.user) orgStore.reset();
        }
        if (newSession?.user && !loggingOut.value && !validToken()) {
          const scheduledGeneration = generation.value;
          requestAnimationFrame(() => {
            if (generation.value === scheduledGeneration)
              void refreshAccessToken();
          });
        }
      },
      { flush: "sync" }
    );
  }

  const initializeAuth = () => {
    if (import.meta.client) {
      if (clientAuthInitialized) {
        return;
      }
      clientAuthInitialized = true;

      requestAnimationFrame(() => {
        fetch().catch((err) => {
          console.error("[useAuth] fetch() error:", err);
        });

        if (!sessionState.value || !user.value) {
          setTimeout(() => {
            trySilentAuth()
              .then((success) => {
                if (!success) {
                  // Silent auth failed, but don't auto-open popup here
                  // Let the middleware handle it if needed
                }
              })
              .catch((err) => {
                console.error("[useAuth] trySilentAuth error:", err);
              });
          }, 500);
        }

        if (!clientTokenCheckInterval) {
          clientTokenCheckInterval = setInterval(() => {
            if (tokenExpiry.value && Date.now() >= tokenExpiry.value) {
              refreshAccessToken(true).catch(console.error);
            }
          }, 60 * 1000);
        }
      });
    }
  };

  if (import.meta.client) {
    if (!clientAuthInitScheduled && !clientAuthInitialized) {
      clientAuthInitScheduled = true;
      // Preserve the server session throughout hydration. A timer can run while
      // an async layout is still hydrating and change its entire navigation.
      onNuxtReady(initializeAuth);
    }
  }

  // Create computed properties lazily to avoid triggering during initialization
  const currentOrganization = computed(() => {
    return orgStore.currentOrg;
  });
  const organizations = computed(() => {
    return orgStore.orgs;
  });
  const currentOrganizationId = computed(() => {
    return orgStore.currentOrgId;
  });

  const authObject = reactive({
    user: user,
    currentOrganization,
    organizations,
    currentOrganizationId,
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
    waitForAuthenticatedClient,
  });

  return authObject;
};
