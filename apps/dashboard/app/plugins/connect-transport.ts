import { createTransport } from "~/lib/transport";

// This plugin runs on the client and creates a transport instance using
// the public runtime config. It then injects it as `$transport` on the Nuxt app.
export default defineNuxtPlugin(async (nuxtApp) => {
  const config = useRuntimeConfig();

  // Function to get the authentication token from the session
  // Using a function so it's evaluated on each request to get the latest token
  const getToken = async (): Promise<string | undefined> => {
    if (import.meta.server) {
      // On server-side, get the token from the event
      const event = useRequestEvent();
      if (!event) return undefined;

      // Try to get the token directly from the cookie first (simpler approach)
      const { AUTH_COOKIE_NAME } = await import("../../server/utils/auth");
      const token = getCookie(event, AUTH_COOKIE_NAME);

      if (token && typeof token === "string" && token.trim() !== "") {
        return token;
      }

      // Fallback to session if cookie approach fails
      try {
        const { getUserSession } = await import("../../server/utils/session");
        const session = await getUserSession(event);
        const sessionToken = session?.secure?.access_token;

        if (
          sessionToken &&
          typeof sessionToken === "string" &&
          sessionToken.trim() !== ""
        ) {
          return sessionToken;
        }
      } catch (e) {
        console.error("Failed to get server-side token from session:", e);
      }

      console.warn("No token available for SSR request");
      return undefined;
    } else {
      // On client-side, use the auth composable to get the token
      const { useAuth } = await import("~/composables/useAuth");
      const auth = useAuth();

      // This will handle token refresh if needed
      const token = await auth.getAccessToken();
      return token || undefined;
    }
  };

  return {
    provide: {
      connect: createTransport(
        import.meta.client
          ? config.public.requestHost + "/api"
          : config.public.apiBaseUrl,
        getToken
      ),
    },
  };
});
