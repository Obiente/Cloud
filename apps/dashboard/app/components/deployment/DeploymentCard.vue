<template>
  <ResourceCard :title="deployment?.name || ''" :subtitle="primaryDomain" :status-meta="statusMeta"
    :created-at="lastDeployedAtDate" :detail-url="detailUrl" :is-actioning="isActioning || showProgress"
    :loading="loading" :resources="resources">
    <template #subtitle>
      <OuiStack gap="xs">
        <a v-if="!loading && primaryDomain" :href="`https://${primaryDomain}`" target="_blank" rel="noopener noreferrer"
          class="inline-flex items-center gap-1.5 text-sm text-secondary hover:text-primary transition-colors"
          @click.stop>
          <span class="truncate">{{ primaryDomain }}</span>
          <ArrowTopRightOnSquareIcon class="h-3.5 w-3.5" />
        </a>
      </OuiStack>
    </template>

    <template #actions>
      <OuiFlex gap="xs" wrap="wrap">
        <OuiBadge v-if="!loading && showContainerStatus" :variant="containerStatusVariant" size="sm">
          {{ deployment?.containersRunning ?? 0 }}/{{ deployment?.containersTotal }}
          {{
            (deployment?.containersRunning ?? 0) === deployment?.containersTotal
              ? 'running'
              : (deployment?.containersRunning ?? 0) > 0
                ? 'running'
                : 'stopped'
          }}
        </OuiBadge>
        <OuiBadge v-if="!loading" v-for="(group, idx) in deploymentGroups" :key="idx" variant="secondary" size="sm">
          {{ group }}
        </OuiBadge>
        <OuiButton v-if="!loading && deployment && deployment.status === DeploymentStatus.RUNNING" variant="ghost"
          size="sm" icon-only @click.stop="handleStop" title="Stop">
          <StopIcon class="h-4 w-4" />
        </OuiButton>
        <OuiButton v-if="!loading && deployment && deployment.status === DeploymentStatus.STOPPED" variant="ghost"
          size="sm" icon-only @click.stop="handleStart" title="Start">
          <PlayIcon class="h-4 w-4" />
        </OuiButton>
        <OuiButton v-if="!loading && deployment" variant="ghost" size="sm" icon-only @click.stop="handleRedeploy"
          title="Redeploy">
          <ArrowPathIcon class="h-4 w-4" />
        </OuiButton>
      </OuiFlex>
    </template>

    <template #resources>
      <!-- Skeleton for resources section -->
      <OuiStack v-if="loading" gap="md">
        <OuiFlex justify="between" align="center">
          <OuiFlex align="center" gap="sm">
            <OuiBox p="xs" rounded="lg" bg="surface-muted"
              class="bg-surface-muted/50 ring-1 ring-border-muted opacity-30">
              <CodeBracketIcon class="h-4 w-4 text-primary"
                :style="{ opacity: iconVar.opacity, transform: `scale(${iconVar.scale})` }" />
            </OuiBox>
            <OuiSkeleton :width="randomTextWidthByType('label')" height="1rem" variant="text" />
          </OuiFlex>
          <OuiFlex align="center" gap="xs" class="text-xs text-secondary opacity-30">
            <CalendarIcon class="h-3.5 w-3.5"
              :style="{ opacity: iconVar.opacity, transform: `scale(${iconVar.scale})` }" />
            <OuiSkeleton :width="randomTextWidthByType('short')" height="0.875rem" variant="text" />
          </OuiFlex>
        </OuiFlex>

        <OuiFlex justify="between" align="center">
          <OuiBox p="sm" rounded="lg" w="4xl" bg="surface-muted"
            class="bg-surface-muted/30 ring-1 ring-border-muted opacity-30">
            <OuiFlex align="center" gap="sm" class="min-w-0">
              <Icon name="uil:github" class="h-4 w-4 text-secondary shrink-0"
                :style="{ opacity: iconVar.opacity, transform: `scale(${iconVar.scale})` }" />
              <OuiSkeleton :width="randomTextWidthByType('subtitle')" height="0.875rem" variant="text" />
            </OuiFlex>
          </OuiBox>
          <OuiSkeleton :width="randomTextWidthByType('short')" height="1.5rem" variant="rectangle" :rounded="true"
            class="opacity-30" />
        </OuiFlex>

        <OuiGrid :cols="{ sm: 2 }" gap="sm">
          <OuiBox p="sm" rounded="lg" bg="surface-muted"
            class="bg-surface-muted/40 ring-1 ring-border-muted opacity-30">
            <OuiStack gap="xs">
              <OuiText size="xs" weight="bold" transform="uppercase" color="secondary"
                class="tracking-wider opacity-50">
                Build Time
              </OuiText>
              <OuiSkeleton :width="randomTextWidthByType('value')" height="1.5rem" variant="text" />
            </OuiStack>
          </OuiBox>

          <OuiBox p="sm" rounded="lg" bg="surface-muted"
            class="bg-surface-muted/40 ring-1 ring-border-muted opacity-30">
            <OuiStack gap="xs">
              <OuiText size="xs" weight="bold" transform="uppercase" color="secondary"
                class="tracking-wider opacity-50">
                Bundle Size
              </OuiText>
              <OuiSkeleton :width="randomTextWidthByType('value')" height="1.5rem" variant="text" />
            </OuiStack>
          </OuiBox>
        </OuiGrid>
      </OuiStack>

      <!-- Actual resources content -->
      <OuiStack v-else-if="deployment" gap="md">
        <OuiFlex justify="between" align="center">
          <OuiFlex align="center" gap="sm">
            <OuiBox p="xs" rounded="lg" bg="surface-muted" class="bg-surface-muted/50 ring-1 ring-border-muted">
              <CodeBracketIcon class="h-4 w-4 text-primary" />
            </OuiBox>
            <OuiText size="sm" weight="medium" color="primary">
              {{ typeLabel }}
            </OuiText>
          </OuiFlex>
          <OuiFlex align="center" gap="xs" class="text-xs text-secondary">
            <CalendarIcon class="h-3.5 w-3.5" />
            <OuiRelativeTime :value="lastDeployedAtDate" :style="'short'" />
          </OuiFlex>
        </OuiFlex>

        <OuiFlex justify="between" align="center">
          <OuiBox v-if="deployment?.repositoryUrl" p="sm" rounded="lg" bg="surface-muted"
            class="bg-surface-muted/30 ring-1 ring-border-muted">
            <OuiFlex align="center" gap="sm">
              <Icon name="uil:github" class="h-4 w-4 text-secondary shrink-0" />
              <OuiText size="xs" color="secondary" truncate >
                {{ cleanRepositoryName(deployment.repositoryUrl) }}
              </OuiText>
            </OuiFlex>
          </OuiBox>
          <span v-if="deployment"
            class="inline-flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-semibold uppercase tracking-wide ml-auto"
            :class="environmentMeta.chipClass">
            <CpuChipIcon class="h-3.5 w-3.5" />
            {{ environmentMeta.label }}
          </span>
        </OuiFlex>

        <OuiGrid :cols="{ sm: 2 }" gap="sm">
          <OuiBox p="sm" rounded="lg" bg="surface-muted" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="xs">
              <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
                Build Time
              </OuiText>
              <OuiText size="lg" weight="bold" color="primary">
                {{ formatBuildTime(deployment?.buildTime ?? 0) }}
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox p="sm" rounded="lg" bg="surface-muted" class="bg-surface-muted/40 ring-1 ring-border-muted">
            <OuiStack gap="xs">
              <OuiText size="xs" weight="bold" transform="uppercase" color="secondary" class="tracking-wider">
                Bundle Size
              </OuiText>
              <OuiText size="lg" weight="bold" color="primary">
                <OuiByte :value="deployment.size ?? 0" unit-display="short" />
              </OuiText>
            </OuiStack>
          </OuiBox>
        </OuiGrid>
      </OuiStack>
    </template>

    <template #info>
      <!-- Build Progress Status -->
      <OuiBox v-if="showProgress" p="md" rounded="xl" class="border backdrop-blur-sm" :class="progressClass">
        <OuiStack gap="sm">
          <OuiFlex align="center" gap="sm" class="text-xs font-bold uppercase tracking-wider"
            :class="progressTextClass">
            <Cog6ToothIcon v-if="!isProgressFailed" class="h-4 w-4 animate-spin" />
            <ExclamationTriangleIcon v-else class="h-4 w-4 text-danger" />
            <span :class="progressTextClass">
              {{ props.progressPhase || "Starting deployment..." }}
            </span>
          </OuiFlex>
          <div class="relative h-2 w-full overflow-hidden rounded-full" :class="progressBarBgClass">
            <div v-if="isProgressFailed" class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
              :class="progressBarFillClass" :style="{ width: '100%' }" />
            <div v-else class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
              :class="progressBarFillClass" :style="{ width: `${props.progressValue || 0}%` }" />
          </div>
        </OuiStack>
      </OuiBox>
    </template>
  </ResourceCard>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import {
  PlayIcon,
  StopIcon,
  ArrowPathIcon,
  ArrowTopRightOnSquareIcon,
  CodeBracketIcon,
  CpuChipIcon,
  CalendarIcon,
  Cog6ToothIcon,
  ExclamationTriangleIcon,
} from "@heroicons/vue/24/outline";
import {
  type Deployment,
  DeploymentType,
  DeploymentStatus,
  Environment as EnvEnum,
  DeploymentService,
} from "@obiente/proto";
import { date } from "@obiente/proto/utils";
import { useDeploymentActions } from "~/composables/useDeploymentActions";
import ResourceCard from "~/components/shared/ResourceCard.vue";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import OuiByte from "~/components/oui/Byte.vue";
import { useSkeletonVariations, randomTextWidthByType, randomIconVariation } from "~/composables/useSkeletonVariations";
import OuiSkeleton from "~/components/oui/Skeleton.vue";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationId } from "~/composables/useOrganizationId";

interface Props {
  deployment?: Deployment;
  progressValue?: number;
  progressPhase?: string;
  loading?: boolean;
  organizationId?: string;
}

const props = withDefaults(defineProps<Props>(), {
  progressValue: 0,
  progressPhase: "",
  loading: false,
});

const emit = defineEmits<{
  refresh: [];
}>();

const { startDeployment, stopDeployment, redeployDeployment } = useDeploymentActions();
const isActioning = ref(false);
const client = useConnectClient(DeploymentService);
const organizationIdFromComposable = useOrganizationId();

// Get organizationId from prop or composable
const organizationId = computed(() => props.organizationId || organizationIdFromComposable.value || "");

// Fetch routing rules to get the primary domain
const { data: routingData } = await useClientFetch(
  () => `deployment-routings-${props.deployment?.id}`,
  async () => {
    if (!props.deployment?.id || !organizationId.value) return null;
    if (isMockDeployment.value) {
      return { rules: [] } as any;
    }
    try {
      const res = await client.getDeploymentRoutings({
        deploymentId: props.deployment.id,
        organizationId: organizationId.value,
      });
      return res;
    } catch (err) {
      console.error("Failed to fetch routing rules:", err);
      return null;
    }
  },
  {
    watch: [() => props.deployment?.id, () => organizationId.value],
    server: false
  }
);

// Computed property to get the primary domain from routing rules
// Prefers custom domains over deploy-XXX.my.obiente.cloud domains
const primaryDomain = computed(() => {
  if (!props.deployment) return "";

  const rules = routingData.value?.rules || [];
  if (rules.length === 0) {
    // Fallback to deployment.domain if no routing rules
    return props.deployment.domain || "";
  }

  // Get the first routing rule's domain
  const firstRuleDomain = rules[0]?.domain || "";

  // If the first rule has a custom domain (not deploy-XXX.my.obiente.cloud), use it
  if (firstRuleDomain && !firstRuleDomain.match(/^deploy-\d+\.my\.obiente\.cloud$/)) {
    return firstRuleDomain;
  }

  // If first rule is deploy-XXX.my.obiente.cloud, look for a custom domain in other rules
  const customDomain = rules.find((rule: { domain?: string }) =>
    rule.domain && !rule.domain.match(/^deploy-\d+\.my\.obiente\.cloud$/)
  );

  if (customDomain?.domain) {
    return customDomain.domain;
  }

  // If no custom domain found, use the first rule's domain (even if it's deploy-XXX.my.obiente.cloud)
  // Only if it's actually routed (has a domain set)
  if (firstRuleDomain) {
    return firstRuleDomain;
  }

  // Final fallback to deployment.domain
  return props.deployment.domain || "";
});

// Generate random variations for skeleton (consistent per instance)
const skeletonVars = useSkeletonVariations();
const iconVar = randomIconVariation();

const STATUS_META = {
  [DeploymentStatus.RUNNING]: {
    badge: "success" as const,
    label: "Running",
    cardClass: "hover:ring-1 hover:ring-success/30",
    beforeGradient:
      "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-success/20 before:via-success/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
    barClass: "bg-gradient-to-r from-success to-success/70",
    iconClass: "text-success",
    progressClass: "border-success/30 bg-success/10 text-success",
  },
  [DeploymentStatus.STOPPED]: {
    badge: "danger" as const,
    label: "Stopped",
    cardClass: "hover:ring-1 hover:ring-danger/30",
    beforeGradient:
      "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-danger/20 before:via-danger/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
    barClass: "bg-gradient-to-r from-danger to-danger/60",
    iconClass: "text-danger",
    progressClass: "border-danger/30 bg-danger/10 text-danger",
  },
  [DeploymentStatus.BUILDING]: {
    badge: "warning" as const,
    label: "Building",
    cardClass: "hover:ring-1 hover:ring-warning/30",
    beforeGradient:
      "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-warning/20 before:via-warning/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
    barClass: "bg-gradient-to-r from-warning to-warning/60 animate-pulse",
    iconClass: "text-warning",
    progressClass: "border-warning/30 bg-warning/10 text-warning",
  },
  [DeploymentStatus.DEPLOYING]: {
    badge: "warning" as const,
    label: "Deploying",
    cardClass: "hover:ring-1 hover:ring-warning/30",
    beforeGradient:
      "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-warning/20 before:via-warning/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
    barClass: "bg-gradient-to-r from-warning to-warning/60 animate-pulse",
    iconClass: "text-warning",
    progressClass: "border-warning/30 bg-warning/10 text-warning",
  },
  [DeploymentStatus.FAILED]: {
    badge: "danger" as const,
    label: "Failed",
    cardClass: "hover:ring-1 hover:ring-danger/30",
    beforeGradient:
      "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-danger/20 before:via-danger/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
    barClass: "bg-gradient-to-r from-danger to-danger/60",
    iconClass: "text-danger",
    progressClass: "border-danger/30 bg-danger/10 text-danger",
  },
} as const;

const statusMeta = computed(() => {
  if (!props.deployment || props.loading) {
    return STATUS_META[DeploymentStatus.STOPPED];
  }
  const status = props.deployment.status as DeploymentStatus;
  if (status in STATUS_META) {
    return STATUS_META[status as keyof typeof STATUS_META];
  }
  return STATUS_META[DeploymentStatus.STOPPED];
});

const resources = computed(() => {
  if (props.loading || !props.deployment) {
    return [
      { label: "Build Time", icon: null },
      { label: "Memory", icon: null },
    ];
  }
  return [];
});

const typeLabel = computed(() => {
  if (!props.deployment || props.loading) return "Unknown";
  const type = props.deployment.type;
  if (!type) return "Unknown";

  const typeMap: Record<number, string> = {
    [DeploymentType.STATIC]: "Static",
    [DeploymentType.DOCKER]: "Docker",
    [DeploymentType.NODE]: "Node.js",
    [DeploymentType.PYTHON]: "Python",
    [DeploymentType.RUBY]: "Ruby",
    [DeploymentType.GO]: "Go",
    [DeploymentType.PHP]: "PHP",
  };

  return typeMap[type] || "Unknown";
});

const environmentLabel = computed(() => {
  if (!props.deployment || props.loading) return "Unknown";
  const env = props.deployment.environment;
  if (!env) return "Unknown";

  const envMap: Record<number, string> = {
    [EnvEnum.PRODUCTION]: "Production",
    [EnvEnum.STAGING]: "Staging",
    [EnvEnum.DEVELOPMENT]: "Development",
  };

  return envMap[env] || "Unknown";
});

const lastDeployedAtDate = computed(() => {
  if (!props.deployment || props.loading) return new Date();
  if (props.deployment.lastDeployedAt) {
    return date(props.deployment.lastDeployedAt);
  }
  if (props.deployment.createdAt) {
    return date(props.deployment.createdAt);
  }
  return new Date();
});

const cleanRepositoryName = (url: string) => {
  if (!url) return "";
  try {
    const parsed = new URL(url);
    const repoPath = parsed.pathname.replace(/\.git$/, "").replace(/^\//, "");
    return repoPath || parsed.hostname;
  } catch (error) {
    return url.replace(/^https?:\/\//, "").replace(/\.git$/, "");
  }
};

const formatBuildTime = (seconds: number) => {
  if (!seconds || seconds === 0) return "0s";
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`;
};

const ENVIRONMENT_META = {
  [EnvEnum.PRODUCTION]: {
    label: "Production",
    chipClass: "bg-success/10 text-success ring-1 ring-success/20",
  },
  [EnvEnum.STAGING]: {
    label: "Staging",
    chipClass: "bg-warning/10 text-warning ring-1 ring-warning/20",
  },
  [EnvEnum.DEVELOPMENT]: {
    label: "Development",
    chipClass: "bg-info/10 text-info ring-1 ring-info/20",
  },
} as const;

const environmentMeta = computed(() => {
  if (!props.deployment || props.loading) {
    return {
      label: "Unknown",
      chipClass: "bg-surface-muted text-secondary ring-1 ring-border-muted",
    };
  }
  const env = props.deployment.environment;
  if (env && env in ENVIRONMENT_META) {
    return ENVIRONMENT_META[env as keyof typeof ENVIRONMENT_META];
  }
  return {
    label: "Unknown",
    chipClass: "bg-surface-muted text-secondary ring-1 ring-border-muted",
  };
});

const deploymentGroups = computed(() => {
  if (!props.deployment || props.loading) return [];
  return (props.deployment as any).groups || [];
});

const isMockDeployment = computed(
  () => props.deployment?.id?.toString().startsWith("mock-") ?? false
);

const showContainerStatus = computed(() => {
  if (!props.deployment || props.loading) return false;
  return (
    props.deployment.containersTotal &&
    props.deployment.containersTotal > 0
  );
});

const containerStatusVariant = computed<"success" | "warning" | "danger">(() => {
  if (!props.deployment || props.loading) return "danger";
  const running = props.deployment.containersRunning ?? 0;
  const total = props.deployment.containersTotal ?? 0;
  if (total === 0) return "danger";
  if (running === total) return "success";
  if (running === 0) return "danger";
  return "warning";
});

const showProgress = computed(() => {
  if (!props.deployment || props.loading) return false;
  return (
    props.deployment.status === DeploymentStatus.BUILDING ||
    props.deployment.status === DeploymentStatus.DEPLOYING ||
    (props.deployment.status === DeploymentStatus.FAILED && props.progressValue !== undefined && props.progressValue > 0)
  );
});

const isProgressFailed = computed(() => {
  if (!props.deployment || props.loading) return false;
  return props.deployment.status === DeploymentStatus.FAILED;
});

const progressClass = computed(() => {
  if (isProgressFailed.value) {
    return "border-danger/30 bg-danger/10 text-danger";
  }
  return statusMeta.value.progressClass || "border-warning/30 bg-warning/10 text-warning";
});

const progressTextClass = computed(() => {
  return isProgressFailed.value ? "text-danger" : "";
});

const progressBarBgClass = computed(() => {
  return isProgressFailed.value ? "bg-danger/20" : "bg-warning/20";
});

const progressBarFillClass = computed(() => {
  return isProgressFailed.value ? "bg-danger" : "bg-warning";
});

const detailUrl = computed(() => {
  const deployment = props.deployment;
  if (!deployment || isMockDeployment.value) return undefined;
  return `/deployments/${deployment.id}`;
});

const handleStart = async () => {
  const deployment = props.deployment;
  if (!deployment) return;
  if (isMockDeployment.value) {
    isActioning.value = true;
    deployment.status = DeploymentStatus.BUILDING;
    deployment.containersRunning = deployment.containersTotal ?? 0;
    setTimeout(() => {
      deployment.status = DeploymentStatus.RUNNING;
      deployment.containersRunning = deployment.containersTotal ?? 0;
      isActioning.value = false;
      emit("refresh");
    }, 600);
    return;
  }
  isActioning.value = true;
  try {
    await startDeployment(deployment.id, null);
    emit("refresh");
  } finally {
    isActioning.value = false;
  }
};

const handleStop = async () => {
  const deployment = props.deployment;
  if (!deployment) return;
  if (isMockDeployment.value) {
    isActioning.value = true;
    deployment.status = DeploymentStatus.STOPPED;
    deployment.containersRunning = 0;
    setTimeout(() => {
      isActioning.value = false;
      emit("refresh");
    }, 400);
    return;
  }
  isActioning.value = true;
  try {
    await stopDeployment(deployment.id, null);
    emit("refresh");
  } finally {
    isActioning.value = false;
  }
};

const handleRedeploy = async () => {
  const deployment = props.deployment;
  if (!deployment) return;
  if (isMockDeployment.value) {
    isActioning.value = true;
    deployment.status = DeploymentStatus.BUILDING;
    setTimeout(() => {
      deployment.status = DeploymentStatus.DEPLOYING;
    }, 300);
    setTimeout(() => {
      deployment.status = DeploymentStatus.RUNNING;
      isActioning.value = false;
      emit("refresh");
    }, 900);
    return;
  }
  isActioning.value = true;
  try {
    await redeployDeployment(deployment.id, null);
    emit("refresh");
  } finally {
    isActioning.value = false;
  }
};
</script>
