<template>
  <SuperadminPageLayout
    title="Deployments"
    description="View and audit all deployments across every organization."
    :columns="tableColumns"
    :rows="tableRows"
    :filters="filterConfigs"
    :search="search"
    :empty-text="isLoading ? 'Loading deployments…' : 'No deployments match your filters.'"
    :loading="isLoading"
    search-placeholder="Search by name, ID, domain, org ID, environment, status…"
    @update:search="search = $event"
    @filter-change="handleFilterChange"
    @refresh="refresh"
    @row-click="openDeployment"
  >
          <template #cell-deployment="{ value, row }">
            <SuperadminResourceCell
              :name="row.name"
              :domain="row.domain"
              :id="row.id"
            />
          </template>
          <template #cell-organization="{ value, row }">
            <SuperadminOrganizationCell
              :organization-name="row.organizationName"
              :organization-id="row.organizationId || value"
            />
          </template>
          <template #cell-environment="{ value }">
            <SuperadminStatusBadge
              :status="value?.toLowerCase()"
              :status-map="environmentStatusMap"
            />
          </template>
          <template #cell-status="{ value }">
            <SuperadminStatusBadge
              :status="value?.toLowerCase()"
              :status-map="deploymentStatusMap"
            />
          </template>
  </SuperadminPageLayout>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useOrganizationsStore } from "~/stores/organizations";
import SuperadminPageLayout from "~/components/superadmin/SuperadminPageLayout.vue";
import SuperadminResourceCell from "~/components/superadmin/SuperadminResourceCell.vue";
import SuperadminOrganizationCell from "~/components/superadmin/SuperadminOrganizationCell.vue";
import SuperadminStatusBadge from "~/components/superadmin/SuperadminStatusBadge.vue";
import type { FilterConfig } from "~/components/superadmin/SuperadminFilterBar.vue";
import type { BadgeVariant } from "~/components/oui/Badge.vue";

definePageMeta({
  middleware: ["auth", "superadmin"],
});

const superAdmin = useSuperAdmin();
// Use client-side fetching for non-blocking navigation
useClientFetch("superadmin-deployments-overview", () => superAdmin.fetchOverview(true));

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

const filterConfigs = computed(() => [
  {
    key: "environment",
    placeholder: "Environment",
    items: environmentOptions.value,
  },
  {
    key: "status",
    placeholder: "Status",
    items: statusOptions.value,
  },
] as FilterConfig[]);

const handleFilterChange = (key: string, value: string) => {
  if (key === "environment") {
    environmentFilter.value = value;
  } else if (key === "status") {
    statusFilter.value = value;
  }
};

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
    organizationName: deployment.organizationName,
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

const environmentStatusMap: Record<string, { label: string; variant: BadgeVariant }> = {
  production: { label: "PRODUCTION", variant: "success" },
  staging: { label: "STAGING", variant: "warning" },
  development: { label: "DEVELOPMENT", variant: "secondary" },
  unspecified: { label: "UNSPECIFIED", variant: "secondary" },
};

const deploymentStatusMap: Record<string, { label: string; variant: BadgeVariant }> = {
  created: { label: "CREATED", variant: "secondary" },
  building: { label: "BUILDING", variant: "warning" },
  running: { label: "RUNNING", variant: "success" },
  stopped: { label: "STOPPED", variant: "secondary" },
  failed: { label: "FAILED", variant: "danger" },
  deploying: { label: "DEPLOYING", variant: "warning" },
  unknown: { label: "UNKNOWN", variant: "secondary" },
};
</script>

