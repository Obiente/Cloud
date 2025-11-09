<template>
  <ResourceCard
    :title="deployment.name"
    :subtitle="deployment.domain"
    :status-meta="statusMeta"
    :created-at="lastDeployedAtDate"
    :detail-url="`/deployments/${deployment.id}`"
    :is-actioning="isActioning || showProgress"
  >
    <template #subtitle>
      <OuiStack gap="xs">
        <a
          v-if="deployment.domain"
          :href="`https://${deployment.domain}`"
          target="_blank"
          rel="noopener noreferrer"
          class="inline-flex items-center gap-1.5 text-sm text-secondary hover:text-primary transition-colors"
          @click.stop
        >
          <span class="truncate">{{ deployment.domain }}</span>
          <ArrowTopRightOnSquareIcon class="h-3.5 w-3.5" />
        </a>
      </OuiStack>
    </template>

    <template #actions>
      <OuiFlex gap="xs" wrap="wrap">
        <OuiBadge
          v-if="showContainerStatus"
          :variant="containerStatusVariant"
          size="sm"
        >
          {{ deployment.containersRunning ?? 0 }}/{{ deployment.containersTotal }} running
        </OuiBadge>
        <OuiBadge
          v-for="(group, idx) in deploymentGroups"
          :key="idx"
          variant="secondary"
          size="sm"
        >
          {{ group }}
        </OuiBadge>
        <OuiButton
          v-if="deployment.status === DeploymentStatus.RUNNING"
          variant="ghost"
          size="sm"
          icon-only
          @click.stop="handleStop"
          title="Stop"
        >
          <StopIcon class="h-4 w-4" />
        </OuiButton>
        <OuiButton
          v-if="deployment.status === DeploymentStatus.STOPPED"
          variant="ghost"
          size="sm"
          icon-only
          @click.stop="handleStart"
          title="Start"
        >
          <PlayIcon class="h-4 w-4" />
        </OuiButton>
        <OuiButton
          variant="ghost"
          size="sm"
          icon-only
          @click.stop="handleRedeploy"
          title="Redeploy"
        >
          <ArrowPathIcon class="h-4 w-4" />
        </OuiButton>
      </OuiFlex>
    </template>

    <template #resources>
      <OuiStack gap="md">
        <OuiFlex justify="between" align="center">
          <OuiFlex align="center" gap="sm">
            <OuiBox
              p="xs"
              rounded="lg"
              bg="surface-muted"
              class="bg-surface-muted/50 ring-1 ring-border-muted"
            >
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
          <OuiBox
            v-if="deployment.repositoryUrl"
            p="sm"
            rounded="lg"
            w="4xl"
            bg="surface-muted"
            class="bg-surface-muted/30 ring-1 ring-border-muted"
          >
            <OuiFlex align="center" gap="sm" class="min-w-0">
              <Icon
                name="uil:github"
                class="h-4 w-4 text-secondary shrink-0"
              />
              <OuiText
                size="xs"
                color="secondary"
                truncate
                class="font-mono"
              >
                {{ cleanRepositoryName(deployment.repositoryUrl) }}
              </OuiText>
            </OuiFlex>
          </OuiBox>
          <span
            class="inline-flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-semibold uppercase tracking-wide ml-auto"
            :class="environmentMeta.chipClass"
          >
            <CpuChipIcon class="h-3.5 w-3.5" />
            {{ environmentMeta.label }}
          </span>
        </OuiFlex>

        <OuiGrid cols="2" gap="sm">
          <OuiBox
            p="sm"
            rounded="lg"
            bg="surface-muted"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiStack gap="xs">
              <OuiText
                size="xs"
                weight="bold"
                transform="uppercase"
                color="secondary"
                class="tracking-wider"
              >
                Build Time
              </OuiText>
              <OuiText size="lg" weight="bold" color="primary">
                {{ formatBuildTime(deployment.buildTime ?? 0) }}
              </OuiText>
            </OuiStack>
          </OuiBox>

          <OuiBox
            p="sm"
            rounded="lg"
            bg="surface-muted"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiStack gap="xs">
              <OuiText
                size="xs"
                weight="bold"
                transform="uppercase"
                color="secondary"
                class="tracking-wider"
              >
                Bundle Size
              </OuiText>
              <OuiText size="lg" weight="bold" color="primary">
                <OuiByte
                  :value="deployment.size ?? 0"
                  unit-display="short"
                />
              </OuiText>
            </OuiStack>
          </OuiBox>
        </OuiGrid>
      </OuiStack>
    </template>

    <template #info>
      <!-- Build Progress Status -->
      <OuiBox
        v-if="showProgress"
        p="md"
        rounded="xl"
        class="border backdrop-blur-sm"
        :class="progressClass"
      >
        <OuiStack gap="sm">
          <OuiFlex
            align="center"
            gap="sm"
            class="text-xs font-bold uppercase tracking-wider"
            :class="progressTextClass"
          >
            <Cog6ToothIcon
              v-if="!isProgressFailed"
              class="h-4 w-4 animate-spin"
            />
            <ExclamationTriangleIcon
              v-else
              class="h-4 w-4 text-danger"
            />
            <span :class="progressTextClass">
              {{ props.progressPhase || "Starting deployment..." }}
            </span>
          </OuiFlex>
          <div
            class="relative h-2 w-full overflow-hidden rounded-full"
            :class="progressBarBgClass"
          >
            <div
              v-if="isProgressFailed"
              class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
              :class="progressBarFillClass"
              :style="{ width: '100%' }"
            />
            <div
              v-else
              class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
              :class="progressBarFillClass"
              :style="{ width: `${props.progressValue || 0}%` }"
            />
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
  } from "@obiente/proto";
  import { date } from "@obiente/proto/utils";
  import { useDeploymentActions } from "~/composables/useDeploymentActions";
  import ResourceCard from "~/components/shared/ResourceCard.vue";
  import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
  import OuiByte from "~/components/oui/Byte.vue";

  interface Props {
    deployment: Deployment;
    progressValue?: number;
    progressPhase?: string;
  }

  const props = withDefaults(defineProps<Props>(), {
    progressValue: 0,
    progressPhase: "",
  });

  const emit = defineEmits<{
    refresh: [];
  }>();

  const { startDeployment, stopDeployment, redeployDeployment } = useDeploymentActions();
  const isActioning = ref(false);

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
    const status = props.deployment.status as DeploymentStatus;
    if (status in STATUS_META) {
      return STATUS_META[status as keyof typeof STATUS_META];
    }
    return STATUS_META[DeploymentStatus.STOPPED];
  });

  const typeLabel = computed(() => {
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
    return (props.deployment as any).groups || [];
  });

  const showContainerStatus = computed(() => {
    return (
      props.deployment.containersTotal &&
      props.deployment.containersTotal > 0 &&
      (props.deployment.containersRunning ?? 0) > 0 &&
      (props.deployment.containersRunning ?? 0) < props.deployment.containersTotal
    );
  });

  const containerStatusVariant = computed<"success" | "warning" | "danger">(() => {
    const running = props.deployment.containersRunning ?? 0;
    const total = props.deployment.containersTotal ?? 0;
    if (total === 0) return "danger";
    if (running === total) return "success";
    if (running === 0) return "danger";
    return "warning";
  });

  const showProgress = computed(() => {
    return (
      props.deployment.status === DeploymentStatus.BUILDING ||
      props.deployment.status === DeploymentStatus.DEPLOYING ||
      (props.deployment.status === DeploymentStatus.FAILED && props.progressValue !== undefined && props.progressValue > 0)
    );
  });

  const isProgressFailed = computed(() => {
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

  const handleStart = async () => {
    isActioning.value = true;
    try {
      await startDeployment(props.deployment.id, null);
      emit("refresh");
    } finally {
      isActioning.value = false;
    }
  };

  const handleStop = async () => {
    isActioning.value = true;
    try {
      await stopDeployment(props.deployment.id, null);
      emit("refresh");
    } finally {
      isActioning.value = false;
    }
  };

  const handleRedeploy = async () => {
    isActioning.value = true;
    try {
      await redeployDeployment(props.deployment.id, null);
      emit("refresh");
    } finally {
      isActioning.value = false;
    }
  };
</script>

