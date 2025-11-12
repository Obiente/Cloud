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
    
    // Cache for disableAuth to avoid fetching on every request
    let cachedDisableAuth: boolean | null = null;
    let disableAuthFetchPromise: Promise<boolean> | null = null;

    // Function to fetch disableAuth from API (public endpoint, no auth needed)
    const fetchDisableAuth = async (): Promise<boolean> => {
      if (cachedDisableAuth !== null) {
        return cachedDisableAuth;
      }
      
      if (disableAuthFetchPromise) {
        const result = await disableAuthFetchPromise;
        return result ?? false;
      }
      
      disableAuthFetchPromise = (async (): Promise<boolean> => {
        try {
          // Create a transport without auth for the public config endpoint
          // Use internal API host for server-side (Docker internal networking)
          let apiHost = (config.apiHostInternal as string) || config.public.apiHost;
          let publicTransport = createConnectTransport({
            baseUrl: apiHost,
            httpVersion: "1.1",
            useBinaryFormat: false,
            defaultTimeoutMs: 5000, // 5 seconds timeout
          });
          
          const { AuthService } = await import("@obiente/proto");
          const { createClient } = await import("@connectrpc/connect");
          let client = createClient(AuthService, publicTransport);
          
          let publicConfig;
          try {
            publicConfig = await client.getPublicConfig({});
          } catch (err: any) {
            // If internal API fails and we have a fallback, try public API
            if (config.apiHostInternal && apiHost === (config.apiHostInternal as string)) {
              console.warn(`[Server Transport] Internal API (${apiHost}) failed (${err?.code || err?.message}), trying public API as fallback`);
              apiHost = config.public.apiHost;
              publicTransport = createConnectTransport({
                baseUrl: apiHost,
                httpVersion: "1.1",
                useBinaryFormat: false,
                defaultTimeoutMs: 5000,
              });
              client = createClient(AuthService, publicTransport);
              publicConfig = await client.getPublicConfig({});
            } else {
              throw err;
            }
          }
          
          const result = publicConfig.disableAuth ?? false;
          cachedDisableAuth = result;
          return result;
        } catch (err) {
          console.warn("[Server Transport] Failed to fetch public config, defaulting to auth required:", err);
          cachedDisableAuth = false;
          return false;
        } finally {
          disableAuthFetchPromise = null;
        }
      })();
      
      return await disableAuthFetchPromise;
    };

    // Function to get the authentication token from the session
    // Using a function so it's evaluated on each request to get the latest token
    const getToken = async (): Promise<string | undefined> => {
      // Check if auth is disabled (development mode)
      const disableAuth = await fetchDisableAuth();

      if (disableAuth) {
        // Return dummy token when auth is disabled - API will ignore it and use mock user
        return "dev-dummy-token";
      }

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

      // Only warn if auth is not disabled
      console.warn("No token available for SSR request");
        return undefined;
    };

    const authInterceptor = createAuthInterceptor(getToken);
    // Use internal API host for server-side (Docker internal networking)
    const apiHost = config.apiHostInternal || config.public.apiHost;
    const transport: Transport = createConnectTransport({
      baseUrl: apiHost,
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
