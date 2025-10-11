<template>
  <div>
    <!-- Page Header -->
    <div class="mb-8">
      <OuiText as="h1" size="3xl" weight="bold" color="primary"
        >Dashboard</OuiText
      >
      <OuiText color="secondary" class="mt-2"
        >Welcome to your Obiente Cloud dashboard</OuiText
      >
    </div>

    <!-- Error State -->
    <div v-if="statsError" class="mb-8">
      <OuiCard variant="outline" class="border-danger">
        <OuiCardBody>
          <div class="flex items-center justify-between">
            <div class="flex items-center">
              <div class="p-2 bg-danger/10 rounded-lg mr-4">
                <svg
                  class="w-6 h-6 text-danger"
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
              </div>
              <div>
                <OuiText as="h3" class="text-danger font-medium">{{ statsError }}</OuiText>
                <OuiText class="text-secondary text-sm">
                  Unable to load dashboard data
                </OuiText>
              </div>
            </div>
            <OuiButton @click="retryLoad" variant="outline" size="sm">
              Retry
            </OuiButton>
          </div>
        </OuiCardBody>
      </OuiCard>
    </div>

    <!-- Quick Stats -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
      <!-- Deployments Card -->
      <OuiCard variant="raised" hoverable @click="navigateTo('/deployments')">
        <OuiCardBody>
          <div class="flex items-center">
            <div class="p-2 bg-primary/10 rounded-lg">
              <RocketLaunchIcon class="w-6 h-6 text-accent-primary" />
            </div>
            <div class="ml-4 flex-1">
              <OuiSkeleton
                v-if="isLoading"
                class="mb-1"
                width="3rem"
                height="1.5rem"
                variant="text"
              />
              <OuiText v-else as="h3" size="lg" weight="semibold" color="primary">
                {{ stats.deployments }}
              </OuiText>
              <OuiText size="sm" color="secondary">Deployments</OuiText>
            </div>
          </div>
        </OuiCardBody>
      </OuiCard>

      <!-- VPS Instances Card -->
      <OuiCard variant="raised" hoverable @click="navigateTo('/vps')">
        <OuiCardBody>
          <div class="flex items-center">
            <div class="p-2 bg-success/10 rounded-lg">
              <ServerIcon class="w-6 h-6 text-success" />
            </div>
            <div class="ml-4 flex-1">
              <OuiSkeleton
                v-if="isLoading"
                class="mb-1"
                width="2rem"
                height="1.5rem"
                variant="text"
              />
              <OuiText v-else as="h3" size="lg" weight="semibold" color="primary">
                {{ stats.vpsInstances }}
              </OuiText>
              <OuiText size="sm" color="secondary">VPS Instances</OuiText>
            </div>
          </div>
        </OuiCardBody>
      </OuiCard>

      <!-- Databases Card -->
      <OuiCard variant="raised" hoverable @click="navigateTo('/databases')">
        <OuiCardBody>
          <div class="flex items-center">
            <div class="p-2 bg-accent-secondary/10 rounded-lg">
              <CircleStackIcon class="w-6 h-6 text-accent-secondary" />
            </div>
            <div class="ml-4 flex-1">
              <OuiSkeleton
                v-if="isLoading"
                class="mb-1"
                width="2rem"
                height="1.5rem"
                variant="text"
              />
                            <OuiText v-else as="h3" size="lg" weight="semibold" color="primary">
                {{ stats.databases }}
              </OuiText>
              <OuiText size="sm" color="secondary">Databases</OuiText>
            </div>
          </div>
        </OuiCardBody>
      </OuiCard>

      <!-- Monthly Spend Card -->
      <OuiCard variant="raised" hoverable @click="navigateTo('/billing')">
        <OuiCardBody>
          <div class="flex items-center">
            <div class="p-2 bg-warning/10 rounded-lg">
              <CreditCardIcon class="w-6 h-6 text-warning" />
            </div>
            <div class="ml-4 flex-1">
              <OuiSkeleton
                v-if="isLoading"
                class="mb-1"
                width="4rem"
                height="1.5rem"
                variant="text"
              />
                            <OuiText v-else as="h3" size="lg" weight="semibold" color="primary">
                {{ formatCurrency(stats.monthlySpend) }}
              </OuiText>
              <OuiText size="sm" color="secondary">This Month</OuiText>
            </div>
          </div>
        </OuiCardBody>
      </OuiCard>
    </div>

    <!-- Recent Activity -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
      <!-- Recent Deployments -->
      <OuiCard>
        <OuiCardHeader>
          <div class="flex items-center justify-between">
            <OuiText as="h2" class="oui-card-title">Recent Deployments</OuiText>
            <div class="flex items-center space-x-2">
              <div
                v-if="!isLoading"
                class="w-2 h-2 bg-success rounded-full"
              ></div>
              <OuiText size="xs" color="secondary">
                {{ isLoading ? "Loading..." : "Live" }}
              </OuiText>
            </div>
          </div>
        </OuiCardHeader>
        <OuiCardBody>
          <div v-if="isLoading" class="space-y-4">
            <div
              v-for="i in 3"
              :key="i"
              class="p-3 border border-default rounded-lg"
            >
              <div class="flex items-center justify-between">
                <div class="space-y-2 flex-1">
                  <OuiSkeleton width="8rem" height="1rem" variant="text" />
                  <OuiSkeleton width="12rem" height="0.75rem" variant="text" />
                </div>
                <OuiSkeleton
                  width="4rem"
                  height="1.5rem"
                  variant="rectangle"
                  rounded
                />
              </div>
            </div>
          </div>
          <div
            v-else-if="recentDeployments.length === 0"
            class="text-secondary text-center py-8"
          >
            <div class="mb-4">
              <svg
                class="w-12 h-12 mx-auto text-muted"
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
            </div>
            <OuiText>No deployments yet</OuiText>
            <OuiText size="xs" class="mt-1">
              Deploy your first application to get started
            </OuiText>
          </div>
          <div v-else class="space-y-4">
            <div
              v-for="deployment in recentDeployments"
              :key="deployment.id"
              class="flex items-center justify-between p-3 border border-default rounded-lg hover:bg-surface-hover transition-colors cursor-pointer"
            >
              <div class="flex-1">
                <OuiText as="h3" weight="medium" color="primary">{{ deployment.name }}</OuiText>
                <OuiText size="sm" color="secondary">{{ deployment.domain }}</OuiText>
              </div>
              <div class="flex items-center space-x-2">
                <OuiBadge
                  :variant="
                    deployment.status === 'RUNNING'
                      ? 'success'
                      : deployment.status === 'BUILDING'
                      ? 'warning'
                      : deployment.status === 'ERROR'
                      ? 'danger'
                      : 'secondary'
                  "
                >
                  {{ deployment.status }}
                </OuiBadge>
              </div>
            </div>
          </div>
        </OuiCardBody>
      </OuiCard>

      <!-- Activity Feed -->
      <OuiCard>
        <OuiCardHeader>
          <div class="flex items-center justify-between">
            <OuiText as="h2" class="oui-card-title">Activity Feed</OuiText>
            <div class="flex items-center space-x-2">
              <div
                v-if="!isLoading"
                class="w-2 h-2 bg-success rounded-full"
              ></div>
              <OuiText size="xs" color="secondary">
                {{ isLoading ? "Loading..." : "Live" }}
              </OuiText>
            </div>
          </div>
        </OuiCardHeader>
        <OuiCardBody>
          <div v-if="isLoading" class="space-y-4">
            <div v-for="i in 3" :key="i" class="flex items-start space-x-3">
              <OuiSkeleton
                width="0.5rem"
                height="0.5rem"
                variant="circle"
                class="mt-2"
              />
              <div class="space-y-2 flex-1">
                <OuiSkeleton width="10rem" height="1rem" variant="text" />
                <OuiSkeleton width="5rem" height="0.75rem" variant="text" />
              </div>
            </div>
          </div>
          <div
            v-else-if="activityFeed.length === 0"
            class="text-secondary text-center py-8"
          >
            <div class="mb-4">
              <svg
                class="w-12 h-12 mx-auto text-muted"
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
            </div>
            <OuiText>No recent activity</OuiText>
            <OuiText size="xs" class="mt-1">
              Activity will appear here as you use the platform
            </OuiText>
          </div>
          <div v-else class="space-y-4">
            <div
              v-for="activity in activityFeed"
              :key="activity.id"
              class="flex items-start space-x-3 hover:bg-surface-hover transition-colors rounded-lg p-2 -m-2"
            >
              <div
                class="w-2 h-2 bg-accent-primary rounded-full mt-2 flex-shrink-0"
              ></div>
              <div class="flex-1 min-w-0">
                <OuiText size="sm" color="primary" class="break-words">
                  {{ activity.message }}
                </OuiText>
                <OuiText size="xs" color="secondary">
                  {{ formatDate(activity.timestamp) }}
                </OuiText>
              </div>
            </div>
          </div>
        </OuiCardBody>
      </OuiCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  RocketLaunchIcon,
  ServerIcon,
  CircleStackIcon,
  CreditCardIcon,
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

// Helper functions
const getStatusClass = (status: string) => {
  switch (status) {
    case "RUNNING":
      return "bg-success/10 text-success border border-success/20";
    case "STOPPED":
      return "bg-danger/10 text-danger border border-danger/20";
    case "BUILDING":
      return "bg-warning/10 text-warning border border-warning/20";
    case "PENDING":
      return "bg-info/10 text-info border border-info/20";
    case "ERROR":
      return "bg-danger/10 text-danger border border-danger/20";
    default:
      return "bg-surface-muted text-secondary border border-muted";
  }
};

const formatDate = (date: Date) => {
  return new Intl.RelativeTimeFormat().format(
    Math.floor((date.getTime() - Date.now()) / (1000 * 60)),
    "minute"
  );
};

const formatCurrency = (amount: number) => {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(amount);
};

// Retry function for error states
const retryLoad = () => {
  loadDashboardStats();
};
</script>
