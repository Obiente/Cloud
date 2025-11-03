<template>
  <OuiStack gap="md">
    <OuiFlex justify="between" align="center">
      <OuiStack gap="none">
        <OuiText as="h3" size="md" weight="semibold">Routing & Domains</OuiText>
        <OuiText size="sm" color="secondary">
          Configure how traffic is routed to your deployment. You can route
          multiple services on different ports to different domains.
        </OuiText>
      </OuiStack>
      <OuiButton size="sm" @click="addRule">
        <PlusIcon class="h-4 w-4 mr-2" />
        Add Rule
      </OuiButton>
    </OuiFlex>

    <OuiFlex v-if="isLoading" justify="center" class="py-8">
      <OuiText color="secondary">Loading routing rules...</OuiText>
    </OuiFlex>

    <OuiFlex
      v-else-if="rules.length === 0"
      direction="col"
      align="center"
      justify="center"
      class="py-12"
    >
      <OuiStack gap="md" align="center">
        <OuiText size="sm" color="secondary">
          No routing rules configured. Add your first rule to get started.
        </OuiText>
        <OuiButton size="sm" @click="addRule">Add First Rule</OuiButton>
      </OuiStack>
    </OuiFlex>

    <OuiStack v-else gap="md">
      <OuiCard
        v-for="(rule, index) in rules"
        :key="rule.id || index"
        variant="outline"
        class="border-default"
      >
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiFlex justify="between" align="start">
              <OuiText size="sm" weight="semibold"
                >Rule {{ index + 1 }}</OuiText
              >
              <OuiButton
                variant="ghost"
                size="sm"
                color="danger"
                @click="removeRule(index)"
              >
                <TrashIcon class="h-4 w-4" />
              </OuiButton>
            </OuiFlex>

            <OuiGrid cols="1" :cols-md="2" gap="md">
              <OuiInput
                v-model="rule.domain"
                label="Domain"
                placeholder="example.com"
                @update:model-value="markDirty"
              />
              <OuiSelect
                v-model="rule.serviceName"
                :items="serviceNameOptions"
                label="Service Name"
                @update:model-value="markDirty"
              />
            </OuiGrid>

            <OuiGrid cols="1" :cols-md="2" gap="md">
              <OuiInput
                v-model="rule.targetPortStr"
                type="number"
                label="Target Port"
                placeholder="80"
                @update:model-value="
                  (val) => {
                    rule.targetPort = parseInt(val) || 80;
                    rule.targetPortStr = val;
                    markDirty();
                  }
                "
              />
              <OuiInput
                v-model="rule.pathPrefix"
                label="Path Prefix (optional)"
                placeholder="/api"
                @update:model-value="markDirty"
              />
            </OuiGrid>

            <OuiGrid cols="1" :cols-md="2" gap="md">
              <OuiSelect
                v-model="rule.protocol"
                :items="protocolOptions"
                label="Protocol"
                @update:model-value="
                  (val) => {
                    rule.protocol = val;
                    // Auto-sync SSL with protocol
                    if (val === 'http') {
                      rule.sslEnabled = false;
                    } else if (val === 'https') {
                      rule.sslEnabled = true;
                    }
                    markDirty();
                  }
                "
              />
              <OuiSwitch
                v-model="rule.sslEnabled"
                label="SSL Enabled"
                :disabled="rule.protocol === 'http'"
                @update:checked="markDirty"
              />
            </OuiGrid>

            <OuiSelect
              v-if="rule.sslEnabled"
              v-model="rule.sslCertResolver"
              :items="sslResolverOptions"
              label="SSL Certificate Resolver"
              @update:model-value="markDirty"
            />
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>

    <OuiFlex v-if="!isLoading" justify="end">
      <OuiButton @click="saveRules" :disabled="!isDirty || isLoading" size="sm">
        {{ isLoading ? "Saving..." : "Save Routing Rules" }}
      </OuiButton>
    </OuiFlex>
  </OuiStack>
</template>

<script setup lang="ts">
  import { ref, onMounted, computed, watch } from "vue";
  import { PlusIcon, TrashIcon } from "@heroicons/vue/24/outline";
  import { useConnectClient } from "~/lib/connect-client";
  import { DeploymentService } from "@obiente/proto";
  import type { Deployment, RoutingRule } from "@obiente/proto";
  import { useOrganizationsStore } from "~/stores/organizations";
  import { useDialog } from "~/composables/useDialog";

  interface Props {
    deployment: Deployment;
  }

  interface LocalRule {
    id?: string;
    domain: string;
    serviceName: string;
    pathPrefix: string;
    targetPort: number;
    targetPortStr: string;
    protocol: string;
    sslEnabled: boolean;
    sslCertResolver: string;
  }

  const props = defineProps<Props>();

  const client = useConnectClient(DeploymentService);
  const orgsStore = useOrganizationsStore();
  const organizationId = computed(() => orgsStore.currentOrgId || "");
  const rules = ref<LocalRule[]>([]);
  const isDirty = ref(false);
  const isLoading = ref(false);
  const { showAlert } = useDialog();

  const protocolOptions = [
    { label: "HTTP", value: "http" },
    { label: "HTTPS", value: "https" },
    { label: "gRPC", value: "grpc" },
  ];

  const serviceNameOptions = ref<Array<{ label: string; value: string }>>([
    { label: "Default", value: "default" },
  ]);

  const sslResolverOptions = [
    { label: "Let's Encrypt", value: "letsencrypt" },
    { label: "Internal (Handled by App)", value: "internal" },
  ];

  const loadServiceNames = async () => {
    try {
      const res = await client.getDeploymentServiceNames({
        organizationId: organizationId.value,
        deploymentId: props.deployment.id,
      });

      // Update service name options from API
      const serviceNames = res.serviceNames || ["default"];
      serviceNameOptions.value = serviceNames.map((name) => ({
        label: name.charAt(0).toUpperCase() + name.slice(1),
        value: name,
      }));
    } catch (error) {
      console.error("Failed to load service names:", error);
      // Keep default options on error
    }
  };

  const loadRules = async () => {
    isLoading.value = true;
    try {
      // Load service names first
      await loadServiceNames();

      const res = await client.getDeploymentRoutings({
        organizationId: organizationId.value,
        deploymentId: props.deployment.id,
      });

      rules.value = (res.rules || []).map((rule) => {
        const protocol = rule.protocol || "http";
        // Sync SSL with protocol - HTTP should not have SSL enabled
        const sslEnabled =
          protocol === "http" ? false : rule.sslEnabled ?? protocol === "https";

        return {
          id: rule.id,
          domain: rule.domain || "",
          serviceName: rule.serviceName || "default",
          pathPrefix: rule.pathPrefix || "",
          targetPort: rule.targetPort || 80,
          targetPortStr: String(rule.targetPort || 80),
          protocol: protocol,
          sslEnabled: sslEnabled,
          sslCertResolver: rule.sslCertResolver || "letsencrypt",
        };
      });

      // If no rules exist, add a default one
      if (rules.value.length === 0 && props.deployment.domain) {
        const defaultPort = props.deployment.port || 80;
        rules.value.push({
          domain: props.deployment.domain,
          serviceName: "default",
          pathPrefix: "",
          targetPort: defaultPort,
          targetPortStr: String(defaultPort),
          protocol: "http",
          sslEnabled: false, // HTTP protocol defaults to no SSL
          sslCertResolver: "letsencrypt",
        });
      }

      isDirty.value = false;
    } catch (error) {
      console.error("Failed to load routing rules:", error);
      await showAlert({
        title: "Error",
        message: "Failed to load routing rules. Please try again.",
      });
    } finally {
      isLoading.value = false;
    }
  };

  const addRule = () => {
    const defaultPort = props.deployment.port || 80;
    rules.value.push({
      domain: "",
      serviceName: "default",
      pathPrefix: "",
      targetPort: defaultPort,
      targetPortStr: String(defaultPort),
      protocol: "http",
      sslEnabled: false, // HTTP protocol defaults to no SSL
      sslCertResolver: "letsencrypt",
    });
    markDirty();
  };

  const removeRule = async (index: number) => {
    const { showConfirm } = useDialog();
    const confirmed = await showConfirm({
      title: "Remove Rule",
      message: "Are you sure you want to remove this routing rule?",
    });

    if (confirmed) {
      rules.value.splice(index, 1);
      markDirty();
    }
  };

  const markDirty = () => {
    isDirty.value = true;
  };

  const saveRules = async () => {
    if (isLoading.value) return;

    // Validate rules
    for (const rule of rules.value) {
      if (!rule.domain.trim()) {
        await showAlert({
          title: "Validation Error",
          message: "All rules must have a domain specified.",
        });
        return;
      }
      if (!rule.targetPort || rule.targetPort < 1 || rule.targetPort > 65535) {
        await showAlert({
          title: "Validation Error",
          message: "All rules must have a valid port (1-65535).",
        });
        return;
      }
    }

    isLoading.value = true;
    try {
      const protoRules = rules.value.map((rule) => {
        const protocol = rule.protocol || "http";
        // Ensure SSL is disabled for HTTP protocol, enabled for HTTPS
        const sslEnabled =
          protocol === "http"
            ? false
            : protocol === "https"
            ? true
            : rule.sslEnabled;

        const protoRule = {
          id: rule.id || "",
          deploymentId: props.deployment.id,
          domain: rule.domain.trim(),
          serviceName: rule.serviceName.trim() || "default",
          pathPrefix: rule.pathPrefix.trim(),
          targetPort: rule.targetPort,
          protocol: protocol,
          sslEnabled: sslEnabled,
          sslCertResolver: rule.sslCertResolver || "",
        };
        return protoRule as RoutingRule;
      });

      const res = await client.updateDeploymentRoutings({
        organizationId: organizationId.value,
        deploymentId: props.deployment.id,
        rules: protoRules,
      });

      // Update local rules with returned IDs
      rules.value = res.rules.map((rule) => ({
        id: rule.id,
        domain: rule.domain,
        serviceName: rule.serviceName,
        pathPrefix: rule.pathPrefix,
        targetPort: rule.targetPort,
        targetPortStr: String(rule.targetPort),
        protocol: rule.protocol,
        sslEnabled: rule.sslEnabled,
        sslCertResolver: rule.sslCertResolver,
      }));

      isDirty.value = false;

      // Reload service names in case compose was updated
      await loadServiceNames();

      await showAlert({
        title: "Success",
        message:
          "Routing rules saved successfully. Note: Changes will take effect on the next deployment.",
      });
    } catch (error: any) {
      console.error("Failed to save routing rules:", error);
      await showAlert({
        title: "Error",
        message:
          error.message || "Failed to save routing rules. Please try again.",
      });
    } finally {
      isLoading.value = false;
    }
  };

  // Watch for deployment changes to refresh service names
  watch(
    () => props.deployment.id,
    () => {
      loadRules();
    }
  );

  onMounted(() => {
    loadRules();
  });
</script>
