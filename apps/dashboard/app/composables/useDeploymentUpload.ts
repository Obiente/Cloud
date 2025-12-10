import { ref } from "vue";
import { create } from "@bufbuild/protobuf";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService, UploadContainerFilesMetadataSchema, UploadContainerFilesRequestSchema } from "@obiente/proto";

export interface DeploymentUploadOptions {
  deploymentId: string;
  organizationId?: string;
  destinationPath?: string;
  volumeName?: string;
  containerId?: string;
  serviceName?: string;
  onProgress?: (progress: {
    fileName: string;
    bytesUploaded: number;
    totalBytes: number;
    percentComplete: number;
  }) => void;
}

function createTarForSingleFile(file: File): Promise<Uint8Array> {
  return new Promise(async (resolve) => {
    const tarParts: Uint8Array[] = [];
    const encoder = new TextEncoder();

    const fileBytes = new Uint8Array(await file.arrayBuffer());
    const header = new Uint8Array(512);

    const writeOctal = (value: number, fieldSize: number) => {
      const str = value.toString(8).padStart(fieldSize - 1, "0") + " ";
      return encoder.encode(str);
    };

    // name (100)
    header.set(encoder.encode(file.name).slice(0, 100), 0);
    // mode (8)
    header.set(encoder.encode("0000644"), 100);
    // uid (8)
    header.set(encoder.encode("0000000"), 108);
    // gid (8)
    header.set(encoder.encode("0000000"), 116);
    // size (12)
    header.set(writeOctal(fileBytes.length, 12), 124);
    // mtime (12)
    header.set(writeOctal(Math.floor(Date.now() / 1000), 12), 136);
    // typeflag (1) regular file '0'
    header[156] = 48;
    // magic (6)
    header.set(encoder.encode("ustar "), 257);
    // version (2)
    header.set(encoder.encode(" "), 263);

    // checksum
    let checksum = 256;
    for (let i = 0; i < 512; i++) {
      if (i >= 148 && i < 156) continue;
      checksum += header[i] ?? 0;
    }
    header.set(writeOctal(checksum, 8), 148);

    tarParts.push(header);
    tarParts.push(fileBytes);

    const padding = 512 - (fileBytes.length % 512);
    if (padding < 512) {
      tarParts.push(new Uint8Array(padding));
    }

    tarParts.push(new Uint8Array(512));
    tarParts.push(new Uint8Array(512));

    const totalLength = tarParts.reduce((sum, arr) => sum + arr.length, 0);
    const result = new Uint8Array(totalLength);
    let offset = 0;
    for (const part of tarParts) {
      result.set(part, offset);
      offset += part.length;
    }
    resolve(result);
  });
}

export function useDeploymentUpload() {
  const isUploading = ref(false);
  const error = ref<string | null>(null);
  const client = useConnectClient(DeploymentService);

  const uploadFile = async (
    file: File,
    options: DeploymentUploadOptions
  ): Promise<boolean> => {
    const {
      deploymentId,
      organizationId,
      destinationPath = "/",
      volumeName,
      containerId,
      serviceName,
      onProgress,
    } = options;

    if (!file) {
      error.value = "No file provided";
      return false;
    }

    try {
      isUploading.value = true;
      error.value = null;

      const tarData = await createTarForSingleFile(file);

      const metadata = create(UploadContainerFilesMetadataSchema, {
        organizationId,
        deploymentId,
        destinationPath,
        volumeName,
        containerId: !volumeName ? containerId : undefined,
        serviceName: !volumeName ? serviceName : undefined,
        files: [
          {
            name: file.name,
            size: BigInt(file.size),
            isDirectory: false,
            path: file.name,
          },
        ],
      });

      const request = create(UploadContainerFilesRequestSchema, {
        metadata,
        tarData,
      });

      // Since this is a single unary upload, progress is coarse: mark as 100% after request completes
      const response = await client.uploadContainerFiles(request);
      if (!response.success) {
        error.value = response.error || "Upload failed";
        return false;
      }

      if (onProgress) {
        onProgress({
          fileName: file.name,
          bytesUploaded: file.size,
          totalBytes: file.size,
          percentComplete: 100,
        });
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
