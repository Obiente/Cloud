import { ref, computed } from "vue";
import { useAuth } from "~/composables/useAuth";
import { useConnectClient } from "~/lib/connect-client";
import { GameServerService } from "@obiente/proto";
import { create } from "@bufbuild/protobuf";
import { ChunkUploadGameServerFilesRequestSchema } from "@obiente/proto";

export interface UploadProgress {
  fileName: string;
  bytesUploaded: number;
  totalBytes: number;
  percentComplete: number;
  /** bytes per second measured for this file (moving average) */
  speedBytesPerSec?: number;
  /** estimated seconds remaining */
  etaSeconds?: number;
}

export interface UploadOptions {
  gameServerId: string;
  destinationPath?: string;
  volumeName?: string;
  chunkSize?: number; // Default: 512 KB
  /**
   * Max parallel chunk uploads per-file. Set to 1 for sequential behaviour.
   */
  maxConcurrency?: number;
  abortSignal?: AbortSignal;
  onProgress?: (progress: UploadProgress) => void;
  onFileComplete?: (fileName: string) => void;
}

// Increase default chunk size to improve throughput on modern networks.
// 512 KB is a good middle-ground to reduce RPC overhead while avoiding very large memory spikes.
const DEFAULT_CHUNK_SIZE = 512 * 1024; // 512 KB chunks

/**
 * Shared streaming file upload composable.
 * Uploads a single file by splitting it into chunks and sending each chunk via unary proto RPC.
 * No buffering of entire file in memory; streaming happens server-side.
 * Compatible with all browsers including Firefox (uses unary RPC, not bidi streaming).
 */
export function useStreamingUpload() {
  const auth = useAuth();
  const client = useConnectClient(GameServerService);
  const isUploading = ref(false);
  const error = ref<string | null>(null);

  const uploadFile = async (file: File, options: UploadOptions): Promise<boolean> => {
    const {
      gameServerId,
      destinationPath = "/",
      volumeName,
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

      // Determine concurrency: either explicit, or adaptive based on network estimate
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

      // Clamp concurrency
      concurrency = Math.max(1, Math.min(8, concurrency));

      let bytesUploaded = 0;
      let failed = false;

      // Moving average of recent throughputs (bytes/sec) for adaptive throttling
      const recentThroughputs: number[] = [];
      const pushThroughput = (bps: number) => {
        recentThroughputs.push(bps);
        if (recentThroughputs.length > 8) recentThroughputs.shift();
      };
      const avgThroughput = () => {
        if (recentThroughputs.length === 0) return 0;
        return Math.max(1, Math.round(recentThroughputs.reduce((a, b) => a + b, 0) / recentThroughputs.length));
      };

      // Helpers to measure time
      const now = typeof performance !== "undefined" && performance.now ? () => performance.now() : () => Date.now();

      // Batch-based parallel uploader with adaptive concurrency adjustments between batches
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

            const request = create(ChunkUploadGameServerFilesRequestSchema, {
              gameServerId,
              destinationPath,
              volumeName,
              fileName: file.name,
              fileSize: BigInt(file.size),
              chunkIndex,
              totalChunks,
              chunkData: new Uint8Array(chunkBytes),
              fileMode: "0644",
            });

            const chunkSentAt = now();
            try {
              const response = await client.chunkUploadGameServerFiles(request);
              const chunkDoneAt = now();

              if (!response.success) {
                failed = true;
                error.value = response.error || `Chunk ${chunkIndex} upload failed`;
                return;
              }

              // update progress using chunk length
              const added = end - start;
              bytesUploaded += added;

              // throughput for this chunk (bytes/sec)
              const durationSec = Math.max(0.001, (chunkDoneAt - chunkSentAt) / 1000);
              const bps = Math.round(added / durationSec);
              pushThroughput(bps);

              // compute moving average speed and ETA
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
              error.value = `Failed to upload chunk ${chunkIndex}: ${err?.message || "Unknown error"}`;
              return;
            }
          })
        );

        const batchEnd = now();
        const batchDurationSec = Math.max(0.001, (batchEnd - batchStart) / 1000);
        const batchBytes = Math.min(chunkSize * batchSize, file.size - (nextIndex - batchSize) * chunkSize + (chunkSize - 1));
        const batchBps = Math.round(batchBytes / batchDurationSec);
        pushThroughput(batchBps);

        // adaptive adjustment: compare avg throughput to previous and adjust concurrency
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

  /**
   * Upload multiple files sequentially.
   * Returns { successful: File[], failed: Array<{file: File, error: string}> }
   */
  const uploadFiles = async (
    files: File[],
    options: Omit<UploadOptions, "onFileComplete">
  ): Promise<{ successful: File[]; failed: Array<{ file: File; error: string }> }> => {
    const successful: File[] = [];
    const failed: Array<{ file: File; error: string }> = [];

    for (const file of files) {
      const success = await uploadFile(file, {
        ...options,
        onFileComplete: () => {
          successful.push(file);
        },
      });

      if (!success) {
        failed.push({
          file,
          error: error.value || "Unknown error",
        });
      }
    }

    return { successful, failed };
  };

  return {
    isUploading,
    error,
    uploadFile,
    uploadFiles,
  };
}
