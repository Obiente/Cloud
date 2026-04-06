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
          <div class="rounded-lg border border-border-default p-3">
            <OuiStack gap="sm">
              <OuiFlex align="center" gap="xs">
                <CpuChipIcon class="h-3 w-3 text-accent-primary" />
                <OuiText size="xs" color="tertiary">CPU</OuiText>
              </OuiFlex>
              <OuiText size="lg" weight="semibold">
                {{ formatCPUUsage(Number(usageData.current.cpuCoreSeconds)) }}
              </OuiText>
              <div class="h-1 rounded-full bg-surface-muted overflow-hidden">
                <div class="h-full rounded-full bg-accent-primary/60" style="width: 40%" />
              </div>
              <OuiText size="xs" color="tertiary">
                {{ formatCurrency((usageData.current.cpuCostCents ? Number(usageData.current.cpuCostCents) : 0) / 100) }}
              </OuiText>
            </OuiStack>
          </div>

          <!-- Memory -->
          <div class="rounded-lg border border-border-default p-3">
            <OuiStack gap="sm">
              <OuiFlex align="center" gap="xs">
                <CircleStackIcon class="h-3 w-3 text-accent-info" />
                <OuiText size="xs" color="tertiary">Memory</OuiText>
              </OuiFlex>
              <OuiText size="lg" weight="semibold">
                <OuiByte :value="Number(usageData.current.memoryByteSeconds) / 3600" unit-display="short" />/hr
              </OuiText>
              <div class="h-1 rounded-full bg-surface-muted overflow-hidden">
                <div class="h-full rounded-full bg-accent-info/60" style="width: 55%" />
              </div>
              <OuiText size="xs" color="tertiary">
                {{ formatUptime(Number(usageData.current.uptimeSeconds)) }} uptime
              </OuiText>
            </OuiStack>
          </div>

          <!-- Bandwidth -->
          <div class="rounded-lg border border-border-default p-3">
            <OuiStack gap="sm">
              <OuiFlex align="center" gap="xs">
                <ArrowsRightLeftIcon class="h-3 w-3 text-success" />
                <OuiText size="xs" color="tertiary">Bandwidth</OuiText>
              </OuiFlex>
              <OuiText size="lg" weight="semibold">
                <OuiByte :value="Number(usageData.current.bandwidthRxBytes) + Number(usageData.current.bandwidthTxBytes)" unit-display="short" base="decimal" />
              </OuiText>
              <div class="h-1 rounded-full bg-surface-muted overflow-hidden">
                <div class="h-full rounded-full bg-success/60" style="width: 30%" />
              </div>
              <OuiText size="xs" color="tertiary">
                <template v-if="usageData.current.requestCount !== undefined && Number(usageData.current.requestCount) > 0">
                  {{ formatNumber(Number(usageData.current.requestCount)) }} requests
                </template>
                <template v-else>
                  Total transfer
                </template>
              </OuiText>
            </OuiStack>
          </div>

          <!-- Storage -->
          <div class="rounded-lg border border-border-default p-3">
            <OuiStack gap="sm">
              <OuiFlex align="center" gap="xs">
                <ArchiveBoxIcon class="h-3 w-3 text-accent-secondary" />
                <OuiText size="xs" color="tertiary">Storage</OuiText>
              </OuiFlex>
              <OuiText size="lg" weight="semibold">
                <OuiByte :value="Number(usageData.current.storageBytes || usageData.current.diskBytes || 0)" unit-display="short" />
              </OuiText>
              <div class="h-1 rounded-full bg-surface-muted overflow-hidden">
                <div class="h-full rounded-full bg-accent-secondary/60" style="width: 25%" />
              </div>
              <OuiText size="xs" color="tertiary">
                <template v-if="usageData.current.errorCount !== undefined && Number(usageData.current.errorCount) > 0">
                  {{ formatNumber(Number(usageData.current.errorCount)) }} errors
                </template>
                <template v-else>
                  Disk usage
                </template>
              </OuiText>
            </OuiStack>
          </div>
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

