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
    popupLogin,

    getAccessToken,
    refreshAccessToken,
  });
};
