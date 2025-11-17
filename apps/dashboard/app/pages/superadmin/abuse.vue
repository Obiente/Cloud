<template>
  <OuiStack gap="xl">
    <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
      <OuiStack gap="xs">
        <OuiText tag="h1" size="3xl" weight="extrabold">Abuse Detection</OuiText>
        <OuiText color="muted">
          Monitor suspicious organizations and activities for potential abuse.
        </OuiText>
      </OuiStack>
      <OuiButton variant="ghost" size="sm" @click="refresh" :disabled="isLoading">
        <span class="flex items-center gap-2">
          <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': isLoading }" />
          Refresh
        </span>
      </OuiButton>
    </OuiFlex>

    <!-- Metrics Overview -->
    <OuiGrid class="gap-4" cols="1" colsMd="2" colsXl="5" cols2xl="6">
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
          <OuiText size="3xl" weight="semibold" :color="metric.color">{{
            metric.value
          }}</OuiText>
        </OuiStack>
      </OuiCard>
    </OuiGrid>

    <!-- Suspicious Organizations -->
    <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
      <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
        <OuiText tag="h2" size="xl" weight="bold">Suspicious Organizations</OuiText>
      </OuiCardHeader>
      <OuiCardBody class="p-0">
        <div v-if="isLoading" class="text-center py-8">
          <OuiText color="muted">Loading...</OuiText>
        </div>
        <OuiTable
          v-else
          :columns="orgColumns"
          :rows="suspiciousOrgs"
          :empty-text="'No suspicious organizations found.'"
        >
          <template #cell-risk="{ value }">
            <OuiBadge :variant="getRiskVariant(value)">
              {{ value }}% Risk
            </OuiBadge>
          </template>
          <template #cell-organization="{ value, row }">
            <div>
              <div class="font-medium text-text-primary">{{ row.organizationName }}</div>
              <div class="text-xs font-mono text-text-tertiary mt-0.5">{{ value }}</div>
            </div>
          </template>
          <template #cell-actions="{ row }">
            <OuiButton
              variant="ghost"
              size="xs"
              @click="viewOrganization(row.organizationId)"
            >
              View
            </OuiButton>
          </template>
        </OuiTable>
      </OuiCardBody>
    </OuiCard>

    <!-- Suspicious Activities -->
    <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
      <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
        <OuiText tag="h2" size="xl" weight="bold">Suspicious Activities</OuiText>
      </OuiCardHeader>
      <OuiCardBody class="p-0">
        <div v-if="isLoading" class="text-center py-8">
          <OuiText color="muted">Loading...</OuiText>
        </div>
        <OuiTable
          v-else
          :columns="activityColumns"
          :rows="suspiciousActivities"
          :empty-text="'No suspicious activities found.'"
        >
          <template #cell-severity="{ value }">
            <OuiBadge :variant="getSeverityVariant(value)">
              {{ value }}% Severity
            </OuiBadge>
          </template>
          <template #cell-organization="{ value, row }">
            <OuiButton
              variant="ghost"
              size="xs"
              @click="viewOrganization(row.organizationId)"
            >
              {{ value }}
            </OuiButton>
          </template>
        </OuiTable>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { ArrowPathIcon } from "@heroicons/vue/24/outline";
import { computed, ref } from "vue";
import { SuperadminService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useRouter } from "vue-router";

definePageMeta({
  middleware: ["auth", "superadmin"],
});

const router = useRouter();
const client = useConnectClient(SuperadminService);

const abuseData = ref<any>(null);

async function fetchAbuseDetection() {
  try {
    const response = await client.getAbuseDetection({});
    abuseData.value = response;
  } catch (err) {
    console.error("Failed to fetch abuse detection:", err);
  }
}

// Use client-side fetching for non-blocking navigation
const { pending: isLoading } = useClientFetch("superadmin-abuse", fetchAbuseDetection);

const metrics = computed(() => {
  const m = abuseData.value?.metrics;
  const activities = suspiciousActivities.value;
  if (!m) return [];
  
  // Count activities by type
  const activityCounts = activities.reduce((acc: Record<string, number>, act: any) => {
    const type = act.activityType || "unknown";
    acc[type] = (acc[type] || 0) + 1;
    return acc;
  }, {});

  return [
    {
      label: "Suspicious Orgs",
      value: m.totalSuspiciousOrgs?.toString() || "0",
      color: "primary" as const,
    },
    {
      label: "High Risk",
      value: m.highRiskOrgs?.toString() || "0",
      color: "danger" as const,
    },
    {
      label: "Total Activities",
      value: activities.length.toString(),
      color: "primary" as const,
    },
    {
      label: "Rapid Creations (24h)",
      value: m.rapidCreations24h?.toString() || "0",
      color: "warning" as const,
    },
    {
      label: "Failed Payments (24h)",
      value: m.failedPaymentAttempts24h?.toString() || "0",
      color: "danger" as const,
    },
    {
      label: "Usage Spikes (24h)",
      value: m.unusualUsageSpikes24h?.toString() || "0",
      color: "warning" as const,
    },
    {
      label: "SSH Brute Force",
      value: (activityCounts["ssh_brute_force"] || 0).toString(),
      color: "danger" as const,
    },
    {
      label: "API Abuse",
      value: (activityCounts["api_abuse"] || 0).toString(),
      color: "warning" as const,
    },
    {
      label: "Failed Auth",
      value: (activityCounts["failed_authentication"] || 0).toString(),
      color: "danger" as const,
    },
    {
      label: "Multiple Accounts",
      value: (activityCounts["multiple_accounts"] || 0).toString(),
      color: "warning" as const,
    },
  ];
});

const suspiciousOrgs = computed(() => {
  return (
    abuseData.value?.suspiciousOrganizations?.map((org: any) => ({
      organizationId: org.organizationId,
      organizationName: org.organizationName || "Unknown",
      reason: org.reason || "—",
      riskScore: org.riskScore || 0,
      createdCount24h: org.createdCount24h || 0,
      failedDeployments24h: org.failedDeployments24h || 0,
      totalCreditsSpent: org.totalCreditsSpent || 0,
      createdAt: formatDate(org.createdAt),
      lastActivity: formatDate(org.lastActivity),
    })) || []
  );
});

const suspiciousActivities = computed(() => {
  return (
    abuseData.value?.suspiciousActivities?.map((act: any) => ({
      id: act.id,
      organizationId: act.organizationId,
      activityType: act.activityType || "—",
      activityTypeLabel: formatActivityType(act.activityType),
      description: act.description || "—",
      severity: act.severity || 0,
      occurredAt: formatDate(act.occurredAt),
    })) || []
  );
});

const orgColumns = computed(() => [
  { key: "organization", label: "Organization", defaultWidth: 200, minWidth: 150 },
  { key: "reason", label: "Reason", defaultWidth: 300, minWidth: 200 },
  { key: "risk", label: "Risk Score", defaultWidth: 120, minWidth: 100 },
  { key: "createdCount24h", label: "Created (24h)", defaultWidth: 120, minWidth: 100 },
  { key: "failedDeployments24h", label: "Failed (24h)", defaultWidth: 120, minWidth: 100 },
  { key: "totalCreditsSpent", label: "Credits Spent", defaultWidth: 120, minWidth: 100 },
  { key: "actions", label: "Actions", defaultWidth: 100, minWidth: 80, resizable: false },
]);

const activityColumns = computed(() => [
  { key: "organization", label: "Organization", defaultWidth: 150, minWidth: 120 },
  { key: "activityTypeLabel", label: "Type", defaultWidth: 180, minWidth: 150 },
  { key: "description", label: "Description", defaultWidth: 400, minWidth: 250 },
  { key: "severity", label: "Severity", defaultWidth: 120, minWidth: 100 },
  { key: "occurredAt", label: "Occurred", defaultWidth: 150, minWidth: 120 },
]);

function getRiskVariant(risk: number): "danger" | "warning" {
  if (risk >= 70) return "danger";
  return "warning";
}

function getSeverityVariant(severity: number): "danger" | "warning" {
  if (severity >= 70) return "danger";
  return "warning";
}

function viewOrganization(orgId: string) {
  router.push(`/superadmin/organizations?org=${orgId}`);
}

function refresh() {
  fetchAbuseDetection();
}

function formatDate(timestamp?: { seconds?: number | bigint; nanos?: number } | null) {
  if (!timestamp || timestamp.seconds === undefined) return "—";
  const seconds = typeof timestamp.seconds === "bigint" ? Number(timestamp.seconds) : timestamp.seconds;
  const millis = seconds * 1000 + Math.floor((timestamp.nanos ?? 0) / 1_000_000);
  const date = new Date(millis);
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date);
}

function formatActivityType(type?: string): string {
  if (!type) return "—";
  const typeMap: Record<string, string> = {
    rapid_creation: "Rapid Resource Creation",
    failed_payments: "Failed Payment Attempts",
    ssh_brute_force: "SSH Brute Force",
    api_abuse: "API Abuse",
    failed_authentication: "Failed Authentication",
    multiple_accounts: "Multiple Account Creation",
    dns_delegation_abuse: "DNS Delegation Abuse",
    usage_spike: "Usage Spike",
  };
  return typeMap[type] || type.replace(/_/g, " ").replace(/\b\w/g, (l) => l.toUpperCase());
}
</script>

