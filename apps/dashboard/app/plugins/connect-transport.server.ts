import { createConnectTransport } from "@connectrpc/connect-node";
import type { Transport } from "@connectrpc/connect";
import { getCookie } from "h3";
import { createAuthInterceptor } from "~/lib/transport";

// Server-side plugin for SSR
export default defineNuxtPlugin({
  name: "connect-transport-server",
  enforce: "pre", // Run early
  setup(nuxtApp) {
    const config = useRuntimeConfig();

    // Function to get the authentication token from the session
    // Using a function so it's evaluated on each request to get the latest token
    const getToken = async (): Promise<string | undefined> => {
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
    };

    const authInterceptor = createAuthInterceptor(getToken);
    const transport: Transport = createConnectTransport({
      baseUrl: config.public.apiHost,
      httpVersion: "1.1", // Use HTTP/1.1 (h2c not supported by connect-node)
      useBinaryFormat: false, // Use Connect Protocol (JSON) instead of gRPC
      interceptors: [authInterceptor],
    });

    return {
      provide: {
        connect: transport,
      },
    };
  },
});
