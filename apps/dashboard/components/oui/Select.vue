<template>
  <div class="w-full">
    <Select.Root 
      :collection="collection"
      :value="selectedValue" 
      @value-change="handleValueChange"
      v-bind="$attrs"
    >
      <Select.Label 
        v-if="label" 
        class="block text-sm font-medium text-secondary mb-2"
      >
        {{ label }}
      </Select.Label>
      
      <Select.Control class="relative">
        <Select.Trigger
          class="oui-input-base w-full flex items-center justify-between text-left"
        >
          <Select.ValueText 
            :placeholder="placeholder || 'Select an option...'" 
            class="truncate"
          />
          <Select.Indicator class="ml-2 flex-shrink-0">
            <ChevronDownIcon class="h-4 w-4 text-secondary" />
          </Select.Indicator>
        </Select.Trigger>
      </Select.Control>

      <Teleport to="body">
        <Select.Positioner>
          <Select.Content
            class="z-50 min-w-[8rem] overflow-hidden rounded-md border border-border-default bg-surface-base p-1 shadow-md animate-in data-[side=bottom]:slide-in-from-top-2"
          >
            <Select.ItemGroup>
              <Select.Item
                v-for="item in collection.items"
                :key="item.value"
                :item="item"
                class="relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm text-text-primary outline-none hover:bg-surface-muted focus:bg-surface-muted data-[disabled]:pointer-events-none data-[disabled]:opacity-50"
              >
                <span class="absolute left-2 flex h-3.5 w-3.5 items-center justify-center">
                  <Select.ItemIndicator>
                    <CheckIcon class="h-4 w-4" />
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
  </div>
</template>

<script setup lang="ts">
import { Select, createListCollection } from '@ark-ui/vue/select'
import { ChevronDownIcon, CheckIcon } from '@heroicons/vue/24/outline'
import { computed } from 'vue'

interface SelectItem {
  label: string
  value: string | number
  disabled?: boolean
}

interface Props {
  label?: string
  placeholder?: string
  modelValue?: any
  items: SelectItem[]
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:modelValue': [value: any]
}>()

const collection = computed(() => createListCollection({
  items: props.items
}))

const selectedValue = computed(() => props.modelValue ? [props.modelValue] : [])

const handleValueChange = (details: any) => {
  const value = details.value[0] || ''
  emit('update:modelValue', value)
}

defineOptions({
  inheritAttrs: false
})
</script>