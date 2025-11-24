import { ref, computed, nextTick, type Ref } from "vue";
import { detectFilePreviewType, type FilePreviewType } from "./useFilePreview";
import { useZipFile } from "./useZipFile";
import { useDialog } from "./useDialog";
import type { ExplorerNode } from "../components/shared/fileExplorerTypes";

export interface FileMetadata {
  mimeType?: string;
  encoding?: string;
  size?: number;
}

export interface FileLoadResponse {
  content: string;
  encoding: string;
  metadata?: {
    mimeType?: string;
    size?: bigint | number;
  };
  size?: bigint | number;
}

export type FileLoader = (node: ExplorerNode) => Promise<FileLoadResponse>;

export function useFileLoader(
  loader: FileLoader,
  options: {
    selectedPath: Ref<string | null>;
    onPathChange?: (path: string) => void;
    detectLanguage?: (path: string) => string;
    isLikelyUnviewableFile?: (path: string) => { unviewable: boolean; reason?: string };
  }
) {
  const {
    selectedPath,
    onPathChange,
    detectLanguage = () => "plaintext",
    isLikelyUnviewableFile = () => ({ unviewable: false }),
  } = options;

  const dialog = useDialog();
  const {
    zipContents,
    zipLoading,
    parseZipFile,
    clearZip,
  } = useZipFile();

  const isLoadingFile = ref(false);
  let currentFileLoadController: AbortController | null = null;

  const fileContent = ref("");
  const originalFileContent = ref("");
  const fileLanguage = ref("plaintext");
  const currentFilePath = ref<string | null>(null);
  const fileError = ref<string | null>(null);
  const fileMetadata = ref<FileMetadata | null>(null);
  const fileBlobUrl = ref<string | null>(null);
  const filePreviewType = ref<FilePreviewType | null>(null);
  const editorRefreshKey = ref(0);

  const hasUnsavedChanges = computed(() => {
    if (!currentFilePath.value) return false;
    return fileContent.value !== originalFileContent.value;
  });

  async function loadFile(node: ExplorerNode, checkUnsaved: boolean = true) {
    if (node.type !== "file") return;

    // Cancel any pending file load request
    if (currentFileLoadController) {
      currentFileLoadController.abort();
      currentFileLoadController = null;
    }

    // Prevent concurrent loads of the same file
    if (isLoadingFile.value && currentFilePath.value === node.path) {
      console.log(
        "[useFileLoader] Already loading this file, skipping duplicate request"
      );
      return;
    }

    // Clean up previous blob URL when switching files
    if (fileBlobUrl.value) {
      URL.revokeObjectURL(fileBlobUrl.value);
      fileBlobUrl.value = null;
    }
    filePreviewType.value = null;
    fileMetadata.value = null;
    clearZip();

    // Check if file is likely unviewable before attempting to load
    const unviewableCheck = isLikelyUnviewableFile(node.path);
    if (unviewableCheck.unviewable) {
      fileError.value = unviewableCheck.reason || "This file cannot be viewed";
      fileContent.value = "";
      fileLanguage.value = "plaintext";
      selectedPath.value = node.path;
      currentFilePath.value = node.path;
      if (onPathChange) onPathChange(node.path);
      return;
    }

    // Check for unsaved changes before switching files
    if (checkUnsaved && hasUnsavedChanges.value && currentFilePath.value) {
      const confirmed = await dialog.showConfirm({
        title: "Unsaved Changes",
        message: `You have unsaved changes in ${currentFilePath.value
          .split("/")
          .pop()}. Open another file?`,
        confirmLabel: "Discard & Open",
        cancelLabel: "Cancel",
      });
      if (!confirmed) return;
    }

    selectedPath.value = node.path;
    currentFilePath.value = node.path;
    fileError.value = null;
    originalFileContent.value = "";
    if (onPathChange) onPathChange(node.path);

    // Create new AbortController for this request
    const abortController = new AbortController();
    currentFileLoadController = abortController;
    isLoadingFile.value = true;

    try {
      // Store the request path to verify it's still the current file after load
      const requestPath = node.path;

      const res = await loader(node);

      // Verify this request is still valid (file hasn't changed during load)
      if (currentFilePath.value !== requestPath) {
        console.log(
          "[useFileLoader] File changed during load, discarding stale response"
        );
        return;
      }

      // Store metadata
      fileMetadata.value = {
        mimeType: res.metadata?.mimeType,
        encoding: res.encoding || "text",
        size: Number(res.size || 0),
      };

      // Determine preview type based on MIME type or file extension
      const mimeType = res.metadata?.mimeType || "";
      const fileSize = Number(res.size || 0);
      const previewType = detectFilePreviewType(node.path, mimeType, fileSize);
      filePreviewType.value = previewType;

      if (previewType === "text") {
        // Text file - show in editor
        const content = res.content || "";
        fileContent.value = content;
        originalFileContent.value = content;
        editorRefreshKey.value++;
        fileLanguage.value = detectLanguage(node.path);
        // Clean up any existing blob URL
        if (fileBlobUrl.value) {
          URL.revokeObjectURL(fileBlobUrl.value);
          fileBlobUrl.value = null;
        }
      } else if (previewType === "zip") {
        // Zip file - parse and show contents
        fileContent.value = "";
        fileLanguage.value = "plaintext";

        try {
          await parseZipFile(res.content, res.encoding || "base64");
        } catch (err) {
          console.error("Failed to parse zip file:", err);
          fileError.value = "Failed to parse zip file. It may be corrupted.";
          filePreviewType.value = "binary";
        }

        // Clean up any existing blob URL
        if (fileBlobUrl.value) {
          URL.revokeObjectURL(fileBlobUrl.value);
          fileBlobUrl.value = null;
        }
      } else {
        // Media file - create blob URL for preview
        fileContent.value = "";
        fileLanguage.value = "plaintext";

        // Create blob from content
        let blob: Blob;
        if (res.encoding === "base64") {
          // Convert base64 to binary
          const binaryString = atob(res.content);
          const bytes = new Uint8Array(binaryString.length);
          for (let i = 0; i < binaryString.length; i++) {
            bytes[i] = binaryString.charCodeAt(i);
          }
          blob = new Blob([bytes], {
            type: mimeType || "application/octet-stream",
          });
        } else {
          // Text content (shouldn't happen for media, but handle it)
          blob = new Blob([res.content], {
            type: mimeType || "text/plain",
          });
        }

        // Create object URL for preview
        if (fileBlobUrl.value) {
          URL.revokeObjectURL(fileBlobUrl.value);
        }
        fileBlobUrl.value = URL.createObjectURL(blob);
      }

      fileError.value = null;
    } catch (err: any) {
      // Don't show error if request was aborted (cancelled)
      if (err?.name === "AbortError" || err?.message?.includes("aborted")) {
        console.log("[useFileLoader] Request was aborted");
        return;
      }

      console.error("load file", err);
      fileError.value = err?.message || "Failed to load file";
      fileContent.value = "";
      fileLanguage.value = "plaintext";
      filePreviewType.value = null;
      fileMetadata.value = null;
      if (fileBlobUrl.value) {
        URL.revokeObjectURL(fileBlobUrl.value);
        fileBlobUrl.value = null;
      }
    } finally {
      isLoadingFile.value = false;
      if (currentFileLoadController === abortController) {
        currentFileLoadController = null;
      }
    }
  }

  function reset() {
    if (fileBlobUrl.value) {
      URL.revokeObjectURL(fileBlobUrl.value);
      fileBlobUrl.value = null;
    }
    fileContent.value = "";
    originalFileContent.value = "";
    fileLanguage.value = "plaintext";
    currentFilePath.value = null;
    fileError.value = null;
    fileMetadata.value = null;
    filePreviewType.value = null;
    clearZip();
  }

  return {
    // State
    fileContent,
    originalFileContent,
    fileLanguage,
    currentFilePath,
    fileError,
    fileMetadata,
    fileBlobUrl,
    filePreviewType,
    editorRefreshKey,
    isLoadingFile,
    hasUnsavedChanges,
    zipContents,
    zipLoading,

    // Methods
    loadFile,
    reset,
  };
}

