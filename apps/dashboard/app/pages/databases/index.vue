<template>
  <OuiContainer size="full">
    <OuiStack gap="xl">
      <OuiFlex justify="between" align="start" wrap="wrap" gap="lg">
        <OuiStack gap="sm" class="max-w-xl">
          <OuiFlex align="center" gap="md">
            <OuiBox
              p="sm"
              rounded="xl"
              bg="accent-primary"
              class="bg-primary/10 ring-1 ring-primary/20"
            >
              <CircleStackIcon class="w-6 h-6 text-primary" />
            </OuiBox>
            <OuiText as="h1" size="3xl" weight="bold"> Databases </OuiText>
          </OuiFlex>
          <OuiText color="secondary" size="md">
            Deploy and manage your databases with automated backups, scaling, and monitoring.
            PostgreSQL, MySQL, MongoDB, and more.
          </OuiText>
        </OuiStack>

        <OuiButton
          color="primary"
          class="gap-2 shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30 transition-all"
          @click="showCreateDialog = true"
        >
          <PlusIcon class="h-4 w-4" />
          <OuiText as="span" size="sm" weight="medium">New Database</OuiText>
        </OuiButton>
      </OuiFlex>

      <!-- Show error alert if there was a problem loading databases -->
      <ErrorAlert
        v-if="listError"
        :error="listError"
        title="Failed to load databases"
        hint="Please try refreshing the page. If the problem persists, contact support."
      />

      <OuiCard
        variant="default"
        class="backdrop-blur-sm border border-border-muted/60"
      >
        <OuiCardBody>
          <OuiGrid cols="1" cols-md="3" gap="md">
            <OuiInput
              v-model="searchQuery"
              placeholder="Search by name or type..."
              clearable
            >
              <template #prefix>
                <MagnifyingGlassIcon class="h-4 w-4 text-secondary" />
              </template>
            </OuiInput>

            <OuiSelect
              v-model="statusFilter"
              :items="statusFilterOptions"
              placeholder="All Status"
            />

            <OuiSelect
              v-model="typeFilter"
              :items="typeFilterOptions"
              placeholder="All Types"
            />
          </OuiGrid>
        </OuiCardBody>
      </OuiCard>

      <!-- Loading State with Skeleton Cards -->
      <OuiGrid v-if="pending && !databasesData" cols="1" cols-md="2" cols-lg="3" gap="lg">
        <DatabaseCard
          v-for="i in 6"
          :key="i"
          :loading="true"
        />
      </OuiGrid>

      <!-- Empty State -->
      <OuiStack
        v-else-if="filteredDatabases.length === 0"
        align="center"
        gap="lg"
        class="text-center py-20"
      >
        <OuiBox
          class="inline-flex items-center justify-center w-20 h-20 rounded-xl bg-surface-muted/50 ring-1 ring-border-muted"
        >
          <CircleStackIcon class="h-10 w-10 text-secondary" />
        </OuiBox>
        <OuiStack align="center" gap="sm">
          <OuiText as="h3" size="xl" weight="semibold" color="primary">
            No databases found
          </OuiText>
          <OuiBox class="max-w-md">
            <OuiText color="secondary">
              {{
                searchQuery || statusFilter || typeFilter
                  ? "Try adjusting your filters to see more results."
                  : "Get started by creating your first database instance."
              }}
            </OuiText>
          </OuiBox>
          <OuiButton
            v-if="!searchQuery && !statusFilter && !typeFilter"
            color="primary"
            @click="showCreateDialog = true"
          >
            <PlusIcon class="h-4 w-4" />
            Create Database
          </OuiButton>
        </OuiStack>
      </OuiStack>

      <!-- Database Grid -->
      <OuiGrid v-else cols="1" cols-md="2" cols-lg="3" gap="lg">
        <DatabaseCard
          v-for="database in filteredDatabases"
          :key="database.id"
          :database="database"
          @click="navigateToDatabase(database.id)"
        />
      </OuiGrid>
    </OuiStack>

    <!-- Create Database Dialog -->
    <CreateDatabaseDialog
      v-model="showCreateDialog"
      @created="handleDatabaseCreated"
    />
  </OuiContainer>
</template>

<script setup lang="ts">
import { CircleStackIcon, PlusIcon, MagnifyingGlassIcon } from "@heroicons/vue/24/outline";
import { computed, ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import { DatabaseService, DatabaseStatus, DatabaseType } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";
import { useClientFetch } from "~/composables/useClientFetch";
import ErrorAlert from "~/components/ErrorAlert.vue";
import DatabaseCard from "~/components/database/DatabaseCard.vue";
import CreateDatabaseDialog from "~/components/database/CreateDatabaseDialog.vue";

definePageMeta({
  layout: "default",
});

const router = useRouter();
const organizationId = useOrganizationId();
const dbClient = useConnectClient(DatabaseService);

const searchQuery = ref("");
const statusFilter = ref<string>("");
const typeFilter = ref<string>("");
const showCreateDialog = ref(false);

const statusFilterOptions = [
  { label: "All Status", value: "" },
  { label: "Running", value: DatabaseStatus.RUNNING.toString() },
  { label: "Stopped", value: DatabaseStatus.STOPPED.toString() },
  { label: "Creating", value: DatabaseStatus.CREATING.toString() },
  { label: "Failed", value: DatabaseStatus.FAILED.toString() },
];

const typeFilterOptions = [
  { label: "All Types", value: "" },
  { label: "PostgreSQL", value: DatabaseType.POSTGRESQL.toString() },
  { label: "MySQL", value: DatabaseType.MYSQL.toString() },
  { label: "MongoDB", value: DatabaseType.MONGODB.toString() },
  { label: "Redis", value: DatabaseType.REDIS.toString() },
  { label: "MariaDB", value: DatabaseType.MARIADB.toString() },
];

// Fetch databases
const {
  data: databasesData,
  pending,
  error: listError,
  refresh,
} = useClientFetch(
  () => `databases-list-${organizationId.value}`,
  async () => {
    if (!organizationId.value) return { databases: [], pagination: null };
    const res = await dbClient.listDatabases({
      organizationId: organizationId.value,
      page: 1,
      perPage: 100,
      status: statusFilter.value ? parseInt(statusFilter.value) as DatabaseStatus : undefined,
      type: typeFilter.value ? parseInt(typeFilter.value) as DatabaseType : undefined,
    });
    return { databases: res.databases || [], pagination: res.pagination };
  },
  { watch: [statusFilter, typeFilter] }
);

const databases = computed(() => (databasesData.value as any)?.databases || []);

// Helper functions for display
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

const filteredDatabases = computed(() => {
  let result = databases.value;

  // Search filter
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    result = result.filter(
      (db: any) =>
        db.name?.toLowerCase().includes(query) ||
        getTypeLabel(db.type)?.toLowerCase().includes(query)
    );
  }

  // Status filter
  if (statusFilter.value) {
    const statusValue = parseInt(statusFilter.value);
    result = result.filter((db: any) => db.status === statusValue);
  }

  // Type filter
  if (typeFilter.value) {
    const typeValue = parseInt(typeFilter.value);
    result = result.filter((db: any) => db.type === typeValue);
  }

  return result;
});

function navigateToDatabase(id: string) {
  router.push(`/databases/${id}`);
}

function handleDatabaseCreated() {
  refresh();
  showCreateDialog.value = false;
}
</script>
