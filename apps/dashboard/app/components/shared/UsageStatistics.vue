<template>
  <OuiCard v-if="usageData && usageData.current">
    <OuiCardHeader>
      <OuiFlex justify="between" align="center">
        <OuiStack gap="xs">
          <OuiText size="lg" weight="bold">Monthly Usage</OuiText>
          <OuiText size="xs" color="muted">
            Current month resource usage and estimated costs
          </OuiText>
        </OuiStack>
        <OuiBadge variant="secondary">
          {{ usageData.month }}
        </OuiBadge>
      </OuiFlex>
    </OuiCardHeader>
    <OuiCardBody>
      <OuiGrid cols="1" cols-md="2" cols-lg="4" gap="lg">
        <!-- CPU Usage -->
        <OuiStack gap="sm">
          <OuiFlex align="center" justify="between">
            <OuiText size="sm" weight="medium" color="primary">CPU</OuiText>
            <OuiText size="xs" color="muted">
              {{ formatCPUUsage(Number(usageData.current.cpuCoreSeconds)) }}
            </OuiText>
          </OuiFlex>
          <OuiText size="xs" color="muted">
            {{ formatCurrency((usageData.current.cpuCostCents ? Number(usageData.current.cpuCostCents) : 0) / 100) }} estimated
          </OuiText>
        </OuiStack>

        <!-- Memory Usage -->
        <OuiStack gap="sm">
          <OuiFlex align="center" justify="between">
            <OuiText size="sm" weight="medium" color="primary">Memory</OuiText>
            <OuiText size="xs" color="muted">
              <OuiByte :value="Number(usageData.current.memoryByteSeconds) / 3600" unit-display="short" />/hr avg
            </OuiText>
          </OuiFlex>
          <OuiText size="xs" color="muted">
            {{ formatUptime(Number(usageData.current.uptimeSeconds)) }} uptime
          </OuiText>
        </OuiStack>

        <!-- Bandwidth Usage -->
        <OuiStack gap="sm">
          <OuiFlex align="center" justify="between">
            <OuiText size="sm" weight="medium" color="primary">Bandwidth</OuiText>
            <OuiText size="xs" color="muted">
              <OuiByte :value="Number(usageData.current.bandwidthRxBytes) + Number(usageData.current.bandwidthTxBytes)" unit-display="short" />
            </OuiText>
          </OuiFlex>
          <OuiText size="xs" color="muted">
            <template v-if="usageData.current.requestCount !== undefined && Number(usageData.current.requestCount) > 0">
              {{ formatNumber(Number(usageData.current.requestCount)) }} requests
            </template>
            <template v-else>
              Total bandwidth usage
            </template>
          </OuiText>
        </OuiStack>

        <!-- Storage Usage -->
        <OuiStack gap="sm">
          <OuiFlex align="center" justify="between">
            <OuiText size="sm" weight="medium" color="primary">Storage</OuiText>
            <OuiText size="xs" color="muted">
              <OuiByte :value="Number(usageData.current.storageBytes)" unit-display="short" />
            </OuiText>
          </OuiFlex>
          <OuiText size="xs" color="muted">
            <template v-if="usageData.current.errorCount !== undefined && Number(usageData.current.errorCount) > 0">
              {{ formatNumber(Number(usageData.current.errorCount)) }} errors
            </template>
            <template v-else>
              Storage usage
            </template>
          </OuiText>
        </OuiStack>
      </OuiGrid>
    </OuiCardBody>
  </OuiCard>
</template>

<script setup lang="ts">
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

