<template>
  <Tabs.Root v-model="modelValue" :defaultValue="defaultValue">
    <Tabs.List :class="listClass">
      <Tabs.Trigger
        v-for="tab in tabs"
        :key="tab.id"
        :value="tab.id"
        :disabled="tab.disabled"
        :class="[
          triggerClass,
          tab.triggerClass,
        ]"
      >
        <component 
          v-if="tab.icon" 
          :is="tab.icon" 
          :class="[
            iconClass,
            modelValue === tab.id ? 'text-primary' : 'text-text-secondary'
          ]"
        />
        <span 
          :class="[
            'relative',
            modelValue === tab.id ? 'text-primary font-semibold' : 'text-text-secondary'
          ]"
        >
          {{ tab.label }}
        </span>
      </Tabs.Trigger>
    </Tabs.List>
    <Tabs.Content
      v-for="tab in tabs"
      :key="`content-${tab.id}`"
      :value="tab.id"
      :class="[contentClass, tab.contentClass]"
    >
      <slot :name="tab.id">
        <component v-if="tab.component" :is="tab.component" v-bind="tab.props || {}" />
      </slot>
    </Tabs.Content>
  </Tabs.Root>
</template>

<script setup lang="ts">
import { Tabs } from '@ark-ui/vue/tabs'
import { computed } from 'vue'
import type { Component } from 'vue'

export interface TabItem {
  id: string
  label: string
  icon?: Component
  disabled?: boolean
  component?: Component
  props?: Record<string, any>
  triggerClass?: string
  contentClass?: string
}

interface Props {
  tabs: TabItem[]
  listClass?: string
  triggerClass?: string
  contentClass?: string
  iconClass?: string
  defaultValue?: string
}

const props = withDefaults(defineProps<Props>(), {
  listClass: 'flex gap-1 border-b border-border-default',
  triggerClass: 'relative flex items-center gap-2 px-4 py-3 text-sm font-medium transition-all duration-200 border-b-2 border-transparent -mb-px text-text-secondary hover:text-text-primary hover:bg-surface-raised/50 rounded-t-md cursor-pointer select-none pointer-events-auto data-[state=active]:text-primary data-[state=active]:font-semibold data-[state=active]:border-primary data-[state=active]:border-b-[3px] data-[state=active]:bg-primary/10 data-[state=active]:shadow-md disabled:opacity-50 disabled:cursor-not-allowed',
  contentClass: 'p-6',
  iconClass: 'h-5 w-5 shrink-0 transition-colors',
  defaultValue: undefined,
})

const modelValue = defineModel<string>({ default: undefined })

// Define slots - slots are named by tab.id, but we can't enforce exact type safety for dynamic slot names
defineSlots<Record<string, () => any>>()

// Compute defaultValue from first tab if not provided and modelValue is not set
// This is just derived state, Ark UI handles all reactivity internally
const defaultValue = computed(() => {
  if (props.defaultValue !== undefined) return props.defaultValue
  if (modelValue.value !== undefined && modelValue.value !== '') return undefined
  const firstTab = props.tabs[0]
  if (firstTab) return firstTab.id
  return undefined
})
</script>
