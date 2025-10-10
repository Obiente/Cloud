<template>
  <div class="oui-org-switcher">
    <Select.RootProvider
      :collection="collection"
      :value="select"
      v-bind="$attrs"
    >
      <Select.Control class="relative">
        <Select.Trigger
          class="inline-flex items-center gap-2 cursor-pointer rounded-md p-1 hover:bg-surface-variant"
        >
          <Select.ValueText class="sr-only" />
          <Select.Indicator>
            <ChevronUpDownIcon class="h-5 w-5 text-secondary" />
          </Select.Indicator>
        </Select.Trigger>
      </Select.Control>

      <Teleport to="body">
        <Select.Positioner>
          <Select.Content
            class="z-50 min-w-[12rem] overflow-hidden rounded-lg border border-border-default bg-surface-base shadow-lg animate-in duration-150 transform-gpu data-[side=bottom]:slide-in-from-top-2"
          >
            <!-- Header -->
            <div
              class="px-4 pt-3 pb-2 text-xs font-semibold text-secondary uppercase"
            >
              Organizations
            </div>

            <Select.ItemGroup>
              <Select.Item
                v-for="item in collection.items"
                :key="item.value"
                :item="item"
                class="relative flex w-full cursor-pointer select-none items-center justify-between gap-2 py-2 px-4 text-sm text-text-primary hover:bg-surface-raised transition-colors duration-150"
              >
                <Select.ItemText class="truncate">{{
                  item.label
                }}</Select.ItemText>
                <Select.ItemIndicator>
                  <CheckIcon class="h-4 w-4 text-primary" />
                </Select.ItemIndicator>
              </Select.Item>
            </Select.ItemGroup>

            <div class="border-t border-border-muted">
              <button
                href="#"
                class="w-full text-left text-sm text-primary font-medium px-4 py-3 hover:bg-surface-raised cursor-pointer"
              >
                + New Org
              </button>
            </div>
          </Select.Content>
        </Select.Positioner>
      </Teleport>

      <Select.HiddenSelect />
    </Select.RootProvider>
  </div>
</template>

<script setup lang="ts">
import { Select, createListCollection, useSelect } from "@ark-ui/vue/select";
import { CheckIcon, ChevronUpDownIcon } from "@heroicons/vue/24/outline";

interface SelectItem {
  label: string;
  value: string | number;
  disabled?: boolean;
}

interface Props {
  items: SelectItem[];
}

const props = defineProps<Props>();

const collection = createListCollection({ items: props.items });

const select = useSelect({ collection, multiple: false });

defineModel<string | string[]>({
  get: () => select.value.value,
  set: (val) => {
    if (!Array.isArray(val)) val = [val];
    select.value.setValue(val);
  },
});

defineOptions({ inheritAttrs: false });
</script>
