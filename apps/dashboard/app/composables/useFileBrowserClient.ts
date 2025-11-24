import type { ExplorerNode } from "~/components/shared/fileExplorerTypes";

export interface FileBrowserClientAdapter {
  // Get file content
  getFile(params: {
    path: string;
    volumeName?: string;
    containerId?: string;
    serviceName?: string;
  }): Promise<{
    content: string;
    encoding: string;
    size: number;
    metadata?: {
      mimeType?: string;
    };
  }>;

  // Create archive
  createArchive(params: {
    sourcePaths: string[];
    destinationPath: string;
    includeParentFolder: boolean;
    volumeName?: string;
    containerId?: string;
    serviceName?: string;
  }): Promise<{
    success: boolean;
    error?: string;
    archivePath?: string;
    filesArchived?: number;
  }>;

  // Extract archive
  extractArchive(params: {
    sourcePath: string;
    destinationPath: string;
    volumeName?: string;
    containerId?: string;
    serviceName?: string;
  }): Promise<{
    success: boolean;
    error?: string;
    filesExtracted?: number;
  }>;

  // Upload files
  uploadFiles(params: {
    destinationPath: string;
    tarData: Uint8Array;
    files: Array<{
      name: string;
      size: number;
      isDirectory: boolean;
      path: string;
    }>;
    volumeName?: string;
    containerId?: string;
    serviceName?: string;
  }): Promise<{
    success: boolean;
    error?: string;
    filesUploaded?: number;
  }>;

  // Search files
  searchFiles?(params: {
    query: string;
    rootPath?: string;
    volumeName?: string;
    maxResults?: number;
    filesOnly?: boolean;
    directoriesOnly?: boolean;
  }): Promise<{
    results: ExplorerNode[];
    totalFound: number;
    hasMore: boolean;
  }>;
}

