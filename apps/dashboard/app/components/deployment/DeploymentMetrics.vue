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
                {{
                  formatCurrency(
                    combinedUsage?.current?.estimatedCostCents ||
                      usageData?.current?.estimatedCostCents ||
                      0
                  )
                }}
              </OuiText>
            </OuiStack>
            <OuiText size="xs" color="muted">
              Est:
              {{
                formatCurrency(
                  combinedUsage?.estimatedMonthly?.estimatedCostCents ||
                    usageData?.estimatedMonthly?.estimatedCostCents ||
                    0
                )
              }}
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
                {{
                  formatCoreSecondsToHours(
                    combinedUsage?.current?.cpuCoreSeconds ?? 0
                  )
                }}
              </OuiText>
            </OuiStack>
            <OuiText size="xs" color="muted">
              Est:
              {{
                formatCoreSecondsToHours(
                  combinedUsage?.estimatedMonthly?.cpuCoreSeconds ?? 0
                )
              }}
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
                {{
                  formatMemoryByteSecondsToGB(
                    combinedUsage?.current?.memoryByteSeconds ?? 0
                  )
                }}
              </OuiText>
            </OuiStack>
            <OuiText size="xs" color="muted">
              Est:
              {{
                formatMemoryByteSecondsToGB(
                  combinedUsage?.estimatedMonthly?.memoryByteSeconds ?? 0
                )
              }}
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
                {{
                  formatBandwidthToGB(
                    combinedUsage?.current?.bandwidthRxBytes ?? 0,
                    combinedUsage?.current?.bandwidthTxBytes ?? 0
                  )
                }}
              </OuiText>
            </OuiStack>
            <OuiText size="xs" color="muted">
              Rx:
              {{
                formatBytesToGB(combinedUsage?.current?.bandwidthRxBytes ?? 0)
              }}
              GB Tx:
              {{
                formatBytesToGB(combinedUsage?.current?.bandwidthTxBytes ?? 0)
              }}
              GB
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

      <!-- Container/Service Tabs -->
      <OuiTabs
        v-model="selectedTab"
        :tabs="containerTabs"
        @update:model-value="onTabChange"
      />

      <OuiGrid cols="1" cols-lg="2" gap="lg">
        <!-- CPU Usage Chart -->
        <OuiCard>
          <OuiCardBody>
            <OuiStack gap="md">
              <OuiText size="lg" weight="semibold">CPU Usage</OuiText>
              <div ref="cpuChartRef" style="width: 100%; height: 300px"></div>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Memory Usage Chart -->
        <OuiCard>
          <OuiCardBody>
            <OuiStack gap="md">
              <OuiText size="lg" weight="semibold">Memory Usage</OuiText>
              <div
                ref="memoryChartRef"
                style="width: 100%; height: 300px"
              ></div>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Network I/O Chart -->
        <OuiCard>
          <OuiCardBody>
            <OuiStack gap="md">
              <OuiText size="lg" weight="semibold">Network I/O</OuiText>
              <div
                ref="networkChartRef"
                style="width: 100%; height: 300px"
              ></div>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Disk I/O Chart -->
        <OuiCard>
          <OuiCardBody>
            <OuiStack gap="md">
              <OuiText size="lg" weight="semibold">Disk I/O</OuiText>
              <div ref="diskChartRef" style="width: 100%; height: 300px"></div>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>
      </OuiGrid>
    </OuiStack>
  </OuiStack>
</template>

<script setup lang="ts">
  import {
    ref,
    computed,
    onMounted,
    onBeforeUnmount,
    nextTick,
    watch,
  } from "vue";
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
  import { DeploymentService } from "@obiente/proto";
  import {
    registerOUIEChartsTheme,
    getOUIEChartsColors,
    createAreaGradient,
  } from "~/utils/echarts-theme";
  import { usePreferencesStore } from "~/stores/preferences";

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

  // Theme will be registered when charts are initialized

  interface Props {
    deploymentId: string;
    organizationId: string;
  }

  const props = defineProps<Props>();

  const client = useConnectClient(DeploymentService);
  const preferencesStore = usePreferencesStore();
  // Containers list
  const containers = ref<Array<{ containerId: string; serviceName?: string }>>(
    []
  );
  const selectedTab = ref<string>("aggregated");

  // Timeframe selection - sync with preferences store (following SettingsPreferences pattern)
  type TimeframeOption = "10m" | "1h" | "24h" | "7d" | "30d" | "custom";
  const selectedTimeframe = computed({
    get: () => {
      const value = preferencesStore.metricsPreferences?.timeframe;
      console.log("[DeploymentMetrics] selectedTimeframe get() called:", {
        value,
        metricsPreferences: preferencesStore.metricsPreferences,
        hydrated: preferencesStore.hydrated,
        isClient: import.meta.client,
      });
      // Return default value if undefined to ensure select always has a valid value
      return value ?? "24h";
    },
    set: (value: TimeframeOption) => {
      console.log("[DeploymentMetrics] selectedTimeframe set() called:", value);
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
      console.log("[DeploymentMetrics] metricsPreferences changed:", {
        new: newMetrics,
        old: oldMetrics,
        newTimeframe: newMetrics?.timeframe,
        oldTimeframe: oldMetrics?.timeframe,
        hydrated: preferencesStore.hydrated,
      });
      
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

  // Container tabs for OuiTabs component
  const containerTabs = computed(() => {
    const tabs = [{ id: "aggregated", label: "Aggregated" }];
    containers.value.forEach((container) => {
      tabs.push({
        id: container.containerId,
        label: container.serviceName || container.containerId.substring(0, 12),
      });
    });
    return tabs;
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

  // Live metrics accumulation (from streaming)
  const liveMetrics = ref<{
    cpuCoreSeconds: number;
    memoryByteSeconds: number;
    bandwidthRxBytes: number;
    bandwidthTxBytes: number;
    diskReadBytes: number;
    diskWriteBytes: number;
    startTime: Date | null;
    lastMetricTime: Date | null;
    intervalSeconds: number;
  }>({
    cpuCoreSeconds: 0,
    memoryByteSeconds: 0,
    bandwidthRxBytes: 0,
    bandwidthTxBytes: 0,
    diskReadBytes: 0,
    diskWriteBytes: 0,
    startTime: null,
    lastMetricTime: null,
    intervalSeconds: 5,
  });

  // Helper to convert bigint to number safely
  const toNumber = (value: bigint | number | undefined | null): number => {
    if (value === undefined || value === null) return 0;
    if (typeof value === "bigint") return Number(value);
    return Number(value) || 0;
  };

  // Combined usage data (live + aggregated)
  const combinedUsage = computed(() => {
    const current = usageData.value?.current || {};
    const estimated = usageData.value?.estimatedMonthly || {};

    // Convert bigint values to numbers (protobuf int64 becomes bigint in TypeScript)
    let currentCpu = toNumber(current.cpuCoreSeconds);
    let currentMemory = toNumber(current.memoryByteSeconds);
    let currentRx = toNumber(current.bandwidthRxBytes);
    let currentTx = toNumber(current.bandwidthTxBytes);
    let currentDiskRead = toNumber(current.diskReadBytes);
    let currentDiskWrite = toNumber(current.diskWriteBytes);
    let currentStorage = toNumber(current.storageBytes);

    const estCpu = toNumber(estimated.cpuCoreSeconds);
    const estMemory = toNumber(estimated.memoryByteSeconds);
    const estRx = toNumber(estimated.bandwidthRxBytes);
    const estTx = toNumber(estimated.bandwidthTxBytes);
    const estStorage = toNumber(estimated.storageBytes);

    // If current values are 0 but we have estimated values, use estimated as baseline
    // This handles cases where aggregation hasn't updated current yet
    // The live metrics will add on top of this
    if (currentCpu === 0 && estCpu > 0) {
      // Use a proportion of estimated based on elapsed time in month
      const now = new Date();
      const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);
      const monthEnd = new Date(
        now.getFullYear(),
        now.getMonth() + 1,
        0,
        23,
        59,
        59
      );
      const monthDuration = monthEnd.getTime() - monthStart.getTime();
      const elapsedInMonth = now.getTime() - monthStart.getTime();
      const progressRatio = Math.min(
        1,
        Math.max(0, elapsedInMonth / monthDuration)
      );
      currentCpu = estCpu * progressRatio;
      currentMemory = estMemory * progressRatio;
      // Network/bandwidth is cumulative, so use the estimated values directly
      currentRx = estRx;
      currentTx = estTx;
    }

    // Get month start for live calculation
    const now = new Date();
    const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);

    // Calculate elapsed seconds from month start or streaming start (whichever is later)
    const startTime = liveMetrics.value.startTime || monthStart;
    const effectiveStart = startTime > monthStart ? startTime : monthStart;
    const elapsedSeconds = Math.max(
      1,
      Math.floor((now.getTime() - effectiveStart.getTime()) / 1000)
    );

    // Combine: aggregated + live (live adds to aggregated)
    const combinedCurrent = {
      cpuCoreSeconds: currentCpu + liveMetrics.value.cpuCoreSeconds,
      memoryByteSeconds: currentMemory + liveMetrics.value.memoryByteSeconds,
      bandwidthRxBytes: currentRx + liveMetrics.value.bandwidthRxBytes,
      bandwidthTxBytes: currentTx + liveMetrics.value.bandwidthTxBytes,
      diskReadBytes: currentDiskRead + liveMetrics.value.diskReadBytes,
      diskWriteBytes: currentDiskWrite + liveMetrics.value.diskWriteBytes,
      storageBytes: currentStorage, // Storage is snapshot, not rate-based
      requestCount: toNumber(current.requestCount),
      errorCount: toNumber(current.errorCount),
      uptimeSeconds: toNumber(current.uptimeSeconds),
    };

    // Use API's calculated costs directly - all calculations are done server-side
    // The API already includes live metrics in its calculations via aggregation
    const combinedEstimated = {
      // Use API estimates (they're already projected for the full month)
      cpuCoreSeconds: estCpu || 0,
      memoryByteSeconds: estMemory || 0,
      bandwidthRxBytes: estRx || 0, // Cumulative
      bandwidthTxBytes: estTx || 0, // Cumulative
      storageBytes: estStorage || 0, // Storage is snapshot
      diskReadBytes: combinedCurrent.diskReadBytes, // Not in API estimates, use current
      diskWriteBytes: combinedCurrent.diskWriteBytes,
      requestCount: toNumber(estimated.requestCount),
      errorCount: toNumber(estimated.errorCount),
      uptimeSeconds: toNumber(estimated.uptimeSeconds),
      estimatedCostCents: toNumber(estimated.estimatedCostCents), // Use API's calculated cost
    };

    return {
      current: {
        ...combinedCurrent,
        // Use API's calculated current cost (includes aggregated metrics)
        estimatedCostCents: toNumber(current.estimatedCostCents),
        // Use API's calculated per-resource costs
        cpuCostCents: current.cpuCostCents != null ? toNumber(current.cpuCostCents) : undefined,
        memoryCostCents: current.memoryCostCents != null ? toNumber(current.memoryCostCents) : undefined,
        bandwidthCostCents: current.bandwidthCostCents != null ? toNumber(current.bandwidthCostCents) : undefined,
        storageCostCents: current.storageCostCents != null ? toNumber(current.storageCostCents) : undefined,
      },
      estimatedMonthly: {
        ...combinedEstimated,
        // Use API's calculated per-resource costs
        cpuCostCents: estimated.cpuCostCents != null ? toNumber(estimated.cpuCostCents) : undefined,
        memoryCostCents: estimated.memoryCostCents != null ? toNumber(estimated.memoryCostCents) : undefined,
        bandwidthCostCents: estimated.bandwidthCostCents != null ? toNumber(estimated.bandwidthCostCents) : undefined,
        storageCostCents: estimated.storageCostCents != null ? toNumber(estimated.storageCostCents) : undefined,
      },
    };
  });

  // Cost breakdown per resource - uses backend calculated values
  const costBreakdown = computed(() => {
    const current = combinedUsage.value?.current || {};
    const estimated = combinedUsage.value?.estimatedMonthly || {};
    
    // All costs are calculated on the backend and returned in cents
    return {
      current: {
        cpu: current.cpuCostCents ?? 0,
        memory: current.memoryCostCents ?? 0,
        bandwidth: current.bandwidthCostCents ?? 0,
        storage: current.storageCostCents ?? 0,
        total: current.estimatedCostCents ?? 0, // Total current cost
      },
      estimated: {
        cpu: estimated.cpuCostCents ?? 0,
        memory: estimated.memoryCostCents ?? 0,
        bandwidth: estimated.bandwidthCostCents ?? 0,
        storage: estimated.storageCostCents ?? 0,
        total: estimated.estimatedCostCents ?? 0, // Total estimated cost
      },
    };
  });

  const refreshUsage = async () => {
    try {
      const res = await (client as any).getDeploymentUsage({
        deploymentId: props.deploymentId,
        organizationId: props.organizationId,
      });
      usageData.value = res;

      // Reset live metrics when refreshing (so we accumulate from this point)
      // Or we could keep accumulating - for now, reset to avoid double counting
      // Actually, we should keep accumulating but mark the start time
      if (!liveMetrics.value.startTime) {
        liveMetrics.value.startTime = new Date();
      }
    } catch (err) {
      console.error("Failed to fetch usage:", err);
      usageData.value = null;
    }
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

  // Initialize charts
  const initCharts = async () => {
    if (
      !cpuChartRef.value ||
      !memoryChartRef.value ||
      !networkChartRef.value ||
      !diskChartRef.value
    ) {
      return;
    }

    // Ensure theme is registered and chroma is loaded
    await registerOUIEChartsTheme(echarts);

    const colors = getOUIEChartsColors();

    // CPU Chart
    cpuChart = echarts.init(cpuChartRef.value, "oui");

    // Calculate dynamic max for CPU chart (allow values > 100%)
    // Use max of: actual max value * 1.2, or 100, whichever is higher
    // Initially use metricsData, will be updated dynamically as data comes in
    const cpuData =
      metricsData.value.cpu.length > 0 ? metricsData.value.cpu : [0];
    const maxCpu = cpuData.length > 0 ? Math.max(...cpuData) : 0;
    const cpuChartMax = Math.max(100, Math.ceil(maxCpu * 1.2));

    cpuChart.setOption({
      title: {
        text: "CPU Usage (%)",
        left: "center",
      },
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "cross" },
        formatter: (params: any) => {
          if (Array.isArray(params)) {
            return params
              .map((p: any) => `${p.seriesName}: ${p.value.toFixed(2)}%`)
              .join("<br>");
          }
          return `${params.seriesName}: ${params.value.toFixed(2)}%`;
        },
      },
      grid: {
        left: "3%",
        right: "4%",
        bottom: "10%",
        top: "15%",
        containLabel: true,
      },
      xAxis: {
        type: "category",
        boundaryGap: false,
        data: metricsData.value.timestamps,
      },
      yAxis: {
        type: "value",
        name: "%",
        min: 0,
        max: cpuChartMax,
        // Allow values > 100% - CPU can exceed 100% when using multiple cores
      },
      dataZoom: [
        {
          type: "inside",
          start: 70,
          end: 100,
        },
        {
          type: "slider",
          start: 70,
          end: 100,
        },
      ],
      series: [
        {
          name: "CPU Usage",
          type: "line",
          smooth: true,
          data: metricsData.value.cpu,
          areaStyle: {
            color: createAreaGradient(colors.primary),
          },
          lineStyle: { color: colors.primary, width: 2 },
          itemStyle: { color: colors.primary },
        },
      ],
    });

    // Memory Chart
    memoryChart = echarts.init(memoryChartRef.value, "oui");
    memoryChart.setOption({
      title: {
        text: "Memory Usage",
        left: "center",
      },
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "cross" },
        formatter: (params: any) => {
          const value = Array.isArray(params) ? params[0] : params;
          return `${value.axisValue}<br/>${value.seriesName}: ${formatBytes(
            Number(value.value)
          )}`;
        },
      },
      grid: {
        left: "3%",
        right: "4%",
        bottom: "10%",
        top: "15%",
        containLabel: true,
      },
      xAxis: {
        type: "category",
        boundaryGap: false,
        data: metricsData.value.timestamps,
      },
      yAxis: {
        type: "value",
        name: "Bytes",
        axisLabel: {
          formatter: (value: number) => formatBytes(value),
        },
      },
      dataZoom: [
        {
          type: "inside",
          start: 70,
          end: 100,
        },
        {
          type: "slider",
          start: 70,
          end: 100,
        },
      ],
      series: [
        {
          name: "Memory Usage",
          type: "line",
          smooth: true,
          data: metricsData.value.memory,
          areaStyle: {
            color: createAreaGradient(colors.success),
          },
          lineStyle: { color: colors.success, width: 2 },
          itemStyle: { color: colors.success },
        },
      ],
    });

    // Network Chart
    networkChart = echarts.init(networkChartRef.value, "oui");
    networkChart.setOption({
      title: {
        text: "Network I/O",
        left: "center",
      },
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "cross" },
        formatter: (params: any) => {
          const paramsArray = Array.isArray(params) ? params : [params];
          let result = `${paramsArray[0].axisValue}<br/>`;
          paramsArray.forEach((param: any) => {
            result += `${param.seriesName}: ${formatBytes(
              Number(param.value)
            )}<br/>`;
          });
          return result;
        },
      },
      legend: {
        data: ["Rx", "Tx"],
        bottom: 0,
      },
      grid: {
        left: "3%",
        right: "4%",
        bottom: "15%",
        top: "15%",
        containLabel: true,
      },
      xAxis: {
        type: "category",
        boundaryGap: false,
        data: metricsData.value.timestamps,
      },
      yAxis: {
        type: "value",
        name: "Bytes",
        axisLabel: {
          formatter: (value: number) => formatBytes(value),
        },
      },
      dataZoom: [
        {
          type: "inside",
          start: 70,
          end: 100,
        },
        {
          type: "slider",
          start: 70,
          end: 100,
        },
      ],
      series: [
        {
          name: "Rx",
          type: "line",
          smooth: true,
          data: metricsData.value.networkRx,
          areaStyle: {
            color: createAreaGradient(colors.secondary),
          },
          lineStyle: { color: colors.secondary, width: 2 },
          itemStyle: { color: colors.secondary },
        },
        {
          name: "Tx",
          type: "line",
          smooth: true,
          data: metricsData.value.networkTx,
          areaStyle: {
            color: createAreaGradient(colors.warning),
          },
          lineStyle: { color: colors.warning, width: 2 },
          itemStyle: { color: colors.warning },
        },
      ],
    });

    // Disk Chart
    diskChart = echarts.init(diskChartRef.value, "oui");
    diskChart.setOption({
      title: {
        text: "Disk I/O",
        left: "center",
      },
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "cross" },
        formatter: (params: any) => {
          const paramsArray = Array.isArray(params) ? params : [params];
          let result = `${paramsArray[0].axisValue}<br/>`;
          paramsArray.forEach((param: any) => {
            result += `${param.seriesName}: ${formatBytes(
              Number(param.value)
            )}<br/>`;
          });
          return result;
        },
      },
      legend: {
        data: ["Read", "Write"],
        bottom: 0,
      },
      grid: {
        left: "3%",
        right: "4%",
        bottom: "15%",
        top: "15%",
        containLabel: true,
      },
      xAxis: {
        type: "category",
        boundaryGap: false,
        data: metricsData.value.timestamps,
      },
      yAxis: {
        type: "value",
        name: "Bytes",
        axisLabel: {
          formatter: (value: number) => formatBytes(value),
        },
      },
      dataZoom: [
        {
          type: "inside",
          start: 70,
          end: 100,
        },
        {
          type: "slider",
          start: 70,
          end: 100,
        },
      ],
      series: [
        {
          name: "Read",
          type: "line",
          smooth: true,
          data: metricsData.value.diskRead,
          areaStyle: {
            color: createAreaGradient(colors.danger),
          },
          lineStyle: { color: colors.danger, width: 2 },
          itemStyle: { color: colors.danger },
        },
        {
          name: "Write",
          type: "line",
          smooth: true,
          data: metricsData.value.diskWrite,
          areaStyle: {
            color: createAreaGradient(colors.info),
          },
          lineStyle: { color: colors.info, width: 2 },
          itemStyle: { color: colors.info },
        },
      ],
    });

    // Load initial historical data
    loadHistoricalMetrics();
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  // Load containers for this deployment
  const loadContainers = async () => {
    try {
      const res = await (client as any).listDeploymentContainers({
        deploymentId: props.deploymentId,
        organizationId: props.organizationId,
      });

      if (res?.containers) {
        containers.value = res.containers
          .filter((c: any) => c.status === "running")
          .map((c: any) => ({
            containerId: c.containerId,
            serviceName: c.serviceName || undefined,
          }));

        // If we have containers and "aggregated" is selected but we want to default to first container
        // Keep aggregated as default for now
      }
    } catch (err) {
      console.error("Failed to load containers:", err);
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

  // Load historical metrics
  const loadHistoricalMetrics = async () => {
    try {
      const { startTime, endTime } = getTimeRange();

      const request: any = {
        deploymentId: props.deploymentId,
        organizationId: props.organizationId,
        startTime: {
          seconds: Math.floor(startTime.getTime() / 1000),
          nanos: 0,
        },
        endTime: {
          seconds: Math.floor(endTime.getTime() / 1000),
          nanos: 0,
        },
      };

      // Add container filter if a specific container is selected
      if (selectedTab.value !== "aggregated") {
        const container = containers.value.find(
          (c) => c.containerId === selectedTab.value
        );
        if (container) {
          if (container.serviceName) {
            request.serviceName = container.serviceName;
          } else {
            request.containerId = container.containerId;
          }
          request.aggregate = false;
        }
      } else {
        request.aggregate = true;
      }

      const res = await (client as any).getDeploymentMetrics(request);

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

        metricsData.value = {
          timestamps,
          cpu,
          memory,
          networkRx,
          networkTx,
          diskRead,
          diskWrite,
        };

        updateCharts();
      }
    } catch (err) {
      console.error("Failed to load historical metrics:", err);
    }
  };

  // Update charts with new data
  const updateCharts = () => {
    if (cpuChart) {
      // Calculate dynamic max for CPU chart (allow values > 100%)
      // CPU can exceed 100% when using multiple cores
      const cpuData =
        metricsData.value.cpu.length > 0 ? metricsData.value.cpu : [0];
      const maxCpu = cpuData.length > 0 ? Math.max(...cpuData) : 0;
      const cpuChartMax = Math.max(100, Math.ceil(maxCpu * 1.2));

      cpuChart.setOption({
        xAxis: { data: metricsData.value.timestamps },
        yAxis: {
          max: cpuChartMax,
          // Allow values > 100% - CPU can exceed 100% when using multiple cores
        },
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

  // Add new metric point
  const addMetricPoint = (metric: any) => {
    const date = metric.timestamp
      ? new Date(Number(metric.timestamp.seconds) * 1000)
      : new Date();
    const timestamp = date.toLocaleTimeString();

    // Initialize start time if not set
    if (!liveMetrics.value.startTime) {
      liveMetrics.value.startTime = date;
    }

    // Calculate interval since last metric (default to configured interval)
    let intervalSeconds = liveMetrics.value.intervalSeconds;
    if (liveMetrics.value.lastMetricTime) {
      intervalSeconds = Math.max(
        1,
        Math.floor(
          (date.getTime() - liveMetrics.value.lastMetricTime.getTime()) / 1000
        )
      );
    }
    liveMetrics.value.lastMetricTime = date;

    // Accumulate live metrics
    const cpuUsage = Number(metric.cpuUsagePercent || 0);
    const memoryUsage = Number(metric.memoryUsageBytes || 0);
    const networkRx = Number(metric.networkRxBytes || 0);
    const networkTx = Number(metric.networkTxBytes || 0);
    const diskRead = Number(metric.diskReadBytes || 0);
    const diskWrite = Number(metric.diskWriteBytes || 0);

    // CPU core-seconds: (cpu% / 100) * intervalSeconds
    // CPU usage can exceed 100% when using multiple cores (e.g., 200% = 2 cores at 100%)
    // Example: 200% CPU over 1 second = (200/100) * 1 = 2 core-seconds ✓
    liveMetrics.value.cpuCoreSeconds += (cpuUsage / 100) * intervalSeconds;

    // Memory byte-seconds: memory bytes * seconds
    liveMetrics.value.memoryByteSeconds += memoryUsage * intervalSeconds;

    // Network and disk are already incremental per interval, just sum them
    liveMetrics.value.bandwidthRxBytes += networkRx;
    liveMetrics.value.bandwidthTxBytes += networkTx;
    liveMetrics.value.diskReadBytes += diskRead;
    liveMetrics.value.diskWriteBytes += diskWrite;

    // Limit to last 100 points
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
    metricsData.value.cpu.push(cpuUsage);
    metricsData.value.memory.push(memoryUsage);
    metricsData.value.networkRx.push(networkRx);
    metricsData.value.networkTx.push(networkTx);
    metricsData.value.diskRead.push(diskRead);
    metricsData.value.diskWrite.push(diskWrite);

    updateCharts();
  };

  // Handle tab change
  const onTabChange = (tab: string) => {
    // Stop current stream
    stopStreaming();

    // Clear current metrics
    metricsData.value = {
      timestamps: [],
      cpu: [],
      memory: [],
      networkRx: [],
      networkTx: [],
      diskRead: [],
      diskWrite: [],
    };
    updateCharts();

    // Reload historical metrics for selected container
    loadHistoricalMetrics();

    // Restart streaming if it was running
    if (streaming.value) {
      startStreaming();
    }
  };

  // Start streaming metrics
  const startStreaming = async () => {
    if (streaming.value || streamController) {
      return;
    }

    streaming.value = true;
    streamController = new AbortController();

    try {
      const request: any = {
        deploymentId: props.deploymentId,
        organizationId: props.organizationId,
        intervalSeconds: 5,
      };

      // Add container filter if a specific container is selected
      if (selectedTab.value !== "aggregated") {
        const container = containers.value.find(
          (c) => c.containerId === selectedTab.value
        );
        if (container) {
          if (container.serviceName) {
            request.serviceName = container.serviceName;
          } else {
            request.containerId = container.containerId;
          }
          request.aggregate = false;
        }
      } else {
        request.aggregate = true;
      }

      const stream = await (client as any).streamDeploymentMetrics(request, {
        signal: streamController.signal,
      });

      for await (const metric of stream) {
        if (streamController?.signal.aborted) {
          break;
        }
        addMetricPoint(metric);
      }
    } catch (err: any) {
      if (err.name === "AbortError") {
        // User intentionally cancelled
        return;
      }

      // Suppress "missing trailer" errors if we successfully received metrics
      const isMissingTrailerError =
        err.message?.toLowerCase().includes("missing trailer") ||
        err.message?.toLowerCase().includes("trailer") ||
        err.code === "unknown";

      if (!isMissingTrailerError) {
        console.error("Failed to stream metrics:", err);
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

  // Handle window resize
  const handleResize = () => {
    cpuChart?.resize();
    memoryChart?.resize();
    networkChart?.resize();
    diskChart?.resize();
  };

  onMounted(async () => {
    if (!import.meta.client) return;

    if (selectedTimeframe.value === "custom") {
      await loadCustomDateRangeFromPreferences();
    }

    await loadContainers();
    // Load usage data
    await refreshUsage();
    initCharts();
    window.addEventListener("resize", handleResize);
    // Auto-start streaming
    startStreaming();
    // Load initial historical data
    loadHistoricalMetrics();
  });

  onBeforeUnmount(() => {
    stopStreaming();
    window.removeEventListener("resize", handleResize);
    cpuChart?.dispose();
    memoryChart?.dispose();
    networkChart?.dispose();
    diskChart?.dispose();
  });

  // Expose refresh method for parent component
  defineExpose({
    refreshUsage,
  });
</script>
