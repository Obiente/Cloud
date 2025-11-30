<template>
  <OuiTabbedWindow
    v-model="isOpen"
    v-model:active-tab-id="activeTabId"
    :tabs="tabs"
    :window-title="windowTitle"
    :draggable="draggable"
    :resizable="resizable"
    :minimizable="minimizable"
    :tabs-draggable="tabsDraggable"
    :show-tab-close="showTabClose"
    :close-on-escape="closeOnEscape"
    :show-close="showClose"
    :default-position="initialPosition"
    :size="initialSize"
    :persist-rect="persistRect"
    :strategy="strategy"
    @close="handleClose"
    @tab-close="handleTabClose"
    @tabs-reorder="handleTabsReorder"
    @tab-drag-out="handleTabDragOut"
    @tab-drop-external="handleTabDropExternal"
  >
    <template v-for="tab in tabs" :key="tab.id" #[`tab-${tab.id}`]="slotProps">
      <slot :name="`tab-${tab.id}`" :tab="slotProps.tab" />
    </template>
    <template #footer="slotProps">
      <slot name="footer" :activeTab="slotProps.activeTab" />
    </template>
  </OuiTabbedWindow>
</template>

<script setup lang="ts">
import { computed } from "vue";
import OuiTabbedWindow from "~/components/oui/TabbedWindow.vue";
import type { Tab as TabbedWindowTab } from "~/components/oui/TabbedWindow.vue";

interface Tab {
  id: string;
  title: string;
  [key: string]: any;
}

interface Props {
  tabs: Tab[];
  modelValue?: string;
  windowTitle?: string;
  draggable?: boolean;
  resizable?: boolean;
  minimizable?: boolean;
  tabsDraggable?: boolean;
  showTabClose?: boolean;
  closeOnEscape?: boolean;
  showClose?: boolean;
  initialPosition?: { x: number; y: number };
  initialSize?: { width: number; height: number };
  persistRect?: boolean;
  strategy?: "absolute" | "fixed";
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
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
  close: [];
  "tab-close": [tabId: string];
  "tab-select": [tabId: string];
  "tabs-reorder": [newOrder: any[]];
  "tab-drag-out": [tabId: string, event: DragEvent];
  "tab-drop-external": [tabId: string, event: DragEvent];
}>();

const isOpen = computed(() => props.tabs.length > 0);

const activeTabId = computed({
  get: () => props.modelValue || props.tabs[0]?.id || "",
  set: (value) => emit("update:modelValue", value),
});

function handleClose() {
  emit("close");
}

function handleTabClose(tabId: string) {
  emit("tab-close", tabId);
}

function handleTabsReorder(newOrder: any[]) {
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
