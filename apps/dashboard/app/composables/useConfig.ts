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
      
      // Use different transports for client vs server
      if (import.meta.server) {
        const { createConnectTransport } = await import("@connectrpc/connect-node");
        publicTransport = createConnectTransport({
          baseUrl: config.public.apiHost,
          httpVersion: "1.1",
          useBinaryFormat: false,
        });
      } else {
        const { createConnectTransport } = await import("@connectrpc/connect-web");
        publicTransport = createConnectTransport({
          baseUrl: config.public.apiHost,
          useBinaryFormat: false,
        });
      }
      
      const publicClient = createClient(AuthService, publicTransport);
      const response = await publicClient.getPublicConfig({});
      
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

