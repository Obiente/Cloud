<template>
  <aside
    class="flex flex-col border border-default rounded-[10px] bg-surface-base overflow-hidden"
    :class="[
      mobileClass
    ]"
    aria-label="File tree"
    @dragenter="handleRootDragEnter"
    @dragover="handleRootDragOver"
    @dragleave="handleRootDragLeave"
    @drop="handleRootDrop"
  >
    <div class="p-3 border-b border-default">
      <OuiFlex v-if="showMobileToggle" justify="between" align="center" class="mb-2">
        <OuiText
          size="xs"
          weight="semibold"
          class="uppercase tracking-[0.08em] text-[11px]"
          >Sources</OuiText
        >
        <OuiButton
          variant="ghost"
          size="xs"
          class="lg:hidden p-1"
          @click="$emit('toggle-mobile')"
          :aria-expanded="showMobileSidebar"
        >
          <XMarkIcon v-if="showMobileSidebar" class="h-4 w-4" />
          <span v-else class="text-xs">Show</span>
        </OuiButton>
      </OuiFlex>
      <OuiText
        v-else
        size="xs"
        weight="semibold"
        class="uppercase tracking-[0.08em] text-[11px]"
        >Sources</OuiText
      >
    </div>

    <!-- Sources Section -->
    <div class="p-3 border-b border-border-default">
      <nav class="flex flex-col gap-1.5">
        <button
          class="flex items-center gap-2 px-2.5 py-1.5 rounded-xl text-[13px] text-left transition-all duration-150 text-text-secondary border bg-transparent cursor-pointer hover:bg-surface-hover hover:text-text-primary disabled:opacity-60 disabled:cursor-not-allowed"
          :class="{
            'bg-surface-selected text-text-primary is-selected-source':
              source.type === 'container',
            'is-hovering-volume': hoveredVolume === 'container' && source.type !== 'container',
            'is-dragging-over-source': isDraggingOverSource === 'container',
          }"
          :style="{
            borderColor: (source.type === 'container')
              ? 'var(--oui-accent-primary)'
              : (hoveredVolume === 'container')
              ? 'var(--oui-border-strong)'
              : 'var(--oui-border-default)'
          }"
          :disabled="!containerRunning"
          @click="$emit('switch-source', 'container')"
          @mouseenter="handleVolumeHoverEnter('container')"
          @mouseleave="handleVolumeHoverLeave"
          @dragenter="handleSourceDragEnter('container', $event)"
          @dragover="handleSourceDragOver($event)"
          @dragleave="handleSourceDragLeave($event)"
          @drop="handleSourceDrop('container', $event)"
        >
          <ServerIcon class="h-4 w-4" />
          <span>Container filesystem</span>
        </button>
        <button
          v-for="volume in volumes"
          :key="volume.name"
          class="flex items-center gap-2 px-2.5 py-1.5 rounded-xl text-[13px] text-left transition-all duration-150 text-text-secondary border bg-transparent cursor-pointer hover:bg-surface-hover hover:text-text-primary"
          :class="{
            'bg-surface-selected text-text-primary is-selected-source':
              source.type === 'volume' && source.volumeName === volume.name,
            'is-hovering-volume': hoveredVolume === volume.name && (source.type !== 'volume' || source.volumeName !== volume.name),
            'is-dragging-over-source': isDraggingOverSource === volume.name,
          }"
          :style="{
            borderColor: source.type === 'volume' && source.volumeName === volume.name
              ? 'var(--oui-accent-primary)'
              : hoveredVolume === volume.name && (source.type !== 'volume' || source.volumeName !== volume.name)
              ? 'var(--oui-border-strong)'
              : 'var(--oui-border-default)'
          }"
          @click="$emit('switch-source', 'volume', volume.name || '')"
          @mouseenter="handleVolumeHoverEnter(volume.name || '')"
          @mouseleave="handleVolumeHoverLeave"
          @dragenter="handleSourceDragEnter(volume.name || '', $event)"
          @dragover="handleSourceDragOver($event)"
          @dragleave="handleSourceDragLeave($event)"
          @drop="handleSourceDrop(volume.name || '', $event)"
        >
          <CubeIcon class="h-4 w-4" />
          <span>{{ getVolumeLabel(volume) }}</span>
          <span v-if="getVolumeSecondaryLabel(volume)" class="ml-auto text-[11px] text-text-tertiary">
            {{ getVolumeSecondaryLabel(volume) }}
          </span>
        </button>
      </nav>
    </div>

    <!-- File Tree Section -->
      <div 
        class="flex-1 overflow-y-auto font-mono" 
        role="tree"
        style="user-select: none;"
      >
      <div class="p-2">
        <div
          v-if="errorMessage"
          class="mb-2 p-2 rounded-xl bg-danger/10 border border-danger/30"
        >
          <div class="flex items-start gap-2">
            <ExclamationTriangleIcon
              class="h-4 w-4 text-danger shrink-0 mt-0.5"
            />
            <div class="flex-1 min-w-0">
              <OuiText
                size="xs"
                weight="semibold"
                color="danger"
                class="block mb-0.5"
              >
                Error loading files
              </OuiText>
              <OuiText size="xs" color="secondary" class="wrap-break-word">
                {{ parseError ? parseError(errorMessage) : errorMessage }}
              </OuiText>
            </div>
            <OuiButton
              variant="ghost"
              size="xs"
              class="shrink-0 -mt-1 -mr-1"
              @click="$emit('clear-error')"
            >
              <XMarkIcon class="h-3.5 w-3.5" />
            </OuiButton>
          </div>
        </div>
        <template v-if="root.children.length === 0 && isLoadingTree">
          <OuiFlex
            direction="col"
            align="center"
            gap="sm"
            class="tree-empty"
          >
            <ArrowPathIcon class="h-5 w-5 animate-spin" />
            <OuiText size="sm" color="secondary">Loading files…</OuiText>
          </OuiFlex>
        </template>
        <template v-else-if="root.children.length === 0">
          <OuiFlex
            direction="col"
            align="center"
            gap="sm"
            class="tree-empty"
          >
            <OuiText size="sm" color="secondary">No files found</OuiText>
          </OuiFlex>
        </template>
        <template v-else>
          <TreeView.Root
            :collection="treeCollection"
            class="file-tree-root"
          >
            <TreeView.Tree>
              <TreeNode
                v-for="(child, idx) in root.children"
                :key="child.id"
                :node="child"
                :indexPath="[idx]"
                :selectedPath="selectedPath"
                :selectedNodes="selectedNodes"
                :allowEditing="allowEditing"
                @toggle="(node, open) => $emit('toggle', node, open)"
                @open="(node, options) => $emit('open', node, options)"
                @select="(node, event) => $emit('select', node, event)"
                @action="(action, node, selectedPaths) => $emit('action', action, node, selectedPaths)"
                @load-more="(node) => $emit('load-more', node)"
                @drop-files="(node: ExplorerNode, files: File[], event?: DragEvent) => $emit('drop-files', node, files, event)"
              />
            </TreeView.Tree>
          </TreeView.Root>
        </template>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
  import { ref, computed } from "vue";
  import { TreeView } from "@ark-ui/vue/tree-view";
  import {
    ServerIcon,
    CubeIcon,
    XMarkIcon,
    ExclamationTriangleIcon,
    ArrowPathIcon,
  } from "@heroicons/vue/24/outline";
  import TreeNode from "./TreeNode.vue";
  import type { ExplorerNode } from "./fileExplorerTypes";
  import type { TreeCollection } from "@ark-ui/vue/collection";

  interface SourceState {
    type: "container" | "volume";
    volumeName?: string;
  }

  interface Volume {
    name?: string;
    mountPoint?: string;
  }

  const props = withDefaults(defineProps<{
    source: SourceState;
    volumes: Volume[];
    root: ExplorerNode;
    selectedPath: string | null;
    selectedNodes?: Set<string>;
    treeCollection: TreeCollection<any>;
    errorMessage: string | null;
    isLoadingTree: boolean;
    containerRunning: boolean;
    allowEditing?: boolean;
    showMobileToggle?: boolean;
    showMobileSidebar?: boolean;
    mobileClass?: string;
    getVolumeLabel?: (volume: Volume) => string;
    getVolumeSecondaryLabel?: (volume: Volume) => string | null;
    parseError?: (error: string) => string;
  }>(), {
    allowEditing: true,
    showMobileToggle: false,
    showMobileSidebar: false,
    mobileClass: "",
    getVolumeLabel: (volume: Volume) => volume.mountPoint || volume.name || "",
    getVolumeSecondaryLabel: () => null,
    parseError: undefined,
  });

  const emit = defineEmits<{
    (e: "switch-source", type: "container" | "volume", name?: string): void;
    (e: "toggle", node: ExplorerNode, open: boolean): void;
    (e: "open", node: ExplorerNode, options?: { ensureExpanded?: boolean }): void;
    (e: "select", node: ExplorerNode, event: MouseEvent): void;
    (e: "action", action: string, node: ExplorerNode, selectedPaths?: string[]): void;
    (e: "load-more", node: ExplorerNode): void;
    (e: "drop-files", node: ExplorerNode, files: File[], event?: DragEvent): void;
    (e: "root-drop", files: File[], event?: DragEvent): void;
    (e: "source-drop", sourceName: string, files: File[], event?: DragEvent): void;
    (e: "toggle-mobile"): void;
    (e: "clear-error"): void;
  }>();

  const VOLUME_SWITCH_DELAY = 800; // 800ms hover delay before switching volume
  const volumeSwitchTimer = ref<ReturnType<typeof setTimeout> | null>(null);
  const hoveredVolume = ref<string | null>(null);
  const isDraggingOverRoot = ref(false);
  const isDraggingOverSource = ref<string | null>(null);

  function handleVolumeHoverEnter(volumeName: string) {
    // Only switch if it's different from current source
    const shouldSwitch = 
      (volumeName === 'container' && props.source.type !== 'container') ||
      (volumeName !== 'container' && (props.source.type !== 'volume' || props.source.volumeName !== volumeName));

    if (!shouldSwitch) return;

    hoveredVolume.value = volumeName;
    clearVolumeSwitchTimer();
    
    volumeSwitchTimer.value = setTimeout(() => {
      if (hoveredVolume.value === volumeName) {
        if (volumeName === 'container') {
          emit('switch-source', 'container');
        } else {
          emit('switch-source', 'volume', volumeName);
        }
      }
      clearVolumeSwitchTimer();
    }, VOLUME_SWITCH_DELAY);
  }

  function handleVolumeHoverLeave() {
    hoveredVolume.value = null;
    clearVolumeSwitchTimer();
  }

  function clearVolumeSwitchTimer() {
    if (volumeSwitchTimer.value) {
      clearTimeout(volumeSwitchTimer.value);
      volumeSwitchTimer.value = null;
    }
  }

  function handleRootDragEnter(event: DragEvent) {
    const hasFiles = event.dataTransfer?.types.includes("Files");
    const hasZipEntry = event.dataTransfer?.types.includes("application/x-zip-entry");
    if (hasFiles || hasZipEntry) {
      const target = event.target as HTMLElement;
      const isOverFolder = target.closest('.tree-node.is-directory');
      const isOverSource = target.closest('nav button');
      
      // Only show root overlay if not over a folder or source button
      if (!isOverFolder && !isOverSource) {
        event.preventDefault();
        isDraggingOverRoot.value = true;
        
        if (event.dataTransfer) {
          event.dataTransfer.dropEffect = "copy";
        }
      } else {
        isDraggingOverRoot.value = false;
      }
    }
  }

  function handleRootDragOver(event: DragEvent) {
    const hasFiles = event.dataTransfer?.types.includes("Files");
    const hasZipEntry = event.dataTransfer?.types.includes("application/x-zip-entry");
    if (hasFiles || hasZipEntry) {
      const target = event.target as HTMLElement;
      const isOverFolder = target.closest('.tree-node.is-directory');
      const isOverSource = target.closest('nav button');
      
      // Only show root overlay if not over a folder or source button
      if (!isOverFolder && !isOverSource) {
        event.preventDefault();
        isDraggingOverRoot.value = true;
        
        if (event.dataTransfer) {
          event.dataTransfer.dropEffect = "copy";
        }
      } else {
        isDraggingOverRoot.value = false;
      }
    }
  }

  function handleRootDragLeave(event: DragEvent) {
    const relatedTarget = event.relatedTarget as HTMLElement | null;
    const currentTarget = event.currentTarget as HTMLElement | null;
    const target = event.target as HTMLElement;
    const isOverSource = target?.closest('nav button');
    
    // Don't clear if moving to another part of the tree
    if (currentTarget && relatedTarget && currentTarget.contains(relatedTarget)) {
      return;
    }
    
    // Only clear if actually leaving the tree area (not going to a source button)
    if (!isOverSource) {
      isDraggingOverRoot.value = false;
    }
  }

  function handleRootDrop(event: DragEvent) {
    event.preventDefault();
    event.stopPropagation();
    isDraggingOverRoot.value = false;

    const files = Array.from(event.dataTransfer?.files || []);
    // Emit even if no files (might be zip entry that will be extracted)
    emit('root-drop', files, event);
  }

  function handleSourceDragEnter(sourceName: string, event: DragEvent) {
    const hasFiles = event.dataTransfer?.types.includes("Files");
    const hasZipEntry = event.dataTransfer?.types.includes("application/x-zip-entry");
    if (hasFiles || hasZipEntry) {
      event.preventDefault();
      event.stopPropagation();
      isDraggingOverSource.value = sourceName;
      
      handleVolumeHoverEnter(sourceName);
      
      if (event.dataTransfer) {
        event.dataTransfer.dropEffect = "copy";
      }
    }
  }

  function handleSourceDragOver(event: DragEvent) {
    const hasFiles = event.dataTransfer?.types.includes("Files");
    const hasZipEntry = event.dataTransfer?.types.includes("application/x-zip-entry");
    if (hasFiles || hasZipEntry) {
      event.preventDefault();
      event.stopPropagation();
      
      if (event.dataTransfer) {
        event.dataTransfer.dropEffect = "copy";
      }
    }
  }

  function handleSourceDragLeave(event: DragEvent) {
    event.preventDefault();
    event.stopPropagation();
    
    const relatedTarget = event.relatedTarget as HTMLElement | null;
    const currentTarget = event.currentTarget as HTMLElement | null;
    
    if (currentTarget && relatedTarget && currentTarget.contains(relatedTarget)) {
      return;
    }
    
    isDraggingOverSource.value = null;
    handleVolumeHoverLeave();
  }

  function handleSourceDrop(sourceName: string, event: DragEvent) {
    event.preventDefault();
    event.stopPropagation();
    isDraggingOverSource.value = null;

    const files = Array.from(event.dataTransfer?.files || []);
    // Emit even if no files (might be zip entry that will be extracted)
    emit('source-drop', sourceName, files, event);
  }
</script>

<style scoped>
  .is-dragging-over-source {
    background: var(--oui-surface-hover) !important;
    position: relative !important;
  }

  .is-dragging-over-source::before {
    content: '';
    position: absolute;
    inset: 0;
    background: var(--oui-accent-primary);
    opacity: 0.12;
    pointer-events: none;
    border-radius: 12px;
    z-index: 0;
  }


  .is-hovering-volume {
    background: var(--oui-surface-hover) !important;
    border-color: var(--oui-border-strong) !important;
  }

  .is-selected-source {
    border-width: 2px;
    font-weight: 500;
  }

  .file-tree-root {
    user-select: none;
    -webkit-user-select: none;
    -moz-user-select: none;
    -ms-user-select: none;
  }

  .file-tree-root * {
    user-select: none;
    -webkit-user-select: none;
    -moz-user-select: none;
    -ms-user-select: none;
  }
</style>

