<template>
  <OuiStack gap="lg">
    <OuiCard>
      <OuiCardHeader>
        <OuiFlex justify="between" align="center">
          <OuiStack gap="xs">
            <OuiCardTitle>Backups</OuiCardTitle>
            <OuiCardDescription>
              Manage database backups and restorations
            </OuiCardDescription>
          </OuiStack>
          <OuiButton @click="showCreateDialog = true">
            <PlusIcon class="h-4 w-4" />
            Create Backup
          </OuiButton>
        </OuiFlex>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <!-- Loading State -->
          <OuiStack v-if="loading" align="center" gap="md" class="py-10">
            <OuiSpinner size="lg" />
            <OuiText color="secondary">Loading backups...</OuiText>
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
            :rows="backups.map((b: any) => ({
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
            <OuiText color="secondary">No backups found</OuiText>
          </OuiStack>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- Create Backup Dialog -->
    <OuiDialog v-model="showCreateDialog" title="Create Backup">
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
import { PlusIcon } from "@heroicons/vue/24/outline";
import { ref, onMounted } from "vue";
import { DatabaseService, DatabaseBackupStatus } from "@obiente/proto";
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
  } catch (err: any) {
    toast.error("Failed to load backups", err.message);
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
  } catch (err: any) {
    toast.error("Failed to create backup", err.message);
  } finally {
    creating.value = false;
  }
}

async function handleRestore(backup: any) {
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
  } catch (err: any) {
    toast.error("Failed to restore backup", err.message);
  }
}

async function handleDelete(backup: any) {
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
  } catch (err: any) {
    toast.error("Failed to delete backup", err.message);
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

