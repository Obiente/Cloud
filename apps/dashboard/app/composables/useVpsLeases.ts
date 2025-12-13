import { ref, readonly } from 'vue';
import { useConnectClient } from '~/lib/connect-client';
import { VPSService, type VPSLease } from '@obiente/proto';

export const useVpsLeases = () => {
  const client = useConnectClient(VPSService);
  const leases = ref<VPSLease[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  const fetchLeases = async (organizationId: string, vpsId?: string) => {
    if (!organizationId) {
      error.value = 'Organization ID is required';
      return;
    }

    loading.value = true;
    error.value = null;

    try {
      const response = await client.getVPSLeases({
        organizationId,
        vpsId: vpsId || undefined,
      });

      leases.value = response.leases || [];
    } catch (err: any) {
      error.value = err?.message || 'Failed to fetch leases';
      leases.value = [];
    } finally {
      loading.value = false;
    }
  };

  return {
    leases,
    loading: readonly(loading),
    error: readonly(error),
    fetchLeases,
  };
};
