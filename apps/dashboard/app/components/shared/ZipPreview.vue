<template>
  <div
    class="w-full h-full flex flex-col overflow-hidden"
  >
    <div class="flex-1 overflow-auto p-4">
      <!-- Breadcrumb navigation -->
      <div class="flex items-center gap-2 mb-4 flex-wrap">
        <div class="flex items-center gap-1.5">
          <FolderIcon class="h-4 w-4 text-text-secondary flex-shrink-0" />
          <div class="flex items-center gap-1.5 flex-wrap">
            <button
              v-if="currentPath"
              class="text-sm text-text-secondary hover:text-text-primary transition-colors"
              @click="$emit('navigate-up')"
            >
              <span class="font-medium">Root</span>
            </button>
            <template v-if="currentPath">
              <span class="text-text-tertiary">/</span>
              <template
                v-for="(segment, index) in pathSegments"
                :key="index"
              >
                <button
                  v-if="index < pathSegments.length - 1"
                  class="text-sm text-text-secondary hover:text-text-primary transition-colors"
                  @click="navigateToSegment(index)"
                >
                  <span class="font-medium">{{ segment }}</span>
                </button>
                <span
                  v-else
                  class="text-sm text-text-primary font-medium"
                >
                  {{ segment }}
                </span>
                <span
                  v-if="index < pathSegments.length - 1"
                  class="text-text-tertiary"
                >
                  /
                </span>
              </template>
            </template>
            <span
              v-else
              class="text-sm text-text-primary font-medium"
            >
              Root
            </span>
          </div>
        </div>
      </div>
      <div
        v-if="loading"
        class="flex items-center justify-center h-full"
      >
        <OuiText color="secondary">Loading archive contents...</OuiText>
      </div>
      <div
        v-else-if="contents.length === 0"
        class="flex items-center justify-center h-full"
      >
        <OuiText color="secondary">Folder is empty</OuiText>
      </div>
      <div v-else class="space-y-1">
        <div
          v-for="entry in contents"
          :key="entry.path"
          class="flex items-center gap-2 p-2 rounded hover:bg-surface-elevated group"
          :class="entry.isDirectory ? 'cursor-pointer' : 'cursor-move'"
          :draggable="!entry.isDirectory"
          @click="entry.isDirectory && $emit('navigate-folder', entry.path)"
          @dragstart="(e) => !entry.isDirectory && $emit('entry-drag-start', e, entry)"
          @dragend="$emit('entry-drag-end')"
        >
          <component
            :is="entry.isDirectory ? FolderIcon : DocumentIcon"
            class="h-5 w-5 flex-shrink-0"
            :class="
              entry.isDirectory
                ? 'text-text-secondary'
                : 'text-text-tertiary'
            "
          />
          <OuiText size="sm" class="flex-1 truncate">
            {{ entry.name }}
          </OuiText>
          <OuiText
            v-if="!entry.isDirectory"
            size="xs"
            color="secondary"
            class="flex-shrink-0"
          >
            {{ formatFileSize(entry.size) }}
          </OuiText>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from "vue";
  import { DocumentIcon, FolderIcon } from "@heroicons/vue/24/outline";
  import type { ZipEntry } from "~/composables/useZipFile";
  import { formatFileSize } from "~/composables/useZipFile";

  const props = defineProps<{
    fileName?: string;
    contents: ZipEntry[];
    loading: boolean;
    currentPath?: string;
  }>();

  const emit = defineEmits<{
    (e: "navigate-folder", path: string): void;
    (e: "navigate-up"): void;
    (e: "entry-drag-start", event: DragEvent, entry: ZipEntry): void;
    (e: "entry-drag-end"): void;
  }>();

  // Compute path segments for breadcrumb
  const pathSegments = computed(() => {
    if (!props.currentPath) return [];
    return props.currentPath.split("/").filter(Boolean);
  });

  // Navigate to a specific segment in the breadcrumb
  function navigateToSegment(index: number) {
    const segments = pathSegments.value;
    if (index < 0 || index >= segments.length) return;
    
    // Build path up to the clicked segment
    const targetPath = segments.slice(0, index + 1).join("/");
    emit("navigate-folder", targetPath);
  }
</script>

