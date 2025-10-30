<template>
  <OuiCardBody>
    <OuiStack gap="md">
      <OuiFlex justify="between" align="center">
        <OuiText as="h3" size="md" weight="semibold">Docker Compose</OuiText>
        <OuiFlex gap="sm">
          <OuiButton variant="ghost" size="sm" @click="validateCompose">
            Validate
          </OuiButton>
          <OuiButton
            size="sm"
            @click="saveCompose"
            :disabled="!isDirty || isLoading"
          >
            {{ isLoading ? "Saving..." : "Save & Apply" }}
          </OuiButton>
        </OuiFlex>
      </OuiFlex>

      <OuiText size="sm" color="secondary">
        Edit your docker-compose.yml configuration. Changes will be applied on the next deployment.
      </OuiText>

      <div v-if="isLoading && !composeYaml" class="flex justify-center py-8">
        <OuiText color="secondary">Loading compose file...</OuiText>
      </div>

      <div v-else class="relative">
        <textarea
          v-model="composeYaml"
          class="w-full h-96 p-4 bg-black text-green-400 font-mono text-sm rounded-xl border border-border-default focus:outline-none focus:ring-2 focus:ring-primary"
          placeholder="version: '3.8'&#10;services:&#10;  app:&#10;    image: nginx&#10;    ports:&#10;      - '80:80'"
          @input="markDirty"
        />
      </div>

      <OuiCard v-if="validationError" variant="danger">
        <OuiCardBody>
          <OuiText size="sm" color="danger">{{ validationError }}</OuiText>
        </OuiCardBody>
      </OuiCard>

      <OuiCard v-if="validationSuccess && !validationError" variant="success">
        <OuiCardBody>
          <OuiText size="sm" color="success">✓ Compose file is valid</OuiText>
        </OuiCardBody>
      </OuiCard>
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

const props = defineProps<Props>();

const emit = defineEmits<{
  save: [composeYaml: string];
}>();

const client = useConnectClient(DeploymentService);
const orgsStore = useOrganizationsStore();
const organizationId = computed(() => orgsStore.currentOrgId || "");
const composeYaml = ref("");
const isDirty = ref(false);
const isLoading = ref(false);
const validationError = ref("");
const validationSuccess = ref(false);

const loadCompose = async () => {
  isLoading.value = true;
  try {
    const res = await client.getDeploymentCompose({
      organizationId: organizationId.value,
      deploymentId: props.deployment.id,
    });
    composeYaml.value = res.composeYaml || "";
    if (!composeYaml.value) {
      // Generate default compose based on deployment
      composeYaml.value = generateDefaultCompose();
    }
    isDirty.value = false;
  } catch (error) {
    console.error("Failed to load compose:", error);
    composeYaml.value = generateDefaultCompose();
  } finally {
    isLoading.value = false;
  }
};

const generateDefaultCompose = () => {
  return `version: '3.8'

services:
  app:
    image: ${props.deployment.image || "nginx"}
    ports:
      - "${props.deployment.port || 8080}:${props.deployment.port || 8080}"
    environment:
      # Add your environment variables here
    volumes:
      # Add volume mounts here
`;
};

const markDirty = () => {
  isDirty.value = true;
  validationError.value = "";
  validationSuccess.value = false;
};

const validateCompose = () => {
  try {
    if (!composeYaml.value.trim()) {
      validationError.value = "Compose file cannot be empty";
      validationSuccess.value = false;
      return;
    }

    // Basic structure validation
    if (!composeYaml.value.includes("services:")) {
      validationError.value = "Missing 'services:' section";
      validationSuccess.value = false;
      return;
    }

    // TODO: Use proper YAML parser for full validation
    validationError.value = "";
    validationSuccess.value = true;
  } catch (error: any) {
    validationError.value = error.message || "Invalid YAML syntax";
    validationSuccess.value = false;
  }
};

const saveCompose = async () => {
  validateCompose();
  if (validationError.value) {
    return;
  }

  isLoading.value = true;
  try {
    const res = await client.updateDeploymentCompose({
      organizationId: organizationId.value,
      deploymentId: props.deployment.id,
      composeYaml: composeYaml.value,
    });
    
    if (res.validationError) {
      validationError.value = res.validationError;
      validationSuccess.value = false;
    } else {
      emit("save", composeYaml.value);
      isDirty.value = false;
      validationSuccess.value = true;
    }
  } catch (error) {
    console.error("Failed to save compose:", error);
    validationError.value = "Failed to save compose file. Please try again.";
    validationSuccess.value = false;
  } finally {
    isLoading.value = false;
  }
};

watch(() => props.deployment.id, () => {
  loadCompose();
}, { immediate: true });

onMounted(() => {
  loadCompose();
});
</script>
