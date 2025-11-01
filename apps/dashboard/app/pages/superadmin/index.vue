<template>
  <OuiStack gap="2xl">
    <OuiStack gap="xs">
      <OuiText tag="h1" size="3xl" weight="extrabold">Superadmin Overview</OuiText>
      <OuiText color="muted">System-wide visibility across organizations, deployments, and usage.</OuiText>
    </OuiStack>

    <OuiContainer>
      <OuiGrid class="gap-4" cols="1" colsMd="2" colsXl="4">
        <OuiCard v-for="metric in metrics" :key="metric.label" class="p-6 bg-surface-raised border border-border-muted rounded-xl">
          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium" color="secondary" transform="uppercase" class="tracking-wide">{{ metric.label }}</OuiText>
            <OuiText size="3xl" weight="semibold" color="primary">{{ metric.value }}</OuiText>
            <OuiText size="xs" color="muted">{{ metric.help }}</OuiText>
          </OuiStack>
        </OuiCard>
      </OuiGrid>
    </OuiContainer>

    <OuiContainer>
      <OuiStack gap="lg">
        <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
          <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
            <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
              <OuiStack gap="xs">
                <OuiText tag="h2" size="xl" weight="bold">Organizations</OuiText>
                <OuiText color="muted" size="sm">{{ filteredOrganizations.length }} of {{ organizations.length }} organizations</OuiText>
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
          <div class="overflow-x-auto">
            <table class="min-w-full text-left text-sm">
              <thead class="bg-surface-subtle text-text-muted uppercase text-xs tracking-wide">
                <tr>
                  <th class="px-6 py-3">Name</th>
                  <th class="px-6 py-3">Plan</th>
                  <th class="px-6 py-3">Members</th>
                  <th class="px-6 py-3">Invites</th>
                  <th class="px-6 py-3">Deployments</th>
                  <th class="px-6 py-3">Created</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="org in filteredOrganizations" :key="org.id" class="border-t border-border-muted/60">
                  <td class="px-6 py-3">
                    <div class="font-medium text-text-primary">{{ org.name || org.slug || "—" }}</div>
                    <div class="text-xs text-text-muted">
                      <span v-if="org.slug">{{ org.slug }}</span>
                      <span v-else class="text-text-tertiary">No slug</span>
                    </div>
                    <div class="text-xs font-mono text-text-tertiary mt-0.5">{{ org.id }}</div>
                    <div v-if="org.domain" class="text-xs text-text-muted mt-0.5">{{ org.domain }}</div>
                  </td>
                  <td class="px-6 py-3 text-text-secondary">{{ prettyPlan(org.plan) }}</td>
                  <td class="px-6 py-3">{{ formatNumber(org.memberCount) }}</td>
                  <td class="px-6 py-3">{{ formatNumber(org.inviteCount) }}</td>
                  <td class="px-6 py-3">{{ formatNumber(org.deploymentCount) }}</td>
                  <td class="px-6 py-3 text-text-secondary">{{ formatDate(org.createdAt) }}</td>
                </tr>
                <tr v-if="!filteredOrganizations.length">
                  <td colspan="6" class="px-6 py-6 text-center text-text-muted">
                    {{ organizations.length === 0 ? "No organizations available." : "No organizations match your search." }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </OuiCardBody>
      </OuiCard>

      <OuiGrid class="gap-6" cols="1" colsXl="2">
        <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
          <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
            <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
              <OuiStack gap="xs">
                <OuiText tag="h3" size="lg" weight="semibold">Pending Invites</OuiText>
                <OuiText color="muted" size="sm">{{ filteredInvites.length }} of {{ invites.length }} invites</OuiText>
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
            <div class="overflow-x-auto">
              <table class="min-w-full text-left text-sm">
                <thead class="bg-surface-subtle text-text-muted uppercase text-xs tracking-wide">
                  <tr>
                    <th class="px-6 py-3">Email</th>
                    <th class="px-6 py-3">Organization</th>
                    <th class="px-6 py-3">Role</th>
                    <th class="px-6 py-3">Invited</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="invite in filteredInvites" :key="invite.id" class="border-t border-border-muted/60">
                    <td class="px-6 py-3">
                      <div class="text-text-primary">{{ invite.email }}</div>
                      <div class="text-xs font-mono text-text-tertiary mt-0.5">{{ invite.id }}</div>
                    </td>
                    <td class="px-6 py-3">
                      <div class="text-text-secondary font-mono text-sm">{{ invite.organizationId }}</div>
                    </td>
                    <td class="px-6 py-3 text-text-secondary uppercase tracking-wide text-xs">{{ invite.role }}</td>
                    <td class="px-6 py-3 text-text-secondary">{{ formatDate(invite.invitedAt) }}</td>
                  </tr>
                  <tr v-if="!filteredInvites.length">
                    <td colspan="4" class="px-6 py-6 text-center text-text-muted">
                      {{ invites.length === 0 ? "No pending invitations." : "No invites match your search." }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </OuiCardBody>
        </OuiCard>

        <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
          <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
            <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
              <OuiStack gap="xs">
                <OuiText tag="h3" size="lg" weight="semibold">Recent Deployments</OuiText>
                <OuiText color="muted" size="sm">{{ filteredDeployments.length }} of {{ deployments.length }} deployments</OuiText>
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
            <div class="overflow-x-auto">
              <table class="min-w-full text-left text-sm">
                <thead class="bg-surface-subtle text-text-muted uppercase text-xs tracking-wide">
                  <tr>
                    <th class="px-6 py-3">Deployment</th>
                    <th class="px-6 py-3">Organization</th>
                    <th class="px-6 py-3">Environment</th>
                    <th class="px-6 py-3">Status</th>
                    <th class="px-6 py-3">Last Deployed</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="deployment in filteredDeployments" :key="deployment.id" class="border-t border-border-muted/60">
                    <td class="px-6 py-3">
                      <div class="font-medium text-text-primary">{{ deployment.name }}</div>
                      <div v-if="deployment.domain" class="text-xs text-text-muted">{{ deployment.domain }}</div>
                      <div class="text-xs font-mono text-text-tertiary mt-0.5">{{ deployment.id }}</div>
                    </td>
                    <td class="px-6 py-3">
                      <div class="text-text-secondary font-mono text-sm">{{ deployment.organizationId }}</div>
                    </td>
                    <td class="px-6 py-3 text-text-secondary uppercase text-xs">{{ formatEnvironment(deployment.environment) }}</td>
                    <td class="px-6 py-3 text-text-secondary uppercase text-xs">{{ formatStatus(deployment.status) }}</td>
                    <td class="px-6 py-3 text-text-secondary">{{ formatDate(deployment.lastDeployedAt || deployment.createdAt) }}</td>
                  </tr>
                  <tr v-if="!filteredDeployments.length">
                    <td colspan="5" class="px-6 py-6 text-center text-text-muted">
                      {{ deployments.length === 0 ? "No deployments found." : "No deployments match your search." }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </OuiCardBody>
        </OuiCard>
      </OuiGrid>
      </OuiStack>
    </OuiContainer>

    <OuiContainer>
      <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
        <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
          <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
            <OuiStack gap="xs">
              <OuiText tag="h2" size="xl" weight="bold">Current Month Usage</OuiText>
              <OuiText color="muted" size="sm">{{ filteredUsages.length }} of {{ usages.length }} organizations</OuiText>
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
          <div class="overflow-x-auto">
            <table class="min-w-full text-left text-sm">
              <thead class="bg-surface-subtle text-text-muted uppercase text-xs tracking-wide">
                <tr>
                  <th class="px-6 py-3">Organization</th>
                  <th class="px-6 py-3">CPU (core-s)</th>
                  <th class="px-6 py-3">Memory (byte-s)</th>
                  <th class="px-6 py-3">Bandwidth RX</th>
                  <th class="px-6 py-3">Bandwidth TX</th>
                  <th class="px-6 py-3">Storage</th>
                  <th class="px-6 py-3">Peak Deployments</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="usage in filteredUsages" :key="`${usage.organizationId}-${usage.month}`" class="border-t border-border-muted/60">
                  <td class="px-6 py-3">
                    <div class="font-medium text-text-primary">{{ usage.organizationName || "—" }}</div>
                    <div class="text-xs font-mono text-text-tertiary mt-0.5">{{ usage.organizationId }}</div>
                    <div class="text-xs text-text-muted mt-0.5">{{ usage.month }}</div>
                  </td>
                  <td class="px-6 py-3">{{ formatNumber(usage.cpuCoreSeconds) }}</td>
                  <td class="px-6 py-3">{{ formatNumber(usage.memoryByteSeconds) }}</td>
                  <td class="px-6 py-3">{{ formatBytes(usage.bandwidthRxBytes) }}</td>
                  <td class="px-6 py-3">{{ formatBytes(usage.bandwidthTxBytes) }}</td>
                  <td class="px-6 py-3">{{ formatBytes(usage.storageBytes) }}</td>
                  <td class="px-6 py-3">{{ formatNumber(usage.deploymentsActivePeak) }}</td>
                </tr>
                <tr v-if="!filteredUsages.length">
                  <td colspan="7" class="px-6 py-6 text-center text-text-muted">
                    {{ usages.length === 0 ? "No usage records for the current month." : "No usage records match your search." }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </OuiCardBody>
      </OuiCard>
    </OuiContainer>
  </OuiStack>
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
    ].filter(Boolean).join(" ").toLowerCase();
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
    ].filter(Boolean).join(" ").toLowerCase();
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
    ].filter(Boolean).join(" ").toLowerCase();
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
    ].filter(Boolean).join(" ").toLowerCase();
    return searchable.includes(term);
  });
});

const numberFormatter = new Intl.NumberFormat();
const dateFormatter = new Intl.DateTimeFormat(undefined, {
  dateStyle: "medium",
  timeStyle: undefined,
});

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

function formatDate(ts?: { seconds?: bigint | number; nanos?: number } | null) {
  if (!ts || ts.seconds === undefined) return "—";
  const seconds = typeof ts.seconds === "bigint" ? Number(ts.seconds) : ts.seconds;
  const millis = seconds * 1000 + Math.floor((ts.nanos ?? 0) / 1_000_000);
  const date = new Date(millis);
  return Number.isNaN(date.getTime()) ? "—" : dateFormatter.format(date);
}

function formatBytes(value?: number | bigint | null) {
  if (value === undefined || value === null) return "0 B";
  let bytes = typeof value === "bigint" ? Number(value) : value;
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB", "PB"];
  const idx = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const sized = bytes / Math.pow(1024, idx);
  return `${sized.toFixed(idx === 0 ? 0 : 1)} ${units[idx]}`;
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
