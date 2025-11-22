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
    @row-click="(row) => viewOrganization(row.id)"
  >
          <template #cell-organization="{ value, row }">
            <div>
              <NuxtLink
                :to="`/superadmin/organizations/${row.id}`"
                class="font-medium text-text-primary hover:text-primary transition-colors cursor-pointer"
                @click.stop
              >
                {{ row.name || row.slug || "—" }}
              </NuxtLink>
              <div v-if="row.slug" class="text-xs text-text-muted mt-0.5">{{ row.slug }}</div>
              <NuxtLink
                :to="`/superadmin/organizations/${row.id}`"
                class="text-xs font-mono text-text-tertiary mt-0.5 hover:text-primary transition-colors cursor-pointer"
                @click.stop
              >
                {{ row.id }}
              </NuxtLink>
              <div v-if="row.domain" class="text-xs text-text-muted mt-0.5">{{ row.domain }}</div>
            </div>
            <div v-if="organizationOwners.get(row.id)" class="text-xs text-text-muted mt-1">
              Owner: 
              <NuxtLink
                :to="`/superadmin/users/${organizationOwners.get(row.id)?.userId}`"
                class="text-primary hover:underline"
                @click.stop
              >
                {{ organizationOwners.get(row.id)?.name }}
              </NuxtLink>
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
            <span class="font-mono">
              <OuiCurrency :value="Number(value || 0)" />
            </span>
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
import { useToast } from "~/composables/useToast";
import SuperadminPageLayout from "~/components/superadmin/SuperadminPageLayout.vue";
import SuperadminResourceCell from "~/components/superadmin/SuperadminResourceCell.vue";
import SuperadminStatusBadge from "~/components/superadmin/SuperadminStatusBadge.vue";
import SuperadminActionsCell, { type Action } from "~/components/superadmin/SuperadminActionsCell.vue";
import type { FilterConfig } from "~/components/superadmin/SuperadminFilterBar.vue";
import type { BadgeVariant } from "~/components/oui/Badge.vue";

const search = ref("");
const planFilter = ref<string>("all");
const statusFilter = ref<string>("all");
const router = useRouter();
const organizationsStore = useOrganizationsStore();
const orgClient = useConnectClient(OrganizationService);
const isLoading = ref(false);

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

// Fetch organizations using listOrganizations endpoint
const { data: organizationsData, refresh: refreshOrganizations } = await useClientFetch(
  "superadmin-organizations-list",
  async () => {
    const response = await orgClient.listOrganizations({
      onlyMine: false, // Superadmin gets all organizations
    });
    return response.organizations || [];
  }
);

const organizations = computed(() => organizationsData.value || []);

// Store owner information for organizations (map of orgId -> { name, userId })
const organizationOwners = ref<Map<string, { name: string; userId: string }>>(new Map());
const loadingOwners = ref<Set<string>>(new Set());

// Fetch owner for an organization
async function fetchOwner(orgId: string): Promise<void> {
  if (organizationOwners.value.has(orgId) || loadingOwners.value.has(orgId)) {
    return;
  }

  loadingOwners.value.add(orgId);
  try {
    const res = await orgClient.listMembers({ organizationId: orgId });
    const owner = res.members?.find((m) => m.role === "owner");
    if (owner?.user) {
      const ownerName = owner.user.name || owner.user.email || owner.user.id || "Unknown";
      const userId = owner.user.id || "";
      if (userId) {
        organizationOwners.value.set(orgId, { name: ownerName, userId });
      }
    }
  } catch (error) {
    console.error(`Failed to fetch owner for org ${orgId}:`, error);
  } finally {
    loadingOwners.value.delete(orgId);
  }
}

// Fetch owners for all organizations
async function loadOwners() {
  if (!organizations.value.length) return;
  const promises = organizations.value.map((org) => fetchOwner(org.id));
  await Promise.all(promises);
}

// Watch for organizations changes and load owners
watch(organizations, () => {
  organizationOwners.value.clear();
  loadingOwners.value.clear();
  if (organizations.value.length > 0) {
    loadOwners();
  }
}, { immediate: true });

const openManageCredits = (orgId: string, currentBalance: number | bigint) => {
  manageCreditsOrgId.value = orgId;
  const balance = typeof currentBalance === 'bigint' ? currentBalance : BigInt(Number(currentBalance) || 0);
  manageCreditsCurrentBalance.value = Number(balance);
  manageCreditsAmount.value = "";
  manageCreditsAction.value = "add";
  manageCreditsNote.value = "";
  manageCreditsDialogOpen.value = true;
};

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
  { key: "createdAt", label: "Created", defaultWidth: 150, minWidth: 120 },
  { key: "actions", label: "Actions", defaultWidth: 200, minWidth: 150, resizable: false },
]);

const tableRows = computed(() => {
  return (filteredOrganizations.value || []).map((org) => ({
    ...org,
    createdAt: formatDate(org.createdAt),
    credits: org.credits ?? BigInt(0),
    owner: organizationOwners.value.get(org.id) || null,
  }));
});

async function refresh() {
  await refreshOrganizations();
  await loadOwners();
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

function viewOrganization(orgId: string) {
  router.push(`/superadmin/organizations/${orgId}`);
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
      onClick: () => openManageCredits(row.id, row.credits ?? BigInt(0)),
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
  const { toast } = useToast();
  try {
    const amountCents = BigInt(Math.round(amount * 100));
    let response;
    if (manageCreditsAction.value === "add") {
      response = await orgClient.adminAddCredits({
        organizationId: manageCreditsOrgId.value,
        amountCents,
        note: manageCreditsNote.value || undefined,
      });
      toast.success(`Successfully added ${formatCurrency(Number(amountCents))} in credits`);
    } else {
      response = await orgClient.adminRemoveCredits({
        organizationId: manageCreditsOrgId.value,
        amountCents,
        note: manageCreditsNote.value || undefined,
      });
      toast.success(`Successfully removed ${formatCurrency(Number(amountCents))} in credits`);
    }
    
    manageCreditsDialogOpen.value = false;
    manageCreditsAmount.value = "";
    manageCreditsNote.value = "";
    await refresh(); // Refresh to get updated organizations with credits
  } catch (err: any) {
    console.error("Failed to manage credits:", err);
    toast.error(err?.message || "Failed to manage credits");
  } finally {
    manageCreditsLoading.value = false;
  }
}
</script>

