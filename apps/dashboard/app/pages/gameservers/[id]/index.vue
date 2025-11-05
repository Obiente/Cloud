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

      <!-- Game Server Content (only show if no access error and game server exists) -->
      <template v-else-if="gameServer">
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
                      <span v-if="gameServer.gameType !== undefined">{{ getGameTypeLabel(gameServer.gameType) }} • </span>
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
                      <OuiText as="span" size="xs" weight="medium">Refresh</OuiText>
                    </OuiButton>
                    <OuiButton
                      :color="gameServer.status === 'RUNNING' ? 'danger' : 'success'"
                      variant="outline"
                      size="sm"
                      class="gap-2"
                      :loading="isStarting || isStopping || isRestarting"
                      :disabled="isActionDisabled(gameServer.status)"
                      @click="toggleServerStatus"
                    >
                      <template v-if="gameServer.status === 'RUNNING'">
                        <StopIcon class="h-4 w-4" />
                        <OuiText as="span" size="xs" weight="medium">Stop</OuiText>
                      </template>
                      <template v-else-if="gameServer.status === 'STOPPED' || gameServer.status === 'FAILED'">
                        <PlayIcon class="h-4 w-4" />
                        <OuiText as="span" size="xs" weight="medium">Start</OuiText>
                      </template>
                      <template v-else-if="gameServer.status === 'STARTING'">
                        <PlayIcon class="h-4 w-4 animate-pulse" />
                        <OuiText as="span" size="xs" weight="medium">Starting...</OuiText>
                      </template>
                      <template v-else-if="gameServer.status === 'STOPPING'">
                        <StopIcon class="h-4 w-4 animate-pulse" />
                        <OuiText as="span" size="xs" weight="medium">Stopping...</OuiText>
                      </template>
                      <template v-else-if="gameServer.status === 'RESTARTING'">
                        <ArrowPathIcon class="h-4 w-4 animate-spin" />
                        <OuiText as="span" size="xs" weight="medium">Restarting...</OuiText>
                      </template>
                      <template v-else>
                        <PlayIcon class="h-4 w-4" />
                        <OuiText as="span" size="xs" weight="medium">Start</OuiText>
                      </template>
                    </OuiButton>
                    <OuiButton
                      variant="outline"
                      color="secondary"
                      size="sm"
                      class="gap-2"
                      :disabled="isActionDisabled(gameServer.status)"
                      @click="restartServer"
                    >
                      <ArrowPathIcon class="h-4 w-4" />
                      <OuiText as="span" size="xs" weight="medium">Restart</OuiText>
                    </OuiButton>
                    <OuiButton
                      variant="outline"
                      color="danger"
                      size="sm"
                      class="gap-2"
                      @click="showDeleteDialog = true"
                    >
                      <TrashIcon class="h-4 w-4" />
                      <OuiText as="span" size="xs" weight="medium">Delete</OuiText>
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
                  <OuiText size="sm" color="secondary">vCPU Cores (Max)</OuiText>
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
                  <OuiText size="sm" color="secondary">Memory (Max)</OuiText>
                </OuiFlex>
                <OuiText size="2xl" weight="bold" color="primary">
                  <OuiByte :value="getMemoryBytesValue(gameServer.memoryBytes)" />
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
                  {{ formatCurrency(estimatedMonthlyCost) }}
                </OuiText>
                <OuiText size="xs" color="secondary" class="mt-1">
                  Pay-as-you-go based on actual usage
                </OuiText>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </OuiGrid>

        <!-- Tabs -->
        <OuiStack gap="md">
          <OuiTabs v-model="activeTab" :tabs="tabs" />
          <OuiCard variant="default">
            <OuiTabs v-model="activeTab" :tabs="tabs" :content-only="true">
              <template #overview>
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
                            {{ gameServer.gameType !== undefined ? getGameTypeLabel(gameServer.gameType) : "Not set" }}
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

                <!-- Connection Information Card -->
                <OuiCard v-if="gameServer.status === 'RUNNING' && gameServer.port" variant="default">
                  <OuiCardBody>
                    <OuiStack gap="lg">
                      <OuiStack gap="sm">
                        <OuiFlex align="center" gap="sm">
                          <GlobeAltIcon class="h-5 w-5 text-accent-primary" />
                          <OuiText as="h2" size="lg" weight="semibold" color="primary">
                            Connection Information
                          </OuiText>
                        </OuiFlex>
                        <OuiText size="sm" color="secondary">
                          Use this information to connect to your game server
                        </OuiText>
                      </OuiStack>

                      <!-- SRV Records (for games that support them) -->
                      <OuiStack v-if="hasSRVRecords" gap="md">
                        <OuiStack gap="xs">
                          <OuiText size="sm" weight="medium" color="secondary">
                            SRV Records (Recommended)
                          </OuiText>
                          <OuiText size="xs" color="secondary" class="opacity-75">
                            Use these domains for automatic port resolution
                          </OuiText>
                        </OuiStack>
                        <OuiCard 
                          v-for="(srv, index) in srvDomains" 
                          :key="index"
                          variant="outline" 
                          class="bg-surface-muted/30"
                        >
                          <OuiCardBody>
                            <OuiStack gap="sm">
                              <OuiText size="xs" weight="medium" color="secondary">
                                {{ srv.label }}
                              </OuiText>
                              <OuiText size="xs" color="secondary" class="opacity-75">
                                {{ srv.description }}
                              </OuiText>
                              <OuiFlex justify="between" align="center" gap="md" wrap="wrap">
                                <OuiText 
                                  size="sm" 
                                  weight="medium" 
                                  color="primary"
                                  class="font-mono break-all"
                                >
                                  {{ srv.domain }}
                                </OuiText>
                                <OuiButton
                                  variant="ghost"
                                  size="sm"
                                  @click="copyToClipboard(srv.domain)"
                                  class="shrink-0 gap-2"
                                >
                                  <ClipboardIcon class="h-4 w-4" />
                                  <OuiText as="span" size="xs">Copy</OuiText>
                                </OuiButton>
                              </OuiFlex>
                            </OuiStack>
                          </OuiCardBody>
                        </OuiCard>
                      </OuiStack>

                      <!-- Direct Connection Info -->
                      <OuiStack gap="md">
                        <OuiStack gap="xs">
                          <OuiText size="sm" weight="medium" color="secondary">
                            Direct Connection
                          </OuiText>
                          <OuiText size="xs" color="secondary" class="opacity-75">
                            Use this if SRV records are not supported
                          </OuiText>
                        </OuiStack>
                        <OuiCard variant="outline" class="bg-surface-muted/30">
                          <OuiCardBody>
                            <OuiStack gap="sm">
                              <OuiFlex justify="between" align="center" gap="md" wrap="wrap">
                                <OuiStack gap="xs" class="flex-1 min-w-0">
                                  <OuiText size="xs" color="secondary">Domain</OuiText>
                                  <OuiText 
                                    size="sm" 
                                    weight="medium" 
                                    color="primary"
                                    class="font-mono break-all"
                                  >
                                    {{ connectionDomain }}
                                  </OuiText>
                                </OuiStack>
                                <OuiButton
                                  variant="ghost"
                                  size="sm"
                                  @click="copyToClipboard(connectionDomain)"
                                  class="shrink-0 gap-2"
                                >
                                  <ClipboardIcon class="h-4 w-4" />
                                  <OuiText as="span" size="xs">Copy</OuiText>
                                </OuiButton>
                              </OuiFlex>
                              <OuiFlex justify="between" align="center" gap="md" wrap="wrap">
                                <OuiStack gap="xs" class="flex-1 min-w-0">
                                  <OuiText size="xs" color="secondary">Port</OuiText>
                                  <OuiText 
                                    size="sm" 
                                    weight="medium" 
                                    color="primary"
                                    class="font-mono"
                                  >
                                    {{ gameServer.port }}
                                  </OuiText>
                                </OuiStack>
                                <OuiButton
                                  variant="ghost"
                                  size="sm"
                                  @click="copyToClipboard(gameServer.port.toString())"
                                  class="shrink-0 gap-2"
                                >
                                  <ClipboardIcon class="h-4 w-4" />
                                  <OuiText as="span" size="xs">Copy</OuiText>
                                </OuiButton>
                              </OuiFlex>
                            </OuiStack>
                          </OuiCardBody>
                        </OuiCard>
                      </OuiStack>

                      <!-- Instructions -->
                      <OuiStack gap="sm" class="mt-2">
                        <OuiText size="xs" weight="medium" color="secondary">
                          How to Connect:
                        </OuiText>
                        <OuiStack gap="xs" class="pl-4">
                          <OuiText size="xs" color="secondary" class="list-item">
                            <span v-if="hasSRVRecords">
                              For games with SRV records: Use the SRV record domain in your game client. The port will be resolved automatically.
                            </span>
                            <span v-else>
                              Use the domain and port shown above to connect to your server.
                            </span>
                          </OuiText>
                          <OuiText size="xs" color="secondary" class="list-item">
                            The domain <code class="px-1 py-0.5 rounded bg-surface-muted text-xs font-mono">{{ connectionDomain }}</code> will automatically resolve to the correct server IP.
                          </OuiText>
                        </OuiStack>
                      </OuiStack>
                    </OuiStack>
                  </OuiCardBody>
                </OuiCard>
              </template>
            <template #logs>
              <GameServerLogs
                :game-server-id="gameServerId"
                :organization-id="gameServer?.organizationId || ''"
              />
            </template>
              <template #metrics>
                <GameServerMetrics
                  :game-server-id="gameServerId"
                  :organization-id="gameServer?.organizationId || ''"
                  :game-server-status="gameServer?.status !== undefined ? Number(gameServer.status) : undefined"
                />
              </template>
            <template #files>
              <GameServerFiles
                :game-server-id="gameServerId"
              />
            </template>
            <template #settings>
              <GameServerSettings
                :game-server="gameServerData as any"
                @saved="refreshAll"
                @delete="showDeleteDialog = true"
              />
            </template>
            </OuiTabs>
          </OuiCard>
        </OuiStack>
      </template>
      
      <!-- Loading or Not Found State -->
      <template v-else-if="!accessError">
        <OuiStack align="center" gap="lg" class="text-center py-20">
          <OuiBox
            class="inline-flex items-center justify-center w-20 h-20 rounded-xl bg-surface-muted/50 ring-1 ring-border-muted"
          >
            <CubeIcon class="h-10 w-10 text-secondary" />
          </OuiBox>
          <OuiStack align="center" gap="sm">
            <OuiText as="h3" size="xl" weight="semibold" color="primary">
              Game server not found
            </OuiText>
            <OuiText color="secondary">
              The game server you are looking for does not exist or you do not have access.
            </OuiText>
          </OuiStack>
          <OuiButton
            color="primary"
            class="gap-2 shadow-lg shadow-primary/20"
            @click="router.push('/gameservers')"
          >
            <ArrowLeftIcon class="h-4 w-4" />
            <OuiText as="span" size="sm" weight="medium"
              >Go to Game Servers List</OuiText
            >
          </OuiButton>
        </OuiStack>
      </template>
    </OuiStack>

    <!-- Delete Confirmation Dialog -->
    <OuiDialog v-model:open="showDeleteDialog" title="Delete Game Server">
      <OuiText color="secondary">
        Are you sure you want to delete the game server
        <OuiText as="span" weight="semibold" color="primary">{{ gameServer?.name }}</OuiText>?
        This action cannot be undone.
      </OuiText>
      <template #footer>
        <OuiButton variant="ghost" @click="showDeleteDialog = false">Cancel</OuiButton>
        <OuiButton color="danger" :loading="isDeleting" @click="deleteGameServer">Delete</OuiButton>
      </template>
    </OuiDialog>
  </OuiContainer>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  ArrowPathIcon,
  ArrowLeftIcon,
  ChartBarIcon,
  Cog6ToothIcon,
  CpuChipIcon,
  CubeIcon,
  PlayIcon,
  ServerIcon,
  StopIcon,
  CircleStackIcon,
  TrashIcon,
  DocumentTextIcon,
  GlobeAltIcon,
  ClipboardIcon,
  FolderIcon,
} from "@heroicons/vue/24/outline";

import type { TabItem } from "~/components/oui/Tabs.vue";
import { useTabQuery } from "~/composables/useTabQuery";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import OuiByte from "~/components/oui/Byte.vue";
import GameServerMetrics from "~/components/gameserver/GameServerMetrics.vue";
import GameServerLogs from "~/components/gameserver/GameServerLogs.vue";
import GameServerFiles from "~/components/gameserver/GameServerFiles.vue";
import GameServerSettings from "~/components/gameserver/GameServerSettings.vue";
import { date } from "@obiente/proto/utils";
import { useToast } from "~/composables/useToast";
import { useConnectClient } from "~/lib/connect-client";
import { GameServerService, GameType } from "@obiente/proto";
import { useOrganizationsStore } from "~/stores/organizations";
import { SuperadminService } from "@obiente/proto";

definePageMeta({
  layout: "default",
  middleware: "auth",
});

const route = useRoute();
const router = useRouter();
const { toast } = useToast();
const orgsStore = useOrganizationsStore();
const client = useConnectClient(GameServerService);
const superadminClient = useConnectClient(SuperadminService);

const gameServerId = computed(() => route.params.id as string);
const effectiveOrgId = computed(() => orgsStore.currentOrgId || "");

// State
const accessError = ref<Error | null>(null);
const errorHint = computed(() => {
  return "You may not have permission to view this game server, or it may not exist.";
});

// Fetch game server data
const { data: gameServerData, refresh: refreshGameServer, error: fetchError } = useAsyncData(
  () => `game-server-${gameServerId.value}`,
  async () => {
    try {
      const res = await client.getGameServer({
        gameServerId: gameServerId.value,
      });
      // Clear any previous errors on success
      accessError.value = null;
      return res.gameServer ?? null;
    } catch (err: any) {
      // Check if it's a permission denied or not found error
      if (err.code === "permission_denied" || err.code === "not_found") {
        accessError.value = err;
        return null;
      }
      // Re-throw other errors
      throw err;
    }
  },
  {
    watch: [gameServerId],
  }
);

// Watch for fetch errors and handle access errors
watch(fetchError, (err) => {
  if (err && (err as any).code === "permission_denied" || (err as any).code === "not_found") {
    accessError.value = err as Error;
  }
});

// Computed game server from fetched data
const gameServer = computed(() => {
  const data = gameServerData.value;
  if (!data) return null;
  
  // Convert status from enum number to string if needed
  let status: string = data.status?.toString() || 'CREATED';
  if (typeof data.status === 'number') {
    // Map GameServerStatus enum values
    const statusMap: Record<number, string> = {
      0: 'CREATED',
      1: 'STARTING',
      2: 'STOPPING',
      3: 'RUNNING',
      4: 'RESTARTING',
      5: 'STOPPED',
      6: 'FAILED',
    };
    status = statusMap[data.status] || 'CREATED';
  }
  
  return {
    ...data,
    status: status,
    gameType: typeof data.gameType === 'number' ? data.gameType : (data.gameType ? Number(data.gameType) : undefined),
  };
});
const usageData = ref<any>(null);

// State variables
const isRefreshing = ref(false);
const isStarting = ref(false);
const isStopping = ref(false);
const isRestarting = ref(false);
const isDeleting = ref(false);
const showDeleteDialog = ref(false);

// Tabs definition
const tabs = computed<TabItem[]>(() => [
  { id: "overview", label: "Overview", icon: CubeIcon },
  { id: "logs", label: "Logs", icon: DocumentTextIcon },
  { id: "metrics", label: "Metrics", icon: ChartBarIcon },
  { id: "files", label: "Files", icon: FolderIcon },
  { id: "settings", label: "Settings", icon: Cog6ToothIcon },
]);

// Use composable for tab query parameter management
const activeTab = useTabQuery(tabs);

// Status metadata helper
const statusMeta = computed(() => {
  if (!gameServer.value) {
    return {
      label: "Unknown",
      badge: "muted" as const,
      dotClass: "bg-muted",
    };
  }

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
    RESTARTING: {
      label: "Restarting",
      badge: "warning" as const,
      dotClass: "bg-warning",
    },
    FAILED: {
      label: "Failed",
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

// Estimated monthly cost based on actual usage and pricing
const estimatedMonthlyCost = computed(() => {
  // Use actual usage data if available
  if (usageData.value?.totalCostCents) {
    return Number(usageData.value.totalCostCents) / 100;
  }
  
  // Otherwise estimate based on configuration if running 24/7
  // This is just an estimate - actual cost depends on usage
  if (gameServer.value?.cpuCores && gameServer.value?.memoryBytes) {
    const cpuCores = gameServer.value.cpuCores;
    const memoryBytes = typeof gameServer.value.memoryBytes === 'bigint' 
      ? Number(gameServer.value.memoryBytes) 
      : gameServer.value.memoryBytes;
    const memoryGB = memoryBytes / (1024 * 1024 * 1024);
    
    // Use actual pricing from API if available, otherwise estimate
    // Pricing: ~$2/core/month + ~$3/GB/month for 24/7 usage
    // Actual cost is pay-as-you-go based on actual CPU and memory usage
    return (cpuCores * 2) + (memoryGB * 3);
  }
  return 0;
});

// Helper function to convert BigInt to number for memoryBytes
const getMemoryBytesValue = (value: bigint | number | undefined | null): number => {
  if (!value) return 0;
  if (typeof value === 'bigint') return Number(value);
  return value;
};

// Format helpers
const formatMemory = (bytes: number) => {
  if (!bytes) return "0 B";
  const gb = bytes / (1024 * 1024 * 1024);
  if (gb >= 1) return `${gb.toFixed(2)} GB`;
  const mb = bytes / (1024 * 1024);
  if (mb >= 1) return `${mb.toFixed(2)} MB`;
  return `${bytes} B`;
};

const formatStorage = (bytes: number) => {
  return formatMemory(bytes);
};

const formatCurrency = (dollars: number) => {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(dollars);
};

const getGameTypeLabel = (gameType: number) => {
  // Map GameType enum values to labels
  const types: Record<number, string> = {
    [GameType.MINECRAFT]: "Minecraft",
    [GameType.MINECRAFT_JAVA]: "Minecraft Java",
    [GameType.MINECRAFT_BEDROCK]: "Minecraft Bedrock",
    [GameType.VALHEIM]: "Valheim",
    [GameType.TERRARIA]: "Terraria",
    [GameType.RUST]: "Rust",
    [GameType.CS2]: "Counter-Strike 2",
    [GameType.TF2]: "Team Fortress 2",
    [GameType.ARK]: "ARK: Survival Evolved",
    [GameType.CONAN]: "Conan Exiles",
    [GameType.SEVEN_DAYS]: "7 Days to Die",
    [GameType.FACTORIO]: "Factorio",
    [GameType.SPACED_ENGINEERS]: "Space Engineers",
    [GameType.OTHER]: "Other",
  };
  return types[gameType] || "Unknown";
};

// Connection domain helpers
const connectionDomain = computed(() => {
  if (!gameServer.value?.id) return "";
  // Format: gameserver-123.my.obiente.cloud
  return `gameserver-${gameServer.value.id}.my.obiente.cloud`;
});

// Get SRV domains based on game type
const srvDomains = computed(() => {
  if (!gameServer.value?.id || !gameServer.value?.gameType) return [];
  
  const gameType = typeof gameServer.value.gameType === 'number'
    ? gameServer.value.gameType as GameType
    : gameServer.value.gameType;
  const id = gameServer.value.id;
  const domains: Array<{ label: string; domain: string; description: string }> = [];
  
  // GameType enum values: MINECRAFT = 1, MINECRAFT_JAVA = 2, MINECRAFT_BEDROCK = 3, RUST = 6
  if (gameType === GameType.MINECRAFT || gameType === GameType.MINECRAFT_JAVA) {
    // Minecraft Java Edition - TCP SRV record
    domains.push({
      label: "Minecraft Java (SRV)",
      domain: `_minecraft._tcp.gameserver-${id}.my.obiente.cloud`,
      description: "Use this domain in Minecraft Java Edition for automatic port resolution"
    });
  }
  
  if (gameType === GameType.MINECRAFT || gameType === GameType.MINECRAFT_BEDROCK) {
    // Minecraft Bedrock Edition - UDP SRV record
    domains.push({
      label: "Minecraft Bedrock (SRV)",
      domain: `_minecraft._udp.gameserver-${id}.my.obiente.cloud`,
      description: "Use this domain in Minecraft Bedrock Edition for automatic port resolution"
    });
  }
  
  if (gameType === GameType.RUST) {
    // Rust - UDP SRV record
    domains.push({
      label: "Rust (SRV)",
      domain: `_rust._udp.gameserver-${id}.my.obiente.cloud`,
      description: "Use this domain in Rust for automatic port resolution"
    });
  }
  
  return domains;
});

const isMinecraftServer = computed(() => {
  if (!gameServer.value?.gameType) return false;
  const gameType = typeof gameServer.value.gameType === 'number'
    ? gameServer.value.gameType as GameType
    : gameServer.value.gameType;
  return (
    gameType === GameType.MINECRAFT ||
    gameType === GameType.MINECRAFT_JAVA ||
    gameType === GameType.MINECRAFT_BEDROCK
  );
});

const hasSRVRecords = computed(() => {
  return srvDomains.value.length > 0;
});

// Copy to clipboard helper
const copyToClipboard = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text);
    toast.success("Copied to clipboard");
  } catch (error) {
    console.error("Failed to copy:", error);
    toast.error("Failed to copy to clipboard");
  }
};

const isActionDisabled = (status: string) => {
  return ["STARTING", "STOPPING", "RESTARTING"].includes(status);
};

// Load game server data
const loadGameServer = async () => {
  await refreshGameServer();
  await loadUsage();
};

// Load usage data
const loadUsage = async () => {
  if (!gameServer.value) return;
  
  try {
    const month = new Date().toISOString().slice(0, 7); // YYYY-MM
    const res = await client.getGameServerUsage({
      gameServerId: gameServerId.value,
      month,
    });
    usageData.value = res;
  } catch (error) {
    console.error("Failed to load usage:", error);
    // Don't show error toast for usage - it's optional
  }
};

// Watch for game server data to load usage
watch(() => gameServer.value, (newValue) => {
  if (newValue) {
    loadUsage();
  }
}, { immediate: true });

// Actions
const refreshAll = async () => {
  isRefreshing.value = true;
  try {
    await loadGameServer();
    toast.success("Game server refreshed");
  } catch (error) {
    toast.error("Failed to refresh game server");
  } finally {
    isRefreshing.value = false;
  }
};

const toggleServerStatus = async () => {
  if (!gameServer.value) return;
  
  if (gameServer.value.status === "RUNNING") {
    await stopServer();
  } else {
    await startServer();
  }
};

const startServer = async () => {
  if (!gameServer.value) return;
  isStarting.value = true;
  try {
    await client.startGameServer({
      gameServerId: gameServerId.value,
    });
    await loadGameServer();
    toast.success("Game server started");
  } catch (error) {
    toast.error("Failed to start game server");
  } finally {
    isStarting.value = false;
  }
};

const stopServer = async () => {
  if (!gameServer.value) return;
  isStopping.value = true;
  try {
    await client.stopGameServer({
      gameServerId: gameServerId.value,
    });
    await loadGameServer();
    toast.success("Game server stopped");
  } catch (error) {
    toast.error("Failed to stop game server");
  } finally {
    isStopping.value = false;
  }
};

const restartServer = async () => {
  if (!gameServer.value) return;
  isRestarting.value = true;
  try {
    await client.restartGameServer({
      gameServerId: gameServerId.value,
    });
    await loadGameServer();
    toast.success("Game server restarted");
  } catch (error) {
    toast.error("Failed to restart game server");
  } finally {
    isRestarting.value = false;
  }
};

const deleteGameServer = async () => {
  if (!gameServer.value) return;
  isDeleting.value = true;
  try {
    await client.deleteGameServer({
      gameServerId: gameServerId.value,
    });
    toast.success("Game server deleted");
    router.push("/gameservers");
  } catch (error) {
    toast.error("Failed to delete game server");
  } finally {
    isDeleting.value = false;
    showDeleteDialog.value = false;
  }
};
</script>

