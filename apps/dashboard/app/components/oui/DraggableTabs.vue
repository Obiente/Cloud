<template>
  <Tabs.Root ref="rootRef" v-model="modelValue" :defaultValue="defaultValue" class="flex flex-col gap-0">
    <div
      v-if="!contentOnly || tabsOnly"
      :class="[
        'flex items-end gap-0 relative border-b border-border-default mb-0 overflow-hidden min-w-0',
        variantClasses.barWrapper,
      ]"
    >
      <!-- Left scroll button -->
      <div :class="variantClasses.controlWrapper">
        <Transition name="fade-scale">
          <button
            v-show="canScrollLeft"
            type="button"
            :class="variantClasses.navButton"
            @click="scrollTabs('left')"
            aria-label="Scroll tabs left"
          >
            <ChevronLeftIcon :class="variantClasses.navIcon" />
          </button>
        </Transition>
      </div>

      <div :class="variantClasses.tabsContainer">
        <ScrollArea.Root class="w-full overflow-hidden group">
          <ScrollArea.Viewport
            ref="scrollViewport"
            class="overflow-x-auto overflow-y-hidden scroll-smooth p-0 overscroll-x-contain [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden"
          >
            <Tabs.List 
              ref="tabsListRef" 
              :class="[variantClasses.tabsList, listClass]"
              @dragover="handleListDragOver"
              @drop="handleListDrop"
              @dragleave="handleListDragLeave"
            >
              <Tabs.Trigger
                v-for="tab in tabs"
                :key="tab.id"
                :ref="(el: Element | ComponentPublicInstance | null) => setTabRef(tab.id, el)"
                :value="tab.id"
                :disabled="tab.disabled"
                :draggable="draggable"
                :class="[
                  variantClasses.triggerBase,
                  draggable ? 'cursor-grab active:cursor-grabbing' : 'cursor-pointer',
                  isDragging && draggedTabId === tab.id ? 'opacity-50' : '',
                  isExternalDragOver ? 'ring-2 ring-primary/50' : '',
                  triggerClass,
                  tab.triggerClass,
                ]"
                @mouseenter="handleTabHover(tab.id)"
                @dragstart="(e: DragEvent) => handleDragStart(e, tab.id)"
                @dragend="handleDragEnd"
                @dragover="(e: DragEvent) => handleDragOver(e, tab.id)"
                @drop="(e: DragEvent) => handleDrop(e, tab.id)"
                @dragleave="handleDragLeave"
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
                <button
                  v-if="showClose && tab.id"
                  @click.stop="$emit('tab-close', tab.id)"
                  @mousedown.stop
                  class="ml-2 p-0.5 rounded hover:bg-surface-muted transition-colors opacity-70 hover:opacity-100"
                  aria-label="Close tab"
                >
                  <XMarkIcon class="w-3.5 h-3.5" />
                </button>
              </Tabs.Trigger>
            </Tabs.List>
          </ScrollArea.Viewport>
          <ScrollArea.Scrollbar orientation="horizontal" :class="variantClasses.scrollbar">
            <ScrollArea.Thumb :class="variantClasses.scrollbarThumb" />
          </ScrollArea.Scrollbar>
        </ScrollArea.Root>
      </div>

      <!-- Right scroll button -->
      <div :class="variantClasses.controlWrapper">
        <Transition name="fade-scale">
          <button
            v-show="canScrollRight"
            type="button"
            :class="variantClasses.navButton"
            @click="scrollTabs('right')"
            aria-label="Scroll tabs right"
          >
            <ChevronRightIcon :class="variantClasses.navIcon" />
          </button>
        </Transition>
      </div>
    </div>

    <Tabs.Content
      v-if="!tabsOnly"
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
import { ChevronLeftIcon, ChevronRightIcon, XMarkIcon } from '@heroicons/vue/24/outline'
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
  tabsOnly?: boolean
  draggable?: boolean
  showClose?: boolean
  variant?: "default" | "window"
}

const props = withDefaults(defineProps<Props>(), {
  listClass: '',
  triggerClass: '',
  contentClass: 'p-4 sm:p-6',
  iconClass: '',
  defaultValue: undefined,
  contentOnly: false,
  tabsOnly: false,
  draggable: false,
  showClose: false,
  variant: "default",
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'tab-close': [tabId: string]
  'tabs-reorder': [newOrder: TabItem[]]
  'tab-drag-out': [tabId: string, event: DragEvent]
  'tab-drop-external': [tabId: string, event: DragEvent]
}>()

const modelValue = defineModel<string>({ default: undefined })

const variantClasses = computed(() => {
  const isWindow = props.variant === "window"
  return {
    barWrapper: isWindow ? 'bg-surface-raised/80 backdrop-blur' : '',
    controlWrapper: isWindow
      ? 'px-1 h-[2.25rem] flex items-center justify-center shrink-0'
      : 'w-[2.05rem] h-[2.75rem] flex items-center justify-center shrink-0',
    navButton: isWindow
      ? 'inline-flex items-center justify-center w-7 h-7 rounded-md border border-border-muted bg-transparent text-text-secondary transition-colors duration-150 hover:text-text-primary hover:bg-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:opacity-40'
      : 'inline-flex items-center justify-center w-[2.05rem] h-[2.05rem] rounded-full border border-[rgba(121,99,196,0.26)] bg-[rgba(22,16,44,0.78)] text-text-secondary shadow-[0_6px_16px_rgba(5,2,15,0.42)] transition-all duration-[0.18s] hover:border-[rgba(168,85,247,0.48)] hover:text-text-primary hover:bg-[rgba(40,27,76,0.9)]',
    navIcon: isWindow ? 'h-3.5 w-3.5' : 'h-4 w-4',
    tabsContainer: isWindow
      ? 'relative flex-1 min-w-0 overflow-hidden flex items-center'
      : 'relative flex-1 min-w-0 overflow-hidden flex items-end',
    tabsList: isWindow
      ? 'flex items-center gap-1 min-w-max h-full w-fit'
      : 'flex items-end gap-0 min-w-max h-full w-fit',
    triggerBase: isWindow
      ? 'relative inline-flex items-center gap-2 py-1.5 px-3 text-[0.85rem] font-medium text-text-secondary/80 bg-transparent border border-transparent rounded-lg transition-all duration-150 min-h-[2.25rem] whitespace-nowrap hover:text-text-primary hover:bg-surface-muted/60 data-state-active:text-text-primary data-state-active:bg-surface-muted data-state-active:border-border-default focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30 data-disabled:opacity-40 data-disabled:cursor-not-allowed'
      : 'relative inline-flex items-center gap-2 py-3 px-4 text-[0.9rem] md:py-[0.55rem] md:px-[1.1rem] md:text-[0.96rem] font-medium text-text-secondary bg-transparent border-none border-b-2 border-b-transparent transition-all duration-200 min-h-[2.75rem] whitespace-nowrap -mb-px hover:text-text-primary hover:bg-[rgba(255,255,255,0.02)] data-state-active:text-primary data-state-active:border-b-primary data-state-active:bg-transparent data-disabled:opacity-40 data-disabled:cursor-not-allowed data-disabled:hover:text-text-secondary data-disabled:hover:bg-transparent [&::after]:hidden',
    scrollbar: isWindow
      ? 'h-1 mt-1 opacity-0 transition-opacity duration-200 group-hover:opacity-80 group-focus-within:opacity-80'
      : 'h-1.5 mt-1 opacity-0 transition-opacity duration-200 group-hover:opacity-100 group-focus-within:opacity-100',
    scrollbarThumb: isWindow
      ? 'bg-border-muted rounded-full'
      : 'bg-gradient-to-r from-[rgba(142,88,245,0.45)] to-[rgba(86,122,255,0.45)] rounded-full',
  }
})

// Drag and drop state
const isDragging = ref(false)
const draggedTabId = ref<string | null>(null)
const dragOverTabId = ref<string | null>(null)
const isExternalDragOver = ref(false)

function handleDragStart(e: DragEvent, tabId: string) {
  if (!props.draggable) return
  isDragging.value = true
  draggedTabId.value = tabId
  dragOverTabId.value = null
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', tabId)
  }
  // Make the dragged element semi-transparent
  if (e.target instanceof HTMLElement) {
    e.target.style.opacity = '0.5'
  }
}

function handleDragEnd(e: DragEvent) {
  if (!props.draggable) return
  const tabId = draggedTabId.value
  isDragging.value = false
  draggedTabId.value = null
  dragOverTabId.value = null
  if (e.target instanceof HTMLElement) {
    e.target.style.opacity = ''
  }
  
  // Check if drag ended outside the component (drag-out to new window)
  // Check both the stored state and the current drop location
  const root = getRootElement()
  const dropTarget = e.target as Node | null
  const isOutside = isDraggingOutside || (root && dropTarget && !root.contains(dropTarget))
  
  if (tabId && isOutside) {
    emit('tab-drag-out', tabId, e)
  }
  isDraggingOutside = false
}

function handleDocumentDragOver(e: DragEvent) {
  if (!props.draggable || !isDragging.value) return
  // Check if drag is outside the tabs component
  const root = getRootElement()
  const target = e.target as Node | null
  if (root && target) {
    // Check if target is within the tabs root component
    isDraggingOutside = !root.contains(target)
  } else {
    // If no target or root, assume we're outside
    isDraggingOutside = true
  }
}

function handleDocumentDrop(e: DragEvent) {
  if (!props.draggable || !isDragging.value) return
  const tabId = draggedTabId.value
  if (!tabId) return
  
  // Check if drop is outside the tabs component
  const root = getRootElement()
  const dropTarget = e.target as Node | null
  const isOutside = isDraggingOutside || (root && dropTarget && !root.contains(dropTarget))
  
  if (isOutside) {
    e.preventDefault()
    e.stopPropagation()
    // Emit drag-out event
    emit('tab-drag-out', tabId, e)
  }
}

function handleDragOver(e: DragEvent, tabId: string) {
  if (!props.draggable || !isDragging.value || draggedTabId.value === tabId) {
    return
  }
  e.preventDefault()
  e.stopPropagation()
  if (e.dataTransfer) {
    e.dataTransfer.dropEffect = 'move'
  }
  dragOverTabId.value = tabId
}

function handleDrop(e: DragEvent, targetTabId: string) {
  if (!props.draggable || !isDragging.value || !draggedTabId.value || draggedTabId.value === targetTabId) {
    return
  }
  e.preventDefault()
  e.stopPropagation()

  const draggedId = draggedTabId.value
  const newTabs = [...props.tabs]
  const draggedIndex = newTabs.findIndex((t) => t.id === draggedId)
  const targetIndex = newTabs.findIndex((t) => t.id === targetTabId)

  if (draggedIndex !== -1 && targetIndex !== -1) {
    // Remove dragged tab from its position
    const removedTabs = newTabs.splice(draggedIndex, 1)
    const draggedTab = removedTabs[0]
    if (draggedTab) {
      // Insert at target position
      newTabs.splice(targetIndex, 0, draggedTab)
      emit('tabs-reorder', newTabs)
    }
  }

  dragOverTabId.value = null
}

function handleDragLeave() {
  dragOverTabId.value = null
}

function handleListDragOver(e: DragEvent) {
  if (!props.draggable) return
  
  // If we're dragging our own tab, don't handle as external (let normal handler deal with it)
  if (isDragging.value && draggedTabId.value) {
    return
  }
  
  // Check if dataTransfer has tab data (from another window)
  // Note: getData only works in drop event, but we can check types
  if (e.dataTransfer && e.dataTransfer.types.includes('text/plain')) {
    e.preventDefault()
    e.stopPropagation()
    e.dataTransfer.dropEffect = 'move'
    isExternalDragOver.value = true
  }
}

function handleListDrop(e: DragEvent) {
  if (!props.draggable) return
  
  // If we're dropping our own tab, let the normal handler deal with it
  if (isDragging.value && draggedTabId.value) {
    return
  }
  
  // Check if this is an external drop (tab from another window)
  if (e.dataTransfer) {
    const tabId = e.dataTransfer.getData('text/plain')
    // Only handle if it's not one of our own tabs
    if (tabId && !props.tabs.some(t => t.id === tabId)) {
      e.preventDefault()
      e.stopPropagation()
      emit('tab-drop-external', tabId, e)
      isExternalDragOver.value = false
    }
  }
}

function handleListDragLeave(e: DragEvent) {
  // Clear external drag state when leaving the list
  isExternalDragOver.value = false
}

// Track which tabs have been loaded or prefetched
const loadedTabs = ref(new Set<string>())
const prefetchedTabs = ref(new Set<string>())

defineSlots<Record<string, () => any>>()

watch(
  () => modelValue.value,
  (newValue) => {
    if (newValue && !loadedTabs.value.has(newValue)) {
      loadedTabs.value.add(newValue)
    }
  },
  { immediate: true }
)

function handleTabHover(tabId: string) {
  if (!loadedTabs.value.has(tabId) && !prefetchedTabs.value.has(tabId)) {
    prefetchedTabs.value.add(tabId)
  }
}

function shouldRenderTab(tabId: string): boolean {
  return modelValue.value === tabId || loadedTabs.value.has(tabId) || prefetchedTabs.value.has(tabId)
}

const defaultValue = computed(() => {
  if (props.defaultValue !== undefined) return props.defaultValue
  if (modelValue.value !== undefined && modelValue.value !== '') return undefined
  if (!props.tabs || props.tabs.length === 0) return undefined
  const firstTab = props.tabs[0]
  if (firstTab) return firstTab.id
  return undefined
})

// Refs for tab triggers to enable scrolling to active tab
const tabRefs = ref<Record<string, HTMLElement | null>>({})
const scrollViewport = ref<HTMLElement | null>(null)
const tabsListRef = ref<HTMLElement | null>(null)
const rootRef = ref<HTMLElement | null>(null)
const canScrollLeft = ref(false)
const canScrollRight = ref(false)
let isDraggingOutside = false

const SCROLL_STEP = 240
let scrollAnimationFrame: number | null = null

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

const getViewportElement = (): HTMLElement | null => {
  const viewport = scrollViewport.value
  if (!viewport) return null
  if (viewport instanceof HTMLElement) return viewport
  if ('$el' in viewport && (viewport as any).$el instanceof HTMLElement) return (viewport as any).$el
  return null
}

// Get the actual DOM element from the root ref
const getRootElement = (): HTMLElement | null => {
  const root = rootRef.value
  if (!root) return null
  if (root instanceof HTMLElement) return root
  if ('$el' in root && (root as any).$el instanceof HTMLElement) return (root as any).$el
  return null
}

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
    if (props.draggable) {
      document.addEventListener('dragover', handleDocumentDragOver)
      document.addEventListener('drop', handleDocumentDrop)
    }
  })
})

onUnmounted(() => {
  const viewport = getViewportElement()
  if (viewport) {
    viewport.removeEventListener('scroll', handleViewportScroll)
  }
  window.removeEventListener('resize', updateScrollEdges)
  if (props.draggable) {
    document.removeEventListener('dragover', handleDocumentDragOver)
    document.removeEventListener('drop', handleDocumentDrop)
  }
  if (scrollAnimationFrame) {
    cancelAnimationFrame(scrollAnimationFrame)
  }
})
</script>

<style scoped>
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

