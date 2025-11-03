<template>
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
          :container-class="'w-full h-[600px] rounded-xl border border-border-default overflow-hidden'"
          @save="saveCompose"
          @change="markDirty"
        />
      </div>

      <!-- Validation Error Card -->
      <OuiCard 
        v-if="hasErrors"
        variant="default" 
        class="border-danger"
      >
        <OuiCardBody>
          <OuiStack gap="sm">
            <OuiText size="sm" color="danger" weight="semibold">
              Validation Failed
            </OuiText>
            <OuiText v-if="validationError" size="sm" color="danger">
              {{ validationError }}
            </OuiText>
            <div v-if="errorList.length > 0" class="mt-1">
              <OuiStack gap="xs">
                <template
                  v-for="(error, index) in errorList"
                  :key="`error-${error.line}-${error.column}-${index}`"
                >
                  <div>
                    <OuiText size="sm" color="danger">
                      Line {{ error.line }}: {{ error.message }}
                    </OuiText>
                  </div>
                </template>
              </OuiStack>
            </div>
            <!-- Show warnings even when there are errors -->
            <div v-if="warningList.length > 0" class="mt-2 pt-2 border-t border-border-default">
              <OuiText size="xs" color="secondary" weight="medium" class="mb-1">
                Also {{ warningList.length }} warning(s):
              </OuiText>
              <OuiStack gap="xs">
                <template
                  v-for="(warning, index) in warningList"
                  :key="`warning-${warning.line}-${warning.column}-${index}`"
                >
                  <div>
                    <OuiText size="sm" color="warning">
                      Line {{ warning.line }}: {{ warning.message }}
                    </OuiText>
                  </div>
                </template>
              </OuiStack>
            </div>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Validation Success/Warning Card -->
      <OuiCard
        v-else-if="hasSuccessOrWarnings"
        variant="default"
        :class="warningList.length > 0 ? 'border-warning' : 'border-success'"
      >
        <OuiCardBody>
          <OuiStack gap="sm">
            <OuiText 
              size="sm" 
              :color="warningList.length > 0 ? 'warning' : 'success'"
              weight="semibold"
            >
              ✓ Compose file is valid
              <span v-if="warningList.length > 0">
                ({{ warningList.length }} warning(s))
              </span>
            </OuiText>
            <div v-if="warningList.length > 0" class="mt-1">
              <OuiStack gap="xs">
                <template
                  v-for="(warning, index) in warningList"
                  :key="`warning-success-${warning.line}-${warning.column}-${index}`"
                >
                  <div>
                    <OuiText size="sm" color="warning">
                      Line {{ warning.line }}: {{ warning.message }}
                    </OuiText>
                  </div>
                </template>
              </OuiStack>
            </div>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>
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

  // Computed properties for cleaner template logic
  const errorList = computed(() => {
    return validationErrors.value.filter(e => e.severity === "error");
  });

  const warningList = computed(() => {
    return validationErrors.value.filter(e => e.severity === "warning");
  });

  const hasErrors = computed(() => {
    return validationError.value !== "" || errorList.value.length > 0;
  });

  const hasSuccessOrWarnings = computed(() => {
    if (hasErrors.value) return false;
    return validationSuccess.value || warningList.value.length > 0;
  });

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
    // Clear validation state when user edits (but do it safely to avoid DOM issues)
    nextTick(() => {
      validationError.value = "";
      validationSuccess.value = false;
      validationErrors.value = [];
    });
  };

  const validateCompose = async () => {
    // Clear previous validation state safely
    validationError.value = "";
    validationErrors.value = [];
    validationSuccess.value = false;
    await nextTick(); // Wait for DOM to update before showing new validation

    try {
      if (!composeYaml.value.trim()) {
        validationError.value = "Compose file cannot be empty";
        validationSuccess.value = false;
        return;
      }

      // Validate via API (validation only, no save)
      const res = await client.validateDeploymentCompose({
        organizationId: organizationId.value,
        deploymentId: props.deployment.id,
        composeYaml: composeYaml.value,
      });

      if (res.validationErrors && res.validationErrors.length > 0) {
        // Convert proto errors to editor format
        const newErrors = res.validationErrors.map((err) => ({
          line: err.line || 1,
          column: err.column || 1,
          message: err.message || "",
          severity: (err.severity || "error") as "error" | "warning",
          startLine: err.startLine || err.line || 1,
          endLine: err.endLine || err.line || 1,
          startColumn: err.startColumn || err.column || 1,
          endColumn: err.endColumn || err.column || 1,
        }));
        
        // Update validationErrors atomically to avoid DOM issues
        validationErrors.value = newErrors;
        
        const errorCount = newErrors.filter(e => e.severity === "error").length;
        const warningCount = newErrors.filter(e => e.severity === "warning").length;
        
        if (errorCount > 0) {
          validationError.value = res.validationError || `Validation failed: ${errorCount} error(s)${warningCount > 0 ? `, ${warningCount} warning(s)` : ""}`;
          validationSuccess.value = false;
        } else if (warningCount > 0) {
          // Only warnings, no errors
          validationError.value = "";
          validationSuccess.value = true;
        } else {
          validationError.value = res.validationError || "Validation failed";
          validationSuccess.value = false;
        }
      } else if (res.validationError) {
        // Legacy single error message
        validationError.value = res.validationError;
        validationSuccess.value = false;
        validationErrors.value = [];
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
    // Don't clear validation state here - keep it so users can see validation results

    try {
      const res = await client.updateDeploymentCompose({
        organizationId: organizationId.value,
        deploymentId: props.deployment.id,
        composeYaml: composeYaml.value,
      });

      if (res.validationErrors && res.validationErrors.length > 0) {
        // Convert proto errors to editor format
        const newErrors = res.validationErrors.map((err) => ({
          line: err.line || 1,
          column: err.column || 1,
          message: err.message || "",
          severity: (err.severity || "error") as "error" | "warning",
          startLine: err.startLine || err.line || 1,
          endLine: err.endLine || err.line || 1,
          startColumn: err.startColumn || err.column || 1,
          endColumn: err.endColumn || err.column || 1,
        }));
        
        // Update validationErrors atomically to avoid DOM issues
        validationErrors.value = newErrors;
        
        const errorCount = newErrors.filter(e => e.severity === "error").length;
        const warningCount = newErrors.filter(e => e.severity === "warning").length;
        
        if (errorCount > 0) {
          validationError.value = res.validationError || "Validation failed";
          validationSuccess.value = false;
          return; // Don't save if there are errors
        }
        // No errors - show success card (will show warnings if any)
        validationError.value = "";
        validationSuccess.value = true;
      } else if (res.validationError) {
        // Legacy single error message
        validationError.value = res.validationError;
        validationSuccess.value = false;
        return;
      }

      // Validation passed, save successful
      emit("save", composeYaml.value);
      isDirty.value = false;
      // Keep validation success state and any warnings
      validationSuccess.value = true;
      // Only clear errors, keep warnings
      validationErrors.value = validationErrors.value.filter(e => e.severity !== "error");
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
