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

      <div v-if="isLoading" class="flex justify-center py-8">
        <OuiText color="secondary">Loading environment variables...</OuiText>
      </div>

      <OuiStack v-else-if="envVars.length === 0" gap="sm">
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
        <OuiButton @click="saveEnvVars" :disabled="!isDirty || isLoading" size="sm">
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

const client = useConnectClient(DeploymentService);
const orgsStore = useOrganizationsStore();
const organizationId = computed(() => orgsStore.currentOrgId || "");
const envVars = ref<EnvVar[]>([]);
const isDirty = ref(false);
const isLoading = ref(false);

const loadEnvVars = async () => {
  isLoading.value = true;
  try {
    const res = await client.getDeploymentEnvVars({
      organizationId: organizationId.value,
      deploymentId: props.deployment.id,
    });
    const vars = res.envVars || {};
    envVars.value = Object.entries(vars).map(([key, value]) => ({
      key,
      value: value || "",
    }));
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

const removeVariable = (idx: number) => {
  envVars.value.splice(idx, 1);
  markDirty();
};

const saveEnvVars = async () => {
  const vars: Record<string, string> = {};
  for (const env of envVars.value) {
    if (env.key.trim()) {
      vars[env.key.trim()] = env.value || "";
    }
  }

  isLoading.value = true;
  try {
    await client.updateDeploymentEnvVars({
      organizationId: organizationId.value,
      deploymentId: props.deployment.id,
      envVars: vars,
    });
    emit("save", vars);
    isDirty.value = false;
  } catch (error) {
    console.error("Failed to save env vars:", error);
    alert("Failed to save environment variables. Please try again.");
  } finally {
    isLoading.value = false;
  }
};

watch(() => props.deployment.id, () => {
  loadEnvVars();
}, { immediate: true });

onMounted(() => {
  loadEnvVars();
});
</script>
