<template>
  <OuiStack gap="xl">
    <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
      <SuperadminPageHeader :title="title" :description="description" />
      <SuperadminFilterBar
        :search="searchModel"
        :filters="filters"
        :show-search="showSearch"
        :show-refresh="showRefresh"
        :search-placeholder="searchPlaceholder"
        :loading="loading"
        @update:search="handleSearchUpdate"
        @filter-change="handleFilterChange"
        @refresh="handleRefresh"
      >
        <slot name="filters" />
      </SuperadminFilterBar>
    </OuiFlex>

    <SuperadminTable
      :columns="columns"
      :rows="rows"
      :empty-text="emptyText"
      :row-class="rowClass"
      :clickable="clickable"
      :loading="loading"
      @row-click="handleRowClick"
    >
      <template v-for="(_, name) in $slots" :key="name" #[name]="slotData">
        <slot :name="name" v-bind="slotData || {}" />
      </template>
    </SuperadminTable>

    <SuperadminPagination
      v-if="pagination"
      :pagination="pagination"
      @page-change="handlePageChange"
    />
  </OuiStack>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import type { TableColumn } from "~/components/oui/Table.vue";
import type { FilterConfig } from "./SuperadminFilterBar.vue";
import type { Pagination } from "./SuperadminPagination.vue";
import SuperadminPageHeader from "./SuperadminPageHeader.vue";
import SuperadminFilterBar from "./SuperadminFilterBar.vue";
import SuperadminTable from "./SuperadminTable.vue";
import SuperadminPagination from "./SuperadminPagination.vue";

const props = withDefaults(defineProps<{
  title: string;
  description?: string;
  columns: TableColumn[];
  rows: any[];
  filters?: FilterConfig[];
  pagination?: Pagination | null;
  search?: string;
  showSearch?: boolean;
  showRefresh?: boolean;
  searchPlaceholder?: string;
  emptyText?: string;
  rowClass?: string;
  clickable?: boolean;
  loading?: boolean;
}>(), {
  description: "",
  filters: () => [],
  pagination: null,
  search: "",
  showSearch: true,
  showRefresh: true,
  searchPlaceholder: "Search…",
  emptyText: "No items found.",
  rowClass: "hover:bg-surface-subtle/50 transition-colors cursor-pointer",
  clickable: true,
  loading: false,
});

const emit = defineEmits<{
  "update:search": [value: string];
  "filter-change": [key: string, value: string];
  "refresh": [];
  "row-click": [row: any, index: number];
  "page-change": [page: number];
}>();

const searchModel = ref(props.search);

// Sync search model with external changes
watch(() => props.search, (newValue) => {
  searchModel.value = newValue;
});

const handleSearchUpdate = (value: string) => {
  searchModel.value = value;
  emit("update:search", value);
};

const handleFilterChange = (key: string, value: string) => {
  emit("filter-change", key, value);
};

const handleRefresh = () => {
  emit("refresh");
};

const handleRowClick = (row: any, index: number) => {
  emit("row-click", row, index);
};

const handlePageChange = (page: number) => {
  emit("page-change", page);
};
</script>

