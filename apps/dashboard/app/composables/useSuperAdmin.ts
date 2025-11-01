import { Code, ConnectError } from "@connectrpc/connect";
import type { GetOverviewResponse } from "@obiente/proto";
import { SuperadminService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";

interface SuperAdminState {
  overview: GetOverviewResponse | null;
  loading: boolean;
  initialized: boolean;
  allowed: boolean | null;
  error: string | null;
}

export const useSuperAdmin = () => {
  const state = useState<SuperAdminState>("superadmin-state", () => ({
    overview: null,
    loading: false,
    initialized: false,
    allowed: null,
    error: null,
  }));

  const client = useConnectClient(SuperadminService);

  const fetchOverview = async (force = false) => {
    if (state.value.loading) return state.value.overview;
    if (state.value.initialized && !force) return state.value.overview;

    state.value.loading = true;
    state.value.error = null;

    try {
      const response = await client.getOverview({});
      state.value.overview = response;
      state.value.allowed = true;
      return response;
    } catch (err) {
      if (err instanceof ConnectError && err.code === Code.PermissionDenied) {
        state.value.allowed = false;
        state.value.overview = null;
        state.value.error = null;
      } else {
        state.value.error = err instanceof Error ? err.message : String(err);
      }
      return null;
    } finally {
      state.value.initialized = true;
      state.value.loading = false;
    }
  };

  const reset = () => {
    state.value = {
      overview: null,
      loading: false,
      initialized: false,
      allowed: null,
      error: null,
    };
  };

  const counts = computed(() => state.value.overview?.counts);

  return {
    state,
    overview: computed(() => state.value.overview),
    counts,
    allowed: computed(() => state.value.allowed),
    loading: computed(() => state.value.loading),
    initialized: computed(() => state.value.initialized),
    error: computed(() => state.value.error),
    fetchOverview,
    reset,
  };
};

