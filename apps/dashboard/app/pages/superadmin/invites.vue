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
                  @click.stop="resendInvite(row as { organizationId: string; id: string; email: string })"
                  :disabled="isLoading || resendingInvite === row.id"
                >
                  {{ resendingInvite === row.id ? "Sending..." : "Resend" }}
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
import { OrganizationService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useToast } from "~/composables/useToast";

definePageMeta({
  middleware: ["auth", "superadmin"],
});

const superAdmin = useSuperAdmin();
// Use client-side fetching for non-blocking navigation
useClientFetch("superadmin-invites-overview", () => superAdmin.fetchOverview(true));

const router = useRouter();
const organizationsStore = useOrganizationsStore();
const orgClient = useConnectClient(OrganizationService);
const { toast } = useToast();

const overview = computed(() => superAdmin.overview.value);
const invites = computed(() => overview.value?.pendingInvites ?? []);
const isLoading = computed(() => superAdmin.loading.value);

const search = ref("");
const roleFilter = ref<string>("all");
const resendingInvite = ref<string | null>(null);

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

async function resendInvite(invite: { organizationId: string; id: string; email: string }) {
  if (resendingInvite.value === invite.id) return;
  
  resendingInvite.value = invite.id;
  try {
    await orgClient.resendInvite({
      organizationId: invite.organizationId,
      memberId: invite.id,
    });
    toast.success(`Invitation email sent to ${invite.email}`);
    // Optionally refresh the overview to update timestamps
    await superAdmin.fetchOverview(true);
  } catch (error: any) {
    toast.error(error?.message || "Failed to resend invitation email");
  } finally {
    resendingInvite.value = null;
  }
}

const { formatDate } = useUtils();
</script>

