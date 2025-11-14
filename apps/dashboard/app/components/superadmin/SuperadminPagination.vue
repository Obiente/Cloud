<template>
  <OuiFlex
    v-if="pagination && pagination.totalPages > 1"
    align="center"
    justify="between"
    class="px-6 py-4 border-t border-border-muted"
  >
    <OuiText color="muted" size="sm">
      Page {{ pagination.page }} of {{ pagination.totalPages }}
      <span v-if="pagination.total !== undefined">
        ({{ pagination.total }} total)
      </span>
    </OuiText>
    <OuiFlex gap="sm">
      <OuiButton
        variant="ghost"
        size="sm"
        :disabled="pagination.page <= 1"
        @click="handlePrevious"
      >
        Previous
      </OuiButton>
      <OuiButton
        variant="ghost"
        size="sm"
        :disabled="pagination.page >= pagination.totalPages"
        @click="handleNext"
      >
        Next
      </OuiButton>
    </OuiFlex>
  </OuiFlex>
</template>

<script setup lang="ts">
export interface Pagination {
  page: number;
  perPage: number;
  totalPages: number;
  total?: number;
}

const props = defineProps<{
  pagination: Pagination | null;
}>();

const emit = defineEmits<{
  "page-change": [page: number];
}>();

const handlePrevious = () => {
  if (props.pagination && props.pagination.page > 1) {
    emit("page-change", props.pagination.page - 1);
  }
};

const handleNext = () => {
  if (props.pagination && props.pagination.page < props.pagination.totalPages) {
    emit("page-change", props.pagination.page + 1);
  }
};
</script>

