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
                  <OuiByte :value="getStorageBytesValue(usageData?.current?.diskBytes)" unit-display="short" />
                </OuiText>
              </OuiFlex>
            </OuiStack>
            <OuiText size="xs" color="muted">
              Disk Storage
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiGrid>

    <!-- Cost Breakdown -->
    <CostBreakdown v-if="usageData" :usage-data="usageData" />

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

        <!-- Disk Usage Chart -->
        <OuiCard>
          <OuiCardBody>
            <OuiStack gap="md">
              <OuiText size="lg" weight="semibold">Disk Usage</OuiText>
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
import { VPSService } from "@obiente/proto";
import {
  registerOUIEChartsTheme,
  getOUIEChartsColors,
  createAreaGradient,
} from "~/utils/echarts-theme";
import { CubeIcon } from "@heroicons/vue/24/outline";
import OuiByte from "~/components/oui/Byte.vue";
import CostBreakdown from "~/components/shared/CostBreakdown.vue";
import { usePreferencesStore } from "~/stores/preferences";
import type { ECharts } from "echarts/core";

interface Props {
  vpsId: string;
  organizationId: string;
  vpsStatus?: number; // VPSStatus enum value (3 = RUNNING)
}

const props = defineProps<Props>();
const client = useConnectClient(VPSService);
const preferencesStore = usePreferencesStore();

// Timeframe selection - sync with preferences store
type TimeframeOption = "10m" | "1h" | "24h" | "7d" | "30d" | "custom";
const selectedTimeframe = computed({
  get: () => {
    const value = preferencesStore.metricsPreferences?.timeframe;
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
    if (preferencesStore.hydrated && newMetrics?.timeframe && !oldMetrics?.timeframe) {
      const timeframe = newMetrics.timeframe;
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
  // VPSStatus.RUNNING = 3
  if (props.vpsStatus === 3) {
    return "Metrics will appear here once collected. Please wait a moment...";
  }
  return "Start the VPS to begin collecting metrics.";
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
  diskUsed: number[];
  diskTotal: number[];
}>({
  timestamps: [],
  cpu: [],
  memory: [],
  networkRx: [],
  networkTx: [],
  diskUsed: [],
  diskTotal: [],
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

// Load usage data
const loadUsageData = async () => {
  try {
    const month = new Date().toISOString().slice(0, 7); // YYYY-MM
    const res = await client.getVPSUsage({
      vpsId: props.vpsId,
      organizationId: props.organizationId,
      month,
    });
    usageData.value = res;
  } catch (err) {
    console.error("[VPSMetrics] Failed to load usage data:", err);
  }
};

// Initialize charts
const initCharts = async () => {
  if (!cpuChartRef.value || !memoryChartRef.value || !networkChartRef.value || !diskChartRef.value) {
    return;
  }

  // Dynamically load ECharts
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
  
  const echarts = echartsModule as typeof import("echarts/core");

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
  const cpuData = metricsData.value.cpu.length > 0 ? metricsData.value.cpu : [0];
  const maxCpu = cpuData.length > 0 ? Math.max(...cpuData) : 0;
  const cpuChartMax = Math.max(100, Math.ceil(maxCpu * 1.2));

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
    yAxis: { type: "value", name: "%", min: 0, max: cpuChartMax },
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

  // Disk Chart
  if (!diskChart) {
    diskChart = echarts.init(diskChartRef.value, "oui");
  }
  if (!diskChart) return;
  diskChart.setOption({
    title: { text: "Disk Usage", left: "center" },
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
    legend: { data: ["Used", "Total"], bottom: 0 },
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
        name: "Used",
        type: "line",
        smooth: true,
        data: metricsData.value.diskUsed,
        areaStyle: { color: createAreaGradient(colors.info) },
        lineStyle: { color: colors.info, width: 2 },
        itemStyle: { color: colors.info },
      },
      {
        name: "Total",
        type: "line",
        smooth: true,
        data: metricsData.value.diskTotal,
        areaStyle: { color: createAreaGradient(colors.warning) },
        lineStyle: { color: colors.warning, width: 2 },
        itemStyle: { color: colors.warning },
      },
    ],
  });

  loadHistoricalMetrics();
};

// Update charts with new data
const updateCharts = () => {
  if (cpuChart) {
    const cpuData = metricsData.value.cpu.length > 0 ? metricsData.value.cpu : [0];
    const maxCpu = cpuData.length > 0 ? Math.max(...cpuData) : 0;
    const cpuChartMax = Math.max(100, Math.ceil(maxCpu * 1.2));
    cpuChart.setOption({
      xAxis: { data: metricsData.value.timestamps },
      yAxis: { max: cpuChartMax },
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
        { data: metricsData.value.diskUsed },
        { data: metricsData.value.diskTotal },
      ],
    });
  }
};

// Helper to convert DateValue to Date
const dateValueToDate = (dateValue: any): Date | null => {
  if (!dateValue) return null;
  if (typeof dateValue === "object" && dateValue.year !== undefined) {
    const year = dateValue.year;
    const month = dateValue.month || 1;
    const day = dateValue.day || 1;
    const hour = dateValue.hour || 0;
    const minute = dateValue.minute || 0;
    const second = dateValue.second || 0;
    return new Date(Date.UTC(year, month - 1, day, hour, minute, second));
  }
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
        startTime = new Date(endTime.getTime() - 24 * 60 * 60 * 1000);
      }
    } else {
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

  const dateUtils = await import("@internationalized/date" as any).catch(() => null);
  if (!dateUtils) {
    customDateRange.value = [];
    return;
  }

  const { parseDateTime, parseDate } = dateUtils;

  try {
    let startStr = savedRange.start.replace("T", " ").replace(/\.\d{3}Z?$/, "").replace("Z", "").trim();
    let endStr = savedRange.end.replace("T", " ").replace(/\.\d{3}Z?$/, "").replace("Z", "").trim();

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

    console.log("[VPSMetrics] Loading metrics for VPS:", props.vpsId);
    const res = await client.getVPSMetrics({
      vpsId: props.vpsId,
      startTime: {
        seconds: BigInt(Math.floor(startTime.getTime() / 1000)),
        nanos: 0,
      },
      endTime: {
        seconds: BigInt(Math.floor(endTime.getTime() / 1000)),
        nanos: 0,
      },
    });

    if (res.metrics && res.metrics.length > 0) {
      const timestamps: string[] = [];
      const cpu: number[] = [];
      const memory: number[] = [];
      const networkRx: number[] = [];
      const networkTx: number[] = [];
      const diskUsed: number[] = [];
      const diskTotal: number[] = [];

      res.metrics.forEach((metric: any) => {
        const date = metric.timestamp
          ? new Date(Number(metric.timestamp.seconds) * 1000)
          : new Date();
        timestamps.push(date.toLocaleTimeString());
        cpu.push(Number(metric.cpuUsagePercent || 0));
        memory.push(Number(metric.memoryUsedBytes || 0));
        networkRx.push(Number(metric.networkRxBytes || 0));
        networkTx.push(Number(metric.networkTxBytes || 0));
        diskUsed.push(Number(metric.diskUsedBytes || 0));
        diskTotal.push(Number(metric.diskTotalBytes || 0));
      });

      metricsData.value = { timestamps, cpu, memory, networkRx, networkTx, diskUsed, diskTotal };
      
      if (!cpuChart && cpuChartRef.value) {
        await initCharts();
      } else {
        updateCharts();
      }
    } else {
      console.warn("[VPSMetrics] No metrics returned from API");
    }
  } catch (err) {
    console.error("[VPSMetrics] Failed to load historical metrics:", err);
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
    metricsData.value.diskUsed.shift();
    metricsData.value.diskTotal.shift();
  }

  metricsData.value.timestamps.push(timestamp);
  metricsData.value.cpu.push(Number(metric.cpuUsagePercent || 0));
  metricsData.value.memory.push(Number(metric.memoryUsedBytes || 0));
  metricsData.value.networkRx.push(Number(metric.networkRxBytes || 0));
  metricsData.value.networkTx.push(Number(metric.networkTxBytes || 0));
  metricsData.value.diskUsed.push(Number(metric.diskUsedBytes || 0));
  metricsData.value.diskTotal.push(Number(metric.diskTotalBytes || 0));

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
    console.error("[VPSMetrics] Max reconnect attempts reached, stopping stream");
    streaming.value = false;
    return;
  }

  reconnectAttempts++;
  const delay = Math.min(RECONNECT_DELAY * Math.pow(2, reconnectAttempts - 1), 30000);

  reconnectTimeout = setTimeout(async () => {
    if (!streaming.value || props.vpsStatus !== 3) {
      return;
    }
    console.log(`[VPSMetrics] Attempting to reconnect stream (attempt ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})...`);
    await startStreaming();
  }, delay);
};

// Start streaming metrics
const startStreaming = async () => {
  if (streaming.value && streamController && !streamController.signal.aborted) {
    return;
  }

  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }

  streaming.value = true;
  streamController = new AbortController();

  try {
    console.log("[VPSMetrics] Starting metrics stream for VPS:", props.vpsId);
    
    const stream = await (client as any).streamVPSMetrics(
      {
        vpsId: props.vpsId,
      },
      {
        signal: streamController.signal,
      }
    );

    console.log("[VPSMetrics] Stream started, listening for metrics...");
    reconnectAttempts = 0;
    
    for await (const metric of stream) {
      if (streamController?.signal.aborted) {
        console.log("[VPSMetrics] Stream aborted, stopping...");
        break;
      }
      
      if (!streaming.value) {
        console.log("[VPSMetrics] Streaming disabled, stopping...");
        break;
      }
      
      const currentStatus = props.vpsStatus;
      if (currentStatus !== undefined && currentStatus !== null && currentStatus !== 3) {
        console.log("[VPSMetrics] VPS not running (status:", currentStatus, "), stopping stream...");
        break;
      }
      
      console.log("[VPSMetrics] Received metric:", metric);
      addMetricPoint(metric);
    }
    
    console.log("[VPSMetrics] Stream ended");
    
    if (streaming.value && props.vpsStatus === 3 && !streamController?.signal.aborted) {
      scheduleReconnect();
    }
  } catch (err: any) {
    if (err.name === "AbortError" || streamController?.signal.aborted) {
      console.log("[VPSMetrics] Stream aborted");
      streaming.value = false;
      streamController = null;
      return;
    }

    const isMissingTrailerError =
      err.message?.toLowerCase().includes("missing trailer") ||
      err.message?.toLowerCase().includes("trailer") ||
      err.message?.toLowerCase().includes("unimplemented") ||
      err.code === "unknown";

    if (!isMissingTrailerError) {
      console.error("[VPSMetrics] Failed to stream metrics:", err);
      
      const isNetworkError =
        err.message?.toLowerCase().includes("networkerror") ||
        err.message?.toLowerCase().includes("failed to fetch") ||
        err.message?.toLowerCase().includes("502") ||
        err.message?.toLowerCase().includes("503") ||
        err.message?.toLowerCase().includes("504") ||
        err.code === "ECONNREFUSED" ||
        err.code === "ETIMEDOUT";
      
      if (isNetworkError) {
        console.warn("[VPSMetrics] Network error detected - backend service may be unavailable");
        if (streaming.value && props.vpsStatus === 3) {
          scheduleReconnect();
        }
      }
    }
    
    streaming.value = false;
    streamController = null;
  }
};

// Stop streaming metrics
const stopStreaming = () => {
  streaming.value = false;
  if (streamController) {
    streamController.abort();
    streamController = null;
  }
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }
  reconnectAttempts = 0;
};

// Toggle streaming
const toggleStreaming = () => {
  if (streaming.value) {
    stopStreaming();
  } else {
    startStreaming();
  }
};

// Lifecycle
onMounted(async () => {
  await loadUsageData();
  await nextTick();
  await initCharts();
  
  // Auto-start streaming if VPS is running
  if (props.vpsStatus === 3) {
    await startStreaming();
  }
});

onBeforeUnmount(() => {
  stopStreaming();
  
  // Cleanup charts
  if (cpuChart) {
    cpuChart.dispose();
    cpuChart = null;
  }
  if (memoryChart) {
    memoryChart.dispose();
    memoryChart = null;
  }
  if (networkChart) {
    networkChart.dispose();
    networkChart = null;
  }
  if (diskChart) {
    diskChart.dispose();
    diskChart = null;
  }
});

// Watch for status changes
watch(
  () => props.vpsStatus,
  (newStatus, oldStatus) => {
    if (newStatus === 3 && oldStatus !== 3) {
      // VPS started - start streaming
      if (!streaming.value) {
        startStreaming();
      }
    } else if (newStatus !== 3 && oldStatus === 3) {
      // VPS stopped - stop streaming
      stopStreaming();
    }
  }
);
</script>


