<template>
  <FileUpload.Root
    :maxFiles="20"
    :maxFileSize="100 * 1024 * 1024"
    @filesAccepted="handleFilesAccepted"
    @fileRejected="handleFileRejected"
  >
        <FileUpload.Context v-slot="api">
          <FileUpload.Dropzone
            :class="[
              'border-2 border-dashed rounded-xl p-8 text-center cursor-pointer transition-colors',
              api.dragging ? 'border-primary bg-primary/10' : 'border-border-default hover:border-primary/50',
            ]"
          >
            <OuiStack gap="sm" align="center">
              <ArrowUpTrayIcon class="h-10 w-10 text-text-tertiary" />
              <OuiStack gap="xs" align="center">
                <OuiText size="md" weight="semibold">
                  {{ api.dragging ? 'Drop files here to upload' : 'Click or drag files to upload' }}
                </OuiText>
                <OuiText size="xs" color="secondary">
                  {{ api.dragging ? 'Release to upload' : 'Upload multiple files at once' }}
                </OuiText>
              </OuiStack>
              <OuiFlex gap="xs" align="center" class="mt-2">
                <OuiText size="xs" color="secondary">or</OuiText>
                <FileUpload.Trigger asChild>
                  <OuiButton variant="outline" size="sm">
                    Browse Files
                  </OuiButton>
                </FileUpload.Trigger>
              </OuiFlex>
              <OuiText size="xs" color="secondary" class="mt-1">
                Maximum size: 100MB per file
              </OuiText>
            </OuiStack>
          </FileUpload.Dropzone>

          <FileUpload.ItemGroup v-if="api.acceptedFiles.length > 0" class="mt-4">
            <OuiStack gap="sm">
              <OuiText size="xs" weight="semibold">
                Selected Files ({{ api.acceptedFiles.length }})
              </OuiText>
              <OuiCard
                v-for="file in api.acceptedFiles"
                :key="file.name"
                variant="outline"
                class="p-3"
              >
                <OuiFlex align="center" gap="md">
                <FileUpload.Item :file="file">
                  <FileUpload.ItemPreview type="image/*">
                    <FileUpload.ItemPreviewImage class="h-10 w-10 rounded object-cover" />
                  </FileUpload.ItemPreview>
                  <FileUpload.ItemPreview type=".*">
                    <DocumentIcon class="h-5 w-5 text-secondary" />
                  </FileUpload.ItemPreview>
                  <OuiStack gap="xs" class="flex-1 min-w-0">
                    <OuiText size="sm" weight="medium" class="truncate">
                      <FileUpload.ItemName />
                    </OuiText>
                    <OuiText size="xs" color="secondary">
                      <FileUpload.ItemSizeText />
                    </OuiText>
                  </OuiStack>
                  <FileUpload.ItemDeleteTrigger asChild>
                    <OuiButton variant="ghost" size="xs" color="danger">
                      Remove
                    </OuiButton>
                  </FileUpload.ItemDeleteTrigger>
                </FileUpload.Item>
                </OuiFlex>
              </OuiCard>

              <OuiFlex justify="end" gap="sm" class="mt-2">
                <FileUpload.ClearTrigger asChild>
                  <OuiButton variant="ghost" size="sm">
                    Clear All
                  </OuiButton>
                </FileUpload.ClearTrigger>
                <OuiButton
                  @click="uploadFiles(api)"
                  :disabled="isUploading || api.acceptedFiles.length === 0"
                  size="sm"
                >
                  {{ isUploading ? "Uploading..." : `Upload ${api.acceptedFiles.length} File${api.acceptedFiles.length !== 1 ? 's' : ''}` }}
                </OuiButton>
              </OuiFlex>
            </OuiStack>
          </FileUpload.ItemGroup>

          <FileUpload.HiddenInput />
        </FileUpload.Context>
      </FileUpload.Root>

    <!-- Rejected Files -->
    <OuiStack v-if="rejectedFiles.length > 0" gap="xs" class="mt-4">
      <OuiText size="xs" weight="semibold" color="danger">Rejected Files:</OuiText>
      <OuiCard variant="default" class="border-danger" v-for="rejection in rejectedFiles" :key="rejection.file.name">
        <OuiCardBody class="py-2">
          <OuiStack gap="xs">
            <OuiText size="xs" weight="medium">{{ rejection.file.name }}</OuiText>
            <OuiText size="xs" color="danger" v-for="error in rejection.errors" :key="error.code">
              {{ error.message }}
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>

    <OuiText v-if="uploadError" size="xs" color="danger" class="mt-4">{{ uploadError }}</OuiText>
    <OuiText v-if="uploadSuccess" size="xs" color="success" class="mt-4">{{ uploadSuccess }}</OuiText>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { FileUpload } from "@ark-ui/vue/file-upload";
import { ArrowUpTrayIcon, DocumentIcon } from "@heroicons/vue/24/outline";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService, UploadContainerFilesRequestSchema, UploadContainerFilesMetadataSchema } from "@obiente/proto";
import { create } from "@bufbuild/protobuf";
import { useOrganizationsStore } from "~/stores/organizations";

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

const isUploading = ref(false);
const uploadError = ref("");
const uploadSuccess = ref("");
const rejectedFiles = ref<Array<{ file: File; errors: Array<{ code: string; message: string }> }>>([]);

const handleFilesAccepted = (details: { acceptedFiles: File[] }) => {
  uploadError.value = "";
  uploadSuccess.value = "";
};

const handleFileRejected = (details: { rejectedFiles: Array<{ file: File; errors: Array<{ code: string; message: string }> }> }) => {
  rejectedFiles.value = [...rejectedFiles.value, ...details.rejectedFiles];
  uploadError.value = `Some files were rejected. Please check the requirements.`;
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
  uploadError.value = "";
  uploadSuccess.value = "";

  try {
    // Create tar archive
    const tarData = await createTarArchive(api.acceptedFiles);
    
    // Create metadata using protobuf schema
    const metadata = create(UploadContainerFilesMetadataSchema, {
      organizationId: organizationId.value,
      deploymentId: props.deploymentId,
      destinationPath: props.destinationPath || "/",
      volumeName: props.volumeName,
      containerId: !props.volumeName && props.containerId ? props.containerId : undefined,
      serviceName: !props.volumeName && props.serviceName ? props.serviceName : undefined,
      files: api.acceptedFiles.map((f: File) => ({
        name: f.name,
        size: BigInt(f.size),
        isDirectory: false,
        path: f.name,
      })),
    });
    
    // Create single non-streaming request with all data
    const request = create(UploadContainerFilesRequestSchema, {
      metadata: metadata,
      tarData: new Uint8Array(tarData),
    });
    
    // Call the upload RPC with a single request (non-streaming)
    const response = await client.uploadContainerFiles(request);

    if (response.success) {
      uploadSuccess.value = `Successfully uploaded ${response.filesUploaded} file(s)`;
      emit("uploaded", api.acceptedFiles);
      api.clearFiles();
      rejectedFiles.value = [];
      
      // Clear success message after 3 seconds
      setTimeout(() => {
        uploadSuccess.value = "";
      }, 3000);
    } else {
      uploadError.value = response.error || "Upload failed";
    }
  } catch (error: any) {
    console.error("Upload error:", error);
    uploadError.value = error.message || "Failed to upload files";
  } finally {
    isUploading.value = false;
  }
};
</script>
