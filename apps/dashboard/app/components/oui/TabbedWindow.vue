<template>
  <OuiDraggableWindow
    v-model="isOpen"
    :title="windowTitle"
    :draggable="draggable"
    :resizable="resizable"
    :minimizable="minimizable"
    :close-on-escape="closeOnEscape"
    :show-close="showClose"
    :default-position="defaultPosition"
    :size="size"
    :persist-rect="persistRect"
    :strategy="strategy"
    :content-class="contentClass"
    :header-class="'p-0'"
    :body-class="'p-0 flex flex-col min-h-0'"
    :footer-class="'p-0'"
    @close="handleClose"
  >
    <template #header>
      <div
        class="flex items-center gap-0 w-full bg-surface-raised border-b border-border-default select-none relative"
        :aria-label="`Window tabs, ${tabs.length} ${tabs.length === 1 ? 'tab' : 'tabs'}`"
        role="tablist"
      >
        <div
          class="flex items-center flex-1 min-w-0 w-full"
          @mousedown.stop
          @touchstart.stop
        >
          <OuiDraggableTabs
            v-model="activeTabId"
            :tabs="tabItems"
            :draggable="tabsDraggable"
            :show-close="showTabClose"
            :tabs-only="true"
            :list-class="'flex-1 w-full'"
            :trigger-class="'min-w-0'"
            :variant="tabsVariant"
            @tab-close="handleTabClose"
            @tabs-reorder="handleTabsReorder"
            @tab-drag-out="handleTabDragOut"
            @tab-drop-external="handleTabDropExternal"
          />
        </div>
        <div class="flex items-center gap-1 px-2 shrink-0">
          <slot name="window-controls">
            <OuiButton
              v-if="minimizable"
              variant="ghost"
              size="xs"
              @click="handleMinimize"
              class="!p-1"
              aria-label="Minimize window"
            >
              <MinusIcon class="w-4 h-4" />
            </OuiButton>
            <OuiButton
              v-if="showClose"
              variant="ghost"
              size="xs"
              @click="handleClose"
              class="!p-1"
              aria-label="Close window"
            >
              <XMarkIcon class="w-4 h-4" />
            </OuiButton>
          </slot>
        </div>
      </div>
    </template>

    <div class="flex-1 min-h-0" role="tabpanel" :aria-labelledby="`tab-${activeTabId}`">
      <div
        v-for="tab in tabs"
        :key="tab.id"
        v-show="activeTabId === tab.id"
        :id="`tabpanel-${tab.id}`"
        :aria-labelledby="`tab-${tab.id}`"
        role="tabpanel"
      >
        <slot :name="`tab-${tab.id}`" :tab="tab" />
      </div>
    </div>

    <template #footer>
      <slot name="footer" :activeTab="activeTab" />
    </template>
  </OuiDraggableWindow>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { MinusIcon, XMarkIcon } from "@heroicons/vue/24/outline";
import OuiDraggableWindow from "./DraggableWindow.vue";
import OuiDraggableTabs from "./DraggableTabs.vue";
import type { TabItem } from "./DraggableTabs.vue";

export interface Tab {
  id: string;
  title: string;
  [key: string]: any;
}

interface Props {
  modelValue: boolean;
  tabs: Tab[];
  activeTabId?: string;
  windowTitle?: string;
  draggable?: boolean;
  resizable?: boolean;
  minimizable?: boolean;
  tabsDraggable?: boolean;
  showTabClose?: boolean;
  closeOnEscape?: boolean;
  showClose?: boolean;
  defaultPosition?: { x: number; y: number };
  size?: { width: number; height: number };
  persistRect?: boolean;
  strategy?: "absolute" | "fixed";
  contentClass?: string;
  tabsVariant?: "default" | "window";
}

const props = withDefaults(defineProps<Props>(), {
  draggable: true,
  resizable: false,
  minimizable: true,
  tabsDraggable: true,
  showTabClose: true,
  closeOnEscape: true,
  showClose: true,
  persistRect: true,
  strategy: "fixed",
  contentClass: "",
  tabsVariant: "window",
});

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  "update:activeTabId": [tabId: string];
  close: [];
  minimize: [];
  restore: [];
  "tab-close": [tabId: string];
  "tab-select": [tabId: string];
  "tabs-reorder": [newOrder: TabItem[]];
  "tab-drag-out": [tabId: string, event: DragEvent];
  "tab-drop-external": [tabId: string, event: DragEvent];
}>();

const isOpen = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit("update:modelValue", v),
});

const activeTabId = computed({
  get: () => props.activeTabId || props.tabs[0]?.id || "",
  set: (value) => {
    emit("update:activeTabId", value);
    emit("tab-select", value);
  },
});

const activeTab = computed(() => props.tabs.find((t) => t.id === activeTabId.value));

const tabItems = computed<TabItem[]>(() =>
  props.tabs.map((tab) => ({
    id: tab.id,
    label: tab.title,
  }))
);

function handleClose() {
  emit("update:modelValue", false);
  emit("close");
}

function handleMinimize() {
  emit("minimize");
}

function handleTabClose(tabId: string) {
  emit("tab-close", tabId);
  // If closing active tab, switch to another
  if (activeTabId.value === tabId) {
    const currentIndex = props.tabs.findIndex((t) => t.id === tabId);
    const remainingTabs = props.tabs.filter((t) => t.id !== tabId);
    if (remainingTabs.length > 0) {
      const newIndex = Math.max(0, currentIndex - 1);
      const newTab = remainingTabs[newIndex] || remainingTabs[0];
      if (newTab) {
        activeTabId.value = newTab.id;
      }
    }
  }
}

function handleTabsReorder(newOrder: TabItem[]) {
  emit("tabs-reorder", newOrder);
}

function handleTabDragOut(tabId: string, event: DragEvent) {
  emit("tab-drag-out", tabId, event);
}

function handleTabDropExternal(tabId: string, event: DragEvent) {
  emit("tab-drop-external", tabId, event);
}

defineOptions({
  inheritAttrs: false,
});
</script>

