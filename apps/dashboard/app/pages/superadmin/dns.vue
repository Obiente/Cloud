<template>
  <OuiStack gap="xl">
    <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
      <OuiStack gap="xs">
        <OuiText tag="h1" size="3xl" weight="extrabold">DNS Management</OuiText>
        <OuiText color="muted">
          Query DNS records and view DNS configuration for deployments.
        </OuiText>
      </OuiStack>
      <OuiButton variant="ghost" size="sm" @click="refresh" :disabled="isLoading">
        <span class="flex items-center gap-2">
          <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': isLoading }" />
          Refresh
        </span>
      </OuiButton>
    </OuiFlex>

    <!-- DNS Query Tool -->
    <OuiCard class="border border-border-muted rounded-xl">
      <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
        <OuiText tag="h2" size="xl" weight="bold">DNS Query</OuiText>
      </OuiCardHeader>
      <OuiCardBody class="p-6">
        <OuiStack gap="md">
          <OuiFlex gap="md" wrap="wrap" align="end">
            <div class="flex-1 min-w-[300px]">
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium">Domain</OuiText>
                <OuiInput
                  v-model="queryDomain"
                  placeholder="deploy-123.my.obiente.cloud"
                  @keyup.enter="queryDNS"
                />
              </OuiStack>
            </div>
            <div class="min-w-[120px]">
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium">Record Type</OuiText>
                <OuiSelect
                  v-model="queryRecordType"
                  :items="recordTypeOptions"
                />
              </OuiStack>
            </div>
            <OuiButton @click="queryDNS" :disabled="queryLoading || !queryDomain">
              Query
            </OuiButton>
          </OuiFlex>

          <!-- Query Results -->
          <div v-if="queryResult" class="mt-4">
            <OuiCard 
              :class="queryResult.error ? 'bg-danger/5 border-danger/20' : 'bg-success/5 border-success/20'"
              class="border rounded-lg p-4"
            >
              <OuiStack gap="sm">
                <OuiFlex align="center" justify="between">
                  <OuiText size="sm" weight="medium">
                    {{ queryResult.domain }} ({{ queryResult.recordType }})
                  </OuiText>
                  <OuiText size="xs" color="muted">
                    TTL: {{ queryResult.ttl }}s
                  </OuiText>
                </OuiFlex>
                <div v-if="queryResult.error" class="text-danger">
                  <OuiText size="sm">{{ queryResult.error }}</OuiText>
                </div>
                <div v-else-if="queryResult.records && queryResult.records.length > 0">
                  <OuiStack gap="xs">
                    <OuiText size="xs" color="muted" weight="medium">Records:</OuiText>
                    <div v-for="(record, idx) in queryResult.records" :key="idx" class="font-mono text-sm">
                      {{ record }}
                    </div>
                  </OuiStack>
                </div>
                <div v-else>
                  <OuiText size="sm" color="muted">No records found</OuiText>
                </div>
              </OuiStack>
            </OuiCard>
          </div>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- DNS Configuration -->
    <OuiCard class="border border-border-muted rounded-xl">
      <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
        <OuiText tag="h2" size="xl" weight="bold">DNS Configuration</OuiText>
      </OuiCardHeader>
      <OuiCardBody class="p-6">
        <div v-if="dnsConfigLoading" class="text-center py-8">
          <OuiText color="muted">Loading configuration...</OuiText>
        </div>
        <div v-else-if="dnsConfig">
          <OuiStack gap="lg">
            <!-- Traefik IPs by Region -->
            <OuiStack gap="md">
              <OuiText size="lg" weight="semibold">Traefik IPs by Region</OuiText>
              <div v-if="traefikIPsByRegion.length === 0" class="text-muted">
                <OuiText size="sm">No regions configured</OuiText>
              </div>
              <div v-else class="space-y-3">
                <div
                  v-for="region in traefikIPsByRegion"
                  :key="region.region"
                  class="border border-border-muted rounded-lg p-4 bg-surface-subtle"
                >
                  <OuiStack gap="xs">
                    <OuiText size="sm" weight="medium">{{ region.region || "default" }}</OuiText>
                    <div class="flex flex-wrap gap-2">
                      <span
                        v-for="ip in region.ips"
                        :key="ip"
                        class="font-mono text-sm px-2 py-1 bg-surface-raised rounded border border-border-muted"
                      >
                        {{ ip }}
                      </span>
                    </div>
                  </OuiStack>
                </div>
              </div>
            </OuiStack>

            <!-- DNS Server Info -->
            <OuiStack gap="md">
              <OuiText size="lg" weight="semibold">DNS Server Info</OuiText>
              <OuiGrid cols="1" colsMd="2" gap="md">
                <div>
                  <OuiText size="xs" color="muted" transform="uppercase" class="tracking-wide">DNS Port</OuiText>
                  <OuiText size="sm" weight="medium" class="font-mono">{{ dnsConfig.dnsPort || "53" }}</OuiText>
                </div>
                <div>
                  <OuiText size="xs" color="muted" transform="uppercase" class="tracking-wide">Cache TTL</OuiText>
                  <OuiText size="sm" weight="medium">{{ dnsConfig.cacheTtlSeconds }} seconds</OuiText>
                </div>
              </OuiGrid>
            </OuiStack>

            <!-- DNS Server IPs -->
            <OuiStack gap="md" v-if="dnsConfig.dnsServerIps && dnsConfig.dnsServerIps.length > 0">
              <OuiText size="lg" weight="semibold">DNS Server IPs</OuiText>
              <div class="flex flex-wrap gap-2">
                <span
                  v-for="ip in dnsConfig.dnsServerIps"
                  :key="ip"
                  class="font-mono text-sm px-2 py-1 bg-surface-raised rounded border border-border-muted"
                >
                  {{ ip }}
                </span>
              </div>
            </OuiStack>
          </OuiStack>
        </div>
        <div v-else class="text-center py-8">
          <OuiText color="muted">Failed to load DNS configuration</OuiText>
        </div>
      </OuiCardBody>
    </OuiCard>

    <!-- DNS Records List -->
    <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
      <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
        <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
          <OuiStack gap="xs">
            <OuiText tag="h2" size="xl" weight="bold">DNS Records</OuiText>
            <OuiText color="muted" size="sm">
              {{ filteredRecords.length }} of {{ dnsRecords.length }} records
            </OuiText>
          </OuiStack>
          <OuiFlex gap="sm" wrap="wrap">
            <div class="w-72 max-w-full">
              <OuiInput
                v-model="recordsSearch"
                type="search"
                placeholder="Search by domain, deployment ID, organization ID..."
                clearable
                size="sm"
              />
            </div>
            <div class="min-w-[160px]">
              <OuiSelect
                v-model="recordsDeploymentFilter"
                :items="deploymentFilterOptions"
                placeholder="Deployment"
                clearable
                size="sm"
              />
            </div>
            <div class="min-w-[160px]">
              <OuiSelect
                v-model="recordsOrgFilter"
                :items="orgFilterOptions"
                placeholder="Organization"
                clearable
                size="sm"
              />
            </div>
          </OuiFlex>
        </OuiFlex>
      </OuiCardHeader>
      <OuiCardBody class="p-0">
        <OuiTable
          :columns="tableColumns"
          :rows="tableRows"
          :empty-text="recordsLoading ? 'Loading DNS records…' : 'No DNS records match your filters.'"
        >
          <template #cell-domain="{ value }">
            <div class="font-mono text-sm">{{ value }}</div>
          </template>
          <template #cell-deployment="{ value, row }">
            <div>
              <div class="font-medium text-text-primary">{{ row.deploymentName || value }}</div>
              <div class="text-xs font-mono text-text-tertiary mt-0.5">{{ value }}</div>
            </div>
          </template>
          <template #cell-ips="{ value }">
            <div v-if="value && value.length > 0" class="flex flex-wrap gap-1">
              <span
                v-for="ip in value"
                :key="ip"
                class="font-mono text-xs px-2 py-0.5 bg-surface-subtle rounded border border-border-muted"
              >
                {{ ip }}
              </span>
            </div>
            <span v-else class="text-text-tertiary text-sm">—</span>
          </template>
          <template #cell-region="{ value }">
            <span v-if="value" class="text-text-secondary uppercase text-xs">{{ value }}</span>
            <span v-else class="text-text-tertiary text-sm">—</span>
          </template>
          <template #cell-status="{ value }">
            <OuiBadge :variant="getStatusBadgeVariant(value)">
              <span
                class="inline-flex h-1.5 w-1.5 rounded-full mr-1.5"
                :class="getStatusDotClass(value)"
              />
              <OuiText
                as="span"
                size="xs"
                weight="semibold"
                transform="uppercase"
                class="text-[11px]"
              >
                {{ getStatusLabel(value) }}
              </OuiText>
            </OuiBadge>
          </template>
        </OuiTable>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
definePageMeta({
  middleware: ["auth", "superadmin"],
});

import { ArrowPathIcon } from "@heroicons/vue/24/outline";
import { computed, ref, onMounted } from "vue";
import { SuperadminService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";

const client = useConnectClient(SuperadminService);

const isLoading = ref(false);
const recordsLoading = ref(false);
const dnsConfigLoading = ref(false);
const queryLoading = ref(false);

const queryDomain = ref("");
const queryRecordType = ref("A");
const queryResult = ref<any>(null);

const recordsSearch = ref("");
const recordsDeploymentFilter = ref<string | null>(null);
const recordsOrgFilter = ref<string | null>(null);

const dnsRecords = ref<any[]>([]);
const dnsConfig = ref<any>(null);

const recordTypeOptions = [
  { label: "A", value: "A" },
];

const traefikIPsByRegion = computed(() => {
  if (!dnsConfig.value?.traefikIpsByRegion) return [];
  return Object.entries(dnsConfig.value.traefikIpsByRegion).map(([region, ips]: [string, any]) => ({
    region,
    ips: ips.ips || [],
  }));
});

const deploymentFilterOptions = computed(() => {
  const deployments = new Set<string>();
  dnsRecords.value.forEach((record) => {
    if (record.deploymentId) deployments.add(record.deploymentId);
  });
  return Array.from(deployments).sort().map((dep) => ({ label: dep, value: dep }));
});

const orgFilterOptions = computed(() => {
  const orgs = new Set<string>();
  dnsRecords.value.forEach((record) => {
    if (record.organizationId) orgs.add(record.organizationId);
  });
  return Array.from(orgs).sort().map((org) => ({ label: org, value: org }));
});

const filteredRecords = computed(() => {
  const term = recordsSearch.value.trim().toLowerCase();
  const deploymentFilter = recordsDeploymentFilter.value;
  const orgFilter = recordsOrgFilter.value;

  return dnsRecords.value.filter((record) => {
    if (deploymentFilter && record.deploymentId !== deploymentFilter) return false;
    if (orgFilter && record.organizationId !== orgFilter) return false;

    if (!term) return true;

    const searchable = [
      record.domain,
      record.deploymentId,
      record.deploymentName,
      record.organizationId,
      record.region,
      record.status,
      ...(record.ipAddresses || []),
    ]
      .filter(Boolean)
      .join(" ")
      .toLowerCase();

    return searchable.includes(term);
  });
});

const tableColumns = computed(() => [
  { key: "domain", label: "Domain", defaultWidth: 250, minWidth: 200 },
  { key: "deployment", label: "Deployment", defaultWidth: 200, minWidth: 150 },
  { key: "ips", label: "IP Addresses", defaultWidth: 250, minWidth: 200 },
  { key: "region", label: "Region", defaultWidth: 120, minWidth: 100 },
  { key: "status", label: "Status", defaultWidth: 120, minWidth: 100 },
  { key: "organizationId", label: "Organization", defaultWidth: 180, minWidth: 150 },
]);

const tableRows = computed(() => {
  return filteredRecords.value.map((record) => {
    // Ensure status is converted to string if it's a number or string number
    let status: string;
    if (typeof record.status === 'number') {
      status = convertStatusNumberToString(record.status);
    } else if (typeof record.status === 'string') {
      // Handle string numbers like "3" -> "RUNNING"
      const numStatus = parseInt(record.status, 10);
      if (!isNaN(numStatus)) {
        status = convertStatusNumberToString(numStatus);
      } else {
        // Already a string status name like "RUNNING"
        status = record.status.toUpperCase();
      }
    } else {
      status = "UNKNOWN";
    }
    return {
      ...record,
      status,
      organizationId: record.organizationId || "—",
    };
  });
});

function convertStatusNumberToString(status: number): string {
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

async function queryDNS() {
  if (!queryDomain.value) return;

  queryLoading.value = true;
  queryResult.value = null;

  try {
    const response = await client.queryDNS({
      domain: queryDomain.value,
      recordType: queryRecordType.value,
    });
    queryResult.value = response;
  } catch (err: any) {
    queryResult.value = {
      domain: queryDomain.value,
      recordType: queryRecordType.value,
      error: err.message || "Failed to query DNS",
      records: [],
      ttl: 0,
    };
  } finally {
    queryLoading.value = false;
  }
}

async function loadDNSRecords() {
  recordsLoading.value = true;
  try {
    const response = await client.listDNSRecords({});
    dnsRecords.value = response.records || [];
  } catch (err) {
    console.error("Failed to load DNS records:", err);
    dnsRecords.value = [];
  } finally {
    recordsLoading.value = false;
  }
}

async function loadDNSConfig() {
  dnsConfigLoading.value = true;
  try {
    const response = await client.getDNSConfig({});
    dnsConfig.value = response.config;
  } catch (err) {
    console.error("Failed to load DNS config:", err);
    dnsConfig.value = null;
  } finally {
    dnsConfigLoading.value = false;
  }
}

async function refresh() {
  await Promise.all([loadDNSRecords(), loadDNSConfig()]);
}

function getStatusBadgeVariant(status: string | number): "primary" | "secondary" | "success" | "warning" | "danger" | "outline" {
  // Handle both string and number status values
  let statusStr: string;
  if (typeof status === 'number') {
    statusStr = convertStatusNumberToString(status);
  } else {
    // Handle string numbers like "3" or status names like "RUNNING"
    const numStatus = parseInt(String(status), 10);
    if (!isNaN(numStatus)) {
      statusStr = convertStatusNumberToString(numStatus);
    } else {
      statusStr = String(status || "").toUpperCase();
    }
  }
  switch (statusStr) {
    case "RUNNING":
      return "success";
    case "STOPPED":
      return "danger";
    case "BUILDING":
    case "DEPLOYING":
      return "warning";
    case "FAILED":
      return "danger";
    case "CREATED":
      return "secondary";
    default:
      return "secondary";
  }
}

function getStatusDotClass(status: string | number): string {
  let statusStr: string;
  if (typeof status === 'number') {
    statusStr = convertStatusNumberToString(status);
  } else {
    const numStatus = parseInt(String(status), 10);
    if (!isNaN(numStatus)) {
      statusStr = convertStatusNumberToString(numStatus);
    } else {
      statusStr = String(status || "").toUpperCase();
    }
  }
  switch (statusStr) {
    case "RUNNING":
      return "bg-success animate-pulse";
    case "STOPPED":
      return "bg-danger";
    case "BUILDING":
    case "DEPLOYING":
      return "bg-warning animate-pulse";
    case "FAILED":
      return "bg-danger";
    case "CREATED":
      return "bg-secondary";
    default:
      return "bg-secondary";
  }
}

function getStatusLabel(status: string | number): string {
  let statusStr: string;
  if (typeof status === 'number') {
    statusStr = convertStatusNumberToString(status);
  } else {
    const numStatus = parseInt(String(status), 10);
    if (!isNaN(numStatus)) {
      statusStr = convertStatusNumberToString(numStatus);
    } else {
      statusStr = String(status || "").toUpperCase();
    }
  }
  switch (statusStr) {
    case "RUNNING":
      return "Running";
    case "STOPPED":
      return "Stopped";
    case "BUILDING":
      return "Building";
    case "DEPLOYING":
      return "Deploying";
    case "FAILED":
      return "Failed";
    case "CREATED":
      return "Created";
    default:
      return statusStr || "Unknown";
  }
}

onMounted(() => {
  refresh();
});
</script>
