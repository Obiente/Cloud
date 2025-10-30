<template>
  <OuiStack gap="md">
    <OuiFlex justify="between" align="center">
      <OuiText as="h4" size="sm" weight="semibold">Upload Files</OuiText>
      <OuiButton variant="ghost" size="sm" @click="triggerFileInput" :disabled="isUploading">
        <ArrowUpTrayIcon class="h-4 w-4 mr-1" />
        {{ isUploading ? "Uploading..." : "Select Files" }}
      </OuiButton>
    </OuiFlex>

    <input
      ref="fileInput"
      type="file"
      multiple
      class="hidden"
      @change="handleFileSelect"
      accept=".zip,.tar,.tar.gz,.tgz"
    />

    <OuiText size="xs" color="secondary">
      Upload a zip or tar archive containing your application files. Maximum size: 100MB.
    </OuiText>

    <div v-if="uploadedFiles.length > 0" class="space-y-2">
      <OuiText size="xs" weight="semibold">Uploaded Files:</OuiText>
      <div
        v-for="(file, idx) in uploadedFiles"
        :key="idx"
        class="flex items-center gap-2 p-2 rounded bg-surface-muted/30"
      >
        <DocumentIcon class="h-4 w-4 text-secondary" />
        <OuiText size="xs" class="flex-1">{{ file.name }}</OuiText>
        <OuiText size="xs" color="secondary">{{ formatSize(file.size) }}</OuiText>
        <OuiButton variant="ghost" size="xs" color="danger" @click="removeFile(idx)">
          Remove
        </OuiButton>
      </div>
    </div>

    <OuiButton
      v-if="uploadedFiles.length > 0"
      @click="uploadFiles"
      :disabled="isUploading"
      size="sm"
    >
      {{ isUploading ? "Uploading..." : "Upload Files" }}
    </OuiButton>

    <OuiText v-if="uploadError" size="xs" color="danger">{{ uploadError }}</OuiText>
  </OuiStack>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { ArrowUpTrayIcon, DocumentIcon } from "@heroicons/vue/24/outline";

interface Props {
  deploymentId: string;
}

interface Emits {
  (e: "uploaded", files: File[]): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const fileInput = ref<HTMLInputElement | null>(null);
const uploadedFiles = ref<File[]>([]);
const isUploading = ref(false);
const uploadError = ref("");

const triggerFileInput = () => {
  fileInput.value?.click();
};

const handleFileSelect = (event: Event) => {
  const target = event.target as HTMLInputElement;
  if (target.files) {
    const files = Array.from(target.files);
    // Filter by size (100MB limit)
    const validFiles = files.filter((f) => f.size <= 100 * 1024 * 1024);
    if (validFiles.length !== files.length) {
      uploadError.value = "Some files exceed 100MB limit and were not added.";
    } else {
      uploadError.value = "";
    }
    uploadedFiles.value = [...uploadedFiles.value, ...validFiles];
  }
};

const removeFile = (idx: number) => {
  uploadedFiles.value.splice(idx, 1);
};

const formatSize = (bytes: number) => {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
};

const uploadFiles = async () => {
  if (uploadedFiles.value.length === 0) return;

  isUploading.value = true;
  uploadError.value = "";

  try {
    // TODO: Implement actual file upload API
    // For now, just emit the files
    emit("uploaded", uploadedFiles.value);
    
    // Reset after successful upload
    uploadedFiles.value = [];
    if (fileInput.value) {
      fileInput.value.value = "";
    }
  } catch (error: any) {
    uploadError.value = error.message || "Failed to upload files";
  } finally {
    isUploading.value = false;
  }
};
</script>

