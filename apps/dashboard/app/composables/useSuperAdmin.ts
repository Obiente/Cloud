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
    if (state.value.loading && !force) return state.value.overview;
    // If forcing, always fetch even if initialized
    if (state.value.initialized && !force) return state.value.overview;

    state.value.loading = true;
    state.value.error = null;

    try {
      const response = await client.getOverview({});
      state.value.overview = response;
      state.value.allowed = true;
      state.value.error = null;
      console.log("[SuperAdmin] Access granted, overview loaded successfully");
      return response;
    } catch (err) {
      console.error("[SuperAdmin] Failed to fetch overview:", err);
      
      if (err instanceof ConnectError && err.code === Code.PermissionDenied) {
        // User is not a superadmin
        console.log("[SuperAdmin] Permission denied - user is not a superadmin");
        state.value.allowed = false;
        state.value.overview = null;
        state.value.error = null;
      } else if (err instanceof ConnectError && err.code === Code.Internal) {
        // Internal server error - log it but don't set allowed to false if already initialized
        console.error("[SuperAdmin] Internal server error:", err.message);
        if (!state.value.initialized) {
          // First time fetch failed with 500 - set allowed to false
          state.value.allowed = false;
        }
        // Otherwise, keep the existing allowed state (don't change it)
        state.value.error = err.message || "Internal server error";
      } else {
        // Network errors or other errors - preserve previous allowed state if initialized
        // This prevents network errors from hiding the sidebar for superadmins
        console.error("[SuperAdmin] Unexpected error:", err);
        if (!state.value.initialized) {
          // First time fetch failed - set allowed to false to prevent showing sidebar until verified
          state.value.allowed = false;
        }
        // Otherwise, keep the existing allowed state (don't change it)
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

