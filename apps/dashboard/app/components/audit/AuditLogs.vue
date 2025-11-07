<template>
  <OuiStack gap="lg">
    <!-- Header -->
    <OuiFlex justify="between" align="center" wrap="wrap" gap="md">
      <OuiText as="h3" size="lg" weight="bold">Audit Logs</OuiText>
      <OuiFlex gap="sm" align="center" wrap="wrap">
        <OuiText v-if="!isLoading" size="sm" color="secondary">
          {{ total }} log{{ total !== 1 ? 's' : '' }}
        </OuiText>
        <OuiButton
          variant="ghost"
          size="sm"
          @click="refresh"
          :loading="isLoading"
        >
          <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': isLoading }" />
        </OuiButton>
      </OuiFlex>
    </OuiFlex>

    <!-- Filters -->
    <OuiCard variant="outline">
      <OuiCardBody>
        <OuiGrid cols="1" cols-md="2" cols-lg="4" gap="md">
          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Service</OuiText>
            <OuiSelect
              v-model="filters.service"
              :items="serviceOptions"
              placeholder="All services"
              clearable
            />
          </OuiStack>
          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Action</OuiText>
            <OuiInput
              v-model="filters.action"
              placeholder="Filter by action"
              clearable
            />
          </OuiStack>
          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">User</OuiText>
            <OuiInput
              v-model="filters.userId"
              placeholder="Filter by user ID"
              clearable
            />
          </OuiStack>
          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Status</OuiText>
            <OuiSelect
              v-model="filters.status"
              :items="statusOptions"
              placeholder="All statuses"
              clearable
            />
          </OuiStack>
        </OuiGrid>
      </OuiCardBody>
    </OuiCard>

    <!-- Loading State -->
    <div v-if="isLoading && auditLogs.length === 0" class="flex justify-center items-center py-12">
      <OuiText size="sm" color="secondary">Loading audit logs...</OuiText>
    </div>

    <!-- Empty State -->
    <div v-else-if="auditLogs.length === 0" class="flex flex-col items-center justify-center py-12">
      <DocumentTextIcon class="h-12 w-12 text-secondary mb-4" />
      <OuiText size="md" weight="medium" class="mb-2">No audit logs found</OuiText>
      <OuiText size="sm" color="secondary" class="text-center max-w-md">
        Audit logs will appear here once actions are performed.
      </OuiText>
    </div>

    <!-- Audit Logs Table -->
    <OuiCard v-else variant="default">
      <OuiCardBody class="p-0">
        <OuiTable
          :columns="columns"
          :rows="auditLogs"
          :row-class-fn="getRowClass"
        >
          <template #cell-action="{ row }">
            <OuiFlex align="center" gap="sm">
              <OuiBadge variant="secondary" size="sm">
                {{ row.action }}
              </OuiBadge>
            </OuiFlex>
          </template>

          <template #cell-service="{ row }">
            <OuiText size="sm" class="font-mono">{{ row.service }}</OuiText>
          </template>

          <template #cell-user="{ row }">
            <OuiStack gap="xs">
              <OuiText size="sm" weight="medium">{{ row.userName || row.userEmail || row.userId }}</OuiText>
              <OuiText size="xs" color="secondary" class="font-mono">{{ row.userId }}</OuiText>
            </OuiStack>
          </template>

          <template #cell-resource="{ row }">
            <OuiStack gap="xs" v-if="row.resourceType">
              <OuiText size="sm" weight="medium">{{ row.resourceType }}</OuiText>
              <OuiText v-if="row.resourceId" size="xs" color="secondary" class="font-mono">
                {{ row.resourceId }}
              </OuiText>
            </OuiStack>
            <OuiText v-else size="sm" color="muted">—</OuiText>
          </template>

          <template #cell-status="{ row }">
            <OuiBadge
              :variant="getStatusVariant(row.responseStatus)"
              size="sm"
            >
              {{ row.responseStatus }}
            </OuiBadge>
          </template>

          <template #cell-duration="{ row }">
            <OuiText size="sm">{{ formatDuration(row.durationMs) }}</OuiText>
          </template>

          <template #cell-time="{ row }">
            <OuiText size="sm">
              <OuiRelativeTime
                :value="row.createdAt ? date(row.createdAt) : undefined"
                :style="'short'"
              />
            </OuiText>
          </template>

          <template #cell-details="{ row }">
            <OuiButton
              variant="ghost"
              size="xs"
              @click="() => showDetails(row)"
            >
              <EyeIcon class="h-4 w-4" />
            </OuiButton>
          </template>
        </OuiTable>
      </OuiCardBody>
    </OuiCard>

    <!-- Pagination -->
    <OuiFlex v-if="hasNextPage" justify="center" gap="md">
      <OuiButton
        variant="outline"
        @click="loadMore"
        :loading="isLoadingMore"
      >
        Load More
      </OuiButton>
    </OuiFlex>

    <!-- Details Dialog -->
    <OuiDialog v-model="detailsDialogOpen" title="Audit Log Details">
      <OuiStack gap="md" v-if="selectedLog">
        <OuiGrid cols="1" cols-md="2" gap="md">
          <OuiStack gap="xs">
            <OuiText size="xs" color="muted" weight="semibold" transform="uppercase">ID</OuiText>
            <OuiText size="sm" class="font-mono">{{ selectedLog.id }}</OuiText>
          </OuiStack>
          <OuiStack gap="xs">
            <OuiText size="xs" color="muted" weight="semibold" transform="uppercase">Time</OuiText>
            <OuiText size="sm">
              <OuiRelativeTime
                :value="selectedLog.createdAt ? date(selectedLog.createdAt) : undefined"
                :style="'long'"
              />
            </OuiText>
          </OuiStack>
          <OuiStack gap="xs">
            <OuiText size="xs" color="muted" weight="semibold" transform="uppercase">User</OuiText>
            <OuiStack gap="xs">
              <OuiText size="sm" weight="medium">{{ selectedLog.userName || selectedLog.userEmail || selectedLog.userId }}</OuiText>
              <OuiText size="xs" color="secondary" class="font-mono">{{ selectedLog.userId }}</OuiText>
            </OuiStack>
          </OuiStack>
          <OuiStack gap="xs">
            <OuiText size="xs" color="muted" weight="semibold" transform="uppercase">IP Address</OuiText>
            <OuiText size="sm" class="font-mono">{{ selectedLog.ipAddress }}</OuiText>
          </OuiStack>
          <OuiStack gap="xs">
            <OuiText size="xs" color="muted" weight="semibold" transform="uppercase">Service</OuiText>
            <OuiText size="sm" class="font-mono">{{ selectedLog.service }}</OuiText>
          </OuiStack>
          <OuiStack gap="xs">
            <OuiText size="xs" color="muted" weight="semibold" transform="uppercase">Action</OuiText>
            <OuiText size="sm">{{ selectedLog.action }}</OuiText>
          </OuiStack>
          <OuiStack gap="xs" v-if="selectedLog.resourceType">
            <OuiText size="xs" color="muted" weight="semibold" transform="uppercase">Resource Type</OuiText>
            <OuiText size="sm">{{ selectedLog.resourceType }}</OuiText>
          </OuiStack>
          <OuiStack gap="xs" v-if="selectedLog.resourceId">
            <OuiText size="xs" color="muted" weight="semibold" transform="uppercase">Resource ID</OuiText>
            <OuiText size="sm" class="font-mono">{{ selectedLog.resourceId }}</OuiText>
          </OuiStack>
          <OuiStack gap="xs">
            <OuiText size="xs" color="muted" weight="semibold" transform="uppercase">Status</OuiText>
            <OuiBadge :variant="getStatusVariant(selectedLog.responseStatus)" size="sm">
              {{ selectedLog.responseStatus }}
            </OuiBadge>
          </OuiStack>
          <OuiStack gap="xs">
            <OuiText size="xs" color="muted" weight="semibold" transform="uppercase">Duration</OuiText>
            <OuiText size="sm">{{ formatDuration(selectedLog.durationMs) }}</OuiText>
          </OuiStack>
        </OuiGrid>

        <OuiStack gap="xs" v-if="selectedLog.errorMessage">
          <OuiText size="xs" color="muted" weight="semibold" transform="uppercase">Error</OuiText>
          <OuiCard variant="outline" class="border-danger/20 bg-danger/5">
            <OuiCardBody class="p-3">
              <OuiText size="sm" color="danger">{{ selectedLog.errorMessage }}</OuiText>
            </OuiCardBody>
          </OuiCard>
        </OuiStack>

        <OuiStack gap="xs">
          <OuiText size="xs" color="muted" weight="semibold" transform="uppercase">Request Data</OuiText>
          <OuiCard variant="outline">
            <OuiCardBody class="p-3">
              <OuiCode :code="formatRequestData(selectedLog.requestData)" language="json" />
            </OuiCardBody>
          </OuiCard>
        </OuiStack>
      </OuiStack>
    </OuiDialog>
  </OuiStack>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue";
import {
  ArrowPathIcon,
  DocumentTextIcon,
  EyeIcon,
} from "@heroicons/vue/24/outline";
import { AuditService, type AuditLogEntry } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { date } from "@obiente/proto/utils";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import OuiCode from "~/components/oui/Code.vue";

interface Props {
  organizationId?: string;
  resourceType?: string;
  resourceId?: string;
  userId?: string;
}

const props = defineProps<Props>();

const client = useConnectClient(AuditService);

const auditLogs = ref<AuditLogEntry[]>([]);
const isLoading = ref(false);
const isLoadingMore = ref(false);
const total = ref(0);
const nextPageToken = ref<string | undefined>(undefined);
const hasNextPage = computed(() => !!nextPageToken.value);

const filters = ref({
  service: undefined as string | undefined,
  action: "",
  userId: "",
  status: undefined as string | undefined,
});

const serviceOptions = [
  { label: "DeploymentService", value: "DeploymentService" },
  { label: "OrganizationService", value: "OrganizationService" },
  { label: "GameServerService", value: "GameServerService" },
  { label: "BillingService", value: "BillingService" },
  { label: "SupportService", value: "SupportService" },
  { label: "AdminService", value: "AdminService" },
  { label: "SuperadminService", value: "SuperadminService" },
];

const statusOptions = [
  { label: "Success (200)", value: "200" },
  { label: "Error (400+)", value: "error" },
];

const columns = [
  { key: "action", label: "Action", sortable: true },
  { key: "service", label: "Service", sortable: true },
  { key: "user", label: "User", sortable: true },
  { key: "resource", label: "Resource", sortable: true },
  { key: "status", label: "Status", sortable: true },
  { key: "duration", label: "Duration", sortable: true },
  { key: "time", label: "Time", sortable: true },
  { key: "details", label: "", sortable: false },
];

const detailsDialogOpen = ref(false);
const selectedLog = ref<AuditLogEntry | null>(null);

const loadAuditLogs = async (append = false) => {
  if (isLoading.value || (isLoadingMore.value && append)) return;

  if (append) {
    isLoadingMore.value = true;
  } else {
    isLoading.value = true;
  }

  try {
    const request: any = {
      pageSize: 50,
    };

    if (props.organizationId) {
      request.organizationId = props.organizationId;
    }

    if (props.resourceType) {
      request.resourceType = props.resourceType;
    }

    if (props.resourceId) {
      request.resourceId = props.resourceId;
    }

    if (props.userId) {
      request.userId = props.userId;
    }

    if (filters.value.service) {
      request.service = filters.value.service;
    }

    if (filters.value.action) {
      request.action = filters.value.action;
    }

    if (filters.value.userId) {
      request.userId = filters.value.userId;
    }

    if (append && nextPageToken.value) {
      request.pageToken = nextPageToken.value;
    }

    const response = await client.listAuditLogs(request);

    if (append) {
      auditLogs.value.push(...(response.auditLogs || []));
    } else {
      auditLogs.value = response.auditLogs || [];
      total.value = response.auditLogs?.length || 0;
    }

    nextPageToken.value = response.nextPageToken || undefined;
  } catch (error) {
    console.error("Failed to load audit logs:", error);
  } finally {
    isLoading.value = false;
    isLoadingMore.value = false;
  }
};

const refresh = () => {
  nextPageToken.value = undefined;
  loadAuditLogs(false);
};

const loadMore = () => {
  loadAuditLogs(true);
};

const showDetails = (log: AuditLogEntry) => {
  selectedLog.value = log;
  detailsDialogOpen.value = true;
};

const getStatusVariant = (status: number): "success" | "danger" | "warning" | "secondary" => {
  if (status >= 200 && status < 300) {
    return "success";
  } else if (status >= 400 && status < 500) {
    return "warning";
  } else if (status >= 500) {
    return "danger";
  }
  return "secondary";
};

const getRowClass = (row: AuditLogEntry) => {
  const status = row.responseStatus;
  if (status >= 400) {
    return "bg-danger/5";
  }
  return "";
};

const formatDuration = (ms: number | bigint): string => {
  const msNum = typeof ms === 'bigint' ? Number(ms) : ms;
  if (msNum < 1000) {
    return `${msNum}ms`;
  }
  return `${(msNum / 1000).toFixed(2)}s`;
};

const formatRequestData = (data: string): string => {
  try {
    const parsed = JSON.parse(data);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return data;
  }
};

// Watch filters and reload
watch(
  () => [filters.value, props.organizationId, props.resourceType, props.resourceId],
  () => {
    refresh();
  },
  { deep: true }
);

onMounted(() => {
  loadAuditLogs();
});
</script>

