<template>
  <OuiStack gap="xl">
    <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
      <OuiStack gap="xs">
        <OuiText tag="h1" size="3xl" weight="extrabold">Pending Invites</OuiText>
        <OuiText color="muted">Follow up with outstanding invitations.</OuiText>
      </OuiStack>
      <OuiFlex gap="sm" wrap="wrap">
        <div class="w-72 max-w-full">
          <OuiInput
            v-model="search"
            type="search"
            placeholder="Search by email, ID, organization ID, role…"
            clearable
            size="sm"
          />
        </div>
        <div class="min-w-[140px]">
          <OuiSelect
            v-model="roleFilter"
            :items="roleOptions"
            placeholder="Role"
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
          :empty-text="isLoading ? 'Loading invites…' : 'No pending invites match your search.'"
        >
          <template #cell-email="{ value, row }">
            <div>
              <div class="font-medium text-text-primary">{{ value }}</div>
              <div class="text-xs font-mono text-text-tertiary mt-0.5">{{ row.id }}</div>
            </div>
          </template>
          <template #cell-organization="{ value }">
            <div class="text-text-secondary font-mono text-sm">{{ value }}</div>
          </template>
          <template #cell-role="{ value }">
            <span class="text-text-secondary uppercase text-xs">{{ value }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="text-right">
              <OuiFlex gap="sm" justify="end">
                <OuiButton size="xs" variant="ghost" @click.stop="switchToOrg(row.organizationId)">
                  Manage
                </OuiButton>
                <OuiButton
                  size="xs"
                  variant="ghost"
                  color="danger"
                  @click.stop="resendInvite(row as { organizationId: string; id: string })"
                  :disabled="isLoading"
                >
                  Resend
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
const invites = computed(() => overview.value?.pendingInvites ?? []);
const isLoading = computed(() => superAdmin.loading.value);

const search = ref("");
const roleFilter = ref<string>("all");

const roleOptions = computed(() => {
  const roles = new Set<string>();
  invites.value.forEach((invite) => {
    if (invite.role) roles.add(invite.role);
  });
  const sortedRoles = Array.from(roles).sort();
  return [
    { label: "All roles", value: "all" },
    ...sortedRoles.map((role) => ({ label: role.toUpperCase(), value: role })),
  ];
});

const filteredInvites = computed(() => {
  const term = search.value.trim().toLowerCase();
  const role = roleFilter.value;
  
  return invites.value.filter((invite) => {
    // Role filter
    if (role !== "all" && invite.role !== role) {
      return false;
    }
    
    // Search filter
    if (!term) return true;
    
    const searchable = [
      invite.email,
      invite.id,
      invite.organizationId,
      invite.role,
    ].filter(Boolean).join(" ").toLowerCase();
    
    return searchable.includes(term);
  });
});

const tableColumns = computed(() => [
  { key: "email", label: "Email", defaultWidth: 250, minWidth: 150 },
  { key: "organization", label: "Organization", defaultWidth: 180, minWidth: 120 },
  { key: "role", label: "Role", defaultWidth: 120, minWidth: 80 },
  { key: "invitedAt", label: "Invited", defaultWidth: 150, minWidth: 120 },
  { key: "actions", label: "Actions", defaultWidth: 180, minWidth: 150, resizable: false },
]);

const tableRows = computed(() => {
  return filteredInvites.value.map((invite) => ({
    ...invite,
    organization: invite.organizationId,
    invitedAt: formatDate(invite.invitedAt),
  }));
});

function refresh() {
  superAdmin.fetchOverview(true).catch(() => null);
}

function switchToOrg(orgId: string) {
  organizationsStore.switchOrganization(orgId);
  router.push({
    path: "/organizations",
    query: { tab: "members", organizationId: orgId },
  });
}

function resendInvite(invite: { organizationId: string; id: string }) {
  organizationsStore.switchOrganization(invite.organizationId);
  router.push({
    path: "/organizations",
    query: { tab: "members", invite: invite.id, organizationId: invite.organizationId },
  });
}

const dateFormatter = new Intl.DateTimeFormat(undefined, { dateStyle: "medium" });
function formatDate(timestamp?: { seconds?: number | bigint; nanos?: number } | null) {
  if (!timestamp || timestamp.seconds === undefined) return "—";
  const seconds = typeof timestamp.seconds === "bigint" ? Number(timestamp.seconds) : timestamp.seconds;
  const millis = seconds * 1000 + Math.floor((timestamp.nanos ?? 0) / 1_000_000);
  const date = new Date(millis);
  return Number.isNaN(date.getTime()) ? "—" : dateFormatter.format(date);
}
</script>

