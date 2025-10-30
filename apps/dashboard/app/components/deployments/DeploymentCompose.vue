<template>
  <OuiCardBody>
    <OuiStack gap="md">
      <OuiFlex justify="between" align="center">
        <OuiText as="h3" size="md" weight="semibold">Docker Compose</OuiText>
        <OuiFlex gap="sm">
          <OuiButton variant="ghost" size="sm" @click="validateCompose">
            Validate
          </OuiButton>
          <OuiButton size="sm" @click="saveCompose" :disabled="!isDirty">
            Save & Apply
          </OuiButton>
        </OuiFlex>
      </OuiFlex>

      <OuiText size="sm" color="secondary">
        Edit your docker-compose.yml configuration. Changes will be applied on save.
      </OuiText>

      <div class="relative">
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

      <OuiCard v-if="validationSuccess" variant="success">
        <OuiCardBody>
          <OuiText size="sm" color="success">✓ Compose file is valid</OuiText>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>
  </OuiCardBody>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import type { Deployment } from "@obiente/proto";

interface Props {
  deployment: Deployment;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  save: [composeYaml: string];
}>();

const composeYaml = ref("");
const isDirty = ref(false);
const validationError = ref("");
const validationSuccess = ref(false);

// Load existing compose if available
watch(
  () => props.deployment,
  (deployment) => {
    // TODO: Fetch compose from deployment config
    if (!composeYaml.value) {
      composeYaml.value = `version: '3.8'

services:
  app:
    image: ${deployment.image || "nginx"}
    ports:
      - "${deployment.port || 8080}:${deployment.port || 8080}"
    environment:
      # Add your environment variables here
    volumes:
      # Add volume mounts here
`;
    }
  },
  { immediate: true }
);

const markDirty = () => {
  isDirty.value = true;
  validationError.value = "";
  validationSuccess.value = false;
};

const validateCompose = () => {
  try {
    // Basic YAML validation (can be enhanced with yaml parser)
    if (!composeYaml.value.trim()) {
      validationError.value = "Compose file cannot be empty";
      validationSuccess.value = false;
      return;
    }

    // Check for basic structure
    if (!composeYaml.value.includes("services:")) {
      validationError.value = "Missing 'services:' section";
      validationSuccess.value = false;
      return;
    }

    // TODO: Use yaml parser for proper validation
    validationError.value = "";
    validationSuccess.value = true;
  } catch (error: any) {
    validationError.value = error.message || "Invalid YAML syntax";
    validationSuccess.value = false;
  }
};

const saveCompose = async () => {
  validateCompose();
  if (!validationError.value && validationSuccess.value) {
    emit("save", composeYaml.value);
    isDirty.value = false;
  }
};
</script>

