<template>
  <div class="h-full overflow-auto p-6">
    <div v-if="loading" class="flex items-center justify-center h-full">
      <ArrowPathIcon class="h-6 w-6 animate-spin text-primary" />
      <OuiText size="sm" color="secondary" class="ml-2">Loading folder contents...</OuiText>
    </div>

    <div v-else-if="!node || !node.children || node.children.length === 0" class="flex flex-col items-center justify-center h-full text-center">
      <FolderIcon class="h-16 w-16 text-text-tertiary mb-4" />
      <OuiText size="lg" weight="semibold" class="mb-2">Folder is empty</OuiText>
      <OuiText size="sm" color="secondary">This folder doesn't contain any files or subdirectories</OuiText>
    </div>

    <div v-else class="space-y-4">
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
        <div class="space-y-1">
          <div
            v-for="child in sortedChildren"
            :key="child.path"
            class="flex items-center gap-3 p-3 rounded-lg hover:bg-surface-elevated transition-colors cursor-pointer border border-transparent hover:border-border-default"
            @click="$emit('select-item', child)"
          >
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
                {{ child.name }}
              </OuiText>
              <OuiText v-if="child.type === 'symlink' && child.symlinkTarget" size="xs" color="secondary" class="truncate">
                → {{ child.symlinkTarget }}
              </OuiText>
            </div>
            <div class="flex items-center gap-4 flex-shrink-0">
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
  import { computed } from "vue";
  import { FolderIcon, DocumentIcon, LinkIcon, ArrowPathIcon } from "@heroicons/vue/24/outline";
  import type { ExplorerNode } from "./fileExplorerTypes";
  import { formatFileSize } from "~/composables/useZipFile";

  const props = defineProps<{
    node: ExplorerNode | null;
    loading?: boolean;
  }>();

  defineEmits<{
    (e: "select-item", node: ExplorerNode): void;
    (e: "load-more", node: ExplorerNode): void;
  }>();

  const sortedChildren = computed(() => {
    if (!props.node?.children) return [];
    // Sort: directories first, then files, both alphabetically
    return [...props.node.children].sort((a, b) => {
      if (a.type === "directory" && b.type !== "directory") return -1;
      if (a.type !== "directory" && b.type === "directory") return 1;
      return a.name.localeCompare(b.name);
    });
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
</script>

