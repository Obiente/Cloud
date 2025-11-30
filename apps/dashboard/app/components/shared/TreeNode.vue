<script setup lang="ts">
  import { computed, ref, onUnmounted, nextTick } from "vue";
  import { TreeView } from "@ark-ui/vue/tree-view";

  import {
    FolderIcon,
    FolderOpenIcon,
    DocumentIcon,
    ChevronRightIcon,
    ChevronDownIcon,
    MinusSmallIcon,
    LinkIcon,
    ArrowPathIcon,
    ArrowUpTrayIcon,
  } from "@heroicons/vue/24/outline";
  import type { ExplorerNode } from "./fileExplorerTypes";

  defineOptions({
    name: "FileTreeNode",
  });

  const props = defineProps<{
    node: ExplorerNode;
    indexPath: number[];
    selectedPath: string | null;
    selectedNodes?: Set<string>;
    allowEditing?: boolean;
  }>();

  const emit = defineEmits<{
    (e: "toggle", node: ExplorerNode, open: boolean): void;
    (e: "open", node: ExplorerNode, options?: { ensureExpanded?: boolean }): void;
    (e: "select", node: ExplorerNode, event: MouseEvent): void;
    (e: "action", action: string, node: ExplorerNode, selectedPaths?: string[]): void;
    (e: "load-more", node: ExplorerNode): void;
    (e: "drop-files", node: ExplorerNode, files: File[], event?: DragEvent): void;
  }>();

  const isDirectory = computed(() => props.node.type === "directory" || !!props.node.children?.length);
  const isSymlink = computed(() => props.node.type === "symlink");
  const isExpanded = computed(() => !!props.node.isExpanded);
  const isSelected = computed(() => {
    if (props.selectedNodes) {
      return props.selectedNodes.has(props.node.path);
    }
    return props.selectedPath === props.node.path;
  });
  const depth = computed(() => Math.max(props.indexPath.length, 1));
  const iconPadding = computed(() => `${(depth.value - 1) * 14 + 4}px`);

  const menuSections = computed(() => {
    const sections: Array<{
      key: string;
      label: string;
      shortcut?: string;
      disabled?: boolean;
      separatorBefore?: boolean;
    }> = [];

    const add = (item: (typeof sections)[number]) => sections.push(item);
    const isMultiSelect = props.selectedNodes && props.selectedNodes.size > 1;

    // Single-item actions (only show if single selection)
    if (!isMultiSelect) {
      add({ key: "open", label: "Open" });

      if (props.node.type === "directory") {
        add({ key: "refresh", label: "Refresh" });
        add({ key: "new-file", label: "New File", shortcut: "N", separatorBefore: true });
        add({ key: "new-folder", label: "New Folder" });
        add({ key: "new-symlink", label: "New Symlink" });
      } else {
        add({
          key: "open-editor",
          label: "Open in Editor",
        });
      }
    }

    // Multi-select actions
    if (isMultiSelect) {
      add({ key: "copy-path", label: `Copy Path${props.selectedNodes.size > 1 ? 's' : ''}`, separatorBefore: false });
      add({ key: "create-archive", label: "Create Archive", separatorBefore: true });
      add({ key: "delete", label: `Delete ${props.selectedNodes.size} item${props.selectedNodes.size > 1 ? 's' : ''}`, shortcut: "Del", separatorBefore: true });
    } else {
      // Single-select actions
      add({ key: "copy-path", label: "Copy Path", separatorBefore: true });

      if (props.node.path !== "/") {
        add({ key: "create-archive", label: "Create Archive", separatorBefore: true });
        add({ key: "rename", label: "Rename", shortcut: "F2", separatorBefore: true });
        add({ key: "delete", label: "Delete", shortcut: "Del" });
      }
    }

    return sections;
  });

  const displaySize = computed(() => {
    if (props.node.type === "directory") return "";
    if (props.node.size == null) return "";
    const size = props.node.size;
    if (size < 1024) return `${size} B`;
    if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
    if (size < 1024 * 1024 * 1024)
      return `${(size / (1024 * 1024)).toFixed(1)} MB`;
    return `${(size / (1024 * 1024 * 1024)).toFixed(1)} GB`;
  });

  const modeString = computed(() => {
    if (!props.node.mode && props.node.mode !== 0) return "";
    const value = props.node.mode;
    return (value & 0o777).toString(8).padStart(3, "0");
  });

  defineExpose({
    getNode: () => props.node,
  });

  function handleBranchClick(event: MouseEvent) {
    console.log("[TreeNode] handleBranchClick", {
      path: props.node.path,
      ctrlKey: event.ctrlKey,
      metaKey: event.metaKey,
      shiftKey: event.shiftKey,
      isSelected: isSelected.value,
    });
    
    // Prevent default to avoid text selection
    event.preventDefault();
    // Stop propagation to prevent parent nodes from toggling
    event.stopPropagation();
    
    // Handle multi-select
    emit("select", props.node, event);
    
    // Handle expand/collapse - only if not Ctrl/Shift
    if (!event.ctrlKey && !event.metaKey && !event.shiftKey) {
      const newOpenState = !props.node.isExpanded;
      emit("toggle", props.node, newOpenState);
      if (newOpenState) {
        emit("open", props.node, { ensureExpanded: true });
      }
    }
  }

  function handleItemClick(event: MouseEvent) {
    console.log("[TreeNode] handleItemClick", {
      path: props.node.path,
      ctrlKey: event.ctrlKey,
      metaKey: event.metaKey,
      shiftKey: event.shiftKey,
      isSelected: isSelected.value,
    });
    
    // Prevent default to avoid text selection
    event.preventDefault();
    event.stopPropagation();
    
    // Emit select event for multi-select handling
    emit("select", props.node, event);
    // Also emit open for single selection behavior
    if (!event.ctrlKey && !event.metaKey && !event.shiftKey) {
      emit("open", props.node);
    }
  }

  function handleBranchItemClick(event: MouseEvent) {
    // For directories, handle selection but don't open on Ctrl/Shift
    emit("select", props.node, event);
    if (!event.ctrlKey && !event.metaKey && !event.shiftKey) {
      emit("open", props.node, { ensureExpanded: props.node.type === "directory" });
    }
  }

  function handleMenuSelect(action: string) {
    console.log("[TreeNode] handleMenuSelect", {
      action,
      nodePath: props.node.path,
      selectedNodesCount: props.selectedNodes?.size || 0,
    });
    
    // If multiple nodes are selected, pass all selected paths
    if (props.selectedNodes && props.selectedNodes.size > 1) {
      console.log("[TreeNode] Multiple nodes selected, passing all paths:", Array.from(props.selectedNodes));
      emit("action", action, props.node, Array.from(props.selectedNodes));
    } else {
      console.log("[TreeNode] Single node selected, passing only clicked node");
      emit("action", action, props.node);
    }
    menuOpen.value = false;
  }

  function handleContextMenu(event: MouseEvent) {
    event.preventDefault();
    event.stopPropagation();
    
    // Set menu position by updating the trigger element's position
    if (menuTriggerRef.value) {
      // Position the hidden trigger at the click location
      menuTriggerRef.value.style.position = 'fixed';
      menuTriggerRef.value.style.left = `${event.clientX}px`;
      menuTriggerRef.value.style.top = `${event.clientY}px`;
      menuTriggerRef.value.style.width = '1px';
      menuTriggerRef.value.style.height = '1px';
      menuTriggerRef.value.style.opacity = '0';
      menuTriggerRef.value.style.pointerEvents = 'none';
      menuTriggerRef.value.style.zIndex = '-1';
    }
    
    // Use nextTick to ensure DOM is updated before opening menu
    nextTick(() => {
      menuOpen.value = true;
    });
  }

  function handleLoadMore(event: Event) {
    event.preventDefault();
    event.stopPropagation();
    if (props.node.isLoading) return;
    emit("load-more", props.node);
  }

  function handleBranchOpenChange(event: any) {
    // Handle the open-change event from Ark UI
    // This fires when the branch is clicked or toggled via keyboard
    let openState: boolean;
    if (typeof event === "boolean") {
      openState = event;
    } else if (event?.detail && typeof event.detail.open === "boolean") {
      openState = event.detail.open;
    } else if (typeof event?.open === "boolean") {
      openState = event.open;
    } else {
      // Fallback: toggle current state
      openState = !props.node.isExpanded;
    }

    // Only emit if state actually changed
    if (openState !== props.node.isExpanded) {
      emit("toggle", props.node, openState);
      if (openState) {
        emit("open", props.node, { ensureExpanded: true });
      }
    }
  }

  const isDraggingOver = ref(false);
  let expandTimer: ReturnType<typeof setTimeout> | null = null;
  const EXPAND_DELAY = 800; // 800ms hover delay before auto-expanding
  
  const menuOpen = ref(false);
  const menuTriggerRef = ref<HTMLElement | null>(null);

  function startExpandTimer() {
    // Only start timer if folder is not expanded and is a directory
    if (!isDirectory.value || isExpanded.value) {
      return;
    }

    // Clear any existing timer
    clearExpandTimer();

    // Start new timer
    expandTimer = setTimeout(() => {
      // Double-check we're still dragging over and not expanded
      if (isDraggingOver.value && !isExpanded.value) {
        emit("toggle", props.node, true);
        emit("open", props.node, { ensureExpanded: true });
      }
      expandTimer = null;
    }, EXPAND_DELAY);
  }

  function handleDragEnter(event: DragEvent) {
    if (!isDirectory.value) return;
    // Check if dragging files or zip entries
    const hasFiles = event.dataTransfer?.types.includes("Files");
    const hasZipEntry = event.dataTransfer?.types.includes("application/x-zip-entry");
    if (hasFiles || hasZipEntry) {
      event.preventDefault();
      // Don't stop propagation - let root handler show the overlay
      isDraggingOver.value = true;
      
      // Set drop effect
      if (event.dataTransfer) {
        event.dataTransfer.dropEffect = "copy";
      }

      // Start timer to auto-expand folder after delay
      startExpandTimer();
    }
  }

  function handleDragOver(event: DragEvent) {
    if (!isDirectory.value) return;
    // Check if dragging files or zip entries
    const hasFiles = event.dataTransfer?.types.includes("Files");
    const hasZipEntry = event.dataTransfer?.types.includes("application/x-zip-entry");
    if (hasFiles || hasZipEntry) {
      event.preventDefault();
      // Don't stop propagation - let root handler show the overlay
      isDraggingOver.value = true;
      
      // Set drop effect
      if (event.dataTransfer) {
        event.dataTransfer.dropEffect = "copy";
      }

      // Restart timer if not already expanded (in case timer was cleared)
      if (!isExpanded.value && !expandTimer) {
        startExpandTimer();
      }
    }
  }

  function handleDragLeave(event: DragEvent) {
    if (!isDirectory.value) return;
    event.preventDefault();
    event.stopPropagation();
    
    // Only clear if we're actually leaving the branch element
    const relatedTarget = event.relatedTarget as HTMLElement | null;
    const currentTarget = event.currentTarget as HTMLElement | null;
    
    // Check if we're moving to a child element - if so, don't clear
    if (currentTarget && relatedTarget && currentTarget.contains(relatedTarget)) {
      return; // Still within the branch, keep timer running
    }
    
    // We're actually leaving, clear everything
    clearExpandTimer();
    isDraggingOver.value = false;
  }

  function handleDrop(event: DragEvent) {
    if (!isDirectory.value) return;
    event.preventDefault();
    event.stopPropagation();
    isDraggingOver.value = false;
    clearExpandTimer();

    const files = Array.from(event.dataTransfer?.files || []);
    if (files.length > 0) {
      emit("drop-files", props.node, files, event);
    }
  }

  function clearExpandTimer() {
    if (expandTimer) {
      clearTimeout(expandTimer);
      expandTimer = null;
    }
  }

  // Cleanup timer on unmount
  onUnmounted(() => {
    clearExpandTimer();
  });
</script>

<template>
  <TreeView.NodeProvider :node="props.node" :indexPath="props.indexPath">
    <template v-if="isDirectory">
      <TreeView.Branch
        class="tree-node"
        :class="{
          'is-selected': isSelected,
          'is-directory': true,
          'is-loading': node.isLoading,
          'is-dragging-over': isDraggingOver,
        }"
        :open="isExpanded"
        @dragenter="handleDragEnter"
        @dragover="handleDragOver"
        @dragleave="handleDragLeave"
        @drop="handleDrop"
      >
        <TreeView.BranchTrigger
          :style="{ paddingLeft: iconPadding }"
          class="tree-trigger"
          :class="{ 
            'is-dragging-over': isDraggingOver,
            'is-selected': isSelected,
          }"
          @click.stop.prevent="handleBranchClick"
          @mousedown.stop.prevent
          @selectstart.prevent
          @contextmenu="handleContextMenu"
        >
          <span class="tree-trigger__chevron">
            <TreeView.BranchIndicator>
              <ChevronRightIcon v-if="!isExpanded" class="chevron" />
              <ChevronDownIcon v-else class="chevron" />
            </TreeView.BranchIndicator>
          </span>
          <span class="tree-trigger__icon">
            <FolderIcon v-if="!isExpanded && !isDraggingOver" class="icon" />
            <FolderOpenIcon v-else-if="isExpanded && !isDraggingOver" class="icon icon--expanded" />
            <ArrowUpTrayIcon v-if="isDraggingOver" class="icon icon--upload" />
          </span>
          <span class="tree-trigger__label">
            <span class="tree-trigger__name">
              {{ node.name || "/" }}
              <span v-if="isDraggingOver" class="tree-trigger__drop-indicator">
                Drop files here
              </span>
              <span v-if="isSymlink" class="tree-trigger__symlink">
                <MinusSmallIcon class="symlink-arrow" />
                <span class="symlink-target">{{ node.symlinkTarget }}</span>
              </span>
            </span>
            <span v-if="displaySize" class="tree-trigger__meta">
              {{ displaySize }}
            </span>
          </span>
          <span class="tree-trigger__actions">
            <ArrowPathIcon v-if="node.isLoading" class="action-icon animate-spin" />
            <OuiMenu v-if="menuSections.length" v-model:open="menuOpen">
              <template #trigger>
                <button
                  ref="menuTriggerRef"
                  type="button"
                  class="action-button"
                  style="position: fixed; opacity: 0; pointer-events: none; width: 1px; height: 1px;"
                  @click.stop
                />
              </template>
              <div class="oui-context-menu-list">
                <!-- Show header for single or multiple selection -->
                <div
                  v-if="selectedNodes && selectedNodes.size > 1"
                  class="selection-header"
                >
                  <div class="selection-header__count">
                    {{ selectedNodes.size }} item{{ selectedNodes.size > 1 ? 's' : '' }} selected
                  </div>
                </div>
                <div
                  v-else
                  class="selection-header"
                >
                  <div class="selection-header__name">
                    {{ node.name || node.path.split('/').pop() || node.path }}
                  </div>
                </div>
                <template v-for="(item, idx) in menuSections" :key="item.key">
                  <OuiMenuSeparator v-if="item.separatorBefore && idx !== 0" />
                  <OuiMenuItem
                    :value="item.key"
                    :shortcut="item.shortcut"
                    :disabled="item.disabled"
                    @select="handleMenuSelect(item.key)"
                  >
                    {{ item.label }}
                  </OuiMenuItem>
                </template>
              </div>
            </OuiMenu>
          </span>
        </TreeView.BranchTrigger>

        <TreeView.BranchContent v-if="isExpanded">
          <TreeView.BranchIndentGuide />
          <div
            class="tree-children-wrapper"
            :class="{ 'is-dragging-over-parent': isDraggingOver }"
          >
            <FileTreeNode
              v-for="(child, idx) in node.children"
              :key="child.id"
              :node="child"
              :indexPath="[...props.indexPath, idx]"
              :selectedPath="selectedPath"
              :selectedNodes="selectedNodes"
              :allowEditing="props.allowEditing ?? true"
              @toggle="(n, open) => emit('toggle', n, open)"
              @open="(n, options) => emit('open', n, options)"
              @select="(n, event) => emit('select', n, event)"
              @action="(action, n) => emit('action', action, n)"
              @load-more="(n) => emit('load-more', n)"
              @drop-files="(n, files) => emit('drop-files', n, files)"
            />
          </div>
          <button
            v-if="node.hasMore"
            class="tree-load-more"
            :disabled="node.isLoading"
            @click="handleLoadMore"
          >
            <ArrowPathIcon class="mr-1 h-4 w-4" />
            <span>{{ node.isLoading ? "Loading…" : "Load more" }}</span>
          </button>
        </TreeView.BranchContent>
      </TreeView.Branch>
    </template>
    <template v-else>
      <TreeView.Item
        class="tree-node"
        :class="{
          'is-selected': isSelected,
          'is-loading': node.isLoading,
        }"
      >
        <div
          :style="{ paddingLeft: iconPadding }"
          class="tree-trigger"
          :class="{ 'is-selected': isSelected }"
          @click.stop.prevent="handleItemClick"
          @mousedown.stop.prevent
          @selectstart.prevent
          @contextmenu="handleContextMenu"
        >
          <span class="tree-trigger__chevron tree-trigger__chevron--placeholder" />
          <span class="tree-trigger__icon">
            <DocumentIcon class="icon" />
          </span>
          <span class="tree-trigger__label">
            <span class="tree-trigger__name">
              {{ node.name || "/" }}
              <span v-if="isSymlink" class="tree-trigger__symlink">
                <MinusSmallIcon class="symlink-arrow" />
                <span class="symlink-target">{{ node.symlinkTarget }}</span>
              </span>
            </span>
            <span v-if="displaySize" class="tree-trigger__meta">
              {{ displaySize }}
            </span>
          </span>
          <span class="tree-trigger__actions">
            <ArrowPathIcon v-if="node.isLoading" class="action-icon animate-spin" />
            <OuiMenu v-if="menuSections.length" v-model:open="menuOpen">
              <template #trigger>
                <button
                  ref="menuTriggerRef"
                  type="button"
                  class="action-button"
                  style="position: fixed; opacity: 0; pointer-events: none; width: 1px; height: 1px;"
                  @click.stop
                />
              </template>
              <div class="oui-context-menu-list">
                <!-- Show header for single or multiple selection -->
                <div
                  v-if="selectedNodes && selectedNodes.size > 1"
                  class="selection-header"
                >
                  <div class="selection-header__count">
                    {{ selectedNodes.size }} item{{ selectedNodes.size > 1 ? 's' : '' }} selected
                  </div>
                </div>
                <div
                  v-else
                  class="selection-header"
                >
                  <div class="selection-header__name">
                    {{ node.name || node.path.split('/').pop() || node.path }}
                  </div>
                </div>
                <template v-for="(item, idx) in menuSections" :key="item.key">
                  <OuiMenuSeparator v-if="item.separatorBefore && idx !== 0" />
                  <OuiMenuItem
                    :value="item.key"
                    :shortcut="item.shortcut"
                    :disabled="item.disabled"
                    @select="handleMenuSelect(item.key)"
                  >
                    {{ item.label }}
                  </OuiMenuItem>
                </template>
              </div>
            </OuiMenu>
          </span>
        </div>
      </TreeView.Item>
    </template>
  </TreeView.NodeProvider>
</template>

<style scoped>
  .tree-node {
    display: flex;
    flex-direction: column;
    color: var(--oui-text-secondary);
    font-size: 13px;
    position: relative;
    user-select: none;
    -webkit-user-select: none;
    -moz-user-select: none;
    -ms-user-select: none;
  }

  /* Selection styling - only apply to the direct trigger of the selected node */
  .tree-node.is-selected > .tree-trigger,
  .tree-node.is-selected > TreeView.BranchTrigger {
    position: relative;
    background: var(--oui-surface-selected) !important;
    border: none !important;
    border-radius: 6px;
    color: var(--oui-text-primary);
  }

  .tree-node.is-selected > .tree-trigger::before,
  .tree-node.is-selected > TreeView.BranchTrigger::before {
    content: '';
    position: absolute;
    inset: 0;
    background: var(--oui-accent-primary);
    opacity: 0.12;
    pointer-events: none;
    border-radius: 4px;
    z-index: 0;
  }

  /* Ensure children don't inherit selection styling - be very specific */
  .tree-node.is-selected .tree-children-wrapper,
  .tree-node.is-selected .tree-children-wrapper .tree-node,
  .tree-node.is-selected .tree-children-wrapper .tree-node .tree-trigger,
  .tree-node.is-selected TreeView.BranchContent .tree-node,
  .tree-node.is-selected TreeView.BranchContent .tree-node .tree-trigger {
    background: transparent !important;
    border: none !important;
  }

  .tree-node.is-selected .tree-children-wrapper .tree-node .tree-trigger::before,
  .tree-node.is-selected TreeView.BranchContent .tree-node .tree-trigger::before {
    display: none !important;
  }

  /* .tree-trigger.is-selected is used for file items (TreeView.Item) */
  .tree-trigger.is-selected {
    position: relative;
    background: var(--oui-surface-selected) !important;
    border: none !important;
    border-radius: 6px;
  }

  .tree-trigger.is-selected::before {
    content: '';
    position: absolute;
    inset: 0;
    background: var(--oui-accent-primary);
    opacity: 0.12;
    pointer-events: none;
    border-radius: 4px;
    z-index: 0;
  }

  /* Ensure selection styling takes priority over directory/expanded states */
  .tree-node.is-selected.is-directory > .tree-trigger,
  .tree-node.is-selected.is-directory > TreeView.BranchTrigger,
  .tree-node.is-selected[data-state="open"] > .tree-trigger,
  .tree-node.is-selected[data-state="open"] > TreeView.BranchTrigger,
  .tree-node.is-selected[data-state="closed"] > .tree-trigger,
  .tree-node.is-selected[data-state="closed"] > TreeView.BranchTrigger {
    background: var(--oui-surface-selected) !important;
    border: none !important;
  }

  .tree-node.is-dragging-over {
    background: transparent;
    position: relative;
  }

  /* Overlay on the folder row itself */
  .tree-node.is-dragging-over .tree-trigger {
    background: transparent;
    position: relative;
  }

  .tree-node.is-dragging-over .tree-trigger::before {
    content: '';
    position: absolute;
    inset: 0;
    background: var(--oui-accent-primary);
    opacity: 0.12;
    pointer-events: none;
    border-radius: 6px;
    z-index: 0;
  }

  .tree-node.is-dragging-over .tree-trigger > * {
    position: relative;
    z-index: 1;
  }

  /* Shared subtle overlay wrapper for all children */
  .tree-children-wrapper.is-dragging-over-parent {
    position: relative;
    margin: 2px 0;
    padding: 2px 0;
  }

  .tree-children-wrapper.is-dragging-over-parent::before {
    content: '';
    position: absolute;
    inset: 0;
    background: var(--oui-accent-primary);
    opacity: 0.06;
    pointer-events: none;
    border-radius: 4px;
    z-index: 0;
  }

  .tree-children-wrapper.is-dragging-over-parent > * {
    position: relative;
    z-index: 1;
  }

  .tree-node.is-dragging-over .tree-trigger__name {
    font-weight: 500;
    color: var(--oui-text-primary);
  }

  .tree-trigger__drop-indicator {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    margin-left: 8px;
    padding: 2px 6px;
    background: var(--oui-surface-muted);
    border: 1px solid var(--oui-border-default);
    border-radius: 4px;
    font-size: 11px;
    font-weight: 500;
    color: var(--oui-text-secondary);
    animation: fadeIn 0.2s ease-in;
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
      transform: translateY(-2px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .icon--upload {
    color: var(--oui-text-secondary);
  }

  @keyframes bounce {
    0%, 100% {
      transform: translateY(0);
    }
    50% {
      transform: translateY(-2px);
    }
  }

  .tree-trigger {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 30px;
    padding-right: 8px;
    cursor: pointer;
    border-radius: 6px;
    transition: background-color 0.12s ease;
    position: relative;
    user-select: none;
    -webkit-user-select: none;
    -moz-user-select: none;
    -ms-user-select: none;
  }

  .tree-trigger::before {
    z-index: 0;
  }

  .tree-trigger > * {
    position: relative;
    z-index: 1;
  }

  .tree-trigger:hover {
    background: var(--oui-surface-hover);
    position: relative;
  }

  .tree-trigger:hover::before {
    content: '';
    position: absolute;
    inset: 0;
    background: var(--oui-accent-primary);
    opacity: 0.04;
    pointer-events: none;
    border-radius: 6px;
    z-index: 0;
  }

  .tree-node.is-selected > .tree-trigger:hover,
  .tree-node.is-selected > TreeView.BranchTrigger:hover {
    background: var(--oui-surface-selected) !important;
    border: none !important;
  }

  .tree-node.is-selected > .tree-trigger:hover::before,
  .tree-node.is-selected > TreeView.BranchTrigger:hover::before {
    opacity: 0.12;
  }

  .tree-trigger.is-selected:hover {
    background: var(--oui-surface-selected) !important;
    border: none !important;
  }

  .tree-trigger.is-selected:hover::before {
    opacity: 0.12;
  }

  .tree-trigger__chevron {
    width: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .tree-trigger__chevron--placeholder {
    visibility: hidden;
  }

  .chevron {
    height: 14px;
    width: 14px;
    color: var(--oui-text-tertiary);
    transition: color 0.12s ease;
  }

  .tree-trigger__icon {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .icon {
    height: 16px;
    width: 16px;
    color: var(--oui-text-tertiary);
  }

  .icon--expanded {
    color: var(--oui-accent-primary);
  }

  .tree-trigger__label {
    display: flex;
    align-items: center;
    gap: 6px;
    flex: 1;
    min-width: 0;
  }

  .tree-trigger__name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--oui-text-primary);
  }

  .tree-trigger__symlink {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    color: var(--oui-text-tertiary);
    margin-left: 6px;
  }

  .symlink-arrow {
    height: 12px;
    width: 12px;
  }

  .symlink-target {
    font-family: var(--oui-font-mono);
    font-size: 11px;
  }

  .tree-trigger__meta {
    font-size: 11px;
    color: var(--oui-text-tertiary);
  }

  .tree-trigger__actions {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    color: var(--oui-text-tertiary);
  }

  .action-icon {
    height: 14px;
    width: 14px;
  }

  .action-button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: inherit;
    cursor: pointer;
  }

  .action-button:hover {
    background: var(--oui-surface-hover);
    color: var(--oui-text-primary);
  }

  .tree-load-more {
    display: flex;
    align-items: center;
    gap: 6px;
    margin: 4px 0 4px 28px;
    padding: 4px 8px;
    border-radius: 6px;
    font-size: 12px;
    color: var(--oui-text-secondary);
    background: transparent;
    border: 1px dashed var(--oui-border-muted);
    transition: background-color 0.12s ease, color 0.12s ease;
  }

  .tree-load-more:hover:not(:disabled) {
    background: var(--oui-surface-hover);
    color: var(--oui-text-primary);
  }

  .selection-header {
    padding: 8px 12px;
    font-size: 12px;
    font-weight: 500;
    color: var(--oui-text-secondary);
    background: var(--oui-surface-raised);
    border-bottom: 1px solid var(--oui-border-default);
    margin: 0;
    user-select: none;
  }

  .selection-header__count {
    font-weight: 600;
    color: var(--oui-text-primary);
  }

  .selection-header__name {
    font-weight: 600;
    color: var(--oui-text-primary);
  }

  .tree-load-more:disabled {
    opacity: 0.6;
    cursor: wait;
  }
</style>

