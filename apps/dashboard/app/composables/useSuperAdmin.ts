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

const overviewRequests = new WeakMap<object, Promise<GetOverviewResponse | null>>();

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

  const app = useNuxtApp();
  const fetchOverview = (force = false): Promise<GetOverviewResponse | null> => {
    const pending = overviewRequests.get(app);
    if (pending) return pending;
    if (state.value.initialized && !force) return Promise.resolve(state.value.overview);
    const target = state.value;
    const request = (async () => {
      target.loading = true;
      target.error = null;
      try {
        const permissions = await client.getMySuperadminPermissions({});
        target.permissions = permissions.permissions || [];
        target.isFullSuperadmin = permissions.isFullSuperadmin || false;
        target.allowed = target.isFullSuperadmin || target.permissions.length > 0;
        const canReadOverview = target.isFullSuperadmin || target.permissions.some(permission =>
          permission === "superadmin.overview.read" || permission === "superadmin.overview.*" || permission === "superadmin.*");
        target.overview = canReadOverview ? await client.getOverview({}) : null;
        target.initialized = true;
        return target.overview;
      } catch (err) {
        target.initialized = false;
        if (err instanceof ConnectError && err.code === Code.PermissionDenied) {
          target.allowed = false;
          target.permissions = [];
          target.isFullSuperadmin = false;
          target.overview = null;
          target.initialized = true;
        } else {
          target.error = "Administrator data is temporarily unavailable. Please try again.";
        }
        return null;
      } finally {
        target.loading = false;
      }
    })();
    overviewRequests.set(app, request);
    void request.finally(() => { if (overviewRequests.get(app) === request) overviewRequests.delete(app); });
    return request;
  };

  const reset = () => {
    overviewRequests.delete(app);
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
        throw new Error((err as Error).message);
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
        throw new Error((err as Error).message);
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
        throw new Error((err as Error).message);
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

