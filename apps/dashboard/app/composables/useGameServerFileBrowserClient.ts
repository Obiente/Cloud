import { useConnectClient } from "~/lib/connect-client";
import { GameServerService } from "@obiente/proto";
import type { FileBrowserClientAdapter } from "./useFileBrowserClient";
import { create } from "@bufbuild/protobuf";
import { 
  UploadGameServerFilesRequestSchema, 
  UploadGameServerFilesMetadataSchema,
} from "@obiente/proto";

export function useGameServerFileBrowserClient(gameServerId: string): FileBrowserClientAdapter {
  const client = useConnectClient(GameServerService);

  return {
    async getFile(params) {
      const res = await client.getGameServerFile({
        gameServerId,
        path: params.path,
        volumeName: params.volumeName,
      });
      return {
        content: res.content || "",
        encoding: res.encoding || "text",
        size: Number(res.size || 0),
        metadata: {
          mimeType: res.metadata?.mimeType,
        },
      };
    },

    async createArchive(params) {
      const response = await client.createGameServerFileArchive({
        gameServerId,
        archiveRequest: {
          sourcePaths: params.sourcePaths,
          destinationPath: params.destinationPath,
          includeParentFolder: params.includeParentFolder,
        },
        volumeName: params.volumeName,
      });
      return {
        success: response.archiveResponse?.success || false,
        error: response.archiveResponse?.error,
        archivePath: response.archiveResponse?.archivePath,
        filesArchived: response.archiveResponse?.filesArchived,
      };
    },

    async extractArchive(params) {
      const response = await client.extractGameServerFile({
        gameServerId,
        zipPath: params.sourcePath,
        destinationPath: params.destinationPath,
        volumeName: params.volumeName,
      });
      return {
        success: response.success || false,
        error: response.error,
        filesExtracted: response.filesExtracted,
      };
    },

    async uploadFiles(params) {
      const metadata = create(UploadGameServerFilesMetadataSchema, {
        gameServerId,
        destinationPath: params.destinationPath,
        files: params.files.map(f => ({
          name: f.name,
          size: BigInt(f.size),
          isDirectory: f.isDirectory,
          path: f.path,
        })),
        volumeName: params.volumeName,
      });

      const request = create(UploadGameServerFilesRequestSchema, {
        metadata,
        tarData: params.tarData,
      });

      const response = await client.uploadGameServerFiles(request);
      return {
        success: response.success || false,
        error: response.error,
        filesUploaded: response.filesUploaded,
      };
    },

    async searchFiles(params) {
      const response = await client.searchGameServerFiles({
        gameServerId,
        query: params.query,
        rootPath: params.rootPath,
        volumeName: params.volumeName,
        maxResults: params.maxResults,
        filesOnly: params.filesOnly,
        directoriesOnly: params.directoriesOnly,
      });
      
      // Helper to convert timestamp to ISO string (matches useFileExplorer implementation)
      const timestampToIso = (value: any): string | undefined => {
        if (!value) return undefined;
        if (typeof value === "string") return value;
        if (value instanceof Date) return value.toISOString();
        if (typeof value === "object") {
          if (typeof value.toDate === "function") {
            return value.toDate().toISOString();
          }
          if (value.seconds !== undefined) {
            const millis = Number(value.seconds) * 1000 + Math.floor((value.nanos ?? 0) / 1e6);
            return new Date(millis).toISOString();
          }
        }
        return undefined;
      };
      
      return {
        results: (response.results || []).map(f => {
          const isDirectory = Boolean(f.isDirectory);
          const isSymlink = Boolean(f.isSymlink);
          const type: "directory" | "file" | "symlink" = isDirectory
            ? "directory"
            : isSymlink
            ? "symlink"
            : "file";
          const path = f.path || "";
          return {
            id: path,
            name: f.name || path.split("/").pop() || path,
            path,
            parentPath: path.split("/").slice(0, -1).join("/") || "/",
            type,
            symlinkTarget: f.symlinkTarget || undefined,
            size: f.size !== undefined && f.size !== null ? Number(f.size) : undefined,
            owner: f.owner || undefined,
            group: f.group || undefined,
            mode: f.modeOctal ? Number(f.modeOctal) : undefined,
            mimeType: f.mimeType || undefined,
            modifiedTime: timestampToIso(f.modifiedTime),
            createdTime: timestampToIso(f.createdTime),
            volumeName: params.volumeName,
            children: [],
            isLoading: false,
            hasLoaded: false,
            hasMore: false,
            nextCursor: null,
            isExpanded: false,
          };
        }),
        totalFound: response.totalFound || 0,
        hasMore: response.hasMore || false,
      };
    },
  };
}

