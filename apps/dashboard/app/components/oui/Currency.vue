<template>
  <span>{{ formattedCurrency }}</span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { formatCurrency } from "~/utils/common";

interface Props {
  value: number | bigint; // Amount in cents
  currency?: string; // Currency code (default: "USD")
}

const props = withDefaults(defineProps<Props>(), {
  currency: "USD",
});

const formattedCurrency = computed(() => {
  return formatCurrency(props.value, props.currency);
});
</script>

