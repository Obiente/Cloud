<script setup lang="ts">
  import { computed } from "vue";
  import { TreeView } from "@ark-ui/vue/tree-view";

  import {
    FolderIcon,
    FolderOpenIcon,
    DocumentIcon,
    ChevronRightIcon,
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
    allowEditing: boolean;
  }>();

  const emit = defineEmits<{
    (e: "toggle", node: ExplorerNode): void;
    (e: "open", node: ExplorerNode): void;
    (e: "action", action: string, node: ExplorerNode): void;
    (e: "load-more", node: ExplorerNode): void;
  }>();

  const isDirectory = computed(() => props.node.type === "directory");
  const isSymlink = computed(() => props.node.type === "symlink");
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
      if (props.allowEditing) {
        add({ key: "new-file", label: "New File", shortcut: "N", separatorBefore: true });
        add({ key: "new-folder", label: "New Folder" });
        add({ key: "new-symlink", label: "New Symlink" });
      }
    } else {
      add({
        key: "open-editor",
        label: props.allowEditing ? "Open in Editor" : "View",
        disabled: !props.allowEditing,
      });
    }

    add({ key: "copy-path", label: "Copy Path", separatorBefore: true });

    if (props.allowEditing && props.node.path !== "/") {
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

  function handleBranchClick(event: Event) {
    event.preventDefault();
    event.stopPropagation();
    emit("toggle", props.node);
  }

  function handleItemClick(event: Event) {
    event.preventDefault();
    event.stopPropagation();
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
</script>

<template>
  <TreeView.NodeProvider :node="props.node" :indexPath="props.indexPath">
    <component
      :is="isDirectory ? TreeView.Branch : TreeView.Item"
      class="tree-node"
      :class="{
        'is-selected': isSelected,
        'is-directory': isDirectory,
        'is-loading': node.isLoading,
      }"
    >
      <component
        :is="isDirectory ? TreeView.BranchTrigger : 'div'"
        :style="{ paddingLeft: iconPadding }"
        class="tree-trigger"
        @click="isDirectory ? handleBranchClick($event) : handleItemClick($event)"
      >
        <span class="tree-trigger__chevron" v-if="isDirectory">
          <TreeView.BranchIndicator>
            <ChevronRightIcon class="chevron" />
          </TreeView.BranchIndicator>
        </span>
        <span class="tree-trigger__icon">
          <TreeView.NodeContext v-if="isDirectory" v-slot="{ expanded }">
            <FolderIcon v-if="!expanded" class="icon" />
            <FolderOpenIcon v-else class="icon icon--expanded" />
          </TreeView.NodeContext>
          <DocumentIcon v-else class="icon" />
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
      </component>

      <TreeView.BranchContent v-if="isDirectory">
        <TreeView.BranchIndentGuide />
        <FileTreeNode
          v-for="(child, idx) in node.children"
          :key="child.id"
          :node="child"
          :indexPath="[...props.indexPath, idx]"
          :selectedPath="selectedPath"
          :allowEditing="allowEditing"
          @toggle="(n) => emit('toggle', n)"
          @open="(n) => emit('open', n)"
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
    </component>
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

  .chevron {
    height: 14px;
    width: 14px;
    color: var(--oui-text-tertiary);
    transition: transform 0.12s ease;
  }

  [data-state="open"] .chevron {
    transform: rotate(90deg);
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
