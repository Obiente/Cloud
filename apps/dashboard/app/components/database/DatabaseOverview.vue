<template>
  <OuiStack gap="lg">
    <OuiGrid cols="1" cols-md="2" cols-lg="4" gap="md">
      <OuiCard>
        <OuiCardBody>
          <OuiStack gap="xs">
            <OuiText color="secondary" size="sm">Status</OuiText>
            <OuiBadge :color="getStatusColor(database.status)">
              {{ getStatusLabel(database.status) }}
            </OuiBadge>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <OuiCard>
        <OuiCardBody>
          <OuiStack gap="xs">
            <OuiText color="secondary" size="sm">CPU Cores</OuiText>
            <OuiText size="xl" weight="bold">{{ database.cpuCores }}</OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <OuiCard>
        <OuiCardBody>
          <OuiStack gap="xs">
            <OuiText color="secondary" size="sm">Memory</OuiText>
            <OuiText size="xl" weight="bold">{{ formatBytes(database.memoryBytes) }}</OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <OuiCard>
        <OuiCardBody>
          <OuiStack gap="xs">
            <OuiText color="secondary" size="sm">Storage</OuiText>
            <OuiText size="xl" weight="bold">{{ formatBytes(database.diskBytes) }}</OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiGrid>

    <OuiCard>
      <OuiCardHeader>
        <OuiCardTitle>Database Information</OuiCardTitle>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiFlex justify="between">
            <OuiText color="secondary">Name</OuiText>
            <OuiText weight="medium">{{ database.name }}</OuiText>
          </OuiFlex>
          <OuiFlex justify="between" v-if="database.description">
            <OuiText color="secondary">Description</OuiText>
            <OuiText>{{ database.description }}</OuiText>
          </OuiFlex>
          <OuiFlex justify="between">
            <OuiText color="secondary">Type</OuiText>
            <OuiText>{{ getTypeLabel(database.type) }}</OuiText>
          </OuiFlex>
          <OuiFlex justify="between" v-if="database.version">
            <OuiText color="secondary">Version</OuiText>
            <OuiText>{{ database.version }}</OuiText>
          </OuiFlex>
          <OuiFlex justify="between">
            <OuiText color="secondary">Size</OuiText>
            <OuiText>{{ database.size }}</OuiText>
          </OuiFlex>
          <OuiFlex justify="between">
            <OuiText color="secondary">Created</OuiText>
            <OuiText>{{ formatDate(database.createdAt) }}</OuiText>
          </OuiFlex>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { formatBytes, formatDate } from "~/utils/common";

const props = defineProps<{
  database: any;
}>();

function getTypeLabel(type: string): string {
  const types: Record<string, string> = {
    POSTGRESQL: "PostgreSQL",
    MYSQL: "MySQL",
    MONGODB: "MongoDB",
    REDIS: "Redis",
    MARIADB: "MariaDB",
  };
  return types[type] || type;
}

function getStatusLabel(status: string): string {
  const statuses: Record<string, string> = {
    CREATING: "Creating",
    STARTING: "Starting",
    RUNNING: "Running",
    STOPPING: "Stopping",
    STOPPED: "Stopped",
    FAILED: "Failed",
  };
  return statuses[status] || status;
}

function getStatusColor(status: string): string {
  const colors: Record<string, string> = {
    CREATING: "warning",
    STARTING: "info",
    RUNNING: "success",
    STOPPING: "warning",
    STOPPED: "secondary",
    FAILED: "danger",
  };
  return colors[status] || "secondary";
}
</script>

