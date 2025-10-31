<template>
  <OuiCardBody>
    <OuiStack gap="md">
      <OuiFlex justify="between" align="center">
        <OuiStack gap="none">
          <OuiText as="h3" size="md" weight="semibold">Docker Compose</OuiText>
          <OuiText size="sm" color="secondary">
            Edit your docker-compose.yml configuration. Changes will be applied
            on the next deployment.
          </OuiText>
        </OuiStack>
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

      <div v-if="isLoading && !composeYaml" class="flex justify-center py-8">
        <OuiText color="secondary">Loading compose file...</OuiText>
      </div>

      <div v-else class="relative">
        <OuiFileEditor
          v-model="composeYaml"
          language="yaml"
          :read-only="false"
          height="600px"
          :minimap="{ enabled: true }"
          :folding="true"
          :format-on-paste="true"
          :format-on-type="true"
          :bracket-pair-colorization="{ enabled: true }"
          :validation-errors="validationErrors"
          container-class="w-full h-[600px] rounded-xl border border-border-default overflow-hidden"
          @save="saveCompose"
          @change="markDirty"
        />
      </div>

      <OuiCard v-if="validationError" variant="default" class="border-danger">
        <OuiCardBody>
          <OuiText size="sm" color="danger">{{ validationError }}</OuiText>
        </OuiCardBody>
      </OuiCard>

      <OuiCard
        v-if="validationSuccess && !validationError"
        variant="default"
        class="border-success"
      >
        <OuiCardBody>
          <OuiText size="sm" color="success">✓ Compose file is valid</OuiText>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>
  </OuiCardBody>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, computed, nextTick } from "vue";
  import { useConnectClient } from "~/lib/connect-client";
  import { DeploymentService } from "@obiente/proto";
  import type { Deployment } from "@obiente/proto";
  import { useOrganizationsStore } from "~/stores/organizations";
  import OuiFileEditor from "~/components/oui/FileEditor.vue";

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
  const validationErrors = ref<Array<{
    line: number;
    column: number;
    message: string;
    severity: "error" | "warning";
    startLine?: number;
    endLine?: number;
    startColumn?: number;
    endColumn?: number;
  }>>([]);

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
    validationErrors.value = [];
  };

  const validateCompose = async () => {
    validationError.value = "";
    validationErrors.value = [];
    validationSuccess.value = false;

    try {
      if (!composeYaml.value.trim()) {
        validationError.value = "Compose file cannot be empty";
        validationSuccess.value = false;
        return;
      }

      // Validate via API (which uses Docker Compose CLI)
      const res = await client.updateDeploymentCompose({
        organizationId: organizationId.value,
        deploymentId: props.deployment.id,
        composeYaml: composeYaml.value,
      });

      if (res.validationErrors && res.validationErrors.length > 0) {
        // Convert proto errors to editor format
        validationErrors.value = res.validationErrors.map((err) => ({
          line: err.line || 1,
          column: err.column || 1,
          message: err.message || "",
          severity: (err.severity || "error") as "error" | "warning",
          startLine: err.startLine || err.line || 1,
          endLine: err.endLine || err.line || 1,
          startColumn: err.startColumn || err.column || 1,
          endColumn: err.endColumn || err.column || 1,
        }));
        
        validationError.value = res.validationError || "Validation failed";
        validationSuccess.value = false;
      } else if (res.validationError) {
        // Legacy single error message
        validationError.value = res.validationError;
        validationSuccess.value = false;
      } else {
        validationError.value = "";
        validationErrors.value = [];
        validationSuccess.value = true;
      }
    } catch (error: any) {
      validationError.value = error.message || "Invalid YAML syntax";
      validationSuccess.value = false;
      validationErrors.value = [];
    }
  };

  const saveCompose = async () => {
    // Validate first (API validates on save anyway, but good to show errors immediately)
    await validateCompose();
    if (validationError.value || (validationErrors.value.length > 0 && validationErrors.value.some(e => e.severity === "error"))) {
      return; // Don't save if there are errors
    }

    isLoading.value = true;
    validationError.value = "";
    validationErrors.value = [];
    validationSuccess.value = false;

    try {
      const res = await client.updateDeploymentCompose({
        organizationId: organizationId.value,
        deploymentId: props.deployment.id,
        composeYaml: composeYaml.value,
      });

      if (res.validationErrors && res.validationErrors.length > 0) {
        // Convert proto errors to editor format
        validationErrors.value = res.validationErrors.map((err) => ({
          line: err.line || 1,
          column: err.column || 1,
          message: err.message || "",
          severity: (err.severity || "error") as "error" | "warning",
          startLine: err.startLine || err.line || 1,
          endLine: err.endLine || err.line || 1,
          startColumn: err.startColumn || err.column || 1,
          endColumn: err.endColumn || err.column || 1,
        }));
        
        validationError.value = res.validationError || "Validation failed";
        validationSuccess.value = false;
        return; // Don't save if there are errors
      } else if (res.validationError) {
        // Legacy single error message
        validationError.value = res.validationError;
        validationSuccess.value = false;
        return;
      }

      // Validation passed, save successful
      emit("save", composeYaml.value);
      isDirty.value = false;
      validationSuccess.value = true;
      validationErrors.value = [];
    } catch (error) {
      console.error("Failed to save compose:", error);
      validationError.value = "Failed to save compose file. Please try again.";
      validationSuccess.value = false;
      validationErrors.value = [];
    } finally {
      isLoading.value = false;
    }
  };

  watch(
    () => props.deployment.id,
    () => {
      loadCompose();
    },
    { immediate: true }
  );

  onMounted(async () => {
    await nextTick();
    await loadCompose();
  });
</script>
