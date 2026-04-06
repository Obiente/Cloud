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
            <OuiText tag="h1" size="3xl" weight="extrabold">User Details</OuiText>
          </OuiFlex>
          <OuiText color="tertiary">View detailed information about this user.</OuiText>
        </OuiStack>
      </OuiFlex>

      <OuiGrid :cols="{ sm: 1, lg: 3 }" gap="lg">
        <!-- User Info Card -->
        <OuiCard class="border border-border-muted rounded-xl">
          <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
            <OuiText tag="h2" size="lg" weight="bold">User Information</OuiText>
          </OuiCardHeader>
          <OuiCardBody class="p-6">
            <OuiStack gap="lg">
              <OuiFlex gap="md" align="center">
                <OuiAvatar
                  :name="user?.name || user?.email || user?.id || ''"
                  :src="user?.avatarUrl"
                  size="xl"
                />
                <OuiStack gap="xs">
                  <OuiText size="xl" weight="bold">
                    {{ user?.name || user?.email || user?.id || "Loading..." }}
                  </OuiText>
                  <OuiText v-if="user?.email" color="tertiary" size="sm">
                    {{ user.email }}
                  </OuiText>
                  <OuiText v-if="user?.id" color="tertiary" size="xs" class="font-mono">
                    {{ user.id }}
                  </OuiText>
                </OuiStack>
              </OuiFlex>

              <OuiStack gap="md" class="border-t border-border-muted pt-4">
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Email</OuiText>
                  <OuiText>{{ user?.email || "—" }}</OuiText>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Username</OuiText>
                  <OuiText>{{ user?.preferredUsername || "—" }}</OuiText>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Locale</OuiText>
                  <OuiText>{{ user?.locale || "—" }}</OuiText>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Email Verified</OuiText>
                  <OuiBadge
                    :variant="user?.emailVerified ? 'success' : 'secondary'"
                    :tone="user?.emailVerified ? 'solid' : 'soft'"
                  >
                    {{ user?.emailVerified ? "Verified" : "Not Verified" }}
                  </OuiBadge>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Roles</OuiText>
                  <OuiFlex gap="xs" wrap="wrap">
                    <OuiBadge
                      v-for="role in user?.roles"
                      :key="role"
                      variant="primary"
                      tone="soft"
                      size="sm"
                    >
                      {{ role }}
                    </OuiBadge>
                    <OuiText v-if="!user?.roles?.length" color="tertiary" size="sm">
                      No roles assigned
                    </OuiText>
                  </OuiFlex>
                </OuiStack>

                <OuiStack gap="xs" v-if="user?.createdAt">
                  <OuiText size="sm" weight="medium" color="tertiary">Created</OuiText>
                  <OuiText size="sm">
                    <OuiDate :value="user.createdAt" />
                  </OuiText>
                </OuiStack>

                <OuiStack gap="xs" v-if="user?.updatedAt">
                  <OuiText size="sm" weight="medium" color="tertiary">Last Updated</OuiText>
                  <OuiText size="sm">
                    <OuiDate :value="user.updatedAt" />
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Organizations Card -->
        <OuiCard class="border border-border-muted rounded-xl col-span-1 lg:col-span-2">
          <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
            <OuiText tag="h2" size="lg" weight="bold">Organizations</OuiText>
          </OuiCardHeader>
          <OuiCardBody class="p-0">
            <OuiTable
              v-if="!loading"
              :columns="orgColumns"
              :rows="tableOrgs"
              empty-text="This user is not a member of any organizations."
              row-class="hover:bg-surface-subtle/50"
              @row-click="(row) => viewOrganization(row.organizationId)"
            >
              <template #cell-organization="{ row }">
                <NuxtLink
                  :to="`/organizations?organizationId=${row.organizationId}`"
                  class="font-medium text-text-primary hover:text-primary transition-colors cursor-pointer"
                >
                  {{ row.organizationName }}
                </NuxtLink>
                <OuiText color="tertiary" size="xs" class="font-mono">
                  {{ row.organizationId }}
                </OuiText>
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
              <template #cell-actions="{ row }">
                <OuiButton
                  size="sm"
                  variant="ghost"
                  @click.stop="viewOrganization(row.organizationId)"
                >
                  View
                </OuiButton>
              </template>
            </OuiTable>
            <div v-if="loading" class="p-6 text-center text-text-muted">
              Loading organizations...
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
              v-if="banStatus?.isActive && banStatus.type === 'banned'"
              variant="danger"
              tone="solid"
            >BANNED</OuiBadge>
            <OuiBadge
              v-else-if="banStatus?.isActive && banStatus.type === 'suspended'"
              variant="warning"
              tone="solid"
            >SUSPENDED</OuiBadge>
            <OuiBadge v-else variant="success" tone="soft">No active ban</OuiBadge>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody class="p-6">
          <OuiStack gap="lg">
            <!-- Active ban details -->
            <OuiStack v-if="banStatus?.isActive" gap="md" class="p-4 rounded-lg bg-surface-subtle border border-border-muted">
              <OuiGrid :cols="{ sm: 1, md: 3 }" gap="md">
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Type</OuiText>
                  <OuiBadge
                    :variant="banStatus.type === 'banned' ? 'danger' : 'warning'"
                    tone="soft"
                    size="sm"
                  >
                    {{ banStatus.type === 'banned' ? 'Permanent Ban' : 'Suspension' }}
                  </OuiBadge>
                </OuiStack>
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Issued By</OuiText>
                  <OuiText size="sm" class="font-mono">{{ banStatus.bannedBy || "—" }}</OuiText>
                </OuiStack>
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Issued At</OuiText>
                  <OuiText size="sm">
                    <OuiDate v-if="banStatus.bannedAt" :value="banStatus.bannedAt" />
                    <span v-else>—</span>
                  </OuiText>
                </OuiStack>
                <OuiStack v-if="banStatus.reason" gap="xs" class="md:col-span-3">
                  <OuiText size="sm" weight="medium" color="tertiary">Reason</OuiText>
                  <OuiText size="sm">{{ banStatus.reason }}</OuiText>
                </OuiStack>
                <OuiStack v-if="banStatus.expiresAt" gap="xs">
                  <OuiText size="sm" weight="medium" color="tertiary">Expires</OuiText>
                  <OuiText size="sm">
                    <OuiDate :value="banStatus.expiresAt" />
                  </OuiText>
                </OuiStack>
              </OuiGrid>
            </OuiStack>

            <!-- Action buttons -->
            <OuiFlex gap="sm" wrap="wrap">
              <template v-if="!banStatus?.isActive">
                <OuiButton
                  color="warning"
                  variant="outline"
                  size="sm"
                  @click="openSuspendDialog"
                  :disabled="isModerating"
                >
                  Suspend User
                </OuiButton>
                <OuiButton
                  color="danger"
                  size="sm"
                  @click="openBanDialog"
                  :disabled="isModerating"
                >
                  Ban User
                </OuiButton>
              </template>
              <template v-else>
                <OuiButton
                  color="primary"
                  size="sm"
                  @click="handleLiftBan"
                  :disabled="isModerating"
                >
                  {{ isModerating ? 'Lifting...' :banStatus.type === 'banned' ? 'Unban User' : 'Lift Suspension' }}
                </OuiButton>
              </template>
            </OuiFlex>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>
  </OuiContainer>

  <!-- Suspend Dialog -->
  <OuiDialog v-model:open="suspendDialogOpen" title="Suspend User">
    <OuiStack gap="lg">
      <OuiText size="sm" color="tertiary">
        Suspend this user account. They will be unable to access the platform until the suspension is lifted.
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
  <OuiDialog v-model:open="banDialogOpen" title="Ban User">
    <OuiStack gap="lg">
      <OuiText size="sm" color="tertiary">
        Permanently ban this user from the platform. This is a serious action.
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
          {{ isModerating ? 'Banning...' : 'Ban User' }}
        </OuiButton>
      </OuiFlex>
    </template>
  </OuiDialog>
</template>

<script setup lang="ts">
import { ArrowLeftIcon } from "@heroicons/vue/24/outline";
import { SuperadminService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";

definePageMeta({
  middleware: ["auth", "superadmin"],
});

const route = useRoute();
const router = useRouter();
const client = useConnectClient(SuperadminService);
const { toast } = useToast();

const userId = computed(() => route.params.userId as string);
const user = ref<any>(null);
const organizations = ref<any[]>([]);
const banStatus = ref<any>(null);

const orgColumns = [
  { key: "organization", label: "Organization", defaultWidth: 250, minWidth: 200 },
  { key: "role", label: "Role", defaultWidth: 120, minWidth: 100 },
  { key: "status", label: "Status", defaultWidth: 120, minWidth: 100 },
  { key: "joined", label: "Joined", defaultWidth: 180, minWidth: 150 },
  { key: "actions", label: "Actions", defaultWidth: 100, minWidth: 80, resizable: false },
];

const tableOrgs = computed(() => organizations.value);

function viewOrganization(orgId: string) {
  router.push({
    path: "/organizations",
    query: { organizationId: orgId },
  });
}

async function loadUser() {
  if (!userId.value) return null;
  try {
    const response = await client.getUser({
      userId: userId.value,
    });
    return {
      user: response.user,
      organizations: response.organizations || [],
    };
  } catch (error: unknown) {
    console.error("Failed to load user:", error);
    const { toast } = useToast();
    toast.error((error as Error | undefined)?.message || "Failed to load user");
    throw error;
  }
}

const { data: userData, pending: loading } = useClientFetch(
  () => `superadmin-user-${userId.value}`,
  loadUser
);

watch(userData, (newData) => {
  if (newData) {
    user.value = newData.user;
    organizations.value = newData.organizations;
  }
}, { immediate: true });

// Load ban status
const loadBanStatus = async () => {
  if (!userId.value) return;
  try {
    const response = await client.getUserBanStatus({ userId: userId.value });
    banStatus.value = response.ban || null;
  } catch {
    banStatus.value = null;
  }
};

watch(userId, () => loadBanStatus(), { immediate: true });

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
  if (!userId.value) return;
  isModerating.value = true;
  try {
    await client.suspendUser({
      userId: userId.value,
      reason: moderationForm.value.reason || undefined,
      expiresAt: moderationForm.value.expiresAt
        ? { seconds: BigInt(Math.floor(new Date(moderationForm.value.expiresAt).getTime() / 1000)), nanos: 0 }
        : undefined,
    });
    toast.success("User suspended.");
    suspendDialogOpen.value = false;
    await loadBanStatus();
  } catch (error: unknown) {
    toast.error(`Failed to suspend: ${(error as any)?.message || "Unknown error"}`);
  } finally {
    isModerating.value = false;
  }
};

const handleBan = async () => {
  if (!userId.value) return;
  isModerating.value = true;
  try {
    await client.banUser({
      userId: userId.value,
      reason: moderationForm.value.reason || undefined,
    });
    toast.success("User banned.");
    banDialogOpen.value = false;
    await loadBanStatus();
  } catch (error: unknown) {
    toast.error(`Failed to ban: ${(error as any)?.message || "Unknown error"}`);
  } finally {
    isModerating.value = false;
  }
};

const handleLiftBan = async () => {
  if (!userId.value || !banStatus.value) return;
  isModerating.value = true;
  try {
    if (banStatus.value.type === "banned") {
      await client.unbanUser({ userId: userId.value });
      toast.success("User unbanned.");
    } else {
      await client.unsuspendUser({ userId: userId.value });
      toast.success("Suspension lifted.");
    }
    await loadBanStatus();
  } catch (error: unknown) {
    toast.error(`Failed to lift ban: ${(error as any)?.message || "Unknown error"}`);
  } finally {
    isModerating.value = false;
  }
};
</script>

