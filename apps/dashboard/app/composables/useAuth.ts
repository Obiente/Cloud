import { appendResponseHeader } from "h3";
import type { User, Organization, UserSession } from "@obiente/types";

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
  const currentOrganization = ref<Organization | null>(null);
  const isAuthenticated = computed(() => sessionState.value && user.value);
  const isLoading = ref(false);

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
    currentOrganization.value = null;
    authReadyState.value = false;
  };

  // Switch organization
  const switchOrganization = async (organizationId: string) => {
    try {
      // TODO: Implement organization switching
      // const response = await $fetch(`/api/organizations/${organizationId}/switch`, {
      //   method: 'POST'
      // });
      // currentOrganization.value = response.organization;

      console.log("Switching to organization:", organizationId);
    } catch (error) {
      console.error("Failed to switch organization:", error);
      throw error;
    }
  };

  // Popup authentication support
  const popupListener = (e: StorageEvent) => {
    if (e.key === "auth-completed") {
      fetch();
      window.removeEventListener("storage", popupListener);
    }
  };

  const popupLogin = (
    route: string = "/auth/login",
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

    window.open(
      route,
      "_blank",
      `width=${width}, height=${height}, top=${top}, left=${left}, toolbar=no, location=no, directories=no, status=no, menubar=no, scrollbars=no, resizable=no, copyhistory=no`
    );
    window.addEventListener("storage", popupListener);
  };

  // Get the current access token
  const getAccessToken = async (
    forceRefresh = false
  ): Promise<string | null> => {
    // If we're on the server, we can access the secure token directly
    if (import.meta.server) {
      return sessionState.value?.secure?.access_token || null;
    }

    // On client side, we need to use our cached token or fetch it
    if (
      !accessToken.value ||
      forceRefresh ||
      (tokenExpiry.value && Date.now() >= tokenExpiry.value)
    ) {
      await refreshAccessToken(forceRefresh);
    }

    return accessToken.value;
  };

  // Refresh the access token
  const refreshAccessToken = async (force = false): Promise<void> => {
    try {
      // Prevent multiple simultaneous refresh attempts
      if (isRefreshing.value && !force) {
        return;
      }

      isRefreshing.value = true;

      // If token is not expired and force is false, just get the current token
      if (!force && tokenExpiry.value && Date.now() < tokenExpiry.value) {
        const response = await $fetch<{ accessToken: string; expiresIn?: number }>("/auth/token");
        if (response.accessToken) {
          accessToken.value = response.accessToken;

          // Update expiry time from expiresIn field
          if (response.expiresIn) {
            // Convert to milliseconds and subtract 30 seconds buffer
            tokenExpiry.value = Date.now() + response.expiresIn * 1000 - 30000;
          }
        }
      } else {
        // Token is expired or force refresh, use refresh endpoint
        const response = await $fetch<{
          accessToken: string;
          expiresIn?: number;
        }>("/auth/refresh", {
          method: "POST",
        });

        if (response.accessToken) {
          accessToken.value = response.accessToken;

          // Calculate expiry time from expiresIn field
          if (response.expiresIn) {
            // Convert to milliseconds and subtract 30 seconds buffer
            tokenExpiry.value = Date.now() + response.expiresIn * 1000 - 30000;
          }
        }
      }
    } catch (error) {
      console.error("Failed to refresh access token:", error);
      accessToken.value = null;
      tokenExpiry.value = null;

      // If authentication failed, log out
      if (
        error &&
        typeof error === "object" &&
        "statusCode" in error &&
        error.statusCode === 401
      ) {
        logout();
      }
    } finally {
      isRefreshing.value = false;
    }
  };

  // Update token when session changes
  watch(
    () => sessionState.value,
    async (newSession) => {
      if (newSession) {
        // When session changes, fetch the token from the token endpoint
        try {
          // Only on client side
          if (import.meta.client) {
            const response = await $fetch<{ accessToken: string; expiresIn?: number }>(
              "/auth/token"
            );
            if (response.accessToken) {
              accessToken.value = response.accessToken;

              // Update expiry time from expiresIn field
              if (response.expiresIn) {
                // Convert to milliseconds and subtract 30 seconds buffer
                tokenExpiry.value = Date.now() + response.expiresIn * 1000 - 30000;
              }
            }
          }
        } catch (e) {
          console.error("Failed to fetch token after session update:", e);
        }
      } else {
        accessToken.value = null;
        tokenExpiry.value = null;
      }
    },
    { immediate: true }
  );

  // Initialize auth state
  onMounted(() => {
    fetch();

    // Set up token refresh before expiration
    if (import.meta.client) {
      const tokenCheckInterval = setInterval(() => {
        if (tokenExpiry.value && Date.now() >= tokenExpiry.value) {
          refreshAccessToken(true).catch(console.error);
        }
      }, 60 * 1000);

      // Clean up interval on component unmount
      onUnmounted(() => {
        clearInterval(tokenCheckInterval);
      });
    }
  });

  return reactive({
    // State
    user: user,
    currentOrganization: readonly(currentOrganization),
    session: readonly(sessionState),
    ready: computed(() => authReadyState.value),
    isAuthenticated,
    isLoading: readonly(isLoading),

    // Methods
    fetch,
    logout,
    switchOrganization,
    getCurrentUser: fetch,
    popupLogin,

    // Token management
    getAccessToken,
    refreshAccessToken,
  });
};
