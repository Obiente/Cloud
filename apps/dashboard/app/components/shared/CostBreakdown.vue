<template>
  <OuiCard v-if="usageData && usageData.estimatedMonthly && usageData.current">
    <OuiCardHeader>
      <OuiFlex justify="between" align="center">
        <OuiStack gap="xs">
          <OuiText size="lg" weight="bold">Estimated Monthly Cost</OuiText>
          <OuiText size="xs" color="muted">
            Cost breakdown by resource type
          </OuiText>
        </OuiStack>
      </OuiFlex>
    </OuiCardHeader>
    <OuiCardBody>
      <OuiStack gap="md">
        <OuiFlex align="center" justify="between" class="pb-3 border-b border-border-muted">
          <OuiStack gap="xs">
            <OuiText size="sm" color="muted">Total Estimated</OuiText>
            <OuiText size="2xl" weight="bold" color="primary">
              {{ formatCurrency(Number(usageData.estimatedMonthly.estimatedCostCents) / 100) }}
            </OuiText>
          </OuiStack>
          <OuiText size="xs" color="muted">
            Current: {{ formatCurrency(Number(usageData.current.estimatedCostCents) / 100) }}
          </OuiText>
        </OuiFlex>
        <OuiGrid :cols="{ sm: 1, md: 2 }" gap="sm">
          <OuiBox
            v-for="cost in costBreakdown"
            :key="cost.label"
            p="sm"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiFlex align="center" justify="between" gap="md">
              <OuiFlex align="center" gap="sm" class="flex-1">
                <OuiBox
                  class="w-3 h-3 rounded-full"
                  :class="cost.color"
                />
                <OuiText size="sm" weight="medium" color="primary">
                  {{ cost.label }}
                </OuiText>
              </OuiFlex>
              <OuiText size="sm" weight="semibold" color="primary">
                {{ cost.value }}
              </OuiText>
            </OuiFlex>
          </OuiBox>
        </OuiGrid>
      </OuiStack>
    </OuiCardBody>
  </OuiCard>
</template>

<script setup lang="ts">
import { computed } from "vue";

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
  const breakdown = [];
  
  const totalCents = Number(estimated.estimatedCostCents);
  if (totalCents > 0) {
    // Use actual cost breakdown if available, otherwise estimate
    breakdown.push(
      { 
        label: "CPU", 
        value: formatCurrency(estimated.cpuCostCents ? Number(estimated.cpuCostCents) / 100 : totalCents * 0.4 / 100), 
        color: "bg-accent-primary" 
      },
      { 
        label: "Memory", 
        value: formatCurrency(estimated.memoryCostCents ? Number(estimated.memoryCostCents) / 100 : totalCents * 0.3 / 100), 
        color: "bg-success" 
      },
      { 
        label: "Bandwidth", 
        value: formatCurrency(estimated.bandwidthCostCents ? Number(estimated.bandwidthCostCents) / 100 : totalCents * 0.2 / 100), 
        color: "bg-accent-secondary" 
      },
      { 
        label: "Storage", 
        value: formatCurrency(estimated.storageCostCents ? Number(estimated.storageCostCents) / 100 : totalCents * 0.1 / 100), 
        color: "bg-warning" 
      },
    );
  }
  
  return breakdown;
});
</script>
