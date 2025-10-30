<template>
  <OuiCardBody>
    <OuiStack gap="md">
      <OuiFlex justify="between" align="center">
        <OuiText as="h3" size="md" weight="semibold">File Browser</OuiText>
        <OuiFlex gap="sm">
          <OuiButton variant="ghost" size="sm" @click="refreshFiles">
            <ArrowPathIcon class="h-4 w-4" />
            Refresh
          </OuiButton>
          <OuiButton variant="ghost" size="sm" @click="uploadFile">
            Upload File
          </OuiButton>
        </OuiFlex>
      </OuiFlex>

      <OuiCard v-if="currentPath" variant="subtle">
        <OuiCardBody>
          <OuiFlex align="center" gap="sm">
            <FolderIcon class="h-4 w-4 text-secondary" />
            <OuiText size="sm" weight="medium">{{ currentPath }}</OuiText>
            <OuiButton
              v-if="currentPath !== '/'"
              variant="ghost"
              size="xs"
              @click="navigateUp"
            >
              ↑ Up
            </OuiButton>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>

      <div v-if="isLoading" class="flex justify-center py-8">
        <OuiText color="secondary">Loading files...</OuiText>
      </div>

      <OuiStack v-else gap="sm">
        <div
          v-for="file in files"
          :key="file.name"
          class="flex items-center gap-3 p-3 rounded-lg hover:bg-surface-muted cursor-pointer transition-colors"
          @click="handleFileClick(file)"
        >
          <component
            :is="file.type === 'directory' ? FolderIcon : DocumentTextIcon"
            class="h-5 w-5 text-secondary"
          />
          <div class="flex-1">
            <OuiText size="sm" weight="medium">{{ file.name }}</OuiText>
            <OuiText v-if="file.size" size="xs" color="secondary">
              {{ formatSize(file.size) }}
            </OuiText>
          </div>
          <OuiButton
            v-if="file.type === 'file'"
            variant="ghost"
            size="xs"
            @click.stop="downloadFile(file)"
          >
            Download
          </OuiButton>
        </div>

        <OuiText v-if="files.length === 0" color="secondary" class="text-center py-4">
          No files found in this directory.
        </OuiText>
      </OuiStack>

      <OuiText size="xs" color="secondary">
        Note: File browser requires container filesystem API. This feature will be fully implemented when the backend API is ready.
      </OuiText>
    </OuiStack>
  </OuiCardBody>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import {
  ArrowPathIcon,
  DocumentTextIcon,
  FolderIcon,
} from "@heroicons/vue/24/outline";

interface Props {
  deploymentId: string;
  organizationId: string;
}

interface FileEntry {
  name: string;
  type: "file" | "directory";
  size?: number;
  path: string;
}

const props = defineProps<Props>();

const files = ref<FileEntry[]>([]);
const currentPath = ref("/");
const isLoading = ref(false);

const refreshFiles = async () => {
  isLoading.value = true;
  // TODO: Implement API call to list files
  // const res = await client.listDeploymentFiles({
  //   organizationId: props.organizationId,
  //   deploymentId: props.deploymentId,
  //   path: currentPath.value,
  // });
  
  // Mock data for now
  setTimeout(() => {
    files.value = [
      { name: "src", type: "directory", path: "/src" },
      { name: "package.json", type: "file", size: 1024, path: "/package.json" },
      { name: "README.md", type: "file", size: 2048, path: "/README.md" },
    ];
    isLoading.value = false;
  }, 500);
};

const handleFileClick = (file: FileEntry) => {
  if (file.type === "directory") {
    currentPath.value = file.path;
    refreshFiles();
  } else {
    // TODO: Open file viewer/edit
    console.log("Open file:", file.path);
  }
};

const navigateUp = () => {
  const parts = currentPath.value.split("/").filter(Boolean);
  parts.pop();
  currentPath.value = "/" + parts.join("/");
  if (currentPath.value === "/") {
    currentPath.value = "/";
  }
  refreshFiles();
};

const downloadFile = (file: FileEntry) => {
  // TODO: Implement file download
  console.log("Download file:", file.path);
};

const uploadFile = () => {
  // TODO: Implement file upload
  console.log("Upload file to:", currentPath.value);
};

const formatSize = (bytes: number) => {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
};

onMounted(() => {
  refreshFiles();
});
</script>

