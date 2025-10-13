<template>
  <OuiContainer size="7xl" py="xl">
    <OuiStack gap="xl">
      <!-- Page Header -->
      <OuiFlex justify="between" align="center" wrap="wrap" gap="lg">
        <OuiStack gap="xs" class="flex-1">
          <OuiText as="h1" size="3xl" weight="bold" color="primary">
            Dashboard
          </OuiText>
          <OuiText color="secondary">
            Welcome back to Obiente Cloud — here's a quick look at your
            infrastructure.
          </OuiText>
        </OuiStack>

        <OuiFlex gap="sm" align="center" wrap="wrap">
          <OuiButton variant="ghost" size="sm" @click="retryLoad" class="gap-2">
            <ArrowPathIcon class="h-4 w-4" />
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

      <!-- Error State -->
      <OuiCard
        v-if="statsError"
        variant="raised"
        border="1"
        borderColor="danger"
        class="bg-danger/10"
      >
        <OuiCardBody>
          <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
            <OuiFlex align="center" gap="md">
              <OuiBox p="sm" rounded="lg" class="bg-danger/10 text-danger">
                <svg
                  class="h-6 w-6"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
              </OuiBox>
              <OuiStack gap="xs">
                <OuiText weight="medium" color="danger">
                  {{ statsError }}
                </OuiText>
                <OuiText size="sm" color="secondary">
                  Unable to load dashboard data at the moment.
                </OuiText>
              </OuiStack>
            </OuiFlex>

            <OuiButton variant="outline" size="sm" @click="retryLoad">
              Try again
            </OuiButton>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>

      <!-- KPI Overview -->
      <OuiGrid cols="1" cols-sm="2" cols-lg="4" gap="lg">
        <OuiCard
          v-for="card in kpiCards"
          :key="card.label"
          variant="raised"
          hoverable
          class="cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl"
          @click="card.href && navigateTo(card.href)"
        >
          <OuiCardBody>
            <OuiFlex align="center" gap="md">
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
              <OuiStack gap="xs" class="flex-1">
                <OuiSkeleton
                  v-if="isLoading"
                  width="3.5rem"
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
                <OuiText size="sm" color="secondary">
                  {{ card.label }}
                </OuiText>
              </OuiStack>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>
      </OuiGrid>

      <OuiGrid cols="1" cols-lg="2" gap="xl">
        <!-- Recent Deployments -->
        <OuiCard>
          <OuiCardHeader>
            <OuiFlex align="center" justify="between">
              <OuiStack gap="xs">
                <OuiText as="h2" class="oui-card-title"
                  >Recent Deployments</OuiText
                >
                <OuiText size="xs" color="secondary">
                  Latest releases across your environments
                </OuiText>
              </OuiStack>
              <OuiFlex align="center" gap="xs">
                <OuiBox
                  v-if="!isLoading"
                  w="fit"
                  h="fit"
                  rounded="full"
                  class="h-2 w-2 bg-success"
                />
                <OuiText size="xs" color="secondary">
                  {{ isLoading ? "Syncing" : "Live" }}
                </OuiText>
              </OuiFlex>
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
                    <OuiSkeleton
                      width="12rem"
                      height="0.75rem"
                      variant="text"
                    />
                  </OuiStack>
                  <OuiSkeleton
                    width="4rem"
                    height="1.5rem"
                    variant="rectangle"
                    rounded
                  />
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
                <svg
                  class="h-10 w-10"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="1"
                    d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"
                  />
                </svg>
              </OuiBox>
              <OuiText as="h3" weight="medium" color="primary">
                No deployments yet
              </OuiText>
              <OuiText size="xs" color="secondary">
                Deploy your first application to see it listed here.
              </OuiText>
              <OuiButton
                variant="ghost"
                size="sm"
                class="mt-2"
                @click="navigateTo('/deployments')"
              >
                View deployments
              </OuiButton>
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
                @click="navigateTo('/deployments')"
              >
                <OuiFlex justify="between" align="center" gap="md">
                  <OuiStack gap="xs" class="flex-1 min-w-0">
                    <OuiText
                      as="h3"
                      weight="medium"
                      color="primary"
                      class="truncate"
                    >
                      {{ deployment.name }}
                    </OuiText>
                    <OuiText size="sm" color="secondary" class="truncate">
                      {{ deployment.domain }}
                    </OuiText>
                  </OuiStack>
                  <OuiBadge :variant="statusVariant(deployment.status)">
                    {{ deployment.status }}
                  </OuiBadge>
                </OuiFlex>
              </OuiBox>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Activity Feed -->
        <OuiCard>
          <OuiCardHeader>
            <OuiFlex align="center" justify="between">
              <OuiStack gap="xs">
                <OuiText as="h2" class="oui-card-title">Activity Feed</OuiText>
                <OuiText size="xs" color="secondary">
                  Latest changes from across your teams
                </OuiText>
              </OuiStack>
              <OuiFlex align="center" gap="xs">
                <OuiBox
                  v-if="!isLoading"
                  w="fit"
                  h="fit"
                  rounded="full"
                  class="h-2 w-2 bg-success"
                />
                <OuiText size="xs" color="secondary">
                  {{ isLoading ? "Syncing" : "Live" }}
                </OuiText>
              </OuiFlex>
            </OuiFlex>
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

            <OuiStack
              v-else-if="activityFeed.length === 0"
              gap="sm"
              align="center"
              class="py-12 text-center"
            >
              <OuiBox p="md" rounded="xl" class="bg-surface-muted text-muted">
                <svg
                  class="h-10 w-10"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="1"
                    d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
                  />
                </svg>
              </OuiBox>
              <OuiText as="h3" weight="medium" color="primary">
                No recent activity
              </OuiText>
              <OuiText size="xs" color="secondary">
                Once things start moving you'll see updates here.
              </OuiText>
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
                  <OuiBox
                    w="fit"
                    h="fit"
                    rounded="full"
                    class="h-2 w-2 bg-accent-primary mt-2"
                  />
                  <OuiStack gap="xs" class="flex-1 min-w-0">
                    <OuiText size="sm" color="primary" class="break-words">
                      {{ activity.message }}
                    </OuiText>
                    <OuiText size="xs" color="secondary">
                      {{ formatDate(activity.timestamp) }}
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiBox>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>
      </OuiGrid>
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
} from "@heroicons/vue/24/outline";

// Import OUI components
import OuiSkeleton from "../components/oui/Skeleton.vue";
import OuiText from "../components/oui/Text.vue";

// Page meta
definePageMeta({
  layout: "default",
  middleware: "auth",
});

// Loading state
const isLoading = ref(true);
const statsError = ref<string | null>(null);

// Dashboard stats with loading simulation
const stats = ref({
  deployments: 0,
  vpsInstances: 0,
  databases: 0,
  monthlySpend: 0,
});

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
const recentDeployments = ref<
  Array<{
    id: string;
    name: string;
    domain: string;
    status: "RUNNING" | "BUILDING" | "STOPPED" | "PENDING" | "ERROR";
  }>
>([]);

// Activity feed with loading state
const activityFeed = ref<
  Array<{
    id: string;
    message: string;
    timestamp: Date;
  }>
>([]);

// Simulated API call for stats
const loadDashboardStats = async () => {
  try {
    isLoading.value = true;
    statsError.value = null;

    // Simulate API delay
    await new Promise((resolve) => setTimeout(resolve, 1500));

    // Mock data - in real app this would come from API
    stats.value = {
      deployments: 12,
      vpsInstances: 4,
      databases: 3,
      monthlySpend: 145.5,
    };

    recentDeployments.value = [
      {
        id: "1",
        name: "My App",
        domain: "myapp.obiente.cloud",
        status: "RUNNING",
      },
      {
        id: "2",
        name: "Marketing Site",
        domain: "marketing.example.com",
        status: "BUILDING",
      },
      {
        id: "3",
        name: "API Server",
        domain: "api.example.com",
        status: "STOPPED",
      },
    ];

    activityFeed.value = [
      {
        id: "1",
        message: 'Deployment "My App" was updated',
        timestamp: new Date(Date.now() - 1000 * 60 * 30), // 30 minutes ago
      },
      {
        id: "2",
        message: 'VPS instance "web-server-1" was created',
        timestamp: new Date(Date.now() - 1000 * 60 * 60 * 2), // 2 hours ago
      },
      {
        id: "3",
        message: 'Database "prod-db" backup completed',
        timestamp: new Date(Date.now() - 1000 * 60 * 60 * 6), // 6 hours ago
      },
    ];
  } catch (error) {
    statsError.value = "Failed to load dashboard data";
    console.error("Dashboard loading error:", error);
  } finally {
    isLoading.value = false;
  }
};

// Auto-refresh data every 30 seconds
const refreshInterval = ref<NodeJS.Timeout | null>(null);

onMounted(() => {
  loadDashboardStats();
  // Set up auto-refresh
  refreshInterval.value = setInterval(loadDashboardStats, 30000);
});

onUnmounted(() => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value);
  }
});

const formatDate = (date: Date) =>
  new Intl.RelativeTimeFormat().format(
    Math.round((date.getTime() - Date.now()) / (1000 * 60)),
    "minute"
  );

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
  loadDashboardStats();
};
</script>
