<template>
  <OuiStack gap="xl">
    <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
      <OuiStack gap="xs">
        <OuiText tag="h1" size="3xl" weight="extrabold">Usage</OuiText>
        <OuiText color="muted">Current month resource consumption across organizations.</OuiText>
      </OuiStack>
      <OuiFlex gap="sm" wrap="wrap">
        <div class="w-72 max-w-full">
          <OuiInput
            v-model="search"
            type="search"
            placeholder="Search by organization name, ID, or month…"
            clearable
            size="sm"
          />
        </div>
        <OuiButton variant="ghost" size="sm" @click="refresh" :disabled="isLoading">
          <span class="flex items-center gap-2">
            <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': isLoading }" />
            Refresh
          </span>
        </OuiButton>
      </OuiFlex>
    </OuiFlex>

    <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
      <OuiCardBody class="p-0">
        <OuiTable
          :columns="tableColumns"
          :rows="tableRows"
          :empty-text="isLoading ? 'Loading usage…' : 'No usage entries match your search.'"
        >
          <template #cell-organization="{ value, row }">
            <div>
              <div class="font-medium text-text-primary">{{ value || "—" }}</div>
              <div class="text-xs font-mono text-text-tertiary mt-0.5">{{ row.organizationId }}</div>
            </div>
          </template>
          <template #cell-actions="{ row }">
            <div class="text-right">
              <OuiButton size="xs" variant="ghost" @click.stop="switchToOrg(row.organizationId)">
                View org
              </OuiButton>
            </div>
          </template>
        </OuiTable>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { ArrowPathIcon } from "@heroicons/vue/24/outline";
import { computed, ref } from "vue";
import { useOrganizationsStore } from "~/stores/organizations";

definePageMeta({
  middleware: ["auth", "superadmin"],
});

const superAdmin = useSuperAdmin();
// Use client-side fetching for non-blocking navigation
useClientFetch("superadmin-usage-overview", () => superAdmin.fetchOverview(true));

const organizationsStore = useOrganizationsStore();
const router = useRouter();

const overview = computed(() => superAdmin.overview.value);
const usageRecords = computed(() => overview.value?.usages ?? []);
const isLoading = computed(() => superAdmin.loading.value);

const search = ref("");

const filteredUsage = computed(() => {
  const term = search.value.trim().toLowerCase();
  if (!term) return usageRecords.value || [];
  return (usageRecords.value || []).filter((usage) => {
    const searchable = [
      usage.organizationName,
      usage.organizationId,
      usage.month,
    ].filter(Boolean).join(" ").toLowerCase();
    
    return searchable.includes(term);
  });
});

const tableColumns = computed(() => [
  { key: "organization", label: "Organization", defaultWidth: 200, minWidth: 150 },
  { key: "month", label: "Month", defaultWidth: 120, minWidth: 100 },
  { key: "cpuCoreSeconds", label: "CPU (core-s)", defaultWidth: 120, minWidth: 100 },
  { key: "memoryByteSeconds", label: "Memory (byte-s)", defaultWidth: 150, minWidth: 120 },
  { key: "bandwidthRxBytes", label: "Bandwidth RX", defaultWidth: 140, minWidth: 110 },
  { key: "bandwidthTxBytes", label: "Bandwidth TX", defaultWidth: 140, minWidth: 110 },
  { key: "storageBytes", label: "Storage", defaultWidth: 120, minWidth: 100 },
  { key: "deploymentsActivePeak", label: "Peak Deployments", defaultWidth: 140, minWidth: 120 },
  { key: "actions", label: "Actions", defaultWidth: 120, minWidth: 100, resizable: false },
]);

const tableRows = computed(() => {
  return (filteredUsage.value || []).map((usage) => ({
    ...usage,
    organization: usage.organizationName,
    cpuCoreSeconds: formatNumber(usage.cpuCoreSeconds),
    memoryByteSeconds: formatNumber(usage.memoryByteSeconds),
    bandwidthRxBytes: formatBytes(usage.bandwidthRxBytes),
    bandwidthTxBytes: formatBytes(usage.bandwidthTxBytes),
    storageBytes: formatBytes(usage.storageBytes),
    deploymentsActivePeak: formatNumber(usage.deploymentsActivePeak),
  }));
});

function refresh() {
  superAdmin.fetchOverview(true).catch(() => null);
}

function switchToOrg(orgId: string) {
  organizationsStore.switchOrganization(orgId);
  router.push({
    path: "/organizations",
    query: { organizationId: orgId },
  });
}

const numberFormatter = new Intl.NumberFormat(undefined, {
  maximumFractionDigits: 0,
});

function formatNumber(value?: number | bigint | null) {
  if (value === undefined || value === null) return "0";
  return numberFormatter.format(Number(value));
}

const { formatBytes } = useUtils();
</script>

