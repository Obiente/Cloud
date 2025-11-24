<template>
  <div class="flex-1 relative min-h-0 overflow-hidden" role="tabpanel">
    <!-- File Uploader (replaces editor when showUpload is true) -->
    <div v-if="showUpload" class="h-full flex items-center justify-center p-8">
      <div class="w-full max-w-2xl">
        <slot name="uploader" />
      </div>
    </div>
    <!-- File Preview/Editor (shown when not uploading) -->
    <template v-else>
      <div
        v-if="!selectedPath"
        class="h-full flex items-center justify-center text-text-tertiary"
      >
        <OuiText size="sm" color="secondary"
          >Select a file to view its contents</OuiText
        >
      </div>
      <div
        v-else-if="fileError"
        class="h-full flex items-center justify-center p-8"
      >
        <div
          class="flex flex-col items-center gap-4 max-w-md text-center"
        >
          <div
            class="flex items-center justify-center w-16 h-16 rounded-full bg-danger/10"
          >
            <ExclamationTriangleIcon class="h-8 w-8 text-danger" />
          </div>
          <div class="flex flex-col gap-2">
            <OuiText size="lg" weight="semibold" color="danger">
              Unable to View File
            </OuiText>
            <OuiText size="sm" color="secondary">
              {{ fileError }}
            </OuiText>
          </div>
          <OuiButton
            v-if="currentNode?.type === 'file'"
            variant="outline"
            size="sm"
            @click="$emit('download')"
          >
            Download Instead
          </OuiButton>
        </div>
      </div>
      <!-- Media Preview (Images, Videos, Audio, PDF) -->
      <div
        v-else-if="
          selectedPath &&
          currentNode?.type === 'file' &&
          !fileError &&
          filePreviewType &&
          filePreviewType !== 'text' &&
          filePreviewType !== 'zip' &&
          fileBlobUrl
        "
        class="h-full flex items-center justify-center p-8 bg-surface-base"
      >
        <div
          class="w-full h-full flex flex-col items-center justify-center gap-4"
        >
          <!-- Image Preview -->
          <img
            v-if="filePreviewType === 'image'"
            :src="fileBlobUrl"
            :alt="currentNode?.name || 'Image preview'"
            class="max-w-full max-h-full object-contain rounded-lg shadow-lg"
            @error="$emit('preview-error')"
          />
          <!-- Video Preview -->
          <video
            v-else-if="filePreviewType === 'video'"
            :src="fileBlobUrl"
            controls
            class="max-w-full max-h-full rounded-lg shadow-lg"
            @error="$emit('preview-error')"
          >
            Your browser does not support the video tag.
          </video>
          <!-- Audio Preview -->
          <div
            v-else-if="filePreviewType === 'audio'"
            class="w-full max-w-md flex flex-col items-center gap-4 p-6 bg-surface-elevated rounded-lg border border-border-default"
          >
            <OuiText size="lg" weight="semibold">
              {{ currentNode?.name || "Audio" }}
            </OuiText>
            <audio
              :src="fileBlobUrl"
              controls
              class="w-full"
              @error="$emit('preview-error')"
            >
              Your browser does not support the audio tag.
            </audio>
          </div>
          <!-- PDF Preview -->
          <iframe
            v-else-if="filePreviewType === 'pdf'"
            :src="fileBlobUrl"
            class="w-full h-full border border-border-default rounded-lg"
            @error="$emit('preview-error')"
          />
          <!-- Binary/Unsupported -->
          <div
            v-else-if="filePreviewType === 'binary'"
            class="flex flex-col items-center gap-4 p-8 max-w-md text-center"
          >
            <div
              class="flex items-center justify-center w-16 h-16 rounded-full bg-surface-elevated border-2 border-border-default"
            >
              <DocumentIcon class="h-8 w-8 text-text-tertiary" />
            </div>
            <div class="flex flex-col gap-2">
              <OuiText size="lg" weight="semibold"> Binary File </OuiText>
              <OuiText size="sm" color="secondary">
                This file type cannot be previewed.
                <template v-if="fileMetadata?.mimeType">
                  <br />
                  MIME type: {{ fileMetadata.mimeType }}
                </template>
              </OuiText>
            </div>
            <OuiButton
              variant="outline"
              size="sm"
              @click="$emit('download')"
            >
              Download File
            </OuiButton>
          </div>
        </div>
      </div>
      <!-- Zip Preview -->
      <ZipPreview
        v-else-if="
          selectedPath &&
          currentNode?.type === 'file' &&
          !fileError &&
          filePreviewType === 'zip'
        "
        :fileName="currentNode?.name"
        :contents="zipContents"
        :loading="zipLoading"
        :current-path="currentZipPath"
        @entry-drag-start="$emit('zip-entry-drag-start', $event)"
        @entry-drag-end="$emit('zip-entry-drag-end')"
      />
      <!-- Text Editor -->
      <OuiFileEditor
        v-else-if="
          selectedPath &&
          currentNode?.type === 'file' &&
          !fileError &&
          (filePreviewType === 'text' || filePreviewType === null)
        "
        :key="`editor-${selectedPath}-${editorRefreshKey}`"
        :model-value="fileContent"
        :language="fileLanguage"
        :read-only="false"
        height="100%"
        :container-class="'w-full h-full border-0 overflow-hidden'"
        class="absolute inset-0"
        @update:model-value="$emit('update:fileContent', $event)"
        @save="$emit('save')"
      />
      <div
        v-else-if="!selectedPath || !currentNode || currentNode.type !== 'file'"
        class="h-full flex items-center justify-center text-text-tertiary"
      >
        <OuiText size="sm" color="secondary"
          >Select a file to view its contents</OuiText
        >
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
  import { ExclamationTriangleIcon, DocumentIcon } from "@heroicons/vue/24/outline";
  import ZipPreview from "./ZipPreview.vue";
  import type { ExplorerNode } from "./fileExplorerTypes";
  import type { FilePreviewType } from "~/composables/useFilePreview";
  import type { FileMetadata, ZipEntry } from "~/composables/useZipFile";

  defineProps<{
    showUpload: boolean;
    selectedPath: string | null;
    currentNode: ExplorerNode | null;
    fileError: string | null;
    filePreviewType: FilePreviewType | null;
    fileBlobUrl: string | null;
    fileMetadata: FileMetadata | null;
    fileContent: string;
    fileLanguage: string;
    editorRefreshKey: number;
    zipContents: ZipEntry[];
    zipLoading: boolean;
    currentZipPath?: string;
  }>();

  defineEmits<{
    (e: "download"): void;
    (e: "preview-error"): void;
    (e: "zip-entry-drag-start", event: DragEvent): void;
    (e: "zip-entry-drag-end"): void;
    (e: "update:fileContent", value: string): void;
    (e: "save"): void;
  }>();
</script>

