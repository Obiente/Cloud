<template>
  <OuiCardBody>
    <OuiStack gap="xl">
      <!-- Header -->
      <OuiFlex justify="between" align="center" wrap="wrap" gap="md">
        <OuiStack gap="xs">
          <OuiText as="h3" size="lg" weight="semibold">Services</OuiText>
          <OuiText size="sm" color="secondary">
            Manage individual services and containers for this compose deployment
          </OuiText>
        </OuiStack>
        <OuiButton
          variant="ghost"
          size="sm"
          @click="refresh"
          :disabled="isLoading"
          class="gap-2"
        >
          <ArrowPathIcon
            class="h-4 w-4"
            :class="{ 'animate-spin': isLoading }"
          />
          <OuiText as="span" size="xs" weight="medium">Refresh</OuiText>
        </OuiButton>
      </OuiFlex>

      <!-- Loading State -->
      <div v-if="isLoading && services.length === 0" class="flex justify-center py-12">
        <OuiStack gap="sm" align="center">
          <ArrowPathIcon class="h-6 w-6 text-secondary animate-spin" />
          <OuiText size="sm" color="secondary">Loading services...</OuiText>
        </OuiStack>
      </div>

      <!-- No Services -->
      <OuiCard v-if="!isLoading && services.length === 0" variant="outline">
        <OuiCardBody>
          <OuiStack gap="sm" align="center" class="py-8">
            <CubeIcon class="h-12 w-12 text-secondary" />
            <OuiText size="sm" color="secondary">
              No services found. This deployment may not be a compose deployment.
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Services List -->
      <OuiStack v-else gap="lg">
        <div
          v-for="service in services"
          :key="service.name"
          class="border border-border-default rounded-xl overflow-hidden"
        >
          <OuiCard variant="default">
            <!-- Service Header -->
            <OuiCardBody class="p-0">
              <div class="p-6 border-b border-border-default">
                <OuiFlex justify="between" align="start" wrap="wrap" gap="md">
                  <OuiStack gap="sm" class="flex-1 min-w-0">
                    <OuiFlex align="center" gap="md">
                      <OuiBox
                        p="xs"
                        rounded="lg"
                        bg="accent-primary"
                        class="bg-primary/10 ring-1 ring-primary/20 shrink-0"
                      >
                        <CubeIcon class="w-5 h-5 text-primary" />
                      </OuiBox>
                      <OuiText as="h4" size="md" weight="semibold" truncate>
                        {{ service.name }}
                      </OuiText>
                    </OuiFlex>
                    <OuiFlex align="center" gap="md" wrap="wrap">
                      <OuiBadge
                        :variant="getServiceStatusVariant(service.status)"
                        size="sm"
                      >
                        <span
                          class="inline-flex h-1.5 w-1.5 rounded-full mr-1.5"
                          :class="getServiceStatusDotClass(service.status)"
                        />
                        <OuiText
                          as="span"
                          size="xs"
                          weight="medium"
                          transform="uppercase"
                        >
                          {{ service.status }}
                        </OuiText>
                      </OuiBadge>
                      <OuiText size="xs" color="secondary">
                        {{ service.containerCount }}
                        {{
                          service.containerCount === 1
                            ? "container"
                            : "containers"
                        }}
                      </OuiText>
                      <OuiText
                        v-if="service.runningCount > 0"
                        size="xs"
                        color="secondary"
                      >
                        {{ service.runningCount }} running
                      </OuiText>
                    </OuiFlex>
                  </OuiStack>

                  <!-- Service Actions -->
                  <OuiFlex gap="sm" wrap="wrap" class="shrink-0">
                    <OuiButton
                      variant="ghost"
                      size="sm"
                      @click="openTerminal(service.name)"
                      class="gap-2"
                      title="Open Terminal"
                    >
                      <CommandLineIcon class="h-4 w-4" />
                      <OuiText as="span" size="xs" weight="medium" class="hidden sm:inline">Terminal</OuiText>
                    </OuiButton>
                    <OuiButton
                      variant="ghost"
                      size="sm"
                      @click="openFiles(service.name)"
                      class="gap-2"
                      title="Open Filesystem"
                    >
                      <FolderIcon class="h-4 w-4" />
                      <OuiText as="span" size="xs" weight="medium" class="hidden sm:inline">Files</OuiText>
                    </OuiButton>
                    <OuiButton
                      variant="ghost"
                      size="sm"
                      @click="toggleServiceRouting(service.name)"
                      class="gap-2"
                      title="Configure Routing"
                    >
                      <GlobeAltIcon class="h-4 w-4" />
                      <OuiText as="span" size="xs" weight="medium" class="hidden sm:inline">Routing</OuiText>
                    </OuiButton>
                    <OuiButton
                      v-if="service.hasRunning"
                      variant="ghost"
                      color="warning"
                      size="sm"
                      @click="stopService(service.name)"
                      :disabled="isProcessingService(service.name)"
                      class="gap-2"
                    >
                      <StopIcon class="h-4 w-4" />
                      <OuiText as="span" size="xs" weight="medium" class="hidden sm:inline">Stop All</OuiText>
                    </OuiButton>
                    <OuiButton
                      v-else-if="service.hasStopped"
                      variant="ghost"
                      color="success"
                      size="sm"
                      @click="startService(service.name)"
                      :disabled="isProcessingService(service.name)"
                      class="gap-2"
                    >
                      <PlayIcon class="h-4 w-4" />
                      <OuiText as="span" size="xs" weight="medium" class="hidden sm:inline">Start All</OuiText>
                    </OuiButton>
                    <OuiButton
                      v-if="service.containerCount > 0"
                      variant="ghost"
                      size="sm"
                      @click="restartService(service.name)"
                      :disabled="isProcessingService(service.name)"
                      class="gap-2"
                    >
                      <ArrowPathIcon
                        class="h-4 w-4"
                        :class="{
                          'animate-spin': isProcessingService(service.name),
                        }"
                      />
                      <OuiText as="span" size="xs" weight="medium" class="hidden sm:inline"
                        >Restart All</OuiText
                      >
                    </OuiButton>
                  </OuiFlex>
                </OuiFlex>
              </div>

              <!-- Containers List -->
              <div v-if="service.containers.length > 0" class="divide-y divide-border-default">
                <div
                  v-for="container in service.containers"
                  :key="container.containerId"
                  class="p-4 hover:bg-surface-subtle/50 transition-colors"
                >
                  <OuiFlex justify="between" align="center" wrap="wrap" gap="md">
                    <OuiStack gap="xs" class="flex-1 min-w-0">
                      <OuiFlex align="center" gap="sm">
                        <OuiBadge
                          :variant="getContainerStatusVariant(container.status)"
                          size="sm"
                        >
                          <span
                            class="inline-flex h-1 w-1 rounded-full mr-1"
                            :class="getContainerStatusDotClass(container.status)"
                          />
                          <OuiText as="span" size="xs" weight="medium">
                            {{ container.status }}
                          </OuiText>
                        </OuiBadge>
                        <OuiText size="xs" color="secondary" class="font-mono">
                          {{ container.containerId.substring(0, 12) }}
                        </OuiText>
                        <OuiText
                          v-if="container.port"
                          size="xs"
                          color="secondary"
                        >
                          Port {{ container.port }}
                        </OuiText>
                      </OuiFlex>
                      <OuiText size="xs" color="secondary">
                        Updated
                        <OuiRelativeTime
                          v-if="container.updatedAt"
                          :value="date(container.updatedAt)"
                          :style="'short'"
                        />
                      </OuiText>
                    </OuiStack>

                    <!-- Container Actions -->
                    <OuiFlex gap="sm" wrap="wrap" class="shrink-0">
                      <OuiButton
                        v-if="container.status === 'running'"
                        variant="ghost"
                        color="warning"
                        size="xs"
                        @click="stopContainer(container.containerId)"
                        :disabled="isProcessingContainer(container.containerId)"
                      >
                        <StopIcon class="h-3 w-3" />
                      </OuiButton>
                      <OuiButton
                        v-else-if="
                          container.status === 'stopped' ||
                          container.status === 'exited'
                        "
                        variant="ghost"
                        color="success"
                        size="xs"
                        @click="startContainer(container.containerId)"
                        :disabled="isProcessingContainer(container.containerId)"
                      >
                        <PlayIcon class="h-3 w-3" />
                      </OuiButton>
                      <OuiButton
                        variant="ghost"
                        size="xs"
                        @click="restartContainer(container.containerId)"
                        :disabled="isProcessingContainer(container.containerId)"
                      >
                        <ArrowPathIcon
                          class="h-3 w-3"
                          :class="{
                            'animate-spin': isProcessingContainer(
                              container.containerId
                            ),
                          }"
                        />
                      </OuiButton>
                    </OuiFlex>
                  </OuiFlex>
                </div>
              </div>

              <!-- No Containers -->
              <div
                v-else
                class="p-8 text-center"
              >
                <OuiText size="sm" color="secondary">
                  No containers found for this service
                </OuiText>
              </div>

              <!-- Routing Configuration Section -->
              <div
                v-if="expandedServiceRouting.has(service.name)"
                class="border-t border-border-default"
              >
                <div class="p-6">
                  <OuiStack gap="md">
                    <OuiFlex justify="between" align="center">
                      <OuiText size="sm" weight="semibold">Routing Configuration</OuiText>
                      <OuiButton
                        variant="ghost"
                        size="xs"
                        @click="addRoutingRule(service.name)"
                      >
                        <PlusIcon class="h-3 w-3 mr-1" />
                        Add Rule
                      </OuiButton>
                    </OuiFlex>

                    <div v-if="getServiceRoutingRules(service.name).length === 0">
                      <OuiText size="xs" color="secondary">
                        No routing rules configured for this service. Add a rule to route traffic to this service.
                      </OuiText>
                    </div>

                    <OuiStack v-else gap="sm">
                      <OuiCard
                        v-for="(rule, index) in getServiceRoutingRules(service.name)"
                        :key="rule.id || index"
                        variant="outline"
                        class="border-default"
                      >
                        <OuiCardBody>
                          <OuiStack gap="sm">
                            <OuiFlex justify="between" align="start">
                              <OuiText size="xs" weight="medium">Rule {{ index + 1 }}</OuiText>
                              <OuiButton
                                variant="ghost"
                                size="xs"
                                color="danger"
                                @click="removeRoutingRule(rule)"
                              >
                                <TrashIcon class="h-3 w-3" />
                              </OuiButton>
                            </OuiFlex>

                            <OuiGrid :cols="{ sm: 1, md: 2 }" gap="sm">
                              <OuiInput
                                v-model="rule.domain"
                                label="Domain"
                                placeholder="example.com"
                                size="sm"
                                @update:model-value="markRoutingDirty"
                              />
                              <OuiInput
                                v-model="rule.targetPortStr"
                                type="number"
                                label="Target Port"
                                placeholder="80"
                                size="sm"
                                @update:model-value="(val) => { rule.targetPort = parseInt(val) || 80; rule.targetPortStr = val; markRoutingDirty(); }"
                              />
                            </OuiGrid>

                            <OuiGrid :cols="{ sm: 1, md: 2 }" gap="sm">
                              <OuiInput
                                v-model="rule.pathPrefix"
                                label="Path Prefix (optional)"
                                placeholder="/api"
                                size="sm"
                                @update:model-value="markRoutingDirty"
                              />
                              <OuiSelect
                                v-model="rule.protocol"
                                :items="protocolOptions"
                                label="Protocol"
                                size="sm"
                                @update:model-value="(val) => { 
                                  rule.protocol = val; 
                                  // Auto-sync SSL with protocol
                                  if (val === 'http') {
                                    rule.sslEnabled = false;
                                  } else if (val === 'https') {
                                    rule.sslEnabled = true;
                                  }
                                  markRoutingDirty();
                                }"
                              />
                            </OuiGrid>

                            <OuiFlex gap="sm">
                              <OuiSwitch
                                v-model="rule.sslEnabled"
                                label="SSL Enabled"
                                size="sm"
                                :disabled="rule.protocol === 'http'"
                                @update:checked="markRoutingDirty"
                              />
                              <OuiSelect
                                v-if="rule.sslEnabled"
                                v-model="rule.sslCertResolver"
                                :items="sslResolverOptions"
                                label="SSL Resolver"
                                size="sm"
                                @update:model-value="markRoutingDirty"
                              />
                            </OuiFlex>
                          </OuiStack>
                        </OuiCardBody>
                      </OuiCard>
                    </OuiStack>

                    <OuiFlex justify="end">
                      <OuiButton
                        @click="saveRoutingRules"
                        :disabled="!isRoutingDirty || isSavingRouting"
                        size="sm"
                      >
                        {{ isSavingRouting ? "Saving..." : "Save Routing Rules" }}
                      </OuiButton>
                    </OuiFlex>
                  </OuiStack>
                </div>
              </div>
            </OuiCardBody>
          </OuiCard>
        </div>
      </OuiStack>
    </OuiStack>
  </OuiCardBody>
</template>

<script setup lang="ts">
  import { ref, computed, onMounted, watch } from "vue";
  import {
    PlayIcon,
    StopIcon,
    ArrowPathIcon,
    CubeIcon,
    GlobeAltIcon,
    CommandLineIcon,
    FolderIcon,
    PlusIcon,
    TrashIcon,
    ChevronDownIcon,
    ChevronUpIcon,
  } from "@heroicons/vue/24/outline";
  import {
    DeploymentService,
    type Deployment,
    type DeploymentContainer,
  } from "@obiente/proto";
  import { date } from "@obiente/proto/utils";
  import { useConnectClient } from "~/lib/connect-client";
  import { useOrganizationsStore } from "~/stores/organizations";
  import { useDialog } from "~/composables/useDialog";
  import { useRouter } from "vue-router";
  import type { RoutingRule } from "@obiente/proto";
  import OuiRelativeTime from "~/components/oui/RelativeTime.vue";

  interface ServiceInfo {
    name: string;
    containers: DeploymentContainer[];
    containerCount: number;
    runningCount: number;
    stoppedCount: number;
    hasRunning: boolean;
    hasStopped: boolean;
    status: string;
  }

  const props = defineProps<{
    deployment: Deployment;
    deploymentId: string;
    organizationId: string;
  }>();

  const { showAlert } = useDialog();
  const orgsStore = useOrganizationsStore();
  const client = useConnectClient(DeploymentService);

  const containers = ref<DeploymentContainer[]>([]);
  const serviceNames = ref<string[]>([]);
  const routingRules = ref<RoutingRule[]>([]);
  const isLoading = ref(false);
  const isLoadingRouting = ref(false);
  const isSavingRouting = ref(false);
  const processingServices = ref<Set<string>>(new Set());
  const processingContainers = ref<Set<string>>(new Set());
  const expandedServiceRouting = ref<Set<string>>(new Set());
  const router = useRouter();

  // Group containers by service name
  const services = computed<ServiceInfo[]>(() => {
    const serviceMap = new Map<string, DeploymentContainer[]>();

    // Initialize all known service names
    serviceNames.value.forEach((name) => {
      serviceMap.set(name, []);
    });

    // Group containers by service name
    containers.value.forEach((container) => {
      const serviceName = container.serviceName || "default";
      if (!serviceMap.has(serviceName)) {
        serviceMap.set(serviceName, []);
      }
      serviceMap.get(serviceName)!.push(container);
    });

    // Convert to array of ServiceInfo
    return Array.from(serviceMap.entries()).map(([name, containerList]) => {
      const runningCount = containerList.filter(
        (c) => c.status?.toLowerCase() === "running"
      ).length;
      const stoppedCount = containerList.filter(
        (c) =>
          c.status?.toLowerCase() === "stopped" ||
          c.status?.toLowerCase() === "exited"
      ).length;

      let status = "unknown";
      if (runningCount === containerList.length && containerList.length > 0) {
        status = "running";
      } else if (stoppedCount === containerList.length && containerList.length > 0) {
        status = "stopped";
      } else if (runningCount > 0) {
        status = "partial";
      } else if (containerList.length === 0) {
        status = "no containers";
      }

      return {
        name,
        containers: containerList.sort((a, b) => {
          // Sort by status (running first) then by container ID
          const aRunning = a.status?.toLowerCase() === "running";
          const bRunning = b.status?.toLowerCase() === "running";
          if (aRunning !== bRunning) {
            return aRunning ? -1 : 1;
          }
          return a.containerId.localeCompare(b.containerId);
        }),
        containerCount: containerList.length,
        runningCount,
        stoppedCount,
        hasRunning: runningCount > 0,
        hasStopped: stoppedCount > 0,
        status,
      };
    });
  });

  const isProcessingService = (serviceName: string) => {
    return processingServices.value.has(serviceName);
  };

  const isProcessingContainer = (containerId: string) => {
    return processingContainers.value.has(containerId);
  };

  const getServiceStatusVariant = (status: string): "success" | "danger" | "warning" | "secondary" => {
    switch (status) {
      case "running":
        return "success";
      case "stopped":
        return "danger";
      case "partial":
        return "warning";
      default:
        return "secondary";
    }
  };

  const getServiceStatusDotClass = (status: string): string => {
    switch (status) {
      case "running":
        return "bg-success";
      case "stopped":
        return "bg-danger";
      case "partial":
        return "bg-warning";
      default:
        return "bg-secondary";
    }
  };

  const getContainerStatusVariant = (status?: string): "success" | "danger" | "warning" | "secondary" => {
    const s = (status || "").toLowerCase();
    if (s === "running") return "success";
    if (s === "stopped" || s === "exited") return "danger";
    return "secondary";
  };

  const getContainerStatusDotClass = (status?: string): string => {
    const s = (status || "").toLowerCase();
    if (s === "running") return "bg-success";
    if (s === "stopped" || s === "exited") return "bg-danger";
    return "bg-secondary";
  };

  const loadServiceNames = async () => {
    try {
      const res = await client.getDeploymentServiceNames({
        organizationId: props.organizationId,
        deploymentId: props.deploymentId,
      });
      serviceNames.value = res.serviceNames || [];
    } catch (error: any) {
      console.error("Failed to load service names:", error);
      // If this fails, we'll still work with containers that have service names
    }
  };

  const loadContainers = async () => {
    isLoading.value = true;
    try {
      const res = await client.listDeploymentContainers({
        organizationId: props.organizationId,
        deploymentId: props.deploymentId,
      });
      containers.value = res.containers || [];
    } catch (error: any) {
      console.error("Failed to load containers:", error);
      await showAlert({
        title: "Failed to Load Containers",
        message:
          error.message ||
          "Failed to load container information. Please try again.",
      });
      containers.value = [];
    } finally {
      isLoading.value = false;
    }
  };

  const refresh = async () => {
    await Promise.all([loadServiceNames(), loadContainers(), loadRoutingRules()]);
  };

  // Routing management
  interface LocalRoutingRule extends RoutingRule {
    targetPortStr: string;
  }

  const localRoutingRules = ref<Map<string, LocalRoutingRule[]>>(new Map());
  const isRoutingDirty = ref(false);

  const protocolOptions = [
    { label: "HTTP", value: "http" },
    { label: "HTTPS", value: "https" },
    { label: "gRPC", value: "grpc" },
  ];

  const sslResolverOptions = [
    { label: "Let's Encrypt", value: "letsencrypt" },
    { label: "Internal (Handled by App)", value: "internal" },
  ];

  const loadRoutingRules = async () => {
    isLoadingRouting.value = true;
    try {
      const res = await client.getDeploymentRoutings({
        organizationId: props.organizationId,
        deploymentId: props.deploymentId,
      });

      routingRules.value = res.rules || [];
      
      // Group rules by service name
      const rulesByService = new Map<string, LocalRoutingRule[]>();
      serviceNames.value.forEach((name) => {
        rulesByService.set(name, []);
      });

      routingRules.value.forEach((rule) => {
        const serviceName = rule.serviceName || "default";
        if (!rulesByService.has(serviceName)) {
          rulesByService.set(serviceName, []);
        }
        
        const protocol = rule.protocol || "http";
        // Sync SSL with protocol - HTTP should not have SSL enabled
        const sslEnabled = protocol === "http" ? false : (rule.sslEnabled ?? (protocol === "https"));
        
        // Create rule object with corrected values (order matters - override after spread)
        rulesByService.get(serviceName)!.push({
          ...rule,
          targetPortStr: String(rule.targetPort || 80),
          protocol: protocol,
          sslEnabled: sslEnabled, // This overrides the spread value to ensure correct state
        });
      });

      localRoutingRules.value = rulesByService;
      isRoutingDirty.value = false;
    } catch (error: any) {
      console.error("Failed to load routing rules:", error);
    } finally {
      isLoadingRouting.value = false;
    }
  };

  const getServiceRoutingRules = (serviceName: string): LocalRoutingRule[] => {
    return localRoutingRules.value.get(serviceName) || [];
  };

  const toggleServiceRouting = (serviceName: string) => {
    if (expandedServiceRouting.value.has(serviceName)) {
      expandedServiceRouting.value.delete(serviceName);
    } else {
      expandedServiceRouting.value.add(serviceName);
      // Load routing rules if not already loaded
      if (routingRules.value.length === 0) {
        loadRoutingRules();
      }
    }
  };

  const addRoutingRule = (serviceName: string) => {
    const rules = localRoutingRules.value.get(serviceName) || [];
    const defaultPort = props.deployment.port || 80;
    const newRule: LocalRoutingRule = {
      id: "",
      deploymentId: props.deploymentId,
      domain: props.deployment.domain || "",
      serviceName: serviceName,
      pathPrefix: "",
      targetPort: defaultPort,
      targetPortStr: String(defaultPort),
      protocol: "http",
      sslEnabled: false, // HTTP protocol defaults to no SSL
      sslCertResolver: "letsencrypt",
    } as LocalRoutingRule;
    rules.push(newRule);
    localRoutingRules.value.set(serviceName, rules);
    markRoutingDirty();
  };

  const removeRoutingRule = (rule: LocalRoutingRule) => {
    const serviceName = rule.serviceName || "default";
    const rules = localRoutingRules.value.get(serviceName) || [];
    const index = rules.findIndex((r) => r.id === rule.id || (r === rule && !r.id));
    if (index >= 0) {
      rules.splice(index, 1);
      localRoutingRules.value.set(serviceName, rules);
      markRoutingDirty();
    }
  };

  const markRoutingDirty = () => {
    isRoutingDirty.value = true;
  };

  const saveRoutingRules = async () => {
    if (isSavingRouting.value) return;

    // Collect all rules from all services
    const allRules: RoutingRule[] = [];
    localRoutingRules.value.forEach((rules, serviceName) => {
      rules.forEach((rule) => {
        if (rule.domain.trim()) {
          const protocol = rule.protocol || "http";
          // Ensure SSL is disabled for HTTP protocol
          const sslEnabled = protocol === "http" ? false : (rule.sslEnabled ?? (protocol === "https"));
          
          allRules.push({
            id: rule.id || "",
            deploymentId: props.deploymentId,
            domain: rule.domain.trim(),
            serviceName: rule.serviceName.trim() || serviceName,
            pathPrefix: rule.pathPrefix?.trim() || "",
            targetPort: rule.targetPort || 80,
            protocol: protocol,
            sslEnabled: sslEnabled,
            sslCertResolver: rule.sslCertResolver || "letsencrypt",
          } as RoutingRule);
        }
      });
    });

    isSavingRouting.value = true;
    try {
      const res = await client.updateDeploymentRoutings({
        organizationId: props.organizationId,
        deploymentId: props.deploymentId,
        rules: allRules,
      });

      routingRules.value = res.rules || [];
      await loadRoutingRules();
      
      await showAlert({
        title: "Success",
        message: "Routing rules saved successfully.",
      });
    } catch (error: any) {
      await showAlert({
        title: "Failed to Save",
        message: error.message || "Failed to save routing rules. Please try again.",
      });
    } finally {
      isSavingRouting.value = false;
    }
  };

  // Navigation helpers
  const openTerminal = (serviceName: string) => {
    router.push({
      path: `/deployments/${props.deploymentId}`,
      query: {
        tab: "terminal",
        service: serviceName,
      },
    });
  };

  const openFiles = (serviceName: string) => {
    router.push({
      path: `/deployments/${props.deploymentId}`,
      query: {
        tab: "files",
        service: serviceName,
      },
    });
  };

  // Service-level operations
  const startService = async (serviceName: string) => {
    processingServices.value.add(serviceName);
    try {
      const serviceContainers = containers.value.filter(
        (c) => (c.serviceName || "default") === serviceName
      );
      
      // Start all stopped containers for this service
      const stoppedContainers = serviceContainers.filter(
        (c) =>
          c.status?.toLowerCase() === "stopped" ||
          c.status?.toLowerCase() === "exited"
      );

      await Promise.all(
        stoppedContainers.map((container) =>
          client.startContainer({
            organizationId: props.organizationId,
            deploymentId: props.deploymentId,
            containerId: container.containerId,
          })
        )
      );

      // Refresh containers after a delay
      setTimeout(() => {
        loadContainers();
      }, 1000);
    } catch (error: any) {
      await showAlert({
        title: "Failed to Start Service",
        message:
          error.message ||
          `Failed to start service "${serviceName}". Please try again.`,
      });
    } finally {
      processingServices.value.delete(serviceName);
    }
  };

  const stopService = async (serviceName: string) => {
    processingServices.value.add(serviceName);
    try {
      const serviceContainers = containers.value.filter(
        (c) => (c.serviceName || "default") === serviceName
      );
      
      // Stop all running containers for this service
      const runningContainers = serviceContainers.filter(
        (c) => c.status?.toLowerCase() === "running"
      );

      if (runningContainers.length === 0) {
        return;
      }

      // Use Promise.allSettled to continue even if some fail
      // This handles network errors/timeouts more gracefully
      const results = await Promise.allSettled(
        runningContainers.map((container) =>
          client.stopContainer({
            organizationId: props.organizationId,
            deploymentId: props.deploymentId,
            containerId: container.containerId,
          })
        )
      );

      // Check if any failed with non-network errors
      const failures = results.filter((result): result is PromiseRejectedResult => {
        if (result.status === "fulfilled") return false;
        const error = result.reason;
        // Ignore network errors and timeouts - container might have stopped anyway
        const isNetworkError = 
          error?.message?.includes("NetworkError") ||
          error?.message?.includes("fetch") ||
          error?.message?.includes("timeout") ||
          error?.message?.includes("Failed to fetch") ||
          error?.code === "unknown" ||
          error?.name === "NetworkError";
        return !isNetworkError;
      });

      // Refresh containers immediately to check actual state
      await loadContainers();

      // Wait a bit and refresh again to catch any delayed state changes
      setTimeout(() => {
        loadContainers();
      }, 1000);

      // Only show error if there were non-network failures
      if (failures.length > 0) {
        const errorMessages = failures
          .map((f) => {
            const error = f.reason;
            return error?.message || error?.toString() || "Unknown error";
          })
          .filter((msg, idx, arr) => arr.indexOf(msg) === idx);
        
        await showAlert({
          title: "Failed to Stop Service",
          message: `Failed to stop some containers for service "${serviceName}": ${errorMessages.join(", ")}`,
        });
      }
      // Otherwise, assume success (containers might have stopped even if API timed out)
    } catch (error: any) {
      // Only show error if it's not a network/timeout error
      const isNetworkError = 
        error?.message?.includes("NetworkError") ||
        error?.message?.includes("fetch") ||
        error?.message?.includes("timeout") ||
        error?.message?.includes("Failed to fetch") ||
        error?.code === "unknown" ||
        error?.name === "NetworkError";

      if (!isNetworkError) {
        await showAlert({
          title: "Failed to Stop Service",
          message:
            error.message ||
            `Failed to stop service "${serviceName}". Please try again.`,
        });
      }
      
      // Refresh containers anyway to check actual state
      await loadContainers();
      setTimeout(() => {
        loadContainers();
      }, 1000);
    } finally {
      processingServices.value.delete(serviceName);
    }
  };

  const restartService = async (serviceName: string) => {
    processingServices.value.add(serviceName);
    try {
      const serviceContainers = containers.value.filter(
        (c) => (c.serviceName || "default") === serviceName
      );

      // Restart all containers for this service
      await Promise.all(
        serviceContainers.map((container) =>
          client.restartContainer({
            organizationId: props.organizationId,
            deploymentId: props.deploymentId,
            containerId: container.containerId,
          })
        )
      );

      // Refresh containers after a delay
      setTimeout(() => {
        loadContainers();
      }, 1000);
    } catch (error: any) {
      await showAlert({
        title: "Failed to Restart Service",
        message:
          error.message ||
          `Failed to restart service "${serviceName}". Please try again.`,
      });
    } finally {
      processingServices.value.delete(serviceName);
    }
  };

  // Container-level operations
  const startContainer = async (containerId: string) => {
    processingContainers.value.add(containerId);
    try {
      await client.startContainer({
        organizationId: props.organizationId,
        deploymentId: props.deploymentId,
        containerId,
      });

      setTimeout(() => {
        loadContainers();
      }, 1000);
    } catch (error: any) {
      await showAlert({
        title: "Failed to Start Container",
        message:
          error.message ||
          "Failed to start container. Please try again.",
      });
    } finally {
      processingContainers.value.delete(containerId);
    }
  };

  const stopContainer = async (containerId: string) => {
    processingContainers.value.add(containerId);
    try {
      await client.stopContainer({
        organizationId: props.organizationId,
        deploymentId: props.deploymentId,
        containerId,
      });

      // Refresh containers immediately and again after delay
      await loadContainers();
      setTimeout(() => {
        loadContainers();
      }, 1000);
    } catch (error: any) {
      // Only show error if it's not a network/timeout error
      // Container might have stopped even if API timed out
      const isNetworkError = 
        error?.message?.includes("NetworkError") ||
        error?.message?.includes("fetch") ||
        error?.message?.includes("timeout") ||
        error?.message?.includes("Failed to fetch") ||
        error?.code === "unknown" ||
        error?.name === "NetworkError";

      if (!isNetworkError) {
        await showAlert({
          title: "Failed to Stop Container",
          message:
            error.message ||
            "Failed to stop container. Please try again.",
        });
      }
      
      // Refresh containers anyway to check actual state
      await loadContainers();
      setTimeout(() => {
        loadContainers();
      }, 1000);
    } finally {
      processingContainers.value.delete(containerId);
    }
  };

  const restartContainer = async (containerId: string) => {
    processingContainers.value.add(containerId);
    try {
      await client.restartContainer({
        organizationId: props.organizationId,
        deploymentId: props.deploymentId,
        containerId,
      });

      setTimeout(() => {
        loadContainers();
      }, 1000);
    } catch (error: any) {
      await showAlert({
        title: "Failed to Restart Container",
        message:
          error.message ||
          "Failed to restart container. Please try again.",
      });
    } finally {
      processingContainers.value.delete(containerId);
    }
  };

  onMounted(() => {
    refresh();
  });

  // Watch for deployment changes
  watch(
    () => props.deploymentId,
    () => {
      refresh();
    }
  );

  // Watch for service names changes to reload routing
  watch(
    () => serviceNames.value,
    () => {
      loadRoutingRules();
    }
  );
</script>

