import { ref } from "vue";
import { useConnectClient } from "~/lib/connect-client";
import { GameServerService } from "@obiente/proto";
import { create } from "@bufbuild/protobuf";
import { ChunkUploadGameServerFilesRequestSchema } from "@obiente/proto";
import { useChunkedUpload } from "./useChunkedUpload";

export interface GameServerUploadProgress {
  fileName: string;
  bytesUploaded: number;
  totalBytes: number;
  percentComplete: number;
  /** bytes per second measured for this file (moving average) */
  speedBytesPerSec?: number;
  /** estimated seconds remaining */
  etaSeconds?: number;
}

export type UploadProgress = GameServerUploadProgress;

export interface GameServerUploadOptions {
  gameServerId: string;
  destinationPath?: string;
  volumeName?: string;
  chunkSize?: number; // Default: 512 KB
  /**
   * Max parallel chunk uploads per-file. Set to 1 for sequential behaviour.
   */
  maxConcurrency?: number;
  abortSignal?: AbortSignal;
  onProgress?: (progress: GameServerUploadProgress) => void;
  onFileComplete?: (fileName: string) => void;
}

export type UploadOptions = GameServerUploadOptions;

// Increase default chunk size to improve throughput on modern networks.
// 512 KB is a good middle-ground to reduce RPC overhead while avoiding very large memory spikes.
const DEFAULT_CHUNK_SIZE = 512 * 1024; // 512 KB chunks

/**
 * Game server upload composable built on the shared chunked uploader.
 * Splits files into chunks and sends each chunk via unary proto RPC (no bidi streaming).
 * No full-file buffering; streaming happens server-side.
 * Compatible with all browsers including Firefox.
 */
export function useChunkedGameServerUpload() {
  const client = useConnectClient(GameServerService);
  const { uploadFile: chunkedUploadFile, isUploading, error: chunkError } = useChunkedUpload();
  const error = ref<string | null>(null);

  const uploadFile = async (file: File, options: GameServerUploadOptions): Promise<boolean> => {
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

    try {
      error.value = null;
      const success = await chunkedUploadFile(file, async ({ fileName, fileSize, chunkIndex, totalChunks, chunkData }) => {
        const request = create(ChunkUploadGameServerFilesRequestSchema, {
          gameServerId,
          upload: {
            destinationPath,
            volumeName,
            fileName,
            fileSize: BigInt(fileSize),
            chunkIndex,
            totalChunks,
            chunkData,
            fileMode: "0644",
          },
        });

        const response = await client.chunkUploadGameServerFiles(request);
        const result = response.result;
        if (!result?.success) {
          throw new Error(result?.error || `Chunk ${chunkIndex} upload failed`);
        }
      }, {
        chunkSize,
        maxConcurrency,
        abortSignal,
        onProgress,
        onFileComplete,
      });

      if (!success && chunkError.value) {
        error.value = chunkError.value;
      }

      return success;
    } catch (err: any) {
      error.value = err?.message || "Upload failed";
      return false;
    }
  };

  /**
   * Upload multiple files sequentially.
   * Returns { successful: File[], failed: Array<{file: File, error: string}> }
   */
  const uploadFiles = async (
    files: File[],
    options: Omit<GameServerUploadOptions, "onFileComplete">
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

// Legacy alias to keep existing imports working while making the relationship explicit.
export const useStreamingUpload = useChunkedGameServerUpload;
