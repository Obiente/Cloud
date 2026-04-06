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
    <UiQuickInfoBar
      :icon="ServerStackIcon"
      :primary="`${connectionDomain}${gameServer.port ? ':' + gameServer.port : ''}`"
      :secondary="gameServer.gameType !== undefined ? getGameTypeLabel(gameServer.gameType) : 'Unknown'"
      mono
    >
      <OuiBadge variant="secondary" size="xs">{{ gameServer.cpuCores || '—' }} vCPU</OuiBadge>
      <OuiBadge variant="secondary" size="xs"><OuiByte :value="getMemoryBytesValue(gameServer.memoryBytes)" /></OuiBadge>
      <OuiBadge v-if="estimatedMonthlyCost > 0" variant="primary" size="xs">{{ formatCurrency(estimatedMonthlyCost) }}/mo</OuiBadge>
    </UiQuickInfoBar>

    <!-- Connection + Details -->
    <OuiGrid :cols="{ sm: 1, lg: 2 }" gap="sm">
      <!-- Connection -->
      <OuiCard v-if="gameServer.status === 'RUNNING' && gameServer.port" variant="outline" status="success">
        <OuiCardBody>
          <OuiStack gap="md">
            <UiSectionHeader :icon="GlobeAltIcon" color="success">Connection</UiSectionHeader>

            <OuiStack gap="sm">
              <!-- SRV Records -->
              <template v-if="hasSRVRecords">
                <UiCopyField
                  v-for="(srv, index) in srvDomains"
                  :key="index"
                  :label="srv.label"
                  :value="srv.domain"
                  variant="field"
                  break-all
                />
              </template>

              <!-- Direct Connect -->
              <UiCopyField
                label="Direct"
                :value="`${connectionDomain}:${gameServer.port}`"
                variant="field"
                break-all
              />
            </OuiStack>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Details -->
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="md">
            <UiSectionHeader :icon="CubeIcon" color="primary">Details</UiSectionHeader>

            <UiKeyValueGrid :items="[
              { label: 'Game', value: gameServer.gameType !== undefined ? getGameTypeLabel(gameServer.gameType) : '—' },
              { label: 'vCPU', value: String(gameServer.cpuCores || '—') },
              { label: 'Memory' },
              { label: 'Port', value: String(gameServer.port || '—'), mono: true },
              { label: 'Cost', value: `${formatCurrency(estimatedMonthlyCost)}/mo` },
              { label: 'Created' },
            ]">
              <template #value-memory>
                <OuiText size="sm" weight="medium"><OuiByte :value="getMemoryBytesValue(gameServer.memoryBytes)" /></OuiText>
              </template>
              <template #value-created>
                <OuiText v-if="gameServer.createdAt" size="sm" weight="medium">
                  <OuiRelativeTime :value="date(gameServer.createdAt)" :style="'short'" />
                </OuiText>
                <OuiText v-else size="sm" color="tertiary">—</OuiText>
              </template>
            </UiKeyValueGrid>
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

</script>
