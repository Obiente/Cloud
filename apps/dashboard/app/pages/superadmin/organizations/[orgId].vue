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
          <OuiText color="muted">View detailed information about this organization.</OuiText>
        </OuiStack>
      </OuiFlex>

      <OuiGrid cols="1" colsLg="3" gap="lg">
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
                <OuiText v-if="organization?.slug" color="muted" size="sm">
                  {{ organization.slug }}
                </OuiText>
                <OuiText v-if="organization?.id" color="muted" size="xs" class="font-mono">
                  {{ organization.id }}
                </OuiText>
              </OuiStack>

              <OuiStack gap="md" class="border-t border-border-muted pt-4">
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="muted">Plan</OuiText>
                  <OuiBadge variant="secondary" tone="soft" size="sm">
                    {{ prettyPlan(organization?.plan) }}
                  </OuiBadge>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="muted">Status</OuiText>
                  <OuiBadge
                    :variant="organization?.status === 'active' ? 'success' : organization?.status === 'suspended' ? 'warning' : 'danger'"
                    :tone="organization?.status === 'active' ? 'solid' : 'soft'"
                    size="sm"
                  >
                    {{ organization?.status?.toUpperCase() || "—" }}
                  </OuiBadge>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="muted">Domain</OuiText>
                  <OuiText>{{ organization?.domain || "—" }}</OuiText>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="muted">Credits</OuiText>
                  <OuiText size="sm" weight="semibold">
                    <OuiCurrency :value="Number(organization?.credits || 0)" />
                  </OuiText>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="muted">Total Paid</OuiText>
                  <OuiText size="sm">
                    <OuiCurrency :value="Number(organization?.totalPaidCents || 0)" />
                  </OuiText>
                </OuiStack>

                <OuiStack gap="xs" v-if="organization?.planInfo">
                  <OuiText size="sm" weight="medium" color="muted">Plan Details</OuiText>
                  <OuiStack gap="xs">
                    <OuiText size="xs" color="muted">
                      CPU: {{ organization.planInfo.cpuCores || 'Unlimited' }} cores
                    </OuiText>
                    <OuiText size="xs" color="muted">
                      Memory: {{ formatBytes(Number(organization.planInfo.memoryBytes || 0)) }}
                    </OuiText>
                    <OuiText size="xs" color="muted">
                      Max Deployments: {{ organization.planInfo.deploymentsMax || 'Unlimited' }}
                    </OuiText>
                    <OuiText size="xs" color="muted">
                      Max VPS: {{ organization.planInfo.maxVpsInstances || 'Unlimited' }}
                    </OuiText>
                  </OuiStack>
                </OuiStack>

                <OuiStack gap="xs" v-if="organization?.createdAt">
                  <OuiText size="sm" weight="medium" color="muted">Created</OuiText>
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
                    <OuiText v-if="row.user?.email" color="muted" size="xs">
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
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
import { ArrowLeftIcon } from "@heroicons/vue/24/outline";
import { OrganizationService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useUtils } from "~/composables/useUtils";

definePageMeta({
  middleware: ["auth", "superadmin"],
});

const route = useRoute();
const router = useRouter();
const orgClient = useConnectClient(OrganizationService);
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
  } catch (error: any) {
    console.error("Failed to load organization:", error);
    const { toast } = useToast();
    toast.error(error?.message || "Failed to load organization");
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
</script>

