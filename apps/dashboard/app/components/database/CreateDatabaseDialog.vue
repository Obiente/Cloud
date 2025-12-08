<template>
  <OuiDialog 
    :open="modelValue" 
    @update:open="$emit('update:modelValue', $event)"
    title="Create New Database"
  >
    <OuiDialogContent size="lg">
      <OuiDialogHeader>
        <OuiDialogTitle>Create New Database</OuiDialogTitle>
        <OuiDialogDescription>
          Deploy a new managed database instance. Choose your database type, size, and configuration.
        </OuiDialogDescription>
      </OuiDialogHeader>

      <OuiStack gap="lg" class="py-4">
        <OuiInput
          v-model="form.name"
          label="Database Name"
          placeholder="my-database"
          required
          :error="errors.name"
        />

          <OuiTextarea
            v-model="form.description"
            label="Description"
            placeholder="Optional description for this database"
            :rows="2"
          />

        <OuiSelect
          v-model="form.type"
          label="Database Type"
          :items="typeOptions"
          required
          :error="errors.type"
        />

        <OuiSelect
          v-model="form.size"
          label="Size"
          :items="sizeOptions"
          required
          :error="errors.size"
        />

        <OuiInput
          v-model="form.version"
          label="Version (Optional)"
          placeholder="e.g., 15, 8.0, 7.0"
        />

        <OuiInput
          v-model="form.initialDatabaseName"
          label="Initial Database Name"
          placeholder="default"
        />

        <OuiInput
          v-model="form.initialUsername"
          label="Initial Username"
          placeholder="admin"
        />

        <OuiInput
          v-model="form.initialPassword"
          label="Initial Password (Optional)"
          type="password"
          placeholder="Leave empty for auto-generated password"
        />
      </OuiStack>

      <OuiDialogFooter>
        <OuiButton variant="ghost" @click="$emit('update:modelValue', false)">
          Cancel
        </OuiButton>
        <OuiButton
          color="primary"
          :loading="creating"
          @click="handleCreate"
        >
          Create Database
        </OuiButton>
      </OuiDialogFooter>
    </OuiDialogContent>
  </OuiDialog>
</template>

<script setup lang="ts">
import { ref, reactive } from "vue";
import { DatabaseService, DatabaseType } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";
import { useToast } from "~/composables/useToast";

// Helper to convert string to enum
function getDatabaseType(type: string): DatabaseType {
  const typeMap: Record<string, DatabaseType> = {
    POSTGRESQL: DatabaseType.POSTGRESQL,
    MYSQL: DatabaseType.MYSQL,
    MONGODB: DatabaseType.MONGODB,
    REDIS: DatabaseType.REDIS,
    MARIADB: DatabaseType.MARIADB,
  };
  return typeMap[type] || DatabaseType.POSTGRESQL;
}

const props = defineProps<{
  modelValue: boolean;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  created: [];
}>();

const organizationId = useOrganizationId();
const { toast } = useToast();
const dbClient = useConnectClient(DatabaseService);

const creating = ref(false);
const errors = reactive<Record<string, string>>({});

const form = reactive({
  name: "",
  description: "",
  type: "POSTGRESQL",
  size: "small",
  version: "",
  initialDatabaseName: "default",
  initialUsername: "admin",
  initialPassword: "",
});

const typeOptions = [
  { label: "PostgreSQL", value: "POSTGRESQL" },
  { label: "MySQL", value: "MYSQL" },
  { label: "MongoDB", value: "MONGODB" },
  { label: "Redis", value: "REDIS" },
  { label: "MariaDB", value: "MARIADB" },
];

const sizeOptions = [
  { label: "Small (1 CPU, 2GB RAM, 10GB Storage)", value: "small" },
  { label: "Medium (2 CPU, 4GB RAM, 50GB Storage)", value: "medium" },
  { label: "Large (4 CPU, 8GB RAM, 100GB Storage)", value: "large" },
];

async function handleCreate() {
  // Reset errors
  Object.keys(errors).forEach((key) => {
    errors[key] = "";
  });

  // Validate
  if (!form.name) {
    errors.name = "Database name is required";
    return;
  }
  if (!form.type) {
    errors.type = "Database type is required";
    return;
  }
  if (!form.size) {
    errors.size = "Size is required";
    return;
  }

  creating.value = true;

  try {
    if (!organizationId.value) {
      toast.error("Organization ID is required");
      return;
    }

    const response = await dbClient.createDatabase({
      organizationId: organizationId.value,
      name: form.name,
      description: form.description || undefined,
      type: getDatabaseType(form.type),
      size: form.size,
      version: form.version || undefined,
      initialDatabaseName: form.initialDatabaseName || undefined,
      initialUsername: form.initialUsername || undefined,
      initialPassword: form.initialPassword || undefined,
    });
    
    toast.success("Database created successfully");
    
    // Show connection info if available
    if (response.connectionInfo) {
      toast.info(
        `Connection info available. Password: ${response.connectionInfo.password}`,
        "Save this password - it won't be shown again!"
      );
    }

    // Reset form
    form.name = "";
    form.description = "";
    form.type = "POSTGRESQL";
    form.size = "small";
    form.version = "";
    form.initialDatabaseName = "default";
    form.initialUsername = "admin";
    form.initialPassword = "";

    emit("update:modelValue", false);
    emit("created");
  } catch (err: any) {
    toast.error("Failed to create database", err.message);
  } finally {
    creating.value = false;
  }
}
</script>

