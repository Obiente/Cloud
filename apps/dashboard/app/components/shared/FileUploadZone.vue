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
          v-if="api.acceptedFiles.length > 0"
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

        <!-- Upload Progress Section (if uploading) -->
        <OuiBox
          v-if="api.acceptedFiles.length > 0 && isUploading && showProgress"
          overflowY="auto"
          class="max-h-56"
        >
          <OuiCard
            bg="transparent"
            border="1"
            borderColor="muted"
            shadow="none"
            rounded="md"
            p="sm"
          >
            <OuiStack gap="sm">
              <OuiStack gap="xs">
                <OuiFlex justify="between" align="center">
                  <OuiText size="sm" weight="semibold"
                    >Overall Progress</OuiText
                  >
                  <OuiStack align="end" gap="xs">
                    <OuiText size="sm" weight="semibold" color="primary"
                      >{{ overallProgress }}%</OuiText
                    >
                    <OuiText size="xs" color="secondary">
                      {{ formatSpeed(overallSpeed) }}
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
                  class="w-full h-3 bg-border-default rounded-md overflow-hidden"
                >
                  <div
                    class="h-full bg-accent-primary rounded-md transition-all duration-300"
                    :style="`width: ${overallProgressClamped}%`"
                  />
                </div>
              </OuiStack>

              <!-- Per-file progress -->
              <OuiStack v-if="Object.keys(progressMap).length > 0" gap="sm">
                <OuiText size="xs" weight="medium">Files:</OuiText>
                <div class="max-h-80 overflow-auto">
                  <OuiStack gap="xs">
                    <OuiCard
                      v-for="(progress, fileName) in progressMap"
                      :key="fileName"
                      bg="transparent"
                      border="1"
                      borderColor="muted"
                      shadow="none"
                      rounded="sm"
                      p="sm"
                    >
                      <OuiStack gap="sm">
                        <OuiFlex justify="between" align="center" gap="sm">
                          <OuiText
                            size="xs"
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
                            {{ formatBytes(progress.totalBytes) }} •
                            {{ formatSpeed(progress.speedBytesPerSec) }}
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
                  </OuiStack>
                </div>
              </OuiStack>
            </OuiStack>
          </OuiCard>
        </OuiBox>

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
              @click="emit('cancelUploads')"
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

  interface Props {
    maxFiles?: number;
    maxFileSize?: number;
    isUploading?: boolean;
    showProgress?: boolean;
    progressMap?: Record<
      string,
      {
        bytesUploaded: number;
        totalBytes: number;
        percentComplete: number;
        speedBytesPerSec?: number;
        etaSeconds?: number;
        chunkIndex?: number;
        totalChunks?: number;
      }
    >;
    completedBytes?: number;
    totalBytesToUpload?: number;
  }

  interface Emits {
    (e: "upload", api: any): void;
    (e: "filesAccepted", details: { acceptedFiles: File[] }): void;
    (
      e: "fileRejected",
      details: {
        rejectedFiles: Array<{
          file: File;
          errors: Array<{ code: string; message: string }>;
        }>;
      }
    ): void;

    (e: "cancelUploads"): void;
  }

  const props = withDefaults(defineProps<Props>(), {
    maxFiles: 500,
    maxFileSize: 1024 * 1024 * 1024, // 1GB default
    isUploading: false,
    showProgress: true,
    progressMap: () => ({}),
    completedBytes: 0,
    totalBytesToUpload: 0,
  });

  const emit = defineEmits<Emits>();

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
      const currentChunk = (p.chunkIndex ?? 0) + 1; // chunkIndex assumed 0-based
      return clampPercent((currentChunk / p.totalChunks) * 100);
    }

    if (p.percentComplete !== undefined) {
      return clampPercent(p.percentComplete);
    }

    return 0;
  };

  const overallProgress = computed(() => {
    const items = Object.values(props.progressMap);

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
      props.totalBytesToUpload && props.totalBytesToUpload > 0
        ? props.totalBytesToUpload
        : inferredTotal;

    // Track max observed total to avoid denominator shrinking mid-flight
    maxObservedTotal.value = Math.max(maxObservedTotal.value, grandTotal);
    const stableTotal = Math.max(grandTotal, maxObservedTotal.value);

    const grandLoaded = loadedInMap;
    const safeLoaded = Math.min(grandLoaded, stableTotal);

    // If we have no byte totals, fall back to average of per-file percents
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

  // Derive overall speed if per-file speeds are missing
  watchEffect(() => {
    const items = Object.values(props.progressMap);
    const loadedInMap = items.reduce(
      (sum, it) => sum + Math.min(it.bytesUploaded || 0, it.totalBytes || 0),
      0
    );
    const totalLoaded = loadedInMap;
    const now = performance.now();
    if (lastTimestamp.value !== null) {
      const dtSeconds = (now - lastTimestamp.value) / 1000;
      if (dtSeconds > 0) {
        const deltaBytes = totalLoaded - lastBytesSnapshot.value;
        const instSpeed = Math.max(0, deltaBytes / dtSeconds);
        const samples = derivedSpeedSamples.value.slice(-4);
        samples.push(instSpeed);
        derivedSpeedSamples.value = samples;
        const avg =
          samples.reduce((acc, v) => acc + v, 0) /
          (samples.length || 1);
        derivedSpeed.value = avg;
      }
    }
    lastBytesSnapshot.value = totalLoaded;
    lastTimestamp.value = now;
  });

  const overallSpeed = computed(() => {
    const items = Object.values(props.progressMap);
    const summedSpeeds = items.reduce(
      (sum, it) => sum + (it.speedBytesPerSec || 0),
      0
    );
    // Prefer summed speeds if present; otherwise fall back to derived speed
    return summedSpeeds > 0 ? summedSpeeds : derivedSpeed.value;
  });

  // Calculate smoothed network speed (averaged over larger window)
  const smoothedNetworkSpeed = computed(() => {
    const items = Object.values(props.progressMap);
    const summedSpeeds = items.reduce(
      (sum, it) => sum + (it.speedBytesPerSec || 0),
      0
    );
    const currentSpeed = summedSpeeds > 0 ? summedSpeeds : derivedSpeed.value;
    
    // Keep a 10-sample buffer for smoother speed averaging
    if (currentSpeed > 0) {
      smoothedSpeedBuffer.value = smoothedSpeedBuffer.value.slice(-9);
      smoothedSpeedBuffer.value.push(currentSpeed);
    }
    
    if (smoothedSpeedBuffer.value.length === 0) return 0;
    return smoothedSpeedBuffer.value.reduce((a, b) => a + b, 0) / smoothedSpeedBuffer.value.length;
  });

  const overallEtaSeconds = computed(() => {
    const items = Object.values(props.progressMap);
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
      props.totalBytesToUpload && props.totalBytesToUpload > 0
        ? props.totalBytesToUpload
        : inferredTotal;

    const stableTotal = Math.max(grandTotal, maxObservedTotal.value || 0);
    const totalLoaded = loadedInMap;
    const remaining = Math.max(0, stableTotal - totalLoaded);

    // Use smoothed speed for more stable ETA
    const speed = smoothedNetworkSpeed.value;
    if (remaining === 0) return 0;
    if (speed === 0) return undefined;

    const eta = remaining / speed;
    // Smooth ETA more aggressively to reduce drastic jumps (0.85 weight to previous)
    smoothedEta.value =
      smoothedEta.value === undefined
        ? eta
        : smoothedEta.value * 0.85 + eta * 0.15;

    return Math.round(smoothedEta.value);
  });

  const handleFilesAccepted = (details: { acceptedFiles: File[] }) => {
    rejectedFiles.value = [];
    emit("filesAccepted", details);
  };

  const handleFileRejected = (details: {
    rejectedFiles: Array<{
      file: File;
      errors: Array<{ code: string; message: string }>;
    }>;
  }) => {
    rejectedFiles.value = [...rejectedFiles.value, ...details.rejectedFiles];
    emit("fileRejected", details);
  };

  const handleUpload = (api: any) => {
    emit("upload", api);
  };

  defineExpose({
    getSmoothedNetworkSpeed: () => smoothedNetworkSpeed.value,
  });
</script>
