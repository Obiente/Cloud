<template>
  <Pagination.Root
    :count="count"
    :page="page"
    :page-size="pageSize"
    :sibling-count="siblingCount"
    @update:page="handlePageChange"
    class="flex items-center gap-1"
  >
    <Pagination.PrevTrigger
      class="flex items-center justify-center h-9 w-9 rounded-lg border border-border-default bg-surface-base text-text-secondary transition-colors duration-150 hover:bg-surface-raised hover:text-primary disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:bg-surface-base disabled:hover:text-text-secondary"
    >
      <ChevronLeftIcon class="h-4 w-4" />
      <span class="sr-only">Previous Page</span>
    </Pagination.PrevTrigger>

    <Pagination.Context v-slot="pagination">
      <template v-for="(pageItem, index) in pagination.pages" :key="index">
        <Pagination.Item
          v-if="pageItem.type === 'page'"
          :value="pageItem.value"
          :type="pageItem.type"
          class="flex items-center justify-center min-w-[36px] h-9 px-3 rounded-lg border border-border-default bg-surface-base text-text-secondary font-medium transition-colors duration-150 hover:bg-surface-raised hover:text-primary data-[selected]:bg-primary data-[selected]:text-primary-foreground data-[selected]:border-primary"
          :data-selected="pageItem.value === page || undefined"
        >
          {{ pageItem.value }}
        </Pagination.Item>
        <Pagination.Ellipsis
          v-else
          :index="index"
          class="flex items-center justify-center h-9 px-2 text-text-secondary"
        >
          &#8230;
        </Pagination.Ellipsis>
      </template>
    </Pagination.Context>

    <Pagination.NextTrigger
      class="flex items-center justify-center h-9 w-9 rounded-lg border border-border-default bg-surface-base text-text-secondary transition-colors duration-150 hover:bg-surface-raised hover:text-primary disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:bg-surface-base disabled:hover:text-text-secondary"
    >
      <ChevronRightIcon class="h-4 w-4" />
      <span class="sr-only">Next Page</span>
    </Pagination.NextTrigger>
  </Pagination.Root>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Pagination } from '@ark-ui/vue/pagination'
import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/vue/24/outline'

interface Props {
  count: number // Total number of items
  page: number // Current page (1-based)
  pageSize?: number // Items per page
  siblingCount?: number // Number of pages to show beside active page
}

const props = withDefaults(defineProps<Props>(), {
  pageSize: 10,
  siblingCount: 1,
})

const emit = defineEmits<{
  'update:page': [page: number]
  'page-change': [page: number]
}>()

const totalPages = computed(() => Math.ceil(props.count / props.pageSize))

const handlePageChange = (page: number) => {
  emit('update:page', page)
  emit('page-change', page)
}
</script>

