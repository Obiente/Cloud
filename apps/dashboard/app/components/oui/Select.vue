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
          <div class="ml-2 shrink-0 flex items-center gap-1">
            <button
              v-if="clearable && hasValue"
              type="button"
              @click.stop.prevent="handleClear"
              class="text-text-secondary hover:text-primary transition-colors p-0.5 rounded"
              aria-label="Clear selection"
              tabindex="-1"
            >
              <XMarkIcon class="h-4 w-4" />
            </button>
            <Select.Indicator>
              <ChevronUpDownIcon class="h-4 w-4 text-secondary" />
            </Select.Indicator>
          </div>
        </Select.Trigger>
      </Select.Control>

      <template v-if="isMounted">
        <Teleport to="body">
          <Select.Positioner class="w-[--reference-width]">
            <Select.Content
              class="z-50 min-w-[12rem] max-h-[300px] w-[--reference-width] overflow-y-auto rounded-xl border border-border-default bg-surface-base shadow-md animate-in data-[side=bottom]:slide-in-from-top-2"
            >
              <Select.ItemGroup>
                <Select.Item
                  v-for="item in collection.items"
                  :key="item.value"
                  :item="item"
                  class="relative text-primary flex w-full cursor-pointer select-none items-center justify-between gap-2 py-2 px-4 text-sm transition-colors duration-150 hover:bg-surface-raised data-disabled:cursor-not-allowed data-disabled:text-text-muted data-disabled:opacity-60"
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
      </template>
      <template v-else>
        <!-- SSR fallback - render inline during SSR to avoid hydration mismatch -->
        <Select.Positioner class="w-[--reference-width]">
          <Select.Content
            class="z-50 min-w-[12rem] max-h-[300px] w-[--reference-width] overflow-y-auto rounded-xl border border-border-default bg-surface-base shadow-md hidden"
          >
            <Select.ItemGroup>
              <Select.Item
                v-for="item in collection.items"
                :key="item.value"
                :item="item"
                class="relative text-primary flex w-full cursor-pointer select-none items-center justify-between gap-2 py-2 px-4 text-sm transition-colors duration-150 hover:bg-surface-raised data-disabled:cursor-not-allowed data-disabled:text-text-muted data-disabled:opacity-60"
              >
                <Select.ItemText>{{ item.label }}</Select.ItemText>

                <Select.ItemIndicator>
                  <CheckIcon class="h-4 w-4 text-primary" />
                </Select.ItemIndicator>
              </Select.Item>
            </Select.ItemGroup>
          </Select.Content>
        </Select.Positioner>
      </template>

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
  import { computed, ref, watch, onMounted } from "vue";
  import { Select, createListCollection } from "@ark-ui/vue/select";
  import { Field } from "@ark-ui/vue/field";
  import { CheckIcon, ChevronUpDownIcon, XMarkIcon } from "@heroicons/vue/24/outline";

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
    clearable?: boolean;
    size?: "sm" | "md" | "lg";
  }

  const props = withDefaults(defineProps<Props>(), {
    size: "md",
    required: false,
    disabled: false,
    clearable: false,
  });

  const collection = computed(() => {
    // Ensure items are properly formatted for Ark UI
    const items = props.items.map((item) => ({
      label: item.label,
      value: String(item.value), // Convert to string for Ark UI internal matching
      disabled: item.disabled,
    }));
    return createListCollection({ items });
  });

  const isMounted = ref(false);

  onMounted(() => {
    isMounted.value = true;
  });

  const triggerClasses = computed(() => [
    "oui-input",
    `oui-input-${props.size}`,
    props.error ? "oui-input-error" : "oui-input-base",
    "w-full flex items-center justify-between text-left",
  ]);

  // External v-model (single value or array for multi); internal always string[] for Ark UI
  const model = defineModel<any>();
  const inner = ref<string[]>([]);

  // Computed to check if there's a value (for showing clear button)
  const hasValue = computed(() => {
    const val = model.value;
    return val != null && val !== undefined && val !== "" && (!Array.isArray(val) || val.length > 0);
  });

  // Sync external -> internal: Convert model value to string array for Ark UI
  watch(
    () => model.value,
    (val) => {
      if (Array.isArray(val)) {
        inner.value = val.map(String);
      } else if (val === null || val === undefined || val === "") {
        inner.value = [];
      } else {
        inner.value = [String(val)];
      }
    },
    { immediate: true }
  );

  // Sync internal -> external: Convert string array back to single value
  watch(
    () => inner.value,
    (arr) => {
      if (!arr?.length) {
        if (model.value !== null) {
          (model as any).value = null;
        }
        return;
      }

      // Get the string value from Ark UI
      const strVal = arr.length === 1 ? arr[0] : arr;

      // Find matching item to preserve original value type
      if (typeof strVal === "string" && arr.length === 1) {
        // Find by comparing string values
        const matchingItem = props.items.find((item) => {
          const itemValueStr = String(item.value);
          return itemValueStr === strVal;
        });
        if (matchingItem) {
          // Update with the original value type from the item
          (model as any).value = matchingItem.value;
          return;
        }
      }

      // Fallback: use the array or string value
      const newValue = arr.length === 1 ? strVal : [...arr];
      (model as any).value = newValue;
    },
    { immediate: false }
  );

  const handleClear = () => {
    (model as any).value = null;
    inner.value = [];
  };

  defineOptions({ inheritAttrs: false });
</script>
