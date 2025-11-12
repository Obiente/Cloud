<template>
  <OuiGrid :cols="cols" :cols-md="colsMd" :cols-lg="colsLg" gap="lg">
    <slot />
  </OuiGrid>
</template>

<script setup lang="ts">
import type { GridColumns } from "~/components/oui/types";

withDefaults(
  defineProps<{
    cols?: GridColumns;
    colsMd?: GridColumns;
    colsLg?: GridColumns;
  }>(),
  {
    cols: 1 as GridColumns,
    colsMd: 2 as GridColumns,
    colsLg: 4 as GridColumns,
  }
);
</script>

