<template>
  <ResourceCard
    :title="database?.name || ''"
    :subtitle="databaseSubtitle"
    :status-meta="statusMeta"
    :resources="resources"
    :created-at="createdAtDate"
    :detail-url="database ? `/databases/${database.id}` : undefined"
    :is-actioning="isActioning"
    :loading="loading"
  >
    <template #icon>
      <CircleStackIcon class="h-5 w-5 shrink-0" />
    </template>

    <template #actions>
      <OuiFlex gap="xs" wrap="wrap">
        <!-- Database Type Badge -->
        <OuiBadge v-if="!loading && database" variant="secondary" size="sm">
          {{ getTypeLabel(database.type) }}
        </OuiBadge>

        <!-- Action Buttons -->
        <OuiButton
          v-if="!loading && database && canRestart"
          variant="ghost"
          size="sm"
          icon-only
          @click.stop="handleRestart"
          title="Restart"
        >
          <ArrowPathIcon class="h-4 w-4" />
        </OuiButton>

        <OuiButton
          v-if="!loading && database && canDelete"
          variant="ghost"
          size="sm"
          icon-only
          @click.stop="handleDelete"
          title="Delete"
        >
          <TrashIcon class="h-4 w-4" />
        </OuiButton>
      </OuiFlex>
    </template>

    <template #resources>
      <!-- Skeleton -->
      <OuiGrid v-if="loading" :cols="{ sm: 2 }" gap="sm">
        <OuiBox p="sm" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted opacity-30">
          <OuiStack gap="xs">
            <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider opacity-50">
              CPU Cores
            </OuiText>
            <OuiSkeleton width="3rem" height="1.5rem" variant="text" />
          </OuiStack>
        </OuiBox>
        <OuiBox p="sm" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted opacity-30">
          <OuiStack gap="xs">
            <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider opacity-50">
              Memory
            </OuiText>
            <OuiSkeleton width="3rem" height="1.5rem" variant="text" />
          </OuiStack>
        </OuiBox>
        <OuiBox p="sm" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted opacity-30">
          <OuiStack gap="xs">
            <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider opacity-50">
              Storage
            </OuiText>
            <OuiSkeleton width="3rem" height="1.5rem" variant="text" />
          </OuiStack>
        </OuiBox>
        <OuiBox p="sm" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted opacity-30">
          <OuiStack gap="xs">
            <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider opacity-50">
              Port
            </OuiText>
            <OuiSkeleton width="3rem" height="1.5rem" variant="text" />
          </OuiStack>
        </OuiBox>
      </OuiGrid>

      <!-- Actual Content -->
      <OuiGrid v-else-if="database" :cols="{ sm: 2 }" gap="sm">
        <OuiBox p="sm" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
          <OuiStack gap="xs">
            <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
              CPU Cores
            </OuiText>
            <OuiText size="lg" weight="bold" color="primary">
              {{ database.cpuCores }}
            </OuiText>
          </OuiStack>
        </OuiBox>

        <OuiBox p="sm" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
          <OuiStack gap="xs">
            <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
              Memory
            </OuiText>
            <OuiText size="lg" weight="bold" color="primary">
              {{ formatBytes(database.memoryBytes) }}
            </OuiText>
          </OuiStack>
        </OuiBox>

        <OuiBox p="sm" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
          <OuiStack gap="xs">
            <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
              Storage
            </OuiText>
            <OuiText size="lg" weight="bold" color="primary">
              {{ formatBytes(database.diskBytes) }}
            </OuiText>
          </OuiStack>
        </OuiBox>

        <OuiBox p="sm" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
          <OuiStack gap="xs">
            <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
              Port
            </OuiText>
            <OuiText size="lg" weight="bold" color="primary" class="font-mono">
              {{ database.port || 'N/A' }}
            </OuiText>
          </OuiStack>
        </OuiBox>
      </OuiGrid>
    </template>

    <template #info>
      <!-- Connection Hostname Info -->
      <OuiStack v-if="!loading && database?.host" gap="sm" p="md" rounded="xl" class="bg-surface-muted/30 border border-border-muted/50">
        <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
          Connection Hostname
        </OuiText>
          <OuiCode :code="database.host" class="flex-1 min-w-0" />
      </OuiStack>
    </template>
  </ResourceCard>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import {
  CircleStackIcon,
  TrashIcon,
  ArrowPathIcon,
  DocumentDuplicateIcon,
  CheckIcon,
} from "@heroicons/vue/24/outline";
import { DatabaseType, DatabaseStatus } from "@obiente/proto";
import { formatBytes } from "~/utils/common";
import { useDialog } from "~/composables/useDialog";
import { useToast } from "~/composables/useToast";
import { useConnectClient } from "~/lib/connect-client";
import { DatabaseService } from "@obiente/proto";
import { useOrganizationId } from "~/composables/useOrganizationId";
import ResourceCard from "~/components/shared/ResourceCard.vue";
import OuiCode from "~/components/oui/Code.vue";

const props = defineProps<{
  database?: any;
  loading?: boolean;
}>();

defineEmits<{
  click: [];
  refresh: [];
}>();

const { showConfirm, showAlert } = useDialog();
const { toast } = useToast();
const client = useConnectClient(DatabaseService);
const organizationId = useOrganizationId();
const isActioning = ref(false);
const hostCopied = ref(false);

const databaseSubtitle = computed(() => {
  if (!props.database) return "";
  return `${getTypeLabel(props.database.type)} • Port ${props.database.port || 'N/A'}`;
});

const createdAtDate = computed(() => {
  if (!props.database?.createdAt) return undefined;
  return new Date(props.database.createdAt);
});

const statusMeta = computed(() => {
  if (!props.database) {
    return {
      label: "Unknown",
      badge: "secondary" as const,
      cardClass: "",
      barClass: "bg-secondary",
      beforeGradient: "",
      iconClass: "",
    };
  }

  const status = props.database.status;
  const colors: Record<number, { label: string; badge: string; bar: string; card: string }> = {
    [DatabaseStatus.CREATING]: { label: "Creating", badge: "warning", bar: "bg-warning", card: "hover:ring-warning/20" },
    [DatabaseStatus.RUNNING]: { label: "Running", badge: "success", bar: "bg-success", card: "hover:ring-success/20" },
    [DatabaseStatus.STOPPED]: { label: "Stopped", badge: "secondary", bar: "bg-secondary", card: "" },
    [DatabaseStatus.FAILED]: { label: "Failed", badge: "danger", bar: "bg-danger", card: "hover:ring-danger/20" },
    [DatabaseStatus.BACKING_UP]: { label: "Backing Up", badge: "warning", bar: "bg-warning", card: "hover:ring-warning/20" },
    [DatabaseStatus.RESTORING]: { label: "Restoring", badge: "warning", bar: "bg-warning", card: "hover:ring-warning/20" },
    [DatabaseStatus.STOPPING]: { label: "Stopping", badge: "warning", bar: "bg-warning", card: "" },
    [DatabaseStatus.DELETING]: { label: "Deleting", badge: "danger", bar: "bg-danger", card: "" },
    [DatabaseStatus.SLEEPING]: { label: "Sleeping", badge: "secondary", bar: "bg-secondary", card: "" },
  };

  const meta = colors[status] || { label: "Unknown", badge: "secondary", bar: "bg-secondary", card: "" };

  return {
    label: meta.label,
    badge: meta.badge as "success" | "danger" | "warning" | "secondary",
    cardClass: meta.card,
    barClass: meta.bar,
    beforeGradient: "",
    iconClass: "",
  };
});

const resources = computed(() => {
  if (!props.database) return [];

  return [
    {
      icon: undefined,
      label: "CPU",
      value: `${props.database.cpuCores} cores`,
    },
    {
      icon: undefined,
      label: "Memory",
      value: formatBytes(props.database.memoryBytes),
    },
    {
      icon: undefined,
      label: "Storage",
      value: formatBytes(props.database.diskBytes),
    },
  ];
});

const canRestart = computed(() => {
  if (!props.database) return false;
  return [DatabaseStatus.RUNNING, DatabaseStatus.STOPPED].includes(props.database.status);
});

const canDelete = computed(() => {
  if (!props.database) return false;
  return props.database.status !== DatabaseStatus.DELETING && props.database.status !== DatabaseStatus.DELETED;
});

function getTypeLabel(type: number | string): string {
  const typeValue = typeof type === "string" ? parseInt(type) : type;
  const types: Record<number, string> = {
    [DatabaseType.POSTGRESQL]: "PostgreSQL",
    [DatabaseType.MYSQL]: "MySQL",
    [DatabaseType.MONGODB]: "MongoDB",
    [DatabaseType.REDIS]: "Redis",
    [DatabaseType.MARIADB]: "MariaDB",
  };
  return types[typeValue] || `Type ${typeValue}`;
}

async function handleRestart() {
  if (!props.database) return;

  const confirmed = await showConfirm({
    title: "Restart Database",
    message: `Are you sure you want to restart ${props.database.name}? This will cause a brief downtime.`,
    confirmLabel: "Restart",
    cancelLabel: "Cancel",
  });

  if (!confirmed) return;

  isActioning.value = true;
  try {
    // Call restart API
    await client.restartDatabase({
      databaseId: props.database.id,
      organizationId: organizationId.value || undefined,
    });
    toast.success("Database restart initiated");
  } catch (error) {
    console.error("Failed to restart database:", error);
    await showAlert({
      title: "Failed to restart database",
      message: error instanceof Error ? error.message : "An unknown error occurred",
    });
  } finally {
    isActioning.value = false;
  }
}

async function handleDelete() {
  if (!props.database) return;

  const confirmed = await showConfirm({
    title: "Delete Database",
    message: `Are you sure you want to delete ${props.database.name}? This action cannot be undone and all data will be lost.`,
    confirmLabel: "Delete",
    cancelLabel: "Cancel",
    variant: "danger",
  });

  if (!confirmed) return;

  isActioning.value = true;
  try {
    await client.deleteDatabase({
      databaseId: props.database.id,
      organizationId: organizationId.value || undefined,
    });
    toast.success("Database deleted successfully");
  } catch (error) {
    console.error("Failed to delete database:", error);
    await showAlert({
      title: "Failed to delete database",
      message: error instanceof Error ? error.message : "An unknown error occurred",
    });
  } finally {
    isActioning.value = false;
  }
}
</script>