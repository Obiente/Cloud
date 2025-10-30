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

    <Select.Root :collection="collection" :disabled="disabled" v-model="inner">
      <Select.Control class="relative">
        <Select.Trigger :class="triggerClasses" :disabled="disabled">
          <Select.ValueText
            :placeholder="placeholder || 'Select an option...'"
            class="truncate"
          />
          <Select.Indicator class="ml-2 shrink-0">
            <ChevronUpDownIcon class="h-4 w-4 text-secondary" />
          </Select.Indicator>
        </Select.Trigger>
      </Select.Control>

      <Teleport to="body">
        <Select.Positioner class="w-[--reference-width]">
          <Select.Content
            class="z-50 min-w-[12rem] w-[--reference-width] overflow-hidden rounded-md border border-border-default bg-surface-base shadow-md animate-in data-[side=bottom]:slide-in-from-top-2"
          >
            <Select.ItemGroup>
              <Select.Item
                v-for="item in collection.items"
                :key="item.value"
                :item="item"
                class="relative flex w-full cursor-pointer select-none items-center justify-between gap-2 py-2 px-4 text-sm text-text-primary hover:bg-surface-raised transition-colors duration-150"
              >
                <Select.ItemText>{{ item.label }}</Select.ItemText>

                <Select.ItemIndicator>
                  <CheckIcon class="h-4 w-4 text-primary" />
                </Select.ItemIndicator>
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
  import { computed, ref, watch } from "vue";
  import { Select, createListCollection } from "@ark-ui/vue/select";
  import { Field } from "@ark-ui/vue/field";
  import { CheckIcon, ChevronUpDownIcon } from "@heroicons/vue/24/outline";

  interface Item {
    label: string;
    value: string | number;
    disabled?: boolean;
  }

  interface Props {
    label?: string;
    placeholder?: string;
    items: Item[];
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

  const collection = computed(() =>
    createListCollection({ items: props.items })
  );

  const triggerClasses = computed(() => [
    "oui-input",
    `oui-input-${props.size}`,
    props.error ? "oui-input-error" : "oui-input-base",
    "w-full flex items-center justify-between text-left",
  ]);

  // External v-model (single value or array for multi); internal always string[] for Ark UI
  const model = defineModel<any>();
  const inner = ref<string[]>([]);

  // Sync external -> internal
  watch(
    () => model.value,
    (val) => {
      if (Array.isArray(val)) inner.value = val.map(String);
      else if (val === null || val === undefined || val === "")
        inner.value = [];
      else inner.value = [String(val)];
    },
    { immediate: true }
  );

  // Sync internal -> external (emit single value for single-select)
  watch(
    () => inner.value,
    (arr) => {
      const next: any = !arr?.length
        ? null
        : arr.length === 1
        ? arr[0]
        : [...arr];
      (model as any).value = next as any;
    }
  );

  defineOptions({ inheritAttrs: false });
</script>
