<template>
  <OuiStack gap="xl">
    <!-- Key Metrics Grid -->
    <OuiGrid cols="1" cols-md="2" cols-lg="3" cols-xl="4" gap="md">
      <!-- Status Card -->
      <OuiCard>
        <OuiCardBody>
          <OuiStack gap="sm">
            <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
              Status
            </OuiText>
            <OuiFlex align="center" gap="sm">
              <span
                class="h-2 w-2 rounded-full"
                :class="getStatusDotClass(gameServer.status)"
              />
              <OuiText size="lg" weight="bold">
                {{ getStatusLabel(gameServer.status) }}
              </OuiText>
            </OuiFlex>
            <OuiText v-if="gameServer.updatedAt" size="xs" color="muted">
              Last updated
              <OuiRelativeTime
                :value="date(gameServer.updatedAt)"
                :style="'short'"
              />
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- vCPU Cores Card -->
      <OuiCard>
        <OuiCardBody>
          <OuiStack gap="sm">
            <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
              vCPU Cores
            </OuiText>
            <OuiFlex align="center" gap="sm">
              <CpuChipIcon class="h-5 w-5 text-secondary" />
              <OuiText size="2xl" weight="bold">
                {{ gameServer.cpuCores || "N/A" }}
              </OuiText>
            </OuiFlex>
            <OuiText size="xs" color="muted">
              Maximum CPU cores
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Memory Card -->
      <OuiCard>
        <OuiCardBody>
          <OuiStack gap="sm">
            <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
              Memory
            </OuiText>
            <OuiFlex align="center" gap="sm">
              <CircleStackIcon class="h-5 w-5 text-secondary" />
              <OuiText size="2xl" weight="bold">
                <OuiByte :value="getMemoryBytesValue(gameServer.memoryBytes)" />
              </OuiText>
            </OuiFlex>
            <OuiText size="xs" color="muted">
              Maximum memory
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Port Card -->
      <OuiCard>
        <OuiCardBody>
          <OuiStack gap="sm">
            <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
              Port
            </OuiText>
            <OuiFlex align="center" gap="sm">
              <ServerIcon class="h-5 w-5 text-secondary" />
              <OuiText size="2xl" weight="bold">
                {{ gameServer.port || "N/A" }}
              </OuiText>
            </OuiFlex>
            <OuiText size="xs" color="muted">
              Game server port
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Monthly Cost Card -->
      <OuiCard>
        <OuiCardBody>
          <OuiStack gap="sm">
            <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
              Monthly Cost
            </OuiText>
            <OuiFlex align="center" gap="sm">
              <ChartBarIcon class="h-5 w-5 text-secondary" />
              <OuiText size="2xl" weight="bold">
                {{ formatCurrency(estimatedMonthlyCost) }}
              </OuiText>
            </OuiFlex>
            <OuiText size="xs" color="muted">
              Based on current usage
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Game Type Card -->
      <OuiCard v-if="gameServer.gameType !== undefined">
        <OuiCardBody>
          <OuiStack gap="sm">
            <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
              Game Type
            </OuiText>
            <OuiFlex align="center" gap="sm">
              <CubeIcon class="h-5 w-5 text-secondary" />
              <OuiText size="lg" weight="bold">
                {{ getGameTypeLabel(gameServer.gameType) }}
              </OuiText>
            </OuiFlex>
            <OuiText size="xs" color="muted">
              Server game type
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Created Card -->
      <OuiCard>
        <OuiCardBody>
          <OuiStack gap="sm">
            <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
              Created
            </OuiText>
            <template v-if="gameServer.createdAt">
              <OuiFlex align="center" gap="sm">
                <CalendarIcon class="h-5 w-5 text-secondary" />
                <OuiText size="lg" weight="bold">
                  <OuiRelativeTime
                    :value="date(gameServer.createdAt)"
                    :style="'short'"
                  />
                </OuiText>
              </OuiFlex>
              <OuiText size="xs" color="muted">
                {{ formatDate(gameServer.createdAt) }}
              </OuiText>
            </template>
            <template v-else>
              <OuiFlex align="center" gap="sm">
                <CalendarIcon class="h-5 w-5 text-secondary" />
                <OuiText size="lg" weight="bold">Unknown</OuiText>
              </OuiFlex>
              <OuiText size="xs" color="muted">
                Creation date unavailable
              </OuiText>
            </template>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiGrid>

    <!-- Usage Statistics Section -->
    <UsageStatistics v-if="usageData" :usage-data="usageData" />

    <!-- Cost Breakdown -->
    <CostBreakdown v-if="usageData" :usage-data="usageData" />

    <!-- Live Metrics -->
    <LiveMetrics
      :is-streaming="isStreaming"
      :latest-metric="latestMetric"
      :current-cpu-usage="currentCpuUsage"
      :current-memory-usage="currentMemoryUsage"
      :current-network-rx="currentNetworkRx"
      :current-network-tx="currentNetworkTx"
    />

    <!-- Main Information Grid -->
    <OuiGrid cols="1" cols-lg="2" gap="lg">
      <!-- Game Server Details Card -->
      <OuiCard>
        <OuiCardHeader>
          <OuiText size="lg" weight="bold">Game Server Details</OuiText>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiStack gap="md">
            <!-- Game Type -->
            <div class="flex items-start justify-between gap-4 py-2 border-b border-border-default">
              <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                <CubeIcon class="h-4 w-4 text-secondary shrink-0" />
                <OuiStack gap="xs" class="min-w-0 flex-1">
                  <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                    Game Type
                  </OuiText>
                  <OuiText size="sm" weight="medium">
                    {{ gameServer.gameType !== undefined ? getGameTypeLabel(gameServer.gameType) : "Not set" }}
                  </OuiText>
                </OuiStack>
              </OuiFlex>
            </div>

            <!-- Status -->
            <div class="flex items-start justify-between gap-4 py-2 border-b border-border-default">
              <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                <span
                  class="h-2 w-2 rounded-full shrink-0"
                  :class="getStatusDotClass(gameServer.status)"
                />
                <OuiStack gap="xs" class="min-w-0 flex-1">
                  <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                    Status
                  </OuiText>
                  <OuiBadge :variant="getStatusBadgeVariant(gameServer.status)" size="sm">
                    {{ getStatusLabel(gameServer.status) }}
                  </OuiBadge>
                </OuiStack>
              </OuiFlex>
            </div>

            <!-- vCPU Cores -->
            <div class="flex items-start justify-between gap-4 py-2 border-b border-border-default">
              <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                <CpuChipIcon class="h-4 w-4 text-secondary shrink-0" />
                <OuiStack gap="xs" class="min-w-0 flex-1">
                  <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                    vCPU Cores
                  </OuiText>
                  <OuiText size="sm" weight="medium">
                    {{ gameServer.cpuCores || "N/A" }}
                  </OuiText>
                </OuiStack>
              </OuiFlex>
            </div>

            <!-- Memory -->
            <div class="flex items-start justify-between gap-4 py-2 border-b border-border-default">
              <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                <CircleStackIcon class="h-4 w-4 text-secondary shrink-0" />
                <OuiStack gap="xs" class="min-w-0 flex-1">
                  <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                    Memory
                  </OuiText>
                  <OuiText size="sm" weight="medium">
                    <OuiByte :value="getMemoryBytesValue(gameServer.memoryBytes)" />
                  </OuiText>
                </OuiStack>
              </OuiFlex>
            </div>

            <!-- Port -->
            <div
              v-if="gameServer.port"
              class="flex items-start justify-between gap-4 py-2 border-b border-border-default"
            >
              <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                <ServerIcon class="h-4 w-4 text-secondary shrink-0" />
                <OuiStack gap="xs" class="min-w-0 flex-1">
                  <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                    Port
                  </OuiText>
                  <OuiText size="sm" weight="medium">
                    {{ gameServer.port }}
                  </OuiText>
                </OuiStack>
              </OuiFlex>
            </div>

            <!-- Created -->
            <div class="flex items-start justify-between gap-4 py-2">
              <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                <CalendarIcon class="h-4 w-4 text-secondary shrink-0" />
                <OuiStack gap="xs" class="min-w-0 flex-1">
                  <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                    Created
                  </OuiText>
                  <OuiText size="sm" weight="medium">
                    <template v-if="gameServer.createdAt">
                      <OuiRelativeTime
                        :value="date(gameServer.createdAt)"
                        :style="'short'"
                      />
                    </template>
                    <template v-else>Unknown</template>
                  </OuiText>
                </OuiStack>
              </OuiFlex>
            </div>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Connection Information Card -->
      <OuiCard v-if="gameServer.status === 'RUNNING' && gameServer.port">
        <OuiCardHeader>
          <OuiText size="lg" weight="bold">Connection Information</OuiText>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiStack gap="lg">
            <OuiText size="sm" color="muted">
              Use this information to connect to your game server
            </OuiText>

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
    </OuiGrid>
  </OuiStack>
</template>

<script setup lang="ts">
import { computed } from "vue";
import {
  CpuChipIcon,
  CircleStackIcon,
  ServerIcon,
  ChartBarIcon,
  CubeIcon,
  CalendarIcon,
  ClipboardIcon,
  GlobeAltIcon,
} from "@heroicons/vue/24/outline";
import { GameType } from "@obiente/proto";
import { date } from "@obiente/proto/utils";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import OuiByte from "~/components/oui/Byte.vue";
import UsageStatistics from "~/components/shared/UsageStatistics.vue";
import CostBreakdown from "~/components/shared/CostBreakdown.vue";
import LiveMetrics from "~/components/shared/LiveMetrics.vue";
import { useToast } from "~/composables/useToast";

interface Props {
  gameServer: any;
  usageData: any;
  isStreaming: boolean;
  latestMetric: any;
  currentCpuUsage: number;
  currentMemoryUsage: number;
  currentNetworkRx: number;
  currentNetworkTx: number;
}

const props = defineProps<Props>();
const { toast } = useToast();

// Computed properties
const estimatedMonthlyCost = computed(() => {
  if (props.usageData?.estimatedMonthly?.estimatedCostCents) {
    return Number(props.usageData.estimatedMonthly.estimatedCostCents) / 100;
  }
  return 0;
});

const connectionDomain = computed(() => {
  if (!props.gameServer?.id) return "";
  return `${props.gameServer.id}.my.obiente.cloud`;
});

const srvDomains = computed(() => {
  if (!props.gameServer?.id || props.gameServer?.gameType === undefined) return [];
  
  const gameType = typeof props.gameServer.gameType === 'number'
    ? props.gameServer.gameType as GameType
    : props.gameServer.gameType;
  const id = props.gameServer.id;
  const domains: Array<{ label: string; domain: string; description: string }> = [];
  
  if (gameType === GameType.MINECRAFT || gameType === GameType.MINECRAFT_JAVA) {
    domains.push({
      label: "Minecraft Java (SRV)",
      domain: `_minecraft._tcp.${id}.my.obiente.cloud`,
      description: "Use this domain in Minecraft Java Edition for automatic port resolution"
    });
  }
  
  if (gameType === GameType.MINECRAFT || gameType === GameType.MINECRAFT_BEDROCK) {
    domains.push({
      label: "Minecraft Bedrock (SRV)",
      domain: `_minecraft._udp.${id}.my.obiente.cloud`,
      description: "Use this domain in Minecraft Bedrock Edition for automatic port resolution"
    });
  }
  
  if (gameType === GameType.RUST) {
    domains.push({
      label: "Rust (SRV)",
      domain: `_rust._udp.${id}.my.obiente.cloud`,
      description: "Use this domain in Rust for automatic port resolution"
    });
  }
  
  return domains;
});

const hasSRVRecords = computed(() => {
  return srvDomains.value.length > 0;
});

// Helper functions
const getMemoryBytesValue = (value: bigint | number | undefined | null): number => {
  if (!value) return 0;
  if (typeof value === 'bigint') return Number(value);
  return value;
};

const formatCurrency = (dollars: number) => {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(dollars);
};

const formatDate = (timestamp: any) => {
  if (!timestamp) return "";
  const d = date(timestamp);
  if (!d) return "";
  return d.toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
};

const getGameTypeLabel = (gameType: number) => {
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

const getStatusLabel = (status: string) => {
  const statusMap: Record<string, string> = {
    RUNNING: "Running",
    STOPPED: "Stopped",
    STARTING: "Starting",
    STOPPING: "Stopping",
    RESTARTING: "Restarting",
    FAILED: "Failed",
    CREATED: "Created",
  };
  return statusMap[status] || "Unknown";
};

const getStatusDotClass = (status: string) => {
  const statusMap: Record<string, string> = {
    RUNNING: "bg-success animate-pulse",
    STOPPED: "bg-muted",
    STARTING: "bg-warning animate-pulse",
    STOPPING: "bg-warning animate-pulse",
    RESTARTING: "bg-warning animate-pulse",
    FAILED: "bg-danger",
    CREATED: "bg-secondary",
  };
  return statusMap[status] || "bg-secondary";
};

const getStatusBadgeVariant = (status: string): "success" | "warning" | "danger" | "secondary" => {
  const statusMap: Record<string, "success" | "warning" | "danger" | "secondary"> = {
    RUNNING: "success",
    STOPPED: "secondary",
    STARTING: "warning",
    STOPPING: "warning",
    RESTARTING: "warning",
    FAILED: "danger",
    CREATED: "secondary",
  };
  return statusMap[status] || "secondary";
};

const copyToClipboard = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text);
    toast.success("Copied to clipboard");
  } catch (error) {
    console.error("Failed to copy:", error);
    toast.error("Failed to copy to clipboard");
  }
};
</script>

