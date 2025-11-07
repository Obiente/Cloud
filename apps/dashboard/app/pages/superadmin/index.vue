<template>
  <OuiContainer size="full">
    <OuiStack gap="2xl">
      <OuiStack gap="xs">
        <OuiText tag="h1" size="3xl" weight="extrabold"
          >Superadmin Overview</OuiText
        >
        <OuiText color="muted"
          >System-wide visibility across organizations, deployments, and
          usage.</OuiText
        >
      </OuiStack>

      <OuiGrid class="gap-4" cols="1" colsMd="2" colsXl="4">
        <OuiCard
          v-for="metric in metrics"
          :key="metric.label"
          class="p-6 bg-surface-raised border border-border-muted rounded-xl"
        >
          <OuiStack gap="xs">
            <OuiText
              size="sm"
              weight="medium"
              color="secondary"
              transform="uppercase"
              class="tracking-wide"
              >{{ metric.label }}</OuiText
            >
            <OuiText size="3xl" weight="semibold" color="primary">{{
              metric.value
            }}</OuiText>
            <OuiText size="xs" color="muted">{{ metric.help }}</OuiText>
          </OuiStack>
        </OuiCard>
      </OuiGrid>

      <!-- Version Information -->
      <OuiCard class="border border-border-muted rounded-xl">
        <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
          <OuiText tag="h2" size="xl" weight="bold">Version Information</OuiText>
        </OuiCardHeader>
        <OuiCardBody class="p-6">
          <OuiStack gap="md">
            <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium" color="secondary">API Commit</OuiText>
                <OuiText size="md" class="font-mono">
                  {{ overview?.apiCommit || "—" }}
                </OuiText>
              </OuiStack>
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium" color="secondary">Dashboard Commit</OuiText>
                <OuiText size="md" class="font-mono">
                  {{ overview?.dashboardCommit || "—" }}
                </OuiText>
              </OuiStack>
            </OuiFlex>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <OuiStack gap="lg">
        <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
          <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
            <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
              <OuiStack gap="xs">
                <OuiText tag="h2" size="xl" weight="bold"
                  >Organizations</OuiText
                >
                <OuiText color="muted" size="sm"
                  >{{ filteredOrganizations.length }} of
                  {{ organizations.length }} organizations</OuiText
                >
              </OuiStack>
              <OuiContainer size="sm" class="w-64">
                <OuiInput
                  v-model="orgSearch"
                  type="search"
                  placeholder="Search organizations…"
                  clearable
                  size="sm"
                />
              </OuiContainer>
            </OuiFlex>
          </OuiCardHeader>
          <OuiCardBody class="p-0">
            <OuiTable
              :columns="orgColumns"
              :rows="tableOrgs"
              :empty-text="organizations.length === 0 ? 'No organizations available.' : 'No organizations match your search.'"
            >
              <template #cell-name="{ row }">
                <div class="font-medium text-text-primary">
                  {{ row.name || row.slug || "—" }}
                </div>
                <div class="text-xs text-text-muted">
                  <span v-if="row.slug">{{ row.slug }}</span>
                  <span v-else class="text-text-tertiary">No slug</span>
                </div>
                <div class="text-xs font-mono text-text-tertiary mt-0.5">
                  {{ row.id }}
                </div>
                <div
                  v-if="row.domain"
                  class="text-xs text-text-muted mt-0.5"
                >
                  {{ row.domain }}
                </div>
              </template>
              <template #cell-plan="{ row }">
                {{ prettyPlan(row.plan) }}
              </template>
              <template #cell-members="{ value }">
                {{ formatNumber(value) }}
              </template>
              <template #cell-invites="{ value }">
                {{ formatNumber(value) }}
              </template>
              <template #cell-deployments="{ value }">
                {{ formatNumber(value) }}
              </template>
              <template #cell-created="{ value }">
                <OuiDate :value="value" />
              </template>
            </OuiTable>
          </OuiCardBody>
        </OuiCard>

        <OuiGrid class="gap-6" cols="1" colsXl="2">
          <OuiCard
            class="border border-border-muted rounded-xl overflow-hidden"
          >
            <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
              <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
                <OuiStack gap="xs">
                  <OuiText tag="h3" size="lg" weight="semibold"
                    >Pending Invites</OuiText
                  >
                  <OuiText color="muted" size="sm"
                    >{{ filteredInvites.length }} of
                    {{ invites.length }} invites</OuiText
                  >
                </OuiStack>
                <OuiContainer size="sm" class="w-56">
                  <OuiInput
                    v-model="inviteSearch"
                    type="search"
                    placeholder="Search invites…"
                    clearable
                    size="sm"
                  />
                </OuiContainer>
              </OuiFlex>
            </OuiCardHeader>
            <OuiCardBody class="p-0">
              <OuiTable
                :columns="inviteColumns"
                :rows="tableInvites"
                :empty-text="invites.length === 0 ? 'No pending invitations.' : 'No invites match your search.'"
              >
                <template #cell-email="{ row }">
                  <div class="text-text-primary">{{ row.email }}</div>
                  <div
                    class="text-xs font-mono text-text-tertiary mt-0.5"
                  >
                    {{ row.id }}
                  </div>
                </template>
                <template #cell-organization="{ value }">
                  <div class="text-text-secondary font-mono text-sm">
                    {{ value }}
                  </div>
                </template>
                <template #cell-role="{ value }">
                  <span class="text-text-secondary uppercase tracking-wide text-xs">
                    {{ value }}
                  </span>
                </template>
                <template #cell-invited="{ value }">
                  <OuiDate :value="value" />
                </template>
              </OuiTable>
            </OuiCardBody>
          </OuiCard>

          <OuiCard
            class="border border-border-muted rounded-xl overflow-hidden"
          >
            <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
              <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
                <OuiStack gap="xs">
                  <OuiText tag="h3" size="lg" weight="semibold"
                    >Recent Deployments</OuiText
                  >
                  <OuiText color="muted" size="sm"
                    >{{ filteredDeployments.length }} of
                    {{ deployments.length }} deployments</OuiText
                  >
                </OuiStack>
                <OuiContainer size="sm" class="w-56">
                  <OuiInput
                    v-model="deploymentSearch"
                    type="search"
                    placeholder="Search deployments…"
                    clearable
                    size="sm"
                  />
                </OuiContainer>
              </OuiFlex>
            </OuiCardHeader>
            <OuiCardBody class="p-0">
              <OuiTable
                :columns="deploymentColumns"
                :rows="tableDeployments"
                :empty-text="deployments.length === 0 ? 'No deployments found.' : 'No deployments match your search.'"
              >
                <template #cell-deployment="{ row }">
                  <div class="font-medium text-text-primary">
                    {{ row.name }}
                  </div>
                  <div
                    v-if="row.domain"
                    class="text-xs text-text-muted"
                  >
                    {{ row.domain }}
                  </div>
                  <div
                    class="text-xs font-mono text-text-tertiary mt-0.5"
                  >
                    {{ row.id }}
                  </div>
                </template>
                <template #cell-organization="{ value }">
                  <div class="text-text-secondary font-mono text-sm">
                    {{ value }}
                  </div>
                </template>
                <template #cell-environment="{ row }">
                  <span class="text-text-secondary uppercase text-xs">
                    {{ formatEnvironment(row.environment) }}
                  </span>
                </template>
                <template #cell-status="{ row }">
                  <span class="text-text-secondary uppercase text-xs">
                    {{ formatStatus(row.status) }}
                  </span>
                </template>
                <template #cell-lastDeployed="{ row }">
                  <OuiDate :value="row.lastDeployedAt || row.createdAt" />
                </template>
              </OuiTable>
            </OuiCardBody>
          </OuiCard>
        </OuiGrid>

        <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
          <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
            <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
              <OuiStack gap="xs">
                <OuiText tag="h2" size="xl" weight="bold"
                  >Current Month Usage</OuiText
                >
                <OuiText color="muted" size="sm"
                  >{{ filteredUsages.length }} of
                  {{ usages.length }} organizations</OuiText
                >
              </OuiStack>
              <OuiContainer size="sm" class="w-64">
                <OuiInput
                  v-model="usageSearch"
                  type="search"
                  placeholder="Search usage…"
                  clearable
                  size="sm"
                />
              </OuiContainer>
            </OuiFlex>
          </OuiCardHeader>
          <OuiCardBody class="p-0">
            <OuiTable
              :columns="usageColumns"
              :rows="tableUsages"
              :empty-text="usages.length === 0 ? 'No usage records for the current month.' : 'No usage records match your search.'"
            >
              <template #cell-organization="{ row }">
                <div class="font-medium text-text-primary">
                  {{ row.organizationName || "—" }}
                </div>
                <div class="text-xs font-mono text-text-tertiary mt-0.5">
                  {{ row.organizationId }}
                </div>
                <div class="text-xs text-text-muted mt-0.5">
                  {{ row.month }}
                </div>
              </template>
              <template #cell-cpu="{ value }">
                {{ formatNumber(value) }}
              </template>
              <template #cell-memory="{ value }">
                {{ formatNumber(value) }}
              </template>
              <template #cell-bandwidthRx="{ value }">
                {{ formatBytes(value) }}
              </template>
              <template #cell-bandwidthTx="{ value }">
                {{ formatBytes(value) }}
              </template>
              <template #cell-storage="{ value }">
                {{ formatBytes(value) }}
              </template>
              <template #cell-peakDeployments="{ value }">
                {{ formatNumber(value) }}
              </template>
            </OuiTable>
          </OuiCardBody>
        </OuiCard>
      </OuiStack>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
  definePageMeta({
    middleware: ["auth", "superadmin"],
  });

  const superAdmin = useSuperAdmin();
  await superAdmin.fetchOverview();

  const overview = computed(() => superAdmin.overview.value);
  const organizations = computed(() => overview.value?.organizations ?? []);
  const invites = computed(() => overview.value?.pendingInvites ?? []);
  const deployments = computed(() => overview.value?.deployments ?? []);
  const usages = computed(() => overview.value?.usages ?? []);

  // Search filters for overview page
  const orgSearch = ref("");
  const inviteSearch = ref("");
  const deploymentSearch = ref("");
  const usageSearch = ref("");

  const filteredOrganizations = computed(() => {
    const term = orgSearch.value.trim().toLowerCase();
    if (!term) return organizations.value;
    return organizations.value.filter((org) => {
      const searchable = [
        org.name,
        org.slug,
        org.id,
        org.domain,
        org.plan,
        org.status,
      ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      return searchable.includes(term);
    });
  });

  const filteredInvites = computed(() => {
    const term = inviteSearch.value.trim().toLowerCase();
    if (!term) return invites.value;
    return invites.value.filter((invite) => {
      const searchable = [
        invite.email,
        invite.id,
        invite.organizationId,
        invite.role,
      ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      return searchable.includes(term);
    });
  });

  const filteredDeployments = computed(() => {
    const term = deploymentSearch.value.trim().toLowerCase();
    if (!term) return deployments.value;
    return deployments.value.filter((deployment) => {
      const searchable = [
        deployment.name,
        deployment.id,
        deployment.domain,
        deployment.organizationId,
        formatEnvironment(deployment.environment),
        formatStatus(deployment.status),
      ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      return searchable.includes(term);
    });
  });

  const filteredUsages = computed(() => {
    const term = usageSearch.value.trim().toLowerCase();
    if (!term) return usages.value;
    return usages.value.filter((usage) => {
      const searchable = [
        usage.organizationName,
        usage.organizationId,
        usage.month,
      ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      return searchable.includes(term);
    });
  });

  const { formatBytes: formatBytesUtil } = useUtils();
  const numberFormatter = new Intl.NumberFormat();
  const dateFormatter = new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: undefined,
  });

  const orgColumns = [
    { key: "name", label: "Name", defaultWidth: 250, minWidth: 200 },
    { key: "plan", label: "Plan", defaultWidth: 120, minWidth: 100 },
    { key: "members", label: "Members", defaultWidth: 100, minWidth: 80 },
    { key: "invites", label: "Invites", defaultWidth: 100, minWidth: 80 },
    { key: "deployments", label: "Deployments", defaultWidth: 120, minWidth: 100 },
    { key: "created", label: "Created", defaultWidth: 150, minWidth: 120 },
  ];

  const inviteColumns = [
    { key: "email", label: "Email", defaultWidth: 200, minWidth: 150 },
    { key: "organization", label: "Organization", defaultWidth: 200, minWidth: 150 },
    { key: "role", label: "Role", defaultWidth: 120, minWidth: 100 },
    { key: "invited", label: "Invited", defaultWidth: 150, minWidth: 120 },
  ];

  const deploymentColumns = [
    { key: "deployment", label: "Deployment", defaultWidth: 250, minWidth: 200 },
    { key: "organization", label: "Organization", defaultWidth: 200, minWidth: 150 },
    { key: "environment", label: "Environment", defaultWidth: 120, minWidth: 100 },
    { key: "status", label: "Status", defaultWidth: 120, minWidth: 100 },
    { key: "lastDeployed", label: "Last Deployed", defaultWidth: 150, minWidth: 120 },
  ];

  const usageColumns = [
    { key: "organization", label: "Organization", defaultWidth: 200, minWidth: 150 },
    { key: "cpu", label: "CPU (core-s)", defaultWidth: 120, minWidth: 100 },
    { key: "memory", label: "Memory (byte-s)", defaultWidth: 150, minWidth: 120 },
    { key: "bandwidthRx", label: "Bandwidth RX", defaultWidth: 140, minWidth: 110 },
    { key: "bandwidthTx", label: "Bandwidth TX", defaultWidth: 140, minWidth: 110 },
    { key: "storage", label: "Storage", defaultWidth: 120, minWidth: 100 },
    { key: "peakDeployments", label: "Peak Deployments", defaultWidth: 140, minWidth: 120 },
  ];

  const tableOrgs = computed(() => filteredOrganizations.value);
  const tableInvites = computed(() => filteredInvites.value);
  const tableDeployments = computed(() => filteredDeployments.value);
  const tableUsages = computed(() => filteredUsages.value);

  const metrics = computed(() => {
    const counts = overview.value?.counts;
    return [
      {
        label: "Organizations",
        value: formatNumber(counts?.totalOrganizations),
        help: "Total tenants across the platform",
      },
      {
        label: "Active Members",
        value: formatNumber(counts?.activeMembers),
        help: "Users with accepted organization membership",
      },
      {
        label: "Pending Invites",
        value: formatNumber(counts?.pendingInvites),
        help: "Waiting for onboarding",
      },
      {
        label: "Deployments",
        value: formatNumber(counts?.totalDeployments),
        help: "Tracked application deployments",
      },
    ];
  });

  function formatNumber(value?: number | bigint | null) {
    if (value === undefined || value === null) return "0";
    return numberFormatter.format(Number(value));
  }

  function formatDate(
    ts?: { seconds?: bigint | number; nanos?: number } | null
  ) {
    if (!ts || ts.seconds === undefined) return "—";
    const seconds =
      typeof ts.seconds === "bigint" ? Number(ts.seconds) : ts.seconds;
    const millis = seconds * 1000 + Math.floor((ts.nanos ?? 0) / 1_000_000);
    const date = new Date(millis);
    return Number.isNaN(date.getTime()) ? "—" : dateFormatter.format(date);
  }

  function formatBytes(value?: number | bigint | null) {
    return formatBytesUtil(value);
  }

  function prettyPlan(plan?: string) {
    if (!plan) return "—";
    return plan.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
  }

  function formatEnvironment(env?: number) {
    switch (env) {
      case 1:
        return "PRODUCTION";
      case 2:
        return "STAGING";
      case 3:
        return "DEVELOPMENT";
      default:
        return "UNSPECIFIED";
    }
  }

  function formatStatus(status?: number) {
    switch (status) {
      case 1:
        return "CREATED";
      case 2:
        return "BUILDING";
      case 3:
        return "RUNNING";
      case 4:
        return "STOPPED";
      case 5:
        return "FAILED";
      case 6:
        return "DEPLOYING";
      default:
        return "UNKNOWN";
    }
  }
</script>
