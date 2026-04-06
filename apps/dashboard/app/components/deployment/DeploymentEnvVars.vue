<template>
    <OuiStack gap="md">
      <OuiFlex justify="between" align="center" wrap="wrap" gap="sm">
        <OuiStack gap="none">
          <OuiText as="h3" size="sm" weight="semibold"
            >Environment Variables</OuiText
          >
          <OuiText size="xs" color="tertiary">
            Changes take effect on the next deployment.
          </OuiText>
        </OuiStack>
        <OuiFlex gap="sm" align="center">
          <!-- View toggle -->
          <div class="flex rounded-lg border border-border-default overflow-hidden">
            <button
              class="px-3 py-1.5 text-xs font-medium transition-colors"
              :class="viewMode === 'list' ? 'bg-surface-muted text-primary' : 'text-tertiary hover:text-secondary'"
              @click="viewMode = 'list'"
            >
              List
            </button>
            <button
              class="px-3 py-1.5 text-xs font-medium transition-colors border-l border-border-default"
              :class="viewMode === 'file' ? 'bg-surface-muted text-primary' : 'text-tertiary hover:text-secondary'"
              @click="viewMode = 'file'"
            >
              .env
            </button>
          </div>
          <OuiButton v-if="viewMode === 'list'" size="sm" @click="addVariable">
            Add Variable
          </OuiButton>
        </OuiFlex>
      </OuiFlex>

      <OuiFlex
        v-if="isLoading && envVars.length === 0"
        justify="center"
        class="py-8"
      >
        <OuiText color="tertiary">Loading environment variables...</OuiText>
      </OuiFlex>

      <!-- List View -->
      <template v-if="viewMode === 'list'">
        <!-- Empty State -->
        <OuiCard v-if="!isLoading && envVars.length === 0" variant="outline">
          <OuiCardBody>
            <OuiStack gap="md" align="center" class="py-6">
              <div class="h-10 w-10 rounded-xl bg-surface-muted flex items-center justify-center">
                <VariableIcon class="h-5 w-5 text-secondary" />
              </div>
              <OuiStack gap="xs" align="center">
                <OuiText size="sm" weight="medium">No variables set</OuiText>
                <OuiText size="xs" color="tertiary">Click "Add Variable" to get started.</OuiText>
              </OuiStack>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Variable Rows -->
        <OuiCard v-else variant="outline">
          <OuiStack gap="none" class="divide-y divide-border-default">
            <div
              v-for="env in envVars"
              :key="env._id"
              class="px-4 py-3 group hover:bg-surface-muted/30 transition-colors"
            >
              <OuiStack gap="xs">
                <OuiFlex align="center" gap="sm">
                  <OuiInput
                    :model-value="env.key"
                    placeholder="KEY"
                    class="flex-1 uppercase font-mono"
                    size="sm"
                    @update:model-value="(val) => handleKeyUpdate(env, val)"
                  />
                  <OuiText size="xs" color="tertiary" class="shrink-0">=</OuiText>
                  <OuiInput
                    v-model="env.value"
                    placeholder="value"
                    class="flex-[2] font-mono"
                    size="sm"
                    @update:model-value="markDirty"
                  />
                  <OuiButton
                    variant="ghost"
                    size="xs"
                    color="danger"
                    @click="removeVariable(env._id)"
                    class="opacity-0 group-hover:opacity-100 transition-opacity shrink-0"
                  >
                    <TrashIcon class="h-3.5 w-3.5" />
                  </OuiButton>
                </OuiFlex>
                <OuiFlex v-if="env.description !== undefined" align="center" gap="xs" class="pl-1">
                  <OuiText size="xs" color="tertiary" class="shrink-0">#</OuiText>
                  <OuiInput
                    v-model="env.description"
                    placeholder="Description (saved as comment)"
                    size="sm"
                    class="flex-1 text-xs"
                    @update:model-value="markDirty"
                  />
                </OuiFlex>
                <OuiButton
                  v-else
                  variant="ghost"
                  size="xs"
                  @click="env.description = ''"
                  class="self-start opacity-0 group-hover:opacity-100 transition-opacity"
                >
                  + Add comment
                </OuiButton>
              </OuiStack>
            </div>
          </OuiStack>
        </OuiCard>
      </template>

      <!-- File View -->
      <OuiStack v-if="viewMode === 'file'" gap="xs">
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
        <OuiText size="xs" color="tertiary">
          Format: KEY=value (one per line). Press Ctrl+S to save.
        </OuiText>
      </OuiStack>

      <OuiFlex justify="between" align="center">
        <OuiText v-if="envVars.length > 0" size="xs" color="tertiary">
          {{ envVars.length }} variable{{ envVars.length !== 1 ? 's' : '' }}
        </OuiText>
        <div v-else />
        <OuiButton
          @click="saveEnvVars"
          :disabled="!isDirty || isLoading"
          size="sm"
        >
          {{ isLoading ? "Saving..." : "Save Changes" }}
        </OuiButton>
      </OuiFlex>
    </OuiStack>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, computed } from "vue";
  import { TrashIcon, VariableIcon } from "@heroicons/vue/24/outline";
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

  let _envVarIdCounter = 0;

  interface EnvVar {
    _id: number;
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
        envVar = { _id: ++_envVarIdCounter, key: trimmed.toUpperCase(), value: "" };
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
        envVar = { _id: ++_envVarIdCounter, key, value: unquotedValue };
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
    envVars.value.push({ _id: ++_envVarIdCounter, key: "", value: "" });
    markDirty();
  };

  // Ensure key is always uppercase when editing
  const handleKeyUpdate = (env: EnvVar, newValue: string) => {
    env.key = newValue.toUpperCase();
    markDirty();
  };

  const removeVariable = (id: number) => {
    const idx = envVars.value.findIndex((e) => e._id === id);
    if (idx !== -1) envVars.value.splice(idx, 1);
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

  const ENV_KEY_REGEX = /^[A-Z_][A-Z0-9_]*$/;

  const validateEnvVars = (): string | null => {
    if (viewMode.value !== "list") return null;
    const keys = new Set<string>();
    for (const env of envVars.value) {
      const key = env.key.trim();
      if (!key) return "All variable keys must be non-empty.";
      if (!ENV_KEY_REGEX.test(key))
        return `Invalid key "${key}". Keys must start with a letter or underscore and contain only uppercase letters, digits, and underscores.`;
      if (keys.has(key)) return `Duplicate key "${key}". Each key must be unique.`;
      keys.add(key);
    }
    return null;
  };

  const saveEnvVars = async () => {
    // Validate list-view entries before saving
    const validationError = validateEnvVars();
    if (validationError) {
      const { showAlert } = useDialog();
      await showAlert({ title: "Validation Error", message: validationError });
      return;
    }

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
