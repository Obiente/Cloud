<template>
  <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
    <OuiCardBody class="p-0">
      <OuiTable
        wrapper-class="mx-0!"
        :columns="columns"
        :rows="rows"
        :empty-text="emptyText"
        :row-class="rowClass"
        :clickable="isClickable"
        :loading="loading"
        @row-click="handleRowClick"
      >
        <template v-for="(_, name) in $slots" :key="name" #[name]="slotData">
          <slot :name="name" v-bind="slotData || {}" />
        </template>
      </OuiTable>
    </OuiCardBody>
  </OuiCard>
</template>

<script setup lang="ts">
import type { TableColumn } from "~/components/oui/Table.vue";

const props = withDefaults(defineProps<{
  columns: TableColumn[];
  rows: any[];
  emptyText?: string;
  rowClass?: string;
  clickable?: boolean;
  loading?: boolean;
}>(), {
  emptyText: "No items found.",
  rowClass: "transition-colors",

  clickable: undefined,
  loading: false,
});

const instance = getCurrentInstance();
const isClickable = computed(() => props.clickable ?? Boolean(instance?.vnode.props?.onRowClick));

const emit = defineEmits<{
  "row-click": [row: any, index: number];
}>();

const handleRowClick = (row: any, index: number) => {
  emit("row-click", row, index);
};
</script>

