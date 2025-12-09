export type ExplorerEntryType = "directory" | "file" | "symlink";

export interface FileUploadProgress {
  fileName: string;
  bytesUploaded: number;
  totalBytes: number;
  percentComplete: number;
}

export interface ExplorerNode {
  id: string;
  name: string;
  path: string;
  parentPath: string;
  type: ExplorerEntryType;
  symlinkTarget?: string;
  size?: number;
  owner?: string;
  group?: string;
  mode?: number;
  mimeType?: string;
  modifiedTime?: string;
  createdTime?: string;
  volumeName?: string;
  children: ExplorerNode[];
  isLoading: boolean;
  hasLoaded: boolean;
  hasMore: boolean;
  nextCursor: string | null;
  isExpanded: boolean;
  // Upload progress tracking
  uploadProgress?: {
    isUploading: boolean;
    bytesUploaded: number;
    totalBytes: number;
    fileCount: number;
    files: FileUploadProgress[];
    onCancel?: () => void;
  };
}




