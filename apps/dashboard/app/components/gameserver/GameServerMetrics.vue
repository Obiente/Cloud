<template>
  <OuiStack gap="xl">
    <!-- Usage Summary Cards -->
    <OuiGrid cols="1" cols-md="2" cols-lg="5" gap="lg">
      <OuiCard>
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiStack gap="xs">
              <OuiText size="sm" color="muted">Current Month Usage</OuiText>
              <OuiText size="2xl" weight="bold">
                {{ formatCurrency(usageData?.current?.estimatedCostCents || 0) }}
              </OuiText>
            </OuiStack>
            <OuiText size="xs" color="muted">
              Est:
              {{ formatCurrency(usageData?.estimatedMonthly?.estimatedCostCents || 0) }}
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <OuiCard>
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiStack gap="xs">
              <OuiText size="sm" color="muted">CPU Hours</OuiText>
              <OuiText size="2xl" weight="bold">
                {{ formatCoreSecondsToHours(usageData?.current?.cpuCoreSeconds ?? 0) }}
              </OuiText>
            </OuiStack>
            <OuiText size="xs" color="muted">
              Est:
              {{ formatCoreSecondsToHours(usageData?.estimatedMonthly?.cpuCoreSeconds ?? 0) }}
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <OuiCard>
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiStack gap="xs">
              <OuiText size="sm" color="muted">Memory (GB avg)</OuiText>
              <OuiText size="2xl" weight="bold">
                {{ formatMemoryByteSecondsToGB(usageData?.current?.memoryByteSeconds ?? 0) }}
              </OuiText>
            </OuiStack>
            <OuiText size="xs" color="muted">
              Est:
              {{ formatMemoryByteSecondsToGB(usageData?.estimatedMonthly?.memoryByteSeconds ?? 0) }}
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <OuiCard>
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiStack gap="xs">
              <OuiText size="sm" color="muted">Network (GB)</OuiText>
              <OuiText size="2xl" weight="bold">
                {{ formatBandwidthToGB(usageData?.current?.bandwidthRxBytes ?? 0, usageData?.current?.bandwidthTxBytes ?? 0) }}
              </OuiText>
            </OuiStack>
            <OuiText size="xs" color="muted">
              Rx: {{ formatBytesToGB(usageData?.current?.bandwidthRxBytes ?? 0) }} GB
              Tx: {{ formatBytesToGB(usageData?.current?.bandwidthTxBytes ?? 0) }} GB
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <OuiCard>
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiStack gap="xs">
              <OuiText size="sm" color="muted">Storage Usage</OuiText>
              <OuiFlex align="center" gap="sm">
                <CubeIcon class="h-5 w-5 text-secondary" />
                <OuiText size="2xl" weight="bold">
                  <OuiByte :value="getStorageBytesValue(usageData?.current?.storageBytes)" unit-display="short" />
                </OuiText>
              </OuiFlex>
            </OuiStack>
            <OuiText size="xs" color="muted">
              Image + Volumes + Disk
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiGrid>

    <!-- Cost Breakdown -->
    <OuiCard>
      <OuiCardBody>
        <OuiStack gap="lg">
          <OuiText size="xl" weight="bold">Cost Breakdown</OuiText>
          
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <!-- Current Cost Breakdown -->
            <OuiCard variant="outline">
              <OuiCardBody>
                <OuiStack gap="md">
                  <OuiText size="md" weight="semibold">Current Month Usage</OuiText>
                  
                  <OuiStack gap="sm">
                    <div class="flex justify-between items-center py-2 border-b border-border-default">
                      <OuiText size="sm" color="muted">CPU</OuiText>
                      <OuiText size="sm" weight="semibold">
                        {{ formatCurrency(costBreakdown.current.cpu) }}
                      </OuiText>
                    </div>
                    <div class="flex justify-between items-center py-2 border-b border-border-default">
                      <OuiText size="sm" color="muted">Memory</OuiText>
                      <OuiText size="sm" weight="semibold">
                        {{ formatCurrency(costBreakdown.current.memory) }}
                      </OuiText>
                    </div>
                    <div class="flex justify-between items-center py-2 border-b border-border-default">
                      <OuiText size="sm" color="muted">Bandwidth</OuiText>
                      <OuiText size="sm" weight="semibold">
                        {{ formatCurrency(costBreakdown.current.bandwidth) }}
                      </OuiText>
                    </div>
                    <div class="flex justify-between items-center py-2 border-b border-border-default">
                      <OuiText size="sm" color="muted">Storage</OuiText>
                      <OuiText size="sm" weight="semibold">
                        {{ formatCurrency(costBreakdown.current.storage) }}
                      </OuiText>
                    </div>
                    <div class="flex justify-between items-center pt-2 border-t-2 border-border-default">
                      <OuiText size="sm" weight="semibold">Total</OuiText>
                      <OuiText size="lg" weight="bold">
                        {{ formatCurrency(costBreakdown.current.total) }}
                      </OuiText>
                    </div>
                  </OuiStack>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>

            <!-- Estimated Cost Breakdown -->
            <OuiCard variant="outline">
              <OuiCardBody>
                <OuiStack gap="md">
                  <OuiText size="md" weight="semibold">Estimated Monthly</OuiText>
                  
                  <OuiStack gap="sm">
                    <div class="flex justify-between items-center py-2 border-b border-border-default">
                      <OuiText size="sm" color="muted">CPU</OuiText>
                      <OuiText size="sm" weight="semibold">
                        {{ formatCurrency(costBreakdown.estimated.cpu) }}
                      </OuiText>
                    </div>
                    <div class="flex justify-between items-center py-2 border-b border-border-default">
                      <OuiText size="sm" color="muted">Memory</OuiText>
                      <OuiText size="sm" weight="semibold">
                        {{ formatCurrency(costBreakdown.estimated.memory) }}
                      </OuiText>
                    </div>
                    <div class="flex justify-between items-center py-2 border-b border-border-default">
                      <OuiText size="sm" color="muted">Bandwidth</OuiText>
                      <OuiText size="sm" weight="semibold">
                        {{ formatCurrency(costBreakdown.estimated.bandwidth) }}
                      </OuiText>
                    </div>
                    <div class="flex justify-between items-center py-2 border-b border-border-default">
                      <OuiText size="sm" color="muted">Storage</OuiText>
                      <OuiText size="sm" weight="semibold">
                        {{ formatCurrency(costBreakdown.estimated.storage) }}
                      </OuiText>
                    </div>
                    <div class="flex justify-between items-center pt-2 border-t-2 border-border-default">
                      <OuiText size="sm" weight="semibold">Total</OuiText>
                      <OuiText size="lg" weight="bold">
                        {{ formatCurrency(costBreakdown.estimated.total) }}
                      </OuiText>
                    </div>
                  </OuiStack>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>
          </div>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- Real-time Metrics Charts -->
    <OuiStack gap="lg">
      <OuiFlex justify="between" align="center" wrap="wrap" gap="md">
        <OuiText size="xl" weight="bold">Real-time Metrics</OuiText>
        <OuiFlex gap="sm" align="center" wrap="wrap">
          <!-- Timeframe Selector -->
          <OuiFlex gap="xs" align="center">
            <OuiText size="sm" color="muted">Timeframe:</OuiText>
            <OuiSelect
              v-model="selectedTimeframe"
              :items="timeframeOptions"
              placeholder="Select timeframe"
              style="min-width: 160px"
            />
          </OuiFlex>

          <!-- Custom Date Range Picker -->
          <OuiFlex
            v-if="selectedTimeframe === 'custom'"
            gap="xs"
            align="center"
            class="ml-2"
          >
            <OuiDatePicker
              v-model="customDateRange"
              selection-mode="range"
              start-placeholder="Start date"
              end-placeholder="End date"
            />
          </OuiFlex>

          <OuiBadge :variant="streaming ? 'success' : 'secondary'">
            <span
              class="inline-flex h-1.5 w-1.5 rounded-full mr-1"
              :class="streaming ? 'bg-success animate-pulse' : 'bg-secondary'"
            />
            {{ streaming ? "Live" : "Paused" }}
          </OuiBadge>
          <OuiButton variant="ghost" size="sm" @click="toggleStreaming">
            {{ streaming ? "Pause" : "Resume" }}
          </OuiButton>
        </OuiFlex>
      </OuiFlex>

      <OuiGrid cols="1" cols-lg="2" gap="lg">
        <!-- CPU Usage Chart -->
        <OuiCard>
          <OuiCardBody>
            <OuiStack gap="md">
              <OuiText size="lg" weight="semibold">CPU Usage</OuiText>
              <div class="relative" style="width: 100%; height: 300px">
                <div ref="cpuChartRef" style="width: 100%; height: 100%"></div>
                <div
                  v-if="!hasMetricsData"
                  class="absolute inset-0 flex items-center justify-center bg-surface-subtle/50 rounded"
                >
                  <OuiText size="sm" color="muted">{{ emptyStateMessage }}</OuiText>
                </div>
              </div>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Memory Usage Chart -->
        <OuiCard>
          <OuiCardBody>
            <OuiStack gap="md">
              <OuiText size="lg" weight="semibold">Memory Usage</OuiText>
              <div class="relative" style="width: 100%; height: 300px">
                <div ref="memoryChartRef" style="width: 100%; height: 100%"></div>
                <div
                  v-if="!hasMetricsData"
                  class="absolute inset-0 flex items-center justify-center bg-surface-subtle/50 rounded"
                >
                  <OuiText size="sm" color="muted">{{ emptyStateMessage }}</OuiText>
                </div>
              </div>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Network I/O Chart -->
        <OuiCard>
          <OuiCardBody>
            <OuiStack gap="md">
              <OuiText size="lg" weight="semibold">Network I/O</OuiText>
              <div class="relative" style="width: 100%; height: 300px">
                <div ref="networkChartRef" style="width: 100%; height: 100%"></div>
                <div
                  v-if="!hasMetricsData"
                  class="absolute inset-0 flex items-center justify-center bg-surface-subtle/50 rounded"
                >
                  <OuiText size="sm" color="muted">{{ emptyStateMessage }}</OuiText>
                </div>
              </div>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Disk I/O Chart -->
        <OuiCard>
          <OuiCardBody>
            <OuiStack gap="md">
              <OuiText size="lg" weight="semibold">Disk I/O</OuiText>
              <div class="relative" style="width: 100%; height: 300px">
                <div ref="diskChartRef" style="width: 100%; height: 100%"></div>
                <div
                  v-if="!hasMetricsData"
                  class="absolute inset-0 flex items-center justify-center bg-surface-subtle/50 rounded"
                >
                  <OuiText size="sm" color="muted">{{ emptyStateMessage }}</OuiText>
                </div>
              </div>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>
      </OuiGrid>
    </OuiStack>
  </OuiStack>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick } from "vue";
import { useConnectClient } from "~/lib/connect-client";
import { GameServerService } from "@obiente/proto";
import {
  registerOUIEChartsTheme,
  getOUIEChartsColors,
  createAreaGradient,
} from "~/utils/echarts-theme";
import { CubeIcon } from "@heroicons/vue/24/outline";
import OuiByte from "~/components/oui/Byte.vue";
import { usePreferencesStore } from "~/stores/preferences";
import type { ECharts } from "echarts/core";

// ECharts will be loaded dynamically to reduce initial bundle size

interface Props {
  gameServerId: string;
  organizationId: string;
  gameServerStatus?: number; // GameServerStatus enum value
}

const props = defineProps<Props>();
const client = useConnectClient(GameServerService);
const preferencesStore = usePreferencesStore();

// Timeframe selection - sync with preferences store (following DeploymentMetrics pattern)
type TimeframeOption = "10m" | "1h" | "24h" | "7d" | "30d" | "custom";
const selectedTimeframe = computed({
  get: () => {
    const value = preferencesStore.metricsPreferences?.timeframe;
    // Return default value if undefined to ensure select always has a valid value
    return value ?? "24h";
  },
  set: (value: TimeframeOption) => {
    preferencesStore.setMetricsPreference("timeframe", value);
    if (value !== "custom") {
      customDateRange.value = [];
      loadHistoricalMetrics();
    } else {
      loadCustomDateRangeFromPreferences();
    }
  },
});

const customDateRange = ref<any[]>([]);

// Watch metricsPreferences to track changes
watch(
  () => preferencesStore.metricsPreferences,
  async (newMetrics, oldMetrics) => {
    // When store hydrates and timeframe is restored from storage, ensure proper initialization
    // Only handle hydration case (when oldMetrics is undefined/null) to avoid duplicate calls
    // when user changes timeframe (which is handled by the setter)
    if (preferencesStore.hydrated && newMetrics?.timeframe && !oldMetrics?.timeframe) {
      const timeframe = newMetrics.timeframe;
      // If restored timeframe is custom, load the custom date range
      if (timeframe === "custom") {
        await loadCustomDateRangeFromPreferences();
        loadHistoricalMetrics();
      }
    }
  },
  { deep: true, immediate: true }
);

// Watch customDateRange for changes and save to preferences
watch(
  customDateRange,
  async (newRange) => {
    if (!import.meta.client) return;
    if (!newRange || newRange.length !== 2) return;

    await nextTick();

    const start = dateValueToDate(newRange[0]);
    const end = dateValueToDate(newRange[1]);
    if (!start || !end || start >= end) return;

    preferencesStore.setMetricsPreference("customDateRange", {
      start: start.toISOString(),
      end: end.toISOString(),
    });

    loadHistoricalMetrics();
  },
  { deep: true }
);

const timeframeOptions = [
  { value: "10m", label: "Last 10 Minutes" },
  { value: "1h", label: "Last Hour" },
  { value: "24h", label: "Last 24 Hours" },
  { value: "7d", label: "Last 7 Days" },
  { value: "30d", label: "Last 30 Days" },
  { value: "custom", label: "Custom Range" },
];

// Track if we have metrics data
const hasMetricsData = computed(() => {
  return metricsData.value.timestamps.length > 0;
});

// Empty state message
const emptyStateMessage = computed(() => {
  // GameServerStatus.RUNNING = 3
  if (props.gameServerStatus === 3) {
    return "Metrics will appear here once collected. Please wait a moment...";
  }
  return "Start the game server to begin collecting metrics.";
});

// Chart refs
const cpuChartRef = ref<HTMLElement | null>(null);
const memoryChartRef = ref<HTMLElement | null>(null);
const networkChartRef = ref<HTMLElement | null>(null);
const diskChartRef = ref<HTMLElement | null>(null);

// Chart instances
let cpuChart: ECharts | null = null;
let memoryChart: ECharts | null = null;
let networkChart: ECharts | null = null;
let diskChart: ECharts | null = null;

// Streaming state
const streaming = ref(false);
let streamController: AbortController | null = null;
let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
const MAX_RECONNECT_ATTEMPTS = 5;
const RECONNECT_DELAY = 2000;
let reconnectAttempts = 0;

// Metrics data
const metricsData = ref<{
  timestamps: string[];
  cpu: number[];
  memory: number[];
  networkRx: number[];
  networkTx: number[];
  diskRead: number[];
  diskWrite: number[];
}>({
  timestamps: [],
  cpu: [],
  memory: [],
  networkRx: [],
  networkTx: [],
  diskRead: [],
  diskWrite: [],
});

// Usage data
const usageData = ref<any>(null);

// Helper to convert bigint to number safely
const toNumber = (value: bigint | number | undefined | null): number => {
  if (value === undefined || value === null) return 0;
  if (typeof value === "bigint") return Number(value);
  return Number(value) || 0;
};

// Helper function to convert BigInt to number for storageBytes
const getStorageBytesValue = (value: bigint | number | undefined | null): number => {
  if (!value) return 0;
  if (typeof value === 'bigint') return Number(value);
  return value;
};

// Format helpers
const formatCurrency = (cents: number | bigint) => {
  const dollars = Number(cents) / 100;
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(dollars);
};

const formatCoreSecondsToHours = (seconds: number | bigint) => {
  const s = Number(seconds);
  if (!Number.isFinite(s) || s === 0) return "0.00";
  return (s / 3600).toFixed(2);
};

const formatMemoryByteSecondsToGB = (byteSeconds: number | bigint) => {
  const bs = Number(byteSeconds);
  if (bs === 0 || !Number.isFinite(bs)) return "0.00";
  const now = new Date();
  const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);
  const secondsInMonth = Math.max(
    1,
    Math.floor((now.getTime() - monthStart.getTime()) / 1000)
  );
  const avgBytes = bs / secondsInMonth;
  return formatBytesToGB(avgBytes);
};

const formatBytesToGB = (bytes: number | bigint) => {
  const b = Number(bytes);
  if (b === 0) return "0.00";
  return (b / (1024 * 1024 * 1024)).toFixed(2);
};

const formatBandwidthToGB = (
  rxBytes: number | bigint,
  txBytes: number | bigint
) => {
  const total = Number(rxBytes) + Number(txBytes);
  return formatBytesToGB(total);
};

const formatBytes = (bytes: number) => {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
};

// Cost breakdown per resource - uses backend calculated values
const costBreakdown = computed(() => {
  const current = usageData.value?.current || {};
  const estimated = usageData.value?.estimatedMonthly || {};
  
  // All costs are calculated on the backend and returned in cents
  return {
    current: {
      cpu: current.cpuCostCents != null ? toNumber(current.cpuCostCents) : 0,
      memory: current.memoryCostCents != null ? toNumber(current.memoryCostCents) : 0,
      bandwidth: current.bandwidthCostCents != null ? toNumber(current.bandwidthCostCents) : 0,
      storage: current.storageCostCents != null ? toNumber(current.storageCostCents) : 0,
      total: current.estimatedCostCents != null ? toNumber(current.estimatedCostCents) : 0,
    },
    estimated: {
      cpu: estimated.cpuCostCents != null ? toNumber(estimated.cpuCostCents) : 0,
      memory: estimated.memoryCostCents != null ? toNumber(estimated.memoryCostCents) : 0,
      bandwidth: estimated.bandwidthCostCents != null ? toNumber(estimated.bandwidthCostCents) : 0,
      storage: estimated.storageCostCents != null ? toNumber(estimated.storageCostCents) : 0,
      total: estimated.estimatedCostCents != null ? toNumber(estimated.estimatedCostCents) : 0,
    },
  };
});

// Initialize charts
const initCharts = async () => {
  if (!cpuChartRef.value || !memoryChartRef.value || !networkChartRef.value || !diskChartRef.value) {
    return;
  }

  // Dynamically load ECharts to reduce initial bundle size
  const [
    echartsModule,
    { LineChart },
    {
      TitleComponent,
      TooltipComponent,
      GridComponent,
      LegendComponent,
      DataZoomComponent,
    },
    { CanvasRenderer },
  ] = await Promise.all([
    import("echarts/core"),
    import("echarts/charts"),
    import("echarts/components"),
    import("echarts/renderers"),
  ]);
  
  // ECharts uses namespace exports, so the module itself is the echarts object
  const echarts = echartsModule as typeof import("echarts/core");

  // Register ECharts components
  echarts.use([
    LineChart,
    TitleComponent,
    TooltipComponent,
    GridComponent,
    LegendComponent,
    DataZoomComponent,
    CanvasRenderer,
  ]);

  await registerOUIEChartsTheme(echarts);
  const colors = getOUIEChartsColors();

  // CPU Chart
  if (!cpuChart) {
    cpuChart = echarts.init(cpuChartRef.value, "oui");
  }
  if (!cpuChart) return;
  cpuChart.setOption({
    title: { text: "CPU Usage (%)", left: "center" },
    tooltip: {
      trigger: "axis",
      axisPointer: { type: "cross" },
      formatter: (params: any) => {
        if (Array.isArray(params)) {
          return params.map((p: any) => `${p.seriesName}: ${p.value.toFixed(2)}%`).join("<br>");
        }
        return `${params.seriesName}: ${params.value.toFixed(2)}%`;
      },
    },
    grid: { left: "3%", right: "4%", bottom: "10%", top: "15%" },
    xAxis: { type: "category", boundaryGap: false, data: metricsData.value.timestamps },
    yAxis: { type: "value", name: "%", min: 0 },
    dataZoom: [{ type: "inside", start: 70, end: 100 }, { type: "slider", start: 70, end: 100 }],
    series: [{
      name: "CPU Usage",
      type: "line",
      smooth: true,
      data: metricsData.value.cpu,
      areaStyle: { color: createAreaGradient(colors.primary) },
      lineStyle: { color: colors.primary, width: 2 },
      itemStyle: { color: colors.primary },
    }],
  });

  // Memory Chart
  if (!memoryChart) {
    memoryChart = echarts.init(memoryChartRef.value, "oui");
  }
  if (!memoryChart) return;
  memoryChart.setOption({
    title: { text: "Memory Usage", left: "center" },
    tooltip: {
      trigger: "axis",
      axisPointer: { type: "cross" },
      formatter: (params: any) => {
        const value = Array.isArray(params) ? params[0] : params;
        return `${value.axisValue}<br/>${value.seriesName}: ${formatBytes(Number(value.value))}`;
      },
    },
    grid: { left: "3%", right: "4%", bottom: "10%", top: "15%" },
    xAxis: { type: "category", boundaryGap: false, data: metricsData.value.timestamps },
    yAxis: {
      type: "value",
      name: "Bytes",
      axisLabel: { formatter: (value: number) => formatBytes(value) },
    },
    dataZoom: [{ type: "inside", start: 70, end: 100 }, { type: "slider", start: 70, end: 100 }],
    series: [{
      name: "Memory Usage",
      type: "line",
      smooth: true,
      data: metricsData.value.memory,
      areaStyle: { color: createAreaGradient(colors.success) },
      lineStyle: { color: colors.success, width: 2 },
      itemStyle: { color: colors.success },
    }],
  });

  // Network Chart
  if (!networkChart) {
    networkChart = echarts.init(networkChartRef.value, "oui");
  }
  if (!networkChart) return;
  networkChart.setOption({
    title: { text: "Network I/O", left: "center" },
    tooltip: {
      trigger: "axis",
      axisPointer: { type: "cross" },
      formatter: (params: any) => {
        const paramsArray = Array.isArray(params) ? params : [params];
        let result = `${paramsArray[0].axisValue}<br/>`;
        paramsArray.forEach((param: any) => {
          result += `${param.seriesName}: ${formatBytes(Number(param.value))}<br/>`;
        });
        return result;
      },
    },
    legend: { data: ["Rx", "Tx"], bottom: 0 },
    grid: { left: "3%", right: "4%", bottom: "15%", top: "15%" },
    xAxis: { type: "category", boundaryGap: false, data: metricsData.value.timestamps },
    yAxis: {
      type: "value",
      name: "Bytes",
      axisLabel: { formatter: (value: number) => formatBytes(value) },
    },
    dataZoom: [{ type: "inside", start: 70, end: 100 }, { type: "slider", start: 70, end: 100 }],
    series: [
      {
        name: "Rx",
        type: "line",
        smooth: true,
        data: metricsData.value.networkRx,
        areaStyle: { color: createAreaGradient(colors.secondary) },
        lineStyle: { color: colors.secondary, width: 2 },
        itemStyle: { color: colors.secondary },
      },
      {
        name: "Tx",
        type: "line",
        smooth: true,
        data: metricsData.value.networkTx,
        areaStyle: { color: createAreaGradient(colors.warning) },
        lineStyle: { color: colors.warning, width: 2 },
        itemStyle: { color: colors.warning },
      },
    ],
  });

  // Disk I/O Chart
  if (!diskChart) {
    diskChart = echarts.init(diskChartRef.value, "oui");
  }
  if (!diskChart) return;
  diskChart.setOption({
    title: { text: "Disk I/O", left: "center" },
    tooltip: {
      trigger: "axis",
      axisPointer: { type: "cross" },
      formatter: (params: any) => {
        const paramsArray = Array.isArray(params) ? params : [params];
        let result = `${paramsArray[0].axisValue}<br/>`;
        paramsArray.forEach((param: any) => {
          result += `${param.seriesName}: ${formatBytes(Number(param.value))}<br/>`;
        });
        return result;
      },
    },
    legend: { data: ["Read", "Write"], bottom: 0 },
    grid: { left: "3%", right: "4%", bottom: "15%", top: "15%" },
    xAxis: { type: "category", boundaryGap: false, data: metricsData.value.timestamps },
    yAxis: {
      type: "value",
      name: "Bytes",
      axisLabel: { formatter: (value: number) => formatBytes(value) },
    },
    dataZoom: [{ type: "inside", start: 70, end: 100 }, { type: "slider", start: 70, end: 100 }],
    series: [
      {
        name: "Read",
        type: "line",
        smooth: true,
        data: metricsData.value.diskRead,
        areaStyle: { color: createAreaGradient(colors.info) },
        lineStyle: { color: colors.info, width: 2 },
        itemStyle: { color: colors.info },
      },
      {
        name: "Write",
        type: "line",
        smooth: true,
        data: metricsData.value.diskWrite,
        areaStyle: { color: createAreaGradient(colors.danger) },
        lineStyle: { color: colors.danger, width: 2 },
        itemStyle: { color: colors.danger },
      },
    ],
  });

  loadHistoricalMetrics();
};

// Update charts with new data
const updateCharts = () => {
  if (cpuChart) {
    cpuChart.setOption({
      xAxis: { data: metricsData.value.timestamps },
      series: [{ data: metricsData.value.cpu }],
    });
  }
  if (memoryChart) {
    memoryChart.setOption({
      xAxis: { data: metricsData.value.timestamps },
      series: [{ data: metricsData.value.memory }],
    });
  }
  if (networkChart) {
    networkChart.setOption({
      xAxis: { data: metricsData.value.timestamps },
      series: [
        { data: metricsData.value.networkRx },
        { data: metricsData.value.networkTx },
      ],
    });
  }
  if (diskChart) {
    diskChart.setOption({
      xAxis: { data: metricsData.value.timestamps },
      series: [
        { data: metricsData.value.diskRead },
        { data: metricsData.value.diskWrite },
      ],
    });
  }
};

// Helper to convert DateValue to Date
const dateValueToDate = (dateValue: any): Date | null => {
  if (!dateValue) return null;
  // DateValue from @internationalized/date can be CalendarDate, CalendarDateTime, etc.
  // Try to get year, month, day, hour, minute, second
  if (typeof dateValue === "object" && dateValue.year !== undefined) {
    // Handle CalendarDate (date only) or CalendarDateTime (with time)
    const year = dateValue.year;
    const month = dateValue.month || 1;
    const day = dateValue.day || 1;
    const hour = dateValue.hour || 0;
    const minute = dateValue.minute || 0;
    const second = dateValue.second || 0;

    return new Date(Date.UTC(year, month - 1, day, hour, minute, second));
  }

  // If it has a toDate method (some date libraries), use it
  if (typeof dateValue.toDate === "function") {
    return dateValue.toDate();
  }

  return null;
};

// Calculate start and end times based on selected timeframe
const getTimeRange = (): { startTime: Date; endTime: Date } => {
  let endTime = new Date();
  let startTime = new Date();

  if (selectedTimeframe.value === "custom") {
    if (customDateRange.value && customDateRange.value.length === 2) {
      const start = dateValueToDate(customDateRange.value[0]);
      const end = dateValueToDate(customDateRange.value[1]);
      if (start && end) {
        startTime = start;
        endTime = end;
      } else {
        // Fallback to last 24 hours if dates not valid
        startTime = new Date(endTime.getTime() - 24 * 60 * 60 * 1000);
      }
    } else {
      // Fallback to last 24 hours if custom dates not set
      startTime = new Date(endTime.getTime() - 24 * 60 * 60 * 1000);
    }
  } else {
    switch (selectedTimeframe.value) {
      case "10m":
        startTime = new Date(endTime.getTime() - 10 * 60 * 1000);
        break;
      case "1h":
        startTime = new Date(endTime.getTime() - 60 * 60 * 1000);
        break;
      case "24h":
        startTime = new Date(endTime.getTime() - 24 * 60 * 60 * 1000);
        break;
      case "7d":
        startTime = new Date(endTime.getTime() - 7 * 24 * 60 * 60 * 1000);
        break;
      case "30d":
        startTime = new Date(endTime.getTime() - 30 * 24 * 60 * 60 * 1000);
        break;
      default:
        startTime = new Date(endTime.getTime() - 24 * 60 * 60 * 1000);
    }
  }

  return { startTime, endTime };
};

// Load custom date range from preferences
const loadCustomDateRangeFromPreferences = async () => {
  const savedRange = preferencesStore.metricsPreferences?.customDateRange;
  if (!savedRange?.start || !savedRange?.end) {
    customDateRange.value = [];
    return;
  }

  const dateUtils = await import("@internationalized/date" as any).catch(
    () => null
  );
  if (!dateUtils) {
    customDateRange.value = [];
    return;
  }

  const { parseDateTime, parseDate } = dateUtils;

  try {
    let startStr = savedRange.start
      .replace("T", " ")
      .replace(/\.\d{3}Z?$/, "")
      .replace("Z", "")
      .trim();
    let endStr = savedRange.end
      .replace("T", " ")
      .replace(/\.\d{3}Z?$/, "")
      .replace("Z", "")
      .trim();

    if (!startStr.match(/\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/)) {
      startStr += ":00";
    }
    if (!endStr.match(/\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/)) {
      endStr += ":00";
    }

    const startDateValue = parseDateTime(startStr);
    const endDateValue = parseDateTime(endStr);
    customDateRange.value = [startDateValue, endDateValue];
  } catch {
    try {
      const startDateValue = parseDate(savedRange.start.slice(0, 10));
      const endDateValue = parseDate(savedRange.end.slice(0, 10));
      customDateRange.value = [startDateValue, endDateValue];
    } catch {
      customDateRange.value = [];
    }
  }
};

// Load historical metrics
const loadHistoricalMetrics = async () => {
  try {
    const { startTime, endTime } = getTimeRange();

    console.log("[GameServerMetrics] Loading metrics for game server:", props.gameServerId);
    console.log("[GameServerMetrics] Time range:", { startTime, endTime });
    const res = await client.getGameServerMetrics({
      gameServerId: props.gameServerId,
      startTime: {
        seconds: BigInt(Math.floor(startTime.getTime() / 1000)),
        nanos: 0,
      },
      endTime: {
        seconds: BigInt(Math.floor(endTime.getTime() / 1000)),
        nanos: 0,
      },
    });

    console.log("[GameServerMetrics] Received metrics response:", res);
    console.log("[GameServerMetrics] Metrics count:", res.metrics?.length || 0);

    if (res.metrics && res.metrics.length > 0) {
      const timestamps: string[] = [];
      const cpu: number[] = [];
      const memory: number[] = [];
      const networkRx: number[] = [];
      const networkTx: number[] = [];
      const diskRead: number[] = [];
      const diskWrite: number[] = [];

      res.metrics.forEach((metric: any) => {
        const date = metric.timestamp
          ? new Date(Number(metric.timestamp.seconds) * 1000)
          : new Date();
        timestamps.push(date.toLocaleTimeString());
        cpu.push(Number(metric.cpuUsagePercent || 0));
        memory.push(Number(metric.memoryUsageBytes || 0));
        networkRx.push(Number(metric.networkRxBytes || 0));
        networkTx.push(Number(metric.networkTxBytes || 0));
        diskRead.push(Number(metric.diskReadBytes || 0));
        diskWrite.push(Number(metric.diskWriteBytes || 0));
      });

      metricsData.value = { timestamps, cpu, memory, networkRx, networkTx, diskRead, diskWrite };
      
      // Initialize charts if they haven't been initialized yet
      if (!cpuChart && cpuChartRef.value) {
        await initCharts();
      } else {
        updateCharts();
      }
    } else {
      console.warn("[GameServerMetrics] No metrics returned from API");
    }
  } catch (err) {
    console.error("[GameServerMetrics] Failed to load historical metrics:", err);
    // Show error to user
    if (err instanceof Error) {
      console.error("[GameServerMetrics] Error details:", err.message, err.stack);
    }
  }
};

// Add new metric point
const addMetricPoint = async (metric: any) => {
  const date = metric.timestamp
    ? new Date(Number(metric.timestamp.seconds) * 1000)
    : new Date();
  const timestamp = date.toLocaleTimeString();

  const maxPoints = 100;
  if (metricsData.value.timestamps.length >= maxPoints) {
    metricsData.value.timestamps.shift();
    metricsData.value.cpu.shift();
    metricsData.value.memory.shift();
    metricsData.value.networkRx.shift();
    metricsData.value.networkTx.shift();
    metricsData.value.diskRead.shift();
    metricsData.value.diskWrite.shift();
  }

  metricsData.value.timestamps.push(timestamp);
  metricsData.value.cpu.push(Number(metric.cpuUsagePercent || 0));
  metricsData.value.memory.push(Number(metric.memoryUsageBytes || 0));
  metricsData.value.networkRx.push(Number(metric.networkRxBytes || 0));
  metricsData.value.networkTx.push(Number(metric.networkTxBytes || 0));
  metricsData.value.diskRead.push(Number(metric.diskReadBytes || 0));
  metricsData.value.diskWrite.push(Number(metric.diskWriteBytes || 0));

  // Initialize charts if this is the first metric
  if (!cpuChart && cpuChartRef.value) {
    await initCharts();
  } else {
    updateCharts();
  }
};

// Schedule reconnect for metrics stream
const scheduleReconnect = () => {
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
  }

  if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
    console.error("[GameServerMetrics] Max reconnect attempts reached, stopping stream");
    streaming.value = false;
    return;
  }

  reconnectAttempts++;
  const delay = Math.min(
    RECONNECT_DELAY * Math.pow(2, reconnectAttempts - 1),
    30000
  );

  reconnectTimeout = setTimeout(async () => {
    if (!streaming.value || props.gameServerStatus !== 3) {
      return;
    }
    console.log(
      `[GameServerMetrics] Attempting to reconnect stream (attempt ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})...`
    );
    await startStreaming();
  }, delay);
};

// Start streaming metrics
const startStreaming = async () => {
  if (streaming.value && streamController && !streamController.signal.aborted) {
    return; // Already streaming
  }

  // Clear any existing reconnect timeout
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }

  streaming.value = true;
  streamController = new AbortController();

  try {
    console.log("[GameServerMetrics] Starting metrics stream for game server:", props.gameServerId);
    
    // Call the stream method - Connect-RPC server streaming methods return async iterables
    const stream = await (client as any).streamGameServerMetrics(
      {
        gameServerId: props.gameServerId,
      },
      {
        signal: streamController.signal,
      }
    );

    console.log("[GameServerMetrics] Stream started, listening for metrics...");
    console.log("[GameServerMetrics] Current game server status:", props.gameServerStatus, "type:", typeof props.gameServerStatus);
    reconnectAttempts = 0; // Reset on successful connection
    
    // Iterate over the stream
    for await (const metric of stream) {
      if (streamController?.signal.aborted) {
        console.log("[GameServerMetrics] Stream aborted, stopping...");
        break;
      }
      
      // Only process metrics if we're still supposed to be streaming
      if (!streaming.value) {
        console.log("[GameServerMetrics] Streaming disabled, stopping...");
        break;
      }
      
      // Only process if game server is still running
      // GameServerStatus.RUNNING = 3
      // Check status - if undefined, assume it's still running (might be loading)
      // Only stop if status is explicitly set and not RUNNING
      const currentStatus = props.gameServerStatus;
      if (currentStatus !== undefined && currentStatus !== null && currentStatus !== 3) {
        console.log("[GameServerMetrics] Game server not running (status:", currentStatus, "type:", typeof currentStatus, "), stopping stream...");
        break;
      }
      // If status is undefined/null, continue streaming (data might still be loading)
      
      console.log("[GameServerMetrics] Received metric:", metric);
      addMetricPoint(metric);
    }
    
    console.log("[GameServerMetrics] Stream ended");
    
    // If we're still supposed to be streaming and server is running, reconnect
    if (streaming.value && props.gameServerStatus === 3 && !streamController?.signal.aborted) {
      scheduleReconnect();
    }
  } catch (err: any) {
    if (err.name === "AbortError" || streamController?.signal.aborted) {
      // User intentionally cancelled
      console.log("[GameServerMetrics] Stream aborted");
      streaming.value = false;
      streamController = null;
      return;
    }

    // Suppress "missing trailer" errors if we successfully received metrics
    const isMissingTrailerError =
      err.message?.toLowerCase().includes("missing trailer") ||
      err.message?.toLowerCase().includes("trailer") ||
      err.message?.toLowerCase().includes("missing endstreamresponse") ||
      err.message?.toLowerCase().includes("endstreamresponse") ||
      err.message?.toLowerCase().includes("unimplemented") ||
      err.message?.toLowerCase().includes("not fully implemented") ||
      err.code === "unknown";

    if (!isMissingTrailerError) {
      console.error("[GameServerMetrics] Failed to stream metrics:", err);
      if (err instanceof Error) {
        console.error("[GameServerMetrics] Stream error details:", err.message, err.stack);
      }
      
      // Check if it's a network/connection error (502, 503, etc.)
      const isNetworkError =
        err.message?.toLowerCase().includes("networkerror") ||
        err.message?.toLowerCase().includes("failed to fetch") ||
        err.message?.toLowerCase().includes("502") ||
        err.message?.toLowerCase().includes("503") ||
        err.message?.toLowerCase().includes("504") ||
        err.code === "ECONNREFUSED" ||
        err.code === "ETIMEDOUT";
      
      if (isNetworkError) {
        console.warn("[GameServerMetrics] Network error detected - backend service may be unavailable");
        // Still try to reconnect, but with a longer delay
        if (streaming.value && props.gameServerStatus === 3) {
          // Use a longer delay for network errors
          reconnectAttempts = Math.min(reconnectAttempts, MAX_RECONNECT_ATTEMPTS - 1);
          scheduleReconnect();
        } else {
          streaming.value = false;
          streamController = null;
        }
        return;
      }
    }
    
    // If we're still supposed to be streaming and server is running, reconnect
    if (streaming.value && props.gameServerStatus === 3) {
      scheduleReconnect();
    } else {
      streaming.value = false;
      streamController = null;
    }
  }
};

// Stop streaming
const stopStreaming = () => {
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }
  reconnectAttempts = 0;
  
  if (streamController) {
    streamController.abort();
    streamController = null;
  }
  streaming.value = false;
};

// Toggle streaming
const toggleStreaming = () => {
  if (streaming.value) {
    stopStreaming();
  } else {
    startStreaming();
  }
};

// Load usage data
const refreshUsage = async () => {
  try {
    const month = new Date().toISOString().slice(0, 7); // YYYY-MM
    const res = await client.getGameServerUsage({
      gameServerId: props.gameServerId,
      organizationId: props.organizationId,
      month,
    });
    usageData.value = res;
  } catch (err) {
    console.error("Failed to fetch usage:", err);
    usageData.value = null;
  }
};

// Handle window resize
const handleResize = () => {
  cpuChart?.resize();
  memoryChart?.resize();
  networkChart?.resize();
  diskChart?.resize();
};

// Watch game server status and start/stop streaming accordingly
watch(
  () => props.gameServerStatus,
  (status) => {
    // GameServerStatus.RUNNING = 3
    if (status === 3 && !streaming.value) {
      startStreaming();
    } else if (status !== 3 && streaming.value) {
      stopStreaming();
    }
  },
  { immediate: true }
);

onMounted(async () => {
  if (!import.meta.client) return;

  if (selectedTimeframe.value === "custom") {
    await loadCustomDateRangeFromPreferences();
  }

  await refreshUsage();
  // Always initialize charts (they'll show empty state initially)
  await initCharts();
  window.addEventListener("resize", handleResize);
  // Streaming will be started by the watch if server is running
});

onBeforeUnmount(() => {
  stopStreaming();
  window.removeEventListener("resize", handleResize);
  cpuChart?.dispose();
  memoryChart?.dispose();
  networkChart?.dispose();
  diskChart?.dispose();
});
</script>
