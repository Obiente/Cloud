<template>
  <div>
    <OuiContainer size="7xl" py="xl">
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

        <!-- Tabbed Content -->
        <OuiCard variant="default">
          <OuiTabs v-model="activeTab" :tabs="tabs">
            <template #overview>
              <DeploymentOverview :deployment="deployment" />
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
              <DeploymentFiles :deployment-id="id" :organization-id="orgId" />
            </template>
            <template #compose>
              <DeploymentCompose
                :deployment="deployment"
                @save="handleComposeSave"
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
    </OuiContainer>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, watchEffect, watch } from "vue";
  import { useRoute, useRouter } from "vue-router";
  import type { TabItem } from "~/components/oui/Tabs.vue";
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
    VariableIcon,
    GlobeAltIcon,
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
  import { useOrganizationsStore } from "~/stores/organizations";

  definePageMeta({
    layout: "default",
    middleware: "auth",
  });

  const route = useRoute();
  const router = useRouter();
  const id = computed(() => String(route.params.id));
  const orgsStore = useOrganizationsStore();
  const orgId = computed(() => orgsStore.currentOrgId || "");

  const client = useConnectClient(DeploymentService);
  const deploymentActions = useDeploymentActions();

  const tabs: TabItem[] = [
    { id: "overview", label: "Overview", icon: RocketLaunchIcon },
    { id: "routing", label: "Routing", icon: GlobeAltIcon },
    { id: "logs", label: "Logs", icon: DocumentTextIcon },
    { id: "terminal", label: "Terminal", icon: CommandLineIcon },
    { id: "files", label: "Files", icon: FolderIcon },
    { id: "compose", label: "Compose", icon: CodeBracketIcon },
    { id: "env", label: "Environment", icon: VariableIcon },
  ];

  // Get initial tab from query parameter or default to "overview"
  const getInitialTab = () => {
    const tabParam = route.query.tab;
    if (typeof tabParam === "string") {
      // Validate that the tab exists
      const tabIds = tabs.map((t) => t.id);
      return tabIds.includes(tabParam) ? tabParam : "overview";
    }
    return "overview";
  };

  const activeTab = ref(getInitialTab());

  // Watch for tab changes and update query parameter
  watch(activeTab, (newTab) => {
    if (route.query.tab !== newTab) {
      router.replace({
        query: {
          ...route.query,
          tab: newTab === "overview" ? undefined : newTab, // Remove query param for default tab
        },
      });
    }
  });

  // Watch for query parameter changes (e.g., back/forward navigation)
  watch(
    () => route.query.tab,
    (tabParam) => {
      if (typeof tabParam === "string") {
        const tabIds = tabs.map((t) => t.id);
        if (tabIds.includes(tabParam) && activeTab.value !== tabParam) {
          activeTab.value = tabParam;
        }
      } else if (!tabParam && activeTab.value !== "overview") {
        activeTab.value = "overview";
      }
    }
  );

  // Fetch deployment data
  const { data: deploymentData } = await useAsyncData(
    () => `deployment-${id.value}`,
    async () => {
      try {
        const res = await client.getDeployment({
          organizationId: orgId.value,
          deploymentId: id.value,
        });
        return res.deployment;
      } catch (e) {
        console.error("Failed to fetch deployment:", e);
      }
    },
    { server: true, watch: [orgId] }
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

  function openDomain() {
    window.open(`https://${deployment.value.domain}`, "_blank");
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
    await deploymentActions.redeployDeployment(id.value, localDeployment.value);
  }

  async function handleComposeSave(composeYaml: string) {
    // TODO: Implement compose save
    console.log("Saving compose:", composeYaml);
  }

  async function handleEnvSave(envVars: Record<string, string>) {
    // TODO: Implement env vars save
    console.log("Saving env vars:", envVars);
  }

  // Expose DeploymentStatus enum to template
  const DeploymentStatusEnum = DeploymentStatus;
</script>
