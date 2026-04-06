<template>
  <OuiCard v-if="usageData && usageData.estimatedMonthly && usageData.current" variant="outline">
    <OuiCardBody>
      <OuiStack gap="md">
        <OuiFlex justify="between" align="center">
          <OuiFlex align="center" gap="sm">
            <BanknotesIcon class="h-4 w-4 text-success" />
            <OuiText size="sm" weight="semibold">Cost Estimate</OuiText>
          </OuiFlex>
          <OuiText size="xs" color="tertiary">
            Current: {{ formatCurrency(Number(usageData.current.estimatedCostCents) / 100) }}
          </OuiText>
        </OuiFlex>

        <!-- Total -->
        <OuiText size="2xl" weight="semibold">
          {{ formatCurrency(Number(usageData.estimatedMonthly.estimatedCostCents) / 100) }}
          <OuiText as="span" size="xs" color="tertiary"> /mo projected</OuiText>
        </OuiText>

        <!-- Stacked bar -->
        <div v-if="costBreakdown.length > 0" class="h-2 rounded-full bg-surface-muted overflow-hidden flex">
          <div
            v-for="cost in costBreakdown"
            :key="cost.label"
            class="h-full first:rounded-l-full last:rounded-r-full"
            :class="cost.color"
            :style="{ width: cost.pct + '%' }"
          />
        </div>

        <!-- Legend -->
        <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
          <OuiFlex
            v-for="cost in costBreakdown"
            :key="cost.label"
            align="center"
            gap="sm"
          >
            <span class="h-2 w-2 rounded-full shrink-0" :class="cost.color" />
            <OuiStack gap="none">
              <OuiText size="xs" color="tertiary">{{ cost.label }}</OuiText>
              <OuiText size="sm" weight="semibold">{{ cost.value }}</OuiText>
            </OuiStack>
          </OuiFlex>
        </div>
      </OuiStack>
    </OuiCardBody>
  </OuiCard>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { BanknotesIcon } from "@heroicons/vue/24/outline";

interface Props {
  usageData: any;
}

const props = defineProps<Props>();

const formatCurrency = (amount: number) =>
  new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount);

// Cost breakdown
const costBreakdown = computed(() => {
  if (!props.usageData?.estimatedMonthly) return [];
  const estimated = props.usageData.estimatedMonthly;
  const totalCents = Number(estimated.estimatedCostCents);
  if (totalCents <= 0) return [];

  const cpu = estimated.cpuCostCents ? Number(estimated.cpuCostCents) : totalCents * 0.4;
  const mem = estimated.memoryCostCents ? Number(estimated.memoryCostCents) : totalCents * 0.3;
  const bw = estimated.bandwidthCostCents ? Number(estimated.bandwidthCostCents) : totalCents * 0.2;
  const stor = estimated.storageCostCents ? Number(estimated.storageCostCents) : totalCents * 0.1;
  const sum = cpu + mem + bw + stor;

  return [
    { label: "CPU", value: formatCurrency(cpu / 100), color: "bg-accent-primary", pct: Math.round((cpu / sum) * 100) },
    { label: "Memory", value: formatCurrency(mem / 100), color: "bg-accent-info", pct: Math.round((mem / sum) * 100) },
    { label: "Bandwidth", value: formatCurrency(bw / 100), color: "bg-success", pct: Math.round((bw / sum) * 100) },
    { label: "Storage", value: formatCurrency(stor / 100), color: "bg-warning", pct: Math.round((stor / sum) * 100) },
  ];
});
</script>
