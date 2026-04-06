<template>
  <OuiGrid :cols="{ sm: 2, md: 4 }" gap="sm">
    <!-- CPU Usage -->
    <UiStatCard
      label="CPU"
      :icon="CpuChipIcon"
      color="primary"
      :value="latestMetric ? currentCpuUsage.toFixed(1) + '%' : '—'"
      :bar="currentCpuUsage"
      :bar-class="cpuBarColor"
      :streaming="isStreaming"
    />

    <!-- Memory Usage -->
    <UiStatCard
      label="Memory"
      :icon="CircleStackIcon"
      color="info"
      :streaming="isStreaming"
      :subtitle="latestMetric ? 'Active' : 'Waiting for data'"
    >
      <template v-if="latestMetric">
        <OuiByte :value="currentMemoryUsage" unit-display="short" />
      </template>
      <template v-else>—</template>
    </UiStatCard>

    <!-- Network Rx -->
    <UiStatCard
      label="Inbound"
      :icon="ArrowDownTrayIcon"
      color="success"
      :streaming="isStreaming"
      :subtitle="latestMetric ? 'Total received' : 'Waiting for data'"
    >
      <template v-if="latestMetric">
        <OuiByte :value="currentNetworkRx" unit-display="short" base="decimal" />
      </template>
      <template v-else>—</template>
    </UiStatCard>

    <!-- Network Tx -->
    <UiStatCard
      label="Outbound"
      :icon="ArrowUpTrayIcon"
      color="secondary"
      :streaming="isStreaming"
      :subtitle="latestMetric ? 'Total sent' : 'Waiting for data'"
    >
      <template v-if="latestMetric">
        <OuiByte :value="currentNetworkTx" unit-display="short" base="decimal" />
      </template>
      <template v-else>—</template>
    </UiStatCard>
  </OuiGrid>
</template>

<script setup lang="ts">
import { computed } from "vue";
import {
  CpuChipIcon,
  CircleStackIcon,
  ArrowDownTrayIcon,
  ArrowUpTrayIcon,
} from "@heroicons/vue/24/outline";
import OuiByte from "~/components/oui/Byte.vue";

interface Props {
  isStreaming: boolean;
  latestMetric: any;
  currentCpuUsage: number;
  currentMemoryUsage: number;
  currentNetworkRx: number;
  currentNetworkTx: number;
}

const props = defineProps<Props>();

const cpuBarColor = computed(() => {
  if (props.currentCpuUsage >= 90) return 'bg-danger';
  if (props.currentCpuUsage >= 70) return 'bg-warning';
  return 'bg-accent-primary';
});
</script>

