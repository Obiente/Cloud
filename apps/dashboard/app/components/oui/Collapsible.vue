<template>
  <Collapsible.Root
    :open="modelValue"
    :disabled="disabled"
    @update:open="handleOpenChange"
    class="w-full"
  >
    <Collapsible.Trigger class="flex items-center justify-between w-full px-4 py-3 text-left font-medium text-primary hover:bg-background-muted transition-colors rounded-lg border border-border-muted">
      <slot name="trigger">
        {{ label }}
      </slot>
      <Collapsible.Indicator class="shrink-0 ml-2">
        <ChevronDownIcon class="h-4 w-4 transition-transform data-[open]:rotate-180" />
      </Collapsible.Indicator>
    </Collapsible.Trigger>
    <Collapsible.Content class="px-4 py-3 text-secondary border-x border-b border-border-muted rounded-b-lg">
      <slot />
    </Collapsible.Content>
  </Collapsible.Root>
</template>

<script setup lang="ts">
import { Collapsible } from "@ark-ui/vue/collapsible";
import { ChevronDownIcon } from "@heroicons/vue/24/outline";

interface Props {
  modelValue?: boolean;
  label?: string;
  disabled?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: false,
  disabled: false,
});

const emit = defineEmits<{
  "update:modelValue": [open: boolean];
}>();

const handleOpenChange = (open: boolean) => {
  emit("update:modelValue", open);
};
</script>

