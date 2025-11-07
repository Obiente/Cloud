<template>
  <span>{{ formattedDate }}</span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { formatDate, formatDateOnly } from "~/utils/common";

interface Props {
  value: any; // Timestamp object with seconds/nanos or Date
  dateOnly?: boolean; // If true, show only date without time
}

const props = withDefaults(defineProps<Props>(), {
  dateOnly: false,
});

const formattedDate = computed(() => {
  if (props.dateOnly) {
    return formatDateOnly(props.value);
  }
  return formatDate(props.value);
});
</script>

