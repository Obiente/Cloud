<template>
  <OuiCardBody>
    <OuiStack gap="md">
      <OuiFlex justify="between" align="center">
        <OuiStack gap="none">
          <OuiText as="h3" size="md" weight="semibold"
            >Environment Variables</OuiText
          >
          <OuiText size="sm" color="secondary">
            Manage environment variables for this deployment. Changes take
            effect on the next deployment.
          </OuiText>
        </OuiStack>
        <OuiFlex gap="sm" align="center">
          <OuiButton
            variant="ghost"
            size="sm"
            @click="viewMode = viewMode === 'list' ? 'file' : 'list'"
          >
            {{ viewMode === "list" ? "File View" : "List View" }}
          </OuiButton>
          <OuiButton v-if="viewMode === 'list'" size="sm" @click="addVariable">
            Add Variable
          </OuiButton>
        </OuiFlex>
      </OuiFlex>

      <div
        v-if="isLoading && envVars.length === 0"
        class="flex justify-center py-8"
      >
        <OuiText color="secondary">Loading environment variables...</OuiText>
      </div>

      <!-- List View -->
      <OuiStack v-if="viewMode === 'list'" gap="sm">
        <OuiStack v-if="!isLoading && envVars.length === 0" gap="sm">
          <OuiText color="secondary" class="text-center py-4">
            No environment variables set. Click "Add Variable" to add one.
          </OuiText>
        </OuiStack>

        <OuiStack v-else gap="sm">
          <div
            v-for="(env, idx) in envVars"
            :key="idx"
            class="flex flex-col gap-2 p-3 rounded-lg border border-border-default bg-surface-muted/30"
          >
            <div class="flex items-center gap-3">
              <OuiInput
                :model-value="env.key"
                placeholder="KEY"
                class="flex-1 uppercase"
                size="sm"
                @update:model-value="(val) => handleKeyUpdate(env, val)"
              />
              <OuiInput
                v-model="env.value"
                placeholder="value"
                class="flex-1"
                size="sm"
                @update:model-value="markDirty"
              />
              <OuiButton
                variant="ghost"
                size="sm"
                color="danger"
                @click="removeVariable(idx)"
              >
                Remove
              </OuiButton>
            </div>
            <div v-if="env.description !== undefined" class="flex items-center gap-2">
              <OuiText size="xs" color="secondary" class="w-4 shrink-0">#</OuiText>
              <OuiInput
                v-model="env.description"
                placeholder="Description (will be saved as comment)"
                size="sm"
                class="flex-1 text-xs"
                @update:model-value="markDirty"
              />
            </div>
            <OuiButton
              v-else
              variant="ghost"
              size="xs"
              @click="env.description = ''"
              class="self-start"
            >
              + Add description
            </OuiButton>
          </div>
        </OuiStack>
      </OuiStack>

      <!-- File View -->
      <div v-if="viewMode === 'file'" class="space-y-2">
        <OuiText size="sm" weight="medium">Edit as .env file</OuiText>
        <OuiFileEditor
          v-model="envFileContent"
          language="dotenv"
          :read-only="false"
          height="384px"
          :minimap="{ enabled: true }"
          :folding="false"
          @save="saveEnvVars"
          @change="markDirty"
        />
        <OuiText size="xs" color="secondary">
          Format: KEY=value (one per line). Empty lines are ignored. Press
          Ctrl+S to save.
        </OuiText>
      </div>

      <OuiFlex justify="end">
        <OuiButton
          @click="saveEnvVars"
          :disabled="!isDirty || isLoading"
          size="sm"
        >
          {{ isLoading ? "Saving..." : "Save Changes" }}
        </OuiButton>
      </OuiFlex>
    </OuiStack>
  </OuiCardBody>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, computed } from "vue";
  import { useConnectClient } from "~/lib/connect-client";
  import { DeploymentService } from "@obiente/proto";
  import type { Deployment } from "@obiente/proto";
  import { useOrganizationsStore } from "~/stores/organizations";
  import { usePreferencesStore } from "~/stores/preferences";
  import { useDialog } from "~/composables/useDialog";
  import OuiFileEditor from "~/components/oui/FileEditor.vue";

  interface Props {
    deployment: Deployment;
  }

  interface EnvVar {
    key: string;
    value: string;
    description?: string; // Comment from .env file
  }

  const props = defineProps<Props>();

  const emit = defineEmits<{
    save: [envFileContent: string];
  }>();

  const client = useConnectClient(DeploymentService);
  const orgsStore = useOrganizationsStore();
  const preferencesStore = usePreferencesStore();
  const organizationId = computed(() => orgsStore.currentOrgId || "");
  const envVars = ref<EnvVar[]>([]);
  const isDirty = ref(false);
  const isLoading = ref(false);
  const viewMode = computed({
    get: () => preferencesStore.envVarsViewMode,
    set: (value) => preferencesStore.setEnvVarsViewMode(value),
  });
  const envFileContent = ref("");

  // Convert env vars array to .env file format
  const envVarsToFile = (vars: EnvVar[]): string => {
    const lines: string[] = [];
    for (const env of vars) {
      if (!env.key.trim()) continue;
      // Write description/comment if present
      if (env.description) {
        lines.push(`# ${env.description}`);
      }
      lines.push(`${env.key.trim()}=${env.value || ""}`);
    }
    return lines.join("\n");
  };

  // Parse .env file format to env vars array
  const fileToEnvVars = (content: string): EnvVar[] => {
    const vars: EnvVar[] = [];
    const lines = content.split("\n");
    let pendingComment: string | undefined = undefined;

    for (const line of lines) {
      const trimmed = line.trim();
      
      // Handle comments - store as pending for next env var
      if (trimmed.startsWith("#")) {
        const commentText = trimmed.substring(1).trim();
        // Accumulate multiple consecutive comments
        pendingComment = pendingComment 
          ? `${pendingComment} ${commentText}`
          : commentText;
        continue;
      }

      // Skip empty lines
      if (!trimmed) {
        // Reset pending comment on empty line (comment belongs to previous var)
        pendingComment = undefined;
        continue;
      }

      // Parse env var line
      const equalIndex = trimmed.indexOf("=");
      let envVar: EnvVar;
      
      if (equalIndex === -1) {
        // No equals sign, treat as key with empty value
        envVar = { key: trimmed.toUpperCase(), value: "" };
      } else {
        const key = trimmed.substring(0, equalIndex).trim().toUpperCase();
        const value = trimmed.substring(equalIndex + 1).trim();
        // Handle quoted values
        const unquotedValue =
          value.startsWith('"') && value.endsWith('"')
            ? value.slice(1, -1)
            : value.startsWith("'") && value.endsWith("'")
            ? value.slice(1, -1)
            : value;
        envVar = { key, value: unquotedValue };
      }

      // Attach pending comment as description
      if (pendingComment) {
        envVar.description = pendingComment;
        pendingComment = undefined; // Reset after using
      }

      vars.push(envVar);
    }

    return vars;
  };


  const loadEnvVars = async () => {
    isLoading.value = true;
    try {
      // Load from backend - now returns raw .env file content with comments
      const res = await client.getDeploymentEnvVars({
        organizationId: organizationId.value,
        deploymentId: props.deployment.id,
      });
      
      envFileContent.value = res.envFileContent || "";
      
      // Parse to update envVars array (with uppercase keys and descriptions)
      envVars.value = fileToEnvVars(envFileContent.value);

      isDirty.value = false;
    } catch (error) {
      console.error("Failed to load env vars:", error);
    } finally {
      isLoading.value = false;
    }
  };

  const markDirty = () => {
    isDirty.value = true;
  };

  const addVariable = () => {
    envVars.value.push({ key: "", value: "" });
    markDirty();
  };

  // Ensure key is always uppercase when editing
  const handleKeyUpdate = (env: EnvVar, newValue: string) => {
    env.key = newValue.toUpperCase();
    markDirty();
  };

  const removeVariable = (idx: number) => {
    envVars.value.splice(idx, 1);
    markDirty();
  };

  // Sync file content to env vars when switching to list view
  watch(viewMode, (newMode, oldMode) => {
    // Only sync if actually changing modes (not on initial load)
    if (oldMode === undefined || oldMode === newMode) return;
    
    if (newMode === "list") {
      // Parse file content back to env vars
      envVars.value = fileToEnvVars(envFileContent.value);
    } else {
      // Sync env vars to file content when switching to file view
      envFileContent.value = envVarsToFile(envVars.value);
    }
  });

  const saveEnvVars = async () => {
    // Store current view mode to preserve it
    const currentViewMode = viewMode.value;

    // Get file content from current view mode
    let fileContentToSave: string = "";

    if (currentViewMode === "file") {
      // Use file content directly (includes comments)
      fileContentToSave = envFileContent.value;
      // Update envVars array to keep in sync (with uppercase keys and descriptions)
      envVars.value = fileToEnvVars(fileContentToSave);
    } else {
      // Generate file content from list view (includes descriptions as comments)
      fileContentToSave = envVarsToFile(envVars.value);
      envFileContent.value = fileContentToSave;
    }

    isLoading.value = true;
    try {
      await client.updateDeploymentEnvVars({
        organizationId: organizationId.value,
        deploymentId: props.deployment.id,
        envFileContent: fileContentToSave,
      });
      
      emit("save", fileContentToSave);
      isDirty.value = false;

      // Ensure we stay in the current view mode
      // This prevents any accidental view mode switching
      if (viewMode.value !== currentViewMode) {
        viewMode.value = currentViewMode;
      }
    } catch (error) {
      console.error("Failed to save env vars:", error);
      const { showAlert } = useDialog();
      await showAlert({
        title: "Error",
        message: "Failed to save environment variables. Please try again.",
      });
    } finally {
      isLoading.value = false;
    }
  };

  watch(
    () => props.deployment.id,
    () => {
      loadEnvVars();
    },
    { immediate: true }
  );

  onMounted(async () => {
    // Ensure preferences are hydrated before reading viewMode
    preferencesStore.hydrate();
    await loadEnvVars();
    // Sync file content based on current view mode after load
    if (viewMode.value === "file") {
      envFileContent.value = envVarsToFile(envVars.value);
    }
  });
</script>
