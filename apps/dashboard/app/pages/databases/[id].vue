<template>
  <OuiContainer size="full">
    <OuiStack gap="xl">
      <!-- Header -->
      <OuiFlex justify="between" align="start" wrap="wrap" gap="lg">
        <OuiStack gap="sm">
          <OuiFlex align="center" gap="md">
            <OuiButton variant="ghost" size="sm" @click="$router.push('/databases')">
              <ArrowLeftIcon class="h-4 w-4" />
            </OuiButton>
            <OuiText as="h1" size="3xl" weight="bold">
              {{ database?.name || "Loading..." }}
            </OuiText>
            <OuiBadge v-if="database" :color="getStatusColor(database.status)">
              {{ getStatusLabel(database.status) }}
            </OuiBadge>
          </OuiFlex>
          <OuiText v-if="database" color="secondary" size="md">
            {{ getTypeLabel(database.type) }}
            <span v-if="database.version"> v{{ database.version }}</span>
          </OuiText>
        </OuiStack>

        <OuiFlex gap="sm">
          <OuiButton
            v-if="database && database.status === DatabaseStatus.RUNNING"
            variant="outline"
            @click="handleStop"
          >
            Stop
          </OuiButton>
          <OuiButton
            v-if="database && database.status === DatabaseStatus.STOPPED"
            variant="outline"
            @click="handleStart"
          >
            Start
          </OuiButton>
          <OuiButton
            v-if="database && database.status === DatabaseStatus.RUNNING"
            variant="outline"
            @click="handleRestart"
          >
            Restart
          </OuiButton>
          <OuiButton variant="outline" color="danger" @click="handleDelete">
            Delete
          </OuiButton>
        </OuiFlex>
      </OuiFlex>

      <!-- Error Alert -->
      <ErrorAlert
        v-if="error"
        :error="error"
        title="Failed to load database"
      />

      <!-- Loading State -->
      <OuiStack v-if="pending && !database" align="center" gap="md" class="py-20">
        <OuiSpinner size="lg" />
        <OuiText color="secondary">Loading database...</OuiText>
      </OuiStack>

      <!-- Database Content -->
      <OuiTabs 
        v-if="database" 
        v-model="activeTab"
        :tabs="[
          { id: 'overview', label: 'Overview' },
          { id: 'browser', label: 'Browser' },
          { id: 'query', label: 'Query' },
          { id: 'backups', label: 'Backups' },
          { id: 'connection', label: 'Connection' },
        ]"
      >
        <template #overview>
          <DatabaseOverview :database="database" />
        </template>
        <template #browser>
          <DatabaseBrowser :database-id="database.id" :database-type="database.type" />
        </template>
        <template #query>
          <DatabaseQuery :database-id="database.id" :database-type="database.type" />
        </template>
        <template #backups>
          <DatabaseBackups :database-id="database.id" />
        </template>
        <template #connection>
          <DatabaseConnection :database-id="database.id" />
        </template>
      </OuiTabs>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
import { ArrowLeftIcon } from "@heroicons/vue/24/outline";
import { ref, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import { DatabaseService, DatabaseType, DatabaseStatus } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";
import { useClientFetch } from "~/composables/useClientFetch";
import { useToast } from "~/composables/useToast";
import ErrorAlert from "~/components/ErrorAlert.vue";
import DatabaseOverview from "~/components/database/DatabaseOverview.vue";
import DatabaseBrowser from "~/components/database/DatabaseBrowser.vue";
import DatabaseQuery from "~/components/database/DatabaseQuery.vue";
import DatabaseBackups from "~/components/database/DatabaseBackups.vue";
import DatabaseConnection from "~/components/database/DatabaseConnection.vue";

definePageMeta({
  layout: "default",
});

const route = useRoute();
const router = useRouter();
const organizationId = useOrganizationId();
const { toast } = useToast();
const dbClient = useConnectClient(DatabaseService);

const activeTab = ref("overview");
const databaseId = computed(() => route.params.id as string);

// Fetch database
const {
  data: databaseData,
  pending,
  error,
  refresh,
} = useClientFetch(
      () => `database-${databaseId.value}`,
      async () => {
        if (!organizationId.value || !databaseId.value) return { database: null };
        const res = await dbClient.getDatabase({
          organizationId: organizationId.value,
          databaseId: databaseId.value,
        });
        return { database: res.database };
      }
);

const database = computed(() => (databaseData.value as any)?.database);

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
    [DatabaseStatus.FAILED]: "Failed",
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
    [DatabaseStatus.FAILED]: "danger",
  };
  return colors[statusValue] || "secondary";
}

async function handleStart() {
  try {
    if (!organizationId.value) return;
    await dbClient.startDatabase({
      organizationId: organizationId.value,
      databaseId: databaseId.value,
    });
    toast.success("Database started");
    refresh();
  } catch (err: any) {
    toast.error("Failed to start database", err.message);
  }
}

async function handleStop() {
  try {
    if (!organizationId.value) return;
    await dbClient.stopDatabase({
      organizationId: organizationId.value,
      databaseId: databaseId.value,
    });
    toast.success("Database stopped");
    refresh();
  } catch (err: any) {
    toast.error("Failed to stop database", err.message);
  }
}

async function handleRestart() {
  try {
    if (!organizationId.value) return;
    await dbClient.restartDatabase({
      organizationId: organizationId.value,
      databaseId: databaseId.value,
    });
    toast.success("Database restarted");
    refresh();
  } catch (err: any) {
    toast.error("Failed to restart database", err.message);
  }
}

async function handleDelete() {
  if (!confirm("Are you sure you want to delete this database? This action cannot be undone.")) {
    return;
  }

  try {
    if (!organizationId.value) return;
    await dbClient.deleteDatabase({
      organizationId: organizationId.value,
      databaseId: databaseId.value,
      force: false,
    });
    toast.success("Database deleted");
    router.push("/databases");
  } catch (err: any) {
    toast.error("Failed to delete database", err.message);
  }
}
</script>

