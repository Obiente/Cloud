import { AuthService } from "@obiente/proto";
import { createClient } from "@connectrpc/connect";

export const useConfig = () => {
  const configState = useState<{
    billingEnabled: boolean | null;
    selfHosted: boolean | null;
    disableAuth: boolean | null;
    loading: boolean;
    error: Error | null;
  }>("app-config", () => ({
    billingEnabled: null,
    selfHosted: null,
    disableAuth: null,
    loading: false,
    error: null,
  }));

  const fetchConfig = async () => {
    if (configState.value.loading) {
      return; // Already fetching
    }

    try {
      configState.value.loading = true;
      configState.value.error = null;

      // Create a public client without auth for the public config endpoint
      const config = useRuntimeConfig();
      let publicTransport;
      let apiHost: string = config.public.apiHost;
      
      // Use different transports for client vs server
      if (import.meta.server) {
        const { createConnectTransport } = await import("@connectrpc/connect-node");
        // Use internal API host for server-side (Docker internal networking)
        // Try internal first, fallback to public if internal fails
        apiHost = (config.apiHostInternal as string) || config.public.apiHost;
        publicTransport = createConnectTransport({
          baseUrl: apiHost,
          httpVersion: "1.1",
          useBinaryFormat: false,
          // Longer timeout for internal API connections (may need time to resolve service name)
          defaultTimeoutMs: 10000, // 10 seconds
        });
      } else {
        const { createConnectTransport } = await import("@connectrpc/connect-web");
        apiHost = config.public.apiHost;
        publicTransport = createConnectTransport({
          baseUrl: apiHost,
          useBinaryFormat: false,
        });
      }
      
      const publicClient = createClient(AuthService, publicTransport);
      
      // Try to fetch config with fallback for server-side
      let response;
      try {
        response = await publicClient.getPublicConfig({});
      } catch (err: any) {
        // On server-side, if internal API fails, try public API as fallback
        if (import.meta.server && config.apiHostInternal && apiHost === (config.apiHostInternal as string)) {
          console.warn(`[Config] Internal API (${apiHost}) failed, trying public API as fallback:`, err?.code || err?.message);
          const { createConnectTransport } = await import("@connectrpc/connect-node");
          const fallbackTransport = createConnectTransport({
            baseUrl: config.public.apiHost,
            httpVersion: "1.1",
            useBinaryFormat: false,
            defaultTimeoutMs: 10000, // 10 seconds
          });
          const fallbackClient = createClient(AuthService, fallbackTransport);
          response = await fallbackClient.getPublicConfig({});
        } else {
          throw err;
        }
      }
      
      configState.value.billingEnabled = response.billingEnabled ?? true;
      configState.value.selfHosted = response.selfHosted ?? false;
      configState.value.disableAuth = response.disableAuth ?? false;
    } catch (err) {
      console.error("Failed to fetch public config:", err);
      configState.value.error = err instanceof Error ? err : new Error(String(err));
      // Set defaults on error
      configState.value.billingEnabled = true;
      configState.value.selfHosted = false;
      configState.value.disableAuth = false;
    } finally {
      configState.value.loading = false;
    }
  };

  // Fetch config on first access if not already loaded
  if (configState.value.billingEnabled === null && !configState.value.loading) {
    fetchConfig();
  }

  return {
    billingEnabled: computed(() => configState.value.billingEnabled ?? true),
    selfHosted: computed(() => configState.value.selfHosted ?? false),
    disableAuth: computed(() => configState.value.disableAuth ?? false),
    loading: computed(() => configState.value.loading),
    error: computed(() => configState.value.error),
    fetchConfig,
  };
};

