<template>
  <OuiContainer size="full" py="xl">
    <OuiStack gap="xl">
      <!-- Page Header -->
      <OuiFlex justify="between" align="center" wrap="wrap" gap="lg">
        <OuiStack gap="xs" class="flex-1">
          <OuiText as="h1" size="3xl" weight="bold" color="primary"
            >Dashboard</OuiText
          >
          <OuiText color="secondary"
            >Comprehensive overview of your cloud infrastructure, usage metrics, and resource health.</OuiText
          >
        </OuiStack>

        <OuiFlex gap="sm" align="center" wrap="wrap">
          <OuiButton variant="ghost" size="sm" @click="retryLoad" class="gap-2">
            <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': isLoading }" />
            Refresh
          </OuiButton>
          <OuiButton
            color="primary"
            size="sm"
            class="gap-2 shadow-md"
            @click="navigateTo('/deployments')"
          >
            <RocketLaunchIcon class="h-4 w-4" />
            New Deployment
          </OuiButton>
        </OuiFlex>
      </OuiFlex>

      <!-- Enhanced KPI Overview -->
      <OuiGrid cols="1" cols-sm="2" cols-lg="4" gap="lg">
        <OuiCard
          v-for="card in kpiCards"
          :key="card.label"
          variant="default"
          hoverable
          class="cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl"
          @click="card.href && navigateTo(card.href)"
        >
          <OuiCardBody>
            <OuiStack gap="sm">
              <OuiFlex align="center" justify="between" gap="md">
                <OuiBox
                  p="sm"
                  rounded="lg"
                  :class="card.iconBg"
                  class="flex items-center justify-center"
                >
                  <component
                    :is="card.icon"
                    class="h-6 w-6"
                    :class="card.iconColor"
                  />
                </OuiBox>
                <OuiBadge v-if="card.badge" :variant="card.badgeVariant" size="sm">
                  {{ card.badge }}
                </OuiBadge>
              </OuiFlex>
              <OuiStack gap="xs">
                <OuiSkeleton
                  v-if="isLoading"
                  width="3.5rem"
                  height="1.5rem"
                  variant="text"
                />
                <OuiText
                  v-else
                  as="h3"
                  size="2xl"
                  weight="bold"
                  color="primary"
                  >{{ card.value }}</OuiText
                >
                <OuiText size="sm" color="secondary">{{ card.label }}</OuiText>
                <OuiText v-if="card.subtitle" size="xs" color="secondary" class="mt-1">
                  {{ card.subtitle }}
                </OuiText>
              </OuiStack>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>
      </OuiGrid>

      <!-- Usage Metrics Section -->
      <OuiCard>
        <OuiCardHeader>
          <OuiFlex align="center" justify="between">
            <OuiStack gap="xs">
              <OuiText as="h2" class="oui-card-title">Resource Usage</OuiText>
              <OuiText size="sm" color="secondary">
                Current month usage and estimated costs
              </OuiText>
            </OuiStack>
            <OuiButton
              variant="ghost"
              size="sm"
              @click="navigateTo('/organizations?tab=billing')"
            >
              View Details
            </OuiButton>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody>
          <template v-if="isLoadingUsage">
            <OuiGrid cols="1" cols-md="2" cols-lg="4" gap="lg">
              <OuiStack v-for="i in 4" :key="i" gap="sm">
                <OuiSkeleton width="6rem" height="1rem" variant="text" />
                <OuiSkeleton width="100%" height="0.5rem" variant="rectangle" rounded />
                <OuiSkeleton width="4rem" height="0.75rem" variant="text" />
              </OuiStack>
            </OuiGrid>
          </template>
          <template v-else-if="usageData && usageData.current && usageData.quota">
            <OuiGrid cols="1" cols-md="2" cols-lg="4" gap="lg">
              <!-- CPU Usage -->
              <OuiStack gap="sm">
                <OuiFlex align="center" justify="between">
                  <OuiText size="sm" weight="medium" color="primary">CPU</OuiText>
                  <OuiText size="xs" color="secondary">
                    {{ formatCPUUsage(Number(usageData.current.cpuCoreSeconds)) }}
                  </OuiText>
                </OuiFlex>
                <OuiBox
                  w="full"
                  class="h-0.5 bg-surface-muted overflow-hidden rounded-full"
                >
                  <OuiBox
                    class="h-full bg-accent-primary transition-all duration-300"
                    :style="{ width: `${cpuUsagePercent}%` }"
                  />
                </OuiBox>
                <OuiText size="xs" color="secondary">
                  {{ formatQuota(Number(usageData.current.cpuCoreSeconds), Number(usageData.quota.cpuCoreSecondsMonthly)) }}
                </OuiText>
              </OuiStack>

              <!-- Memory Usage -->
              <OuiStack gap="sm">
                <OuiFlex align="center" justify="between">
                  <OuiText size="sm" weight="medium" color="primary">Memory</OuiText>
                  <OuiText size="xs" color="secondary">
                    {{ formatBytes(Number(usageData.current.memoryByteSeconds) / 3600) }}/hr avg
                  </OuiText>
                </OuiFlex>
                <OuiBox
                  w="full"
                  class="h-0.5 bg-surface-muted overflow-hidden rounded-full"
                >
                  <OuiBox
                    class="h-full bg-success transition-all duration-300"
                    :style="{ width: `${memoryUsagePercent}%` }"
                  />
                </OuiBox>
                <OuiText size="xs" color="secondary">
                  {{ formatQuota(Number(usageData.current.memoryByteSeconds), Number(usageData.quota.memoryByteSecondsMonthly)) }}
                </OuiText>
              </OuiStack>

              <!-- Bandwidth Usage -->
              <OuiStack gap="sm">
                <OuiFlex align="center" justify="between">
                  <OuiText size="sm" weight="medium" color="primary">Bandwidth</OuiText>
                  <OuiText size="xs" color="secondary">
                    {{ formatBytes(Number(usageData.current.bandwidthRxBytes) + Number(usageData.current.bandwidthTxBytes)) }}
                  </OuiText>
                </OuiFlex>
                <OuiBox
                  w="full"
                  class="h-0.5 bg-surface-muted overflow-hidden rounded-full"
                >
                  <OuiBox
                    class="h-full bg-accent-secondary transition-all duration-300"
                    :style="{ width: `${bandwidthUsagePercent}%` }"
                  />
                </OuiBox>
                <OuiText size="xs" color="secondary">
                  {{ formatQuota(Number(usageData.current.bandwidthRxBytes) + Number(usageData.current.bandwidthTxBytes), Number(usageData.quota.bandwidthBytesMonthly)) }}
                </OuiText>
              </OuiStack>

              <!-- Storage Usage -->
              <OuiStack gap="sm">
                <OuiFlex align="center" justify="between">
                  <OuiText size="sm" weight="medium" color="primary">Storage</OuiText>
                  <OuiText size="xs" color="secondary">
                    {{ formatBytes(Number(usageData.current.storageBytes)) }}
                  </OuiText>
                </OuiFlex>
                <OuiBox
                  w="full"
                  class="h-0.5 bg-surface-muted overflow-hidden rounded-full"
                >
                  <OuiBox
                    class="h-full bg-warning transition-all duration-300"
                    :style="{ width: `${storageUsagePercent}%` }"
                  />
                </OuiBox>
                <OuiText size="xs" color="secondary">
                  {{ formatQuota(Number(usageData.current.storageBytes), Number(usageData.quota.storageBytes)) }}
                </OuiText>
              </OuiStack>
            </OuiGrid>
          </template>
          <OuiText v-else size="sm" color="secondary" class="text-center py-4">
            Usage data unavailable
          </OuiText>
        </OuiCardBody>
      </OuiCard>

      <!-- Cost Breakdown & Health Row -->
      <OuiGrid cols="1" cols-lg="2" gap="xl">
        <!-- Cost Breakdown -->
        <OuiCard>
          <OuiCardHeader>
            <OuiFlex align="center" justify="between">
              <OuiStack gap="xs">
                <OuiText as="h2" class="oui-card-title">Cost Breakdown</OuiText>
                <OuiText size="xs" color="secondary">
                  Estimated monthly costs by resource
                </OuiText>
              </OuiStack>
            </OuiFlex>
          </OuiCardHeader>
          <OuiCardBody>
            <template v-if="isLoadingUsage">
              <OuiStack gap="md">
                <OuiSkeleton width="100%" height="3rem" variant="rectangle" rounded />
                <OuiSkeleton width="100%" height="3rem" variant="rectangle" rounded />
                <OuiSkeleton width="100%" height="3rem" variant="rectangle" rounded />
              </OuiStack>
            </template>
            <template v-else-if="usageData && usageData.estimatedMonthly">
              <OuiStack gap="md">
                <OuiFlex align="center" justify="between" class="pb-3 border-b border-border-muted">
                  <OuiStack gap="xs">
                    <OuiText size="sm" color="secondary">Total Estimated</OuiText>
                    <OuiText size="2xl" weight="bold" color="primary">
                      {{ formatCurrency(Number(usageData.estimatedMonthly.estimatedCostCents) / 100) }}
                    </OuiText>
                  </OuiStack>
                  <OuiBadge variant="secondary">
                    {{ usageData.month }}
                  </OuiBadge>
                </OuiFlex>
                <OuiStack gap="sm">
                  <OuiBox
                    v-for="cost in costBreakdown"
                    :key="cost.label"
                    p="sm"
                    rounded="lg"
                    class="bg-surface-muted/40 ring-1 ring-border-muted"
                  >
                    <OuiFlex align="center" justify="between" gap="md">
                      <OuiFlex align="center" gap="sm" class="flex-1">
                        <OuiBox
                          class="w-3 h-3 rounded-full"
                          :class="cost.color"
                        />
                        <OuiText size="sm" weight="medium" color="primary">
                          {{ cost.label }}
                        </OuiText>
                      </OuiFlex>
                      <OuiText size="sm" weight="semibold" color="primary">
                        {{ cost.value }}
                      </OuiText>
                    </OuiFlex>
                  </OuiBox>
                </OuiStack>
                <OuiButton
                  variant="ghost"
                  size="sm"
                  class="self-start mt-2"
                  @click="navigateTo('/organizations?tab=billing')"
                >
                  View billing details →
                </OuiButton>
              </OuiStack>
            </template>
            <OuiText v-else size="sm" color="secondary" class="text-center py-4">
              Cost data unavailable
            </OuiText>
          </OuiCardBody>
        </OuiCard>

        <!-- Enhanced Health & Alerts -->
        <OuiCard>
          <OuiCardHeader>
            <OuiFlex align="center" justify="between">
              <OuiStack gap="xs">
                <OuiText as="h2" class="oui-card-title">Deployment Health</OuiText>
                <OuiText size="xs" color="secondary">
                  Status overview of all deployments
                </OuiText>
              </OuiStack>
              <OuiBadge :variant="allHealthy ? 'success' : 'warning'">{{
                allHealthy ? "All Healthy" : "Issues Detected"
              }}</OuiBadge>
            </OuiFlex>
          </OuiCardHeader>
          <OuiCardBody>
            <template v-if="isLoading">
              <OuiStack gap="sm">
                <OuiSkeleton width="80%" height="1rem" variant="text" />
                <OuiSkeleton width="70%" height="1rem" variant="text" />
                <OuiSkeleton width="60%" height="1rem" variant="text" />
              </OuiStack>
            </template>
            <template v-else>
              <OuiGrid cols="2" gap="sm" class="mb-4">
                <OuiBox
                  p="md"
                  rounded="lg"
                  class="bg-success/10 ring-1 ring-success/20"
                >
                  <OuiStack align="center" gap="xs">
                    <OuiText size="2xl" weight="bold" class="text-success">
                      {{ runningCount }}
                    </OuiText>
                    <OuiText size="xs" color="secondary">Running</OuiText>
                  </OuiStack>
                </OuiBox>
                <OuiBox
                  p="md"
                  rounded="lg"
                  class="bg-warning/10 ring-1 ring-warning/20"
                >
                  <OuiStack align="center" gap="xs">
                    <OuiText size="2xl" weight="bold" class="text-warning">
                      {{ buildingCount }}
                    </OuiText>
                    <OuiText size="xs" color="secondary">Building</OuiText>
                  </OuiStack>
                </OuiBox>
                <OuiBox
                  p="md"
                  rounded="lg"
                  class="bg-secondary/10 ring-1 ring-secondary/20"
                >
                  <OuiStack align="center" gap="xs">
                    <OuiText size="2xl" weight="bold" class="text-secondary">
                      {{ stoppedCount }}
                    </OuiText>
                    <OuiText size="xs" color="secondary">Stopped</OuiText>
                  </OuiStack>
                </OuiBox>
                <OuiBox
                  p="md"
                  rounded="lg"
                  class="bg-danger/10 ring-1 ring-danger/20"
                >
                  <OuiStack align="center" gap="xs">
                    <OuiText size="2xl" weight="bold" class="text-danger">
                      {{ errorCount }}
                    </OuiText>
                    <OuiText size="xs" color="secondary">Errors</OuiText>
                  </OuiStack>
                </OuiBox>
              </OuiGrid>
              <OuiStack v-if="attentionDeployments.length > 0" gap="sm">
                <OuiText size="sm" weight="medium" color="primary" class="mb-2">
                  Requires Attention
                </OuiText>
                <OuiBox
                  v-for="d in attentionDeployments.slice(0, 3)"
                  :key="d.id"
                  p="sm"
                  rounded="lg"
                  class="ring-1 ring-border-muted bg-surface-muted/40 cursor-pointer hover:bg-surface-muted transition-colors"
                  @click="navigateTo(`/deployments/${d.id}`)"
                >
                  <OuiFlex justify="between" align="center" gap="md">
                    <OuiStack gap="xs" class="min-w-0 flex-1">
                      <OuiText size="sm" weight="medium" class="truncate">{{
                        d.name
                      }}</OuiText>
                      <OuiText size="xs" color="secondary" class="truncate">{{
                        d.domain
                      }}</OuiText>
                    </OuiStack>
                    <OuiBadge :variant="statusVariant(d.status)">{{
                      d.status
                    }}</OuiBadge>
                  </OuiFlex>
                </OuiBox>
                <OuiButton
                  variant="ghost"
                  size="sm"
                  class="self-start mt-2"
                  @click="navigateTo('/deployments')"
                >
                  View all deployments →
                </OuiButton>
              </OuiStack>
              <OuiBox v-else p="md" rounded="lg" class="bg-success/5 border border-success/20">
                <OuiFlex align="center" gap="sm">
                  <CheckCircleIcon class="h-5 w-5 text-success" />
                  <OuiText size="sm" color="secondary">All deployments are running smoothly</OuiText>
                </OuiFlex>
              </OuiBox>
            </template>
          </OuiCardBody>
        </OuiCard>
      </OuiGrid>

      <!-- Recent Deployments -->
      <OuiCard>
        <OuiCardHeader>
          <OuiFlex align="center" justify="between">
            <OuiStack gap="xs">
              <OuiText as="h2" class="oui-card-title">Recent Deployments</OuiText>
              <OuiText size="xs" color="secondary">
                Latest {{ recentDeployments.length }} deployments across your environments
              </OuiText>
            </OuiStack>
            <OuiButton
              variant="ghost"
              size="sm"
              @click="navigateTo('/deployments')"
            >
              View All →
            </OuiButton>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiStack v-if="isLoading" gap="md">
            <OuiBox
              v-for="i in 5"
              :key="i"
              p="md"
              rounded="lg"
              border="1"
              borderColor="muted"
            >
              <OuiFlex justify="between" align="center" gap="md">
                <OuiStack gap="xs" class="flex-1">
                  <OuiSkeleton width="8rem" height="1rem" variant="text" />
                  <OuiSkeleton width="12rem" height="0.75rem" variant="text" />
                </OuiStack>
                <OuiSkeleton width="4rem" height="1.5rem" variant="rectangle" rounded />
              </OuiFlex>
            </OuiBox>
          </OuiStack>

          <OuiStack
            v-else-if="recentDeployments.length === 0"
            gap="sm"
            align="center"
            class="py-12 text-center"
          >
            <OuiBox p="md" rounded="xl" class="bg-surface-muted text-muted">
              <RocketLaunchIcon class="h-10 w-10" />
            </OuiBox>
            <OuiText as="h3" weight="medium" color="primary"
              >No deployments yet</OuiText
            >
            <OuiText size="xs" color="secondary"
              >Deploy your first application to see it listed here.</OuiText
            >
            <OuiButton
              variant="ghost"
              size="sm"
              class="mt-2"
              @click="navigateTo('/deployments')"
              >Create Deployment</OuiButton
            >
          </OuiStack>

          <OuiStack v-else gap="md">
            <OuiBox
              v-for="deployment in recentDeployments"
              :key="deployment.id"
              p="md"
              rounded="lg"
              border="1"
              borderColor="muted"
              class="cursor-pointer transition-all duration-150 hover:border-default hover:bg-surface-muted hover:shadow-sm"
              @click="navigateTo(`/deployments/${deployment.id}`)"
            >
              <OuiFlex justify="between" align="start" gap="md">
                <OuiStack gap="xs" class="flex-1 min-w-0">
                  <OuiFlex align="center" gap="sm" wrap="wrap">
                    <OuiText
                      as="h3"
                      weight="medium"
                      color="primary"
                      class="truncate"
                      >{{ deployment.name }}</OuiText
                    >
                    <OuiBadge variant="secondary" size="sm">
                      {{ deployment.environment }}
                    </OuiBadge>
                  </OuiFlex>
                  <OuiText size="sm" color="secondary" class="truncate">{{
                    deployment.domain
                  }}</OuiText>
                  <OuiFlex align="center" gap="sm" wrap="wrap" class="mt-1">
                    <OuiText size="xs" color="secondary">
                      <OuiRelativeTime :value="deployment.updatedAt" :style="'short'" />
                    </OuiText>
                    <OuiText size="xs" color="secondary">•</OuiText>
                    <OuiText size="xs" color="secondary">
                      {{ deployment.type || 'Unknown' }}
                    </OuiText>
                  </OuiFlex>
                </OuiStack>
                <OuiBadge :variant="statusVariant(deployment.status)">{{
                  deployment.status
                }}</OuiBadge>
              </OuiFlex>
            </OuiBox>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
import {
  ArrowPathIcon,
  CreditCardIcon,
  RocketLaunchIcon,
  ServerIcon,
  CircleStackIcon,
  CheckCircleIcon,
} from "@heroicons/vue/24/outline";

// Page meta
definePageMeta({
  layout: "default",
  middleware: "auth",
});

// Live stats via ConnectRPC
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationsStore } from "~/stores/organizations";
import { 
  DeploymentService, 
  DeploymentStatus, 
  Environment as EnvEnum,
  DeploymentType,
  OrganizationService,
} from "@obiente/proto";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";

type DashboardData = {
  stats: {
    deployments: number;
    vpsInstances: number;
    databases: number;
    monthlySpend: number;
    statuses: Array<{ status: string; count: number }>;
  };
  recentDeployments: Array<{
    id: string;
    name: string;
    domain: string;
    status: "RUNNING" | "BUILDING" | "STOPPED" | "PENDING" | "ERROR";
    updatedAt: string;
    environment: string;
    type: string;
  }>;
  activity: Array<{ id: string; message: string; timestamp: string }>;
};

const deploymentClient = useConnectClient(DeploymentService);
const orgClient = useConnectClient(OrganizationService);
const orgsStore = useOrganizationsStore();
const organizationId = computed(() => orgsStore.currentOrgId || "");

const toMs = (s: number | bigint | undefined | null) => Number(s ?? 0) * 1000;

const { data, status, refresh: refreshDashboard } = await useAsyncData<DashboardData>(
  () => `dashboard-${organizationId.value}`,
  async () => {
    // Fetch deployments for selected org (server will resolve if empty)
    const res = await deploymentClient.listDeployments({ organizationId: organizationId.value });
    const deployments = res.deployments ?? [];

    // Status breakdown
    const statusesMap: Record<string, number> = {};
    for (const d of deployments) {
      let s: string = "PENDING";
      switch (d.status) {
        case DeploymentStatus.RUNNING: s = "RUNNING"; break;
        case DeploymentStatus.BUILDING: s = "BUILDING"; break;
        case DeploymentStatus.STOPPED: s = "STOPPED"; break;
        case DeploymentStatus.FAILED: s = "ERROR"; break;
        default: s = "PENDING";
      }
      statusesMap[s] = (statusesMap[s] || 0) + 1;
    }
    const statuses = Object.entries(statusesMap).map(([status, count]) => ({ status, count }));

    // Recent deployments
    const recentDeployments = [...deployments]
      .sort((a, b) => {
        const at = toMs(a.lastDeployedAt?.seconds ?? a.createdAt?.seconds);
        const bt = toMs(b.lastDeployedAt?.seconds ?? b.createdAt?.seconds);
        return bt - at;
      })
      .slice(0, 5)
      .map((d) => {
        let env = "production";
        switch (d.environment) {
          case EnvEnum.STAGING: env = "staging"; break;
          case EnvEnum.DEVELOPMENT: env = "development"; break;
          default: env = "production";
        }
        const status = (() => {
          switch (d.status) {
            case DeploymentStatus.RUNNING: return "RUNNING" as const;
            case DeploymentStatus.BUILDING: return "BUILDING" as const;
            case DeploymentStatus.STOPPED: return "STOPPED" as const;
            case DeploymentStatus.FAILED: return "ERROR" as const;
            default: return "PENDING" as const;
          }
        })();
        const type = (() => {
          switch (d.type) {
            case DeploymentType.DOCKER: return "Docker";
            case DeploymentType.STATIC: return "Static";
            case DeploymentType.NODE: return "Node.js";
            case DeploymentType.GO: return "Go";
            case DeploymentType.PYTHON: return "Python";
            case DeploymentType.RUBY: return "Ruby";
            case DeploymentType.RUST: return "Rust";
            case DeploymentType.JAVA: return "Java";
            case DeploymentType.PHP: return "PHP";
            default: return "Generic";
          }
        })();

        return {
          id: d.id,
          name: d.name,
          domain: d.domain,
          status,
          environment: env,
          type,
          updatedAt: new Date(toMs(d.lastDeployedAt?.seconds ?? d.createdAt?.seconds)).toISOString(),
        };
      });

    const stats = {
      deployments: deployments.length,
      vpsInstances: 0,
      databases: 0,
      monthlySpend: 0,
      statuses,
    };

    const activity: Array<{ id: string; message: string; timestamp: string }> = [];

    return { stats, recentDeployments, activity };
  },
  { watch: [organizationId] }
);

// Fetch organization usage data
const { data: usageData, status: usageStatus, refresh: refreshUsage } = await useAsyncData(
  () => `org-usage-${organizationId.value}`,
  async () => {
    if (!organizationId.value) return null;
    try {
      const res = await orgClient.getUsage({
        organizationId: organizationId.value,
      });
      return res;
    } catch (err) {
      console.error("Failed to fetch usage:", err);
      return null;
    }
  },
  { watch: [organizationId], server: false }
);

const isLoadingUsage = computed(() => usageStatus.value === "pending" || usageStatus.value === "idle");
const isLoading = computed(
  () => status.value === "pending" || status.value === "idle"
);
const stats = computed(
  () =>
    data.value?.stats ?? {
      deployments: 0,
      vpsInstances: 0,
      databases: 0,
      monthlySpend: 0,
    }
);
const statusBreakdown = computed(
  () =>
    (data.value?.stats?.statuses ?? []) as Array<{
      status: "RUNNING" | "BUILDING" | "STOPPED" | "PENDING" | "ERROR";
      count: number;
    }>
);

const formatNumber = (value: number) =>
  new Intl.NumberFormat("en-US").format(value);

const formatCurrency = (amount: number) =>
  new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount);

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
};

const formatCPUUsage = (coreSeconds: number): string => {
  const hours = coreSeconds / 3600;
  if (hours < 1) {
    return `${(coreSeconds / 60).toFixed(1)} min`;
  }
  return `${hours.toFixed(1)} core-hrs`;
};

const formatQuota = (used: number, limit: number): string => {
  if (limit === 0) return "Unlimited";
  const percent = (used / limit) * 100;
  return `${percent.toFixed(1)}% of quota`;
};

// Usage percentages
const cpuUsagePercent = computed(() => {
  if (!usageData.value?.current || !usageData.value?.quota) return 0;
  const limit = Number(usageData.value.quota.cpuCoreSecondsMonthly);
  if (limit === 0) return 0;
  const used = Number(usageData.value.current.cpuCoreSeconds);
  return Math.min((used / limit) * 100, 100);
});

const memoryUsagePercent = computed(() => {
  if (!usageData.value?.current || !usageData.value?.quota) return 0;
  const limit = Number(usageData.value.quota.memoryByteSecondsMonthly);
  if (limit === 0) return 0;
  const used = Number(usageData.value.current.memoryByteSeconds);
  return Math.min((used / limit) * 100, 100);
});

const bandwidthUsagePercent = computed(() => {
  if (!usageData.value?.current || !usageData.value?.quota) return 0;
  const limit = Number(usageData.value.quota.bandwidthBytesMonthly);
  if (limit === 0) return 0;
  const used = Number(usageData.value.current.bandwidthRxBytes) + Number(usageData.value.current.bandwidthTxBytes);
  return Math.min((used / limit) * 100, 100);
});

const storageUsagePercent = computed(() => {
  if (!usageData.value?.current || !usageData.value?.quota) return 0;
  const limit = Number(usageData.value.quota.storageBytes);
  if (limit === 0) return 0;
  const used = Number(usageData.value.current.storageBytes);
  return Math.min((used / limit) * 100, 100);
});

// Cost breakdown
const costBreakdown = computed(() => {
  if (!usageData.value?.estimatedMonthly) return [];
  const estimated = usageData.value.estimatedMonthly;
  const breakdown = [];
  
  const totalCents = Number(estimated.estimatedCostCents);
  if (totalCents > 0) {
    // Calculate approximate breakdown (we don't have per-resource cost in org usage, so estimate)
    breakdown.push(
      { label: "CPU", value: formatCurrency(totalCents * 0.4 / 100), color: "bg-accent-primary" },
      { label: "Memory", value: formatCurrency(totalCents * 0.3 / 100), color: "bg-success" },
      { label: "Bandwidth", value: formatCurrency(totalCents * 0.2 / 100), color: "bg-accent-secondary" },
      { label: "Storage", value: formatCurrency(totalCents * 0.1 / 100), color: "bg-warning" },
    );
  }
  
  return breakdown;
});

const kpiCards = computed(() => {
  const monthlyCost = usageData.value?.estimatedMonthly
    ? Number(usageData.value.estimatedMonthly.estimatedCostCents) / 100 
    : stats.value.monthlySpend;
  
  return [
    {
      label: "Active Deployments",
      value: formatNumber(stats.value.deployments),
      subtitle: `${runningCount.value} running`,
      icon: RocketLaunchIcon,
      iconBg: "bg-primary/10",
      iconColor: "text-accent-primary",
      href: "/deployments",
      badge: runningCount.value > 0 ? `${runningCount.value}` : undefined,
      badgeVariant: "success" as const,
    },
    {
      label: "VPS Instances",
      value: formatNumber(stats.value.vpsInstances),
      icon: ServerIcon,
      iconBg: "bg-success/10",
      iconColor: "text-success",
      href: "/vps",
    },
    {
      label: "Databases",
      value: formatNumber(stats.value.databases),
      icon: CircleStackIcon,
      iconBg: "bg-accent-secondary/10",
      iconColor: "text-accent-secondary",
      href: "/databases",
    },
    {
      label: "Estimated Monthly Cost",
      value: formatCurrency(monthlyCost),
      subtitle: usageData.value?.month || "This month",
      icon: CreditCardIcon,
      iconBg: "bg-warning/10",
      iconColor: "text-warning",
      href: "/organizations?tab=billing",
    },
  ];
});

// Recent deployments with loading state
const recentDeployments = computed(
  () =>
    (data.value?.recentDeployments ?? []) as Array<{
      id: string;
      name: string;
      domain: string;
      status: "RUNNING" | "BUILDING" | "STOPPED" | "PENDING" | "ERROR";
      updatedAt: string;
      environment: string;
      type: string;
    }>
);

// Activity feed with loading state
const activityFeed = computed(
  () =>
    (data.value?.activity ?? []) as Array<{
      id: string;
      message: string;
      timestamp: string;
    }>
);

// Health metrics
const runningCount = computed(
  () => statusBreakdown.value.find((s) => s.status === "RUNNING")?.count ?? 0
);
const buildingCount = computed(
  () => statusBreakdown.value.find((s) => s.status === "BUILDING")?.count ?? 0
);
const stoppedCount = computed(
  () => statusBreakdown.value.find((s) => s.status === "STOPPED")?.count ?? 0
);
const errorCount = computed(
  () => statusBreakdown.value.find((s) => s.status === "ERROR")?.count ?? 0
);
const allHealthy = computed(() => errorCount.value === 0);
const attentionDeployments = computed(() =>
  recentDeployments.value
    .filter((d) => ["ERROR", "STOPPED", "BUILDING"].includes(d.status))
    .slice(0, 4)
);

// Auto-refresh using useAsyncData refresh
const refreshInterval = ref<ReturnType<typeof setInterval> | null>(null);
onMounted(() => {
  refreshInterval.value = setInterval(() => {
    refreshDashboard();
    refreshUsage();
  }, 30000);
});
onUnmounted(() => {
  if (refreshInterval.value) clearInterval(refreshInterval.value);
});


const statusVariant = (
  status: "RUNNING" | "BUILDING" | "STOPPED" | "PENDING" | "ERROR"
) => {
  switch (status) {
    case "RUNNING":
      return "success";
    case "BUILDING":
      return "warning";
    case "ERROR":
      return "danger";
    default:
      return "secondary";
  }
};

// Retry function for error states
const retryLoad = () => {
  refreshDashboard();
  refreshUsage();
};
</script>
