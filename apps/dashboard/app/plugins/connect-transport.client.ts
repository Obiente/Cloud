import { createAuthInterceptor, createWebTransport } from "~/lib/transport";
import { watch } from "vue";
import { useAuth } from "~/composables/useAuth";

// Client-side plugin
export default defineNuxtPlugin({
  name: "connect-transport-client",
  dependsOn: ["pinia"],
  setup() {
    const config = useRuntimeConfig();

    // Initialize inside the app context after the store plugin is ready.
    const auth = useAuth();

    // useAuth owns token lifetime. Share pending reads without extending expiry.
    let tokenFetchPromise: Promise<string | undefined> | null = null;
    let generation = 0;

    // Reject pending reads only when the identity changes. A normal session
    // refresh returns a new object for the same account and must not strip the
    // token from requests already waiting for it.
    watch(
      () =>
        JSON.stringify([
          auth.session?.id,
          auth.session?.user?.sub ||
            (auth.session?.user as { id?: string } | undefined)?.id,
        ]),
      () => {
        generation++;
        tokenFetchPromise = null;
      },
      { flush: "sync" }
    );

    const getToken = async (): Promise<string | undefined> => {
      // A bounded loop leaves no background timer behind when auth is unavailable.
      for (let attempt = 0; !auth.ready && attempt < 10; attempt++) {
        await new Promise<void>((resolve) => setTimeout(resolve, 50));
      }
      if (tokenFetchPromise) return tokenFetchPromise;
      const startedGeneration = generation;
      const request = (async () => {
        try {
          const token = await auth.getAccessToken();
          if (generation !== startedGeneration) return undefined;
          return token || undefined;
        } catch (error) {
          if (generation === startedGeneration)
            console.error(
              "[Connect Transport] Error getting access token:",
              error
            );
          return undefined;
        }
      })();
      tokenFetchPromise = request;
      try {
        return await request;
      } finally {
        if (tokenFetchPromise === request) tokenFetchPromise = null;
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
