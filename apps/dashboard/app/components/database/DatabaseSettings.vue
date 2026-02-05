<template>
  <OuiStack gap="lg" p="lg" md:p="xl">
    <form @submit.prevent="handleSave">
      <OuiStack gap="lg">
        <!-- Basic Settings -->`
        <OuiStack gap="md">
          <OuiText as="h3" size="lg" weight="bold">Basic Settings</OuiText>
          
          <OuiStack gap="md">
            <OuiFormField label="Database Name" required>
              <OuiInput
                v-model="formData.name"
                placeholder="Enter database name"
                :disabled="isSaving"
              />
            </OuiFormField>

            <OuiFormField label="Database Type">
              <OuiSelect
                v-model="formData.type"
                :items="databaseTypeOptions"
                :disabled="true"
                placeholder="Select type"
              />
            </OuiFormField>
          </OuiStack>
        </OuiStack>

        <!-- Resource Settings -->
        <OuiStack gap="md">
          <OuiText as="h3" size="lg" weight="bold">Resources</OuiText>
          
          <OuiStack gap="md">
            <OuiFormField label="CPU Cores">
              <OuiInput
                :model-value="String(formData.cpuCores)"
                type="number"
                :min="1"
                :max="32"
                :disabled="true"
                placeholder="Number of CPU cores"
              />
              <OuiText size="xs" color="secondary" class="mt-1">
                Modify through recreating the database
              </OuiText>
            </OuiFormField>

            <OuiFormField label="Memory (GB)">
              <OuiInput
                :model-value="String(formData.memoryGb)"
                type="number"
                :min="1"
                :max="512"
                :disabled="true"
                placeholder="Amount in GB"
              />
              <OuiText size="xs" color="secondary" class="mt-1">
                Modify through recreating the database
              </OuiText>
            </OuiFormField>

            <OuiFormField label="Storage (GB)">
              <OuiInput
                :model-value="String(formData.diskGb)"
                type="number"
                :min="10"
                :max="10000"
                :disabled="true"
                placeholder="Amount in GB"
              />
              <OuiText size="xs" color="secondary" class="mt-1">
                Modify through recreating the database
              </OuiText>
            </OuiFormField>
          </OuiStack>
        </OuiStack>

        <!-- Connection Settings -->
        <OuiStack gap="md">
          <OuiText as="h3" size="lg" weight="bold">Connection Settings</OuiText>
          
          <OuiStack gap="md">
            <OuiFormField label="Connection Username">
              <OuiInput
                v-model="formData.username"
                placeholder="Username"
                :disabled="true"
              />
              <OuiText size="xs" color="secondary" class="mt-1">
                Set during database creation
              </OuiText>
            </OuiFormField>

            <OuiFormField label="Hostname">
              <OuiInput
                v-model="formData.host"
                placeholder="Hostname"
                :disabled="true"
              />
              <OuiText size="xs" color="secondary" class="mt-1">
                Auto-generated for routing
              </OuiText>
            </OuiFormField>

            <OuiFormField label="Port">
              <OuiInput
                :model-value="String(formData.port)"
                type="number"
                placeholder="Port"
                :disabled="true"
              />
              <OuiText size="xs" color="secondary" class="mt-1">
                Standard port for database type
              </OuiText>
            </OuiFormField>
          </OuiStack>
        </OuiStack>

        <!-- Auto-Sleep Settings -->
        <OuiStack gap="md">
          <OuiText as="h3" size="lg" weight="bold">Auto-Sleep</OuiText>

          <OuiStack gap="md">
            <OuiFormField label="Enable Auto-Sleep">
              <OuiCheckbox
                v-model="formData.autoSleepEnabled"
                label="Automatically sleep database after inactivity"
                :disabled="isSaving"
              />
              <OuiText size="xs" color="secondary" class="mt-1">
                The database will stop when idle to save resources. It will start automatically when a connection is made.
              </OuiText>
            </OuiFormField>

            <OuiFormField
              v-if="formData.autoSleepEnabled"
              label="Sleep After (minutes)"
            >
              <OuiInput
                :model-value="String(formData.autoSleepMinutes)"
                @update:model-value="formData.autoSleepMinutes = Number($event) || 30"
                type="number"
                :min="5"
                :max="1440"
                placeholder="Minutes of inactivity"
                :disabled="isSaving"
              />
              <OuiText size="xs" color="secondary" class="mt-1">
                Minimum 5 minutes. The database will sleep after this many minutes without connections.
              </OuiText>
            </OuiFormField>
          </OuiStack>
        </OuiStack>

        <!-- Backup & Recovery Settings -->
        <OuiStack gap="md">
          <OuiText as="h3" size="lg" weight="bold">Backup & Recovery</OuiText>
          
          <OuiStack gap="md">
            <OuiFormField label="Auto Backups Enabled">
              <OuiCheckbox
                v-model="formData.autoBackupEnabled"
                label="Enable automatic backups"
                :disabled="isSaving"
              />
              <OuiText size="xs" color="secondary" class="mt-1">
                Automatically backup database daily
              </OuiText>
            </OuiFormField>

            <OuiFormField 
              v-if="formData.autoBackupEnabled" 
              label="Backup Retention Days"
            >
              <OuiInput
                v-model.number="formData.backupRetentionDays"
                type="number"
                :min="1"
                :max="365"
                placeholder="Number of days"
                :disabled="isSaving"
              />
              <OuiText size="xs" color="secondary" class="mt-1">
                Keep backups for this many days
              </OuiText>
            </OuiFormField>
          </OuiStack>
        </OuiStack>

        <!-- Actions -->
        <OuiFlex gap="md" wrap="wrap" justify="between" align="center">
          <OuiText size="sm" color="secondary">
            {{ unsavedChanges ? '⚠ You have unsaved changes' : 'All changes saved' }}
          </OuiText>
          <OuiFlex gap="sm">
            <OuiButton
              variant="ghost"
              color="secondary"
              @click="handleReset"
              :disabled="isSaving || !unsavedChanges"
            >
              Cancel
            </OuiButton>
            <OuiButton
              variant="solid"
              color="primary"
              type="submit"
              :loading="isSaving"
              :disabled="!unsavedChanges"
            >
              Save Changes
            </OuiButton>
          </OuiFlex>
        </OuiFlex>
      </OuiStack>
    </form>
  </OuiStack>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { DatabaseType, DatabaseService, type DatabaseInstance } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";
import { useToast } from "~/composables/useToast";

const props = defineProps<{
  database: DatabaseInstance;
}>();

const emit = defineEmits<{
  save: [];
}>();

const isSaving = ref(false);

const databaseTypeOptions = [
  { label: "PostgreSQL", value: DatabaseType.POSTGRESQL.toString() },
  { label: "MySQL", value: DatabaseType.MYSQL.toString() },
  { label: "MongoDB", value: DatabaseType.MONGODB.toString() },
  { label: "Redis", value: DatabaseType.REDIS.toString() },
  { label: "MariaDB", value: DatabaseType.MARIADB.toString() },
];

const db = props.database as any;
const formData = ref({
  name: db?.name || "",
  type: db?.type?.toString() || "",
  cpuCores: db?.cpuCores || 1,
  memoryGb: Math.round(Number(db?.memoryBytes || 0) / 1024 / 1024 / 1024),
  diskGb: Math.round(Number(db?.diskBytes || 0) / 1024 / 1024 / 1024),
  username: db?.username || "",
  host: db?.host || "",
  port: db?.port || 5432,
  autoSleepEnabled: (db?.autoSleepSeconds || 0) > 0,
  autoSleepMinutes: Math.max(5, Math.round((db?.autoSleepSeconds || 0) / 60)) || 30,
  autoBackupEnabled: db?.autoBackupEnabled !== false,
  backupRetentionDays: db?.backupRetentionDays || 7,
});

const originalFormData = ref({ ...formData.value });

const unsavedChanges = computed(() => {
  return JSON.stringify(formData.value) !== JSON.stringify(originalFormData.value);
});

watch(() => props.database, (newDatabase) => {
  if (newDatabase) {
    const d = newDatabase as any;
    formData.value = {
      name: d.name || "",
      type: d.type?.toString() || "",
      cpuCores: d.cpuCores || 1,
      memoryGb: Math.round(Number(d.memoryBytes || 0) / 1024 / 1024 / 1024),
      diskGb: Math.round(Number(d.diskBytes || 0) / 1024 / 1024 / 1024),
      username: d.username || "",
      host: d.host || "",
      port: d.port || 5432,
      autoSleepEnabled: (d.autoSleepSeconds || 0) > 0,
      autoSleepMinutes: Math.max(5, Math.round((d.autoSleepSeconds || 0) / 60)) || 30,
      autoBackupEnabled: d.autoBackupEnabled !== false,
      backupRetentionDays: d.backupRetentionDays || 7,
    };
    originalFormData.value = { ...formData.value };
  }
}, { deep: true });

async function handleSave() {
  if (!unsavedChanges.value) return;
  
  isSaving.value = true;
  const client = useConnectClient(DatabaseService);
  const organizationId = useOrganizationId();
  const { toast } = useToast();
  
  try {
    // Only send changed fields that are editable
    const updates: any = {};
    
    const dbRef = props.database as any;
    if (formData.value.name !== dbRef.name) {
      updates.name = formData.value.name;
    }

    if (formData.value.autoBackupEnabled !== dbRef.autoBackupEnabled) {
      updates.autoBackupEnabled = formData.value.autoBackupEnabled;
    }

    if (formData.value.backupRetentionDays !== dbRef.backupRetentionDays) {
      updates.backupRetentionDays = formData.value.backupRetentionDays;
    }

    // Auto-sleep: convert minutes to seconds for API
    const newAutoSleepSeconds = formData.value.autoSleepEnabled
      ? Math.max(300, formData.value.autoSleepMinutes * 60)
      : 0;
    const oldAutoSleepSeconds = dbRef.autoSleepSeconds || 0;
    if (newAutoSleepSeconds !== oldAutoSleepSeconds) {
      updates.autoSleepSeconds = newAutoSleepSeconds;
    }

    await client.updateDatabase({
      organizationId: organizationId.value || "",
      databaseId: props.database.id,
      ...updates,
    });

    toast.success("Settings saved successfully");
    originalFormData.value = { ...formData.value };
    emit("save");
  } catch (error) {
    console.error("Failed to save settings:", error);
    const { toast } = useToast();
    toast.error(
      error instanceof Error ? error.message : "Failed to save settings. Please try again."
    );
  } finally {
    isSaving.value = false;
  }
}

function handleReset() {
  formData.value = { ...originalFormData.value };
}
</script>
