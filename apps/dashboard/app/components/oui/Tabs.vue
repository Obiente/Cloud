<template>
  <Tabs.Root v-model="modelValue" :defaultValue="defaultValue" class="flex flex-col gap-0">
    <div v-if="!contentOnly" class="flex items-end gap-0 relative border-b border-border-default mb-0 overflow-hidden min-w-0">
      <!-- Left scroll button - always reserved space to prevent layout shift -->
      <div class="w-[2.05rem] h-[2.75rem] flex items-center justify-center shrink-0">
        <Transition name="fade-scale">
          <button
            v-show="canScrollLeft"
            type="button"
            class="inline-flex items-center justify-center w-[2.05rem] h-[2.05rem] rounded-full border border-[rgba(121,99,196,0.26)] bg-[rgba(22,16,44,0.78)] text-text-secondary shadow-[0_6px_16px_rgba(5,2,15,0.42)] transition-all duration-[0.18s] hover:border-[rgba(168,85,247,0.48)] hover:text-text-primary hover:bg-[rgba(40,27,76,0.9)]"
            @click="scrollTabs('left')"
            aria-label="Scroll tabs left"
          >
            <ChevronLeftIcon class="h-4 w-4" />
          </button>
        </Transition>
      </div>

      <div class="relative flex-1 min-w-0 overflow-hidden flex items-end">
        <ScrollArea.Root class="w-full overflow-hidden group">
          <ScrollArea.Viewport
            ref="scrollViewport"
            class="overflow-x-auto overflow-y-hidden scroll-smooth p-0 overscroll-x-contain [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden"
          >
            <Tabs.List class="flex items-end gap-0 min-w-max h-full w-fit" :class="listClass">
              <Tabs.Trigger
                v-for="tab in tabs"
                :key="tab.id"
                :ref="(el: Element | ComponentPublicInstance | null) => setTabRef(tab.id, el)"
                :value="tab.id"
                :disabled="tab.disabled"
                :class="[
                  'relative inline-flex items-center gap-2 py-3 px-4 text-[0.9rem] md:py-[0.55rem] md:px-[1.1rem] md:text-[0.96rem] font-medium text-text-secondary bg-transparent border-none border-b-2 border-b-transparent transition-all duration-200 min-h-[2.75rem] cursor-pointer whitespace-nowrap -mb-px hover:text-text-primary hover:bg-[rgba(255,255,255,0.02)] data-state-active:text-primary data-state-active:border-b-primary data-state-active:bg-transparent data-disabled:opacity-40 data-disabled:cursor-not-allowed data-disabled:hover:text-text-secondary data-disabled:hover:bg-transparent [&::after]:hidden',
                  triggerClass,
                  tab.triggerClass,
                ]"
                @mouseenter="handleTabHover(tab.id)"
              >
                <component
                  v-if="tab.icon"
                  :is="tab.icon"
                  :class="[
                    'h-[1.05rem] w-[1.05rem] shrink-0 opacity-85 transition-[inherit]',
                    { 'opacity-100': modelValue === tab.id },
                    iconClass,
                  ]"
                  :style="{ color: 'inherit' }"
                />
                <span
                  class="relative whitespace-nowrap tracking-[0.004em] transition-[inherit]"
                  :class="{
                    'font-semibold': modelValue === tab.id,
                  }"
                >
                  {{ tab.label }}
                </span>
              </Tabs.Trigger>
            </Tabs.List>
          </ScrollArea.Viewport>
          <ScrollArea.Scrollbar orientation="horizontal" class="h-1.5 mt-1 opacity-0 transition-opacity duration-200 group-hover:opacity-100 group-focus-within:opacity-100">
            <ScrollArea.Thumb class="bg-gradient-to-r from-[rgba(142,88,245,0.45)] to-[rgba(86,122,255,0.45)] rounded-full" />
          </ScrollArea.Scrollbar>
        </ScrollArea.Root>
      </div>

      <!-- Right scroll button - always reserved space to prevent layout shift -->
      <div class="w-[2.05rem] h-[2.75rem] flex items-center justify-center shrink-0">
        <Transition name="fade-scale">
          <button
            v-show="canScrollRight"
            type="button"
            class="inline-flex items-center justify-center w-[2.05rem] h-[2.05rem] rounded-full border border-[rgba(121,99,196,0.26)] bg-[rgba(22,16,44,0.78)] text-text-secondary shadow-[0_6px_16px_rgba(5,2,15,0.42)] transition-all duration-[0.18s] hover:border-[rgba(168,85,247,0.48)] hover:text-text-primary hover:bg-[rgba(40,27,76,0.9)]"
            @click="scrollTabs('right')"
            aria-label="Scroll tabs right"
          >
            <ChevronRightIcon class="h-4 w-4" />
          </button>
        </Transition>
      </div>
    </div>

    <Tabs.Content
      v-for="tab in tabs"
      :key="`content-${tab.id}`"
      :value="tab.id"
      :class="['relative pt-6', contentClass, tab.contentClass]"
    >
      <div v-if="shouldRenderTab(tab.id)">
        <slot :name="tab.id">
          <component v-if="tab.component" :is="tab.component" v-bind="tab.props || {}" />
        </slot>
      </div>
    </Tabs.Content>
  </Tabs.Root>
</template>

<script setup lang="ts">
import { Tabs } from '@ark-ui/vue/tabs'
import { ScrollArea } from '@ark-ui/vue/scroll-area'
import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/vue/24/outline'
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import type { Component, ComponentPublicInstance } from 'vue'

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

// Track which tabs have been loaded or prefetched
const loadedTabs = ref(new Set<string>())
const prefetchedTabs = ref(new Set<string>())

// Define slots - slots are named by tab.id, but we can't enforce exact type safety for dynamic slot names
defineSlots<Record<string, () => any>>()

// Load initial tab
watch(
  () => modelValue.value,
  (newValue) => {
    if (newValue && !loadedTabs.value.has(newValue)) {
      loadedTabs.value.add(newValue)
    }
  },
  { immediate: true }
)

// Handle tab trigger hover for prefetching
function handleTabHover(tabId: string) {
  if (!loadedTabs.value.has(tabId) && !prefetchedTabs.value.has(tabId)) {
    prefetchedTabs.value.add(tabId)
  }
}

// Handle tab content hover for prefetching
function handleTabContentHover(tabId: string) {
  if (!loadedTabs.value.has(tabId) && !prefetchedTabs.value.has(tabId)) {
    prefetchedTabs.value.add(tabId)
  }
}

// Determine if tab should be rendered
function shouldRenderTab(tabId: string): boolean {
  // Render if active, already loaded, or prefetched
  return modelValue.value === tabId || loadedTabs.value.has(tabId) || prefetchedTabs.value.has(tabId)
}

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
const setTabRef = (tabId: string, el: Element | ComponentPublicInstance | null) => {
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

// Get the actual DOM element from the viewport ref
const getViewportElement = (): HTMLElement | null => {
  const viewport = scrollViewport.value
  if (!viewport) return null
  if (viewport instanceof HTMLElement) return viewport
  if ('$el' in viewport && (viewport as any).$el instanceof HTMLElement) return (viewport as any).$el
  return null
}

// Scroll active tab into view
const updateScrollEdges = () => {
  const viewport = getViewportElement()
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
  const viewport = getViewportElement()
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

    const viewport = getViewportElement()
    if (viewport) {
      viewport.addEventListener('scroll', handleViewportScroll, { passive: true })
    }
    window.addEventListener('resize', updateScrollEdges)
  })
})

onUnmounted(() => {
  const viewport = getViewportElement()
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
/* Smooth fade and scale transition for scroll buttons */
.fade-scale-enter-active,
.fade-scale-leave-active {
  transition: opacity 0.3s cubic-bezier(0.4, 0, 0.2, 1),
              transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  pointer-events: none;
}

.fade-scale-enter-active {
  pointer-events: auto;
}

.fade-scale-enter-from {
  opacity: 0;
  transform: scale(0.9);
}

.fade-scale-leave-to {
  opacity: 0;
  transform: scale(0.9);
}

.fade-scale-enter-to,
.fade-scale-leave-from {
  opacity: 1;
  transform: scale(1);
}
</style>
