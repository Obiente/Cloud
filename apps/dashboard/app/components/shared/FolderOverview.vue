<template>
  <div
    class="h-full overflow-auto p-6"
    @dragenter="handleRootDragEnter"
    @dragover="handleRootDragOver"
    @dragleave="handleRootDragLeave"
    @drop="handleRootDrop"
  >
    <div v-if="loading" class="flex items-center justify-center h-full">
      <ArrowPathIcon class="h-6 w-6 animate-spin text-primary" />
      <OuiText size="sm" color="secondary" class="ml-2">Loading folder contents...</OuiText>
    </div>

    <div v-else-if="!node || !node.children || node.children.length === 0" class="flex flex-col items-center justify-center h-full text-center">
      <FolderIcon class="h-16 w-16 text-text-tertiary mb-4" />
      <OuiText size="lg" weight="semibold" class="mb-2">Folder is empty</OuiText>
      <OuiText size="sm" color="secondary">This folder doesn't contain any files or subdirectories</OuiText>
    </div>

    <div v-else class="space-y-4" :class="{ 'bg-accent-primary/5 border-2 border-accent-primary rounded-lg p-4': isDraggingOverRoot }">
      <!-- Folder Statistics -->
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
        <div class="bg-surface-elevated rounded-lg p-4 border border-border-default">
          <OuiText size="xs" color="secondary" class="mb-1">Total Size</OuiText>
          <OuiText size="xl" weight="bold">{{ formatFileSize(totalSize) }}</OuiText>
          <OuiText v-if="node.hasMore" size="xs" color="secondary" class="mt-1">
            (partial)
          </OuiText>
        </div>
        <div class="bg-surface-elevated rounded-lg p-4 border border-border-default">
          <OuiText size="xs" color="secondary" class="mb-1">Files</OuiText>
          <OuiText size="xl" weight="bold">{{ fileCount }}</OuiText>
        </div>
        <div class="bg-surface-elevated rounded-lg p-4 border border-border-default">
          <OuiText size="xs" color="secondary" class="mb-1">Folders</OuiText>
          <OuiText size="xl" weight="bold">{{ folderCount }}</OuiText>
        </div>
      </div>

      <!-- Folder Contents -->
      <div class="space-y-2">
        <OuiText size="sm" weight="semibold" class="mb-3">Contents</OuiText>
        <div v-if="sortedChildren.length > 0" class="space-y-1">
          <div
            v-for="child in sortedChildren"
            :key="child.path"
            data-drop-target="child"
            class="flex items-center gap-3 p-3 rounded-lg hover:bg-surface-elevated transition-colors cursor-pointer border border-transparent hover:border-border-default relative overflow-hidden"
            :class="{
              'bg-accent-primary/10 border-accent-primary': isDraggingOverChild === child.path && child.type === 'directory',
            }"
            @click="$emit('select-item', child)"
            @dragenter="handleChildDragEnter(child, $event)"
            @dragover="handleChildDragOver(child, $event)"
            @dragleave="handleChildDragLeave(child, $event)"
            @drop="handleChildDrop(child, $event)"
          >
            <!-- Upload progress bar background -->
            <div
              v-if="child.uploadProgress?.isUploading"
              class="absolute inset-0 bg-accent-primary/10"
              :style="{ width: `${(child.uploadProgress.bytesUploaded / child.uploadProgress.totalBytes) * 100}%`, transition: 'width 0.2s ease' }"
            ></div>

            <!-- Content -->
            <div class="relative flex items-center gap-3 w-full">
              <component
                :is="child.type === 'directory' ? FolderIcon : child.type === 'symlink' ? LinkIcon : DocumentIcon"
                class="h-5 w-5 flex-shrink-0"
                :class="
                  child.type === 'directory'
                    ? 'text-text-secondary'
                    : child.type === 'symlink'
                    ? 'text-accent-primary'
                    : 'text-text-tertiary'
                "
              />
              <div class="flex-1 min-w-0">
                <OuiText size="sm" weight="medium" class="truncate">
                  <span v-if="!child.uploadProgress?.isUploading">{{ child.name }}</span>
                  <span v-else class="text-[12px]">
                    {{ Math.round((child.uploadProgress.bytesUploaded / child.uploadProgress.totalBytes) * 100) }}% 
                    ({{ child.uploadProgress.fileCount }} file{{ child.uploadProgress.fileCount > 1 ? 's' : '' }})
                  </span>
                </OuiText>
                <OuiText v-if="child.type === 'symlink' && child.symlinkTarget && !child.uploadProgress?.isUploading" size="xs" color="secondary" class="truncate">
                  → {{ child.symlinkTarget }}
                </OuiText>
              </div>
              <div v-if="!child.uploadProgress?.isUploading" class="flex items-center gap-4 flex-shrink-0">
                <OuiText v-if="child.size !== undefined && child.size !== null" size="xs" color="secondary" class="min-w-[4rem] text-right">
                  {{ formatFileSize(child.size) }}
                </OuiText>
                <OuiText v-else-if="child.type === 'directory'" size="xs" color="secondary" class="min-w-[4rem] text-right">
                  —
                </OuiText>
                <OuiText v-if="child.modifiedTime" size="xs" color="secondary" class="min-w-[10rem] text-right hidden sm:block">
                  {{ formatDate(child.modifiedTime) }}
                </OuiText>
              </div>
            </div>
          </div>
        </div>

        <!-- Root Node Uploading Files List -->
        <div v-if="node.uploadProgress?.isUploading && node.uploadProgress.files?.length" class="space-y-2 mt-4 pt-4 border-t border-border-default">
          <OuiText size="xs" weight="semibold" class="text-text-secondary">Uploading to {{ node.name || 'Root' }}</OuiText>
          <div class="space-y-2">
            <div
              v-for="file in node.uploadProgress.files"
              :key="file.fileName"
              class="flex items-center gap-3 p-2 rounded-lg bg-surface-elevated border border-border-default text-xs"
            >
              <DocumentIcon class="h-4 w-4 flex-shrink-0 text-text-tertiary" />
              <span class="flex-1 truncate text-text-primary">{{ file.fileName }}</span>
              <span class="text-text-secondary">{{ file.percentComplete }}%</span>
              <div class="w-12 h-2 bg-surface-base rounded-full overflow-hidden border border-border-default">
                <div
                  class="h-full bg-accent-primary transition-all"
                  :style="{ width: `${file.percentComplete}%` }"
                ></div>
              </div>
            </div>
          </div>
        </div>

          <!-- Uploading Files List (for child folders) -->
          <div
            v-for="child in childrenWithUploadingFiles"
            :key="child.path"
            class="ml-8 space-y-2 mt-4"
          >
            <div
              v-for="file in child.uploadProgress?.files"
              :key="file.fileName"
              class="flex items-center gap-3 p-2 rounded-lg bg-surface-elevated border border-border-default text-xs"
            >
              <DocumentIcon class="h-4 w-4 flex-shrink-0 text-text-tertiary" />
              <span class="flex-1 truncate text-text-primary">{{ file.fileName }}</span>
              <span class="text-text-secondary">{{ file.percentComplete }}%</span>
              <div class="w-12 h-2 bg-surface-base rounded-full overflow-hidden border border-border-default">
                <div
                  class="h-full bg-accent-primary transition-all"
                  :style="{ width: `${file.percentComplete}%` }"
                ></div>
              </div>
            </div>
          </div>

        <!-- Load More Button -->
        <button
          v-if="node.hasMore"
          class="w-full p-3 rounded-lg border border-border-default hover:bg-surface-elevated transition-colors text-center"
          :disabled="node.isLoading"
          @click="$emit('load-more', node)"
        >
          <OuiFlex align="center" justify="center" gap="sm">
            <ArrowPathIcon v-if="node.isLoading" class="h-4 w-4 animate-spin" />
            <OuiText size="sm" color="secondary">
              {{ node.isLoading ? "Loading..." : "Load more" }}
            </OuiText>
          </OuiFlex>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, ref } from "vue";
  import { FolderIcon, DocumentIcon, LinkIcon, ArrowPathIcon } from "@heroicons/vue/24/outline";
  import type { ExplorerNode } from "./fileExplorerTypes";
  import { formatFileSize } from "~/composables/useZipFile";

  const props = defineProps<{
    node: ExplorerNode | null;
    loading?: boolean;
  }>();

  const emit = defineEmits<{
    (e: "select-item", node: ExplorerNode): void;
    (e: "load-more", node: ExplorerNode): void;
    (e: "drop-files", node: ExplorerNode, files: File[], event?: DragEvent): void;
  }>();

  const isDraggingOverChild = ref<string | null>(null);
  const isDraggingOverRoot = ref(false);

  const sortedChildren = computed(() => {
    if (!props.node?.children) return [];
    // Sort: directories first, then files, both alphabetically
    return [...props.node.children].sort((a, b) => {
      if (a.type === "directory" && b.type !== "directory") return -1;
      if (a.type !== "directory" && b.type === "directory") return 1;
      return a.name.localeCompare(b.name);
    });
  });

  const childrenWithUploadingFiles = computed(() => {
    return sortedChildren.value.filter((child) => child.uploadProgress?.isUploading && child.uploadProgress.files?.length);
  });

  const fileCount = computed(() => {
    return props.node?.children?.filter((c) => c.type === "file").length || 0;
  });

  const folderCount = computed(() => {
    return props.node?.children?.filter((c) => c.type === "directory").length || 0;
  });

  const totalSize = computed(() => {
    if (!props.node?.children) return 0;
    // Sum up sizes of all files (direct children only, not recursive)
    return props.node.children
      .filter((c) => c.type === "file" && c.size !== undefined && c.size !== null)
      .reduce((sum, file) => sum + (file.size || 0), 0);
  });

  function formatDate(dateString: string | undefined): string {
    if (!dateString) return "";
    try {
      const date = new Date(dateString);
      return new Intl.DateTimeFormat("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      }).format(date);
    } catch {
      return dateString;
    }
  }

  function handleChildDragEnter(child: ExplorerNode, event: DragEvent) {
    if (child.type !== "directory") return;
    // Check if dragging files or zip entries
    const hasFiles = event.dataTransfer?.types.includes("Files");
    const hasZipEntry = event.dataTransfer?.types.includes("application/x-zip-entry");
    if (hasFiles || hasZipEntry) {
      event.preventDefault();
      isDraggingOverChild.value = child.path;
      
      // Set drop effect
      if (event.dataTransfer) {
        event.dataTransfer.dropEffect = "copy";
      }
    }
  }

  function handleChildDragOver(child: ExplorerNode, event: DragEvent) {
    if (child.type !== "directory") return;
    // Check if dragging files or zip entries
    const hasFiles = event.dataTransfer?.types.includes("Files");
    const hasZipEntry = event.dataTransfer?.types.includes("application/x-zip-entry");
    if (hasFiles || hasZipEntry) {
      event.preventDefault();
      isDraggingOverChild.value = child.path;
      
      // Set drop effect
      if (event.dataTransfer) {
        event.dataTransfer.dropEffect = "copy";
      }
    }
  }

  function handleChildDragLeave(child: ExplorerNode, event: DragEvent) {
    if (child.type !== "directory") return;
    event.preventDefault();
    event.stopPropagation();
    
    // Only clear if we're actually leaving the element
    const relatedTarget = event.relatedTarget as HTMLElement | null;
    const currentTarget = event.currentTarget as HTMLElement | null;
    
    // Check if we're moving to a child element - if so, don't clear
    if (currentTarget && relatedTarget && currentTarget.contains(relatedTarget)) {
      return; // Still within the element, keep highlighting
    }
    
    // We're actually leaving
    isDraggingOverChild.value = null;
  }

  function handleChildDrop(child: ExplorerNode, event: DragEvent) {
    if (child.type !== "directory") return;
    event.preventDefault();
    event.stopPropagation();
    isDraggingOverChild.value = null;

    const files = Array.from(event.dataTransfer?.files || []);
    if (files.length > 0) {
      emit("drop-files", child, files, event);
    }
  }

  function handleRootDragEnter(event: DragEvent) {
    const hasFiles = event.dataTransfer?.types.includes("Files");
    const hasZipEntry = event.dataTransfer?.types.includes("application/x-zip-entry");
    if (hasFiles || hasZipEntry) {
      event.preventDefault();
      const target = event.target as HTMLElement;
      const isOverChild = target.closest('[data-drop-target="child"]');
      
      // Only show root overlay if not over a child folder
      if (!isOverChild) {
        isDraggingOverRoot.value = true;
      }
      
      if (event.dataTransfer) {
        event.dataTransfer.dropEffect = "copy";
      }
    }
  }

  function handleRootDragOver(event: DragEvent) {
    const hasFiles = event.dataTransfer?.types.includes("Files");
    const hasZipEntry = event.dataTransfer?.types.includes("application/x-zip-entry");
    if (hasFiles || hasZipEntry) {
      event.preventDefault();
      const target = event.target as HTMLElement;
      const isOverChild = target.closest('[data-drop-target="child"]');
      
      // Only show root overlay if not over a child folder
      if (!isOverChild) {
        isDraggingOverRoot.value = true;
      } else {
        isDraggingOverRoot.value = false;
      }
      
      if (event.dataTransfer) {
        event.dataTransfer.dropEffect = "copy";
      }
    }
  }

  function handleRootDragLeave(event: DragEvent) {
    event.preventDefault();
    event.stopPropagation();
    
    // Only clear if we're actually leaving the root container
    const relatedTarget = event.relatedTarget as HTMLElement | null;
    const currentTarget = event.currentTarget as HTMLElement | null;
    
    // Check if we're moving to a child element - if so, don't clear
    if (currentTarget && relatedTarget && currentTarget.contains(relatedTarget)) {
      return; // Still within root container
    }
    
    // We're actually leaving
    isDraggingOverRoot.value = false;
  }

  function handleRootDrop(event: DragEvent) {
    if (!props.node || isDraggingOverChild.value !== null) return;
    event.preventDefault();
    event.stopPropagation();
    isDraggingOverRoot.value = false;

    const files = Array.from(event.dataTransfer?.files || []);
    if (files.length > 0 && props.node) {
      emit("drop-files", props.node, files, event);
    }
  }
</script>

