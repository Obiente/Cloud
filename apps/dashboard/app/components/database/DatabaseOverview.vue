<template>
  <OuiStack gap="md">
    <!-- Sleeping Notice -->
    <OuiCard v-if="props.database?.status === DatabaseStatus.SLEEPING" variant="outline" status="warning">
      <OuiCardBody>
        <OuiFlex align="center" gap="sm">
          <OuiText size="lg">💤</OuiText>
          <OuiStack gap="xs">
            <OuiText size="sm" weight="semibold">Database is sleeping</OuiText>
            <OuiText size="xs" color="tertiary">
              It will start automatically when a connection is made. First connection may take a few seconds.
            </OuiText>
          </OuiStack>
        </OuiFlex>
      </OuiCardBody>
    </OuiCard>

    <!-- Quick Connect Bar -->
    <OuiCard variant="outline">
      <OuiCardBody>
        <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
          <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
            <div class="h-8 w-8 rounded-lg bg-surface-muted flex items-center justify-center shrink-0">
              <ServerStackIcon class="h-4 w-4 text-accent-primary" />
            </div>
            <OuiStack gap="none" class="min-w-0">
              <OuiText size="sm" weight="medium" truncate class="font-mono">
                {{ database.host || 'N/A' }}:{{ database.port }}
              </OuiText>
              <OuiText size="xs" color="tertiary">{{ getTypeLabel(database.type) }} · {{ database.name || 'N/A' }}</OuiText>
            </OuiStack>
          </OuiFlex>
          <OuiFlex gap="xs" class="shrink-0">
            <OuiBadge :variant="statusMeta.variant" size="xs">
              <span class="inline-flex h-1.5 w-1.5 rounded-full mr-1" :class="statusMeta.dotClass" />
              {{ statusMeta.label }}
            </OuiBadge>
          </OuiFlex>
        </OuiFlex>
      </OuiCardBody>
    </OuiCard>

    <!-- Resource Cards -->
    <OuiGrid :cols="{ sm: 2, md: 4 }" gap="sm">
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="sm">
            <OuiFlex align="center" gap="xs">
              <CpuChipIcon class="h-3.5 w-3.5 text-accent-primary" />
              <OuiText size="xs" color="tertiary">CPU Cores</OuiText>
            </OuiFlex>
            <OuiText size="xl" weight="semibold">
              {{ database.cpuCores }}
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="sm">
            <OuiFlex align="center" gap="xs">
              <CircleStackIcon class="h-3.5 w-3.5 text-accent-info" />
              <OuiText size="xs" color="tertiary">Memory</OuiText>
            </OuiFlex>
            <OuiText size="xl" weight="semibold">
              {{ formatBytes(database.memoryBytes) }}
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="sm">
            <OuiFlex align="center" gap="xs">
              <ArchiveBoxIcon class="h-3.5 w-3.5 text-success" />
              <OuiText size="xs" color="tertiary">Storage</OuiText>
            </OuiFlex>
            <OuiText size="xl" weight="semibold">
              {{ formatBytes(database.diskBytes) }}
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="sm">
            <OuiFlex align="center" gap="xs">
              <ClockIcon class="h-3.5 w-3.5 text-accent-secondary" />
              <OuiText size="xs" color="tertiary">Created</OuiText>
            </OuiFlex>
            <OuiText size="sm" weight="semibold">
              {{ formatDate(database.createdAt) }}
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiGrid>

    <!-- Usage & Billing -->
    <UsageStatistics v-if="usageData" :usage-data="usageData" />
    <CostBreakdown v-if="usageData" :usage-data="usageData" />
  </OuiStack>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import {
  ServerStackIcon,
  CpuChipIcon,
  CircleStackIcon,
  ArchiveBoxIcon,
  ClockIcon,
} from "@heroicons/vue/24/outline";
import { DatabaseType, DatabaseStatus, DatabaseService, type DatabaseInstance } from "@obiente/proto";
import { formatBytes, formatDate } from "~/utils/common";
import { useToast } from "~/composables/useToast";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";
import UsageStatistics from "~/components/shared/UsageStatistics.vue";
import CostBreakdown from "~/components/shared/CostBreakdown.vue";

const props = defineProps<{
  database: DatabaseInstance;
}>();

const { toast } = useToast();
const organizationId = useOrganizationId();
const dbClient = useConnectClient(DatabaseService);
const usageData = ref<any>(null);

async function loadUsage() {
  try {
    if (!organizationId.value || !props.database?.id) return;
    const month = new Date().toISOString().slice(0, 7);
    const res = await dbClient.getDatabaseUsage({
      databaseId: props.database.id,
      organizationId: organizationId.value,
      month,
    });
    usageData.value = res;
  } catch (err) {
    // Usage data is optional, don't block the overview
    console.error("Failed to fetch database usage:", err);
  }
}

onMounted(() => {
  loadUsage();
});
const hostCopied = ref(false);
const portCopied = ref(false);
const usernameCopied = ref(false);

function getTypeLabel(type: number): string {
  const types: Record<number, string> = {
    [DatabaseType.POSTGRESQL]: "PostgreSQL",
    [DatabaseType.MYSQL]: "MySQL",
    [DatabaseType.MONGODB]: "MongoDB",
    [DatabaseType.REDIS]: "Redis",
    [DatabaseType.MARIADB]: "MariaDB",
  };
  return types[type] || `Type ${type}`;
}

const statusMetaMap: Record<number, { label: string; variant: "success" | "danger" | "warning" | "primary" | "secondary"; dotClass: string }> = {
  [DatabaseStatus.CREATING]: { label: "Creating", variant: "warning" as const, dotClass: "bg-warning" },
  [DatabaseStatus.RUNNING]: { label: "Running", variant: "success" as const, dotClass: "bg-success" },
  [DatabaseStatus.STOPPED]: { label: "Stopped", variant: "secondary" as const, dotClass: "bg-secondary" },
  [DatabaseStatus.FAILED]: { label: "Failed", variant: "danger" as const, dotClass: "bg-danger" },
  [DatabaseStatus.BACKING_UP]: { label: "Backing Up", variant: "primary" as const, dotClass: "bg-primary" },
  [DatabaseStatus.RESTORING]: { label: "Restoring", variant: "primary" as const, dotClass: "bg-primary" },
  [DatabaseStatus.STOPPING]: { label: "Stopping", variant: "warning" as const, dotClass: "bg-warning" },
  [DatabaseStatus.DELETING]: { label: "Deleting", variant: "danger" as const, dotClass: "bg-danger" },
  [DatabaseStatus.SLEEPING]: { label: "Sleeping", variant: "secondary" as const, dotClass: "bg-secondary" },
};

const statusMeta = statusMetaMap[props.database?.status] || { label: "Unknown", variant: "secondary" as const, dotClass: "bg-secondary" };
</script>
