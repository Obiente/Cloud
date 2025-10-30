<template>
  <Field.Root
    :invalid="!!error"
    :required="required"
    :disabled="disabled"
    class="oui-field space-y-1 w-full"
  >
    <Field.Label v-if="label" class="block text-sm font-medium text-primary">
      {{ label }}
    </Field.Label>

    <Combobox.Root
      :collection="collection"
      :disabled="disabled"
      :value="comboboxValue"
      v-bind="$attrs"
      @input-value-change="handleInputChange"
      @value-change="handleValueChange"
    >
      <Combobox.Control class="relative">
        <Combobox.Input
          :placeholder="placeholder"
          :disabled="disabled"
          :class="[inputClasses, 'pr-10']"
        />
        <Combobox.Trigger
          class="absolute inset-y-0 right-0 flex items-center pr-2"
        >
          <ChevronDownIcon class="h-5 w-5 text-secondary" />
        </Combobox.Trigger>
        <Combobox.ClearTrigger
          v-if="showClear"
          class="absolute inset-y-0 right-8 flex items-center pr-2"
        >
          <XMarkIcon
            class="h-4 w-4 text-secondary hover:text-primary cursor-pointer"
          />
        </Combobox.ClearTrigger>
      </Combobox.Control>

      <Teleport to="body">
        <Combobox.Positioner class="w-[--reference-width]">
          <Combobox.Content
            class="z-50 min-w-[8rem] w-[--reference-width] overflow-hidden rounded-md border border-border-default bg-surface-base p-1 shadow-md animate-in data-[side=bottom]:slide-in-from-top-2"
          >
            <Combobox.ItemGroup>
              <Combobox.Item
                v-for="item in collection.items"
                :key="typeof item === 'string' ? item : item.value"
                :item="item"
                class="relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 px-2 text-sm text-text-primary outline-none hover:bg-surface-muted focus:bg-surface-muted data-[disabled]:pointer-events-none data-[disabled]:opacity-50"
              >
                <Combobox.ItemText>
                  {{ typeof item === "string" ? item : item.label }}
                </Combobox.ItemText>
                <Combobox.ItemIndicator class="ml-auto">
                  <CheckIcon class="h-4 w-4" />
                </Combobox.ItemIndicator>
              </Combobox.Item>
            </Combobox.ItemGroup>

            <div
              v-if="collection.items.length === 0"
              class="px-3 py-2 text-sm text-text-secondary"
            >
              No results found
            </div>
          </Combobox.Content>
        </Combobox.Positioner>
      </Teleport>
    </Combobox.Root>

    <Field.ErrorText v-if="error" class="text-sm text-danger">
      {{ error }}
    </Field.ErrorText>
    <Field.HelperText v-else-if="helperText" class="text-sm text-secondary">
      {{ helperText }}
    </Field.HelperText>
  </Field.Root>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { Combobox, useListCollection } from "@ark-ui/vue/combobox";
import { Field } from "@ark-ui/vue/field";
import { useFilter } from "@ark-ui/vue/locale";
import {
  ChevronDownIcon,
  CheckIcon,
  XMarkIcon,
} from "@heroicons/vue/24/outline";

interface Option {
  label: string;
  value: string;
}

interface Props {
  modelValue?: string;
  placeholder?: string;
  label?: string;
  options: (string | Option)[];
  showClear?: boolean;
  helperText?: string;
  error?: string;
  required?: boolean;
  disabled?: boolean;
  size?: "sm" | "md" | "lg";
}

const props = withDefaults(defineProps<Props>(), {
  showClear: true,
  size: "md",
  required: false,
  disabled: false,
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

const filters = useFilter({ sensitivity: "base" });

const { collection, filter } = useListCollection({
  initialItems: props.options,
  filter: filters.value.contains,
});

// Controlled value for Ark UI Combobox (expects an array)
const comboboxValue = computed(() => {
  if (!props.modelValue) return [] as (string | Option)[];
  const match = props.options.find((opt) =>
    typeof opt === "string"
      ? opt === props.modelValue
      : opt.value === props.modelValue
  );
  return match ? [match] : [];
});

const inputClasses = computed(() => [
  "oui-input",
  `oui-input-${props.size}`,
  props.error ? "oui-input-error" : "oui-input-base",
]);

const handleInputChange = (details: Combobox.InputValueChangeDetails) => {
  filter(details.inputValue);
};

const handleValueChange = (details: Combobox.ValueChangeDetails) => {
  const value = details.value[0];
  if (!value) {
    emit("update:modelValue", "");
    return;
  }
  if (typeof value === "string") {
    emit("update:modelValue", value);
  } else if (typeof value === "object" && "value" in value) {
    emit("update:modelValue", (value as Option).value);
  }
};
</script>
