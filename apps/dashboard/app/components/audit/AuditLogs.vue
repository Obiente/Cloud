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
    <OuiCard v-if="hasMultipleFilterOptions" variant="outline">
      <OuiCardBody>
        <OuiGrid cols="1" cols-md="2" cols-lg="4" gap="md">
          <OuiStack v-if="serviceOptions.length > 1" gap="xs">
            <OuiText size="sm" weight="medium">Service</OuiText>
            <OuiSelect
              v-model="filters.service"
              :items="serviceOptions"
              placeholder="All services"
              clearable
            />
          </OuiStack>
          <OuiStack v-if="actionOptions.length > 1" gap="xs">
            <OuiText size="sm" weight="medium">Action</OuiText>
            <OuiSelect
              v-model="filters.action"
              :items="actionOptions"
              placeholder="All actions"
              clearable
            />
          </OuiStack>
          <OuiStack v-if="userOptions.length > 1" gap="xs">
            <OuiText size="sm" weight="medium">User</OuiText>
            <OuiSelect
              v-model="filters.userId"
              :items="userOptions"
              placeholder="All users"
              clearable
            />
          </OuiStack>
          <OuiStack v-if="statusOptions.length > 1" gap="xs">
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

// Separate data for filter options (loaded without filters to show all available values)
const filterOptionsData = ref<AuditLogEntry[]>([]);
const isLoadingFilterOptions = ref(false);

const filters = ref({
  service: undefined as string | undefined,
  action: undefined as string | undefined,
  userId: undefined as string | undefined,
  status: undefined as string | undefined,
});

// Load filter options from a large unfiltered sample
const loadFilterOptions = async () => {
  if (isLoadingFilterOptions.value) return;
  isLoadingFilterOptions.value = true;

  try {
    const request: any = {
      pageSize: 1000, // Load a large sample to get all available filter values
    };

    // Apply organization filter for filter options (if provided as prop)
    if (props.organizationId) {
      request.organizationId = props.organizationId;
    }

    // If resourceType/resourceId are provided, filter options to that resource
    // This ensures filter options only show values relevant to the current resource
    if (props.resourceType) {
      request.resourceType = props.resourceType;
    }
    if (props.resourceId) {
      request.resourceId = props.resourceId;
    }

    const response = await client.listAuditLogs(request);
    filterOptionsData.value = response.auditLogs || [];
  } catch (error) {
    console.error("Failed to load filter options:", error);
  } finally {
    isLoadingFilterOptions.value = false;
  }
};

// Dynamic filter options based on all available audit logs (not filtered results)
const serviceOptions = computed(() => {
  const services = new Set<string>();
  filterOptionsData.value.forEach((log) => {
    if (log.service) {
      services.add(log.service);
    }
  });
  return Array.from(services)
    .sort()
    .map((service) => ({ label: service, value: service }));
});

const actionOptions = computed(() => {
  const actions = new Set<string>();
  filterOptionsData.value.forEach((log) => {
    if (log.action) {
      actions.add(log.action);
    }
  });
  return Array.from(actions)
    .sort()
    .map((action) => ({ label: action, value: action }));
});

const userOptions = computed(() => {
  const users = new Map<string, { name?: string; email?: string }>();
  filterOptionsData.value.forEach((log) => {
    if (log.userId) {
      if (!users.has(log.userId)) {
        users.set(log.userId, {
          name: log.userName || undefined,
          email: log.userEmail || undefined,
        });
      } else {
        // Update if we have better info (name/email)
        const existing = users.get(log.userId)!;
        if (!existing.name && log.userName) {
          existing.name = log.userName;
        }
        if (!existing.email && log.userEmail) {
          existing.email = log.userEmail;
        }
      }
    }
  });
  return Array.from(users.entries())
    .sort(([a], [b]) => {
      const aInfo = users.get(a)!;
      const bInfo = users.get(b)!;
      const aDisplay = aInfo.name || aInfo.email || a;
      const bDisplay = bInfo.name || bInfo.email || b;
      return aDisplay.localeCompare(bDisplay);
    })
    .map(([userId, info]) => {
      const displayName = info.name || info.email || userId;
      return { label: displayName, value: userId };
    });
});

const statusOptions = computed(() => {
  const statuses = new Set<number>();
  filterOptionsData.value.forEach((log) => {
    if (log.responseStatus) {
      statuses.add(log.responseStatus);
    }
  });
  const options = Array.from(statuses)
    .sort((a, b) => a - b)
    .map((status) => {
      const label = status >= 200 && status < 300 
        ? `Success (${status})` 
        : status >= 400 && status < 500
        ? `Client Error (${status})`
        : status >= 500
        ? `Server Error (${status})`
        : `Status ${status}`;
      return { label, value: status.toString() };
    });
  // Add common options if not already present
  if (!statuses.has(200)) {
    options.unshift({ label: "Success (200)", value: "200" });
  }
  if (!statuses.has(400) && !statuses.has(500)) {
    options.push({ label: "Error (400+)", value: "error" });
  }
  return options;
});

// Check if there are multiple filter options (to decide if we should show the filter card)
const hasMultipleFilterOptions = computed(() => {
  return serviceOptions.value.length > 1 ||
         actionOptions.value.length > 1 ||
         userOptions.value.length > 1 ||
         statusOptions.value.length > 1;
});

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

    // Apply filters - check for truthy values (handles undefined, null, empty string)
    // OuiSelect sets value to null when cleared
    if (filters.value.service != null && filters.value.service !== "") {
      request.service = filters.value.service;
    }

    if (filters.value.action != null && filters.value.action !== "") {
      request.action = filters.value.action;
    }

    if (filters.value.userId != null && filters.value.userId !== "") {
      request.userId = filters.value.userId;
    }

    // Only use pagination token when appending (loading more)
    if (append && nextPageToken.value) {
      request.pageToken = nextPageToken.value;
    }

    const response = await client.listAuditLogs(request);

    let logs = response.auditLogs || [];
    
    // Client-side status filtering (since backend doesn't support it)
    if (filters.value.status != null && filters.value.status !== "") {
      if (filters.value.status === "error") {
        logs = logs.filter((log) => log.responseStatus >= 400);
      } else {
        const statusNum = parseInt(filters.value.status, 10);
        if (!isNaN(statusNum)) {
          logs = logs.filter((log) => log.responseStatus === statusNum);
        }
      }
    }
    
    if (append) {
      auditLogs.value.push(...logs);
    } else {
      auditLogs.value = logs;
      total.value = logs.length;
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

// Watch filters and reload when they change
watch(
  () => [
    filters.value.service,
    filters.value.action,
    filters.value.userId,
    filters.value.status,
  ],
  () => {
    // Reset pagination when filters change
    nextPageToken.value = undefined;
    loadAuditLogs(false);
  }
);

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
  
  // Less than 1 second - show milliseconds
  if (msNum < 1000) {
    return `${msNum}ms`;
  }
  
  // Less than 1 minute - show seconds with appropriate precision
  if (msNum < 60000) {
    const seconds = msNum / 1000;
    // If less than 10 seconds, show 2 decimal places, otherwise 1
    if (seconds < 10) {
      return `${seconds.toFixed(2)}s`;
    }
    return `${seconds.toFixed(1)}s`;
  }
  
  // 1 minute or more - show minutes and seconds
  const minutes = Math.floor(msNum / 60000);
  const remainingSeconds = Math.floor((msNum % 60000) / 1000);
  
  if (remainingSeconds === 0) {
    return `${minutes}m`;
  }
  return `${minutes}m ${remainingSeconds}s`;
};

const formatRequestData = (data: string): string => {
  try {
    const parsed = JSON.parse(data);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return data;
  }
};

// Watch props and reload when they change
watch(
  () => [props.organizationId, props.resourceType, props.resourceId, props.userId],
  () => {
    // Reload filter options when resource context changes
    loadFilterOptions();
    refresh();
  }
);

onMounted(() => {
  // Load filter options first, then load the actual filtered results
  loadFilterOptions();
  loadAuditLogs();
});
</script>

