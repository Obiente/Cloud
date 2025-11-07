<template>
  <div 
    class="oui-table-wrapper overflow-x-auto -mx-4 md:mx-0"
    :class="wrapperClass"
  >
    <table 
      class="min-w-full text-left text-xs md:text-sm"
      :class="tableClass"
      :style="tableStyles"
      role="table"
      :aria-label="ariaLabel"
    >
      <thead>
        <tr>
          <th
            v-for="(column, index) in columns"
            :key="column.key || index"
            :class="[
              'px-3 md:px-6 py-2 md:py-3 font-medium relative text-xs md:text-sm',
              headerClass,
              column.headerClass,
              { 'cursor-pointer select-none': sortable && column.sortable !== false }
            ]"
            :style="getColumnStyle(column, index)"
            :scope="'col'"
            :aria-sort="getAriaSort(column)"
            @click="sortable && column.sortable !== false ? handleSort(column) : undefined"
          >
            <div class="flex items-center gap-2">
              <slot 
                :name="`header-${column.key}`" 
                :column="column"
                :sorted="sortedColumn?.key === column.key"
                :sortDirection="sortedColumn?.key === column.key ? sortDirection : null"
              >
                {{ column.label }}
              </slot>
              <span 
                v-if="sortable && column.sortable !== false && sortedColumn?.key === column.key"
                class="inline-flex items-center"
                aria-hidden="true"
              >
                <svg 
                  v-if="sortDirection === 'asc'"
                  class="h-4 w-4"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7" />
                </svg>
                <svg 
                  v-else-if="sortDirection === 'desc'"
                  class="h-4 w-4"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
              </span>
            </div>
            <div
              v-if="resizable && column.resizable !== false && index < columns.length - 1"
              class="absolute top-0 right-0 bottom-0 w-1 cursor-col-resize group z-10"
              :style="{ marginRight: '-2px' }"
              @mousedown.stop="handleResizeStart($event, index)"
              @dblclick.stop="handleResizeReset(index)"
              title="Drag to resize column"
            >
              <div class="absolute inset-y-0 left-1/2 w-0.5 -translate-x-1/2 bg-border-muted/30 group-hover:bg-border-primary transition-colors" />
            </div>
          </th>
        </tr>
      </thead>
      <tbody>
        <tr 
          v-for="(row, rowIndex) in sortedRows" 
          :key="getRowKey(row, rowIndex)"
          :class="[
            'border-t border-border-muted/60 transition-colors',
            rowClass,
            typeof rowClassFn === 'function' ? rowClassFn(row, rowIndex) : '',
            { 'cursor-pointer hover:bg-surface-subtle/50': clickable }
          ]"
          :aria-rowindex="rowIndex + 2"
          @click="clickable ? handleRowClick(row, rowIndex) : undefined"
        >
          <td
            v-for="(column, colIndex) in columns"
            :key="column.key || colIndex"
            :class="[
              'px-3 md:px-6 py-2 md:py-3',
              cellClass,
              column.cellClass
            ]"
            :style="getColumnStyle(column, colIndex)"
          >
            <slot 
              :name="`cell-${column.key}`" 
              :row="row" 
              :column="column"
              :value="getCellValue(row, column)"
              :index="rowIndex"
            >
              {{ getCellValue(row, column) }}
            </slot>
          </td>
        </tr>
        <tr v-if="!sortedRows.length">
          <td 
            :colspan="columns.length" 
            :class="['px-3 md:px-6 py-4 md:py-8 text-center text-text-muted text-xs md:text-sm', emptyClass]"
            role="status"
            :aria-live="'polite'"
          >
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
import { ref, computed, watch, onMounted, onUnmounted } from 'vue';

export interface TableColumn<T = any> {
  key: string;
  label: string;
  minWidth?: number;
  defaultWidth?: number;
  width?: number | string;
  headerClass?: string;
  cellClass?: string;
  resizable?: boolean;
  sortable?: boolean;
  sortFn?: (a: T, b: T) => number;
  accessor?: (row: T) => any;
}

export interface TableProps<T = any> {
  columns: TableColumn<T>[];
  rows: T[];
  emptyText?: string;
  headerClass?: string;
  rowClass?: string;
  rowClassFn?: (row: T, index: number) => string;
  cellClass?: string;
  emptyClass?: string;
  wrapperClass?: string;
  tableClass?: string;
  resizable?: boolean;
  sortable?: boolean;
  clickable?: boolean;
  ariaLabel?: string;
  rowKey?: string | ((row: T, index: number) => string | number);
}

export type SortDirection = 'asc' | 'desc' | null;

const emit = defineEmits<{
  "row-click": [row: any, index: number];
  "sort": [column: TableColumn, direction: SortDirection];
}>();

const props = withDefaults(defineProps<TableProps>(), {
  emptyText: 'No data available.',
  headerClass: 'bg-surface-subtle text-text-muted uppercase tracking-wide text-xs',
  resizable: true,
  sortable: false,
  clickable: false,
});

// Column resizing state
const columnWidths = ref<Map<number, number>>(new Map());
const isResizing = ref(false);
const resizeStartX = ref(0);
const resizeColumnIndex = ref<number | null>(null);
const resizeStartWidth = ref(0);

// Sorting state
const sortedColumn = ref<TableColumn | null>(null);
const sortDirection = ref<SortDirection>(null);

// Initialize column widths from defaultWidth or width props, or distribute evenly
const initializeColumnWidths = () => {
  if (!props.resizable) return;
  
  const hasExplicitWidths = props.columns.some(col => col.width || col.defaultWidth);
  
  if (hasExplicitWidths) {
    // Use explicit widths
    props.columns.forEach((column, index) => {
      if (column.width) {
        const width = typeof column.width === 'string' 
          ? parseFloat(column.width) 
          : column.width;
        columnWidths.value.set(index, width);
      } else if (column.defaultWidth) {
        columnWidths.value.set(index, column.defaultWidth);
      }
    });
  } else {
    // Distribute columns evenly - calculate widths that will space out properly
    // Estimate a reasonable container width (most tables are 1200-1600px wide)
    const estimatedContainerWidth = 1400;
    const minColumnWidth = 120; // Minimum width to ensure readability
    const columnCount = props.columns.length;
    
    // Calculate base width per column (distribute evenly)
    const baseWidthPerColumn = estimatedContainerWidth / columnCount;
    
    props.columns.forEach((column, index) => {
      // Use the calculated width, but respect minWidth
      const calculatedWidth = Math.max(
        column.minWidth || minColumnWidth,
        Math.floor(baseWidthPerColumn)
      );
      columnWidths.value.set(index, calculatedWidth);
    });
  }
};

onMounted(() => {
  initializeColumnWidths();
});

// Re-initialize when columns change
watch(() => props.columns, () => {
  initializeColumnWidths();
}, { deep: true });

// Column style computation
const getColumnStyle = (column: TableColumn, index: number): Record<string, string> => {
  const style: Record<string, string> = {};
  
  if (props.resizable) {
    // When resizable is enabled, always use fixed widths
    if (columnWidths.value.has(index)) {
      const width = columnWidths.value.get(index)!;
      style.width = `${width}px`;
      if (column.minWidth) {
        style.minWidth = `${column.minWidth}px`;
      }
    } else if (column.width) {
      // Fallback to explicit width
      style.width = typeof column.width === 'string' ? column.width : `${column.width}px`;
      if (column.minWidth) {
        style.minWidth = `${column.minWidth}px`;
      }
    } else if (column.defaultWidth) {
      style.width = `${column.defaultWidth}px`;
      if (column.minWidth) {
        style.minWidth = `${column.minWidth}px`;
      }
    } else if (column.minWidth) {
      style.minWidth = `${column.minWidth}px`;
    }
  } else {
    // When not resizable, use explicit widths if provided
    if (column.width) {
      style.width = typeof column.width === 'string' ? column.width : `${column.width}px`;
      if (column.minWidth) {
        style.minWidth = `${column.minWidth}px`;
      }
    } else if (column.defaultWidth) {
      style.width = `${column.defaultWidth}px`;
      if (column.minWidth) {
        style.minWidth = `${column.minWidth}px`;
      }
    } else if (column.minWidth) {
      style.minWidth = `${column.minWidth}px`;
    }
  }
  
  return style;
};

// Table styles
const tableStyles = computed(() => {
  // Always use fixed layout when resizable is enabled for proper column spacing
  // Also use fixed layout if any column has explicit width
  const hasExplicitWidths = props.columns.some(col => col.width || col.defaultWidth);
  if (props.resizable || hasExplicitWidths) {
    return { tableLayout: 'fixed' as const, width: '100%' } as const;
  }
  return undefined;
});

// Resize handlers
const handleResizeStart = (event: MouseEvent, columnIndex: number) => {
  if (!props.resizable) return;
  
  const column = props.columns[columnIndex];
  if (!column) return;
  
  event.preventDefault();
  isResizing.value = true;
  resizeStartX.value = event.clientX;
  resizeColumnIndex.value = columnIndex;
  
  const currentWidth = columnWidths.value.get(columnIndex) || column.defaultWidth || 150;
  resizeStartWidth.value = currentWidth;
  
  document.addEventListener('mousemove', handleResizeMove);
  document.addEventListener('mouseup', handleResizeEnd);
  document.body.style.cursor = 'col-resize';
  document.body.style.userSelect = 'none';
};

const handleResizeMove = (event: MouseEvent) => {
  if (!isResizing.value || resizeColumnIndex.value === null) return;
  
  const column = props.columns[resizeColumnIndex.value];
  if (!column) return;
  
  const deltaX = event.clientX - resizeStartX.value;
  const newWidth = Math.max(
    column.minWidth || 50,
    resizeStartWidth.value + deltaX
  );
  
  columnWidths.value.set(resizeColumnIndex.value, newWidth);
};

const handleResizeEnd = () => {
  isResizing.value = false;
  resizeColumnIndex.value = null;
  document.removeEventListener('mousemove', handleResizeMove);
  document.removeEventListener('mouseup', handleResizeEnd);
  document.body.style.cursor = '';
  document.body.style.userSelect = '';
};

const handleResizeReset = (columnIndex: number) => {
  const column = props.columns[columnIndex];
  if (!column) return;
  
  if (column.defaultWidth) {
    columnWidths.value.set(columnIndex, column.defaultWidth);
  } else {
    columnWidths.value.delete(columnIndex);
  }
};

onUnmounted(() => {
  document.removeEventListener('mousemove', handleResizeMove);
  document.removeEventListener('mouseup', handleResizeEnd);
});

// Sorting
const handleSort = (column: TableColumn) => {
  if (!props.sortable || column.sortable === false) return;
  
  if (sortedColumn.value?.key === column.key) {
    // Cycle: asc -> desc -> null
    if (sortDirection.value === 'asc') {
      sortDirection.value = 'desc';
    } else if (sortDirection.value === 'desc') {
      sortDirection.value = null;
      sortedColumn.value = null;
    }
  } else {
    sortedColumn.value = column;
    sortDirection.value = 'asc';
  }
  
  emit('sort', column, sortDirection.value);
};

const getAriaSort = (column: TableColumn): 'ascending' | 'descending' | 'none' | undefined => {
  if (!props.sortable || column.sortable === false) return undefined;
  if (sortedColumn.value?.key !== column.key) return 'none';
  return sortDirection.value === 'asc' ? 'ascending' : 'descending';
};

const sortedRows = computed(() => {
  if (!props.sortable || !sortedColumn.value || !sortDirection.value) {
    return props.rows;
  }
  
  const column = sortedColumn.value;
  const direction = sortDirection.value;
  const rows = [...props.rows];
  
  rows.sort((a, b) => {
    let comparison = 0;
    
    if (column.sortFn) {
      comparison = column.sortFn(a, b);
    } else {
      const aValue = getCellValue(a, column);
      const bValue = getCellValue(b, column);
      
      if (aValue === bValue) {
        comparison = 0;
      } else if (aValue == null) {
        comparison = 1;
      } else if (bValue == null) {
        comparison = -1;
      } else if (typeof aValue === 'string' && typeof bValue === 'string') {
        comparison = aValue.localeCompare(bValue);
      } else if (typeof aValue === 'number' && typeof bValue === 'number') {
        comparison = aValue - bValue;
      } else {
        comparison = String(aValue).localeCompare(String(bValue));
      }
    }
    
    return direction === 'asc' ? comparison : -comparison;
  });
  
  return rows;
});

// Cell value accessor
const getCellValue = (row: any, column: TableColumn): any => {
  if (column.accessor) {
    return column.accessor(row);
  }
  return row[column.key];
};

// Row key generation
const getRowKey = (row: any, index: number): string | number => {
  if (props.rowKey) {
    if (typeof props.rowKey === 'function') {
      return props.rowKey(row, index);
    }
    return row[props.rowKey];
  }
  return index;
};

// Row click handler
const handleRowClick = (row: any, index: number) => {
  if (props.clickable) {
    emit('row-click', row, index);
  }
};
</script>

<style scoped>
.oui-table-wrapper {
  position: relative;
}

/* Ensure proper table layout */
.oui-table-wrapper table {
  border-collapse: collapse;
}

/* Resize handle styling - make it more visible */
.oui-table-wrapper :deep(th) .group {
  transition: background-color 0.15s;
}

.oui-table-wrapper :deep(th) .group:hover {
  background-color: rgba(0, 0, 0, 0.02);
}

.oui-table-wrapper :deep(th) .group .bg-border-primary {
  background-color: hsl(var(--oui-border-primary, 214 32% 91%));
}

.oui-table-wrapper :deep(th) .group:hover .bg-border-primary {
  background-color: hsl(var(--oui-accent-primary, 221 83% 53%));
  opacity: 0.8;
}

.oui-table-wrapper :deep(th) .group:active .bg-border-primary {
  background-color: hsl(var(--oui-accent-primary, 221 83% 53%));
  opacity: 1;
  width: 2px;
}
</style>
