<template>
  <OuiCard v-if="usageData && usageData.estimatedMonthly && usageData.current" variant="outline">
    <OuiCardBody>
      <OuiStack gap="md">
        <OuiFlex justify="between" align="center">
          <UiSectionHeader :icon="BanknotesIcon" color="success">Cost Estimate</UiSectionHeader>
          <OuiText size="xs" color="tertiary">
            Current: {{ usageData.current.costEstimateUnavailable ? "Unavailable" : formatCurrency(Number(usageData.current.estimatedCostCents) / 100) }}
          </OuiText>
        </OuiFlex>

        <!-- Total -->
        <OuiText size="2xl" weight="semibold">
          {{ usageData.estimatedMonthly.costEstimateUnavailable ? "Unavailable" : formatCurrency(Number(usageData.estimatedMonthly.estimatedCostCents) / 100) }}
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
  if (!props.usageData?.estimatedMonthly || props.usageData.estimatedMonthly.costEstimateUnavailable) return [];
  const estimated = props.usageData.estimatedMonthly;
  const totalCents = Number(estimated.estimatedCostCents);
  if (totalCents <= 0) return [];

  const cpu = Number(estimated.cpuCostCents ?? 0);
  const mem = Number(estimated.memoryCostCents ?? 0);
  const bw = Number(estimated.bandwidthCostCents ?? 0);
  const stor = Number(estimated.storageCostCents ?? 0);
  const sum = cpu + mem + bw + stor;
  if (sum <= 0) return [];

  return [
    { label: "CPU", value: formatCurrency(cpu / 100), color: "bg-accent-primary", pct: Math.round((cpu / sum) * 100) },
    { label: "Memory", value: formatCurrency(mem / 100), color: "bg-accent-info", pct: Math.round((mem / sum) * 100) },
    { label: "Bandwidth", value: formatCurrency(bw / 100), color: "bg-success", pct: Math.round((bw / sum) * 100) },
    { label: "Storage", value: formatCurrency(stor / 100), color: "bg-warning", pct: Math.round((stor / sum) * 100) },
  ];
});
</script>
