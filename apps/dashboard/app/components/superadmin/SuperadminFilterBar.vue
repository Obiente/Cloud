<template>
  <OuiFlex gap="sm" wrap="wrap" align="center">
    <div v-if="showSearch" class="w-72 max-w-full">
      <OuiInput
        v-model="searchValue"
        type="search"
        :placeholder="searchPlaceholder"
        clearable
        size="sm"
        @update:model-value="handleSearchChange"
      />
    </div>
    
    <template v-for="(filter, index) in filters" :key="index">
      <div :class="filter.class || 'min-w-[160px]'">
        <OuiSelect
          v-model="filterValues[filter.key]"
          :items="filter.items"
          :placeholder="filter.placeholder"
          size="sm"
          @update:model-value="handleFilterChange(filter.key, $event)"
        />
      </div>
    </template>
    
    <OuiButton 
      v-if="showRefresh"
      variant="ghost" 
      size="sm" 
      @click="handleRefresh" 
      :disabled="loading"
    >
      <span class="flex items-center gap-2">
        <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': loading }" />
        Refresh
      </span>
    </OuiButton>
    
    <slot />
  </OuiFlex>
</template>

<script setup lang="ts">
import { ArrowPathIcon } from "@heroicons/vue/24/outline";
import { ref, watch } from "vue";

export interface FilterOption {
  key: string;
  label: string;
  value: string;
}

export interface FilterConfig {
  key: string;
  placeholder: string;
  items: FilterOption[];
  class?: string;
}

const props = withDefaults(defineProps<{
  search?: string;
  filters?: FilterConfig[];
  showSearch?: boolean;
  showRefresh?: boolean;
  searchPlaceholder?: string;
  loading?: boolean;
}>(), {
  search: "",
  filters: () => [],
  showSearch: true,
  showRefresh: true,
  searchPlaceholder: "Search…",
  loading: false,
});

const emit = defineEmits<{
  "update:search": [value: string];
  "filter-change": [key: string, value: string];
  "refresh": [];
}>();

const searchValue = ref(props.search);
const filterValues = ref<Record<string, string>>({});

// Initialize filter values
props.filters.forEach((filter) => {
  const defaultItem = filter.items.find((item) => item.value === "all" || item.value === "");
  filterValues.value[filter.key] = defaultItem?.value || filter.items[0]?.value || "";
});

watch(() => props.search, (newValue) => {
  if (searchValue.value !== newValue) {
    searchValue.value = newValue;
  }
});

const handleSearchChange = (value: string) => {
  emit("update:search", value);
};

const handleFilterChange = (key: string, value: string) => {
  filterValues.value[key] = value;
  emit("filter-change", key, value);
};

const handleRefresh = () => {
  emit("refresh");
};
</script>

