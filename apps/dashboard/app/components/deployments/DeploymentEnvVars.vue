<template>
  <OuiCardBody>
    <OuiStack gap="md">
      <OuiFlex justify="between" align="center">
        <OuiText as="h3" size="md" weight="semibold">Environment Variables</OuiText>
        <OuiButton size="sm" @click="addVariable">Add Variable</OuiButton>
      </OuiFlex>

      <OuiText size="sm" color="secondary">
        Manage environment variables for this deployment. Changes take effect on the next deployment.
      </OuiText>

      <OuiStack v-if="envVars.length === 0" gap="sm">
        <OuiText color="secondary" class="text-center py-4">
          No environment variables set. Click "Add Variable" to add one.
        </OuiText>
      </OuiStack>

      <OuiStack v-else gap="sm">
        <div
          v-for="(env, idx) in envVars"
          :key="idx"
          class="flex items-center gap-3 p-3 rounded-lg border border-border-default bg-surface-muted/30"
        >
          <OuiInput
            v-model="env.key"
            placeholder="KEY"
            class="flex-1"
            size="sm"
            @update:model-value="markDirty"
          />
          <OuiInput
            v-model="env.value"
            placeholder="value"
            type="password"
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
      </OuiStack>

      <OuiFlex justify="end">
        <OuiButton @click="saveEnvVars" :disabled="!isDirty" size="sm">
          Save Changes
        </OuiButton>
      </OuiFlex>
    </OuiStack>
  </OuiCardBody>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from "vue";
import type { Deployment } from "@obiente/proto";

interface Props {
  deployment: Deployment;
}

interface EnvVar {
  key: string;
  value: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  save: [envVars: Record<string, string>];
}>();

const envVars = ref<EnvVar[]>([]);
const isDirty = ref(false);

// Load existing env vars
watch(
  () => props.deployment,
  (deployment) => {
    // TODO: Parse from deployment.env if it exists in proto
    // For now, initialize empty
    if (envVars.value.length === 0) {
      envVars.value = [];
    }
  },
  { immediate: true }
);

const markDirty = () => {
  isDirty.value = true;
};

const addVariable = () => {
  envVars.value.push({ key: "", value: "" });
  markDirty();
};

const removeVariable = (idx: number) => {
  envVars.value.splice(idx, 1);
  markDirty();
};

const saveEnvVars = () => {
  const vars: Record<string, string> = {};
  for (const env of envVars.value) {
    if (env.key.trim()) {
      vars[env.key.trim()] = env.value || "";
    }
  }
  emit("save", vars);
  isDirty.value = false;
};
</script>

