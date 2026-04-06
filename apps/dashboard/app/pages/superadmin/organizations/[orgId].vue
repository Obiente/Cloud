<template>
  <OuiContainer size="full">
    <OuiStack gap="2xl">
      <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
        <OuiStack gap="xs">
          <OuiFlex gap="sm" align="center">
            <OuiButton
              variant="ghost"
              size="sm"
              @click="router.back()"
            >
              <ArrowLeftIcon class="h-4 w-4 mr-1" />
              Back
            </OuiButton>
            <OuiText tag="h1" size="3xl" weight="extrabold">Organization Details</OuiText>
          </OuiFlex>
          <OuiText color="tertiary">View detailed information about this organization.</OuiText>
        </OuiStack>
      </OuiFlex>

      <OuiGrid :cols="{ sm: 1, lg: 3 }" gap="lg">
        <!-- Organization Info Card -->
        <OuiCard class="border border-border-muted rounded-xl">
          <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
            <OuiText tag="h2" size="lg" weight="bold">Organization Information</OuiText>
          </OuiCardHeader>
          <OuiCardBody class="p-6">
            <OuiStack gap="lg">
              <OuiStack gap="xs">
                <OuiText size="xl" weight="bold">
                  {{ organization?.name || "Loading..." }}
                </OuiText>
                <OuiText v-if="organization?.slug" color="tertiary" size="sm">
                  {{ organization.slug }}
                </OuiText>
                <OuiText v-if="organization?.id" color="tertiary" size="xs" class="font-mono">
                  {{ organization.id }}
                </OuiText>
              </OuiStack>

              <OuiStack gap="md" class="border-t border-border-muted pt-4">
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Plan</OuiText>
                  <OuiBadge variant="secondary" tone="soft" size="sm">
                    {{ prettyPlan(organization?.plan) }}
                  </OuiBadge>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Status</OuiText>
                  <OuiBadge
                    :variant="organization?.status === 'active' ? 'success' : organization?.status === 'suspended' ? 'warning' : 'danger'"
                    :tone="organization?.status === 'active' ? 'solid' : 'soft'"
                    size="sm"
                  >
                    {{ organization?.status?.toUpperCase() || "—" }}
                  </OuiBadge>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Domain</OuiText>
                  <OuiText>{{ organization?.domain || "—" }}</OuiText>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Credits</OuiText>
                  <OuiText size="sm" weight="semibold">
                    <OuiCurrency :value="Number(organization?.credits || 0)" />
                  </OuiText>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Total Paid</OuiText>
                  <OuiText size="sm">
                    <OuiCurrency :value="Number(organization?.totalPaidCents || 0)" />
                  </OuiText>
                </OuiStack>

                <OuiStack gap="xs" v-if="organization?.planInfo">
                  <OuiText size="sm" weight="medium" color="tertiary">Plan Details</OuiText>
                  <OuiStack gap="xs">
                    <OuiText size="xs" color="tertiary">
                      CPU: {{ organization.planInfo.cpuCores || 'Unlimited' }} cores
                    </OuiText>
                    <OuiText size="xs" color="tertiary">
                      Memory: {{ formatBytes(Number(organization.planInfo.memoryBytes || 0)) }}
                    </OuiText>
                    <OuiText size="xs" color="tertiary">
                      Max Deployments: {{ organization.planInfo.deploymentsMax || 'Unlimited' }}
                    </OuiText>
                    <OuiText size="xs" color="tertiary">
                      Max VPS: {{ organization.planInfo.maxVpsInstances || 'Unlimited' }}
                    </OuiText>
                  </OuiStack>
                </OuiStack>

                <OuiStack gap="xs" v-if="organization?.createdAt">
                  <OuiText size="sm" weight="medium" color="tertiary">Created</OuiText>
                  <OuiText size="sm">
                    <OuiDate :value="organization.createdAt" />
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Members Card -->
        <OuiCard class="border border-border-muted rounded-xl col-span-1 lg:col-span-2">
          <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
            <OuiText tag="h2" size="lg" weight="bold">Members</OuiText>
          </OuiCardHeader>
          <OuiCardBody class="p-0">
            <OuiTable
              v-if="!loading"
              :columns="memberColumns"
              :rows="tableMembers"
              empty-text="This organization has no members."
              row-class="hover:bg-surface-subtle/50"
            >
              <template #cell-member="{ row }">
                <OuiFlex gap="sm" align="center">
                  <OuiAvatar
                    :name="row.user?.name || row.user?.email || row.user?.id || ''"
                    :src="row.user?.avatarUrl"
                  />
                  <OuiStack gap="xs">
                    <NuxtLink
                      v-if="row.user?.id"
                      :to="`/superadmin/users/${row.user.id}`"
                      class="font-medium text-text-primary hover:text-primary transition-colors cursor-pointer"
                    >
                      {{ row.user?.name || row.user?.email || row.user?.id || "Unknown" }}
                    </NuxtLink>
                    <OuiText v-else weight="medium">
                      {{ row.user?.name || row.user?.email || row.user?.id || "Unknown" }}
                    </OuiText>
                    <OuiText v-if="row.user?.email" color="tertiary" size="xs">
                      {{ row.user.email }}
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </template>
              <template #cell-role="{ row }">
                <OuiBadge
                  :variant="row.role === 'owner' ? 'primary' : 'secondary'"
                  tone="soft"
                  size="sm"
                >
                  {{ row.role }}
                </OuiBadge>
              </template>
              <template #cell-status="{ row }">
                <OuiBadge
                  :variant="row.status === 'active' ? 'success' : 'secondary'"
                  :tone="row.status === 'active' ? 'solid' : 'soft'"
                  size="sm"
                >
                  {{ row.status }}
                </OuiBadge>
              </template>
              <template #cell-joined="{ value }">
                <OuiDate :value="value" />
              </template>
            </OuiTable>
            <div v-if="loading" class="p-6 text-center text-text-muted">
              Loading members...
            </div>
          </OuiCardBody>
        </OuiCard>
      </OuiGrid>

      <!-- Moderation Card -->
      <OuiCard class="border border-border-muted rounded-xl">
        <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
          <OuiFlex align="center" justify="between">
            <OuiText tag="h2" size="lg" weight="bold">Moderation</OuiText>
            <OuiBadge
              v-if="organization?.status === 'banned'"
              variant="danger"
              tone="solid"
            >BANNED</OuiBadge>
            <OuiBadge
              v-else-if="organization?.status === 'suspended'"
              variant="warning"
              tone="solid"
            >SUSPENDED</OuiBadge>
            <OuiBadge v-else variant="success" tone="soft">Active</OuiBadge>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody class="p-6">
          <OuiStack gap="lg">
            <!-- Suspension details -->
            <OuiStack
              v-if="organization?.status === 'suspended'"
              gap="md"
              class="p-4 rounded-lg bg-surface-subtle border border-border-muted"
            >
              <OuiGrid :cols="{ sm: 1, md: 3 }" gap="md">
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Suspended By</OuiText>
                  <OuiText size="sm" class="font-mono">{{ organization.suspendedBy || "—" }}</OuiText>
                </OuiStack>
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Suspended At</OuiText>
                  <OuiText size="sm">
                    <OuiDate v-if="organization.suspendedAt" :value="organization.suspendedAt" />
                    <span v-else>—</span>
                  </OuiText>
                </OuiStack>
                <OuiStack v-if="organization.suspensionExpires" gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Expires</OuiText>
                  <OuiText size="sm">
                    <OuiDate :value="organization.suspensionExpires" />
                  </OuiText>
                </OuiStack>
                <OuiStack v-if="organization.suspensionReason" gap="xs" class="md:col-span-3">
                  <OuiText size="sm" weight="medium" color="tertiary">Reason</OuiText>
                  <OuiText size="sm">{{ organization.suspensionReason }}</OuiText>
                </OuiStack>
              </OuiGrid>
            </OuiStack>

            <!-- Ban details -->
            <OuiStack
              v-if="organization?.status === 'banned'"
              gap="md"
              class="p-4 rounded-lg bg-surface-subtle border border-border-muted"
            >
              <OuiGrid :cols="{ sm: 1, md: 3 }" gap="md">
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Banned By</OuiText>
                  <OuiText size="sm" class="font-mono">{{ organization.bannedBy || "—" }}</OuiText>
                </OuiStack>
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Banned At</OuiText>
                  <OuiText size="sm">
                    <OuiDate v-if="organization.bannedAt" :value="organization.bannedAt" />
                    <span v-else>—</span>
                  </OuiText>
                </OuiStack>
                <OuiStack v-if="organization.banReason" gap="xs" class="md:col-span-3">
                  <OuiText size="sm" weight="medium" color="tertiary">Reason</OuiText>
                  <OuiText size="sm">{{ organization.banReason }}</OuiText>
                </OuiStack>
              </OuiGrid>
            </OuiStack>

            <!-- Action buttons -->
            <OuiFlex gap="sm" wrap="wrap">
              <template v-if="organization?.status === 'active'">
                <OuiButton
                  color="warning"
                  variant="outline"
                  size="sm"
                  @click="openSuspendDialog"
                  :disabled="isModerating"
                >
                  Suspend Organization
                </OuiButton>
                <OuiButton
                  color="danger"
                  size="sm"
                  @click="openBanDialog"
                  :disabled="isModerating"
                >
                  Ban Organization
                </OuiButton>
              </template>
              <template v-else-if="organization?.status === 'suspended'">
                <OuiButton
                  color="primary"
                  size="sm"
                  @click="handleUnsuspend"
                  :disabled="isModerating"
                >
                  {{ isModerating ? 'Unsuspending...' : 'Unsuspend' }}
                </OuiButton>
                <OuiButton
                  color="danger"
                  size="sm"
                  @click="openBanDialog"
                  :disabled="isModerating"
                >
                  Ban Organization
                </OuiButton>
              </template>
              <template v-else-if="organization?.status === 'banned'">
                <OuiButton
                  color="primary"
                  size="sm"
                  @click="handleUnban"
                  :disabled="isModerating"
                >
                  {{ isModerating ? 'Unbanning...' : 'Unban Organization' }}
                </OuiButton>
              </template>
            </OuiFlex>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>
  </OuiContainer>

  <!-- Suspend Dialog -->
  <OuiDialog v-model:open="suspendDialogOpen" title="Suspend Organization">
    <OuiStack gap="lg">
      <OuiText size="sm" color="tertiary">
        Suspend this organization. Members will not be able to access platform resources until unsuspended.
      </OuiText>
      <OuiInput
        v-model="moderationForm.reason"
        label="Reason (Optional)"
        placeholder="Reason for suspension"
      />
      <OuiInput
        v-model="moderationForm.expiresAt"
        label="Expires At (Optional, ISO date)"
        placeholder="e.g. 2025-12-31T00:00:00Z"
      />
    </OuiStack>
    <template #footer>
      <OuiFlex justify="end" gap="sm">
        <OuiButton variant="ghost" @click="suspendDialogOpen = false">Cancel</OuiButton>
        <OuiButton color="warning" @click="handleSuspend" :disabled="isModerating">
          {{ isModerating ? 'Suspending...' : 'Suspend' }}
        </OuiButton>
      </OuiFlex>
    </template>
  </OuiDialog>

  <!-- Ban Dialog -->
  <OuiDialog v-model:open="banDialogOpen" title="Ban Organization">
    <OuiStack gap="lg">
      <OuiText size="sm" color="tertiary">
        Permanently ban this organization. This is a serious action that cannot be easily reversed.
      </OuiText>
      <OuiInput
        v-model="moderationForm.reason"
        label="Reason (Optional)"
        placeholder="Reason for ban"
      />
    </OuiStack>
    <template #footer>
      <OuiFlex justify="end" gap="sm">
        <OuiButton variant="ghost" @click="banDialogOpen = false">Cancel</OuiButton>
        <OuiButton color="danger" @click="handleBan" :disabled="isModerating">
          {{ isModerating ? 'Banning...' : 'Ban Organization' }}
        </OuiButton>
      </OuiFlex>
    </template>
  </OuiDialog>
</template>

<script setup lang="ts">
import { ArrowLeftIcon } from "@heroicons/vue/24/outline";
import { OrganizationService, SuperadminService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useUtils } from "~/composables/useUtils";

definePageMeta({
  middleware: ["auth", "superadmin"],
});

const route = useRoute();
const router = useRouter();
const orgClient = useConnectClient(OrganizationService);
const superadminClient = useConnectClient(SuperadminService);
const { toast } = useToast();
const { formatBytes, formatCurrency } = useUtils();

const orgId = computed(() => route.params.orgId as string);
const organization = ref<any>(null);
const members = ref<any[]>([]);

const memberColumns = [
  { key: "member", label: "Member", defaultWidth: 250, minWidth: 200 },
  { key: "role", label: "Role", defaultWidth: 120, minWidth: 100 },
  { key: "status", label: "Status", defaultWidth: 120, minWidth: 100 },
  { key: "joined", label: "Joined", defaultWidth: 180, minWidth: 150 },
];

const tableMembers = computed(() => members.value);

function prettyPlan(plan?: string | null) {
  if (!plan) return "—";
  return plan.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

async function loadOrganization() {
  if (!orgId.value) return null;
  try {
    const orgResponse = await orgClient.getOrganization({
      organizationId: orgId.value,
    });
    const membersResponse = await orgClient.listMembers({
      organizationId: orgId.value,
    });
    return {
      organization: orgResponse.organization,
      members: membersResponse.members || [],
    };
  } catch (error: unknown) {
    console.error("Failed to load organization:", error);
    const { toast } = useToast();
    toast.error((error as Error | undefined)?.message || "Failed to load organization");
    throw error;
  }
}

const { data: orgData, pending: loading } = await useClientFetch(
  () => `superadmin-organization-${orgId.value}`,
  loadOrganization
);

watch(orgData, (newData) => {
  if (newData) {
    organization.value = newData.organization;
    members.value = newData.members;
  }
}, { immediate: true });

// Reload org data (after moderation action)
const reloadOrg = async () => {
  if (!orgId.value) return;
  try {
    const orgResponse = await orgClient.getOrganization({ organizationId: orgId.value });
    organization.value = orgResponse.organization;
  } catch {
    // silently ignore
  }
};

// Moderation
const suspendDialogOpen = ref(false);
const banDialogOpen = ref(false);
const isModerating = ref(false);
const moderationForm = ref({ reason: "", expiresAt: "" });

function openSuspendDialog() {
  moderationForm.value = { reason: "", expiresAt: "" };
  suspendDialogOpen.value = true;
}

function openBanDialog() {
  moderationForm.value = { reason: "", expiresAt: "" };
  banDialogOpen.value = true;
}

const handleSuspend = async () => {
  if (!orgId.value) return;
  isModerating.value = true;
  try {
    await superadminClient.suspendOrganization({
      organizationId: orgId.value,
      reason: moderationForm.value.reason || undefined,
      expiresAt: moderationForm.value.expiresAt
        ? { seconds: BigInt(Math.floor(new Date(moderationForm.value.expiresAt).getTime() / 1000)), nanos: 0 }
        : undefined,
    });
    toast.success("Organization suspended.");
    suspendDialogOpen.value = false;
    await reloadOrg();
  } catch (error: unknown) {
    toast.error(`Failed to suspend: ${(error as any)?.message || "Unknown error"}`);
  } finally {
    isModerating.value = false;
  }
};

const handleUnsuspend = async () => {
  if (!orgId.value) return;
  isModerating.value = true;
  try {
    await superadminClient.unsuspendOrganization({ organizationId: orgId.value });
    toast.success("Organization unsuspended.");
    await reloadOrg();
  } catch (error: unknown) {
    toast.error(`Failed to unsuspend: ${(error as any)?.message || "Unknown error"}`);
  } finally {
    isModerating.value = false;
  }
};

const handleBan = async () => {
  if (!orgId.value) return;
  isModerating.value = true;
  try {
    await superadminClient.banOrganization({
      organizationId: orgId.value,
      reason: moderationForm.value.reason || undefined,
    });
    toast.success("Organization banned.");
    banDialogOpen.value = false;
    await reloadOrg();
  } catch (error: unknown) {
    toast.error(`Failed to ban: ${(error as any)?.message || "Unknown error"}`);
  } finally {
    isModerating.value = false;
  }
};

const handleUnban = async () => {
  if (!orgId.value) return;
  isModerating.value = true;
  try {
    await superadminClient.unbanOrganization({ organizationId: orgId.value });
    toast.success("Organization unbanned.");
    await reloadOrg();
  } catch (error: unknown) {
    toast.error(`Failed to unban: ${(error as any)?.message || "Unknown error"}`);
  } finally {
    isModerating.value = false;
  }
};
</script>

