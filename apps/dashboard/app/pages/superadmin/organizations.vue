<template>
  <OuiStack gap="xl">
    <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
      <OuiStack gap="xs">
        <OuiText tag="h1" size="3xl" weight="extrabold">Organizations</OuiText>
        <OuiText color="muted">Manage every tenant across the platform.</OuiText>
      </OuiStack>
      <OuiFlex gap="sm" wrap="wrap">
        <div class="w-72 max-w-full">
          <OuiInput
            v-model="search"
            type="search"
            placeholder="Search by name, slug, ID, domain, plan, status…"
            clearable
            size="sm"
          />
        </div>
        <div class="min-w-[140px]">
          <OuiSelect
            v-model="planFilter"
            :items="planOptions"
            placeholder="Plan"
            size="sm"
          />
        </div>
        <div class="min-w-[140px]">
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
          :empty-text="isLoading ? 'Loading organizations…' : 'No organizations match your filters.'"
        >
          <template #cell-organization="{ value, row }">
            <div>
              <div class="font-medium text-text-primary">{{ row.name || row.slug || "—" }}</div>
              <div class="text-xs text-text-muted">
                <span v-if="row.slug">{{ row.slug }}</span>
                <span v-else class="text-text-tertiary">No slug</span>
              </div>
              <div class="text-xs font-mono text-text-tertiary mt-0.5">{{ row.id }}</div>
              <div v-if="row.domain" class="text-xs text-text-muted mt-0.5">{{ row.domain }}</div>
            </div>
          </template>
          <template #cell-plan="{ value }">
            <span class="text-text-secondary">{{ prettyPlan(value) }}</span>
          </template>
          <template #cell-status="{ value }">
            <span class="uppercase text-xs">{{ value || "—" }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="text-right">
              <OuiFlex gap="sm" justify="end">
                <OuiButton size="xs" variant="ghost" @click="openMembers(row.id)">
                  Members
                </OuiButton>
                <OuiButton size="xs" variant="ghost" @click="openDeployments(row.id)">
                  Deployments
                </OuiButton>
                <OuiButton size="xs" @click="switchToOrg(row.id)">
                  Manage
                </OuiButton>
              </OuiFlex>
            </div>
          </template>
        </OuiTable>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
definePageMeta({
  middleware: ["auth", "superadmin"],
});

import { ArrowPathIcon } from "@heroicons/vue/24/outline";
import { computed, ref } from "vue";
import { useOrganizationsStore } from "~/stores/organizations";

const superAdmin = useSuperAdmin();
await superAdmin.fetchOverview(true);

const search = ref("");
const planFilter = ref<string>("all");
const statusFilter = ref<string>("all");
const router = useRouter();
const organizationsStore = useOrganizationsStore();

const overview = computed(() => superAdmin.overview.value);
const organizations = computed(() => overview.value?.organizations ?? []);
const isLoading = computed(() => superAdmin.loading.value);

const planOptions = computed(() => {
  const plans = new Set<string>();
  organizations.value.forEach((org) => {
    if (org.plan) plans.add(org.plan);
  });
  const sortedPlans = Array.from(plans).sort();
  return [
    { label: "All plans", value: "all" },
    ...sortedPlans.map((plan) => ({ label: prettyPlan(plan), value: plan })),
  ];
});

const statusOptions = computed(() => {
  const statuses = new Set<string>();
  organizations.value.forEach((org) => {
    if (org.status) statuses.add(org.status);
  });
  const sortedStatuses = Array.from(statuses).sort();
  return [
    { label: "All statuses", value: "all" },
    ...sortedStatuses.map((status) => ({ label: status.toUpperCase(), value: status })),
  ];
});

const filteredOrganizations = computed(() => {
  const term = search.value.trim().toLowerCase();
  const plan = planFilter.value;
  const status = statusFilter.value;
  
  return organizations.value.filter((org) => {
    // Plan filter
    if (plan !== "all" && org.plan !== plan) {
      return false;
    }
    
    // Status filter
    if (status !== "all" && org.status !== status) {
      return false;
    }
    
    // Search filter
    if (!term) return true;
    
    const searchable = [
      org.name,
      org.slug,
      org.id,
      org.domain,
      org.plan,
      org.status,
    ].filter(Boolean).join(" ").toLowerCase();
    
    return searchable.includes(term);
  });
});

const tableColumns = computed(() => [
  { key: "organization", label: "Organization", defaultWidth: 250, minWidth: 150 },
  { key: "plan", label: "Plan", defaultWidth: 120, minWidth: 80 },
  { key: "status", label: "Status", defaultWidth: 100, minWidth: 70 },
  { key: "memberCount", label: "Members", defaultWidth: 100, minWidth: 80 },
  { key: "deploymentCount", label: "Deployments", defaultWidth: 120, minWidth: 100 },
  { key: "createdAt", label: "Created", defaultWidth: 150, minWidth: 120 },
  { key: "actions", label: "Actions", defaultWidth: 200, minWidth: 150, resizable: false },
]);

const tableRows = computed(() => {
  return filteredOrganizations.value.map((org) => ({
    ...org,
    memberCount: formatNumber(org.memberCount),
    deploymentCount: formatNumber(org.deploymentCount),
    createdAt: formatDate(org.createdAt),
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

function openMembers(orgId: string) {
  organizationsStore.switchOrganization(orgId);
  router.push({
    path: "/organizations",
    query: { tab: "members", organizationId: orgId },
  });
}

function openDeployments(orgId: string) {
  organizationsStore.switchOrganization(orgId);
  router.push({
    path: "/deployments",
    query: { organizationId: orgId },
  });
}

const numberFormatter = new Intl.NumberFormat();
const dateFormatter = new Intl.DateTimeFormat(undefined, { dateStyle: "medium" });

function formatNumber(value?: number | bigint | null) {
  if (value === undefined || value === null) return "0";
  return numberFormatter.format(Number(value));
}

function formatDate(timestamp?: { seconds?: number | bigint; nanos?: number } | null) {
  if (!timestamp || timestamp.seconds === undefined) return "—";
  const seconds = typeof timestamp.seconds === "bigint" ? Number(timestamp.seconds) : timestamp.seconds;
  const millis = seconds * 1000 + Math.floor((timestamp.nanos ?? 0) / 1_000_000);
  const date = new Date(millis);
  return Number.isNaN(date.getTime()) ? "—" : dateFormatter.format(date);
}

function prettyPlan(plan?: string | null) {
  if (!plan) return "—";
  return plan.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}
</script>

