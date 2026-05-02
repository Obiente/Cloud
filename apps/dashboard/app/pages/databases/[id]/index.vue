<template>
  <OuiContainer size="full" py="sm" class="md:py-6">
    <OuiStack gap="md" class="md:gap-xl">
      <!-- Access Error State -->
      <OuiCard v-if="accessError" variant="outline" class="border-danger/20">
        <OuiCardBody>
          <OuiStack gap="lg" align="center">
            <ErrorAlert
              :error="accessError"
              title="Access Denied"
              :hint="errorHint"
            />
            <OuiButton
              variant="solid"
              color="primary"
              @click="router.push('/databases')"
            >
              Go to Databases
            </OuiButton>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Database Content (only show if no access error) -->
      <template v-else>
        <!-- Loading Skeleton -->
        <template v-if="pending && !databaseData">
          <OuiCard variant="outline" class="border-border-default/50">
            <OuiCardBody class="p-3 md:p-6">
              <OuiStack gap="md" class="md:gap-lg">
                <OuiFlex justify="between" align="start" wrap="wrap" gap="md">
                  <OuiStack gap="sm" class="flex-1 min-w-0">
                    <OuiFlex align="center" gap="sm">
                      <OuiSkeleton width="3rem" height="3rem" variant="rectangle" :rounded="true" class="rounded-lg" />
                      <OuiStack gap="xs" class="flex-1">
                        <OuiSkeleton width="20rem" height="2rem" variant="text" />
                        <OuiSkeleton width="12rem" height="1rem" variant="text" />
                      </OuiStack>
                    </OuiFlex>
                  </OuiStack>
                  <OuiFlex gap="sm">
                    <OuiSkeleton width="6rem" height="2rem" variant="rectangle" rounded />
                    <OuiSkeleton width="6rem" height="2rem" variant="rectangle" rounded />
                  </OuiFlex>
                </OuiFlex>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </template>

        <!-- Header -->
        <Transition name="fade" mode="out-in">
          <OuiCard v-if="!pending && database" variant="outline" class="border-border-default/50">
            <OuiCardBody class="p-3 md:p-6">
              <OuiFlex
                justify="between"
                align="start"
                wrap="wrap"
                gap="md"
                class="md:gap-lg md:items-center"
              >
                <OuiStack gap="sm" class="flex-1 min-w-0 md:gap-md">
                  <OuiFlex align="center" gap="sm" wrap="wrap" class="md:gap-md">
                    <OuiBox
                      p="xs"
                      rounded="lg"
                      bg="accent-primary"
                      class="bg-primary/10 ring-1 ring-primary/20 shrink-0 md:p-sm md:rounded-xl"
                    >
                      <CircleStackIcon
                        class="w-6 h-6 md:w-8 md:h-8 text-primary"
                      />
                    </OuiBox>
                    <OuiStack gap="xs" class="min-w-0 flex-1 md:gap-none">
                      <OuiFlex
                        align="center"
                        justify="between"
                        gap="md"
                        wrap="wrap"
                        class="md:justify-start"
                      >
                        <OuiText
                          as="h1"
                          size="xl"
                          weight="bold"
                          truncate
                          class="md:text-2xl"
                        >
                          {{ database.name }}
                        </OuiText>
                        <OuiBadge :variant="statusMeta.badgeVariant" size="xs">
                          <span
                            class="inline-flex h-1.5 w-1.5 rounded-full mr-1.5"
                            :class="statusMeta.dotClass"
                          />
                          <OuiText
                            as="span"
                            size="xs"
                            weight="semibold"
                            transform="uppercase"
                            >{{ statusMeta.label }}</OuiText
                          >
                        </OuiBadge>
                      </OuiFlex>
                      <OuiText size="xs" color="tertiary" class="md:text-sm">
                        {{ getTypeLabel(database.type) }} • Port {{ database.port || 'N/A' }}
                      </OuiText>
                    </OuiStack>
                  </OuiFlex>
                </OuiStack>
                <OuiFlex
                  gap="xs"
                  wrap="wrap"
                  class="w-full md:w-auto shrink-0 md:gap-sm md:flex-nowrap"
                >
                  <OuiButton
                    variant="ghost"
                    color="secondary"
                    size="sm"
                    @click="refreshDatabase"
                    :loading="isRefreshing"
                    class="gap-1.5 md:gap-2 flex-1 md:flex-initial"
                  >
                    <ArrowPathIcon
                      class="h-4 w-4"
                      :class="{ 'animate-spin': isRefreshing }"
                    />
                    <OuiText
                      as="span"
                      size="xs"
                      weight="medium"
                      class="hidden sm:inline"
                      >Refresh</OuiText
                    >
                  </OuiButton>
                  <OuiButton
                    v-if="canStart"
                    variant="outline"
                    color="success"
                    size="sm"
                    @click="handleStart"
                    :loading="isStarting"
                    :disabled="isOperationActive"
                    class="gap-1.5 md:gap-2 flex-1 md:flex-initial"
                  >
                    <ArrowPathIcon v-if="activeOperation?.kind === 'start'" class="h-4 w-4 animate-spin" />
                    <PlayIcon v-else class="h-4 w-4" />
                    <OuiText
                      as="span"
                      size="xs"
                      weight="medium"
                      class="hidden sm:inline"
                      >{{ activeOperation?.kind === "start" ? "Starting" : "Start" }}</OuiText
                    >
                  </OuiButton>
                  <OuiButton
                    v-if="canSleep"
                    variant="outline"
                    color="secondary"
                    size="sm"
                    @click="handleSleep"
                    :loading="isSleeping"
                    :disabled="isOperationActive"
                    class="gap-1.5 md:gap-2 flex-1 md:flex-initial"
                    title="Put to sleep (auto-wakes on connection)"
                  >
                    <MoonIcon class="h-4 w-4" />
                    <OuiText
                      as="span"
                      size="xs"
                      weight="medium"
                      class="hidden sm:inline"
                      >{{ activeOperation?.kind === "sleep" ? "Sleeping" : "Sleep" }}</OuiText
                    >
                  </OuiButton>
                  <OuiButton
                    v-if="canStop"
                    variant="outline"
                    color="warning"
                    size="sm"
                    @click="handleStop"
                    :loading="isStopping"
                    :disabled="isOperationActive"
                    class="gap-1.5 md:gap-2 flex-1 md:flex-initial"
                    title="Fully stop (no auto-wake)"
                  >
                    <StopIcon class="h-4 w-4" />
                    <OuiText
                      as="span"
                      size="xs"
                      weight="medium"
                      class="hidden sm:inline"
                      >{{ activeOperation?.kind === "stop" ? "Stopping" : "Stop" }}</OuiText
                    >
                  </OuiButton>
                  <OuiButton
                    v-if="canRestart"
                    variant="outline"
                    color="primary"
                    size="sm"
                    @click="handleRestart"
                    :loading="isRestarting"
                    :disabled="isOperationActive"
                    class="gap-1.5 md:gap-2 flex-1 md:flex-initial"
                  >
                    <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': activeOperation?.kind === 'restart' }" />
                    <OuiText
                      as="span"
                      size="xs"
                      weight="medium"
                      class="hidden sm:inline"
                      >{{ activeOperation?.kind === "restart" ? "Restarting" : "Restart" }}</OuiText
                    >
                  </OuiButton>
                  <OuiButton
                    variant="solid"
                    color="danger"
                    size="sm"
                    @click="handleDelete"
                    :loading="isDeleting"
                    class="gap-1.5 md:gap-2 flex-1 md:flex-initial"
                  >
                    <TrashIcon class="h-4 w-4" />
                    <OuiText
                      as="span"
                      size="xs"
                      weight="medium"
                      class="hidden sm:inline"
                      >Delete</OuiText
                    >
                  </OuiButton>
                </OuiFlex>
              </OuiFlex>
            </OuiCardBody>
          </OuiCard>
        </Transition>

        <OuiCard
          v-if="activeOperation || operationError"
          variant="outline"
          :class="operationError ? 'border-danger/30 bg-danger/5' : 'border-warning/30 bg-warning/5'"
        >
          <OuiCardBody>
            <OuiFlex align="start" gap="sm">
              <ArrowPathIcon
                v-if="activeOperation"
                class="mt-0.5 h-4 w-4 shrink-0 animate-spin text-warning"
              />
              <ExclamationTriangleIcon
                v-else
                class="mt-0.5 h-4 w-4 shrink-0 text-danger"
              />
              <OuiStack gap="xs" class="min-w-0">
                <OuiText size="sm" weight="medium">
                  {{ activeOperation?.label || "Command failed" }}
                </OuiText>
                <OuiText size="sm" color="tertiary">
                  {{ operationError || activeOperation?.description }}
                </OuiText>
              </OuiStack>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>

        <!-- Tabbed Content -->
        <OuiStack gap="sm" class="md:gap-md" v-if="!pending && database">
          <OuiTabs v-model="activeTab" :tabs="tabs" />
          <OuiCard variant="default">
            <OuiTabs v-model="activeTab" :tabs="tabs" :content-only="true">
              <template #overview>
                <DatabaseOverview :database="database" />
              </template>
              <template #connection>
                <DatabaseConnection :database-id="database.id" />
              </template>
              <template #browser>
                <DatabaseBrowser :database-id="database.id" :database-type="String(database.type)" />
              </template>
              <template #query>
                <DatabaseQuery :database-id="database.id" :database-type="String(database.type)" />
              </template>
              <template #backups>
                <DatabaseBackups :database-id="database.id" />
              </template>
              <template #settings>
                <DatabaseSettings :database="database" @save="handleSettingsSave" />
              </template>
              <template #audit-logs>
                <AuditLogs
                  :organization-id="orgId"
                  resource-type="database"
                  :resource-id="id"
                />
              </template>
            </OuiTabs>
          </OuiCard>
        </OuiStack>

        <OuiCard v-else-if="!pending" variant="outline" class="border-warning/20">
          <OuiCardBody>
            <OuiStack gap="lg" align="center">
              <ErrorAlert
                :error="databaseLoadError"
                title="Database unavailable"
                :hint="databaseLoadHint"
              />
              <OuiFlex gap="sm" wrap="wrap" justify="center">
                <OuiButton
                  variant="solid"
                  color="primary"
                  :loading="isRefreshing"
                  @click="refreshDatabase"
                >
                  Retry
                </OuiButton>
                <OuiButton
                  variant="ghost"
                  color="secondary"
                  @click="router.push('/databases')"
                >
                  Go to Databases
                </OuiButton>
              </OuiFlex>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>
      </template>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
import {
  CircleStackIcon,
  ArrowPathIcon,
  TrashIcon,
  PlayIcon,
  StopIcon,
  MoonIcon,
  ExclamationTriangleIcon,
} from "@heroicons/vue/24/outline";
import { computed, ref, watch, onUnmounted } from "vue";
import { useRouter, useRoute } from "vue-router";
import { DatabaseService, DatabaseStatus, DatabaseType } from "@obiente/proto";
import { ConnectError, Code } from "@connectrpc/connect";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";
import { useClientFetch } from "~/composables/useClientFetch";
import { useDialog } from "~/composables/useDialog";
import { useToast } from "~/composables/useToast";
import { useTabQuery } from "~/composables/useTabQuery";
import { useResourceOperation } from "~/composables/useResourceOperation";
import ErrorAlert from "~/components/ErrorAlert.vue";
import DatabaseOverview from "~/components/database/DatabaseOverview.vue";
import DatabaseConnection from "~/components/database/DatabaseConnection.vue";
import DatabaseBrowser from "~/components/database/DatabaseBrowser.vue";
import DatabaseQuery from "~/components/database/DatabaseQuery.vue";
import DatabaseBackups from "~/components/database/DatabaseBackups.vue";
import DatabaseSettings from "~/components/database/DatabaseSettings.vue";
import AuditLogs from "~/components/audit/AuditLogs.vue";

definePageMeta({
  layout: "default",
  middleware: "auth",
});

const router = useRouter();
const route = useRoute();
const id = computed(() => String(route.params.id));
const organizationId = useOrganizationId();
const { showConfirm, showAlert } = useDialog();
const { toast } = useToast();
const {
  activeOperation,
  operationError,
  isOperationActive,
  beginOperation,
  finishOperation,
  failOperation,
  getErrorMessage,
} = useResourceOperation();
const dbClient = useConnectClient(DatabaseService);

const orgId = computed(() => organizationId.value || "");
const isRefreshing = ref(false);
const isRestarting = ref(false);
const isDeleting = ref(false);
const isStarting = ref(false);
const isStopping = ref(false);
const isSleeping = ref(false);
const accessError = ref<Error | null>(null);
let operationPollingInterval: ReturnType<typeof setInterval> | null = null;

// Fetch database data
const {
  data: databaseData,
  pending,
  refresh: refreshDatabaseBase,
  error: fetchError,
} = useClientFetch(
  `database-${id.value}`,
  async () => {
    if (!orgId.value) return null;
    try {
      const response = await dbClient.getDatabase({
        organizationId: orgId.value,
        databaseId: id.value,
      });
      accessError.value = null;
      return response.database ?? null;
    } catch (err) {
      if (err instanceof ConnectError) {
        if (err.code === Code.PermissionDenied || err.code === Code.NotFound) {
          accessError.value = err;
          return null;
        }
      }
      throw err;
    }
  },
  { watch: [orgId, id] }
);

// Custom refresh function
const refreshDatabase = async () => {
  isRefreshing.value = true;
  try {
    await refreshDatabaseBase();
  } finally {
    isRefreshing.value = false;
  }
};

const stopOperationPolling = () => {
  if (operationPollingInterval) {
    clearInterval(operationPollingInterval);
    operationPollingInterval = null;
  }
};

const hasReachedOperationTarget = () => {
  const operation = activeOperation.value;
  if (!operation || !database.value) return false;
  if (operation.kind === "start") return database.value.status === DatabaseStatus.RUNNING;
  if (operation.kind === "stop") return database.value.status === DatabaseStatus.STOPPED;
  if (operation.kind === "sleep") return database.value.status === DatabaseStatus.SLEEPING;
  if (operation.kind === "restart") return database.value.status === DatabaseStatus.RUNNING;
  return false;
};

const startOperationPolling = () => {
  stopOperationPolling();
  operationPollingInterval = setInterval(async () => {
    if (!activeOperation.value) {
      stopOperationPolling();
      return;
    }
    await refreshDatabase();
    if (database.value?.status === DatabaseStatus.FAILED) {
      failOperation(`${activeOperation.value.label} failed. Check database logs or events for backend details.`);
      stopOperationPolling();
      return;
    }
    if (hasReachedOperationTarget()) {
      finishOperation();
      stopOperationPolling();
    }
  }, 3_000);
};

onUnmounted(stopOperationPolling);

// Watch for fetch errors
watch(fetchError, (err) => {
  if (err instanceof ConnectError) {
    if (err.code === Code.PermissionDenied || err.code === Code.NotFound) {
      accessError.value = err;
    }
  }
});

const database = computed(() => databaseData.value);
const databaseLoadError = computed(
  () => fetchError.value || new Error("The database could not be loaded.")
);
const databaseLoadHint = computed(() => {
  if (fetchError.value) {
    return "We couldn't load this database right now. Check your connection or try again.";
  }
  return "This database may not exist yet, or it may still be provisioning.";
});

const errorHint = computed(() => {
  if (!accessError.value || !(accessError.value instanceof ConnectError)) {
    return "You don't have permission to view this database, or it doesn't exist.";
  }
  if (accessError.value.code === Code.PermissionDenied) {
    return "You don't have permission to view this database. Please contact your organization administrator.";
  }
  if (accessError.value.code === Code.NotFound) {
    return "This database doesn't exist or may have been deleted.";
  }
  return "You don't have permission to view this database, or it doesn't exist.";
});

const tabs = computed(() => [
  { id: "overview", label: "Overview" },
  { id: "connection", label: "Connection" },
  { id: "browser", label: "Browser" },
  { id: "query", label: "Query" },
  { id: "backups", label: "Backups" },
  { id: "settings", label: "Settings" },
  { id: "audit-logs", label: "Audit Logs" },
]);

const activeTab = useTabQuery(tabs);

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

const statusMeta = computed(() => {
  if (!database.value) {
    return {
      label: "Unknown",
      badgeVariant: "secondary" as const,
      dotClass: "bg-secondary",
    };
  }

  const status = database.value.status;
  const statusMap: Record<number, { label: string; variant: "success" | "danger" | "warning" | "primary" | "secondary"; dotClass: string }> = {
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

  const meta = statusMap[status] || { label: "Unknown", variant: "secondary" as const, dotClass: "bg-secondary" };
  return {
    label: meta.label,
    badgeVariant: meta.variant,
    dotClass: meta.dotClass,
  };
});

const canStart = computed(() => {
  if (!database.value) return false;
  return [DatabaseStatus.STOPPED, DatabaseStatus.SLEEPING].includes(database.value.status);
});

const canStop = computed(() => {
  if (!database.value) return false;
  return database.value.status === DatabaseStatus.RUNNING;
});

const canSleep = computed(() => {
  if (!database.value) return false;
  return database.value.status === DatabaseStatus.RUNNING;
});

const canRestart = computed(() => {
  if (!database.value) return false;
  return database.value.status === DatabaseStatus.RUNNING;
});

async function handleStart() {
  if (!database.value) return;
  isStarting.value = true;
  beginOperation({
    kind: "start",
    label: "Starting database",
    description: "The command was sent. Waiting for the backend to report the database is running.",
    failureMessage: "Failed to start database",
  });
  try {
    await dbClient.startDatabase({
      databaseId: id.value,
      organizationId: orgId.value,
    });
    toast.success("Database start initiated");
    await refreshDatabase();
    if (hasReachedOperationTarget()) {
      finishOperation();
    } else {
      startOperationPolling();
    }
  } catch (error) {
    console.error("Failed to start database:", error);
    failOperation(getErrorMessage(error, "An unknown error occurred"));
    await showAlert({
      title: "Failed to start database",
      message: error instanceof Error ? error.message : "An unknown error occurred",
    });
  } finally {
    isStarting.value = false;
  }
}

async function handleStop() {
  if (!database.value) return;

  const confirmed = await showConfirm({
    title: "Stop Database",
    message: `Are you sure you want to stop ${database.value.name}? The database will not accept connections until manually started.`,
    confirmLabel: "Stop",
    cancelLabel: "Cancel",
  });

  if (!confirmed) return;

  isStopping.value = true;
  beginOperation({
    kind: "stop",
    label: "Stopping database",
    description: "The command was sent. Waiting for the backend to confirm the database stopped.",
    failureMessage: "Failed to stop database",
  });
  try {
    await dbClient.stopDatabase({
      databaseId: id.value,
      organizationId: orgId.value,
    });
    toast.success("Database stop initiated");
    await refreshDatabase();
    if (hasReachedOperationTarget()) {
      finishOperation();
    } else {
      startOperationPolling();
    }
  } catch (error) {
    console.error("Failed to stop database:", error);
    failOperation(getErrorMessage(error, "An unknown error occurred"));
    await showAlert({
      title: "Failed to stop database",
      message: error instanceof Error ? error.message : "An unknown error occurred",
    });
  } finally {
    isStopping.value = false;
  }
}

async function handleSleep() {
  if (!database.value) return;

  const confirmed = await showConfirm({
    title: "Sleep Database",
    message: `Put ${database.value.name} to sleep? It will automatically wake up when a connection is made.`,
    confirmLabel: "Sleep",
    cancelLabel: "Cancel",
  });

  if (!confirmed) return;

  isSleeping.value = true;
  beginOperation({
    kind: "sleep",
    label: "Putting database to sleep",
    description: "The command was sent. Waiting for the backend to report the database is sleeping.",
    failureMessage: "Failed to sleep database",
  });
  try {
    await dbClient.sleepDatabase({
      databaseId: id.value,
      organizationId: orgId.value,
    });
    toast.success("Database is going to sleep");
    await refreshDatabase();
    if (hasReachedOperationTarget()) {
      finishOperation();
    } else {
      startOperationPolling();
    }
  } catch (error) {
    console.error("Failed to sleep database:", error);
    failOperation(getErrorMessage(error, "An unknown error occurred"));
    await showAlert({
      title: "Failed to sleep database",
      message: error instanceof Error ? error.message : "An unknown error occurred",
    });
  } finally {
    isSleeping.value = false;
  }
}

async function handleRestart() {
  if (!database.value) return;

  const confirmed = await showConfirm({
    title: "Restart Database",
    message: `Are you sure you want to restart ${database.value.name}? This will cause a brief downtime.`,
    confirmLabel: "Restart",
    cancelLabel: "Cancel",
  });

  if (!confirmed) return;

  isRestarting.value = true;
  beginOperation({
    kind: "restart",
    label: "Restarting database",
    description: "The command was sent. Waiting for the backend to report the database is running again.",
    failureMessage: "Failed to restart database",
  });
  try {
    await dbClient.restartDatabase({
      databaseId: id.value,
      organizationId: orgId.value,
    });
    toast.success("Database restart initiated");
    await refreshDatabase();
    if (hasReachedOperationTarget()) {
      finishOperation();
    } else {
      startOperationPolling();
    }
  } catch (error) {
    console.error("Failed to restart database:", error);
    failOperation(getErrorMessage(error, "An unknown error occurred"));
    await showAlert({
      title: "Failed to restart database",
      message: error instanceof Error ? error.message : "An unknown error occurred",
    });
  } finally {
    isRestarting.value = false;
  }
}

async function handleDelete() {
  if (!database.value) return;

  const confirmed = await showConfirm({
    title: "Delete Database",
    message: `Are you sure you want to delete ${database.value.name}? This action cannot be undone and all data will be lost.`,
    confirmLabel: "Delete",
    cancelLabel: "Cancel",
    variant: "danger",
  });

  if (!confirmed) return;

  isDeleting.value = true;
  try {
    await dbClient.deleteDatabase({
      databaseId: id.value,
      organizationId: orgId.value,
    });
    toast.success("Database deleted successfully");
    router.push("/databases");
  } catch (error) {
    console.error("Failed to delete database:", error);
    await showAlert({
      title: "Failed to delete database",
      message: error instanceof Error ? error.message : "An unknown error occurred",
    });
  } finally {
    isDeleting.value = false;
  }
}

async function handleSettingsSave() {
  await refreshDatabase();
  toast.success("Database settings updated");
}
</script>
