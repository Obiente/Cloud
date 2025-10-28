<template>
  <OuiContainer size="7xl" py="xl" class="min-h-screen">
    <OuiStack gap="xl">
      <!-- Header -->
      <OuiFlex justify="between" align="start" wrap="wrap" gap="lg">
        <OuiStack gap="xs" class="min-w-0">
          <OuiFlex align="center" gap="md">
            <OuiBox
              p="sm"
              rounded="xl"
              bg="accent-primary"
              class="bg-primary/10 ring-1 ring-primary/20"
            >
              <RocketLaunchIcon class="w-6 h-6 text-primary" />
            </OuiBox>
            <OuiText as="h1" size="2xl" weight="bold" truncate>
              {{ deployment.name }}
            </OuiText>
          </OuiFlex>
          <OuiFlex align="center" gap="md" wrap="wrap">
            <OuiBadge :variant="statusMeta.badge">
              <span
                class="inline-flex h-1.5 w-1.5 rounded-full"
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
            <OuiText size="sm" color="secondary"
              >Last deployed
              {{ formatRelativeTime(deployment.lastDeployedAt) }}</OuiText
            >
          </OuiFlex>
        </OuiStack>

        <OuiFlex gap="sm" wrap="wrap">
          <OuiButton
            variant="ghost"
            size="sm"
            @click="openDomain"
            class="gap-2"
          >
            <ArrowTopRightOnSquareIcon class="h-4 w-4" />
            <OuiText as="span" size="xs" weight="medium">Open</OuiText>
          </OuiButton>
          <OuiButton
            variant="ghost"
            color="warning"
            size="sm"
            @click="redeploy"
            class="gap-2"
          >
            <ArrowPathIcon class="h-4 w-4" />
            <OuiText as="span" size="xs" weight="medium">Redeploy</OuiText>
          </OuiButton>
          <OuiButton
            v-if="deployment.status === DeploymentStatusEnum.RUNNING"
            variant="solid"
            color="danger"
            size="sm"
            @click="stop"
            class="gap-2"
          >
            <StopIcon class="h-4 w-4" />
            <OuiText as="span" size="xs" weight="medium">Stop</OuiText>
          </OuiButton>
          <OuiButton
            v-else
            variant="solid"
            color="success"
            size="sm"
            @click="start"
            class="gap-2"
          >
            <PlayIcon class="h-4 w-4" />
            <OuiText as="span" size="xs" weight="medium">Start</OuiText>
          </OuiButton>
        </OuiFlex>
      </OuiFlex>

      <!-- Content layout -->
      <OuiGrid cols="1" :cols-xl="3" gap="lg">
        <!-- Main column -->
        <div class="xl:col-span-2 space-y-6">
          <!-- Overview -->
          <OuiCard variant="default">
            <OuiCardHeader>
              <OuiText as="h2" size="lg" weight="semibold">Overview</OuiText>
            </OuiCardHeader>
            <OuiCardBody>
              <OuiGrid cols="1" :cols-md="2" gap="md">
                <OuiBox
                  p="md"
                  rounded="xl"
                  class="ring-1 ring-border-muted bg-surface-muted/30"
                >
                  <OuiText
                    size="xs"
                    color="secondary"
                    transform="uppercase"
                    weight="bold"
                    >Domain</OuiText
                  >
                  <OuiFlex align="center" gap="sm" class="mt-1">
                    <Icon name="uil:globe" class="h-4 w-4 text-secondary" />
                    <OuiText size="sm" weight="medium">{{
                      deployment.domain
                    }}</OuiText>
                  </OuiFlex>
                </OuiBox>
                <OuiBox
                  p="md"
                  rounded="xl"
                  class="ring-1 ring-border-muted bg-surface-muted/30"
                >
                  <OuiText
                    size="xs"
                    color="secondary"
                    transform="uppercase"
                    weight="bold"
                    >Framework</OuiText
                  >
                  <OuiFlex align="center" gap="sm" class="mt-1">
                    <CodeBracketIcon class="h-4 w-4 text-primary" />
                    <OuiText size="sm" weight="medium">{{
                      getTypeLabel((deployment as any).type)
                    }}</OuiText>
                  </OuiFlex>
                </OuiBox>
                <OuiBox
                  p="md"
                  rounded="xl"
                  class="ring-1 ring-border-muted bg-surface-muted/30"
                >
                  <OuiText
                    size="xs"
                    color="secondary"
                    transform="uppercase"
                    weight="bold"
                    >Environment</OuiText
                  >
                  <OuiFlex align="center" gap="sm" class="mt-1">
                    <CpuChipIcon class="h-4 w-4 text-secondary" />
                    <OuiText size="sm" weight="medium">{{
                      getEnvironmentLabel(deployment.environment)
                    }}</OuiText>
                  </OuiFlex>
                </OuiBox>
                <OuiBox
                  p="md"
                  rounded="xl"
                  class="ring-1 ring-border-muted bg-surface-muted/30"
                >
                  <OuiText
                    size="xs"
                    color="secondary"
                    transform="uppercase"
                    weight="bold"
                    >Build Time</OuiText
                  >
                  <OuiText size="lg" weight="bold"
                    >{{ deployment.buildTime }}s</OuiText
                  >
                </OuiBox>
              </OuiGrid>
            </OuiCardBody>
          </OuiCard>

          <!-- Configuration -->
          <OuiCard variant="default">
            <OuiCardHeader>
              <OuiFlex justify="between" align="center">
                <OuiText as="h3" size="lg" weight="semibold"
                  >Configuration</OuiText
                >
                <OuiButton
                  size="sm"
                  variant="ghost"
                  @click="saveConfig"
                  :disabled="!isConfigDirty"
                  >Save</OuiButton
                >
              </OuiFlex>
            </OuiCardHeader>
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiInput
                  v-model="config.repositoryUrl"
                  label="Repository URL"
                  placeholder="https://github.com/org/repo"
                />
                <OuiGrid cols="1" :cols-md="2" gap="md">
                  <OuiInput
                    v-model="config.branch"
                    label="Branch"
                    placeholder="main"
                    class="w-full"
                  />
                  <OuiSelect
                    v-model="config.runtime"
                    :items="runtimeOptions"
                    label="Runtime"
                    placeholder="Select runtime"
                    class="w-full"
                  />
                </OuiGrid>
                <OuiGrid cols="1" :cols-md="2" gap="md">
                  <OuiInput
                    v-model="config.installCommand"
                    label="Install Command"
                    placeholder="pnpm install"
                    class="w-full"
                  />
                  <OuiInput
                    v-model="config.buildCommand"
                    label="Build Command"
                    placeholder="pnpm build"
                    class="w-full"
                  />
                </OuiGrid>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Logs -->
          <OuiCard variant="default">
            <OuiCardHeader>
              <OuiText as="h3" size="lg" weight="semibold">Recent Logs</OuiText>
            </OuiCardHeader>
            <OuiCardBody>
              <pre
                class="bg-black text-green-400 p-4 rounded-xl text-xs overflow-auto max-h-64"
                >{{ logs }}</pre
              >
            </OuiCardBody>
          </OuiCard>
        </div>

        <!-- Sidebar column -->
        <div class="space-y-6">
          <OuiCard variant="default">
            <OuiCardHeader>
              <OuiText as="h3" size="base" weight="semibold">Actions</OuiText>
            </OuiCardHeader>
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiButton
                  size="sm"
                  color="warning"
                  variant="solid"
                  @click="redeploy"
                  class="gap-2"
                >
                  <ArrowPathIcon class="h-4 w-4" /> Redeploy
                </OuiButton>
                <OuiButton
                  size="sm"
                  color="secondary"
                  variant="ghost"
                  @click="copyDomain"
                  class="gap-2"
                >
                  <Icon name="uil:copy" class="h-4 w-4" /> Copy Domain
                </OuiButton>
                <OuiButton
                  size="sm"
                  color="danger"
                  variant="ghost"
                  @click="deleteDeployment"
                  class="gap-2"
                >
                  <Icon name="uil:trash" class="h-4 w-4" /> Delete
                </OuiButton>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <OuiCard variant="default">
            <OuiCardHeader>
              <OuiText as="h3" size="base" weight="semibold"
                >Build Info</OuiText
              >
            </OuiCardHeader>
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex justify="between"
                  ><OuiText size="sm" color="secondary">Size</OuiText
                  ><OuiText size="sm" weight="medium">{{
                    deployment.size
                  }}</OuiText></OuiFlex
                >
                <OuiFlex justify="between"
                  ><OuiText size="sm" color="secondary">Framework</OuiText
                  ><OuiText size="sm" weight="medium">{{
                    getTypeLabel((deployment as any).type)
                  }}</OuiText></OuiFlex
                >
                <OuiFlex justify="between"
                  ><OuiText size="sm" color="secondary">Environment</OuiText
                  ><OuiText size="sm" weight="medium">{{
                    getEnvironmentLabel(deployment.environment)
                  }}</OuiText></OuiFlex
                >
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </div>
      </OuiGrid>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, watchEffect } from "vue";
  import { useRoute } from "vue-router";
  import {
    ArrowPathIcon,
    ArrowTopRightOnSquareIcon,
    CodeBracketIcon,
    CpuChipIcon,
    PlayIcon,
    RocketLaunchIcon,
    StopIcon,
  } from "@heroicons/vue/24/outline";
  import {
    DeploymentService,
    type Deployment,
    DeploymentType,
    DeploymentStatus,
    Environment as EnvEnum,
  } from "@obiente/proto";
  import { date, timestamp } from "@obiente/proto/utils";
  import { useConnectClient } from "~/lib/connect-client";
  import { useDeploymentActions } from "~/composables/useDeploymentActions";

  const route = useRoute();
  const id = computed(() => String(route.params.id));

  const client = useConnectClient(DeploymentService);
  const deploymentActions = useDeploymentActions();

  // Fetch deployment data
  const { data: deploymentData } = await useAsyncData(
    `deployment-${id.value}`,
    async () => {
      try {
        const res = await client.getDeployment({
          organizationId: "default", // TODO: get from auth context
          deploymentId: id.value,
        });
        return res.deployment;
      } catch (e) {
        console.error("Failed to fetch deployment:", e);
      }
    },
    { server: true }
  );

  // Local reactive reference for mutations
  const localDeployment = ref<Deployment | null>(null);

  // Sync with async data
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

  const config = reactive({
    repositoryUrl: "",
    branch: "main",
    runtime: "node",
    installCommand: "",
    buildCommand: "",
  });

  // Initialize config from deployment data
  watchEffect(() => {
    if (deploymentData.value) {
      config.repositoryUrl = deploymentData.value.repositoryUrl ?? "";
      config.branch = deploymentData.value.branch ?? "main";
      config.installCommand = deploymentData.value.installCommand ?? "";
      config.buildCommand = deploymentData.value.buildCommand ?? "";
    }
  });

  const isConfigDirty = ref(false);

  const runtimeOptions = [
    { label: "Node.js", value: "node" },
    { label: "Go", value: "go" },
    { label: "Docker", value: "docker" },
    { label: "Static", value: "static" },
  ];

  const logs = ref(
    "[info] Initializing build...\n[info] Installing dependencies...\n"
  );

  const formatRelativeTime = (ts: any) => {
    const dateValue = ts ? date(ts) : new Date();
    const now = new Date();
    const diffMs = now.getTime() - dateValue.getTime();
    const diffSec = Math.floor(diffMs / 1000);
    const diffMin = Math.floor(diffSec / 60);
    const diffHour = Math.floor(diffMin / 60);
    const diffDay = Math.floor(diffHour / 24);
    if (diffSec < 60) return "just now";
    if (diffMin < 60) return `${diffMin}m ago`;
    if (diffHour < 24) return `${diffHour}h ago`;
    if (diffDay < 7) return `${diffDay}d ago`;
    return new Intl.DateTimeFormat("en-US", {
      month: "short",
      day: "numeric",
    }).format(dateValue);
  };

  const getTypeLabel = (t: DeploymentType | number | undefined) => {
    switch (t) {
      case DeploymentType.DOCKER:
        return "Docker";
      case DeploymentType.STATIC:
        return "Static Site";
      case DeploymentType.NODE:
        return "Node.js";
      case DeploymentType.GO:
        return "Go";
      default:
        return "Custom";
    }
  };

  const getEnvironmentLabel = (env: string | EnvEnum | number) => {
    if (typeof env === "number") {
      switch (env) {
        case EnvEnum.PRODUCTION:
          return "Production";
        case EnvEnum.STAGING:
          return "Staging";
        case EnvEnum.DEVELOPMENT:
          return "Development";
        default:
          return "Environment";
      }
    }
    return String(env);
  };

  function openDomain() {
    window.open(`https://${deployment.value.domain}`, "_blank");
  }

  function copyDomain() {
    navigator.clipboard?.writeText(deployment.value.domain);
  }

  async function start() {
    if (!localDeployment.value) return;
    await deploymentActions.startDeployment(id.value, localDeployment.value);
  }

  async function stop() {
    if (!localDeployment.value) return;
    await deploymentActions.stopDeployment(id.value, localDeployment.value);
  }

  async function redeploy() {
    if (!localDeployment.value) return;
    logs.value += `[info] Triggering redeploy at ${new Date().toISOString()}\n`;
    await deploymentActions.redeployDeployment(id.value, localDeployment.value);
  }

  async function deleteDeployment() {
    if (!confirm("Are you sure you want to delete this deployment?")) return;
    try {
      await deploymentActions.deleteDeployment(id.value);
      navigateTo("/deployments");
    } catch (error) {
      console.error("Failed to delete deployment:", error);
      alert("Failed to delete deployment. Please try again.");
    }
  }

  async function saveConfig() {
    if (!localDeployment.value) return;
    try {
      await deploymentActions.updateDeployment(id.value, {
        branch: config.branch,
        buildCommand: config.buildCommand,
        installCommand: config.installCommand,
      });

      // Refresh deployment data
      await refreshNuxtData(`deployment-${id.value}`);
      isConfigDirty.value = false;
    } catch (error) {
      console.error("Failed to save config:", error);
    }
  }

  // Expose DeploymentStatus enum to template
  const DeploymentStatusEnum = DeploymentStatus;
</script>

<style scoped>
  .log-line-enter-active,
  .log-line-leave-active {
    transition: opacity 0.2s;
  }
  .log-line-enter-from,
  .log-line-leave-to {
    opacity: 0;
  }
</style>
