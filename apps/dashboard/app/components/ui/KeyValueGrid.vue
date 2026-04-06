<template>
  <!--
    A responsive 2-column (default) grid of label+value pairs.
    Used for resource detail cards (VPS specs, game server info, etc.)

    <UiKeyValueGrid :items="[
      { label: 'CPU', value: '4 cores' },
      { label: 'Memory', value: '8 GB' },
    ]" />

    Use the `value` slot for complex values:
    <UiKeyValueGrid :items="items">
      <template #value-cpu><OuiByte :value="..." /></template>
    </UiKeyValueGrid>
  -->
  <div :class="gridClass">
    <template v-for="item in items" :key="item.label">
      <OuiStack v-if="!item.hidden" gap="xs">
        <OuiText size="xs" color="tertiary">{{ item.label }}</OuiText>
        <!-- Named slot per label key (lowercased, spaces→dashes) -->
        <slot :name="`value-${slugify(item.label)}`">
          <OuiText :size="valueSize" weight="medium" :class="item.mono ? 'font-mono' : ''">
            {{ item.value ?? '—' }}
          </OuiText>
        </slot>
      </OuiStack>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

export interface KeyValueItem {
  label: string;
  value?: string | number | null;
  /** Render value in font-mono */
  mono?: boolean;
  /** Conditionally hide the item */
  hidden?: boolean;
}

const props = withDefaults(defineProps<{
  items: KeyValueItem[];
  /** Number of columns */
  cols?: 1 | 2 | 3 | 4;
  /** Value text size */
  valueSize?: "xs" | "sm" | "md";
}>(), {
  cols: 2,
  valueSize: "sm",
});

const colsClass: Record<number, string> = {
  1: "grid grid-cols-1 gap-3",
  2: "grid grid-cols-2 gap-3",
  3: "grid grid-cols-3 gap-3",
  4: "grid grid-cols-2 md:grid-cols-4 gap-3",
};

const gridClass = computed(() => colsClass[props.cols]);

function slugify(label: string): string {
  return label.toLowerCase().replace(/\s+/g, "-").replace(/[^a-z0-9-]/g, "");
}
</script>
