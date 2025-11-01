<template>
  <OuiStack gap="xl">
    <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
      <OuiStack gap="xs">
        <OuiText tag="h1" size="3xl" weight="extrabold">Deployments</OuiText>
        <OuiText color="muted">
          View and audit all deployments across every organization.
        </OuiText>
      </OuiStack>
      <OuiFlex gap="sm" wrap="wrap">
        <div class="w-72 max-w-full">
          <OuiInput
            v-model="search"
            type="search"
            placeholder="Search by name, ID, domain, org ID, environment, status…"
            clearable
            size="sm"
          />
        </div>
        <div class="min-w-[160px]">
          <OuiSelect
            v-model="environmentFilter"
            :items="environmentOptions"
            placeholder="Environment"
            size="sm"
          />
        </div>
        <div class="min-w-[160px]">
          <OuiSelect
            v-model="statusFilter"
            :items="statusOptions"
            placeholder="Status"
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
          :empty-text="isLoading ? 'Loading deployments…' : 'No deployments match your filters.'"
          row-class="hover:bg-surface-subtle/50 transition-colors cursor-pointer"
          @row-click="openDeployment"
        >
          <template #cell-deployment="{ value, row }">
            <div>
              <div class="font-medium text-text-primary hover:text-primary transition-colors">{{ row.name }}</div>
              <div v-if="row.domain" class="text-xs text-text-muted">{{ row.domain }}</div>
              <div class="text-xs font-mono text-text-tertiary mt-0.5">{{ row.id }}</div>
            </div>
          </template>
          <template #cell-organization="{ value }">
            <div class="text-text-secondary font-mono text-sm">{{ value }}</div>
          </template>
          <template #cell-environment="{ value }">
            <span class="text-text-secondary uppercase text-xs">{{ value }}</span>
          </template>
          <template #cell-status="{ value }">
            <span class="text-text-secondary uppercase text-xs">{{ value }}</span>
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
await superAdmin.fetchOverview(true);

const router = useRouter();
const organizationsStore = useOrganizationsStore();

const overview = computed(() => superAdmin.overview.value);
const deployments = computed(() => overview.value?.deployments ?? []);
const isLoading = computed(() => superAdmin.loading.value);

const search = ref("");
const environmentFilter = ref<string>("all");
const statusFilter = ref<string>("all");

const environmentOptions = computed(() => [
  { label: "All environments", value: "all" },
  { label: "Production", value: "production" },
  { label: "Staging", value: "staging" },
  { label: "Development", value: "development" },
]);

const statusOptions = computed(() => {
  const statuses = new Set<string>();
  deployments.value.forEach((dep) => {
    const status = formatStatus(dep.status);
    if (status) statuses.add(status);
  });
  const sortedStatuses = Array.from(statuses).sort();
  return [
    { label: "All statuses", value: "all" },
    ...sortedStatuses.map((status) => ({ label: status, value: status.toLowerCase() })),
  ];
});

const filteredDeployments = computed(() => {
  const term = search.value.trim().toLowerCase();
  const env = environmentFilter.value;
  const status = statusFilter.value;
  
  return deployments.value.filter((deployment) => {
    // Environment filter
    if (env !== "all" && formatEnvironment(deployment.environment).toLowerCase() !== env.toLowerCase()) {
      return false;
    }
    
    // Status filter
    if (status !== "all") {
      const deploymentStatus = formatStatus(deployment.status).toLowerCase();
      if (deploymentStatus !== status) {
        return false;
      }
    }
    
    // Search filter
    if (!term) return true;
    
    const searchable = [
      deployment.name,
      deployment.id,
      deployment.domain,
      deployment.organizationId,
      formatEnvironment(deployment.environment),
      formatStatus(deployment.status),
    ].filter(Boolean).join(" ").toLowerCase();
    
    return searchable.includes(term);
  });
});

const tableColumns = computed(() => [
  { key: "deployment", label: "Deployment", defaultWidth: 250, minWidth: 150 },
  { key: "organization", label: "Organization", defaultWidth: 180, minWidth: 120 },
  { key: "environment", label: "Environment", defaultWidth: 120, minWidth: 100 },
  { key: "status", label: "Status", defaultWidth: 120, minWidth: 100 },
  { key: "created", label: "Created", defaultWidth: 150, minWidth: 120 },
  { key: "lastDeployed", label: "Last Deployed", defaultWidth: 150, minWidth: 120 },
]);

const tableRows = computed(() => {
  return filteredDeployments.value.map((deployment) => ({
    ...deployment,
    organization: deployment.organizationId,
    environment: formatEnvironment(deployment.environment),
    status: formatStatus(deployment.status),
    created: formatDate(deployment.createdAt),
    lastDeployed: formatDate(deployment.lastDeployedAt || deployment.createdAt),
  }));
});

function refresh() {
  superAdmin.fetchOverview(true).catch(() => null);
}

function openDeployment(row: Record<string, any>) {
  if (row.id && row.organizationId) {
    organizationsStore.switchOrganization(row.organizationId);
    router.push(`/deployments/${row.id}`);
  }
}

const dateFormatter = new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" });

function formatDate(timestamp?: { seconds?: number | bigint; nanos?: number } | null) {
  if (!timestamp || timestamp.seconds === undefined) return "—";
  const seconds = typeof timestamp.seconds === "bigint" ? Number(timestamp.seconds) : timestamp.seconds;
  const millis = seconds * 1000 + Math.floor((timestamp.nanos ?? 0) / 1_000_000);
  const date = new Date(millis);
  return Number.isNaN(date.getTime()) ? "—" : dateFormatter.format(date);
}

function formatEnvironment(env?: number) {
  switch (env) {
    case 1:
      return "PRODUCTION";
    case 2:
      return "STAGING";
    case 3:
      return "DEVELOPMENT";
    default:
      return "UNSPECIFIED";
  }
}

function formatStatus(status?: number) {
  switch (status) {
    case 1:
      return "CREATED";
    case 2:
      return "BUILDING";
    case 3:
      return "RUNNING";
    case 4:
      return "STOPPED";
    case 5:
      return "FAILED";
    case 6:
      return "DEPLOYING";
    default:
      return "UNKNOWN";
  }
}
</script>

