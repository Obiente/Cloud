<template>
  <OuiCardBody>
    <OuiStack gap="lg">
      <OuiText as="h3" size="md" weight="semibold">File Browser & Upload</OuiText>
      
      <OuiCard variant="subtle">
        <OuiCardBody>
          <FileUploader :deployment-id="deploymentId" @uploaded="handleFilesUploaded" />
        </OuiCardBody>
      </OuiCard>

      <OuiText size="sm" color="secondary">
        Container filesystem browser coming soon! For now, you can upload files directly.
      </OuiText>

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
import { ref } from "vue";
import FileUploader from "~/components/deployments/FileUploader.vue";

interface Props {
  deploymentId: string;
  organizationId?: string;
}

const props = defineProps<Props>();
const uploadedFilesCount = ref(0);

const handleFilesUploaded = (files: File[]) => {
  uploadedFilesCount.value += files.length;
  console.log(`${files.length} files uploaded for deployment ${props.deploymentId}`);
};
</script>
