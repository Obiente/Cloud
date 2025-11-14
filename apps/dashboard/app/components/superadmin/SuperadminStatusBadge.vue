<template>
  <OuiBadge :variant="computedVariant" :size="size">
    {{ computedLabel }}
  </OuiBadge>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { BadgeVariant, BadgeSize } from "~/components/oui/Badge.vue";

const props = withDefaults(defineProps<{
  status: string | number | null | undefined;
  statusMap?: Record<string | number, { label: string; variant: BadgeVariant }>;
  label?: string;
  variant?: BadgeVariant;
  size?: BadgeSize;
}>(), {
  size: "sm",
});

const computedLabel = computed(() => {
  if (props.label) return props.label;
  if (props.statusMap && props.status !== null && props.status !== undefined) {
    return props.statusMap[props.status]?.label || String(props.status);
  }
  return props.status ? String(props.status) : "—";
});

const computedVariant = computed(() => {
  if (props.variant) return props.variant;
  if (props.statusMap && props.status !== null && props.status !== undefined) {
    return props.statusMap[props.status]?.variant || "secondary";
  }
  return "secondary";
});
</script>

