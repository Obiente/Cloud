<template>
  <OuiCardBody>
    <OuiStack gap="lg">
      <OuiFlex justify="between" align="center">
        <OuiText as="h3" size="md" weight="semibold">File Browser & Upload</OuiText>
        <OuiButton variant="ghost" size="sm" @click="refreshFiles" :disabled="isLoading">
          <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': isLoading }" />
          Refresh
        </OuiButton>
      </OuiFlex>

      <!-- Breadcrumb Navigation -->
      <OuiFlex v-if="currentPath !== '/'" gap="sm" align="center" class="flex-wrap">
        <OuiButton variant="ghost" size="xs" @click="navigateUp">
          <ArrowUturnLeftIcon class="h-3 w-3 mr-1" />
          Up
        </OuiButton>
        <OuiText size="xs" color="secondary">{{ currentPath }}</OuiText>
      </OuiFlex>

      <!-- File List -->
      <div v-if="isLoading" class="flex justify-center py-8">
        <OuiText color="secondary">Loading files...</OuiText>
      </div>

      <OuiStack v-else-if="files.length > 0" gap="sm">
        <div
          v-for="file in files"
          :key="file.path"
          class="flex items-center gap-3 p-3 rounded-lg border border-border-default bg-surface-muted/30 hover:bg-surface-raised cursor-pointer transition-colors"
          @click="file.isDirectory ? navigateTo(file.path) : openFile(file.path)"
        >
          <component :is="file.isDirectory ? FolderIcon : DocumentIcon" class="h-5 w-5 text-secondary" />
          <div class="flex-1">
            <OuiText size="sm" weight="medium">{{ file.name }}</OuiText>
            <OuiText size="xs" color="secondary">
              {{ file.isDirectory ? "Directory" : formatSize(file.size) }}
            </OuiText>
          </div>
        </div>
      </OuiStack>

      <OuiText v-else size="sm" color="secondary" class="text-center py-4">
        No files found in this directory.
      </OuiText>

      <!-- File Upload Section -->
      <OuiCard variant="subtle">
        <OuiCardBody>
          <FileUploader :deployment-id="deploymentId" @uploaded="handleFilesUploaded" />
        </OuiCardBody>
      </OuiCard>

      <!-- File Viewer Modal -->
      <FileViewerModal
        v-model:open="fileViewerOpen"
        :deployment-id="deploymentId"
        :file-path="selectedFilePath"
        :organization-id="organizationId"
      />

      <OuiStack v-if="uploadedFilesCount > 0" gap="sm">
        <OuiText size="sm" weight="semibold">
          Recent Uploads ({{ uploadedFilesCount }})
        </OuiText>
        <OuiText size="xs" color="secondary">
          Files are being processed and will be available in your deployment shortly.
        </OuiText>
      </OuiStack>
    </OuiStack>
  </OuiCardBody>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { ArrowPathIcon, ArrowUturnLeftIcon, FolderIcon, DocumentIcon } from "@heroicons/vue/24/outline";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService } from "@obiente/proto";
import { useOrganizationsStore } from "~/stores/organizations";
import FileUploader from "~/components/deployments/FileUploader.vue";
import FileViewerModal from "~/components/deployments/FileViewerModal.vue";

interface Props {
  deploymentId: string;
  organizationId?: string;
}

const props = defineProps<Props>();
const orgsStore = useOrganizationsStore();
const organizationId = computed(() => props.organizationId || orgsStore.currentOrgId || "");

const client = useConnectClient(DeploymentService);
const files = ref<any[]>([]);
const currentPath = ref("/");
const isLoading = ref(false);
const uploadedFilesCount = ref(0);

const loadFiles = async (path: string = "/") => {
  isLoading.value = true;
  try {
    const res = await client.listContainerFiles({
      organizationId: organizationId.value,
      deploymentId: props.deploymentId,
      path: path,
    });
    files.value = res.files || [];
    currentPath.value = res.currentPath || path;
  } catch (error) {
    console.error("Failed to load files:", error);
    files.value = [];
  } finally {
    isLoading.value = false;
  }
};

const refreshFiles = () => {
  loadFiles(currentPath.value);
};

const navigateTo = (path: string) => {
  loadFiles(path);
};

const navigateUp = () => {
  const parts = currentPath.value.split("/").filter(Boolean);
  if (parts.length > 0) {
    parts.pop();
    loadFiles("/" + parts.join("/") || "/");
  } else {
    loadFiles("/");
  }
};

const fileViewerOpen = ref(false);
const selectedFilePath = ref("");

const openFile = async (path: string) => {
  selectedFilePath.value = path;
  fileViewerOpen.value = true;
};

const formatSize = (bytes: number | bigint) => {
  const size = typeof bytes === "bigint" ? Number(bytes) : bytes;
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
};

const handleFilesUploaded = (files: File[]) => {
  uploadedFilesCount.value += files.length;
  refreshFiles(); // Refresh to show new files
};

onMounted(() => {
  loadFiles();
});
</script>
