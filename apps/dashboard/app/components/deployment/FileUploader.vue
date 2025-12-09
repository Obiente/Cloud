<template>
  <FileUploadZone
    :maxFiles="20"
    :maxFileSize="100 * 1024 * 1024"
    :isUploading="isUploading"
    :showProgress="false"
    @upload="uploadFiles"
    @filesAccepted="handleFilesAccepted"
    @fileRejected="handleFileRejected"
  />
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService, UploadContainerFilesRequestSchema, UploadContainerFilesMetadataSchema } from "@obiente/proto";
import { create } from "@bufbuild/protobuf";
import { useOrganizationsStore } from "~/stores/organizations";
import { useUploadManager } from "~/composables/useUploadManager";
import FileUploadZone from "~/components/shared/FileUploadZone.vue";

interface Props {
  deploymentId: string;
  destinationPath?: string;
  volumeName?: string;
  containerId?: string;
  serviceName?: string;
}

interface Emits {
  (e: "uploaded", files: File[]): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();
const orgsStore = useOrganizationsStore();
const organizationId = computed(() => orgsStore.currentOrgId || "");
const client = useConnectClient(DeploymentService);
const uploadManager = useUploadManager();

const isUploading = ref(false);

const handleFilesAccepted = (details: { acceptedFiles: File[] }) => {
  // Handled by FileUploadZone component
};

const handleFileRejected = (details: { rejectedFiles: Array<{ file: File; errors: Array<{ code: string; message: string }> }> }) => {
  // Handled by FileUploadZone component
};

// Helper to create a simple tar archive from files
const createTarArchive = async (files: File[]): Promise<Uint8Array> => {
  const tarData: Uint8Array[] = [];
  
  for (const file of files) {
    const name = file.name;
    const content = await file.arrayBuffer();
    const fileBytes = new Uint8Array(content);
    
    // Tar header: 512 bytes
    const header = new Uint8Array(512);
    
    // Write file name (100 bytes)
    const nameBytes = new TextEncoder().encode(name);
    header.set(nameBytes.slice(0, 100), 0);
    
    // Write file mode (8 bytes) - 0644
    header.set(new TextEncoder().encode("0000644"), 100);
    
    // Write UID (8 bytes) - 0
    header.set(new TextEncoder().encode("0000000"), 108);
    
    // Write GID (8 bytes) - 0
    header.set(new TextEncoder().encode("0000000"), 116);
    
    // Write size (12 bytes) - octal
    const sizeStr = fileBytes.length.toString(8).padStart(11, "0") + " ";
    header.set(new TextEncoder().encode(sizeStr), 124);
    
    // Write mtime (12 bytes) - current time in octal
    const mtime = Math.floor(Date.now() / 1000).toString(8).padStart(11, "0") + " ";
    header.set(new TextEncoder().encode(mtime), 136);
    
    // Write typeflag (1 byte) - regular file (0)
    header[156] = 48; // '0'
    
    // Write magic (6 bytes)
    header.set(new TextEncoder().encode("ustar "), 257);
    
    // Write version (2 bytes)
    header.set(new TextEncoder().encode(" "), 263);
    
    // Calculate checksum
    let checksum = 256; // Sum of all header bytes with checksum field as spaces
    for (let i = 0; i < 512; i++) {
      if (i >= 148 && i < 156) continue; // Skip checksum field
      checksum += header[i] ?? 0;
    }
    const checksumStr = checksum.toString(8).padStart(6, "0") + "\0 ";
    header.set(new TextEncoder().encode(checksumStr), 148);
    
    tarData.push(header);
    tarData.push(fileBytes);
    
    // Pad to 512-byte boundary
    const padding = 512 - (fileBytes.length % 512);
    if (padding < 512) {
      tarData.push(new Uint8Array(padding));
    }
  }
  
  // Two empty blocks to mark end of archive
  tarData.push(new Uint8Array(512));
  tarData.push(new Uint8Array(512));
  
  // Concatenate all parts
  const totalLength = tarData.reduce((sum, arr) => sum + arr.length, 0);
  const result = new Uint8Array(totalLength);
  let offset = 0;
  for (const arr of tarData) {
    result.set(arr, offset);
    offset += arr.length;
  }
  
  return result;
};

const uploadFiles = async (api: any) => {
  if (!api || api.acceptedFiles.length === 0) return;

  isUploading.value = true;
  
  // Set up upload manager
  const totalBytes = api.acceptedFiles.reduce((sum: number, f: File) => sum + f.size, 0);
  uploadManager.setTotalBytesToUpload(totalBytes);
  uploadManager.resetForNewBatch();

  try {
    // Upload in batches to avoid creating a single huge tar in memory
    const MAX_BATCH_BYTES = 25 * 1024 * 1024; // 25 MB per batch
    const MAX_BATCH_FILES = 5;

    const files = api.acceptedFiles.slice();
    let uploadedFilesCount = 0;

    while (files.length > 0) {
      const batch: File[] = [];
      let batchBytes = 0;

      while (files.length > 0 && batch.length < MAX_BATCH_FILES) {
        const next = files[0];
        if (batch.length === 0 && next.size > MAX_BATCH_BYTES) {
          batch.push(files.shift() as File);
          break;
        }
        if (batchBytes + next.size > MAX_BATCH_BYTES) break;
        batch.push(files.shift() as File);
        batchBytes += next.size;
      }

      const tarData = await createTarArchive(batch);

      const metadata = create(UploadContainerFilesMetadataSchema, {
        organizationId: organizationId.value,
        deploymentId: props.deploymentId,
        destinationPath: props.destinationPath || "/",
        volumeName: props.volumeName,
        containerId: !props.volumeName && props.containerId ? props.containerId : undefined,
        serviceName: !props.volumeName && props.serviceName ? props.serviceName : undefined,
        files: batch.map((f: File) => ({
          name: f.name,
          size: BigInt(f.size),
          isDirectory: false,
          path: f.name,
        })),
      });

      const request = create(UploadContainerFilesRequestSchema, {
        metadata: metadata,
        tarData: new Uint8Array(tarData),
      });

      // Update manager progress for batch
      uploadManager.updateProgress("batch", {
        bytesUploaded: uploadedFilesCount * 1024 * 1024, // Rough estimate
        totalBytes: totalBytes,
        percentComplete: Math.round((uploadedFilesCount / totalBytes) * 100),
      });

      const response = await client.uploadContainerFiles(request);
      if (!response.success) {
        throw new Error(response.error || "Upload failed");
      }

      uploadedFilesCount += response.filesUploaded || 0;
      
      // Update progress after successful batch upload
      uploadManager.updateProgress("batch", {
        bytesUploaded: uploadedFilesCount * 1024 * 1024,
        totalBytes: totalBytes,
        percentComplete: Math.round((uploadedFilesCount / totalBytes) * 100),
      });
    }

    emit("uploaded", api.acceptedFiles);
    api.clearFiles();
  } catch (error: any) {
    console.error("Upload error:", error);
  } finally {
    isUploading.value = false;
    uploadManager.clearProgress();
  }
};
</script>
