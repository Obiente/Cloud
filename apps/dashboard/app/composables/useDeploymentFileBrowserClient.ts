import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService } from "@obiente/proto";
import type { FileBrowserClientAdapter } from "./useFileBrowserClient";
import { create } from "@bufbuild/protobuf";
import { 
  UploadContainerFilesRequestSchema, 
  UploadContainerFilesMetadataSchema,
} from "@obiente/proto";

export function useDeploymentFileBrowserClient(
  deploymentId: string,
  organizationId: () => string
): FileBrowserClientAdapter {
  const client = useConnectClient(DeploymentService);

  return {
    async getFile(params) {
      const res = await client.getContainerFile({
        organizationId: organizationId(),
        deploymentId,
        path: params.path,
        volumeName: params.volumeName,
        containerId: params.containerId,
        serviceName: params.serviceName,
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
      const response = await client.createDeploymentFileArchive({
        deploymentId,
        organizationId: organizationId(),
        archiveRequest: {
          sourcePaths: params.sourcePaths,
          destinationPath: params.destinationPath,
          includeParentFolder: params.includeParentFolder,
        },
        volumeName: params.volumeName,
        containerId: params.containerId,
        serviceName: params.serviceName,
      });
      return {
        success: response.archiveResponse?.success || false,
        error: response.archiveResponse?.error,
        archivePath: response.archiveResponse?.archivePath,
        filesArchived: response.archiveResponse?.filesArchived,
      };
    },

    async extractArchive(params) {
      const response = await client.extractDeploymentFile({
        deploymentId,
        organizationId: organizationId(),
        zipPath: params.sourcePath,
        destinationPath: params.destinationPath,
        volumeName: params.volumeName,
        containerId: params.containerId,
        serviceName: params.serviceName,
      });
      return {
        success: response.success || false,
        error: response.error,
        filesExtracted: response.filesExtracted,
      };
    },

    async uploadFiles(params) {
      const metadata = create(UploadContainerFilesMetadataSchema, {
        organizationId: organizationId(),
        deploymentId,
        destinationPath: params.destinationPath,
        volumeName: params.volumeName,
        containerId: params.containerId,
        serviceName: params.serviceName,
        files: params.files.map(f => ({
          name: f.name,
          size: BigInt(f.size),
          isDirectory: f.isDirectory,
          path: f.path,
        })),
      });

      const request = create(UploadContainerFilesRequestSchema, {
        metadata,
        tarData: params.tarData,
      });

      const response = await client.uploadContainerFiles(request);
      return {
        success: response.success || false,
        error: response.error,
        filesUploaded: response.filesUploaded,
      };
    },
  };
}

