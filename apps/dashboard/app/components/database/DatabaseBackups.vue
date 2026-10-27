<template>
  <OuiStack gap="lg">
    <OuiCard v-if="supportsSqlDump" variant="outline">
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiFlex justify="between" align="start" gap="md" wrap="wrap">
            <OuiStack gap="xs">
              <OuiText size="sm" weight="semibold">SQL Dumps</OuiText>
              <OuiText size="xs" color="tertiary">
                Export or import portable .sql dumps for PostgreSQL, MySQL, and
                MariaDB.
              </OuiText>
            </OuiStack>
            <OuiFlex gap="sm" wrap="wrap">
              <OuiButton
                variant="outline"
                size="sm"
                :loading="exportingDump"
                @click="handleExportDump"
              >
                <ArrowDownTrayIcon class="h-3.5 w-3.5" />
                Export SQL
              </OuiButton>
              <OuiButton
                color="primary"
                size="sm"
                :loading="importingDump"
                @click="dumpFileInput?.click()"
              >
                <ArrowUpTrayIcon class="h-3.5 w-3.5" />
                Import SQL
              </OuiButton>
              <input
                ref="dumpFileInput"
                type="file"
                accept=".sql,.dump,text/sql,application/sql,text/plain"
                class="hidden"
                @change="handleImportDump"
              />
            </OuiFlex>
          </OuiFlex>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard variant="outline">
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiFlex justify="between" align="start" gap="md" wrap="wrap">
            <OuiStack gap="xs">
              <OuiText as="h2" size="lg" weight="semibold"
                >Managed backups</OuiText
              >
              <OuiText v-if="available" size="sm" color="secondary"
                >Create an on-demand backup of the primary database. The latest
                {{ retainedCount }} completed backups are kept.</OuiText
              >
            </OuiStack>
            <OuiFlex gap="sm" wrap="wrap">
              <OuiButton
                variant="outline"
                size="sm"
                :loading="loading"
                @click="loadBackups"
                >Refresh backups</OuiButton
              >
              <OuiButton
                v-if="available"
                size="sm"
                :disabled="operationPending || !!action"
                :loading="action === 'create'"
                @click="createBackup"
                >Create backup</OuiButton
              >
            </OuiFlex>
          </OuiFlex>
          <OuiAlert
            v-if="loadError"
            variant="danger"
            title="Could not refresh backups"
            >{{ loadError }}</OuiAlert
          >
          <OuiAlert
            v-else-if="loaded && !available"
            variant="warning"
            title="Managed backups are unavailable"
            >{{
              unavailableReason ||
              "Contact your administrator to enable the shared backup volume."
            }}</OuiAlert
          >
          <OuiText v-if="!loaded" color="secondary">Loading backups…</OuiText>
          <OuiText
            v-else-if="!backups.length && available && !loadError"
            color="secondary"
            >No backups yet.</OuiText
          >
          <OuiCard v-for="backup in backups" :key="backup.id" variant="outline">
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex justify="between" align="start" gap="md" wrap="wrap">
                  <OuiStack gap="xs" class="min-w-0">
                    <OuiText weight="semibold" class="break-words">{{
                      backup.name
                    }}</OuiText>
                    <OuiText size="sm" color="secondary"
                      >{{ formatDate(backup.createdAt) }} ·
                      {{ backupStatus(backup)
                      }}<template
                        v-if="
                          backup.status ===
                          DatabaseBackupStatus.BACKUP_COMPLETED
                        "
                      >
                        · {{ formatBytes(backup.sizeBytes) }}</template
                      ></OuiText
                    >
                    <OuiText
                      v-if="backup.databaseName"
                      size="sm"
                      color="secondary"
                      >Database: {{ backup.databaseName }}</OuiText
                    >
                  </OuiStack>
                  <OuiFlex gap="sm" wrap="wrap">
                    <OuiButton
                      v-if="
                        backup.status === DatabaseBackupStatus.BACKUP_COMPLETED
                      "
                      variant="outline"
                      size="sm"
                      :disabled="!available || operationPending || !!action"
                      :aria-label="`Restore backup ${backup.name}`"
                      @click="selectRestore(backup)"
                      >Restore</OuiButton
                    >
                    <OuiButton
                      variant="ghost"
                      color="danger"
                      size="sm"
                      :disabled="!available || operationPending || !!action"
                      :aria-label="`Delete backup ${backup.name}`"
                      @click="deleteBackup(backup)"
                      >Delete</OuiButton
                    >
                  </OuiFlex>
                </OuiFlex>
                <OuiAlert v-if="backup.errorMessage" variant="danger">{{
                  backup.errorMessage
                }}</OuiAlert>
                <OuiAlert
                  v-if="backup.restoreStatus"
                  :variant="
                    backup.restoreStatus === 'failed'
                      ? 'danger'
                      : backup.restoreStatus === 'completed'
                      ? 'success'
                      : 'info'
                  "
                >
                  <template v-if="backup.restoreStatus === 'completed'"
                    >Restored to {{ backup.restoreTarget }}. Use this database
                    name with your existing connection credentials.</template
                  >
                  <template v-else-if="backup.restoreStatus === 'failed'">{{
                    backup.restoreError ||
                    "Restore failed. Inspect the target database before retrying."
                  }}</template>
                  <template v-else
                    >Restore
                    {{
                      backup.restoreStatus === "queued"
                        ? "queued"
                        : "in progress"
                    }}: {{ backup.restoreTarget }}</template
                  >
                </OuiAlert>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
          <OuiFlex v-if="total > 50" align="center" justify="between" gap="sm">
            <OuiButton
              variant="outline"
              size="sm"
              :disabled="page <= 1 || loading"
              @click="page--"
              >Previous backups</OuiButton
            >
            <OuiText size="sm"
              >Page {{ page }} of {{ Math.ceil(total / 50) }}</OuiText
            >
            <OuiButton
              variant="outline"
              size="sm"
              :disabled="page * 50 >= total || loading"
              @click="page++"
              >Next backups</OuiButton
            >
          </OuiFlex>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
    <OuiDialog
      v-model:open="restoreOpen"
      title="Restore backup"
      description="Create a new database from this backup. Existing databases are kept intact."
    >
      <form id="restore-backup-form" @submit.prevent="restoreBackup">
        <OuiInput
          v-model="restoreTarget"
          label="New database name"
          helper-text="Start with a letter. Use letters, digits and underscores, up to 63 characters."
          required
          pattern="[A-Za-z][A-Za-z0-9_]{0,62}"
        />
      </form>
      <template #footer>
        <OuiButton variant="outline" @click="restoreOpen = false"
          >Cancel</OuiButton
        >
        <OuiButton
          type="submit"
          form="restore-backup-form"
          :loading="action === 'restore'"
          :disabled="!validRestoreTarget || !!action"
          >Restore to new database</OuiButton
        >
      </template>
    </OuiDialog>
  </OuiStack>
</template>

<script setup lang="ts">
import { ArrowDownTrayIcon, ArrowUpTrayIcon } from "@heroicons/vue/24/outline";
import { ref, computed, watch, onMounted, onUnmounted } from "vue";
import {
  DatabaseService,
  DatabaseDumpFormat,
  DatabaseType,
  DatabaseBackupStatus,
  type DatabaseBackup,
} from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";
import { useToast } from "~/composables/useToast";
import { useAutoRefresh } from "~/composables/useAutoRefresh";
import { formatDate, formatBytes } from "~/utils/common";

const props = defineProps<{
  databaseId: string;
  databaseType: DatabaseType;
}>();

const supportsSqlDump = computed(() =>
  [DatabaseType.POSTGRESQL, DatabaseType.MYSQL, DatabaseType.MARIADB].includes(
    props.databaseType
  )
);
const organizationId = useOrganizationId();
const { toast } = useToast();
const dbClient = useConnectClient(DatabaseService);

const exportingDump = ref(false);
const importingDump = ref(false);
const dumpFileInput = ref<HTMLInputElement | null>(null);

const backups = ref<DatabaseBackup[]>([]);
const available = ref(false);
const loaded = ref(false);
const loading = ref(false);
const loadError = ref("");
const unavailableReason = ref("");
const retainedCount = ref(10);
const page = ref(1);
const total = ref(0);
const action = ref("");
const restoreOpen = ref(false);
const restoreTarget = ref("");
const restoreSelection = ref<DatabaseBackup | null>(null);
const resourceKey = computed(
  () => `${organizationId.value}:${props.databaseId}`
);
const validRestoreTarget = computed(
  () =>
    /^[A-Za-z][A-Za-z0-9_]{0,62}$/.test(restoreTarget.value) &&
    restoreTarget.value !== restoreSelection.value?.databaseName
);
const operationPending = computed(() =>
  backups.value.some(
    (item) =>
      [
        DatabaseBackupStatus.BACKUP_CREATING,
        DatabaseBackupStatus.BACKUP_DELETING,
      ].includes(item.status) ||
      ["queued", "restoring"].includes(item.restoreStatus)
  )
);
let generation = 0;
let alive = true;
let controller: AbortController | undefined;
const backupStatus = (backup: DatabaseBackup) =>
  ({
    [DatabaseBackupStatus.DATABASE_BACKUP_STATUS_UNSPECIFIED]: "Unknown",
    [DatabaseBackupStatus.BACKUP_CREATING]: "Creating",
    [DatabaseBackupStatus.BACKUP_COMPLETED]: "Ready",
    [DatabaseBackupStatus.BACKUP_FAILED]: "Failed",
    [DatabaseBackupStatus.BACKUP_DELETING]: "Deleting",
    [DatabaseBackupStatus.BACKUP_DELETED]: "Deleted",
  }[backup.status] || "Unknown");
async function loadBackups() {
  if (!organizationId.value) return;
  const current = ++generation;
  controller?.abort();
  controller = new AbortController();
  loading.value = true;
  try {
    const result = await dbClient.listBackups(
      {
        organizationId: organizationId.value,
        databaseId: props.databaseId,
        page: page.value,
        perPage: 50,
      },
      { signal: controller.signal }
    );
    if (current !== generation) return;
    backups.value = result.backups;
    total.value = result.pagination?.total || 0;
    available.value = result.managedAvailable;
    unavailableReason.value = result.unavailableReason;
    retainedCount.value = result.retainedCount || 10;
    loaded.value = true;
    loadError.value = "";
    if (page.value > 1 && !result.backups.length)
      page.value = Math.max(1, Math.ceil(total.value / 50));
  } catch (error) {
    if (current !== generation) return;
    loadError.value = (error as Error).message;
    available.value = false;
    loaded.value = true;
  } finally {
    if (current === generation) loading.value = false;
  }
}
async function managedAction(
  name: string,
  perform: (ids: {
    organizationId: string;
    databaseId: string;
  }) => Promise<unknown>,
  message: string
) {
  if (!organizationId.value || action.value) return;
  const key = resourceKey.value;
  action.value = name;
  try {
    await perform({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
    });
    if (!alive || key !== resourceKey.value) return;
    toast.success(message);
    restoreOpen.value = false;
    await loadBackups();
  } catch (error) {
    if (alive && key === resourceKey.value)
      toast.error("Backup operation failed", (error as Error).message);
  } finally {
    if (key === resourceKey.value) action.value = "";
  }
}
const createBackup = () =>
  managedAction("create", (ids) => dbClient.createBackup(ids), "Backup queued");
function selectRestore(backup: DatabaseBackup) {
  restoreSelection.value = backup;
  restoreTarget.value = `restored_${Date.now()}`;
  restoreOpen.value = true;
}
function restoreBackup() {
  if (!validRestoreTarget.value || !restoreSelection.value) return;
  const backupId = restoreSelection.value.id,
    targetDatabaseName = restoreTarget.value;
  return managedAction(
    "restore",
    (ids) => dbClient.restoreBackup({ ...ids, backupId, targetDatabaseName }),
    "Restore queued"
  );
}
function deleteBackup(backup: DatabaseBackup) {
  if (
    !confirm(
      `Delete backup "${backup.name}"? This permanently removes its backup file.`
    )
  )
    return;
  return managedAction(
    "delete",
    (ids) => dbClient.deleteBackup({ ...ids, backupId: backup.id }),
    "Backup deletion queued"
  );
}
watch(resourceKey, () => {
  generation++;
  controller?.abort();
  backups.value = [];
  available.value = false;
  loaded.value = false;
  page.value = 1;
  total.value = 0;
  action.value = "";
  restoreOpen.value = false;
  restoreSelection.value = null;
  void loadBackups();
});
watch(page, () => {
  void loadBackups();
});
onMounted(() => {
  void loadBackups();
});
onUnmounted(() => {
  alive = false;
  generation++;
  controller?.abort();
});
useAutoRefresh(loadBackups, 5000);

async function handleExportDump() {
  exportingDump.value = true;
  try {
    if (!organizationId.value) return;
    const res = await dbClient.exportDatabaseDump({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      format: DatabaseDumpFormat.SQL,
      includeSchema: true,
      includeData: true,
    });

    const blob = new Blob([new TextDecoder().decode(res.dumpData)], {
      type: res.contentType || "application/sql",
    });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = res.fileName || `${props.databaseId}.sql`;
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
    URL.revokeObjectURL(url);
    toast.success("SQL dump exported");
  } catch (err: unknown) {
    toast.error("Failed to export SQL dump", (err as Error).message);
  } finally {
    exportingDump.value = false;
  }
}

async function handleImportDump(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (!file) return;

  if (
    !confirm(
      `Import "${file.name}" into this database? Existing objects may be modified by the SQL in the dump.`
    )
  ) {
    return;
  }

  importingDump.value = true;
  try {
    if (!organizationId.value) return;
    const data = new Uint8Array(await file.arrayBuffer());
    const res = await dbClient.importDatabaseDump({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      format: DatabaseDumpFormat.SQL,
      dumpData: data,
      dropExisting: false,
    });
    toast.success(res.message || "SQL dump imported");
  } catch (err: unknown) {
    toast.error("Failed to import SQL dump", (err as Error).message);
  } finally {
    importingDump.value = false;
  }
}
</script>
