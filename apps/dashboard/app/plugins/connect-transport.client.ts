import { createAuthInterceptor, createWebTransport } from "~/lib/transport";

// Client-side plugin
export default defineNuxtPlugin({
  name: "connect-transport-client",
  async setup(nuxtApp) {
    const config = useRuntimeConfig();

    // Import and cache the auth composable once
    const { useAuth } = await import("~/composables/useAuth");
    const auth = useAuth();

    // Cache token and expiry to avoid repeated fetches
    let cachedToken: string | null = null;
    let tokenExpiry: number | null = null;
    let tokenFetchPromise: Promise<string | null> | null = null;

    // Function to get the authentication token from the session
    // With caching to prevent repeated fetches
    const getToken = async (): Promise<string | undefined> => {
      try {
        // Wait for auth to be ready if it's still loading
        if (!auth.ready) {
          // Give it a moment to initialize (but with a timeout)
          await Promise.race([
            new Promise<void>((resolve) => {
              const checkReady = () => {
                if (auth.ready) {
                  resolve();
                } else {
                  setTimeout(checkReady, 50);
                }
              };
              checkReady();
            }),
            new Promise<void>((resolve) => setTimeout(resolve, 500)), // Max 500ms wait
          ]);
        }

        // Check if we have a valid cached token
        if (cachedToken && tokenExpiry && Date.now() < tokenExpiry) {
          return cachedToken;
        }

        // If there's already a token fetch in progress, wait for it
        if (tokenFetchPromise) {
          const token = await tokenFetchPromise;
          return token || undefined;
        }

        // Fetch token (this handles refresh if needed)
        tokenFetchPromise = auth.getAccessToken();
        const token = await tokenFetchPromise;
        tokenFetchPromise = null;

        if (token) {
          // Cache the token - we'll use a conservative expiry (60 seconds)
          // The auth composable already handles proper expiry, but we add
          // an extra layer of caching here to prevent excessive calls
          cachedToken = token;
          tokenExpiry = Date.now() + 60000; // Cache for 60 seconds
        }

        if (!token) {
          console.warn("[Connect Transport] No access token available - user may need to log in");
        }
        return token || undefined;
      } catch (error) {
        tokenFetchPromise = null;
        console.error("[Connect Transport] Error getting access token:", error);
        return undefined;
      }
    };

    const authInterceptor = createAuthInterceptor(getToken);
    const transport = createWebTransport(
      config.public.apiHost,
      authInterceptor
    );

    return {
      provide: {
        connect: transport,
      },
    };
  },
});
