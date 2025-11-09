<template>
  <Accordion.Root
    :modelValue="modelValue"
    :multiple="multiple"
    :collapsible="collapsible"
    :disabled="disabled"
    @update:modelValue="handleValueChange"
    class="w-full space-y-2"
  >
    <Accordion.Item
      v-for="item in items"
      :key="item.value"
      :value="item.value"
      :disabled="item.disabled"
      class="border border-border-muted rounded-lg overflow-hidden"
    >
      <Accordion.ItemTrigger class="flex items-center justify-between w-full px-4 py-3 text-left font-medium text-primary hover:bg-background-muted transition-colors">
        <slot name="trigger" :item="item">
          {{ item.label }}
        </slot>
        <Accordion.ItemIndicator class="shrink-0 ml-2">
          <ChevronDownIcon 
            :class="[
              'h-4 w-4 transition-transform duration-200',
              isItemOpen(item.value) ? 'rotate-180' : ''
            ]" 
          />
        </Accordion.ItemIndicator>
      </Accordion.ItemTrigger>
      <Accordion.ItemContent class="px-4 pb-3 text-secondary">
        <slot name="content" :item="item">
          {{ item.content }}
        </slot>
      </Accordion.ItemContent>
    </Accordion.Item>
  </Accordion.Root>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { Accordion } from "@ark-ui/vue/accordion";
import { ChevronDownIcon } from "@heroicons/vue/24/outline";
import type { Component } from "vue";

interface AccordionItem {
  value: string;
  label: string;
  content?: string;
  disabled?: boolean;
  icon?: Component;
}

interface Props {
  items: AccordionItem[];
  modelValue?: string[];
  multiple?: boolean;
  collapsible?: boolean;
  disabled?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  multiple: false,
  collapsible: true,
  disabled: false,
});

const emit = defineEmits<{
  "update:modelValue": [value: string[]];
}>();

const openValues = ref<string[]>(props.modelValue || []);

watch(() => props.modelValue, (newValue) => {
  openValues.value = newValue || [];
}, { immediate: true });

const handleValueChange = (value: string[]) => {
  openValues.value = value;
  emit("update:modelValue", value);
};

const isItemOpen = (itemValue: string) => {
  return openValues.value.includes(itemValue);
};
</script>

