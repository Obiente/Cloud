import { createAuthInterceptor, createWebTransport } from "~/lib/transport";

// Client-side plugin
export default defineNuxtPlugin({
  name: 'connect-transport-client',
  enforce: 'pre', // Run early
  setup(nuxtApp) {
    const config = useRuntimeConfig();

    // Function to get the authentication token from the session
    // Using a function so it's evaluated on each request to get the latest token
    const getToken = async (): Promise<string | undefined> => {
      // On client-side, use the auth composable to get the token
      const { useAuth } = await import("~/composables/useAuth");
      const auth = useAuth();

      // This will handle token refresh if needed
      const token = await auth.getAccessToken();
      return token || undefined;
    };

    const authInterceptor = createAuthInterceptor(getToken);
    const transport = createWebTransport(config.public.apiHost, authInterceptor);

    return {
      provide: {
        connect: transport,
      },
    };
  },
});

