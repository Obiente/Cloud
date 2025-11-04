<template>
  <OuiContainer>
    <OuiStack gap="xl">
      <!-- Access Error State -->
      <OuiCard v-if="accessError" variant="outline" class="border-danger/20">
        <OuiCardBody>
          <OuiStack gap="lg" align="center">
            <ErrorAlert
              :error="accessError"
              title="Access Denied"
              :hint="errorHint"
            />
            <OuiButton
              variant="solid"
              color="primary"
              @click="router.push('/gameservers')"
            >
              Go to Game Servers
            </OuiButton>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Game Server Content (only show if no access error) -->
      <template v-else>
        <!-- Header -->
        <OuiCard variant="outline" class="border-border-default/50">
          <OuiCardBody>
            <OuiFlex justify="between" align="start" wrap="wrap" gap="lg">
              <OuiStack gap="md" class="flex-1 min-w-0">
                <OuiFlex align="center" gap="md" wrap="wrap">
                  <OuiBox
                    p="sm"
                    rounded="xl"
                    bg="accent-primary"
                    class="bg-primary/10 ring-1 ring-primary/20 shrink-0"
                  >
                    <CubeIcon class="w-6 h-6 text-primary" />
                  </OuiBox>
                  <OuiStack gap="none" class="min-w-0 flex-1">
                    <OuiFlex align="center" gap="md">
                      <OuiText as="h1" size="2xl" weight="bold" truncate>
                        {{ gameServer.name || "Loading..." }}
                      </OuiText>
                      <OuiBadge v-if="gameServer.status" :variant="statusMeta.badge" size="xs">
                        <span
                          class="inline-flex h-1.5 w-1.5 rounded-full mr-1.5"
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
                    </OuiFlex>
                    <OuiText size="sm" color="secondary" class="hidden sm:inline">
                      <span v-if="gameServer.gameType">{{ gameServer.gameType }} • </span>
                      Last updated
                      <OuiRelativeTime
                        :value="gameServer.updatedAt ? date(gameServer.updatedAt) : undefined"
                        :style="'short'"
                      />
                    </OuiText>
                  </OuiStack>

                  <OuiFlex gap="sm" wrap="wrap" class="shrink-0">
                    <OuiButton
                      variant="ghost"
                      color="secondary"
                      size="sm"
                      @click="refreshAll"
                      :loading="isRefreshing"
                      class="gap-2"
                    >
                      <ArrowPathIcon
                        class="h-4 w-4"
                        :class="{ 'animate-spin': isRefreshing }"
                      />
                      Refresh
                    </OuiButton>
                    <OuiButton
                      v-if="gameServer.status === 'RUNNING'"
                      variant="solid"
                      color="danger"
                      size="sm"
                      @click="handleStop"
                      :loading="isStopping"
                      class="gap-2"
                    >
                      <StopIcon class="h-4 w-4" />
                      Stop
                    </OuiButton>
                    <OuiButton
                      v-if="gameServer.status === 'STOPPED'"
                      variant="solid"
                      color="success"
                      size="sm"
                      @click="handleStart"
                      :loading="isStarting"
                      class="gap-2"
                    >
                      <PlayIcon class="h-4 w-4" />
                      Start
                    </OuiButton>
                    <OuiButton
                      variant="outline"
                      size="sm"
                      @click="showSettingsDialog = true"
                      class="gap-2"
                    >
                      <Cog6ToothIcon class="h-4 w-4" />
                      Settings
                    </OuiButton>
                  </OuiFlex>
                </OuiFlex>
              </OuiStack>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>

        <!-- Overview Cards -->
        <OuiGrid cols="1" cols-md="2" cols-lg="4" gap="lg">
          <OuiCard variant="default">
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex align="center" gap="sm">
                  <CpuChipIcon class="h-5 w-5 text-secondary" />
                  <OuiText size="sm" color="secondary">vCPU Cores</OuiText>
                </OuiFlex>
                <OuiText size="2xl" weight="bold" color="primary">
                  {{ gameServer.cpuCores || "N/A" }}
                </OuiText>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <OuiCard variant="default">
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex align="center" gap="sm">
                  <CircleStackIcon class="h-5 w-5 text-secondary" />
                  <OuiText size="sm" color="secondary">Memory</OuiText>
                </OuiFlex>
                <OuiText size="2xl" weight="bold" color="primary">
                  <OuiByte :bytes="gameServer.memoryBytes || 0" />
                </OuiText>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <OuiCard variant="default">
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex align="center" gap="sm">
                  <ServerIcon class="h-5 w-5 text-secondary" />
                  <OuiText size="sm" color="secondary">Port</OuiText>
                </OuiFlex>
                <OuiText size="2xl" weight="bold" color="primary">
                  {{ gameServer.port || "N/A" }}
                </OuiText>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <OuiCard variant="default">
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex align="center" gap="sm">
                  <ChartBarIcon class="h-5 w-5 text-secondary" />
                  <OuiText size="sm" color="secondary">Monthly Cost</OuiText>
                </OuiFlex>
                <OuiText size="2xl" weight="bold" color="primary">
                  {{ formatCurrency(gameServer.estimatedMonthlyCost || 0) }}
                </OuiText>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </OuiGrid>

        <!-- Tabs -->
        <OuiTabs v-model="activeTab" class="w-full">
          <OuiTabsList class="w-full">
            <OuiTabsTrigger value="overview">Overview</OuiTabsTrigger>
            <OuiTabsTrigger value="logs">Logs</OuiTabsTrigger>
            <OuiTabsTrigger value="metrics">Metrics</OuiTabsTrigger>
            <OuiTabsTrigger value="settings">Settings</OuiTabsTrigger>
          </OuiTabsList>

          <OuiTabsContent value="overview">
            <OuiCard variant="default">
              <OuiCardBody>
                <OuiStack gap="lg">
                  <OuiText as="h2" size="lg" weight="semibold" color="primary">
                    Game Server Information
                  </OuiText>
                  <OuiStack gap="md">
                    <OuiFlex justify="between" align="center">
                      <OuiText size="sm" color="secondary">Game Type</OuiText>
                      <OuiText size="sm" weight="medium" color="primary">
                        {{ gameServer.gameType || "Not set" }}
                      </OuiText>
                    </OuiFlex>
                    <OuiFlex justify="between" align="center">
                      <OuiText size="sm" color="secondary">Status</OuiText>
                      <OuiBadge :variant="statusMeta.badge" size="sm">
                        {{ statusMeta.label }}
                      </OuiBadge>
                    </OuiFlex>
                    <OuiFlex justify="between" align="center">
                      <OuiText size="sm" color="secondary">Created</OuiText>
                      <OuiText size="sm" weight="medium" color="primary">
                        <OuiRelativeTime
                          :value="gameServer.createdAt ? date(gameServer.createdAt) : undefined"
                        />
                      </OuiText>
                    </OuiFlex>
                    <OuiFlex justify="between" align="center">
                      <OuiText size="sm" color="secondary">Last Updated</OuiText>
                      <OuiText size="sm" weight="medium" color="primary">
                        <OuiRelativeTime
                          :value="gameServer.updatedAt ? date(gameServer.updatedAt) : undefined"
                        />
                      </OuiText>
                    </OuiFlex>
                  </OuiStack>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>
          </OuiTabsContent>

          <OuiTabsContent value="logs">
            <OuiCard variant="default">
              <OuiCardBody>
                <OuiStack gap="md">
                  <OuiText as="h2" size="lg" weight="semibold" color="primary">
                    Game Server Logs
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Logs will appear here once the game server API is implemented.
                  </OuiText>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>
          </OuiTabsContent>

          <OuiTabsContent value="metrics">
            <OuiCard variant="default">
              <OuiCardBody>
                <OuiStack gap="md">
                  <OuiText as="h2" size="lg" weight="semibold" color="primary">
                    Performance Metrics
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Metrics will appear here once the game server API is implemented.
                  </OuiText>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>
          </OuiTabsContent>

          <OuiTabsContent value="settings">
            <OuiCard variant="default">
              <OuiCardBody>
                <OuiStack gap="md">
                  <OuiText as="h2" size="lg" weight="semibold" color="primary">
                    Game Server Settings
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Settings will appear here once the game server API is implemented.
                  </OuiText>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>
          </OuiTabsContent>
        </OuiTabs>
      </template>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  ArrowPathIcon,
  ChartBarIcon,
  Cog6ToothIcon,
  CpuChipIcon,
  CubeIcon,
  PlayIcon,
  ServerIcon,
  StopIcon,
  CircleStackIcon,
} from "@heroicons/vue/24/outline";

import ErrorAlert from "~/components/ErrorAlert.vue";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import OuiByte from "~/components/oui/Byte.vue";
import { date } from "@obiente/proto/utils";
import { useToast } from "~/composables/useToast";

definePageMeta({
  layout: "default",
  middleware: "auth",
});

const route = useRoute();
const router = useRouter();
const { toast } = useToast();

const gameServerId = computed(() => route.params.id as string);

// State
const accessError = ref<Error | null>(null);
const errorHint = computed(() => {
  return "You may not have permission to view this game server, or it may not exist.";
});

const isRefreshing = ref(false);
const isStarting = ref(false);
const isStopping = ref(false);
const showSettingsDialog = ref(false);
const activeTab = ref("overview");

// Placeholder game server data (will be replaced with API call)
const gameServer = ref<{
  id: string;
  name: string;
  gameType?: string;
  status: string;
  port?: number;
  cpuCores?: number;
  memoryBytes?: number;
  estimatedMonthlyCost?: number;
  createdAt?: string;
  updatedAt?: string;
}>({
  id: gameServerId.value,
  name: "Loading...",
  status: "STOPPED",
});

// Status metadata helper
const statusMeta = computed(() => {
  const statusMap: Record<string, any> = {
    RUNNING: {
      label: "Running",
      badge: "success" as const,
      dotClass: "bg-success",
    },
    STOPPED: {
      label: "Stopped",
      badge: "muted" as const,
      dotClass: "bg-muted",
    },
    STARTING: {
      label: "Starting",
      badge: "warning" as const,
      dotClass: "bg-warning",
    },
    STOPPING: {
      label: "Stopping",
      badge: "warning" as const,
      dotClass: "bg-warning",
    },
    ERROR: {
      label: "Error",
      badge: "danger" as const,
      dotClass: "bg-danger",
    },
  };

  return (
    statusMap[gameServer.value.status] || {
      label: "Unknown",
      badge: "muted" as const,
      dotClass: "bg-muted",
    }
  );
});

// Actions
const refreshAll = async () => {
  isRefreshing.value = true;
  try {
    // TODO: Implement refresh API call
    await new Promise((resolve) => setTimeout(resolve, 1000));
    toast.success("Game server refreshed");
  } catch (error) {
    toast.error("Failed to refresh game server");
  } finally {
    isRefreshing.value = false;
  }
};

const handleStart = async () => {
  isStarting.value = true;
  try {
    // TODO: Implement start API call
    await new Promise((resolve) => setTimeout(resolve, 1000));
    gameServer.value.status = "RUNNING";
    toast.success("Game server started");
  } catch (error) {
    toast.error("Failed to start game server");
  } finally {
    isStarting.value = false;
  }
};

const handleStop = async () => {
  isStopping.value = true;
  try {
    // TODO: Implement stop API call
    await new Promise((resolve) => setTimeout(resolve, 1000));
    gameServer.value.status = "STOPPED";
    toast.success("Game server stopped");
  } catch (error) {
    toast.error("Failed to stop game server");
  } finally {
    isStopping.value = false;
  }
};

const formatCurrency = (cents: number) => {
  return `$${(cents / 100).toFixed(2)}`;
};

// TODO: Load game server data on mount
// onMounted(() => {
//   loadGameServer();
// });
</script>

