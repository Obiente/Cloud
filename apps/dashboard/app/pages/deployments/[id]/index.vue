<template>
  <OuiContainer size="full" p="none">
    <OuiStack gap="lg">
      <!-- Access Error State -->
      <OuiCard v-if="accessError" variant="outline" class="border-danger/20">
        <OuiCardBody>
          <OuiStack gap="lg" align="center">
            <ErrorAlert
              :error="accessError"
              title="Access Denied"
              :hint="errorHint"
            />
            <OuiButton
              variant="solid"
              color="primary"
              @click="router.push('/deployments')"
            >
              Go to Deployments
            </OuiButton>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Deployment Content (only show if no access error) -->
      <template v-else>
        <!-- Loading Skeleton -->
        <template v-if="pending && !deploymentData">
          <OuiCard variant="outline" class="border-border-default/50">
            <OuiCardBody>
              <OuiFlex justify="between" align="start" wrap="wrap" gap="md">
                <OuiStack gap="sm" class="flex-1 min-w-0">
                  <OuiSkeleton width="16rem" height="1.5rem" variant="text" />
                  <OuiSkeleton width="10rem" height="0.875rem" variant="text" />
                </OuiStack>
                <OuiFlex gap="xs">
                  <OuiSkeleton width="5rem" height="2rem" variant="rectangle" rounded />
                  <OuiSkeleton width="5rem" height="2rem" variant="rectangle" rounded />
                </OuiFlex>
              </OuiFlex>
            </OuiCardBody>
          </OuiCard>
          <OuiCard variant="outline" class="border-border-default/50">
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiSkeleton width="100%" height="2rem" variant="text" />
                <OuiSkeleton width="80%" height="1rem" variant="text" />
                <OuiSkeleton width="60%" height="1rem" variant="text" />
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </template>

        <!-- Header -->
        <Transition name="fade" mode="out-in">
          <OuiCard v-if="!pending && deployment" variant="outline" class="border-border-default/50">
            <OuiCardBody>
              <OuiFlex justify="between" align="start" wrap="wrap" gap="md">
                <OuiStack gap="xs" class="flex-1 min-w-0">
                  <OuiFlex align="center" gap="sm" wrap="wrap">
                    <OuiText as="h1" size="lg" weight="semibold" truncate>{{ deployment.name }}</OuiText>
                    <OuiBadge :variant="statusMeta.badge" size="xs">
                      <span class="inline-flex h-1.5 w-1.5 rounded-full mr-1" :class="statusMeta.dotClass" />
                      {{ statusMeta.label }}
                    </OuiBadge>
                    <OuiBadge
                      v-if="containerStats.totalCount > 0"
                      :variant="
                        containerStats.runningCount === containerStats.totalCount
                          ? 'success'
                          : containerStats.runningCount > 0
                            ? 'warning'
                            : 'secondary'
                      "
                      size="xs"
                    >
                      {{ containerStats.runningCount }}/{{ containerStats.totalCount }} running
                    </OuiBadge>
                  </OuiFlex>
                  <OuiText size="xs" color="tertiary">
                    Last deployed
                    <OuiRelativeTime
                      :value="deployment.lastDeployedAt ? date(deployment.lastDeployedAt) : undefined"
                      :style="'short'"
                    />
                  </OuiText>
                </OuiStack>

                <OuiFlex gap="xs" wrap="wrap" class="shrink-0">
                  <OuiButton variant="ghost" size="sm" @click="refreshAll" :loading="isRefreshing" class="gap-1.5">
                    <ArrowPathIcon class="h-3.5 w-3.5" :class="{ 'animate-spin': isRefreshing }" />
                    <span class="hidden sm:inline">Refresh</span>
                  </OuiButton>
                  <OuiButton variant="ghost" size="sm" @click="openDomain" :disabled="!deployment.domain" class="gap-1.5">
                    <ArrowTopRightOnSquareIcon class="h-3.5 w-3.5" />
                    <span class="hidden sm:inline">Open</span>
                  </OuiButton>
                  <OuiButton
                    variant="outline"
                    size="sm"
                    @click="redeploy"
                    :disabled="commandBusy || deployment.status === DeploymentStatusEnum.BUILDING || deployment.status === DeploymentStatusEnum.DEPLOYING"
                    class="gap-1.5"
                  >
                    <ArrowPathIcon class="h-3.5 w-3.5" :class="{ 'animate-spin': operationKind === 'redeploy' || deployment.status === DeploymentStatusEnum.BUILDING || deployment.status === DeploymentStatusEnum.DEPLOYING }" />
                    <span class="hidden sm:inline">Redeploy</span>
                  </OuiButton>
                  <OuiButton
                    v-if="hasRunningContainers"
                    variant="outline"
                    size="sm"
                    @click="restart"
                    :disabled="commandBusy || deployment.status === DeploymentStatusEnum.BUILDING || deployment.status === DeploymentStatusEnum.DEPLOYING"
                    class="gap-1.5"
                    title="Restart without rebuilding"
                  >
                    <ArrowPathIcon class="h-3.5 w-3.5" :class="{ 'animate-spin': operationKind === 'restart' }" />
                    <span class="hidden sm:inline">Restart</span>
                  </OuiButton>
                  <OuiButton
                    v-if="hasRunningContainers"
                    variant="solid"
                    color="danger"
                    size="sm"
                    @click="stop"
                    :disabled="commandBusy || deployment.status === DeploymentStatusEnum.BUILDING || deployment.status === DeploymentStatusEnum.DEPLOYING"
                    class="gap-1.5"
                  >
                    <StopIcon class="h-3.5 w-3.5" />
                    <span class="hidden sm:inline">{{ operationKind === "stop" ? "Stopping" : "Stop" }}</span>
                  </OuiButton>
                  <OuiButton
                    v-else-if="!hasRunningContainers && (containerStats.totalCount > 0 || deployment.status === DeploymentStatusEnum.STOPPED)"
                    variant="solid"
                    color="success"
                    size="sm"
                    @click="start"
                    :disabled="commandBusy || deployment.status === DeploymentStatusEnum.BUILDING || deployment.status === DeploymentStatusEnum.DEPLOYING"
                    class="gap-1.5"
                  >
                    <ArrowPathIcon v-if="operationKind === 'start'" class="h-3.5 w-3.5 animate-spin" />
                    <PlayIcon v-else class="h-3.5 w-3.5" />
                    <span class="hidden sm:inline">{{ operationKind === "start" ? "Starting" : "Start" }}</span>
                  </OuiButton>
                </OuiFlex>
              </OuiFlex>
            </OuiCardBody>
          </OuiCard>
        </Transition>

        <OuiCard
          v-if="activeOperation || operationError"
          variant="outline"
          :class="operationError ? 'border-danger/30 bg-danger/5' : 'border-warning/30 bg-warning/5'"
        >
          <OuiCardBody>
            <OuiFlex align="start" gap="sm">
              <ArrowPathIcon
                v-if="activeOperation"
                class="mt-0.5 h-4 w-4 shrink-0 animate-spin text-warning"
              />
              <ExclamationTriangleIcon
                v-else
                class="mt-0.5 h-4 w-4 shrink-0 text-danger"
              />
              <OuiStack gap="xs" class="min-w-0">
                <OuiText size="sm" weight="medium">
                  {{ activeOperation?.label || "Command failed" }}
                </OuiText>
                <OuiText size="sm" color="tertiary">
                  {{ operationError || activeOperation?.description }}
                </OuiText>
              </OuiStack>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>

        <!-- Tabbed Content -->
        <OuiStack gap="sm">
          <OuiTabs v-model="activeTab" :tabs="tabs" />
          <OuiCard variant="default">
            <OuiTabs v-model="activeTab" :tabs="tabs" :content-only="true">
              <template #overview>
                <DeploymentOverview
                  :deployment="deployment"
                  :organization-id="orgId"
                  @navigate="(tab) => (activeTab = tab)"
                />
              </template>
              <template #settings>
                <DeploymentSettings :deployment="deployment" />
              </template>
              <template #builds>
                <DeploymentBuilds
                  :deployment-id="id"
                  :organization-id="orgId"
                />
              </template>
              <template #metrics>
                <DeploymentMetrics
                  ref="metricsRef"
                  :deployment-id="id"
                  :organization-id="orgId"
                />
              </template>
              <template #routing>
                <DeploymentRouting :deployment="deployment" />
              </template>
              <template #logs>
                <DeploymentLogs :deployment-id="id" :organization-id="orgId" />
              </template>
              <template #terminal>
                <DeploymentTerminal
                  :deployment-id="id"
                  :organization-id="orgId"
                />
              </template>
              <template #files>
                <DeploymentFiles
                  ref="filesRef"
                  :deployment-id="id"
                  :organization-id="orgId"
                />
              </template>
              <template #compose>
                <DeploymentCompose
                  :deployment="deployment"
                  @save="handleComposeSave"
                />
              </template>
              <template #services>
                <DeploymentServices
                  :deployment="deployment"
                  :deployment-id="id"
                  :organization-id="orgId"
                />
              </template>
              <template #env>
                <DeploymentEnvVars
                  :deployment="deployment"
                  @save="handleEnvSave"
                />
              </template>
            </OuiTabs>
          </OuiCard>
        </OuiStack>
      </template>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
  import {
    ref,
    computed,
    watchEffect,
    watch,
    nextTick,
    onMounted,
    onUnmounted,
    defineAsyncComponent,
  } from "vue";
  import { useRoute, useRouter } from "vue-router";
  import type { TabItem } from "~/components/oui/Tabs.vue";
  import { useTabQuery } from "~/composables/useTabQuery";
  import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
  import {
    ArrowPathIcon,
    ArrowTopRightOnSquareIcon,
    CodeBracketIcon,
    CommandLineIcon,
    DocumentTextIcon,
    FolderIcon,
    PlayIcon,
    RocketLaunchIcon,
    StopIcon,
    TrashIcon,
    VariableIcon,
    GlobeAltIcon,
    Cog6ToothIcon,
    ChartBarIcon,
    CubeIcon,
    ClockIcon,
    ExclamationTriangleIcon,
  } from "@heroicons/vue/24/outline";
  import {
    DeploymentService,
    type Deployment,
    type DeploymentContainer,
    DeploymentType,
    DeploymentStatus,
    Environment as EnvEnum,
    BuildStrategy,
  } from "@obiente/proto";
  import { date, timestamp } from "@obiente/proto/utils";
  import { useConnectClient } from "~/lib/connect-client";
  import { useDeploymentActions } from "~/composables/useDeploymentActions";
  import { useOrganizationsStore } from "~/stores/organizations";
  import { useDialog } from "~/composables/useDialog";
  import { ConnectError, Code } from "@connectrpc/connect";
  import ErrorAlert from "~/components/ErrorAlert.vue";
  
  // Lazy load tab components for better performance
  const DeploymentOverview = defineAsyncComponent(() => import("~/components/deployment/DeploymentOverview.vue"));
  const DeploymentSettings = defineAsyncComponent(() => import("~/components/deployment/DeploymentSettings.vue"));
  const DeploymentBuilds = defineAsyncComponent(() => import("~/components/deployment/DeploymentBuilds.vue"));
  const DeploymentMetrics = defineAsyncComponent(() => import("~/components/deployment/DeploymentMetrics.vue"));
  const DeploymentRouting = defineAsyncComponent(() => import("~/components/deployment/DeploymentRouting.vue"));
  const DeploymentLogs = defineAsyncComponent(() => import("~/components/deployment/DeploymentLogs.vue"));
  const DeploymentTerminal = defineAsyncComponent(() => import("~/components/deployment/DeploymentTerminal.vue"));
  const DeploymentFiles = defineAsyncComponent(() => import("~/components/deployment/DeploymentFiles.vue"));
  const DeploymentCompose = defineAsyncComponent(() => import("~/components/deployment/DeploymentCompose.vue"));
  const DeploymentServices = defineAsyncComponent(() => import("~/components/deployment/DeploymentServices.vue"));
  const DeploymentEnvVars = defineAsyncComponent(() => import("~/components/deployment/DeploymentEnvVars.vue"));

  definePageMeta({
    layout: "default",
    middleware: "auth",
  });

  const route = useRoute();
  const router = useRouter();
  const id = computed(() => String(route.params.id));
  const orgsStore = useOrganizationsStore();
  const orgId = computed(() => orgsStore.currentOrgId || "");
  const { showAlert, showConfirm } = useDialog();

  const client = useConnectClient(DeploymentService);
  const deploymentActions = useDeploymentActions();

  // Initialize deployment with a placeholder to avoid temporal dead zone
  const localDeployment = ref<Deployment | null>(null);

  // Error state for access/permission errors
  const accessError = ref<Error | null>(null);

  // Fetch deployment data
  const {
    data: deploymentData,
    pending,
    refresh: refreshDeploymentBase,
    error: fetchError,
  } = useClientFetch(
    `deployment-${id.value}`,
    async () => {
      if (!orgId.value) {
        return null;
      }
      try {
        const response = await client.getDeployment({
          organizationId: orgId.value,
          deploymentId: id.value,
        });
        // Clear any previous errors on success
        accessError.value = null;
        return response.deployment ?? null;
      } catch (err) {
        // Check if it's a permission denied or not found error
        if (err instanceof ConnectError) {
          if (
            err.code === Code.PermissionDenied ||
            err.code === Code.NotFound
          ) {
            accessError.value = err;
            // Don't throw - let the error state be handled by the UI
            return null;
          }
        }
        // Re-throw other errors
        throw err;
      }
    },
    {
      watch: [orgId, id],
    }
  );

  // Custom refresh function that doesn't set pending state (for polling)
  // This prevents UI jumps during background updates
  const refreshDeployment = async (silent = false) => {
    if (silent) {
      // Silent refresh: update data without triggering pending state
      try {
        if (!orgId.value) {
          return;
        }
        const response = await client.getDeployment({
          organizationId: orgId.value,
          deploymentId: id.value,
        });
        // Directly update the data without going through the composable
        // This prevents the pending state from being set
        if (response.deployment) {
          deploymentData.value = response.deployment;
          accessError.value = null;
        }
      } catch (err) {
        // Only handle errors silently - don't update state on error during polling
        // The next non-silent refresh will handle errors properly
        if (err instanceof ConnectError) {
          if (
            err.code === Code.PermissionDenied ||
            err.code === Code.NotFound
          ) {
            // Only set access error if it's a new error (not during polling)
            if (!silent) {
              accessError.value = err;
            }
          }
        }
        // Don't throw during silent refresh to prevent breaking the UI
        if (!silent) {
          throw err;
        }
      }
    } else {
      // Normal refresh: use the base refresh function
      return refreshDeploymentBase();
    }
  };

  // Watch for fetch errors and handle access errors
  watch(fetchError, (err) => {
    if (err instanceof ConnectError) {
      if (err.code === Code.PermissionDenied || err.code === Code.NotFound) {
        accessError.value = err;
      }
    }
  });

  // Computed error hint message
  const errorHint = computed(() => {
    if (!accessError.value || !(accessError.value instanceof ConnectError)) {
      return "You don't have permission to view this deployment, or it doesn't exist.";
    }

    if (accessError.value.code === Code.PermissionDenied) {
      return "You don't have permission to view this deployment. Please contact your organization administrator if you believe you should have access.";
    }

    if (accessError.value.code === Code.NotFound) {
      return "This deployment doesn't exist or may have been deleted.";
    }

    return "You don't have permission to view this deployment, or it doesn't exist.";
  });

  // Sync deploymentData to localDeployment
  watchEffect(() => {
    if (deploymentData.value) {
      localDeployment.value = deploymentData.value;
    }
  });

  const deployment = computed(
    () =>
      localDeployment.value ??
      ({
        id: id.value,
        name: "Loading...",
        domain: `${id.value}.obiente.cloud`,
        status: DeploymentStatus.CREATED,
        lastDeployedAt: timestamp(new Date()),
        environment: EnvEnum.DEVELOPMENT,
        type: DeploymentType.DOCKER,
        buildTime: 0,
        size: "--",
        branch: "main",
      } as Deployment)
  );

  // Computed tabs - conditionally show compose tab based on deployment type
  const tabs = computed<TabItem[]>(() => {
    const baseTabs: TabItem[] = [
      { id: "overview", label: "Overview", icon: RocketLaunchIcon },
      { id: "logs", label: "Logs", icon: DocumentTextIcon },
      { id: "builds", label: "Builds", icon: ClockIcon },
      { id: "metrics", label: "Metrics", icon: ChartBarIcon },
      { id: "terminal", label: "Terminal", icon: CommandLineIcon },
      { id: "routing", label: "Routing", icon: GlobeAltIcon },
      { id: "files", label: "Files", icon: FolderIcon },
    ];

    const dep = deployment.value;

    // Show compose tab only for PLAIN_COMPOSE without repository
    const isPlainComposeWithoutRepo =
      dep?.buildStrategy === BuildStrategy.PLAIN_COMPOSE && !dep?.repositoryUrl;

    if (isPlainComposeWithoutRepo) {
      baseTabs.push({ id: "compose", label: "Compose", icon: CodeBracketIcon });
    }

    // Show services tab for compose deployments
    const isComposeDeployment =
      dep?.buildStrategy === BuildStrategy.PLAIN_COMPOSE ||
      dep?.buildStrategy === BuildStrategy.COMPOSE_REPO;

    if (isComposeDeployment) {
      baseTabs.push({ id: "services", label: "Services", icon: CubeIcon });
    }

    baseTabs.push({ id: "env", label: "Environment", icon: VariableIcon });
    baseTabs.push({ id: "settings", label: "Settings", icon: Cog6ToothIcon });

    return baseTabs;
  });

  // Use composable for tab query parameter management
  const activeTab = useTabQuery(tabs);

  const getStatusMeta = (status: number) => {
    switch (status) {
      case DeploymentStatus.RUNNING:
        return {
          badge: "success" as const,
          label: "Running",
          dotClass: "bg-success",
        };
      case DeploymentStatus.STOPPED:
        return {
          badge: "danger" as const,
          label: "Stopped",
          dotClass: "bg-danger",
        };
      case DeploymentStatus.BUILDING:
      case DeploymentStatus.DEPLOYING:
        return {
          badge: "warning" as const,
          label: "Building",
          dotClass: "bg-warning",
        };
      case DeploymentStatus.FAILED:
        return {
          badge: "danger" as const,
          label: "Failed",
          dotClass: "bg-danger",
        };
      default:
        return {
          badge: "secondary" as const,
          label: "Unknown",
          dotClass: "bg-secondary",
        };
    }
  };

  const statusMeta = computed(() => getStatusMeta(deployment.value.status));

  // Container status tracking for better start/stop button logic
  const containers = ref<
    Array<{ containerId: string; serviceName?: string; status?: string }>
  >([]);
  const isLoadingContainers = ref(false);

  const containerStats = computed(() => {
    const runningCount = containers.value.filter(
      (c) => (c.status || "").toLowerCase() === "running"
    ).length;
    const stoppedCount = containers.value.filter(
      (c) =>
        (c.status || "").toLowerCase() === "stopped" ||
        (c.status || "").toLowerCase() === "exited"
    ).length;
    const totalCount = containers.value.length;
    const hasRunning = runningCount > 0;
    const hasStopped = stoppedCount > 0;

    return {
      runningCount,
      stoppedCount,
      totalCount,
      hasRunning,
      hasStopped,
    };
  });

  const hasRunningContainers = computed(() => {
    // If we have container data, use it; otherwise fall back to deployment status
    if (containerStats.value.totalCount > 0) {
      return containerStats.value.hasRunning;
    }
    // Fallback to deployment status if no containers loaded yet
    return deployment.value.status === DeploymentStatusEnum.RUNNING;
  });

  // Load containers to determine actual status
  const loadContainers = async () => {
    if (!id.value || !orgId.value) return;

    isLoadingContainers.value = true;
    try {
      const res = await (client as any).listDeploymentContainers({
        deploymentId: id.value,
        organizationId: orgId.value,
      });

      if (res?.containers) {
        containers.value = res.containers.map((c: DeploymentContainer) => ({
          containerId: c.containerId,
          serviceName: c.serviceName || undefined,
          status: c.status || "unknown",
        }));
      }
    } catch (err) {
      console.error("Failed to load containers:", err);
      containers.value = [];
    } finally {
      isLoadingContainers.value = false;
    }
  };

  // Load containers on mount and when deployment changes
  onMounted(() => {
    loadContainers();
  });

  watch(
    () => deployment.value?.id,
    () => {
      loadContainers();
    }
  );

  // Refresh containers after start/stop operations
  watch(
    () => deployment.value?.status,
    () => {
      // Debounce container refresh to avoid too many requests
      setTimeout(() => {
        loadContainers();
      }, 1000);
    }
  );

  type DeploymentOperationKind = "start" | "stop" | "redeploy" | "restart";
  type TrackedDeploymentOperation = {
    kind: DeploymentOperationKind;
    label: string;
    description: string;
    targetStatuses: DeploymentStatus[];
    timeoutId?: ReturnType<typeof setTimeout>;
  };

  const activeOperation = ref<TrackedDeploymentOperation | null>(null);
  const operationError = ref<string | null>(null);
  const operationKind = computed(() => activeOperation.value?.kind ?? deploymentActions.currentOperation.value);
  const isProcessing = computed(() => deploymentActions.isProcessing.value);
  const commandBusy = computed(() => isProcessing.value || activeOperation.value !== null);

  const getErrorMessage = (error: unknown, fallback: string) => {
    if (error instanceof Error && error.message) return error.message;
    return fallback;
  };

  const clearOperationTimeout = () => {
    if (activeOperation.value?.timeoutId) {
      clearTimeout(activeOperation.value.timeoutId);
    }
  };

  const finishTrackedOperation = () => {
    clearOperationTimeout();
    activeOperation.value = null;
  };

  const beginTrackedOperation = (operation: Omit<TrackedDeploymentOperation, "timeoutId">) => {
    clearOperationTimeout();
    operationError.value = null;
    activeOperation.value = {
      ...operation,
      timeoutId: setTimeout(() => {
        if (activeOperation.value?.kind !== operation.kind) return;
        operationError.value = `${operation.label} is taking longer than expected. The page is still refreshing; check the logs or refresh if the state does not settle.`;
        finishTrackedOperation();
      }, 90_000),
    };
  };

  const hasReachedOperationTarget = () => {
    const operation = activeOperation.value;
    if (!operation) return false;

    const status = deployment.value?.status;
    if (operation.targetStatuses.includes(status)) return true;

    if (operation.kind === "stop" && containerStats.value.totalCount > 0) {
      return containerStats.value.runningCount === 0;
    }

    if ((operation.kind === "start" || operation.kind === "restart") && containerStats.value.totalCount > 0) {
      return containerStats.value.runningCount === containerStats.value.totalCount;
    }

    return false;
  };

  const refreshAfterCommand = async () => {
    await Promise.allSettled([
      refreshDeployment(true),
      loadContainers(),
    ]);
    startPolling();
  };

  const runDeploymentCommand = async (
    operation: Omit<TrackedDeploymentOperation, "timeoutId">,
    command: () => Promise<unknown>,
    failureMessage: string
  ) => {
    if (!localDeployment.value || commandBusy.value) return;
    beginTrackedOperation(operation);
    try {
      await command();
      await refreshAfterCommand();
      if (hasReachedOperationTarget()) {
        finishTrackedOperation();
      }
    } catch (error: unknown) {
      console.error(failureMessage, error);
      operationError.value = getErrorMessage(error, failureMessage);
      finishTrackedOperation();
    }
  };

  function openDomain() {
    window.open(`https://${deployment.value.domain}`, "_blank");
  }

  async function start() {
    await runDeploymentCommand(
      {
        kind: "start",
        label: "Starting deployment",
        description: "The command was sent. Waiting for the backend to report running containers.",
        targetStatuses: [DeploymentStatus.RUNNING],
      },
      () => deploymentActions.startDeployment(id.value, localDeployment.value),
      "Failed to start deployment."
    );
  }

  async function stop() {
    await runDeploymentCommand(
      {
        kind: "stop",
        label: "Stopping deployment",
        description: "The command was sent. Waiting for the backend to confirm the containers stopped.",
        targetStatuses: [DeploymentStatus.STOPPED],
      },
      () => deploymentActions.stopDeployment(id.value, localDeployment.value),
      "Failed to stop deployment."
    );
  }

  const metricsRef = ref<{
    refreshUsage?: () => Promise<void>;
  } | null>(null);

  const filesRef = ref<{
    refreshRoot?: () => Promise<void>;
  } | null>(null);

  const isRefreshing = ref(false);
  const isBuildingOrDeploying = computed(() => {
    const status = deployment.value?.status;
    return (
      status === DeploymentStatus.BUILDING ||
      status === DeploymentStatus.DEPLOYING
    );
  });

  // Polling for deployment status and containers during transitions
  let pollingInterval: ReturnType<typeof setInterval> | null = null;
  const isPolling = ref(false);

  const startPolling = () => {
    if (pollingInterval) return; // Already polling

    isPolling.value = true;
    pollingInterval = setInterval(async () => {
      if (!id.value || !orgId.value) {
        stopPolling();
        return;
      }

      // Check if we should continue polling
      const status = deployment.value?.status;
      const shouldPoll =
        activeOperation.value !== null ||
        status === DeploymentStatus.BUILDING ||
        status === DeploymentStatus.DEPLOYING ||
        // Also poll if containers are partially running (transitioning)
        (containerStats.value.totalCount > 0 &&
          containerStats.value.runningCount > 0 &&
          containerStats.value.runningCount < containerStats.value.totalCount);

      if (!shouldPoll) {
        stopPolling();
        return;
      }

      // Refresh both deployment and containers
      // Use a flag to prevent concurrent refreshes
      if (isRefreshing.value) {
        return; // Skip if already refreshing
      }
      
      try {
        isRefreshing.value = true;
        // Use silent refresh to prevent UI jumps - this doesn't set pending state
        // Use Promise.allSettled to prevent one failure from stopping the other
        await Promise.allSettled([
          refreshDeployment(true), // silent = true to prevent pending state
          loadContainers(),
        ]);
        if (deployment.value?.status === DeploymentStatus.FAILED && activeOperation.value) {
          operationError.value = `${activeOperation.value.label} failed. Check the logs or builds tab for the backend error details.`;
          finishTrackedOperation();
        } else if (hasReachedOperationTarget()) {
          finishTrackedOperation();
        }
      } catch (error) {
        console.error("Failed to poll deployment status:", error);
      } finally {
        isRefreshing.value = false;
      }
    }, 3000); // Poll every 3 seconds (reduced from 2 to minimize revalidation impact)
  };

  const stopPolling = () => {
    if (pollingInterval) {
      clearInterval(pollingInterval);
      pollingInterval = null;
    }
    isPolling.value = false;
  };

  // Start polling when deployment is transitioning
  watch(
    () => [
      deployment.value?.status,
      containerStats.value.runningCount,
      containerStats.value.totalCount,
    ],
    () => {
      const status = deployment.value?.status;
      const shouldPoll =
        activeOperation.value !== null ||
        status === DeploymentStatus.BUILDING ||
        status === DeploymentStatus.DEPLOYING ||
        (containerStats.value.totalCount > 0 &&
          containerStats.value.runningCount > 0 &&
          containerStats.value.runningCount < containerStats.value.totalCount);

      if (shouldPoll && !pollingInterval) {
        startPolling();
      } else if (!shouldPoll && pollingInterval) {
        stopPolling();
      }
    },
    { immediate: true }
  );

  // Stop polling when component unmounts
  onUnmounted(() => {
    clearOperationTimeout();
    stopPolling();
  });

  async function redeploy() {
    // Switch to builds tab to monitor progress
    activeTab.value = "builds";

    await runDeploymentCommand(
      {
        kind: "redeploy",
        label: "Redeploying",
        description: "A new build/deploy command was sent. Waiting for the backend to report the final deployment state.",
        targetStatuses: [DeploymentStatus.RUNNING],
      },
      () => deploymentActions.redeployDeployment(id.value, localDeployment.value),
      "Failed to redeploy."
    );
  }

  async function restart() {
    await runDeploymentCommand(
      {
        kind: "restart",
        label: "Restarting deployment",
        description: "The restart command was sent. Waiting for the backend to report healthy running containers.",
        targetStatuses: [DeploymentStatus.RUNNING],
      },
      () => deploymentActions.reloadDeployment(id.value, localDeployment.value),
      "Failed to restart deployment."
    );
  }

  async function handleComposeSave(composeYaml: string) {
    if (!localDeployment.value) return;

    try {
      const res = await client.updateDeploymentCompose({
        organizationId: orgId.value,
        deploymentId: id.value,
        composeYaml: composeYaml,
      });

      // Update local deployment with response
      if (res.deployment) {
        localDeployment.value = res.deployment;
      }

      // Refresh deployment data
      await refreshDeployment();
    } catch (error: unknown) {
      console.error("Failed to save compose:", error);
      await showAlert({
        title: "Failed to Save",
        message:
          (error as Error).message ||
          "Failed to save Docker Compose configuration. Please try again.",
      });
    }
  }

  async function handleEnvSave(envFileContent: string) {
    // DeploymentEnvVars component already saves internally
    // This handler is just for refresh/notification
    if (!localDeployment.value) return;

    try {
      // Refresh deployment data to get updated env vars
      await refreshDeployment();

      // Reload deployment to sync local state
      const res = await client.getDeployment({
        organizationId: orgId.value,
        deploymentId: id.value,
      });
      if (res.deployment) {
        localDeployment.value = res.deployment;
      }
    } catch (error: unknown) {
      console.error("Failed to refresh deployment after env vars save:", error);
    }
  }

  // Refresh all deployment data
  async function refreshAll() {
    if (isRefreshing.value) return;
    isRefreshing.value = true;
    try {
      // Refresh deployment data
      await refreshDeployment();

      // Reload containers
      await loadContainers();

      // Refresh child components that expose refresh methods
      if (metricsRef.value?.refreshUsage) {
        await metricsRef.value.refreshUsage();
      }

      if (filesRef.value?.refreshRoot) {
        await filesRef.value.refreshRoot();
      }

      // Build logs, logs, and terminal components refresh automatically
      // or through their own mechanisms
    } catch (error) {
      console.error("Failed to refresh deployment:", error);
    } finally {
      isRefreshing.value = false;
    }
  }

  // Expose DeploymentStatus enum to template
  const DeploymentStatusEnum = DeploymentStatus;
</script>
