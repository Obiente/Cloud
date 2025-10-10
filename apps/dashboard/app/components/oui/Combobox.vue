<template>
  <Combobox.Root
    :collection="collection"
    @input-value-change="handleInputChange"
    @value-change="handleValueChange"
  >
    <Combobox.Label
      v-if="label"
      class="block text-sm font-medium text-primary mb-1"
    >
      {{ label }}
    </Combobox.Label>

    <Combobox.Control class="relative">
      <Combobox.Input :placeholder="placeholder" class="oui-input-base pr-10" />
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
      <Combobox.Positioner>
        <Combobox.Content
          class="card-base max-h-60 overflow-auto z-50 animate-in fade-in-0 zoom-in-95 duration-200 data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=closed]:zoom-out-95"
        >
          <Combobox.ItemGroup>
            <Combobox.Item
              v-for="item in collection.items"
              :key="typeof item === 'string' ? item : item.value"
              :item="item"
              class="relative flex items-center px-3 py-2 text-sm cursor-pointer select-none text-primary hover:bg-hover data-[highlighted]:bg-hover data-[highlighted]:text-primary data-[state=checked]:bg-primary data-[state=checked]:text-white"
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
</template>

<script setup lang="ts">
import { Combobox, useListCollection } from "@ark-ui/vue/combobox";
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
}

const props = withDefaults(defineProps<Props>(), {
  showClear: false,
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

const filters = useFilter({ sensitivity: "base" });

const { collection, filter } = useListCollection({
  initialItems: props.options,
  filter: filters.value.contains,
});

const handleInputChange = (details: Combobox.InputValueChangeDetails) => {
  filter(details.inputValue);
};

const handleValueChange = (details: Combobox.ValueChangeDetails) => {
  const value = details.value[0];
  if (typeof value === "string") {
    emit("update:modelValue", value);
  } else if (value && typeof value === "object" && "value" in value) {
    emit("update:modelValue", (value as Option).value);
  }
};
</script>
