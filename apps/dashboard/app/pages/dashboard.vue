<template>
  <OuiContainer size="full" p="none">
    <OuiStack gap="lg">
      <!-- Page Header -->
      <OuiFlex align="center" justify="between" gap="md" wrap="wrap">
        <OuiStack gap="xs">
          <OuiText as="h1" size="xl" weight="semibold">Dashboard</OuiText>
          <OuiText size="sm" color="tertiary" class="hidden sm:block">
            Overview of your infrastructure and usage.
          </OuiText>
        </OuiStack>

        <OuiButton
          variant="ghost"
          size="sm"
          @click="retryLoad"
          class="gap-1.5"
        >
          <ArrowPathIcon
            class="h-4 w-4"
            :class="{ 'animate-spin': isLoading }"
          />
          Refresh
        </OuiButton>
      </OuiFlex>

      <!-- KPI Cards -->
      <OuiGrid :cols="{ sm: 2, md: 3, lg: 5 }" gap="sm">
        <OuiBox
          v-for="card in kpiCards"
          :key="card.label"
          p="md"
          class="app-stat-card cursor-pointer"
          @click="card.href && navigateTo(card.href)"
        >
          <OuiStack gap="xs" class="min-w-0">
            <OuiFlex align="center" justify="between">
              <OuiText size="xs" color="tertiary" weight="medium">
                {{ card.label }}
              </OuiText>
              <component
                :is="card.icon"
                class="h-3.5 w-3.5"
                :class="card.iconColor"
              />
            </OuiFlex>
            <OuiSkeleton
              v-if="isLoading"
              width="3rem"
              height="1.5rem"
              variant="text"
            />
            <OuiText
              v-else
              as="h3"
              size="xl"
              weight="semibold"
              color="primary"
            >
              {{ card.value }}
            </OuiText>
          </OuiStack>
        </OuiBox>
      </OuiGrid>

      <!-- Resource Usage -->
      <OuiCard class="app-panel">
        <OuiCardHeader>
          <OuiFlex align="center" justify="between" gap="md">
            <OuiStack gap="xs">
              <OuiText as="h2" size="sm" weight="semibold">Resource Usage</OuiText>
              <OuiText size="xs" color="tertiary" class="hidden sm:block">
                Current month usage against quota
              </OuiText>
            </OuiStack>
            <OuiButton variant="ghost" size="sm" @click="navigateTo('/billing')">
              View Details
            </OuiButton>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody>
          <template v-if="isLoadingUsage">
            <OuiGrid :cols="{ sm: 1, md: 2, lg: 4 }" gap="lg">
              <OuiStack v-for="i in 4" :key="i" gap="sm">
                <OuiSkeleton width="6rem" height="1rem" variant="text" />
                <OuiSkeleton
                  width="100%"
                  height="0.5rem"
                  variant="rectangle"
                  rounded
                />
                <OuiSkeleton width="4rem" height="0.75rem" variant="text" />
              </OuiStack>
            </OuiGrid>
          </template>
          <template
            v-else-if="usageData && usageData.current && usageData.quota"
          >
            <OuiGrid :cols="{ sm: 1, md: 2, lg: 4 }" gap="lg">
              <!-- CPU Usage -->
              <OuiStack gap="sm">
                <OuiFlex align="center" justify="between">
                  <OuiText size="sm" weight="medium" color="primary"
                    >CPU</OuiText
                  >
                  <OuiText size="xs" color="tertiary">
                    {{
                      formatCPUUsage(Number(usageData.current.cpuCoreSeconds))
                    }}
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
                <OuiText size="xs" color="tertiary">
                  {{
                    formatQuota(
                      Number(usageData.current.cpuCoreSeconds),
                      Number(usageData.quota.cpuCoreSecondsMonthly)
                    )
                  }}
                </OuiText>
              </OuiStack>

              <!-- Memory Usage -->
              <OuiStack gap="sm">
                <OuiFlex align="center" justify="between">
                  <OuiText size="sm" weight="medium" color="primary"
                    >Memory</OuiText
                  >
                  <OuiText size="xs" color="tertiary">
                    {{
                      formatBytes(
                        Number(usageData.current.memoryByteSeconds) / 3600
                      )
                    }}/hr avg
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
                <OuiText size="xs" color="tertiary">
                  {{
                    formatQuota(
                      Number(usageData.current.memoryByteSeconds),
                      Number(usageData.quota.memoryByteSecondsMonthly)
                    )
                  }}
                </OuiText>
              </OuiStack>

              <!-- Bandwidth Usage -->
              <OuiStack gap="sm">
                <OuiFlex align="center" justify="between">
                  <OuiText size="sm" weight="medium" color="primary"
                    >Bandwidth</OuiText
                  >
                  <OuiText size="xs" color="tertiary">
                    {{
                      formatBytes(
                        Number(usageData.current.bandwidthRxBytes) +
                          Number(usageData.current.bandwidthTxBytes)
                      )
                    }}
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
                <OuiText size="xs" color="tertiary">
                  {{
                    formatQuota(
                      Number(usageData.current.bandwidthRxBytes) +
                        Number(usageData.current.bandwidthTxBytes),
                      Number(usageData.quota.bandwidthBytesMonthly)
                    )
                  }}
                </OuiText>
              </OuiStack>

              <!-- Storage Usage -->
              <OuiStack gap="sm">
                <OuiFlex align="center" justify="between">
                  <OuiText size="sm" weight="medium" color="primary"
                    >Storage</OuiText
                  >
                  <OuiText size="xs" color="tertiary">
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
                <OuiText size="xs" color="tertiary">
                  {{
                    formatQuota(
                      Number(usageData.current.storageBytes),
                      Number(usageData.quota.storageBytes)
                    )
                  }}
                </OuiText>
              </OuiStack>
            </OuiGrid>
          </template>
          <OuiText v-else size="sm" color="tertiary" class="text-center py-4">
            Usage data unavailable
          </OuiText>
        </OuiCardBody>
      </OuiCard>

      <!-- Cost Breakdown & Health -->
      <OuiGrid :cols="{ sm: 1, lg: 2 }" gap="md">
        <!-- Cost Breakdown -->
        <OuiCard class="app-panel">
          <OuiCardHeader>
            <OuiStack gap="xs">
              <OuiText as="h2" size="sm" weight="semibold">Cost Breakdown</OuiText>
              <OuiText size="xs" color="tertiary">
                Estimated monthly costs
              </OuiText>
            </OuiStack>
          </OuiCardHeader>
          <OuiCardBody>
            <template v-if="isLoadingUsage">
              <OuiStack gap="sm">
                <OuiSkeleton width="100%" height="2.5rem" variant="rectangle" rounded />
                <OuiSkeleton width="100%" height="2.5rem" variant="rectangle" rounded />
                <OuiSkeleton width="100%" height="2.5rem" variant="rectangle" rounded />
              </OuiStack>
            </template>
            <template v-else-if="usageData && usageData.estimatedMonthly">
              <OuiStack gap="md">
                <OuiFlex align="center" justify="between" class="pb-3 border-b border-border-muted">
                  <OuiStack gap="xs">
                    <OuiText size="xs" color="tertiary">Total Estimated</OuiText>
                    <OuiText size="xl" weight="semibold" color="primary">
                      {{ formatCurrency(Number(usageData.estimatedMonthly.estimatedCostCents) / 100) }}
                    </OuiText>
                  </OuiStack>
                  <OuiBadge variant="secondary" size="sm">{{ usageData.month }}</OuiBadge>
                </OuiFlex>
                <OuiStack gap="xs">
                  <OuiBox
                    v-for="cost in costBreakdown"
                    :key="cost.label"
                    p="sm"
                    rounded="lg"
                    class="app-resource-metric"
                  >
                    <OuiFlex align="center" justify="between" gap="sm">
                      <OuiFlex align="center" gap="xs">
                        <OuiBox as="span" rounded="full" class="w-2 h-2" :class="cost.color" />
                        <OuiText size="sm" color="primary">{{ cost.label }}</OuiText>
                      </OuiFlex>
                      <OuiText size="sm" weight="medium" color="primary">{{ cost.value }}</OuiText>
                    </OuiFlex>
                  </OuiBox>
                </OuiStack>
              </OuiStack>
            </template>
            <OuiText v-else size="sm" color="tertiary" class="text-center py-4">
              Cost data unavailable
            </OuiText>
          </OuiCardBody>
        </OuiCard>

        <!-- Deployment Health -->
        <OuiCard class="app-panel">
          <OuiCardHeader>
            <OuiFlex align="center" justify="between" gap="sm">
              <OuiStack gap="xs">
                <OuiText as="h2" size="sm" weight="semibold">Deployment Health</OuiText>
                <OuiText size="xs" color="tertiary">Status overview</OuiText>
              </OuiStack>
              <OuiBadge
                :variant="allHealthy ? 'success' : 'warning'"
                size="sm"
              >{{ allHealthy ? "All Healthy" : "Issues Detected" }}</OuiBadge>
            </OuiFlex>
          </OuiCardHeader>
          <OuiCardBody>
            <template v-if="isLoading">
              <OuiStack gap="xs">
                <OuiSkeleton width="80%" height="1rem" variant="text" />
                <OuiSkeleton width="70%" height="1rem" variant="text" />
              </OuiStack>
            </template>
            <template v-else>
              <OuiGrid :cols="{ sm: 2, md: 4 }" gap="xs" class="mb-4">
                <OuiBox
                  p="sm"
                  rounded="lg"
                  class="app-resource-metric text-center cursor-pointer"
                  @click="navigateTo(runningDeploymentsUrl)"
                >
                  <OuiText size="lg" weight="semibold" class="text-success">{{ runningCount }}</OuiText>
                  <OuiText size="xs" color="tertiary">Running</OuiText>
                </OuiBox>
                <OuiBox
                  p="sm"
                  rounded="lg"
                  class="app-resource-metric text-center cursor-pointer"
                  @click="navigateTo(buildingDeploymentsUrl)"
                >
                  <OuiText size="lg" weight="semibold" class="text-warning">{{ buildingCount }}</OuiText>
                  <OuiText size="xs" color="tertiary">Building</OuiText>
                </OuiBox>
                <OuiBox
                  p="sm"
                  rounded="lg"
                  class="app-resource-metric text-center cursor-pointer"
                  @click="navigateTo(stoppedDeploymentsUrl)"
                >
                  <OuiText size="lg" weight="semibold" class="text-secondary">{{ stoppedCount }}</OuiText>
                  <OuiText size="xs" color="tertiary">Stopped</OuiText>
                </OuiBox>
                <OuiBox
                  p="sm"
                  rounded="lg"
                  class="app-resource-metric text-center cursor-pointer"
                  @click="navigateTo(errorDeploymentsUrl)"
                >
                  <OuiText size="lg" weight="semibold" class="text-danger">{{ errorCount }}</OuiText>
                  <OuiText size="xs" color="tertiary">Errors</OuiText>
                </OuiBox>
              </OuiGrid>

              <OuiStack v-if="attentionDeployments.length > 0" gap="xs">
                <OuiText size="xs" weight="medium" color="tertiary">
                  Needs Attention
                </OuiText>
                <OuiBox
                  v-for="d in attentionDeployments.slice(0, 3)"
                  :key="d.id"
                  p="sm"
                  rounded="lg"
                  class="app-resource-metric cursor-pointer hover:bg-surface-muted transition-colors"
                  @click="navigateTo(`/deployments/${d.id}`)"
                >
                  <OuiFlex align="center" justify="between" gap="sm">
                    <OuiStack gap="xs" class="min-w-0">
                      <OuiText size="sm" weight="medium" truncate>{{ d.name }}</OuiText>
                      <OuiText size="xs" color="tertiary" truncate>{{ d.domain }}</OuiText>
                    </OuiStack>
                    <OuiBadge :variant="statusVariant(d.status)" size="sm">{{ d.status }}</OuiBadge>
                  </OuiFlex>
                </OuiBox>
              </OuiStack>

              <OuiBox v-else p="sm" rounded="lg" class="app-resource-metric">
                <OuiFlex align="center" gap="xs">
                  <CheckCircleIcon class="h-4 w-4 text-success shrink-0" />
                  <OuiText size="sm" color="tertiary">All deployments running smoothly</OuiText>
                </OuiFlex>
              </OuiBox>
            </template>
          </OuiCardBody>
        </OuiCard>
      </OuiGrid>

      <!-- Recent Deployments -->
      <OuiCard class="app-panel">
        <OuiCardHeader>
          <OuiFlex align="center" justify="between" gap="sm">
            <OuiStack gap="xs">
              <OuiText as="h2" size="sm" weight="semibold">Recent Deployments</OuiText>
              <OuiText size="xs" color="tertiary">
                Latest {{ recentDeployments.length }} deployments
              </OuiText>
            </OuiStack>
            <OuiButton variant="ghost" size="sm" @click="navigateTo('/deployments')">
              View All
            </OuiButton>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiStack v-if="isLoading" gap="sm">
            <OuiFlex v-for="i in 5" :key="i" align="center" justify="between" gap="md" class="p-3">
              <OuiStack gap="xs" grow>
                <OuiSkeleton width="8rem" height="1rem" variant="text" />
                <OuiSkeleton width="12rem" height="0.75rem" variant="text" />
              </OuiStack>
              <OuiSkeleton width="4rem" height="1.5rem" variant="rectangle" rounded />
            </OuiFlex>
          </OuiStack>

          <OuiStack v-else-if="recentDeployments.length === 0" align="center" gap="sm" class="py-10">
            <RocketLaunchIcon class="h-8 w-8 text-tertiary" />
            <OuiText weight="medium" color="primary">No deployments yet</OuiText>
            <OuiText size="xs" color="tertiary">Deploy your first application to get started.</OuiText>
            <OuiButton variant="ghost" size="sm" @click="navigateTo('/deployments')">
              Create Deployment
            </OuiButton>
          </OuiStack>

          <OuiStack v-else gap="none" divider>
            <OuiFlex
              v-for="deployment in recentDeployments"
              :key="deployment.id"
              align="center"
              justify="between"
              gap="md"
              class="px-1 py-3 cursor-pointer hover:bg-surface-raised transition-colors"
              @click="navigateTo(`/deployments/${deployment.id}`)"
            >
              <OuiStack gap="xs" grow class="min-w-0">
                <OuiFlex align="center" gap="xs">
                  <OuiText size="sm" weight="medium" color="primary" truncate>
                    {{ deployment.name }}
                  </OuiText>
                  <OuiBadge variant="secondary" size="xs">
                    {{ deployment.environment }}
                  </OuiBadge>
                </OuiFlex>
                <OuiFlex align="center" gap="xs">
                  <OuiText size="xs" color="tertiary" truncate>{{ deployment.domain }}</OuiText>
                  <OuiText as="span" size="xs" color="tertiary">·</OuiText>
                  <OuiText size="xs" color="tertiary">
                    <OuiRelativeTime :value="deployment.updatedAt" :style="'short'" />
                  </OuiText>
                </OuiFlex>
              </OuiStack>
              <OuiBadge :variant="statusVariant(deployment.status)" size="sm">
                {{ deployment.status }}
              </OuiBadge>
            </OuiFlex>
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
    CubeIcon,
  } from "@heroicons/vue/24/outline";

  // Page meta
  definePageMeta({
    layout: "default",
    middleware: "auth",
  });

  // Live stats via ConnectRPC
  import { useConnectClient } from "~/lib/connect-client";
  import { useOrganizationsStore } from "~/stores/organizations";
  import { useAuth } from "~/composables/useAuth";
  import { useOrganizationId } from "~/composables/useOrganizationId";
  import type { UserSession } from "@obiente/types";
  import {
    DeploymentService,
    DeploymentStatus,
    Environment as EnvEnum,
    DeploymentType,
    OrganizationService,
    VPSService,
  } from "@obiente/proto";
  import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
  import { composeQueryUrl } from "~/utils/queryParams";

  type DashboardData = {
    stats: {
      deployments: number;
      gameServers: number;
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
  const vpsClient = useConnectClient(VPSService);
  const orgsStore = useOrganizationsStore();
  const auth = useAuth();

  const toMs = (s: number | bigint | undefined | null) => Number(s ?? 0) * 1000;

  // Get organizationId using SSR-compatible composable
  const organizationId = useOrganizationId();

  const {
    data,
    status,
    refresh: refreshDashboard,
  } = await useClientFetch<DashboardData>(
    () => `dashboard-data-${organizationId.value}`,
    async () => {
      // Use organizationId from composable (SSR-compatible)
      const orgId = organizationId.value;
      // Fetch deployments for selected org (server will resolve if empty)
      const res = await deploymentClient.listDeployments({
        organizationId: orgId,
      });
      const deployments = res.deployments ?? [];

      // Status breakdown
      const statusesMap: Record<string, number> = {};
      for (const d of deployments) {
        let s: string = "PENDING";
        switch (d.status) {
          case DeploymentStatus.RUNNING:
            s = "RUNNING";
            break;
          case DeploymentStatus.BUILDING:
            s = "BUILDING";
            break;
          case DeploymentStatus.STOPPED:
            s = "STOPPED";
            break;
          case DeploymentStatus.FAILED:
            s = "ERROR";
            break;
          default:
            s = "PENDING";
        }
        statusesMap[s] = (statusesMap[s] || 0) + 1;
      }
      const statuses = Object.entries(statusesMap).map(([status, count]) => ({
        status,
        count,
      }));

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
            case EnvEnum.STAGING:
              env = "staging";
              break;
            case EnvEnum.DEVELOPMENT:
              env = "development";
              break;
            default:
              env = "production";
          }
          const status = (() => {
            switch (d.status) {
              case DeploymentStatus.RUNNING:
                return "RUNNING" as const;
              case DeploymentStatus.BUILDING:
                return "BUILDING" as const;
              case DeploymentStatus.STOPPED:
                return "STOPPED" as const;
              case DeploymentStatus.FAILED:
                return "ERROR" as const;
              default:
                return "PENDING" as const;
            }
          })();
          const type = (() => {
            switch (d.type) {
              case DeploymentType.DOCKER:
                return "Docker";
              case DeploymentType.STATIC:
                return "Static";
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
              default:
                return "Generic";
            }
          })();

          return {
            id: d.id,
            name: d.name,
            domain: d.domain,
            status,
            environment: env,
            type,
            updatedAt: new Date(
              toMs(d.lastDeployedAt?.seconds ?? d.createdAt?.seconds)
            ).toISOString(),
          };
        });

      // Fetch VPS count
      let vpsCount = 0;
      try {
        const vpsResponse = await vpsClient.listVPS({
          organizationId: orgId || undefined,
          page: 1,
          perPage: 1,
        });
        vpsCount = vpsResponse.pagination?.total || 0;
      } catch (error) {
        console.error("Failed to fetch VPS count:", error);
      }

      const stats = {
        deployments: deployments.length,
        gameServers: 0, // TODO: Fetch from game server API
        vpsInstances: vpsCount,
        databases: 0,
        monthlySpend: 0,
        statuses,
      };

      const activity: Array<{
        id: string;
        message: string;
        timestamp: string;
      }> = [];

      return { stats, recentDeployments, activity };
    },
    {
      watch: [organizationId],
    }
  );

  // Fetch organization usage data
  const {
    data: usageData,
    status: usageStatus,
    refresh: refreshUsage,
  } = await useClientFetch(
    () => `org-usage-data-${organizationId.value}`,
    async () => {
      // Use organizationId from composable (SSR-compatible)
      const orgId = organizationId.value;
      // Try to fetch even if orgId is empty (server will resolve if empty)
      try {
        const res = await orgClient.getUsage({
          organizationId: orgId || undefined,
        });
        return res;
      } catch (err) {
        console.error("Failed to fetch usage:", err);
        return null;
      }
    },
    {
      watch: [organizationId],
    }
  );

  // Only show loading if data is not available and status is pending/idle
  const isLoadingUsage = computed(
    () =>
      !usageData.value &&
      (usageStatus.value === "pending" || usageStatus.value === "idle")
  );
  const isLoading = computed(
    () => !data.value && (status.value === "pending" || status.value === "idle")
  );
  const stats = computed(
    () =>
      data.value?.stats ?? {
        deployments: 0,
        gameServers: 0,
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
    const used =
      Number(usageData.value.current.bandwidthRxBytes) +
      Number(usageData.value.current.bandwidthTxBytes);
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
      // Use actual per-resource costs if available, otherwise fall back to percentages
      const cpuCents =
        estimated.cpuCostCents != null
          ? Number(estimated.cpuCostCents)
          : totalCents * 0.4;
      const memoryCents =
        estimated.memoryCostCents != null
          ? Number(estimated.memoryCostCents)
          : totalCents * 0.3;
      const bandwidthCents =
        estimated.bandwidthCostCents != null
          ? Number(estimated.bandwidthCostCents)
          : totalCents * 0.2;
      const storageCents =
        estimated.storageCostCents != null
          ? Number(estimated.storageCostCents)
          : totalCents * 0.1;

      breakdown.push(
        {
          label: "CPU",
          value: formatCurrency(cpuCents / 100),
          color: "bg-accent-primary",
        },
        {
          label: "Memory",
          value: formatCurrency(memoryCents / 100),
          color: "bg-success",
        },
        {
          label: "Bandwidth",
          value: formatCurrency(bandwidthCents / 100),
          color: "bg-accent-secondary",
        },
        {
          label: "Storage",
          value: formatCurrency(storageCents / 100),
          color: "bg-warning",
        }
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
        label: "Game Servers",
        value: formatNumber(stats.value.gameServers),
        icon: CubeIcon,
        iconBg: "bg-accent-success/10",
        iconColor: "text-accent-success",
        href: "/gameservers",
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
        href: "/billing",
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

  // Deployment URLs with status filters
  const runningDeploymentsUrl = computed(() =>
    composeQueryUrl("/deployments", {
      status: String(DeploymentStatus.RUNNING),
    })
  );
  const buildingDeploymentsUrl = computed(() =>
    composeQueryUrl("/deployments", {
      status: String(DeploymentStatus.BUILDING),
    })
  );
  const stoppedDeploymentsUrl = computed(() =>
    composeQueryUrl("/deployments", {
      status: String(DeploymentStatus.STOPPED),
    })
  );
  const errorDeploymentsUrl = computed(() =>
    composeQueryUrl("/deployments", { status: String(DeploymentStatus.FAILED) })
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
