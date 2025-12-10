<template>
  <BaseFileUploader
    :uploaderId="deploymentId"
    :destinationPath="destinationPath"
    :volumeName="volumeName"
    :organizationId="organizationId"
    uploadKind="deployment"
    :additionalParams="{
      containerId,
      serviceName,
    }"
    :targetNode="targetNode"
    :sourceObject="sourceObject"
    @uploaded="emit('uploaded', $event)"
    @uploadProgress="emit('uploadProgress', $event)"
  />
</template>

<script setup lang="ts">
  import BaseFileUploader from "~/components/shared/BaseFileUploader.vue";

  interface Props {
    deploymentId: string;
    destinationPath?: string;
    volumeName?: string;
    organizationId?: string;
    containerId?: string;
    serviceName?: string;
    targetNode?: any;
    sourceObject?: any;
  }

  interface Emits {
    (e: "uploaded", files: File[]): void;
    (e: "uploadProgress", progress: any): void;
  }

  const props = defineProps<Props>();
  const emit = defineEmits<Emits>();
</script>
