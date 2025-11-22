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
          <OuiText color="muted">View detailed information about this user.</OuiText>
        </OuiStack>
      </OuiFlex>

      <OuiGrid cols="1" colsLg="3" gap="lg">
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
                  <OuiText v-if="user?.email" color="muted" size="sm">
                    {{ user.email }}
                  </OuiText>
                  <OuiText v-if="user?.id" color="muted" size="xs" class="font-mono">
                    {{ user.id }}
                  </OuiText>
                </OuiStack>
              </OuiFlex>

              <OuiStack gap="md" class="border-t border-border-muted pt-4">
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="muted">Email</OuiText>
                  <OuiText>{{ user?.email || "—" }}</OuiText>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="muted">Username</OuiText>
                  <OuiText>{{ user?.preferredUsername || "—" }}</OuiText>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="muted">Locale</OuiText>
                  <OuiText>{{ user?.locale || "—" }}</OuiText>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="muted">Email Verified</OuiText>
                  <OuiBadge
                    :variant="user?.emailVerified ? 'success' : 'secondary'"
                    :tone="user?.emailVerified ? 'solid' : 'soft'"
                  >
                    {{ user?.emailVerified ? "Verified" : "Not Verified" }}
                  </OuiBadge>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="muted">Roles</OuiText>
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
                    <OuiText v-if="!user?.roles?.length" color="muted" size="sm">
                      No roles assigned
                    </OuiText>
                  </OuiFlex>
                </OuiStack>

                <OuiStack gap="xs" v-if="user?.createdAt">
                  <OuiText size="sm" weight="medium" color="muted">Created</OuiText>
                  <OuiText size="sm">
                    <OuiDate :value="user.createdAt" />
                  </OuiText>
                </OuiStack>

                <OuiStack gap="xs" v-if="user?.updatedAt">
                  <OuiText size="sm" weight="medium" color="muted">Last Updated</OuiText>
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
                <OuiText color="muted" size="xs" class="font-mono">
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
    </OuiStack>
  </OuiContainer>
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

const userId = computed(() => route.params.userId as string);
const user = ref<any>(null);
const organizations = ref<any[]>([]);

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
  } catch (error: any) {
    console.error("Failed to load user:", error);
    const { toast } = useToast();
    toast.error(error?.message || "Failed to load user");
    throw error;
  }
}

// Use client-side fetching for non-blocking navigation
const { data: userData, pending: loading } = useClientFetch(
  () => `superadmin-user-${userId.value}`,
  loadUser
);

// Update refs when data is loaded
watch(userData, (newData) => {
  if (newData) {
    user.value = newData.user;
    organizations.value = newData.organizations;
  }
}, { immediate: true });
</script>

