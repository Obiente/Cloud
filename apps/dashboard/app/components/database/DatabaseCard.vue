<template>
  <OuiCard
    v-if="!loading"
    variant="default"
    class="cursor-pointer hover:ring-2 hover:ring-primary/20 transition-all"
    @click="$emit('click')"
  >
    <OuiCardBody>
      <OuiStack gap="md">
        <OuiFlex justify="between" align="start">
          <OuiStack gap="xs">
            <OuiText as="h3" size="lg" weight="semibold" class="truncate">
              {{ database.name }}
            </OuiText>
            <OuiText color="secondary" size="sm">
              {{ getTypeLabel(database.type) }}
            </OuiText>
          </OuiStack>
          <OuiBadge :color="getStatusColor(database.status)">
            {{ getStatusLabel(database.status) }}
          </OuiBadge>
        </OuiFlex>

        <OuiStack gap="xs">
          <OuiFlex justify="between" align="center">
            <OuiText color="secondary" size="sm">Size</OuiText>
            <OuiText size="sm" weight="medium">{{ database.size || "N/A" }}</OuiText>
          </OuiFlex>
          <OuiFlex justify="between" align="center">
            <OuiText color="secondary" size="sm">CPU</OuiText>
            <OuiText size="sm" weight="medium">{{ database.cpuCores }} cores</OuiText>
          </OuiFlex>
          <OuiFlex justify="between" align="center">
            <OuiText color="secondary" size="sm">Memory</OuiText>
            <OuiText size="sm" weight="medium">{{ formatBytes(database.memoryBytes) }}</OuiText>
          </OuiFlex>
          <OuiFlex justify="between" align="center">
            <OuiText color="secondary" size="sm">Storage</OuiText>
            <OuiText size="sm" weight="medium">{{ formatBytes(database.diskBytes) }}</OuiText>
          </OuiFlex>
        </OuiStack>

        <OuiFlex justify="between" align="center" class="pt-2 border-t border-border-muted">
          <OuiText color="secondary" size="xs">
            Created {{ formatDate(database.createdAt) }}
          </OuiText>
          <OuiButton
            variant="ghost"
            size="sm"
            @click.stop="navigateToDatabase(database.id)"
          >
            View
          </OuiButton>
        </OuiFlex>
      </OuiStack>
    </OuiCardBody>
  </OuiCard>

  <OuiCard v-else variant="default">
    <OuiCardBody>
      <OuiStack gap="md">
        <OuiSkeleton height="24px" width="60%" />
        <OuiSkeleton height="16px" width="40%" />
        <OuiStack gap="xs">
          <OuiSkeleton height="16px" />
          <OuiSkeleton height="16px" />
          <OuiSkeleton height="16px" />
        </OuiStack>
      </OuiStack>
    </OuiCardBody>
  </OuiCard>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";
import { DatabaseType, DatabaseStatus } from "@obiente/proto";
import { formatBytes, formatDate } from "~/utils/common";

const props = defineProps<{
  database?: any;
  loading?: boolean;
}>();

defineEmits<{
  click: [];
}>();

const router = useRouter();

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

function getStatusLabel(status: number | string): string {
  const statusValue = typeof status === "string" ? parseInt(status) : status;
  const statuses: Record<number, string> = {
    [DatabaseStatus.CREATING]: "Creating",
    [DatabaseStatus.STARTING]: "Starting",
    [DatabaseStatus.RUNNING]: "Running",
    [DatabaseStatus.STOPPING]: "Stopping",
    [DatabaseStatus.STOPPED]: "Stopped",
    [DatabaseStatus.BACKING_UP]: "Backing Up",
    [DatabaseStatus.RESTORING]: "Restoring",
    [DatabaseStatus.FAILED]: "Failed",
    [DatabaseStatus.DELETING]: "Deleting",
    [DatabaseStatus.DELETED]: "Deleted",
    [DatabaseStatus.SUSPENDED]: "Suspended",
  };
  return statuses[statusValue] || `Status ${statusValue}`;
}

function getStatusColor(status: number | string): string {
  const statusValue = typeof status === "string" ? parseInt(status) : status;
  const colors: Record<number, string> = {
    [DatabaseStatus.CREATING]: "warning",
    [DatabaseStatus.STARTING]: "info",
    [DatabaseStatus.RUNNING]: "success",
    [DatabaseStatus.STOPPING]: "warning",
    [DatabaseStatus.STOPPED]: "secondary",
    [DatabaseStatus.BACKING_UP]: "info",
    [DatabaseStatus.RESTORING]: "info",
    [DatabaseStatus.FAILED]: "danger",
    [DatabaseStatus.DELETING]: "warning",
    [DatabaseStatus.DELETED]: "secondary",
    [DatabaseStatus.SUSPENDED]: "secondary",
  };
  return colors[statusValue] || "secondary";
}

function navigateToDatabase(id: string) {
  router.push(`/databases/${id}`);
}
</script>

