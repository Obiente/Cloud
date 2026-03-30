import { createAuthInterceptor, createWebTransport } from "~/lib/transport";
import { watch } from "vue";

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

    // Invalidate transport-layer cache whenever the session changes so we never
    // serve a stale token after logout or a token rotation.
    watch(
      () => auth.session,
      () => {
        cachedToken = null;
        tokenExpiry = null;
        tokenFetchPromise = null;
      }
    );

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
          // Cache for 30 seconds — conservative enough to batch rapid requests
          // while ensuring a rotation or logout invalidates quickly.
          cachedToken = token;
          tokenExpiry = Date.now() + 30000;
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
