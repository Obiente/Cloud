<template>
  <div class="oui-org-switcher">
    <Select.RootProvider :collection="props.collection" :value="select">
      <Select.Control class="relative">
        <Select.Trigger
          class="inline-flex items-center gap-2 cursor-pointer rounded-md p-1 hover:bg-surface-variant"
        >
          <Select.ValueText class="sr-only" />
          <Select.Indicator>
            <slot name="icon">
              <ChevronUpDownIcon class="h-5 w-5 text-secondary" />
            </slot>
          </Select.Indicator>
        </Select.Trigger>
      </Select.Control>

      <Teleport to="body">
        <Select.Positioner>
          <Select.Content
            class="z-50 min-w-[12rem] overflow-hidden rounded-lg border border-border-default bg-surface-base shadow-lg animate-in duration-150 transform-gpu data-[side=bottom]:slide-in-from-top-2"
          >
            <!-- Header -->
            <OuiText
              class="px-4 pt-3 pb-2 text-xs font-semibold text-secondary uppercase"
            >
              Organizations
            </OuiText>

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
import { Select, useSelect, type SelectRootProps } from "@ark-ui/vue/select";
import { CheckIcon, ChevronUpDownIcon } from "@heroicons/vue/24/outline";

interface SelectItem {
  label: string;
  value: string | number;
}

const props = defineProps<SelectRootProps<SelectItem>>();
const select = useSelect({ collection: props.collection });
// const modelValue = defineModel<string[]>();

defineOptions({ inheritAttrs: false });
</script>
