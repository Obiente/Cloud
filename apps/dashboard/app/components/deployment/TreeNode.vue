<script setup lang="ts">
  import { computed } from "vue";
  import { TreeView } from "@ark-ui/vue/tree-view";

  import {
    FolderIcon,
    FolderOpenIcon,
    DocumentIcon,
    ChevronRightIcon,
    ChevronDownIcon,
    MinusSmallIcon,
    LinkIcon,
    EllipsisVerticalIcon,
    ArrowPathIcon,
  } from "@heroicons/vue/24/outline";
  import type { ExplorerNode } from "./fileExplorerTypes";

  defineOptions({
    name: "FileTreeNode",
  });

  const props = defineProps<{
    node: ExplorerNode;
    indexPath: number[];
    selectedPath: string | null;
    allowEditing?: boolean;
  }>();

  const emit = defineEmits<{
    (e: "toggle", node: ExplorerNode, open: boolean): void;
    (e: "open", node: ExplorerNode, options?: { ensureExpanded?: boolean }): void;
    (e: "action", action: string, node: ExplorerNode): void;
    (e: "load-more", node: ExplorerNode): void;
  }>();

  const isDirectory = computed(() => props.node.type === "directory" || !!props.node.children?.length);
  const isSymlink = computed(() => props.node.type === "symlink");
  const isExpanded = computed(() => !!props.node.isExpanded);
  const isSelected = computed(() => props.selectedPath === props.node.path);
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

    add({ key: "copy-path", label: "Copy Path", separatorBefore: true });

    if (props.node.path !== "/") {
      add({ key: "rename", label: "Rename", shortcut: "F2", separatorBefore: true });
      add({ key: "delete", label: "Delete", shortcut: "Del" });
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
    // Stop propagation to prevent parent nodes from toggling
    event.stopPropagation();
    // In controlled mode, we need to manually toggle since Ark UI won't fire open-change
    const newOpenState = !props.node.isExpanded;
    emit("toggle", props.node, newOpenState);
    if (newOpenState) {
      emit("open", props.node, { ensureExpanded: true });
    }
  }

  function handleItemClick() {
    emit("open", props.node);
  }

  function handleMenuSelect(action: string) {
    emit("action", action, props.node);
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
        }"
        :open="isExpanded"
      >
        <TreeView.BranchTrigger
          :style="{ paddingLeft: iconPadding }"
          class="tree-trigger"
          @click="handleBranchClick"
        >
          <span class="tree-trigger__chevron">
            <TreeView.BranchIndicator>
              <ChevronRightIcon v-if="!isExpanded" class="chevron" />
              <ChevronDownIcon v-else class="chevron" />
            </TreeView.BranchIndicator>
          </span>
          <span class="tree-trigger__icon">
            <FolderIcon v-if="!isExpanded" class="icon" />
            <FolderOpenIcon v-else class="icon icon--expanded" />
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
            <span v-if="modeString" class="tree-trigger__meta">
              {{ modeString }}
            </span>
            <span v-if="node.owner" class="tree-trigger__meta">
              {{ node.owner }}
            </span>
          </span>
          <span class="tree-trigger__actions">
            <ArrowPathIcon v-if="node.isLoading" class="action-icon animate-spin" />
            <OuiMenu v-if="menuSections.length">
              <template #trigger>
                <button type="button" class="action-button" @click.stop>
                  <EllipsisVerticalIcon class="action-icon" />
                </button>
              </template>
              <div class="oui-context-menu-list">
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
          <FileTreeNode
            v-for="(child, idx) in node.children"
            :key="child.id"
            :node="child"
            :indexPath="[...props.indexPath, idx]"
            :selectedPath="selectedPath"
            :allowEditing="props.allowEditing ?? true"
            @toggle="(n, open) => emit('toggle', n, open)"
            @open="(n, options) => emit('open', n, options)"
            @action="(action, n) => emit('action', action, n)"
            @load-more="(n) => emit('load-more', n)"
          />
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
          @click="handleItemClick"
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
            <span v-if="modeString" class="tree-trigger__meta">
              {{ modeString }}
            </span>
            <span v-if="node.owner" class="tree-trigger__meta">
              {{ node.owner }}
            </span>
          </span>
          <span class="tree-trigger__actions">
            <ArrowPathIcon v-if="node.isLoading" class="action-icon animate-spin" />
            <OuiMenu v-if="menuSections.length">
              <template #trigger>
                <button type="button" class="action-button" @click.stop>
                  <EllipsisVerticalIcon class="action-icon" />
                </button>
              </template>
              <div class="oui-context-menu-list">
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
  }

  .tree-node.is-selected {
    color: var(--oui-text-primary);
    background: var(--oui-surface-selected);
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
  }

  .tree-trigger:hover {
    background: var(--oui-surface-hover);
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

  .tree-load-more:disabled {
    opacity: 0.6;
    cursor: wait;
  }
</style>
