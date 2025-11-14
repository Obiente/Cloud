<template>
  <SuperadminPageLayout
    title="Organizations"
    description="Manage every tenant across the platform."
    :columns="tableColumns"
    :rows="tableRows"
    :filters="filterConfigs"
    :search="search"
    :empty-text="isLoading ? 'Loading organizations…' : 'No organizations match your filters.'"
    :loading="isLoading"
    search-placeholder="Search by name, slug, ID, domain, plan, status…"
    @update:search="search = $event"
    @filter-change="handleFilterChange"
    @refresh="refresh"
  >
          <template #cell-organization="{ value, row }">
            <SuperadminResourceCell
              :name="row.name || row.slug"
              :subtitle="row.slug ? undefined : 'No slug'"
              :id="row.id"
              :domain="row.domain"
            />
            <div v-if="row.ownerName" class="text-xs text-text-muted mt-0.5">
              Owner: {{ row.ownerName }}
            </div>
          </template>
          <template #cell-plan="{ value }">
            <span class="text-text-secondary">{{ prettyPlan(value) }}</span>
          </template>
          <template #cell-status="{ value }">
            <SuperadminStatusBadge
              :status="value?.toLowerCase()"
              :status-map="orgStatusMap"
            />
          </template>
          <template #cell-credits="{ value, row }">
            <span class="font-mono"><OuiCurrency :value="value" /></span>
          </template>
          <template #cell-actions="{ row }">
            <SuperadminActionsCell :actions="getOrgActions(row)" />
          </template>
  </SuperadminPageLayout>

    <!-- Manage Credits Dialog -->
    <OuiDialog v-model:open="manageCreditsDialogOpen" :title="manageCreditsAction === 'add' ? 'Add Credits' : 'Remove Credits'">
      <OuiStack gap="lg">
        <OuiStack gap="xs">
          <OuiText size="sm" color="muted">
            Current balance: <OuiCurrency :value="manageCreditsCurrentBalance" />
          </OuiText>
        </OuiStack>
        
        <OuiStack gap="md">
          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Action</OuiText>
            <OuiFlex gap="sm">
              <OuiButton
                :variant="manageCreditsAction === 'add' ? 'solid' : 'outline'"
                size="sm"
                @click="manageCreditsAction = 'add'"
              >
                <PlusIcon class="h-4 w-4 mr-2" />
                Add
              </OuiButton>
              <OuiButton
                :variant="manageCreditsAction === 'remove' ? 'solid' : 'outline'"
                size="sm"
                @click="manageCreditsAction = 'remove'"
              >
                <MinusIcon class="h-4 w-4 mr-2" />
                Remove
              </OuiButton>
            </OuiFlex>
          </OuiStack>
          
          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Amount (USD)</OuiText>
            <OuiInput
              v-model="manageCreditsAmount"
              type="number"
              step="0.01"
              min="0.01"
              placeholder="0.00"
            />
          </OuiStack>
          
          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Note (Optional)</OuiText>
            <OuiInput
              v-model="manageCreditsNote"
              type="text"
              placeholder="Reason for this action"
            />
          </OuiStack>
        </OuiStack>

        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="manageCreditsDialogOpen = false">
            Cancel
          </OuiButton>
          <OuiButton 
            variant="solid" 
            @click="manageCredits"
            :disabled="manageCreditsLoading || !manageCreditsAmount || parseFloat(manageCreditsAmount) <= 0"
          >
            {{ manageCreditsLoading ? (manageCreditsAction === 'add' ? 'Adding...' : 'Removing...') : (manageCreditsAction === 'add' ? 'Add Credits' : 'Remove Credits') }}
          </OuiButton>
        </OuiFlex>
      </OuiStack>
    </OuiDialog>
</template>

<script setup lang="ts">
definePageMeta({
  middleware: ["auth", "superadmin"],
});

import { PlusIcon, MinusIcon } from "@heroicons/vue/24/outline";
import { computed, ref } from "vue";
import { useOrganizationsStore } from "~/stores/organizations";
import { OrganizationService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import SuperadminPageLayout from "~/components/superadmin/SuperadminPageLayout.vue";
import SuperadminResourceCell from "~/components/superadmin/SuperadminResourceCell.vue";
import SuperadminStatusBadge from "~/components/superadmin/SuperadminStatusBadge.vue";
import SuperadminActionsCell, { type Action } from "~/components/superadmin/SuperadminActionsCell.vue";
import type { FilterConfig } from "~/components/superadmin/SuperadminFilterBar.vue";
import type { BadgeVariant } from "~/components/oui/Badge.vue";

const superAdmin = useSuperAdmin();
await superAdmin.fetchOverview(true);

const search = ref("");
const planFilter = ref<string>("all");
const statusFilter = ref<string>("all");
const router = useRouter();
const organizationsStore = useOrganizationsStore();
const orgClient = useConnectClient(OrganizationService);

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

const filterConfigs = computed(() => [
  {
    key: "plan",
    placeholder: "Plan",
    items: planOptions.value,
  },
  {
    key: "status",
    placeholder: "Status",
    items: statusOptions.value,
  },
] as FilterConfig[]);

function handleFilterChange(key: string, value: string) {
  if (key === "plan") {
    planFilter.value = value;
  } else if (key === "status") {
    statusFilter.value = value;
  }
}

const manageCreditsDialogOpen = ref(false);
const manageCreditsOrgId = ref<string | null>(null);
const manageCreditsCurrentBalance = ref<number>(0);
const manageCreditsAmount = ref("");
const manageCreditsAction = ref<"add" | "remove">("add");
const manageCreditsNote = ref("");
const manageCreditsLoading = ref(false);

const openManageCredits = (orgId: string, currentBalance: number | bigint) => {
  manageCreditsOrgId.value = orgId;
  manageCreditsCurrentBalance.value = Number(currentBalance);
  manageCreditsAmount.value = "";
  manageCreditsAction.value = "add";
  manageCreditsNote.value = "";
  manageCreditsDialogOpen.value = true;
};

const overview = computed(() => superAdmin.overview.value);
const organizations = computed(() => overview.value?.organizations ?? []);
const isLoading = computed(() => superAdmin.loading.value);

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
  { key: "credits", label: "Credits", defaultWidth: 120, minWidth: 100 },
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
const { formatDate, formatCurrency } = useUtils();

function formatNumber(value?: number | bigint | null) {
  if (value === undefined || value === null) return "0";
  return numberFormatter.format(Number(value));
}

function prettyPlan(plan?: string | null) {
  if (!plan) return "—";
  return plan.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

const orgStatusMap: Record<string, { label: string; variant: BadgeVariant }> = {
  active: { label: "ACTIVE", variant: "success" },
  suspended: { label: "SUSPENDED", variant: "warning" },
  cancelled: { label: "CANCELLED", variant: "danger" },
};

const getOrgActions = (row: any): Action[] => {
  return [
    {
      key: "credits",
      label: "Credits",
      onClick: () => openManageCredits(row.id, row.credits || 0),
    },
    {
      key: "members",
      label: "Members",
      onClick: () => openMembers(row.id),
    },
    {
      key: "deployments",
      label: "Deployments",
      onClick: () => openDeployments(row.id),
    },
    {
      key: "manage",
      label: "Manage",
      onClick: () => switchToOrg(row.id),
      variant: "solid",
    },
  ];
};

async function manageCredits() {
  if (!manageCreditsOrgId.value || !manageCreditsAmount.value) return;
  const amount = parseFloat(manageCreditsAmount.value);
  if (isNaN(amount) || amount <= 0) {
    return;
  }
  
  manageCreditsLoading.value = true;
  try {
    const amountCents = Math.round(amount * 100);
    if (manageCreditsAction.value === "add") {
      // Note: Proto files need to be generated
      await (orgClient as any).adminAddCredits({
        organizationId: manageCreditsOrgId.value,
        amountCents,
        note: manageCreditsNote.value || undefined,
      });
    } else {
      await (orgClient as any).adminRemoveCredits({
        organizationId: manageCreditsOrgId.value,
        amountCents,
        note: manageCreditsNote.value || undefined,
      });
    }
    manageCreditsDialogOpen.value = false;
    manageCreditsAmount.value = "";
    manageCreditsNote.value = "";
    await refresh(); // Refresh to get updated credits
  } catch (err: any) {
    console.error("Failed to manage credits:", err);
  } finally {
    manageCreditsLoading.value = false;
  }
}
</script>

