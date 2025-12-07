import { Code, ConnectError } from "@connectrpc/connect";
import type { GetOverviewResponse } from "@obiente/proto";
import { SuperadminService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";

interface SuperAdminState {
  overview: GetOverviewResponse | null;
  loading: boolean;
  initialized: boolean;
  allowed: boolean | null;
  permissions: string[]; // List of permission IDs the user has
  isFullSuperadmin: boolean; // true if user is a full superadmin (email-based)
  error: string | null;
}

export const useSuperAdmin = () => {
  const state = useState<SuperAdminState>("superadmin-state", () => ({
    overview: null,
    loading: false,
    initialized: false,
    allowed: null,
    permissions: [],
    isFullSuperadmin: false,
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
      // Fetch permissions first to determine access
      const permsResponse = await client.getMySuperadminPermissions({});
      state.value.permissions = permsResponse.permissions || [];
      state.value.isFullSuperadmin = permsResponse.isFullSuperadmin || false;
      
      // Helper function to check permission (defined inline to avoid hoisting issues)
      const checkPerm = (permission: string): boolean => {
        if (state.value.isFullSuperadmin) {
          return true;
        }
        const perms = state.value.permissions;
        for (const perm of perms) {
          if (perm === permission) {
            return true;
          }
          if (perm.endsWith(".*")) {
            const prefix = perm.slice(0, -2);
            if (permission.startsWith(prefix + ".")) {
              return true;
            }
          }
        }
        return false;
      };
      
      // If user has any superadmin permissions, they're allowed
      // Full superadmins are always allowed
      if (state.value.isFullSuperadmin || state.value.permissions.length > 0) {
        state.value.allowed = true;
        
        // Fetch overview if user has permission
        if (state.value.isFullSuperadmin || checkPerm("superadmin.overview.read")) {
      const response = await client.getOverview({});
      state.value.overview = response;
        } else {
          state.value.overview = null;
        }
      } else {
        state.value.allowed = false;
        state.value.overview = null;
      }
      
      state.value.error = null;
      return state.value.overview;
    } catch (err) {
      console.error("[SuperAdmin] Failed to fetch overview:", err);
      
      if (err instanceof ConnectError) {
        // Handle unauthenticated errors (no token on server side)
        if (err.code === Code.Unauthenticated) {
          // On server side, we can't check auth, so don't set allowed to false
          // This will be checked again on client side
          if (import.meta.server) {
            state.value.allowed = null; // Keep as null, will be checked on client
            state.value.error = null;
            return null;
          } else {
            // On client side, unauthenticated means not allowed
            state.value.allowed = false;
            state.value.overview = null;
            state.value.error = null;
            return null;
          }
        }
        
        if (err.code === Code.PermissionDenied) {
          // User is not a superadmin
          state.value.allowed = false;
          state.value.overview = null;
          state.value.error = null;
        } else if (err.code === Code.Internal) {
          // Internal server error - log it but don't set allowed to false if already initialized
          console.error("[SuperAdmin] Internal server error:", err.message);
          if (!state.value.initialized) {
            // First time fetch failed with 500 - set allowed to false
            state.value.allowed = false;
            state.value.permissions = [];
            state.value.isFullSuperadmin = false;
          }
          // Otherwise, keep the existing allowed state (don't change it)
          state.value.error = err.message || "Internal server error";
        } else {
          // Other ConnectError codes - preserve previous allowed state if initialized
          console.error("[SuperAdmin] Unexpected ConnectError:", err);
          if (!state.value.initialized) {
            // First time fetch failed - set allowed to false to prevent showing sidebar until verified
            state.value.allowed = false;
            state.value.permissions = [];
            state.value.isFullSuperadmin = false;
          }
          // Otherwise, keep the existing allowed state (don't change it)
          state.value.error = err.message || String(err);
        }
      } else {
        // Network errors or other errors - preserve previous allowed state if initialized
        // This prevents network errors from hiding the sidebar for superadmins
        console.error("[SuperAdmin] Unexpected error:", err);
        if (!state.value.initialized) {
          // First time fetch failed - set allowed to false to prevent showing sidebar until verified
          state.value.allowed = false;
          state.value.permissions = [];
          state.value.isFullSuperadmin = false;
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
      permissions: [],
      isFullSuperadmin: false,
      error: null,
    };
  };

  // Check if user has a specific permission (supports wildcards)
  const hasPermission = (permission: string): boolean => {
    if (state.value.isFullSuperadmin) {
      return true; // Full superadmins have all permissions
    }
    
    const perms = state.value.permissions;
    for (const perm of perms) {
      // Exact match
      if (perm === permission) {
        return true;
      }
      // Wildcard match (e.g., "superadmin.vps.*" matches "superadmin.vps.read")
      if (perm.endsWith(".*")) {
        const prefix = perm.slice(0, -2); // Remove ".*"
        if (permission.startsWith(prefix + ".")) {
          return true;
        }
      }
      // Reverse wildcard match (e.g., "superadmin.vps.read" matches "superadmin.vps.*")
      if (permission.endsWith(".*")) {
        const prefix = permission.slice(0, -2);
        if (perm.startsWith(prefix + ".")) {
          return true;
        }
      }
    }
    return false;
  };

  const counts = computed(() => state.value.overview?.counts);

  const listNodes = async (filters?: {
    role?: string;
    availability?: string;
    status?: string;
    region?: string;
  }) => {
    try {
      const response = await client.listNodes({
        role: filters?.role ? filters.role : undefined,
        availability: filters?.availability ? filters.availability : undefined,
        status: filters?.status ? filters.status : undefined,
        region: filters?.region ? filters.region : undefined,
      });
      return response;
    } catch (err) {
      if (err instanceof ConnectError) {
        throw new Error(err.message);
      }
      throw err;
    }
  };

  const getNode = async (nodeId: string) => {
    try {
      const response = await client.getNode({ nodeId });
      return response;
    } catch (err) {
      if (err instanceof ConnectError) {
        throw new Error(err.message);
      }
      throw err;
    }
  };

  const updateNodeConfig = async (config: {
    nodeId: string;
    subdomain?: string;
    useNodeSpecificDomains?: boolean;
    serviceDomainPattern?: string;
    region?: string;
    maxDeployments?: number;
    customLabels?: Record<string, string>;
  }) => {
    try {
      const response = await client.updateNodeConfig({
        nodeId: config.nodeId,
        subdomain: config.subdomain,
        useNodeSpecificDomains: config.useNodeSpecificDomains,
        serviceDomainPattern: config.serviceDomainPattern,
        region: config.region,
        maxDeployments: config.maxDeployments,
        customLabels: config.customLabels,
      });
      return response;
    } catch (err) {
      if (err instanceof ConnectError) {
        throw new Error(err.message);
      }
      throw err;
    }
  };

  return {
    state,
    overview: computed(() => state.value.overview),
    counts,
    allowed: computed(() => state.value.allowed),
    permissions: computed(() => state.value.permissions),
    isFullSuperadmin: computed(() => state.value.isFullSuperadmin),
    loading: computed(() => state.value.loading),
    initialized: computed(() => state.value.initialized),
    error: computed(() => state.value.error),
    fetchOverview,
    reset,
    hasPermission,
    listNodes,
    getNode,
    updateNodeConfig,
  };
};

