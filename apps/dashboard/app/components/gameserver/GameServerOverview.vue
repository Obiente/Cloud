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

    <!-- Quick Info Bar -->
    <OuiCard variant="outline">
      <OuiCardBody>
        <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
          <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
            <div class="h-8 w-8 rounded-lg bg-surface-muted flex items-center justify-center shrink-0">
              <ServerStackIcon class="h-4 w-4 text-accent-primary" />
            </div>
            <OuiStack gap="none" class="min-w-0">
              <OuiText size="sm" weight="medium" class="font-mono" truncate>
                {{ connectionDomain }}{{ gameServer.port ? ':' + gameServer.port : '' }}
              </OuiText>
              <OuiText size="xs" color="tertiary">{{ gameServer.gameType !== undefined ? getGameTypeLabel(gameServer.gameType) : 'Unknown' }}</OuiText>
            </OuiStack>
          </OuiFlex>
          <OuiFlex gap="xs" align="center" class="shrink-0">
            <OuiBadge variant="secondary" size="xs">{{ gameServer.cpuCores || '—' }} vCPU</OuiBadge>
            <OuiBadge variant="secondary" size="xs"><OuiByte :value="getMemoryBytesValue(gameServer.memoryBytes)" /></OuiBadge>
            <OuiBadge v-if="estimatedMonthlyCost > 0" variant="primary" size="xs">{{ formatCurrency(estimatedMonthlyCost) }}/mo</OuiBadge>
          </OuiFlex>
        </OuiFlex>
      </OuiCardBody>
    </OuiCard>

    <!-- Connection + Details -->
    <OuiGrid :cols="{ sm: 1, lg: 2 }" gap="sm">
      <!-- Connection -->
      <OuiCard v-if="gameServer.status === 'RUNNING' && gameServer.port" variant="outline" status="success">
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiFlex align="center" gap="xs">
              <GlobeAltIcon class="h-3.5 w-3.5 text-success" />
              <OuiText size="sm" weight="semibold">Connection</OuiText>
            </OuiFlex>

            <OuiStack gap="sm">
              <!-- SRV Records -->
              <template v-if="hasSRVRecords">
                <div
                  v-for="(srv, index) in srvDomains"
                  :key="index"
                  class="group rounded-lg border border-border-default px-3 py-2.5"
                >
                  <OuiFlex align="center" justify="between" gap="sm">
                    <OuiStack gap="xs" class="min-w-0">
                      <OuiText size="xs" color="tertiary">{{ srv.label }}</OuiText>
                      <OuiText size="sm" weight="medium" class="font-mono break-all">{{ srv.domain }}</OuiText>
                    </OuiStack>
                    <button
                      class="p-1 rounded text-tertiary hover:text-primary opacity-0 group-hover:opacity-100 transition-opacity shrink-0"
                      @click="copyToClipboard(srv.domain)"
                    >
                      <ClipboardIcon class="h-3.5 w-3.5" />
                    </button>
                  </OuiFlex>
                </div>
              </template>

              <!-- Direct Connect -->
              <div class="group rounded-lg border border-border-default px-3 py-2.5">
                <OuiFlex align="center" justify="between" gap="sm">
                  <OuiStack gap="xs" class="min-w-0">
                    <OuiText size="xs" color="tertiary">Direct</OuiText>
                    <OuiText size="sm" weight="medium" class="font-mono break-all">{{ connectionDomain }}:{{ gameServer.port }}</OuiText>
                  </OuiStack>
                  <button
                    class="p-1 rounded text-tertiary hover:text-primary opacity-0 group-hover:opacity-100 transition-opacity shrink-0"
                    @click="copyToClipboard(connectionDomain + ':' + gameServer.port)"
                  >
                    <ClipboardIcon class="h-3.5 w-3.5" />
                  </button>
                </OuiFlex>
              </div>
            </OuiStack>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Details -->
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiFlex align="center" gap="xs">
              <CubeIcon class="h-3.5 w-3.5 text-accent-primary" />
              <OuiText size="sm" weight="semibold">Details</OuiText>
            </OuiFlex>

            <div class="grid grid-cols-2 gap-3">
              <OuiStack gap="xs">
                <OuiText size="xs" color="tertiary">Game</OuiText>
                <OuiText size="sm" weight="medium">{{ gameServer.gameType !== undefined ? getGameTypeLabel(gameServer.gameType) : '—' }}</OuiText>
              </OuiStack>
              <OuiStack gap="xs">
                <OuiText size="xs" color="tertiary">vCPU</OuiText>
                <OuiText size="sm" weight="medium">{{ gameServer.cpuCores || '—' }}</OuiText>
              </OuiStack>
              <OuiStack gap="xs">
                <OuiText size="xs" color="tertiary">Memory</OuiText>
                <OuiText size="sm" weight="medium"><OuiByte :value="getMemoryBytesValue(gameServer.memoryBytes)" /></OuiText>
              </OuiStack>
              <OuiStack gap="xs">
                <OuiText size="xs" color="tertiary">Port</OuiText>
                <OuiText size="sm" weight="medium" class="font-mono">{{ gameServer.port || '—' }}</OuiText>
              </OuiStack>
              <OuiStack gap="xs">
                <OuiText size="xs" color="tertiary">Cost</OuiText>
                <OuiText size="sm" weight="medium">{{ formatCurrency(estimatedMonthlyCost) }}/mo</OuiText>
              </OuiStack>
              <OuiStack gap="xs">
                <OuiText size="xs" color="tertiary">Created</OuiText>
                <OuiText v-if="gameServer.createdAt" size="sm" weight="medium">
                  <OuiRelativeTime :value="date(gameServer.createdAt)" :style="'short'" />
                </OuiText>
                <OuiText v-else size="sm" color="tertiary">—</OuiText>
              </OuiStack>
            </div>
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
import { computed } from "vue";
import {
  ClipboardIcon,
  ServerStackIcon,
  GlobeAltIcon,
  CubeIcon,
} from "@heroicons/vue/24/outline";
import { GameType } from "@obiente/proto";
import { date } from "@obiente/proto/utils";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import OuiByte from "~/components/oui/Byte.vue";
import UsageStatistics from "~/components/shared/UsageStatistics.vue";
import CostBreakdown from "~/components/shared/CostBreakdown.vue";
import LiveMetrics from "~/components/shared/LiveMetrics.vue";
import { useToast } from "~/composables/useToast";
import {
  getGameServerConnectionDomain,
  getGameServerSrvDomains,
} from "~/utils/domains";

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
  return getGameServerConnectionDomain(props.gameServer?.id);
});

const srvDomains = computed(() => {
  return getGameServerSrvDomains(
    props.gameServer?.id,
    props.gameServer?.gameType
  );
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
