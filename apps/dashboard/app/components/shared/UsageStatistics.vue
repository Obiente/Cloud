<template>
  <OuiCard v-if="usageData && usageData.current" variant="outline">
    <OuiCardBody>
      <OuiStack gap="md">
        <OuiFlex justify="between" align="center">
          <OuiFlex align="center" gap="sm">
            <ChartBarIcon class="h-4 w-4 text-accent-primary" />
            <OuiText size="sm" weight="semibold">Monthly Usage</OuiText>
          </OuiFlex>
          <OuiBadge variant="secondary" size="xs">
            {{ usageData.month }}
          </OuiBadge>
        </OuiFlex>

        <OuiGrid :cols="{ sm: 2, lg: 4 }" gap="sm">
          <!-- CPU -->
          <UiStatCard
            label="CPU"
            :icon="CpuChipIcon"
            color="primary"
            :value="formatCPUUsage(Number(usageData.current.cpuCoreSeconds))"
            :bar="40"
            :subtitle="formatCurrency((usageData.current.cpuCostCents ? Number(usageData.current.cpuCostCents) : 0) / 100)"
            value-size="lg"
          />

          <!-- Memory -->
          <UiStatCard
            label="Memory"
            :icon="CircleStackIcon"
            color="info"
            :bar="55"
            :subtitle="formatUptime(Number(usageData.current.uptimeSeconds)) + ' uptime'"
            value-size="lg"
          >
            <OuiByte :value="Number(usageData.current.memoryByteSeconds) / 3600" unit-display="short" />/hr
          </UiStatCard>

          <!-- Bandwidth -->
          <UiStatCard
            label="Bandwidth"
            :icon="ArrowsRightLeftIcon"
            color="success"
            :bar="30"
            :subtitle="usageData.current.requestCount !== undefined && Number(usageData.current.requestCount) > 0 ? formatNumber(Number(usageData.current.requestCount)) + ' requests' : 'Total transfer'"
            value-size="lg"
          >
            <OuiByte :value="Number(usageData.current.bandwidthRxBytes) + Number(usageData.current.bandwidthTxBytes)" unit-display="short" base="decimal" />
          </UiStatCard>

          <!-- Storage -->
          <UiStatCard
            label="Storage"
            :icon="ArchiveBoxIcon"
            color="secondary"
            :bar="25"
            :subtitle="usageData.current.errorCount !== undefined && Number(usageData.current.errorCount) > 0 ? formatNumber(Number(usageData.current.errorCount)) + ' errors' : 'Disk usage'"
            value-size="lg"
          >
            <OuiByte :value="Number(usageData.current.storageBytes || usageData.current.diskBytes || 0)" unit-display="short" />
          </UiStatCard>
        </OuiGrid>
      </OuiStack>
    </OuiCardBody>
  </OuiCard>
</template>

<script setup lang="ts">
import {
  ChartBarIcon,
  CpuChipIcon,
  CircleStackIcon,
  ArrowsRightLeftIcon,
  ArchiveBoxIcon,
} from "@heroicons/vue/24/outline";
import OuiByte from "~/components/oui/Byte.vue";

interface Props {
  usageData: any;
}

defineProps<Props>();

const formatCurrency = (amount: number) =>
  new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount);

const formatNumber = (value: number) =>
  new Intl.NumberFormat("en-US").format(value);

const formatCPUUsage = (coreSeconds: number): string => {
  const hours = coreSeconds / 3600;
  if (hours < 1) {
    return `${(coreSeconds / 60).toFixed(1)} min`;
  }
  return `${hours.toFixed(1)} core-hrs`;
};

const formatUptime = (seconds: number): string => {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
  return `${Math.floor(seconds / 86400)}d`;
};
</script>

