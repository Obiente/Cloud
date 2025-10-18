<template>
  <OuiContainer size="7xl" py="xl">
    <OuiStack gap="xl">
      <!-- Page Header -->
      <OuiFlex justify="between" align="center" wrap="wrap" gap="lg">
        <OuiStack gap="xs" class="flex-1">
          <OuiText as="h1" size="3xl" weight="bold" color="primary">Overview</OuiText>
          <OuiText color="secondary">A quick glance at your cloud resources and recent activity.</OuiText>
        </OuiStack>

        <OuiFlex gap="sm" align="center" wrap="wrap">
          <OuiButton variant="ghost" size="sm" @click="retryLoad" class="gap-2">
            <ArrowPathIcon class="h-4 w-4" />
            Refresh
          </OuiButton>
          <OuiButton color="primary" size="sm" class="gap-2 shadow-md" @click="navigateTo('/deployments')">
            <RocketLaunchIcon class="h-4 w-4" />
            New Deployment
          </OuiButton>
        </OuiFlex>
      </OuiFlex>

      <!-- Quick Links -->
      <OuiCard variant="raised">
        <OuiCardBody>
          <OuiFlex wrap="wrap" gap="sm" align="center">
            <OuiButton variant="ghost" size="sm" class="gap-1.5" @click="navigateTo('/deployments')">Deployments</OuiButton>
            <OuiButton variant="ghost" size="sm" class="gap-1.5" @click="navigateTo('/vps')">VPS</OuiButton>
            <OuiButton variant="ghost" size="sm" class="gap-1.5" @click="navigateTo('/databases')">Databases</OuiButton>
            <OuiButton variant="ghost" size="sm" class="gap-1.5" @click="navigateTo('/billing')">Billing</OuiButton>
            <OuiButton variant="ghost" size="sm" class="gap-1.5" @click="navigateTo('/organizations')">Organizations</OuiButton>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>

      <!-- KPI Overview -->
      <OuiGrid cols="1" cols-sm="2" cols-lg="4" gap="lg">
        <OuiCard v-for="card in kpiCards" :key="card.label" variant="raised" hoverable class="cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl" @click="card.href && navigateTo(card.href)">
          <OuiCardBody>
            <OuiFlex align="center" gap="md">
              <OuiBox p="sm" rounded="lg" :class="card.iconBg" class="flex items-center justify-center">
                <component :is="card.icon" class="h-6 w-6" :class="card.iconColor" />
              </OuiBox>
              <OuiStack gap="xs" class="flex-1">
                <OuiSkeleton v-if="isLoading" width="3.5rem" height="1.5rem" variant="text" />
                <OuiText v-else as="h3" size="xl" weight="semibold" color="primary">{{ card.value }}</OuiText>
                <OuiText size="sm" color="secondary">{{ card.label }}</OuiText>
              </OuiStack>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>
      </OuiGrid>

      <!-- Health + Activity Row -->
      <OuiGrid cols="1" cols-lg="2" gap="xl">
        <!-- Health & Alerts -->
        <OuiCard>
          <OuiCardHeader>
            <OuiFlex align="center" justify="between">
              <OuiText as="h2" class="oui-card-title">Health</OuiText>
              <OuiBadge :variant="allHealthy ? 'success' : 'warning'">{{ allHealthy ? 'All systems nominal' : 'Attention required' }}</OuiBadge>
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
              <OuiFlex gap="sm" wrap="wrap" class="mb-3">
                <OuiBadge variant="danger">Errors: {{ errorCount }}</OuiBadge>
                <OuiBadge variant="warning">Building: {{ buildingCount }}</OuiBadge>
                <OuiBadge variant="secondary">Stopped: {{ stoppedCount }}</OuiBadge>
                <OuiBadge variant="success">Running: {{ runningCount }}</OuiBadge>
              </OuiFlex>
              <OuiStack v-if="attentionDeployments.length > 0" gap="sm">
                <OuiBox
                  v-for="d in attentionDeployments.slice(0,4)"
                  :key="d.id"
                  p="sm"
                  rounded="lg"
                  class="ring-1 ring-border-muted bg-surface-muted/40"
                >
                  <OuiFlex justify="between" align="center" gap="md">
                    <OuiStack gap="xs" class="min-w-0">
                      <OuiText size="sm" weight="medium" class="truncate">{{ d.name }}</OuiText>
                      <OuiText size="xs" color="secondary" class="truncate">{{ d.domain }}</OuiText>
                    </OuiStack>
                    <OuiBadge :variant="statusVariant(d.status)">{{ d.status }}</OuiBadge>
                  </OuiFlex>
                </OuiBox>
                <OuiButton variant="ghost" size="sm" class="self-start" @click="navigateTo('/deployments')">View deployments</OuiButton>
              </OuiStack>
              <OuiText v-else color="secondary">No issues detected.</OuiText>
            </template>
          </OuiCardBody>
        </OuiCard>

        <!-- Activity Feed -->
        <OuiCard>
          <OuiCardHeader>
            <OuiText as="h2" class="oui-card-title">Activity</OuiText>
          </OuiCardHeader>
          <OuiCardBody>
            <OuiStack v-if="isLoading" gap="md">
              <OuiFlex v-for="i in 3" :key="i" align="start" gap="md">
                <OuiSkeleton
                  width="0.5rem"
                  height="0.5rem"
                  variant="circle"
                  class="mt-2"
                />
                <OuiStack gap="xs" class="flex-1">
                  <OuiSkeleton width="10rem" height="1rem" variant="text" />
                  <OuiSkeleton width="5rem" height="0.75rem" variant="text" />
                </OuiStack>
              </OuiFlex>
            </OuiStack>

            <OuiStack v-else-if="activityFeed.length === 0" gap="sm" align="center" class="py-12 text-center">
              <OuiText size="sm" color="secondary">No recent activity.</OuiText>
            </OuiStack>

            <OuiStack v-else gap="md">
              <OuiBox
                v-for="activity in activityFeed"
                :key="activity.id"
                p="sm"
                rounded="lg"
                class="hover:bg-surface-muted transition-colors"
              >
                <OuiFlex align="start" gap="md">
                  <OuiBox w="fit" h="fit" rounded="full" class="h-2 w-2 bg-accent-primary mt-2" />
                  <OuiStack gap="xs" class="flex-1 min-w-0">
                    <OuiText size="sm" color="primary" class="break-words">{{ activity.message }}</OuiText>
                    <OuiText size="xs" color="secondary">{{ formatRelative(activity.timestamp) }}</OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiBox>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>
      </OuiGrid>

      <OuiGrid cols="1" cols-lg="2" gap="xl">
        <!-- Recent Deployments -->
        <OuiCard>
          <OuiCardHeader>
            <OuiFlex align="center" justify="between">
              <OuiStack gap="xs">
                <OuiText as="h2" class="oui-card-title">Recent Deployments</OuiText>
                <OuiText size="xs" color="secondary">
                  Latest releases across your environments
                </OuiText>
              </OuiStack>
              <div />
            </OuiFlex>
          </OuiCardHeader>
          <OuiCardBody>
            <OuiStack v-if="isLoading" gap="md">
              <OuiBox
                v-for="i in 3"
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

            <OuiStack v-else-if="recentDeployments.length === 0" gap="sm" align="center" class="py-12 text-center">
              <OuiBox p="md" rounded="xl" class="bg-surface-muted text-muted">
                <svg class="h-10 w-10" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
                </svg>
              </OuiBox>
              <OuiText as="h3" weight="medium" color="primary">No deployments yet</OuiText>
              <OuiText size="xs" color="secondary">Deploy your first application to see it listed here.</OuiText>
              <OuiButton variant="ghost" size="sm" class="mt-2" @click="navigateTo('/deployments')">View deployments</OuiButton>
            </OuiStack>

            <OuiStack v-else gap="md">
              <OuiBox
                v-for="deployment in recentDeployments"
                :key="deployment.id"
                p="md"
                rounded="lg"
                border="1"
                borderColor="muted"
                class="cursor-pointer transition-colors duration-150 hover:border-default hover:bg-surface-muted"
                @click="navigateTo(`/deployments/${deployment.id}`)"
              >
                <OuiFlex justify="between" align="center" gap="md">
                  <OuiStack gap="xs" class="flex-1 min-w-0">
                    <OuiText as="h3" weight="medium" color="primary" class="truncate">{{ deployment.name }}</OuiText>
                    <OuiText size="sm" color="secondary" class="truncate">{{ deployment.domain }}</OuiText>
                  </OuiStack>
                  <OuiFlex align="center" gap="sm">
                    <OuiBadge :variant="statusVariant(deployment.status)">{{ deployment.status }}</OuiBadge>
                    <OuiText size="xs" color="secondary">{{ formatRelative(deployment.updatedAt) }}</OuiText>
                  </OuiFlex>
                </OuiFlex>
              </OuiBox>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Spend Overview -->
        <OuiCard>
          <OuiCardHeader>
            <OuiText as="h2" class="oui-card-title">Spend</OuiText>
          </OuiCardHeader>
          <OuiCardBody>
            <OuiFlex align="center" justify="between">
              <OuiStack gap="xs">
                <OuiText size="sm" color="secondary">This Month</OuiText>
                <OuiText size="2xl" weight="bold" color="primary">{{ formatCurrency(stats.monthlySpend) }}</OuiText>
              </OuiStack>
              <OuiButton variant="ghost" size="sm" @click="navigateTo('/billing')">View billing</OuiButton>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>
      </OuiGrid>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
import { ArrowPathIcon, CreditCardIcon, RocketLaunchIcon, ServerIcon, CircleStackIcon } from "@heroicons/vue/24/outline";

// Page meta
definePageMeta({
  layout: "default",
  middleware: "auth",
});

// Dashboard stats with loading simulation
const { data, status, refresh: refreshDashboard } = await useAsyncData(
  'dashboard',
  () => $fetch('/api/cloud', { headers: { accept: 'application/json' } }),
  { server: true }
);
const isLoading = computed(() => status.value === 'pending');
const stats = computed(() => data.value?.stats ?? { deployments: 0, vpsInstances: 0, databases: 0, monthlySpend: 0 });
const statusBreakdown = computed(() => (data.value?.stats?.statuses ?? []) as Array<{ status: 'RUNNING' | 'BUILDING' | 'STOPPED' | 'PENDING' | 'ERROR'; count: number }>);

const formatNumber = (value: number) =>
  new Intl.NumberFormat("en-US").format(value);

const formatCurrency = (amount: number) =>
  new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(amount);

const kpiCards = computed(() => [
  {
    label: "Deployments",
    value: formatNumber(stats.value.deployments),
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
    label: "Databases",
    value: formatNumber(stats.value.databases),
    icon: CircleStackIcon,
    iconBg: "bg-accent-secondary/10",
    iconColor: "text-accent-secondary",
    href: "/databases",
  },
  {
    label: "This Month",
    value: formatCurrency(stats.value.monthlySpend),
    icon: CreditCardIcon,
    iconBg: "bg-warning/10",
    iconColor: "text-warning",
    href: "/billing",
  },
]);

// Recent deployments with loading state
const recentDeployments = computed(() => (data.value?.recentDeployments ?? []) as Array<{ id: string; name: string; domain: string; status: 'RUNNING' | 'BUILDING' | 'STOPPED' | 'PENDING' | 'ERROR'; updatedAt: string; environment: string; }>);

// Activity feed with loading state
const activityFeed = computed(() => (data.value?.activity ?? []) as Array<{ id: string; message: string; timestamp: string }>)

// Health metrics
const runningCount = computed(() => statusBreakdown.value.find(s => s.status === 'RUNNING')?.count ?? 0);
const buildingCount = computed(() => statusBreakdown.value.find(s => s.status === 'BUILDING')?.count ?? 0);
const stoppedCount = computed(() => statusBreakdown.value.find(s => s.status === 'STOPPED')?.count ?? 0);
const errorCount = computed(() => statusBreakdown.value.find(s => s.status === 'ERROR')?.count ?? 0);
const allHealthy = computed(() => errorCount.value === 0);
const attentionDeployments = computed(() => recentDeployments.value.filter(d => ['ERROR', 'STOPPED', 'BUILDING'].includes(d.status)).slice(0, 4));

// Auto-refresh using useAsyncData refresh
const refreshInterval = ref<ReturnType<typeof setInterval> | null>(null);
onMounted(() => {
  refreshInterval.value = setInterval(() => {
    refreshDashboard();
  }, 30000);
});
onUnmounted(() => {
  if (refreshInterval.value) clearInterval(refreshInterval.value);
});

const formatRelative = (dateISO: string | Date) => {
  const date = typeof dateISO === 'string' ? new Date(dateISO) : dateISO;
  const diffMs = date.getTime() - Date.now();
  const minutes = Math.round(diffMs / (1000 * 60));
  // Prefer minutes for recent events; could expand to hours/days if needed
  return new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' }).format(minutes, 'minute');
};

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
};
</script>
