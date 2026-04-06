<template>
    <OuiStack gap="md">
      <!-- Live Metrics -->
      <LiveMetrics
        :is-streaming="isStreaming"
        :latest-metric="latestMetric"
        :current-cpu-usage="currentCpuUsage"
        :current-memory-usage="currentMemoryUsage"
        :current-network-rx="currentNetworkRx"
        :current-network-tx="currentNetworkTx"
      />

      <!-- Quick Info Bar: Domain + Environment + Type -->
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
            <!-- Domain -->
            <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
              <GlobeAltIcon class="h-4 w-4 text-accent-primary shrink-0" />
              <OuiText v-if="primaryDomain" size="sm" weight="medium" truncate class="font-mono">{{ primaryDomain }}</OuiText>
              <OuiText v-else size="sm" color="tertiary">No domain configured</OuiText>
              <OuiButton v-if="primaryDomain" variant="ghost" size="xs" @click="openDomain">
                <ArrowTopRightOnSquareIcon class="h-3 w-3" />
              </OuiButton>
            </OuiFlex>

            <!-- Badges -->
            <OuiFlex gap="xs" wrap="wrap" class="shrink-0">
              <OuiBadge :variant="getEnvironmentVariant(deployment.environment)" size="xs">
                {{ getEnvironmentLabel(deployment.environment) }}
              </OuiBadge>
              <OuiBadge variant="secondary" size="xs">
                {{ getTypeLabel((deployment as any).type) }}
              </OuiBadge>
              <OuiBadge v-if="deployment.buildStrategy !== undefined" variant="secondary" size="xs">
                {{ getBuildStrategyLabel(deployment.buildStrategy) }}
              </OuiBadge>
              <OuiBadge
                v-for="domain in getDisplayDomains(deployment.customDomains || [])"
                :key="domain.domain"
                :variant="getDomainStatusVariant(domain.status)"
                size="xs"
              >
                {{ domain.domain }}
              </OuiBadge>
            </OuiFlex>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>

      <!-- Configuration Grid -->
      <OuiGrid :cols="{ sm: 1, lg: 2 }" gap="sm">
        <!-- Infrastructure Card -->
        <OuiCard variant="outline">
          <OuiCardBody>
            <OuiStack gap="md">
              <OuiFlex align="center" gap="sm">
                <ServerStackIcon class="h-4 w-4 text-accent-secondary" />
                <OuiText size="sm" weight="semibold">Infrastructure</OuiText>
              </OuiFlex>

              <OuiStack gap="none" class="divide-y divide-border-default">
                <!-- Port -->
                <OuiFlex v-if="deployment.port" align="center" justify="between" gap="sm" class="py-2.5">
                  <OuiText size="xs" color="tertiary">Port</OuiText>
                  <OuiText size="sm" weight="medium" class="font-mono">:{{ deployment.port }}</OuiText>
                </OuiFlex>

                <!-- Containers -->
                <OuiFlex v-if="deployment.containersTotal !== undefined" align="center" justify="between" gap="sm" class="py-2.5">
                  <OuiText size="xs" color="tertiary">Containers</OuiText>
                  <OuiFlex align="center" gap="sm">
                    <!-- Container dots visualization -->
                    <OuiFlex gap="xs" align="center">
                      <span
                        v-for="i in deployment.containersTotal"
                        :key="i"
                        class="h-2 w-2 rounded-full transition-colors"
                        :class="i <= (deployment.containersRunning ?? 0) ? 'bg-success' : 'bg-border-strong'"
                      />
                    </OuiFlex>
                    <OuiText size="xs" color="tertiary">{{ deployment.containersRunning ?? 0 }}/{{ deployment.containersTotal }}</OuiText>
                  </OuiFlex>
                </OuiFlex>

                <!-- Health -->
                <OuiFlex align="center" justify="between" gap="sm" class="py-2.5">
                  <OuiText size="xs" color="tertiary">Health</OuiText>
                  <OuiFlex align="center" gap="xs">
                    <component v-if="deployment.healthStatus" :is="getHealthIcon(deployment.healthStatus)" :class="`h-3.5 w-3.5 ${getHealthIconClass(deployment.healthStatus)}`" />
                    <OuiText size="sm" weight="medium">{{ deployment.healthStatus || 'Unknown' }}</OuiText>
                  </OuiFlex>
                </OuiFlex>

                <!-- Build Time -->
                <OuiFlex align="center" justify="between" gap="sm" class="py-2.5">
                  <OuiText size="xs" color="tertiary">Last Build</OuiText>
                  <OuiText size="sm" weight="medium">{{ formatBuildTime(deployment.buildTime ?? 0) }}</OuiText>
                </OuiFlex>

                <!-- Bundle Size -->
                <OuiFlex v-if="deployment.size && deployment.size !== '--'" align="center" justify="between" gap="sm" class="py-2.5">
                  <OuiText size="xs" color="tertiary">Bundle Size</OuiText>
                  <OuiText size="sm" weight="medium"><OuiByte :value="deployment.size ?? 0" unit-display="short" /></OuiText>
                </OuiFlex>

                <!-- Storage -->
                <OuiFlex v-if="deployment.storageUsage !== undefined && deployment.storageUsage !== null && Number(deployment.storageUsage) > 0" align="center" justify="between" gap="sm" class="py-2.5">
                  <OuiText size="xs" color="tertiary">Storage</OuiText>
                  <OuiText size="sm" weight="medium"><OuiByte :value="deployment.storageUsage" /></OuiText>
                </OuiFlex>

                <!-- Groups -->
                <OuiFlex v-if="deployment.groups?.length" align="start" justify="between" gap="sm" class="py-2.5">
                  <OuiText size="xs" color="tertiary">Groups</OuiText>
                  <OuiFlex gap="xs" wrap="wrap" class="justify-end">
                    <OuiBadge v-for="group in deployment.groups" :key="group" variant="secondary" size="xs">{{ group }}</OuiBadge>
                  </OuiFlex>
                </OuiFlex>

                <!-- Created -->
                <OuiFlex align="center" justify="between" gap="sm" class="py-2.5">
                  <OuiText size="xs" color="tertiary">Created</OuiText>
                  <OuiText v-if="deployment.createdAt" size="sm" weight="medium">
                    <OuiRelativeTime :value="date(deployment.createdAt)" :style="'short'" />
                  </OuiText>
                  <OuiText v-else size="sm" color="tertiary">—</OuiText>
                </OuiFlex>
              </OuiStack>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Source Card -->
        <OuiCard variant="outline">
          <OuiCardBody>
            <OuiStack gap="md">
              <OuiFlex align="center" gap="sm">
                <CodeBracketSquareIcon class="h-4 w-4 text-accent-primary" />
                <OuiText size="sm" weight="semibold">Source</OuiText>
              </OuiFlex>

              <!-- Repository Info - prominent display -->
              <template v-if="deployment.repositoryUrl">
                <OuiCard variant="outline" class="bg-surface-muted/30">
                  <OuiCardBody class="!py-3 !px-4">
                    <OuiStack gap="sm">
                      <OuiFlex align="center" justify="between" gap="sm">
                        <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                          <Icon name="uil:github" class="h-4 w-4 text-secondary shrink-0" />
                          <OuiText size="sm" weight="medium" truncate class="font-mono">{{ repoDisplayName }}</OuiText>
                        </OuiFlex>
                        <OuiButton variant="ghost" size="xs" @click="openRepository">
                          <ArrowTopRightOnSquareIcon class="h-3 w-3" />
                        </OuiButton>
                      </OuiFlex>
                      <OuiFlex v-if="deployment.branch" align="center" gap="xs">
                        <svg class="h-3 w-3 text-tertiary shrink-0" viewBox="0 0 16 16" fill="currentColor"><path d="M9.5 3.25a2.25 2.25 0 1 1 3 2.122V6A2.5 2.5 0 0 1 10 8.5H6a1 1 0 0 0-1 1v1.128a2.251 2.251 0 1 1-1.5 0V5.372a2.25 2.25 0 1 1 1.5 0v1.836A2.492 2.492 0 0 1 6 7h4a1 1 0 0 0 1-1v-.628A2.25 2.25 0 0 1 9.5 3.25Z" /></svg>
                        <OuiText size="xs" color="tertiary" class="font-mono">{{ deployment.branch }}</OuiText>
                      </OuiFlex>
                    </OuiStack>
                  </OuiCardBody>
                </OuiCard>
              </template>

              <OuiStack gap="none" class="divide-y divide-border-default">
                <!-- Dockerfile Path -->
                <OuiFlex v-if="deployment.dockerfilePath" align="center" justify="between" gap="sm" class="py-2.5">
                  <OuiText size="xs" color="tertiary">Dockerfile</OuiText>
                  <OuiText size="sm" weight="medium" truncate class="font-mono">{{ deployment.dockerfilePath }}</OuiText>
                </OuiFlex>

                <!-- Compose File Path -->
                <OuiFlex v-if="deployment.composeFilePath" align="center" justify="between" gap="sm" class="py-2.5">
                  <OuiText size="xs" color="tertiary">Compose File</OuiText>
                  <OuiText size="sm" weight="medium" truncate class="font-mono">{{ deployment.composeFilePath }}</OuiText>
                </OuiFlex>

                <!-- Image -->
                <OuiFlex v-if="deployment.image" align="center" justify="between" gap="sm" class="py-2.5">
                  <OuiText size="xs" color="tertiary">Image</OuiText>
                  <OuiText size="sm" weight="medium" truncate class="font-mono">{{ deployment.image }}</OuiText>
                </OuiFlex>
              </OuiStack>

              <!-- No Source -->
              <OuiFlex v-if="!deployment.repositoryUrl && !deployment.image" align="center" justify="center" class="py-6">
                <OuiStack align="center" gap="sm">
                  <CodeBracketSquareIcon class="h-8 w-8 text-border-strong" />
                  <OuiText size="sm" color="tertiary">No source configured</OuiText>
                </OuiStack>
              </OuiFlex>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>
      </OuiGrid>

      <!-- Usage & Cost -->
      <UsageStatistics v-if="usageData" :usage-data="usageData" />
      <CostBreakdown v-if="usageData" :usage-data="usageData" />
    </OuiStack>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, watch } from "vue";
import {
  ArrowTopRightOnSquareIcon,
  HeartIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  XCircleIcon,
  GlobeAltIcon,
  ServerStackIcon,
  CodeBracketSquareIcon,
} from "@heroicons/vue/24/outline";
import { type Deployment, type StreamDeploymentMetricsRequest } from "@obiente/proto";
import {
  DeploymentType,
  DeploymentStatus,
  Environment as EnvEnum,
  BuildStrategy,
  DeploymentService,
} from "@obiente/proto";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { date } from "@obiente/proto/utils";
import { useConnectClient } from "~/lib/connect-client";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import OuiByte from "~/components/oui/Byte.vue";
import UsageStatistics from "~/components/shared/UsageStatistics.vue";
import CostBreakdown from "~/components/shared/CostBreakdown.vue";
import LiveMetrics from "~/components/shared/LiveMetrics.vue";
import { isDefaultObienteDomain } from "~/utils/domains";

interface Props {
  deployment: Deployment;
  organizationId?: string;
}

const props = defineProps<Props>();

defineEmits<{
  navigate: [tab: string];
}>();

const client = useConnectClient(DeploymentService);

// Fetch deployment usage data
const { data: usageData } = await useClientFetch(
  () => `deployment-usage-${props.deployment.id}`,
  async () => {
    if (!props.deployment?.id || !props.organizationId) return null;
    try {
      const res = await client.getDeploymentUsage({
        deploymentId: props.deployment.id,
        organizationId: props.organizationId,
      });
      return res;
    } catch (err) {
      console.error("Failed to fetch deployment usage:", err);
      return null;
    }
  },
  { watch: [() => props.deployment.id, () => props.organizationId], server: false }
);

// Fetch routing rules to get the primary domain
const { data: routingData } = await useClientFetch(
  () => `deployment-routings-${props.deployment.id}`,
  async () => {
    if (!props.deployment?.id || !props.organizationId) return null;
    try {
      const res = await client.getDeploymentRoutings({
        deploymentId: props.deployment.id,
        organizationId: props.organizationId,
      });
      return res;
    } catch (err) {
      console.error("Failed to fetch routing rules:", err);
      return null;
    }
  },
  { watch: [() => props.deployment.id, () => props.organizationId], server: false }
);

// Prefers custom domains over the managed default *.my.obiente.cloud domain.
const primaryDomain = computed(() => {
  const rules = routingData.value?.rules || [];
  if (rules.length === 0) {
    // Fallback to deployment.domain if no routing rules
    return props.deployment.domain || "";
  }

  // Get the first routing rule's domain
  const firstRuleDomain = rules[0]?.domain || "";
  
  if (firstRuleDomain && !isDefaultObienteDomain(firstRuleDomain, ["deploy"])) {
    return firstRuleDomain;
  }

  // If the first rule is the managed default domain, look for a custom domain in other rules.
  const customDomain = rules.find(
    (rule) => rule.domain && !isDefaultObienteDomain(rule.domain, ["deploy"])
  );
  
  if (customDomain?.domain) {
    return customDomain.domain;
  }

  // If no custom domain is available, use the first routed domain.
  if (firstRuleDomain) {
    return firstRuleDomain;
  }

  // Final fallback to deployment.domain
  return props.deployment.domain || "";
});

// Live metrics state
const isStreaming = ref(false);
const latestMetric = ref<any>(null);
const streamController = ref<AbortController | null>(null);

// Computed metrics from latest data
const currentCpuUsage = computed(() => {
  return latestMetric.value?.cpuUsagePercent ?? 0;
});

const currentMemoryUsage = computed(() => {
  return latestMetric.value?.memoryUsageBytes ?? 0;
});

const currentNetworkRx = computed(() => {
  return latestMetric.value?.networkRxBytes ?? 0;
});

const currentNetworkTx = computed(() => {
  return latestMetric.value?.networkTxBytes ?? 0;
});

// Format bytes helper
const formatBytes = (bytes: number | bigint) => {
  const b = Number(bytes);
  if (b === 0) return "0 B";
  if (b < 1024) return `${b} B`;
  if (b < 1024 * 1024) return `${(b / 1024).toFixed(2)} KB`;
  if (b < 1024 * 1024 * 1024) return `${(b / (1024 * 1024)).toFixed(2)} MB`;
  return `${(b / (1024 * 1024 * 1024)).toFixed(2)} GB`;
};


// Start streaming metrics
const startStreaming = async () => {
  if (isStreaming.value || streamController.value || !props.deployment?.id) {
    return;
  }

  isStreaming.value = true;
  streamController.value = new AbortController();

  try {
    const request: Partial<StreamDeploymentMetricsRequest> = {
      deploymentId: props.deployment.id,
      organizationId: props.organizationId || "",
      intervalSeconds: 5,
      aggregate: true, // Get aggregated metrics for all containers
    };

    if (!request.organizationId) {
      console.warn("No organizationId provided for metrics streaming");
      isStreaming.value = false;
      streamController.value = null;
      return;
    }

    const stream = await (client as any).streamDeploymentMetrics(request, {
      signal: streamController.value.signal,
    });

    for await (const metric of stream) {
      if (streamController.value?.signal.aborted) {
        break;
      }
      latestMetric.value = metric;
    }
  } catch (err: unknown) {
    if ((err as any).name === "AbortError") {
      return;
    }
    // Suppress "missing trailer" errors
    const isMissingTrailerError =
      (err as Error).message?.toLowerCase().includes("missing trailer") ||
      (err as Error).message?.toLowerCase().includes("trailer") ||
      (err as any).code === "unknown";

    if (!isMissingTrailerError) {
      console.error("Failed to stream metrics:", err);
    }
  } finally {
    isStreaming.value = false;
    streamController.value = null;
  }
};

// Stop streaming
const stopStreaming = () => {
  if (streamController.value) {
    streamController.value.abort();
    streamController.value = null;
  }
  isStreaming.value = false;
};

// Start streaming when component mounts if deployment is running
onMounted(() => {
  if (props.deployment?.status === DeploymentStatus.RUNNING) {
    startStreaming();
  }
});

// Watch deployment status and start/stop streaming accordingly
watch(
  () => props.deployment?.status,
  (status) => {
    if (status === DeploymentStatus.RUNNING && !isStreaming.value) {
      startStreaming();
    } else if (status !== DeploymentStatus.RUNNING && isStreaming.value) {
      stopStreaming();
    }
  }
);

// Clean up on unmount
onUnmounted(() => {
  stopStreaming();
});

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
    case DeploymentType.PYTHON:
      return "Python";
    case DeploymentType.RUBY:
      return "Ruby";
    case DeploymentType.RUST:
      return "Rust";
    case DeploymentType.JAVA:
      return "Java";
    case DeploymentType.PHP:
      return "PHP";
    case DeploymentType.GENERIC:
      return "Generic";
    default:
      return "Custom";
  }
};

const getBuildStrategyLabel = (strategy: BuildStrategy | number | undefined) => {
  switch (strategy) {
    case BuildStrategy.RAILPACK:
      return "Railpack";
    case BuildStrategy.NIXPACKS:
      return "Nixpacks";
    case BuildStrategy.DOCKERFILE:
      return "Dockerfile";
    case BuildStrategy.PLAIN_COMPOSE:
      return "Docker Compose";
    case BuildStrategy.COMPOSE_REPO:
      return "Compose from Repo";
    case BuildStrategy.STATIC_SITE:
      return "Static Site";
    default:
      return "Unknown";
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

const getEnvironmentVariant = (env: string | EnvEnum | number): "success" | "warning" | "secondary" => {
  if (typeof env === "number") {
    switch (env) {
      case EnvEnum.PRODUCTION:
        return "success";
      case EnvEnum.STAGING:
        return "warning";
      case EnvEnum.DEVELOPMENT:
        return "secondary";
      default:
        return "secondary";
    }
  }
  return "secondary";
};

const getDisplayDomains = (customDomains: string[]) => {
  return customDomains.map((entry) => {
    const parts = entry.split(":");
    const domain = parts[0] || "";
    let status = "pending";
    
    if (parts.length >= 4 && parts[1] === "token" && parts[3]) {
      status = parts[3];
    } else if (parts.length >= 2 && parts[1] === "verified") {
      status = "verified";
    }
    
    return { domain, status };
  });
};

const getDomainStatusVariant = (status: string): "success" | "warning" | "danger" | "secondary" => {
  switch (status) {
    case "verified":
      return "success";
    case "failed":
      return "danger";
    case "expired":
      return "warning";
    default:
      return "secondary";
  }
};

const getStatusLabel = (status: DeploymentStatus | number) => {
  switch (status) {
    case DeploymentStatus.RUNNING:
      return "Running";
    case DeploymentStatus.STOPPED:
      return "Stopped";
    case DeploymentStatus.BUILDING:
      return "Building";
    case DeploymentStatus.DEPLOYING:
      return "Deploying";
    case DeploymentStatus.FAILED:
      return "Failed";
    case DeploymentStatus.CREATED:
      return "Created";
    default:
      return "Unknown";
  }
};

const getStatusDotClass = (status: DeploymentStatus | number) => {
  switch (status) {
    case DeploymentStatus.RUNNING:
      return "bg-success animate-pulse";
    case DeploymentStatus.STOPPED:
      return "bg-danger";
    case DeploymentStatus.BUILDING:
    case DeploymentStatus.DEPLOYING:
      return "bg-warning animate-pulse";
    case DeploymentStatus.FAILED:
      return "bg-danger";
    default:
      return "bg-secondary";
  }
};

const getContainerStatusVariant = () => {
  const running = props.deployment.containersRunning ?? 0;
  const total = props.deployment.containersTotal ?? 0;
  
  if (running === 0) return "danger";
  if (running === total) return "success";
  return "warning";
};

const getContainerStatusLabel = () => {
  const running = props.deployment.containersRunning ?? 0;
  const total = props.deployment.containersTotal ?? 0;
  
  if (running === 0) return "Stopped";
  if (running === total) return "Running";
  return "Partial";
};

const formatBuildTime = (seconds: number) => {
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`;
};

const getHealthIcon = (healthStatus: string) => {
  const status = healthStatus.toLowerCase();
  if (status.includes("healthy") || status === "ok" || status === "up") {
    return CheckCircleIcon;
  }
  if (status.includes("unhealthy") || status.includes("down") || status === "error") {
    return XCircleIcon;
  }
  if (status.includes("warning") || status === "degraded") {
    return ExclamationTriangleIcon;
  }
  return HeartIcon;
};

const getHealthIconClass = (healthStatus: string) => {
  const status = healthStatus.toLowerCase();
  if (status.includes("healthy") || status === "ok" || status === "up") {
    return "text-success";
  }
  if (status.includes("unhealthy") || status.includes("down") || status === "error") {
    return "text-danger";
  }
  if (status.includes("warning") || status === "degraded") {
    return "text-warning";
  }
  return "text-secondary";
};

const getHealthVariant = (healthStatus: string): "success" | "warning" | "danger" | "secondary" => {
  const status = healthStatus.toLowerCase();
  if (status.includes("healthy") || status === "ok" || status === "up") {
    return "success";
  }
  if (status.includes("unhealthy") || status.includes("down") || status === "error") {
    return "danger";
  }
  if (status.includes("warning") || status === "degraded") {
    return "warning";
  }
  return "secondary";
};

const getHealthLabel = (healthStatus: string) => {
  const status = healthStatus.toLowerCase();
  if (status.includes("healthy") || status === "ok" || status === "up") {
    return "Healthy";
  }
  if (status.includes("unhealthy") || status.includes("down") || status === "error") {
    return "Unhealthy";
  }
  if (status.includes("warning") || status === "degraded") {
    return "Degraded";
  }
  return "Unknown";
};

const formatDate = (timestamp: Timestamp | null | undefined) => {
  if (!timestamp) return "";
  const d = date(timestamp);
  if (!d) return "";
  return d.toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
};

const openDomain = () => {
  if (primaryDomain.value) {
    window.open(`https://${primaryDomain.value}`, "_blank");
  }
};

const openRepository = () => {
  if (props.deployment.repositoryUrl) {
    window.open(props.deployment.repositoryUrl, "_blank");
  }
};

const repoDisplayName = computed(() => {
  const url = props.deployment.repositoryUrl || '';
  // Extract "owner/repo" from GitHub-style URLs
  const match = url.match(/(?:github\.com|gitlab\.com|bitbucket\.org)[/:](.+?)(?:\.git)?$/);
  return match ? match[1] : url;
});
</script>
