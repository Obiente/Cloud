<template>
  <OuiStack gap="lg">
    <OuiCard variant="outline">
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiFlex justify="between" align="start" gap="md" wrap="wrap">
            <OuiStack gap="xs">
              <OuiText size="sm" weight="semibold">SQL Dumps</OuiText>
              <OuiText size="xs" color="tertiary">
                Export or import portable .sql dumps for PostgreSQL, MySQL, and MariaDB.
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
          <OuiFlex justify="between" align="center">
            <OuiText size="sm" weight="semibold">Backups</OuiText>
            <OuiButton size="xs" @click="showCreateDialog = true">
              <PlusIcon class="h-3.5 w-3.5" />
              Create Backup
            </OuiButton>
          </OuiFlex>
          <!-- Loading State -->
          <OuiStack v-if="loading" align="center" gap="md" class="py-10">
            <OuiSpinner size="lg" />
            <OuiText color="tertiary">Loading backups...</OuiText>
          </OuiStack>

          <!-- Backups List -->
          <OuiTable 
            v-else-if="backups.length > 0"
            :columns="[
              { key: 'name', label: 'Name' },
              { key: 'size', label: 'Size' },
              { key: 'status', label: 'Status' },
              { key: 'created', label: 'Created' },
              { key: 'actions', label: 'Actions' },
            ]"
            :rows="backups.map((b: DatabaseBackup) => ({
              name: b.name,
              size: formatBytes(b.sizeBytes),
              status: b.status,
              created: formatDate(b.createdAt),
              actions: b,
            }))"
          >
            <template #cell-status="{ row }">
              <OuiBadge :color="getStatusColor(row.status)">
                {{ getStatusLabel(row.status) }}
              </OuiBadge>
            </template>
            <template #cell-actions="{ row }">
              <OuiFlex gap="sm">
                <OuiButton
                  variant="ghost"
                  size="sm"
                  @click="handleRestore(row.actions)"
                >
                  Restore
                </OuiButton>
                <OuiButton
                  variant="ghost"
                  size="sm"
                  color="danger"
                  @click="handleDelete(row.actions)"
                >
                  Delete
                </OuiButton>
              </OuiFlex>
            </template>
          </OuiTable>

          <!-- Empty State -->
          <OuiStack v-else align="center" gap="md" class="py-10">
            <OuiText color="tertiary">No backups found</OuiText>
          </OuiStack>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- Create Backup Dialog -->
    <OuiDialog v-model:open="showCreateDialog" title="Create Backup">
      <OuiDialogContent>
        <OuiDialogHeader>
          <OuiDialogTitle>Create Backup</OuiDialogTitle>
          <OuiDialogDescription>
            Create a new backup of your database
          </OuiDialogDescription>
        </OuiDialogHeader>

        <OuiStack gap="md" class="py-4">
          <OuiInput
            v-model="backupName"
            label="Backup Name"
            placeholder="backup-2024-01-01"
          />
          <OuiTextarea
            v-model="backupDescription"
            label="Description"
            placeholder="Optional description"
            :rows="3"
          />
        </OuiStack>

        <OuiDialogFooter>
          <OuiButton variant="ghost" @click="showCreateDialog = false">
            Cancel
          </OuiButton>
          <OuiButton color="primary" :loading="creating" @click="handleCreate">
            Create Backup
          </OuiButton>
        </OuiDialogFooter>
      </OuiDialogContent>
    </OuiDialog>
  </OuiStack>
</template>

<script setup lang="ts">
import { ArrowDownTrayIcon, ArrowUpTrayIcon, PlusIcon } from "@heroicons/vue/24/outline";
import { ref, onMounted } from "vue";
import { DatabaseService, DatabaseBackupStatus, DatabaseDumpFormat, type DatabaseBackup } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";
import { useToast } from "~/composables/useToast";
import { formatBytes, formatDate } from "~/utils/common";

const props = defineProps<{
  databaseId: string;
}>();

const organizationId = useOrganizationId();
const { toast } = useToast();
const dbClient = useConnectClient(DatabaseService);

const loading = ref(false);
const backups = ref<any[]>([]);
const showCreateDialog = ref(false);
const backupName = ref("");
const backupDescription = ref("");
const creating = ref(false);
const exportingDump = ref(false);
const importingDump = ref(false);
const dumpFileInput = ref<HTMLInputElement | null>(null);

async function loadBackups() {
  loading.value = true;
  try {
    if (!organizationId.value) return;
    const res = await dbClient.listBackups({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      page: 1,
      perPage: 100,
    });
    backups.value = res.backups || [];
  } catch (err: unknown) {
    toast.error("Failed to load backups", (err as Error).message);
  } finally {
    loading.value = false;
  }
}

async function handleCreate() {
  creating.value = true;
  try {
    if (!organizationId.value) return;
    await dbClient.createBackup({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      name: backupName.value || undefined,
      description: backupDescription.value || undefined,
    });
    toast.success("Backup created");
    showCreateDialog.value = false;
    backupName.value = "";
    backupDescription.value = "";
    loadBackups();
  } catch (err: unknown) {
    toast.error("Failed to create backup", (err as Error).message);
  } finally {
    creating.value = false;
  }
}

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

  if (!confirm(`Import "${file.name}" into this database? Existing objects may be modified by the SQL in the dump.`)) {
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

async function handleRestore(backup: DatabaseBackup) {
  if (!confirm(`Are you sure you want to restore from backup "${backup.name}"?`)) {
    return;
  }

  try {
    if (!organizationId.value) return;
    await dbClient.restoreBackup({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      backupId: backup.id,
    });
    toast.success("Backup restoration started");
  } catch (err: unknown) {
    toast.error("Failed to restore backup", (err as Error).message);
  }
}

async function handleDelete(backup: DatabaseBackup) {
  if (!confirm(`Are you sure you want to delete backup "${backup.name}"?`)) {
    return;
  }

  try {
    if (!organizationId.value) return;
    await dbClient.deleteBackup({
      organizationId: organizationId.value,
      databaseId: props.databaseId,
      backupId: backup.id,
    });
    toast.success("Backup deleted");
    loadBackups();
  } catch (err: unknown) {
    toast.error("Failed to delete backup", (err as Error).message);
  }
}

function getStatusLabel(status: number | string): string {
  const statusValue = typeof status === "string" ? parseInt(status) : status;
  const statuses: Record<number, string> = {
    [DatabaseBackupStatus.BACKUP_CREATING]: "Creating",
    [DatabaseBackupStatus.BACKUP_COMPLETED]: "Completed",
    [DatabaseBackupStatus.BACKUP_FAILED]: "Failed",
    [DatabaseBackupStatus.BACKUP_DELETING]: "Deleting",
    [DatabaseBackupStatus.BACKUP_DELETED]: "Deleted",
  };
  return statuses[statusValue] || `Status ${statusValue}`;
}

function getStatusColor(status: number | string): string {
  const statusValue = typeof status === "string" ? parseInt(status) : status;
  const colors: Record<number, string> = {
    [DatabaseBackupStatus.BACKUP_CREATING]: "warning",
    [DatabaseBackupStatus.BACKUP_COMPLETED]: "success",
    [DatabaseBackupStatus.BACKUP_FAILED]: "danger",
    [DatabaseBackupStatus.BACKUP_DELETING]: "warning",
    [DatabaseBackupStatus.BACKUP_DELETED]: "secondary",
  };
  return colors[statusValue] || "secondary";
}

onMounted(() => {
  loadBackups();
});
</script>
