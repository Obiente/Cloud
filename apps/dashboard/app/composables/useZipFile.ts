import { ref, type Ref } from "vue";
import type JSZip from "jszip";

export interface ZipEntry {
  name: string;
  size: number;
  isDirectory: boolean;
  path: string;
}

export interface FileMetadata {
  mimeType?: string;
  encoding?: string;
  size?: number;
}

export function useZipFile() {
  const zipContents = ref<ZipEntry[]>([]);
  const zipLoading = ref(false);
  const zipInstance = ref<JSZip | null>(null);
  const draggedZipEntry = ref<{ name: string; path: string } | null>(null);
  const currentZipPath = ref<string>(""); // Current path in zip (for folder navigation)

  /**
   * Parse zip file content from base64 or text encoded string
   */
  async function parseZipFile(
    content: string,
    encoding: string
  ): Promise<ZipEntry[]> {
    zipLoading.value = true;
    zipContents.value = [];

    try {
      // Get binary data
      let zipData: Uint8Array;
      if (encoding === "base64") {
        // Convert base64 to binary
        const binaryString = atob(content);
        zipData = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
          zipData[i] = binaryString.charCodeAt(i);
        }
      } else {
        // Shouldn't happen for zip files, but handle it
        const encoder = new TextEncoder();
        zipData = encoder.encode(content);
      }

      // Dynamically load JSZip to reduce initial bundle size
      const JSZipModule = await import("jszip");
      const JSZip = JSZipModule.default;
      
      // Parse zip file
      const zip = await JSZip.loadAsync(zipData);
      zipInstance.value = zip;
      currentZipPath.value = ""; // Reset to root
      
      const entries = getZipEntriesForPath(zip, "");
      zipContents.value = entries;
      return entries;
    } catch (err) {
      console.error("Failed to parse zip file:", err);
      zipInstance.value = null;
      throw err;
    } finally {
      zipLoading.value = false;
    }
  }

  /**
   * Handle drag start for a zip entry
   */
  async function handleZipEntryDragStart(
    event: DragEvent,
    entry: ZipEntry
  ): Promise<void> {
    if (entry.isDirectory || !zipInstance.value) {
      event.preventDefault();
      return;
    }

    draggedZipEntry.value = entry;

    try {
      // Get the file from zip
      const zipEntry = zipInstance.value.file(entry.path);
      if (!zipEntry) {
        event.preventDefault();
        return;
      }

      // Extract file content
      const blob = await zipEntry.async("blob");
      const file = new File([blob], entry.name, {
        type: blob.type || "application/octet-stream",
      });

      // Set drag data
      if (event.dataTransfer) {
        event.dataTransfer.effectAllowed = "copy";
        // Store zip entry info for extraction on drop
        event.dataTransfer.setData(
          "application/x-zip-entry",
          JSON.stringify({
            path: entry.path,
            name: entry.name,
          })
        );
        // Store the file directly in the dataTransfer
        const dataTransfer = event.dataTransfer as any;
        if (dataTransfer.items) {
          dataTransfer.items.add(file);
        }
      }
    } catch (err) {
      console.error("Failed to extract file from zip:", err);
      event.preventDefault();
      draggedZipEntry.value = null;
    }
  }

  /**
   * Handle drag end for a zip entry
   */
  function handleZipEntryDragEnd(): void {
    draggedZipEntry.value = null;
  }

  /**
   * Extract files from zip entry data on drop
   */
  async function extractZipEntryOnDrop(
    event: DragEvent
  ): Promise<File[] | null> {
    if (!event.dataTransfer || !zipInstance.value) {
      return null;
    }

    try {
      const zipEntryData = event.dataTransfer.getData("application/x-zip-entry");
      if (zipEntryData) {
        const entry = JSON.parse(zipEntryData);
        // Extract file from zip
        const zipEntry = zipInstance.value.file(entry.path);
        if (zipEntry) {
          const blob = await zipEntry.async("blob");
          const file = new File([blob], entry.name, {
            type: blob.type || "application/octet-stream",
          });
          return [file];
        }
      }
    } catch (err) {
      console.error("Failed to extract zip entry on drop:", err);
    }

    return null;
  }

  /**
   * Format file size for display
   */
  function formatFileSize(bytes: number): string {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
  }

  /**
   * Get entries for a specific path in the zip
   */
  function getZipEntriesForPath(zip: JSZip, path: string): ZipEntry[] {
    const entries: ZipEntry[] = [];
    const pathPrefix = path ? (path.endsWith("/") ? path : path + "/") : "";
    const seenNames = new Set<string>();

    // Process all files in the zip
    for (const [relativePath, zipEntry] of Object.entries(zip.files)) {
      if (!zipEntry) continue;

      // Filter by current path
      if (path) {
        if (!relativePath.startsWith(pathPrefix)) continue;
        // Skip if it's not directly in this directory
        const relativeToPath = relativePath.slice(pathPrefix.length);
        if (!relativeToPath || relativeToPath.includes("/")) {
          // Check if it's a direct child
          const parts = relativeToPath.split("/");
          if (parts.length > 1 && parts[0]) {
            // It's in a subdirectory, skip
            continue;
          }
        }
      } else {
        // At root, only show root-level items
        if (relativePath.includes("/") && !relativePath.endsWith("/")) {
          const parts = relativePath.split("/");
          if (parts.length > 1) continue;
        }
      }

      // Skip if it's a directory marker (ends with /)
      const isDirectory = zipEntry.dir || relativePath.endsWith("/");
      const name =
        relativePath.slice(pathPrefix.length).split("/").filter(Boolean)[0] || relativePath;

      // Avoid duplicates
      if (seenNames.has(name)) continue;
      seenNames.add(name);

      // Get file size - JSZip stores it in _data.uncompressedSize
      const zipEntryAny = zipEntry as any;
      const size = zipEntryAny._data?.uncompressedSize || zipEntryAny._uncompressedSize || 0;
      
      entries.push({
        name: name,
        size: typeof size === 'number' ? size : 0,
        isDirectory: isDirectory,
        path: relativePath,
      });
    }

    // Sort entries: directories first, then files, both alphabetically
    entries.sort((a, b) => {
      if (a.isDirectory !== b.isDirectory) {
        return a.isDirectory ? -1 : 1;
      }
      return a.name.localeCompare(b.name);
    });

    return entries;
  }

  /**
   * Navigate into a folder in the zip
   */
  function navigateZipFolder(folderPath: string): void {
    if (!zipInstance.value) return;
    currentZipPath.value = folderPath;
    const entries = getZipEntriesForPath(zipInstance.value, folderPath);
    zipContents.value = entries;
  }

  /**
   * Navigate up in the zip folder structure
   */
  function navigateZipUp(): void {
    if (!currentZipPath.value) return;
    const parts = currentZipPath.value.split("/").filter(Boolean);
    parts.pop();
    const newPath = parts.length > 0 ? parts.join("/") : "";
    navigateZipFolder(newPath);
  }

  /**
   * Clear zip instance and contents
   */
  function clearZip(): void {
    zipInstance.value = null;
    zipContents.value = [];
    zipLoading.value = false;
    draggedZipEntry.value = null;
    currentZipPath.value = "";
  }

  return {
    zipContents,
    zipLoading,
    zipInstance,
    draggedZipEntry,
    currentZipPath,
    parseZipFile,
    handleZipEntryDragStart,
    handleZipEntryDragEnd,
    extractZipEntryOnDrop,
    formatFileSize,
    navigateZipFolder,
    navigateZipUp,
    clearZip,
  };
}

// Export formatFileSize as a standalone function for use in components
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
}

