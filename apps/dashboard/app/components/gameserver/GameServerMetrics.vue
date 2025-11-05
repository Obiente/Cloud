<template>
  <OuiStack gap="xl">
    <!-- Usage Summary Cards -->
    <OuiGrid cols="1" cols-md="2" cols-lg="4" gap="lg">
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
    </OuiGrid>

    <!-- Real-time Metrics Charts -->
    <OuiStack gap="lg">
      <OuiFlex justify="between" align="center" wrap="wrap" gap="md">
        <OuiText size="xl" weight="bold">Real-time Metrics</OuiText>
        <OuiFlex gap="sm" align="center" wrap="wrap">
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
import { ref, computed, onMounted, onBeforeUnmount, watch } from "vue";
import * as echarts from "echarts/core";
import { LineChart } from "echarts/charts";
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  DataZoomComponent,
} from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";
import { useConnectClient } from "~/lib/connect-client";
import { GameServerService } from "@obiente/proto";
import {
  registerOUIEChartsTheme,
  getOUIEChartsColors,
  createAreaGradient,
} from "~/utils/echarts-theme";

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

interface Props {
  gameServerId: string;
  organizationId: string;
  gameServerStatus?: number; // GameServerStatus enum value
}

const props = defineProps<Props>();
const client = useConnectClient(GameServerService);

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
let cpuChart: echarts.ECharts | null = null;
let memoryChart: echarts.ECharts | null = null;
let networkChart: echarts.ECharts | null = null;
let diskChart: echarts.ECharts | null = null;

// Streaming state
const streaming = ref(false);
let streamController: AbortController | null = null;

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

// Initialize charts
const initCharts = async () => {
  if (!cpuChartRef.value || !memoryChartRef.value || !networkChartRef.value || !diskChartRef.value) {
    return;
  }

  await registerOUIEChartsTheme(echarts);
  const colors = getOUIEChartsColors();

  // CPU Chart
  if (!cpuChart) {
    cpuChart = echarts.init(cpuChartRef.value, "oui");
  }
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

// Load historical metrics
const loadHistoricalMetrics = async () => {
  try {
    const endTime = new Date();
    const startTime = new Date(endTime.getTime() - 24 * 60 * 60 * 1000); // Last 24 hours

    console.log("[GameServerMetrics] Loading metrics for game server:", props.gameServerId);
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

// Start streaming metrics
const startStreaming = async () => {
  if (streaming.value || streamController) return;

  streaming.value = true;
  streamController = new AbortController();

  try {
    console.log("[GameServerMetrics] Starting metrics stream for game server:", props.gameServerId);
    const stream = await client.streamGameServerMetrics({
      gameServerId: props.gameServerId,
    });

    console.log("[GameServerMetrics] Stream started, listening for metrics...");
    for await (const metric of stream) {
      if (streamController?.signal.aborted) break;
      console.log("[GameServerMetrics] Received metric:", metric);
      addMetricPoint(metric);
    }
  } catch (err: any) {
    if (err.name !== "AbortError") {
      console.error("[GameServerMetrics] Failed to stream metrics:", err);
      if (err instanceof Error) {
        console.error("[GameServerMetrics] Stream error details:", err.message, err.stack);
      }
    }
  } finally {
    streaming.value = false;
    streamController = null;
  }
};

// Stop streaming
const stopStreaming = () => {
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

onMounted(async () => {
  if (!import.meta.client) return;
  await refreshUsage();
  // Always initialize charts (they'll show empty state initially)
  await initCharts();
  window.addEventListener("resize", handleResize);
  // Only start streaming if server is running
  if (props.gameServerStatus === 3) {
    startStreaming();
  }
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
