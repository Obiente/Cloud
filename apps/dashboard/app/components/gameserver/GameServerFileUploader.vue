<template>
  <FileUpload.Root
    :maxFiles="500"
    :maxFileSize="1024 * 1024 * 1024"
    @filesAccepted="handleFilesAccepted"
    @fileRejected="handleFileRejected"
  >
    <FileUpload.Context v-slot="api">
      <FileUpload.Dropzone
        :class="[
          'border-2 border-dashed rounded-xl p-8 text-center cursor-pointer transition-colors',
          api.dragging
            ? 'border-primary bg-primary/10'
            : 'border-border-default hover:border-primary/50',
        ]"
      >
        <OuiStack gap="sm" align="center">
          <ArrowUpTrayIcon class="h-10 w-10 text-text-tertiary" />
          <OuiStack gap="xs" align="center">
            <OuiText size="md" weight="semibold">
              {{
                api.dragging
                  ? "Drop files here to upload"
                  : "Click or drag files to upload"
              }}
            </OuiText>
            <OuiText size="xs" color="secondary">
              {{
                api.dragging
                  ? "Release to upload"
                  : "Upload multiple files at once"
              }}
            </OuiText>
          </OuiStack>
          <OuiFlex gap="xs" align="center" class="mt-2">
            <OuiText size="xs" color="secondary">or</OuiText>
            <FileUpload.Trigger asChild>
              <OuiButton variant="outline" size="sm"> Browse Files </OuiButton>
            </FileUpload.Trigger>
          </OuiFlex>
          <OuiText size="xs" color="secondary" class="mt-1">
            Maximum size: 1GB per file (up to 500 files)
          </OuiText>
        </OuiStack>
      </FileUpload.Dropzone>

      <FileUpload.ItemGroup
        v-if="api.acceptedFiles.length > 0"
        class="mt-4 overflow-y-scroll"
      >
        <OuiStack gap="sm">
          <OuiText size="xs" weight="semibold">
            Selected Files ({{ api.acceptedFiles.length }})
          </OuiText>
          <div class="selected-files-list">
            <OuiCard
              v-for="(file, idx) in api.acceptedFiles"
              :key="file.name"
              variant="outline"
              class="selected-file-item p-3"
              :class="{
                'selected-file-item-first': idx === 0,
                'selected-file-item-last': idx === api.acceptedFiles.length - 1,
              }"
            >
              <OuiFlex align="center" gap="md">
                <FileUpload.Item :file="file">
                  <FileUpload.ItemPreview type="image/*">
                    <FileUpload.ItemPreviewImage
                      class="h-10 w-10 rounded object-cover"
                    />
                  </FileUpload.ItemPreview>
                  <FileUpload.ItemPreview type=".*">
                    <DocumentIcon class="h-5 w-5 text-secondary" />
                  </FileUpload.ItemPreview>
                  <OuiStack gap="xs" class="flex-1 min-w-0">
                    <OuiText size="sm" weight="medium" class="truncate">
                      <FileUpload.ItemName />
                    </OuiText>
                    <OuiText size="xs" color="secondary">
                      <FileUpload.ItemSizeText />
                    </OuiText>
                  </OuiStack>
                  <FileUpload.ItemDeleteTrigger asChild>
                    <OuiButton variant="ghost" size="xs" color="danger">
                      Remove
                    </OuiButton>
                  </FileUpload.ItemDeleteTrigger>
                </FileUpload.Item>
              </OuiFlex>
            </OuiCard>
          </div>

          <!-- Upload Progress Section -->
          <OuiStack v-if="isUploading" gap="sm" class="mt-4 p-4 bg-secondary/5 rounded-lg">
            <OuiStack gap="xs">
              <OuiFlex justify="between" align="center">
                <OuiText size="sm" weight="semibold">Overall Progress</OuiText>
                  <OuiStack align="end" gap="xs">
                    <OuiText size="sm" weight="semibold" color="primary">{{ overallProgress }}%</OuiText>
                    <OuiText size="xs" color="secondary">{{ formatSpeed(overallSpeed) }} • {{ overallEtaSeconds !== undefined && overallEtaSeconds !== null ? `${overallEtaSeconds}s left` : "—" }}</OuiText>
                  </OuiStack>
              </OuiFlex>
              <div class="w-full bg-border-default rounded-full h-2 overflow-hidden">
                <div
                  class="bg-primary h-full transition-all duration-300"
                  :style="{ width: `${overallProgress}%` }"
                />
              </div>
            </OuiStack>

            <!-- Per-file progress -->
            <OuiStack gap="sm" v-if="Object.keys(progressMap).length > 0" class="mt-3">
              <OuiText size="xs" weight="medium">Files:</OuiText>
              <OuiStack gap="xs" class="max-h-40 overflow-y-auto">
                <div v-for="(progress, fileName) in progressMap" :key="fileName" class="p-2 bg-white rounded border border-border-default">
                  <OuiFlex justify="between" align="center" gap="sm">
                    <OuiText size="xs" class="truncate flex-1">{{ fileName }}</OuiText>
                    <OuiText size="xs" color="secondary" class="whitespace-nowrap">
                      {{ formatBytes(progress.bytesUploaded) }} / {{ formatBytes(progress.totalBytes) }} • {{ formatSpeed(progress.speedBytesPerSec) }}
                    </OuiText>
                  </OuiFlex>
                  <div class="w-full bg-border-default rounded-full h-1 overflow-hidden mt-1">
                    <div
                      class="bg-primary h-full transition-all duration-300"
                      :style="{ width: `${progress.percentComplete}%` }"
                    />
                  </div>
                </div>
              </OuiStack>
            </OuiStack>
          </OuiStack>

          <OuiFlex justify="end" gap="sm" class="mt-2">
            <FileUpload.ClearTrigger asChild>
              <OuiButton variant="ghost" size="sm" :disabled="isUploading"> Clear All </OuiButton>
            </FileUpload.ClearTrigger>
            <OuiButton
              @click="uploadFiles(api)"
              :disabled="isUploading || api.acceptedFiles.length === 0"
              size="sm"
            >
              {{
                isUploading
                  ? "Uploading..."
                  : `Upload ${api.acceptedFiles.length} File${
                      api.acceptedFiles.length !== 1 ? "s" : ""
                    }`
              }}
            </OuiButton>
          </OuiFlex>
        </OuiStack>
      </FileUpload.ItemGroup>

      <FileUpload.HiddenInput />
    </FileUpload.Context>
  </FileUpload.Root>

  <!-- Rejected Files -->
  <OuiStack v-if="rejectedFiles.length > 0" gap="xs" class="mt-4">
    <OuiText size="xs" weight="semibold" color="danger"
      >Rejected Files:</OuiText
    >
    <OuiCard
      variant="default"
      class="border-danger"
      v-for="rejection in rejectedFiles"
      :key="rejection.file.name"
    >
      <OuiCardBody class="py-2">
        <OuiStack gap="xs">
          <OuiText size="xs" weight="medium">{{ rejection.file.name }}</OuiText>
          <OuiText
            size="xs"
            color="danger"
            v-for="error in rejection.errors"
            :key="error.code"
          >
            {{ error.message }}
          </OuiText>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>

  <OuiText v-if="uploadError" size="xs" color="danger" class="mt-4">{{
    uploadError
  }}</OuiText>
  <OuiText v-if="uploadSuccess" size="xs" color="success" class="mt-4">{{
    uploadSuccess
  }}</OuiText>
</template>

<script setup lang="ts">
  import { ref, computed } from "vue";
  import { FileUpload } from "@ark-ui/vue/file-upload";
  import { ArrowUpTrayIcon, DocumentIcon } from "@heroicons/vue/24/outline";
  import { useStreamingUpload } from "~/composables/useStreamingUpload";
  import { useToast } from "~/composables/useToast";
  import type { ExplorerNode } from "~/components/shared/fileExplorerTypes";

  interface Props {
    gameServerId: string;
    destinationPath?: string;
    destinationNode?: ExplorerNode;
    volumeName?: string;
  }

  interface Emits {
    (e: "uploaded", files: File[]): void;
  }

  const props = defineProps<Props>();
  const emit = defineEmits<Emits>();
  const { uploadFile, isUploading, error } = useStreamingUpload();
  const { toast } = useToast();

  const progressMap = ref<Record<string, { bytesUploaded: number; totalBytes: number; percentComplete: number; speedBytesPerSec?: number; etaSeconds?: number }>>(
    {}
  );

  // Total bytes from files that have finished uploading (used to compute overall progress)
  const completedBytes = ref(0);
  // Grand total bytes for all selected files (including not-yet-started)
  const totalBytesToUpload = ref(0);

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  const formatSpeed = (bytesPerSec?: number) => {
    if (!bytesPerSec || bytesPerSec <= 0) return "—";
    // Use same units as formatBytes but per second
    const k = 1024;
    const sizes = ["B/s", "KB/s", "MB/s", "GB/s"];
    const i = Math.floor(Math.log(bytesPerSec) / Math.log(k));
    return parseFloat((bytesPerSec / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  const overallProgress = computed(() => {
    const items = Object.values(progressMap.value);
    // Sum loaded for in-progress files
    let loaded = 0;
    for (const it of items) {
      loaded += it.bytesUploaded;
    }
    // Add completed bytes to loaded
    const completed = completedBytes.value;
    // Use the grand total of all selected files (including not-yet-started)
    const grandTotal = totalBytesToUpload.value;
    const grandLoaded = loaded + completed;
    if (!grandTotal || grandTotal === 0) return 0;
    return Math.round((grandLoaded / grandTotal) * 100);
  });

  const overallSpeed = computed(() => {
    // Sum speeds of all current progress entries (best-effort)
    const items = Object.values(progressMap.value);
    let speed = 0;
    for (const it of items) {
      speed += it.speedBytesPerSec || 0;
    }
    return speed;
  });

  const overallEtaSeconds = computed(() => {
    const items = Object.values(progressMap.value);
    let loaded = 0;
    for (const it of items) {
      loaded += it.bytesUploaded;
    }
    const completed = completedBytes.value;
    const grandTotal = totalBytesToUpload.value;
    const remaining = grandTotal ? Math.max(0, grandTotal - (loaded + completed)) : 0;
    const speed = overallSpeed.value;
    if (remaining === 0) return 0;
    if (speed === 0) return undefined;
    return Math.round(remaining / speed);
  });

  const uploadError = ref("");
  const uploadSuccess = ref("");
  const rejectedFiles = ref<
    Array<{ file: File; errors: Array<{ code: string; message: string }> }>
  >([]);

  const handleFilesAccepted = (details: { acceptedFiles: File[] }) => {
    uploadError.value = "";
    uploadSuccess.value = "";
    // Calculate total bytes for overall progress (includes files not yet started)
    totalBytesToUpload.value = details.acceptedFiles.reduce((acc, f) => acc + f.size, 0);
  };

  const handleFileRejected = (details: {
    rejectedFiles: Array<{
      file: File;
      errors: Array<{ code: string; message: string }>;
    }>;
  }) => {
    rejectedFiles.value = [...rejectedFiles.value, ...details.rejectedFiles];
    uploadError.value = `Some files were rejected. Please check the requirements.`;
  };

  const uploadFiles = async (api: any) => {
    if (!api || api.acceptedFiles.length === 0) return;

    uploadError.value = "";
    uploadSuccess.value = "";

    const files = api.acceptedFiles.slice();
    // Ensure grand total is up-to-date in case upload was triggered without a recent filesAccepted event
    totalBytesToUpload.value = files.reduce((acc: number, f: File) => acc + f.size, 0);
    
    // Create abort controller for cancellation
    const abortController = new AbortController();
    
    // Initialize node upload progress if node is provided
    if (props.destinationNode) {
      props.destinationNode.uploadProgress = {
        isUploading: true,
        bytesUploaded: 0,
        totalBytes: totalBytesToUpload.value,
        fileCount: files.length,
        files: files.map((f: File) => ({
          fileName: f.name,
          bytesUploaded: 0,
          totalBytes: f.size,
          percentComplete: 0,
        })),
        onCancel: () => {
          abortController.abort();
          toast.info("Upload cancelled", "Stopping upload...");
        },
      };
    }

    // Show toast with initial progress
    const progressToastId = toast.loading(
      `Uploading ${files.length} file(s)...`,
      "0% complete"
    );

    try {
      let uploadedFilesCount = 0;

      // Upload files sequentially with progress tracking
      for (const file of files) {
        if (abortController.signal.aborted) {
          throw new Error("Upload cancelled");
        }

        progressMap.value[file.name] = { bytesUploaded: 0, totalBytes: file.size, percentComplete: 0 };

        const success = await uploadFile(file, {
          gameServerId: props.gameServerId,
          destinationPath: props.destinationPath || "/",
          volumeName: props.volumeName,
          abortSignal: abortController.signal,
          onProgress: (progress) => {
            progressMap.value[file.name] = {
              bytesUploaded: progress.bytesUploaded,
              totalBytes: progress.totalBytes,
              percentComplete: progress.percentComplete,
            };
            
            // Update individual file progress in node
            if (props.destinationNode?.uploadProgress) {
              const fileProgress = props.destinationNode.uploadProgress.files.find(f => f.fileName === file.name);
              if (fileProgress) {
                fileProgress.bytesUploaded = progress.bytesUploaded;
                fileProgress.percentComplete = progress.percentComplete;
              }
            }
            
            // Calculate total uploaded across all files
            const totalUploaded = completedBytes.value + 
              Object.values(progressMap.value).reduce((acc, p) => acc + p.bytesUploaded, 0);
            
            // Update node progress
            if (props.destinationNode?.uploadProgress) {
              props.destinationNode.uploadProgress.bytesUploaded = totalUploaded;
            }
            
            // Update toast progress
            const percent = Math.round((totalUploaded / totalBytesToUpload.value) * 100);
            toast.update(
              progressToastId,
              `Uploading ${files.length} file(s)...`,
              `${percent}% complete`
            );
          },
          onFileComplete: () => {
            uploadedFilesCount += 1;
            // Mark file as completed: add its bytes to completedBytes
            completedBytes.value += file.size;
            
            // Mark file as complete in node progress
            if (props.destinationNode?.uploadProgress) {
              const fileProgress = props.destinationNode.uploadProgress.files.find(f => f.fileName === file.name);
              if (fileProgress) {
                fileProgress.percentComplete = 100;
                fileProgress.bytesUploaded = fileProgress.totalBytes;
              }
            }
            
            // Ensure UI shows 100% for a short time, then remove entry
            progressMap.value[file.name] = {
              bytesUploaded: file.size,
              totalBytes: file.size,
              percentComplete: 100,
            };
            setTimeout(() => {
              delete progressMap.value[file.name];
              // Remove file from node progress list
              if (props.destinationNode?.uploadProgress) {
                props.destinationNode.uploadProgress.files = props.destinationNode.uploadProgress.files.filter(f => f.fileName !== file.name);
              }
            }, 1500);
          },
        });

        if (!success) {
          throw new Error(error.value || "Upload failed");
        }
      }

      // Clear node progress
      if (props.destinationNode) {
        props.destinationNode.uploadProgress = undefined;
      }
      
      // Dismiss loading toast and show success
      toast.dismiss(progressToastId);
      toast.success(
        "Files uploaded successfully",
        `${uploadedFilesCount} file(s) uploaded to ${props.destinationPath || '/'}`
      );
      
      emit("uploaded", api.acceptedFiles);
      api.clearFiles();
      rejectedFiles.value = [];

      // reset overall tracking
      totalBytesToUpload.value = 0;
      completedBytes.value = 0;
      progressMap.value = {} as typeof progressMap.value;
    } catch (err: any) {
      console.error("Upload error:", err);
      uploadError.value = err.message || "Failed to upload files";
      
      // Clear node progress
      if (props.destinationNode) {
        props.destinationNode.uploadProgress = undefined;
      }
      
      // Dismiss loading toast and show error
      toast.dismiss(progressToastId);
      
      if (abortController.signal.aborted || err.message === "Upload cancelled") {
        toast.info("Upload cancelled", "Upload was stopped");
      } else {
        toast.error("Upload Error", err.message || "Failed to upload files");
      }
    }
  };
</script>

<style scoped>
  .selected-files-list {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 1px;
    overflow: hidden;
    border-radius: 6px;
    background: var(--oui-border-default);
  }

  .selected-file-item {
    border: none !important;
    background: var(--oui-surface-base);
    margin: 0 !important;
    border-radius: 0 !important;
  }

  .selected-file-item-first {
    border-top-left-radius: 6px !important;
    border-top-right-radius: 6px !important;
  }

  .selected-file-item-last {
    border-bottom-left-radius: 6px !important;
    border-bottom-right-radius: 6px !important;
  }

  /* Handle multi-row grids - add rounded corners to edges */
  .selected-file-item:nth-last-child(-n + 4) {
    border-bottom-left-radius: 6px !important;
    border-bottom-right-radius: 6px !important;
  }

  .selected-file-item:nth-child(4n + 1) {
    border-top-left-radius: 6px !important;
    border-bottom-left-radius: 6px !important;
  }

  .selected-file-item:nth-child(4n) {
    border-top-right-radius: 6px !important;
    border-bottom-right-radius: 6px !important;
  }
</style>
