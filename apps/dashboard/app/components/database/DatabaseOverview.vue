<template>
  <OuiStack gap="lg" p="lg" md:p="xl">
    <!-- Connection Information -->
    <OuiStack gap="md">
      <OuiText as="h3" size="lg" weight="bold">Connection Information</OuiText>

      <OuiGrid :cols="{ sm: 1, md: 2 }" gap="md">
        <OuiStack gap="sm" p="md" rounded="lg" class="bg-surface-muted/30 border border-border-muted/50">
          <OuiFlex align="center" gap="sm" justify="between" wrap="wrap">
            <OuiStack gap="xs" class="flex-1 min-w-0">
              <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
                Hostname
              </OuiText>
              <OuiCode :code="database.host || 'N/A'" class="flex-1 min-w-0" />
            </OuiStack>
          </OuiFlex>
        </OuiStack>

        <OuiStack gap="sm" p="md" rounded="lg" class="bg-surface-muted/30 border border-border-muted/50">
          <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
            Port
          </OuiText>
          <OuiCode :code="String(database.port)" />
        </OuiStack>

        <!-- <OuiStack gap="sm" p="md" rounded="lg" class="bg-surface-muted/30 border border-border-muted/50">
          <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
            Username
          </OuiText>
            <OuiCode :code="database.username" />
        </OuiStack> -->
      </OuiGrid>
    </OuiStack>

    <!-- Sleeping Notice -->
    <OuiStack v-if="props.database?.status === DatabaseStatus.SLEEPING" gap="sm" p="md" rounded="lg"
      class="bg-secondary/10 border border-secondary/30">
      <OuiFlex align="center" gap="sm">
        <OuiText size="lg">💤</OuiText>
        <OuiStack gap="xs">
          <OuiText size="sm" weight="semibold" color="primary">This database is sleeping</OuiText>
          <OuiText size="xs" color="secondary">It will start automatically when a connection is made. The first
            connection may take a few seconds while the container starts.</OuiText>
        </OuiStack>
      </OuiFlex>
    </OuiStack>

    <!-- Resource Information -->
    <OuiStack gap="md">
      <OuiText as="h3" size="lg" weight="bold">Resources</OuiText>

      <OuiGrid :cols="{ sm: 2, md: 4 }" gap="md">
        <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
          <OuiStack gap="xs">
            <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
              CPU Cores
            </OuiText>
            <OuiText size="lg" weight="bold" color="primary">
              {{ database.cpuCores }}
            </OuiText>
          </OuiStack>
        </OuiBox>

        <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
          <OuiStack gap="xs">
            <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
              Memory
            </OuiText>
            <OuiText size="lg" weight="bold" color="primary">
              {{ formatBytes(database.memoryBytes) }}
            </OuiText>
          </OuiStack>
        </OuiBox>

        <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
          <OuiStack gap="xs">
            <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
              Storage
            </OuiText>
            <OuiText size="lg" weight="bold" color="primary">
              {{ formatBytes(database.diskBytes) }}
            </OuiText>
          </OuiStack>
        </OuiBox>

        <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
          <OuiStack gap="xs">
            <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
              Database Name
            </OuiText>
            <OuiText size="lg" weight="bold" color="primary" class="truncate">
              {{ database.name || 'N/A' }}
            </OuiText>
          </OuiStack>
        </OuiBox>
      </OuiGrid>
    </OuiStack>

    <!-- Usage & Billing -->
    <UsageStatistics v-if="usageData" :usage-data="usageData" />
    <CostBreakdown v-if="usageData" :usage-data="usageData" />

    <!-- Database Details -->
    <OuiStack gap="md">
      <OuiText as="h3" size="lg" weight="bold">Details</OuiText>

      <OuiGrid :cols="{ sm: 1, md: 2 }" gap="md">
        <OuiStack gap="sm" p="md" rounded="lg" class="bg-surface-muted/30 border border-border-muted/50">
          <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
            Type
          </OuiText>
          <OuiText size="sm" color="primary" weight="medium">
            {{ getTypeLabel(database.type) }}
          </OuiText>
        </OuiStack>

        <OuiStack gap="sm" p="md" rounded="lg" class="bg-surface-muted/30 border border-border-muted/50">
          <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
            Status
          </OuiText>
          <OuiBadge :variant="statusMeta.variant" size="sm">
            <span class="inline-flex h-1.5 w-1.5 rounded-full mr-1.5" :class="statusMeta.dotClass" />
            <OuiText as="span" size="xs" weight="semibold">{{ statusMeta.label }}</OuiText>
          </OuiBadge>
        </OuiStack>

        <OuiStack gap="sm" p="md" rounded="lg" class="bg-surface-muted/30 border border-border-muted/50">
          <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
            Created
          </OuiText>
          <OuiText size="sm" color="primary" weight="medium">
            {{ formatDate(database.createdAt) }}
          </OuiText>
        </OuiStack>

        <OuiStack gap="sm" p="md" rounded="lg" class="bg-surface-muted/30 border border-border-muted/50">
          <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
            Last Updated
          </OuiText>
          <OuiText size="sm" color="primary" weight="medium">
            {{ formatDate(database.updatedAt) }}
          </OuiText>
        </OuiStack>
      </OuiGrid>
    </OuiStack>
  </OuiStack>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { DatabaseType, DatabaseStatus, DatabaseService, type DatabaseInstance } from "@obiente/proto";
import { formatBytes, formatDate } from "~/utils/common";
import { useToast } from "~/composables/useToast";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";
import UsageStatistics from "~/components/shared/UsageStatistics.vue";
import CostBreakdown from "~/components/shared/CostBreakdown.vue";
import OuiCode from "~/components/oui/Code.vue";

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
