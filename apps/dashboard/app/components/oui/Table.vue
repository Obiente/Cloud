<template>
  <div class="oui-table-wrapper overflow-x-auto relative">
    <!-- Splitter overlay for resizing - positioned over header row -->
    <div class="absolute top-0 left-0 right-0 pointer-events-none z-10" style="height: calc(1.5rem + 1.5rem + 1px);">
      <Splitter.Root
        :panels="splitterPanels"
        :default-size="defaultSizes"
        orientation="horizontal"
        class="w-full h-full"
        @resize-end="handleResizeEnd"
        v-model:size="panelSizes"
      >
        <div class="flex w-full h-full">
          <template v-for="(column, index) in columns" :key="index">
            <Splitter.Panel :id="String(index)" class="shrink-0 pointer-events-none" :style="{ minWidth: (column.minWidth || 50) + 'px' }" />
            <Splitter.ResizeTrigger
              v-if="column.resizable !== false && index < columns.length - 1"
              :id="`${index}:${index + 1}`"
              class="pointer-events-auto shrink-0"
              aria-label="Resize column"
            />
          </template>
        </div>
      </Splitter.Root>
    </div>
    
    <!-- Actual table rendered with widths from Splitter -->
    <table class="min-w-full text-left text-sm" :style="{ tableLayout: 'fixed', width: '100%' }">
      <thead>
        <tr>
          <th
            v-for="(column, index) in columns"
            :key="index"
            :class="[
              'px-6 py-3 relative',
              headerClass,
              column.headerClass
            ]"
            :style="{ width: panelSizes[index] + '%', minWidth: (column.minWidth || 50) + 'px' }"
          >
            <slot :name="`header-${column.key}`" :column="column">
              {{ column.label }}
            </slot>
          </th>
        </tr>
      </thead>
      <tbody>
        <tr 
          v-for="(row, rowIndex) in rows" 
          :key="rowIndex"
          :class="[
            'border-t border-border-muted/60',
            rowClass,
            typeof rowClassFn === 'function' ? rowClassFn(row, rowIndex) : ''
          ]"
          @click="handleRowClick(row, rowIndex)"
        >
          <td
            v-for="(column, colIndex) in columns"
            :key="colIndex"
            :class="[
              'px-6 py-3',
              cellClass,
              column.cellClass
            ]"
            :style="{ width: panelSizes[colIndex] + '%', minWidth: (column.minWidth || 50) + 'px' }"
          >
            <slot 
              :name="`cell-${column.key}`" 
              :row="row" 
              :column="column"
              :value="row[column.key]"
            >
              {{ row[column.key] }}
            </slot>
          </td>
        </tr>
        <tr v-if="!rows.length">
          <td :colspan="columns.length" :class="['px-6 py-8 text-center text-text-muted', emptyClass]">
            <slot name="empty">
              {{ emptyText }}
            </slot>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { Splitter } from '@ark-ui/vue/splitter';

interface Column {
  key: string;
  label: string;
  minWidth?: number;
  defaultWidth?: number;
  headerClass?: string;
  cellClass?: string;
  resizable?: boolean;
}

interface Props {
  columns: Column[];
  rows: Record<string, any>[];
  emptyText?: string;
  headerClass?: string;
  rowClass?: string;
  rowClassFn?: (row: any, index: number) => string;
  cellClass?: string;
  emptyClass?: string;
}

const emit = defineEmits<{
  "row-click": [row: Record<string, any>, index: number];
}>();

function handleRowClick(row: Record<string, any>, index: number) {
  emit("row-click", row, index);
}

const props = withDefaults(defineProps<Props>(), {
  emptyText: 'No data available.',
  headerClass: 'bg-surface-subtle text-text-muted uppercase tracking-wide',
});

// Generate panel configuration for Splitter
const splitterPanels = computed(() => {
  return props.columns.map((col, index) => ({
    id: String(index),
    minSize: col.minWidth ? (col.minWidth / 1000 * 100) : 5, // Convert px to percentage estimate
  }));
});

// Calculate default sizes as percentages
const defaultSizes = computed(() => {
  const totalWidth = props.columns.reduce((sum, col) => sum + (col.defaultWidth || 150), 0);
  return props.columns.map(col => {
    const width = col.defaultWidth || 150;
    return (width / totalWidth) * 100;
  });
});

// Track panel sizes from Splitter
const panelSizes = ref<number[]>(defaultSizes.value);

// Handle resize end to update panel sizes
const handleResizeEnd = (details: { size?: number[] }) => {
  if (details.size) {
    panelSizes.value = [...details.size];
  }
};

// Watch for changes in default sizes
watch(defaultSizes, (newSizes) => {
  if (!panelSizes.value || panelSizes.value.length !== newSizes.length || 
      panelSizes.value.some((size, i) => {
        const newSize = newSizes[i];
        return newSize !== undefined && Math.abs(size - newSize) > 0.1;
      })) {
    panelSizes.value = [...newSizes];
  }
}, { immediate: true });
</script>

<style scoped>
.oui-table-wrapper {
  position: relative;
}

/* Style the splitter resize trigger to look good in table context */
:deep([data-part="resize-trigger"]) {
  cursor: col-resize;
  width: 4px;
  margin-left: -2px;
  margin-right: -2px;
  background: transparent;
  transition: background-color 0.15s;
  position: relative;
}

:deep([data-part="resize-trigger"]::before) {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  left: 50%;
  transform: translateX(-50%);
  width: 2px;
  background: transparent;
  transition: background-color 0.15s;
}

:deep([data-part="resize-trigger"]:hover::before),
:deep([data-part="resize-trigger"][data-state="dragging"]::before) {
  background-color: var(--oui-accent-primary);
  opacity: 0.6;
}

:deep([data-part="resize-trigger"][data-state="dragging"]::before) {
  opacity: 1;
}

/* Ensure panels don't interfere with table layout */
:deep([data-part="panel"]) {
  min-width: 0;
  pointer-events: none;
}
</style>

