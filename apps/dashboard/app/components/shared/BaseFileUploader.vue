<template>
  <FileUpload.Root
    :maxFiles="maxFiles"
    :maxFileSize="maxFileSize"
    @filesAccepted="handleFilesAccepted"
    @fileRejected="handleFileRejected"
    class="w-full h-full"
  >
    <FileUpload.Context v-slot="api">
      <OuiStack
        direction="vertical"
        h="full"
        w="full"
        overflow="hidden"
        gap="md"
      >
        <!-- Header: Drop zone -->
        <FileUpload.Trigger asChild>
          <OuiBox :shrink="false" w="full">
            <FileUpload.Dropzone
              :style="{
                border: `calc(var(--spacing) * 0.5) dashed ${
                  api.dragging
                    ? 'var(--oui-accent-primary)'
                    : 'var(--oui-border-default)'
                }`,
                borderRadius: 'var(--radius-xl)',
                padding:
                  api.acceptedFiles.length > 0
                    ? 'var(--spacing-lg)'
                    : 'var(--spacing-xl)',
                textAlign: 'center',
                cursor: 'pointer',
                transition: 'all 0.2s ease',
                height:
                  api.acceptedFiles.length > 0
                    ? 'calc(var(--spacing) * 30)'
                    : 'calc(var(--spacing) * 50)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: api.dragging
                  ? 'color-mix(in srgb, var(--oui-accent-primary) 10%, transparent)'
                  : 'transparent',
              }"
            >
              <OuiStack gap="sm" align="center">
                <ArrowUpTrayIcon
                  :style="{
                    height:
                      api.acceptedFiles.length > 0
                        ? 'calc(var(--spacing) * 6)'
                        : 'calc(var(--spacing) * 10)',
                    width:
                      api.acceptedFiles.length > 0
                        ? 'calc(var(--spacing) * 6)'
                        : 'calc(var(--spacing) * 10)',
                    color: 'var(--oui-text-tertiary)',
                    transition: 'all 0.2s ease',
                  }"
                />
                <OuiStack gap="xs" align="center">
                  <OuiText size="md" weight="semibold">
                    {{
                      api.dragging
                        ? "Release to upload"
                        : "Click or drag files to upload"
                    }}
                  </OuiText>
                </OuiStack>

                <OuiText
                  v-if="api.acceptedFiles.length === 0"
                  size="xs"
                  color="secondary"
                >
                  {{ maxFileSizeLabel }}
                </OuiText>
              </OuiStack>
            </FileUpload.Dropzone>
          </OuiBox>
        </FileUpload.Trigger>

        <!-- Content: Scrollable grid -->
        <OuiFlex
          v-if="api.acceptedFiles.length > 0 && !isUploading"
          direction="col"
          :grow="true"
          overflow="hidden"
          minH="0"
          w="full"
        >
          <OuiFlex justify="between" align="center" :shrink="false" mb="sm">
            <OuiText size="xs" weight="semibold">
              Selected Files ({{ api.acceptedFiles.length }})
            </OuiText>
            <OuiText size="xs" color="secondary">
              {{ api.acceptedFiles.length }} / {{ maxFiles }} files
            </OuiText>
          </OuiFlex>

          <OuiBox :grow="true" minH="0" overflowY="auto" w="full">
            <FileUpload.ItemGroup as="div" class="block min-h-0">
              <OuiGrid autoFit="xs" gap="sm" p="xs" pr="sm">
                <FileUpload.Item
                  v-for="file in api.acceptedFiles"
                  :key="file.name"
                  :file="file"
                >
                  <OuiCard variant="outline" w="44" h="56">
                    <OuiStack
                      gap="xs"
                      direction="vertical"
                      justify="between"
                      p="sm"
                      h="full"
                    >
                      <OuiFlex
                        bg="surface-muted"
                        rounded="md"
                        overflow="hidden"
                        justify="center"
                        align="center"
                        h="full"
                        class="w-full aspect-square"
                      >
                        <FileUpload.ItemPreview
                          type="image/*"
                          class="block w-full h-full"
                        >
                          <FileUpload.ItemPreviewImage
                            class="w-full h-full object-cover object-center"
                          />
                        </FileUpload.ItemPreview>
                        <FileUpload.ItemPreview
                          v-if="!file.type.startsWith('image/')"
                          type=".*"
                        >
                          <DocumentIcon
                            style="
                              height: 4rem;
                              width: 4rem;
                              color: var(--oui-text-tertiary);
                            "
                          />
                        </FileUpload.ItemPreview>
                      </OuiFlex>

                      <OuiStack gap="xs">
                        <OuiFlex align="start" justify="between" gap="xs">
                          <OuiText
                            as="span"
                            size="xs"
                            weight="medium"
                            lineClamp="2"
                            leading="snug"
                          >
                            <FileUpload.ItemName />
                          </OuiText>
                          <FileUpload.ItemDeleteTrigger asChild>
                            <OuiButton variant="ghost" size="xs" color="danger">
                              <XMarkIcon style="height: 1rem; width: 1rem" />
                            </OuiButton>
                          </FileUpload.ItemDeleteTrigger>
                        </OuiFlex>
                        <OuiText size="xs" color="secondary">
                          <FileUpload.ItemSizeText />
                        </OuiText>
                      </OuiStack>
                    </OuiStack>
                  </OuiCard>
                </FileUpload.Item>
              </OuiGrid>
            </FileUpload.ItemGroup>
          </OuiBox>
        </OuiFlex>

        <!-- Upload Progress Section (replaces file grid when uploading) -->
        <OuiFlex
          v-if="api.acceptedFiles.length > 0 && isUploading && showProgress"
          direction="col"
          :grow="true"
          overflow="hidden"
          minH="0"
          w="full"
        >
          <OuiCard
            variant="outline"
            rounded="md"
            p="md"
            :shrink="false"
            mb="sm"
          >
            <OuiStack gap="sm">
              <OuiFlex justify="between" align="center">
                <OuiText size="sm" weight="semibold"
                  >Overall Progress</OuiText
                >
                <OuiStack align="end" gap="xs">
                  <OuiText size="sm" weight="semibold" color="primary"
                    >{{ overallProgress }}%</OuiText
                  >
                  <OuiText size="xs" color="secondary">
                    {{ formatSpeed(smoothedNetworkSpeed) }}
                    <template
                      v-if="
                        overallEtaSeconds !== undefined &&
                        overallEtaSeconds !== null &&
                        overallEtaSeconds > 0
                      "
                    >
                      •
                      <OuiDuration :value="overallEtaSeconds * 1000" unitDisplay="short" />
                      left
                    </template>
                  </OuiText>
                </OuiStack>
              </OuiFlex>
              <div
                class="w-full h-2 bg-border-default rounded-md overflow-hidden"
              >
                <div
                  class="h-full bg-accent-primary rounded-md transition-all duration-300"
                  :style="`width: ${overallProgressClamped}%`"
                />
              </div>
            </OuiStack>
          </OuiCard>

          <!-- Per-file progress with active file at top -->
          <OuiBox :grow="true" minH="0" overflowY="auto" w="full">
            <OuiStack gap="sm" pb="sm">
              <template v-for="(progress, fileName) in sortedProgressMap" :key="fileName">
                <OuiCard
                  variant="outline"
                  rounded="sm"
                  p="sm"
                >
                  <OuiStack gap="sm">
                    <OuiFlex justify="between" align="center" gap="sm">
                      <OuiText
                        size="xs"
                        :weight="progress.percentComplete < 100 ? 'semibold' : 'normal'"
                        :color="progress.percentComplete === 100 ? 'secondary' : 'primary'"
                        style="
                          flex: 1;
                          overflow: hidden;
                          text-overflow: ellipsis;
                          white-space: nowrap;
                        "
                      >
                        {{ fileName }}
                      </OuiText>
                      <OuiText
                        size="xs"
                        color="secondary"
                        style="flex-shrink: 0; white-space: nowrap"
                      >
                        {{ formatBytes(progress.bytesUploaded) }} /
                        {{ formatBytes(progress.totalBytes) }}
                        <template v-if="progress.speedBytesPerSec && progress.speedBytesPerSec > 0">
                          • {{ formatSpeed(progress.speedBytesPerSec) }}
                        </template>
                      </OuiText>
                    </OuiFlex>
                    <div
                      class="w-full h-2 bg-border-default rounded-md overflow-hidden"
                    >
                      <div
                        class="h-full bg-accent-primary rounded-md transition-all duration-300"
                        :style="`width: ${clampPercent(progress.percentComplete)}%`"
                      />
                    </div>
                  </OuiStack>
                </OuiCard>
              </template>
            </OuiStack>
          </OuiBox>
        </OuiFlex>

        <!-- Rejected Files -->
        <OuiBox
          v-if="rejectedFiles.length > 0"
          mt="md"
          :shrink="false"
          w="full"
        >
          <OuiStack gap="xs">
            <OuiText size="xs" weight="semibold" color="danger"
              >Rejected Files:</OuiText
            >
            <OuiCard
              v-for="rejection in rejectedFiles"
              :key="rejection.file.name"
              variant="outline"
              style="border-color: var(--oui-error-border)"
            >
              <OuiStack gap="xs">
                <OuiText size="xs" weight="medium">{{
                  rejection.file.name
                }}</OuiText>
                <OuiText
                  size="xs"
                  color="danger"
                  v-for="error in rejection.errors"
                  :key="error.code"
                >
                  {{ error.message }}
                </OuiText>
              </OuiStack>
            </OuiCard>
          </OuiStack>
        </OuiBox>
        <OuiBox v-if="api.acceptedFiles.length > 0" mt="md" w="full">
          <OuiFlex justify="end" gap="sm">
            <!-- Show Cancel Uploads during upload, Clear All otherwise -->
            <OuiButton
              v-if="isUploading"
              variant="ghost"
              size="sm"
              color="danger"
              @click="handleCancelUploads"
            >
              Cancel
            </OuiButton>
            <OuiButton
              v-else
              variant="ghost"
              size="sm"
              @click="api.clearFiles()"
            >
              Clear All
            </OuiButton>
            <OuiButton
              @click="handleUpload(api)"
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
        </OuiBox>
      </OuiStack>
      <FileUpload.HiddenInput />
    </FileUpload.Context>
  </FileUpload.Root>
</template>

<script setup lang="ts">
  import { ref, computed, watchEffect } from "vue";
  import { FileUpload } from "@ark-ui/vue/file-upload";
  import {
    ArrowUpTrayIcon,
    DocumentIcon,
    XMarkIcon,
  } from "@heroicons/vue/24/outline";
  import OuiDuration from "~/components/oui/Duration.vue";
  import type { GameServerUploadProgress } from "~/composables/useStreamingUpload";
  import { useChunkedGameServerUpload } from "~/composables/useStreamingUpload";
  import { useDeploymentUpload } from "../../composables/useDeploymentUpload";
  import { useToast } from "~/composables/useToast";

  interface Props {
    uploaderId: string;
    destinationPath?: string;
    volumeName?: string;
    organizationId?: string;
    uploadKind?: "gameserver" | "deployment";
    additionalParams?: Record<string, any>;
    maxFiles?: number;
    maxFileSize?: number;
    showProgress?: boolean;
    targetNode?: any; // The tree node to update with upload progress
    sourceObject?: any; // Optional source object to also update for UI display
  }

  interface Emits {
    (e: "uploaded", files: File[]): void;
    (e: "uploadProgress", progress: { 
      bytesUploaded: number; 
      totalBytes: number; 
      percentComplete: number;
      files: Array<{ fileName: string; bytesUploaded: number; totalBytes: number; percentComplete: number }>;
    }): void;
  }

  const props = withDefaults(defineProps<Props>(), {
    maxFiles: 500,
    maxFileSize: 1024 * 1024 * 1024, // 1GB default
    showProgress: true,
    uploadKind: "gameserver",
  });

  const emit = defineEmits<Emits>();

  const { uploadFile } = useChunkedGameServerUpload();
  const { uploadFile: uploadDeploymentFile } = useDeploymentUpload();
  const { toast } = useToast();

  const isUploading = ref(false);
  const uploadCancelled = ref(false);
  const progressMap = ref<Record<string, {
    fileName?: string;
    bytesUploaded: number;
    totalBytes: number;
    percentComplete: number;
    speedBytesPerSec?: number;
    etaSeconds?: number;
    chunkIndex?: number;
    totalChunks?: number;
  }>>({});
  const totalBytesToUpload = ref(0);

  const rejectedFiles = ref<
    Array<{ file: File; errors: Array<{ code: string; message: string }> }>
  >([]);

  const maxFileSizeLabel = computed(() => {
    const size = props.maxFileSize;
    if (size >= 1024 * 1024 * 1024) {
      return `Maximum size: ${Math.round(
        size / (1024 * 1024 * 1024)
      )}GB per file`;
    } else if (size >= 1024 * 1024) {
      return `Maximum size: ${Math.round(size / (1024 * 1024))}MB per file`;
    }
    return `Maximum size: ${Math.round(size / 1024)}KB per file`;
  });

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
  };

  const formatSpeed = (bytesPerSec?: number) => {
    if (!bytesPerSec || bytesPerSec <= 0) return "—";
    const k = 1024;
    const sizes = ["B/s", "KB/s", "MB/s", "GB/s"];
    const i = Math.floor(Math.log(bytesPerSec) / Math.log(k));
    return (
      parseFloat((bytesPerSec / Math.pow(k, i)).toFixed(2)) + " " + sizes[i]
    );
  };

  const clampPercent = (value?: number) => {
    const num = Number.isFinite(value) ? (value as number) : 0;
    return Math.min(100, Math.max(0, num));
  };

  const maxObservedTotal = ref(0);
  const lastBytesSnapshot = ref(0);
  const lastTimestamp = ref<number | null>(null);
  const derivedSpeed = ref(0);
  const derivedSpeedSamples = ref<number[]>([]);
  const smoothedEta = ref<number | undefined>(undefined);
  const smoothedSpeedBuffer = ref<number[]>([]);

  const filePercent = (p: {
    bytesUploaded?: number;
    totalBytes?: number;
    percentComplete?: number;
    chunkIndex?: number;
    totalChunks?: number;
  }) => {
    if (p.totalBytes && p.totalBytes > 0) {
      const loaded = Math.min(p.bytesUploaded || 0, p.totalBytes);
      return clampPercent((loaded / p.totalBytes) * 100);
    }

    if (p.totalChunks && p.totalChunks > 0) {
      const currentChunk = (p.chunkIndex ?? 0) + 1;
      return clampPercent((currentChunk / p.totalChunks) * 100);
    }

    if (p.percentComplete !== undefined) {
      return clampPercent(p.percentComplete);
    }

    return 0;
  };

  const overallProgress = computed(() => {
    const items = Object.values(progressMap.value);

    const loadedInMap = items.reduce(
      (sum, it) => sum + Math.min(it.bytesUploaded || 0, it.totalBytes || 0),
      0
    );

    const totalInMap = items.reduce(
      (sum, it) => sum + (it.totalBytes || 0),
      0
    );

    const inferredTotal = totalInMap;
    const grandTotal =
      totalBytesToUpload.value && totalBytesToUpload.value > 0
        ? totalBytesToUpload.value
        : inferredTotal;

    maxObservedTotal.value = Math.max(maxObservedTotal.value, grandTotal);
    const stableTotal = Math.max(grandTotal, maxObservedTotal.value);

    const grandLoaded = loadedInMap;
    const safeLoaded = Math.min(grandLoaded, stableTotal);

    const haveTotals = stableTotal > 0;
    if (haveTotals) {
      const percent = (safeLoaded / stableTotal) * 100;
      return Math.round(percent);
    }

    if (items.length === 0) return 0;
    const avgPercent =
      items.reduce((sum, it) => sum + filePercent(it), 0) / items.length;
    return Math.round(avgPercent);
  });

  const overallProgressClamped = computed(() => clampPercent(overallProgress.value));

  // Sort progress map: active files first (uploading), completed files at bottom
  const sortedProgressMap = computed(() => {
    const entries = Object.entries(progressMap.value);
    const active = entries.filter(([_, progress]) => progress.percentComplete < 100);
    const completed = entries.filter(([_, progress]) => progress.percentComplete === 100);
    
    return Object.fromEntries([...active, ...completed]);
  });

  const overallSpeed = computed(() => {
    const items = Object.values(progressMap.value);
    const summedSpeeds = items.reduce(
      (sum, it) => sum + (it.speedBytesPerSec || 0),
      0
    );
    return summedSpeeds;
  });

  // Track speed history for smoothing
  watch(overallSpeed, (newSpeed) => {
    if (newSpeed > 0 && isUploading.value) {
      smoothedSpeedBuffer.value = [...smoothedSpeedBuffer.value.slice(-9), newSpeed];
    }
  });

  const smoothedNetworkSpeed = computed(() => {
    if (smoothedSpeedBuffer.value.length === 0) return overallSpeed.value;
    const sum = smoothedSpeedBuffer.value.reduce((a, b) => a + b, 0);
    return sum / smoothedSpeedBuffer.value.length;
  });

  const overallEtaSeconds = computed(() => {
    const items = Object.values(progressMap.value);
    const loadedInMap = items.reduce(
      (sum, it) => sum + Math.min(it.bytesUploaded || 0, it.totalBytes || 0),
      0
    );
    const totalInMap = items.reduce(
      (sum, it) => sum + (it.totalBytes || 0),
      0
    );

    const inferredTotal = totalInMap;
    const grandTotal =
      totalBytesToUpload.value && totalBytesToUpload.value > 0
        ? totalBytesToUpload.value
        : inferredTotal;

    const stableTotal = Math.max(grandTotal, maxObservedTotal.value || 0);
    const totalLoaded = loadedInMap;
    const remaining = Math.max(0, stableTotal - totalLoaded);

    const speed = smoothedNetworkSpeed.value;
    if (remaining === 0) return 0;
    if (speed === 0) return undefined;

    const eta = remaining / speed;
    smoothedEta.value =
      smoothedEta.value === undefined
        ? eta
        : smoothedEta.value * 0.85 + eta * 0.15;

    return Math.round(smoothedEta.value);
  });

  const handleFilesAccepted = (details: { acceptedFiles: File[] }) => {
    rejectedFiles.value = [];
    totalBytesToUpload.value = details.acceptedFiles.reduce((sum, f) => sum + f.size, 0);
  };

  const handleFileRejected = (details: {
    rejectedFiles: Array<{
      file: File;
      errors: Array<{ code: string; message: string }>;
    }>;
  }) => {
    rejectedFiles.value = [...rejectedFiles.value, ...details.rejectedFiles];
  };

  const handleCancelUploads = () => {
    uploadCancelled.value = true;
    isUploading.value = false;
    progressMap.value = {};
  };

  const handleUpload = async (api: any) => {
    if (!props.uploaderId || !props.destinationPath) {
      toast.error("Missing Parameters", "Upload ID and destination path are required");
      return;
    }

    isUploading.value = true;
    uploadCancelled.value = false;
    const files = api.acceptedFiles.slice();
    
    // Calculate total bytes to upload
    const totalBytes = files.reduce((sum: number, f: File) => sum + f.size, 0);
    totalBytesToUpload.value = totalBytes;
    
    // Initialize ALL files in progressMap at once
    for (const file of files) {
      progressMap.value[file.name] = {
        fileName: file.name,
        bytesUploaded: 0,
        totalBytes: file.size,
        percentComplete: 0,
        speedBytesPerSec: 0,
        etaSeconds: undefined,
      };
    }

    const toastId = toast.loading(
      `Uploading ${files.length} file(s)...`,
      "0% complete"
    );

    // Throttle toast updates to every 200ms
    let lastToastUpdate = 0;

    try {
      for (const file of files) {
        if (uploadCancelled.value) break;

        const isDeployment = props.uploadKind === "deployment";

        await (isDeployment
          ? uploadDeploymentFile(file, {
              deploymentId: props.uploaderId,
              organizationId: props.organizationId,
              destinationPath: props.destinationPath || "/",
              volumeName: props.volumeName,
              containerId: props.additionalParams?.containerId,
              serviceName: props.additionalParams?.serviceName,
              onProgress: (progress: { fileName: string; bytesUploaded: number; totalBytes: number; percentComplete: number; }) => {
                progressMap.value[file.name] = {
                  fileName: file.name,
                  bytesUploaded: progress.bytesUploaded,
                  totalBytes: progress.totalBytes,
                  percentComplete: progress.percentComplete,
                  speedBytesPerSec: undefined,
                  etaSeconds: undefined,
                };

                const now = Date.now();
                if (now - lastToastUpdate > 200) {
                  const totalLoaded = Object.values(progressMap.value).reduce(
                    (sum, p) => sum + p.bytesUploaded,
                    0
                  );
                  const percent = totalBytes > 0
                    ? Math.round((totalLoaded / totalBytes) * 100)
                    : 0;

                  toast.update(toastId, `Uploading ${files.length} file(s)...`, `${percent}% complete`);

                  const progressData = {
                    bytesUploaded: totalLoaded,
                    totalBytes: totalBytes,
                    percentComplete: percent,
                    files: Object.values(progressMap.value).map(p => ({
                      fileName: p.fileName || '',
                      bytesUploaded: p.bytesUploaded,
                      totalBytes: p.totalBytes,
                      percentComplete: p.percentComplete || 0
                    }))
                  };

                  emit('uploadProgress', progressData);

                  if (props.targetNode) {
                    props.targetNode.uploadProgress = {
                      isUploading: true,
                      bytesUploaded: totalLoaded,
                      totalBytes: totalBytes,
                      fileCount: progressData.files.length,
                      files: progressData.files,
                      onCancel: undefined,
                    };
                  }

                  if (props.sourceObject) {
                    props.sourceObject.uploadProgress = {
                      isUploading: true,
                      bytesUploaded: totalLoaded,
                      totalBytes: totalBytes,
                      fileCount: progressData.files.length,
                      files: progressData.files,
                      onCancel: undefined,
                    };
                  }

                  lastToastUpdate = now;
                }
              },
            })
          : uploadFile(file, {
              gameServerId: props.uploaderId,
              destinationPath: props.destinationPath || "/",
              volumeName: props.volumeName,
              ...props.additionalParams,
              onProgress: (progress: GameServerUploadProgress) => {
                progressMap.value[file.name] = {
                  fileName: file.name,
                  bytesUploaded: progress.bytesUploaded,
                  totalBytes: progress.totalBytes,
                  percentComplete: progress.percentComplete,
                  speedBytesPerSec: progress.speedBytesPerSec,
                  etaSeconds: progress.etaSeconds,
                };
                
                const now = Date.now();
                if (now - lastToastUpdate > 200) {
                  const totalLoaded = Object.values(progressMap.value).reduce(
                    (sum, p) => sum + p.bytesUploaded,
                    0
                  );
                  const percent = totalBytes > 0 
                    ? Math.round((totalLoaded / totalBytes) * 100)
                    : 0;
                  
                  toast.update(toastId, `Uploading ${files.length} file(s)...`, `${percent}% complete`);
                  
                  const progressData = {
                    bytesUploaded: totalLoaded,
                    totalBytes: totalBytes,
                    percentComplete: percent,
                    files: Object.values(progressMap.value).map(p => ({
                      fileName: p.fileName || '',
                      bytesUploaded: p.bytesUploaded,
                      totalBytes: p.totalBytes,
                      percentComplete: p.percentComplete || 0
                    }))
                  };
                  
                  emit('uploadProgress', progressData);
                  
                  if (props.targetNode) {
                    props.targetNode.uploadProgress = {
                      isUploading: true,
                      bytesUploaded: totalLoaded,
                      totalBytes: totalBytes,
                      fileCount: progressData.files.length,
                      files: progressData.files,
                      onCancel: undefined,
                    };
                  }
                  
                  if (props.sourceObject) {
                    props.sourceObject.uploadProgress = {
                      isUploading: true,
                      bytesUploaded: totalLoaded,
                      totalBytes: totalBytes,
                      fileCount: progressData.files.length,
                      files: progressData.files,
                      onCancel: undefined,
                    };
                  }
                  
                  lastToastUpdate = now;
                }
              },
              onFileComplete: () => {
                progressMap.value[file.name] = {
                  fileName: file.name,
                  bytesUploaded: file.size,
                  totalBytes: file.size,
                  percentComplete: 100,
                  speedBytesPerSec: 0,
                  etaSeconds: 0,
                };
              },
            }));
      }

      toast.dismiss(toastId);
      toast.success(
        "Upload Complete",
        `Successfully uploaded ${files.length} file(s) to ${props.destinationPath}`
      );
      
      emit("uploaded", files);
      
      // Clear upload progress from target node if provided
      if (props.targetNode) {
        props.targetNode.uploadProgress = undefined;
      }
      
      // Clear upload progress from source object if provided
      if (props.sourceObject) {
        props.sourceObject.uploadProgress = undefined;
      }
      
      api.clearFiles();
      progressMap.value = {};
      totalBytesToUpload.value = 0;
      smoothedSpeedBuffer.value = [];
      smoothedEta.value = undefined;
    } catch (error: any) {
      toast.dismiss(toastId);
      if (!uploadCancelled.value) {
        toast.error("Upload Failed", error?.message || "An error occurred during upload");
      } else {
        toast.info("Upload Cancelled", "Upload was stopped");
      }
      progressMap.value = {};
      smoothedSpeedBuffer.value = [];
      smoothedEta.value = undefined;
    } finally {
      isUploading.value = false;
    }
  };
</script>
