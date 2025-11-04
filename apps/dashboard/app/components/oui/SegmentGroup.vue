<template>
  <SegmentGroup.Root
    :modelValue="modelValue"
    :disabled="disabled"
    @update:modelValue="handleValueChange"
    class="inline-flex rounded-lg bg-border-muted p-1"
  >
    <SegmentGroup.Indicator class="absolute rounded-md bg-background shadow-sm transition-all duration-200" />
    <SegmentGroup.Item
      v-for="option in options"
      :key="option.value"
      :value="option.value"
      :disabled="option.disabled"
      class="relative px-4 py-2 rounded-md transition-colors cursor-pointer data-[hover]:bg-background/50 data-[checked]:text-primary data-[checked]:font-medium data-[disabled]:opacity-50 data-[disabled]:cursor-not-allowed"
    >
      <SegmentGroup.ItemText class="text-sm font-medium">
        {{ option.label }}
      </SegmentGroup.ItemText>
      <SegmentGroup.ItemControl />
      <SegmentGroup.ItemHiddenInput />
    </SegmentGroup.Item>
  </SegmentGroup.Root>
</template>

<script setup lang="ts">
import { SegmentGroup } from "@ark-ui/vue/segment-group";

interface Option {
  label: string;
  value: string;
  disabled?: boolean;
}

interface Props {
  modelValue: string;
  options: Option[];
  disabled?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

const handleValueChange = (value: string | null) => {
  if (value !== null) {
    emit("update:modelValue", value);
  }
};
</script>

