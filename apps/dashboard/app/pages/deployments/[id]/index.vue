<template>
  <OuiContainer size="full" py="sm" class="md:py-6">
    <OuiStack gap="md" class="md:gap-xl">
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
            <OuiCardBody class="p-3 md:p-6">
              <OuiStack gap="md" class="md:gap-lg">
                <OuiFlex justify="between" align="start" wrap="wrap" gap="md">
                  <OuiStack gap="sm" class="flex-1 min-w-0">
                    <OuiFlex align="center" gap="sm">
                      <OuiSkeleton width="3rem" height="3rem" variant="rectangle" :rounded="true" class="rounded-lg" />
                      <OuiStack gap="xs" class="flex-1">
                        <OuiSkeleton width="20rem" height="2rem" variant="text" />
                        <OuiSkeleton width="12rem" height="1rem" variant="text" />
                      </OuiStack>
                    </OuiFlex>
                  </OuiStack>
                  <OuiFlex gap="sm">
                    <OuiSkeleton width="6rem" height="2rem" variant="rectangle" rounded />
                    <OuiSkeleton width="6rem" height="2rem" variant="rectangle" rounded />
                  </OuiFlex>
                </OuiFlex>
              </OuiStack>
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
            <OuiCardBody class="p-3 md:p-6">
              <OuiFlex
                justify="between"
                align="start"
                wrap="wrap"
                gap="md"
                class="md:gap-lg md:items-center"
              >
                <OuiStack gap="sm" class="flex-1 min-w-0 md:gap-md">
                  <OuiFlex align="center" gap="sm" wrap="wrap" class="md:gap-md">
                    <OuiBox
                      p="xs"
                      rounded="lg"
                      bg="accent-primary"
                      class="bg-primary/10 ring-1 ring-primary/20 shrink-0 md:p-sm md:rounded-xl"
                    >
                      <RocketLaunchIcon
                        class="w-6 h-6 md:w-8 md:h-8 text-primary"
                      />
                    </OuiBox>
                    <OuiStack gap="xs" class="min-w-0 flex-1 md:gap-none">
                      <OuiFlex
                        align="center"
                        justify="between"
                        gap="md"
                        wrap="wrap"
                        class="md:justify-start"
                      >
                        <OuiText
                          as="h1"
                          size="xl"
                          weight="bold"
                          truncate
                          class="md:text-2xl"
                        >
                          {{ deployment.name }}
                        </OuiText>
                      <OuiBadge :variant="statusMeta.badge" size="xs">
                        <span
                          class="inline-flex h-1.5 w-1.5 rounded-full mr-1.5"
                          :class="statusMeta.dotClass"
                        />
                        <OuiText
                          as="span"
                          size="xs"
                          weight="semibold"
                          transform="uppercase"
                          >{{ statusMeta.label }}</OuiText
                        >
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
                        class="md:size-sm"
                      >
                        <OuiText as="span" size="xs" weight="medium">
                          {{ containerStats.runningCount }}/{{
                            containerStats.totalCount
                          }}
                          {{
                            containerStats.runningCount === containerStats.totalCount
                              ? 'running'
                              : containerStats.runningCount > 0
                              ? 'running'
                              : 'stopped'
                          }}
                        </OuiText>
                      </OuiBadge>
                    </OuiFlex>
                    <OuiText size="xs" color="secondary" class="md:text-sm">
                      Last deployed
                      <OuiRelativeTime
                        :value="
                          deployment.lastDeployedAt
                            ? date(deployment.lastDeployedAt)
                            : undefined
                        "
                        :style="'short'"
                      />
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiStack>

              <OuiFlex
                gap="xs"
                wrap="wrap"
                class="w-full md:w-auto shrink-0 md:gap-sm md:flex-nowrap"
              >
                <OuiButton
                  variant="ghost"
                  color="secondary"
                  size="sm"
                  @click="refreshAll"
                  :loading="isRefreshing"
                  class="gap-1.5 md:gap-2 flex-1 md:flex-initial"
                >
                  <ArrowPathIcon
                    class="h-4 w-4"
                    :class="{ 'animate-spin': isRefreshing }"
                  />
                  <OuiText
                    as="span"
                    size="xs"
                    weight="medium"
                    class="hidden sm:inline"
                    >Refresh</OuiText
                  >
                </OuiButton>
                <OuiButton
                  variant="ghost"
                  color="success"
                  size="sm"
                  @click="openDomain"
                  :disabled="!deployment.domain"
                  class="gap-1.5 md:gap-2 flex-1 md:flex-initial"
                >
                  <ArrowTopRightOnSquareIcon class="h-4 w-4" />
                  <OuiText
                    as="span"
                    size="xs"
                    weight="medium"
                    class="hidden sm:inline"
                    >Open</OuiText
                  >
                </OuiButton>
                <OuiButton
                  variant="outline"
                  color="warning"
                  size="sm"
                  @click="redeploy"
                  :disabled="
                    isProcessing ||
                    deployment.status === DeploymentStatusEnum.BUILDING ||
                    deployment.status === DeploymentStatusEnum.DEPLOYING
                  "
                  class="gap-1.5 md:gap-2 flex-1 md:flex-initial"
                >
                  <ArrowPathIcon
                    class="h-4 w-4"
                    :class="{
                      'animate-spin':
                        deployment.status === DeploymentStatusEnum.BUILDING ||
                        deployment.status === DeploymentStatusEnum.DEPLOYING,
                    }"
                  />
                  <OuiText
                    as="span"
                    size="xs"
                    weight="medium"
                    class="hidden sm:inline"
                    >Redeploy</OuiText
                  >
                </OuiButton>
                <OuiButton
                  v-if="hasRunningContainers"
                  variant="outline"
                  color="primary"
                  size="sm"
                  @click="restart"
                  :disabled="
                    isProcessing ||
                    deployment.status === DeploymentStatusEnum.BUILDING ||
                    deployment.status === DeploymentStatusEnum.DEPLOYING
                  "
                  class="gap-1.5 md:gap-2 flex-1 md:flex-initial"
                  title="Restart (restart without rebuilding)"
                >
                  <ArrowPathIcon class="h-4 w-4" />
                  <OuiText
                    as="span"
                    size="xs"
                    weight="medium"
                    class="hidden sm:inline"
                    >Restart</OuiText
                  >
                </OuiButton>
                <OuiButton
                  v-if="hasRunningContainers"
                  variant="solid"
                  color="danger"
                  size="sm"
                  @click="stop"
                  :disabled="
                    isProcessing ||
                    deployment.status === DeploymentStatusEnum.BUILDING ||
                    deployment.status === DeploymentStatusEnum.DEPLOYING
                  "
                  class="gap-1.5 md:gap-2 flex-1 md:flex-initial"
                >
                  <StopIcon class="h-4 w-4" />
                  <OuiText
                    as="span"
                    size="xs"
                    weight="medium"
                    class="hidden sm:inline"
                    >Stop</OuiText
                  >
                </OuiButton>
                <OuiButton
                  v-else-if="
                    !hasRunningContainers &&
                    (containerStats.totalCount > 0 ||
                      deployment.status === DeploymentStatusEnum.STOPPED)
                  "
                  variant="solid"
                  color="success"
                  size="sm"
                  @click="start"
                  :disabled="
                    isProcessing ||
                    deployment.status === DeploymentStatusEnum.BUILDING ||
                    deployment.status === DeploymentStatusEnum.DEPLOYING
                  "
                  class="gap-1.5 md:gap-2 flex-1 md:flex-initial"
                >
                  <PlayIcon class="h-4 w-4" />
                  <OuiText
                    as="span"
                    size="xs"
                    weight="medium"
                    class="hidden sm:inline"
                    >Start</OuiText
                  >
                </OuiButton>
              </OuiFlex>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>
        </Transition>

        <!-- Tabbed Content -->
        <OuiStack gap="sm" class="md:gap-md">
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
              <template #build-logs>
                <DeploymentBuildLogs
                  :key="`build-logs-${id}`"
                  ref="buildLogsRef"
                  :deployment-id="id"
                  :organization-id="orgId"
                  :auto-start="isBuildingOrDeploying"
                />
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
              <template #audit-logs>
                <AuditLogs
                  :organization-id="orgId"
                  resource-type="deployment"
                  :resource-id="id"
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
    ClipboardDocumentListIcon,
  } from "@heroicons/vue/24/outline";
  import {
    DeploymentService,
    type Deployment,
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
  const DeploymentBuildLogs = defineAsyncComponent(() => import("~/components/deployment/DeploymentBuildLogs.vue"));
  const DeploymentLogs = defineAsyncComponent(() => import("~/components/deployment/DeploymentLogs.vue"));
  const DeploymentTerminal = defineAsyncComponent(() => import("~/components/deployment/DeploymentTerminal.vue"));
  const DeploymentFiles = defineAsyncComponent(() => import("~/components/deployment/DeploymentFiles.vue"));
  const DeploymentCompose = defineAsyncComponent(() => import("~/components/deployment/DeploymentCompose.vue"));
  const DeploymentServices = defineAsyncComponent(() => import("~/components/deployment/DeploymentServices.vue"));
  const DeploymentEnvVars = defineAsyncComponent(() => import("~/components/deployment/DeploymentEnvVars.vue"));
  const AuditLogs = defineAsyncComponent(() => import("~/components/audit/AuditLogs.vue"));

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
      { id: "settings", label: "Settings", icon: Cog6ToothIcon },
      { id: "builds", label: "Builds", icon: ClockIcon },
      { id: "metrics", label: "Metrics", icon: ChartBarIcon },
      { id: "routing", label: "Routing", icon: GlobeAltIcon },
      { id: "build-logs", label: "Build Logs", icon: Cog6ToothIcon },
      { id: "logs", label: "Logs", icon: DocumentTextIcon },
      { id: "terminal", label: "Terminal", icon: CommandLineIcon },
      { id: "files", label: "Files", icon: FolderIcon },
    ];

    // Show compose tab only for PLAIN_COMPOSE without repository (manual compose editing)
    // Hide for repo-based compose (compose from repository)
    const dep = deployment.value;
    const isPlainComposeWithoutRepo =
      dep?.buildStrategy === BuildStrategy.PLAIN_COMPOSE && !dep?.repositoryUrl;

    if (isPlainComposeWithoutRepo) {
      baseTabs.push({ id: "compose", label: "Compose", icon: CodeBracketIcon });
    }

    // Show services tab for compose deployments (both PLAIN_COMPOSE and COMPOSE_REPO)
    const isComposeDeployment =
      dep?.buildStrategy === BuildStrategy.PLAIN_COMPOSE ||
      dep?.buildStrategy === BuildStrategy.COMPOSE_REPO;

    if (isComposeDeployment) {
      baseTabs.push({ id: "services", label: "Services", icon: CubeIcon });
    }

    baseTabs.push({ id: "env", label: "Environment", icon: VariableIcon });
    baseTabs.push({
      id: "audit-logs",
      label: "Audit Logs",
      icon: ClipboardDocumentListIcon,
    });

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
        containers.value = res.containers.map((c: any) => ({
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

  const isProcessing = computed(() => deploymentActions.isProcessing.value);

  function openDomain() {
    window.open(`https://${deployment.value.domain}`, "_blank");
  }

  async function start() {
    if (!localDeployment.value || isProcessing.value) return;
    try {
      await deploymentActions.startDeployment(id.value, localDeployment.value);
      // Immediately refresh to get updated status
      await refreshDeployment();
      await loadContainers();
      // Start polling to catch status changes as containers start
      startPolling();
    } catch (error: any) {
      console.error("Failed to start deployment:", error);
    }
  }

  async function stop() {
    if (!localDeployment.value || isProcessing.value) return;
    try {
      await deploymentActions.stopDeployment(id.value, localDeployment.value);
      // Immediately refresh to get updated status
      await refreshDeployment();
      await loadContainers();
      // Start polling to catch status changes as containers stop
      startPolling();
    } catch (error: any) {
      console.error("Failed to stop deployment:", error);
    }
  }

  const buildLogsRef = ref<{
    startStream: () => void;
    stopStream: () => void;
    clearLogs: () => void;
  } | null>(null);

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
    stopPolling();
  });

  async function redeploy() {
    if (!localDeployment.value) return;

    // Switch to build logs tab and start streaming
    activeTab.value = "build-logs";

    // Wait a tick for the component to mount, then start streaming
    await nextTick();
    if (buildLogsRef.value) {
      buildLogsRef.value.startStream();
    }

    // Trigger the redeployment
    await deploymentActions.redeployDeployment(id.value, localDeployment.value);
  }

  async function restart() {
    if (!localDeployment.value) return;

    // Restart deployment (restart containers without rebuilding)
    await deploymentActions.reloadDeployment(id.value, localDeployment.value);

    // Refresh deployment data to get updated status
    await refreshDeployment();
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
    } catch (error: any) {
      console.error("Failed to save compose:", error);
      await showAlert({
        title: "Failed to Save",
        message:
          error.message ||
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
    } catch (error: any) {
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
