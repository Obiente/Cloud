import { ref } from "vue";

export interface ChunkProgress {
  fileName: string;
  bytesUploaded: number;
  totalBytes: number;
  percentComplete: number;
  speedBytesPerSec?: number;
  etaSeconds?: number;
}

export interface ChunkedUploadOptions {
  chunkSize?: number;
  maxConcurrency?: number;
  abortSignal?: AbortSignal;
  onProgress?: (progress: ChunkProgress) => void;
  onFileComplete?: (fileName: string) => void;
}

export interface ChunkSenderContext {
  fileName: string;
  fileSize: number;
  chunkIndex: number;
  totalChunks: number;
  chunkData: Uint8Array;
}

export type SendChunkFn = (ctx: ChunkSenderContext) => Promise<void>;

const DEFAULT_CHUNK_SIZE = 512 * 1024; // 512 KB

export function useChunkedUpload() {
  const isUploading = ref(false);
  const error = ref<string | null>(null);

  const uploadFile = async (
    file: File,
    sendChunk: SendChunkFn,
    options: ChunkedUploadOptions = {}
  ): Promise<boolean> => {
    const {
      chunkSize = DEFAULT_CHUNK_SIZE,
      maxConcurrency,
      abortSignal,
      onProgress,
      onFileComplete,
    } = options;

    if (!file) {
      error.value = "No file provided";
      return false;
    }

    if (abortSignal?.aborted) {
      error.value = "Upload cancelled";
      return false;
    }

    isUploading.value = true;
    error.value = null;

    try {
      const totalChunks = Math.ceil(file.size / chunkSize);

      // Determine concurrency
      let concurrency = 1;
      if (typeof maxConcurrency === "number" && maxConcurrency > 0) {
        concurrency = maxConcurrency;
      } else if (typeof navigator !== "undefined" && (navigator as any).connection) {
        const downlink = (navigator as any).connection.downlink || 0;
        if (downlink >= 20) concurrency = 6;
        else if (downlink >= 8) concurrency = 4;
        else if (downlink >= 2) concurrency = 2;
        else concurrency = 1;
      } else {
        concurrency = 2;
      }
      concurrency = Math.max(1, Math.min(8, concurrency));

      let bytesUploaded = 0;
      let failed = false;

      const recentThroughputs: number[] = [];
      const pushThroughput = (bps: number) => {
        recentThroughputs.push(bps);
        if (recentThroughputs.length > 8) recentThroughputs.shift();
      };
      const avgThroughput = () => {
        if (recentThroughputs.length === 0) return 0;
        return Math.max(
          1,
          Math.round(recentThroughputs.reduce((a, b) => a + b, 0) / recentThroughputs.length)
        );
      };

      const now =
        typeof performance !== "undefined" && performance.now ? () => performance.now() : () => Date.now();

      let nextIndex = 0;
      let previousAvg = 0;

      while (nextIndex < totalChunks && !failed) {
        if (abortSignal?.aborted) {
          error.value = "Upload cancelled";
          failed = true;
          break;
        }

        const batchSize = Math.min(concurrency, totalChunks - nextIndex);
        const batchIndices: number[] = [];
        for (let i = 0; i < batchSize; i++) batchIndices.push(nextIndex + i);
        nextIndex += batchSize;

        const batchStart = now();

        await Promise.all(
          batchIndices.map(async (chunkIndex) => {
            if (failed || abortSignal?.aborted) return;
            const start = chunkIndex * chunkSize;
            const end = Math.min(start + chunkSize, file.size);
            const chunkBytes = await file.slice(start, end).arrayBuffer();

            const chunkSentAt = now();
            try {
              await sendChunk({
                fileName: file.name,
                fileSize: file.size,
                chunkIndex,
                totalChunks,
                chunkData: new Uint8Array(chunkBytes),
              });
              const chunkDoneAt = now();

              const added = end - start;
              bytesUploaded += added;

              const durationSec = Math.max(0.001, (chunkDoneAt - chunkSentAt) / 1000);
              const bps = Math.round(added / durationSec);
              pushThroughput(bps);

              const speedBps = avgThroughput();
              const remaining = Math.max(0, file.size - bytesUploaded);
              const eta = speedBps > 0 ? Math.round(remaining / speedBps) : undefined;

              if (onProgress) {
                onProgress({
                  fileName: file.name,
                  bytesUploaded,
                  totalBytes: file.size,
                  percentComplete: Math.round((bytesUploaded / file.size) * 100),
                  speedBytesPerSec: speedBps,
                  etaSeconds: eta,
                });
              }
            } catch (err: any) {
              failed = true;
              error.value = err?.message || `Failed to upload chunk ${chunkIndex}`;
              return;
            }
          })
        );

        const batchEnd = now();
        const batchDurationSec = Math.max(0.001, (batchEnd - batchStart) / 1000);
        const batchBytes = Math.min(
          chunkSize * batchSize,
          file.size - (nextIndex - batchSize) * chunkSize + (chunkSize - 1)
        );
        const batchBps = Math.round(batchBytes / batchDurationSec);
        pushThroughput(batchBps);

        const currentAvg = avgThroughput();
        if (previousAvg > 0) {
          if (currentAvg > previousAvg * 1.15 && concurrency < (maxConcurrency || 8)) {
            concurrency = Math.min(8, concurrency + 1);
          } else if (currentAvg < previousAvg * 0.85 && concurrency > 1) {
            concurrency = Math.max(1, concurrency - 1);
          }
        }
        previousAvg = currentAvg;
      }

      if (failed) return false;

      if (onFileComplete) {
        onFileComplete(file.name);
      }

      return true;
    } catch (err: any) {
      error.value = err?.message || "Upload failed";
      return false;
    } finally {
      isUploading.value = false;
    }
  };

  return {
    uploadFile,
    isUploading,
    error,
  };
}
