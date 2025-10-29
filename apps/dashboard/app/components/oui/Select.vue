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

    <Select.Root :collection="collection" :disabled="disabled" v-bind="$attrs">
      <Select.Control class="relative">
        <Select.Trigger :class="triggerClasses" :disabled="disabled">
          <Select.ValueText
            :placeholder="placeholder || 'Select an option...'"
            class="truncate"
          />
          <Select.Indicator class="ml-2 flex-shrink-0">
            <ChevronUpDownIcon class="h-4 w-4 text-secondary" />
          </Select.Indicator>
        </Select.Trigger>
      </Select.Control>

      <Teleport to="body">
        <Select.Positioner class="w-[--reference-width]">
          <Select.Content
            class="z-50 min-w-[8rem] w-[--reference-width] overflow-hidden rounded-md border border-border-default bg-surface-base p-1 shadow-md animate-in data-[side=bottom]:slide-in-from-top-2"
          >
            <Select.ItemGroup>
              <Select.Item
                v-for="item in collection.items"
                :key="item.value"
                :item="item"
                class="relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm text-text-primary outline-none hover:bg-surface-muted focus:bg-surface-muted data-[disabled]:pointer-events-none data-[disabled]:opacity-50"
              >
                <span
                  class="absolute left-2 flex h-3.5 w-3.5 items-center justify-center"
                >
                  <Select.ItemIndicator>
                    <CheckIcon class="h-4 w-4 text-primary" />
                  </Select.ItemIndicator>
                </span>
                <Select.ItemText>{{ item.label }}</Select.ItemText>
              </Select.Item>
            </Select.ItemGroup>
          </Select.Content>
        </Select.Positioner>
      </Teleport>

      <Select.HiddenSelect />
    </Select.Root>

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
  import {
    Select,
    SelectItem,
    createListCollection,
    useSelect,
  } from "@ark-ui/vue/select";
  import { Field } from "@ark-ui/vue/field";
  import { CheckIcon, ChevronUpDownIcon } from "@heroicons/vue/24/outline";

  interface SelectItem {
    label: string;
    value: string | number;
    disabled?: boolean;
  }

  interface Props {
    label?: string;
    placeholder?: string;
    items: SelectItem[];
    helperText?: string;
    error?: string;
    required?: boolean;
    disabled?: boolean;
    size?: "sm" | "md" | "lg";
  }

  const props = withDefaults(defineProps<Props>(), {
    size: "md",
    required: false,
    disabled: false,
  });

  const collection = createListCollection({
    items: props.items,
  });

  const select = useSelect({
    collection: collection,
    multiple: false,
  });
  const triggerClasses = computed(() => [
    "oui-input",
    `oui-input-${props.size}`,
    props.error ? "oui-input-error" : "oui-input-base",
    "w-full flex items-center justify-between text-left",
  ]);

  defineModel<string | string[]>({
    get: () => select.value.value,
    set: (val) => {
      if (!Array.isArray(val)) {
        val = [val];
      }
      select.value.setValue(val);
    },
  });

  defineOptions({
    inheritAttrs: false,
  });
</script>
