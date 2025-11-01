<template>
  <Tabs.Root v-model="modelValue" :defaultValue="defaultValue" class="oui-tabs-wrapper">
    <div v-if="!contentOnly" class="oui-tabs-header">
      <button
        v-if="canScrollLeft"
        type="button"
        class="oui-tabs-nav oui-tabs-nav--left"
        @click="scrollTabs('left')"
        aria-label="Scroll tabs left"
      >
        <ChevronLeftIcon class="h-4 w-4" />
      </button>

      <div class="oui-tabs-scroll-wrapper">
        <ScrollArea.Root class="oui-tabs-scroll">
          <ScrollArea.Viewport
            ref="scrollViewport"
            class="oui-tabs-viewport"
          >
            <Tabs.List class="oui-tabs-list" :class="listClass">
              <Tabs.Trigger
                v-for="tab in tabs"
                :key="tab.id"
                :ref="(el) => setTabRef(tab.id, el)"
                :value="tab.id"
                :disabled="tab.disabled"
                :class="[
                  'oui-tab-trigger',
                  triggerClass,
                  tab.triggerClass,
                ]"
              >
                <component
                  v-if="tab.icon"
                  :is="tab.icon"
                  :class="[
                    'oui-tab-icon',
                    iconClass,
                    { 'oui-tab-icon--active': modelValue === tab.id }
                  ]"
                />
                <span
                  class="oui-tab-label"
                  :class="{
                    'oui-tab-label--active': modelValue === tab.id,
                  }"
                >
                  {{ tab.label }}
                </span>
              </Tabs.Trigger>
            </Tabs.List>
          </ScrollArea.Viewport>
          <ScrollArea.Scrollbar orientation="horizontal" class="oui-tabs-scrollbar">
            <ScrollArea.Thumb class="oui-tabs-thumb" />
          </ScrollArea.Scrollbar>
        </ScrollArea.Root>
        <div v-show="canScrollLeft" class="oui-tabs-fade oui-tabs-fade--left" />
        <div v-show="canScrollRight" class="oui-tabs-fade oui-tabs-fade--right" />
      </div>

      <button
        v-if="canScrollRight"
        type="button"
        class="oui-tabs-nav oui-tabs-nav--right"
        @click="scrollTabs('right')"
        aria-label="Scroll tabs right"
      >
        <ChevronRightIcon class="h-4 w-4" />
      </button>
    </div>

    <Tabs.Content
      v-for="tab in tabs"
      :key="`content-${tab.id}`"
      :value="tab.id"
      :class="['oui-tabs-panel', contentClass, tab.contentClass]"
    >
      <slot :name="tab.id">
        <component v-if="tab.component" :is="tab.component" v-bind="tab.props || {}" />
      </slot>
    </Tabs.Content>
  </Tabs.Root>
</template>

<script setup lang="ts">
import { Tabs } from '@ark-ui/vue/tabs'
import { ScrollArea } from '@ark-ui/vue/scroll-area'
import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/vue/24/outline'
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
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
  contentOnly?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  listClass: '',
  triggerClass: '',
  contentClass: 'p-4 sm:p-6',
  iconClass: '',
  defaultValue: undefined,
  contentOnly: false,
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

// Refs for tab triggers to enable scrolling to active tab
const tabRefs = ref<Record<string, HTMLElement | null>>({})
const scrollViewport = ref<HTMLElement | null>(null)
const canScrollLeft = ref(false)
const canScrollRight = ref(false)

const SCROLL_STEP = 240
let scrollAnimationFrame: number | null = null

// Store refs for each tab trigger
const setTabRef = (tabId: string, el: any) => {
  if (el && el instanceof HTMLElement) {
    tabRefs.value[tabId] = el
    requestAnimationFrame(updateScrollEdges)
  } else if (el && '$el' in el && el.$el instanceof HTMLElement) {
    tabRefs.value[tabId] = el.$el
    requestAnimationFrame(updateScrollEdges)
  } else if (!el) {
    delete tabRefs.value[tabId]
    requestAnimationFrame(updateScrollEdges)
  }
}

// Scroll active tab into view
const updateScrollEdges = () => {
  const viewport = scrollViewport.value
  if (!viewport) {
    canScrollLeft.value = false
    canScrollRight.value = false
    return
  }

  const { scrollLeft, scrollWidth, clientWidth } = viewport
  canScrollLeft.value = scrollLeft > 8
  canScrollRight.value = scrollLeft + clientWidth < scrollWidth - 8
}

const handleViewportScroll = () => {
  if (scrollAnimationFrame) {
    cancelAnimationFrame(scrollAnimationFrame)
  }
  scrollAnimationFrame = requestAnimationFrame(updateScrollEdges)
}

const scrollTabs = (direction: 'left' | 'right') => {
  const viewport = scrollViewport.value
  if (!viewport) return

  const offset = direction === 'left' ? -SCROLL_STEP : SCROLL_STEP
  viewport.scrollBy({ left: offset, behavior: 'smooth' })

  // Update edges after scroll animation
  setTimeout(updateScrollEdges, 320)
}

const scrollToActiveTab = async () => {
  await nextTick()
  const activeTabId = modelValue.value
  if (!activeTabId) return

  const activeTabEl = tabRefs.value[activeTabId]
  if (activeTabEl) {
    activeTabEl.scrollIntoView({
      behavior: 'smooth',
      inline: 'center',
      block: 'nearest',
    })
    requestAnimationFrame(updateScrollEdges)
    setTimeout(updateScrollEdges, 300)
  }
}

// Watch for tab changes and scroll to active tab
watch(modelValue, () => {
  scrollToActiveTab()
})

const refreshScrollState = () => {
  updateScrollEdges()
  scrollToActiveTab()
}

watch(
  () => props.tabs.map((tab) => tab.id).join(','),
  () => {
    nextTick(refreshScrollState)
  }
)

onMounted(() => {
  nextTick(() => {
    updateScrollEdges()
    scrollToActiveTab()

    const viewport = scrollViewport.value
    if (viewport) {
      viewport.addEventListener('scroll', handleViewportScroll, { passive: true })
    }
    window.addEventListener('resize', updateScrollEdges)
  })
})

onUnmounted(() => {
  const viewport = scrollViewport.value
  if (viewport) {
    viewport.removeEventListener('scroll', handleViewportScroll)
  }
  window.removeEventListener('resize', updateScrollEdges)
  if (scrollAnimationFrame) {
    cancelAnimationFrame(scrollAnimationFrame)
  }
})
</script>

<style scoped>
.oui-tabs-wrapper {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.oui-tabs-header {
  display: flex;
  align-items: flex-end;
  gap: 0.5rem;
  position: relative;
  border-bottom: 1px solid var(--oui-border-default, rgba(255, 255, 255, 0.1));
  margin-bottom: 0;
}

.oui-tabs-scroll-wrapper {
  position: relative;
  flex: 1;
}

.oui-tabs-scroll {
  width: 100%;
}

.oui-tabs-viewport {
  overflow-x: auto;
  overflow-y: hidden;
  scroll-behavior: smooth;
  scrollbar-width: none;
  -ms-overflow-style: none;
  padding: 0;
}

.oui-tabs-viewport::-webkit-scrollbar {
  display: none;
}

.oui-tabs-list {
  display: flex;
  align-items: flex-end;
  gap: 0;
  min-width: max-content;
  height: 100%;
}

.oui-tab-trigger {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--oui-text-secondary, rgba(226, 223, 255, 0.7));
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  transition: all 0.2s ease;
  min-height: 2.75rem;
  cursor: pointer;
  white-space: nowrap;
  margin-bottom: -1px;
}

.oui-tab-trigger:hover:not([data-disabled]) {
  color: var(--oui-text-primary, #f5f3ff);
  background: rgba(255, 255, 255, 0.02);
}

.oui-tab-trigger[data-state="active"] {
  color: var(--oui-accent-primary, #a855f7);
  border-bottom-color: var(--oui-accent-primary, #a855f7);
  background: transparent;
}

.oui-tab-trigger[data-disabled] {
  opacity: 0.4;
  cursor: not-allowed;
}

.oui-tab-trigger::after {
  display: none;
}

.oui-tab-icon {
  height: 1.05rem;
  width: 1.05rem;
  flex-shrink: 0;
  color: inherit;
  opacity: 0.85;
  transition: inherit;
}

.oui-tab-icon--active {
  opacity: 1;
}

.oui-tab-label {
  position: relative;
  white-space: nowrap;
  letter-spacing: 0.004em;
  transition: inherit;
}

.oui-tab-label--active {
  font-weight: 600;
}

.oui-tabs-nav {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.05rem;
  height: 2.05rem;
  border-radius: 999px;
  border: 1px solid rgba(121, 99, 196, 0.26);
  background: rgba(22, 16, 44, 0.78);
  color: var(--oui-text-secondary, rgba(226, 223, 255, 0.75));
  box-shadow: 0 6px 16px rgba(5, 2, 15, 0.42);
  transition: all 0.18s ease;
}

.oui-tabs-nav:hover {
  border-color: rgba(168, 85, 247, 0.48);
  color: var(--oui-text-primary, #f5f3ff);
  background: rgba(40, 27, 76, 0.9);
}

.oui-tabs-fade {
  position: absolute;
  top: 0.15rem;
  bottom: 0.15rem;
  width: 2.2rem;
  pointer-events: none;
  z-index: 2;
}

.oui-tabs-fade--left {
  left: 0;
  background: linear-gradient(90deg, rgba(12, 7, 26, 0.9), rgba(12, 7, 26, 0));
}

.oui-tabs-fade--right {
  right: 0;
  background: linear-gradient(270deg, rgba(12, 7, 26, 0.9), rgba(12, 7, 26, 0));
}

.oui-tabs-scrollbar {
  height: 6px;
  margin: 0.25rem 0 0;
  opacity: 0;
  transition: opacity 0.2s ease;
}

.oui-tabs-scroll:hover .oui-tabs-scrollbar,
.oui-tabs-scroll:focus-within .oui-tabs-scrollbar {
  opacity: 1;
}

.oui-tabs-thumb {
  background: linear-gradient(90deg, rgba(142, 88, 245, 0.45), rgba(86, 122, 255, 0.45));
  border-radius: 999px;
}

.oui-tabs-panel {
  position: relative;
  padding-top: 1.5rem;
}

@media (min-width: 768px) {
  .oui-tab-trigger {
    padding: 0.55rem 1.1rem;
    font-size: 0.96rem;
  }
}
</style>
