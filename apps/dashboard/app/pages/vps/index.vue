<template>
  <OuiContainer size="full">
    <OuiStack gap="xl">
      <OuiFlex justify="between" align="start" wrap="wrap" gap="lg">
        <OuiStack gap="sm" class="max-w-xl">
          <OuiFlex align="center" gap="md">
            <OuiBox
              p="sm"
              rounded="xl"
              bg="accent-primary"
              class="bg-success/10 ring-1 ring-success/20"
            >
              <ServerIcon class="w-6 h-6 text-success" />
            </OuiBox>
            <OuiText as="h1" size="3xl" weight="bold"> VPS Instances </OuiText>
          </OuiFlex>
          <OuiText color="secondary" size="md">
            Provision and manage virtual private servers with full root access.
          </OuiText>
        </OuiStack>

        <OuiButton
          color="primary"
          class="gap-2 shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30 transition-all"
          @click="showCreateDialog = true"
        >
          <PlusIcon class="h-4 w-4" />
          <OuiText as="span" size="sm" weight="medium">New VPS</OuiText>
        </OuiButton>
      </OuiFlex>

      <!-- Error Alert -->
      <ErrorAlert
        v-if="listError"
        :error="listError"
        title="Failed to load VPS instances"
        hint="Please try refreshing the page. If the problem persists, contact support."
      />

      <!-- Filters -->
      <OuiCard variant="default" class="backdrop-blur-sm border border-border-muted/60">
        <OuiCardBody>
          <OuiGrid :cols="{ sm: 1, md: 3 }" gap="md">
            <OuiInput
              v-model="searchQuery"
              placeholder="Search by name..."
              clearable
            >
              <template #prefix>
                <MagnifyingGlassIcon class="h-4 w-4 text-secondary" />
              </template>
            </OuiInput>

            <OuiSelect
              v-model="statusFilter"
              :items="statusFilterOptions"
              placeholder="All Status"
            />

            <OuiSelect
              v-model="regionFilter"
              :items="regionOptions"
              placeholder="All Regions"
              clearable
            />
          </OuiGrid>
        </OuiCardBody>
      </OuiCard>

      <!-- Empty State -->
      <OuiStack
        v-if="filteredVPS.length === 0 && !isLoading"
        align="center"
        gap="lg"
        class="text-center py-20"
      >
        <OuiBox
          class="inline-flex items-center justify-center w-20 h-20 rounded-xl bg-surface-muted/50 ring-1 ring-border-muted"
        >
          <ServerIcon class="h-10 w-10 text-secondary" />
        </OuiBox>
        <OuiStack align="center" gap="sm">
          <OuiText as="h3" size="xl" weight="semibold" color="primary">
            No VPS instances found
          </OuiText>
          <OuiBox class="max-w-md">
            <OuiText color="secondary">
              {{
                searchQuery || statusFilter || regionFilter
                  ? "Try adjusting your filters to see more results."
                  : "Get started by creating your first VPS instance."
              }}
            </OuiText>
          </OuiBox>
          <OuiButton
            v-if="!searchQuery && !statusFilter && !regionFilter"
            color="primary"
            @click="showCreateDialog = true"
          >
            <PlusIcon class="h-4 w-4" />
            Create VPS Instance
          </OuiButton>
        </OuiStack>
      </OuiStack>

      <!-- Loading State with Skeleton Cards -->
      <OuiGrid v-if="isLoading && !vpsInstances" :cols="{ sm: 1, md: 2, lg: 3 }" gap="lg">
        <VPSCard
          v-for="i in 6"
          :key="i"
          :loading="true"
        />
      </OuiGrid>

      <!-- VPS Grid -->
      <OuiGrid v-if="filteredVPS.length > 0" :cols="{ sm: 1, md: 2, lg: 3 }" gap="lg">
        <VPSCard
          v-for="vps in filteredVPS"
          :key="vps.id"
          :vps="vps"
          @refresh="refreshVPS"
          @delete="handleDelete"
          @retry="handleRetry"
        />
      </OuiGrid>

      <!-- Create VPS Dialog -->
      <CreateVPSDialog
        v-model="showCreateDialog"
        :initial-values="retryVPSValues"
        @created="handleVPSCreated"
      />
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
  import { ref, computed, watch, onMounted, onUnmounted } from "vue";
  import {
    ServerIcon,
    PlusIcon,
    MagnifyingGlassIcon,
  } from "@heroicons/vue/24/outline";
  import { VPSService, VPSStatus, type VPSInstance } from "@obiente/proto";
  import { useConnectClient } from "~/lib/connect-client";
  import { useOrganizationId } from "~/composables/useOrganizationId";
  import { useDocumentVisibility } from "@vueuse/core";
  import ErrorAlert from "~/components/ErrorAlert.vue";
  import VPSCard from "~/components/vps/VPSCard.vue";
  import CreateVPSDialog from "~/components/vps/CreateVPSDialog.vue";
  import OuiSkeleton from "~/components/oui/Skeleton.vue";
  import { useToast } from "~/composables/useToast";

  definePageMeta({
    layout: "default",
    middleware: "auth",
  });

  const client = useConnectClient(VPSService);
  const organizationId = useOrganizationId();
  const { toast } = useToast();

  // Filters
  const searchQuery = ref("");
  const statusFilter = ref("");
  const regionFilter = ref("");
  const showCreateDialog = ref(false);
  const listError = ref<Error | null>(null);
  const retryVPSValues = ref<{
    name?: string;
    description?: string;
    region?: string;
    image?: number;
    size?: string;
  } | null>(null);

  const statusFilterOptions = [
    { label: "All Status", value: "" },
    { label: "Running", value: String(VPSStatus.RUNNING) },
    { label: "Stopped", value: String(VPSStatus.STOPPED) },
    { label: "Creating", value: String(VPSStatus.CREATING) },
    { label: "Starting", value: String(VPSStatus.STARTING) },
    { label: "Stopping", value: String(VPSStatus.STOPPING) },
    { label: "Rebooting", value: String(VPSStatus.REBOOTING) },
    { label: "Failed", value: String(VPSStatus.FAILED) },
    { label: "Deleting", value: String(VPSStatus.DELETING) },
    { label: "Deleted", value: String(VPSStatus.DELETED) },
  ];

  const regionOptions = ref<Array<{ label: string; value: string }>>([]);

  // Fetch VPS instances
  const {
    data: vpsInstances,
    status,
    refresh: refreshVPS,
  } = await useClientFetch(
    () => `vps-list-${organizationId.value}`,
    async () => {
      try {
        const response = await client.listVPS({
          organizationId: organizationId.value || undefined,
          page: 1,
          perPage: 100,
        });
        return response.vpsInstances || [];
      } catch (error) {
        console.error("Failed to list VPS:", error);
        listError.value = error as Error;
        return [];
      }
    }
  );

  // Fetch regions for filter
  const { data: regions, error: regionsError } = await useClientFetch(
    () => "vps-regions",
    async () => {
      try {
        const response = await client.listVPSRegions({});
        return response.regions || [];
      } catch (error) {
        console.error("Failed to list regions:", error);
        throw error; // Re-throw to be handled by error state
      }
    }
  );

  // Update region options
  watch(regions, (newRegions) => {
    if (newRegions && newRegions.length > 0) {
      const availableRegions = newRegions.filter((r) => r.available);
      // If only one region, don't show filter (or show it as default)
      if (availableRegions.length === 1) {
        const region = availableRegions[0];
        if (region) {
          regionOptions.value = [
            { label: region.name, value: region.id },
          ];
          // Auto-select the single region
          if (!regionFilter.value) {
            regionFilter.value = region.id;
          }
        }
      } else {
        regionOptions.value = [
          { label: "All Regions", value: "" },
          ...availableRegions.map((r) => ({ label: r.name, value: r.id })),
        ];
      }
    } else {
      // No regions available - clear options
      regionOptions.value = [];
    }
  }, { immediate: true });

  // Filtered VPS instances
  const filteredVPS = computed(() => {
    if (!vpsInstances.value) return [];

    let filtered = [...vpsInstances.value];

    // Search filter
    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase();
      filtered = filtered.filter(
        (vps) =>
          vps.name?.toLowerCase().includes(query) ||
          vps.id?.toLowerCase().includes(query)
      );
    }

    // Status filter
    if (statusFilter.value) {
      filtered = filtered.filter(
        (vps) => String(vps.status) === statusFilter.value
      );
    }

    // Region filter
    if (regionFilter.value) {
      filtered = filtered.filter((vps) => vps.region === regionFilter.value);
    }

    return filtered;
  });

  const isLoading = computed(
    () => !vpsInstances.value && (status.value === "pending" || status.value === "idle")
  );

  // Refresh function
  const refreshVPSWithoutClearing = async () => {
    try {
      const response = await client.listVPS({
        organizationId: organizationId.value || undefined,
        page: 1,
        perPage: 100,
      });
      vpsInstances.value = response.vpsInstances || [];
      listError.value = null;
    } catch (error) {
      console.error("Failed to refresh VPS:", error);
    }
  };

  // Periodic refresh
  const hasActiveVPS = computed(() => {
    return (vpsInstances.value ?? []).some(
      (v) =>
        v.status === VPSStatus.CREATING ||
        v.status === VPSStatus.STARTING ||
        v.status === VPSStatus.STOPPING ||
        v.status === VPSStatus.REBOOTING
    );
  });

  const refreshIntervalMs = computed(() => (hasActiveVPS.value ? 5000 : 30000));
  const visibility = useDocumentVisibility();
  const isVisible = computed(() => visibility.value === "visible");
  const refreshIntervalId = ref<ReturnType<typeof setInterval> | null>(null);

  const setupRefreshInterval = () => {
    if (refreshIntervalId.value) {
      clearInterval(refreshIntervalId.value);
      refreshIntervalId.value = null;
    }

    if (isVisible.value && !listError.value) {
      refreshIntervalId.value = setInterval(async () => {
        if (isVisible.value && !listError.value) {
          await refreshVPSWithoutClearing();
        }
      }, refreshIntervalMs.value);
    }
  };

  watch([refreshIntervalMs, isVisible], () => {
    setupRefreshInterval();
  });

  onMounted(() => {
    setupRefreshInterval();
  });

  onUnmounted(() => {
    if (refreshIntervalId.value) {
      clearInterval(refreshIntervalId.value);
      refreshIntervalId.value = null;
    }
  });

  // Handlers
  const handleVPSCreated = async () => {
    showCreateDialog.value = false;
    retryVPSValues.value = null; // Clear retry values after creation
    // Wait a brief moment for backend to create the VPS record, then refresh
    // Use refreshVPSWithoutClearing to avoid clearing existing data during refresh
    await new Promise(resolve => setTimeout(resolve, 500));
    try {
      await refreshVPSWithoutClearing();
    } catch (error) {
      // If refresh fails, try again after a longer delay
      // The VPS might still be creating and not yet available
      console.warn("Initial refresh failed, retrying after delay:", error);
      await new Promise(resolve => setTimeout(resolve, 2000));
      try {
        await refreshVPSWithoutClearing();
      } catch (retryError) {
        console.error("Retry refresh also failed:", retryError);
        // Don't show error to user - periodic refresh will pick it up
      }
    }
  };

  const handleDelete = async (deletedVPS?: VPSInstance) => {
    // Optimistically update status to DELETING for immediate UI feedback
    if (deletedVPS && vpsInstances.value) {
      const index = vpsInstances.value.findIndex((v) => v.id === deletedVPS.id);
      if (index !== -1) {
        // Create a new object with updated status
        // Use type assertion since we're modifying a protobuf Message type
        vpsInstances.value[index] = {
          ...vpsInstances.value[index],
          status: VPSStatus.DELETING,
        } as VPSInstance;
      }
    }
    // Then refresh to get actual status from backend
    await refreshVPS();
  };

  const handleRetry = (vps: VPSInstance) => {
    // Extract values from failed VPS to pre-fill the form
    retryVPSValues.value = {
      name: vps.name || "",
      description: vps.description || "",
      region: vps.region || "",
      image: vps.image || undefined,
      size: vps.size || "",
    };
    showCreateDialog.value = true;
  };
</script>
