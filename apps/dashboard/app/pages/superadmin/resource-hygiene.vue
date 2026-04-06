<template>
  <OuiStack gap="xl">
    <OuiGrid :cols="{ sm: 1, md: 2, xl: 4 }" gap="md">
      <OuiCard
        v-for="metric in metrics"
        :key="metric.label"
        class="border border-border-muted rounded-xl"
      >
        <OuiCardBody class="p-5">
          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium" color="tertiary">{{
              metric.label
            }}</OuiText>
            <OuiText size="2xl" weight="semibold">{{ metric.value }}</OuiText>
            <OuiText size="xs" color="tertiary">{{ metric.help }}</OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiGrid>

    <SuperadminPageLayout
      title="Resource Hygiene"
      description="Find inactive users who still have retained infrastructure consuming provisioned capacity."
      :columns="columns"
      :rows="tableRows"
      :filters="filterConfigs"
      :search="search"
      :empty-text="
        loading
          ? 'Loading dormant resource owners…'
          : 'No dormant resource owners match your filters.'
      "
      :loading="loading"
      :pagination="{
        page: pagination.page,
        totalPages: pagination.totalPages,
        total: pagination.total,
        perPage: pagination.perPage,
      }"
      search-placeholder="Search by user, email, username, organization, or ID…"
      @update:search="handleSearchUpdate"
      @filter-change="handleFilterChange"
      @refresh="refreshOwners"
      @row-click="(row) => viewUser(row.userId)"
      @page-change="goToPage"
    >
      <template #cell-user="{ row }">
        <OuiFlex gap="sm" align="center">
          <OuiAvatar
            :name="row.name || row.email || row.userId"
            :src="row.avatarUrl"
          />
          <OuiStack gap="xs">
            <SuperadminResourceCell
              :name="row.name || row.email || row.userId"
              :subtitle="
                row.email ||
                row.preferredUsername ||
                row.lastActivitySourceLabel
              "
              :id="row.userId"
            />
            <OuiFlex gap="xs" wrap="wrap">
              <OuiBadge
                v-for="role in row.roles"
                :key="role"
                variant="primary"
                tone="soft"
                size="sm"
              >
                {{ role }}
              </OuiBadge>
            </OuiFlex>
          </OuiStack>
        </OuiFlex>
      </template>

      <template #cell-lastActivity="{ row }">
        <OuiStack gap="xs">
          <OuiText size="sm">
            {{ row.lastActivityLabel }}
          </OuiText>
          <OuiText size="xs" color="tertiary">
            {{ row.lastActivitySourceLabel }}
          </OuiText>
        </OuiStack>
      </template>

      <template #cell-inactive="{ row }">
        <OuiStack gap="xs">
          <OuiText size="sm" weight="semibold">
            {{ row.inactiveLabel }}
          </OuiText>
          <OuiText size="xs" color="tertiary">
            Last resource change: {{ row.lastResourceLabel }}
          </OuiText>
        </OuiStack>
      </template>

      <template #cell-resources="{ row }">
        <OuiFlex gap="xs" wrap="wrap">
          <OuiBadge
            v-for="resource in row.resources"
            :key="resource.key"
            variant="secondary"
            tone="soft"
            size="sm"
          >
            {{ resource.label }}
          </OuiBadge>
          <OuiText v-if="!row.resources.length" size="sm" color="tertiary"
            >—</OuiText
          >
        </OuiFlex>
      </template>

      <template #cell-storage="{ row }">
        <OuiText size="sm" weight="medium">{{
          row.totalReservedBytesLabel
        }}</OuiText>
      </template>

      <template #cell-organizations="{ row }">
        <OuiStack gap="xs">
          <OuiFlex gap="xs" wrap="wrap">
            <OuiBadge
              v-for="org in row.organizationsPreview"
              :key="org.organizationId"
              variant="secondary"
              tone="soft"
              size="sm"
            >
              {{ org.label }}
            </OuiBadge>
          </OuiFlex>
          <OuiText
            v-if="row.remainingOrganizations > 0"
            size="xs"
            color="tertiary"
          >
            +{{ row.remainingOrganizations }} more organization{{
              row.remainingOrganizations === 1 ? "" : "s"
            }}
          </OuiText>
        </OuiStack>
      </template>

      <template #cell-actions="{ row }">
        <SuperadminActionsCell :actions="getActions(row)" />
      </template>
    </SuperadminPageLayout>
  </OuiStack>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import type {
  DormantResourceOwner,
  DormantResourceSummary,
} from "@obiente/proto";
import { SuperadminService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { formatBytes } from "~/utils/common";
import SuperadminPageLayout from "~/components/superadmin/SuperadminPageLayout.vue";
import SuperadminResourceCell from "~/components/superadmin/SuperadminResourceCell.vue";
import SuperadminActionsCell, {
  type Action,
} from "~/components/superadmin/SuperadminActionsCell.vue";
import type { FilterConfig } from "~/components/superadmin/SuperadminFilterBar.vue";

definePageMeta({
  middleware: ["auth", "superadmin"],
});

const client = useConnectClient(SuperadminService);
const router = useRouter();
const { toast } = useToast();

const owners = ref<DormantResourceOwner[]>([]);
const summary = ref<DormantResourceSummary | null>(null);
const pagination = ref({
  page: 1,
  perPage: 25,
  total: 0,
  totalPages: 0,
});
const search = ref("");
const minInactiveDays = ref("30");
let searchTimeout: ReturnType<typeof setTimeout> | null = null;

const columns = [
  { key: "user", label: "User", defaultWidth: 280, minWidth: 240 },
  {
    key: "lastActivity",
    label: "Last Activity",
    defaultWidth: 180,
    minWidth: 150,
  },
  { key: "inactive", label: "Inactivity", defaultWidth: 200, minWidth: 170 },
  {
    key: "resources",
    label: "Retained Resources",
    defaultWidth: 260,
    minWidth: 220,
  },
  {
    key: "storage",
    label: "Reserved Capacity",
    defaultWidth: 160,
    minWidth: 140,
  },
  {
    key: "organizations",
    label: "Organizations",
    defaultWidth: 260,
    minWidth: 220,
  },
  {
    key: "actions",
    label: "Actions",
    defaultWidth: 110,
    minWidth: 90,
    resizable: false,
  },
] as const;

const filterConfigs = computed(
  () =>
    [
      {
        key: "minInactiveDays",
        placeholder: "Inactivity",
        items: [
          { key: "30", label: "30+ days", value: "30" },
          { key: "60", label: "60+ days", value: "60" },
          { key: "90", label: "90+ days", value: "90" },
          { key: "180", label: "180+ days", value: "180" },
        ],
      },
    ] satisfies FilterConfig[]
);

function formatTimestamp(
  timestamp?: { seconds?: bigint | number; nanos?: number } | null
) {
  if (!timestamp?.seconds) {
    return "—";
  }
  const seconds =
    typeof timestamp.seconds === "bigint"
      ? Number(timestamp.seconds)
      : timestamp.seconds;
  const millis =
    seconds * 1000 + Math.floor((timestamp.nanos ?? 0) / 1_000_000);
  const date = new Date(millis);
  if (Number.isNaN(date.getTime())) {
    return "—";
  }
  return date.toLocaleDateString() + " " + date.toLocaleTimeString();
}

function formatInactiveDays(days?: number) {
  if (!days || days <= 0) {
    return "Active today";
  }
  if (days === 1) {
    return "1 day inactive";
  }
  return `${days} days inactive`;
}

function formatLastActivitySource(source?: string) {
  switch (source) {
    case "audit_log":
      return "Last seen in audit logs";
    case "profile_updated":
      return "Last seen from profile update";
    case "profile_created":
      return "Only profile creation is known";
    case "resource_updated":
      return "Fell back to resource update timestamp";
    case "resource_created":
      return "Fell back to resource creation timestamp";
    default:
      return "No direct activity signal";
  }
}

const metrics = computed(() => [
  {
    label: "Dormant Users",
    value: String(summary.value?.dormantUsers ?? 0),
    help: `Users inactive for at least ${minInactiveDays.value} days with retained resources.`,
  },
  {
    label: "Users With VPS",
    value: String(summary.value?.usersWithVps ?? 0),
    help: "Dormant users who still have at least one VPS instance.",
  },
  {
    label: "Users With Databases",
    value: String(summary.value?.usersWithDatabases ?? 0),
    help: "Dormant users who still have at least one managed database.",
  },
  {
    label: "Reserved Capacity",
    value: formatBytes(summary.value?.totalReservedBytes ?? 0),
    help: "Provisioned storage and disk still reserved across dormant users.",
  },
]);

const tableRows = computed(() =>
  (owners.value || []).map((owner) => {
    const organizations = owner.organizations || [];
    const organizationsPreview = organizations
      .slice(0, 3)
      .map((organization) => {
        const resourceLabels = [
          organization.vpsCount ? `${organization.vpsCount} VPS` : null,
          organization.databaseCount
            ? `${organization.databaseCount} DB`
            : null,
          organization.deploymentCount
            ? `${organization.deploymentCount} Deploy`
            : null,
          organization.gameServerCount
            ? `${organization.gameServerCount} GS`
            : null,
        ]
          .filter(Boolean)
          .join(" • ");

        return {
          organizationId: organization.organizationId,
          label: resourceLabels
            ? `${organization.organizationName} (${resourceLabels})`
            : organization.organizationName,
        };
      });

    const resources = [
      owner.vpsCount ? { key: "vps", label: `${owner.vpsCount} VPS` } : null,
      owner.databaseCount
        ? { key: "db", label: `${owner.databaseCount} DB` }
        : null,
      owner.deploymentCount
        ? { key: "deployments", label: `${owner.deploymentCount} Deployments` }
        : null,
      owner.gameServerCount
        ? { key: "gameservers", label: `${owner.gameServerCount} Game Servers` }
        : null,
    ].filter(Boolean) as { key: string; label: string }[];

    return {
      userId: owner.user?.id || "",
      name: owner.user?.name || "",
      email: owner.user?.email || "",
      preferredUsername: owner.user?.preferredUsername || "",
      avatarUrl: owner.user?.avatarUrl,
      roles: owner.user?.roles || [],
      lastActivityLabel: formatTimestamp(owner.lastActivityAt),
      lastActivitySourceLabel: formatLastActivitySource(
        owner.lastActivitySource
      ),
      inactiveLabel: formatInactiveDays(owner.inactiveDays),
      lastResourceLabel: formatTimestamp(
        owner.lastResourceUpdatedAt || owner.lastResourceCreatedAt
      ),
      resources,
      totalReservedBytesLabel: formatBytes(owner.totalReservedBytes),
      organizationsPreview,
      remainingOrganizations: Math.max(
        organizations.length - organizationsPreview.length,
        0
      ),
      organizations,
    };
  })
);

function handleSearchUpdate(value: string) {
  search.value = value;
  if (searchTimeout) {
    clearTimeout(searchTimeout);
  }
  searchTimeout = setTimeout(() => {
    pagination.value.page = 1;
    refreshOwners();
  }, 250);
}

function handleFilterChange(key: string, value: string) {
  if (key === "minInactiveDays") {
    minInactiveDays.value = value || "30";
    pagination.value.page = 1;
    refreshOwners();
  }
}

async function fetchOwners() {
  try {
    const response = await client.listDormantResourceOwners({
      page: pagination.value.page,
      perPage: pagination.value.perPage,
      search: search.value || undefined,
      minInactiveDays: Number(minInactiveDays.value || "30"),
    });

    return {
      owners: response.owners || [],
      summary: response.summary || null,
      pagination: {
        page: response.pagination?.page || 1,
        perPage: response.pagination?.perPage || 25,
        total: response.pagination?.total || 0,
        totalPages: response.pagination?.totalPages || 0,
      },
    };
  } catch (error: unknown) {
    console.error("Failed to load dormant resource owners:", error);
    toast.error(
      (error as Error | undefined)?.message ||
        "Failed to load resource hygiene report"
    );
    throw error;
  }
}

const {
  data: ownersData,
  pending: loading,
  refresh: refreshOwners,
} = useClientFetch(
  () =>
    `superadmin-resource-hygiene-${pagination.value.page}-${minInactiveDays.value}-${search.value}`,
  fetchOwners
);

watch(
  ownersData,
  (newData) => {
    if (!newData) {
      return;
    }
    owners.value = newData.owners;
    summary.value = newData.summary;
    pagination.value = newData.pagination;
  },
  { immediate: true }
);

function goToPage(page: number) {
  pagination.value.page = page;
  refreshOwners();
}

function viewUser(userId: string) {
  router.push(`/superadmin/users/${userId}`);
}

function viewOrganization(organizationId: string) {
  router.push(`/superadmin/organizations/${organizationId}`);
}

function getActions(row: {
  userId: string;
  organizations: Array<{ organizationId: string }>;
}): Action[] {
  const actions: Action[] = [
    {
      label: "View User",
      onClick: () => viewUser(row.userId),
    },
  ];
  if (row.organizations.length === 1) {
    actions.push({
      label: "Open Org",
      onClick: () =>
        viewOrganization(row.organizations[0]?.organizationId || ""),
    });
  }
  return actions;
}
</script>
