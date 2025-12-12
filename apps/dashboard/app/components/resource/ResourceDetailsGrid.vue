<template>
  <OuiGrid :cols="finalCols" gap="lg">
    <slot />
  </OuiGrid>
</template>

<script setup lang="ts">
  import { computed } from "vue";
  import type {
    GridColumns,
    Responsive,
    Breakpoint,
  } from "~/components/oui/types";

  const props = defineProps<{
    cols?: Responsive<GridColumns>;
  }>();

  const finalCols = computed<Responsive<GridColumns>>(() => {
    return props.cols ?? { sm: 2, lg: 4 };
  });
</script>
